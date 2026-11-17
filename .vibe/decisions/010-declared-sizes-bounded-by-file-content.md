---
date: 2026-11-14
status: accepted
---
# Declared sizes are bounded by what the file holds, not by a fixed cap

**Context:** Backlog item 009 asks that a malicious `.snd` cannot exhaust memory in the browser. The parsers preallocated the sound table from the declared sound count and followed next-pointers without cycle detection; the decoders allocated each sound's declared length before reading it.

**Decision:** Nothing is allocated from a declared count or length. The sound table grows only as subheaders are actually read, a next-pointer that revisits an offset already read is refused, and a sound's bytes are read incrementally, refused when the file holds fewer than declared. Fuzz targets cover v1/v2 parsing and decoding; their seeds run under `go test`, and the release workflow adds a short fuzzing pass.

**Reason:** Bounding by file content needs no magic number and cannot reject a legitimate file, whatever its size, so MUGEN and Ikemen GO files read exactly as before. Revisit detection makes no assumption on pointer direction, which no source guarantees for real files.

**Rejected alternatives:** A fixed cap on sound count or length (arbitrary, could reject a large legitimate file, and still allocates up to the cap); requiring strictly increasing next-pointers (cheaper, but assumes a file layout neither MUGEN nor Ikemen GO is known to enforce).
