package snd

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixture-driven v1 sound test suite (backlog item 001): decodes real,
// trimmed .snd v1 fixtures under testdata/files (see testdata/README.md)
// and compares the result against expected values independently derived
// with Python's standard-library "wave" module reading the exact same
// embedded audio bytes — not with this package's own decoder — so a
// subtly wrong Go decoder cannot happen to agree with itself.

func openTestdataFile(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "files", name))
	if err != nil {
		t.Fatalf("opening testdata file %s: %v", name, err)
	}
	return f
}

func TestV1Fixtures_Decodes16BitMonoVoiceClip_FromRealCharacterFile(t *testing.T) {
	f := openTestdataFile(t, "v1-basic.snd")
	defer f.Close()

	table, err := ParseV1(f)
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}
	if table.Header.NumberOfSounds != 1 {
		t.Fatalf("NumberOfSounds = %d, want 1", table.Header.NumberOfSounds)
	}

	sound, err := DecodeV1Sound(f, table, 1, 0)
	if err != nil {
		t.Fatalf("DecodeV1Sound(1, 0): %v", err)
	}

	if sound.Channels != 1 {
		t.Errorf("Channels = %d, want 1", sound.Channels)
	}
	if sound.SampleRate != 11025 {
		t.Errorf("SampleRate = %d, want 11025", sound.SampleRate)
	}
	if sound.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", sound.BitsPerSample)
	}
	if len(sound.PCM) != 2772 {
		t.Fatalf("len(PCM) = %d, want 2772", len(sound.PCM))
	}

	wantFirst := []int16{0, 0, 0, 0, 0}
	for i, w := range wantFirst {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
	wantLast := []int16{-15, -14, -13, -12, -11}
	base := len(sound.PCM) - len(wantLast)
	for i, w := range wantLast {
		if sound.PCM[base+i] != w {
			t.Errorf("PCM[%d] = %d, want %d", base+i, sound.PCM[base+i], w)
		}
	}
}

func TestV1Fixtures_Decodes8BitMonoClip_FromRealCharacterFile(t *testing.T) {
	f := openTestdataFile(t, "v1-8bit.snd")
	defer f.Close()

	table, err := ParseV1(f)
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}

	sound, err := DecodeV1Sound(f, table, 40, 0)
	if err != nil {
		t.Fatalf("DecodeV1Sound(40, 0): %v", err)
	}

	if sound.Channels != 1 {
		t.Errorf("Channels = %d, want 1", sound.Channels)
	}
	if sound.SampleRate != 11025 {
		t.Errorf("SampleRate = %d, want 11025", sound.SampleRate)
	}
	if sound.BitsPerSample != 8 {
		t.Errorf("BitsPerSample = %d, want 8", sound.BitsPerSample)
	}
	if len(sound.PCM) != 621 {
		t.Fatalf("len(PCM) = %d, want 621", len(sound.PCM))
	}

	// Raw source bytes (independently read via Python's wave module):
	// first 5 = [128,128,128,128,129], last 5 all 128. Centered at 0 and
	// scaled to 16-bit: 128 -> 0, 129 -> 256.
	wantFirst := []int16{0, 0, 0, 0, 256}
	for i, w := range wantFirst {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
	wantLast := []int16{0, 0, 0, 0, 0}
	base := len(sound.PCM) - len(wantLast)
	for i, w := range wantLast {
		if sound.PCM[base+i] != w {
			t.Errorf("PCM[%d] = %d, want %d", base+i, sound.PCM[base+i], w)
		}
	}
}

func TestV1Fixtures_ReadsMultiDigitSample_FromRealCharacterFile(t *testing.T) {
	// Backlog item 004: a real v1 file's subheader stores Group and Sample
	// as adjacent 4-byte fields, not 2-byte-plus-4-reserved as ParseV1
	// previously assumed. Sample 143 does not fit in a 2-byte field read
	// alongside a small Group, so a wrong reading collides it onto
	// (group 1, sample 0) instead. See
	// .vibe/decisions/005-v1-group-sample-fields-are-4-bytes-not-2.md.
	f := openTestdataFile(t, "v1-multidigit-sample.snd")
	defer f.Close()

	table, err := ParseV1(f)
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}
	if len(table.Sounds) != 1 {
		t.Fatalf("len(Sounds) = %d, want 1", len(table.Sounds))
	}

	entry := table.Sounds[0]
	if entry.Group != 1 || entry.Sample != 143 {
		t.Fatalf("Sounds[0] = (group %d, sample %d), want (group 1, sample 143)", entry.Group, entry.Sample)
	}

	if _, ok := table.Index(1, 143); !ok {
		t.Fatal("Index(1, 143): not found")
	}

	sound, err := DecodeV1Sound(f, table, 1, 143)
	if err != nil {
		t.Fatalf("DecodeV1Sound(1, 143): %v", err)
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
		t.Fatal("len(PCM) = 0, want > 0")
	}
}
