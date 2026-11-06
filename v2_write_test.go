package snd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "files", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

func TestEncodeV2_EmbeddedAndExternalSoundsReadBackIdentically(t *testing.T) {
	wav := buildWAV(t, 1, 8000, 8, []byte{0x80, 0xff, 0x00})
	set := &V2SoundSet{
		HeaderPadding: []byte("a comment"),
		Sounds: []V2Sound{
			{Group: 1, Sample: 0, Audio: wav},
			{Group: 2, Sample: 3, ExternalPath: "sfx/hit.wav"},
		},
	}

	data, err := EncodeV2(set)
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	back, err := ReadV2Set(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ReadV2Set: %v", err)
	}

	if back.Version != [4]byte{2, 0, 0, 0} {
		t.Errorf("Version = %v, want the default v2 stamp [2 0 0 0]", back.Version)
	}
	if string(back.HeaderPadding) != "a comment" {
		t.Errorf("HeaderPadding = %q, want %q", back.HeaderPadding, "a comment")
	}
	if len(back.Sounds) != 2 {
		t.Fatalf("got %d sounds, want 2", len(back.Sounds))
	}
	if b := back.Sounds[0]; b.Group != 1 || b.Sample != 0 || !bytes.Equal(b.Audio, wav) || b.ExternalPath != "" {
		t.Errorf("embedded sound changed: %+v", b)
	}
	if b := back.Sounds[1]; b.Group != 2 || b.Sample != 3 || b.ExternalPath != "sfx/hit.wav" || len(b.Audio) != 0 {
		t.Errorf("external sound was not kept as a reference: %+v", b)
	}
}

func TestEncodeV2_ExternalReferenceIsStoredAsPathNotInlined(t *testing.T) {
	data, err := EncodeV2(&V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 1, ExternalPath: "voice.wav"}}})
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}
	if got := table.Sounds[0].Length; got != len("voice.wav") {
		t.Errorf("payload length = %d, want %d (the path only)", got, len("voice.wav"))
	}
}

func TestEncodeV2_OutputIsReadByDecodeSoundAsV2WithExternalFile(t *testing.T) {
	external := buildWAV(t, 1, 11025, 16, []byte{1, 0, 2, 0})
	data, err := EncodeV2(&V2SoundSet{Sounds: []V2Sound{
		{Group: 1, Sample: 0, Audio: buildWAV(t, 1, 8000, 8, []byte{0x80, 0xff})},
		{Group: 1, Sample: 1, ExternalPath: "ext.wav"},
	}})
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}

	open := func(path string) ([]byte, error) {
		if path != "ext.wav" {
			t.Errorf("opener asked for %q, want ext.wav", path)
		}
		return external, nil
	}

	embedded, err := DecodeSound(readerAtBytes(data), 1, 0, open)
	if err != nil {
		t.Fatalf("DecodeSound(1, 0): %v", err)
	}
	if embedded.SampleRate != 8000 || len(embedded.PCM) != 2 {
		t.Errorf("embedded sound decoded wrongly: %+v", embedded)
	}
	ext, err := DecodeSound(readerAtBytes(data), 1, 1, open)
	if err != nil {
		t.Fatalf("DecodeSound(1, 1): %v", err)
	}
	if ext.SampleRate != 11025 || len(ext.PCM) != 2 {
		t.Errorf("external sound decoded wrongly: %+v", ext)
	}
}

func TestEncodeV2_EmptySetIsValid(t *testing.T) {
	data, err := EncodeV2(&V2SoundSet{})
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	back, err := ReadV2Set(readerAtBytes(data))
	if err != nil || len(back.Sounds) != 0 {
		t.Fatalf("ReadV2Set = %+v, %v; want an empty set", back, err)
	}
}

func TestEncodeV2_KeepsAnExplicitV2Stamp(t *testing.T) {
	data, err := EncodeV2(&V2SoundSet{Version: [4]byte{2, 0, 1, 0}})
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	back, _ := ReadV2Set(readerAtBytes(data))
	if back.Version != [4]byte{2, 0, 1, 0} {
		t.Errorf("Version = %v, want [2 0 1 0]", back.Version)
	}
}

func TestEncodeV2_RejectsInvalidSets(t *testing.T) {
	wav := buildWAV(t, 1, 8000, 8, []byte{0x80})
	tests := []struct {
		name string
		set  *V2SoundSet
		want string
	}{
		{"nil set", nil, "nil"},
		{"duplicate key", &V2SoundSet{Sounds: []V2Sound{
			{Group: 1, Sample: 2, Audio: wav}, {Group: 1, Sample: 2, ExternalPath: "a.wav"},
		}}, "share the key (group 1, sample 2)"},
		{"empty payload", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2}}}, "empty payload"},
		{"both kinds", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2, Audio: wav, ExternalPath: "a.wav"}}}, "both"},
		{"audio not RIFF", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2, Audio: []byte("junk junk junk junk")}}}, "RIFF/WAVE"},
		{"path without audio extension", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2, ExternalPath: "notes.txt"}}}, "recognized audio extension"},
		{"path with control character", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2, ExternalPath: "a\n.wav"}}}, "printable ASCII"},
		{"path with trailing NUL", &V2SoundSet{Sounds: []V2Sound{{Group: 1, Sample: 2, ExternalPath: "a.wav\x00"}}}, "printable ASCII"},
		{"v1 stamp", &V2SoundSet{Version: [4]byte{1, 0, 1, 0}}, "first byte must be 2"},
		{"group out of range", &V2SoundSet{Sounds: []V2Sound{{Group: 1 << 31, Sample: 0, Audio: wav}}}, "group"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeV2(tt.set)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want an error mentioning %q", err, tt.want)
			}
		})
	}
}

func TestReadV2Set_ReadsRealFixturesAndReEncodesThemReadably(t *testing.T) {
	for _, name := range []string{"v2-basic.snd", "v2-external-ref.snd"} {
		t.Run(name, func(t *testing.T) {
			data := readFixture(t, name)
			set, err := ReadV2Set(readerAtBytes(data))
			if err != nil {
				t.Fatalf("ReadV2Set: %v", err)
			}
			encoded, err := EncodeV2(set)
			if err != nil {
				t.Fatalf("EncodeV2: %v", err)
			}
			back, err := ReadV2Set(readerAtBytes(encoded))
			if err != nil {
				t.Fatalf("ReadV2Set of the re-encoded file: %v", err)
			}
			if len(back.Sounds) != len(set.Sounds) {
				t.Fatalf("got %d sounds, want %d", len(back.Sounds), len(set.Sounds))
			}
			for i, s := range set.Sounds {
				b := back.Sounds[i]
				if b.Group != s.Group || b.Sample != s.Sample || b.ExternalPath != s.ExternalPath || !bytes.Equal(b.Audio, s.Audio) {
					t.Errorf("sound %d changed through the round trip", i)
				}
			}
		})
	}
}

func TestReadV2Set_RejectsTruncatedFile(t *testing.T) {
	data := readFixture(t, "v2-basic.snd")
	if _, err := ReadV2Set(readerAtBytes(data[:len(data)-5])); err == nil {
		t.Fatal("want an error for a file cut inside a payload, got nil")
	}
}
