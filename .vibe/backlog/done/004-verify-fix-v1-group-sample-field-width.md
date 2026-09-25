---
status: done
---
# Verify and Fix V1 Group/Sample Field Width Against Real Files

## Description
While implementing item 002 (v2 read path), evidence surfaced that `ParseV1`'s sound-table subheader layout (2-byte Group, 2-byte Sample, 4 bytes unread as "reserved") does not match how real `.snd` files actually store these fields. Ikemen GO's own reference loader (`src/sound.go`, `LoadSndFiltered`) reads Group and Sample as two adjacent 4-byte fields filling the whole subheader, with nothing reserved. A direct byte-level check of a real, unmodified file (`Teenage Mutant Ninja Turtles/Leonardo/TMNT Leonardo.snd`) confirms this: several consecutive entries' true Sample values (1, 2, 3, 4, 5, 6, 13, 7, 8) live in the 4 bytes `ParseV1` currently ignores as reserved, while the bytes it reads as "Sample" are actually the unused upper half of a small Group value. In practice this means `ParseV1` likely returns `Sample = 0` for most entries in most real files once Group is small — see `.vibe/decisions/002-v2-subheader-uses-4-byte-group-and-sample-fields.md` for the full evidence trail.

## Acceptance Criteria
- [x] Confirm at corpus scale (not just the one file above) whether `ParseV1` misreads `Sample` for real files, and how often — confirmed: 482 of 483 real v1 files (99.8%), 38,093 of 52,758 entries (72.2%)
- [x] If confirmed, `ParseV1` reads Group and Sample as 4-byte fields (matching Ikemen GO's reference loader and v2's own subheader layout), with existing v1 fixtures/tests updated or extended as needed
- [x] A regression test using a real fixture with a non-zero, multi-digit Sample value (e.g. group 1, sample 143, from the real Popeye character file already vendored as `testdata/files/v2-basic.snd`, or an equivalent real v1 fixture) demonstrates the corrected reading — `testdata/files/v1-multidigit-sample.snd`, same real entry under a real v1 header
- [x] `character#057` and any other consumer relying on `ParseV1`'s current (group, sample) keys are flagged if this is a breaking change to already-decoded data — flagged in decision `005`; `character` exposes `ParseV1`'s keys unchanged through its own `Sounds`/WASM contract

## Notes
Depends on nothing to *investigate*, but a fix changes already-shipped, closed item 001's behavior — treat it as its own ticket (bug fix workflow), not a silent side effect of unrelated work. If the corpus-scale check finds this is rare/inconsequential in practice (e.g. because real characters happen to keep Sample small enough that it doesn't matter, which the one file checked so far contradicts), downgrade this item accordingly instead of forcing a fix.
