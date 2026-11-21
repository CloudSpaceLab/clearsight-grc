import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence-v210/shell-contract");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const routes = [
  ["programs", "/#/programs", "Sample bank assurance"],
  ["forms", "/#/forms", "Sample vendor review"],
  ["vendors", "/#/vendors", "Sample Atlas Ltd"],
  ["risks", "/#/risks", "Payment service disruption"],
  ["rcsa", "/#/risks/rcsa", "Sample annual assessment"],
  ["losses", "/#/risks/losses", "Sample payment outage"],
  ["privacy-isolated", "/privacy-table.html", "Customer onboarding"],
];
const results = [];
try {
  for (const width of [320, 390, 768, 1024, 1280, 1440]) {
    for (const [name, route, sample] of routes) {
      const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
      await page.goto(`${base}${route}`, { waitUntil: "domcontentloaded" });
      const link = page.getByRole("link", { name: sample });
      await link.waitFor();
      // Preserve link controls while exercising each column with a long value.
      await page.locator(".v-program-table tbody td").evaluateAll((cells) => cells.forEach((cell) => {
        const target = cell.querySelector("a") || cell; target.textContent = "PROGRAM-" + "UNBROKENIDENTIFIER".repeat(10);
      }));
      await page.keyboard.press("Tab");
      const focus = await page.evaluate(() => { const active = document.activeElement; if (!(active instanceof HTMLElement)) return null; const css = getComputedStyle(active); return { tag: active.tagName, outline: css.outlineStyle, width: css.outlineWidth }; });
      assert.ok(focus && focus.tag !== "BODY" && focus.outline !== "none" && parseFloat(focus.width) > 0, `${name} focus ${width}: ${JSON.stringify(focus)}`);
      const geometry = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, viewport: innerWidth, cells: [...document.querySelectorAll(".v-program-table td")].map((cell) => ({ width: cell.clientWidth, content: cell.scrollWidth })) }));
      assert.ok(geometry.page <= geometry.viewport && geometry.cells.length > 0 && geometry.cells.every((cell) => cell.content <= cell.width), `${name} overflow ${width}: ${JSON.stringify(geometry)}`);
      await page.screenshot({ path: path.join(output, `${name}-long-identifier-${width}.png`), fullPage: true });
      results.push({ name, width, overflow: false, keyboardFocus: true }); await page.close();
    }
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
} finally { await browser.close(); }
