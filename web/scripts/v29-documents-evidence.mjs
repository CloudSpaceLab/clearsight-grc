import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence/v29-documents");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.CHROMIUM_EXECUTABLE ? { executablePath: process.env.CHROMIUM_EXECUTABLE } : {}) });
const results = [];

try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 1280, 1024, 768, 390]) {
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, reducedMotion: "reduce", colorScheme: theme });
    const page = await context.newPage();
    await page.goto(`${base}/?fixture=forms-documents#documents`, { waitUntil: "domcontentloaded" });
    const region = page.getByRole("region", { name: "Submitted documents" });
    await region.getByRole("row", { name: /Sample security certification\.pdf/ }).waitFor();
    const heading = page.getByRole("heading", { name: "Documents", level: 1 });
    assert.equal(await heading.count(), 1);
    if (width === 390) assert.equal(await region.getByRole("button", { name: "All documents Document view" }).count(), 1);
    else assert.equal(await region.getByRole("tab", { name: "All documents" }).count(), 1);
    const geometry = await page.evaluate(() => {
      const title = document.querySelector(".document-browser h1")?.getBoundingClientRect();
      const search = document.querySelector('.document-browser input[type="search"]')?.getBoundingClientRect();
      const first = document.querySelector(".document-browser tbody tr")?.getBoundingClientRect();
      return { overflow: document.documentElement.scrollWidth > innerWidth, title: title && { y: title.y, height: title.height }, search: search && { y: search.y, height: search.height }, first: first && { y: first.y, height: first.height } };
    });
    assert.equal(geometry.overflow, false, `${width}: horizontal overflow`);
    assert.ok(geometry.title && geometry.search && geometry.first && geometry.first.y < (width === 390 ? 844 : 900), `${width}: first record is outside the initial viewport`);
    const search = region.getByRole("searchbox", { name: "Search file names" });
    await search.focus();
    assert.equal(await search.evaluate((element) => document.activeElement === element), true);
    await search.fill("security");
    await region.getByRole("row", { name: /Sample security certification\.pdf/ }).waitFor();
    await page.waitForFunction(() => document.querySelectorAll(".document-browser tbody tr").length === 1);
    await search.fill("");
    await page.waitForFunction(() => document.querySelectorAll(".document-browser tbody tr").length > 1);
    await page.screenshot({ path: path.join(output, `loaded-${theme}-${width}.png`), fullPage: true });
    const preview = region.getByRole("button", { name: /Preview for Sample security certification\.pdf/ });
    await preview.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog", { name: "Preview Sample security certification.pdf" });
    await dialog.waitFor();
    await page.keyboard.press("Escape");
    await dialog.waitFor({ state: "hidden" });
    assert.equal(await preview.evaluate((element) => document.activeElement === element), true, `${width}: preview focus was not restored`);
    if (width === 390) {
      await region.getByRole("button", { name: "All documents Document view" }).click();
      await page.getByRole("option", { name: "Awaiting assessment review" }).click();
    } else await region.getByRole("tab", { name: "Awaiting assessment review" }).click();
    await region.getByText("Only recorded vendor assessment reviews are shown.").waitFor();
    await page.screenshot({ path: path.join(output, `awaiting-review-${theme}-${width}.png`), fullPage: true });
    results.push({ theme, width, geometry, heading: await heading.innerText(), searchWorks: true, previewKeyboardReturn: true });
    await context.close();
  }
  for (const [fixture, text] of [["home-shell", "No matching documents"], ["forms-documents-error", "Documents could not be loaded."]]) {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
    await page.goto(`${base}/?fixture=${fixture}#documents`, { waitUntil: "domcontentloaded" });
    await page.getByText(text, { exact: true }).first().waitFor();
    await page.screenshot({ path: path.join(output, `${fixture}-390.png`), fullPage: true });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ results }, null, 2));
  console.log(`Documents evidence passed in ${results.length} width/theme states; empty and unavailable states rendered.`);
} finally {
  await browser.close();
}
