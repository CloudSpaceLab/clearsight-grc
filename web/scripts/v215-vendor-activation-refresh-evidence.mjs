import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:43112";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-vendor-activation-refresh");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(`${base}/vendor-activation-refresh.html`);
    await page.getByRole("button", { name: "Activate relationship" }).click();
    await page.getByLabel("Activation rationale").fill("Required approvals completed.");
    await page.screenshot({ path: path.join(output, `before-${width}.png`), fullPage: true });
    await page.getByRole("button", { name: "Confirm activation" }).click();
    await page.getByText("Relationship activated.", { exact: true }).waitFor();
    await page.getByRole("alert").getByText("Relationship details could not be refreshed. Reload the relationship.").waitFor();
    const alertStyle = await page.getByRole("alert").locator(".v-status--high").evaluate((element) => { const style = getComputedStyle(element); return [style.color, style.backgroundColor, style.display]; });
    assert.deepEqual(alertStyle, ["rgb(160, 53, 53)", "rgb(255, 240, 238)", "inline-block"]);
    assert.equal(await page.getByText(/Activation status could not be confirmed/).count(), 0);
    assert.equal(await page.getByRole("button", { name: "Confirm activation" }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `after-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, activationConfirmed: true, refreshFailureShown: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "SYNTHETIC_NOT_BACKEND_AUTHORIZATION", results }, null, 2));
  console.log(`Vendor activation refresh failure verified at ${results.length} widths.`);
} finally { await browser.close(); }
