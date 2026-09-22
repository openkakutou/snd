# snd

A read/write Go library for MUGEN/Ikemen GO sound (`.snd`) files — v1 and v2 parsing and decoding to raw PCM — a shared dependency of the [OpenKakutou](https://github.com/openkakutou) project, needed independently by [`character`](https://github.com/openkakutou/character) (a character's own sounds) and [`mode-quick-versus`](https://github.com/openkakutou/mode-quick-versus) (system/common sound sets). No playback dependency; compiles to WebAssembly.

<!-- vibe:begin:features -->
This project is in early-stage development — no functionality yet, see the roadmap's decision `026-scope-org-wide-audio-snd-and-bgm-support` for the scoping.

Planned:

- Reading `.snd` v1 sound files (header, sound table, PCM/ADPCM sample decode)
- Reading `.snd` v2 sound files (sound table, embedded samples, and Ikemen GO's external-audio-file-reference extension)
- Decoding samples to raw PCM
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
No functionality is implemented yet — the parsing/decoding API will be documented here as it lands. For now the package only exposes its version:

```go
package main

import (
	"fmt"

	"github.com/openkakutou/snd"
)

func main() {
	fmt.Println(snd.Version)
}
```
<!-- vibe:end:usage -->

<!-- vibe:begin:docs-index -->
No additional documentation yet.
<!-- vibe:end:docs-index -->
