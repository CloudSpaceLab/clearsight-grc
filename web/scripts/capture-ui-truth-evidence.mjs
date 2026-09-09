import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4179";
const output = path.resolve(process.env.UI_TRUTH_EVIDENCE_DIR ?? "../docs/evidence/2026-09-09-ui-audit-corrections/truth");
const states = ["program-missing", "program-stale", "program-current", "reviewer-missing", "reviewer-reassigned", "schedule-missing", "schedule-invalid", "setup", "requirement"];
const selectedStates = process.env.UI_TRUTH_STATES?.split(",");
const browser = await chromium.launch({ headless: true });
const captures = [];
await mkdir(output, { recursive: true });

try {
  for (const state of states.filter((state) => !selectedStates || selectedStates.includes(state))) for (const theme of ["light", "dark"]) for (const width of [1440, 390, 320, 720]) {
    const id = `${state}-${theme}-${width}`;
    const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: "reduce" });
    await context.addInitScript((theme) => localStorage.setItem("clearsight.theme", theme), theme);
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    const result = { id, state, theme, width, capturedAt: new Date().toISOString(), errors, reflowNote: width === 720 ? "720 CSS px approximates the layout viewport of 1440px at 200% browser zoom; not an actual browser zoom measurement." : undefined };
    captures.push(result);
    try {
      await page.goto(`${baseURL}/?tour=off&fixture=ui-truth-${state}`, { waitUntil: "networkidle" });
      await page.getByRole("heading", { level: 1 }).waitFor();
      if (state === "setup") {
        await page.getByLabel("Program name", { exact: true }).fill("Delivery record retention");
        await page.getByLabel("Code", { exact: true }).fill("DELIVERY");
        await page.getByLabel("Owning function", { exact: true }).fill("Logistics");
        await page.getByLabel("Scope", { exact: true }).fill("Carrier delivery records");
        await page.getByRole("button", { name: "Create Program", exact: true }).click();
        await page.getByRole("heading", { name: "Requirements", exact: true }).waitFor();
        for (const label of ["Who must act?", "What must they do?", "What does it apply to?"]) {
          const field = page.getByLabel(label, { exact: false });
          if (await field.inputValue() !== "" || !(await field.getAttribute("required") !== null)) throw new Error(`Required authoring field not empty: ${label}`);
        }
        if (!(await page.getByRole("button", { name: /Obligation strength/ }).innerText()).includes("Select the source's wording")) throw new Error("Obligation strength is preselected");
      }
      if (state === "requirement") await page.getByRole("button", { name: "Add requirement", exact: true }).click();
      if (state.startsWith("reviewer-")) {
        await page.getByText("View outcome result history (1)").click();
        const history = page.locator(".matter-outcome-card ol");
        const expected = state === "reviewer-reassigned" ? "Morgan Reed" : "Reviewer name unavailable";
        if (!(await history.innerText()).includes(expected) || (await history.innerText()).includes("Jordan Ellis")) throw new Error("Historical attribution changed with current reviewer");
      }
      if (state.startsWith("schedule-")) {
        await page.getByText("Outcome check details", { exact: true }).click();
        await page.getByText("Not scheduled", { exact: true }).waitFor();
      }
      if (state === "program-missing") await page.getByRole("heading", { name: "Unknown", exact: true }).waitFor();
      if (state === "program-stale") await page.getByRole("heading", { name: "Out of date", exact: true }).waitFor();
      await page.evaluate(() => document.fonts.ready);
      result.overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
      result.bodyText = (await page.locator("main").innerText()).slice(0, 12000);
      await page.screenshot({ path: path.join(output, `${id}.png`), fullPage: true });
      result.passed = !result.overflow && errors.length === 0;
    } catch (error) {
      result.failure = error instanceof Error ? error.message : String(error);
      result.passed = false;
    } finally {
      await context.close();
    }
    process.stdout.write(`${id}: ${result.passed ? "PASS" : "FAIL"}\n`);
  }
} finally {
  await browser.close();
  const previous = selectedStates ? JSON.parse(await readFile(path.join(output, "manifest.json"), "utf8").catch(() => '{"captures":[]}')).captures : [];
  const updated = new Map(previous.map((capture) => [capture.id, capture]));
  for (const capture of captures) updated.set(capture.id, capture);
  await writeFile(path.join(output, "manifest.json"), JSON.stringify({ generatedAt: new Date().toISOString(), baseURL, captures: [...updated.values()] }, null, 2));
}
if (captures.some((capture) => !capture.passed)) process.exitCode = 1;
