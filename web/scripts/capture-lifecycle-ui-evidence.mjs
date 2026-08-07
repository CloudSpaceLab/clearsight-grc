import { chromium } from "playwright";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const baseURL = process.env.PAGE_URL ?? "http://127.0.0.1:4173";
const outputDir = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence");
const manifestPath = path.join(outputDir, "manifest.json");
const browser = await chromium.launch({ headless: true });

try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: "light", reducedMotion: "reduce", locale: "en-NG", timezoneId: "Africa/Lagos" });
  await context.addInitScript(() => {
    localStorage.setItem("clearsight.theme", "light");
    localStorage.setItem("clearsight.density", "comfortable");
  });
  const page = await context.newPage();
  const browserErrors = [];
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });

  await page.goto(`${baseURL}/?tour=off&fixture=today-lifecycle#today`, { waitUntil: "networkidle" });
  await page.getByRole("heading", { name: "Today", exact: true }).waitFor({ state: "visible" });
  await page.getByText("Confirm restored ATM availability", { exact: true }).waitFor({ state: "visible" });
  await page.getByText("NDPC incident response", { exact: true }).waitFor({ state: "visible" });
  await page.getByText("External response", { exact: true }).waitFor({ state: "visible" });
  if (browserErrors.length) throw new Error(`lifecycle Today emitted browser errors:\n${browserErrors.join("\n")}`);
  if (await page.getByText("External approval", { exact: true }).count()) throw new Error("lifecycle Today still describes external representation as approval");
  if (await page.getByText("Recommended action", { exact: true }).count()) throw new Error("lifecycle work was relabelled as a recommendation without a governed recommendation record");
  if (await page.getByText("Prepared next step", { exact: true }).count()) throw new Error("lifecycle work was relabelled as prepared work without a receipt");

  const detail = page.locator("details.intervention-verification");
  if (await detail.count() !== 1) throw new Error("expected exactly one governed verification detail");
  if (await detail.evaluate((element) => element.open)) throw new Error("verification detail should be collapsed by default");
  await assertNoHorizontalOverflow(page, "33-today-lifecycle-collapsed-light-1440x900");
  await assertFirstActionVisible(page, 900, "33-today-lifecycle-collapsed-light-1440x900");
  await page.screenshot({ path: path.join(outputDir, "33-today-lifecycle-collapsed-light-1440x900.png"), fullPage: false, animations: "disabled", caret: "hide" });
  await appendRecord(page, "33-today-lifecycle-collapsed-light-1440x900", "lifecycle-work-collapsed");

  await detail.locator("summary").click();
  await page.getByText("ATM remains available for one hour after restoration.", { exact: true }).waitFor({ state: "visible" });
  await page.getByText("Independent outcome review", { exact: true }).waitFor({ state: "visible" });
  await assertNoHorizontalOverflow(page, "34-today-lifecycle-expanded-light-1440x900");
  await page.screenshot({ path: path.join(outputDir, "34-today-lifecycle-expanded-light-1440x900.png"), fullPage: false, animations: "disabled", caret: "hide" });
  await appendRecord(page, "34-today-lifecycle-expanded-light-1440x900", "lifecycle-work-expanded");

  await context.close();
} catch (error) {
  await recordFailure(error);
  throw error;
} finally {
  await browser.close();
}

async function assertNoHorizontalOverflow(page, name) {
  const metrics = await layoutMetrics(page);
  if (metrics.scrollWidth > metrics.clientWidth + 1) throw new Error(`${name} has horizontal overflow: ${metrics.scrollWidth}px content in ${metrics.clientWidth}px viewport`);
}

async function assertFirstActionVisible(page, viewportHeight, name) {
  const action = page.locator(".intervention-next .primary-button").first();
  await action.waitFor({ state: "visible" });
  const box = await action.boundingBox();
  if (!box || box.y < 0 || box.y + box.height > viewportHeight) throw new Error(`${name} does not keep the first Today action inside the initial viewport`);
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
    activeElement: document.activeElement instanceof HTMLElement ? document.activeElement.textContent?.trim().slice(0, 80) ?? document.activeElement.tagName : "unknown",
  }));
}

async function appendRecord(page, name, state) {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  manifest.captures.push({ name, route: "#today", fixture: "today-lifecycle", state, viewport: { width: 1440, height: 900 }, theme: "light", density: "comfortable", metrics: await layoutMetrics(page) });
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
