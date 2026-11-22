import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-matter-responses');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await page.goto('http://127.0.0.1:43122/evidence/?fixture=matter-work', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === '/api/v1/forms/responses' && url.searchParams.get('subject_id') === 'sample-issue') return Promise.resolve(new Response(JSON.stringify({ items: window.emptyMatterResponses ? [] : [{ id: 'sample-response', title: 'Access owner confirmation', subject_type: 'MATTER', subject_id: 'sample-issue', revision: 1, current: true, completed_at: '2026-11-20T10:00:00Z', submitted_by_display_name: 'Ada Okafor' }] }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
        return original(input, init);
      };
      location.hash = '#/work/issues/sample-issue';
    });
    const link = page.getByRole('link', { name: 'View Access owner confirmation' });
    await link.waitFor();
    assert.equal(await link.getAttribute('href'), '#/forms/responses/sample-response');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `responses-${width}.png`), fullPage: true });
    await page.evaluate(() => { window.emptyMatterResponses = true; location.hash = '#/home'; });
    await page.getByRole('heading', { name: 'Overview' }).waitFor();
    await page.evaluate(() => { location.hash = '#/work/issues/sample-issue'; });
    await page.getByText('No current responses available to you for this issue.').waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `empty-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
