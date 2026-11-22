import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-vendor-checklist');
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
        if (!['/api/v1/vendors/sample-vendor/assessments/current', '/api/v1/vendor-assessments/sample-assessment', '/api/v1/vendor-assessments/sample-assessment/collection'].includes(url.pathname)) return response;
        const value = await response.json();
        if (value.assessment) Object.assign(value.assessment, { tenant_id: 'sample-bank', legal_entity_id: 'sample-entity', status: 'UNDER_REVIEW' });
        if (url.pathname.endsWith('/sample-assessment')) value.documents = [{ field_id: 'sample-document-field', request_id: 'sample-request', artifact_id: 'sample-artifact', file_name: 'Audit report.pdf', media_type: 'application/pdf', size_bytes: 100, artifact_status: 'AVAILABLE', document_type: 'AUDIT', status: 'SUBMITTED', evidence_class: 'VENDOR_SUPPLIED' }];
        if (url.pathname.endsWith('/collection')) value.fields = [{ field_id: 'sample-document-field', label: 'Audit report', type: 'vendor_document', collection_state: 'RECEIVED', bank_review_state: 'PENDING' }];
        return new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
      };
      location.hash = '#/vendors/sample-vendor';
    });
    await page.getByRole('button', { name: 'Review Audit report', exact: true }).click();
    await page.getByLabel('Decision').selectOption('VALIDATE');
    await page.getByLabel('Document source').selectOption('VENDOR_SUPPLIED');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => { document.activeElement?.blur(); window.scrollTo(0, 0); });
    await page.screenshot({ path: path.join(output, `review-${width}.png`), fullPage: true });
    await page.close();
  }
} finally { await browser.close(); }
