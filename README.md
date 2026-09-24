# snd

A read/write Go library for MUGEN/Ikemen GO sound (`.snd`) files — v1 and v2 parsing and decoding to raw PCM — a shared dependency of the [OpenKakutou](https://github.com/openkakutou) project, needed independently by [`character`](https://github.com/openkakutou/character) (a character's own sounds) and [`mode-quick-versus`](https://github.com/openkakutou/mode-quick-versus) (system/common sound sets). No playback dependency; compiles to WebAssembly.

<!-- vibe:begin:features -->
- Read a `.snd` v1 or v2 sound file's header and sound table, addressed the same way a character's own sound-triggering controller addresses sounds
- Decode each sound's embedded audio to raw PCM samples, ready to play back
- Decode a sound without knowing ahead of time whether its file is v1 or v2 — the version is auto-detected
- A v2 sound can also point at an external audio file instead of embedding its samples (Ikemen GO's own extension) — resolving and decoding it works the same way, from the caller's perspective, as an embedded sound
- Malformed or truncated sound data, or a missing/invalid external audio file, is reported with a clear error naming the affected sound, never a crash or silently wrong audio
- Validated against real, unmodified community `.snd` files, not just hand-built test data
- A WebAssembly build (downloadable from each release) so web apps and other JS hosts can decode sounds directly, without a Go toolchain of their own
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

For a `.snd` v2 file, use `snd.ParseV2` and `snd.DecodeV2Sound` instead. A v2
sound may point at an external audio file instead of embedding its samples;
pass a function that resolves that file's path to its bytes however your
own application does (`nil` if you know the file has no such sounds):

```go
sound, err := snd.DecodeV2Sound(f, table, 0, 0, func(path string) ([]byte, error) {
	return os.ReadFile(path) // or fetch it however your app stores assets
})
```

ADPCM-encoded sounds, and an external file in a format other than WAV/PCM,
are out of scope by design — see the documentation index below for details.

If you don't know ahead of time whether a file is v1 or v2, `DecodeSound`
detects it for you:

```go
sound, err := snd.DecodeSound(f, 0, 0, nil) // group 0, sample 0
```

For decoding sounds directly in a browser or other JS host, without a Go
toolchain, see the WebAssembly documentation linked below.
<!-- vibe:end:usage -->

<!-- vibe:begin:docs-index -->
- [`docs/api.md`](docs/api.md) — the public API for reading `.snd` v1 and v2 files and decoding sounds to PCM, including external audio files
- [`docs/testing.md`](docs/testing.md) — the kinds of tests in this repo and how to run and regenerate them
- [`docs/wasm.md`](docs/wasm.md) — building and using the WebAssembly entrypoint, its JS API, and the release pipeline that publishes it
<!-- vibe:end:docs-index -->
