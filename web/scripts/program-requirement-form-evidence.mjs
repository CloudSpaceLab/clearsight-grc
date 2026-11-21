import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence-requirements");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) for (const theme of ["light", "dark"]) {
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, colorScheme: theme, hasTouch: width === 390, reducedMotion: "reduce" });
    await context.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const page = await context.newPage();
    await page.goto(`${base}/#programs/program-ndpa/requirements-controls`, { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "Record applicability" }).click();
    const choice = page.getByRole("combobox", { name: "Does this apply?", exact: true });
    if (!await choice.evaluate((element) => element === document.activeElement)) throw new Error("Applicability did not receive focus");
    if (await page.getByLabel("Why is this not applicable?").count()) throw new Error("Unexpected reason before selection");
    const save = page.getByRole("button", { name: "Save applicability decision" });
    if (await save.isEnabled()) throw new Error("An unselected decision is enabled");
    await choice.selectOption("APPLICABLE");
    if (await page.getByLabel("Why is this not applicable?").count() || !await save.isEnabled()) throw new Error("Applicable decision is not ready");
    await choice.selectOption("NOT_APPLICABLE");
    const reason = page.getByLabel("Why is this not applicable?");
    if (!await reason.isVisible() || await save.isEnabled()) throw new Error("Exception reason is not required");
    await reason.fill("The service does not use privileged accounts.");
    if (!await save.isEnabled()) throw new Error("The completed decision is unavailable");
    const geometry = await page.evaluate(() => {
      const form = document.querySelector(".program-applicability-choice").closest("form");
      const choice = form.querySelector(".program-applicability-choice select").getBoundingClientRect();
      const reason = form.querySelector(".program-applicability-choice textarea").getBoundingClientRect();
      const buttons = [...form.querySelectorAll(".program-form-actions button")].map((node) => node.getBoundingClientRect());
      const overlaps = buttons.some((a, i) => buttons.some((b, j) => i < j && a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom));
      return { choice: { x: choice.x, y: choice.y, right: choice.right, bottom: choice.bottom }, reason: { x: reason.x, y: reason.y }, overlaps,
        minButtonHeight: Math.min(...buttons.map((box) => box.height)), overflow: document.documentElement.scrollWidth - innerWidth };
    });
    if (geometry.overlaps || geometry.minButtonHeight < 44 || geometry.overflow > 1 || (width <= 560 ? geometry.reason.y < geometry.choice.bottom : geometry.reason.x < geometry.choice.right - 1)) throw new Error(`Invalid layout ${width}/${theme}: ${JSON.stringify(geometry)}`);
    await reason.scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(output, `applicability-${theme}-${width}.png`) });
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    if (!await page.getByRole("button", { name: "Record applicability" }).evaluate((element) => element === document.activeElement)) throw new Error("Applicability focus did not return");
    if (width <= 650) await page.getByRole("button", { name: "Workspace controls" }).click();
    await page.getByRole("button", { name: /Viewing as Chief Risk Officer/ }).click();
    await page.getByRole("button", { name: "Switch to Data Protection Officer" }).click();
    const edit = page.getByRole("button", { name: "Edit requirement" }).first();
    await edit.waitFor();
    await edit.click();
    const title = page.getByLabel("Requirement title");
    if (!await title.evaluate((element) => element === document.activeElement) || !(await title.inputValue()).trim()) throw new Error("Edit did not focus the existing title");
    if (!await page.getByText("Saving creates a new requirement version.").isVisible()) throw new Error("Version consequence is missing");
    if (await page.locator(".program-requirements-panel .program-panel-actions").count()) throw new Error("Competing actions remain visible while editing");
    await page.getByLabel("Requirement text").scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(output, `edit-${theme}-${width}.png`) });
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    if (!await page.getByRole("button", { name: "Edit requirement" }).first().evaluate((element) => element === document.activeElement)) throw new Error("Edit focus did not return");
    results.push({ width, theme, ...geometry, editFocus: true });
    await context.close();
  }
  await writeFile(path.join(output, "geometry.json"), JSON.stringify({ results }, null, 2));
  console.log(`Checked ${results.length} sample Program states.`);
} finally { await browser.close(); }
