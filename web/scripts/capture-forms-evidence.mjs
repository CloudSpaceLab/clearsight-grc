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
  });
  try {
    await context.addInitScript(({ selectedTheme }) => {
      localStorage.setItem("clearsight.theme", selectedTheme);
      localStorage.setItem("clearsight.density", "comfortable");
    }, { selectedTheme: scenario.theme });
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
    await page.goto(scenarioURL(scenario), { waitUntil: "networkidle" });
    if (scenario.zoom === 2) await page.evaluate(() => { document.documentElement.style.zoom = "2"; });
    await scenario.run(page);
    await page.evaluate(() => document.fonts?.ready);
    if (errors.length) throw new Error(`${scenario.name} emitted browser errors:\n${errors.join("\n")}`);
    await verifyAndCapture(page, scenario);
  } finally {
    await context.close();
  }
}

function scenarioURL(scenario) {
  const fixture = encodeURIComponent(scenario.fixture);
  if (scenario.route === "/capture") return `${baseURL}/capture?fixture=${fixture}&capture_invite=task22-${fixture}`;
  return `${baseURL}/?tour=off&fixture=${fixture}${scenario.route}`;
}

async function verifyAndCapture(page, scenario) {
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
