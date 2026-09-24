# Glossary

## Sound reference (group, sample)
A pair of integers identifying one sound in a `.snd` file, the same
addressing scheme `character`'s `cns` `PlaySnd` controller and `engine`'s
`PlaySnd` event use to name a sound to play. `group` collects related
sounds (e.g. all of a character's voice clips); `sample` selects one
within that group. This package resolves the pair to a sound table
position, and to decoded PCM audio, but never interprets what a given
group or sample number *means* — that mapping is each consumer's own
concern.
**Do not confuse with:** a sprite's `(group, image)` key in `sff` — a
different file format's own, unrelated addressing scheme.
_Sources: `v1.go`, `v1_decoder.go`, `v2.go`, `v2_decoder.go`_

## External-file-reference (v2 extension)
An Ikemen GO extension to the `.snd` v2 sound table: instead of embedding a
sound's own audio bytes, a table entry can instead carry a relative path
naming another audio file to play in its place. This package resolves such
an entry the same way it resolves an embedded one, from the caller's
perspective — the caller supplies how the path is actually opened (a local
directory, a bundled archive, ...), since this package never does file or
network I/O itself.
**Do not confuse with:** `sff`'s external `.act` palette override — a
different file format's own, unrelated "point outside this file" mechanism.
_Sources: `v2_decoder.go`_
