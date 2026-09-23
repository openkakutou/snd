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

See `.vibe/fixture-sources.md` for where the source files come from and
the corpus-wide scan backing decision
`001-v1-pcm-only-decode-adpcm-out-of-scope.md`.

To regenerate (e.g. after adding a new scenario to `gen/main.go`):

```
SRC_DIR=/path/to/chars go run ./testdata/gen
```
