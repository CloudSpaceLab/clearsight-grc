import { createRequire } from 'node:module';
import { mkdir } from 'node:fs/promises';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const output = new URL('../../docs/quality/screenshots/program-follow-up/', import.meta.url).pathname.replace(/^\/([A-Z]:)/, '$1');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  for (const [name, viewport] of [['desktop', { width: 1440, height: 1000 }], ['mobile', { width: 390, height: 844 }]]) {
    const context = await browser.newContext({ viewport, reducedMotion: 'reduce' });
    await context.addInitScript(() => { localStorage.setItem('clearsight.theme', 'light'); });
    const page = await context.newPage();
    await page.route('**/staticDemoFixtures-*.json', async route => {
      const response = await route.fetch();
      const fixtures = await response.json();
      fixtures.programDetail.current_state.reasons = fixtures.programDetail.evidence_contracts.map(contract => ({ code: 'EVIDENCE_EXPIRED', object_type: 'EVIDENCE_CONTRACT', object_id: contract.id, summary: `The Program assessment for ${contract.name} has passed its validity date.` }));
      fixtures.programDetail.current_state.reasons.push({ code: 'OPEN_MATTERS', summary: '2 open issues affect this program.' });
      await route.fulfill({ json: fixtures });
    });
    await page.goto('http://127.0.0.1:4187/?tour=off&fixture=program-responses#programs/program-ndpa/overview');
    await page.getByRole('button', { name: /Review assessments/ }).waitFor();
    await page.screenshot({ path: `${output}/${name}-overview.png`, fullPage: true });
    await page.getByRole('button', { name: /Review assessments/ }).click();
    await page.getByRole('heading', { name: 'Submitted data' }).waitFor();
    if (!page.url().endsWith('/evidence-results')) throw new Error('Follow-up navigation failed');
    await page.getByRole('heading', { name: 'Submitted data' }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${output}/${name}-responses.png`, fullPage: false });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
    console.log(JSON.stringify({ name, overflow, followUpNavigation: 'passed' }));
    if (overflow) throw new Error(`${name}: horizontal page overflow`);
    await context.close();
  }
} finally { await browser.close(); }
