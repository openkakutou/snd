package snd

import (
	"encoding/binary"
	"testing"
)

func TestDecodeV2Sound_Decodes16BitStereoPCM_ForEmbeddedEntry(t *testing.T) {
	samples := []int16{-100, 200, 30000, -30000}
	raw := make([]byte, 8)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(raw[i*2:i*2+2], uint16(s))
	}
	wav := buildWAV(t, 2, 44100, 16, raw)

	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	sound, err := DecodeV2Sound(readerAtBytes(data), table, 0, 0, nil)
	if err != nil {
		t.Fatalf("DecodeV2Sound: unexpected error: %v", err)
	}

	if sound.Channels != 2 {
		t.Errorf("Channels = %d, want 2", sound.Channels)
	}
	if sound.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", sound.SampleRate)
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

func TestDecodeV2Sound_Decodes8BitMonoPCM_CenteredAndScaledTo16Bit(t *testing.T) {
	raw := []byte{128, 0, 255}
	wav := buildWAV(t, 1, 11025, 8, raw)

	// A Sample value that would not fit alongside Group in v1's 2-byte
	// field, deliberately exercising v2's 4-byte Sample field (decision 002).
	data := buildV2File(t, []v1TestEntry{{group: 3, sample: 1000, payload: wav}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	sound, err := DecodeV2Sound(readerAtBytes(data), table, 3, 1000, nil)
	if err != nil {
		t.Fatalf("DecodeV2Sound: unexpected error: %v", err)
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

func TestDecodeV2Sound_ReturnsError_WhenGroupSampleNotInTable(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0})
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 9, 9, nil)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for a (group, sample) not in the table, got nil")
	}
}

func TestDecodeV2Sound_ReturnsError_OnUnsupportedFormatTag(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0, 0, 0})
	binary.LittleEndian.PutUint16(wav[20:22], 17) // IMA ADPCM, unsupported

	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 0, 0, nil)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for an unsupported (non-PCM) format tag, got nil")
	}
}

func TestDecodeV2Sound_ReturnsError_OnTruncatedSampleData(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{0, 0, 0, 0})
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: wav}})

	truncated := data[:len(data)-3]

	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	_, err = DecodeV2Sound(readerAtBytes(truncated), table, 0, 0, nil)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for truncated sample data, got nil")
	}
}

func TestDecodeV2Sound_ReturnsError_WhenBlobIsCorruptGarbage_NotAPathAndNotRIFFWAVE(t *testing.T) {
	// Bytes that are neither a valid RIFF/WAVE blob nor a plausible external
	// file path (contains control bytes, no recognized audio extension) —
	// must surface the same "not a valid RIFF/WAVE blob" error v1 already
	// gives for corrupt embedded entries, not be misread as an external
	// reference.
	garbage := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0x10, 0x20}
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: garbage}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 0, 0, nil)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for corrupt, non-path garbage data, got nil")
	}
}
