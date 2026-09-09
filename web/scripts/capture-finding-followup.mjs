import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { findingFollowUpFixture } from "../src/findingFollowUpFixture.ts";
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const root = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-09-09-finding-followup-defects");
await mkdir(root, { recursive: true });
const results = [];
const browser = await chromium.launch({ headless: true });
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 1000 }, colorScheme: theme, reducedMotion: "reduce" });
    page.setDefaultTimeout(10000);
    const result = { theme, width, errors: [] };
    page.on("pageerror", (error) => result.errors.push(error.message));
    await page.addInitScript((value) => { localStorage.setItem("clearsight.theme", value); window.findingTestNativeFetch = window.fetch.bind(window); }, theme);
    let prepareCalls = 0;
    await page.route("**/api/v1/document-imports/sample-register/form-template-proposals", async (route) => {
      const body = route.request().postDataJSON();
      if (body.finding_assessment_id !== "assessment-b" || body.expected_document_version !== 1) throw new Error("Wrong selected assessment or source version");
      prepareCalls++;
      if (prepareCalls === 1) return route.fulfill({ status: 503, json: { error: { code: "form_proposals_unavailable", message: "Sample generation outage. Try again." } } });
      await route.fulfill({ status: 202, json: findingFollowUpFixture(body.finding_assessment_id) });
    });
    await page.route("**/api/v1/forms/proposals/sample-assessment-b/accept", async (route) => {
      const body = route.request().postDataJSON();
      if (!body.assessment_confirmed || body.change_ids.length !== 5 || body.expected_version !== 2) throw new Error("Partial or unconfirmed assessment sent");
      result.acceptedFields = body.change_ids.length;
      await route.fulfill({ status: 200, json: { ...findingFollowUpFixture("assessment-b"), status: "ACCEPTED", accepted_change_ids: body.change_ids, result_template_id: "sample-draft", result_template_version: 1, version: 3 } });
    });
    try {
      await page.goto(`${process.env.PAGE_URL ?? "http://127.0.0.1:5196"}/?tour=off&fixture=finding-followup`, { waitUntil: "networkidle" });
      // This dedicated interaction test supplies its own API responses through
      // Playwright, instead of the broader evidence application's HTTP fixture.
      await page.evaluate(() => { window.fetch = window.findingTestNativeFetch; });
      await page.getByText("Prepare finding follow-up", { exact: true }).click();
      await page.getByRole("button", { name: /Source assessment/ }).click();
      await page.getByRole("option", { name: /Northstar/ }).click();
      await page.screenshot({ path: path.join(root, `choice-${theme}-${width}.png`), fullPage: true });
      const prepare = page.getByRole("button", { name: "Prepare follow-up questions", exact: true });
      await prepare.click();
      await page.getByRole("alert").waitFor();
      await prepare.click();
      const create = page.getByRole("button", { name: "Create follow-up draft", exact: true });
      await create.waitFor();
      if (await create.isEnabled()) throw new Error("Draft enabled without confirmation");
      const excerpts = await page.locator(".form-proposal-source blockquote").allTextContents();
      if (excerpts.length !== 5 || excerpts.some((text) => !text.includes("Northstar") || text.includes("Harbor"))) throw new Error("Wrong assessment source in preview");
      const confirmation = page.getByRole("checkbox", { name: "I have checked the vendor, service and historical assessment details" });
      await page.getByText("I have checked the vendor, service and historical assessment details", { exact: true }).click();
      if (!await confirmation.isChecked()) throw new Error("Confirmation label did not select checkbox");
      await confirmation.focus();
      await confirmation.press("Space");
      if (await create.isEnabled()) throw new Error("Keyboard uncheck did not disable draft action");
      await confirmation.press("Space");
      result.overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
      if (result.overflow) throw new Error("Horizontal overflow");
      await page.addScriptTag({ path: require.resolve("axe-core/axe.min.js") });
      result.violations = await page.evaluate(async () => (await window.axe.run(document.querySelector("main"))).violations.map(({ id, impact }) => ({ id, impact })));
      if (result.violations.length) throw new Error("Accessibility violations");
      await page.screenshot({ path: path.join(root, `review-${theme}-${width}.png`), fullPage: true });
      await create.click();
      await page.getByRole("link", { name: "Open draft template" }).waitFor();
      result.prepareCalls = prepareCalls;
    } catch (error) { result.failure = String(error); result.prepareCalls = prepareCalls; result.pageText = await page.locator("main").innerText().catch(() => "Unavailable"); }
    console.log(JSON.stringify(result));
    results.push(result);
    await page.close();
  }
} finally {
  await browser.close();
  await writeFile(path.join(root, "manifest.json"), JSON.stringify(results, null, 2));
}
console.log(JSON.stringify(results, null, 2));
if (results.some((result) => result.failure || result.errors.length)) process.exitCode = 1;
