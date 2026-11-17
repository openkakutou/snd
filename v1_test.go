package snd

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"strings"
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

// rawSndHeader builds the 24-byte .snd header (signature, version stamp,
// declared sound count, first subheader offset) without any sound table.
func rawSndHeader(version byte, count, next uint32) []byte {
	h := make([]byte, 24)
	copy(h, v1Signature)
	h[12] = version
	binary.LittleEndian.PutUint32(h[16:], count)
	binary.LittleEndian.PutUint32(h[20:], next)
	return h
}

// rawSubheader builds one 16-byte sound subheader.
func rawSubheader(next, length uint32, group, sample int32) []byte {
	s := make([]byte, 16)
	binary.LittleEndian.PutUint32(s[0:], next)
	binary.LittleEndian.PutUint32(s[4:], length)
	binary.LittleEndian.PutUint32(s[8:], uint32(group))
	binary.LittleEndian.PutUint32(s[12:], uint32(sample))
	return s
}

// allocatedBytes reports how many bytes f allocated in total.
func allocatedBytes(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// hostileTables returns inputs whose headers lie, for both versions: a huge
// declared count over a one-entry table, a self-pointing subheader and an
// A->B->A loop.
func hostileTables(version byte) map[string][]byte {
	huge := append(rawSndHeader(version, 1<<24, 24), rawSubheader(0, 0, 1, 1)...)
	self := append(rawSndHeader(version, 5, 24), rawSubheader(24, 0, 1, 1)...)
	loop := append(rawSndHeader(version, 5, 24), rawSubheader(40, 0, 1, 1)...)
	loop = append(loop, rawSubheader(24, 0, 1, 2)...)
	return map[string][]byte{"huge-count": huge, "self-pointer": self, "two-cycle": loop}
}

func TestParseV1_ReturnsError_OnNegativeSoundCount(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{{group: 0, sample: 0, payload: []byte("x")}})
	binary.LittleEndian.PutUint32(data[16:20], 0xFFFFFFFF) // int32 -1

	_, err := ParseV1(readerAtBytes(data))
	if err == nil || !strings.Contains(err.Error(), "negative sound count") {
		t.Fatalf("ParseV1: err = %v; want a negative sound count error", err)
	}
}

func TestParseV1_AllocatesLittle_WhenHeaderDeclaresHugeSoundCount(t *testing.T) {
	data := hostileTables(1)["huge-count"]
	var err error
	got := allocatedBytes(func() { _, err = ParseV1(bytes.NewReader(data)) })
	if err == nil {
		t.Fatal("ParseV1 returned nil error for a table ending after 1 of 16777216 sounds")
	}
	if !strings.Contains(err.Error(), "1 of 16777216") {
		t.Errorf("err = %v; want it to name the declared count (1 of 16777216)", err)
	}
	if got > 1<<20 {
		t.Errorf("ParseV1 allocated %d bytes for a %d-byte file; want <= 1 MiB", got, len(data))
	}
}

func TestParseV1_ReturnsErrorNamingOffset_WhenNextPointerCycles(t *testing.T) {
	for _, name := range []string{"self-pointer", "two-cycle"} {
		_, err := ParseV1(bytes.NewReader(hostileTables(1)[name]))
		if err == nil || !strings.Contains(err.Error(), "cycle") || !strings.Contains(err.Error(), "offset 24") {
			t.Errorf("%s: err = %v; want a cycle error naming offset 24", name, err)
		}
	}
}

func TestParseV1_AcceptsBackwardPointer_WhenChainDoesNotRepeat(t *testing.T) {
	// Second subheader sits before the first: pointer direction is not
	// constrained, only revisiting an offset is.
	data := append(rawSndHeader(1, 2, 40), rawSubheader(0, 0, 1, 2)...)
	data = append(data, rawSubheader(24, 0, 1, 1)...)
	table, err := ParseV1(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseV1: %v", err)
	}
	if len(table.Sounds) != 2 || table.Sounds[0].Sample != 1 || table.Sounds[1].Sample != 2 {
		t.Errorf("Sounds = %+v; want samples 1 then 2", table.Sounds)
	}
}

func FuzzParseV1(f *testing.F) {
	for _, name := range []string{"v1-8bit.snd", "v1-basic.snd", "v1-multidigit-sample.snd"} {
		f.Add(readSeed(f, name))
	}
	for _, d := range hostileTables(1) {
		f.Add(d)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		table, err := ParseV1(bytes.NewReader(data))
		if err != nil {
			return
		}
		if len(table.Sounds) != table.Header.NumberOfSounds {
			t.Fatalf("parsed %d sounds, header declares %d", len(table.Sounds), table.Header.NumberOfSounds)
		}
		for _, e := range table.Sounds {
			if e.Offset > int64(len(data)) {
				t.Fatalf("entry %+v starts beyond the %d-byte input", e, len(data))
			}
		}
	})
}
