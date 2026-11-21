import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-20-v210-program-requirements");
await mkdir(output, { recursive: true });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${process.env.PAGE_URL ?? "http://127.0.0.1:4188"}/#/programs/sample-program`);
    await page.getByRole("button", { name: "Set applicability for Review access" }).click();
    assert.equal(await page.getByLabel("Why is this not applicable?").count(), 0);
    await page.getByLabel("Does this apply?").selectOption("NOT_APPLICABLE");
    await page.getByLabel("Why is this not applicable?").waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `applicability-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Cancel" }).click();
    assert.equal(await page.getByRole("button", { name: "Set applicability for Review access" }).evaluate(el => el === document.activeElement), true);
    await page.getByRole("button", { name: "Edit Review access" }).click();
    assert.equal(await page.getByLabel("Requirement title").inputValue(), "Review access");
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `edit-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Cancel" }).click();
    await page.getByRole("button", { name: "Link A1 to Review access" }).click();
    await page.getByLabel("Reason for link").fill("Same control");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `link-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, conditionalReason: true, focusRestored: true, linkProposal: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`Program requirements evidence passed at ${results.length} widths.`);
} finally { await browser.close(); }
