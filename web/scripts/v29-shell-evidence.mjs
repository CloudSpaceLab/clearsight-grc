import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const runtimeURL = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
const referenceURL = process.env.REFERENCE_URL ?? "http://127.0.0.1:4175";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence-v29-shell");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const records = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) {
    for (const theme of ["light", "dark"]) {
      const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, colorScheme: theme, hasTouch: width === 390, reducedMotion: "reduce" });
      await context.addInitScript((selectedTheme) => { localStorage.setItem("clearsight.theme", selectedTheme); }, theme);
      const page = await context.newPage();
      await page.goto(`${runtimeURL}/?fixture=home-shell#oversight`, { waitUntil: "domcontentloaded" });
      await page.locator(".oversight-workspace").waitFor();
      await page.getByRole("region", { name: "Assigned work" }).waitFor();
      await page.locator(".oversight-pressure-table").waitFor();
      await page.evaluate(async () => { await document.fonts.ready; await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))); });
      const geometry = await page.evaluate(() => {
        const header = document.querySelector(".app-shell > header.context-bar").getBoundingClientRect();
        const main = document.querySelector(".app-shell > main").getBoundingClientRect();
        const rail = document.querySelector(".sidebar.app-navigation").getBoundingClientRect();
        const assigned = document.querySelector('[aria-labelledby="overview-assigned-heading"]').getBoundingClientRect();
        const analysis = document.querySelector(".oversight-posture-analysis").getBoundingClientRect();
        const lastColumn = document.querySelector(".oversight-pressure-table th:last-child")?.getBoundingClientRect();
        const pressurePanel = document.querySelector(".oversight-pressure-table")?.closest(".oversight-panel")?.getBoundingClientRect();
        const range = document.createRange();
        const lastHeader = document.querySelector(".oversight-pressure-table th:last-child");
        if (lastHeader) range.selectNodeContents(lastHeader);
        const lastHeaderText = lastHeader ? range.getBoundingClientRect() : null;
        return { header: { x: header.x, y: header.y, width: header.width, bottom: header.bottom }, main: { x: main.x, y: main.y }, railWidth: rail.width, overflow: document.documentElement.scrollWidth - innerWidth,
          assigned: { x: assigned.x, y: assigned.y, width: assigned.width }, analysis: { x: analysis.x, right: analysis.right },
          pressureColumnsContained: Boolean(lastColumn && pressurePanel && lastHeaderText && lastColumn.right <= pressurePanel.right + 1 && lastHeaderText.right <= lastColumn.right + 1),
          pressureBounds: { columnRight: lastColumn?.right, panelRight: pressurePanel?.right, textRight: lastHeaderText?.right } };
      });
      const expectedRail = width <= 560 ? 0 : width <= 1024 ? 72 : 228;
      if (geometry.header.x !== 0 || Math.abs(geometry.header.width - width) > 1 || geometry.main.y < geometry.header.bottom - 1 || Math.abs(geometry.railWidth - expectedRail) > 1 || geometry.overflow > 1) throw new Error(`Invalid shell geometry at ${width}/${theme}: ${JSON.stringify(geometry)}`);
      if (width >= 1280 && geometry.assigned.x < geometry.analysis.right - 1) throw new Error(`Assigned work is not beside the overview analysis at ${width}/${theme}`);
      if (width >= 1280 && !geometry.pressureColumnsContained) throw new Error(`Risk pressure columns are clipped at ${width}/${theme}: ${JSON.stringify(geometry.pressureBounds)}`);
      if (width === 390) {
        const opener = page.getByRole("button", { name: "Open navigation" });
        await opener.click();
        await page.getByRole("dialog", { name: "Navigation" }).waitFor();
        await page.keyboard.press("Escape");
        await page.getByRole("dialog", { name: "Navigation" }).waitFor({ state: "hidden" });
        if (!await opener.evaluate((node) => node === document.activeElement)) throw new Error(`Navigation focus did not return at ${width}/${theme}`);
        await opener.blur();
      }
      await page.screenshot({ path: path.join(output, `runtime-${theme}-${width}.png`), fullPage: true });
      records.push({ kind: "runtime", width, theme, geometry });
      if (theme === "light") {
        await page.goto(`${referenceURL}/index.html`, { waitUntil: "domcontentloaded" });
        await page.locator("h1").first().waitFor();
        await page.evaluate(async () => { await document.fonts.ready; });
        await page.screenshot({ path: path.join(output, `reference-light-${width}.png`), fullPage: true });
        records.push({ kind: "reference", width, theme });
      }
      await context.close();
    }
  }
  await writeFile(path.join(output, "geometry.json"), JSON.stringify({ records }, null, 2));
  console.log(`Captured ${records.length} reference and runtime states.`);
} finally { await browser.close(); }
