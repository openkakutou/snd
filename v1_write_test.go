package snd

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildRealLayoutV1File builds a v1 file the way real community files are
// laid out: a 512-byte header whose tail is a comment, contiguous
// subheaders, and a last subheader that points at the end of the file.
func buildRealLayoutV1File(entries []v1TestEntry) []byte {
	const firstSubheader = 512
	var buf bytes.Buffer

	header := make([]byte, firstSubheader)
	copy(header[0:12], v1Signature)
	header[12], header[13], header[14], header[15] = 1, 0, 1, 0
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(entries)))
	binary.LittleEndian.PutUint32(header[20:24], firstSubheader)
	copy(header[24:], "Made in a test\nwith a comment")
	buf.Write(header)

	pos := firstSubheader
	for _, e := range entries {
		pos += 16 + len(e.payload)
		sub := make([]byte, 16)
		binary.LittleEndian.PutUint32(sub[0:4], uint32(pos))
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(e.payload)))
		binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(e.group)))
		binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(e.sample)))
		buf.Write(sub)
		buf.Write(e.payload)
	}
	return buf.Bytes()
}

func TestEncodeV1_ReproducesRealLayoutFileByteForByte(t *testing.T) {
	original := buildRealLayoutV1File([]v1TestEntry{
		{group: 1, sample: 0, payload: buildWAV(t, 1, 11025, 16, []byte{1, 0, 2, 0})},
		{group: 1, sample: 143, payload: buildWAV(t, 1, 8000, 8, []byte{0x80, 0xff})},
		{group: 40, sample: 7, payload: []byte("not a wav at all")},
	})

	set, err := ReadV1Set(readerAtBytes(original))
	if err != nil {
		t.Fatalf("ReadV1Set: %v", err)
	}
	got, err := EncodeV1(set)
	if err != nil {
		t.Fatalf("EncodeV1: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("re-encoded file differs from the original (%d vs %d bytes)", len(got), len(original))
	}
}

func TestReadV1Set_KeepsKeysVersionPaddingAndAudio(t *testing.T) {
	wav := buildWAV(t, 1, 11025, 16, []byte{1, 0, 2, 0})
	original := buildRealLayoutV1File([]v1TestEntry{{group: 3, sample: 9, payload: wav}})

	set, err := ReadV1Set(readerAtBytes(original))
	if err != nil {
		t.Fatalf("ReadV1Set: %v", err)
	}

	if set.Version != [4]byte{1, 0, 1, 0} {
		t.Errorf("Version = %v, want [1 0 1 0]", set.Version)
	}
	if len(set.HeaderPadding) != 512-24 {
		t.Errorf("len(HeaderPadding) = %d, want %d", len(set.HeaderPadding), 512-24)
	}
	if !strings.HasPrefix(string(set.HeaderPadding), "Made in a test\nwith a comment") {
		t.Errorf("HeaderPadding does not start with the header comment: %q", set.HeaderPadding[:30])
	}
	if len(set.Sounds) != 1 || set.Sounds[0].Group != 3 || set.Sounds[0].Sample != 9 {
		t.Fatalf("Sounds = %+v, want one sound (3, 9)", set.Sounds)
	}
	if !bytes.Equal(set.Sounds[0].Audio, wav) {
		t.Errorf("Audio differs from the embedded blob")
	}
}

func TestEncodeV1_BuiltSetParsesAndDecodesToTheSameSounds(t *testing.T) {
	set := &V1SoundSet{
		Version: [4]byte{1, 0, 1, 0},
		Sounds: []V1Sound{
			{Group: 1, Sample: 0, Audio: buildWAV(t, 1, 11025, 16, []byte{1, 0, 2, 0, 3, 0})},
			{Group: 2, Sample: 5, Audio: buildWAV(t, 1, 8000, 8, []byte{0x80, 0xff})},
		},
	}

	data, err := EncodeV1(set)
	if err != nil {
		t.Fatalf("EncodeV1: %v", err)
	}
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1 of the encoded file: %v", err)
	}
	if table.Header.NumberOfSounds != 2 {
		t.Fatalf("NumberOfSounds = %d, want 2", table.Header.NumberOfSounds)
	}

	first, err := DecodeV1Sound(readerAtBytes(data), table, 1, 0)
	if err != nil {
		t.Fatalf("DecodeV1Sound(1, 0): %v", err)
	}
	if first.SampleRate != 11025 || first.BitsPerSample != 16 || len(first.PCM) != 3 || first.PCM[2] != 3 {
		t.Errorf("sound (1, 0) decoded wrongly: %+v", first)
	}
	second, err := DecodeV1Sound(readerAtBytes(data), table, 2, 5)
	if err != nil {
		t.Fatalf("DecodeV1Sound(2, 5): %v", err)
	}
	if second.SampleRate != 8000 || second.BitsPerSample != 8 || len(second.PCM) != 2 {
		t.Errorf("sound (2, 5) decoded wrongly: %+v", second)
	}
}

func TestEncodeV1_EmptySetRoundTripsToAnEmptySet(t *testing.T) {
	data, err := EncodeV1(&V1SoundSet{Version: [4]byte{1, 0, 1, 0}})
	if err != nil {
		t.Fatalf("EncodeV1: %v", err)
	}
	set, err := ReadV1Set(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ReadV1Set: %v", err)
	}
	if len(set.Sounds) != 0 || set.Version != [4]byte{1, 0, 1, 0} {
		t.Errorf("got %+v, want an empty set with the same version", set)
	}
}

func TestEncodeV1_KeepsDuplicateKeysAndEmptyAudio(t *testing.T) {
	set := &V1SoundSet{Sounds: []V1Sound{
		{Group: 1, Sample: 1, Audio: []byte("aaaa")},
		{Group: 1, Sample: 1, Audio: []byte("bb")},
		{Group: 2, Sample: 2, Audio: nil},
	}}

	data, err := EncodeV1(set)
	if err != nil {
		t.Fatalf("EncodeV1: %v", err)
	}
	back, err := ReadV1Set(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ReadV1Set: %v", err)
	}
	if len(back.Sounds) != 3 {
		t.Fatalf("got %d sounds, want 3", len(back.Sounds))
	}
	if string(back.Sounds[0].Audio) != "aaaa" || string(back.Sounds[1].Audio) != "bb" || len(back.Sounds[2].Audio) != 0 {
		t.Errorf("audio changed through the round trip: %+v", back.Sounds)
	}
}

func TestEncodeV1_KeepsNegativeAndMaximalKeys(t *testing.T) {
	set := &V1SoundSet{Sounds: []V1Sound{
		{Group: -1, Sample: 2147483647, Audio: []byte("x")},
	}}
	data, err := EncodeV1(set)
	if err != nil {
		t.Fatalf("EncodeV1: %v", err)
	}
	back, err := ReadV1Set(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ReadV1Set: %v", err)
	}
	if back.Sounds[0].Group != -1 || back.Sounds[0].Sample != 2147483647 {
		t.Errorf("key = (%d, %d), want (-1, 2147483647)", back.Sounds[0].Group, back.Sounds[0].Sample)
	}
}

func TestEncodeV1_RejectsWhatTheFormatCannotHold(t *testing.T) {
	tests := []struct {
		name string
		set  *V1SoundSet
		want string
	}{
		{"nil set", nil, "nil"},
		{"group above int32", &V1SoundSet{Sounds: []V1Sound{{Group: 1 << 31, Sample: 0}}}, "group"},
		{"group below int32", &V1SoundSet{Sounds: []V1Sound{{Group: -(1 << 31) - 1, Sample: 0}}}, "group"},
		{"sample above int32", &V1SoundSet{Sounds: []V1Sound{{Group: 1, Sample: 1 << 31}}}, "sample"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeV1(tt.set)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want an error mentioning %q", err, tt.want)
			}
		})
	}
}

func TestReadV1Set_RejectsMalformedInput(t *testing.T) {
	valid := buildRealLayoutV1File([]v1TestEntry{{group: 1, sample: 0, payload: []byte("0123456789")}})

	t.Run("truncated audio", func(t *testing.T) {
		_, err := ReadV1Set(readerAtBytes(valid[:len(valid)-4]))
		if err == nil {
			t.Fatal("want an error for a file cut inside the audio, got nil")
		}
	})
	t.Run("bad signature", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[0] = 'X'
		if _, err := ReadV1Set(readerAtBytes(bad)); err == nil {
			t.Fatal("want an error for a wrong signature, got nil")
		}
	})
	t.Run("first subheader inside the header", func(t *testing.T) {
		bad := buildV1File(t, []v1TestEntry{{group: 1, sample: 0, payload: []byte("abcd")}})
		binary.LittleEndian.PutUint32(bad[20:24], 8)
		if _, err := ReadV1Set(readerAtBytes(bad)); err == nil {
			t.Fatal("want an error when the first subheader overlaps the header, got nil")
		}
	})
}

// TestEncodeV1_CorpusFilesReEncodeByteIdentical checks every real v1 file of
// a local corpus: reading then re-encoding must give back the same bytes.
// The corpus is machine-specific and never vendored, so the directory is
// given through SND_CORPUS_DIR and the test does nothing without it.
func TestEncodeV1_CorpusFilesReEncodeByteIdentical(t *testing.T) {
	dir := os.Getenv("SND_CORPUS_DIR")
	if dir == "" {
		t.Log("SND_CORPUS_DIR not set: no real-file corpus to check")
		return
	}

	checked := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".snd") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		set, err := ReadV1Set(readerAtBytes(data))
		if err != nil {
			return nil // not a v1-shaped file (e.g. a misnamed one)
		}
		got, err := EncodeV1(set)
		if err != nil {
			t.Errorf("%s: EncodeV1: %v", path, err)
			return nil
		}
		if !bytes.Equal(got, data) {
			t.Errorf("%s: re-encoded file differs from the original", path)
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	t.Logf("%d real files re-encoded byte-identical", checked)
}
