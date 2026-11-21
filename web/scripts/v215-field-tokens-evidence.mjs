import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-field-tokens");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    for (const [route, label, suffix] of [["programs", "Search programs", "search"], ["vendors", "Legal name", "intake"]]) {
      await page.goto(`${base}/#/${route}`, { waitUntil: "domcontentloaded" });
      if (route === "vendors") await page.getByRole("button", { name: "Add vendor" }).click();
      const field = page.getByLabel(label, { exact: true });
      await field.waitFor();
      const actual = await field.evaluate((node) => { const c = getComputedStyle(node); return [c.backgroundColor, c.color, c.borderTopColor, c.minHeight, c.borderRadius]; });
      assert.deepEqual(actual, ["rgb(255, 255, 255)", route === "programs" ? "rgb(0, 0, 0)" : "rgb(24, 51, 94)", "rgb(220, 231, 245)", "44px", "8px"]);
      await page.evaluate(() => document.documentElement.style.setProperty("--v-field-background", "rgb(1, 2, 3)"));
      assert.equal(await field.evaluate((node) => getComputedStyle(node).backgroundColor), "rgb(1, 2, 3)");
      await page.evaluate(() => document.documentElement.style.removeProperty("--v-field-background"));
      if (route === "programs") {
        const button = page.locator(".v-program-search button");
        await button.waitFor();
        const buttonStyle = await button.evaluate((node) => { const c = getComputedStyle(node); return [c.backgroundColor, c.color, c.borderTopColor, c.minHeight, c.borderRadius]; });
        assert.deepEqual(buttonStyle, ["rgb(255, 255, 255)", "rgb(24, 51, 94)", "rgb(220, 231, 245)", "44px", "8px"]);
        await page.evaluate(() => document.documentElement.style.setProperty("--v-color-surface", "rgb(1, 2, 3)"));
        assert.equal(await button.evaluate((node) => getComputedStyle(node).backgroundColor), "rgb(1, 2, 3)");
        await page.evaluate(() => document.documentElement.style.removeProperty("--v-color-surface"));
        assert.equal(await button.evaluate((node) => getComputedStyle(node).backgroundColor), "rgb(255, 255, 255)");
      }
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await page.screenshot({ path: path.join(output, `${suffix}-${width}.png`), fullPage: true });
    }
    await page.close();
  }
  console.log("Field tokens verified at 1440, 768 and 390 pixels.");
} finally { await browser.close(); }
