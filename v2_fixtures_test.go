package snd

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixture-driven v2 sound test suite (backlog item 002): decodes real,
// trimmed .snd v2 fixtures under testdata/files (see testdata/README.md).
//
// v2-basic.snd carries real, unmodified embedded audio and real sound-table
// bytes located via v2's own (4-byte Group/Sample) subheader layout — only
// its header's version stamp is synthesized, since no file in the available
// corpus declares itself version 2 (see
// .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md).
//
// v2-external-ref.snd is a clearly-marked synthetic fixture for Ikemen GO's
// external-file-reference extension (no real character using it was found —
// see .vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md);
// the audio it resolves to (v2-external-audio.wav) is nonetheless real,
// unmodified sample data from the same corpus as every other fixture here.

func TestV2Fixtures_DecodesRealClip_UsingV2s4ByteGroupSampleFields(t *testing.T) {
	f := openTestdataFile(t, "v2-basic.snd")
	defer f.Close()

	table, err := ParseV2(f)
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}
	if table.Header.NumberOfSounds != 1 {
		t.Fatalf("NumberOfSounds = %d, want 1", table.Header.NumberOfSounds)
	}

	// Sample 143 does not fit alongside Group in v1's 2-byte field — this is
	// exactly the real-world case decision 002 is about.
	sound, err := DecodeV2Sound(f, table, 1, 143, nil)
	if err != nil {
		t.Fatalf("DecodeV2Sound(1, 143): %v", err)
	}

	if sound.Channels != 1 {
		t.Errorf("Channels = %d, want 1", sound.Channels)
	}
	if sound.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", sound.SampleRate)
	}
	if sound.BitsPerSample != 8 {
		t.Errorf("BitsPerSample = %d, want 8", sound.BitsPerSample)
	}
	if len(sound.PCM) == 0 {
		t.Fatal("len(PCM) = 0, want decoded samples")
	}
}

func TestV2Fixtures_ResolvesAndDecodesRealExternalAudioFile(t *testing.T) {
	f := openTestdataFile(t, "v2-external-ref.snd")
	defer f.Close()

	table, err := ParseV2(f)
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	opener := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join("testdata", "files", path))
	}

	sound, err := DecodeV2Sound(f, table, 0, 7, opener)
	if err != nil {
		t.Fatalf("DecodeV2Sound(0, 7): %v", err)
	}

	if sound.Channels != 1 {
		t.Errorf("Channels = %d, want 1", sound.Channels)
	}
	if sound.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", sound.SampleRate)
	}
	if len(sound.PCM) == 0 {
		t.Fatal("len(PCM) = 0, want decoded samples resolved from the real external file")
	}
}

func TestV2Fixtures_ReturnsError_WhenExternalFileGenuinelyMissing(t *testing.T) {
	f := openTestdataFile(t, "v2-external-ref.snd")
	defer f.Close()

	table, err := ParseV2(f)
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	opener := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join("testdata", "files", "does-not-exist.wav"))
	}

	_, err = DecodeV2Sound(f, table, 0, 7, opener)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for a genuinely missing external file, got nil")
	}
}
