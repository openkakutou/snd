# testdata

Real test fixtures used by the `.snd` v1 fixture-driven test suite
(`v1_fixtures_test.go`). Mirrors `sff`'s own `testdata/` layout and
trimming approach.

## `files/*.snd`

**Trimmed, not byte-identical to upstream.** A real character `.snd` file
carries every sound the character uses (dozens to low hundreds of
entries); the source files these are trimmed from are not vendored into
this repository. Each fixture here was produced from a real source file by
`gen/main.go`, a standalone regeneration tool (not part of the `snd`
package's public API): it locates the exact `(group, sample)` entry a test
needs (via `ParseV1`), copies that entry's real embedded audio bytes
verbatim, and writes a minimal valid `.snd` file containing just that
entry.

No audio *content* is invented — every embedded byte is copied from a real
upstream file. Only the surrounding container (header, sound table) is
authored by the trimming tool, to keep the fixture small.

- `v1-basic.snd` — one real 16-bit mono PCM entry (group 1, sample 0),
  11025 Hz, 2772 samples.
- `v1-8bit.snd` — one real 8-bit PCM entry (group 40, sample 0), 11025 Hz,
  621 samples.
- `v2-basic.snd` — one real 8-bit mono PCM entry (group 1, sample 143),
  8000 Hz, from a real character file, real sound-table bytes located via
  v2's own 4-byte Group/Sample subheader fields. **Only its header's version
  stamp is synthesized** (set to `2,0,0,0`) — no file in the available
  corpus declares itself version 2 in the conventional sense; see
  `.vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md`.
  Sample 143 is deliberately real: it is exactly the kind of value v1's own
  2-byte Sample field cannot represent correctly.

**`v2-external-ref.snd` and `v2-external-audio.wav` are a different kind of
fixture — clearly-marked synthetic, not trimmed from a real file.** No real
Ikemen GO character using the external-file-reference extension was found
(see `.vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md`,
mirroring `sff`'s own accepted-gap precedent for RLE5, its decision `014`):

- `v2-external-ref.snd` is hand-built by `gen/main.go`'s
  `writeSyntheticExternalRefFixture`: a single v2 entry (group 0, sample 7)
  whose payload is the literal path string `"v2-external-audio.wav"`
  instead of embedded audio.
- `v2-external-audio.wav` — the file that path resolves to — is nonetheless
  **real, unmodified audio**: one real 8-bit mono PCM clip (8000 Hz), copied
  verbatim from the same real-character corpus every other fixture here
  comes from. Only the *reference itself* is synthetic; the audio it points
  to is not.

See `.vibe/fixture-sources.md` for where the source files come from and
the corpus-wide scans backing decisions `001`, `002`, and `003`.

To regenerate (e.g. after adding a new scenario to `gen/main.go`):

```
SRC_DIR=/path/to/chars go run ./testdata/gen
```
