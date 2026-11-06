package snd

import (
	"fmt"
	"io"
)

// v2DefaultVersion is the version stamp EncodeV2 writes when the set has
// none: the one DecodeSound recognizes as v2 (first byte 2).
var v2DefaultVersion = [4]byte{2, 0, 0, 0}

// V2Sound is one sound of a V2SoundSet. Exactly one of Audio and
// ExternalPath is set: either the audio is embedded, or the sound points at
// another audio file (Ikemen GO's external-file-reference extension).
type V2Sound struct {
	Group  int
	Sample int
	// Audio is the embedded RIFF/WAVE blob, copied verbatim.
	Audio []byte
	// ExternalPath is the path of the external audio file, as written in
	// the file (printable ASCII, ending in a recognized audio extension).
	// It is stored as a reference, never inlined.
	ExternalPath string
}

// V2SoundSet is a whole .snd v2 file held in memory, in the shape EncodeV2
// writes back out.
type V2SoundSet struct {
	// Version holds the four raw version bytes of the file header. The
	// zero value stands for the default v2 stamp.
	Version [4]byte
	// HeaderPadding is the free-form comment area between the header
	// fields and the first subheader, written verbatim.
	HeaderPadding []byte
	Sounds        []V2Sound
}

// ReadV2Set reads a whole .snd v2 file into memory without decoding any
// audio. A payload is read as an external reference when it looks like one
// (the same rule DecodeV2Sound applies), and as embedded audio otherwise.
// Unlike EncodeV2, it keeps entries as they are, malformed ones included.
func ReadV2Set(r io.ReaderAt) (*V2SoundSet, error) {
	table, err := ParseV2(r)
	if err != nil {
		return nil, err
	}

	entries := make([]containerEntry, len(table.Sounds))
	for i, e := range table.Sounds {
		entries[i] = containerEntry{Group: e.Group, Sample: e.Sample, Offset: e.Offset, Length: e.Length}
	}
	padding, payloads, err := readContainer(r, "v2", entries)
	if err != nil {
		return nil, err
	}

	set := &V2SoundSet{
		Version:       table.Header.Version,
		HeaderPadding: padding,
		Sounds:        make([]V2Sound, len(entries)),
	}
	for i, e := range entries {
		sound := V2Sound{Group: e.Group, Sample: e.Sample}
		if path, ok := externalFileReferencePath(payloads[i]); ok && !isEmbeddedRIFFWAVE(payloads[i]) {
			sound.ExternalPath = path
		} else {
			sound.Audio = payloads[i]
		}
		set.Sounds[i] = sound
	}
	return set, nil
}

// EncodeV2 serialises set as a .snd v2 file, in the same layout as
// EncodeV1. External references are written as their path, not inlined.
// It refuses a set DecodeV2Sound could not read back as written: a
// duplicate (group, sample) key, a sound with no payload or with both
// kinds, embedded audio that is not RIFF/WAVE, an external path that is not
// printable ASCII ending in a recognized audio extension, a version stamp
// that DecodeSound would not read as v2, plus what EncodeV1 refuses.
func EncodeV2(set *V2SoundSet) ([]byte, error) {
	if set == nil {
		return nil, fmt.Errorf("snd: encoding a nil v2 sound set")
	}

	version := set.Version
	if version == ([4]byte{}) {
		version = v2DefaultVersion
	}
	if version[0] != 2 {
		return nil, fmt.Errorf("snd: v2 version stamp %v would be read as v1: its first byte must be 2", version)
	}

	type key struct{ group, sample int }
	seen := make(map[key]int, len(set.Sounds))
	payloads := make([]containerPayload, len(set.Sounds))

	for i, s := range set.Sounds {
		k := key{s.Group, s.Sample}
		if first, dup := seen[k]; dup {
			return nil, fmt.Errorf("snd: v2 sounds %d and %d share the key (group %d, sample %d)", first, i, s.Group, s.Sample)
		}
		seen[k] = i

		payload, err := v2Payload(s)
		if err != nil {
			return nil, fmt.Errorf("snd: v2 sound %d (group %d, sample %d): %w", i, s.Group, s.Sample, err)
		}
		payloads[i] = containerPayload{Group: s.Group, Sample: s.Sample, Payload: payload}
	}

	return encodeContainer(version, set.HeaderPadding, payloads)
}

// v2Payload checks one sound and returns the bytes to store for it.
func v2Payload(s V2Sound) ([]byte, error) {
	switch {
	case len(s.Audio) > 0 && s.ExternalPath != "":
		return nil, fmt.Errorf("has both embedded audio and an external path")
	case len(s.Audio) == 0 && s.ExternalPath == "":
		return nil, fmt.Errorf("has an empty payload: set either Audio or ExternalPath")
	case s.ExternalPath != "":
		if path, ok := externalFileReferencePath([]byte(s.ExternalPath)); !ok || path != s.ExternalPath {
			return nil, fmt.Errorf("external path %q must be printable ASCII ending in a recognized audio extension", s.ExternalPath)
		}
		return []byte(s.ExternalPath), nil
	case !isEmbeddedRIFFWAVE(s.Audio):
		return nil, fmt.Errorf("embedded audio is not a RIFF/WAVE blob")
	}
	return s.Audio, nil
}
