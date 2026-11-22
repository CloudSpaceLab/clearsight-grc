import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-risk-indicator-drill');
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
        const json = (value) => new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/monitoring-checks/sample-indicator/results') return json({ items: [2, 1].map((value) => ({ id: `sample-result-${value}`, program_id: 'sample-program', monitoring_check_id: 'sample-indicator', monitoring_check_version: 4, evaluated_at: `2026-11-${18 + value}T10:00:00Z`, evaluation: { band: value === 2 ? 'HIGH' : 'LOW', coverage: 1, measurement: { field: 'failed', unit: 'COUNT', value: String(value), limits: [{ operator: 'LESS_OR_EQUAL', expected: '1' }], condition: value === 2 ? 'BREACHED' : 'WITHIN' }, rule_results: [{ field_id: 'failed', reason: value === 2 ? 'Two backups failed.' : 'Backups completed.' }] } })) });
        if (url.pathname === '/api/v1/risks/sample-risk') return json({ risk: { id: 'sample-risk', code: 'R-01', name: 'Service disruption', version: 2, status: 'ACTIVE', category: 'Operational', statement: 'Service may be unavailable.', impact: 'Delayed payments.' }, assessments: [], control_details_complete: true, indicator_details_complete: true, indicator_details: [{ link: { id: 'link-1', risk_id: 'sample-risk', risk_version: 2, program_id: 'sample-program', monitoring_check_id: 'sample-indicator', monitoring_check_version: 4 }, program_id: 'sample-program', program_name: 'Continuity', check_id: 'sample-indicator', check_version: 4, check_code: 'KRI-1', check_name: 'Sample backup failures', state: 'UNKNOWN', reason: 'Latest monitoring result is stale.', denominator: 100, score: 80 }] });
        return original(input, init);
      };
      location.hash = '#/risks/sample-risk';
    });
    await page.getByRole('button', { name: 'View Sample backup failures' }).click();
    const dialog = page.getByRole('dialog', { name: 'Sample backup failures' });
    await dialog.getByRole('table', { name: 'Sample backup failures evaluation history' }).waitFor();
    assert.equal(await dialog.getByText('Range: 1–2').count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.equal(await dialog.evaluate((element) => element.scrollWidth <= element.clientWidth), true);
    await page.screenshot({ path: path.join(output, `history-${width}.png`), fullPage: true });
    await dialog.getByRole('button', { name: /View evaluation for/ }).first().click();
    assert.equal(await dialog.getByText('Two backups failed.').count(), 1);
    assert.equal(await dialog.locator('.v-indicator-evaluation').evaluate((element) => document.activeElement === element && element.getBoundingClientRect().top < innerHeight), true);
    await page.screenshot({ path: path.join(output, `evaluation-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
