import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { inspectRenderedContrast } from "./rendered-contrast.mjs";
import { reportsDesignAccessibilityScenarios } from "./reports-design-evidence-scenarios.mjs";
import { notificationsAccessibilityScenarios } from "./notifications-evidence-scenarios.mjs";
import { prepareNotifications } from "./notifications-evidence.mjs";
import { vendorRelationshipAccessibilityScenarios } from "./vendor-relationship-evidence-scenarios.mjs";
import { prepareVendorRelationship } from "./vendor-relationship-evidence.mjs";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const axeSource = await readFile(path.resolve("node_modules/axe-core/axe.min.js"), "utf8");
const browser = await chromium.launch({ headless: true });
const baseScenarios = [
  { name: "today-desktop", path: "/?tour=off#today", heading: "Oversight", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "today-mobile", path: "/?tour=off#today", heading: "Oversight", viewport: { width: 390, height: 844 }, theme: "dark", touch: true },
  { name: "today-empty", path: "/?tour=off&fixture=today-empty#today", heading: "Oversight", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "today-unavailable", path: "/?tour=off&fixture=today-unavailable#today", heading: "Oversight", viewport: { width: 1440, height: 900 }, theme: "dark" },
  { name: "program", path: "/?tour=off#programs/program-ndpa", heading: "Programs", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "evidence", path: "/?tour=off#work/evidence", heading: "Work", viewport: { width: 1440, height: 900 }, theme: "light" },
  { name: "imports", path: "/?tour=off#imports", heading: "Imports", viewport: { width: 1440, height: 900 }, theme: "dark" },
  { name: "configure", path: "/?tour=off#configure", heading: "Configuration", viewport: { width: 1440, height: 900 }, theme: "light" },
];
const scenarios = [...baseScenarios.filter((scenario) => scenario.name !== "today-mobile").flatMap((scenario) =>
  ["light", "dark"].flatMap((theme) => [1440, 390].map((width) => ({
    ...scenario,
    name: `${scenario.name}-${theme}-${width}`,
    theme,
    viewport: { width, height: width === 390 ? 844 : 900 },
    touch: width === 390,
  }))),
), ...reportsDesignAccessibilityScenarios, ...notificationsAccessibilityScenarios, ...vendorRelationshipAccessibilityScenarios].filter((scenario) =>
  process.env.UI_EVIDENCE_SCOPE === "report-design" ? scenario.reportView : process.env.UI_EVIDENCE_SCOPE === "notifications" ? scenario.notificationView
    : process.env.UI_EVIDENCE_SCOPE === "vendor-relationship" ? scenario.vendorRelationshipView : true);

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
      const workspace = scenario.path.includes("#programs/") && !scenario.path.includes("#programs/library")
        ? page.getByRole("region", { name: "Program record" })
        : page.getByRole("heading", { name: scenario.heading, exact: true }).first();
      await workspace.waitFor({ state: "visible" });
      if (scenario.notificationView) await prepareNotifications(page, { ...scenario, fixture: "notifications-loaded" });
      if (scenario.vendorRelationshipView) await prepareVendorRelationship(page, scenario);
      if (scenario.reportView === "builder") {
        await page.locator(".reports-workspace__actions").getByRole("button", { name: "New report", exact: true }).click();
        await page.locator(".report-builder").getByRole("button", { name: "Save setup", exact: true }).waitFor({ state: "visible" });
        await page.locator(".report-builder__setup .cs-select-field__trigger:not(:disabled)").first().waitFor({ state: "visible" });
      } else if (scenario.reportView) {
        await page.getByRole("tab", { name: "Explore", exact: true }).click();
        await page.locator('.reports-explore__chart[data-state="live"]').waitFor({ state: "visible" });
        if (scenario.reportView === "explore-members") {
          await page.locator(".reports-explore").getByRole("button", { name: "Review underlying records", exact: true }).click();
          await page.getByRole("dialog", { name: "Risks outside appetite records", exact: true })
            .getByRole("table", { name: "Risks outside appetite records", exact: true }).waitFor({ state: "visible" });
        }
      }
      await page.evaluate(() => document.fonts?.ready);
      await page.addScriptTag({ content: axeSource });
      const axe = await page.evaluate(async (selector) => {
        const result = await globalThis.axe.run(selector || document, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"] },
        });
        const summarize = (items) => items.map((violation) => ({
          id: violation.id,
          impact: violation.impact,
          help: violation.help,
          nodes: violation.nodes.map((node) => ({
            target: node.target,
            html: node.html,
            failureSummary: node.failureSummary,
            checks: [...node.any, ...node.all, ...node.none].map(({ id, message, data }) => ({ id, message, data })),
          })),
        }));
        return { violations: summarize(result.violations), incomplete: summarize(result.incomplete), passes: result.passes.map(({ id, nodes }) => ({ id, count: nodes.length })) };
      }, scenario.scopeSelector);
      let contrast = await page.evaluate(inspectRenderedContrast);
      if (scenario.scopeSelector) {
        contrast = await page.evaluate(({ readings, scope }) => readings.filter((reading) =>
          [...document.querySelectorAll(reading.selector)].some((node) => node.closest(scope))), { readings: contrast, scope: scenario.scopeSelector });
      }
      const layout = await page.evaluate(() => ({
        clientWidth: document.documentElement.clientWidth,
        scrollWidth: document.documentElement.scrollWidth,
      }));
      const undersizedPrimaryControls = scenario.touch
        ? await page.locator(scenario.scopeSelector ?? "body").locator(scenario.vendorRelationshipView
          ? 'button:visible, input:not([type="hidden"]):visible, select:visible, textarea:visible, summary:visible, [role="tab"]:visible'
          : scenario.reportView || scenario.notificationView
          ? 'button:visible, input:not([type="hidden"]):visible, [role="tab"]:visible' : ".primary-button:visible").evaluateAll((elements, minimum) => [...new Set(elements.map((element) => element.closest(".cs-checkbox-field") ?? element))].filter((element) => !element.closest('[aria-hidden="true"], [inert]')).flatMap((element) => {
            const box = element.getBoundingClientRect();
            return box.width < minimum || box.height < minimum
              ? [{ text: element.textContent?.trim().slice(0, 80) ?? "", width: box.width, height: box.height }]
              : [];
          }), scenario.reportView || scenario.notificationView || scenario.vendorRelationshipView ? 44 : 40)
        : [];
      const blockingViolations = axe.violations.filter((violation) => scenario.reportView || scenario.notificationView || scenario.vendorRelationshipView || violation.impact === "critical" || violation.impact === "serious");
      const errors = [
        ...browserErrors.map((message) => `browser: ${message}`),
        ...(layout.scrollWidth > layout.clientWidth + 1 ? [`horizontal overflow: ${layout.scrollWidth}px in ${layout.clientWidth}px`] : []),
        ...blockingViolations.map((violation) => `axe ${violation.impact}: ${violation.id} (${violation.nodes.length} nodes)`),
        ...contrast.filter((reading) => reading.outcome === "FAIL" && (reading.kind === "placeholder" || reading.essential)).map((reading) => `contrast ${reading.kind}: ${reading.selector} (${reading.ratio.toFixed(2)} < ${reading.minimum})`),
        ...undersizedPrimaryControls.map((control) => `touch target: ${control.text} is ${Math.round(control.width)}x${Math.round(control.height)}`),
      ];
      results.push({ ...scenario, ...axe, contrast, layout, touchTargetMinimum: scenario.reportView || scenario.notificationView || scenario.vendorRelationshipView ? 44 : 40,
        undersizedPrimaryControls, errors, status: errors.length ? "FAIL" : "PASS" });
      if (errors.length) failure = "One or more accessibility scenarios failed";
    } catch (error) {
      failure = "One or more accessibility scenarios failed";
      results.push({ ...scenario, status: "ERROR", errors: [...browserErrors, error instanceof Error ? error.message : String(error)] });
    } finally {
      await context.close();
    }
  }
} catch (error) {
  failure = error instanceof Error ? error.message : String(error);
  throw error;
} finally {
  await writeFile(path.join(outputDir, "accessibility.json"), JSON.stringify({ generatedAt: new Date().toISOString(), failure,
    scope: "Automated checks for the listed fixture, theme and viewport states. PASS means no blocking finding in these checks; incomplete and unsupported readings remain unresolved.",
    contrastLimits: "Canvas colors are rounded to 8-bit sRGB. Supplemental readings do not model overlapping siblings, pseudo-element paint, blur or position-dependent gradients. Only shared field boundaries are automatically classified as essential; other low border readings require human interpretation.",
    scenarios: results }, null, 2));
  await browser.close();
}
if (failure) { console.error(failure); process.exitCode = 1; }
