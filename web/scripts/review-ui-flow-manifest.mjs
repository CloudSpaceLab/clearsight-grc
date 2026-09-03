import { createHash } from "node:crypto";
import { readdir, readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import { gzipSync } from "node:zlib";

const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const expectedNames = [
  "01-today-dark-comfortable-1440x900",
  "02-today-light-comfortable-1440x900",
  "03-today-dark-compact-1440x900",
  "04-program-light-1440x900",
  "05-matter-dark-1440x900",
  "06-evidence-light-1440x900",
  "07-import-dark-1440x900",
  "08-configure-light-1440x900",
  "09-today-dark-tablet-1024x768",
  "10-today-light-mobile-390x844",
  "11-today-dark-reflow-320x800",
  "12-authority-dark-1440x900",
  "13-capture-entry-light-1440x900",
  "14-capture-review-light-1440x900",
  "15-capture-receipt-light-1440x900",
  "16-today-light-200pct-zoom-proxy",
  "17-today-empty-light-1440x900",
  "18-today-loading-dark-1440x900",
  "19-today-unavailable-light-1440x900",
  "20-evidence-partial-light-1440x900",
  "21-configure-partial-dark-1440x900",
  "22-no-config-access-light-1440x900",
  "23-authority-forbidden-light-1440x900",
  "24-capture-not-found-dark-1440x900",
  "25-capture-expired-light-1440x900",
  "26-capture-conflict-light-1440x900",
  "27-evidence-long-content-mobile-390x844",
  "28-capture-mobile-light-390x844",
  "29-field-visit-entry-light-390x844",
  "30-field-visit-review-light-390x844",
  "31-field-visit-receipt-light-390x844",
  "32-import-dropzone-selected-light-1440x900",
  "33-today-lifecycle-collapsed-light-1440x900",
  "34-today-lifecycle-expanded-light-1440x900",
  "35-today-lifecycle-collapsed-light-mobile-390x844",
  "36-today-lifecycle-expanded-light-mobile-390x844",
  "30-operating-actions-light-1440x900",
  "31-operating-actions-dark-mobile-390x844",
  "37-program-review-changed-light-1440x900",
  "38-program-review-acknowledged-light-1440x900",
  "39-program-review-changed-light-mobile-390x844",
  "40-program-overview-dark-1440x900",
  "41-program-evidence-results-light-1024x768",
  "42-program-monitoring-light-1440x900",
  "43-program-monitoring-blocked-dark-1440x900",
  "44-program-monitoring-light-mobile-390x844",
  "45-program-monitoring-long-dark-reflow-320x800",
  "46-program-monitoring-light-200pct-reflow-proxy",
  "47-program-sections-tab-focus-light-1440x900",
  "37-new-work-light-1440x900",
  "38-new-work-dark-mobile-390x844",
];
const requiredStates = [
  "baseline",
  "fixture:today-empty",
  "fixture:today-loading",
  "fixture:today-unavailable",
  "permission-denied",
  "response-entry",
  "response-review",
  "submission-receipt",
  "not-found",
  "terminal-expired",
  "optimistic-conflict",
  "mobile-focused-capture",
  "css-zoom-200pct-proxy",
  "external-field-visit-entry",
  "external-field-visit-review",
  "external-field-visit-receipt",
  "document-selected-before-import",
  "lifecycle-work-mobile-expanded",
  "operating-actions-desktop",
  "operating-actions-mobile",
  "program-review-changed",
  "program-review-acknowledged",
  "program-review-mobile-changed",
  "program-overview-dark",
  "program-evidence-results-tablet",
  "collection-current",
  "collection-delivery-blocked",
  "collection-mobile-selector",
  "collection-long-content",
  "collection-200pct-reflow-proxy",
  "program-tab-focus",
  "matter-create-open",
];

const failures = [];
const checks = [];
const safeReadJSON = async (name) => {
  try {
    return JSON.parse(await readFile(path.join(outputDir, name), "utf8"));
  } catch (error) {
    failures.push(`${name} is missing or invalid: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
};

const runner = await safeReadJSON("runner.json");
if (runner) {
  const failedRuns = runner.runs.filter((run) => run.status !== 0);
  checks.push({ name: "executable review runners", status: failedRuns.length ? "FAIL" : "PASS", detail: `${runner.runs.length} scripts executed` });
  for (const run of failedRuns) failures.push(`${run.script} exited with status ${run.status}`);
}

const defects = await safeReadJSON("defects.json");
if (defects) {
  if (defects.failure) failures.push(`defect review failed: ${defects.failure}`);
  const failed = defects.scenarios.filter((scenario) => scenario.status !== "PASS");
  for (const scenario of failed) failures.push(`${scenario.name}: ${scenario.detail ?? "failed"}`);
  checks.push({ name: "UI/UX and functional defect review", status: defects.failure || failed.length ? "FAIL" : "PASS", detail: `${defects.scenarios.length} behavioral scenarios` });
}

const manifest = await safeReadJSON("manifest.json");
if (manifest) {
  if (manifest.failure) failures.push(`flow manifest recorded failure: ${manifest.failure}`);
  const names = manifest.captures.map((capture) => capture.name);
  const uniqueNames = new Set(names);
  if (uniqueNames.size !== names.length) failures.push("flow manifest contains duplicate capture names");
  const missingRecords = expectedNames.filter((name) => !uniqueNames.has(name));
  const unexpectedRecords = names.filter((name) => !expectedNames.includes(name));
  if (missingRecords.length) failures.push(`flow manifest is missing: ${missingRecords.join(", ")}`);
  if (unexpectedRecords.length) failures.push(`flow manifest contains unexpected captures: ${unexpectedRecords.join(", ")}`);
  const states = new Set(manifest.captures.map((capture) => capture.state));
  const missingStates = requiredStates.filter((state) => !states.has(state));
  if (missingStates.length) failures.push(`flow state coverage is missing: ${missingStates.join(", ")}`);
  for (const capture of manifest.captures) {
    if (capture.metrics && capture.metrics.scrollWidth > capture.metrics.clientWidth + 1) failures.push(`${capture.name} recorded horizontal overflow`);
  }
  const widths = new Set(manifest.captures.map((capture) => capture.viewport?.width));
  const themes = new Set(manifest.captures.map((capture) => capture.theme));
  if (![...widths].some((width) => width >= 1280) || !widths.has(1024) || !widths.has(390) || !widths.has(320)) failures.push("responsive coverage must include desktop, tablet, mobile and 320px reflow");
  if (!themes.has("light") || !themes.has("dark")) failures.push("theme coverage must include light and dark modes");
  checks.push({ name: "rendered state coverage", status: missingRecords.length || unexpectedRecords.length || missingStates.length ? "FAIL" : "PASS", detail: `${uniqueNames.size}/${expectedNames.length} flow records` });
}

let evidence = [];
try {
  const files = (await readdir(outputDir)).filter((name) => name.endsWith(".png")).sort();
  const expectedFiles = expectedNames.map((name) => `${name}.png`).sort();
  const missingFiles = expectedFiles.filter((name) => !files.includes(name));
  const unexpectedFiles = files.filter((name) => !expectedFiles.includes(name));
  if (missingFiles.length) failures.push(`screenshots are missing: ${missingFiles.join(", ")}`);
  if (unexpectedFiles.length) failures.push(`unexpected screenshots exist: ${unexpectedFiles.join(", ")}`);
  for (const file of files) {
    const filePath = path.join(outputDir, file);
    const metadata = await stat(filePath);
    const bytes = await readFile(filePath);
    if (metadata.size < 1024) failures.push(`${file} is unexpectedly small (${metadata.size} bytes)`);
    evidence.push({ file, bytes: metadata.size, sha256: createHash("sha256").update(bytes).digest("hex") });
  }
  checks.push({ name: "review artifact completeness", status: missingFiles.length || unexpectedFiles.length ? "FAIL" : "PASS", detail: `${files.length}/${expectedFiles.length} screenshots retained` });
} catch (error) {
  failures.push(`screenshot artifacts could not be read: ${error instanceof Error ? error.message : String(error)}`);
}

const accessibility = await safeReadJSON("accessibility.json");
if (accessibility) {
  if (accessibility.failure) failures.push(`accessibility sweep failed: ${accessibility.failure}`);
  const failed = accessibility.scenarios.filter((scenario) => scenario.status !== "PASS");
  for (const scenario of failed) failures.push(`${scenario.name}: ${scenario.errors.join("; ")}`);
  checks.push({ name: "accessibility and touch", status: failed.length ? "FAIL" : "PASS", detail: `${accessibility.scenarios.length} rendered route states` });
}

const bundle = { javascript: { raw: 0, gzip: 0 }, css: { raw: 0, gzip: 0 } };
try {
  const assetDir = path.resolve("dist/assets");
  for (const name of await readdir(assetDir)) {
    const bytes = await readFile(path.join(assetDir, name));
    if (name.endsWith(".js")) {
      bundle.javascript.raw += bytes.length;
      bundle.javascript.gzip += gzipSync(bytes).length;
    } else if (name.endsWith(".css")) {
      bundle.css.raw += bytes.length;
      bundle.css.gzip += gzipSync(bytes).length;
    }
  }
  const bundleFailures = [];
  if (bundle.javascript.raw > 500 * 1024) bundleFailures.push(`JavaScript bundle exceeds 500 KiB raw (${bundle.javascript.raw} bytes)`);
  if (bundle.javascript.gzip > 160 * 1024) bundleFailures.push(`JavaScript bundle exceeds 160 KiB gzip (${bundle.javascript.gzip} bytes)`);
  if (bundle.css.gzip > 32 * 1024) bundleFailures.push(`CSS bundle exceeds 32 KiB gzip (${bundle.css.gzip} bytes)`);
  failures.push(...bundleFailures);
  checks.push({ name: "interaction bundle budget", status: bundleFailures.length ? "FAIL" : "PASS", detail: `${Math.round(bundle.javascript.gzip / 1024)} KiB JS gzip, ${Math.round(bundle.css.gzip / 1024)} KiB CSS gzip` });
} catch (error) {
  failures.push(`built assets could not be assessed: ${error instanceof Error ? error.message : String(error)}`);
}

const status = failures.length ? "FAIL" : "PASS";
const review = { generatedAt: new Date().toISOString(), status, checks, failures, bundle, defects, evidence };
await writeFile(path.join(outputDir, "review.json"), JSON.stringify(review, null, 2));
const markdown = [
  "# Automated UI/UX and functional defect review",
  "",
  `**Status:** ${status}`,
  "",
  ...checks.map((check) => `- ${check.status === "PASS" ? "✅" : "❌"} **${check.name}:** ${check.detail}`),
  "",
  failures.length ? "## Blocking findings" : "All executable usability, functional, accessibility, responsive-layout and bundle-budget gates passed.",
  ...(failures.length ? failures.map((failure) => `- ${failure}`) : []),
  "",
  `Artifacts: ${evidence.length} screenshots retained for diagnosis. Digests in \`review.json\` are metadata only and do not determine PASS.`,
].join("\n");
await writeFile(path.join(outputDir, "review.md"), markdown);
process.stdout.write(`${markdown}\n`);
if (failures.length) process.exit(1);
