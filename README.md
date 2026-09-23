# snd

A read/write Go library for MUGEN/Ikemen GO sound (`.snd`) files — v1 and v2 parsing and decoding to raw PCM — a shared dependency of the [OpenKakutou](https://github.com/openkakutou) project, needed independently by [`character`](https://github.com/openkakutou/character) (a character's own sounds) and [`mode-quick-versus`](https://github.com/openkakutou/mode-quick-versus) (system/common sound sets). No playback dependency; compiles to WebAssembly.

<!-- vibe:begin:features -->
- Read a `.snd` v1 sound file's header and sound table, addressed the same way a character's own sound-triggering controller addresses sounds
- Decode each sound's embedded audio to raw PCM samples, ready to play back
- Malformed or truncated sound data is reported with a clear error naming the affected sound, never a crash or silently wrong audio
- Validated against real, unmodified community `.snd` files, not just hand-built test data

Planned, see the roadmap's decision `026-scope-org-wide-audio-snd-and-bgm-support` for the scoping:

- Reading `.snd` v2 sound files (sound table, embedded samples, and Ikemen GO's external-audio-file-reference extension)
- A WebAssembly build so web apps can decode sounds without a Go toolchain
<!-- vibe:end:features -->

<!-- vibe:begin:install -->
Requires [Go](https://go.dev/) 1.26 or later.

```sh
go get github.com/openkakutou/snd
```

Verify the install by importing the module in a Go file and running `go build`:

```go
import "github.com/openkakutou/snd"
```

To update to the latest version:

```sh
go get -u github.com/openkakutou/snd
```
<!-- vibe:end:install -->

<!-- vibe:begin:usage -->
Open a `.snd` v1 file, read its sound table, and decode a sound to raw PCM by its `(group, sample)` key:

```go
package main

import (
	"fmt"
	"os"

	"github.com/openkakutou/snd"
)

func main() {
	f, err := os.Open("fighter.snd")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	table, err := snd.ParseV1(f)
	if err != nil {
		panic(err)
	}

	sound, err := snd.DecodeV1Sound(f, table, 0, 0) // group 0, sample 0
	if err != nil {
		panic(err)
	}

	fmt.Printf("decoded %d samples at %d Hz, %d channel(s)\n",
		len(sound.PCM), sound.SampleRate, sound.Channels)
}
```

`.snd` v2 files are not supported yet. ADPCM-encoded sounds are out of
scope by design — see the documentation index below for details.
<!-- vibe:end:usage -->

<!-- vibe:begin:docs-index -->
- [`docs/api.md`](docs/api.md) — the public API for reading `.snd` v1 files and decoding sounds to PCM
- [`docs/testing.md`](docs/testing.md) — the kinds of tests in this repo and how to run and regenerate them
<!-- vibe:end:docs-index -->
