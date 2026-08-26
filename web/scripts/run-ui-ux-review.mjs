import { mkdir, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { reviewRunnerEnvironment, startManagedPreview } from "./ui-review-runtime.mjs";

const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const logDir = path.join(outputDir, "flow-logs");
const scripts = [
  "scripts/capture-ui-evidence.mjs",
  "scripts/capture-lifecycle-ui-evidence.mjs",
  "scripts/capture-operating-mutations-evidence.mjs",
  "scripts/capture-program-review-evidence.mjs",
  "scripts/review-ui-defects.mjs",
  "scripts/review-ui-accessibility.mjs",
];

await rm(outputDir, { recursive: true, force: true });
await mkdir(logDir, { recursive: true });

const buildEnvironment = { ...process.env, VITE_STATIC_DEMO: "true" };
const build = spawnSync(process.platform === "win32" ? "npm.cmd" : "npm", ["run", "build", "--", "--base=/"], {
  cwd: process.cwd(),
  env: buildEnvironment,
  encoding: "utf8",
  maxBuffer: 16 * 1024 * 1024,
  shell: process.platform === "win32",
});
await writeFile(path.join(logDir, "managed-build.log"), `${build.stdout ?? ""}${build.stderr ? `\n${build.stderr}` : ""}`);
if (build.status !== 0) {
  process.stderr.write(build.stderr || build.stdout || "The managed UI review build failed.\n");
  process.exit(1);
}

const preview = await startManagedPreview({ cwd: process.cwd(), environment: buildEnvironment });
const runnerEnvironment = reviewRunnerEnvironment(process.env, preview.url);

const runs = [];
let review;
try {
  for (const script of scripts) {
    const result = spawnSync(process.execPath, [script], {
      cwd: process.cwd(),
      env: runnerEnvironment,
      encoding: "utf8",
      maxBuffer: 16 * 1024 * 1024,
    });
    const run = {
      script,
      status: result.status ?? 1,
      signal: result.signal ?? null,
      stdout: result.stdout ?? "",
      stderr: result.stderr ?? "",
    };
    runs.push(run);
    await writeFile(
      path.join(logDir, `${path.basename(script, ".mjs")}.log`),
      `${run.stdout}${run.stderr ? `\n${run.stderr}` : ""}`,
    );
  }

  await writeFile(path.join(outputDir, "runner.json"), JSON.stringify({ previewURL: preview.url, runs }, null, 2));

  review = spawnSync(process.execPath, ["scripts/review-ui-flow-manifest.mjs"], {
    cwd: process.cwd(),
    env: runnerEnvironment,
    encoding: "utf8",
    maxBuffer: 16 * 1024 * 1024,
  });
} finally {
  await writeFile(path.join(logDir, "managed-preview.log"), preview.output());
  await preview.stop();
}

if (review?.stdout) process.stdout.write(review.stdout);
if (review?.stderr) process.stderr.write(review.stderr);

const failedRuns = runs.filter((run) => run.status !== 0);
if (failedRuns.length) process.stderr.write(`UI/UX review runners failed: ${failedRuns.map((run) => run.script).join(", ")}.\n`);
if (failedRuns.length || review?.status !== 0) process.exit(1);
