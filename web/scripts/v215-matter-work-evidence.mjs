import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const axe = require("axe-core");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-matter-work");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/?fixture=matter-work#/work`);
    const link = page.getByRole("link", { name: "Review the sample access assessment" });
    await link.waitFor();
    await page.screenshot({ path: path.join(output, `work-${width}.png`), fullPage: true });
    await link.focus(); await page.keyboard.press("Enter");
    await page.getByRole("heading", { name: "Review privileged access exceptions" }).waitFor();
    assert.equal(await page.getByText("Decision unavailable.").count(), 1);
    assert.equal(await page.getByRole("button", { name: /approve|reject/i }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.addScriptTag({ content: axe.source });
    const accessibility = await page.evaluate(async () => window.axe.run(document, { rules: { "color-contrast": { enabled: false } } }));
    assert.deepEqual(accessibility.violations.map((violation) => violation.id), []);
    await page.screenshot({ path: path.join(output, `issue-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, axeViolations: 0, keyboardLink: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", colorContrastExcluded: true, results }, null, 2));
  console.log(`Matter work verified at ${results.length} widths.`);
} finally { await browser.close(); }
