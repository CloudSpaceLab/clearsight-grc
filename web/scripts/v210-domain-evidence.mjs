import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence/v210-domains");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    for (const [route, name] of [["home", "home"], ["risks", "risk-register"], ["risks/rcsa", "rcsa"], ["privacy", "privacy-unavailable"], ["documents", "documents"]]) {
      await page.goto(`${base}/#/${route}`); await page.reload();
      if (route === "home") await page.getByLabel("From").waitFor();
      if (route === "risks") await page.getByRole("link", { name: "Payment service disruption" }).waitFor();
      if (route === "risks/rcsa") await page.getByRole("link", { name: "Sample annual assessment" }).waitFor();
      if (route === "privacy") await page.getByRole("heading", { name: "Privacy unavailable" }).waitFor();
      if (route === "documents") {
        await page.getByRole("button", { name: "Open Sample access review.png" }).click();
        await page.getByRole("button", { name: "Grid view" }).click();
        const preview = page.locator('img[alt="Preview: Sample access review.png"]');
        await preview.waitFor();
        await page.waitForFunction(() => {
          const image = document.querySelector('img[alt="Preview: Sample access review.png"]');
          return image instanceof HTMLImageElement && image.complete && image.naturalWidth >= 500 && image.naturalHeight >= 600;
        });
      }
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `${name} ${width} overflow`);
      await page.evaluate(() => scrollTo(0, 0));
      await page.screenshot({ path: path.join(output, `${name}-${width}.png`), fullPage: true });
    }
    results.push({ width, overflow: false }); await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`V2.10 domain evidence passed at ${results.length} widths.`);
} finally { await browser.close(); }
