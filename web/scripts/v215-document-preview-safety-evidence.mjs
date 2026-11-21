import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-document-preview-safety");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(process.env.PAGE_URL ?? "http://127.0.0.1:43115/", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === "/api/v1/forms/documents") {
          const base = { artifact_id: "sample-artifact", submission_id: "sample-submission", field_id: "sample-field", form_title: "Sample review", current: true, submitted_at: "2026-11-15T10:00:00Z", artifact_status: "AVAILABLE" };
          return new Response(JSON.stringify({ items: [{ ...base, id: "image", file_name: "Sample access review.png", file_kind: "IMAGE", media_type: "image/png" }, { ...base, id: "office", file_name: "Sample register.xlsx", file_kind: "SPREADSHEET", media_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" }] }), { status: 200, headers: { "Content-Type": "application/json" } });
        }
        return original(input, init);
      };
      location.hash = "#/documents";
    });
    await page.getByRole("button", { name: "Open Sample access review.png" }).click();
    await page.getByRole("img", { name: "Preview: Sample access review.png" }).waitFor();
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `image-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Open Sample register.xlsx" }).click();
    await page.getByText("Preview unavailable for this file type. Download the document.").waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `unsupported-${width}.png`), fullPage: true });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname.endsWith("/sample-submission/sample-field/sample-artifact/content")) return new Response("bad image", { headers: { "Content-Type": "image/png" } });
        return original(input, init);
      };
    });
    await page.getByRole("button", { name: "Open Sample access review.png" }).click();
    await page.getByText("Preview unavailable. Try again.").waitFor();
    assert.equal(await page.getByRole("button", { name: "Preview document" }).isEnabled(), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `decode-error-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, imagePreview: true, unsupportedFallback: true, decodeError: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Document preview safety verified at ${results.length} widths.`);
} finally { await browser.close(); }
