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

## Corpus scan results (backlog item 002, run 2026-09-24)

A byte-level scan of the same local corpus for `.snd` v2 support (real
files' 4-byte header version fields, run over all 496 parseable files):
**zero files declare version 2** in the conventional sense — every real
file's version tuple is one of `(1,1)`, `(4,0)`, or `(256,256)` (the latter
two both being byte-order variants of "version 1.1", a known MUGEN-tooling
quirk), never `2`. Ikemen GO's own reference loader (`src/sound.go`) applies
the same parsing regardless of this field's value, so this is consistent
with the field being largely decorative rather than a genuine format
discriminator in real-world files.

A targeted byte-level check of individual real subheaders (60 files,
6,509 entries) found the true Group/Sample layout: both are 4-byte
little-endian signed integers filling the entire 16-byte subheader, with
nothing reserved — not v1's shipped 2-byte-plus-4-reserved-bytes reading.
This matches Ikemen GO's own reference parser exactly and is the basis for
`ParseV2`'s subheader layout; see
`.vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md`
for the full evidence and its implication for the already-shipped v1 path
(tracked as backlog item `004`).

No file in the corpus contains an entry using Ikemen GO's
external-file-reference extension (an entry whose payload is a path rather
than embedded audio) — see
`.vibe/decisions/003-v2-external-file-reference-detection-and-resolution.md`
for the resulting accepted validation gap, mirroring `sff`'s own precedent
for RLE5 (its decision `014`).

`testdata/files/v2-basic.snd` is trimmed from a real file the same way as
the v1 fixtures above, using v2's own subheader layout to locate the entry:

- `v2-basic.snd` ← `Misc/Popeye/popeye.snd`, entry (group 1, sample 143): a
  real 8-bit mono clip, 8000 Hz. Only the header's version stamp is
  synthesized (no real file declares version 2 — see above); the
  sound-table bytes and audio are real.

`testdata/files/v2-external-audio.wav` is real, unmodified audio trimmed the
same way, saved standalone (not wrapped in a `.snd` container) since it is
what an external-file-reference entry resolves to:

- `v2-external-audio.wav` ← `City Hunter/Ryo Saeba/RS.snd`, entry (group 0,
  sample 0): a real 8-bit mono clip, 8000 Hz.

`testdata/files/v2-external-ref.snd` is hand-built, not trimmed from any
real file — see `testdata/README.md` for what it contains and why.

## Corpus scan results (backlog item 004, run 2026-09-25)

A byte-level scan of the full local corpus (483 files that parse as a v1
header) comparing `ParseV1`'s shipped 2-byte Group/Sample reading against
the corrected 4-byte reading, at the level of every individual sound entry:
**482 of 483 files (99.8%) have at least one entry that the 2-byte reading
gets wrong**, and **38,093 of 52,758 entries (72.2%) get a wrong
(Group, Sample) key** — not the rare/inconsequential case that would have
justified downgrading this item. See
`.vibe/decisions/005-v1-group-sample-fields-are-4-bytes-not-2.md` for the
decision this evidence supports and the fix it describes.

`testdata/files/v1-multidigit-sample.snd` is trimmed from the same real
file `v2-basic.snd` uses (`Misc/Popeye/popeye.snd`, entry group 1, sample
143), this time keeping that file's own real v1 header version stamp
instead of synthesizing a v2 one — no synthesizing needed, since the
underlying entry bytes are identical either way (see decision `005`).
