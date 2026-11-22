import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import path from "node:path";

// Run against the synthetic evidence entry, never the customer entry.
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43192/evidence/";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-v215-response-detail");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}#/forms/responses/sample-program-response`, { waitUntil: "domcontentloaded" });
    await page.getByRole("heading", { name: "Sample access assessment" }).waitFor();
    await page.getByRole("heading", { name: "Answers", exact: true }).waitFor();
    const aside = page.getByRole("complementary", { name: "Response review" });
    await aside.getByRole("heading", { name: "Review", exact: true }).waitFor();
    const bounds = await page.evaluate(() => {
      const main = document.querySelector(".v-response-review-main").getBoundingClientRect();
      const side = document.querySelector(".v-response-review-side").getBoundingClientRect();
      return { main: { right: main.right, bottom: main.bottom }, side: { left: side.left, top: side.top }, scroll: document.documentElement.scrollWidth, viewport: innerWidth };
    });
    assert.ok(bounds.scroll <= bounds.viewport, `document overflow at ${width}px`);
    if (width > 1000) assert.ok(bounds.side.left >= bounds.main.right, `review overlaps answers at ${width}px`);
    else assert.ok(bounds.side.top >= bounds.main.bottom, `review does not stack at ${width}px`);
    await page.screenshot({ path: path.join(output, `response-review-${width}.png`), fullPage: true });
    await page.close();
  }
  console.log("Synthetic response review evidence passed at 1440, 1280, 1024, 768 and 390px.");
} finally {
  await browser.close();
}
