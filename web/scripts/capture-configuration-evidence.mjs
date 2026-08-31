import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const configurationDir = path.join(outputDir, "configuration");
const browser = await chromium.launch({ headless: true });
const results = [];

await mkdir(configurationDir, { recursive: true });

const scenarios = [
  { name: "configuration-overview-desktop-light-1440x900", route: "#configure", viewport: { width: 1440, height: 900 }, theme: "light", expectation: "overview" },
  { name: "configuration-authority-desktop-dark-1440x900", route: "#configure/authority", viewport: { width: 1440, height: 900 }, theme: "dark", expectation: "authority" },
  { name: "configuration-access-desktop-light-1440x900", route: "#configure/access", viewport: { width: 1440, height: 900 }, theme: "light", expectation: "access" },
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
    await page.goto(`${baseURL}/?tour=off${scenario.route}`, { waitUntil: "networkidle" });
    await page.getByRole("heading", { name: "Configuration", exact: true }).waitFor({ state: "visible" });
    if (browserErrors.length) throw new Error(`${scenario.name} emitted browser errors: ${browserErrors.join(" | ")}`);

    if (scenario.expectation === "overview" || scenario.expectation === "mobile") await assertOverview(page, scenario.name);
    if (scenario.expectation === "authority") await assertAuthority(page, scenario.name);
    if (scenario.expectation === "access") await assertAccessUnavailable(page, scenario.name);
    if (scenario.expectation === "mobile") await assertMobileShell(page, scenario.name);
    await assertNoHorizontalOverflow(page, scenario.name);

    await page.screenshot({ path: path.join(configurationDir, `${scenario.name}.png`), fullPage: false, animations: "disabled", caret: "hide" });
    results.push({ name: scenario.name, route: scenario.route, viewport: scenario.viewport, theme: scenario.theme, expectation: scenario.expectation });
  } finally {
    await context.close();
  }
}

async function assertOverview(page, name) {
  await page.getByRole("heading", { name: "Control plane", exact: true }).waitFor({ state: "visible" });
  const overview = page.locator(".configure-area-list");
  for (const label of ["People & access", "Authority & routing", "Data & integrations", "Automation", "AI governance", "System operations"]) {
    await overview.getByRole("button", { name: new RegExp(`^${label}\\b`, "i") }).waitFor({ state: "visible" });
  }
  if (await page.getByRole("dialog").count()) throw new Error(`${name} opens a mutation dialog before the administrator chooses an action`);
}

async function assertAuthority(page, name) {
  await page.getByRole("heading", { name: "Authority & routing", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("heading", { name: "Governance policies and delegations", exact: true }).waitFor({ state: "visible" });
  if (await page.getByRole("dialog").count()) throw new Error(`${name} renders governance creation before an explicit action`);
}

async function assertAccessUnavailable(page, name) {
  await page.getByRole("heading", { name: "People & access", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("heading", { name: "Enterprise access unavailable", exact: true }).waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Retry", exact: true }).waitFor({ state: "visible" });
  if (await page.getByRole("dialog").count()) throw new Error(`${name} opens a mutation dialog while access administration is unavailable`);
  const selected = page.getByRole("navigation", { name: "Configuration areas" }).getByRole("button", { name: /^People & access\b/i });
  if (!(await selected.getAttribute("aria-current"))) throw new Error(`${name} loses its selected Configuration domain when access data is unavailable`);
}

async function assertMobileShell(page, name) {
  const mobileNav = page.getByRole("navigation", { name: "Mobile navigation" });
  await mobileNav.waitFor({ state: "visible" });
  if (await mobileNav.getByRole("button", { name: /Configure/i }).count()) throw new Error(`${name} puts Configuration in the daily mobile navigation`);
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth }));
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
}
