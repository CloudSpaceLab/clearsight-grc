import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const sourceRoot = fileURLToPath(new URL("../../docs/design/references/clearsight-2026-10/source/", import.meta.url));

test("approved design source files retain the exact archived bytes", async () => {
  const manifest = JSON.parse(await readFile(path.join(sourceRoot, "MANIFEST.json"), "utf8"));
  assert.ok(manifest.files.length > 0, "approved source manifest must not be empty");
  const mismatches = [];
  for (const entry of manifest.files) {
    const bytes = await readFile(path.join(sourceRoot, entry.path));
    const digest = createHash("sha256").update(bytes).digest("hex");
    if (bytes.length !== entry.bytes || digest !== entry.sha256) mismatches.push(entry.path);
  }
  assert.deepEqual(mismatches, [], "approved design source bytes do not match the manifest");
});
