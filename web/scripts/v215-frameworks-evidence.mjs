import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-frameworks');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({headless:true,...(process.env.PLAYWRIGHT_BROWSER_PATH ? {executablePath:process.env.PLAYWRIGHT_BROWSER_PATH} : {})});
try {
 for (const width of [1440, 1280, 1024, 768, 390]) {
  const page = await browser.newPage({viewport:{width,height:900}, reducedMotion:'reduce'});
  await page.goto(process.env.PAGE_URL ?? 'http://127.0.0.1:43911/evidence/#/frameworks');
  await page.getByRole('heading', {name:'Framework library'}).waitFor();
  const item = page.getByRole('button', {name:/Sample security framework/});
  await item.waitFor();
  assert.equal(await item.getAttribute('aria-pressed'), 'true');
  await page.getByRole('heading', {name:'Sample security framework'}).waitFor();
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
  await page.screenshot({path:path.join(output, `after-${width}.png`),fullPage:true});
  await item.focus(); await page.keyboard.press('Enter');
  assert.equal(await item.getAttribute('aria-pressed'), 'true');
  await page.close();
 }
 console.log('Synthetic Framework library evidence passed at five widths.');
} finally { await browser.close(); }
