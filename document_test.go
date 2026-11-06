package snd

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serializeDoc(t *testing.T, d *Document) []byte {
	t.Helper()
	out, err := d.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	return out
}

func TestDocument_UntouchedFixturesRoundTripByteIdentical(t *testing.T) {
	for _, name := range []string{
		"v1-basic.snd", "v1-8bit.snd", "v1-multidigit-sample.snd",
		"v2-basic.snd", "v2-external-ref.snd",
	} {
		t.Run(name, func(t *testing.T) {
			data := readFixture(t, name)
			doc, err := ParseDocument(data)
			if err != nil {
				t.Fatalf("ParseDocument: %v", err)
			}
			if got := serializeDoc(t, doc); !bytes.Equal(got, data) {
				t.Fatalf("serialized file differs from the original (%d vs %d bytes)", len(got), len(data))
			}
		})
	}
}

func TestDocument_KeepsWhatASoundOnlyModelDrops(t *testing.T) {
	// Real layout, plus a gap between two sounds, bytes after the last
	// sound, and a version stamp and comment that must not be touched.
	base := buildRealLayoutV1File([]v1TestEntry{
		{group: 1, sample: 0, payload: []byte("first")},
		{group: 1, sample: 1, payload: []byte("second")},
	})
	secondStart := 512 + 16 + len("first")
	gap := []byte("GAP!")
	withGap := append(append(append([]byte{}, base[:secondStart]...), gap...), base[secondStart:]...)
	binary.LittleEndian.PutUint32(withGap[512:516], uint32(secondStart+len(gap)))
	trailer := []byte("trailing junk")
	data := append(append([]byte{}, withGap...), trailer...)
	// the last pointer still says "end of the sounds", not "end of file"
	lastSub := secondStart + len(gap)
	binary.LittleEndian.PutUint32(data[lastSub:lastSub+4], uint32(len(withGap)))

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if got := serializeDoc(t, doc); !bytes.Equal(got, data) {
		t.Fatalf("gap, trailing bytes or pointer lost: serialized %d bytes, original %d", len(got), len(data))
	}
}

func TestDocument_KeepsEachKindOfLastPointer(t *testing.T) {
	base := buildV1File(t, []v1TestEntry{{group: 1, sample: 0, payload: []byte("abcd")}}) // last pointer 0
	for name, value := range map[string]uint32{
		"zero":    0,
		"end":     uint32(len(base)),
		"literal": 77,
	} {
		t.Run(name, func(t *testing.T) {
			data := append([]byte{}, base...)
			binary.LittleEndian.PutUint32(data[24:28], value)
			doc, err := ParseDocument(data)
			if err != nil {
				t.Fatalf("ParseDocument: %v", err)
			}
			if got := serializeDoc(t, doc); !bytes.Equal(got, data) {
				t.Fatalf("last pointer %d not kept", value)
			}
		})
	}
}

func TestDocument_FileWithoutSoundsKeepsEveryByte(t *testing.T) {
	data := make([]byte, 40)
	copy(data[0:12], v1Signature)
	data[12] = 1
	binary.LittleEndian.PutUint32(data[20:24], 99)
	copy(data[24:], "left over bytes")

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if doc.Len() != 0 {
		t.Fatalf("Len = %d, want 0", doc.Len())
	}
	if got := serializeDoc(t, doc); !bytes.Equal(got, data) {
		t.Fatal("bytes of an empty file not kept")
	}
}

func TestDocument_EditingOneSoundChangesOnlyThatSoundAndTheOffsetsAfterIt(t *testing.T) {
	original := buildRealLayoutV1File([]v1TestEntry{
		{group: 1, sample: 0, payload: []byte("aaaa")},
		{group: 1, sample: 1, payload: []byte("bbbbbb")},
		{group: 1, sample: 2, payload: []byte("cc")},
	})
	doc, err := ParseDocument(original)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}

	if err := doc.setPayload(1, []byte("B")); err != nil { // shrinks by 5 bytes
		t.Fatalf("SetPayload: %v", err)
	}
	edited := serializeDoc(t, doc)

	if len(edited) != len(original)-5 {
		t.Fatalf("len = %d, want %d", len(edited), len(original)-5)
	}
	firstEnd := 512 + 16 + 4
	if !bytes.Equal(edited[:firstEnd-4], original[:firstEnd-4]) {
		t.Error("header, comment or first subheader changed")
	}
	if !bytes.Equal(edited[firstEnd-4:firstEnd], original[firstEnd-4:firstEnd]) {
		t.Error("first sound's payload changed")
	}
	if !bytes.HasSuffix(edited, []byte("cc")) {
		t.Error("third sound's payload changed")
	}

	back, err := ParseDocument(edited)
	if err != nil {
		t.Fatalf("ParseDocument of the edited file: %v", err)
	}
	want := []string{"aaaa", "B", "cc"}
	for i, w := range want {
		s, _ := back.Sound(i)
		if string(s.Payload) != w || s.Group != 1 || s.Sample != i {
			t.Errorf("sound %d = (%d, %d, %q), want (1, %d, %q)", i, s.Group, s.Sample, s.Payload, i, w)
		}
	}
	// the last pointer still points at the end of the (shorter) file
	lastSub := len(edited) - 2 - 16
	if got := binary.LittleEndian.Uint32(edited[lastSub : lastSub+4]); int(got) != len(edited) {
		t.Errorf("last pointer = %d, want end of file %d", got, len(edited))
	}
}

func TestDocument_GrowingAPayloadKeepsGapsAndTrailingBytes(t *testing.T) {
	data := buildV1File(t, []v1TestEntry{
		{group: 1, sample: 0, payload: []byte("aa")},
		{group: 2, sample: 0, payload: []byte("bb")},
	})
	data = append(data, "tail"...)
	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if err := doc.setPayload(0, []byte("aaaaaaaa")); err != nil {
		t.Fatalf("SetPayload: %v", err)
	}
	out := serializeDoc(t, doc)
	if !bytes.HasSuffix(out, []byte("bbtail")) {
		t.Errorf("tail = %q, want it to end with bbtail", out[len(out)-8:])
	}
	back, err := ParseDocument(out)
	if err != nil {
		t.Fatalf("ParseDocument of the edited file: %v", err)
	}
	if s, _ := back.Sound(1); string(s.Payload) != "bb" || s.Group != 2 {
		t.Errorf("second sound = %+v", s)
	}
}

func TestDocument_SoundAndSetPayloadRejectBadIndexAndDoNotAlias(t *testing.T) {
	doc, err := ParseDocument(readFixture(t, "v1-basic.snd"))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	for _, i := range []int{-1, 1} {
		if _, err := doc.Sound(i); err == nil {
			t.Errorf("Sound(%d): want an error", i)
		}
		if err := doc.setPayload(i, nil); err == nil {
			t.Errorf("SetPayload(%d): want an error", i)
		}
	}

	s, _ := doc.Sound(0)
	s.Payload[0] ^= 0xff
	again, _ := doc.Sound(0)
	if again.Payload[0] == s.Payload[0] {
		t.Error("Sound returned the document's own bytes: editing the copy changed the document")
	}
	payload := []byte("xy")
	_ = doc.setPayload(0, payload)
	payload[0] = 'Z'
	if got, _ := doc.Sound(0); string(got.Payload) != "xy" {
		t.Errorf("SetPayload kept the caller's slice: payload now %q", got.Payload)
	}
}

func TestParseDocument_RejectsMalformedFiles(t *testing.T) {
	valid := buildRealLayoutV1File([]v1TestEntry{
		{group: 1, sample: 0, payload: []byte("0123456789")},
		{group: 1, sample: 1, payload: []byte("abcdef")},
	})

	t.Run("truncated in a payload", func(t *testing.T) {
		if _, err := ParseDocument(valid[:len(valid)-3]); err == nil {
			t.Fatal("want an error, got nil")
		}
	})
	t.Run("bad signature", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[0] = 'X'
		if _, err := ParseDocument(bad); err == nil {
			t.Fatal("want an error, got nil")
		}
	})
	t.Run("too short for a header", func(t *testing.T) {
		if _, err := ParseDocument([]byte("ElecbyteSnd")); err == nil {
			t.Fatal("want an error, got nil")
		}
	})
	t.Run("second sound overlaps the first", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		binary.LittleEndian.PutUint32(bad[512:516], 520) // first sound points inside itself
		_, err := ParseDocument(bad)
		if err == nil || !strings.Contains(err.Error(), "unsupported layout") {
			t.Fatalf("err = %v, want an unsupported layout error", err)
		}
	})
	t.Run("first subheader inside the header", func(t *testing.T) {
		bad := buildV1File(t, []v1TestEntry{{group: 1, sample: 0, payload: []byte("abcd")}})
		binary.LittleEndian.PutUint32(bad[20:24], 8)
		if _, err := ParseDocument(bad); err == nil {
			t.Fatal("want an error, got nil")
		}
	})
}

// TestDocument_CorpusFilesRoundTripByteIdentical checks every readable
// file of a local real-file corpus given through SND_CORPUS_DIR; without
// it the test does nothing, since the corpus is never vendored.
func TestDocument_CorpusFilesRoundTripByteIdentical(t *testing.T) {
	dir := os.Getenv("SND_CORPUS_DIR")
	if dir == "" {
		t.Skip("SND_CORPUS_DIR not set: no real-file corpus to check")
	}

	checked := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".snd") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := ParseDocument(data)
		if err != nil {
			return nil // not a readable .snd (e.g. a misnamed file)
		}
		out, err := doc.Serialize()
		if err != nil || !bytes.Equal(out, data) {
			t.Errorf("%s: round trip differs (err %v)", path, err)
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	if checked == 0 {
		t.Fatalf("no readable .snd file found under %s", dir)
	}
	t.Logf("%d real files round-tripped byte-identical", checked)
}
