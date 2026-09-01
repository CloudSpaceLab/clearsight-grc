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
  { name: "08-configure-light-1440x900", route: "#configure", title: "Configuration", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Control plane" },
  { name: "09-today-dark-tablet-1024x768", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 1024, height: 768 }, touch: true, assertFirstActionVisible: true },
  { name: "10-today-light-mobile-390x844", route: "#today", title: "Today", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, assertFirstActionVisible: true },
  { name: "11-today-dark-reflow-320x800", route: "#today", title: "Today", theme: "dark", density: "comfortable", viewport: { width: 320, height: 800 }, touch: true },
  { name: "17-today-empty-light-1440x900", route: "#today", title: "Today", fixture: "today-empty", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Nothing needs your action right now" },
  { name: "18-today-loading-dark-1440x900", route: "#today", title: "Today", fixture: "today-loading", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Loading Today…" },
  { name: "19-today-unavailable-light-1440x900", route: "#today", title: "Today", fixture: "today-unavailable", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Today is unavailable" },
  { name: "20-evidence-partial-light-1440x900", route: "#work/evidence", title: "Work", fixture: "evidence-requests-unavailable", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Evidence requests are unavailable" },
  { name: "21-configure-partial-dark-1440x900", route: "#configure/authority", title: "Configuration", fixture: "configure-partial", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Routing policies are unavailable" },
  { name: "22-no-config-access-light-1440x900", route: "#configure", title: "Today", fixture: "no-config-access", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, expectText: "Assigned work and operational exceptions you are permitted to handle.", assertNoConfigureNav: true },
  { name: "27-evidence-long-content-mobile-390x844", route: "#work/evidence", title: "Work", fixture: "long-content", theme: "light", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, expectText: "Confirm the accountable owner for the processor register" },
  { name: "37-new-work-light-1440x900", route: "#work/matters", title: "Work", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, openMatterSetup: true },
  { name: "38-new-work-dark-mobile-390x844", route: "#work/matters", title: "Work", theme: "dark", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, openMatterSetup: true },
  { name: "83-program-filters-light-1440x900", route: "#programs?overall_state=EVIDENCE_INSUFFICIENT&jurisdiction=Nigeria", title: "Programs", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "program-portfolio-filters" },
  { name: "84-matter-filters-dark-1440x900", route: "#work/matters?matter_type=REGULATORY_CHANGE&priority=4&due=DUE_7_DAYS", title: "Work", theme: "dark", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "matter-portfolio-filters" },
  { name: "85-vendor-add-light-1440x900", route: "#vendors", title: "Vendors", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "vendor-add-website-address", openVendorSetup: true },
  { name: "86-vendor-form-readiness-light-1440x900", route: "#vendors", title: "Vendors", fixture: "vendor-no-form", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "vendor-form-readiness", openFormReadiness: true },
  { name: "87-vendor-link-sheet-light-1440x900", route: "#programs/program-ndpa", title: "Programs", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "vendor-link-focused-sheet", openVendorLink: true },
  { name: "88-vendor-link-sheet-dark-mobile-390x844", route: "#programs/program-ndpa", title: "Programs", theme: "dark", density: "comfortable", viewport: { width: 390, height: 844 }, touch: true, state: "vendor-link-focused-sheet-mobile", openVendorLink: true },
  { name: "89-matter-action-reassignment-light-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", fixture: "matter-action-reassignment", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 }, state: "matter-action-reassignment", openActionReassignment: true },
];

try {
  await captureVendorLinkedWorkflows();
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
  await captureVendorWorkflows();
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
    if (capture.openVendorSetup) {
      await page.getByRole("button", { name: "Add vendor" }).click();
      await page.getByRole("heading", { name: "Add a vendor and service" }).waitFor({ state: "visible" });
      await page.getByLabel("Website").waitFor({ state: "visible" });
      await page.getByLabel("Registered address").waitFor({ state: "visible" });
    }
    if (capture.openFormReadiness) {
      await page.getByRole("button", { name: /Acme Processing Limited.*Card transaction processing/ }).click();
      await page.getByRole("heading", { name: "Due diligence" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: "Use a starter template" }).click();
      await page.getByRole("dialog", { name: "Set up due-diligence form" }).waitFor({ state: "visible" });
      await page.getByLabel("Program").waitFor({ state: "visible" });
    }
    if (capture.openVendorLink) {
      const link = page.getByRole("button", { name: "Link vendor" });
      await link.scrollIntoViewIfNeeded();
      await link.click();
      const dialog = page.getByRole("dialog", { name: "Link vendor to this Program" });
      await dialog.waitFor({ state: "visible" });
      await dialog.getByLabel("Search vendor relationships").fill("Acme");
      await dialog.getByRole("radio", { name: /Acme Processing Limited.*Card transaction processing/ }).waitFor({ state: "visible" });
    }
    if (capture.openActionReassignment) {
      await page.getByRole("button", { name: /Change owner for Complete the annual return evidence checklist/ }).click();
      const dialog = page.getByRole("dialog", { name: "Change action owner" });
      await dialog.waitFor({ state: "visible" });
      await dialog.getByText("Privacy Control Owner", { exact: true }).waitFor({ state: "visible" });
      await dialog.getByRole("button", { name: /Select an eligible performer New action owner/ }).waitFor({ state: "visible" });
      if (await dialog.getByRole("button", { name: "Assign action owner" }).isEnabled()) throw new Error(`${capture.name} permits reassignment before a replacement and reason are entered`);
    }
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state ?? (capture.openMatterSetup ? "matter-create-open" : capture.fixture ? `fixture:${capture.fixture}` : "baseline"));
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
  try {
    await page.waitForFunction(() => Boolean(document.activeElement?.closest(".cs-sheet")));
  } catch {
    throw new Error(`${name} allowed keyboard focus to escape the focused-work sheet`);
  }
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
    const authoritySheet = page.locator(".cs-sheet");
    if (await authoritySheet.getByText("Data Protection Compliance Officer").count()) throw new Error("Forbidden authority state leaked candidate details");
    await saveScreenshot(page, capture.name);
    await record(page, capture, "permission-denied");
  } finally {
    await context.close();
  }
}

async function openEvidenceCapture(page) {
  await page.getByRole("button", { name: "Respond to evidence request" }).click();
  await page.locator(".cs-sheet").waitFor({ state: "visible" });
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
    await page.locator(".cs-sheet").getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
    await assertFocusInsideSheet(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "response-entry");
    await fillCapture(page);
    const reviewButton = page.getByRole("button", { name: "Review response" });
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
    if (await page.getByRole("button", { name: "Review response" }).count()) throw new Error("Expired request still exposed response submission");
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
    await page.locator(".cs-sheet").getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
    await fillCapture(page);
    await page.getByRole("button", { name: "Review response" }).click();
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
    await page.locator(".cs-sheet").getByRole("heading", { name: "Confirm the remaining annual-return evidence owners" }).waitFor();
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
    const dropzoneBox = await form.locator(".file-dropzone").boundingBox();
    const replaceBox = await form.getByRole("button", { name: "Replace file" }).boundingBox();
    if (!dropzoneBox || !replaceBox || replaceBox.width < 80 || replaceBox.x + replaceBox.width > dropzoneBox.x + dropzoneBox.width + 1) {
      throw new Error("Selected document replacement action is clipped or too narrow to read.");
    }
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "document-selected-before-import");
  } finally {
    await context.close();
  }
}

async function assertVendorWorkContained(page, name) {
  const result = await page.locator(".vendor-work-panel").evaluate((panel) => {
    const viewportWidth = document.documentElement.clientWidth;
    const elements = [panel, ...panel.querySelectorAll(".vendor-work-form, .vendor-work-card, .vendor-work-response, input, select, textarea")];
    return elements.map((element) => {
      const rect = element.getBoundingClientRect();
      return { tag: element.tagName, className: element.className, left: rect.left, right: rect.right };
    }).filter((rect) => rect.left < 0 || rect.right > viewportWidth + 1);
  });
  if (result.length) throw new Error(`${name} clips vendor work content: ${JSON.stringify(result.slice(0, 3))}`);
}

async function captureVendorWorkflows() {
  const scenarios = [
    { name: "40-vendor-start-light-1440x900", fixture: undefined, state: "vendor-due-diligence-start", viewport: { width: 1440, height: 900 }, action: "Start due diligence" },
    { name: "41-vendor-ready-dark-1440x900", fixture: "vendor-ready", state: "vendor-request-ready", viewport: { width: 1440, height: 900 }, action: "Send due diligence request", theme: "dark" },
    { name: "42-vendor-review-light-1440x900", fixture: "vendor-submitted", state: "vendor-response-review", viewport: { width: 1440, height: 900 }, action: "Review vendor response", startReview: true },
    { name: "43-vendor-review-light-390x844", fixture: "vendor-submitted", state: "vendor-response-review-mobile", viewport: { width: 390, height: 844 }, action: "Review vendor response", startReview: true, touch: true },
    { name: "44-vendor-source-degraded-light-1440x900", fixture: "vendor-source-degraded", state: "vendor-form-source-unavailable", viewport: { width: 1440, height: 900 }, expectText: "Due-diligence forms are unavailable" },
  ];
  for (const scenario of scenarios) {
    const capture = { ...scenario, route: "#vendors", title: "Vendors", theme: scenario.theme ?? "light", density: "comfortable" };
    const { context, page } = await openPage(capture);
    try {
      await page.getByRole("button", { name: /Acme Processing Limited/ }).click();
      await page.getByRole("heading", { name: scenario.expectText ?? "Due diligence" }).waitFor({ state: "visible" });
      if (scenario.action) await page.getByRole("button", { name: scenario.action }).waitFor({ state: "visible" });
      if (scenario.startReview) {
        await page.getByRole("button", { name: "Review vendor response" }).click();
        await page.getByRole("button", { name: "Record assessment conclusion" }).waitFor({ state: "visible" });
      }
      await assertNoHorizontalOverflow(page, capture.name);
      await saveScreenshot(page, capture.name);
      await record(page, capture, scenario.state);
    } finally {
      await context.close();
    }
  }

  const partial = { name: "45-vendor-delivery-partial-light-1440x900", fixture: "vendor-partial-delivery", state: "vendor-delivery-partial", route: "#vendors", title: "Vendors", theme: "light", density: "comfortable", viewport: { width: 1440, height: 900 } };
  const { context, page } = await openPage(partial);
  try {
    await page.getByRole("button", { name: /Acme Processing Limited/ }).click();
    await page.getByRole("button", { name: "Send due diligence request" }).click();
    await page.getByLabel("Vendor contact email").fill("security@acme.example");
    await page.getByLabel("Response due date").fill("2026-09-10");
    await page.getByRole("button", { name: "Send due diligence request" }).last().click();
    await page.getByText("Email delivery did not complete", { exact: true }).waitFor({ state: "visible" });
    if (await page.getByLabel("Vendor contact email").count()) throw new Error("Vendor contact email remained visible after the send attempt");
    await assertNoHorizontalOverflow(page, partial.name);
    await saveScreenshot(page, partial.name);
    await record(page, partial, partial.state);
  } finally {
    await context.close();
  }
}

async function captureVendorLinkedWorkflows() {
  await captureVendorTargetEntry({ name: "46-vendor-work-program-entry-light-1440x900", route: "#programs/program-ndpa", title: "Programs", fixture: "vendor-work-empty", state: "vendor-work-program-entry", theme: "light", viewport: { width: 1440, height: 900 } });
  await captureVendorTargetEntry({ name: "47-vendor-work-matter-entry-dark-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", fixture: "vendor-work-empty", state: "vendor-work-matter-entry", theme: "dark", viewport: { width: 1440, height: 900 } });
  await captureVendorWorkCreation();
  await captureVendorWorkCreationMobile();
  await captureVendorWorkDeliveryRecovery();
  await captureVendorWorkReview();
  await captureVendorWorkReviewMobile(390, "55-vendor-work-response-mobile-light-390x844", "vendor-work-response-mobile");
  await captureVendorWorkReviewMobile(320, "56-vendor-work-response-reflow-light-320x800", "vendor-work-response-reflow");
  await captureVendorWorkHistory();
}

async function openVendorWorkTarget(capture) {
  const opened = await openPage({ ...capture, density: "comfortable" });
  const heading = opened.page.getByRole("heading", { name: "Vendor requests", exact: true });
  await heading.waitFor({ state: "visible" });
  await heading.scrollIntoViewIfNeeded();
  return opened;
}

async function captureVendorTargetEntry(capture) {
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await page.getByRole("button", { name: "Request vendor work" }).waitFor({ state: "visible" });
    await page.getByText("No vendor requests have been recorded for this", { exact: false }).waitFor({ state: "visible" });
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, { ...capture, density: "comfortable" }, capture.state);
  } finally {
    await context.close();
  }
}

async function fillVendorWorkCreation(page, layout) {
  await page.getByRole("button", { name: "Request vendor work" }).click();
  await chooseSharedSelectOption(page, "Vendor relationship", 1);
  await page.getByLabel(/Request purpose/).fill("Confirm payment-service controls");
  await page.getByLabel(/Instructions for the vendor/).fill("Complete the control questions and provide the current independent assurance report.");
  await chooseSharedSelectOption(page, "Collection form", /version 3/);
  await assertSharedSelectOptions(page, "Form layout", ["Automatic", "Classic", "Wizard"]);
  await chooseSharedSelectOption(page, "Form layout", "Classic");
  await chooseSharedSelectOption(page, "Form layout", layout === "WIZARD" ? "Wizard" : "Automatic");
  const contact = page.getByLabel(/Vendor contact/);
  const due = page.getByLabel(/Due date/);
  if (await contact.getAttribute("type") !== "email" || await due.getAttribute("type") !== "date") throw new Error("Vendor work delivery fields do not use email and date input types");
  await contact.fill("security@acme.example");
  await due.fill("2026-09-30");
  await page.getByText("8 fields · 8 required · 1 document upload", { exact: true }).waitFor({ state: "visible" });
  if (!(await page.getByRole("button", { name: "Prepare and send request" }).isEnabled())) throw new Error("Vendor work request remains unavailable after every required field is completed");
}

async function chooseSharedSelectOption(page, label, option) {
  const trigger = page.getByRole("button", { name: new RegExp(label, "i") });
  await trigger.scrollIntoViewIfNeeded();
  await trigger.click();
  const listbox = page.getByRole("listbox");
  await listbox.waitFor({ state: "visible" });
  const options = listbox.getByRole("option");
  if (typeof option === "number") await options.nth(option).click();
  else await options.filter({ hasText: option }).click();
  await listbox.waitFor({ state: "detached" });
}

async function assertSharedSelectOptions(page, label, expected) {
  const trigger = page.getByRole("button", { name: new RegExp(label, "i") });
  await trigger.scrollIntoViewIfNeeded();
  await trigger.click();
  const listbox = page.getByRole("listbox");
  await listbox.waitFor({ state: "visible" });
  const actual = (await listbox.getByRole("option").allTextContents()).map((value) => value.trim());
  if (JSON.stringify(actual) !== JSON.stringify(expected)) throw new Error(`Vendor work form layout options were ${JSON.stringify(actual)} instead of ${JSON.stringify(expected)}.`);
  await page.keyboard.press("Escape");
  await listbox.waitFor({ state: "detached" });
}

async function captureVendorWorkCreation() {
  const capture = { name: "48-vendor-work-create-light-1440x900", route: "#programs/program-ndpa", title: "Programs", fixture: "vendor-work-empty", state: "vendor-work-create-layouts-and-typed-fields", theme: "light", viewport: { width: 1440, height: 900 }, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await fillVendorWorkCreation(page, "AUTOMATIC");
    await page.getByRole("heading", { name: "Request vendor work", exact: true }).scrollIntoViewIfNeeded();
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state);
  } finally {
    await context.close();
  }
}

async function captureVendorWorkCreationMobile() {
  const capture = { name: "49-vendor-work-create-wizard-light-390x844", route: "#work/matters/matter-gaid-change", title: "Work", fixture: "vendor-work-empty", state: "vendor-work-create-wizard-mobile", theme: "light", viewport: { width: 390, height: 844 }, touch: true, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await fillVendorWorkCreation(page, "WIZARD");
    await page.getByRole("heading", { name: "Request vendor work", exact: true }).scrollIntoViewIfNeeded();
    await assertVendorWorkContained(page, capture.name);
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state);
  } finally {
    await context.close();
  }
}

async function captureVendorWorkDeliveryRecovery() {
  const capture = { name: "50-vendor-work-delivery-partial-light-1440x900", route: "#programs/program-ndpa", title: "Programs", fixture: "vendor-work-partial-delivery", state: "vendor-work-delivery-partial", theme: "light", viewport: { width: 1440, height: 900 }, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    const retry = page.getByRole("button", { name: "Retry delivery" });
    await retry.waitFor({ state: "visible" });
    await page.getByText("Email delivery was not confirmed", { exact: false }).waitFor({ state: "visible" });
    await page.getByRole("heading", { name: "Current requests" }).scrollIntoViewIfNeeded();
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state);
    await page.getByLabel(/Vendor contact/).fill("security@acme.example");
    await retry.click();
    await page.getByText("Vendor request sent.", { exact: true }).waitFor({ state: "visible" });
    if (await page.getByRole("button", { name: "Retry delivery" }).count()) throw new Error("Retry delivery remained available after delivery succeeded");
    const recovered = { ...capture, name: "51-vendor-work-delivery-recovered-light-1440x900" };
    await saveScreenshot(page, recovered.name);
    await record(page, recovered, "vendor-work-delivery-recovered");
  } finally {
    await context.close();
  }
}

async function openVendorWorkResponse(page) {
  await page.getByRole("button", { name: "Review response" }).click();
  await page.getByRole("heading", { name: "Vendor response", exact: true }).waitFor({ state: "visible" });
  await page.getByText("Required response missing", { exact: true }).waitFor({ state: "visible" });
  await page.getByText("Not requested because its condition was not met", { exact: true }).waitFor({ state: "visible" });
  const available = page.getByRole("article", { name: "acme-iso-27001-certificate.pdf" });
  const quarantined = page.getByRole("article", { name: "acme-penetration-test-report.pdf" });
  if (await available.getByRole("button", { name: "Open document" }).count() !== 1 || await quarantined.getByRole("button", { name: "Open document" }).count() !== 0) throw new Error("Vendor response document actions do not match artifact availability");
}

async function captureVendorWorkReview() {
  const capture = { name: "52-vendor-work-response-light-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", fixture: "vendor-work-submitted", state: "vendor-work-response-review", theme: "light", viewport: { width: 1440, height: 900 }, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await openVendorWorkResponse(page);
    await page.getByRole("heading", { name: "Vendor response", exact: true }).scrollIntoViewIfNeeded();
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state);
    await page.getByRole("heading", { name: "Supporting documents", exact: true }).scrollIntoViewIfNeeded();
    const documents = { ...capture, name: "53-vendor-work-documents-light-1440x900" };
    await saveScreenshot(page, documents.name);
    await record(page, documents, "vendor-work-document-review");
    await page.getByRole("button", { name: "Begin review" }).click();
    await page.getByRole("button", { name: "Request changes" }).click();
    const changeMessage = page.getByLabel("What the vendor must change", { exact: true });
    await changeMessage.fill("Provide a clean current assurance report and identify the accountable control owner.");
    await page.getByLabel("Control owner", { exact: true }).check();
    await page.getByLabel(/Vendor contact/).fill("security@acme.example");
    await page.getByLabel("Revised due date", { exact: true }).fill("2026-10-07");
    if (!(await page.getByRole("button", { name: "Send change request" }).isEnabled())) throw new Error("Change request remains unavailable after its required decision record is complete");
    await page.locator(".vendor-work-decision textarea").first().scrollIntoViewIfNeeded();
    const changes = { ...capture, name: "54-vendor-work-changes-light-1440x900" };
    await saveScreenshot(page, changes.name);
    await record(page, changes, "vendor-work-change-request");
  } finally {
    await context.close();
  }
}

async function captureVendorWorkReviewMobile(width, name, state) {
  const capture = { name, route: "#work/matters/matter-gaid-change", title: "Work", fixture: "vendor-work-submitted", state, theme: "light", viewport: { width, height: width === 390 ? 844 : 800 }, touch: true, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await openVendorWorkResponse(page);
    await page.getByRole("heading", { name: "Vendor response", exact: true }).scrollIntoViewIfNeeded();
    const answerLayout = await page.locator(".vendor-work-answer-list > div").first().evaluate((element) => {
      const style = getComputedStyle(element);
      return { display: style.display, columns: style.gridTemplateColumns.split(" ").filter(Boolean).length };
    });
    if (answerLayout.display !== "grid" || answerLayout.columns !== 1) throw new Error(`${capture.name} does not stack answer, value and provenance for narrow review`);
    await assertVendorWorkContained(page, capture.name);
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, capture.state);
  } finally {
    await context.close();
  }
}

async function captureVendorWorkHistory() {
  const capture = { name: "57-vendor-work-accepted-history-light-1440x900", route: "#work/matters/matter-gaid-change", title: "Work", fixture: "vendor-work-accepted", state: "vendor-work-accepted-history", theme: "light", viewport: { width: 1440, height: 900 }, density: "comfortable" };
  const { context, page } = await openVendorWorkTarget(capture);
  try {
    await page.getByText("Request history", { exact: true }).waitFor({ state: "visible" });
    await page.getByText("Accepted", { exact: true }).waitFor({ state: "visible" });
    await page.getByText("The response and current assurance report address this request.", { exact: false }).waitFor({ state: "visible" });
    if (await page.getByRole("button", { name: /Accept|Request changes|Cancel request|Retry delivery/ }).count()) throw new Error("Accepted vendor work exposes a material command");
    await page.getByText("Request history", { exact: true }).scrollIntoViewIfNeeded();
    await assertNoHorizontalOverflow(page, capture.name);
    await saveScreenshot(page, capture.name);
    await record(page, capture, "vendor-work-accepted-history");
  } finally {
    await context.close();
  }
}
