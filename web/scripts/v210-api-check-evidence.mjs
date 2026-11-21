import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence/v210-api-check");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 1280, 1024, 768, 390]) {
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, colorScheme: theme, reducedMotion: "reduce" });
    await context.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const page = await context.newPage();
    await page.goto(`${base}/?tour=off&fixture=matter-action-reassignment#programs/program-ndpa/monitoring`, { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "Add data check" }).click();
    await page.getByRole("button", { name: "Connected data", exact: true }).click();
    const builder = page.locator(".data-source-builder");
    await builder.getByRole("heading", { name: "New endpoint check" }).waitFor();
    await builder.getByLabel("Sample result type").selectOption("number");
    await builder.getByLabel("Sample response").fill('{"result":72,"message":"Sample"}');
    await builder.getByRole("button", { name: "Test sample" }).focus();
    await page.keyboard.press("Enter");
    await builder.getByText("Sample valid. Endpoint not contacted.").waitFor();
    await builder.screenshot({ path: path.join(output, `${theme}-${width}-sample.png`) });
    await builder.getByLabel("Source name").fill("Sample service check");
    await builder.getByLabel("Code").fill("SAMPLE-CHECK");
    await builder.getByLabel("Status endpoint").fill("https://example.test/check");
    await builder.getByLabel("Maximum age (minutes)").fill("1440");
    await builder.getByRole("button", { name: "Test endpoint" }).click();
    try { await builder.getByText("Endpoint reached. Source approval is unavailable. The check condition cannot be saved or run yet.").waitFor({ timeout: 10000 }); }
    catch (error) { console.error(`${theme} ${width}: ${await builder.innerText()}`); throw error; }
    assert.equal(await builder.getByLabel("Observed field").count(), 0);
    assert.equal(await builder.getByLabel("Condition").count(), 0);
    assert.equal(await builder.getByLabel("Expected value").count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `${theme} ${width}: horizontal overflow`);
    await builder.screenshot({ path: path.join(output, `${theme}-${width}-draft.png`) });
    await builder.getByRole("button", { name: "Save draft" }).click();
    await builder.getByText("Source draft saved. The check condition has not been saved. Source approval is unavailable.").waitFor();
    assert.equal(await page.getByText("Connected-data check saved as a draft.").count(), 0);
    await builder.screenshot({ path: path.join(output, `${theme}-${width}-saved-source.png`) });
    results.push({ theme, width, sample: true, draft: true, noFalseCheck: true, keyboard: true });
    await context.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`API check evidence passed in ${results.length} width/theme states.`);
} finally { await browser.close(); }
