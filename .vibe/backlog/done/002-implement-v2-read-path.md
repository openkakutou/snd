---
status: done
---
# Implement V2 Read Path (Parse + PCM Decode, Incl. Ikemen GO External-File Extension)

## Description
Same shape as item `001` for `.snd` v2: header + sound table parsing, PCM decode of embedded entries. V2 additionally supports Ikemen GO's own extension where a sound table entry points at an external audio file (e.g. `.wav`/`.ogg`) instead of embedding samples — part of this org's stated "MUGEN 1.0/1.1 and Ikemen GO" compatibility bar, not an optional add-on.

## Acceptance Criteria
- [x] A v2 `.snd` file's header and sound table parse into the same `(group, sample)`-keyed data model item `001` establishes for v1
- [x] Each table entry's embedded audio decodes to raw PCM, matching item `001`'s decode contract
- [x] An entry using Ikemen GO's external-file-reference extension resolves and decodes the referenced file's audio the same way an embedded entry does, from the caller's perspective (same return shape)
- [x] A referenced external file that can't be found or decoded returns a descriptive error naming the entry and the missing/invalid path, not a panic or silent empty audio
- [x] Validated against at least one real, unmodified community `.snd` v2 file (vendored as a trimmed fixture in `testdata/`); the external-file-reference case validated against a real Ikemen GO character if one is found, otherwise a clearly-marked synthetic fixture with the gap noted (mirroring `sff`'s own accepted-gap precedent for RLE5, `.vibe/decisions/014`)

## Notes
No dependency on item `001` — v1 and v2 are structurally separate formats. Feeds `character#057` and, through it, `character-editor#017` and `mode-quick-versus#013`.

## Resolution
`ParseV2`/`DecodeV2Sound` implemented, mirroring `ParseV1`/`DecodeV1Sound`'s shape. No authoritative spec for either v2's exact subheader layout or the external-file-reference extension's encoding could be found (checked: Elecbyte's docs, Ikemen GO's own reference loader on both `develop` and `release/1.0`, public web search, GitHub issues), so both were pinned down against the best available evidence and documented as ADRs:

- `.vibe/decisions/002` — v2's subheader stores `Group`/`Sample` as 4-byte fields (not v1's 2-byte-plus-reserved layout), confirmed both by Ikemen GO's own reference loader source and a direct byte-level check of a real, unmodified character file. This also surfaced a likely latent bug in the already-shipped v1 path (tracked as new backlog item `004`, not fixed here).
- `.vibe/decisions/003` — external-file-reference entries are detected by a path-shaped payload heuristic (no binary flag exists to distinguish them) and resolved through a caller-supplied `ExternalAudioOpener` callback, keeping this package WASM-safe; only a WAV/PCM external file actually decodes, mirroring decision `001`'s ADPCM scope cut.

Validated against `testdata/files/v2-basic.snd` (real audio and real sound-table bytes from a real character file; only the header's version stamp is synthesized, since no file in the 496-file local corpus declares itself version 2). No real character using the external-file-reference extension was found, so that case is validated against a clearly-marked synthetic fixture (`v2-external-ref.snd`) pointing at real, unmodified audio (`v2-external-audio.wav`), per this item's own acceptance criteria and mirroring `sff`'s decision `014` precedent.
