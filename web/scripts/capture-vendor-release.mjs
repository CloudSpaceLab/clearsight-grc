import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-09-09-vendor-release/onboarding");
await mkdir(root, { recursive: true });
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4187";
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 390]) for (const state of ["assurance-missing", "assurance-available", "spreadsheet", "spreadsheet-unsectioned", "create"]) {
    const page = await browser.newPage({ viewport: { width, height: 1000 }, colorScheme: theme, reducedMotion: "reduce" });
    page.setDefaultTimeout(10000);
    await page.addInitScript((theme) => localStorage.setItem("clearsight.theme", theme), theme);
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    const result = { theme, width, state, errors };
    try {
      await page.goto(`${base}/?tour=off&fixture=${state === "create" ? "" : `vendor-release-${state}`}#vendors`, { waitUntil: "networkidle" });
      if (state === "create") {
        await page.getByRole("button", { name: "Add vendor", exact: true }).click();
        await page.getByRole("button", { name: /Select criticality/ }).waitFor();
        await page.getByRole("button", { name: /Select privacy role/ }).waitFor();
      } else if (state.startsWith("spreadsheet")) {
        await page.getByRole("heading", { name: "Review proposed form fields" }).waitFor();
        result.rows = await page.getByRole("checkbox", { name: /^Include Sample requirement/ }).count();
        const expected = state === "spreadsheet" ? 47 : 3;
        if (result.rows !== expected) throw new Error(`Expected ${expected} rows; received ${result.rows}`);
        if (await page.getByRole("textbox", { name: /^Sample requirement/ }).count() !== expected) throw new Error("Proposal preview omitted source fields");
      } else {
        await page.getByText("Missing assurance requires review before approval.").waitFor();
        if (state === "assurance-missing") {
          await page.getByRole("textbox", { name: /Missing assurance/ }).waitFor();
          await page.getByRole("radio", { name: "Yes", exact: true }).check();
          await page.getByRole("group", { name: "Current independent assurance document *" }).waitFor();
          await page.getByRole("radio", { name: "No", exact: true }).check();
          await page.getByRole("textbox", { name: /Missing assurance/ }).waitFor();
          result.conditionalToggle = "passed";
        }
        else await page.getByRole("group", { name: "Current independent assurance document *" }).waitFor();
      }
      result.layout = await page.evaluate(() => ({ viewport: innerWidth, width: document.documentElement.scrollWidth, height: document.documentElement.scrollHeight }));
      if (result.layout.width > width + 1) throw new Error("Horizontal page overflow");
      if (state === "spreadsheet" && result.layout.height > 2000) throw new Error("Spreadsheet preview displaced the draft action");
      await page.screenshot({ path: path.join(root, `${state}-${theme}-${width}.png`), fullPage: state !== "spreadsheet" });
      if (state === "spreadsheet") {
        await page.getByRole("button", { name: "Create draft from selected fields" }).scrollIntoViewIfNeeded();
        await page.screenshot({ path: path.join(root, `${state}-actions-${theme}-${width}.png`) });
      }
    } catch (error) { result.failure = String(error); }
    results.push(result);
    await page.close();
  }
} finally { await browser.close(); await writeFile(path.join(root, "manifest.json"), JSON.stringify(results, null, 2)); }
if (results.some((result) => result.failure || result.errors.length)) process.exitCode = 1;
console.log(JSON.stringify(results.map(({ theme, width, state, failure, layout }) => ({ theme, width, state, failure, layout })), null, 2));
