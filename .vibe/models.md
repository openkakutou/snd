# Data models

## V1Header
| Field | Type | Notes |
|---|---|---|
| Version | [4]byte | Raw version bytes as stored in the file |
| NumberOfSounds | int | Declared count of sound entries |
Defined in: `v1.go`

## V1SoundEntry
| Field | Type | Notes |
|---|---|---|
| Group | int | Sound group index |
| Sample | int | Sample index within Group |
| Offset | int64 | Absolute file offset of the embedded RIFF/WAVE audio blob |
| Length | int | Length of that blob, in bytes |
Defined in: `v1.go`

## V1SoundTable
| Field | Type | Notes |
|---|---|---|
| Header | V1Header | |
| Sounds | []V1SoundEntry | |
Defined in: `v1.go`. `Index(group, sample int) (int, bool)` resolves a `(group, sample)` key to its position in `Sounds`.

## DecodedSound
| Field | Type | Notes |
|---|---|---|
| SampleRate | int | Hz |
| Channels | int | Interleaved channel count |
| BitsPerSample | int | Original source bit depth (8 or 16) before normalization |
| PCM | []int16 | Interleaved samples, normalized to signed 16-bit regardless of source depth |
Defined in: `v1_decoder.go`. Produced by `DecodeV1Sound` and `DecodeV2Sound`
alike — same shape regardless of source format version.

## V2Header
| Field | Type | Notes |
|---|---|---|
| Version | [4]byte | Raw version bytes as stored in the file |
| NumberOfSounds | int | Declared count of sound entries |
Defined in: `v2.go`

## V2SoundEntry
| Field | Type | Notes |
|---|---|---|
| Group | int | Sound group index |
| Sample | int | Sound's sample index within Group |
| Offset | int64 | Absolute file offset of the entry's payload (embedded audio, or an external-file-reference path) |
| Length | int | Length of that payload, in bytes |
Defined in: `v2.go`. Unlike `V1SoundEntry`, `Group`/`Sample` are read from a
4-byte on-disk field each, not 2 — see
`.vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md`.

## V2SoundTable
| Field | Type | Notes |
|---|---|---|
| Header | V2Header | |
| Sounds | []V2SoundEntry | |
Defined in: `v2.go`. `Index(group, sample int) (int, bool)` resolves a
`(group, sample)` key to its position in `Sounds`, same as `V1SoundTable`.

## ExternalAudioOpener
`func(path string) ([]byte, error)` — a caller-supplied callback
`DecodeV2Sound` calls to resolve an Ikemen GO external-file-reference
entry's path to that file's complete bytes. This package never performs
this I/O itself (filesystem or network), keeping it WASM-safe.
Defined in: `v2_decoder.go`.
