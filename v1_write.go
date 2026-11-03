package snd

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// v1HeaderFieldsSize is the size, in bytes, of the .snd v1 header fields
// this package interprets: signature(12) + version(4) + sound count(4) +
// first subheader offset(4). Anything between this and the first
// subheader is a free-form comment area, kept verbatim as
// V1SoundSet.HeaderPadding.
const v1HeaderFieldsSize = 24

// V1Sound is one sound of a V1SoundSet: its (group, sample) key and its
// embedded audio blob, exactly as stored in the file.
type V1Sound struct {
	Group  int
	Sample int
	// Audio is the embedded RIFF/WAVE blob, copied verbatim. It is neither
	// decoded nor validated here — see DecodeV1Sound.
	Audio []byte
}

// V1SoundSet is a whole .snd v1 file held in memory, in the shape
// EncodeV1 writes back out. Duplicate (group, sample) keys are kept as
// they are: real files can contain them, and a faithful round trip must
// not drop an entry.
type V1SoundSet struct {
	// Version holds the four raw version bytes of the file header.
	Version [4]byte
	// HeaderPadding is the free-form comment area between the header
	// fields and the first subheader (real files use 488 bytes, so the
	// first subheader sits at offset 512). It is written verbatim.
	HeaderPadding []byte
	Sounds        []V1Sound
}

// ReadV1Set reads a whole .snd v1 file into memory without decoding any
// audio. EncodeV1 writes the result back byte-identical when the file
// follows the layout real files have: subheaders laid out one after the
// other, no bytes after the last sound, and the last subheader pointing
// at the end of the file. A file with no sounds yields an empty
// HeaderPadding.
func ReadV1Set(r io.ReaderAt) (*V1SoundSet, error) {
	table, err := ParseV1(r)
	if err != nil {
		return nil, err
	}

	set := &V1SoundSet{
		Version:       table.Header.Version,
		HeaderPadding: []byte{},
		Sounds:        make([]V1Sound, 0, len(table.Sounds)),
	}

	if len(table.Sounds) > 0 {
		first := table.Sounds[0].Offset - v1SoundSubheaderSize
		if first < v1HeaderFieldsSize {
			return nil, fmt.Errorf("snd: v1 first sound subheader at offset %d overlaps the file header", first)
		}
		// ParseV1 already read a subheader at first, so the file is at
		// least that long: the padding allocation is bounded by real data.
		set.HeaderPadding = make([]byte, first-v1HeaderFieldsSize)
		if _, err := r.ReadAt(set.HeaderPadding, v1HeaderFieldsSize); err != nil {
			return nil, fmt.Errorf("snd: reading v1 header padding: %w", err)
		}
	}

	for i, e := range table.Sounds {
		audio, err := io.ReadAll(io.NewSectionReader(r, e.Offset, int64(e.Length)))
		if err != nil {
			return nil, fmt.Errorf("snd: reading v1 sound %d (group %d, sample %d): %w", i, e.Group, e.Sample, err)
		}
		if len(audio) != e.Length {
			return nil, fmt.Errorf("snd: v1 sound %d (group %d, sample %d) is truncated: want %d bytes, file holds %d", i, e.Group, e.Sample, e.Length, len(audio))
		}
		set.Sounds = append(set.Sounds, V1Sound{Group: e.Group, Sample: e.Sample, Audio: audio})
	}

	return set, nil
}

// EncodeV1 serialises set as a .snd v1 file: header, header padding, then
// each sound's subheader and audio one after the other. The last
// subheader points at the end of the file, as in real files. It refuses a
// set the format cannot represent — a group or sample outside the signed
// 32-bit range, or a file larger than the format's 4 GiB offsets allow.
func EncodeV1(set *V1SoundSet) ([]byte, error) {
	if set == nil {
		return nil, fmt.Errorf("snd: encoding a nil v1 sound set")
	}

	total := int64(v1HeaderFieldsSize) + int64(len(set.HeaderPadding))
	for i, s := range set.Sounds {
		if s.Group < math.MinInt32 || s.Group > math.MaxInt32 {
			return nil, fmt.Errorf("snd: v1 sound %d has group %d outside the signed 32-bit range", i, s.Group)
		}
		if s.Sample < math.MinInt32 || s.Sample > math.MaxInt32 {
			return nil, fmt.Errorf("snd: v1 sound %d (group %d) has sample %d outside the signed 32-bit range", i, s.Group, s.Sample)
		}
		total += v1SoundSubheaderSize + int64(len(s.Audio))
		if total > math.MaxUint32 {
			return nil, fmt.Errorf("snd: v1 file would exceed the format's 4 GiB limit at sound %d (group %d, sample %d)", i, s.Group, s.Sample)
		}
	}

	out := make([]byte, 0, total)

	header := make([]byte, v1HeaderFieldsSize)
	copy(header[0:12], v1Signature)
	copy(header[12:16], set.Version[:])
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(set.Sounds)))
	binary.LittleEndian.PutUint32(header[20:24], uint32(v1HeaderFieldsSize+len(set.HeaderPadding)))
	out = append(out, header...)
	out = append(out, set.HeaderPadding...)

	for _, s := range set.Sounds {
		next := int64(len(out)) + v1SoundSubheaderSize + int64(len(s.Audio))
		sub := make([]byte, v1SoundSubheaderSize)
		binary.LittleEndian.PutUint32(sub[0:4], uint32(next))
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(s.Audio)))
		binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(s.Group)))
		binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(s.Sample)))
		out = append(out, sub...)
		out = append(out, s.Audio...)
	}

	return out, nil
}
