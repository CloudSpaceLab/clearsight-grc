import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-20-v210-form-choices");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/#/forms/new`);
    await page.getByLabel("Answer type 1").selectOption("single_select");
    await page.getByLabel("Options 1").fill("Yes\nNo");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(output, `form-${width}.png`), fullPage: true });
    await page.goto(`${base}/#/vendors/sample-vendor`);
    await page.getByRole("button", { name: "Select document for Audit report" }).click();
    await page.getByText("No eligible documents in the loaded results.").waitFor();
    assert.equal(await page.getByRole("button", { name: "Link document" }).isDisabled(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `vendor-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, emptyDocumentLinkDisabled: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`Form choice and vendor document evidence passed at ${results.length} widths.`);
} finally { await browser.close(); }
