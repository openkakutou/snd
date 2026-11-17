package snd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decode_test.go covers DecodeSound, the version-agnostic entry point
// backlog item 003 adds so a caller (in practice, the WASM entrypoint under
// cmd/wasm) doesn't need to know ahead of time whether a .snd file is v1 or
// v2 — mirroring sff.Load's own auto-detection of its file format's
// version. See .vibe/decisions/004-version-agnostic-decode-detects-v2-via-header-version-stamp.md.

func TestDecodeSound_DecodesRealV1File_WithoutCallerSpecifyingVersion(t *testing.T) {
	f := openTestdataFile(t, "v1-basic.snd")
	defer f.Close()

	sound, err := DecodeSound(f, 1, 0, nil)
	if err != nil {
		t.Fatalf("DecodeSound(1, 0): unexpected error: %v", err)
	}
	if sound.SampleRate != 11025 {
		t.Errorf("SampleRate = %d, want 11025", sound.SampleRate)
	}
	if len(sound.PCM) != 2772 {
		t.Fatalf("len(PCM) = %d, want 2772", len(sound.PCM))
	}
}

func TestDecodeSound_DecodesRealV2File_WithoutCallerSpecifyingVersion(t *testing.T) {
	f := openTestdataFile(t, "v2-basic.snd")
	defer f.Close()

	// Sample 143 only round-trips correctly through v2's own 4-byte
	// Group/Sample subheader fields (decision 002) — if DecodeSound
	// mistakenly ran this file through the v1 code path, this lookup
	// would fail to find the entry at all.
	sound, err := DecodeSound(f, 1, 143, nil)
	if err != nil {
		t.Fatalf("DecodeSound(1, 143): unexpected error: %v", err)
	}
	if sound.SampleRate != 8000 {
		t.Errorf("SampleRate = %d, want 8000", sound.SampleRate)
	}
	if len(sound.PCM) == 0 {
		t.Fatal("len(PCM) = 0, want decoded samples")
	}
}

func TestDecodeSound_ResolvesV2ExternalFileReference_WithoutCallerSpecifyingVersion(t *testing.T) {
	f := openTestdataFile(t, "v2-external-ref.snd")
	defer f.Close()

	opener := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join("testdata", "files", path))
	}

	sound, err := DecodeSound(f, 0, 7, opener)
	if err != nil {
		t.Fatalf("DecodeSound(0, 7): unexpected error: %v", err)
	}
	if len(sound.PCM) == 0 {
		t.Fatal("len(PCM) = 0, want decoded samples resolved from the real external file")
	}
}

func TestDecodeSound_ReturnsError_ForUnknownGroupSample(t *testing.T) {
	f := openTestdataFile(t, "v1-basic.snd")
	defer f.Close()

	_, err := DecodeSound(f, 999, 999, nil)
	if err == nil {
		t.Fatal("DecodeSound(999, 999): expected an error, got nil")
	}
}

func TestDecodeSound_ReturnsError_OnBadSignature(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})
	copy(data[0:12], []byte("NotASndFile\x00"))

	_, err := DecodeSound(readerAtBytes(data), 0, 0, nil)
	if err == nil {
		t.Fatal("DecodeSound: expected an error for a bad signature, got nil")
	}
	if !strings.Contains(err.Error(), "not a .snd file") {
		t.Errorf("error %q does not identify an unrecognized file", err.Error())
	}
}

func TestDecodeSound_ReturnsError_WhenHeaderIsTruncated(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})[:8]

	_, err := DecodeSound(readerAtBytes(data), 0, 0, nil)
	if err == nil {
		t.Fatal("DecodeSound: expected an error for a truncated header, got nil")
	}
}

func TestDecodeSound_AutoDetectsV1_ForABuiltV1File(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 2, sample: 5, payload: buildWAV(t, 1, 8000, 8, []byte{128, 130, 132})}})

	sound, err := DecodeSound(readerAtBytes(data), 2, 5, nil)
	if err != nil {
		t.Fatalf("DecodeSound(2, 5): unexpected error: %v", err)
	}
	want := []int16{0, 512, 1024}
	if len(sound.PCM) != len(want) {
		t.Fatalf("len(PCM) = %d, want %d", len(sound.PCM), len(want))
	}
	for i, w := range want {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
}

func TestDecodeSound_AutoDetectsV2_ForABuiltV2File(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 1000, payload: buildWAV(t, 1, 8000, 8, []byte{128, 130})}})

	sound, err := DecodeSound(readerAtBytes(data), 0, 1000, nil)
	if err != nil {
		t.Fatalf("DecodeSound(0, 1000): unexpected error: %v", err)
	}
	if len(sound.PCM) != 2 {
		t.Fatalf("len(PCM) = %d, want 2", len(sound.PCM))
	}
}

var fuzzSeedFiles = []string{"v1-8bit.snd", "v1-basic.snd", "v1-multidigit-sample.snd", "v2-basic.snd", "v2-external-ref.snd"}

func readSeed(tb testing.TB, name string) []byte {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "files", name))
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

// fuzzOpener answers every external reference with a WAV built from the
// fuzz input itself, so the external path reaches the WAV decoder.
func fuzzOpener(data []byte) ExternalAudioOpener {
	return func(string) ([]byte, error) { return data, nil }
}

func TestDecodeSound_NeverPanics_ForEveryTruncationOfFixtures(t *testing.T) {
	wav := readSeed(t, "v2-external-audio.wav")
	for _, name := range fuzzSeedFiles {
		full := readSeed(t, name)
		for n := 0; n <= len(full); n++ {
			prefix := full[:n]
			checkParseAndDecodeAll(t, prefix, wav)
		}
	}
}

// checkParseAndDecodeAll parses data as v1 and v2 and decodes every parsed
// entry, failing if a successful decode read beyond the input.
func checkParseAndDecodeAll(tb testing.TB, data, external []byte) {
	tb.Helper()
	r := bytes.NewReader(data)
	if t1, err := ParseV1(r); err == nil {
		for _, e := range t1.Sounds {
			if _, err := DecodeV1Sound(r, t1, e.Group, e.Sample); err == nil && e.Offset+int64(e.Length) > int64(len(data)) {
				tb.Fatalf("v1 decode succeeded beyond input: %+v, len %d", e, len(data))
			}
		}
	}
	if t2, err := ParseV2(r); err == nil {
		for _, e := range t2.Sounds {
			if _, err := DecodeV2Sound(r, t2, e.Group, e.Sample, fuzzOpener(external)); err == nil && e.Offset+int64(e.Length) > int64(len(data)) {
				tb.Fatalf("v2 decode succeeded beyond input: %+v, len %d", e, len(data))
			}
		}
	}
	if t1, err := ParseV1(r); err == nil && len(t1.Sounds) > 0 {
		_, _ = DecodeSound(r, t1.Sounds[0].Group, t1.Sounds[0].Sample, fuzzOpener(external))
	}
}

func FuzzDecodeSound(f *testing.F) {
	wav := readSeed(f, "v2-external-audio.wav")
	for _, name := range fuzzSeedFiles {
		f.Add(readSeed(f, name), wav)
	}
	f.Add(append(rawSndHeader(2, 1, 24), rawSubheader(0, 0xFFFFFFF0, 1, 1)...), wav)
	f.Add(append(rawSndHeader(1, 1<<30, 24), rawSubheader(24, 0, 1, 1)...), wav)
	// Malformed WAV blobs embedded as a v1 and a v2 entry's audio.
	for _, c := range wavBlobCases() {
		f.Add(append(append(rawSndHeader(1, 1, 24), rawSubheader(0, uint32(len(c.blob)), 1, 1)...), c.blob...), wav)
	}
	f.Fuzz(func(t *testing.T, data, external []byte) {
		checkParseAndDecodeAll(t, data, external)
		// DecodeSound's own version detection, on both versions.
		for _, v := range []byte{1, 2} {
			if len(data) > 12 {
				d := append([]byte(nil), data...)
				d[12] = v
				_, _ = DecodeSound(bytes.NewReader(d), 1, 1, fuzzOpener(external))
			}
		}
	})
}

func TestReadBlob_ReturnsUnexpectedEOF_WhenLengthOrOffsetExceedsData(t *testing.T) {
	data := []byte("0123456789")
	for name, c := range map[string]struct {
		offset int64
		length int
	}{
		"length-longer-than-data": {0, 11},
		"range-straddles-end":     {8, 5},
		"offset-past-end":         {20, 4},
	} {
		blob, err := readBlob(bytes.NewReader(data), c.offset, c.length)
		if !errors.Is(err, io.ErrUnexpectedEOF) || blob != nil {
			t.Errorf("%s: readBlob = (%q, %v); want (nil, io.ErrUnexpectedEOF)", name, blob, err)
		}
	}
	blob, err := readBlob(bytes.NewReader(data), 2, 3)
	if err != nil || string(blob) != "234" {
		t.Errorf("in-range readBlob = (%q, %v); want (\"234\", nil)", blob, err)
	}
}
