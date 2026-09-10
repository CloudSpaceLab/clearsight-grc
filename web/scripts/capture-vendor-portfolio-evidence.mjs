import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const out = path.resolve('../docs/evidence/2026-09-10-vendor-portfolio');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of ['light', 'dark']) for (const width of [1440, 390, 320]) {
    const context = await browser.newContext({ viewport: { width, height: 1000 }, colorScheme: theme, reducedMotion: 'reduce' });
    await context.addInitScript(value => localStorage.setItem('clearsight.theme', value), theme);
    const page = await context.newPage();
    for (const state of ['live', 'unavailable']) {
      const fixture = state === 'live' ? 'vendor-form-assessment' : 'vendor-form-assessment-error';
      await page.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:18080'}/?tour=off&fixture=${fixture}#vendors`, { waitUntil: 'networkidle' });
      await page.getByRole('region', { name: 'Vendor portfolio metrics' }).waitFor();
      const file = `${state}-${theme}-${width}.png`;
      await page.screenshot({ path: path.join(out, file), fullPage: true });
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
      if (overflow) throw new Error(`Horizontal overflow: ${file}`);
      if (state === 'unavailable' && await page.getByRole('group', { name: 'Overdue forms' }).getByText('Unknown').count() !== 1) throw new Error('Missing summaries represented as known');
      results.push({ file, state, theme, width, overflow });
      if (state === 'live') {
        await page.getByRole('button', { name: 'Review overdue forms', exact: true }).click();
        await page.locator('.vendor-row').first().click();
        await page.getByRole('button', { name: 'Back to vendor register', exact: true }).waitFor();
        if (await page.locator('.vendor-register').isVisible()) throw new Error('Vendor register did not yield to full-width detail');
        await page.screenshot({ path: path.join(out, `detail-${theme}-${width}.png`), fullPage: true });
        await page.getByRole('button', { name: 'Back to vendor register', exact: true }).click();
        await page.getByRole('region', { name: 'Vendor portfolio metrics' }).waitFor();
      }
    }
    await context.close();
  }
  await writeFile(path.join(out, 'manifest.json'), JSON.stringify(results, null, 2));
  console.log(`${results.length} portfolio captures plus six detail/back-navigation checks passed.`);
} finally { await browser.close(); }
