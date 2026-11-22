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
        if (url.pathname === '/api/v1/programs/sample-program/operations') return new Response(JSON.stringify({ program_id: 'sample-program', program_version: 1, authority_available: true, operations: [{ command: 'program.requirement.add', can_act: true }, { command: 'program.requirement.review.propose', can_act: true }, { command: 'program.applicability.decide', can_act: true }, { command: 'program.requirement.supersede', subresource_id: 'sample-requirement', can_act: true }] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        if (url.pathname === '/api/v1/programs/sample-program/requirement-reviews') {
          const items = window.programReviewPagedFixture && !url.searchParams.has('after_id') ? Array.from({ length: 100 }, (_, index) => ({ id: `sample-review-${String(index).padStart(3, '0')}`, tenant_id: 'sample-bank', program_id: 'sample-program', requirement_id: 'sample-draft', proposed_by: 'sample-person', proposal_rationale: 'Review this requirement.', status: 'REJECTED', version: 2 })) : [];
          return new Response(JSON.stringify({ program_id: 'sample-program', program_version: 1, items, next_after_id: items.length ? items[99].id : '' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (url.pathname === '/api/v1/programs/sample-program') {
          const response = await original(input, init);
          const value = await response.json();
          value.program = { ...value.program, tenant_id: 'sample-bank', legal_entity_id: 'sample-entity', owner_principal_id: 'sample-person', authority_principal_id: 'sample-reviewer' };
          value.requirements[0] = { ...value.requirements[0], actor: 'Service owner', action: 'review', object: 'privileged access' };
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
    await page.getByRole('button', { name: 'Send Sample draft for review' }).click();
    assert.equal(await page.getByLabel('Reason for review').count(), 1);
    assert.equal(await page.getByRole('button', { name: 'Send for review', exact: true }).isDisabled(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `review-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Cancel' }).click();
    await add.click();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    for (const label of ['Requirement code', 'Requirement title', 'Requirement text', 'Source and section', 'Obligation strength', 'Who must act?', 'What must they do?', 'What does it apply to?', 'Effective date']) assert.equal(await page.getByLabel(label).count(), 1, label);
    assert.equal(await page.getByRole('button', { name: 'Save requirement' }).isDisabled(), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `add-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Cancel' }).click();
    await page.getByRole('button', { name: 'Edit Review access' }).click();
    assert.equal(await page.getByText('Service owner').count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `edit-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Cancel' }).click();
    await page.getByRole('button', { name: 'Set applicability for Review access' }).click();
    assert.equal(await page.getByLabel('Why is this not applicable?').count(), 0);
    await page.getByLabel('Does this apply?').selectOption('NOT_APPLICABLE');
    assert.equal(await page.getByLabel('Why is this not applicable?').count(), 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `not-applicable-${width}.png`), fullPage: true });
    await page.evaluate(() => { window.programReviewPagedFixture = true; location.hash = '#/home'; });
    await page.getByRole('heading', { name: 'Overview' }).waitFor();
    await page.evaluate(() => { location.hash = '#/programs/sample-program'; });
    await page.getByRole('button', { name: 'Load more reviews' }).waitFor();
    assert.equal(await page.getByRole('button', { name: 'Send Sample draft for review' }).count(), 0);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, `review-incomplete-${width}.png`), fullPage: true });
    await page.getByRole('button', { name: 'Load more reviews' }).click();
    await page.getByRole('button', { name: 'Send Sample draft for review' }).waitFor();
    results.push({ width, overflow: false, source: 'SYNTHETIC' });
    await page.close();
    const reviewer = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' });
    await reviewer.addInitScript(() => {
      let delegate = window.fetch.bind(window);
      Object.defineProperty(window, 'fetch', { configurable: true, get() { return delegate; }, set(original) {
        delegate = async (input, init) => {
          const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.origin);
          const json = (value) => new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } });
          if (url.pathname === '/api/v1/context') return json({ tenant: { id: 'sample-bank', name: 'Sample Bank' }, legal_entity: { id: 'sample-entity', name: 'Sample Bank' }, actor: { id: 'sample-reviewer', name: 'Sample Reviewer', kind: 'PERSON' }, mode: 'sample' });
          if (url.pathname === '/api/v1/programs/sample-program') return json({ program: { id: 'sample-program', tenant_id: 'sample-bank', legal_entity_id: 'sample-entity', owner_principal_id: 'sample-person', authority_principal_id: 'sample-reviewer', code: 'P-01', name: 'Sample bank assurance', status: 'ACTIVE', version: 1 }, requirements: [{ id: 'sample-draft', code: 'R-02', title: 'Sample draft', statement: 'Review the sample procedure.', status: 'DRAFT' }], applicability: [], framework_adoptions: [] });
          if (url.pathname === '/api/v1/programs/sample-program/operations') return json({ program_id: 'sample-program', program_version: 1, authority_available: true, operations: [{ command: 'program.requirement.review.approve', can_act: true }, { command: 'program.requirement.review.reject', can_act: true }] });
          if (url.pathname === '/api/v1/programs/sample-program/requirement-reviews') return json({ program_id: 'sample-program', program_version: 1, next_after_id: '', items: [{ id: 'sample-review', tenant_id: 'sample-bank', program_id: 'sample-program', requirement_id: 'sample-draft', proposed_by: 'sample-person', proposal_rationale: 'Review this requirement.', status: 'PENDING', version: 1 }] });
          return original(input, init);
        };
      } });
    });
    await reviewer.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:43118/'}#/programs/sample-program`, { waitUntil: 'domcontentloaded' });
    await reviewer.getByRole('button', { name: 'Review Sample draft' }).click();
    assert.equal(await reviewer.getByLabel('Review reason').count(), 1);
    assert.equal(await reviewer.getByRole('button', { name: 'Approve requirement' }).isDisabled(), true);
    assert.equal(await reviewer.getByRole('button', { name: 'Reject requirement' }).isDisabled(), true);
    assert.equal(await reviewer.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    await reviewer.evaluate(() => window.scrollTo(0, 0));
    await reviewer.screenshot({ path: path.join(output, `decision-${width}.png`), fullPage: true });
    await reviewer.close();
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ fixture: 'SYNTHETIC_NOT_BACKEND_AUTHORIZATION', results }, null, 2));
  console.log(`Requirement authoring verified at ${results.length} widths.`);
} finally { await browser.close(); }
