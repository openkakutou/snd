---
date: 2026-11-03
status: accepted
---
# v1 write path follows the layout of real files and keeps the header comment

**Context:** Backlog item 005 asks to write `.snd` v1 files, with real unmodified files re-encoding byte-identical. A scan of the local corpus (483 v1-shaped files) showed every file is contiguous with no trailing bytes, has its first subheader at offset 512 (a comment fills bytes 24..511), and has a last subheader whose next-offset equals the file length.

**Decision:** `ReadV1Set` loads a file into a `V1SoundSet` (version bytes, header padding, sounds with raw audio) and `EncodeV1` writes it back: header, padding verbatim, then contiguous subheaders, the last one pointing at the end of the file. Duplicate keys and non-WAV payloads are kept as they are. Only a group/sample outside the signed 32-bit range or a file past 4 GiB is refused. A corpus check gated by `SND_CORPUS_DIR` re-encoded 496 real files byte-identical.

**Reason:** Byte identity needs the header comment and the end-of-file pointer, which `ParseV1` does not keep. Duplicates and odd payloads exist in real files, so refusing them would break the round trip.

**Rejected alternatives:** Padding every new file to 512 bytes by default (a hidden default; the caller sets padding). Rejecting duplicate keys now (belongs to the editing API, item 008). A full Document preserving unknown bytes and gaps (item 007).
