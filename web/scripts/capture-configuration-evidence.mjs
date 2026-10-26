import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const configurationDir = path.join(outputDir, "configuration");
const browser = await chromium.launch({ headless: true });
const results = [];

await mkdir(configurationDir, { recursive: true });

const scenarios = [
  { name: "configuration-organization-desktop-light-1440x900", route: "#configure/access/positions", viewport: { width: 1440, height: 900 }, theme: "light", expectation: "organization" },
  { name: "configuration-checks-partial-desktop-dark-1440x900", route: "#configure/authority/checks", fixture: "people-v2-checks-partial", viewport: { width: 1440, height: 900 }, theme: "dark", expectation: "checks" },
  { name: "configuration-overview-desktop-light-1440x900", route: "#configure", viewport: { width: 1440, height: 900 }, theme: "light", expectation: "overview" },
  { name: "configuration-authority-desktop-dark-1440x900", route: "#configure/authority", fixture: "people-v2-current", viewport: { width: 1440, height: 900 }, theme: "dark", expectation: "authority" },
  { name: "configuration-access-desktop-light-1440x900", route: "#configure/access", fixture: "people-v2-current", viewport: { width: 1440, height: 900 }, theme: "light", expectation: "access" },
  { name: "configuration-overview-tablet-dark-1024x768", route: "#configure", viewport: { width: 1024, height: 768 }, theme: "dark", touch: true, expectation: "overview" },
  { name: "configuration-overview-mobile-light-390x844", route: "#configure", viewport: { width: 390, height: 844 }, theme: "light", touch: true, expectation: "mobile" },
  { name: "configuration-overview-reflow-dark-320x800", route: "#configure", viewport: { width: 320, height: 800 }, theme: "dark", touch: true, expectation: "mobile" },
];

try {
  for (const scenario of scenarios) await capture(scenario);
  await writeFile(path.join(outputDir, "configuration-review.json"), JSON.stringify({ status: "PASS", evidenceDirectory: "configuration", scenarios: results }, null, 2));
} catch (error) {
  await writeFile(path.join(outputDir, "configuration-review.json"), JSON.stringify({ status: "FAIL", evidenceDirectory: "configuration", error: error instanceof Error ? error.message : String(error), scenarios: results }, null, 2));
  throw error;
} finally {
  await browser.close();
}

async function capture(scenario) {
  const context = await browser.newContext({
    viewport: scenario.viewport,
    colorScheme: scenario.theme,
    hasTouch: scenario.touch === true,
    reducedMotion: "reduce",
    locale: "en-NG",
    timezoneId: "Africa/Lagos",
  });
  await context.addInitScript(({ theme }) => {
    localStorage.setItem("clearsight.theme", theme);
    localStorage.setItem("clearsight.density", "comfortable");
  }, { theme: scenario.theme });
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });

  try {
    await page.goto(`${baseURL}/?tour=off${scenario.fixture ? `&fixture=${scenario.fixture}` : ""}${scenario.route}`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: ["authority", "access", "organization", "checks"].includes(scenario.expectation) ? "People & access" : "Configuration", exact: true }).waitFor({ state: "visible" });
    if (browserErrors.length) throw new Error(`${scenario.name} emitted browser errors: ${browserErrors.join(" | ")}`);

    if (scenario.expectation === "overview" || scenario.expectation === "mobile") await assertOverview(page, scenario.name);
    if (scenario.expectation === "authority") await assertAuthority(page, scenario.name);
    if (scenario.expectation === "access") await assertPeople(page, scenario.name);
    if (scenario.expectation === "organization") await assertAccessLive(page, scenario.name);
    if (scenario.expectation === "checks") await assertChecks(page, scenario.name);
    if (scenario.expectation === "mobile") await assertMobileShell(page, scenario.name);
    await assertNoHorizontalOverflow(page, scenario.name);

    await page.screenshot({ path: path.join(configurationDir, `${scenario.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
    results.push({ name: scenario.name, route: scenario.route, viewport: scenario.viewport, theme: scenario.theme, expectation: scenario.expectation });
  } finally {
    await context.close();
  }
}

async function assertOverview(page, name) {
  await page.getByRole("heading", { name: "Configuration areas", exact: true }).waitFor({ state: "visible" });
  const overview = page.locator(".configure-area-list");
  for (const label of ["People & access", "Approval routes", "Data & integrations", "Automation", "AI governance", "System operations"]) {
    await overview.getByRole("button", { name: new RegExp(`^${label}\\b`, "i") }).waitFor({ state: "visible" });
  }
  if (await page.getByRole("dialog").count()) throw new Error(`${name} opens a mutation dialog before the administrator chooses an action`);
}

async function assertAuthority(page, name) {
  await page.getByRole("tab", { name: "Approval routes", exact: true }).waitFor();
  await page.locator(".people-governance-layout__records").getByText("Material risk approval", { exact: true }).waitFor();
  if (await page.getByRole("dialog").count()) throw new Error(`${name} renders governance creation before an explicit action`);
}
async function assertPeople(page, name) {
  await page.getByRole("button", { name: "Inspect Amina Yusuf", exact: true }).waitFor();
  await page.getByRole("searchbox", { name: "Search people", exact: true }).waitFor();
  if (await page.locator(".people-directory__records .cs-selectable-record").count() !== 12) throw new Error(`${name} truncates loaded people`);
  if (await page.getByRole("dialog").count()) throw new Error(`${name} opens a mutation before an explicit action`);
}
async function assertChecks(page, name) {
  await page.getByRole("heading", { name: "Routing checks are unavailable", exact: true }).waitFor();
  await page.getByText("Confirm the final DPCO review date", { exact: true }).waitFor();
  if (!await page.getByRole("button", { name: "Retry checks", exact: true }).isEnabled()) throw new Error(`${name} loses routing-check recovery`);
}

async function assertAccessLive(page, name) {
  await page.getByRole("tab", { name: "Organization settings", exact: true }).waitFor();
  await page.getByRole("heading", { name: "Areas", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Add area", exact: true }).waitFor({ state: "visible" });
  await page.getByText("Pending changes", { exact: true }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Approve", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Reject", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("heading", { name: "Positions & roles", exact: true }).waitFor({ state: "visible" });
  if (await page.getByRole("dialog").count()) throw new Error(`${name} opens an organization mutation before the administrator chooses an action`);
  const selected = page.getByRole("tab", { name: "Organization settings", exact: true });
  if (await selected.getAttribute("aria-selected") !== "true") throw new Error(`${name} loses its selected organization destination`);
}

async function assertMobileShell(page, name) {
  const sidebar = page.getByRole("complementary", { name: "Primary navigation" });
  await sidebar.waitFor({ state: "visible" });
  const settings = sidebar.getByRole("link", { name: "Settings", exact: true });
  if (await settings.count() !== 1 || await settings.getAttribute("href") !== "#configure"
      || await settings.getAttribute("aria-current") !== "page") {
    throw new Error(`${name} does not retain the permitted Configuration route`);
  }
  if (await sidebar.locator(".sidebar-secondary").getByRole("link", { name: "Settings", exact: true }).count() !== 1) {
    throw new Error(`${name} does not separate Settings from the primary destinations`);
  }
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
}
