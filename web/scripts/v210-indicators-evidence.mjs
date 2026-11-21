import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v210-indicators");
await mkdir(output, { recursive: true });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${process.env.PAGE_URL ?? "http://127.0.0.1:4188"}/#/risks/indicators`);
    await page.getByText("Sample backup failures").waitFor();
    await page.getByText(/^Evaluated /).waitFor();
    assert.equal(await page.getByText(/^Observed /).count(), 0);
    assert.equal(await page.getByText("Unknown", { exact: true }).count(), 1);
    await page.getByText("Latest monitoring result is stale.").waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(output, `indicators-${width}.png`), fullPage: true });
    await page.getByLabel("Indicator type").selectOption("KCI");
    await page.getByText("No indicators found.").waitFor();
    results.push({ width, overflow: false, staleUnknown: true, filteredEmpty: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`Indicator evidence passed at ${results.length} widths.`);
} finally { await browser.close(); }
