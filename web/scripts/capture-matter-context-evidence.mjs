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
  { name: "188-matter-loss-context-light-1440x900", fixture: "matter-loss-context", state: "matter-loss-context", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "189-matter-indicator-context-light-1440x900", fixture: "matter-indicator-context", state: "matter-indicator-context", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "190-matter-indicator-context-dark-mobile-390x844", fixture: "matter-indicator-context", state: "matter-indicator-context-mobile", theme: "dark", viewport: { width: 390, height: 844 }, touch: true },
  { name: "194-issue-linked-form-handoff-light-1440x900", fixture: "matter-workspace-follow-through", state: "issue-linked-form-handoff", kind: "linked-form", tab: "Evidence and requests", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "195-issue-paged-form-activity-dark-mobile-390x844", fixture: "matter-workspace-follow-through", state: "issue-paged-form-activity-mobile", kind: "paged-activity", tab: "Evidence and requests", theme: "dark", viewport: { width: 390, height: 844 }, touch: true },
  { name: "196-issue-off-page-sent-form-light-1440x900", fixture: "matter-workspace-follow-through", state: "issue-off-page-sent-form", kind: "off-page-sent-form", route: "#forms?section=sent-forms&distribution=distribution-matter-28", theme: "light", viewport: { width: 1440, height: 900 } },
  { name: "197-issue-board-brief-unavailable-light-1440x900", fixture: "matter-workspace-follow-through", state: "issue-board-brief-unavailable", kind: "board-brief-unavailable", theme: "light", viewport: { width: 1440, height: 900 } },
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
    const route = capture.route ?? "#work/matters/matter-gaid-change";
    await page.goto(`${baseURL}/?fixture=${capture.fixture}${route}`, { waitUntil: "networkidle" });

    if (route.startsWith("#work/matters/")) {
      if (capture.viewport.width <= 760) {
        await page.getByLabel("Issue section", { exact: true }).waitFor({ state: "visible" });
      } else {
        const issueWork = page.getByLabel("Issue work");
        await issueWork.getByRole("tab", { name: "Overview" }).waitFor({ state: "visible" });
        await issueWork.getByRole("tab", { name: "Work" }).waitFor({ state: "visible" });
        await issueWork.getByRole("tab", { name: "Evidence and requests" }).waitFor({ state: "visible" });
        await issueWork.getByRole("tab", { name: "Decisions" }).waitFor({ state: "visible" });
      }
      if (capture.tab) await openMatterTab(page, capture.tab);
    }

    if (capture.fixture === "matter-loss-context") {
      await page.getByRole("region", { name: "Operational loss context" }).waitFor({ state: "visible" });
      await page.getByText("LOSS-2041 · Payments outage loss", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Open loss record" }).waitFor({ state: "visible" });
      await page.getByText("Partly recovered", { exact: true }).waitFor({ state: "visible" });
    } else if (capture.fixture === "matter-indicator-context") {
      await page.getByRole("region", { name: "Indicator observation context" }).waitFor({ state: "visible" });
      await page.getByText("Mobile transaction success rate", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("99.10%", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("Limit ≥ 99.50%", { exact: true }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Open indicator" }).waitFor({ state: "visible" });
      if (await page.getByText("98.70%", { exact: true }).count()) throw new Error("Matter context substituted the current indicator value for the exact source observation.");
    } else if (capture.kind === "linked-form") {
      const row = page.locator(".subject-form-activity__group li").filter({ hasText: "Annual return evidence draft" }).first();
      await row.waitFor({ state: "visible" });
      await row.getByText("Revision 1", { exact: true }).waitFor({ state: "visible" });
      await row.getByText("Draft", { exact: true }).waitFor({ state: "visible" });
      if (await row.getByRole("link", { name: "Open form" }).getAttribute("href") !== "#forms/form-matter-linked-01") throw new Error("Linked form handoff is not exact.");
      await page.getByRole("button", { name: "Load more linked forms" }).waitFor({ state: "visible" });
      await row.scrollIntoViewIfNeeded();
    } else if (capture.kind === "paged-activity") {
      const requests = page.getByRole("button", { name: "Load more form requests" });
      const responses = page.getByRole("button", { name: "Load more submitted responses" });
      await requests.waitFor({ state: "visible" });
      await responses.waitFor({ state: "visible" });
      await requests.click();
      await page.getByText("Issue evidence request 7", { exact: true }).waitFor({ state: "visible" });
      await responses.click();
      await page.getByText("Submitted issue evidence 7", { exact: true }).waitFor({ state: "visible" });
      await page.getByText("Submitted issue evidence 7", { exact: true }).scrollIntoViewIfNeeded();
    } else if (capture.kind === "off-page-sent-form") {
      await page.getByRole("heading", { name: "Sent forms" }).waitFor({ state: "visible" });
      await page.getByText("Off-page executive evidence request", { exact: true }).waitFor({ state: "visible" });
      await page.getByLabel("Off-page executive evidence request details").waitFor({ state: "visible" });
    } else if (capture.kind === "board-brief-unavailable") {
      await page.getByText("No active board brief setup is available for this issue.", { exact: false }).waitFor({ state: "visible" });
      const reports = page.getByRole("link", { name: "Open Reports" });
      await reports.waitFor({ state: "visible" });
      if (await reports.getAttribute("href") !== "#reports") throw new Error("Board brief recovery does not open Reports.");
      if (await page.getByRole("button", { name: "Generate board brief" }).count()) throw new Error("Unavailable board brief exposed generation.");
    }

    if (!page.url().includes(route)) throw new Error("Exact Matter route was not retained.");
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
    await appendRecord(capture, route, metrics);
    await context.close();
  }
} finally {
  await browser.close();
}

async function openMatterTab(page, label) {
  const tab = page.getByRole("tab", { name: label, exact: true });
  if (await tab.isVisible().catch(() => false)) {
    await tab.click();
    return;
  }
  const trigger = page.locator(".matter-workspace-main .cs-tabs__compact .cs-select-field__trigger").first();
  await trigger.waitFor({ state: "visible" });
  await trigger.click();
  await page.getByRole("option", { name: label, exact: true }).click();
}

async function appendRecord(capture, route, metrics) {
  let manifest;
  try {
    manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  } catch {
    manifest = { generatedAt: new Date().toISOString(), baseURL, failure: null, captures: [] };
  }
  manifest.captures.push({
    name: capture.name,
    route,
    fixture: capture.fixture,
    state: capture.state,
    viewport: capture.viewport,
    theme: capture.theme,
    density: "comfortable",
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}
