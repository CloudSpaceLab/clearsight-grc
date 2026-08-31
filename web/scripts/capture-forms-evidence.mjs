import { chromium } from "playwright";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

import { formsEvidenceScenarios } from "./forms-evidence-scenarios.mjs";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const browser = await chromium.launch({ headless: true });

try {
  for (const scenario of formsEvidenceScenarios) await captureScenario(scenario);
} catch (error) {
  await recordFailure(error);
  throw error;
} finally {
  await browser.close();
}

async function captureScenario(scenario) {
  const context = await browser.newContext({
    viewport: scenario.viewport,
    colorScheme: scenario.theme,
    hasTouch: Boolean(scenario.touch),
    reducedMotion: "reduce",
    locale: "en-NG",
    timezoneId: "Africa/Lagos",
    deviceScaleFactor: scenario.zoom,
  });
  try {
    await context.addInitScript(({ selectedTheme }) => {
      localStorage.setItem("clearsight.theme", selectedTheme);
      localStorage.setItem("clearsight.density", "comfortable");
    }, { selectedTheme: scenario.theme });
    const page = await context.newPage();
    if (scenario.fixture === "forms-recovery-restored") await seedRecoveryEnvelope(page);
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
    await page.goto(scenarioURL(scenario), { waitUntil: "networkidle" });
    const scenarioMetrics = await scenario.run(page);
    await page.evaluate(() => document.fonts?.ready);
    if (errors.length) throw new Error(`${scenario.name} emitted browser errors:\n${errors.join("\n")}`);
    await verifyAndCapture(page, scenario, scenarioMetrics);
  } finally {
    await context.close();
  }
}

async function seedRecoveryEnvelope(page) {
  await page.goto(`${baseURL}/favicon.ico`, { waitUntil: "domcontentloaded" });
  await page.evaluate(async () => {
    const database = await new Promise((resolve, reject) => {
      const request = indexedDB.open("clearsight-capture-recovery", 1);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains("envelopes")) request.result.createObjectStore("envelopes");
        if (!request.result.objectStoreNames.contains("keys")) request.result.createObjectStore("keys");
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
    const origin = window.location.origin;
    const context = { origin, legalEntityID: "entity-demo", distributionID: "field-distribution-1", schemaVersion: 1 };
    const key = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 24 * 60 * 60 * 1000).toISOString();
    const envelope = {
      payloadVersion: 2,
      distributionID: context.distributionID,
      workspaceID: "field-workspace-1",
      schemaVersion: context.schemaVersion,
      serverVersion: 0,
      page: 0,
      presentationMode: "AUTOMATIC",
      basePresentationMode: "AUTOMATIC",
      presentationModeDirty: false,
      edits: [
        { fieldID: "visit_note", baseSequence: 0, operation: "set", value: { text: "Recovered on this device" } },
        { fieldID: "site_photo", baseSequence: 0, operation: "reselect" },
      ],
      complete: false,
      localSequence: 2,
      updatedAt: now.toISOString(),
      expiresAt,
    };
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const aad = new TextEncoder().encode(["clearsight.capture-recovery.v1", context.origin, context.legalEntityID, context.distributionID, String(context.schemaVersion)].join("\n"));
    const ciphertext = await crypto.subtle.encrypt({ name: "AES-GCM", iv, additionalData: aad, tagLength: 128 }, key, new TextEncoder().encode(JSON.stringify(envelope)));
    await new Promise((resolve, reject) => {
      const transaction = database.transaction(["keys", "envelopes"], "readwrite");
      transaction.objectStore("keys").put(key, `capture-recovery-device:${origin}`);
      transaction.objectStore("envelopes").put({ version: 1, algorithm: "AES-GCM", iv: iv.buffer, ciphertext, expiresAt, schemaVersion: 1 }, `capture-recovery-v2|${origin}|entity-demo|field-distribution-1|1`);
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error);
      transaction.onabort = () => reject(transaction.error);
    });
    database.close();
  });
}

function scenarioURL(scenario) {
  const fixture = encodeURIComponent(scenario.fixture);
  if (scenario.route === "/capture") return `${baseURL}/capture?fixture=${fixture}&capture_invite=task22-${fixture}`;
  return `${baseURL}/?tour=off&fixture=${fixture}${scenario.route}`;
}

async function verifyAndCapture(page, scenario, scenarioMetrics) {
  const metrics = await layoutMetrics(page);
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${scenario.name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
  await page.screenshot({ path: path.join(outputDir, `${scenario.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  manifest.captures.push({
    name: scenario.name,
    route: scenario.route,
    fixture: scenario.fixture,
    state: scenario.state,
    viewport: page.viewportSize(),
    theme: scenario.theme,
    density: "comfortable",
    ...(scenarioMetrics ? { scenario_metrics: scenarioMetrics } : {}),
    zoom: scenario.zoom,
    capabilities: [...scenario.capabilities],
    metrics,
  });
  await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
}

function layoutMetrics(page) {
  return page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
    clientHeight: document.documentElement.clientHeight,
    scrollHeight: document.documentElement.scrollHeight,
    scrollY: window.scrollY,
    theme: document.documentElement.dataset.theme ?? "unknown",
    density: document.documentElement.dataset.density ?? "unknown",
    zoom: document.documentElement.style.zoom || "1",
  }));
}

async function recordFailure(error) {
  try {
    const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
    manifest.failure = error instanceof Error ? error.message : String(error);
    await writeFile(manifestPath, JSON.stringify(manifest, null, 2));
  } catch { /* The primary evidence runner owns manifest creation. */ }
}
