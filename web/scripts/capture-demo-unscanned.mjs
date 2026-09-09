import { createRequire } from 'node:module';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const output = path.resolve(process.env.UI_EVIDENCE_DIR ?? path.join(root, 'docs/evidence/2026-09-09-demo-unscanned'));
const base = process.env.PAGE_URL ?? 'http://127.0.0.1:4187';
const cases = [
  { allowance: 'allowed', state: 'collection' }, { allowance: 'blocked', state: 'collection' },
  { allowance: 'allowed', state: 'picker' }, { allowance: 'blocked', state: 'picker' },
  { allowance: 'allowed', state: 'review' },
];
const selectedState = process.env.CAPTURE_STATE;
const selectedCases = cases.filter((item) => !selectedState || item.state === selectedState);
if (selectedCases.length === 0) throw new Error(`Unknown capture state: ${selectedState}`);
const priorManifest = selectedState ? JSON.parse(await readFile(path.join(output, 'manifest.json'), 'utf8')) : undefined;
const results = priorManifest ? priorManifest.results.filter((item) => item.state !== selectedState) : [];
const assert = (condition, message) => { if (!condition) throw new Error(message); };
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  for (const item of selectedCases) for (const theme of ['light', 'dark']) for (const width of [1440, 390]) {
    const fixture = `vendor-collection-unscanned-${item.allowance}`;
    const name = `${item.allowance}-${item.state}-${theme}-${width}`;
    const result = { ...item, name, fixture, theme, width, capturedAt: new Date().toISOString(), pageErrors: [], protectedContentRequests: [] };
    const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce', hasTouch: width < 1000 });
    await context.addInitScript((value) => localStorage.setItem('clearsight.theme', value), theme);
    const page = await context.newPage();
    page.setDefaultTimeout(12000);
    page.on('pageerror', (error) => result.pageErrors.push(error.message));
    // These fixtures contain placeholder PDF metadata. Never fetch it as real file evidence.
    await page.route('**/api/v1/forms/documents/**/content*', async (route) => {
      result.protectedContentRequests.push(route.request().url());
      await route.abort();
    });
    try {
      await page.goto(`${base}/?tour=off&fixture=${fixture}#vendors`, { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: /Acme Processing Limited.*Card transaction processing/ }).click();
      const dueDiligenceTab = page.getByRole('tab', { name: 'Due diligence', exact: true });
      if (await dueDiligenceTab.isVisible()) await dueDiligenceTab.click();
      else {
        await page.getByRole('button', { name: / Vendor section$/ }).click();
        await page.getByRole('listbox').getByRole('option', { name: 'Due diligence', exact: true }).click();
      }
      const checklist = page.locator('.vendor-checklist');
      const iso = checklist.getByRole('article', { name: 'ISO 27001 assurance', exact: true });
      await iso.waitFor();
      const allowed = item.allowance === 'allowed';
      const reviewButton = iso.getByRole('button', { name: 'Review document', exact: true });
      if (allowed) {
        await iso.getByText('Unscanned · Demo', { exact: true }).waitFor();
        await iso.getByText('Unscanned file. Review is enabled in demo mode.', { exact: true }).waitFor();
        assert(await reviewButton.isEnabled(), 'Allowed unscanned receipt review is disabled.');
      } else {
        await iso.getByText('Missing', { exact: true }).waitFor();
        await iso.getByText('Linked document unavailable. Review its source.', { exact: true }).waitFor();
        assert(await reviewButton.count() === 0, 'Blocked receipt incorrectly offers review.');
        assert(await iso.getByText('Unscanned · Demo', { exact: true }).count() === 0, 'Blocked file incorrectly has demo review labeling.');
      }
      result.receiptReviewEnabled = allowed;
      let surface = checklist;
      let action = allowed ? reviewButton : iso.getByRole('button', { name: 'Use existing document', exact: true });
      if (item.state === 'picker') {
        await checklist.getByRole('article', { name: 'Vulnerability test report', exact: true }).getByRole('button', { name: 'Use existing document', exact: true }).click();
        surface = page.getByRole('dialog', { name: 'Use existing document for Vulnerability test report', exact: true });
        const row = surface.getByRole('row', { name: /Security test report.pdf/ });
        await row.click();
        action = surface.getByRole('button', { name: 'Choose this document', exact: true });
        result.chooseEnabled = await action.isEnabled();
        assert(result.chooseEnabled === allowed, 'Picker permission does not match the explicit allowance.');
        await row.getByText(allowed ? 'Unscanned · Demo' : 'Safety check pending', { exact: true }).waitFor();
        if (!allowed) await surface.getByText('This file is not available for use. Choose an available document.', { exact: true }).waitFor();
        if (width >= 1000) {
          const navigation = surface.getByRole('navigation', { name: 'File types', exact: true });
          result.pickerNavigation = await navigation.evaluate((element) => ({
            clientWidth: element.clientWidth, scrollWidth: element.scrollWidth,
            buttons: [...element.querySelectorAll('button')].map((button) => {
              const label = button.querySelector('.document-kind-label');
              const text = label.children[1].getBoundingClientRect();
              const marker = label.children[2].getBoundingClientRect();
              return { label: label.children[1].textContent, markerGap: marker.left - text.right, height: button.getBoundingClientRect().height };
            }),
          }));
          assert(result.pickerNavigation.buttons.every((button) => button.markerGap >= 7.9 && button.height >= 44), 'File type marker spacing or touch target height regressed.');
          const lastType = navigation.getByRole('button').last();
          await lastType.scrollIntoViewIfNeeded();
          result.pickerNavigation.lastTypeReachable = await lastType.evaluate((element) => {
            const rect = element.getBoundingClientRect();
            const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
            return Boolean(hit && (hit === element || element.contains(hit)));
          });
          assert(result.pickerNavigation.lastTypeReachable, 'The final file type cannot be reached in the horizontal navigation.');
          await navigation.evaluate((element) => { element.scrollLeft = 0; });
        }
      } else if (item.state === 'review') {
        await reviewButton.click();
        surface = page.getByRole('dialog', { name: 'Review document', exact: true });
        await surface.getByText('Unscanned file. Review is enabled in demo mode.', { exact: true }).waitFor();
        assert(!/cannot support an approval|scan complete|demo check complete/i.test(await surface.textContent()), 'Demo review shows a contradictory scan/approval claim.');
        action = surface.getByRole('button', { name: 'Record validation', exact: true });
        assert(await action.isEnabled(), 'Demo review validation action is disabled.');
      }
      await page.evaluate(() => document.fonts.ready);
      await action.evaluate((element) => element.scrollIntoView({ block: 'center' }));
      result.actionReachable = await action.evaluate((element) => {
        const rect = element.getBoundingClientRect();
        const target = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
        return Boolean(target && (target === element || element.contains(target)));
      });
      assert(result.actionReachable, 'The relevant picker/review control is covered after scrolling.');
      if (await action.isEnabled()) await action.click({ trial: true });
      result.layout = await page.evaluate(() => ({ viewport: innerWidth, documentWidth: document.documentElement.scrollWidth }));
      result.surfaceLayout = await surface.evaluate((element) => ({ clientWidth: element.clientWidth, scrollWidth: element.scrollWidth }));
      assert(result.layout.documentWidth <= width + 1 && result.surfaceLayout.scrollWidth <= result.surfaceLayout.clientWidth + 1, 'Horizontal overflow in the workspace or review surface.');
      await page.addScriptTag({ path: require.resolve('axe-core/axe.min.js') });
      result.axe = await page.evaluate(async () => {
        const data = await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa'] } });
        const summary = (rule) => ({ id: rule.id, impact: rule.impact, help: rule.help, helpUrl: rule.helpUrl,
          nodeCount: rule.nodes.length, examples: rule.nodes.slice(0, 3).map((node) => ({ target: node.target, failureSummary: node.failureSummary })) });
        return { violations: data.violations.map(summary), incomplete: data.incomplete.map(summary) };
      });
      result.viewportFile = `${name}-viewport.png`;
      result.surfaceFile = `${name}-surface.png`;
      await page.screenshot({ path: path.join(output, result.viewportFile) });
      await surface.screenshot({ path: path.join(output, result.surfaceFile) });
      assert(result.pageErrors.length === 0, 'Page JavaScript errors occurred.');
      assert(result.protectedContentRequests.length === 0, 'A placeholder PDF content fetch was attempted.');
      result.status = 'CAPTURED';
    } catch (error) {
      result.status = 'ERROR'; result.error = error.message;
      await page.screenshot({ path: path.join(output, `${name}-error.png`) }).catch(() => {});
    }
    results.push(result);
    await writeFile(path.join(output, 'manifest.json'), JSON.stringify({ capturedAt: new Date().toISOString(), base, fixtureOnly: true, fileBytesPreviewed: false, results }, null, 2));
    console.log(JSON.stringify({ name, status: result.status, error: result.error, chooseEnabled: result.chooseEnabled, reachable: result.actionReachable, violations: result.axe?.violations.length, incomplete: result.axe?.incomplete.length }));
    await context.close();
  }
} finally { await browser.close(); }
if (results.some((result) => result.status !== 'CAPTURED' || result.axe?.violations.length)) process.exitCode = 1;
