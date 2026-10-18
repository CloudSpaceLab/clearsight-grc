import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const out = path.resolve('../docs/evidence/2026-09-10-vendor-findings');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ['light', 'dark']) for (const width of [1440, 390]) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce' });
    await context.addInitScript(value => localStorage.setItem('clearsight.theme', value), theme);
    const page = await context.newPage();
    for (const state of ['live', 'unavailable']) {
      const fixture = state === 'live' ? 'vendor-form-assessment' : 'vendor-form-assessment-error';
      await page.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:18080'}/?tour=off&fixture=${fixture}#vendors/overview`, { waitUntil: 'networkidle' });
      await page.getByRole('region', { name: 'Vendor overview summary' }).waitFor();
      if (state === 'live') await page.getByRole('button', { name: '5 open exceptions', exact: true }).waitFor();
      else await page.getByText('Some vendor exceptions could not be checked. Totals unknown.', { exact: false }).waitFor();
      const file = `${state}-${theme}-${width}.png`;
      await page.screenshot({ path: path.join(out, file), fullPage: true });
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
      if (overflow) throw new Error(`Horizontal overflow: ${file}`);
      if (await page.locator('.vendor-metric').count() !== 0) throw new Error(`Legacy metric card rendered: ${file}`);
      if (state === 'unavailable' && await page.getByRole('button', { name: 'Unknown overdue actions' }).count() !== 1) throw new Error('Missing exceptions represented as known');
      if (state === 'live') {
        const cards = page.locator('.vendor-summary-counts > .cs-button');
        const boxes = await Promise.all([0, 1, 2, 3].map(index => cards.nth(index).boundingBox()));
        if (await cards.count() !== 4 || boxes.some(box => !box || box.height < 90)) {
          throw new Error(`Approved Vendor measures not rendered as four cards: ${file}`);
        }
        const [first, second, third, fourth] = boxes;
        if (width === 1440 && !(first.x < second.x && second.x < third.x && third.x < fourth.x &&
            Math.abs(first.y - fourth.y) < 3)) {
          throw new Error(`Vendor cards must form a single desktop row: ${file}`);
        }
        if (width === 390 && !(first.x < second.x && Math.abs(first.y - second.y) < 3 &&
            third.x < fourth.x && third.y > first.y + first.height - 3)) {
          throw new Error(`Vendor cards must form a 2x2 mobile grid: ${file}`);
        }
      }
      results.push({ file, state, theme, width, overflow });
      if (state === 'live') {
        if (await page.getByRole('heading', { name: 'Vendor register', exact: true }).count() !== 0) throw new Error('Register leaked into vendor overview');
        await page.getByRole('navigation', { name: 'Vendor sections' }).getByRole('button', { name: 'Register', exact: true }).click();
        await page.getByRole('heading', { name: 'Vendor register', exact: true }).waitFor();
        if (await page.getByRole('region', { name: 'Vendor overview summary' }).count() !== 0) throw new Error('Portfolio leaked into vendor register');
        if (!page.url().endsWith('#vendors/register')) throw new Error(`Register route not reflected in URL: ${page.url()}`);
        const registerFile = `register-${theme}-${width}.png`;
        await page.screenshot({ path: path.join(out, registerFile), fullPage: true });
        const registerOverflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
        if (registerOverflow) throw new Error(`Horizontal overflow: ${registerFile}`);
        results.push({ file: registerFile, state: 'register', theme, width, overflow: registerOverflow });
        await page.locator('.vendor-row').first().click();
        await page.getByRole('button', { name: 'Back to vendor register', exact: true }).waitFor();
        if (await page.locator('.vendor-register').isVisible()) throw new Error('Vendor register did not yield to full-width detail');
        await page.getByRole('button', { name: '5 open exceptions', exact: true }).waitFor();
        await page.getByText('Superseded', { exact: true }).waitFor();
        await page.screenshot({ path: path.join(out, `detail-${theme}-${width}.png`), fullPage: true });
        await page.getByRole('button', { name: 'Back to vendor register', exact: true }).click();
        await page.getByRole('heading', { name: 'Vendor register', exact: true }).waitFor();
        if (await page.getByRole('region', { name: 'Vendor overview summary' }).count() !== 0) throw new Error('Back navigation left the register route');
        await page.getByRole('navigation', { name: 'Vendor sections' }).getByRole('button', { name: 'Overview', exact: true }).click();
        await page.getByRole('region', { name: 'Vendor overview summary' }).waitFor();
        if (!page.url().endsWith('#vendors/overview')) throw new Error(`Overview route not reflected in URL: ${page.url()}`);
      }
    }
    await context.close();
  }
  await writeFile(path.join(out, 'manifest.json'), JSON.stringify(results, null, 2));
  console.log(`${results.length} overview/register captures plus four detail/back-navigation checks passed.`);
} finally { await browser.close(); }
