---
status: done
---
# Implement V1 Read Path (Parse + PCM Decode)

## Description
No `.snd` parsing exists anywhere in the org yet — this is from-scratch implementation against the MUGEN `.snd` v1 format spec and Ikemen GO's own reference source (`sound.go`), not a migration like `sff`'s equivalent item was. Read a v1 `.snd` file's header and sound table (group/sample-indexed entries), and decode each entry's embedded audio (PCM, and ADPCM where used) to raw PCM samples.

## Acceptance Criteria
- [x] A v1 `.snd` file's header and sound table parse into a pure-data model, keyed by `(group, sample)` the same way `character/cns`'s `PlaySnd` controller already addresses sounds
- [x] Each table entry's embedded audio decodes to raw PCM (`[]int16` or equivalent), correct for plain PCM entries — ADPCM decode is a deliberate, documented scope cut, see the note below
- [x] A malformed header or truncated sample data returns a descriptive error naming the offending entry, not a panic or silently wrong audio
- [x] Validated against real, unmodified community `.snd` files (vendored as trimmed fixtures in `testdata/`), not only hand-built synthetic data

## Notes
No dependency on item `002` (v2 read path) — v1 and v2 are structurally separate formats, same relationship as `sff`'s own v1/v2 split. Feeds `character#057` and, through it, `character-editor#017` and `mode-quick-versus#013`.

## Resolution
A corpus scan of 496 real, unmodified community `.snd` files (54,306 sound entries) found zero ADPCM-encoded entries — every one is plain PCM. Ikemen GO's own current reference sound loader also only accepts PCM. ADPCM decode is therefore a deliberate, permanent scope cut rather than unverifiable code validated only against a hand-built synthetic fixture — see `.vibe/decisions/001-v1-pcm-only-decode-adpcm-out-of-scope.md`. A non-PCM entry decodes to a descriptive error instead of wrong or missing audio, satisfying the "no silently wrong audio" requirement.
