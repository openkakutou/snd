---
date: 2026-11-06
status: accepted
---
# Editing API refuses audio it cannot decode instead of converting it

**Context:** Backlog item 008 asks for add, replace and remove operations with format validation (sample rate, channels, bit depth), "rejected or converted per a documented rule", in Go and WASM.

**Decision:** The operations live on `Document` and work by `(group, sample)` key. New audio must be a RIFF/WAVE blob that the existing decoder reads (plain PCM, 8 or 16 bits, at least one channel) with a non-zero sample rate; anything else is refused, never converted. A refused edit leaves the document unchanged. Add refuses an existing key; replace and remove refuse a missing one. The WASM build exposes `addSound`, `replaceSound` and `removeSound` over bytes, returning `{ bytes, error }`.

**Reason:** The check reuses the decoder, so what the API accepts is exactly what the library can read back. Conversion needs a resampler and codecs, which the stdlib-only, playback-free design (CLAUDE.md constraints 1 and 2) rules out.

**Rejected alternatives:** Converting to a canonical format on the way in (extra code and dependencies, lossy). Restricting sample rate or channel count (real files vary; the decoder does not care). Separate edit-by-index functions (a key is how consumers address sounds).
