package snd

import (
	"encoding/binary"
	"fmt"
	"io"
)

// pcmFormatTag is the WAVE format tag value for uncompressed PCM. This is
// the only format tag this package decodes — see
// .vibe/decisions/001-v1-pcm-only-decode-adpcm-out-of-scope.md for why
// ADPCM (and any other compressed format) is a permanent, explicitly
// named gap rather than unverifiable speculative code.
const pcmFormatTag = 1

// DecodedSound is one .snd v1 sound entry's audio, decoded to raw PCM.
type DecodedSound struct {
	// SampleRate is the audio's sample rate, in Hz.
	SampleRate int
	// Channels is the number of interleaved audio channels.
	Channels int
	// BitsPerSample is the original source bit depth (8 or 16) before
	// normalization to PCM.
	BitsPerSample int
	// PCM is the decoded audio, interleaved by channel and normalized to
	// signed 16-bit samples regardless of the original bit depth.
	PCM []int16
}

// DecodeV1Sound reads and decodes the embedded audio for the sound entry
// keyed by (group, sample) in table, from r. The entry's embedded audio
// data must be a RIFF/WAVE blob carrying uncompressed PCM (8-bit unsigned
// or 16-bit signed) — the only format real .snd files, and Ikemen GO's own
// current sound loader, are known to use.
func DecodeV1Sound(r io.ReaderAt, table *V1SoundTable, group, sample int) (*DecodedSound, error) {
	i, ok := table.Index(group, sample)
	if !ok {
		return nil, fmt.Errorf("snd: no sound (group %d, sample %d) in table", group, sample)
	}
	entry := table.Sounds[i]

	blob := make([]byte, entry.Length)
	if _, err := r.ReadAt(blob, entry.Offset); err != nil {
		return nil, fmt.Errorf("snd: sound (group %d, sample %d): reading %d bytes of audio data: %w", group, sample, entry.Length, err)
	}

	return decodeWAVPCM(blob, group, sample)
}

// decodeWAVPCM parses a RIFF/WAVE blob and decodes its "data" chunk to
// signed 16-bit PCM, guided by its "fmt " chunk. group and sample are only
// used to name the offending entry in error messages.
func decodeWAVPCM(blob []byte, group, sample int) (*DecodedSound, error) {
	return decodeWAVPCMWithContext(blob, fmt.Sprintf("sound (group %d, sample %d)", group, sample))
}

// decodeWAVPCMWithContext is decodeWAVPCM's shared core: context is
// prepended to every error message ("snd: <context>: ...") so callers other
// than DecodeV1Sound — namely DecodeV2Sound, for both an entry's embedded
// audio and an Ikemen GO external-file-reference entry's resolved audio —
// can name themselves precisely (e.g. including the external file's path)
// without duplicating this RIFF/WAVE parsing logic.
func decodeWAVPCMWithContext(blob []byte, context string) (*DecodedSound, error) {
	if len(blob) < 12 || string(blob[0:4]) != "RIFF" || string(blob[8:12]) != "WAVE" {
		return nil, fmt.Errorf("snd: %s: audio data is not a valid RIFF/WAVE blob", context)
	}

	var fmtChunk, dataChunk []byte
	pos := 12
	for pos+8 <= len(blob) {
		id := string(blob[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(blob[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(blob) {
			return nil, fmt.Errorf("snd: %s: truncated %q chunk", context, id)
		}
		switch id {
		case "fmt ":
			fmtChunk = blob[pos : pos+size]
		case "data":
			dataChunk = blob[pos : pos+size]
		}
		pos += size
		if size%2 == 1 {
			pos++ // chunks are word-aligned; skip the pad byte
		}
	}

	if fmtChunk == nil {
		return nil, fmt.Errorf("snd: %s: missing fmt chunk", context)
	}
	if len(fmtChunk) < 16 {
		return nil, fmt.Errorf("snd: %s: fmt chunk too short (%d bytes)", context, len(fmtChunk))
	}
	if dataChunk == nil {
		return nil, fmt.Errorf("snd: %s: missing data chunk", context)
	}

	formatTag := binary.LittleEndian.Uint16(fmtChunk[0:2])
	channels := int(binary.LittleEndian.Uint16(fmtChunk[2:4]))
	sampleRate := int(binary.LittleEndian.Uint32(fmtChunk[4:8]))
	bitsPerSample := int(binary.LittleEndian.Uint16(fmtChunk[14:16]))

	if formatTag != pcmFormatTag {
		return nil, fmt.Errorf("snd: %s: unsupported WAVE format tag %d (only PCM is supported)", context, formatTag)
	}
	if channels <= 0 {
		return nil, fmt.Errorf("snd: %s: invalid channel count %d", context, channels)
	}

	var pcm []int16
	switch bitsPerSample {
	case 8:
		pcm = make([]int16, len(dataChunk))
		for i, b := range dataChunk {
			// 8-bit WAV PCM is unsigned, centered at 128. Re-center to 0
			// and scale up to use the full signed 16-bit range.
			pcm[i] = (int16(b) - 128) << 8
		}
	case 16:
		if len(dataChunk)%2 != 0 {
			return nil, fmt.Errorf("snd: %s: truncated 16-bit sample data (%d bytes, not a multiple of 2)", context, len(dataChunk))
		}
		pcm = make([]int16, len(dataChunk)/2)
		for i := range pcm {
			pcm[i] = int16(binary.LittleEndian.Uint16(dataChunk[i*2 : i*2+2]))
		}
	default:
		return nil, fmt.Errorf("snd: %s: unsupported bit depth %d (only 8-bit and 16-bit PCM are supported)", context, bitsPerSample)
	}

	return &DecodedSound{
		SampleRate:    sampleRate,
		Channels:      channels,
		BitsPerSample: bitsPerSample,
		PCM:           pcm,
	}, nil
}
