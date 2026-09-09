import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const output = path.resolve('../docs/evidence/2026-09-09-risk-register-migration');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const results = [];
try {
  for (const state of ['review', 'missing', 'authority', 'ambiguous', 'receipt', 'retry', 'workspace']) for (const theme of ['light', 'dark']) for (const width of [1440, 390]) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, reducedMotion: 'reduce', colorScheme: theme });
    await context.addInitScript((theme) => localStorage.setItem('clearsight.theme', theme), theme);
    const page = await context.newPage();
    const errors = []; page.on('pageerror', (e) => errors.push(e.message));
    await page.goto(`http://localhost:5189/register-migration.html?state=${state}`, { waitUntil: 'networkidle' });
    await page.getByRole('region', { name: 'Risk register migration' }).waitFor();
    if (state === 'workspace') {
      const analysis = page.locator('details').filter({ has: page.locator(':scope > summary', { hasText: 'Review document analysis' }) });
      if (await analysis.count() !== 1 || await analysis.getAttribute('open') !== null) throw new Error('Register analysis must begin collapsed.');
    }
    if (state === 'retry') {
      await page.getByRole('combobox', { name: /^Bank owner for Vendor/ }).selectOption('other');
      await page.getByLabel('I confirm the vendor, owner and finding assignments.').check();
      await page.getByRole('button', { name: 'Import 2 findings' }).click();
      await page.getByRole('alert').waitFor();
    }
    await page.screenshot({ path: path.join(output, `${state}-${theme}-${width}.png`), fullPage: true });
    await page.addScriptTag({ path: require.resolve('axe-core/axe.min.js') });
    const violations = await page.evaluate(async () => (await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa'] } })).violations.map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) })));
    const layout = await page.evaluate(() => ({ viewport: innerWidth, scroll: document.documentElement.scrollWidth }));
    results.push({ state, theme, width, errors, violations, layout });
    if (errors.length || violations.length || layout.scroll > width) throw new Error(JSON.stringify(results.at(-1)));
    if (state === 'retry') { await page.getByRole('button', { name: 'Retry import' }).click(); await page.getByRole('heading', { name: '2 findings imported' }).waitFor(); }
    await context.close();
  }
} finally { await browser.close(); await writeFile(path.join(output, 'results.json'), JSON.stringify(results, null, 2)); }
console.log(`Verified ${results.length} rendered states without accessibility violations or horizontal overflow.`);
