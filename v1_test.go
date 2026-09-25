package snd

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// buildV1File assembles a minimal, well-formed v1 .snd file in memory from
// a list of (group, sample, payload) sound entries, chaining each
// subheader's NextSubHeaderOffset the way a real file does. Group and
// Sample are written as adjacent 4-byte fields, matching the real subheader
// layout ParseV1 reads (see
// .vibe/decisions/005-v1-group-sample-fields-are-4-bytes-not-2.md).
func buildV1File(t *testing.T, entries []v1TestEntry) []byte {
	t.Helper()

	const headerSize = 24 // only the fields this package reads; real files pad further with a comment
	var buf bytes.Buffer

	header := make([]byte, headerSize)
	copy(header[0:12], []byte(v1Signature))
	header[12], header[13], header[14], header[15] = 1, 0, 1, 0
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(entries)))
	binary.LittleEndian.PutUint32(header[20:24], uint32(headerSize))
	buf.Write(header)

	offsets := make([]int, len(entries))
	pos := headerSize
	for i, e := range entries {
		offsets[i] = pos
		pos += 16 + len(e.payload)
	}

	for i, e := range entries {
		sub := make([]byte, 16)
		if i+1 < len(entries) {
			binary.LittleEndian.PutUint32(sub[0:4], uint32(offsets[i+1]))
		}
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(e.payload)))
		binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(e.group)))
		binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(e.sample)))
		buf.Write(sub)
		buf.Write(e.payload)
	}

	return buf.Bytes()
}

type v1TestEntry struct {
	group, sample int
	payload       []byte
}

type readerAtBytes []byte

func (r readerAtBytes) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r)) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, io.ErrUnexpectedEOF
	}
	return n, nil
}

func TestParseV1_ParsesHeaderAndSoundTable_ForWellFormedFile(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{
		{group: 0, sample: 0, payload: []byte("aaaa")},
		{group: 1, sample: 2, payload: []byte("bbbbbb")},
	})

	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: unexpected error: %v", err)
	}

	if table.Header.NumberOfSounds != 2 {
		t.Fatalf("NumberOfSounds = %d, want 2", table.Header.NumberOfSounds)
	}
	if len(table.Sounds) != 2 {
		t.Fatalf("len(Sounds) = %d, want 2", len(table.Sounds))
	}

	i, ok := table.Index(1, 2)
	if !ok {
		t.Fatalf("Index(1, 2): not found")
	}
	if table.Sounds[i].Group != 1 || table.Sounds[i].Sample != 2 {
		t.Fatalf("Sounds[%d] = %+v, want group=1 sample=2", i, table.Sounds[i])
	}
	if table.Sounds[i].Length != 6 {
		t.Fatalf("Sounds[%d].Length = %d, want 6", i, table.Sounds[i].Length)
	}
}

func TestParseV1_IndexReturnsFalse_ForUnknownGroupSample(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})

	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1: unexpected error: %v", err)
	}

	if _, ok := table.Index(9, 9); ok {
		t.Fatal("Index(9, 9) = true, want false: no such sound in the table")
	}
}

func TestParseV1_ReturnsError_OnBadSignature(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})
	copy(data[0:12], []byte("NotASndFile\x00"))

	_, err := ParseV1(readerAtBytes(data))
	if err == nil {
		t.Fatal("ParseV1: expected an error for a bad signature, got nil")
	}
}

func TestParseV1_ReturnsError_WhenSoundTableIsTruncated(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{
		{group: 0, sample: 0, payload: []byte("aaaa")},
		{group: 1, sample: 0, payload: []byte("bbbb")},
	})
	// Declare 2 sounds but cut the file off before the second subheader.
	firstSubEnd := 24 + 16 + 4
	truncated := data[:firstSubEnd]

	_, err := ParseV1(readerAtBytes(truncated))
	if err == nil {
		t.Fatal("ParseV1: expected an error for a truncated sound table, got nil")
	}
}
