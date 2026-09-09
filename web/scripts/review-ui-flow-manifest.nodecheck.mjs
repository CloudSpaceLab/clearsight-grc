import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const webRoot = fileURLToPath(new URL("..", import.meta.url));
const vendorStates = ["conflict", "reloaded", "late-result-isolated"];
const vendorCaptures = ["light", "dark"].flatMap((theme) => [1440, 390, 320].flatMap((width) => vendorStates.map((state) => ({
  name: `vendor-activation-${state}-${theme}-${width}`,
  state: `vendor-activation-${state}`,
  theme,
  viewport: { width, height: width === 1440 ? 900 : 844 },
}))));

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

test("the manifest reviewer accepts the supported vendor activation matrix", async () => {
  const { failures } = await review(vendorCaptures);
  assert.deepEqual(failures.filter((failure) => failure.startsWith("flow manifest contains unexpected captures:")), []);
  for (const capture of vendorCaptures) {
    assert.equal(failures.some((failure) => failure.startsWith("flow manifest is missing:") && failure.includes(capture.name)), false, capture.name);
    assert.ok(failures.some((failure) => failure.startsWith("screenshots are missing:") && failure.includes(`${capture.name}.png`)), `${capture.name} screenshot is required`);
  }
});

test("the manifest reviewer requires every vendor activation theme and viewport capture", async (t) => {
  for (const capture of vendorCaptures) {
    await t.test(capture.name, async () => {
      const { failures } = await review(vendorCaptures.filter((candidate) => candidate !== capture));
      assert.ok(failures.some((failure) => failure.startsWith("flow manifest is missing:") && failure.includes(capture.name)), capture.name);
    });
  }
});

test("the manifest reviewer requires vendor activation state metadata", async (t) => {
  for (const state of vendorStates) {
    await t.test(state, async () => {
      const requiredState = `vendor-activation-${state}`;
      const { failures } = await review(vendorCaptures.map((capture) => capture.state === requiredState ? { ...capture, state: "baseline" } : capture));
      assert.ok(failures.some((failure) => failure.startsWith("flow state coverage is missing:") && failure.includes(requiredState)), requiredState);
    });
  }
});

test("the manifest reviewer still rejects duplicate and unknown vendor captures", async () => {
  const unknown = { ...vendorCaptures[0], name: "vendor-activation-unknown-light-1440" };
  const { failures } = await review([...vendorCaptures, vendorCaptures[0], unknown]);
  assert.ok(failures.includes("flow manifest contains duplicate capture names"));
  assert.ok(failures.some((failure) => failure.startsWith("flow manifest contains unexpected captures:") && failure.includes(unknown.name)));
});
