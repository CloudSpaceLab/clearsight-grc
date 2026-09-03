import { chromium } from "playwright";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const browser = await chromium.launch({ headless: true });

try {
  await captureDesktopReview();
  await captureMobileReview();
  await captureProgramSectionsAndCollections();
} catch (error) {
  await recordFailure(error);
  throw error;
} finally {
  await browser.close();
}

async function captureDesktopReview() {
  const context = await openContext({ width: 1440, height: 900 });
  const page = await openProgram(context);
  try {
    const digest = page.locator(".program-review-digest");
    await digest.scrollIntoViewIfNeeded();
    await page.getByRole("heading", { name: "3 changes since your last review" }).waitFor({ state: "visible" });
    await assertChangedContent(page);
    await assertNoHorizontalOverflow(page, "37-program-review-changed-light-1440x900");
    await capture(page, "37-program-review-changed-light-1440x900", "program-review-changed");

    await page.getByRole("button", { name: "Mark current state reviewed" }).click();
    await page.getByRole("heading", { name: "No changes since your review" }).waitFor({ state: "visible" });
    if (await page.getByRole("button", { name: "Mark current state reviewed" }).count()) throw new Error("acknowledged Program still exposes a redundant review action");
    await assertNoHorizontalOverflow(page, "38-program-review-acknowledged-light-1440x900");
    await capture(page, "38-program-review-acknowledged-light-1440x900", "program-review-acknowledged");
  } finally {
    await context.close();
  }
}

async function captureMobileReview() {
  const context = await openContext({ width: 390, height: 844 }, true);
  const page = await openProgram(context);
  try {
    const digest = page.locator(".program-review-digest");
    await digest.scrollIntoViewIfNeeded();
    await page.getByRole("heading", { name: "3 changes since your last review" }).waitFor({ state: "visible" });
    await assertChangedContent(page);
    await assertNoHorizontalOverflow(page, "39-program-review-changed-light-mobile-390x844");
    const button = page.getByRole("button", { name: "Mark current state reviewed" });
    const box = await button.boundingBox();
    if (!box || box.width < 44 || box.height < 40) throw new Error("mobile Program review action is too small for reliable touch use");
    await capture(page, "39-program-review-changed-light-mobile-390x844", "program-review-mobile-changed");
  } finally {
    await context.close();
  }
}

async function openContext(viewport, touch = false, theme = "light") {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript((selectedTheme) => {
    localStorage.setItem("clearsight.theme", selectedTheme);
    localStorage.setItem("clearsight.density", "comfortable");
  }, theme);
  return context;
}

async function openProgram(context, { section = "overview", fixture } = {}) {
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });
  const params = new URLSearchParams({ tour: "off" });
  if (fixture) params.set("fixture", fixture);
  await page.goto(`${baseURL}/?${params.toString()}#programs/program-ndpa/${section}`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Programs", exact: true }).waitFor({ state: "visible" });
  await page.getByText("Nigeria Data Protection Programme", { exact: true }).first().waitFor({ state: "visible" });
  await page.evaluate(() => document.fonts?.ready);
  if (browserErrors.length) throw new Error(`Program review emitted browser errors:\n${browserErrors.join("\n")}`);
  return page;
}

async function assertChangedContent(page) {
  await page.getByText("Overall status changed from current to evidence insufficient.", { exact: true }).waitFor({ state: "visible" });
  await page.getByText("Two annual-return evidence sections still need an accountable owner.", { exact: true }).first().waitFor({ state: "visible" });
  if (await page.getByText(/recommendation/i).count()) throw new Error("Program review digest invented recommendation semantics");
}

async function captureProgramSectionsAndCollections() {
  await captureProgramSection({ name: "40-program-overview-dark-1440x900", section: "overview", theme: "dark", viewport: { width: 1440, height: 900 }, state: "program-overview-dark" });
  await captureProgramSection({ name: "41-program-evidence-results-light-1024x768", section: "evidence-results", theme: "light", viewport: { width: 1024, height: 768 }, touch: true, state: "program-evidence-results-tablet" });
  await captureProgramSection({ name: "42-program-monitoring-light-1440x900", section: "monitoring", fixture: "collection-renewal-states", theme: "light", viewport: { width: 1440, height: 900 }, state: "collection-current", expectedText: "Ada Okafor · Vendor assurance lead" });
  await captureProgramSection({ name: "43-program-monitoring-blocked-dark-1440x900", section: "monitoring", fixture: "collection-renewal-states", theme: "dark", viewport: { width: 1440, height: 900 }, state: "collection-delivery-blocked", expectedText: "Renewal blocked", scrollText: "Renewal blocked" });
  await captureProgramSection({ name: "44-program-monitoring-light-mobile-390x844", section: "monitoring", fixture: "collection-renewal-states", theme: "light", viewport: { width: 390, height: 844 }, touch: true, compact: true, state: "collection-mobile-selector", expectedText: "Sample · Current vendor security confirmation" });
  await captureProgramSection({ name: "45-program-monitoring-long-dark-reflow-320x800", section: "monitoring", fixture: "collection-long-content", theme: "dark", viewport: { width: 320, height: 800 }, touch: true, compact: true, state: "collection-long-content", expectedText: "cross-border payment-processing" });
  await captureProgramSection({ name: "46-program-monitoring-light-200pct-reflow-proxy", section: "monitoring", fixture: "collection-renewal-states", theme: "light", viewport: { width: 720, height: 900 }, compact: true, state: "collection-200pct-reflow-proxy", expectedText: "Sample · Current vendor security confirmation" });

  const context = await openContext({ width: 1440, height: 900 });
  const page = await openProgram(context, { section: "overview" });
  try {
    const monitoringTab = page.getByRole("tab", { name: "Monitoring" });
    await monitoringTab.focus();
    if (!(await monitoringTab.evaluate((element) => element === document.activeElement))) throw new Error("Program Monitoring tab did not retain keyboard focus");
    await capture(page, "47-program-sections-tab-focus-light-1440x900", "program-tab-focus", { route: "#programs/program-ndpa/overview" });
  } finally {
    await context.close();
  }
}

async function captureProgramSection({ name, section, fixture, theme, viewport, touch = false, compact = false, state, expectedText, scrollText }) {
  const context = await openContext(viewport, touch, theme);
  const page = await openProgram(context, { section, fixture });
  try {
    if (compact) {
      await page.getByRole("combobox", { name: "Program section" }).waitFor({ state: "visible" });
      if (await page.getByRole("tablist", { name: "Program sections" }).count()) throw new Error(`${name} retained the desktop tablist in a compact viewport`);
    } else {
      await page.getByRole("tablist", { name: "Program sections" }).waitFor({ state: "visible" });
    }
    if (expectedText) await page.getByText(expectedText, { exact: false }).first().waitFor({ state: "visible" });
    const anchor = scrollText ? page.getByText(scrollText, { exact: true }).first() : page.locator(".program-detail-sections");
    await anchor.evaluate((element) => element.scrollIntoView({ block: "start" }));
    await page.evaluate(() => window.scrollBy(0, -70));
    await assertNoHorizontalOverflow(page, name);
    await capture(page, name, state, { route: `#programs/program-ndpa/${section}`, fixture: fixture ?? null, theme });
  } finally {
    await context.close();
  }
}

async function capture(page, name, state, metadata = {}) {
  await page.screenshot({ path: path.join(outputDir, `${name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  await appendRecord(page, name, state, metadata);
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await layoutMetrics(page);
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
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

async function appendRecord(page, name, state, metadata = {}) {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  const viewport = page.viewportSize() ?? { width: 0, height: 0 };
  manifest.captures.push({ name, route: metadata.route ?? "#programs/program-ndpa/overview", fixture: metadata.fixture ?? null, state, viewport, theme: metadata.theme ?? "light", density: "comfortable", metrics: await layoutMetrics(page) });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}

async function recordFailure(error) {
  try {
    const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
    manifest.failure = error instanceof Error ? error.message : String(error);
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
  } catch {
    // The primary evidence runner owns manifest creation.
  }
}
