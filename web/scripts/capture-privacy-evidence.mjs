import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { privacyEvidenceScenarios } from "./privacy-evidence-scenarios.mjs";
import { measurePrivacyLayout, openPrivacyActivity, verifyPrivacyFilters, verifyPrivacyHistory, waitForPrivacy } from "./privacy-evidence.mjs";
import { privacyCaptureFailures } from "./privacy-evidence-contract.mjs";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173", out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const requested = process.env.UI_EVIDENCE_CASE;
const cases = privacyEvidenceScenarios.filter((item) => !requested || requested.split(",").some((pattern) => item.name.includes(pattern)));
const manifestPath = path.join(out, "manifest.json");
await mkdir(out, { recursive: true });
const manifest = await readFile(manifestPath, "utf8").then(JSON.parse).catch((error) => { if (error.code !== "ENOENT") throw error; return { generatedAt: new Date().toISOString(), baseURL, captures: [] }; });
manifest.failure = null;
const digest = (bytes) => createHash("sha256").update(bytes).digest("hex");
const buildDir = path.resolve(process.env.UI_EVIDENCE_BUILD_DIR ?? "dist-evidence");
async function buildFiles(directory = buildDir, prefix = "") {
  const result = {};
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const name = `${prefix}${entry.name}`, filename = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(result, await buildFiles(filename, `${name}/`));
    else if (entry.isFile()) { const bytes = await readFile(filename); result[name] = { bytes: bytes.length, sha256: digest(bytes) }; }
  }
  return result;
}
const assets = await buildFiles(), buildDigest = digest(JSON.stringify(assets));
const revision = process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA ?? null;
if (manifest.runtimeBuild && (manifest.runtimeBuild.digest !== buildDigest || manifest.runtimeBuild.revision !== revision)) throw new Error("Privacy captures must use one unchanged revision/build per output directory");
manifest.runtimeBuild = { revision, digest: buildDigest, assets };
async function evidenceInputs() {
  const files = {};
  for (const name of ["capture-privacy-evidence.mjs", "privacy-evidence.mjs", "privacy-evidence-scenarios.mjs", "privacy-evidence-contract.mjs"]) {
    const bytes = await readFile(new URL(name, import.meta.url)); files[`scripts/${name}`] = { bytes: bytes.length, sha256: digest(bytes) };
  }
  return { files, digest: digest(JSON.stringify(files)) };
}
const inputs = await evidenceInputs();
inputs.revision = process.env.UI_EVIDENCE_SCRIPTS_REVISION ?? process.env.GITHUB_SHA ?? null;
if (manifest.privacyEvidenceInputs && manifest.privacyEvidenceInputs.digest !== inputs.digest) throw new Error("Privacy capture instructions changed; use a new output directory");
manifest.privacyEvidenceInputs = inputs;
const browser = await chromium.launch({ headless: true });
manifest.privacyBrowser = { version: browser.version(), node: process.version, locale: "en-NG", timezone: "Africa/Lagos", reducedMotion: "reduce" };
try {
  for (const scenario of cases) {
    const context = await browser.newContext({ viewport: scenario.viewport, colorScheme: scenario.theme, hasTouch: scenario.touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
    await context.addInitScript((theme) => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, scenario.theme);
    const page = await context.newPage(), errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    try {
      await page.goto(`${baseURL}/?tour=off&fixture=${scenario.fixture}${scenario.route}`, { waitUntil: "networkidle" });
      await waitForPrivacy(page, scenario);
      let detail, history, filters, recovery, pageRecovery;
      if (["detail", "not-found"].includes(scenario.privacyCheck)) detail = await openPrivacyActivity(page, { notFound: scenario.privacyCheck === "not-found" });
      if (scenario.privacyCheck === "history") history = await verifyPrivacyHistory(page);
      if (scenario.privacyCheck === "filters") filters = await verifyPrivacyFilters(page);
      if (scenario.privacyCheck === "recovery") {
        await page.locator(".ropa-register-list").getByRole("button", { name: "Retry", exact: true }).click();
        await page.locator(".ropa-register-identity strong").first().waitFor();
        recovery = { retry: true };
      }
      if (["page-empty", "page-error"].includes(scenario.privacyCheck)) {
        await page.getByRole("button", { name: "Load next page", exact: true }).click();
        const register = page.locator(".ropa-register-list");
        if (scenario.privacyCheck === "page-empty") await register.getByRole("heading", { name: "No more activities", exact: true }).waitFor();
        else await register.getByText("Couldn’t load activities.", { exact: true }).waitFor();
        const previous = page.getByRole("button", { name: "Load previous page", exact: true });
        const previousBounds = await previous.evaluate((node) => { const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; });
        const image = `${scenario.name}-page.png`;
        await page.screenshot({ path: path.join(out, image), fullPage: false, animations: "disabled" });
        const bytes = await readFile(path.join(out, image));
        pageRecovery = { before: await measurePrivacyLayout(page, scenario), previousBounds, image, imageSHA256: digest(bytes), imageBytes: bytes.length,
          beforeRead: await page.evaluate(() => window.privacyEvidence.reads.filter((item) => item.path === "/api/v1/ropa/processing-activities").at(-1)) };
        await previous.click();
        await register.locator(".ropa-register-identity strong").first().waitFor();
      }
      if (!detail) { await page.evaluate(() => { window.scrollTo(0, 0); if (document.activeElement instanceof HTMLElement) document.activeElement.blur(); }); }
      const privacyLayout = await measurePrivacyLayout(page, scenario);
      const observation = await page.evaluate(() => window.privacyEvidence);
      const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
      const record = { ...scenario, buildDigest, evidenceInputDigest: inputs.digest, privacyLayout, detail, history, filters, recovery, pageRecovery, observation, metrics, errors };
      const failures = privacyCaptureFailures(record, scenario);
      if (failures.length) throw new Error(failures.join("\n"));
      const image = path.join(out, `${scenario.name}.png`);
      await page.screenshot({ path: image, fullPage: false, animations: "disabled", caret: "hide" });
      const bytes = await readFile(image);
      Object.assign(record, { imageBytes: bytes.length, imageSHA256: digest(bytes) });
      manifest.captures = manifest.captures.filter((item) => item.name !== record.name); manifest.captures.push(record);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      process.stdout.write(`PASS ${scenario.name}\n`);
    } catch (error) {
      manifest.failure = error instanceof Error ? error.message : String(error);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      await writeFile(path.join(out, `${scenario.name}-failure.json`), JSON.stringify({ scenario, layout: await measurePrivacyLayout(page, scenario).catch(() => null), observation: await page.evaluate(() => window.privacyEvidence).catch(() => null) }, null, 2));
      await page.screenshot({ path: path.join(out, `${scenario.name}-failure.png`), fullPage: true }).catch(() => {});
      throw error;
    } finally { await context.close(); }
  }
  if (digest(JSON.stringify(await buildFiles())) !== buildDigest || (await evidenceInputs()).digest !== inputs.digest) {
    manifest.failure = "Privacy evidence build or capture instructions changed during capture";
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
    throw new Error(manifest.failure);
  }
} finally { await browser.close(); }
