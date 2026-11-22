import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');

const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-vendor-reassessment');
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
        if (url.pathname === '/api/v1/forms/templates') return new Response(JSON.stringify({ items: [{ template: { id: 'sample-form', name: 'Vendor security review', code: 'F-01' }, active_version: 2, active_status: 'ACTIVE' }] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/forms/templates/sample-form/revisions/2') return new Response(JSON.stringify({ id: 'sample-form', tenant_id: 'sample-bank', version: 2, status: 'ACTIVE', is_current: true, fields: [{ id: 'access', label: 'Privileged access reviewed' }, { id: 'audit', label: 'Current audit report' }] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        const response = await original(input, init);
        if (!['/api/v1/vendors/sample-vendor', '/api/v1/vendors/sample-vendor/assessments/current', '/api/v1/vendor-assessments/sample-assessment'].includes(url.pathname)) return response;
        const value = await response.json();
        if (value.relationship) value.relationship.status = 'ACTIVE';
        if (value.assessment) value.assessment.status = 'COMPLETED';
        return new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
      };
      location.hash = '#/vendors/sample-vendor';
    });
    await page.getByRole('button', { name: 'Start reassessment' }).click();
    await page.getByLabel('Assessment form').waitFor();
    await page.getByLabel('Assessment form').selectOption('sample-form');
    await page.getByLabel('Assessment scope').selectOption('FOCUSED');
    await page.getByLabel('Current audit report').check();
    await page.getByLabel('Review type').selectOption('TRIGGERED');
    await page.getByLabel('Review reference').fill('Service change 42');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => { document.activeElement?.blur(); window.scrollTo(0, 0); });
    await page.screenshot({ path: path.join(output, `reassessment-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
