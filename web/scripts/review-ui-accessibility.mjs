import { chromium } from "playwright";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const axeSource = await readFile(path.resolve("node_modules/axe-core/axe.min.js"), "utf8");
const browser = await chromium.launch({ headless: true });
const scenarios = [
  { name: "today-desktop", path: "/?tour=off#today", heading: "Today", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "today-mobile", path: "/?tour=off#today", heading: "Today", viewport: { width: 390, height: 844 }, theme: "dark", touch: true },
  { name: "today-empty", path: "/?tour=off&fixture=today-empty#today", heading: "Today", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "today-unavailable", path: "/?tour=off&fixture=today-unavailable#today", heading: "Today", viewport: { width: 1440, height: 900 }, theme: "dark" },
  { name: "program", path: "/?tour=off#programs/program-ndpa", heading: "Programs", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "evidence", path: "/?tour=off#work/evidence", heading: "Work", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "imports", path: "/?tour=off#imports", heading: "Imports", viewport: { width: 1440, height: 900 }, theme: "dark" },
  { name: "configure", path: "/?tour=off#configure", heading: "Routing and approvals", viewport: { width: 1440, height: 900 }, theme: "light" },
];

await mkdir(outputDir, { recursive: true });
const results = [];
let failure = null;

try {
  for (const scenario of scenarios) {
    const context = await browser.newContext({
      viewport: scenario.viewport,
      colorScheme: scenario.theme,
      hasTouch: scenario.touch === true,
      reducedMotion: "reduce",
      locale: "en-NG",
      timezoneId: "Africa/Lagos",
    });
    await context.addInitScript((theme) => {
      localStorage.setItem("clearsight.theme", theme);
      localStorage.setItem("clearsight.density", "comfortable");
    }, scenario.theme);
    const page = await context.newPage();
    const browserErrors = [];
    page.on("pageerror", (error) => browserErrors.push(error.message));
    page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });
    try {
      await page.goto(`${baseURL}${scenario.path}`, { waitUntil: "networkidle" });
      await page.getByRole("heading", { name: scenario.heading, exact: true }).first().waitFor({ state: "visible" });
      await page.evaluate(() => document.fonts?.ready);
      await page.addScriptTag({ content: axeSource });
      const axe = await page.evaluate(async () => {
        const result = await globalThis.axe.run(document, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"] },
        });
        return result.violations.map((violation) => ({
          id: violation.id,
          impact: violation.impact,
          help: violation.help,
          nodes: violation.nodes.map((node) => ({
            target: node.target,
            html: node.html,
            failureSummary: node.failureSummary,
          })),
        }));
      });
      const layout = await page.evaluate(() => ({
        clientWidth: document.documentElement.clientWidth,
        scrollWidth: document.documentElement.scrollWidth,
      }));
      const undersizedPrimaryControls = scenario.touch
        ? await page.locator(".primary-button:visible").evaluateAll((elements) => elements.flatMap((element) => {
            const box = element.getBoundingClientRect();
            return box.width < 40 || box.height < 40
              ? [{ text: element.textContent?.trim().slice(0, 80) ?? "", width: box.width, height: box.height }]
              : [];
          }))
        : [];
      const blockingViolations = axe.filter((violation) => violation.impact === "critical" || violation.impact === "serious");
      const errors = [
        ...browserErrors.map((message) => `browser: ${message}`),
        ...(layout.scrollWidth > layout.clientWidth + 1 ? [`horizontal overflow: ${layout.scrollWidth}px in ${layout.clientWidth}px`] : []),
        ...blockingViolations.map((violation) => `axe ${violation.impact}: ${violation.id} (${violation.nodes.length} nodes)`),
        ...undersizedPrimaryControls.map((control) => `touch target: ${control.text} is ${Math.round(control.width)}x${Math.round(control.height)}`),
      ];
      results.push({ ...scenario, violations: axe, layout, undersizedPrimaryControls, errors, status: errors.length ? "FAIL" : "PASS" });
      if (errors.length) throw new Error(`${scenario.name}: ${errors.join("; ")}`);
    } finally {
      await context.close();
    }
  }
} catch (error) {
  failure = error instanceof Error ? error.message : String(error);
  throw error;
} finally {
  await writeFile(path.join(outputDir, "accessibility.json"), JSON.stringify({ generatedAt: new Date().toISOString(), failure, scenarios: results }, null, 2));
  await browser.close();
}
