import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-risk-indicator-linker');
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
        if (url.pathname === '/api/v1/risks/sample-risk') return json({ risk: { id: 'sample-risk', tenant_id: 'bank', legal_entity_id: 'entity', code: 'R-01', name: 'Service disruption', version: 2, status: 'ACTIVE', category: 'Operational', statement: 'Service may be unavailable.', impact: 'Delayed payments.' }, owner_display_name:'Ada Okafor', assessments: [], control_details_complete: true, indicator_details_complete: true, indicator_details: [{ link: { id: 'link-1', risk_id: 'sample-risk', risk_version: 2, program_id: 'sample-program', monitoring_check_id: 'sample-indicator', monitoring_check_version: 4 }, program_id: 'sample-program', program_name: 'Continuity', check_id: 'sample-indicator', check_version: 4, check_code: 'KRI-1', check_name: 'Sample backup failures', state: 'UNKNOWN', reason: 'Latest monitoring result is stale.', denominator: 100, score: 80 }] });
        if (url.pathname === '/api/v1/risks/sample-risk/operations') return json({risk_id:'sample-risk', risk_version:2, can_act:true, authority_available:true});
        if (url.pathname === '/api/v1/program-summaries') return json({items:[{program:{id:'sample-program', tenant_id:'bank', legal_entity_id:'entity', name:'Continuity', status:'ACTIVE'}}]});
        if (url.pathname === '/api/v1/programs/sample-program/monitoring-checks') return json({items:[{id:'check-2', tenant_id:'bank', program_id:'sample-program', version:3, status:'ACTIVE', is_current:true, code:'KRI-2', name:'Backup recovery'}]});
        return original(input, init);
      };
      location.hash = '#/risks/sample-risk';
    });
    await page.getByRole('button', { name: 'Link indicator', exact:true }).click();
    await page.getByRole('button', { name: 'Search programs' }).click();
    await page.locator('.v-risk-linker select').nth(0).selectOption('sample-program');
    await page.locator('.v-risk-linker select').nth(1).selectOption('check-2:3');
    assert.equal(await page.getByRole('button',{name:'Confirm link'}).isEnabled(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.getByRole('heading',{name:'Link indicator'}).scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(output, `link-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
