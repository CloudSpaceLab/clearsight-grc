import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence/integrations");
const source = path.resolve("../docs/design/references/clearsight-2026-10/source/clearsight-integrations-designs/integrations-workspace-preview.html");
const sha256 = createHash("sha256").update(await readFile(source)).digest("hex");
assert.equal(sha256, "871cbaf590a1909d1c40947cd38379f6ca987a101e16f2a1ecd16f024c79814d");
await mkdir(output, { recursive: true });
const report = { evidenceKind: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", sourceSHA256: sha256, revision: process.env.UI_EVIDENCE_REVISION ?? null, layouts: [], interactions: [], failure: null };
const browser = await chromium.launch({ headless: true });
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 1280, 1024, 768, 390]) {
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, reducedMotion: "reduce" });
    await context.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const page = await context.newPage(), errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(`${base}/?tour=off#configure/data`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Integrations & data sources" }).waitFor();
    const geometry = await page.evaluate(() => {
      const box = (selector) => { const r = document.querySelector(selector)?.getBoundingClientRect(); return r && { x: r.x, y: r.y, right: r.right, bottom: r.bottom, width: r.width, height: r.height }; };
      return { heading: box(".integrations-workspace__heading"), tabs: box(".integrations-workspace [role=tablist]"), source: box(".integrations-workspace__layout > section"), imports: box(".integrations-workspace__layout > aside"), documentWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth };
    });
    assert.ok(geometry.heading.bottom <= geometry.tabs.y && geometry.tabs.bottom <= geometry.source.y);
    assert.ok(geometry.source.bottom < (width === 390 ? 844 : 900));
    assert.ok(geometry.scrollWidth <= geometry.documentWidth + 1);
    assert.ok(width > 820 ? geometry.imports.x >= geometry.source.right : geometry.imports.y >= geometry.source.bottom);
    assert.deepEqual(await page.getByRole("tab").allTextContents(), ["Sources", "Field mapping", "Sync history"]);
    assert.equal(await page.getByRole("tab", { name: "Sources" }).getAttribute("aria-selected"), "true");
    await page.getByText("Connected sources are unavailable.").waitFor();
    await page.getByRole("button", { name: "Open document imports" }).waitFor();
    assert.deepEqual(errors, []);
    const image = `layout-${theme}-${width}.png`;
    await page.screenshot({ path: path.join(output, image), animations: "disabled" });
    report.layouts.push({ theme, width, geometry, image, sha256: createHash("sha256").update(await readFile(path.join(output, image))).digest("hex") });
    if (width === 390 || width === 1440) {
      await page.getByRole("tab", { name: "Sources" }).focus();
      await page.keyboard.press("ArrowRight");
      await page.getByText("Field mappings are unavailable.").waitFor();
      assert.equal(new URL(page.url()).hash, "#configure/data?tab=mapping");
      await page.keyboard.press("ArrowRight");
      await page.getByText("Sync history is unavailable.").waitFor();
      await page.goBack(); await page.getByText("Field mappings are unavailable.").waitFor();
      await page.goForward(); await page.getByText("Sync history is unavailable.").waitFor();
      await page.getByRole("tab", { name: "Sources" }).click();
      await page.getByRole("button", { name: "Open document imports" }).click();
      await page.waitForURL(/#imports/);
      report.interactions.push({ theme, width, keyboard: true, history: true, imports: true });
    }
    await context.close();
  }
  const page = await browser.newPage();
  await page.goto(`${base}/?fixture=no-config-access#configure/data`, { waitUntil: "networkidle" });
  assert.equal(new URL(page.url()).hash, "#oversight");
  assert.equal(await page.getByRole("heading", { name: "Integrations & data sources" }).count(), 0);
  report.denied = true;
} catch (error) { report.failure = error.stack ?? String(error); process.exitCode = 1; }
finally { await browser.close(); await writeFile(path.join(output, "receipt.json"), JSON.stringify(report, null, 2)); }
if (report.failure) throw new Error(report.failure);
process.stdout.write(`PASS integrations: ${report.layouts.length} layouts, ${report.interactions.length} navigation checks and denied configuration.\n`);
