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
	entries []entryRef
}

type entryRef struct {
	group, sample int
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
			name: "v1-basic.snd",
			src:  "Guilty Gear/Ky Kiske/Sound.snd",
			entries: []entryRef{
				{group: 1, sample: 0}, // real 16-bit mono voice clip
			},
		},
		{
			name: "v1-8bit.snd",
			src:  "Capcom/Amaterasu/ama.snd",
			entries: []entryRef{
				{group: 40, sample: 0}, // real 8-bit mono clip
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
}

// trim builds a minimal, real-bytes-only .snd v1 file exposing exactly the
// (group, sample) entries sc asks for, each entry's embedded audio copied
// verbatim from the real source file.
func trim(data []byte, sc scenario) ([]byte, error) {
	table, err := snd.ParseV1(readerAt(data))
	if err != nil {
		return nil, err
	}

	type sound struct {
		group, sample int
		payload       []byte
	}
	var sounds []sound
	for _, ref := range sc.entries {
		i, ok := table.Index(ref.group, ref.sample)
		if !ok {
			return nil, fmt.Errorf("(group %d, sample %d) not found in %s", ref.group, ref.sample, sc.src)
		}
		e := table.Sounds[i]
		sounds = append(sounds, sound{
			group:   e.Group,
			sample:  e.Sample,
			payload: data[e.Offset : e.Offset+int64(e.Length)],
		})
	}

	const headerSize = 24
	var buf bytes.Buffer
	header := make([]byte, headerSize)
	copy(header[0:12], []byte("ElecbyteSnd\x00"))
	copy(header[12:16], table.Header.Version[:])
	binary.LittleEndian.PutUint32(header[16:20], uint32(len(sounds)))
	binary.LittleEndian.PutUint32(header[20:24], uint32(headerSize))
	buf.Write(header)

	offsets := make([]int, len(sounds))
	pos := headerSize
	for i, s := range sounds {
		offsets[i] = pos
		pos += 16 + len(s.payload)
	}

	for i, s := range sounds {
		sub := make([]byte, 16)
		if i+1 < len(sounds) {
			binary.LittleEndian.PutUint32(sub[0:4], uint32(offsets[i+1]))
		}
		binary.LittleEndian.PutUint32(sub[4:8], uint32(len(s.payload)))
		binary.LittleEndian.PutUint16(sub[8:10], uint16(int16(s.group)))
		binary.LittleEndian.PutUint16(sub[10:12], uint16(int16(s.sample)))
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
