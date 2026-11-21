import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-review-unavailable");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${process.env.PAGE_URL ?? "http://127.0.0.1:43114"}/#/forms/responses/sample-response`);
    await page.getByText("Review unavailable.", { exact: true }).waitFor();
    assert.equal(await page.getByText("You cannot review this response.", { exact: true }).count(), 0);
    assert.equal(await page.getByRole("button", { name: "Save review" }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `response-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, reviewUnavailable: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Review unavailable verified at ${results.length} widths.`);
} finally { await browser.close(); }
