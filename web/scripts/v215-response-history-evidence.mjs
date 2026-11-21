import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const axe = require("axe-core");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-response-history");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/?fixture=response-values#/forms/responses/sample-response?distribution_id=other`);
    await page.getByRole("heading", { name: "Response unavailable" }).waitFor();
    assert.equal(await page.getByText("Access checked for October.").count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.addScriptTag({ content: axe.source });
    const accessibility = await page.evaluate(async () => window.axe.run(document, { rules: { "color-contrast": { enabled: false } } }));
    assert.deepEqual(accessibility.violations.map((violation) => violation.id), []);
    await page.screenshot({ path: path.join(output, `unavailable-${width}.png`), fullPage: true });
    await page.close();
    const missing = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await missing.addInitScript(() => {
      const original = Response.prototype.json;
      Response.prototype.json = async function () {
        const value = await original.call(this);
        if (value?.response?.id === "sample-response") {
          value.response.distribution_id = "";
          value.revision.supersedes_revision_id = "sample-prior";
        }
        return value;
      };
    });
    await missing.goto(`${base}/?fixture=response-values#/forms/responses/sample-response`);
    await missing.getByText("Earlier response unavailable.").waitFor();
    assert.equal(await missing.getByRole("link", { name: "Previous response" }).count(), 0);
    assert.equal(await missing.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await missing.screenshot({ path: path.join(output, `missing-distribution-${width}.png`), fullPage: true });
    await missing.close();
    const unknown = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await unknown.addInitScript(() => {
      const originalJSON = Response.prototype.json;
      Response.prototype.json = async function () {
        const value = await originalJSON.call(this);
        if (value && value.version === 0 && "may_review" in value) value.may_review = true;
        return value;
      };
      let currentFetch = globalThis.fetch;
      Object.defineProperty(globalThis, "fetch", {
        configurable: true,
        get: () => currentFetch,
        set: (value) => {
          currentFetch = (input, init) => {
            if (String(input).includes("/api/v1/forms/responses/sample-response/review") && init?.method === "POST") {
              return Promise.resolve(new Response(JSON.stringify({ id: "sample-review", response_id: "other", response_revision: 1, version: 1, decision: "REVIEWED" }), { status: 200, headers: { "Content-Type": "application/json" } }));
            }
            return value(input, init);
          };
        },
      });
    });
    await unknown.goto(`${base}/?fixture=response-values#/forms/responses/sample-response`);
    await unknown.getByLabel("Review decision").selectOption("REVIEWED");
    await unknown.getByRole("button", { name: "Save review" }).click();
    await unknown.getByText("Review status unknown. Reload before trying again.").waitFor();
    assert.equal(await unknown.getByRole("button", { name: "Save review" }).count(), 0);
    assert.equal(await unknown.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await unknown.evaluate(() => { if (document.activeElement instanceof HTMLElement) document.activeElement.blur(); window.scrollTo(0, 0); });
    await unknown.screenshot({ path: path.join(output, `receipt-unknown-${width}.png`), fullPage: true });
    await unknown.close();
    results.push({ width, overflow: false, axeViolations: 0, missingDistributionBlocked: true, mismatchedReceiptBlocked: true });
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", axeScope: "response_unavailable_only", colorContrastExcluded: true, results }, null, 2));
  console.log(`Response scope rejection verified at ${results.length} widths.`);
} finally { await browser.close(); }
