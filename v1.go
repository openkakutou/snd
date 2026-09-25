package snd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// v1Signature is the fixed 12-byte signature every .snd v1 file starts
	// with.
	v1Signature = "ElecbyteSnd\x00"
	// v1SoundSubheaderSize is the fixed size, in bytes, of one sound
	// entry's subheader, immediately preceding its embedded audio data.
	// Like v2's own subheader, Group and Sample each occupy a full 4
	// bytes with nothing reserved: NextSubHeaderOffset(4) +
	// SubFileLength(4) + Group(4) + Sample(4). This matches Ikemen GO's
	// own reference loader, which applies it regardless of declared
	// version, and was confirmed against real, unmodified character
	// files at corpus scale — see
	// .vibe/decisions/005-v1-group-sample-fields-are-4-bytes-not-2.md.
	v1SoundSubheaderSize = 16
)

// V1Header is the parsed MUGEN/Ikemen GO .snd v1 file header.
type V1Header struct {
	// Version holds the four raw version bytes as stored in the file.
	Version [4]byte
	// NumberOfSounds is the number of sound entries declared in the
	// header.
	NumberOfSounds int
}

// V1SoundEntry is one sound's entry in a .snd v1 sound table: its
// (group, sample) key, and where its embedded audio data lives in the
// file. The audio itself is not decoded here — see DecodeV1Sound.
type V1SoundEntry struct {
	// Group is the sound group index.
	Group int
	// Sample is the sound's sample index within Group.
	Sample int
	// Offset is the absolute file offset of this sound's embedded audio
	// data (a RIFF/WAVE blob), immediately following its own subheader.
	Offset int64
	// Length is the embedded audio data's length in bytes.
	Length int
}

// V1SoundTable is a parsed .snd v1 file header plus its sound table, as
// produced by ParseV1.
type V1SoundTable struct {
	Header V1Header
	Sounds []V1SoundEntry
}

// Index resolves the (group, sample) pair to its position within
// t.Sounds — the index DecodeV1Sound takes. The second return value is
// false if no sound with that (group, sample) exists in the table.
func (t *V1SoundTable) Index(group, sample int) (int, bool) {
	for i, s := range t.Sounds {
		if s.Group == group && s.Sample == sample {
			return i, true
		}
	}
	return 0, false
}

// ParseV1 reads a MUGEN/Ikemen GO .snd v1 file header and its sound table
// from r. Each sound entry's embedded audio data is located (offset and
// length) but not decoded.
func ParseV1(r io.ReaderAt) (*V1SoundTable, error) {
	header := make([]byte, 24)
	if _, err := r.ReadAt(header, 0); err != nil {
		return nil, fmt.Errorf("snd: reading v1 header: %w", err)
	}

	if sig := string(header[0:12]); sig != v1Signature {
		return nil, fmt.Errorf("snd: not a v1 .snd file: unexpected signature %q", sig)
	}

	var h V1Header
	copy(h.Version[:], header[12:16])
	numberOfSounds := int32(binary.LittleEndian.Uint32(header[16:20]))
	nextOffset := int64(binary.LittleEndian.Uint32(header[20:24]))

	if numberOfSounds < 0 {
		return nil, errors.New("snd: v1 header declares a negative sound count")
	}
	h.NumberOfSounds = int(numberOfSounds)

	table := &V1SoundTable{
		Header: h,
		Sounds: make([]V1SoundEntry, 0, h.NumberOfSounds),
	}

	for i := 0; i < h.NumberOfSounds; i++ {
		if nextOffset == 0 {
			return nil, fmt.Errorf("snd: v1 sound table ended after %d of %d declared sounds", i, h.NumberOfSounds)
		}

		sub := make([]byte, v1SoundSubheaderSize)
		if _, err := r.ReadAt(sub, nextOffset); err != nil {
			return nil, fmt.Errorf("snd: reading v1 sound subheader %d: %w", i, err)
		}

		length := binary.LittleEndian.Uint32(sub[4:8])
		entry := V1SoundEntry{
			Group:  int(int32(binary.LittleEndian.Uint32(sub[8:12]))),
			Sample: int(int32(binary.LittleEndian.Uint32(sub[12:16]))),
			Offset: nextOffset + v1SoundSubheaderSize,
			Length: int(length),
		}
		table.Sounds = append(table.Sounds, entry)

		nextOffset = int64(binary.LittleEndian.Uint32(sub[0:4]))
	}

	return table, nil
}
