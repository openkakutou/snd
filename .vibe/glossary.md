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
_Sources: `v1.go`, `v1_decoder.go`_
