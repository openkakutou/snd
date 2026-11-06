# Module: root (package `snd`)

**Role:** MUGEN/Ikemen GO `.snd` v1 and v2 read, write and edit paths. It
parses a file's header and sound table, decodes each entry's audio to raw
PCM (including a v2 entry's Ikemen GO external-file-reference extension),
writes sound sets back, round-trips a whole file byte-identical as a
`Document`, and adds, replaces and removes sounds by key. A version-agnostic
decode call auto-detects v1 vs v2; it is the entry point `cmd/wasm` (see
`modules/wasm.md`) builds on.

**Files:** `version.go`, `v1.go`, `v1_decoder.go`, `v2.go`, `v2_decoder.go`,
`decode.go`, `v1_write.go`, `v2_write.go`, `document.go`, `document_edit.go`

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
- `DecodeSound(r io.ReaderAt, group, sample int, openExternal ExternalAudioOpener) (*DecodedSound, error)` — auto-detects v1 vs v2 from the file's header (`header[12] == 2` signals v2, mirroring `sff.Load`'s own auto-detection technique at a different byte offset — see `.vibe/decisions/004-wasm-entrypoint-api-shape-and-version-detection.md`) and dispatches to the matching `ParseV*`+`Decode*Sound` pair; `openExternal` is ignored for a v1 file

- `ReadV1Set(r io.ReaderAt) (*V1SoundSet, error)` / `EncodeV1(set *V1SoundSet) ([]byte, error)`, `V1SoundSet`, `V1Sound` — v1 write path (backlog item 005): byte-identical round trip of real files, see `.vibe/decisions/006-v1-write-path-layout-and-scope.md`
- `ReadV2Set` / `EncodeV2`, `V2SoundSet`, `V2Sound` — v2 write path (backlog item 006), strict validation, shared container reader/writer with v1, see `.vibe/decisions/007-v2-write-path-validation-and-shared-container.md`
- `ParseDocument(data []byte) (*Document, error)`, `Document` (`Len`, `Sound`, `Serialize`), `DocumentSound` — byte-exact round trip (backlog item 007), see `.vibe/decisions/008-document-keeps-layout-and-pointer-kinds.md`
- `(*Document).Find` / `AddSound` / `ReplaceSound` / `RemoveSound` — editing by key (backlog item 008); audio must be 8/16-bit PCM WAV, else refused, see `.vibe/decisions/009-editing-api-rejects-non-conformant-audio.md`

**Depends on:** nothing internal yet — stdlib only (`encoding/binary`, `io`,
`errors`, `fmt`, `path/filepath`, `strings`).
