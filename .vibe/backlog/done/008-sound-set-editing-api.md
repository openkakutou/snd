---
status: done
depends_on: [005,006]
---
# Add, replace and remove sounds API and WASM export

## Description
Provide the editing operations editors need: add a sound (group/number + WAV/PCM), replace its audio, remove it, with validation of format (sample rate, channels, bit depth) and expose them through the WASM build.

## Acceptance Criteria
- [x] Add/replace/remove are available in Go and WASM
- [x] Non-conformant audio is rejected or converted per a documented rule
- [x] Document round trip stays valid after edits

## Notes
Consumer: character-editor sound import/replace.
