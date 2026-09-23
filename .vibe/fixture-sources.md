# Fixture sources for `.snd` test data

Reference material and real-world files used to source or validate test
fixtures for this repo. Kept separate from the codebase index proper since
none of this is referenced by path from Go code — it is a local,
machine-specific resource unavailable in CI or on other machines. Mirrors
`sff`'s own `.vibe/fixture-sources.md`.

## Local real-character corpus (not referenced from code)

`~/workspace/ikemen-quick-versus/chars/` on the machine this repo is
usually developed on: the same real Ikemen GO frontend install `sff` uses,
with **497 real character `.snd` files**. Available interactively for
finding real fixtures and statistically validating format assumptions.

**This path must never be hardcoded into Go source, tests, or committed
config** — it only exists on this machine, not in CI or for other
contributors. Use it to *find and verify* candidate real fixtures, then
trim/vendor the result into `testdata/` via `testdata/gen`, exactly as if
it had been sourced from any other one-off, non-reproducible location.

## Corpus scan results (backlog item 001, run 2026-09-23)

A one-off scan of the full local corpus above (not a committed test —
this repo has no `sff`-style always-available `_CORPUS_DIR`-gated
compatibility test yet): **496 of 497 files parsed** (1 had a bad
signature — a genuinely non-`.snd` file misnamed with the extension),
**54,309 sound entries** located across them.

- **54,306 entries (99.99%) are WAVE format tag 1 (PCM)** — 16-bit:
  35,670 entries; 8-bit: 18,636 entries.
- **3 entries** have embedded data that is not a valid RIFF/WAVE blob at
  all (`not_riff_wave`) — not investigated further; `DecodeV1Sound`
  reports these with a descriptive error rather than panicking or
  producing silently-wrong audio, which is this backlog item's own
  acceptance criterion for malformed entries.
- **Zero ADPCM (or any other non-PCM format tag) entries found.** See
  decision `001-v1-pcm-only-decode-adpcm-out-of-scope.md` for what this
  means for the decoder's scope.

This finding, combined with Ikemen GO's own reference sound loader
rejecting non-PCM format tags too (see the same decision), is the
evidence base for treating ADPCM as a permanent, documented gap rather
than implementing it against an unverifiable synthetic-only fixture.

## Fixture provenance

`testdata/files/v1-basic.snd` and `testdata/files/v1-8bit.snd` are trimmed
from two real files in the corpus above via `testdata/gen`:

- `v1-basic.snd` ← `Guilty Gear/Ky Kiske/Sound.snd`, entry (group 1,
  sample 0): a real 16-bit mono voice clip, 11025 Hz.
- `v1-8bit.snd` ← `Capcom/Amaterasu/ama.snd`, entry (group 40, sample 0):
  a real 8-bit mono clip, 11025 Hz.

Both entries' embedded WAV bytes are copied verbatim; only the
surrounding container (header, sound table) is rebuilt to keep the
fixture small. See `testdata/README.md`.
