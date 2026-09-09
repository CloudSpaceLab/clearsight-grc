import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const root = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-09-09-form-preview");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4187";
await mkdir(root, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ["light", "dark"]) for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: "reduce" });
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript((theme) => localStorage.setItem("clearsight.theme", theme), theme);
    try {
      await page.goto(`${base}/?tour=off&fixture=forms-builder-mobile#forms`, { waitUntil: "networkidle" });
      await page.getByRole("button", { name: "Open Compliance scoring review" }).click();
      await page.getByRole("button", { name: "Edit draft", exact: true }).click();
      await page.getByRole("button", { name: "Preview", exact: true }).click();
      const preview = page.getByRole("dialog", { name: "Form preview", exact: true });
      await preview.getByRole("radio", { name: "Yes", exact: true }).waitFor();
      if (await preview.getByRole("group", { name: "Question layout" }).count() !== 1) throw new Error("Expected one layout control");
      if (await preview.getByRole("button", { name: /^Preview (Classic|Wizard)$/ }).count()) throw new Error("Duplicate preview controls");
      await preview.getByText("Step 1 of 1", { exact: true }).waitFor();
      for (const state of ["immediate", "all-questions"]) {
        if (state === "all-questions") {
          await preview.getByRole("button", { name: "Show all questions" }).click();
          if (await preview.getByText("Step 1 of 1", { exact: true }).count()) throw new Error("Classic layout retained wizard progress");
        }
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
        if (overflow) throw new Error("Horizontal page overflow");
        await page.screenshot({ path: path.join(root, `${state}-${theme}-${width}.png`) });
        const close = await preview.getByRole("button", { name: "Close form preview" }).boundingBox();
        if (!close || close.x < 0 || close.x + close.width > width || close.y < 0 || close.y + close.height > 900) throw new Error(`Close action outside viewport: ${JSON.stringify(close)}`);
        results.push({ theme, width, state, errors: [...errors], overflow });
      }
    } catch (error) { results.push({ theme, width, errors, failure: String(error) }); }
    await page.close();
  }
} finally {
  await browser.close();
  await writeFile(path.join(root, "manifest.json"), JSON.stringify(results, null, 2));
}
console.log(JSON.stringify(results, null, 2));
if (results.some((result) => result.failure || result.errors.length)) process.exitCode = 1;
