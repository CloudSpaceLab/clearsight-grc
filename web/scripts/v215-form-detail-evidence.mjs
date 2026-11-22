import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43192/evidence/";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-form-detail");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}#/forms/sample-form`, { waitUntil: "domcontentloaded" });
    await page.getByRole("region", { name: "Questions" }).waitFor();
    await page.getByRole("region", { name: "Form details" }).waitFor();
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `overflow at ${width}px`);
    await page.screenshot({ path: path.join(output, `form-${width}.png`), fullPage: true });
    await page.close();
  }
  console.log("Synthetic form detail evidence passed at 1440, 768 and 390px.");
} finally { await browser.close(); }
