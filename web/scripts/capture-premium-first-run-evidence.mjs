import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { deflateSync } from "node:zlib";
import { chromium } from "playwright";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const coverPath = path.resolve("..", "docs", "presentation-assets", "clearsight-premium-first-run-cover.png");
const browser = await chromium.launch({ headless: true });
const websiteLogo = createLogoPNG("website");
const approvedLogo = createLogoPNG("approved");

await mkdir(outputDir, { recursive: true });
await mkdir(path.dirname(coverPath), { recursive: true });

try {
  await captureTodayIntroductions();
  await captureVendorIntroductions();
  await captureVendorBrandGallery();
  await captureVendorIdentityWorkflows();
  await capturePresentationCover();
} catch (error) {
  await recordFailure(error);
  throw error;
} finally {
  await browser.close();
}

async function captureTodayIntroductions() {
  const captures = [
    { name: "58-premium-today-intro-dark-1440x900", theme: "dark", viewport: { width: 1440, height: 900 }, state: "premium-today-intro-dark" },
    { name: "59-premium-today-intro-light-1440x900", theme: "light", viewport: { width: 1440, height: 900 }, state: "premium-today-intro-light" },
    { name: "60-premium-today-intro-dark-tablet-1024x768", theme: "dark", viewport: { width: 1024, height: 768 }, state: "premium-today-intro-tablet-1024", touch: true },
    { name: "61-premium-today-intro-light-tablet-768x900", theme: "light", viewport: { width: 768, height: 900 }, state: "premium-today-intro-tablet-768", touch: true },
    { name: "62-premium-today-intro-dark-mobile-390x844", theme: "dark", viewport: { width: 390, height: 844 }, state: "premium-today-intro-mobile-390", touch: true },
    { name: "63-premium-today-intro-light-reflow-320x800", theme: "light", viewport: { width: 320, height: 800 }, state: "premium-today-intro-reflow-320", touch: true },
  ];
  for (const capture of captures) {
    const { context, page } = await openPage({ ...capture, route: "#today", tour: "on", reducedMotion: "no-preference" });
    try {
      await assertCinematicGuide(page, "Today guide", capture.name);
      await capturePage(page, capture, "#today");
    } finally {
      await context.close();
    }
  }

  const reduced = { name: "64-premium-today-intro-reduced-motion-1440x900", theme: "dark", viewport: { width: 1440, height: 900 }, state: "premium-today-intro-reduced-motion" };
  const reducedPage = await openPage({ ...reduced, route: "#today", tour: "on", reducedMotion: "reduce" });
  try {
    await assertCinematicGuide(reducedPage.page, "Today guide", reduced.name);
    const animations = await reducedPage.page.locator(".cinematic-guide__scene-layer, .cinematic-guide__scene-focus, .cinematic-guide__content").evaluateAll((elements) => elements.map((element) => getComputedStyle(element).animationName));
    if (animations.some((name) => name !== "none")) throw new Error(`${reduced.name} still animates under reduced motion: ${animations.join(", ")}`);
    await capturePage(reducedPage.page, reduced, "#today");
  } finally {
    await reducedPage.context.close();
  }

  const zoomed = { name: "65-premium-today-intro-200pct-zoom-proxy", theme: "light", viewport: { width: 720, height: 900 }, state: "premium-today-intro-200pct-proxy" };
  const zoomPage = await openPage({ ...zoomed, route: "#today", tour: "on", reducedMotion: "reduce", deviceScaleFactor: 2 });
  try {
    await zoomPage.page.getByRole("button", { name: "Start guide" }).scrollIntoViewIfNeeded();
    await assertNoHorizontalOverflow(zoomPage.page, zoomed.name);
    await zoomPage.page.getByRole("button", { name: "Today", exact: true }).waitFor({ state: "visible" });
    await capturePage(zoomPage.page, zoomed, "#today");
  } finally {
    await zoomPage.context.close();
  }
}

async function captureVendorIntroductions() {
  for (const capture of [
    { name: "66-premium-vendors-intro-populated-dark-1440x900", theme: "dark", viewport: { width: 1440, height: 900 }, state: "premium-vendors-intro-populated" },
    { name: "67-premium-vendors-intro-empty-light-1440x900", theme: "light", viewport: { width: 1440, height: 900 }, state: "premium-vendors-intro-empty", fixture: "vendor-guide-empty" },
  ]) {
    const { context, page } = await openPage({ ...capture, route: "#vendors", tour: "on", reducedMotion: "reduce" });
    try {
      await assertCinematicGuide(page, "Vendor guide", capture.name);
      if (capture.fixture) {
        await page.getByRole("heading", { name: "No vendor relationships found for Meridian Trust Bank Nigeria." }).waitFor({ state: "visible" });
        await page.getByRole("button", { name: "Add vendor" }).waitFor({ state: "visible" });
      } else {
        await page.getByText("Acme Processing Limited", { exact: true }).first().waitFor({ state: "visible" });
      }
      await capturePage(page, capture, "#vendors");
    } finally {
      await context.close();
    }
  }
}

async function captureVendorBrandGallery() {
  const captures = [
    { name: "68-vendor-brand-website-light-1440x900", fixture: "vendor-brand-website", theme: "light", state: "vendor-brand-website-icon", label: "Website icon available", image: true },
    { name: "69-vendor-brand-approved-dark-1440x900", fixture: "vendor-brand-approved", theme: "dark", state: "vendor-brand-approved-logo", label: "Approved logo", image: true },
    { name: "70-vendor-brand-pending-light-1440x900", fixture: "vendor-brand-pending", theme: "light", state: "vendor-brand-pending", label: "Website icon pending" },
    { name: "71-vendor-brand-unavailable-light-1440x900", fixture: "vendor-brand-unavailable", theme: "light", state: "vendor-brand-unavailable", label: "Vendor icon unavailable" },
    { name: "72-vendor-brand-broken-fallback-light-1440x900", fixture: "vendor-brand-broken", theme: "light", state: "vendor-brand-broken-monogram", label: "Website icon available", broken: true },
  ];
  for (const capture of captures) {
    const opened = await openVendor(capture);
    try {
      await opened.page.getByText(capture.label, { exact: true }).last().waitFor({ state: "visible" });
      if (capture.image) await opened.page.getByRole("img", { name: "Acme Processing Limited icon" }).first().waitFor({ state: "visible" });
      if (capture.broken) await opened.page.locator(".vendor-detail-heading .vendor-brand-monogram").waitFor({ state: "visible" });
      await capturePage(opened.page, capture, "#vendors");
    } finally {
      await opened.context.close();
    }
  }
}

async function captureVendorIdentityWorkflows() {
  const staged = { name: "73-vendor-identity-validation-staged-light-1440x900", fixture: "vendor-brand-pending", theme: "light", viewport: { width: 1440, height: 900 }, state: "vendor-identity-validation-and-staged-upload" };
  const opened = await openVendorEditor(staged);
  try {
    await opened.page.getByLabel("Website domain").fill("http://acme.example");
    await opened.page.getByRole("button", { name: "Save vendor details" }).click();
    await opened.page.getByText("Enter a website hostname or full HTTPS URL without credentials, a port or an IP address.", { exact: true }).waitFor({ state: "visible" });
    await stageLogo(opened.page, "approved-logo.png");
    await opened.page.getByText("Selected file is ready to save.", { exact: true }).waitFor({ state: "visible" });
    await capturePage(opened.page, staged, "#vendors");
  } finally {
    await opened.context.close();
  }

  const conflict = { name: "74-vendor-identity-conflict-preserves-entry-light-1440x900", fixture: "vendor-identity-brand-errors", theme: "light", viewport: { width: 1440, height: 900 }, state: "vendor-identity-optimistic-conflict" };
  const conflictPage = await openVendorEditor(conflict);
  try {
    await conflictPage.page.getByLabel("Trading name").fill("Acme Payments Operations");
    await conflictPage.page.getByRole("button", { name: "Save vendor details" }).click();
    await conflictPage.page.getByText("Vendor details changed. Reload the current vendor, then save your entries again.", { exact: true }).waitFor({ state: "visible" });
    if (await conflictPage.page.getByLabel("Trading name").inputValue() !== "Acme Payments Operations") throw new Error(`${conflict.name} discarded the user's current entry`);
    await capturePage(conflictPage.page, conflict, "#vendors");
  } finally {
    await conflictPage.context.close();
  }

  const forbidden = { name: "75-vendor-brand-permission-error-preserves-upload-light-1440x900", fixture: "vendor-identity-brand-errors", theme: "light", viewport: { width: 1440, height: 900 }, state: "vendor-brand-permission-error-preserves-upload" };
  const forbiddenPage = await openVendorEditor(forbidden);
  try {
    await stageLogo(forbiddenPage.page, "approved-logo.png");
    await forbiddenPage.page.getByRole("button", { name: "Use approved logo" }).click();
    await forbiddenPage.page.getByText("Your current role cannot change the approved vendor logo. The selected file is still here.", { exact: true }).waitFor({ state: "visible" });
    await forbiddenPage.page.getByText("approved-logo.png", { exact: false }).waitFor({ state: "visible" });
    await capturePage(forbiddenPage.page, forbidden, "#vendors");
  } finally {
    await forbiddenPage.context.close();
  }

  await captureLogoRemoval({ name: "76-vendor-brand-remove-restores-website-light-1440x900", fixture: "vendor-brand-approved", theme: "light", viewport: { width: 1440, height: 900 }, state: "vendor-brand-remove-restores-website", expected: "Website icon restored.", image: true });
  await captureLogoRemoval({ name: "77-vendor-brand-remove-restores-monogram-light-1440x900", fixture: "vendor-brand-approved-no-discovered", theme: "light", viewport: { width: 1440, height: 900 }, state: "vendor-brand-remove-restores-monogram", expected: "Approved logo removed. The vendor monogram is shown until a website icon is available." });

  const mobile = { name: "78-vendor-identity-mobile-dark-390x844", fixture: "vendor-brand-approved", theme: "dark", viewport: { width: 390, height: 844 }, state: "vendor-identity-mobile", touch: true };
  const mobilePage = await openVendorEditor(mobile);
  try {
    await mobilePage.page.getByRole("heading", { name: "Edit vendor details" }).scrollIntoViewIfNeeded();
    await capturePage(mobilePage.page, mobile, "#vendors");
  } finally {
    await mobilePage.context.close();
  }
}

async function captureLogoRemoval(capture) {
  const opened = await openVendorEditor(capture);
  try {
    await opened.page.getByRole("button", { name: "Remove approved logo" }).click();
    await opened.page.getByText(capture.expected, { exact: true }).waitFor({ state: "visible" });
    if (capture.image) await opened.page.getByRole("img", { name: "Acme Processing Limited icon" }).waitFor({ state: "visible" });
    else await opened.page.locator(".vendor-identity-form-heading .vendor-brand-monogram").waitFor({ state: "visible" });
    await capturePage(opened.page, capture, "#vendors");
  } finally {
    await opened.context.close();
  }
}

async function capturePresentationCover() {
  const capture = { name: "presentation-cover", theme: "dark", viewport: { width: 1600, height: 900 }, state: "presentation-cover" };
  const { context, page } = await openPage({ ...capture, route: "#today", tour: "on", reducedMotion: "reduce" });
  try {
    await assertCinematicGuide(page, "Today guide", "presentation cover");
    if (await page.locator("[role=dialog], .cs-sheet").count()) throw new Error("presentation cover contains an open modal or focused-work panel");
    await page.screenshot({ path: coverPath, fullPage: false, animations: "disabled", caret: "hide" });
    const dimensions = await page.evaluate(() => ({ width: innerWidth, height: innerHeight }));
    if (dimensions.width !== 1600 || dimensions.height !== 900) throw new Error(`presentation cover viewport is ${dimensions.width}x${dimensions.height}`);
  } finally {
    await context.close();
  }
}

async function openVendor(capture) {
  const opened = await openPage({ ...capture, viewport: capture.viewport ?? { width: 1440, height: 900 }, route: "#vendors", tour: "off", reducedMotion: "reduce" });
  await opened.page.getByRole("button", { name: /Acme Processing Limited/ }).first().click();
  await opened.page.getByRole("heading", { name: "Acme Processing Limited", exact: true }).waitFor({ state: "visible" });
  return opened;
}

async function openVendorEditor(capture) {
  const opened = await openVendor(capture);
  await opened.page.getByRole("button", { name: "Edit vendor details" }).click();
  await opened.page.getByRole("heading", { name: "Edit vendor details", exact: true }).waitFor({ state: "visible" });
  await opened.page.getByRole("button", { name: "Return to relationship" }).waitFor({ state: "visible" });
  if (await opened.page.getByRole("button", { name: "Add vendor" }).count()) throw new Error(`${capture.name} exposes Add vendor while editing the shared identity`);
  const focused = await opened.page.getByLabel("Legal name").evaluate((element) => element === document.activeElement);
  if (!focused) throw new Error(`${capture.name} did not focus the first identity field`);
  await assertNoHorizontalOverflow(opened.page, capture.name);
  return opened;
}

async function openPage({ theme, viewport, touch = false, route, fixture, tour, reducedMotion, deviceScaleFactor = 1 }) {
  const context = await browser.newContext({ viewport, deviceScaleFactor, colorScheme: theme, hasTouch: touch, reducedMotion, locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(({ selectedTheme }) => {
    localStorage.setItem("clearsight.theme", selectedTheme);
    localStorage.setItem("clearsight.density", "comfortable");
  }, { selectedTheme: theme });
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().includes("Failed to load resource")) browserErrors.push(message.text());
  });
  await page.route("**/api/v1/vendor-identities/*/brand?version=*", async (route) => {
    const token = new URL(route.request().url()).searchParams.get("version") ?? "";
    if (token.startsWith("broken-")) return route.fulfill({ status: 404, contentType: "application/json", body: "{}" });
    return route.fulfill({ status: 200, contentType: "image/png", headers: { "Cache-Control": "public, max-age=31536000, immutable" }, body: token.startsWith("approved-") ? approvedLogo : websiteLogo });
  });
  const params = new URLSearchParams({ tour });
  if (fixture) params.set("fixture", fixture);
  await page.goto(`${baseURL}/?${params.toString()}${route}`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: route === "#vendors" ? "Vendors" : "Today", exact: true }).first().waitFor({ state: "visible" });
  await page.evaluate(() => document.fonts?.ready);
  if (browserErrors.length) throw new Error(`${route} emitted browser errors:\n${browserErrors.join("\n")}`);
  return { context, page };
}

async function assertCinematicGuide(page, label, name) {
  await page.getByRole("complementary", { name: label }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Start guide" }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Skip for now" }).waitFor({ state: "visible" });
  await assertNoHorizontalOverflow(page, name);
}

async function stageLogo(page, name) {
  await page.getByLabel("Approved logo file").setInputFiles({ name, mimeType: "image/png", buffer: approvedLogo });
}

async function capturePage(page, capture, route) {
  await assertNoHorizontalOverflow(page, capture.name);
  await page.screenshot({ path: path.join(outputDir, `${capture.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  await appendRecord(page, capture, route);
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

async function appendRecord(page, capture, route) {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  manifest.captures.push({ name: capture.name, route, fixture: capture.fixture ?? null, state: capture.state, viewport: capture.viewport ?? page.viewportSize(), theme: capture.theme, density: "comfortable", metrics: await layoutMetrics(page) });
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

function createLogoPNG(kind) {
  const width = 128;
  const height = 128;
  const pixels = Buffer.alloc((width * 4 + 1) * height);
  const background = kind === "approved" ? [54, 45, 125] : [4, 103, 126];
  const accent = kind === "approved" ? [167, 139, 250] : [77, 221, 240];
  for (let y = 0; y < height; y += 1) {
    const row = y * (width * 4 + 1);
    pixels[row] = 0;
    for (let x = 0; x < width; x += 1) {
      const offset = row + 1 + x * 4;
      const dx = x - 64;
      const dy = y - 64;
      const ring = dx * dx + dy * dy > 46 * 46 && dx * dx + dy * dy < 55 * 55;
      const left = x >= 36 && x <= 48 && y >= 34 && y <= 94;
      const right = x >= 80 && x <= 92 && y >= 34 && y <= 94;
      const bar = y >= 56 && y <= 68 && x >= 42 && x <= 86;
      const mark = ring || left || right || bar;
      const color = mark ? accent : background;
      pixels[offset] = color[0];
      pixels[offset + 1] = color[1];
      pixels[offset + 2] = color[2];
      pixels[offset + 3] = 255;
    }
  }
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  return Buffer.concat([signature, pngChunk("IHDR", ihdr), pngChunk("IDAT", deflateSync(pixels)), pngChunk("IEND", Buffer.alloc(0))]);
}

function pngChunk(type, data) {
  const typeBytes = Buffer.from(type);
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length, 0);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(Buffer.concat([typeBytes, data])), 0);
  return Buffer.concat([length, typeBytes, data, crc]);
}

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}
