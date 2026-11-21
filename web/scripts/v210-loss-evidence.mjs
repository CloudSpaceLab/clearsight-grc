import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4179";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v210-loss");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: width === 390 ? 844 : 900 }, reducedMotion: "reduce" });
    const errors = []; page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(`${base}/#/risks/losses`);
    await page.getByRole("heading", { name: "Loss events", exact: true }).waitFor();
    await page.getByRole("link", { name: "Sample payment outage" }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(output, `loss-register-${width}.png`), fullPage: true });
    await page.getByRole("link", { name: "Sample payment outage" }).click();
    await page.getByRole("heading", { name: "Sample payment outage" }).waitFor();
    await page.getByRole("heading", { name: "Recovery history" }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    assert.deepEqual(errors, []);
    await page.screenshot({ path: path.join(output, `loss-detail-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, sampleOnly: true }); await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Loss register and detail rendered at ${results.length} widths.`);
} finally { await browser.close(); }
