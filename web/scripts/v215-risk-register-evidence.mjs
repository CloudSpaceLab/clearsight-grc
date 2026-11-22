import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-risk-register");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto("http://127.0.0.1:43866/evidence/risk-register.html", { waitUntil: "domcontentloaded" });
    await page.getByRole("link", { name: "Core banking service interruption", exact: true }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(out, `runtime-${width}.png`), fullPage: true });
    await page.getByLabel("Find a risk").fill("banking");
    await page.getByRole("button", { name: "Search" }).click();
    await page.getByRole("link", { name: "Third-party data exposure", exact: true }).waitFor({ state: "detached" });
    assert.equal(await page.getByText("Counts cover loaded risks matching the search.").count(), 1);
    await page.close();
  }
} finally { await browser.close(); }
