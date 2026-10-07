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
  { name: "185-rcsa-register-light-1440x900", state: "rcsa-register", route: "#rcsa", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "186-rcsa-detail-light-1440x900", state: "rcsa-detail", route: "#rcsa/cycle-rcsa-1", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "187-rcsa-detail-dark-mobile-390x844", state: "rcsa-detail-mobile", route: "#rcsa/cycle-rcsa-1", theme: "dark", viewport: { width: 390, height: 844 }, touch: true },
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
    await page.goto(`${baseURL}/?fixture=rcsa-cycles${capture.route}`, { waitUntil: "networkidle" });

    if (capture.state === "rcsa-register") {
      await page.getByRole("heading", { name: "RCSA cycles" }).waitFor({ state: "visible" });
      await page.getByRole("table", { name: "RCSA cycles" }).waitFor({ state: "visible" });
      await page.getByText("Q3 Technology RCSA", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("3 Risks", { exact: true }).waitFor({ state: "visible" });
      const q3Row = page.getByRole("row", { name: /Q3 Technology RCSA/ });
      await q3Row.getByRole("cell", { name: /Stage: Completed/ }).waitFor({ state: "visible" });
    } else {
      await page.getByRole("heading", { name: "Q3 Technology RCSA" }).waitFor({ state: "visible" });
      await page.getByText("1 Jul 2026 – 30 Sep 2026", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Open challenge work" }).waitFor({ state: "visible" });
      await page.getByRole("heading", { name: "Cycle progress" }).waitFor({ state: "visible" });
      await page.getByText("Deficiency confirmed", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("Remediation in progress", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("table", { name: "RCSA frozen Risks" }).waitFor({ state: "visible" });
      if (!page.url().includes("#rcsa/cycle-rcsa-1")) throw new Error("Exact RCSA cycle route was not retained.");
      if (capture.state === "rcsa-detail") {
        await page.getByRole("button", { name: "Open challenge work" }).click();
        await page.waitForURL(/#work\/matters\/matter-gaid-change/);
        await page.goBack({ waitUntil: "networkidle" });
        await page.waitForURL(/#rcsa\/cycle-rcsa-1/);
        await page.getByRole("heading", { name: "Q3 Technology RCSA" }).waitFor({ state: "visible" });
        await page.getByText("Remediation in progress", { exact: true }).waitFor({ state: "visible" });
      }
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

  const returnContext = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    colorScheme: "light",
    reducedMotion: "reduce",
    locale: "en-NG",
    timezoneId: "Africa/Lagos",
  });
  try {
    const page = await returnContext.newPage();
    await page.goto(`${baseURL}/?fixture=rcsa-cycles#rcsa/cycle-rcsa-2`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Q4 Operations RCSA" }).waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Open first-line assessment" }).click();
    await page.waitForURL(/#work\/evidence\/request-rcsa-2/);
    await page.goBack({ waitUntil: "networkidle" });
    await page.waitForURL(/#rcsa\/cycle-rcsa-2/);
    await page.getByRole("heading", { name: "Q4 Operations RCSA" }).waitFor({ state: "visible" });
    await page.getByText("First-line collection", { exact: true }).waitFor({ state: "visible" });
  } finally {
    await returnContext.close();
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
    fixture: "rcsa-cycles",
    state: capture.state,
    viewport: capture.viewport,
    theme: capture.theme,
    density: "comfortable",
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}
