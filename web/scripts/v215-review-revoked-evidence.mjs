import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-review-revoked");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.addInitScript(() => {
      let inner = window.fetch;
      let reviewReads = 0;
      Object.defineProperty(window, "fetch", { configurable: true, get: () => async (...args) => {
        const url = new URL(typeof args[0] === "string" ? args[0] : args[0] instanceof URL ? args[0].href : args[0].url, location.origin);
        if (url.pathname === "/api/v1/forms/responses/sample-response/review") {
          reviewReads++;
          if (reviewReads > 1) return new Response(JSON.stringify({ message: "Forbidden" }), { status: 403, headers: { "Content-Type": "application/json" } });
          return new Response(JSON.stringify({ version: 1, may_review: true, items: [{ id: "sample-review", decision: "REVIEWED", reviewer_display_name: "Sample Reviewer", reviewed_at: "2026-11-20T11:42:00Z" }], next_before_version: 1 }), { headers: { "Content-Type": "application/json" } });
        }
        return inner(...args);
      }, set: (value) => { inner = value; } });
    });
    await page.goto(`${base}/#/forms/responses/sample-response`);
    await page.getByText("Sample Reviewer", { exact: false }).waitFor();
    await page.getByRole("button", { name: "Load earlier reviews" }).click();
    await page.getByText("Review unavailable.").waitFor();
    assert.equal(await page.getByText("Sample Reviewer", { exact: false }).count(), 0);
    assert.equal(await page.getByRole("button", { name: "Save review" }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `review-${width}.png`), fullPage: true });
    results.push({ width, deniedReviewRemoved: true, overflow: false });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Denied review recovery verified at ${results.length} widths.`);
} finally { await browser.close(); }
