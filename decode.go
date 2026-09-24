package snd

import (
	"fmt"
	"io"
)

// headerPeekSize is the number of leading bytes DecodeSound reads to
// identify a file: the 12-byte signature both .snd versions share, plus the
// 4 version bytes that distinguish them (mirrors sff's own
// signaturePeekSize, github.com/openkakutou/sff's load.go).
const headerPeekSize = 24

// DecodeSound reads a MUGEN/Ikemen GO .snd file from r — version 1 or
// version 2, auto-detected from the file's own header version stamp —
// and decodes the sound entry keyed by (group, sample) to raw PCM,
// mirroring sff.Load's own auto-detection of its file format's version.
//
// openExternal resolves a v2 entry's Ikemen GO external-file-reference
// extension (see ExternalAudioOpener); it is ignored when r turns out to
// hold a v1 file, which has no such extension, and may be nil for a v2 file
// known to have no external-reference entries.
//
// This is the entry point cmd/wasm's JS-callable global uses: a JS caller
// hands over a .snd file's raw bytes without knowing (or needing to know)
// which version it is, the same way a native Go caller can now use
// DecodeSound instead of choosing between ParseV1+DecodeV1Sound and
// ParseV2+DecodeV2Sound itself. See
// .vibe/decisions/004-version-agnostic-decode-detects-v2-via-header-version-stamp.md.
func DecodeSound(r io.ReaderAt, group, sample int, openExternal ExternalAudioOpener) (*DecodedSound, error) {
	isV2, err := detectVersion(r)
	if err != nil {
		return nil, err
	}

	if isV2 {
		table, err := ParseV2(r)
		if err != nil {
			return nil, err
		}
		return DecodeV2Sound(r, table, group, sample, openExternal)
	}

	table, err := ParseV1(r)
	if err != nil {
		return nil, err
	}
	return DecodeV1Sound(r, table, group, sample)
}

// detectVersion peeks at a .snd file's own signature and header version
// stamp to tell v1 and v2 files apart, the same way DecodeSound needs to
// before picking which version-specific parser to use.
//
// header[12] (Header.Version's first byte) is 2 only for a v2 file — this
// is not a guess: it is the exact byte this repo's own v2 test/fixture
// builders (buildV2File, testdata/gen/main.go) already set to distinguish a
// v2 file from a v1 one (which sets header[12] to 1), and matches every
// real v1 file in the corpus (none of which declares itself version 2 in
// the conventional sense — see
// .vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md).
// This is a different byte position than sff's own detectVersion (which
// checks its version field's *last* byte) because the two file formats'
// version stamps are laid out differently on disk; the underlying
// technique — peek the header, branch on one already-established version
// byte — is the same.
func detectVersion(r io.ReaderAt) (isV2 bool, err error) {
	peek := make([]byte, headerPeekSize)
	if _, err := r.ReadAt(peek, 0); err != nil {
		return false, fmt.Errorf("snd: reading file header: %w", err)
	}
	if sig := string(peek[0:12]); sig != v1Signature {
		return false, fmt.Errorf("snd: not a .snd file: unexpected signature %q", sig)
	}
	return peek[12] == 2, nil
}
