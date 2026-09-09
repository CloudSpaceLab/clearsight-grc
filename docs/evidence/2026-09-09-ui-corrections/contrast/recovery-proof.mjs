import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { inspectRenderedContrast } from '../../../../web/scripts/rendered-contrast.mjs';
const { chromium } = createRequire(import.meta.url)('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out = path.join(path.dirname(fileURLToPath(import.meta.url)), 'recovery-interaction');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ['light', 'dark']) for (const width of [1440, 320]) {
    const context = await browser.newContext({ viewport: { width, height: width === 320 ? 844 : 900 }, reducedMotion: 'reduce' });
    await context.addInitScript((theme) => localStorage.setItem('clearsight.theme', theme), theme);
    const page = await context.newPage();
    await page.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:4179'}/?tour=off&fixture=forms-response-history#forms`, { waitUntil: 'networkidle' });
    await page.evaluate(() => {
      const fixtureFetch = globalThis.fetch;
      globalThis.assessmentFailureCount = 0;
      globalThis.fetch = (input, init) => {
        const url = typeof input === 'string' ? input : input.url ?? input.toString();
        if (/\/forms\/responses\/[^/]+\/assessment(?:\?|$)/.test(url)) {
          globalThis.assessmentFailureCount++;
          return Promise.resolve(new Response(JSON.stringify({ message: 'Response assessment is temporarily unavailable. Try again.' }), { status: 503, headers: { 'Content-Type': 'application/json' } }));
        }
        return fixtureFetch(input, init);
      };
    });
    await page.getByRole('tab', { name: 'Responses', exact: true }).click();
    await page.getByRole('button', { name: 'Review Vendor due diligence review response' }).click();
    await page.getByRole('button', { name: 'Reload assessment', exact: true }).waitFor();
    const automaticResult = await page.getByRole('region', { name: 'Automatic result', exact: true }).innerText();
    assert.match(automaticResult, /86/);
    assert.match(automaticResult, /14 concern points/);
    const failureCount = await page.evaluate(() => globalThis.assessmentFailureCount);
    assert.ok(failureCount > 0);
    const documents = page.getByRole('button', { name: 'View submitted documents', exact: true });
    assert.equal(await documents.isEnabled(), true);
    await documents.scrollIntoViewIfNeeded();
    await page.screenshot({ path: path.join(out, `assessment-unavailable-${theme}-${width}.png`) });
    await documents.click();
    await page.locator('.document-list-footer').waitFor();
    const documentCount = await page.locator('.document-list-footer > span').first().innerText();
    assert.match(documentCount, /^[1-9]\d* files on this page/);
    await page.screenshot({ path: path.join(out, `documents-after-assessment-failure-${theme}-${width}.png`) });
    results.push({ theme, width, failureCount, automaticResult, documentCount, documents: await page.evaluate(inspectRenderedContrast), layout: await page.evaluate(() => ({ clientWidth: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth })) });
    await context.close();
  }
  await writeFile(path.join(out, 'receipt.json'), JSON.stringify({ generatedAt: new Date().toISOString(), results }, null, 2));
} finally { await browser.close(); }
