---
date: 2026-09-25
status: accepted
---
# ParseV1 reads Group and Sample as 4-byte fields, matching v2 and real files

**Context:** Backlog item 004 asked to verify, at corpus scale, whether `ParseV1`'s shipped subheader reading (2-byte Group, 2-byte Sample, 4 bytes left unread as "reserved") misreads real `.snd` v1 files — a discrepancy decision `002` first surfaced while building the v2 read path, based on a single real file (`TMNT Leonardo.snd`).

**Corpus-scale evidence:** A byte-level scan of the full local corpus (`.vibe/fixture-sources.md`, 483 files that parse as a v1-style header) compared the currently-shipped 2-byte reading against the corrected 4-byte reading for every sound entry:

- **482 of 483 files (99.8%) have at least one entry whose (Group, Sample) the 2-byte reading gets wrong.**
- **38,093 of 52,758 entries (72.2%)** get a wrong (Group, Sample) key under the 2-byte reading — the true Sample value's low half is read as Group's high 16 bits and the true Sample's own low bits are skipped as "reserved", so most entries collapse onto `(Group, Sample=0)` the moment a group holds more than a couple of samples.
- Runtime replay against the exact file backlog item 004's report cites (`TMNT Leonardo.snd`) confirms the corrected reading recovers the reported sequence: Group 1's samples read `0, 1, 2, 3, 4, 5, 6, 13, 7, 8, 9, 10, 11, 12` — matching the file's real, sequential sample indices — instead of collapsing to 0.

This is not a rare edge case: it affects nearly every real v1 file with more than a couple of samples per group, so the item is not downgraded — it is fixed.

**Decision:** `ParseV1`'s subheader layout changes to match `ParseV2`'s exactly: `NextSubHeaderOffset` (4 bytes) + `SubFileLength` (4 bytes) + `Group` (4-byte little-endian signed int) + `Sample` (4-byte little-endian signed int) — 16 bytes total, nothing reserved. This is the same layout Ikemen GO's own reference loader (`src/sound.go`) already applies unconditionally, and the same one decision `002` established for v2. v1 and v2 subheaders are now identical in shape; only the file header's own version stamp (largely decorative — see decision `002`) differs.

Existing fixtures (`v1-basic.snd`, `v1-8bit.snd`) are unaffected: both use small Group/Sample values (fitting entirely within 16 bits with the rest zeroed), so they parse identically under the old and new reading. A new fixture, `testdata/files/v1-multidigit-sample.snd`, sourced from a real, genuinely v1-declared file (`Misc/Popeye/popeye.snd`, entry group 1, sample 143 — the same real entry `v2-basic.snd` uses, but this time under a real, not synthesized, v1 header), demonstrates the corrected reading: 143 does not fit in the old 2-byte-alongside-Group field and would have silently read as Sample 0.

**Breaking change flagged for a downstream consumer:** `character` (backlog item 057, already shipped) calls `snd.ParseV1`/`DecodeV1Sound` directly and exposes the decoded `(Group, Sample)` keys unchanged through its own `Character.Sounds`/WASM JSON contract. Once `character` upgrades to a `snd` release containing this fix, any already-decoded v1 character sound data it exposes will change: entries previously colliding onto `(group, 0)` will now appear under their real, distinct Sample indices. This is a correctness fix, not a regression, but it changes `character`'s own output shape for real files — flagged here so `character`'s own maintainers can track it (a corresponding backlog note is expected on that repo's side, out of scope to add from here per this repo's domain-independence constraint).

**Reason:** The org's compatibility bar (`CLAUDE.md`, design constraint 4: "real-file compatibility over spec purity") requires fixing a confirmed, high-incidence real-file bug rather than leaving a shipped item's known-wrong behavior in place once corpus-scale evidence rules out "rare/inconsequential".

**Rejected alternatives:**
- *Leave `ParseV1` as shipped, since item 001 is closed* — rejected: the corpus scan shows this is not a rare edge case (72% of entries affected); leaving a confirmed, high-incidence bug in place because a ticket is closed contradicts the org's own real-file-compatibility bar.
- *Downgrade item 004 per its own "if rare/inconsequential" escape hatch* — rejected: the escape hatch's own precondition (rare/inconsequential) does not hold; the evidence points the other way.
