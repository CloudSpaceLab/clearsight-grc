import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-program-creation');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await page.goto(process.env.PAGE_URL ?? 'http://127.0.0.1:43120/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        const json = (value) => new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/programs/setup-candidates') return json({ owner_candidates: [{ id: 'sample-owner', display_name: 'Ada Okafor', kind: 'PERSON' }], approval_authority_candidates: [{ id: 'sample-checker', display_name: 'Bola Adebayo', kind: 'PERSON' }], has_more: false });
        if (url.pathname === '/api/v1/programs' && init?.method === 'POST') {
          const body = JSON.parse(String(init.body)); window.programCreationBody = body;
          return json({ program: { id: 'created-program', tenant_id: 'sample-bank', legal_entity_id: 'sample-entity', code: body.code, name: body.name, type: body.type, owning_function: body.owning_function, owner_principal_id: body.owner_candidate_id, authority_principal_id: body.approval_authority_candidate_id, scope: body.scope, status: 'DRAFT', version: 1 }, requirements: [], applicability: [] });
        }
        return original(input, init);
      };
      location.hash = '#/programs';
    });
    await page.getByRole('link', { name: 'Create program' }).waitFor();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `register-${width}.png`), fullPage: true });
    await page.getByRole('link', { name: 'Create program' }).click();
    await page.getByLabel('Program name').waitFor();
    assert.equal(await page.getByRole('button', { name: 'Create program' }).isDisabled(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `create-${width}.png`), fullPage: true });
    await page.getByLabel('Program name').fill('Sample access review');
    await page.getByLabel('Program code').fill('ACCESS');
    await page.getByLabel('Program type').selectOption('COMPLIANCE');
    await page.getByLabel('Owning function').fill('Risk');
    await page.getByLabel('Scope', { exact: true }).fill('Access reviews');
    await page.getByLabel('Program owner').selectOption('sample-owner');
    await page.getByLabel('Approval authority').selectOption('sample-checker');
    await page.getByRole('button', { name: 'Create program' }).click();
    await page.getByText('Program created.').waitFor();
    assert.equal(await page.getByRole('link', { name: 'Open program' }).getAttribute('href'), '#/programs/created-program');
    const body = await page.evaluate(() => window.programCreationBody);
    assert.equal(body.legal_entity_id, 'sample-entity'); assert.equal(body.owner_candidate_id, 'sample-owner'); assert.equal(body.approval_authority_candidate_id, 'sample-checker'); assert.equal('actor_id' in body, false);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.screenshot({ path: path.join(output, `created-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, created: 'SYNTHETIC' }); await page.close();
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ fixture: 'SYNTHETIC_NOT_BACKEND_AUTHORIZATION', results }, null, 2));
  console.log(`Program creation verified at ${results.length} widths.`);
} finally { await browser.close(); }
