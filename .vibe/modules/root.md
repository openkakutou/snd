# Module: root (package `snd`)

**Role:** MUGEN/Ikemen GO `.snd` v1 read path — parses a file's header and
sound table, and decodes each entry's embedded audio to raw PCM.

**Files:** `version.go`, `v1.go`, `v1_decoder.go`

**Exports:**
- `Version` (string constant)
- `ParseV1(r io.ReaderAt) (*V1SoundTable, error)`
- `V1Header`, `V1SoundEntry`, `V1SoundTable` (with `(*V1SoundTable) Index(group, sample int) (int, bool)`)
- `DecodeV1Sound(r io.ReaderAt, table *V1SoundTable, group, sample int) (*DecodedSound, error)`
- `DecodedSound`

**Depends on:** nothing internal yet — stdlib only (`encoding/binary`, `io`, `errors`, `fmt`).
