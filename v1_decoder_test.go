package snd

import (
	"encoding/binary"
	"testing"
)

// buildWAV assembles a minimal RIFF/WAVE blob with a "fmt " chunk (PCM,
// format tag 1) and a "data" chunk holding raw sample bytes.
func buildWAV(t *testing.T, channels, sampleRate, bitsPerSample int, data []byte) []byte {
	t.Helper()

	fmtChunk := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtChunk[0:2], 1) // PCM
	binary.LittleEndian.PutUint16(fmtChunk[2:4], uint16(channels))
	binary.LittleEndian.PutUint32(fmtChunk[4:8], uint32(sampleRate))
	blockAlign := channels * bitsPerSample / 8
	binary.LittleEndian.PutUint32(fmtChunk[8:12], uint32(sampleRate*blockAlign))
	binary.LittleEndian.PutUint16(fmtChunk[12:14], uint16(blockAlign))
	binary.LittleEndian.PutUint16(fmtChunk[14:16], uint16(bitsPerSample))

	var buf []byte
	buf = append(buf, []byte("RIFF")...)
	buf = append(buf, 0, 0, 0, 0) // filled in below
	buf = append(buf, []byte("WAVE")...)

	buf = append(buf, []byte("fmt ")...)
	buf = appendUint32(buf, uint32(len(fmtChunk)))
	buf = append(buf, fmtChunk...)

	buf = append(buf, []byte("data")...)
	buf = appendUint32(buf, uint32(len(data)))
	buf = append(buf, data...)
	if len(data)%2 == 1 {
		buf = append(buf, 0)
	}

	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(buf)-8))
	return buf
}

func appendUint32(b []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(b, tmp[:]...)
}

func TestDecodeV1Sound_Decodes16BitStereoPCM(t *testing.T) {
	samples := []int16{-100, 200, 30000, -30000}
	raw := make([]byte, 8) // 2 frames * 2 channels * 2 bytes
	for i, s := range samples {
		binary.LittleEndian.PutUint16(raw[i*2:i*2+2], uint16(s))
	}
	wav := buildWAV(t, 2, 44100, 16, raw)

	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	sound, err := DecodeV1Sound(readerAtBytes(data), table, 0, 0)
	if err != nil {
		t.Fatalf("DecodeV1Sound: unexpected error: %v", err)
	}

	if sound.Channels != 2 {
		t.Errorf("Channels = %d, want 2", sound.Channels)
	}
	if sound.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", sound.SampleRate)
	}
	if sound.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", sound.BitsPerSample)
	}

	want := []int16{-100, 200, 30000, -30000}
	if len(sound.PCM) != len(want) {
		t.Fatalf("len(PCM) = %d, want %d", len(sound.PCM), len(want))
	}
	for i, w := range want {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
}

func TestDecodeV1Sound_Decodes8BitMonoPCM_CenteredAndScaledTo16Bit(t *testing.T) {
	// 8-bit WAV PCM is unsigned, centered at 128. 0 -> silence -> 0;
	// 255 (max) -> most positive; 0 (min) -> most negative.
	raw := []byte{128, 0, 255}
	wav := buildWAV(t, 1, 11025, 8, raw)

	data := buildV1File(t, []v1TestEntry{{group: 3, sample: 1, payload: wav}})
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	sound, err := DecodeV1Sound(readerAtBytes(data), table, 3, 1)
	if err != nil {
		t.Fatalf("DecodeV1Sound: unexpected error: %v", err)
	}

	want := []int16{0, -32768, 32512}
	if len(sound.PCM) != len(want) {
		t.Fatalf("len(PCM) = %d, want %d", len(sound.PCM), len(want))
	}
	for i, w := range want {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
}

func TestDecodeV1Sound_ReturnsError_WhenGroupSampleNotInTable(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0})
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	_, err = DecodeV1Sound(readerAtBytes(data), table, 9, 9)
	if err == nil {
		t.Fatal("DecodeV1Sound: expected an error for a (group, sample) not in the table, got nil")
	}
}

func TestDecodeV1Sound_ReturnsError_WhenBlobIsNotRIFFWAVE(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("not a wav file at all")}})
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	_, err = DecodeV1Sound(readerAtBytes(data), table, 0, 0)
	if err == nil {
		t.Fatal("DecodeV1Sound: expected an error for a non-RIFF/WAVE blob, got nil")
	}
}

func TestDecodeV1Sound_ReturnsError_OnUnsupportedFormatTag(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0, 0, 0})
	// Format tag lives at byte offset 20 in the RIFF stream (right after
	// "RIFF"+size+"WAVE"+"fmt "+size): set it to 17 (IMA ADPCM), which
	// this package does not support (see decision 001).
	binary.LittleEndian.PutUint16(wav[20:22], 17)

	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	_, err = DecodeV1Sound(readerAtBytes(data), table, 0, 0)
	if err == nil {
		t.Fatal("DecodeV1Sound: expected an error for an unsupported (non-PCM) format tag, got nil")
	}
}

func TestDecodeV1Sound_ReturnsError_OnTruncatedSampleData(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0, 0, 0})
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})

	// Truncate the file so the declared sound data does not fully fit.
	truncated := data[:len(data)-3]

	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	_, err = DecodeV1Sound(readerAtBytes(truncated), table, 0, 0)
	if err == nil {
		t.Fatal("DecodeV1Sound: expected an error for truncated sample data, got nil")
	}
}
