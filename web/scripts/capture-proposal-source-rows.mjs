import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const baseline = process.env.CAPTURE_BASELINE === "true";
const root = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-09-09-finding-followup-salvage", baseline ? "before" : "after");
await mkdir(root, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 1000 }, colorScheme: theme, reducedMotion: "reduce" });
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const result = { theme, width, errors, baseline };
    try {
      await page.goto(`${process.env.PAGE_URL ?? "http://127.0.0.1:5194"}/?tour=off&fixture=vendor-release-spreadsheet-source-rows`, { waitUntil: "networkidle" });
      const cards = page.locator(".form-proposal-change");
      await cards.nth(1).waitFor();
      result.excerpts = await cards.locator("blockquote").allTextContents();
      if (result.excerpts.length !== 2) throw new Error("Two retained row excerpts must be visible");
      if (!baseline && (!result.excerpts[0].includes("signed access register") || !result.excerpts[1].includes("recovery exercise report"))) throw new Error("Source excerpts did not match their row anchors");
      if (baseline && result.excerpts[0] !== result.excerpts[1]) throw new Error("Baseline no longer reproduces repeated first-row excerpt");
      result.layout = await page.evaluate(() => ({ viewport: innerWidth, width: document.documentElement.scrollWidth }));
      if (result.layout.width > width + 1) throw new Error("Horizontal page overflow");
      await page.screenshot({ path: path.join(root, `source-rows-${theme}-${width}.png`), fullPage: true });
      if (!baseline) {
        const second = cards.nth(1).getByRole("checkbox");
        await second.uncheck();
        if (await page.getByRole("textbox", { name: /^Sample requirement 2/ }).count() !== 0) throw new Error("Deselected field remained in preview");
        await second.check();
        if (await page.getByRole("textbox", { name: /^Sample requirement 2/ }).count() !== 1) throw new Error("Reselected field missing from preview");
        const create = page.getByRole("button", { name: "Create draft from selected fields" });
        await create.scrollIntoViewIfNeeded();
        if (!await create.isEnabled()) throw new Error("Draft action inaccessible after row selection");
        await page.addScriptTag({ path: require.resolve("axe-core/axe.min.js") });
        result.accessibilityViolations = await page.evaluate(async () => (await window.axe.run(document.querySelector(".form-proposal-review"))).violations.map(({ id, impact }) => ({ id, impact })));
        if (result.accessibilityViolations.length) throw new Error("Proposal accessibility violations");
      }
    } catch (error) { result.failure = String(error); }
    results.push(result);
    await page.close();
  }
} finally {
  await browser.close();
  await writeFile(path.join(root, "manifest.json"), JSON.stringify(results, null, 2));
}
console.log(JSON.stringify(results, null, 2));
if (results.some((result) => result.failure || result.errors.length)) process.exitCode = 1;
