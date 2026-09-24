# Module: root (package `snd`)

**Role:** MUGEN/Ikemen GO `.snd` v1 and v2 read paths — parses a file's
header and sound table, and decodes each entry's audio to raw PCM,
including a v2 entry's Ikemen GO external-file-reference extension.

**Files:** `version.go`, `v1.go`, `v1_decoder.go`, `v2.go`, `v2_decoder.go`

**Exports:**
- `Version` (string constant)
- `ParseV1(r io.ReaderAt) (*V1SoundTable, error)`
- `V1Header`, `V1SoundEntry`, `V1SoundTable` (with `(*V1SoundTable) Index(group, sample int) (int, bool)`)
- `DecodeV1Sound(r io.ReaderAt, table *V1SoundTable, group, sample int) (*DecodedSound, error)`
- `ParseV2(r io.ReaderAt) (*V2SoundTable, error)`
- `V2Header`, `V2SoundEntry`, `V2SoundTable` (with `(*V2SoundTable) Index(group, sample int) (int, bool)`)
- `DecodeV2Sound(r io.ReaderAt, table *V2SoundTable, group, sample int, openExternal ExternalAudioOpener) (*DecodedSound, error)`
- `ExternalAudioOpener` (`func(path string) ([]byte, error)`)
- `DecodedSound` (shared by both v1 and v2 decode)

**Depends on:** nothing internal yet — stdlib only (`encoding/binary`, `io`,
`errors`, `fmt`, `path/filepath`, `strings`).
