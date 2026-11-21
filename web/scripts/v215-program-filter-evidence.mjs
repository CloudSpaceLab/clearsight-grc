import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-21-v215-program-filter');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await page.goto(process.env.PAGE_URL ?? 'http://127.0.0.1:43116/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === '/api/v1/program-summaries') {
          window.programFilterRequest = url.searchParams.get('overall_state');
          const common = { program_version: 5, assessed_program_version: 5, projection_version: 2, projection_stale: false, state_generated_at: '2026-11-20T10:00:00Z', requirement_count: 8, open_matter_count: 1 };
          const current = { ...common, overall_state: 'CURRENT', state_label: 'Up to date', program: { id: 'access', name: 'Sample access program', code: 'P-01', version: 5 } };
          const incomplete = { ...common, overall_state: 'EVIDENCE_INSUFFICIENT', state_label: 'Evidence incomplete', program: { id: 'vendor', name: 'Sample vendor program', code: 'P-02', version: 5 } };
          const items = url.searchParams.get('overall_state') === 'EVIDENCE_INSUFFICIENT' ? [incomplete] : [current, incomplete];
          return new Response(JSON.stringify({ items, generated_at: '2026-11-20T10:00:00Z' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return original(input, init);
      };
      location.hash = '#/programs';
    });
    await page.getByRole('link', { name: 'Sample access program' }).waitFor();
    await page.screenshot({ path: path.join(output, `programs-before-${width}.png`), fullPage: true });
    await page.getByLabel('Assessed status').selectOption('EVIDENCE_INSUFFICIENT');
    await page.getByRole('button', { name: 'Apply filters' }).click();
    await page.getByText('1 loaded', { exact: true }).waitFor();
    assert.equal(await page.evaluate(() => window.programFilterRequest), 'EVIDENCE_INSUFFICIENT');
    assert.equal(await page.getByRole('link', { name: 'Sample access program' }).count(), 0);
    assert.equal(await page.getByRole('link', { name: 'Sample vendor program' }).count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    for (const locator of [page.getByLabel('Search programs'), page.getByLabel('Assessed status'), page.getByRole('button', { name: 'Apply filters' })]) {
      const box = await locator.boundingBox();
      assert.ok(box && box.height >= 40);
    }
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `programs-filtered-${width}.png`), fullPage: true });
    await page.evaluate(() => {
      const original = window.fetch;
      let fail = true;
      window.fetch = (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (fail && url.pathname === '/api/v1/program-summaries') {
          fail = false;
          return Promise.resolve(new Response(JSON.stringify({ error: { code: 'summary_unavailable', message: 'Programs unavailable.' } }), { status: 503, headers: { 'Content-Type': 'application/json' } }));
        }
        return original(input, init);
      };
      location.hash = '#/forms';
    });
    await page.getByRole('heading', { name: 'Forms & assessments' }).waitFor();
    await page.evaluate(() => { location.hash = '#/programs?overall_state=CURRENT'; });
    await page.getByRole('alert').getByText('Programs unavailable.').waitFor();
    assert.equal(await page.getByLabel('Assessed status').count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `programs-unavailable-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Clear filters' }).click();
    await page.getByText('2 loaded', { exact: true }).waitFor();
    results.push({ width, overflow: false, server_filter: 'EVIDENCE_INSUFFICIENT', loaded: 1 });
    await page.close();
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ fixture: 'SYNTHETIC_NOT_BACKEND_AUTHORIZATION', results }, null, 2));
  console.log(`Program filter verified at ${results.length} widths.`);
} finally { await browser.close(); }
