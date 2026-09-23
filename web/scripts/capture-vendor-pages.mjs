import { chromium } from 'playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const out = path.resolve('../docs/evidence/2026-09-10-vendor-pages');
await mkdir(out, { recursive: true });
const baseline = process.argv.includes('--before');
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const theme of baseline ? ['light'] : ['light', 'dark']) for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 1000 }, colorScheme: theme, reducedMotion: 'reduce' });
    await page.addInitScript(value => localStorage.setItem('clearsight.theme', value), theme);
    await page.goto('http://localhost:18090/?tour=off&fixture=vendor-form-assessment#vendors');
    await page.getByRole('group', { name: 'Open findings', exact: true }).getByText('5', { exact: true }).waitFor();
    async function capture(state) {
      const file = `${state}-${theme}-${width}.png`;
      await page.screenshot({ path: path.join(out, file), fullPage: true });
      const dimensions = await page.evaluate(() => ({ overflow: document.documentElement.scrollWidth > innerWidth + 1, heights: [...document.querySelectorAll('.vendor-metric')].map(e => e.getBoundingClientRect().height) }));
      if (dimensions.overflow) throw new Error(`Overflow: ${file}`);
      results.push({ file, ...dimensions });
    }
    await capture(baseline ? 'before' : 'dashboard');
    if (!baseline) {
      if (await page.locator('.vendor-register').count()) throw new Error('Register remains on dashboard');
      await page.getByRole('link', { name: 'Register', exact: true }).click();
      await page.getByRole('searchbox', { name: 'Search vendors and services' }).waitFor();
      if (await page.locator('.vendor-portfolio').count()) throw new Error('Dashboard remains on register');
      await capture('register');
      await page.locator('.vendor-row').first().click();
      await page.getByRole('button', { name: 'Back to vendor register', exact: true }).waitFor();
      await page.getByRole('group', { name: 'Open findings', exact: true }).getByText('5', { exact: true }).waitFor();
      await capture('detail');
      await page.getByRole('button', { name: 'Back to vendor register', exact: true }).click();
      await page.getByRole('searchbox', { name: 'Search vendors and services' }).waitFor();
      if (!page.url().endsWith('#vendors/register')) throw new Error('Incorrect back route');
      await page.getByRole('link', { name: 'Dashboard', exact: true }).click();
      await page.getByRole('region', { name: 'Vendor portfolio metrics' }).waitFor();
      await page.goBack();
      await page.getByRole('searchbox', { name: 'Search vendors and services' }).waitFor();
      await page.goForward();
      await page.getByRole('region', { name: 'Vendor portfolio metrics' }).waitFor();
      for (const state of ['empty', 'error']) {
        await page.goto(`http://localhost:18090/?tour=off&fixture=vendor-form-assessment-${state}#vendors`);
        await page.getByRole('group', { name: 'Open findings', exact: true }).getByText(state === 'empty' ? '0' : 'Unknown', { exact: true }).waitFor();
        if (state === 'empty') await page.getByText('No linked findings match this filter for the loaded services.').waitFor();
        else await page.getByRole('button', { name: 'Retry findings' }).waitFor();
        await capture(`dashboard-${state}`);
        await page.getByRole('link', { name: 'Register', exact: true }).click();
        await page.getByRole('searchbox', { name: 'Search vendors and services' }).waitFor();
        await capture(`register-${state}`);
      }
    }
    await page.close();
  }
  await writeFile(path.join(out, baseline ? 'before.json' : 'manifest.json'), JSON.stringify(results, null, 2));
  console.log(JSON.stringify(results));
} finally { await browser.close(); }
