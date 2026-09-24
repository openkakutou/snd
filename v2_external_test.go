package snd

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// buildV2FileWithExternalRef builds a minimal v2 .snd file whose single
// (group, sample) entry's payload is path (Ikemen GO's external-file-
// reference extension — see
// .vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md)
// instead of embedded audio.
func buildV2FileWithExternalRef(t *testing.T, group, sample int, path string) []byte {
	t.Helper()
	return buildV2File(t, []v1TestEntry{{group: group, sample: sample, payload: []byte(path)}})
}

func TestDecodeV2Sound_ResolvesAndDecodesExternalFileReference(t *testing.T) {
	samples := []int16{111, -222, 333}
	raw := make([]byte, 6)
	for i, s := range samples {
		raw[i*2] = byte(uint16(s))
		raw[i*2+1] = byte(uint16(s) >> 8)
	}
	wav := buildWAV(t, 1, 22050, 16, raw)

	data := buildV2FileWithExternalRef(t, 5, 2, "extra/voice1.wav")
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	var openedPath string
	opener := func(path string) ([]byte, error) {
		openedPath = path
		return wav, nil
	}

	sound, err := DecodeV2Sound(readerAtBytes(data), table, 5, 2, opener)
	if err != nil {
		t.Fatalf("DecodeV2Sound: unexpected error: %v", err)
	}
	if openedPath != "extra/voice1.wav" {
		t.Fatalf("opener called with path %q, want %q", openedPath, "extra/voice1.wav")
	}
	if sound.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", sound.SampleRate)
	}
	want := []int16{111, -222, 333}
	if len(sound.PCM) != len(want) {
		t.Fatalf("len(PCM) = %d, want %d", len(sound.PCM), len(want))
	}
	for i, w := range want {
		if sound.PCM[i] != w {
			t.Errorf("PCM[%d] = %d, want %d", i, sound.PCM[i], w)
		}
	}
}

func TestDecodeV2Sound_ReturnsError_WhenExternalReferenceHasNoOpener(t *testing.T) {
	data := buildV2FileWithExternalRef(t, 5, 2, "extra/voice1.wav")
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 5, 2, nil)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error when an external reference has no opener, got nil")
	}
	if !strings.Contains(err.Error(), "extra/voice1.wav") {
		t.Errorf("error %q does not name the missing path", err.Error())
	}
}

func TestDecodeV2Sound_ReturnsError_WhenExternalFileNotFound(t *testing.T) {
	data := buildV2FileWithExternalRef(t, 5, 2, "extra/missing.wav")
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	notFound := errors.New("no such file")
	opener := func(path string) ([]byte, error) {
		return nil, fmt.Errorf("open %s: %w", path, notFound)
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 5, 2, opener)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error when the external file cannot be opened, got nil")
	}
	if !errors.Is(err, notFound) {
		t.Errorf("error %v does not wrap the opener's underlying cause", err)
	}
	if !strings.Contains(err.Error(), "extra/missing.wav") {
		t.Errorf("error %q does not name the missing path", err.Error())
	}
}

func TestDecodeV2Sound_ReturnsError_WhenExternalFileIsUnsupportedFormat(t *testing.T) {
	data := buildV2FileWithExternalRef(t, 5, 2, "extra/voice1.ogg")
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	// A real Ogg Vorbis file would not start with "RIFF"/"WAVE" — this
	// package only decodes WAV/PCM (see decision 003), so a genuinely
	// non-WAV external file is a descriptive, named scope cut rather than a
	// silent wrong decode.
	oggLikeBytes := []byte("OggS\x00\x02\x00\x00not a real ogg stream but not RIFF either")
	opener := func(path string) ([]byte, error) {
		return oggLikeBytes, nil
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 5, 2, opener)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for an unsupported external audio format, got nil")
	}
	if !strings.Contains(err.Error(), "extra/voice1.ogg") {
		t.Errorf("error %q does not name the offending path", err.Error())
	}
}

func TestDecodeV2Sound_DoesNotTreatShortEmbeddedGarbageAsExternalPath(t *testing.T) {
	// A short, non-printable blob must never be misclassified as a file
	// path just because it also fails the RIFF/WAVE check.
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte{0x01, 0x02, 0x03}}})
	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: %v", err)
	}

	called := false
	opener := func(path string) ([]byte, error) {
		called = true
		return nil, errors.New("should not be called")
	}

	_, err = DecodeV2Sound(readerAtBytes(data), table, 0, 0, opener)
	if err == nil {
		t.Fatal("DecodeV2Sound: expected an error for corrupt embedded data, got nil")
	}
	if called {
		t.Error("DecodeV2Sound: opener was called for non-path-like garbage bytes")
	}
}
