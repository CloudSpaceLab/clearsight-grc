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

async function openContext(viewport, touch = false) {
  const context = await browser.newContext({ viewport, colorScheme: "light", hasTouch: touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(() => {
    localStorage.setItem("clearsight.theme", "light");
    localStorage.setItem("clearsight.density", "comfortable");
  });
  return context;
}

async function openProgram(context) {
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });
  await page.goto(`${baseURL}/?tour=off#programs/program-ndpa`, { waitUntil: "networkidle" });
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

async function capture(page, name, state) {
  await page.screenshot({ path: path.join(outputDir, `${name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  await appendRecord(page, name, state);
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

async function appendRecord(page, name, state) {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  const viewport = page.viewportSize() ?? { width: 0, height: 0 };
  manifest.captures.push({ name, route: "#programs/program-ndpa", fixture: null, state, viewport, theme: "light", density: "comfortable", metrics: await layoutMetrics(page) });
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
