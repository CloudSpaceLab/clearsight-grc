import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-v215-form-order");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PLAYWRIGHT_BROWSER_PATH });
try {
  const results = [];
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
    await page.goto(process.env.PAGE_URL ?? "http://127.0.0.1:43103/", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const native = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === "/api/v1/forms/templates/sample-form") {
          const template = { id: "sample-form", code: "F-01", version: 2, status: "DRAFT", name: "Sample vendor review", purpose: "Review service controls.", scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "general", title: "Questions" }], fields: [{ id: "q1", section_id: "general", label: "Is access reviewed?", type: "yes_no", required: true }, { id: "q2", section_id: "general", label: "When was access reviewed?", type: "date", required: true }] };
          return new Response(JSON.stringify({ template, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] }), { status: 200, headers: { "Content-Type": "application/json" } });
        }
        return native(input, init);
      };
      location.hash = "#/forms/sample-form/edit";
    });
    await page.getByRole("button", { name: "Move question 2: When was access reviewed? up" }).click();
    const focused = await page.evaluate(() => document.activeElement?.getAttribute("id"));
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
    const disabled = await page.getByRole("button", { name: "Move question 1: When was access reviewed? up" }).isDisabled();
    if (focused !== "v-authoring-question-0" || overflow || !disabled) throw new Error(`Invalid form-order state at width ${width}`);
    await page.screenshot({ path: path.join(output, `runtime-${width}.png`), fullPage: true });
    results.push({ width, focused, overflow, disabled });
    await page.close();
  }
  await writeFile(path.join(output, "results.json"), JSON.stringify({ fixture: "synthetic, not backend authorization", results }, null, 2));
  console.log(JSON.stringify(results));
} finally { await browser.close(); }
