import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? '../.codex-tmp/vendor-review-panels');
const base = process.env.PAGE_URL ?? 'http://127.0.0.1:4179';
const cases = [
  ['start', '', 'Start due diligence'],
  ['start-error', 'vendor-requests-error', 'Start due diligence'],
  ['prepare', 'vendor-collection-prepare', 'Prepare request'],
  ['reissue', 'vendor-collecting', 'Send another link'],
  ['clarification', 'vendor-collection', 'Request clarification'],
  ['finding', 'vendor-collection', 'Record finding'],
  ['document', 'vendor-collection', 'Review document'],
  ['conclusion', 'vendor-collection', 'Record assessment conclusion'],
  ['cancel', 'vendor-collection', 'Cancel assessment'],
].filter(([name]) => !process.env.UI_CASES || process.env.UI_CASES.split(',').includes(name));
const results = [];
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  for (const [name, fixture, action] of cases) for (const theme of (process.env.UI_THEMES ?? 'light,dark').split(',')) for (const width of (process.env.UI_WIDTHS ?? '1440,390,320').split(',').map(Number)) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce', hasTouch: width < 1000 });
    await context.addInitScript((value) => localStorage.setItem('clearsight.theme', value), theme);
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on('pageerror', (error) => errors.push(error.message));
    const record = { name, fixture, theme, width, errors };
    try {
      await page.goto(`${base}/?tour=off&fixture=${fixture}#vendors`, { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: /Acme Processing Limited.*Card transaction processing/ }).click();
      if (name.startsWith('start')) {
        const disclosures = page.locator('.vendor-request-details, .vendor-linked-work');
        await disclosures.first().waitFor();
        record.disclosures = await disclosures.evaluateAll((elements) => elements.map((element) => ({ label: element.querySelector('summary').textContent, open: element.open })));
        if (record.disclosures.length !== 2 || record.disclosures.some((item) => item.open)) throw new Error('Selected vendor secondary requests/work must begin collapsed.');
        if (name === 'start') {
          await page.getByText('Sample · Complete the due diligence review before activation.', { exact: true }).waitFor();
          if ((await disclosures.first().textContent()).includes('Requests · 1 on this page') === false) throw new Error('Normal selected vendor fixture did not load its scoped request.');
        } else {
          await page.getByText('Vendor form requests could not be loaded. Try again.', { exact: true }).waitFor();
          await page.getByText('Activation checks could not be loaded. The relationship remains unchanged.', { exact: true }).waitFor();
        }
        await page.screenshot({ path: path.join(output, `selected-vendor-${theme}-${width}.png`), fullPage: true });
        for (const disclosure of await disclosures.all()) {
          await disclosure.locator(':scope > summary').click();
          if (!await disclosure.evaluate((element) => element.open)) throw new Error('Vendor secondary disclosure did not expand.');
        }
        await page.screenshot({ path: path.join(output, `selected-vendor-expanded-${theme}-${width}.png`), fullPage: true });
        for (const disclosure of await disclosures.all()) await disclosure.locator(':scope > summary').click();
      }
      const scope = page.locator('.vdd-workspace');
      if (name === 'document') {
        await page.locator('.vendor-checklist').getByRole('article', { name: 'ISO 27001 assurance', exact: true }).getByRole('button', { name: action, exact: true }).click();
      } else {
        await scope.getByRole('button', { name: action, exact: true }).click();
      }
      const panel = page.locator('.vdd-panel');
      await panel.waitFor();
      await panel.scrollIntoViewIfNeeded();
      await page.evaluate(() => document.fonts.ready);
      record.layout = await page.evaluate(() => ({ viewport: innerWidth, scrollWidth: document.documentElement.scrollWidth }));
      record.controls = await panel.locator('input,textarea,button').evaluateAll((elements) => elements.map((element) => ({ tag: element.tagName, type: element.getAttribute('type'), disabled: element.matches(':disabled'), text: element.textContent?.trim() })));
      await page.addScriptTag({ path: require.resolve('axe-core/axe.min.js') });
      record.axe = await page.evaluate(async () => {
        const result = await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa'] } });
        return { violations: result.violations, incomplete: result.incomplete };
      });
      record.file = `${name}-${theme}-${width}.png`;
      await panel.screenshot({ path: path.join(output, record.file) });
      if (width < 1000) {
        const actionButton = panel.locator('button').last();
        await actionButton.evaluate((element) => element.scrollIntoView({ block: 'center' }));
        record.actionVisible = await actionButton.evaluate((element) => {
          const rect = element.getBoundingClientRect();
          const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
          return Boolean(hit && (hit === element || element.contains(hit)));
        });
        if (!record.actionVisible) throw new Error('Panel final action is covered after scrolling it into view.');
        await page.screenshot({ path: path.join(output, `${name}-${theme}-${width}-actions-viewport.png`) });
      }
      record.status = 'CAPTURED';
    } catch (error) {
      record.status = 'ERROR';
      record.error = error.message;
      await page.screenshot({ path: path.join(output, `${name}-${theme}-${width}-error.png`), fullPage: true }).catch(() => {});
    }
    results.push(record);
    await writeFile(path.join(output, 'manifest.json'), JSON.stringify({ generatedAt: new Date().toISOString(), results }, null, 2));
    console.log(JSON.stringify({ name, theme, width, status: record.status, error: record.error, violations: record.axe?.violations.length, incomplete: record.axe?.incomplete.length, layout: record.layout }));
    await context.close();
  }
} finally { await browser.close(); }
if (results.some((result) => result.status !== 'CAPTURED' || result.errors.length || result.axe?.violations.length || result.layout.scrollWidth > result.layout.viewport + 1)) process.exitCode = 1;
