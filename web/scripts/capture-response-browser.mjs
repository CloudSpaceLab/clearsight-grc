import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const stage = process.argv[2] || 'after';
const output = new URL(`../../docs/quality/screenshots/response-browser/${stage}/`, import.meta.url).pathname.replace(/^\/([A-Z]:)/, '$1');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const receipts = [];
try {
  for (const theme of ['light', 'dark']) for (const width of [1440, 390, 320]) for (const surface of ['program', 'forms']) {
    const context = await browser.newContext({ viewport: { width, height: 1000 }, reducedMotion: 'reduce' });
    await context.addInitScript(value => localStorage.setItem('clearsight.theme', value), theme);
    const page = await context.newPage();
    page.on('pageerror', error => console.log(`Page error: ${error.message}`));
    await page.goto(`http://127.0.0.1:4187/?tour=off&fixture=program-responses#${surface === 'program' ? 'programs/program-ndpa/evidence-results' : 'forms?section=responses'}`);
    const heading = page.getByRole('heading', { name: surface === 'program' ? 'Submitted data' : 'Responses', exact: true });
    await heading.waitFor().catch(async error => { console.log((await page.locator('body').innerText()).slice(0, 4000)); throw error; });
    await page.getByRole('button', { name: /Review .* response/ }).first().waitFor();
    if (stage === 'after' && width === 1440) {
      const activeSort = page.getByRole('button', { name: surface === 'program' ? 'Sort by Submitted' : 'Sort by Concern', exact: true });
      await activeSort.click();
      await page.getByRole('button', { name: /Review .* response/ }).first().waitFor();
    }
    await heading.scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${output}/${surface}-${theme}-${width}.png`, fullPage: true });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
    receipts.push({ surface, theme, width, overflow });
    if (overflow) throw new Error(`${surface}/${theme}/${width}: horizontal overflow`);
    if (stage === 'after' && theme === 'light' && width === 1440) {
      await page.getByRole('button', { name: /Review .* response/ }).first().click();
      await page.getByRole('dialog').waitFor();
      await page.screenshot({ path: `${output}/${surface}-review.png`, fullPage: false });
      await page.getByRole('button', { name: 'Close', exact: true }).click();
      await page.getByRole('searchbox', { name: 'Search response titles' }).fill('no matching form');
      await page.getByText('No responses match these filters', { exact: true }).waitFor();
      await page.screenshot({ path: `${output}/${surface}-filtered-empty.png`, fullPage: false });
      await page.getByRole('button', { name: 'Clear response filters', exact: true }).click();
      await page.getByRole('button', { name: /Review .* response/ }).first().waitFor();
      await page.getByText('Columns', { exact: true }).click();
      const submitted = page.getByRole('checkbox', { name: 'Submitted', exact: true });
      await submitted.focus();
      await submitted.press('Space');
      if (await submitted.isChecked()) throw new Error('Submitted column was not hidden');
      await page.screenshot({ path: `${output}/${surface}-columns.png`, fullPage: false });
      await submitted.focus();
      await submitted.press('Space');
      if (!await submitted.isChecked()) throw new Error('Submitted column was not restored');
      await page.getByText('Columns', { exact: true }).click();
      await page.getByLabel('Submitted from', { exact: true }).fill('2026-10-05');
      await page.getByLabel('Submitted until', { exact: true }).fill('2026-10-04');
      await page.getByRole('alert').last().waitFor();
      await page.screenshot({ path: `${output}/${surface}-invalid-date.png`, fullPage: false });
      await page.getByRole('button', { name: 'Clear response filters', exact: true }).click();
      await page.getByRole('button', { name: /Review .* response/ }).first().waitFor();
      // Local evidence transport only: exercise real list-error and retry handling.
      await page.evaluate(() => {
        const original = window.fetch;
        window.fetch = async (input, init) => String(input).includes('/api/v1/forms/responses?')
          ? new Response(JSON.stringify({ error: { message: 'Responses are temporarily unavailable. Retry.' } }), { status: 503, headers: { 'Content-Type': 'application/json' } })
          : original(input, init);
      });
      await page.getByRole('searchbox', { name: 'Search response titles' }).fill('annual');
      await page.getByRole('button', { name: surface === 'program' ? 'Retry submitted data' : 'Try again', exact: true }).waitFor();
      await page.screenshot({ path: `${output}/${surface}-error.png`, fullPage: false });
    }
    await context.close();
  }
} finally { await browser.close(); }
await writeFile(`${output}/receipt.json`, JSON.stringify(receipts, null, 2));
console.log(JSON.stringify(receipts));
