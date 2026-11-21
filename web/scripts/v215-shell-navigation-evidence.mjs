import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-shell-navigation");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [390, 768, 1440]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/#/home`);
    await page.getByRole("heading", { name: "Overview" }).waitFor();
    await page.screenshot({ path: path.join(output, `home-${width}.png`), fullPage: true });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    if (width === 390) {
      // Exercise the drawer over a competing positioned feature layer. A
      // lower stacking layer must not intercept the close action.
      await page.evaluate(() => {
        const sticky = document.createElement("div");
        sticky.dataset.testSticky = "true";
        Object.assign(sticky.style, { position: "fixed", inset: "0", zIndex: "1", pointerEvents: "auto" });
        document.querySelector("#v-main").append(sticky);
      });
      await page.getByRole("button", { name: "Open navigation" }).click();
      const close = page.getByRole("button", { name: "Close navigation" });
      const closeAboveContent = await close.evaluate((node) => {
        const rect = node.getBoundingClientRect();
        const top = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
        return rect.width >= 44 && rect.height >= 44 && (top === node || node.contains(top));
      });
      assert.equal(closeAboveContent, true, "navigation close control is covered or too small");
      assert.equal(await close.evaluate((node) => document.activeElement === node), true);
      assert.equal(await page.locator("#v-main").evaluate((node) => node.inert), true);
      assert.equal(await page.locator(".v-topbar").evaluate((node) => node.inert), true);
      assert.equal(await page.locator(".v-skip").evaluate((node) => node.inert), true);
      await page.locator(".v-brand").evaluate((node) => node.focus());
      assert.equal(await close.evaluate((node) => document.activeElement === node), true);
      await page.screenshot({ path: path.join(output, `navigation-${width}.png`), fullPage: true });
      await page.keyboard.press("Shift+Tab");
      assert.equal(await page.getByRole("link", { name: "Work & Approvals" }).evaluate((node) => document.activeElement === node), true);
      await page.keyboard.press("Tab");
      assert.equal(await close.evaluate((node) => document.activeElement === node), true);
      await page.keyboard.press("Escape");
      await page.evaluate(() => document.querySelector("[data-test-sticky]")?.remove());
      assert.equal(await page.getByRole("button", { name: "Open navigation" }).evaluate((node) => document.activeElement === node), true);
      assert.equal(await page.locator(".v-topbar").evaluate((node) => node.inert), false);
      assert.equal(await page.locator(".v-skip").evaluate((node) => node.inert), false);
      await page.getByRole("button", { name: "Open navigation" }).click();
      await page.getByRole("link", { name: "Programs", exact: true }).click();
      await page.getByRole("heading", { name: "Compliance programs" }).waitFor();
      assert.equal(await page.locator("#v-main").evaluate((node) => document.activeElement === node && !node.inert), true);
      await page.getByRole("button", { name: "Open navigation" }).click();
      await page.setViewportSize({ width: 1024, height: 900 });
      await page.waitForFunction(() => !document.querySelector("#v-main").inert && document.body.style.overflow !== "hidden");
      await page.setViewportSize({ width: 390, height: 900 });
      await page.getByRole("button", { name: "Open navigation" }).click();
      await page.evaluate(() => window.dispatchEvent(new Event("clearsight:session-ended")));
      await page.getByRole("heading", { name: "Sign in to ClearSight" }).waitFor();
      assert.equal(await page.evaluate(() => document.body.style.overflow), "");
      await page.getByRole("link", { name: "Sign in" }).focus();
      await page.keyboard.press("Tab");
      assert.notEqual(await page.evaluate(() => document.activeElement?.closest('[role="dialog"]') !== null), true);
      results.push({ width, closeAboveContent, trappedFocus: true, outsideInert: true, escapeRestore: true, destinationFocus: true, resizeUnlock: true, sessionEndUnlock: true, overflow: false });
    } else results.push({ width, overflow: false });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Shell navigation evidence passed at ${results.length} widths.`);
} finally { await browser.close(); }
