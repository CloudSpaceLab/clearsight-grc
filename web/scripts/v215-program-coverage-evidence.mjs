import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-program-coverage');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await page.goto('http://127.0.0.1:43122/evidence/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        const response = await original(input, init);
        if (url.pathname !== '/api/v1/programs/sample-program') return response;
        const value = await response.json();
        value.requirements.push({ id: 'sample-requirement-2', code: 'R-02', title: 'Review access changes', statement: 'Review changes to privileged access.', status: 'APPROVED' });
        value.program.calculation_policy = { active: { number: 1, checksum: 'sample', approver_id: 'sample-reviewer', approved_at: '2026-11-01T00:00:00Z', definition: { revision: 'v1' } } };
        value.current_state = { generated_at: '2026-11-20T10:00:00Z', program_version: window.coverageStale ? 2 : 1, projection_version: 4, calculation: { policy_revision: 'v1', state: 'PROVISIONAL', categories: [{ code: 'REQUIREMENTS', eligible_count: 2, assessed_count: 1, unknown_count: 1, excluded_count: 0 }] } };
        if (window.coverageUnavailable) delete value.program.calculation_policy;
        return new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
      };
      location.hash = '#/programs/sample-program';
    });
    await page.getByText('1 of 2 requirements assessed').waitFor();
    assert.equal(await page.getByText('50%', { exact: true }).count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `current-${width}.png`), fullPage: true });
    await page.evaluate(() => { window.coverageStale = true; location.hash = '#/home'; });
    await page.getByRole('heading', { name: 'Overview' }).waitFor();
    await page.evaluate(() => { location.hash = '#/programs/sample-program'; });
    await page.getByText('Out of date', { exact: true }).waitFor();
    await page.screenshot({ path: path.join(output, `stale-${width}.png`), fullPage: true });
    await page.evaluate(() => { window.coverageUnavailable = true; location.hash = '#/home'; });
    await page.getByRole('heading', { name: 'Overview' }).waitFor();
    await page.evaluate(() => { location.hash = '#/programs/sample-program'; });
    await page.getByText('Coverage unavailable.').waitFor();
    assert.equal(await page.getByRole('progressbar', { name: 'Assessed requirements' }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `unavailable-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
