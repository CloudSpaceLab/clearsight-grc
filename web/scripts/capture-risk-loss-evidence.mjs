import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { riskLossEvidenceScenarios } from "./risk-loss-evidence-scenarios.mjs";
import { prepareRiskLoss, measureRiskLossLayout, assertRiskLossLayout, assertRiskLossDetail, openRiskLossDetail, verifyRiskLossHistory, verifyRiskLossFilters, verifyRiskLossMenu } from "./risk-loss-evidence.mjs";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const baseline = process.env.UI_EVIDENCE_BASELINE === "1";
const wanted = process.env.UI_EVIDENCE_CASE;
const scenarios = riskLossEvidenceScenarios.filter((item) => (!baseline || item.riskLossMatrix) && (!wanted || wanted.split(",").some((pattern) => item.name.includes(pattern))));
const manifestPath = path.join(outputDir, "manifest.json");
await mkdir(outputDir, { recursive: true });
let manifest = await readFile(manifestPath, "utf8").then(JSON.parse).catch((error) => { if (error.code !== "ENOENT") throw error; return { generatedAt: new Date().toISOString(), baseURL, failure: null, captures: [] }; });
manifest.failure = null;
manifest.evidenceKind = "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION";
if (baseline) manifest.baselineRevision = process.env.UI_EVIDENCE_REVISION ?? manifest.baselineRevision ?? null;
const revision = process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA;
const buildDir = path.resolve(process.env.UI_EVIDENCE_BUILD_DIR ?? "dist-evidence");
async function measureBuild(directory = buildDir, prefix = "") {
  const assets = {};
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((left, right) => left.name.localeCompare(right.name))) {
    const name = `${prefix}${entry.name}`, absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(assets, await measureBuild(absolute, `${name}/`));
    else if (entry.isFile()) { const bytes = await readFile(absolute); assets[name] = { bytes: bytes.length, sha256: createHash("sha256").update(bytes).digest("hex") }; }
  }
  return assets;
}
const buildAssets = !baseline && revision ? await measureBuild() : null;
const buildDigest = buildAssets ? createHash("sha256").update(JSON.stringify(buildAssets)).digest("hex") : null;
if (buildDigest) {
  if (manifest.runtimeBuild && (manifest.runtimeBuild.revision !== revision || manifest.runtimeBuild.digest !== buildDigest)) throw new Error("Existing evidence belongs to another revision/build; use a separate output directory.");
  manifest.runtimeBuild = { revision, digest: buildDigest, assets: buildAssets };
}
const browser = await chromium.launch({ headless: true });
try {
  for (const capture of scenarios) {
    if (baseline && process.env.UI_EVIDENCE_RESUME === "1" && manifest.captures.some((item) => item.name === capture.name)) continue;
    const context = await browser.newContext({ viewport: capture.viewport, colorScheme: capture.theme, hasTouch: capture.touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
    await context.addInitScript((theme) => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, capture.theme);
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    try {
      await page.goto(`${baseURL}/?tour=off&fixture=${capture.fixture}#oversight`, { waitUntil: "networkidle" });
      await page.getByRole("heading", { name: "Oversight", exact: true }).first().waitFor();
      await prepareRiskLoss(page, capture, baseline);
      let detail, history, filters, navigation;
      if (!baseline && capture.verifyRiskLossNavigation) navigation = await verifyRiskLossMenu(page, capture);
      if (!baseline && capture.riskLossCheck === "detail") detail = await openRiskLossDetail(page, capture);
      if (!baseline && capture.riskLossCheck === "history") history = await verifyRiskLossHistory(page, capture);
      if (!baseline && capture.riskLossCheck === "filters") filters = await verifyRiskLossFilters(page, capture);
      if (!detail) await page.evaluate(() => window.scrollTo(0, 0));
      if (!detail) await page.evaluate(() => { if (document.activeElement instanceof HTMLElement) document.activeElement.blur(); });
      const riskLossLayout = await measureRiskLossLayout(page, capture);
      if (!baseline && !detail) assertRiskLossLayout(riskLossLayout, capture);
      if (detail) assertRiskLossDetail(detail, capture);
      const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth, clientHeight: document.documentElement.clientHeight, scrollHeight: document.documentElement.scrollHeight }));
      if (!baseline && metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${capture.name}: horizontal overflow`);
      if (errors.length) throw new Error(`${capture.name}: browser errors: ${errors.join("; ")}`);
      const imagePath = path.join(outputDir, `${capture.name}.png`);
      await page.screenshot({ path: imagePath, fullPage: false, animations: "disabled", caret: "hide" });
      const bytes = await readFile(imagePath);
      const observation = await page.evaluate(() => window.riskLossEvidence);
      const record = { ...capture, baseline, buildDigest, riskLossLayout, detail, history, filters, navigation, observation, metrics, imageBytes: bytes.length, imageSHA256: createHash("sha256").update(bytes).digest("hex") };
      manifest.captures = manifest.captures.filter((item) => item.name !== record.name);
      manifest.captures.push(record);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      process.stdout.write(`PASS ${capture.name}\n`);
    } catch (error) {
      manifest.failure = error instanceof Error ? error.message : String(error);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      await page.screenshot({ path: path.join(outputDir, `${capture.name}-failure.png`), fullPage: true }).catch(() => {});
      throw error;
    } finally { await context.close(); }
  }
  if (buildDigest && createHash("sha256").update(JSON.stringify(await measureBuild())).digest("hex") !== buildDigest) {
    manifest.failure = "Evidence build changed during capture";
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
    throw new Error(manifest.failure);
  }
} finally { await browser.close(); }
