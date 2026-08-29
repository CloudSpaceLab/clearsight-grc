import { chromium } from "playwright";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const browser = await chromium.launch({ headless: true });

try {
  await captureLibrary();
  await captureSearchEmpty();
  await captureSentForms();
  await captureMobileAmendment();
  await captureResponses();
} catch (error) {
  await recordFailure(error);
  throw error;
} finally {
  await browser.close();
}

async function captureLibrary() {
  const { context, page } = await openForms({ width: 1440, height: 900 }, "light");
  try {
    await page.getByText("Vendor security and privacy review", { exact: true }).first().waitFor({ state: "visible" });
    await verifyAndCapture(page, "89-forms-library-light-1440x900", "forms-template-library", "#forms", "light");
  } finally { await context.close(); }
}

async function captureSearchEmpty() {
  const { context, page } = await openForms({ width: 390, height: 844 }, "dark", true);
  try {
    await page.getByLabel("Search templates").fill("no matching bank form");
    await page.getByRole("heading", { name: "No templates match “no matching bank form”" }).waitFor({ state: "visible" });
    await verifyAndCapture(page, "90-forms-search-empty-dark-mobile-390x844", "forms-template-search-empty", "#forms", "dark");
  } finally { await context.close(); }
}

async function captureSentForms() {
  const { context, page } = await openForms({ width: 1440, height: 900 }, "light");
  try {
    await page.getByRole("button", { name: "Sent forms", exact: true }).click();
    await page.getByRole("heading", { name: "Sent forms", exact: true }).waitFor({ state: "visible" });
    await page.getByText("Acme annual vendor review", { exact: true }).first().waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Amend distribution" }).waitFor({ state: "visible" });
    await verifyAndCapture(page, "91-forms-sent-light-1440x900", "forms-sent-management", "#forms", "light");
  } finally { await context.close(); }
}

async function captureMobileAmendment() {
  const { context, page } = await openForms({ width: 390, height: 844 }, "dark", true);
  try {
    await page.getByRole("button", { name: "Sent forms", exact: true }).click();
    await page.getByRole("button", { name: "Amend distribution" }).waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Amend distribution" }).click();
    await page.getByRole("heading", { name: "Amend distribution" }).waitFor({ state: "visible" });
    for (const label of ["Response deadline", "Access expiry"]) {
      const input = page.locator("label", { hasText: label }).first().locator("input");
      if (await input.getAttribute("type") !== "datetime-local") throw new Error(`${label} is not a native calendar and time input`);
    }
    await page.getByLabel("External email").fill("updated.contact@acme.example");
    await page.getByRole("button", { name: "Add external To" }).click();
    await page.getByText("u***@acme.example", { exact: true }).waitFor({ state: "visible" });
    await page.getByRole("heading", { name: "Amend distribution" }).scrollIntoViewIfNeeded();
    await verifyAndCapture(page, "92-forms-amend-dark-mobile-390x844", "forms-amend-mobile", "#forms", "dark");
  } finally { await context.close(); }
}

async function captureResponses() {
  const { context, page } = await openForms({ width: 1440, height: 900 }, "light");
  try {
    await page.getByRole("button", { name: "Responses", exact: true }).click();
    await page.getByRole("heading", { name: "Responses", exact: true }).waitFor({ state: "visible" });
    await page.getByText("Revision 2 · Current", { exact: true }).waitFor({ state: "visible" });
    await page.getByText("86%", { exact: true }).waitFor({ state: "visible" });
    await verifyAndCapture(page, "93-forms-responses-light-1440x900", "forms-response-revisions", "#forms", "light");
  } finally { await context.close(); }
}

async function openForms(viewport, theme, touch = false) {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(({ selectedTheme }) => {
    localStorage.setItem("clearsight.theme", selectedTheme);
    localStorage.setItem("clearsight.density", "comfortable");
  }, { selectedTheme: theme });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  await page.goto(`${baseURL}/?tour=off#forms`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Forms", exact: true }).waitFor({ state: "visible" });
  await page.getByLabel("Search templates").waitFor({ state: "visible" });
  await page.evaluate(() => document.fonts?.ready);
  if (errors.length) throw new Error(`Forms emitted browser errors:\n${errors.join("\n")}`);
  return { context, page };
}

async function verifyAndCapture(page, name, state, route, theme) {
  const metrics = await layoutMetrics(page);
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
  await page.screenshot({ path: path.join(outputDir, `${name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  manifest.captures.push({ name, route, fixture: null, state, viewport: page.viewportSize(), theme, density: "comfortable", metrics });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}

function layoutMetrics(page) {
  return page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth, clientHeight: document.documentElement.clientHeight, scrollHeight: document.documentElement.scrollHeight, scrollY: window.scrollY, theme: document.documentElement.dataset.theme ?? "unknown", density: document.documentElement.dataset.density ?? "unknown" }));
}

async function recordFailure(error) {
  try {
    const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
    manifest.failure = error instanceof Error ? error.message : String(error);
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
  } catch { /* The primary evidence runner owns manifest creation. */ }
}
