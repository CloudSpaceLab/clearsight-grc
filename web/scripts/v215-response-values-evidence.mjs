import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const axe = require("axe-core");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-response-values");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/?fixture=response-values#/forms/responses/sample-response`);
    await page.getByRole("heading", { name: "Sample vendor review" }).waitFor();
    const rows = page.locator(".v-response-answers dl");
    assert.match(await rows.innerText(), /1250000\.50/);
    assert.match(await rows.innerText(), /98\.70%/);
    assert.match(await rows.innerText(), /Confirmed/);
    assert.match(await rows.innerText(), /2 documents submitted/);
    assert.match(await rows.innerText(), /Signature submitted/);
    const text = page.getByText("Review notes", { exact: true });
    assert.equal(await text.evaluate((el) => el.nextElementSibling?.textContent), "Access checked for October.\nTwo exceptions require follow-up.");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.addScriptTag({ content: axe.source });
    const accessibility = await page.evaluate(async () => window.axe.run(document, { rules: { "color-contrast": { enabled: false } } }));
    assert.deepEqual(accessibility.violations.map((v) => v.id), []);
    await page.screenshot({ path: path.join(output, `response-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, axeViolations: 0 });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", colorContrastExcluded: true, results }, null, 2));
  console.log(`Response values verified at ${results.length} widths.`);
} finally { await browser.close(); }
