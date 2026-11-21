import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../docs/evidence/2026-11-22-v215-requirement-authoring');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.PLAYWRIGHT_BROWSER_PATH ? { executablePath: process.env.PLAYWRIGHT_BROWSER_PATH } : {}) });
const results = [];
try {
  for (const width of [1440, 768, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await page.goto(process.env.PAGE_URL ?? 'http://127.0.0.1:43118/', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      const original = window.fetch;
      window.fetch = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
        if (url.pathname === '/api/v1/programs/sample-program/operations') return new Response(JSON.stringify({ program_id: 'sample-program', program_version: 1, authority_available: true, operations: [{ command: 'program.requirement.add', can_act: true }, { command: 'program.applicability.decide', can_act: true }, { command: 'program.requirement.supersede', subresource_id: 'sample-requirement', can_act: true }] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/programs/sample-program') {
          const response = await original(input, init);
          const value = await response.json();
          value.requirements.push({ id: 'sample-draft', code: 'R-02', title: 'Sample draft', statement: 'Review the sample procedure.', status: 'DRAFT' });
          return new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return original(input, init);
      };
      location.hash = '#/programs/sample-program';
    });
    const add = page.getByRole('button', { name: 'Add requirement' });
    await add.waitFor();
    assert.equal(await page.getByText('Draft · Review required').count(), 1);
    assert.equal(await page.getByRole('button', { name: 'Set applicability for Sample draft' }).count(), 0);
    await page.screenshot({ path: path.join(output, `before-${width}.png`), fullPage: true });
    await add.click();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    for (const label of ['Requirement code', 'Requirement title', 'Requirement text', 'Source and section', 'Obligation strength', 'Who must act?', 'What must they do?', 'What does it apply to?', 'Effective date']) assert.equal(await page.getByLabel(label).count(), 1, label);
    assert.equal(await page.getByRole('button', { name: 'Save requirement' }).isDisabled(), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `add-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Cancel' }).click();
    await page.getByRole('button', { name: 'Set applicability for Review access' }).click();
    assert.equal(await page.getByLabel('Why is this not applicable?').count(), 0);
    await page.getByLabel('Does this apply?').selectOption('NOT_APPLICABLE');
    assert.equal(await page.getByLabel('Why is this not applicable?').count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `not-applicable-${width}.png`), fullPage: true });
    results.push({ width, overflow: false, source: 'SYNTHETIC' });
    await page.close();
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ fixture: 'SYNTHETIC_NOT_BACKEND_AUTHORIZATION', results }, null, 2));
  console.log(`Requirement authoring verified at ${results.length} widths.`);
} finally { await browser.close(); }
