---
status: done
depends_on: [005]
---
# Write .snd v2 files, including external-file references

## Description
Implement encoding for the v2 container including Ikemen GO's external audio file extension (a sound that points to a file instead of embedding samples).

## Acceptance Criteria
- [x] A v2 sound set is serialised and re-read identically
- [x] External-file references are preserved, not inlined
- [x] Invalid sets (duplicate group/number, empty payload) are rejected with a clear error
