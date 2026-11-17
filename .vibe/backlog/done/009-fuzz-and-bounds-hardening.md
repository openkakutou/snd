---
status: done
---
# Fuzz and bound-check the decoders

## Description
Add Go fuzz targets for v1/v2 parse and decode (including WAV and external-file paths) and enforce limits on declared sizes so a malicious file cannot exhaust memory in the browser.

## Acceptance Criteria
- [x] Fuzz targets for v1 and v2 exist and run in CI
- [x] Declared sizes beyond file length or a sane cap return an error
- [x] No panic on truncated input
