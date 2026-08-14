import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const browser = await chromium.launch({ headless: true });
const scenarios = [];
let failure = null;

await mkdir(outputDir, { recursive: true });

try {
  await auditDeepLinkedWorkspaces();
  await auditProgramStatusControl();
  await auditEvidenceCaptureSubmission();
  await auditEvidenceCopy();
  await auditResponsiveZoom();
  await auditMobileTouchTargets();
} catch (error) {
  failure = error instanceof Error ? error.message : String(error);
  throw error;
} finally {
  await writeFile(path.join(outputDir, "defects.json"), JSON.stringify({ generatedAt: new Date().toISOString(), failure, scenarios }, null, 2));
  await browser.close();
}

async function auditDeepLinkedWorkspaces() {
  for (const test of [
    { name: "program-deep-link", route: "#programs/program-ndpa", heading: "Programs", target: ".program-card.targeted .program-card-main" },
    { name: "matter-deep-link", route: "#work/matters/matter-gaid-change", heading: "Work", target: ".matter-card.targeted .matter-card-main" },
    { name: "evidence-deep-link", route: "#work/evidence", heading: "Work", target: ".evidence-workbench .section-header" },
  ]) {
    const { context, page, browserErrors } = await openPage({ viewport: { width: 1440, height: 900 }, route: test.route, heading: test.heading });
    try {
      const target = page.locator(test.target).first();
      await target.waitFor({ state: "visible" });
      await target.scrollIntoViewIfNeeded();
      await assertNotCoveredBy(page, target, ".context-bar", `${test.name} target`);
      await assertNoHorizontalOverflow(page, test.name);
      assertNoBrowserErrors(browserErrors, test.name);
      pass(test.name, "Deep-linked work remains visible below sticky workspace chrome.");
    } finally {
      await context.close();
    }
  }

  const { context, page, browserErrors } = await openPage({ viewport: { width: 390, height: 844 }, route: "#programs/program-ndpa", heading: "Programs", touch: true });
  try {
    const action = page.getByRole("button", { name: /Request (pause|activation|retirement)/ }).first();
    await action.waitFor({ state: "visible" });
    await action.scrollIntoViewIfNeeded();
    await assertNotCoveredBy(page, action, ".mobile-nav", "mobile Program action");
    await assertNoHorizontalOverflow(page, "mobile-program-action");
    assertNoBrowserErrors(browserErrors, "mobile-program-action");
    pass("mobile-program-action", "The fixed mobile navigation does not obscure the current Program action.");
  } finally {
    await context.close();
  }
}

async function auditProgramStatusControl() {
  const { context, page, browserErrors } = await openPage({ viewport: { width: 1440, height: 900 }, query: "fixture=operating-mutations", heading: "Operating actions" });
  try {
    const rationale = page.getByLabel("Rationale");
    const submit = page.getByRole("button", { name: "Request pause" });
    await rationale.waitFor({ state: "visible" });
    if (await submit.isEnabled()) throw new Error("Program status action is enabled before a rationale is entered");
    await rationale.fill("   ");
    if (await submit.isEnabled()) throw new Error("Program status action accepts a whitespace-only rationale");
    await rationale.fill("Pause while ownership is corrected.");
    if (!(await submit.isEnabled())) throw new Error("Program status action remains disabled after a valid rationale");

    await installDelayedFetch(page, "/api/v1/programs/program-ndpa/transition", 250);
    await submit.click();
    if (!(await submit.isDisabled())) throw new Error("Program status action is not disabled while the command is in flight");
    await submit.evaluate((element) => element.click());
    await page.getByText("Program status updated.", { exact: true }).waitFor({ state: "visible" });
    const calls = await delayedFetchCalls(page);
    if (calls !== 1) throw new Error(`Program status action submitted ${calls} commands instead of one`);
    assertNoBrowserErrors(browserErrors, "program-status-control");
    pass("program-status-control", "Rationale validation and in-flight double-submit prevention passed.");
  } finally {
    await context.close();
  }
}

async function auditEvidenceCaptureSubmission() {
  const { context, page, browserErrors } = await openPage({ viewport: { width: 1440, height: 900 }, route: "#today", heading: "Today" });
  try {
    await page.getByRole("button", { name: "Respond to evidence request" }).click();
    await page.getByRole("textbox", { name: /Processor register owner/ }).fill("Privacy Operations");
    await page.getByLabel(/DPCO review date/).fill("2027-03-01");
    await page.getByRole("button", { name: "Review and submit" }).click();
    const submit = page.getByRole("button", { name: "Submit response" });
    await installDelayedFetch(page, "/submissions", 250);
    await submit.click();
    if (!(await submit.isDisabled())) throw new Error("Evidence submission is not disabled while the request is in flight");
    await submit.evaluate((element) => element.click());
    await page.getByRole("heading", { name: "Response submitted" }).waitFor({ state: "visible" });
    const calls = await delayedFetchCalls(page);
    if (calls !== 1) throw new Error(`Evidence response submitted ${calls} times instead of once`);
    assertNoBrowserErrors(browserErrors, "evidence-submit");
    pass("evidence-submit", "Evidence review submits once and exposes a durable receipt.");
  } finally {
    await context.close();
  }
}

async function auditEvidenceCopy() {
  const { context, page, browserErrors } = await openPage({ viewport: { width: 1440, height: 900 }, route: "#work/evidence", heading: "Work" });
  try {
    await page.getByText("1 open request", { exact: true }).waitFor({ state: "visible" });
    await page.getByText("1 source issue", { exact: true }).waitFor({ state: "visible" });
    if (await page.getByText("1 open requests", { exact: true }).count()) throw new Error("Evidence summary uses a plural noun for a singular request");
    if (await page.getByText("1 source issues", { exact: true }).count()) throw new Error("Evidence summary uses a plural noun for a singular source issue");
    const dueText = await page.locator(".request-row summary time, .request-row summary div:nth-child(2) span").allTextContents();
    if (dueText.some((value) => /\b\d{1,2}:\d{2}:\d{2}\b/.test(value))) throw new Error("Evidence due dates expose unnecessary seconds");
    assertNoBrowserErrors(browserErrors, "evidence-copy");
    pass("evidence-copy", "Counts and operational timestamps are concise and grammatically correct.");
  } finally {
    await context.close();
  }
}

async function auditResponsiveZoom() {
  const context = await browser.newContext({ viewport: { width: 720, height: 450 }, deviceScaleFactor: 2, colorScheme: "light", hasTouch: true, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await applyPreferences(context, "light", "comfortable");
  const page = await context.newPage();
  const browserErrors = collectBrowserErrors(page);
  try {
    await page.goto(`${baseURL}/?tour=off#today`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Today", exact: true }).waitFor({ state: "visible" });
    const action = page.locator(".intervention-next .primary-button").first();
    await action.waitFor({ state: "visible" });
    await action.scrollIntoViewIfNeeded();
    await assertNotCoveredBy(page, action, ".mobile-nav", "200% reflow action");
    await assertNoHorizontalOverflow(page, "200-percent-reflow");
    assertNoBrowserErrors(browserErrors, "200-percent-reflow");
    pass("200-percent-reflow", "A real 200% CSS-pixel viewport reflows without covered actions or horizontal overflow.");
  } finally {
    await context.close();
  }
}

async function auditMobileTouchTargets() {
  const { context, page, browserErrors } = await openPage({ viewport: { width: 390, height: 844 }, query: "capture_invite=field-agent-demo", heading: "Open your request", touch: true });
  try {
    await page.getByRole("textbox", { name: "Email or phone number" }).fill("field.agent@example.com");
    await page.getByRole("button", { name: "Open request" }).click();
    await page.getByRole("heading", { name: "Verify ATM location after your visit" }).waitFor({ state: "visible" });
    const undersized = await page.evaluate(() => {
      const selectors = ["button", "a[href]", "summary", "input[type=radio]", "input[type=checkbox]"];
      return [...document.querySelectorAll(selectors.join(","))]
        .filter((element) => element instanceof HTMLElement && !element.hidden && getComputedStyle(element).visibility !== "hidden" && element.getClientRects().length > 0)
        .map((element) => {
          const rect = element.getBoundingClientRect();
          const label = element.getAttribute("aria-label") ?? element.textContent?.trim().replace(/\s+/g, " ").slice(0, 80) ?? element.tagName;
          return { label, width: Math.round(rect.width), height: Math.round(rect.height) };
        })
        .filter((target) => target.width < 24 || target.height < 24);
    });
    if (undersized.length) throw new Error(`Mobile controls below 24 CSS px: ${undersized.map((target) => `${target.label} (${target.width}×${target.height})`).join(", ")}`);
    assertNoBrowserErrors(browserErrors, "mobile-touch-targets");
    pass("mobile-touch-targets", "All exposed field-visit controls meet the rendered minimum target size.");
  } finally {
    await context.close();
  }
}

async function openPage({ viewport, route = "", query = "tour=off", heading, touch = false }) {
  const context = await browser.newContext({ viewport, colorScheme: "light", hasTouch: touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await applyPreferences(context, "light", "comfortable");
  const page = await context.newPage();
  const browserErrors = collectBrowserErrors(page);
  await page.goto(`${baseURL}/?${query}${route}`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: heading, exact: true }).first().waitFor({ state: "visible" });
  await page.evaluate(() => document.fonts?.ready);
  return { context, page, browserErrors };
}

async function applyPreferences(context, theme, density) {
  await context.addInitScript(({ theme, density }) => {
    localStorage.setItem("clearsight.theme", theme);
    localStorage.setItem("clearsight.density", density);
  }, { theme, density });
}

function collectBrowserErrors(page) {
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  return errors;
}

function assertNoBrowserErrors(errors, name) {
  if (errors.length) throw new Error(`${name} emitted browser errors: ${errors.join(" | ")}`);
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px in ${metrics.clientWidth}px`);
}

async function assertNotCoveredBy(page, target, chromeSelector, label) {
  const result = await page.evaluate(({ targetSelector, chromeSelector }) => {
    const targetElement = document.querySelector(targetSelector);
    const chrome = document.querySelector(chromeSelector);
    if (!(targetElement instanceof HTMLElement) || !(chrome instanceof HTMLElement) || chrome.getClientRects().length === 0) return { covered: false };
    const targetRect = targetElement.getBoundingClientRect();
    const chromeRect = chrome.getBoundingClientRect();
    const covered = targetRect.left < chromeRect.right && targetRect.right > chromeRect.left && targetRect.top < chromeRect.bottom && targetRect.bottom > chromeRect.top;
    return { covered, targetRect: { top: targetRect.top, right: targetRect.right, bottom: targetRect.bottom, left: targetRect.left }, chromeRect: { top: chromeRect.top, right: chromeRect.right, bottom: chromeRect.bottom, left: chromeRect.left } };
  }, { targetSelector: await selectorFor(target), chromeSelector });
  if (result.covered) throw new Error(`${label} is covered by ${chromeSelector}`);
}

async function selectorFor(locator) {
  const token = `ui-defect-${Math.random().toString(36).slice(2)}`;
  await locator.evaluate((element, token) => element.setAttribute("data-ui-defect-target", token), token);
  return `[data-ui-defect-target="${token}"]`;
}

async function installDelayedFetch(page, pathFragment, delayMS) {
  await page.evaluate(({ pathFragment, delayMS }) => {
    const original = window.fetch.bind(window);
    window.__uiDefectFetchCalls = 0;
    window.fetch = async (...args) => {
      const raw = typeof args[0] === "string" ? args[0] : args[0] instanceof URL ? args[0].toString() : args[0].url;
      if (raw.includes(pathFragment)) {
        window.__uiDefectFetchCalls += 1;
        await new Promise((resolve) => window.setTimeout(resolve, delayMS));
      }
      return original(...args);
    };
  }, { pathFragment, delayMS });
}

async function delayedFetchCalls(page) {
  return page.evaluate(() => window.__uiDefectFetchCalls ?? 0);
}

function pass(name, detail) {
  scenarios.push({ name, status: "PASS", detail });
}
