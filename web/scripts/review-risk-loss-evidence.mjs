import { createHash } from "node:crypto";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { riskLossAccessibilityScenarios, riskLossEvidenceScenarios, riskLossViews, riskLossViewports } from "./risk-loss-evidence-scenarios.mjs";
import { reviewRiskLossCaptures } from "./risk-loss-evidence-contract.mjs";
import { riskLossSourcePath, riskLossSourceSHA256 } from "./risk-loss-evidence.mjs";

const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const digest = (bytes) => createHash("sha256").update(bytes).digest("hex");
const failures = [], files = [];
const load = async (name) => {
  try { const bytes = await readFile(path.join(outputDir, name)); files.push({ name, bytes: bytes.length, sha256: digest(bytes) }); return JSON.parse(bytes); }
  catch (error) { failures.push(`${name}: ${error.message}`); return null; }
};
const manifest = await load("manifest.json");
const captured = manifest?.captures ?? [];
const contract = reviewRiskLossCaptures(captured);
failures.push(...contract.failures);
if (manifest?.failure) failures.push(`capture failed: ${manifest.failure}`);
if (!/^[a-f0-9]{40}$/.test(manifest?.runtimeBuild?.revision ?? "") || !/^[a-f0-9]{64}$/.test(manifest?.runtimeBuild?.digest ?? "")
  || !Object.keys(manifest?.runtimeBuild?.assets ?? {}).length) failures.push("runtime captures require an exact revision and built-asset binding");
if (digest(JSON.stringify(manifest?.runtimeBuild?.assets ?? {})) !== manifest?.runtimeBuild?.digest) failures.push("runtime build digest does not match its retained asset list");
const expectedRevision = process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA;
if (expectedRevision && manifest?.runtimeBuild?.revision !== expectedRevision) failures.push("runtime captures belong to another revision");
const image = async (name, expectedSHA256, expectedBytes) => {
  try {
    const bytes = await readFile(path.join(outputDir, name));
    if (digest(bytes) !== expectedSHA256 || bytes.length !== expectedBytes) failures.push(`${name}: image binding differs`);
    files.push({ name, bytes: bytes.length, sha256: digest(bytes) });
  } catch (error) { failures.push(`${name}: ${error.message}`); }
};
for (const scenario of riskLossEvidenceScenarios) {
  const capture = captured.find((item) => item.name === scenario.name);
  if (capture && capture.buildDigest !== manifest?.runtimeBuild?.digest) failures.push(`${scenario.name}: runtime build binding differs`);
  if (capture) await image(`${scenario.name}.png`, capture.imageSHA256, capture.imageBytes);
}
const sources = await load("approved-reference/manifest.json");
const sourceCaptures = sources?.captures ?? [];
for (const view of riskLossViews) for (const viewport of riskLossViewports) {
  const source = sourceCaptures.find((item) => item.id === `risk-loss-${view.id}` && item.viewport.width === viewport.width && item.viewport.height === viewport.height);
  if (!source || source.source !== riskLossSourcePath || source.source_sha256 !== riskLossSourceSHA256
    || source.source_tab !== view.source || source.risk_loss_layout?.comparison !== view.sourceComparison
    || source.layout_comparison?.sourceComparison !== view.sourceComparison || source.layout_comparison?.sameScreen !== (view.sourceComparison === "SAME_SCREEN")
    || source.layout_comparison?.menuAdaptation !== "APPROVED_FOUR_ROUTE_NAVIGATION"
    || source.layout_comparison?.conclusion !== "STRUCTURE_CHECKED_VISUAL_REVIEW_REQUIRED") {
    failures.push(`source ${view.id}/${viewport.width}: immutable source identity, same-screen boundary or menu adaptation is missing`);
    continue;
  }
  await image(`approved-reference/${source.image}`, source.image_sha256, source.image_bytes);
  for (const theme of ["light", "dark"]) {
    const actual = captured.find((item) => item.riskLossMatrix && item.riskLossView === view.id && item.theme === theme && item.viewport.width === viewport.width);
    const pair = source.layout_comparison.runtime_pairs?.find((item) => item.runtime_theme === theme);
    if (!actual || !pair || pair.runtime_capture !== `${actual.name}.png` || pair.runtime_image_sha256 !== actual.imageSHA256 || pair.runtime_image_bytes !== actual.imageBytes
      || pair.runtimeFirstUsefulContentVisible !== true || pair.runtimeSelectedTabVisible !== true || pair.sourceComparison !== view.sourceComparison
      || pair.sameScreen !== (view.sourceComparison === "SAME_SCREEN")) failures.push(`source ${view.id}/${theme}/${viewport.width}: exact runtime screenshot pair is missing`);
  }
}
const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const sourceBytes = await readFile(path.join(repositoryRoot, "docs/design/references/clearsight-2026-10/source", riskLossSourcePath));
if (digest(sourceBytes) !== riskLossSourceSHA256) failures.push("approved operational-risk source changed");
const accessibility = await load("accessibility.json");
for (const expected of riskLossAccessibilityScenarios) {
  const actual = accessibility?.scenarios?.find((item) => item.name === expected.name);
  if (!actual || actual.status !== "PASS" || actual.errors?.length || actual.violations?.length || actual.touchTargetMinimum !== 44
    || actual.scopeSelector !== expected.scopeSelector || actual.theme !== expected.theme || actual.viewport?.width !== expected.viewport.width || actual.viewport?.height !== expected.viewport.height) {
    failures.push(`${expected.name}: scoped accessibility, contrast or 44px touch evidence failed or is absent`);
  }
}
const review = { generatedAt: new Date().toISOString(), status: failures.length ? "FAIL" : "PASS", check: contract.check,
  visualParity: "NOT_ASSESSED", liveAuthority: "NOT_ASSESSED", evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION",
  sourceSHA256: riskLossSourceSHA256, captures: riskLossEvidenceScenarios.length, sourceFrames: riskLossViews.length * riskLossViewports.length,
  runtimeRevision: manifest?.runtimeBuild?.revision ?? null, runtimeBuildDigest: manifest?.runtimeBuild?.digest ?? null,
  sourceRuntimePairs: riskLossViews.length * riskLossViewports.length * 2, accessibilityScenarios: riskLossAccessibilityScenarios.length, files, failures };
await writeFile(path.join(outputDir, "risk-loss-review.json"), JSON.stringify(review, null, 2));
process.stdout.write(JSON.stringify({ ...review, files: `${files.length} artifacts hash-bound` }, null, 2) + "\n");
if (failures.length) process.exitCode = 1;
