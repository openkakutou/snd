package snd

import (
	"encoding/binary"
	"testing"
)

// buildV2File assembles a minimal, well-formed v2 .snd file in memory from a
// list of (group, sample, payload) sound entries, chaining each subheader's
// NextSubHeaderOffset the way a real file does. Unlike v1's subheader, a v2
// subheader stores Group and Sample as 4-byte fields (see
// .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md).
func buildV2File(t *testing.T, entries []v1TestEntry) []byte {
	t.Helper()

	const headerSize = 24
	var buf []byte

	header := make([]byte, headerSize)
	copy(header[0:12], []byte(v1Signature))
	header[12], header[13], header[14], header[15] = 2, 0, 0, 0
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(entries)))
	binary.LittleEndian.PutUint32(header[20:24], uint32(headerSize))
	buf = append(buf, header...)

	offsets := make([]int, len(entries))
	pos := headerSize
	for i, e := range entries {
		offsets[i] = pos
		pos += v2SoundSubheaderSize + len(e.payload)
	}

	for i, e := range entries {
		sub := make([]byte, v2SoundSubheaderSize)
		if i+1 < len(entries) {
			binary.LittleEndian.PutUint32(sub[0:4], uint32(offsets[i+1]))
		}
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(e.payload)))
		binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(e.group)))
		binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(e.sample)))
		buf = append(buf, sub...)
		buf = append(buf, e.payload...)
	}

	return buf
}

func TestParseV2_ParsesHeaderAndSoundTable_ForWellFormedFile(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{
		{group: 0, sample: 1000, payload: []byte("aaaa")},
		{group: 1, sample: 13, payload: []byte("bbbbbb")},
	})

	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: unexpected error: %v", err)
	}

	if table.Header.NumberOfSounds != 2 {
		t.Fatalf("NumberOfSounds = %d, want 2", table.Header.NumberOfSounds)
	}
	if len(table.Sounds) != 2 {
		t.Fatalf("len(Sounds) = %d, want 2", len(table.Sounds))
	}

	i, ok := table.Index(1, 13)
	if !ok {
		t.Fatalf("Index(1, 13): not found")
	}
	if table.Sounds[i].Group != 1 || table.Sounds[i].Sample != 13 {
		t.Fatalf("Sounds[%d] = %+v, want group=1 sample=13", i, table.Sounds[i])
	}
	if table.Sounds[i].Length != 6 {
		t.Fatalf("Sounds[%d].Length = %d, want 6", i, table.Sounds[i].Length)
	}

	// This is exactly the case v1's own 2-byte Sample field cannot represent
	// (a Sample value that does not fit alongside Group in a shared 4-byte
	// span) — see decision 002.
	j, ok := table.Index(0, 1000)
	if !ok {
		t.Fatalf("Index(0, 1000): not found")
	}
	if table.Sounds[j].Sample != 1000 {
		t.Fatalf("Sounds[%d].Sample = %d, want 1000", j, table.Sounds[j].Sample)
	}
}

func TestParseV2_IndexReturnsFalse_ForUnknownGroupSample(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})

	table, err := ParseV2(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV2: unexpected error: %v", err)
	}

	if _, ok := table.Index(9, 9); ok {
		t.Fatal("Index(9, 9) = true, want false: no such sound in the table")
	}
}

func TestParseV2_ReturnsError_OnBadSignature(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})
	copy(data[0:12], []byte("NotASndFile\x00"))

	_, err := ParseV2(readerAtBytes(data))
	if err == nil {
		t.Fatal("ParseV2: expected an error for a bad signature, got nil")
	}
}

func TestParseV2_ReturnsError_WhenSoundTableIsTruncated(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{
		{group: 0, sample: 0, payload: []byte("aaaa")},
		{group: 1, sample: 0, payload: []byte("bbbb")},
	})
	// Declare 2 sounds but cut the file off before the second subheader.
	firstSubEnd := 24 + v2SoundSubheaderSize + 4
	truncated := data[:firstSubEnd]

	_, err := ParseV2(readerAtBytes(truncated))
	if err == nil {
		t.Fatal("ParseV2: expected an error for a truncated sound table, got nil")
	}
}

func TestParseV2_ReturnsError_OnNegativeSoundCount(t *testing.T) {
	data := buildV2File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})
	var negativeCount int32 = -1
	binary.LittleEndian.PutUint32(data[16:20], uint32(negativeCount))

	_, err := ParseV2(readerAtBytes(data))
	if err == nil {
		t.Fatal("ParseV2: expected an error for a negative declared sound count, got nil")
	}
}
