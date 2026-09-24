package snd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ExternalAudioOpener resolves a .snd v2 entry's Ikemen GO external-file-
// reference path (as recorded in the sound table, relative to wherever the
// caller's own asset layout keeps it — this package has no opinion on that)
// to that file's complete raw bytes. This package never touches the
// filesystem or network itself: doing so would break its no-playback-
// dependency, WASM-safe design (see
// .vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md).
// A nil opener is valid for a table with no external-reference entries.
type ExternalAudioOpener func(path string) ([]byte, error)

// externalAudioExtensions are the file extensions DecodeV2Sound recognizes
// as a plausible external-file-reference path, matching the acceptance
// criteria's own examples plus the formats Ikemen GO's own audio stack
// otherwise supports. Only ".wav" content actually decodes today (see
// decision 003) — the others are still recognized so a genuinely unsupported
// external file reports a precise "unsupported format" error instead of
// being misread as corrupt embedded audio.
var externalAudioExtensions = map[string]bool{
	".wav":  true,
	".ogg":  true,
	".mp3":  true,
	".flac": true,
}

// DecodeV2Sound reads and decodes the audio for the sound entry keyed by
// (group, sample) in table, from r. The entry's payload is either:
//
//   - embedded audio: a RIFF/WAVE blob carrying uncompressed PCM, decoded the
//     same way DecodeV1Sound decodes a v1 entry; or
//   - an Ikemen GO external-file-reference: a relative path naming another
//     audio file, resolved via openExternal and then decoded the same way.
//     openExternal may be nil if table is known to have no such entries.
func DecodeV2Sound(r io.ReaderAt, table *V2SoundTable, group, sample int, openExternal ExternalAudioOpener) (*DecodedSound, error) {
	i, ok := table.Index(group, sample)
	if !ok {
		return nil, fmt.Errorf("snd: no sound (group %d, sample %d) in table", group, sample)
	}
	entry := table.Sounds[i]

	blob := make([]byte, entry.Length)
	if _, err := r.ReadAt(blob, entry.Offset); err != nil {
		return nil, fmt.Errorf("snd: sound (group %d, sample %d): reading %d bytes of payload: %w", group, sample, entry.Length, err)
	}

	if isEmbeddedRIFFWAVE(blob) {
		return decodeWAVPCM(blob, group, sample)
	}

	path, isExternal := externalFileReferencePath(blob)
	if !isExternal {
		// Neither a valid RIFF/WAVE blob nor a plausible external path:
		// genuinely corrupt/malformed embedded data, same as v1's own
		// error for this case.
		return decodeWAVPCM(blob, group, sample)
	}

	context := fmt.Sprintf("sound (group %d, sample %d): external file %q", group, sample, path)
	if openExternal == nil {
		return nil, fmt.Errorf("snd: %s: entry references an external file but no external file opener was provided", context)
	}

	raw, err := openExternal(path)
	if err != nil {
		return nil, fmt.Errorf("snd: %s: %w", context, err)
	}

	if !isEmbeddedRIFFWAVE(raw) {
		return nil, fmt.Errorf("snd: %s: unsupported external audio format (only WAV/PCM is supported)", context)
	}
	return decodeWAVPCMWithContext(raw, context)
}

// isEmbeddedRIFFWAVE reports whether blob starts with a RIFF/WAVE magic
// header, without validating the rest of it — decodeWAVPCM does that.
func isEmbeddedRIFFWAVE(blob []byte) bool {
	return len(blob) >= 12 && string(blob[0:4]) == "RIFF" && string(blob[8:12]) == "WAVE"
}

// externalFileReferencePath reports whether blob plausibly holds an Ikemen
// GO external-file-reference path rather than corrupt embedded audio: after
// trimming one optional trailing NUL terminator, it must be non-empty,
// printable ASCII, and end in a recognized audio file extension. This
// heuristic exists because no binary flag distinguishes the two cases (see
// decision 003) — it deliberately errs toward "corrupt embedded data" for
// anything that isn't unambiguously path-shaped.
func externalFileReferencePath(blob []byte) (string, bool) {
	b := blob
	if len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return "", false
	}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return "", false
		}
	}
	path := string(b)
	ext := strings.ToLower(filepath.Ext(path))
	if !externalAudioExtensions[ext] {
		return "", false
	}
	return path, true
}
