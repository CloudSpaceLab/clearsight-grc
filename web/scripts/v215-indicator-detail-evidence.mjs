import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-indicator-detail');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    const pageErrors = [];
    page.on('pageerror', (error) => pageErrors.push(error.message));
    await page.goto('http://127.0.0.1:43122/evidence/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        const json = (value) => new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/monitoring-checks/sample-indicator/results') return json({ items: [2, 1].map((value) => ({ id: `sample-result-${value}`, program_id: 'sample-program', monitoring_check_id: 'sample-indicator', monitoring_check_version: 4, evaluated_at: `2026-11-${18 + value}T10:00:00Z`, evaluation: { band: value === 2 ? 'HIGH' : 'LOW', coverage: 1, measurement: { field: 'failed', unit: 'COUNT', value: String(value), limits: [{ operator: 'LESS_OR_EQUAL', expected: window.__incompatibleIndicator && value === 1 ? '5' : '1' }], condition: value === 2 ? 'BREACHED' : 'WITHIN' }, rule_results: [{ field_id: 'failed', reason: value === 2 ? 'Two backups failed.' : 'Backups completed.' }] } })) });
        const response = await original(input, init);
        if (url.pathname !== '/api/v1/risk-indicators') return response;
        const value = await response.json();
        Object.assign(value.items[0].indicator, { check_version: 4, result_id: 'sample-result-2', evaluated_at: '2026-11-20T10:00:00Z', owner_display_name: 'Ada Okafor', reviewer_display_name: 'Amina Bello', denominator: 100, score: 80, intervention: { matter_id: 'sample-issue', reference: 'ISS-001', status: 'ASSESSMENT', priority: 3, created_at: '2026-11-20T10:00:00Z' } });
        value.items[0].indicator.native_measurement.field = 'failed';
        Object.assign(value.items[0].indicator.link, { program_id: 'sample-program', monitoring_check_id: 'sample-indicator', monitoring_check_version: 4 });
        return json(value);
      };
      location.hash = '#/risks/indicators';
    });
    await page.getByRole('button', { name: 'View Sample backup failures' }).click();
    const dialog = page.getByRole('dialog', { name: 'Sample backup failures' });
    await dialog.getByRole('table', { name: 'Sample backup failures evaluation history' }).waitFor();
    assert.equal(await dialog.getByText('Range: 1–2').count(), 1);
    assert.equal(await dialog.getByRole('link', { name: 'View issue ISS-001' }).getAttribute('href'), '#/work/issues/sample-issue');
    assert.equal(await dialog.getByText('Issue status: Assessment').count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.deepEqual(pageErrors, []);
    assert.equal(await dialog.evaluate((element) => element.scrollWidth <= element.clientWidth), true);
    await page.screenshot({ path: path.join(output, `history-${width}.png`), fullPage: true });
    await dialog.getByRole('button', { name: /View evaluation for/ }).first().click();
    assert.equal(await dialog.getByText('Two backups failed.').count(), 1);
    assert.equal(await dialog.locator('.v-indicator-evaluation').evaluate((element) => document.activeElement === element && element.getBoundingClientRect().top < innerHeight), true);
    await page.screenshot({ path: path.join(output, `evaluation-${width}.png`), fullPage: true });
    await dialog.getByRole('button', { name: 'Close sample backup failures' }).click();
    await page.evaluate(() => { window.__incompatibleIndicator = true; });
    await page.getByRole('button', { name: 'View Sample backup failures' }).click();
    await dialog.getByRole('table', { name: 'Sample backup failures evaluation history' }).waitFor();
    assert.equal(await dialog.getByRole('img', { name: /evaluation trend/ }).count(), 0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `incomparable-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
