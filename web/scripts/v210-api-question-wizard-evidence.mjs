import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import assert from "node:assert/strict";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4174";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-20-api-question-wizard");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PLAYWRIGHT_BROWSER_PATH });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/?fixture=api-question#/forms/sample-form`, { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "Prepare API question" }).click();
    await page.getByText("Instructions for developers").waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(output, `contract-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Continue" }).click();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(output, `test-${width}.png`), fullPage: true });
    results.push({ width, overflow: false });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`API question wizard rendered at ${results.length} widths.`);
} finally { await browser.close(); }
