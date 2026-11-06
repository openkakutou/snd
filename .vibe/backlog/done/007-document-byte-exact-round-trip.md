---
status: done
depends_on: [005,006]
---
# Byte-exact Document round trip for .snd

## Description
Follow the Document/ParseDocument pattern used by character: an unmodified file round-trips byte-for-byte, preserving unknown header bytes and ordering, so editors can save without damaging files.

## Acceptance Criteria
- [x] ParseDocument + Serialize is byte-identical on the real-file corpus
- [x] Modifying one sound only changes that sound's table entry and payload
- [x] Unknown or padding bytes are retained
