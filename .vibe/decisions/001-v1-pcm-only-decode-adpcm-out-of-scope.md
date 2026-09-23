---
date: 2026-09-23
status: accepted
---
# V1 sound decode supports PCM only; ADPCM is a permanent, documented gap

**Context:** Backlog item 001 asks the v1 `.snd` read path to decode each sound table entry's embedded audio to raw PCM, "correct for both plain PCM and ADPCM-encoded entries" per its acceptance criteria.

**Decision:** `DecodeV1Sound` decodes only WAVE format tag 1 (PCM), at 8-bit unsigned or 16-bit signed depth — the two shapes found in real files. Any other format tag or bit depth returns a descriptive error naming the offending `(group, sample)` entry instead of attempting to decode it.

**Reason:** A full scan of 496 real, unmodified community `.snd` files (54,306 sound entries total) found zero entries using anything but WAVE format tag 1 — every single embedded WAV is plain PCM. Separately, Ikemen GO's own current sound loader decodes each entry's embedded WAV via the `beep` library's WAV decoder, which itself only accepts format tag 1 (PCM) or -2 (WAVEFORMATEXTENSIBLE, treated as PCM) and rejects everything else — so no `.snd` file this library must stay compatible with could rely on ADPCM actually playing in the reference engine either. This mirrors this org's own precedent for `sff`'s RLE5 sprite format (see `sff`'s decision `018-pcx-3-plane-and-confirmed-corrupt-files-are-permanent-gaps.md`): a format described by the wider spec, with no real fixture and no working reference consumer, is a permanent, explicitly named scope cut rather than unverifiable speculative code.

**Rejected alternatives:** Implementing an ADPCM decoder (IMA or Microsoft ADPCM) validated only against a hand-built synthetic fixture, with no real file and no working reference decoder to check it against — rejected because it is exactly the synthetic-only-fixture risk this project's own testing conventions warn against (`sff`/`character` history repeatedly found such fixtures missed real-file bugs), spent on decoding a byte stream Ikemen GO itself cannot play.
