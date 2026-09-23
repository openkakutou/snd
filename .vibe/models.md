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
Defined in: `v1_decoder.go`. Produced by `DecodeV1Sound`.
