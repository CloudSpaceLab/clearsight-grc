import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence/v210-vendor-intake");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 1280, 1024, 768, 390]) {
    if (process.env.EVIDENCE_WIDTH && width !== Number(process.env.EVIDENCE_WIDTH)) continue;
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, colorScheme: theme, reducedMotion: "reduce" });
    await context.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const page = await context.newPage();
    await page.goto(`${base}/?tour=off&fixture=home-shell#vendors`, { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "Add vendor", exact: true }).click();
    await page.getByRole("heading", { name: "Vendor details" }).waitFor();
    const form = page.locator("form.vendor-form");
    await form.getByRole("button", { name: "Continue" }).scrollIntoViewIfNeeded();
    await page.waitForTimeout(300);
    await form.getByRole("button", { name: "Continue" }).click();
    await page.getByText("Enter the vendor's legal name.").waitFor();
    await page.getByLabel("Legal name", { exact: false }).fill("Sample Vendor Limited");
    await page.screenshot({ path: path.join(output, `${theme}-${width}-vendor.png`) });
    await form.getByRole("button", { name: "Continue" }).scrollIntoViewIfNeeded();
    await page.waitForTimeout(300);
    await form.getByRole("button", { name: "Continue" }).click();
    await page.getByRole("heading", { name: "Service details" }).waitFor();
    await page.getByLabel("Service supplied", { exact: false }).fill("Payment processing");
    await page.getByRole("button", { name: /Select criticality/ }).click();
    await page.getByRole("option", { name: "Important", exact: true }).click();
    await page.getByRole("button", { name: /Select privacy role/ }).click();
    await page.getByRole("option", { name: "Processor", exact: true }).click();
    await page.screenshot({ path: path.join(output, `${theme}-${width}-service.png`) });
    await form.getByRole("button", { name: "Continue" }).click();
    await page.getByRole("heading", { name: "Review vendor and service" }).waitFor();
    assert.equal(await page.getByText("Sample Vendor Limited", { exact: true }).count(), 1);
    assert.equal(await page.getByText("Payment processing", { exact: true }).count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `${theme} ${width}: horizontal overflow`);
    await page.screenshot({ path: path.join(output, `${theme}-${width}-review.png`) });
    await form.getByRole("button", { name: "Back" }).focus();
    await page.keyboard.press("Enter");
    assert.equal(await page.getByLabel("Service supplied", { exact: false }).inputValue(), "Payment processing");
    results.push({ theme, width, validation: true, review: true, keyboardBack: true });
    await context.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`Vendor intake evidence passed in ${results.length} width/theme states.`);
} finally { await browser.close(); }
