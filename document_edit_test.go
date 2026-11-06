package snd

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// threeSounds is a real-layout file with three WAV sounds and a header
// comment, plus a helper to read a serialized file back as keyed audio.
func threeSounds(t *testing.T) (*Document, [][]byte) {
	t.Helper()
	wavs := [][]byte{
		buildWAV(t, 1, 11025, 16, []byte{1, 0, 2, 0}),
		buildWAV(t, 1, 8000, 8, []byte{0x80, 0xff, 0x00}),
		buildWAV(t, 2, 22050, 16, []byte{1, 0, 2, 0, 3, 0, 4, 0}),
	}
	data := buildRealLayoutV1File([]v1TestEntry{
		{group: 1, sample: 0, payload: wavs[0]},
		{group: 1, sample: 1, payload: wavs[1]},
		{group: 2, sample: 0, payload: wavs[2]},
	})
	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	return doc, wavs
}

// decodeAll serializes doc, checks it parses and that every sound decodes,
// and returns the keys in file order.
func decodeAll(t *testing.T, doc *Document) [][2]int {
	t.Helper()
	data := serializeDoc(t, doc)
	table, err := ParseV1(readerAtBytes(data))
	if err != nil {
		t.Fatalf("ParseV1 of the edited file: %v", err)
	}
	keys := make([][2]int, len(table.Sounds))
	for i, s := range table.Sounds {
		if _, err := DecodeV1Sound(readerAtBytes(data), table, s.Group, s.Sample); err != nil {
			t.Errorf("sound (%d, %d) no longer decodes: %v", s.Group, s.Sample, err)
		}
		keys[i] = [2]int{s.Group, s.Sample}
	}
	return keys
}

func TestDocument_AddSoundAppendsADecodableSoundAndLeavesTheRestAlone(t *testing.T) {
	doc, wavs := threeSounds(t)
	before := serializeDoc(t, doc)
	added := buildWAV(t, 1, 44100, 16, []byte{9, 0, 8, 0})

	if err := doc.AddSound(5, 7, added); err != nil {
		t.Fatalf("AddSound: %v", err)
	}

	keys := decodeAll(t, doc)
	if len(keys) != 4 || keys[3] != [2]int{5, 7} {
		t.Fatalf("keys = %v, want the new (5, 7) last of 4", keys)
	}
	after := serializeDoc(t, doc)
	thirdStart := len(before) - len(wavs[2]) - 16 // start of the old last subheader
	if !bytes.Equal(after[20:thirdStart], before[20:thirdStart]) {
		t.Error("header comment or the first two sounds changed")
	}
	if got := binary.LittleEndian.Uint32(after[16:20]); got != 4 {
		t.Errorf("header sound count = %d, want 4", got)
	}
	if !bytes.Equal(after[thirdStart+16:thirdStart+16+len(wavs[2])], wavs[2]) {
		t.Error("the old last sound's audio changed")
	}
	if got := binary.LittleEndian.Uint32(after[len(after)-len(added)-16 : len(after)-len(added)-12]); int(got) != len(after) {
		t.Errorf("new last pointer = %d, want end of file %d", got, len(after))
	}
	if s, _ := doc.Sound(3); !bytes.Equal(s.Payload, added) {
		t.Error("stored audio differs from the audio given")
	}
}

func TestDocument_AddSoundToAFileWithoutSounds(t *testing.T) {
	doc, err := ParseDocument(bytes.Join([][]byte{buildV1File(t, nil)}, nil))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if err := doc.AddSound(1, 1, buildWAV(t, 1, 8000, 8, []byte{0x80})); err != nil {
		t.Fatalf("AddSound: %v", err)
	}
	if keys := decodeAll(t, doc); len(keys) != 1 || keys[0] != [2]int{1, 1} {
		t.Fatalf("keys = %v, want [(1, 1)]", keys)
	}
}

func TestDocument_ReplaceSoundChangesOnlyThatSoundsAudio(t *testing.T) {
	doc, wavs := threeSounds(t)
	replacement := buildWAV(t, 1, 16000, 16, []byte{7, 0})

	if err := doc.ReplaceSound(1, 1, replacement); err != nil {
		t.Fatalf("ReplaceSound: %v", err)
	}

	decodeAll(t, doc)
	for i, want := range [][]byte{wavs[0], replacement, wavs[2]} {
		if s, _ := doc.Sound(i); !bytes.Equal(s.Payload, want) {
			t.Errorf("sound %d audio differs from what was expected", i)
		}
	}
}

func TestDocument_RemoveSoundKeepsTheOthersDecodable(t *testing.T) {
	for name, tc := range map[string]struct {
		group, sample int
		want          [][2]int
	}{
		"first":  {1, 0, [][2]int{{1, 1}, {2, 0}}},
		"middle": {1, 1, [][2]int{{1, 0}, {2, 0}}},
		"last":   {2, 0, [][2]int{{1, 0}, {1, 1}}},
	} {
		t.Run(name, func(t *testing.T) {
			doc, _ := threeSounds(t)
			if err := doc.RemoveSound(tc.group, tc.sample); err != nil {
				t.Fatalf("RemoveSound: %v", err)
			}
			keys := decodeAll(t, doc)
			if len(keys) != len(tc.want) {
				t.Fatalf("keys = %v, want %v", keys, tc.want)
			}
			for i := range keys {
				if keys[i] != tc.want[i] {
					t.Fatalf("keys = %v, want %v", keys, tc.want)
				}
			}
			data := serializeDoc(t, doc)
			if got := binary.LittleEndian.Uint32(data[20:24]); got != 512 {
				t.Errorf("first subheader offset = %d, want 512 (comment kept)", got)
			}
		})
	}
}

func TestDocument_RemoveLastSoundKeepsTheEndOfFilePointer(t *testing.T) {
	doc, _ := threeSounds(t)
	if err := doc.RemoveSound(2, 0); err != nil {
		t.Fatalf("RemoveSound: %v", err)
	}
	data := serializeDoc(t, doc)
	table, _ := ParseV1(readerAtBytes(data))
	lastSub := table.Sounds[1].Offset - 16
	if got := binary.LittleEndian.Uint32(data[lastSub : lastSub+4]); int(got) != len(data) {
		t.Errorf("last pointer = %d, want end of file %d", got, len(data))
	}
}

func TestDocument_RemovingEverySoundLeavesAValidEmptyFile(t *testing.T) {
	doc, _ := threeSounds(t)
	for _, k := range [][2]int{{1, 0}, {1, 1}, {2, 0}} {
		if err := doc.RemoveSound(k[0], k[1]); err != nil {
			t.Fatalf("RemoveSound%v: %v", k, err)
		}
	}
	back, err := ParseDocument(serializeDoc(t, doc))
	if err != nil {
		t.Fatalf("ParseDocument of the emptied file: %v", err)
	}
	if back.Len() != 0 {
		t.Errorf("Len = %d, want 0", back.Len())
	}
}

func TestDocument_AddThenRemoveGivesBackTheOriginalFile(t *testing.T) {
	doc, _ := threeSounds(t)
	original := serializeDoc(t, doc)
	if err := doc.AddSound(9, 9, buildWAV(t, 1, 8000, 8, []byte{0x80})); err != nil {
		t.Fatalf("AddSound: %v", err)
	}
	if err := doc.RemoveSound(9, 9); err != nil {
		t.Fatalf("RemoveSound: %v", err)
	}
	if got := serializeDoc(t, doc); !bytes.Equal(got, original) {
		t.Error("adding then removing a sound did not give back the original bytes")
	}
}

func TestDocument_EditsRefuseWhatTheRuleOrTheKeysForbid(t *testing.T) {
	good := buildWAV(t, 1, 8000, 8, []byte{0x80})
	adpcm := buildWAV(t, 1, 8000, 8, []byte{0x80})
	binary.LittleEndian.PutUint16(adpcm[20:22], 2) // WAVE format tag 2: ADPCM
	bits24 := buildWAV(t, 1, 8000, 24, []byte{1, 2, 3})
	zeroRate := buildWAV(t, 1, 0, 16, []byte{1, 0})
	zeroChannels := buildWAV(t, 0, 8000, 16, []byte{1, 0})
	oddData := buildWAV(t, 1, 8000, 16, []byte{1, 0, 2})

	bad := map[string][]byte{
		"not a wav":  []byte("definitely not audio"),
		"empty":      nil,
		"adpcm":      adpcm,
		"24-bit":     bits24,
		"rate 0":     zeroRate,
		"0 channels": zeroChannels,
		"odd 16-bit": oddData,
		"short riff": []byte("RIFF"),
	}
	for name, wav := range bad {
		t.Run("add "+name, func(t *testing.T) {
			doc, _ := threeSounds(t)
			if err := doc.AddSound(9, 9, wav); err == nil {
				t.Fatal("want an error, got nil")
			}
			if doc.Len() != 3 {
				t.Errorf("a refused add changed the document: Len = %d", doc.Len())
			}
		})
		t.Run("replace "+name, func(t *testing.T) {
			doc, wavs := threeSounds(t)
			if err := doc.ReplaceSound(1, 0, wav); err == nil {
				t.Fatal("want an error, got nil")
			}
			if s, _ := doc.Sound(0); !bytes.Equal(s.Payload, wavs[0]) {
				t.Error("a refused replace changed the audio")
			}
		})
	}

	doc, _ := threeSounds(t)
	if err := doc.AddSound(1, 0, good); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate add: err = %v, want 'already exists'", err)
	}
	if err := doc.ReplaceSound(8, 8, good); err == nil || !strings.Contains(err.Error(), "no sound (group 8, sample 8)") {
		t.Errorf("replace of a missing sound: err = %v", err)
	}
	if err := doc.RemoveSound(8, 8); err == nil || !strings.Contains(err.Error(), "no sound (group 8, sample 8)") {
		t.Errorf("remove of a missing sound: err = %v", err)
	}
	if doc.Len() != 3 {
		t.Errorf("refused edits changed the document: Len = %d", doc.Len())
	}
}

func TestDocument_FindReturnsTheFirstMatch(t *testing.T) {
	doc, _ := threeSounds(t)
	if i, ok := doc.Find(2, 0); !ok || i != 2 {
		t.Errorf("Find(2, 0) = %d, %v; want 2, true", i, ok)
	}
	if _, ok := doc.Find(3, 3); ok {
		t.Error("Find(3, 3) found a sound that is not there")
	}
}
