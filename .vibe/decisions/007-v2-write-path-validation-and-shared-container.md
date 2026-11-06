---
date: 2026-11-06
status: accepted
---
# v2 write path shares the v1 container writer and validates strictly

**Context:** Backlog item 006 asks to write `.snd` v2 files, keep Ikemen GO external-file references as references, and reject invalid sets.

**Decision:** v1 and v2 share one container layout, so both encoders call one internal writer and both readers one internal reader. `V2Sound` holds either embedded `Audio` or an `ExternalPath`, never both. `EncodeV2` refuses: a duplicate (group, sample) key, an empty payload, both kinds at once, embedded audio that is not RIFF/WAVE, an external path that is not printable ASCII ending in a recognized audio extension, and a version stamp whose first byte is not 2 (`DecodeSound` would read it as v1). A zero version means the default stamp `2,0,0,0`. `ReadV2Set` stays lenient and keeps malformed entries.

**Reason:** `EncodeV2` output must read back exactly as written through `DecodeV2Sound` and `DecodeSound`, which classify a payload by content. Strictness lives in the encoder, so a read-then-write can fail loudly instead of writing a file that decodes differently.

**Rejected alternatives:** Accepting any bytes as embedded audio like `EncodeV1` (a garbage payload could later be misread as a path). Keeping a trailing NUL on external paths (byte-exact preservation belongs to item 007). Duplicating the container code per version.
