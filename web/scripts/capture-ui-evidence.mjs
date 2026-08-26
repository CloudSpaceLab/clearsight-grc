import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const browser = await chromium.launch({ headless: true });
const results = [];
let failure = null;

await mkdir(outputDir, { recursive: true });

const captures = [
  { name: "01-today-dark-comfortable-1440x900", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, assertFirstActionVisible: true },
  { name: "02-today-light-comfortable-1440x900", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, assertFirstActionVisible: true },
  { name: "03-today-dark-compact-1440x900", route: "#today", title: "Today", theme: "dark", density: "compact", viewport: { width: 1440, height: 900 }, assertFirstActionVisible: true },
  { name: "04-program-light-1440x900", route: "#programs/program-ndpa", title: "Programs", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } },
  { name: "05-matter-dark-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 } },
  { name: "06-evidence-light-1440x900", route: "#work/evidence", title: "Work", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } },
  { name: "07-import-dark-1440x900", route: "#imports", title: "Imports", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 } },
  { name: "08-configure-light-1440x900", route: "#configure", title: "Routing and approvals", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } },
  { name: "09-today-dark-tablet-1024x768", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 1024, height: 768 }, touch: true, assertFirstActionVisible: true },
  { name: "10-today-light-mobile-390x844", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, assertFirstActionVisible: true },
  { name: "11-today-dark-reflow-320x800", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 320, height: 800 }, touch: true },
  { name: "17-today-empty-light-1440x900", route: "#today", title: "Today", fixture: "today-empty", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Nothing needs your action right now" },
  { name: "18-today-loading-dark-1440x900", route: "#today", title: "Today", fixture: "today-loading", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Loading Today…" },
  { name: "19-today-unavailable-light-1440x900", route: "#today", title: "Today", fixture: "today-unavailable", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Today is unavailable" },
  { name: "20-evidence-partial-light-1440x900", route: "#work/evidence", title: "Work", fixture: "evidence-requests-unavailable", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Evidence requests are unavailable" },
  { name: "21-configure-partial-dark-1440x900", route: "#configure", title: "Routing and approvals", fixture: "configure-partial", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Routing policies are unavailable" },
  { name: "22-no-config-access-light-1440x900", route: "#configure", title: "Today", fixture: "no-config-access", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Reviews, approvals and evidence requests assigned to you.", assertNoConfigureNav: true },
  { name: "27-evidence-long-content-mobile-390x844", route: "#work/evidence", title: "Work", fixture: "long-content", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, expectText: "Confirm the accountable owner for the processor register" },
  { name: "37-new-work-light-1440x900", route: "#work/matters", title: "Work", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, openMatterSetup: true },
  { name: "38-new-work-dark-mobile-390x844", route: "#work/matters", title: "Work", theme: "dark", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, openMatterSetup: true },
];

try {
  for (const capture of captures) await capturePage(capture);
  await captureRouting();
  await captureAuthorityForbidden();
  await captureEvidenceReviewAndReceipt();
  await captureCaptureNotFound();
  await captureCaptureTerminal();
  await captureCaptureConflict();
  await captureMobileCaptureAndFocus();
  await captureZoomProxy();
  await captureFieldVisit();
  await captureImportSelection();
} catch (error) {
  failure = error instanceof Error ? error.message : String(error);
  throw error;
} finally {
  await writeManifest();
  await browser.close();
}

async function capturePage(capture) {
  const { context, page } = await openPage(capture);
  try {
    if (capture.expectText) await page.getByText(capture.expectText, { exact: false }).first().waitFor({ state: "visible" });
    if (capture.openMatterSetup) {
      await page.getByRole("button", { name: "New issue or change" }).click();
      const heading = page.getByRole("heading", { name: "New issue or change" });
      await heading.waitFor({ state: "visible" });
      await heading.scrollIntoViewIfNeeded();
      const workType = page.locator('select[name="type"]');
      if (!await workType.evaluate((element) => element === document.activeElement)) throw new Error(`${capture.name} did not focus the first creation field`);
    }
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.openMatterSetup ? "matter-create-open" : capture.fixture ? `fixture:${capture.fixture}` : "baseline");
    await assertNoHorizontalOverflow(page, capture.name);
    if (capture.assertFirstActionVisible) await assertFirstActionVisible(page, capture.viewport.height, capture.name, capture.touch === true);
    if (capture.assertNoConfigureNav && await page.getByRole("button", { name: /Configure/ }).count()) throw new Error(`${capture.name} exposes Configure without config-read capability`);
    if (capture.fixture === "evidence-requests-unavailable") {
      const inventory = page.locator(".source-inventory");
      await inventory.locator("summary").click();
      await page.getByText("Identity and access records").waitFor({ state: "visible" });
    }
    if (capture.fixture === "configure-partial" && !(await page.getByText("Confirm the final DPCO review date").isVisible())) throw new Error("Configure partial-degradation state hid still-available workflow ownership");
  } finally {
    await context.close();
  }
}

async function openPage({ route, title, theme, density, viewport, touch = false, fixture }) {
  const context = await browser.newContext({ viewport, colorScheme: theme, hasTouch: touch, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(({ theme, density }) => {
    localStorage.setItem("clearsight.theme", theme);
    localStorage.setItem("clearsight.density", density);
  }, { theme, density });
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });
  const params = new URLSearchParams({ tour: "off" });
  if (fixture) params.set("fixture", fixture);
  await page.goto(`${baseURL}/?${params.toString()}${route}`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: title, exact: true }).first().waitFor({ state: "visible" });
  await page.evaluate(() => document.fonts?.ready);
  if (browserErrors.length) throw new Error(`${route} emitted browser errors:\n${browserErrors.join("\n")}`);
  return { context, page };
}

async function saveScreenshot(page, name) {
  await page.screenshot({ path: path.join(outputDir, `${name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
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
    activeElement: document.activeElement instanceof HTMLElement ? document.activeElement.getAttribute("aria-label") ?? document.activeElement.textContent?.trim().slice(0, 80) ?? document.activeElement.tagName : "unknown",
  }));
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await layoutMetrics(page);
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
}

async function assertFirstActionVisible(page, viewportHeight, name, touch) {
  const action = page.locator(".intervention-next .primary-button").first();
  await action.waitFor({ state: "visible" });
  const box = await action.boundingBox();
  const safeBottom = touch ? 82 : 0;
  if (!box || box.y < 0 || box.y + box.height > viewportHeight - safeBottom) throw new Error(`${name} does not keep the first Today action inside the unobstructed first viewport`);
}

async function assertFocusInsideSheet(page, name) {
  const inside = await page.evaluate(() => Boolean(document.activeElement?.closest(".side-panel")));
  if (!inside) throw new Error(`${name} allowed keyboard focus to escape the focused-work sheet`);
}

async function record(page, capture, state) {
  results.push({ name: capture.name, route: capture.route, fixture: capture.fixture ?? null, state, viewport: capture.viewport, theme: capture.theme, density: capture.density, metrics: await layoutMetrics(page) });
  await writeManifest();
}

async function writeManifest() {
  await writeFile(path.join(outputDir, "manifest.json"), JSON.stringify({ generatedAt: new Date().toISOString(), baseURL, failure, captures: results }, null, 2));
}

async function captureRouting() {
  const capture = { name: "12-authority-dark-1440x900", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await page.getByRole("button", { name: "Check authority" }).click();
    await page.getByRole("heading", { name: "Authority for this item" }).waitFor();
    await page.getByText("Deputy Data Protection Compliance Officer").waitFor();
    await assertFocusInsideSheet(page, capture.name);
    if (await page.getByText("Control Assurance", { exact: true }).count()) throw new Error("Authority evidence still contains the removed hard-coded Control Assurance stage");
    await saveScreenshot(page, capture.name);
    await record(page, capture, "exact-authority-candidate-set");
  } finally {
    await context.close();
  }
}

async function captureAuthorityForbidden() {
  const capture = { name: "23-authority-forbidden-light-1440x900", route: "#today", title: "Today", fixture: "authority-forbidden", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await page.getByRole("button", { name: "Check authority" }).click();
    await page.getByRole("heading", { name: "Authority details are restricted" }).waitFor();
    await assertFocusInsideSheet(page, capture.name);
    if (await page.getByText("Data Protection Compliance Officer").count()) throw new Error("Forbidden authority state leaked candidate details");
    await saveScreenshot(page, capture.name);
    await record(page, capture, "permission-denied");
  } finally {
    await context.close();
  }
}

async function openEvidenceCapture(page) {
  await page.getByRole("button", { name: "Respond to evidence request" }).click();
  await page.locator(".side-panel").waitFor({ state: "visible" });
}

async function fillCapture(page) {
  await page.getByRole("textbox", { name: /Processor register owner/ }).fill("Privacy Operations");
  await page.getByLabel(/DPCO review date/).fill("2027-03-01");
}

async function captureEvidenceReviewAndReceipt() {
  const capture = { name: "13-capture-entry-light-1440x900", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await openEvidenceCapture(page);
    await page.getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
    await assertFocusInsideSheet(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "response-entry");
    await fillCapture(page);
    const reviewButton = page.getByRole("button", { name: "Review and submit" });
    if (!(await reviewButton.isEnabled())) throw new Error("Capture review remains disabled after every required field is completed");
    await reviewButton.click();
    await page.getByRole("heading", { name: "Check your response" }).waitFor();
    const reviewCapture = { ...capture, name: "14-capture-review-light-1440x900" };
    await saveScreenshot(page, reviewCapture.name);
    await record(page, reviewCapture, "response-review");
    await page.getByRole("button", { name: "Submit response" }).click();
    await page.getByRole("heading", { name: "Response submitted" }).waitFor();
    const receiptCapture = { ...capture, name: "15-capture-receipt-light-1440x900" };
    await saveScreenshot(page, receiptCapture.name);
    await record(page, receiptCapture, "submission-receipt");
  } finally {
    await context.close();
  }
}

async function captureCaptureNotFound() {
  const capture = { name: "24-capture-not-found-dark-1440x900", route: "#today", title: "Today", fixture: "capture-not-found", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await openEvidenceCapture(page);
    await page.getByRole("heading", { name: "This request is no longer available" }).waitFor();
    await saveScreenshot(page, capture.name);
    await record(page, capture, "not-found");
  } finally {
    await context.close();
  }
}

async function captureCaptureTerminal() {
  const capture = { name: "25-capture-expired-light-1440x900", route: "#today", title: "Today", fixture: "capture-terminal", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await openEvidenceCapture(page);
    await page.getByRole("heading", { name: "This request has expired" }).waitFor();
    if (await page.getByRole("button", { name: "Review and submit" }).count()) throw new Error("Expired request still exposed response submission");
    await saveScreenshot(page, capture.name);
    await record(page, capture, "terminal-expired");
  } finally {
    await context.close();
  }
}

async function captureCaptureConflict() {
  const capture = { name: "26-capture-conflict-light-1440x900", route: "#today", title: "Today", fixture: "capture-conflict", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await openEvidenceCapture(page);
    await page.getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
    await fillCapture(page);
    await page.getByRole("button", { name: "Review and submit" }).click();
    await page.getByRole("button", { name: "Submit response" }).click();
    await page.getByText("This request changed while you were working. Reload it before submitting. Your current entries remain on this screen.").waitFor();
    await page.getByRole("button", { name: "Reload request" }).waitFor();
    await saveScreenshot(page, capture.name);
    await record(page, capture, "optimistic-conflict");
  } finally {
    await context.close();
  }
}

async function captureMobileCaptureAndFocus() {
  const capture = { name: "28-capture-mobile-light-390x844", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true };
  const { context, page } = await openPage(capture);
  try {
    const more = page.getByText("More actions", { exact: true });
    if (await more.count()) await more.click();
    await page.getByRole("button", { name: "Respond to evidence request" }).click();
    await page.getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
    await assertFocusInsideSheet(page, capture.name);
    await assertNoHorizontalOverflow(page, capture.name);
    await page.keyboard.press("Tab");
    await assertFocusInsideSheet(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "mobile-focused-capture");
  } finally {
    await context.close();
  }
}

async function captureZoomProxy() {
  const capture = { name: "16-today-light-200pct-zoom-proxy", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    await page.evaluate(() => { document.documentElement.style.zoom = "2"; });
    await saveScreenshot(page, capture.name);
    await record(page, capture, "css-zoom-200pct-proxy");
    await assertNoHorizontalOverflow(page, capture.name);
  } finally {
    await context.close();
  }
}

async function captureFieldVisit() {
  const capture = { name: "29-field-visit-entry-light-390x844", route: "?capture_invite=field-agent-demo", title: "Verify ATM location after your visit", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true };
  const context = await browser.newContext({ viewport: capture.viewport, colorScheme: capture.theme, hasTouch: true, reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(() => {
    localStorage.setItem("clearsight.theme", "light");
    localStorage.setItem("clearsight.density", "comfortable");
  });
  const page = await context.newPage();
  try {
    await page.goto(`${baseURL}/?capture_invite=field-agent-demo`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Open your request" }).waitFor();
    await page.getByRole("textbox", { name: "Email or phone number" }).fill("field.agent@example.com");
    await page.getByRole("button", { name: "Open request" }).click();
    await page.getByRole("heading", { name: capture.title }).waitFor();
    await page.getByText("12 Admiralty Way, Lekki Phase 1, Lagos", { exact: true }).waitFor();
    if (await page.getByRole("textbox", { name: /address/i }).count()) throw new Error("Field visit asks the agent to re-enter the known address");
    const note = page.locator('textarea[aria-label="Anything the reviewer should know?"]');
    if (await note.count() !== 1) throw new Error("Field visit does not expose exactly one optional exception note");
    if (await note.isVisible()) throw new Error("Optional field-visit note is expanded on the normal happy path");
    await page.getByRole("radio", { name: "Yes" }).nth(0).check();
    await page.getByRole("radio", { name: "Yes" }).nth(1).check();
    const sitePhoto = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=", "base64");
    await page.getByLabel(/Site photo/).setInputFiles({ name: "atm-site.png", mimeType: "image/png", buffer: sitePhoto });
    await page.getByText(/atm-site\.png/).waitFor();
    const photoPreview = page.locator(".file-dropzone-preview");
    await photoPreview.waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Add signature" }).click();
    await page.getByRole("button", { name: "Type" }).click();
    await page.getByRole("textbox", { name: "Your name" }).fill("Amina Bello");
    await page.getByRole("button", { name: "Use signature" }).click();
    if (!(await page.getByRole("button", { name: "Review and submit" }).isEnabled())) throw new Error("Field visit is not ready after two confirmations, one photo and one signature");
    await photoPreview.scrollIntoViewIfNeeded();
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "external-field-visit-entry");
    await page.getByRole("button", { name: "Review and submit" }).click();
    await page.getByRole("heading", { name: "Check your response" }).waitFor();
    await page.getByText("Photo attached · atm-site.png").waitFor();
    await page.getByText("Signed", { exact: true }).waitFor();
    const reviewCapture = { ...capture, name: "30-field-visit-review-light-390x844" };
    await saveScreenshot(page, reviewCapture.name);
    await record(page, reviewCapture, "external-field-visit-review");
    await page.getByRole("button", { name: "Submit evidence" }).click();
    await page.getByRole("heading", { name: "Submitted" }).waitFor();
    const receiptCapture = { ...capture, name: "31-field-visit-receipt-light-390x844" };
    await saveScreenshot(page, receiptCapture.name);
    await record(page, receiptCapture, "external-field-visit-receipt");
  } finally {
    await context.close();
  }
}

async function captureImportSelection() {
  const capture = { name: "32-import-dropzone-selected-light-1440x900", route: "#imports", title: "Imports", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(capture);
  try {
    const form = page.locator(".document-import-form");
    if (!(await form.isVisible())) {
      await page.getByRole("button", { name: "Import document", exact: true }).click();
      await form.waitFor({ state: "visible" });
    }
    const purpose = form.getByRole("textbox", { name: "What should reviewers look for?" });
    if ((await purpose.inputValue()) !== "") throw new Error("Document import still starts with a persisted template purpose");
    await form.locator(".file-dropzone-input").setInputFiles({ name: "outsourcing-policy.pdf", mimeType: "application/pdf", buffer: Buffer.from("sample-policy") });
    await page.getByText(/outsourcing-policy\.pdf/).waitFor();
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "document-selected-before-import");
  } finally {
    await context.close();
  }
}
