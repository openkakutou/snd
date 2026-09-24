---
status: done
depends_on: [001, 002]
---
# WASM Entrypoint And Release Pipeline

## Description
This repo needs its own WASM build and tagged-release pipeline, mirroring `sff`'s `cmd/wasm/` + `.github/workflows/release.yml` pattern, since `mode-quick-versus` needs to decode sound directly (system/common sound sets, not routed through `character`), independent of `character`'s own WASM build.

## Acceptance Criteria
- [x] `GOOS=js GOARCH=wasm` build produces a working `snd.wasm` + `wasm_exec.js`, exposing a version-agnostic decode call (v1/v2 auto-detected, mirroring `sff.Load`'s own auto-detection) as a JS-callable global
- [x] A Node-based smoke test verifies it loads and decodes a real fixture sound file
- [x] A GitHub Actions workflow publishes both artifacts on every tag, mirroring `sff`'s release workflow

## Notes
Depends on both `001` and `002` since the exposed decode call needs to handle whichever version a real file turns out to be, the same way `sff.Load` is version-agnostic. Cross-repo: this is what unblocks `mode-quick-versus#013`'s direct dependency on `snd` (system/common sounds); `character#057` only needs the native Go API (items `001`/`002`), not this WASM build, since `character`'s own WASM build is what web consumers actually use.
