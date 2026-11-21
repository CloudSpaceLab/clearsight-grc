import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-status-tokens");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/#/home`);
    const high = page.locator(".v-status--high").first();
    await high.waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    const before = await high.evaluate((el) => ({ color: getComputedStyle(el).color, background: getComputedStyle(el).backgroundColor }));
    assert.deepEqual(before, { color: "rgb(160, 53, 53)", background: "rgb(255, 240, 238)" });
    await page.evaluate(() => document.documentElement.style.setProperty("--v-color-danger-background", "rgb(1, 2, 3)"));
    assert.equal(await high.evaluate((el) => getComputedStyle(el).backgroundColor), "rgb(1, 2, 3)");
    await page.evaluate(() => document.documentElement.style.removeProperty("--v-color-danger-background"));
    assert.equal(await high.evaluate((el) => getComputedStyle(el).backgroundColor), before.background);
    await page.screenshot({ path: path.join(output, `home-${width}.png`), fullPage: true });
    await page.goto(`${base}/#/forms/sample-form`);
    const neutral = page.locator(".v-status").first();
    await neutral.waitFor();
    assert.deepEqual(await neutral.evaluate((el) => ({ color: getComputedStyle(el).color, background: getComputedStyle(el).backgroundColor })), { color: "rgb(87, 114, 146)", background: "rgb(241, 245, 251)" });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `form-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, defaultColorsPreserved: true, semanticOverride: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Status tokens verified at ${results.length} widths.`);
} finally { await browser.close(); }
