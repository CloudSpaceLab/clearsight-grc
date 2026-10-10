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
  { name: "181-native-indicator-light-1440x900", state: "native-indicator-detail", fixture: "native-indicator", route: "", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "182-native-indicator-dark-mobile-390x844", state: "native-indicator-detail-mobile", fixture: "native-indicator", route: "", theme: "dark", viewport: { width: 390, height: 844 }, touch: true },
  { name: "183-indicator-insights-light-1440x900", state: "indicator-insights", fixture: "indicator-insights", route: "#insights?kind=KRI", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "184-indicator-insights-dark-mobile-390x844", state: "indicator-insights-mobile", fixture: "indicator-insights", route: "#insights?kind=KRI", theme: "dark", viewport: { width: 390, height: 844 }, touch: true },
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
    await page.goto(`${baseURL}/?fixture=${capture.fixture}${capture.route}`, { waitUntil: "networkidle" });
    if (capture.fixture === "native-indicator") {
      await page.getByRole("heading", { name: "Mobile transaction success rate" }).waitFor({ state: "visible" });
      await page.getByText("98.70%", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByText("Limit ≥ 99.50%", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByRole("table", { name: "Mobile transaction success rate observation history" }).waitFor({ state: "visible" });
    } else {
      await page.getByRole("heading", { name: "Insights" }).waitFor({ state: "visible" });
      await page.getByRole("table", { name: "Risk indicators" }).waitFor({ state: "visible" });
      await page.getByText("Mobile transaction success rate", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByText("Failed ATM transactions", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByText("98.70%", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByText("Limit ≥ 99.50%", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByText("-0.40 pp", { exact: true }).first().waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Review indicator for Mobile transaction success rate" }).click();
      await page.getByRole("heading", { name: "Mobile transaction success rate" }).waitFor({ state: "visible" });
      if (!page.url().includes("indicator=sample-mobile-success")) throw new Error("Exact Indicator target was not retained in the Insights route.");
      await page.getByRole("button", { name: "Close" }).click();
      await page.getByRole("table", { name: "Risk indicators" }).waitFor({ state: "visible" });
      if (!page.url().includes("#insights?kind=KRI")) throw new Error("Insights return route lost the selected lens.");
    }
    if (errors.length) throw new Error(`${capture.name} emitted browser errors: ${errors.join("; ")}`);

    const metrics = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
      clientHeight: document.documentElement.clientHeight,
      scrollHeight: document.documentElement.scrollHeight,
    }));
    if (metrics.scrollWidth > metrics.clientWidth + 1) {
      const overflowing = await page.evaluate(() => {
        const rightEdge = document.documentElement.clientWidth;
        return [...document.body.querySelectorAll("*")].map((element) => {
          const rect = element.getBoundingClientRect();
          const style = window.getComputedStyle(element);
          return {
            tag: element.tagName.toLowerCase(),
            cls: element.getAttribute("class")?.slice(0, 120),
            parent: element.parentElement?.getAttribute("class")?.slice(0, 100),
            left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width),
            minWidth: style.minWidth, overflowX: style.overflowX,
          };
        }).filter((element) => element.right > rightEdge + 1 && element.left < element.right)
          .sort((left, right) => right.right - left.right)
          .slice(0, 18);
      });
      throw new Error(`${capture.name} has horizontal overflow: ${metrics.scrollWidth}px in ${metrics.clientWidth}px. Offenders: ${JSON.stringify(overflowing)}`);
    }

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
    fixture: capture.fixture,
    state: capture.state,
    viewport: capture.viewport,
    theme: capture.theme,
    density: "comfortable",
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}
