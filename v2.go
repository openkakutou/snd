package snd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// v2SoundSubheaderSize is the fixed size, in bytes, of one sound entry's
// subheader in a .snd v2 file, immediately preceding its payload (either
// embedded audio or an external-file-reference path — see DecodeV2Sound).
// Unlike v1's subheader, both Group and Sample occupy a full 4 bytes each,
// with no unused/reserved region: NextSubHeaderOffset(4) + SubFileLength(4)
// + Group(4) + Sample(4). This matches Ikemen GO's own reference loader
// (which applies it to every .snd file, regardless of declared version) and
// was independently confirmed against a real, unmodified character file —
// see .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md.
const v2SoundSubheaderSize = 16

// V2Header is the parsed MUGEN/Ikemen GO .snd v2 file header. Its layout is
// identical to V1Header's (same signature, same 24-byte header) — only the
// sound table subheader that follows differs. See V2SoundTable.
type V2Header struct {
	// Version holds the four raw version bytes as stored in the file.
	Version [4]byte
	// NumberOfSounds is the number of sound entries declared in the header.
	NumberOfSounds int
}

// V2SoundEntry is one sound's entry in a .snd v2 sound table: its
// (group, sample) key, and where its payload lives in the file. The payload
// is not read here — see DecodeV2Sound, which also resolves the
// Ikemen GO external-file-reference extension a v2 entry's payload may use
// instead of embedding its audio directly.
type V2SoundEntry struct {
	// Group is the sound group index.
	Group int
	// Sample is the sound's sample index within Group.
	Sample int
	// Offset is the absolute file offset of this sound's payload,
	// immediately following its own subheader.
	Offset int64
	// Length is the payload's length in bytes.
	Length int
}

// V2SoundTable is a parsed .snd v2 file header plus its sound table, as
// produced by ParseV2. Its shape mirrors V1SoundTable's.
type V2SoundTable struct {
	Header V2Header
	Sounds []V2SoundEntry
}

// Index resolves the (group, sample) pair to its position within
// t.Sounds — the index DecodeV2Sound takes. The second return value is
// false if no sound with that (group, sample) exists in the table.
func (t *V2SoundTable) Index(group, sample int) (int, bool) {
	for i, s := range t.Sounds {
		if s.Group == group && s.Sample == sample {
			return i, true
		}
	}
	return 0, false
}

// ParseV2 reads a MUGEN/Ikemen GO .snd v2 file header and its sound table
// from r. Each sound entry's payload is located (offset and length) but not
// read — decoding, including resolving an external-file-reference entry, is
// DecodeV2Sound's job.
func ParseV2(r io.ReaderAt) (*V2SoundTable, error) {
	header := make([]byte, 24)
	if _, err := r.ReadAt(header, 0); err != nil {
		return nil, fmt.Errorf("snd: reading v2 header: %w", err)
	}

	if sig := string(header[0:12]); sig != v1Signature {
		return nil, fmt.Errorf("snd: not a v2 .snd file: unexpected signature %q", sig)
	}

	var h V2Header
	copy(h.Version[:], header[12:16])
	numberOfSounds := int32(binary.LittleEndian.Uint32(header[16:20]))
	nextOffset := int64(binary.LittleEndian.Uint32(header[20:24]))

	if numberOfSounds < 0 {
		return nil, errors.New("snd: v2 header declares a negative sound count")
	}
	h.NumberOfSounds = int(numberOfSounds)

	table := &V2SoundTable{
		Header: h,
		Sounds: make([]V2SoundEntry, 0, h.NumberOfSounds),
	}

	for i := 0; i < h.NumberOfSounds; i++ {
		if nextOffset == 0 {
			return nil, fmt.Errorf("snd: v2 sound table ended after %d of %d declared sounds", i, h.NumberOfSounds)
		}

		sub := make([]byte, v2SoundSubheaderSize)
		if _, err := r.ReadAt(sub, nextOffset); err != nil {
			return nil, fmt.Errorf("snd: reading v2 sound subheader %d: %w", i, err)
		}

		length := binary.LittleEndian.Uint32(sub[4:8])
		entry := V2SoundEntry{
			Group:  int(int32(binary.LittleEndian.Uint32(sub[8:12]))),
			Sample: int(int32(binary.LittleEndian.Uint32(sub[12:16]))),
			Offset: nextOffset + v2SoundSubheaderSize,
			Length: int(length),
		}
		table.Sounds = append(table.Sounds, entry)

		nextOffset = int64(binary.LittleEndian.Uint32(sub[0:4]))
	}

	return table, nil
}
