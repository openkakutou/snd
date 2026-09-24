# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-09-24

### Added

- A `.snd` sound can now be decoded without knowing ahead of time whether the file is v1 or v2 — the version is auto-detected the same way the file format's other tooling already does. This is also what powers a new WebAssembly build (`snd.wasm` + `wasm_exec.js`): any JS host (a browser, a game engine's scripting layer) can decode a sound directly, with no Go toolchain of its own, including resolving a v2 sound's external-audio-file extension via a caller-supplied callback. A GitHub Actions workflow publishes both files as release assets on every tagged release, gated on a real smoke test of the built artifact.

## [0.2.0] - 2026-09-24

### Added

- Read a `.snd` v2 file's header and sound table, keyed by `(group, sample)`, and decode each entry's embedded audio to raw PCM samples, the same as v1. A v2 entry can also use Ikemen GO's own extension letting it point at an external audio file instead of embedding samples: given a caller-supplied opener for resolving that file's bytes (this library never touches the filesystem or network itself, to stay usable from a browser), the referenced file's audio decodes exactly like an embedded entry. A missing, unresolvable, or unsupported-format external file returns a descriptive error naming the entry and the path, never a panic or silent empty audio. Validated against a real, unmodified community `.snd` file; no real character using the external-file-reference extension could be found, so that case is validated against a clearly-marked synthetic reference pointing at real, unmodified audio instead.

## [0.1.0] - 2026-09-23

### Added

- Read a `.snd` v1 file's header and sound table, keyed by `(group, sample)`, and decode each entry's embedded audio to raw PCM samples. Malformed headers or truncated sample data return a descriptive error naming the offending entry instead of panicking or producing silently wrong audio. Validated against real, unmodified community `.snd` files. ADPCM-encoded entries are not supported — real files and Ikemen GO's own reference sound loader only ever use plain PCM, so this is a documented, permanent scope cut rather than unverifiable speculative code.

[Unreleased]: https://github.com/openkakutou/snd/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/openkakutou/snd/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/openkakutou/snd/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/openkakutou/snd/releases/tag/v0.1.0
