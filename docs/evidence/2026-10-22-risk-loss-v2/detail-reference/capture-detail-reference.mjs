import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const scriptPath = fileURLToPath(import.meta.url);
const supplementDirectory = path.dirname(scriptPath);
const repositoryRoot = path.resolve(process.env.RISK_LOSS_REPOSITORY_ROOT ?? path.join(supplementDirectory, "../clearsight-grc"));
const runtimeDirectory = path.resolve(process.env.RISK_LOSS_RUNTIME_EVIDENCE ?? path.join(supplementDirectory, "../risk-loss-closeout"));
const outputDirectory = path.resolve(process.env.RISK_LOSS_DETAIL_OUTPUT ?? supplementDirectory);
const sourcePath = "docs/design/references/clearsight-2026-10/source/clearsight-operational-risk-designs/operational-risk-preview.html";
const sourceSHA256 = "cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd";
const runtimeRevision = "527b348631e2f92fc920125b7da28692a2743e92";
const digest = (bytes) => createHash("sha256").update(bytes).digest("hex");
const fileBinding = async (file) => { const bytes = await readFile(file); return { bytes: bytes.length, sha256: digest(bytes) }; };
async function directoryBinding(directory, prefix = "") {
  const result = {};
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const name = `${prefix}${entry.name}`, absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(result, await directoryBinding(absolute, `${name}/`));
    else if (entry.isFile()) result[name] = await fileBinding(absolute);
  }
  return result;
}

const sourceAbsolute = path.join(repositoryRoot, sourcePath);
const sourceBinding = await fileBinding(sourceAbsolute);
assert.equal(sourceBinding.sha256, sourceSHA256, "Immutable approved HTML hash differs");
const coreBefore = await directoryBinding(runtimeDirectory);
const runtimeManifest = JSON.parse(await readFile(path.join(runtimeDirectory, "manifest.json"), "utf8"));
assert.equal(runtimeManifest.failure, null);
assert.equal(runtimeManifest.runtimeBuild.revision, runtimeRevision);
assert.equal(runtimeManifest.captures.length, 68);
assert.ok(!outputDirectory.startsWith(`${runtimeDirectory}${path.sep}`) && outputDirectory !== runtimeDirectory, "Supplement must remain outside core evidence");
const views = [
  { source: "rcsa", runtime: "rcsa", attribute: "data-rcsa", selectedID: "0", selectedLabel: "Head Office", title: "Head Office assessment",
    adaptation: "INSPECTOR_CONTENT_AND_WORKFLOW_ADAPTATION", explanation: "The original illustrative Head Office review percentage and pending total are not the runtime Q3 Technology RCSA population. Runtime retains the 480px/full-phone inspector and presents frozen risk/control records plus the authorized challenge handoff." },
  { source: "kri", runtime: "insights", attribute: "data-indicator", selectedID: "KRI-104", selectedLabel: "Failed privileged logins", title: "Failed privileged logins",
    adaptation: "INSPECTOR_CONTENT_AND_WORKFLOW_ADAPTATION", explanation: "The original KRI-104 illustrative count is a different record from the runtime Mobile transaction success rate indicator. Runtime retains the 480px/full-phone inspector while showing native percentages, observation history and its Program handoff." },
  { source: "losses", runtime: "losses", attribute: "data-loss", selectedID: "LOSS-0146", selectedLabel: "Account settlement exception", title: "Account settlement exception",
    adaptation: "INSPECTOR_TO_STANDALONE_ROUTE_WORKFLOW_ADAPTATION", explanation: "The original illustrative LOSS-0146 inspector is a different record from runtime Duplicate merchant settlement. Runtime deliberately uses its existing standalone exact-record route with distinct gross, recovery and net amounts, recovery history and a Back to losses action; inspector geometry is not claimed as equivalent." },
];
const viewports = [{ width: 1440, height: 900 }, { width: 390, height: 844 }];
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
await mkdir(outputDirectory, { recursive: true });
const browser = await chromium.launch({ headless: true });
const manifest = {
  generatedAt: new Date().toISOString(), status: "RUNNING", failure: null,
  evidenceKind: "IMMUTABLE_HTML_SELECTED_RECORD_REFERENCE_SUPPLEMENT", comparisonScope: "STRUCTURE_AND_WORKFLOW_ADAPTATION_ONLY",
  sourcePath, sourceSHA256, sourceBytes: sourceBinding.bytes, sourceTheme: "light",
  runtimeRevision, runtimeBuildDigest: runtimeManifest.runtimeBuild.digest,
  runtimeManifest: coreBefore["manifest.json"], captureScript: { file: path.basename(scriptPath), ...await fileBinding(scriptPath) },
  browser: { engine: "Chromium", version: browser.version(), networkPolicy: "OFFLINE_AND_ALL_HTTP_HTTPS_REQUESTS_ABORTED", sourceEdits: false },
  sameRecord: false, pixelEquivalence: false, visualParity: "NOT_ASSESSED", liveAuthority: "NOT_ASSESSED",
  excluded: "Risks has no dedicated original register or record detail; no dedicated selected-record source comparison is asserted.",
  captures: [], preservedCoreEvidence: { files: coreBefore, fileCount: Object.keys(coreBefore).length, unchanged: null },
};
try {
  for (const view of views) for (const viewport of viewports) {
    const runtimeTheme = viewport.width === 390 ? "dark" : "light";
    const runtime = runtimeManifest.captures.find((item) => item.riskLossView === view.runtime && item.riskLossCheck === "detail"
      && item.theme === runtimeTheme && item.viewport.width === viewport.width && item.viewport.height === viewport.height);
    assert.ok(runtime?.detail?.exactTarget && runtime.detail.titleMatches, "Exact runtime reader receipt is absent");
    const runtimeFile = `${runtime.name}.png`, runtimeImage = await fileBinding(path.join(runtimeDirectory, runtimeFile));
    assert.equal(runtimeImage.sha256, runtime.imageSHA256); assert.equal(runtimeImage.bytes, runtime.imageBytes);
    const context = await browser.newContext({ viewport, colorScheme: "light", hasTouch: viewport.width === 390, reducedMotion: "reduce", deviceScaleFactor: 1, locale: "en-GB", timezoneId: "UTC" });
    const errors = [], blockedNetwork = [], successfulNetwork = [];
    try {
      await context.setOffline(true);
      await context.route(/^https?:\/\//, (route) => { blockedNetwork.push(route.request().url()); return route.abort(); });
      const page = await context.newPage();
      page.on("pageerror", (error) => errors.push(error.message));
      page.on("response", (response) => { if (/^https?:\/\//.test(response.url())) successfulNetwork.push(response.url()); });
      await page.goto(pathToFileURL(sourceAbsolute).href, { waitUntil: "load" });
      await page.evaluate(() => document.fonts.ready);
      const tab = page.locator(`[role="tab"][data-tab="${view.source}"]`);
      await tab.click(); assert.equal(await tab.getAttribute("aria-selected"), "true");
      const pane = page.locator(`#pane-${view.source}`); await pane.waitFor({ state: "visible" });
      const trigger = pane.locator(`button[${view.attribute}]`).first();
      assert.equal(await trigger.getAttribute(view.attribute), view.selectedID);
      const selectedLabel = await trigger.evaluate((node) => node.closest("article, tr")?.querySelector("strong")?.textContent.trim());
      assert.equal(selectedLabel, view.selectedLabel);
      await trigger.click();
      await page.getByRole("dialog").getByRole("heading", { name: view.title, exact: true }).waitFor();
      const geometry = await page.evaluate(() => {
        const rect = (node) => { const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom }; };
        const drawer = document.querySelector("#risk-drawer"), body = document.querySelector("#drawer-body"), close = document.querySelector("#drawer-x"), action = document.querySelector("#drawer-confirm");
        return { placement: "right-docked-inspector", inspector: rect(drawer), title: rect(document.querySelector("#drawer-title")),
          body: { ...rect(body), clientWidth: body.clientWidth, scrollWidth: body.scrollWidth, clientHeight: body.clientHeight, scrollHeight: body.scrollHeight, scrollTop: body.scrollTop },
          close: { ...rect(close), label: close.getAttribute("aria-label") }, primaryAction: { ...rect(action), label: action.textContent.trim() },
          titleText: document.querySelector("#drawer-title").textContent.trim(), kicker: document.querySelector("#drawer-kicker").textContent.trim(),
          selectedTab: document.querySelector('.risk-tabs [aria-selected="true"]').textContent.trim(), focusTarget: document.activeElement.id,
          fields: Object.fromEntries([...drawer.querySelectorAll("dl > div")].map((row) => [row.querySelector("dt").textContent.trim(), row.querySelector("dd").textContent.trim()])),
          metrics: [...drawer.querySelectorAll(".drawer-kpi > div")].map((node) => ({ label: node.querySelector("small").textContent.trim(), value: node.querySelector("strong").textContent.trim() })),
          documentWidth: document.documentElement.clientWidth, documentScrollWidth: document.documentElement.scrollWidth };
      });
      const inspectorWidth = Math.min(480, viewport.width);
      assert.equal(geometry.inspector.width, inspectorWidth); assert.equal(geometry.inspector.height, viewport.height);
      assert.equal(geometry.inspector.x, viewport.width - inspectorWidth); assert.equal(geometry.inspector.right, viewport.width);
      assert.equal(geometry.titleText, view.title); assert.equal(geometry.focusTarget, "drawer-x");
      assert.ok(geometry.documentScrollWidth <= viewport.width + 1); assert.ok(geometry.body.scrollWidth <= geometry.body.clientWidth + 1);
      assert.deepEqual(errors, []); assert.deepEqual(successfulNetwork, []);
      const image = `source-${view.source}-selected-${viewport.width}x${viewport.height}.png`;
      await page.screenshot({ path: path.join(outputDirectory, image), fullPage: false, animations: "disabled", caret: "hide" });
      manifest.captures.push({ id: `${view.source}-selected-${viewport.width}`, sourcePath, sourceSHA256, sourceTheme: "light", viewport,
        sourceSelection: { tab: view.source, triggerAttribute: view.attribute, triggerValue: view.selectedID, collectionLabel: selectedLabel, inspectorTitle: geometry.titleText },
        image: { file: image, ...await fileBinding(path.join(outputDirectory, image)) }, geometry, errors, blockedNetwork, successfulNetwork,
        runtimePair: { file: runtimeFile, ...runtimeImage, revision: runtimeRevision, theme: runtimeTheme, route: runtime.detail.route, recordID: runtime.detailID, recordTitle: runtime.firstRecord, detail: runtime.detail },
        adaptation: view.adaptation, explanation: view.explanation, sameRecord: false, pixelEquivalence: false,
        themeComparison: runtimeTheme === "light" ? "LIGHT_SOURCE_TO_LIGHT_RUNTIME" : "LIGHT_SOURCE_TO_DARK_RUNTIME_ADAPTATION" });
      console.log(`PASS ${view.source} selected inspector ${viewport.width}x${viewport.height}: ${geometry.titleText}`);
    } finally { await context.close(); }
  }
  assert.equal(manifest.captures.length, 6);
  assert.deepEqual(await fileBinding(sourceAbsolute), sourceBinding, "Approved source changed during capture");
  assert.deepEqual(await directoryBinding(runtimeDirectory), coreBefore, "Core runtime/source evidence changed during supplement capture");
  manifest.preservedCoreEvidence.unchanged = true;
  manifest.status = "PASS";
} catch (error) {
  manifest.failure = error instanceof Error ? error.message : String(error); manifest.status = "FAIL"; throw error;
} finally {
  await browser.close();
  await writeFile(path.join(outputDirectory, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n");
}
