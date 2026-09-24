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
