import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence", "automation");
const baseline = process.env.UI_EVIDENCE_BASELINE === "1";
const fixturePath = path.resolve("src/staticDemoAutomationRuntime.js");
const sourcePath = "docs/design/references/clearsight-2026-10/source/clearsight-automation-governance-designs/automation-governance-preview.html";
const sourceSHA256 = "7dad1ffc651fd03821cc996ff13f1203cb1ec1344a8a5bdec7b00b49035a7b6f";
const sha = bytes => createHash("sha256").update(bytes).digest("hex");
assert.equal(sha(await readFile(path.resolve("..", sourcePath))), sourceSHA256, "Approved Automation HTML changed.");
const viewports = [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }, { width: 768, height: 900 }, { width: 390, height: 844 }];
const tabLabels = ["Automations", "Schedules", "Escalations", "Activity"];
await mkdir(out, { recursive: true });
async function files(directory, prefix = "") {
  const result = {};
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const name = prefix + entry.name, absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(result, await files(absolute, name + "/"));
    else { const bytes = await readFile(absolute); result[name] = { bytes: bytes.length, sha256: sha(bytes) }; }
  }
  return result;
}
const buildDirectory = path.resolve(process.env.UI_EVIDENCE_BUILD_DIR ?? "dist-evidence");
const assets = await files(buildDirectory);
const report = { kind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", baseline,
  revision: process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA ?? null,
  build: { digest: sha(JSON.stringify(assets)), assets }, source: { path: sourcePath, sha256: sourceSHA256 },
  inputs: { fixtureSHA256: sha(await readFile(fixturePath)), captureSHA256: sha(await readFile(new URL(import.meta.url))), reviewerSHA256: sha(await readFile(new URL("./review-automation-evidence.mjs", import.meta.url))) },
  layouts: [], sources: [], behaviors: [], images: [], failure: null,
  limitations: ["Synthetic transport does not prove live authorization or production acceptance.", "The approved source is light-only; dark renders verify the runtime adaptation.", "Remote source fonts are blocked; environment font fallback affects glyph widths.", "Axe incomplete findings need separate review; this is not full WCAG certification."],
};
const browser = await chromium.launch({ headless: true });
report.browser = browser.version();
async function settle(page) { await page.evaluate(async () => { await document.fonts.ready; await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))); }); }
async function image(page, name, fullPage = false) {
  await settle(page);
  const bytes = await page.screenshot({ path: path.join(out, name), fullPage, animations: "disabled", caret: "hide" });
  report.images.push({ file: name, bytes: bytes.length, sha256: sha(bytes) }); return name;
}
const observation = page => page.evaluate(() => { const { release, ...receipt } = window.automationEvidence; return receipt; });
const workspace = page => page.locator(".automation-workspace");
const records = page => workspace(page).locator(".cs-selectable-record");
const selected = page => page.getByRole("complementary", { name: "Selected automation policy", exact: true });
const tabs = page => page.getByRole("tablist", { name: "Automation and governance views", exact: true });
const param = (page, key) => page.evaluate(key => new URLSearchParams(location.hash.split("?")[1]).get(key), key);
async function open(viewport, theme, scenario = "current", hash = "#configure/automation") {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: viewport.width === 390, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript({ path: fixturePath });
  await context.addInitScript(theme => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, theme);
  const page = await context.newPage(), errors = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.clock.setFixedTime(new Date("2026-11-01T10:00:00Z"));
  await page.goto(baseURL + "/?tour=off&automationCase=" + scenario + hash, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: baseline ? "Automation" : "Automation & governance", exact: true }).waitFor();
  await page.waitForFunction(loading => window.automationEvidence.reads.length > 0 && (loading || window.automationEvidence.pending === 0), scenario === "loading");
  await settle(page);
  return { context, page, errors };
}
async function geometry(page, source = false) {
  return page.evaluate(({ source, baseline }) => {
    const root = document.querySelector(source ? ".auto-content" : baseline ? ".configure-workspace" : ".automation-workspace");
    const box = node => { const r = node?.getBoundingClientRect(); return r ? { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom } : null; };
    const rect = selector => box(root?.querySelector(selector));
    const heading = rect(source ? ".auto-heading" : "h1"), menu = rect(source ? ".auto-tabs" : ".cs-tabs__list"), metrics = rect(source ? ".auto-summary" : ".automation-workspace__metrics");
    const record = root?.querySelector(source ? ".rule-row" : baseline ? ".automation-policy-row" : ".cs-selectable-record");
    const firstRecord = box(record), context = rect(source ? ".rule-detail" : 'aside[aria-label="Selected automation policy"]');
    const layout = rect(source ? ".auto-layout" : ".automation-workspace__layout"), filters = rect(source ? ".auto-toolbar" : ".automation-workspace__filters");
    const metricNode = root?.querySelector(source ? ".auto-summary" : ".automation-workspace__metrics");
    const controls = [...(root?.querySelectorAll("button,input,select,a[href],[role=tab],[role=button]") ?? [])].filter(node => node.getClientRects().length && getComputedStyle(node).visibility !== "hidden").map(node => ({ label: node.getAttribute("aria-label") ?? node.textContent.trim(), ...box(node) }));
    return { heading, menu, metrics, filters, firstRecord, context, layout, controls,
      headings: [...(root?.querySelectorAll("h1") ?? [])].map(node => node.textContent.trim()),
      tabLabels: [...(root?.querySelectorAll(source ? ".auto-tabs [role=tab]" : ".cs-tabs__list [role=tab]") ?? [])].map(node => node.textContent.trim()),
      selectedTabs: [...(root?.querySelectorAll('[role=tab][aria-selected="true"]') ?? [])].map(node => node.textContent.trim()),
      metricColumns: metricNode ? getComputedStyle(metricNode).gridTemplateColumns.split(/\s+/).filter(Boolean).length : 0,
      contextPlacement: firstRecord && context ? context.x >= firstRecord.right ? "beside" : "below" : "missing",
      sourceOrder: !!heading && !!menu && !!metrics && !!firstRecord && heading.bottom <= menu.y + 2 && menu.bottom <= metrics.y + 2 && metrics.bottom <= firstRecord.y + 2,
      firstRecordInViewport: !!firstRecord && firstRecord.y >= 0 && firstRecord.bottom <= innerHeight,
      documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      settingsCurrent: document.querySelector('.sidebar-secondary a[href="#configure"]')?.getAttribute("aria-current"),
    };
  }, { source, baseline });
}
async function accessibility(page) {
  await page.addScriptTag({ path: path.resolve("node_modules/axe-core/axe.min.js") });
  return page.evaluate(async () => {
    const result = await window.axe.run(document.querySelector(".automation-workspace"), { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa"] } });
    const compact = items => items.map(({ id, nodes }) => ({ id, targets: nodes.map(node => node.target) }));
    return { violations: compact(result.violations), incomplete: compact(result.incomplete) };
  });
}
async function contextGeometry(page) {
  return selected(page).evaluate(node => {
    const heading = node.querySelector("h2,h3"), r = heading?.getBoundingClientRect(), shell = document.querySelector(".context-bar")?.getBoundingClientRect();
    return { title: heading?.textContent.trim(), titleVisible: !!r && r.y >= Math.max(0, shell?.bottom ?? 0) && r.bottom <= innerHeight, focused: node === document.activeElement || node.contains(document.activeElement), hash: location.hash };
  });
}
async function behavior(name, viewport, theme, scenario, fn, hash) {
  const { context, page, errors } = await open(viewport, theme, scenario, hash), item = { name, viewport, theme, scenario, checks: {}, errors };
  try {
    await fn(page, item);
    const g = await geometry(page); assert.ok(g.scrollWidth <= g.documentWidth + 1, "Horizontal overflow");
    item.checks.noHorizontalOverflow = true; item.observation = await observation(page);
    assert.equal(item.observation.commands.length, 0, "Unexpected business command");
    assert.equal(errors.length, 0); item.image = await image(page, name + ".png"); item.passed = true;
  } finally { report.behaviors.push(item); await context.close(); }
}
async function choose(page, label, option) {
  await workspace(page).getByRole("button", { name: new RegExp(label) }).click();
  await page.getByRole("option", { name: option, exact: true }).click();
}
try {
  for (const viewport of viewports) {
    const sourceContext = await browser.newContext({ viewport, colorScheme: "light", reducedMotion: "reduce" });
    await sourceContext.route(/^https?:\/\//, route => route.abort());
    const sourcePage = await sourceContext.newPage(); await sourcePage.goto(pathToFileURL(path.resolve("..", sourcePath)).href, { waitUntil: "load" }); await settle(sourcePage);
    const source = { viewport, geometry: await geometry(sourcePage, true), image: await image(sourcePage, "source-" + viewport.width + ".png") };
    if (viewport.width <= 1100) { await sourcePage.locator(".rule-row").first().click(); source.contextImage = await image(sourcePage, "source-" + viewport.width + "-context.png"); }
    report.sources.push(source); await sourceContext.close();
    for (const theme of ["light", "dark"]) {
      const { context, page, errors } = await open(viewport, theme);
      try {
        const g = await geometry(page), item = { viewport, theme, geometry: g, errors, observation: await observation(page), image: await image(page, "layout-" + theme + "-" + viewport.width + ".png") };
        report.layouts.push(item);
        if ([1440, 390].includes(viewport.width)) item.fullImage = await image(page, "layout-" + theme + "-" + viewport.width + "-full.png", true);
        if (!baseline) {
          assert.equal(errors.length, 0); assert.ok(g.sourceOrder, "Heading, tabs, summaries and records must follow source order");
          assert.equal(g.tabLabels.join("|"), tabLabels.join("|")); assert.equal(g.selectedTabs.join("|"), "Automations");
          assert.ok(g.firstRecordInViewport, "The first complete policy must be visible without scrolling");
          assert.ok(g.scrollWidth <= g.documentWidth + 1, "Horizontal overflow");
          assert.equal(g.contextPlacement, source.geometry.contextPlacement); assert.equal(g.metricColumns, source.geometry.metricColumns);
          assert.ok(g.controls.every(box => box.width >= 44 && box.height >= 44), "Primary controls must retain 44px targets");
          item.accessibility = await accessibility(page); assert.equal(item.accessibility.violations.length, 0);
          if (viewport.width <= 1100) {
            await records(page).first().click(); item.contextGeometry = await contextGeometry(page);
            assert.ok(item.contextGeometry.focused && item.contextGeometry.titleVisible, "Selection must reveal and focus the lower context");
            item.contextImage = await image(page, "layout-" + theme + "-" + viewport.width + "-context.png");
          }
        }
      } finally { await context.close(); }
    }
  }
  if (!baseline) {
    for (const [viewport, theme] of [[viewports[0], "light"], [viewports[4], "dark"]]) await behavior("selection-history-" + theme + "-" + viewport.width, viewport, theme, "current", async (page, item) => {
      const search = page.getByRole("searchbox", { name: "Search loaded policies", exact: true });
      await search.fill("routing"); await records(page).filter({ hasText: "Routing review" }).waitFor(); assert.equal(await records(page).count(), 1); assert.equal(await param(page, "q"), "routing");
      item.checks.searchFiltersLoadedPolicies = true;
      const row = records(page).first(); if (viewport.width === 390) await row.tap(); else { await row.focus(); await row.press("Enter"); }
      await selected(page).getByRole("heading", { name: "Routing review", exact: true }).waitFor(); assert.equal(await param(page, "policy"), "sample-auto-routing");
      item.checks.selectionUpdatesContextAndURL = true;
      await page.goBack(); await page.waitForFunction(() => new URLSearchParams(location.hash.split("?")[1]).get("policy") !== "sample-auto-routing");
      await page.goForward(); await selected(page).getByRole("heading", { name: "Routing review", exact: true }).waitFor(); item.checks.historyRestoresSelection = true;
      await page.reload({ waitUntil: "networkidle" }); await selected(page).getByRole("heading", { name: "Routing review", exact: true }).waitFor(); assert.equal(await search.inputValue(), "routing"); item.checks.deepLinkRestoresSelection = true;
      if (viewport.width <= 1100) { await selected(page).getByRole("button", { name: "Back to policies", exact: true }).click(); assert.ok(await records(page).first().evaluate(node => node === document.activeElement)); item.checks.returnRestoresFocus = true; }
      await search.fill("no-matching-policy"); assert.equal(await records(page).count(), 0); item.checks.noMatch = true;
      await search.fill(""); await choose(page, "Status", "Active"); assert.equal(await records(page).count(), 2); item.checks.statusFiltersLatestRevisions = true;
    });
    await behavior("tabs-and-handoff", viewports[0], "light", "current", async (page, item) => {
      for (const label of ["Schedules", "Activity"]) { await tabs(page).getByRole("tab", { name: label, exact: true }).click(); await workspace(page).getByText(/unavailable|not available/i).first().waitFor(); assert.equal(await records(page).count(), 0); }
      item.checks.unavailableTabsAreExplicit = true;
      await page.goBack(); assert.equal(await tabs(page).getByRole("tab", { name: "Schedules", selected: true }).count(), 1); item.checks.historyRestoresTab = true;
      await tabs(page).getByRole("tab", { name: "Escalations", exact: true }).click(); await workspace(page).getByRole("link", { name: /escalation/i }).click();
      await page.waitForFunction(() => location.hash.includes("escalation")); await page.getByRole("heading", { name: "People & access", exact: true }).waitFor(); item.checks.existingEscalationsHandoff = true;
      await page.goBack(); await tabs(page).getByRole("tab", { name: "Escalations", selected: true }).waitFor(); item.checks.returnKeepsAutomationTab = true;
    });
    for (const scenario of ["empty", "truncated", "denied", "failure", "recovery", "readonly", "long", "loading"]) await behavior(scenario, viewports[4], "dark", scenario, async (page, item) => {
      if (["denied", "failure", "recovery"].includes(scenario)) {
        assert.equal(await records(page).count(), 0); await workspace(page).getByText(/not permitted|permission|could not|unable|unavailable|access/i).first().waitFor(); item.checks.failedReadHidesRecords = true;
        if (scenario === "recovery") { await workspace(page).getByRole("button", { name: /retry/i }).click(); await records(page).first().waitFor(); item.checks.retryRestoresRecords = true; }
      } else if (scenario === "empty") { assert.equal(await records(page).count(), 0); await workspace(page).getByText(/No .*polic/i).first().waitFor(); item.checks.emptyPopulationHonest = true; }
      else if (scenario === "truncated") {
        assert.equal(await records(page).count(), 200); item.checks.actualBoundedPopulation = true;
        const disclosure = workspace(page).getByText(/More policies (are )?not shown/i).first(); await disclosure.waitFor();
        item.limitGeometry = await disclosure.evaluate(node => { const r = node.getBoundingClientRect(), shell = document.querySelector(".context-bar")?.getBoundingClientRect(); return { text: node.textContent.trim(), y: r.y, bottom: r.bottom, x: r.x, right: r.right, shellBottom: shell?.bottom ?? 0, width: innerWidth, height: innerHeight }; });
        assert.ok(item.limitGeometry.y >= item.limitGeometry.shellBottom && item.limitGeometry.bottom <= item.limitGeometry.height); item.checks.truncationVisibleInViewport = true;
      } else if (scenario === "loading") {
        assert.equal(await records(page).count(), 0); await workspace(page).getByText(/Loading automation polic/i).waitFor(); item.loadingImage = await image(page, "loading-pending.png");
        await page.evaluate(() => window.automationEvidence.release()); await records(page).first().waitFor(); item.checks.loadingThenRecords = true;
      } else { assert.equal(await records(page).count(), 6); item.checks.recordsReadable = true; }
      if (scenario === "long") { await records(page).first().click(); item.contextGeometry = await contextGeometry(page); assert.ok(item.contextGeometry.titleVisible); item.checks.longContextReadable = true; }
      assert.equal(await workspace(page).getByRole("button", { name: /^(Create|New automation|Edit|Simulate|Activate|Approve|Suspend)/i }).filter({ has: page.locator(":scope:not([disabled])") }).count(), 0);
      item.checks.noUnsupportedCommands = true;
    });
  }
} catch (error) { report.failure = error.stack ?? String(error); process.exitCode = 1; }
finally {
  await browser.close(); report.build.afterDigest = sha(JSON.stringify(await files(buildDirectory)));
  if (report.build.afterDigest !== report.build.digest) { report.failure = [report.failure, "Application assets changed during capture."].filter(Boolean).join("\n"); process.exitCode = 1; }
  await writeFile(path.join(out, "receipt.json"), JSON.stringify(report, null, 2) + "\n");
}
if (report.failure) throw new Error(report.failure);
process.stdout.write((baseline ? "BASELINE" : "PASS") + " Automation: " + report.layouts.length + " layouts, " + report.behaviors.length + " behavior scenarios, " + report.images.length + " images.\n");
