import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43185/evidence/";
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-vendor-detail");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}#/vendors/sample-vendor`, { waitUntil: "domcontentloaded" });
    await page.getByRole("heading", { name: "Hosting", exact: true }).waitFor();
    await page.getByRole("heading", { name: "Due diligence", exact: true }).waitFor();
    const facts = page.getByRole("region", { name: "Relationship details" });
    assert.match(await facts.innerText(), /Important/);
    assert.match(await facts.innerText(), /Processor/);
    assert.match(await facts.innerText(), /Not recorded/);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `overflow at ${width}`);
    await page.screenshot({ path: path.join(out, `vendor-detail-${width}.png`), fullPage: true });
    await page.close();
  }
  console.log("Vendor relationship detail captured at five widths.");
} finally { await browser.close(); }
