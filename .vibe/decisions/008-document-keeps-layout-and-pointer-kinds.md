---
date: 2026-11-06
status: accepted
---
# Document keeps gaps, trailing bytes and what each last pointer pointed at

**Context:** Backlog item 007 asks for a Document whose untouched round trip is byte-identical and whose edits change only the edited sound's entry and payload. The sound-only sets of items 005/006 drop bytes between sounds and after the last sound, and normalize the last pointer.

**Decision:** `ParseDocument(data []byte)` keeps the header version bytes and comment, the gap after each sound, the bytes after the last sound, and the kind of each last pointer (end of file, zero, or a literal). `Serialize` rebuilds offsets from that: pointers between sounds follow the layout, an end-of-file pointer follows the new end. `SetPayload` is the only edit; it copies its input. v1 and v2 share the layout, so one Document serves both. A file whose sounds are not in increasing offset order is refused as an unsupported layout. A corpus check gated by `SND_CORPUS_DIR` round-tripped 496 real files byte-identical.

**Reason:** The data slice is needed to know the end of the file and what follows the last sound. Keeping the pointer kind (rather than always writing end of file) keeps trimmed fixtures byte-identical too.

**Rejected alternatives:** An io.ReaderAt input (file size unknown, so trailing bytes cannot be kept). Keeping the raw file and patching it in place (growing a payload shifts every later offset anyway). Supporting out-of-order layouts (none among 496 real files).
