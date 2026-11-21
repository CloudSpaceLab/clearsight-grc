import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-program-projection");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(process.env.PAGE_URL ?? "http://127.0.0.1:43113/", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === "/api/v1/program-summaries") {
          const provenance = { overall_state: "CURRENT", state_label: "Up to date", program_version: 5, assessed_program_version: 5, projection_version: 2, projection_stale: false, state_generated_at: "2026-11-20T10:00:00Z" };
          const items = [
            { ...provenance, assessed_program_version: 4, program: { id: "stale", name: "Sample access program", code: "P-01", version: 5 }, requirement_count: 8, open_matter_count: 1 },
            { program: { id: "unknown", name: "Sample vendor program", code: "P-02", version: 5 }, state_label: "Up to date", overall_state: "CURRENT" },
            { ...provenance, program: { id: "current", name: "Sample privacy program", code: "P-03", version: 5 }, requirement_count: 6, open_matter_count: 0 },
          ];
          return new Response(JSON.stringify({ items, generated_at: "2026-11-20T10:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } });
        }
        return original(input, init);
      };
      location.hash = "#/programs";
    });
    const stale = page.getByRole("row").filter({ has: page.getByRole("link", { name: "Sample access program" }) });
    const unknown = page.getByRole("row").filter({ has: page.getByRole("link", { name: "Sample vendor program" }) });
    const current = page.getByRole("row").filter({ has: page.getByRole("link", { name: "Sample privacy program" }) });
    await stale.getByText("Out of date").waitFor();
    assert.equal(await page.locator("th").filter({ hasText: "Visible open issues" }).count(), 1);
    assert.equal(await stale.locator(".v-mobile-label").filter({ hasText: "Visible open issues" }).count(), 1);
    assert.equal(await stale.getByText("Up to date").count(), 0);
    assert.equal(await unknown.getByText("Unknown").count(), 3);
    await current.getByText("Up to date").waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `programs-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, stale: "Out of date", incomplete: "Unknown", current: "Up to date" });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Program projection states verified at ${results.length} widths.`);
} finally { await browser.close(); }
