---
status: todo
---
# Implement V1 Read Path (Parse + PCM Decode)

## Description
No `.snd` parsing exists anywhere in the org yet — this is from-scratch implementation against the MUGEN `.snd` v1 format spec and Ikemen GO's own reference source (`sound.go`), not a migration like `sff`'s equivalent item was. Read a v1 `.snd` file's header and sound table (group/sample-indexed entries), and decode each entry's embedded audio (PCM, and ADPCM where used) to raw PCM samples.

## Acceptance Criteria
- [ ] A v1 `.snd` file's header and sound table parse into a pure-data model, keyed by `(group, sample)` the same way `character/cns`'s `PlaySnd` controller already addresses sounds
- [ ] Each table entry's embedded audio decodes to raw PCM (`[]int16` or equivalent), correct for both plain PCM and ADPCM-encoded entries
- [ ] A malformed header or truncated sample data returns a descriptive error naming the offending entry, not a panic or silently wrong audio
- [ ] Validated against at least one real, unmodified community `.snd` file (vendored as a trimmed fixture in `testdata/`), not only hand-built synthetic data — this org's own history (`sff`, `character`) repeatedly found synthetic-only fixtures missed real-file bugs

## Notes
No dependency on item `002` (v2 read path) — v1 and v2 are structurally separate formats, same relationship as `sff`'s own v1/v2 split. Feeds `character#057` and, through it, `character-editor#017` and `mode-quick-versus#013`.
