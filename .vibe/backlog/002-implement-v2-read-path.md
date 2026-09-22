---
status: todo
---
# Implement V2 Read Path (Parse + PCM Decode, Incl. Ikemen GO External-File Extension)

## Description
Same shape as item `001` for `.snd` v2: header + sound table parsing, PCM decode of embedded entries. V2 additionally supports Ikemen GO's own extension where a sound table entry points at an external audio file (e.g. `.wav`/`.ogg`) instead of embedding samples — part of this org's stated "MUGEN 1.0/1.1 and Ikemen GO" compatibility bar, not an optional add-on.

## Acceptance Criteria
- [ ] A v2 `.snd` file's header and sound table parse into the same `(group, sample)`-keyed data model item `001` establishes for v1
- [ ] Each table entry's embedded audio decodes to raw PCM, matching item `001`'s decode contract
- [ ] An entry using Ikemen GO's external-file-reference extension resolves and decodes the referenced file's audio the same way an embedded entry does, from the caller's perspective (same return shape)
- [ ] A referenced external file that can't be found or decoded returns a descriptive error naming the entry and the missing/invalid path, not a panic or silent empty audio
- [ ] Validated against at least one real, unmodified community `.snd` v2 file (vendored as a trimmed fixture in `testdata/`); the external-file-reference case validated against a real Ikemen GO character if one is found, otherwise a clearly-marked synthetic fixture with the gap noted (mirroring `sff`'s own accepted-gap precedent for RLE5, `.vibe/decisions/014`)

## Notes
No dependency on item `001` — v1 and v2 are structurally separate formats. Feeds `character#057` and, through it, `character-editor#017` and `mode-quick-versus#013`.
