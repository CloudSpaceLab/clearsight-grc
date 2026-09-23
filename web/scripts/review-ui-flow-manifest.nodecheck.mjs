import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const webRoot = fileURLToPath(new URL("..", import.meta.url));

// These deliberately incomplete manifests test reviewer diagnostics, not rendered proof.
async function review(captures) {
  const evidenceDir = await mkdtemp(path.join(tmpdir(), "clearsight-manifest-nodecheck-"));
  try {
    await writeFile(path.join(evidenceDir, "manifest.json"), JSON.stringify({ captures }));
    assert.throws(() => execFileSync(process.execPath, ["scripts/review-ui-flow-manifest.mjs"], {
      cwd: webRoot,
      env: { ...process.env, UI_EVIDENCE_DIR: evidenceDir },
      stdio: "pipe",
    }), (error) => error.status === 1);
    return JSON.parse(await readFile(path.join(evidenceDir, "review.json"), "utf8"));
  } finally {
    await rm(evidenceDir, { recursive: true, force: true });
  }
}

test("the manifest reviewer rejects duplicate capture names", async () => {
  const capture = { name: "01-today-dark-comfortable-1440x900", state: "baseline", theme: "dark", viewport: { width: 1440, height: 900 } };
  const { failures } = await review([capture, capture]);
  assert.ok(failures.includes("flow manifest contains duplicate capture names"));
});

test("the manifest reviewer rejects unknown capture names", async () => {
  const capture = { name: "01-today-dark-comfortable-1440x900", state: "baseline", theme: "dark", viewport: { width: 1440, height: 900 } };
  const unknown = { ...capture, name: "unknown-capture-light-1440x900" };
  const { failures } = await review([capture, unknown]);
  assert.ok(failures.some((failure) => failure.startsWith("flow manifest contains unexpected captures:") && failure.includes(unknown.name)));
});
