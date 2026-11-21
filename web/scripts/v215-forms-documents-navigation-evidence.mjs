import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43112";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-forms-documents-navigation");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/#/forms`);
    const views = page.getByRole("navigation", { name: "Forms views" });
    const link = views.getByRole("link", { name: "Documents" });
    await link.waitFor();
    assert.equal(await link.getAttribute("href"), "#/documents");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `forms-${width}.png`), fullPage: true });
    await link.click();
    await page.getByRole("heading", { name: "Documents", exact: true, level: 1 }).waitFor();
    await page.getByRole("button", { name: "Open Sample access review.png" }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `documents-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, documentRoute: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Forms to Documents navigation verified at ${results.length} widths.`);
} finally { await browser.close(); }
