import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const browser = await chromium.launch({ headless: true });

await mkdir(outputDir, { recursive: true });

for (const capture of [
  { name: "30-operating-actions-light-1440x900", viewport: { width: 1440, height: 900 }, colorScheme: "light" },
  { name: "31-operating-actions-dark-mobile-390x844", viewport: { width: 390, height: 844 }, colorScheme: "dark", touch: true },
]) {
  const context = await browser.newContext({ viewport: capture.viewport, colorScheme: capture.colorScheme, hasTouch: capture.touch === true, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript((theme) => localStorage.setItem("clearsight.theme", theme), capture.colorScheme);
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });

  await page.goto(`${baseURL}/?fixture=operating-mutations`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Operating actions", exact: true }).waitFor();
  await page.getByRole("heading", { name: "Update action", exact: true }).waitFor();
  await page.getByRole("heading", { name: "Change operating status", exact: true }).waitFor();

  const actionTargets = await page.getByLabel("Next state").locator("option").allTextContents();
  if (actionTargets.join("|") !== "Implemented|Blocked|Cancelled") throw new Error(`${capture.name} exposed unexpected Matter Action targets: ${actionTargets.join(", ")}`);
  const programTargets = await page.getByLabel("Requested status").locator("option").allTextContents();
  if (programTargets.join("|") !== "Paused|Retired") throw new Error(`${capture.name} exposed unexpected Program targets: ${programTargets.join(", ")}`);

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  if (overflow > 1) throw new Error(`${capture.name} has ${overflow}px horizontal overflow`);
  if (errors.length) throw new Error(`${capture.name} emitted browser errors:\n${errors.join("\n")}`);

  await page.screenshot({ path: path.join(outputDir, `${capture.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  await context.close();
}

await browser.close();
