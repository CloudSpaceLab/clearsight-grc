import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { chromium } from "playwright";

const require = createRequire(import.meta.url);
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
await mkdir(outputDir, { recursive: true });

const captures = [
  { name: "191-loss-register-light-1440x900", state: "loss-register-s5", route: "#losses", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "192-loss-detail-light-1440x900", state: "loss-detail-s5", route: "#losses/loss-s5-1", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "193-loss-authoring-dark-mobile-390x844", state: "loss-authoring-s5-mobile", route: "#losses", theme: "dark", viewport: { width: 390, height: 844 }, touch: true, openAuthoring: true },
];

const browser = await chromium.launch({ headless: true });
try {
  for (const capture of captures) {
    const context = await browser.newContext({
      viewport: capture.viewport,
      colorScheme: capture.theme,
      hasTouch: capture.touch ?? false,
      reducedMotion: "reduce",
      locale: "en-NG",
      timezoneId: "Africa/Lagos",
    });
    await context.addInitScript((theme) => {
      localStorage.setItem("clearsight.theme", theme);
      localStorage.setItem("clearsight.density", "comfortable");
    }, capture.theme);

    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
    await page.goto(`${baseURL}/?fixture=loss-s5${capture.route}`, { waitUntil: "networkidle" });

    if (capture.openAuthoring) {
      await page.getByRole("heading", { name: "Losses" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: /Organization scope/ }).click();
      const scopeDialog = page.getByRole("dialog", { name: "Change organization scope" });
      await scopeDialog.getByRole("button", { name: /Risk Operations/ }).click();
      await page.getByRole("button", { name: /Organization scope/ }).filter({ hasText: "Risk Operations" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Record loss" }).click();
      const dialog = page.getByRole("dialog", { name: "Record loss" });
      await dialog.waitFor({ state: "visible" });
      await dialog.getByText("BANK / RISK / OPERATIONS", { exact: true }).waitFor({ state: "visible" });
      await dialog.getByRole("textbox", { name: /Loss code/ }).waitFor({ state: "visible" });
      await dialog.getByRole("button", { name: /Event type/ }).waitFor({ state: "visible" });
      await dialog.getByRole("textbox", { name: /Gross loss/ }).waitFor({ state: "visible" });
    } else if (capture.state === "loss-register-s5") {
      await page.getByRole("heading", { name: "Losses" }).waitFor({ state: "visible" });
      await page.getByRole("table", { name: "Operational loss register" }).waitFor({ state: "visible" });
      await page.getByText("Duplicate merchant settlement", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("cell", { name: "Recovery: Partly recovered" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Record loss" }).waitFor({ state: "visible" });
    } else {
      await page.getByRole("heading", { name: "Duplicate merchant settlement" }).waitFor({ state: "visible" });
      await page.getByRole("group", { name: "Current loss state" }).waitFor({ state: "visible" });
      await page.getByText("Nneka Okafor", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("BANK / RISK / OPERATIONS", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("OPS-SETTLE-07 · Settlement processing error", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("table", { name: "Loss recoveries" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "View intervention" }).waitFor({ state: "visible" });
      if (!page.url().includes("#losses/loss-s5-1")) throw new Error("Exact Loss route was not retained.");
    }

    if (errors.length) throw new Error(`${capture.name} emitted browser errors: ${errors.join("; ")}`);
    const metrics = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
      clientHeight: document.documentElement.clientHeight,
      scrollHeight: document.documentElement.scrollHeight,
    }));
    if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${capture.name} has horizontal overflow: ${metrics.scrollWidth}px in ${metrics.clientWidth}px`);

    await page.addScriptTag({ path: require.resolve("axe-core/axe.min.js") });
    const violations = await page.evaluate(async () => {
      const result = await window.axe.run(document, { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa"] } });
      return result.violations.map((violation) => ({ id: violation.id, targets: violation.nodes.map((node) => node.target) }));
    });
    if (violations.length) throw new Error(`${capture.name} accessibility violations: ${JSON.stringify(violations)}`);

    await page.screenshot({ path: path.join(outputDir, `${capture.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
    await appendRecord(capture, metrics);
    await context.close();
  }
} finally {
  await browser.close();
}

async function appendRecord(capture, metrics) {
  let manifest;
  try {
    manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  } catch {
    manifest = { generatedAt: new Date().toISOString(), baseURL, failure: null, captures: [] };
  }
  manifest.captures.push({
    name: capture.name,
    route: capture.route,
    fixture: "loss-s5",
    state: capture.state,
    viewport: capture.viewport,
    theme: capture.theme,
    density: "comfortable",
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}
