import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const base = process.env.PAGE_URL ?? "http://127.0.0.1:5181";
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? "../docs/evidence/2026-11-21-report-recovery");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
async function select(page) { await page.getByRole("button", { name: "View Sample risk report" }).click(); await page.getByRole("button", { name: "Generate report" }).waitFor(); }
try {
  const parallel = await browser.newContext(); const a = await parallel.newPage(), b = await parallel.newPage();
  await Promise.all([a.goto(`${base}/report-recovery.html`), b.goto(`${base}/report-recovery.html`)]);
  await Promise.all([select(a), select(b)]);
  await a.getByRole("button", { name: "Generate report" }).click();
  await b.getByRole("button", { name: "Generate report" }).click();
  await a.getByText("Report requested.").waitFor();
  await b.getByText("Another report request is in progress.").waitFor();
  await b.getByRole("button", { name: "Generate report" }).click();
  await b.getByText("Report requested.").waitFor();
  assert.equal(await a.evaluate(() => localStorage.getItem("sample-report-creates")), "1");
  assert.equal(await a.evaluate(() => localStorage.getItem("sample-report-calls")), "1");
  await parallel.close();

  const recovery = await browser.newContext(); const page = await recovery.newPage();
  await page.goto(`${base}/report-recovery.html?drop=1`); await select(page);
  await page.getByRole("button", { name: "Generate report" }).click();
  await page.getByText("Report request status unknown. Check recent reports before requesting another.").waitFor();
  await page.goto(`${base}/report-recovery.html`);
  await page.getByRole("button", { name: "Retry report request" }).click();
  await page.getByText("Report requested.").waitFor();
  assert.equal(await page.evaluate(() => localStorage.getItem("sample-report-creates")), "1");
  assert.equal(await page.evaluate(() => localStorage.getItem("sample-report-calls")), "2");
  await page.screenshot({ path: path.join(output, "recovered.png"), fullPage: true });
  await recovery.close();
  await writeFile(path.join(output, "results.json"), JSON.stringify({ evidence: "SYNTHETIC_BROWSER_TRANSPORT_NOT_BACKEND_AUTHORIZATION", parallelCreates: 1, recoveryCreates: 1, recoveryCalls: 2 }, null, 2));
  console.log("Cross-tab and reload recovery evidence passed.");
} finally { await browser.close(); }
