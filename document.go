package snd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// nextPointer says what a sound's NextSubHeaderOffset pointed at in the
// file it was read from, so Serialize can move it with the layout.
type nextPointer int

const (
	// nextFollows: the pointer leads to the next sound's subheader.
	nextFollows nextPointer = iota
	// nextEnd: the last sound points at the end of the file, as real
	// files do.
	nextEnd
	// nextZero: the last sound's pointer is 0, as in trimmed files.
	nextZero
	// nextLiteral: any other value, kept as read.
	nextLiteral
)

type documentSound struct {
	group, sample int
	payload       []byte
	// gap holds the bytes between this sound's payload and the next
	// sound's subheader (empty in every real file seen so far).
	gap     []byte
	next    nextPointer
	literal uint32
}

// Document is a .snd file (v1 or v2 — they share one layout) held in a
// shape that writes back byte-identical when untouched. Beyond the sounds
// it keeps what a sound-only model drops: the header version bytes and
// comment area, bytes between sounds, bytes after the last sound, and
// where each sound's next-pointer led. Editing one sound's payload moves
// only the offsets that must move.
type Document struct {
	version [4]byte
	// padding is the comment area between the header fields and the
	// first subheader.
	padding []byte
	sounds  []documentSound
	// trailing holds every byte after the last sound's payload (or after
	// the 24 header bytes when there is no sound).
	trailing []byte
	// firstLiteral is the header's first-subheader offset, kept as read
	// when there is no sound for it to lead to.
	firstLiteral uint32
}

// DocumentSound is a copy of one sound of a Document.
type DocumentSound struct {
	Group   int
	Sample  int
	Payload []byte
}

// ParseDocument reads a whole .snd file. It refuses a file whose sounds
// are not laid out in increasing order of offset, or whose payload runs
// past the end of data: neither occurs in real files, and writing them
// back unchanged would need a different layout model.
func ParseDocument(data []byte) (*Document, error) {
	table, err := ParseV1(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	doc := &Document{
		version:      table.Header.Version,
		padding:      []byte{},
		firstLiteral: binary.LittleEndian.Uint32(data[20:24]),
	}

	if len(table.Sounds) == 0 {
		doc.trailing = append([]byte{}, data[v1HeaderFieldsSize:]...)
		return doc, nil
	}

	first := table.Sounds[0].Offset - v1SoundSubheaderSize
	if first < v1HeaderFieldsSize {
		return nil, fmt.Errorf("snd: first sound subheader at offset %d overlaps the file header", first)
	}
	doc.padding = append([]byte{}, data[v1HeaderFieldsSize:first]...)

	for i, e := range table.Sounds {
		end := e.Offset + int64(e.Length)
		if end > int64(len(data)) {
			return nil, fmt.Errorf("snd: sound %d (group %d, sample %d) is truncated: its audio ends at %d but the file holds %d bytes", i, e.Group, e.Sample, end, len(data))
		}

		s := documentSound{group: e.Group, sample: e.Sample, payload: append([]byte{}, data[e.Offset:end]...)}

		if i+1 < len(table.Sounds) {
			nextStart := table.Sounds[i+1].Offset - v1SoundSubheaderSize
			if nextStart < end {
				return nil, fmt.Errorf("snd: sound %d (group %d, sample %d) is followed by a sound at offset %d, before its own end %d: unsupported layout", i, e.Group, e.Sample, nextStart, end)
			}
			s.gap = append([]byte{}, data[end:nextStart]...)
			s.next = nextFollows
		} else {
			doc.trailing = append([]byte{}, data[end:]...)
			s.literal = binary.LittleEndian.Uint32(data[e.Offset-v1SoundSubheaderSize : e.Offset-v1SoundSubheaderSize+4])
			switch {
			case int64(s.literal) == int64(len(data)):
				s.next = nextEnd
			case s.literal == 0:
				s.next = nextZero
			default:
				s.next = nextLiteral
			}
		}
		doc.sounds = append(doc.sounds, s)
	}

	return doc, nil
}

// Len returns the number of sounds.
func (d *Document) Len() int { return len(d.sounds) }

// Sound returns a copy of the i-th sound, in file order.
func (d *Document) Sound(i int) (DocumentSound, error) {
	if i < 0 || i >= len(d.sounds) {
		return DocumentSound{}, fmt.Errorf("snd: sound index %d out of range [0, %d)", i, len(d.sounds))
	}
	s := d.sounds[i]
	return DocumentSound{Group: s.group, Sample: s.sample, Payload: append([]byte{}, s.payload...)}, nil
}

// SetPayload replaces the i-th sound's payload. Nothing else of the
// document changes. It does not check that payload is audio — the
// editing API validates what it hands in.
func (d *Document) SetPayload(i int, payload []byte) error {
	if i < 0 || i >= len(d.sounds) {
		return fmt.Errorf("snd: sound index %d out of range [0, %d)", i, len(d.sounds))
	}
	d.sounds[i].payload = append([]byte{}, payload...)
	return nil
}

// Serialize writes the document out. An untouched document gives back the
// bytes ParseDocument read. After SetPayload, only the edited sound's
// payload and length, and the pointers that have to follow, differ.
func (d *Document) Serialize() ([]byte, error) {
	total := int64(v1HeaderFieldsSize) + int64(len(d.padding)) + int64(len(d.trailing))
	for _, s := range d.sounds {
		total += v1SoundSubheaderSize + int64(len(s.payload)) + int64(len(s.gap))
	}
	if total > math.MaxUint32 {
		return nil, fmt.Errorf("snd: file would exceed the format's 4 GiB limit")
	}

	out := make([]byte, 0, total)

	header := make([]byte, v1HeaderFieldsSize)
	copy(header[0:12], v1Signature)
	copy(header[12:16], d.version[:])
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(d.sounds)))
	first := d.firstLiteral
	if len(d.sounds) > 0 {
		first = uint32(v1HeaderFieldsSize + len(d.padding))
	}
	binary.LittleEndian.PutUint32(header[20:24], first)
	out = append(out, header...)
	out = append(out, d.padding...)

	for _, s := range d.sounds {
		end := int64(len(out)) + v1SoundSubheaderSize + int64(len(s.payload))
		var next uint32
		switch s.next {
		case nextFollows:
			next = uint32(end + int64(len(s.gap)))
		case nextEnd:
			next = uint32(total)
		case nextZero:
			next = 0
		default:
			next = s.literal
		}

		sub := make([]byte, v1SoundSubheaderSize)
		binary.LittleEndian.PutUint32(sub[0:4], next)
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(s.payload)))
		binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(s.group)))
		binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(s.sample)))
		out = append(out, sub...)
		out = append(out, s.payload...)
		out = append(out, s.gap...)
	}
	out = append(out, d.trailing...)

	return out, nil
}
