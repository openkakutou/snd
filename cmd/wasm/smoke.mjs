#!/usr/bin/env node
// smoke.mjs is a Node.js verification harness for the WASM entrypoint built
// from this directory (see main.go) — it exercises the module the same way
// a browser consumer would (fetch/instantiate the .wasm, call the exposed
// global function, read back the result), without requiring an actual
// browser. It is not part of `go test` — syscall/js glue cannot run under
// the plain Go toolchain — and doubles as a minimal usage example for a JS
// consumer. Mirrors sibling repo sff's own cmd/wasm/smoke.mjs.
//
// Usage: node cmd/wasm/smoke.mjs [path/to/snd.wasm]
// (defaults to ./snd.wasm, relative to the repo root)

import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");
const wasmPath = path.resolve(process.argv[2] || path.join(repoRoot, "snd.wasm"));

const goroot = execSync("go env GOROOT").toString().trim();
const wasmExecPath = path.join(goroot, "lib", "wasm", "wasm_exec.js");

// wasm_exec.js defines a global `Go` constructor; importing it for its
// side effect is the same pattern used to load it in a browser <script> tag.
await import(`file://${wasmExecPath}`);

const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
go.run(instance); // does not return: keeps the Go runtime (and its registered function) alive

function toUint8Array(relativePath) {
	return new Uint8Array(readFileSync(path.join(repoRoot, relativePath)));
}

function assert(condition, message) {
	if (!condition) {
		console.error(`FAIL: ${message}`);
		process.exitCode = 1;
	} else {
		console.log(`ok - ${message}`);
	}
}

// --- decode: nominal path, a real v1 fixture, version not specified by the caller ---
const v1Bytes = toUint8Array("testdata/files/v1-basic.snd");
const v1Result = globalThis.OpenKakutouSnd.decode(v1Bytes, 1, 0);
assert(v1Result.error === null, `decode: v1 nominal reports no error (got: ${v1Result.error})`);
assert(v1Result.sampleRate === 11025, `decode: v1 nominal reports sampleRate 11025 (got: ${v1Result.sampleRate})`);
assert(v1Result.channels === 1, `decode: v1 nominal reports channels 1 (got: ${v1Result.channels})`);
assert(v1Result.bitsPerSample === 16, `decode: v1 nominal reports bitsPerSample 16 (got: ${v1Result.bitsPerSample})`);
assert(v1Result.pcm instanceof Int16Array, "decode: v1 nominal returns an Int16Array pcm buffer");
assert(v1Result.pcm.length === 2772, `decode: v1 nominal pcm has 2772 samples (got: ${v1Result.pcm.length})`);

// --- decode: nominal path, a real v2 fixture, auto-detected without the caller saying "v2" ---
// Sample 143 does not fit in v1's 2-byte Sample field (decision 002) — if
// the module mistakenly ran this through the v1 code path, this lookup
// would fail to find the entry at all.
const v2Bytes = toUint8Array("testdata/files/v2-basic.snd");
const v2Result = globalThis.OpenKakutouSnd.decode(v2Bytes, 1, 143);
assert(v2Result.error === null, `decode: v2 nominal reports no error (got: ${v2Result.error})`);
assert(v2Result.sampleRate === 8000, `decode: v2 nominal reports sampleRate 8000 (got: ${v2Result.sampleRate})`);
assert(v2Result.pcm instanceof Int16Array && v2Result.pcm.length > 0, "decode: v2 nominal returns a non-empty pcm buffer");

// --- decode: v2 external-file-reference, resolved via a synchronous JS callback ---
const externalRefBytes = toUint8Array("testdata/files/v2-external-ref.snd");
const resolveExternal = (requestedPath) => toUint8Array(path.join("testdata", "files", requestedPath));
const externalResult = globalThis.OpenKakutouSnd.decode(externalRefBytes, 0, 7, resolveExternal);
assert(externalResult.error === null, `decode: external reference reports no error (got: ${externalResult.error})`);
assert(externalResult.pcm instanceof Int16Array && externalResult.pcm.length > 0, "decode: external reference returns a non-empty pcm buffer resolved from the real external file");

// --- decode: v2 external-file-reference with no resolver supplied: a clean error, not a throw ---
const noResolverResult = globalThis.OpenKakutouSnd.decode(externalRefBytes, 0, 7);
assert(noResolverResult.pcm === null, "decode: external reference without a resolver returns null pcm");
assert(typeof noResolverResult.error === "string" && noResolverResult.error.length > 0, "decode: external reference without a resolver reports an error");

// --- decode: unknown (group, sample) ---
const unknownResult = globalThis.OpenKakutouSnd.decode(v1Bytes, 999, 999);
assert(unknownResult.pcm === null, "decode: unknown (group, sample) returns null pcm");
assert(typeof unknownResult.error === "string" && unknownResult.error.length > 0, "decode: unknown (group, sample) reports an error");

// --- decode: malformed bytes, must not crash the module ---
const malformedResult = globalThis.OpenKakutouSnd.decode(new TextEncoder().encode("garbage, not a real snd file, but long enough"), 0, 0);
assert(malformedResult.pcm === null, "decode: malformed bytes returns null pcm");
assert(typeof malformedResult.error === "string" && malformedResult.error.includes("not a .snd file"), `decode: malformed bytes error identifies an unrecognized file (got: ${malformedResult.error})`);

// --- decode: wrong argument count, must not crash the module ---
const argCountResult = globalThis.OpenKakutouSnd.decode();
assert(argCountResult.pcm === null, "decode: missing arguments returns null pcm");
assert(typeof argCountResult.error === "string" && argCountResult.error.length > 0, "decode: missing arguments reports an error");

// --- editing: add, replace and remove, each result decoded again by the same module ---
function wav16(rate, samples) {
	const data = new Uint8Array(samples.length * 2);
	samples.forEach((v, i) => new DataView(data.buffer).setInt16(i * 2, v, true));
	const out = new Uint8Array(44 + data.length);
	const dv = new DataView(out.buffer);
	out.set(new TextEncoder().encode("RIFF"), 0);
	dv.setUint32(4, 36 + data.length, true);
	out.set(new TextEncoder().encode("WAVEfmt "), 8);
	dv.setUint32(16, 16, true);
	dv.setUint16(20, 1, true); // PCM
	dv.setUint16(22, 1, true); // mono
	dv.setUint32(24, rate, true);
	dv.setUint32(28, rate * 2, true);
	dv.setUint16(32, 2, true);
	dv.setUint16(34, 16, true);
	out.set(new TextEncoder().encode("data"), 36);
	dv.setUint32(40, data.length, true);
	out.set(data, 44);
	return out;
}

const added = globalThis.OpenKakutouSnd.addSound(v1Bytes, 7, 3, wav16(16000, [1, -2, 3]));
assert(added.error === null && added.bytes instanceof Uint8Array, `addSound: nominal returns bytes (got: ${added.error})`);
const addedDecoded = globalThis.OpenKakutouSnd.decode(added.bytes, 7, 3);
assert(addedDecoded.error === null && addedDecoded.sampleRate === 16000 && addedDecoded.pcm.length === 3, "addSound: the new sound decodes with the rate and samples given");
assert(globalThis.OpenKakutouSnd.decode(added.bytes, 1, 0).pcm.length === 2772, "addSound: the existing sound is untouched");

const replaced = globalThis.OpenKakutouSnd.replaceSound(added.bytes, 7, 3, wav16(8000, [5, 6]));
assert(replaced.error === null, `replaceSound: nominal reports no error (got: ${replaced.error})`);
const replacedDecoded = globalThis.OpenKakutouSnd.decode(replaced.bytes, 7, 3);
assert(replacedDecoded.sampleRate === 8000 && replacedDecoded.pcm.length === 2, "replaceSound: the sound now decodes as the new audio");

const removed = globalThis.OpenKakutouSnd.removeSound(replaced.bytes, 7, 3);
assert(removed.error === null, `removeSound: nominal reports no error (got: ${removed.error})`);
assert(globalThis.OpenKakutouSnd.decode(removed.bytes, 7, 3).pcm === null, "removeSound: the sound is gone");
assert(globalThis.OpenKakutouSnd.decode(removed.bytes, 1, 0).error === null, "removeSound: the other sound still decodes");

const duplicate = globalThis.OpenKakutouSnd.addSound(v1Bytes, 1, 0, wav16(8000, [1]));
assert(duplicate.bytes === null && /already exists/.test(duplicate.error), `addSound: a duplicate key is refused (got: ${duplicate.error})`);
const notWav = globalThis.OpenKakutouSnd.addSound(v1Bytes, 9, 9, new TextEncoder().encode("not audio at all"));
assert(notWav.bytes === null && typeof notWav.error === "string", "addSound: non-WAV audio is refused");
const missing = globalThis.OpenKakutouSnd.removeSound(v1Bytes, 999, 999);
assert(missing.bytes === null && /no sound/.test(missing.error), `removeSound: a missing sound is refused (got: ${missing.error})`);
const editArgs = globalThis.OpenKakutouSnd.replaceSound();
assert(editArgs.bytes === null && typeof editArgs.error === "string", "replaceSound: missing arguments report an error");

// The module must still respond correctly after every error above — proves
// none of them left the Go runtime in a broken state.
const afterErrorsResult = globalThis.OpenKakutouSnd.decode(v1Bytes, 1, 0);
assert(afterErrorsResult.error === null, "decode: module still works after prior errors");

if (process.exitCode) {
	console.error("\nsmoke test FAILED");
} else {
	console.log("\nsmoke test passed");
}
process.exit(process.exitCode ?? 0);
