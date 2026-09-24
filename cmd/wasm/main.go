//go:build js && wasm

// Command wasm is the WASM entrypoint for the snd library: thin syscall/js
// glue exposing this package's version-agnostic DecodeSound to a browser
// (or any JS host) as a single JS-callable global function, so a consumer
// (mode-quick-versus, for system/common sound sets not routed through any
// one character) can decode a MUGEN/Ikemen GO .snd sound file without a Go
// toolchain of its own — independent of character's own WASM build, which
// consumes this same package internally but publishes its own artifact.
//
// It carries no decoding logic beyond argument conversion, calling into the
// root snd package, and marshaling results to JS — all real behavior lives
// in that package, which is unit-tested independently of this file (see
// .vibe/decisions/004-wasm-entrypoint-api-shape-and-version-detection.md,
// which mirrors the shape sff's own cmd/wasm already established). This
// file's own behavior is instead verified by smoke.mjs, a Node.js script
// that loads the built module the way a real JS consumer would —
// syscall/js code cannot run under the plain `go test` toolchain.
package main

import (
	"fmt"
	"syscall/js"

	snd "github.com/openkakutou/snd"
)

func main() {
	js.Global().Set("OpenKakutouSnd", js.ValueOf(map[string]any{
		"decode": js.FuncOf(decode),
	}))

	// Registering the js.FuncOf callback does not keep the Go runtime alive
	// on its own; block forever so OpenKakutouSnd.decode keeps working for
	// the lifetime of the page.
	select {}
}

// decode is OpenKakutouSnd.decode(sndBytes, group, sample, resolveExternal)
// as seen from JS:
//
//   - sndBytes is a Uint8Array (or any JS value js.CopyBytesToGo accepts)
//     holding a .snd file's raw bytes — version 1 or version 2,
//     auto-detected the same way snd.DecodeSound detects it.
//   - group, sample are the sound's (group, sample) key, as JS numbers.
//   - resolveExternal is optional: a JS function (path string) => Uint8Array
//     resolving a v2 entry's Ikemen GO external-file-reference path to that
//     file's bytes, called synchronously via js.Value.Invoke. Because the
//     call is synchronous, resolveExternal cannot be backed by fetch/a
//     Promise — the caller must already hold the external file's bytes
//     (typically preloaded the same way sndBytes itself was) before calling
//     decode. Passing undefined/null (or omitting the argument) means "no
//     external references expected", matching snd.DecodeSound's own
//     nil-opener contract; an entry that turns out to need one anyway
//     reports a descriptive error instead of panicking.
//
// Always returns a JS object shaped
// { sampleRate, channels, bitsPerSample, pcm, error } — exactly one of
// pcm/error is non-null, the others are 0/null on error — never throws and
// never lets an internal panic escape to the JS caller.
func decode(this js.Value, args []js.Value) (result any) {
	defer func() {
		// A panic here would otherwise propagate out of the js.Func
		// callback and tear down the whole page's WASM instance —
		// mode-quick-versus keeps one instance alive for a whole session
		// and calls decode repeatedly, so this boundary's own recovery
		// must never let one bad call break every call after it.
		if r := recover(); r != nil {
			result = decodeResult(nil, fmt.Errorf("OpenKakutouSnd.decode: panic: %v", r))
		}
	}()

	if len(args) < 3 {
		return decodeResult(nil, fmt.Errorf("OpenKakutouSnd.decode: expected at least 3 arguments (sndBytes, group, sample[, resolveExternal]), got %d", len(args)))
	}

	sndBytes, err := bytesFromJS(args[0])
	if err != nil {
		return decodeResult(nil, fmt.Errorf("OpenKakutouSnd.decode: sndBytes: %w", err))
	}

	opener, err := externalOpenerFromJS(args)
	if err != nil {
		return decodeResult(nil, fmt.Errorf("OpenKakutouSnd.decode: resolveExternal: %w", err))
	}

	sound, err := snd.DecodeSound(readerAtBytes(sndBytes), args[1].Int(), args[2].Int(), opener)
	if err != nil {
		return decodeResult(nil, err)
	}
	return decodeResult(sound, nil)
}

// readerAtBytes adapts a plain []byte to io.ReaderAt, the interface every
// snd package entry point (DecodeSound included) reads from.
type readerAtBytes []byte

func (r readerAtBytes) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(r)) {
		return 0, fmt.Errorf("offset %d out of range (length %d)", off, len(r))
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, fmt.Errorf("short read at offset %d: got %d bytes, wanted %d", off, n, len(p))
	}
	return n, nil
}

// externalOpenerFromJS builds a snd.ExternalAudioOpener from decode's
// optional 4th argument, or returns (nil, nil) when it is absent, JS
// undefined, or JS null — the ways a caller says "no external references
// expected", matching snd.DecodeV2Sound's own nil-opener contract. Any other
// non-function value is a descriptive error rather than a confusing failure
// deep inside js.Value.Invoke.
func externalOpenerFromJS(args []js.Value) (snd.ExternalAudioOpener, error) {
	if len(args) < 4 || args[3].IsUndefined() || args[3].IsNull() {
		return nil, nil
	}
	fn := args[3]
	if fn.Type() != js.TypeFunction {
		return nil, fmt.Errorf("expected a function, got %v", fn)
	}

	return func(path string) (b []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				b, err = nil, fmt.Errorf("resolveExternal(%q): panic: %v", path, r)
			}
		}()

		v := fn.Invoke(path)
		if v.Type() != js.TypeObject || v.Get("constructor").Get("name").String() != "Uint8Array" {
			return nil, fmt.Errorf("resolveExternal(%q): expected a synchronous Uint8Array return value, got %v (an async/Promise-returning resolveExternal is not supported)", path, v)
		}
		return bytesFromJS(v)
	}, nil
}

// bytesFromJS copies a JS Uint8Array-like value into a Go []byte via
// js.CopyBytesToGo, the standard syscall/js conversion. It returns a
// descriptive error instead of panicking if v is not a byte-array-like
// value (e.g. undefined, or missing a numeric "length").
func bytesFromJS(v js.Value) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("expected a byte array, got %v (%v)", v, r)
		}
	}()

	length := v.Get("length").Int()
	buf := make([]byte, length)
	js.CopyBytesToGo(buf, v)
	return buf, nil
}

// decodeResult builds this module's
// { sampleRate, channels, bitsPerSample, pcm, error } JS return shape.
// Exactly one of sound/err is expected to be non-nil.
func decodeResult(sound *snd.DecodedSound, err error) map[string]any {
	if err != nil {
		return map[string]any{
			"sampleRate":    0,
			"channels":      0,
			"bitsPerSample": 0,
			"pcm":           nil,
			"error":         err.Error(),
		}
	}
	return map[string]any{
		"sampleRate":    sound.SampleRate,
		"channels":      sound.Channels,
		"bitsPerSample": sound.BitsPerSample,
		"pcm":           pcmToJS(sound.PCM),
		"error":         nil,
	}
}

// pcmToJS copies a []int16 PCM buffer into a JS Int16Array — directly
// usable by a Web Audio API consumer without a further conversion step.
func pcmToJS(pcm []int16) js.Value {
	buf := make([]byte, len(pcm)*2)
	for i, s := range pcm {
		buf[i*2] = byte(uint16(s))
		buf[i*2+1] = byte(uint16(s) >> 8)
	}
	arrayBuffer := js.Global().Get("Uint8Array").New(len(buf))
	js.CopyBytesToJS(arrayBuffer, buf)
	return js.Global().Get("Int16Array").New(arrayBuffer.Get("buffer"))
}
