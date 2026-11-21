import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const axe = require("axe-core");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const mode = process.env.EVIDENCE_MODE ?? "after";
const out = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-form-approvals");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    if (mode === "reference") {
      await page.goto(`${base}/approval-inbox.html`);
      await page.locator("#app").waitFor();
      await page.screenshot({ path: path.join(out, `reference-inbox-${width}.png`), fullPage: true });
      await page.close();
      continue;
    }
    await page.goto(`${base}/?fixture=form-approvals#/work`);
    await page.getByRole("heading", { name: "Work & approvals" }).waitFor();
    await page.getByText("Review the sample access assessment").waitFor();
    await page.screenshot({ path: path.join(out, `${mode}-work-${width}.png`), fullPage: true });
    if (mode === "after") {
      await page.getByRole("button", { name: "Approvals & reviews" }).click();
      await page.getByRole("link", { name: "Review Sample access assessment" }).waitFor();
      await page.getByRole("link", { name: "Review Sample vendor review" }).waitFor();
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await page.addScriptTag({ content: axe.source });
      const scan = await page.evaluate(async () => window.axe.run(document, { rules: { "color-contrast": { enabled: false } } }));
      assert.deepEqual(scan.violations.map((violation) => violation.id), []);
      await page.screenshot({ path: path.join(out, `after-forms-${width}.png`), fullPage: true });
      await page.getByRole("link", { name: "Review Sample access assessment" }).click();
      await page.getByRole("heading", { name: "Sample access assessment" }).waitFor();
      await page.getByLabel("Review decision").waitFor();
      assert.equal(page.url().endsWith("#/forms/responses/sample-program-response"), true);
      await page.goto(`${base}/?fixture=form-approvals#/work`);
      await page.getByRole("button", { name: "Approvals & reviews" }).click();
      await page.getByRole("link", { name: "Review Sample vendor review" }).click();
      await page.getByRole("button", { name: "Approve form" }).waitFor();
      await page.getByRole("button", { name: "Reject form" }).waitFor();
      assert.equal(page.url().endsWith("#/forms/sample-form?approval_version=2"), true);
      await page.evaluate(() => {
        const original = window.fetch.bind(window); let approved = false;
        window.fetch = async (input, init) => {
          const path = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, window.location.origin).pathname;
          if (path === "/api/v1/forms/templates/sample-form/transition" && init?.method === "POST") {
            approved = true;
            return new Response(JSON.stringify({ id: "sample-form", status: "ACTIVE", version: 3 }), { status: 200, headers: { "Content-Type": "application/json" } });
          }
          const response = await original(input, init);
          if (approved && path === "/api/v1/forms/templates/sample-form" && (!init?.method || init.method === "GET")) {
            const value = await response.json();
            return new Response(JSON.stringify({ ...value, template: { ...value.template, status: "ACTIVE", version: 3 }, operations: [] }), { status: 200, headers: { "Content-Type": "application/json" } });
          }
          return response;
        };
      });
      await page.getByRole("button", { name: "Approve form" }).click();
      await page.getByText("Form approved.").waitFor();
      assert.equal(await page.getByRole("button", { name: "Approve form" }).count(), 0);
      await page.screenshot({ path: path.join(out, `confirmed-form-${width}.png`), fullPage: true });
      await page.goto(`${base}/?fixture=form-approvals#/forms/sample-form?approval_version=1`);
      await page.reload();
      await page.getByText("Form version changed. Review the latest version before deciding.").waitFor();
      assert.equal(await page.getByRole("button", { name: "Approve form" }).count(), 0);
      assert.equal(await page.getByRole("button", { name: "Reject form" }).count(), 0);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await page.screenshot({ path: path.join(out, `stale-form-${width}.png`), fullPage: true });
      await page.getByRole("link", { name: "Review latest form" }).click();
      await page.getByRole("button", { name: "Approve form" }).waitFor();
      await page.goto(`${base}/?fixture=program-reviews-unavailable#/work`);
      await page.getByRole("button", { name: "Approvals & reviews" }).click();
      await page.getByText("Program reviews unavailable.").waitFor();
      assert.equal(await page.getByRole("button", { name: "Try again" }).count(), 0);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await page.screenshot({ path: path.join(out, `program-unavailable-${width}.png`), fullPage: true });
      results.push({ width, overflow: false, axeViolations: 0, staleVersionBlocksDecision: true });
    }
    await page.close();
  }
  if (mode === "after") await writeFile(path.join(out, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_UI_FIXTURE_NOT_BACKEND_AUTHORIZATION", colorContrastExcluded: true, results }, null, 2));
} finally { await browser.close(); }
