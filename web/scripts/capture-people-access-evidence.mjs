import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { peopleAccessEvidenceScenarios } from "./people-access-evidence-scenarios.mjs";
import { measurePeopleAccessLayout, openPeopleAccessContext, showPeopleAccessContinuation, verifyPeopleAccessComposer, verifyPeopleAccessDefinition, verifyPeopleAccessFilters, verifyPeopleAccessHistory, verifyPeopleAccessStickyContext, waitForPeopleAccess } from "./people-access-evidence.mjs";
import { peopleAccessCaptureFailures } from "./people-access-evidence-contract.mjs";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173", out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const requested = process.env.UI_EVIDENCE_CASE;
const cases = peopleAccessEvidenceScenarios.filter((item) => !requested || requested.split(",").some((pattern) => item.name.includes(pattern)));
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
if (manifest.runtimeBuild && (manifest.runtimeBuild.digest !== buildDigest || manifest.runtimeBuild.revision !== revision)) throw new Error("PeopleAccess captures must use one unchanged revision/build per output directory");
manifest.runtimeBuild = { revision, digest: buildDigest, assets };
async function evidenceInputs() {
  const files = {};
  for (const name of ["capture-people-access-evidence.mjs", "people-access-evidence.mjs", "people-access-evidence-scenarios.mjs", "people-access-evidence-contract.mjs"]) {
    const bytes = await readFile(new URL(name, import.meta.url)); files[`scripts/${name}`] = { bytes: bytes.length, sha256: digest(bytes) };
  }
  return { files, digest: digest(JSON.stringify(files)) };
}
const inputs = await evidenceInputs();
inputs.revision = process.env.UI_EVIDENCE_SCRIPTS_REVISION ?? process.env.GITHUB_SHA ?? null;
if (manifest.peopleAccessEvidenceInputs && manifest.peopleAccessEvidenceInputs.digest !== inputs.digest) throw new Error("PeopleAccess capture instructions changed; use a new output directory");
manifest.peopleAccessEvidenceInputs = inputs;
const browser = await chromium.launch({ headless: true });
manifest.peopleAccessBrowser = { version: browser.version(), node: process.version, locale: "en-NG", timezone: "Africa/Lagos", reducedMotion: "reduce" };
try {
  for (const scenario of cases) {
    const context = await browser.newContext({ viewport: scenario.viewport, colorScheme: scenario.theme, hasTouch: scenario.touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
    await context.addInitScript((theme) => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, scenario.theme);
    const page = await context.newPage(), errors = [];
    let diagnostic;
    page.on("pageerror", (error) => errors.push(error.message));
    try {
      await page.goto(`${baseURL}/?tour=off&fixture=${scenario.fixture}${scenario.route}`, { waitUntil: "networkidle" });
      await waitForPeopleAccess(page, scenario);
      let contextReceipt, history, filters, recovery, missing, readonly, composer, sticky, definition;
      if (scenario.peopleAccessCheck === "context") contextReceipt = await openPeopleAccessContext(page, scenario);
      if (scenario.peopleAccessCheck === "history") history = await verifyPeopleAccessHistory(page);
      if (scenario.peopleAccessCheck === "filters") filters = await verifyPeopleAccessFilters(page);
      if (scenario.peopleAccessCheck === "composer") composer = await verifyPeopleAccessComposer(page, scenario);
      if (scenario.peopleAccessCheck === "definition") definition = await verifyPeopleAccessDefinition(page);
      if (scenario.peopleAccessCheck === "layout" && scenario.peopleAccessView === "people" && scenario.viewport.width === 1440 && scenario.theme === "light") sticky = await verifyPeopleAccessStickyContext(page);
      if (scenario.peopleAccessCheck === "recovery") {
        await page.locator(".people-access-workspace").getByRole("button", { name: "Retry", exact: true }).click();
        await page.locator(".people-directory__records .cs-selectable-record").first().waitFor(); recovery = { retry: true };
      }
      if (scenario.peopleAccessCheck === "missing") {
        const key = ({ people: "person", roles: "role", routes: "policy", delegations: "delegation" })[scenario.peopleAccessView];
        const noun = ({ person: "Person", role: "Role", policy: "Policy", delegation: "Delegation" })[key];
        const returnLabel = ({ person: "Return to listed people", role: "Return to listed roles", policy: "Return to listed routing policies", delegation: "Return to listed delegations" })[key];
        await page.evaluate(({ route, key }) => { location.hash = `${route}?${key}=sample-record-not-returned`; }, { route: scenario.route, key });
        const heading = page.getByRole("heading", { name: `${noun} not in this list`, exact: true });
        await heading.waitFor();
        const unavailableHeading = await heading.textContent();
        await page.getByRole("button", { name: returnLabel, exact: true }).click();
        await page.locator(".people-access-detail h2").first().waitFor();
        missing = { recovered: true, unavailableHeading, returnRoute: await page.evaluate(() => location.hash) };
        await page.evaluate(({ route, key }) => { location.hash = `${route}?${key}=sample-record-not-returned`; }, { route: scenario.route, key });
        await heading.waitFor();
      }
      if (scenario.peopleAccessCheck === "readonly") {
        const forbidden = page.getByRole("button", { name: /(New routing policy|Propose delegation|Create policy|Create route|Propose route|Add route|Create delegation|Approve|Reject|Activate|Revoke|Retire|Add source|Add mapping)/i });
        readonly = { mutationControlsAbsent: await forbidden.count() === 0 };
      }
      if (!contextReceipt && !composer && !definition) await page.evaluate(() => { window.scrollTo({ top: 0, behavior: "instant" }); if (document.activeElement instanceof HTMLElement) document.activeElement.blur(); });
      const peopleAccessLayout = await measurePeopleAccessLayout(page, scenario);
      const observation = await page.evaluate(() => window.peopleAccessEvidence);
      const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
      const record = { ...scenario, buildDigest, evidenceInputDigest: inputs.digest, peopleAccessLayout, context: contextReceipt, history, filters, recovery, missing, readonly, composer, sticky, definition, observation, metrics, errors };
      diagnostic = record;
      const image = path.join(out, `${scenario.name}.png`);
      await page.screenshot({ path: image, fullPage: false, animations: "disabled", caret: "hide" });
      const bytes = await readFile(image);
      Object.assign(record, { imageBytes: bytes.length, imageSHA256: digest(bytes) });
      if (scenario.viewport.width === 390 && ["composer", "definition"].includes(scenario.peopleAccessCheck)) {
        const target = await showPeopleAccessContinuation(page, scenario), name = `${scenario.name}-continuation.png`;
        await page.screenshot({ path: path.join(out, name), fullPage: false, animations: "disabled", caret: "hide" });
        const continuationBytes = await readFile(path.join(out, name));
        record.continuation = { image: name, imageBytes: continuationBytes.length, imageSHA256: digest(continuationBytes), target };
      }
      const failures = peopleAccessCaptureFailures(record, scenario);
      if (failures.length) throw new Error(failures.join("\n"));
      manifest.captures = manifest.captures.filter((item) => item.name !== record.name); manifest.captures.push(record);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      process.stdout.write(`PASS ${scenario.name}\n`);
    } catch (error) {
      manifest.failure = error instanceof Error ? error.message : String(error);
      await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
      await writeFile(path.join(out, `${scenario.name}-failure.json`), JSON.stringify({ scenario, diagnostic, layout: await measurePeopleAccessLayout(page, scenario).catch(() => null), observation: await page.evaluate(() => window.peopleAccessEvidence).catch(() => null) }, null, 2));
      await page.screenshot({ path: path.join(out, `${scenario.name}-failure.png`), fullPage: true }).catch(() => {});
      throw error;
    } finally { await context.close(); }
  }
  if (digest(JSON.stringify(await buildFiles())) !== buildDigest || (await evidenceInputs()).digest !== inputs.digest) {
    manifest.failure = "PeopleAccess evidence build or capture instructions changed during capture";
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
    throw new Error(manifest.failure);
  }
} finally { await browser.close(); }
