import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence", "work-source");
const baseline = process.env.UI_EVIDENCE_BASELINE === "1";
const fixturePath = path.resolve("src/staticDemoWorkSourceRuntime.js");
const sourcePath = "docs/design/references/clearsight-2026-10/source/clearsight-work-approvals-designs/work-approvals-preview.html";
const sourceSHA256 = "202ed9c2b22ac4e921d9ba14aa41b61644ea54cbe55abcc81a86ec10b10ee8d2";
const sha = (bytes) => createHash("sha256").update(bytes).digest("hex");
const sourceBytes = await readFile(path.resolve("..", sourcePath));
assert.equal(sha(sourceBytes), sourceSHA256, "The approved Work HTML changed.");
const viewports = [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }, { width: 768, height: 900 }, { width: 390, height: 844 }];
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
const assets = await files(path.resolve(process.env.UI_EVIDENCE_BUILD_DIR ?? "dist-evidence"));
const report = { kind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", baseline,
  revision: process.env.UI_EVIDENCE_REVISION ?? process.env.GITHUB_SHA ?? null,
  build: { digest: sha(JSON.stringify(assets)), assets }, source: { path: sourcePath, sha256: sourceSHA256 },
  inputs: { fixtureSHA256: sha(await readFile(fixturePath)), captureSHA256: sha(await readFile(new URL(import.meta.url))) },
  layouts: [], sources: [], behaviors: [], images: [], failure: null };
const browser = await chromium.launch({ headless: true });
async function image(page, name) {
  await page.screenshot({ path: path.join(out, name), animations: "disabled", caret: "hide" });
  const bytes = await readFile(path.join(out, name));
  report.images.push({ file: name, bytes: bytes.length, sha256: sha(bytes) });
  return name;
}
const table = (page) => page.getByRole("table", { name: "Assigned work", exact: true });
const rows = (page) => table(page).locator("tbody tr");
const cards = (page) => page.locator(".assigned-workspace__signals .cs-metric-card");
const sourceReads = (page) => page.evaluate(() => window.workSourceEvidence.reads.filter((entry) => entry.path === "/api/v1/work/assignments"));
async function waitRows(page, count) {
  await page.waitForFunction((expected) => document.querySelectorAll('.assigned-workspace table tbody tr').length === expected, count);
}
async function open(viewport, theme, scenario = "normal") {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: viewport.width === 390, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript({ path: fixturePath });
  await context.addInitScript((theme) => { localStorage.setItem("clearsight.theme", theme); localStorage.setItem("clearsight.density", "comfortable"); }, theme);
  const page = await context.newPage(), errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.clock.setFixedTime(new Date("2026-10-31T10:00:05.123Z"));
  await page.goto(`${baseURL}/?tour=off&workSourceCase=${scenario}#work`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Assigned to me", exact: true }).waitFor();
  await page.waitForFunction(() => window.workSourceEvidence.reads.some((entry) => entry.path.endsWith("assignment-counts")) && window.workSourceEvidence.pending === 0);
  await page.evaluate(() => document.fonts.ready);
  return { context, page, errors };
}
async function observeLayout(page) {
  return page.evaluate(() => {
    const root = document.querySelector(".assigned-workspace");
    const bounds = (selector) => { const box = root.querySelector(selector)?.getBoundingClientRect(); return box ? { x: box.x, y: box.y, width: box.width, height: box.height, bottom: box.bottom } : null; };
    const tabs = bounds(".cs-tabs__list"), measures = bounds(".assigned-workspace__signals"), heading = bounds("#assigned-work-heading"), filters = bounds(".cs-filter-bar"), firstRecord = bounds("tbody tr");
    const controls = [...root.querySelectorAll('.cs-metric-card, .cs-filter-bar button, .cs-filter-bar input')].filter((node) => node.getBoundingClientRect().width > 0);
    return { tabs, measures, heading, filters, firstRecord, context: bounds(".assigned-workspace__context"),
      tabLabels: [...root.querySelectorAll('[role="tab"]')].map((node) => node.textContent.trim()),
      measureLabels: [...root.querySelectorAll('.cs-metric-card')].map((node) => node.getAttribute("aria-label")),
      measureColumns: [...root.querySelectorAll('.cs-metric-card')].filter((node) => Math.abs(node.getBoundingClientRect().y - measures.y) < 2).length,
      controls: controls.map((node) => { const box = node.getBoundingClientRect(); return { width: box.width, height: box.height }; }),
      clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth,
      firstRecordInViewport: Boolean(firstRecord && firstRecord.y < innerHeight - 8),
      sourceOrder: Boolean(tabs && measures && heading && filters && tabs.y < measures.y && measures.bottom <= heading.y + 8 && heading.y < filters.y),
    };
  });
}
async function choose(page, label, option) {
  await page.getByRole("button", { name: new RegExp(label) }).click();
  await page.getByRole("option", { name: option, exact: true }).click();
}
async function verifyDrills(viewport, theme) {
  const { context, page, errors } = await open(viewport, theme);
  const result = { name: `deadline-drills-${theme}-${viewport.width}`, checks: {}, reads: [], errors };
  try {
    if (baseline) {
      await cards(page).nth(0).click();
      await page.waitForTimeout(300);
      result.checks.sourceDrillAbsent = (await sourceReads(page)).length === 0;
      result.checks.renderedRows = await rows(page).count();
      result.image = await image(page, `baseline-missed-drill-${theme}-${viewport.width}.png`);
      assert.equal(result.checks.sourceDrillAbsent, true, "The baseline no longer reproduces the reported defect.");
      return;
    }
    const activateMetric = async (index) => {
      const metric = cards(page).nth(index);
      if (viewport.width === 390) await metric.tap();
      else { await metric.focus(); await page.keyboard.press("Enter"); }
    };
    const search = page.getByRole("searchbox", { name: "Search assignment titles" });
    await search.fill("upcoming");
    await waitRows(page, 50);
    await choose(page, "Work status", "Ready");
    await page.waitForFunction(() => window.workSourceEvidence.reads.some((entry) => entry.query.status === "READY"));
    await choose(page, "Due-date window", "Next 30 days + overdue");
    await activateMetric(0);
    await waitRows(page, 50);
    if (viewport.width !== 390) assert.equal(await cards(page).nth(0).evaluate((node) => node === document.activeElement), true);
    assert.equal(await search.inputValue(), "");
    const first = (await sourceReads(page)).at(-1);
    assert.equal(first.query.q ?? "", "");
    assert.equal(first.query.status, undefined);
    assert.equal(first.query.due_before, "2026-10-31T10:00:00.123456Z");
    assert.equal(first.query.due_from, undefined);
    result.checks.cardClearsOtherFilters = true;
    result.checks.overdueFirstPage = await rows(page).count();
    await page.getByRole("button", { name: "Load more assignments", exact: true }).click();
    await waitRows(page, 61);
    const more = (await sourceReads(page)).at(-1);
    assert.equal(more.query.due_before, first.query.due_before);
    assert.ok(more.query.cursor);
    assert.equal(await table(page).getByRole("row", { name: "Sample just before the deadline window", exact: true }).count(), 1);
    assert.equal(await table(page).getByRole("row", { name: "Sample at the deadline window start", exact: true }).count(), 0);
    result.checks.overdueComplete = true;
    await page.evaluate(() => scrollTo(0, 0));
    result.overdueImage = await image(page, `overdue-complete-${theme}-${viewport.width}.png`);
    await activateMetric(1);
    await waitRows(page, 50);
    const due = (await sourceReads(page)).at(-1);
    assert.equal(due.query.due_from, "2026-10-31T10:00:00.123456Z");
    assert.equal(due.query.due_before, "2026-11-07T10:00:00.123457Z");
    await page.getByRole("button", { name: "Load more assignments", exact: true }).click();
    await waitRows(page, 62);
    for (const name of ["Sample at the deadline window start", "Sample at the deadline window end"]) assert.equal(await table(page).getByRole("row", { name, exact: true }).count(), 1);
    for (const name of ["Sample just before the deadline window", "Sample just after the deadline window"]) assert.equal(await table(page).getByRole("row", { name, exact: true }).count(), 0);
    result.checks.upcomingInclusiveMicrosecondBounds = true;
    result.checks.metricActivation = viewport.width === 390 ? "touch" : "keyboard";
    await page.evaluate(() => scrollTo(0, 0));
    result.upcomingImage = await image(page, `upcoming-complete-${theme}-${viewport.width}.png`);
    await page.getByRole("button", { name: "Clear filters", exact: true }).click();
    await waitRows(page, 4);
    assert.equal(await table(page).getByRole("row", { name: "Sample loaded recovery approval", exact: true }).count(), 1);
    assert.equal(await page.getByRole("button", { name: "Load more assignments", exact: true }).count(), 0);
    result.checks.clearRestoresLoadedFeed = true;
    await cards(page).nth(2).click();
    await waitRows(page, 1);
    assert.equal(await table(page).getByRole("row", { name: "Sample loaded recovery approval", exact: true }).count(), 1);
    result.checks.approvalsStayLoaded = true;
  } finally {
    result.reads = await sourceReads(page);
    result.commands = await page.evaluate(() => window.workSourceEvidence.commands);
    report.behaviors.push(result);
    assert.deepEqual(result.commands, []);
    assert.deepEqual(errors, []);
    await context.close();
  }
}
try {
  for (const viewport of viewports) {
    const sourcePage = await browser.newPage({ viewport });
    await sourcePage.goto(pathToFileURL(path.resolve("..", sourcePath)).href, { waitUntil: "load" });
    await sourcePage.locator("#queue-list").waitFor();
    const sourceGeometry = await sourcePage.evaluate(() => {
      const measureTop = document.querySelector(".signal-row").getBoundingClientRect().y;
      return { measureColumns: [...document.querySelectorAll(".signal-row .signal")].filter((node) => Math.abs(node.getBoundingClientRect().y - measureTop) < 2).length,
        firstRecordY: document.querySelector("#queue-list .queue-row")?.getBoundingClientRect().y ?? null };
    });
    report.sources.push({ viewport, geometry: sourceGeometry, image: await image(sourcePage, `source-${viewport.width}.png`) });
    await sourcePage.close();
    for (const theme of ["light", "dark"]) {
      const { context, page, errors } = await open(viewport, theme);
      try {
        await waitRows(page, 4);
        await page.waitForFunction(() => document.querySelector('.cs-metric-card')?.getAttribute('aria-label')?.includes('61'));
        const geometry = await observeLayout(page);
        const record = { viewport, theme, geometry, image: await image(page, `layout-${theme}-${viewport.width}.png`), errors };
        report.layouts.push(record);
        assert.deepEqual(errors, []);
        if (!baseline) {
          assert.ok(geometry.scrollWidth <= geometry.clientWidth + 1, `Work overflow at ${viewport.width}`);
          assert.ok(geometry.sourceOrder, `Work source anatomy changed at ${viewport.width}`);
          assert.ok(geometry.firstRecordInViewport, `First assignment below viewport at ${viewport.width}`);
          assert.equal(geometry.measureColumns, sourceGeometry.measureColumns, `Work measure columns differ from approved HTML at ${viewport.width}`);
          assert.deepEqual(geometry.tabLabels, ["My work", "Approvals"]);
          assert.ok(geometry.controls.every((box) => box.height >= 44 && box.width >= 44), "Undersized primary control");
          await page.addScriptTag({ path: path.resolve("node_modules/axe-core/axe.min.js") });
          const accessibility = await page.evaluate(async () => {
            const result = await window.axe.run(document.querySelector(".assigned-workspace"), { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa"] } });
            return result.violations.map(({ id, nodes }) => ({ id, targets: nodes.map((node) => node.target) }));
          });
          record.accessibility = accessibility;
          assert.deepEqual(accessibility, []);
        }
      } finally { await context.close(); }
    }
  }
  for (const [viewport, theme] of [[viewports[0], "light"], [viewports[4], "dark"]]) await verifyDrills(viewport, theme);
  if (!baseline) for (const scenario of ["counts-unavailable", "counts-scope", "page-scope"]) {
    const { context, page, errors } = await open(viewports[0], "light", scenario);
    try {
      if (scenario === "page-scope") {
        await cards(page).nth(0).click();
        await page.getByText(/Assignment(s| search)? unavailable|Assigned work could not be loaded/).first().waitFor();
        assert.equal(await rows(page).count(), 0);
      } else {
        await page.getByText(/Assignment totals unavailable/).waitFor();
        assert.match(await cards(page).nth(0).getAttribute("aria-label"), /Loaded work/);
        await cards(page).nth(0).click();
        await waitRows(page, 1);
        assert.equal((await sourceReads(page)).length, 0);
      }
      const observation = await page.evaluate(() => window.workSourceEvidence);
      assert.deepEqual(observation.commands, []);
      assert.deepEqual(errors, []);
      report.behaviors.push({ name: scenario, passed: true, observation, image: await image(page, `${scenario}.png`) });
    } finally { await context.close(); }
  }
} catch (error) { report.failure = error.stack ?? String(error); process.exitCode = 1; }
finally {
  await browser.close();
  await writeFile(path.join(out, "receipt.json"), JSON.stringify(report, null, 2));
}
if (report.failure) throw new Error(report.failure);
process.stdout.write(`${baseline ? "BASELINE" : "PASS"} Work source counts: ${report.layouts.length} layouts, ${report.behaviors.length} behavior scenarios, ${report.images.length} images.\n`);
