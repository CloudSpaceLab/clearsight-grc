import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const axePath = require.resolve("axe-core/axe.min.js");
const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");

const scenarios = [
  { id: "group-light-desktop", fixture: "home-t7-group", theme: "light", width: 1440, height: 900, mode: "desktop" },
  { id: "group-dark-desktop", fixture: "home-t7-group", theme: "dark", width: 1440, height: 900, mode: "desktop" },
  { id: "group-dark-mobile", fixture: "home-t7-group", theme: "dark", width: 390, height: 844, mode: "mobile" },
  { id: "group-light-320", fixture: "home-t7-group", theme: "light", width: 320, height: 800, mode: "mobile" },
  { id: "group-dark-200pct-proxy", fixture: "home-t7-group", theme: "dark", width: 720, height: 900, mode: "zoom" },
  { id: "group-partial-light-desktop", fixture: "home-t7-group-partial", theme: "light", width: 1440, height: 900, mode: "partial" },
  { id: "entity-light-desktop", fixture: "oversight", theme: "light", width: 1440, height: 900, mode: "entity" },
  { id: "entity-dark-320", fixture: "oversight", theme: "dark", width: 320, height: 800, mode: "entity" },
];

// Shared Home Tabs use a compact React Aria Select at <=760px.
function primaryTabs(page) {
  return page.locator(".cs-tabs--compact-select").first();
}

function primaryTab(page, label) {
  return primaryTabs(page).locator(":scope > .cs-tabs__list [role=tab]").filter({ hasText: label });
}

async function selectPrimaryTab(page, label) {
  const tab = primaryTab(page, label);
  if (await tab.isVisible()) {
    await tab.click();
    return;
  }
  await primaryTabs(page).locator(":scope > .cs-tabs__compact button").click();
  await page.getByRole("listbox").getByRole("option", { name: label, exact: true }).click();
}

async function primaryTabSelected(page, label) {
  return (await primaryTab(page, label).getAttribute("aria-selected")) === "true";
}

await mkdir(outputDir, { recursive: true });
const browser = await chromium.launch({ headless: true });
const records = [];

for (const item of scenarios) {
  const context = await browser.newContext({
    viewport: { width: item.width, height: item.height },
    colorScheme: item.theme,
    reducedMotion: "reduce",
    hasTouch: item.mode === "mobile",
    locale: "en-NG",
    timezoneId: "Africa/Lagos",
  });
  await context.addInitScript((theme) => {
    localStorage.setItem("clearsight.theme", theme);
    localStorage.setItem("clearsight.density", "comfortable");
  }, item.theme);
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });

  const record = { ...item, failures: [], assertions: [], layout: null, axe: null };
  const assert = (condition, message) => {
    record.assertions.push({ label: message, pass: Boolean(condition) });
    if (!condition) record.failures.push(message);
  };

  try {
    await page.goto(`${baseURL}/?fixture=${item.fixture}`, { waitUntil: "networkidle" });
    if (item.mode === "zoom") await page.evaluate(() => { document.body.style.zoom = "200%"; });
    await page.evaluate(() => document.fonts?.ready);

    if (item.mode === "entity") {
      await page.getByRole("heading", { name: "Home", exact: true }).waitFor({ state: "visible" });
      const tabs = primaryTabs(page).locator(":scope > .cs-tabs__list [role=tab]");
      assert(await tabs.count() === 3, "Legal-entity Home exposes three intent tabs");
      assert(await primaryTabSelected(page, "Oversight"), "Oversight is initial entity tab");
      await selectPrimaryTab(page, "Attention");
      assert(await primaryTabSelected(page, "Attention"), "Attention opens independently");
      await selectPrimaryTab(page, "My work");
      assert(await primaryTabSelected(page, "My work"), "My work opens independently");
      await selectPrimaryTab(page, "Oversight");
    } else {
      await page.getByRole("heading", { name: "Group Home" }).waitFor({ state: "visible" });
      await page.getByRole("button", { name: /Outside appetite: 9/ }).waitFor({ state: "visible" });
      const tabs = primaryTabs(page).locator(":scope > .cs-tabs__list [role=tab]");
      assert(await tabs.count() === 3, "Group Home exposes three intent tabs");
      assert(await primaryTabSelected(page, "Oversight"), "Group Oversight initial tab");
      assert(await page.getByRole("list", { name: "Outside appetite by OpCo" }).isVisible(), "Authorized OpCo risk concentration visible");
      assert(await page.getByRole("button", { name: /Loss events: 5/ }).isVisible(), "Loss is period event count, not cross-currency money");
      if (item.mode === "partial") {
        assert(await page.getByText(/Group risk posture is incomplete/).count() > 0, "Missing OpCo Risk coverage disclosed");
        assert((await page.getByRole("button", { name: /Outside appetite: 9/ }).getAttribute("data-quality")) !== "current", "Partial Group posture is not presented as current/clear");
      }
    }

    record.layout = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
      rootWidth: document.getElementById("root")?.scrollWidth ?? null,
      viewportHeight: window.innerHeight,
    }));
    assert(record.layout.scrollWidth <= record.layout.clientWidth + 1, "No horizontal overflow");

    await page.addScriptTag({ path: axePath });
    record.axe = await page.evaluate(async () => {
      const result = await window.axe.run(document, { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"] } });
      return {
        violations: result.violations.map((v) => ({ id: v.id, impact: v.impact, targets: v.nodes.map((n) => n.target) })),
        incompleteCount: result.incomplete.reduce((sum, v) => sum + v.nodes.length, 0),
      };
    });
    assert(record.axe.violations.length === 0, "No axe A/AA violations");

    const shot = `home-t7-${item.id}.png`;
    await page.screenshot({ path: path.join(outputDir, shot), fullPage: true, animations: "disabled" });
    record.screenshot = shot;

    if (item.mode.startsWith("entity")) {
      // Entity data is an existing illustrative fixture, not a governed source snapshot.
    } else {
      const first = page.getByRole("button", { name: "Open Alpha Bank" });
      await first.focus();
      await page.keyboard.press("Enter");
      assert(await page.locator("main[data-opened-opco='opco-alpha']").count() > 0, "Keyboard opens only the selected authorized OpCo");

      const oversightTab = primaryTab(page, "Oversight");
      if (await oversightTab.isVisible()) {
        await oversightTab.focus();
        await page.keyboard.press("ArrowRight");
        assert(await primaryTabSelected(page, "Attention"), "Right arrow moves to Attention");
        await page.keyboard.press("ArrowRight");
        assert(await primaryTabSelected(page, "My work"), "Right arrow moves to My work");
      } else {
        await selectPrimaryTab(page, "Attention");
        assert(await primaryTabSelected(page, "Attention"), "Compact navigation opens Attention");
        await selectPrimaryTab(page, "My work");
        assert(await primaryTabSelected(page, "My work"), "Compact navigation opens My work");
      }
      await selectPrimaryTab(page, "Attention");
      assert(await page.getByRole("table", { name: "Group OpCo attention" }).isVisible(), "Workflow pressure remains in Attention");
      await selectPrimaryTab(page, "My work");
      await page.getByRole("button", { name: "Open current OpCo My work" }).click();
      assert(await page.locator("main[data-work-handoff='true']").count() > 0, "My work hands off to OpCo work instead of aggregating Group identities");
      await selectPrimaryTab(page, "Oversight");
      await page.getByRole("button", { name: /Reporting period/ }).click();
      await page.getByRole("button", { name: "90 days" }).click();
      await page.getByRole("button", { name: /Loss events: 9/ }).waitFor({ state: "visible" });
      assert(await page.getByRole("button", { name: /Loss events: 9/ }).count() === 1, "Loss period change updates Group counts");
    }
  } catch (error) {
    record.failures.push(error instanceof Error ? error.message : String(error));
  } finally {
    if (errors.length) record.failures.push(...errors.map((value) => `browser: ${value}`));
    record.status = record.failures.length ? "FAIL" : "PASS";
    records.push(record);
    await context.close();
  }
}

await browser.close();
const failed = records.filter((item) => item.status !== "PASS");
const receipt = {
  revision: "home-t7-browser-v1",
  generated_at: new Date().toISOString(),
  coverage: "Deterministic Group and legal-entity fixture rendering, keyboard, axe and 200% CSS-zoom proxy. Not actual CRO comprehension, real browser zoom or production/20k-node performance.",
  scenarios: records,
  status: failed.length ? "FAIL" : "PASS",
};
await writeFile(path.join(outputDir, "home-t7.json"), JSON.stringify(receipt, null, 2));
if (failed.length) {
  for (const item of failed) console.error(`${item.id}: ${item.failures.join("; ")}`);
  process.exitCode = 1;
} else {
  console.log(`Home T7 browser scenarios passed: ${records.length}`);
}
