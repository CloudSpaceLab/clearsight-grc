import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtemp, mkdir, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { verifyReference } from "./v29-source-integrity.mjs";

const digest = (value) => createHash("sha256").update(value).digest("hex");

test("the complete 2.15 reference matches the pinned release", async () => {
  const root = fileURLToPath(new URL("../../clearsight-v2.15/", import.meta.url));
  assert.equal(await verifyReference(root, "4c77f905df2e667da76092fd462c181f5913d93f475f06b8411480377a46440b", "2.15"), 398);
});

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), "clearsight-v29-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  await mkdir(path.join(root, "assets"));
  const source = Buffer.from("reference bytes\n");
  await writeFile(path.join(root, "assets", "app.js"), source);
  const manifest = JSON.stringify({ release: "2.9", files: [{ path: "assets/app.js", bytes: source.length, sha256: digest(source) }] });
  await writeFile(path.join(root, "MANIFEST.json"), manifest);
  return { root, manifestDigest: digest(manifest) };
}

test("accepts the pinned manifest and the exact reference file set", async (t) => {
  const { root, manifestDigest } = await fixture(t);
  assert.equal(await verifyReference(root, manifestDigest), 1);
});

test("rejects a pinned manifest from a different release", async (t) => {
  const { root, manifestDigest } = await fixture(t);
  await assert.rejects(verifyReference(root, manifestDigest, "2.15"), /Invalid 2\.15 reference manifest/);
});

test("rejects changed reference bytes, even when their length is unchanged", async (t) => {
  const { root, manifestDigest } = await fixture(t);
  await writeFile(path.join(root, "assets", "app.js"), "reference byteS\n");
  await assert.rejects(verifyReference(root, manifestDigest), /assets\/app\.js/);
});

test("rejects extra files and unsafe manifest paths", async (t) => {
  const { root, manifestDigest } = await fixture(t);
  await writeFile(path.join(root, "extra.txt"), "extra");
  await assert.rejects(verifyReference(root, manifestDigest), /extra\.txt/);
  const unsafe = JSON.stringify({ release: "2.9", files: [{ path: "../outside", bytes: 0, sha256: digest("") }] });
  await writeFile(path.join(root, "MANIFEST.json"), unsafe);
  await assert.rejects(verifyReference(root, digest(unsafe)), /unsafe/i);
});

test("rejects a manifest changed without updating the pinned digest", async (t) => {
  const { root, manifestDigest } = await fixture(t);
  await writeFile(path.join(root, "MANIFEST.json"), "{}");
  await assert.rejects(verifyReference(root, manifestDigest), /manifest/i);
});
