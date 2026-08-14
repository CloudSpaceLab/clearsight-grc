import { chromium } from "playwright";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const browser = await chromium.launch({ headless: true });

await mkdir(outputDir, { recursive: true });

try {
  for (const capture of [
    { name: "30-operating-actions-light-1440x900", state: "operating-actions-desktop", viewport: { width: 1440, height: 900 }, colorScheme: "light" },
    { name: "31-operating-actions-dark-mobile-390x844", state: "operating-actions-mobile", viewport: { width: 390, height: 844 }, colorScheme: "dark", touch: true },
  ]) {
    const context = await browser.newContext({ viewport: capture.viewport, colorScheme: capture.colorScheme, hasTouch: capture.touch === true, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
    await context.addInitScript((theme) => localStorage.setItem("clearsight.theme", theme), capture.colorScheme);
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });

    try {
      await page.goto(`${baseURL}/?fixture=operating-mutations`, { waitUntil: "networkidle" });
      await page.getByRole("heading", { name: "Operating actions", exact: true }).waitFor();
      await page.getByRole("heading", { name: "Update action", exact: true }).waitFor();
      await page.getByRole("heading", { name: "Change operating status", exact: true }).waitFor();

      const actionTargets = await page.getByLabel("Next state").locator("option").allTextContents();
      if (actionTargets.join("|") !== "Implemented|Blocked|Cancelled") throw new Error(`${capture.name} exposed unexpected Matter Action targets: ${actionTargets.join(", ")}`);
      const programTargets = await page.getByLabel("Requested status").locator("option").allTextContents();
      if (programTargets.join("|") !== "Paused|Retired") throw new Error(`${capture.name} exposed unexpected Program targets: ${programTargets.join(", ")}`);

      const metrics = await layoutMetrics(page);
      if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${capture.name} has ${metrics.scrollWidth - metrics.clientWidth}px horizontal overflow`);
      if (errors.length) throw new Error(`${capture.name} emitted browser errors:\n${errors.join("\n")}`);

      await page.screenshot({ path: path.join(outputDir, `${capture.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
      await appendRecord(capture, metrics);
    } finally {
      await context.close();
    }
  }
} finally {
  await browser.close();
}

async function layoutMetrics(page) {
  return page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
    clientHeight: document.documentElement.clientHeight,
    scrollHeight: document.documentElement.scrollHeight,
    scrollY: window.scrollY,
    theme: document.documentElement.dataset.theme ?? "unknown",
    density: document.documentElement.dataset.density ?? "unknown",
  }));
}

async function appendRecord(capture, metrics) {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  manifest.captures.push({
    name: capture.name,
    route: "?fixture=operating-mutations",
    fixture: "operating-mutations",
    state: capture.state,
    viewport: capture.viewport,
    theme: capture.colorScheme,
    density: "comfortable",
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}