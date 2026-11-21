import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-assignment-authority");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    const base = process.env.PAGE_URL ?? "http://127.0.0.1:43112";
    await page.goto(`${base}/form-assignment-authority.html`);
    await page.getByText("Choose a program before searching for a respondent.").waitFor();
    assert.equal(await page.getByRole("button", { name: "Search people" }).isDisabled(), true);
    await page.screenshot({ path: path.join(output, `choose-${width}.png`), fullPage: true });
    await page.locator(".v-assignment-form select").selectOption("sample-program");
    await page.getByRole("button", { name: "Create assignment" }).waitFor();
    await page.getByLabel("Find respondent").fill("Ada");
    await page.getByRole("button", { name: "Search people" }).click();
    await page.getByRole("button", { name: /Ada Okafor/ }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `allowed-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: /Ada Okafor/ }).click();
    await page.getByLabel("Purpose").fill("Confirm access reviews");
    await page.getByLabel("Response deadline").fill("2099-01-01T10:00");
    await page.getByRole("button", { name: "Create assignment" }).click();
    await page.getByRole("button", { name: "Creating…" }).waitFor();
    assert.equal(await page.locator(".v-assignment-form select").isDisabled(), true);
    assert.equal(await page.getByLabel("Find respondent").isDisabled(), true);
    assert.equal(await page.getByLabel("Find a program").isDisabled(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `creating-${width}.png`), fullPage: true });
    await page.goto(`${base}/form-assignment-authority.html?denied=1`);
    await page.locator(".v-assignment-form select").selectOption("sample-program");
    await page.getByRole("alert").getByText("Assignment unavailable. Check your role or contact an administrator.").waitFor();
    assert.equal(await page.getByRole("button", { name: "Search people" }).isDisabled(), true);
    assert.equal(await page.getByRole("button", { name: "Create assignment" }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `denied-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, noAuthoritySearchDisabled: true, deniedSearchDisabled: true, authorizedSearchAvailable: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION_OR_RELEASE", results }, null, 2));
  console.log(`Assignment authority states verified at ${results.length} widths.`);
} finally { await browser.close(); }
