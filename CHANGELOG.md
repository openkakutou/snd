# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-09-23

### Added

- Read a `.snd` v1 file's header and sound table, keyed by `(group, sample)`, and decode each entry's embedded audio to raw PCM samples. Malformed headers or truncated sample data return a descriptive error naming the offending entry instead of panicking or producing silently wrong audio. Validated against real, unmodified community `.snd` files. ADPCM-encoded entries are not supported — real files and Ikemen GO's own reference sound loader only ever use plain PCM, so this is a documented, permanent scope cut rather than unverifiable speculative code.

[Unreleased]: https://github.com/openkakutou/snd/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/openkakutou/snd/releases/tag/v0.1.0
