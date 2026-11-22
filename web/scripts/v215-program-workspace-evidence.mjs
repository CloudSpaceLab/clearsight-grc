import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-program-workspace");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43911/evidence/";
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}#/programs/sample-program`, { waitUntil: "domcontentloaded" });
    await page.getByRole("heading", { name: "Sample bank assurance" }).waitFor();
    await page.getByRole("heading", { name: "Requirements", exact: true }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `overflow at ${width}px`);
    assert.equal(await page.getByRole("link", { name: "Open issues" }).getAttribute("href"), "#/programs/sample-program/issues");
    await page.screenshot({ path: path.join(output, `workspace-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Requirements", exact: true }).focus();
    await page.keyboard.press("Enter");
    assert.equal(await page.getByRole("region", { name: "Program requirements" }).evaluate((node) => document.activeElement === node), true);
    await page.close();
  }
  console.log("Synthetic Program workspace evidence passed at five widths.");
} finally { await browser.close(); }
