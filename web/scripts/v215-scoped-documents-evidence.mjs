import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-22-scoped-documents");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(process.env.PAGE_URL ?? "http://127.0.0.1:43117/", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === "/api/v1/forms/documents") {
          const relationship = url.searchParams.get("relationship_id"), response = url.searchParams.get("response_revision_id");
          const base = { artifact_id: "sample-artifact", submission_id: "sample-submission", field_id: "sample-field", form_title: "Sample review", current: true, submitted_at: "2026-11-15T10:00:00Z", artifact_status: "AVAILABLE", file_kind: "PDF", media_type: "application/pdf", size_bytes: 10 };
          return new Response(JSON.stringify({ items: relationship || response ? [{ ...base, id: "file", file_name: "Sample report.pdf", ...(relationship ? { relationship_id: relationship } : { response_revision_id: response }) }] : [] }), { status: 200, headers: { "Content-Type": "application/json" } });
        }
        return original(input, init);
      };
      location.hash = "#/documents?relationship_id=sample-vendor&return_to=%23%2Fvendors%2Fsample-vendor%2Fassessments%2Fsample-assessment";
    });
    await page.getByText("Vendor documents", { exact: true }).waitFor();
    assert.equal(await page.getByRole("link", { name: "← Vendor" }).getAttribute("href"), "#/vendors/sample-vendor/assessments/sample-assessment");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `vendor-${width}.png`), fullPage: true });
    await page.evaluate(() => { location.hash = "#/documents?response_revision_id=sample-response"; });
    await page.getByText("Response documents", { exact: true }).waitFor();
    assert.equal(await page.getByRole("link", { name: "← Response" }).getAttribute("href"), "#/forms/responses/sample-response");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `response-${width}.png`), fullPage: true });
    results.push({ width, vendorScope: true, responseScope: true, overflow: false });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Scoped Documents verified at ${results.length} widths.`);
} finally { await browser.close(); }
