import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-21-v215-capture-entry');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    let sends = 0;
    await page.route('**/api/v1/evidence/access/**', async route => {
      const endpoint = new URL(route.request().url()).pathname;
      if (endpoint.endsWith('/start')) return route.fulfill({ json: { policy: 'DIRECT_LINK_EMAIL_OTP', expires_at: '2027-01-01T00:00:00Z', recipients: [{ selector_id: 'sample-recipient', hint: 'a***@example.com' }] } });
      if (endpoint.endsWith('/otp/send')) {
        sends++;
        if (sends === 1) return route.fulfill({ status: 503, json: { error: 'unavailable', message: 'Unavailable' } });
        return route.fulfill({ json: { challenge_id: 'sample-challenge', hint: 'a***@example.com', expires_at: '2027-01-01T00:00:00Z' } });
      }
      return route.fulfill({ status: 404, json: { error: 'not_found' } });
    });
    await page.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:43117/'}?view=sample#form_access=sample-invitation`, { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: 'Open request' }).click();
    await page.getByRole('alert').getByText(/Code could not be sent/).waitFor();
    assert.equal(await page.evaluate(() => location.hash), '');
    assert.equal(await page.evaluate(() => location.search), '?view=sample');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `retry-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Send code' }).click();
    await page.getByLabel('Verification code').waitFor();
    assert.equal(sends, 2);
    assert.equal(await page.getByRole('radio').count(), 0);
    assert.equal(await page.evaluate(() => sessionStorage.length), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `code-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, retry: true, masked_recipient: true });
    await page.close();
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ fixture: 'SYNTHETIC_NOT_BACKEND_AUTHORIZATION', results }, null, 2));
  console.log(`Capture entry verified at ${results.length} widths.`);
} finally { await browser.close(); }
