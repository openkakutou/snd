---
status: done
---
# Write .snd v1 files

## Description
snd is read-only today although the org describes it as read/write (mirroring sff). Implement encoding of a v1 sound file from an in-memory model: header, sound table and raw WAV payloads.

## Acceptance Criteria
- [x] A sound set can be serialised to a valid v1 .snd
- [x] Parse -> serialise -> parse yields identical sounds
- [x] Real unmodified v1 files re-encode byte-identical when untouched
