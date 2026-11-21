import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "ui-evidence-home-insights");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:4176";
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const records = [];
try {
  for (const width of [1440, 1280, 1024, 768, 390]) for (const theme of ["light", "dark"]) {
    const context = await browser.newContext({ viewport: { width, height: width === 390 ? 844 : 900 }, colorScheme: theme, reducedMotion: "reduce", hasTouch: width === 390 });
    await context.addInitScript((value) => localStorage.setItem("clearsight.theme", value), theme);
    const page = await context.newPage();
    await page.goto(`${base}/?fixture=home-shell-insights#oversight`, { waitUntil: "domcontentloaded" });
    const risks = page.getByRole("button", { name: /Outside appetite: 4.*50%.*4 of 8 risks/ });
    const indicators = page.getByRole("button", { name: /Indicator breaches: 3.*50%.*3 of 6 linked indicators/ });
    await risks.waitFor();
    await indicators.waitFor();
    if (!(await page.getByRole("button", { name: /Reporting period/ }).textContent()).includes("Date range")) throw new Error(`Date range is missing at ${width}/${theme}`);
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
    if (overflow > 1) throw new Error(`Overflow at ${width}/${theme}: ${overflow}`);
    await page.screenshot({ path: path.join(output, `overview-${theme}-${width}.png`), fullPage: true });
    if ([1440, 390].includes(width)) {
      await page.getByRole("tab", { name: "Attention" }).click();
      await page.getByRole("heading", { name: "Issues needing attention" }).waitFor();
      const critical = page.getByText("Critical", { exact: true });
      if (!(await critical.getAttribute("class"))?.includes("cs-tone--error")) throw new Error(`Critical priority styling is missing at ${width}/${theme}`);
      await page.screenshot({ path: path.join(output, `attention-${theme}-${width}.png`), fullPage: true });
    }
    records.push({ width, theme, overflow, percentages: true });
    await context.close();
  }
  await writeFile(path.join(output, "geometry.json"), JSON.stringify({ fixture: "synthetic-domain-values", records }, null, 2));
  console.log(`Captured ${records.length} synthetic Home states.`);
} finally { await browser.close(); }
