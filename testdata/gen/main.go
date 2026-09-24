// Command gen regenerates the trimmed .snd test fixtures under
// testdata/files from real, full-size character .snd files. It is a
// standalone regeneration tool, not part of the snd package's public API
// (mirrors sff's own testdata/gen).
//
// It is not meant to run in CI. Run it manually, pointing SRC_DIR at a
// local checkout of real character folders (see testdata/README.md for
// where they come from), whenever a fixture needs to be regenerated:
//
//	SRC_DIR=/path/to/chars go run ./testdata/gen
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openkakutou/snd"
)

// scenario describes one trimmed fixture to produce: which real source
// .snd file to pull entries from (relative to SRC_DIR), and which
// (group, sample) entries to keep, verbatim, in the output file.
type scenario struct {
	name    string // output file, under testdata/files
	src     string // source file, relative to SRC_DIR
	version int    // 1 or 2: which subheader layout to locate entries with
	entries []entryRef
}

type entryRef struct {
	group, sample int
}

// rawAudioExtract describes one real embedded audio blob to save standalone
// (not wrapped in a .snd container) under testdata/files — used for the v2
// external-file-reference fixture's companion file, since Ikemen GO's
// extension resolves a path to a real audio file, not another .snd table.
type rawAudioExtract struct {
	name    string // output file, under testdata/files
	src     string // source file, relative to SRC_DIR
	version int
	entry   entryRef
}

func main() {
	srcDir := os.Getenv("SRC_DIR")
	if srcDir == "" {
		fmt.Fprintln(os.Stderr, "SRC_DIR environment variable must point at the real .snd files (see testdata/README.md)")
		os.Exit(1)
	}
	outDir := "testdata/files"
	if _, err := os.Stat(outDir); err != nil {
		fmt.Fprintln(os.Stderr, "run this from the repo root:", err)
		os.Exit(1)
	}

	scenarios := []scenario{
		{
			name:    "v1-basic.snd",
			src:     "Guilty Gear/Ky Kiske/Sound.snd",
			version: 1,
			entries: []entryRef{
				{group: 1, sample: 0}, // real 16-bit mono voice clip
			},
		},
		{
			name:    "v1-8bit.snd",
			src:     "Capcom/Amaterasu/ama.snd",
			version: 1,
			entries: []entryRef{
				{group: 40, sample: 0}, // real 8-bit mono clip
			},
		},
		{
			// Real audio, real sound-table bytes — only the version stamp is
			// synthesized (no real file in the available corpus declares
			// itself version 2; see
			// .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md).
			// (group 1, sample 143) is a real entry whose Sample value does
			// not fit v1's 2-byte field the way this one's 4-byte field
			// does.
			name:    "v2-basic.snd",
			src:     "Misc/Popeye/popeye.snd",
			version: 2,
			entries: []entryRef{
				{group: 1, sample: 143}, // real 8-bit mono clip
			},
		},
	}

	for _, sc := range scenarios {
		srcPath := filepath.Join(srcDir, sc.src)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v (skipping)\n", sc.name, err)
			continue
		}

		out, err := trim(data, sc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", sc.name, err)
			continue
		}

		if err := os.WriteFile(filepath.Join(outDir, sc.name), out, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: writing: %v\n", sc.name, err)
			continue
		}
		fmt.Printf("%s: %d bytes (from %d)\n", sc.name, len(out), len(data))
	}

	rawExtracts := []rawAudioExtract{
		{
			// Real audio for the v2 external-file-reference fixture's
			// companion file (see v2-external-ref.snd, hand-built
			// separately — no real Ikemen GO character using this
			// extension was found; see
			// .vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md).
			name:    "v2-external-audio.wav",
			src:     "City Hunter/Ryo Saeba/RS.snd",
			version: 2,
			entry:   entryRef{group: 0, sample: 0}, // real 8-bit mono clip
		},
	}

	for _, ex := range rawExtracts {
		srcPath := filepath.Join(srcDir, ex.src)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v (skipping)\n", ex.name, err)
			continue
		}
		blob, err := extractRaw(data, ex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", ex.name, err)
			continue
		}
		if err := os.WriteFile(filepath.Join(outDir, ex.name), blob, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: writing: %v\n", ex.name, err)
			continue
		}
		fmt.Printf("%s: %d bytes\n", ex.name, len(blob))
	}

	if err := writeSyntheticExternalRefFixture(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "v2-external-ref.snd: %v\n", err)
	} else {
		fmt.Println("v2-external-ref.snd: written (synthetic — see testdata/README.md)")
	}
}

// writeSyntheticExternalRefFixture writes testdata/files/v2-external-ref.snd:
// a hand-built, clearly-marked-as-synthetic v2 file whose single entry's
// payload is a relative path, not embedded audio — no real Ikemen GO
// character exercising this extension was found (see
// .vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md).
// It always runs, independent of SRC_DIR, since nothing here comes from a
// real corpus file. It points at v2-external-audio.wav, this same
// directory's real-audio companion fixture.
func writeSyntheticExternalRefFixture(outDir string) error {
	const path = "v2-external-audio.wav"
	out, err := trimSynthetic(0, 7, []byte(path))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "v2-external-ref.snd"), out, 0o644)
}

// trimSynthetic builds a single-entry v2 .snd file whose payload is exactly
// payload (a raw byte string, not necessarily audio) for the given
// (group, sample) key.
func trimSynthetic(group, sample int, payload []byte) ([]byte, error) {
	const headerSize = 24
	var buf bytes.Buffer
	header := make([]byte, headerSize)
	copy(header[0:12], []byte("ElecbyteSnd\x00"))
	header[12], header[13], header[14], header[15] = 2, 0, 0, 0
	binary.LittleEndian.PutUint32(header[16:20], 1)
	binary.LittleEndian.PutUint32(header[20:24], uint32(headerSize))
	buf.Write(header)

	sub := make([]byte, 16)
	binary.LittleEndian.PutUint32(sub[4:8], uint32(len(payload)))
	binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(group)))
	binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(sample)))
	buf.Write(sub)
	buf.Write(payload)

	return buf.Bytes(), nil
}

// extractRaw locates ex.entry in the real source file and returns its
// embedded audio bytes verbatim, unwrapped from any .snd container.
func extractRaw(data []byte, ex rawAudioExtract) ([]byte, error) {
	_, _, offset, length, err := locate(data, ex.version, ex.entry)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ex.src, err)
	}
	if offset+int64(length) > int64(len(data)) {
		return nil, fmt.Errorf("entry (group %d, sample %d) payload out of bounds", ex.entry.group, ex.entry.sample)
	}
	return data[offset : offset+int64(length)], nil
}

// locate finds ref's (group, sample) entry in a real .snd file's own bytes,
// using either v1's or v2's subheader layout depending on version, and
// returns its exact (group, sample, offset, length).
func locate(data []byte, version int, ref entryRef) (group, sample int, offset int64, length int, err error) {
	switch version {
	case 1:
		table, err := snd.ParseV1(readerAt(data))
		if err != nil {
			return 0, 0, 0, 0, err
		}
		i, ok := table.Index(ref.group, ref.sample)
		if !ok {
			return 0, 0, 0, 0, fmt.Errorf("(group %d, sample %d) not found", ref.group, ref.sample)
		}
		e := table.Sounds[i]
		return e.Group, e.Sample, e.Offset, e.Length, nil
	case 2:
		table, err := snd.ParseV2(readerAt(data))
		if err != nil {
			return 0, 0, 0, 0, err
		}
		i, ok := table.Index(ref.group, ref.sample)
		if !ok {
			return 0, 0, 0, 0, fmt.Errorf("(group %d, sample %d) not found", ref.group, ref.sample)
		}
		e := table.Sounds[i]
		return e.Group, e.Sample, e.Offset, e.Length, nil
	default:
		return 0, 0, 0, 0, fmt.Errorf("unsupported version %d", version)
	}
}

// trim builds a minimal, real-bytes-only .snd file exposing exactly the
// (group, sample) entries sc asks for, each entry's embedded audio copied
// verbatim from the real source file. sc.version selects which subheader
// layout to both locate entries with and write back out — v1's 2-byte
// Group/Sample fields, or v2's 4-byte ones (see
// .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md).
func trim(data []byte, sc scenario) ([]byte, error) {
	type sound struct {
		group, sample int
		payload       []byte
	}
	var sounds []sound
	for _, ref := range sc.entries {
		group, sample, offset, length, err := locate(data, sc.version, ref)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", sc.src, err)
		}
		sounds = append(sounds, sound{
			group:   group,
			sample:  sample,
			payload: data[offset : offset+int64(length)],
		})
	}

	const headerSize = 24
	var buf bytes.Buffer
	header := make([]byte, headerSize)
	copy(header[0:12], []byte("ElecbyteSnd\x00"))
	if sc.version == 2 {
		// No real file in the available corpus declares itself version 2 in
		// the conventional sense (see decision 002) — stamp it explicitly
		// so the fixture is unambiguous about which read path it exercises.
		header[12], header[13], header[14], header[15] = 2, 0, 0, 0
	} else if len(data) >= 16 {
		// v1: preserve the real source file's own version bytes verbatim.
		copy(header[12:16], data[12:16])
	}
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(sounds)))
	binary.LittleEndian.PutUint32(header[20:24], uint32(headerSize))
	buf.Write(header)

	subheaderSize := 16
	offsets := make([]int, len(sounds))
	pos := headerSize
	for i, s := range sounds {
		offsets[i] = pos
		pos += subheaderSize + len(s.payload)
	}

	for i, s := range sounds {
		sub := make([]byte, subheaderSize)
		if i+1 < len(sounds) {
			binary.LittleEndian.PutUint32(sub[0:4], uint32(offsets[i+1]))
		}
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(s.payload)))
		if sc.version == 2 {
			binary.LittleEndian.PutUint32(sub[8:12], uint32(int32(s.group)))
			binary.LittleEndian.PutUint32(sub[12:16], uint32(int32(s.sample)))
		} else {
			binary.LittleEndian.PutUint16(sub[8:10], uint16(int16(s.group)))
			binary.LittleEndian.PutUint16(sub[10:12], uint16(int16(s.sample)))
		}
		buf.Write(sub)
		buf.Write(s.payload)
	}

	return buf.Bytes(), nil
}

// readerAtBytes adapts a byte slice to io.ReaderAt.
type readerAtBytes []byte

func (r readerAtBytes) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r)) {
		return 0, fmt.Errorf("offset %d past end of data (len %d)", off, len(r))
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, fmt.Errorf("short read at offset %d: got %d, want %d", off, n, len(p))
	}
	return n, nil
}

func readerAt(data []byte) readerAtBytes { return readerAtBytes(data) }
