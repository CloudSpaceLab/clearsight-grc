import { chromium } from '../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import {createHash} from 'node:crypto';

const out = path.dirname(fileURLToPath(import.meta.url));
const base = process.env.PAGE_URL || 'http://127.0.0.1:4188';
const axe = await readFile(path.resolve(out, '../../..', 'web/node_modules/axe-core/axe.min.js'), 'utf8');
const browser = await chromium.launch({ headless: true });
const sourceSha256 = {};
for (const file of ['web/src/components/FormsWorkspace.tsx','web/src/components/forms/dashboard/TemplateLibraryTable.tsx','web/src/components/forms/dashboard/TemplateDetailDrawer.tsx','web/src/components/forms/dashboard/templateActions.ts','web/src/forms-dashboard-shell.css','web/src/forms-dashboard-table.css','web/src/form-builder-responsive.css','web/dist-evidence/index.html']) {
  sourceSha256[file] = createHash('sha256').update(await readFile(path.resolve(out,'../../..',file))).digest('hex');
}
const results = [];
await mkdir(path.join(out, 'after'), { recursive: true });
const states = process.env.STATES?.split(',') || ['library', 'draft', 'active', 'published-draft', 'pending', 'denied', 'unavailable', 'busy', 'editor'];
try {
  for (const theme of ['light', 'dark']) for (const width of process.env.WIDTHS?.split(',').map(Number) || [1440, 390, 320]) for (const state of states) {
    const name = `${state}-${theme}-${width}`;
    if (process.env.CASES && !process.env.CASES.split(',').includes(name)) continue;
    const context = await browser.newContext({ viewport: { width, height: width === 1440 ? 900 : width === 390 ? 844 : 800 }, colorScheme: theme, reducedMotion: 'reduce', locale: 'en-NG', timezoneId: 'Africa/Lagos', hasTouch: width < 700 });
    await context.addInitScript(({theme, state}) => {
      localStorage.setItem('clearsight.theme', theme);
      localStorage.setItem('clearsight.density', 'comfortable');
      let underlying = window.fetch.bind(window);
      const wrapped = async (input, init) => {
        const url = new URL(typeof input === 'string' ? input : input.url || String(input), location.origin);
        if (state === 'busy' && init?.method === 'POST' && url.pathname.includes('/transition')) return new Promise(() => {});
        const response = await underlying(input, init);
        if (url.pathname !== '/api/v1/forms/templates' || (init?.method && init.method !== 'GET')) return response;
        const data = await response.json();
        for (const item of data.items) {
          if (item.template.status === 'ACTIVE') item.operations.push({ command: 'forms.template.revise', can_act: true, responsibility: 'ACCOUNTABLE_OWNER', reason: 'You can prepare a new draft version.' });
          if (state === 'published-draft' && item.template.id === 'form-lifecycle-draft') { item.template.version = 4; item.active_version = 3; item.active_status = 'ACTIVE'; }
          if (state === 'denied') for (const operation of item.operations) { operation.can_act = false; operation.assigned_to = { id: 'sample-form-editor', display_name: 'Form Owner', kind: 'POSITION' }; }
          if (state === 'unavailable') { item.authority_available = false; item.operations = []; }
        }
        return new Response(JSON.stringify(data), { status: response.status, headers: response.headers });
      };
      Object.defineProperty(window, 'fetch', { configurable: true, get: () => wrapped, set: value => { underlying = value; } });
    }, {theme, state});
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    const result = { name, state, theme, width, errors, fixture: 'forms-library-lifecycle', fixtureOverrides: ['active author permission', ...(['published-draft', 'denied', 'unavailable', 'busy'].includes(state) ? [state] : [])] };
    try {
      await page.goto(`${base}/?tour=off&fixture=forms-library-lifecycle#forms`, { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: 'Details for Customer complaint review draft', exact: true }).waitFor();
      if (state === 'editor') {
        const editRow = page.getByRole('button', { name: 'Edit draft Customer complaint review draft', exact: true });
        await editRow.focus(); await page.keyboard.press('Enter');
        await page.getByLabel('Form canvas').waitFor();
        await page.getByRole('button', { name: 'Save draft', exact: true }).waitFor();
        await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'Edit Customer complaint review draft');
        await page.getByRole('button', { name: 'Back to Forms', exact: true }).click();
        await editRow.waitFor();
        await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'Edit draft Customer complaint review draft');
        result.keyboard = {entry: 'named editor region', back: 'originating row Edit draft'};
        await page.keyboard.press('Enter');
        await page.getByLabel('Form canvas').waitFor();
        await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'Edit Customer complaint review draft');
      } else if (state !== 'library') {
        const title = state === 'active' ? 'Vendor security and privacy review' : state === 'pending' ? 'Payments control owner confirmation' : 'Customer complaint review draft';
        await page.getByRole('button', { name: `Details for ${title}`, exact: true }).click();
        const dialog = page.getByRole('dialog', { name: 'Selected form template' });
        await dialog.waitFor();
        await page.waitForFunction(() => {
          const el = document.querySelector('[aria-label="Close form detail"]');
          const box = el?.getBoundingClientRect();
          return box && box.x >= 0 && box.right <= innerWidth && box.y >= 0 && box.bottom <= innerHeight;
        });
        if (state === 'pending' && await dialog.getByRole('button', { name: /^Edit/ }).count()) throw new Error('Pending approval exposes an edit action.');
        if (state === 'denied') await dialog.getByText('Editor: Form Owner', { exact: true }).waitFor();
        if (state === 'unavailable') await dialog.getByText('Permissions unavailable. Reload Forms to retry.', { exact: true }).waitFor();
        if (['denied','unavailable'].includes(state) && await page.getByRole('button', { name: /^Edit/ }).count()) throw new Error('Denied/unavailable authority exposes an edit action.');
        if (state === 'busy') {
          // Hold the real transition request to inspect the actual in-flight controls.
          await dialog.getByRole('button', { name: 'Send for approval', exact: true }).click();
          await page.waitForFunction(() => document.querySelector('.forms-detail-actions button')?.disabled);
          if (!await dialog.getByRole('button', { name: 'Edit draft', exact: true }).isDisabled()) throw new Error('Edit remains enabled during transition.');
        }
        const close = await dialog.getByRole('button', { name: 'Close form detail' }).boundingBox();
        if (!close || close.x < 0 || close.x + close.width > width || close.y < 0 || close.y + close.height > page.viewportSize().height) throw new Error('Sheet close action is not reachable.');
        if (['draft', 'active', 'published-draft'].includes(state)) {
          const action = await dialog.getByRole('button', { name: state === 'active' ? 'Edit form' : 'Edit draft', exact: true }).boundingBox();
          const facts = await dialog.locator('.forms-detail-state').boundingBox();
          if (!action || !facts || action.y + action.height > facts.y || action.x < 0 || action.x + action.width > width) throw new Error('Editing action must precede version facts and fit the sheet.');
        }
      }
      await page.evaluate(() => document.fonts.ready);
      result.layout = await page.evaluate(() => ({ width: document.documentElement.clientWidth, scrollWidth: document.documentElement.scrollWidth, heading: [...document.querySelectorAll('h1,h2')].filter(el => el.getClientRects().length).map(el => el.textContent), controls: [...document.querySelectorAll('.forms-library-actions button,.forms-detail-actions button,.form-builder-toolbar button')].filter(el => el.getClientRects().length).map(el => ({ text: el.textContent, disabled: el.disabled, x: el.getBoundingClientRect().x, right: el.getBoundingClientRect().right, height: el.getBoundingClientRect().height })) }));
      if (result.layout.scrollWidth > width + 1) throw new Error('Horizontal page overflow.');
      if (result.layout.controls.some(c => c.x < 0 || c.right > width + 1)) throw new Error('Action control exceeds viewport.');
      if (state === 'library') {
        result.versionLabels = await page.locator('.forms-library-revision span').evaluateAll(els => els.map(el => ({text:el.textContent,wrap:getComputedStyle(el).whiteSpace,ellipsis:getComputedStyle(el).textOverflow})));
        if (result.versionLabels.some(label => label.wrap === 'nowrap' || label.ellipsis === 'ellipsis')) throw new Error('Version status must wrap without truncation.');
        if (width > 700) {
          result.actionRows = await page.locator('.forms-library-actions').evaluateAll(els => els.map(el => [...el.querySelectorAll('button')].map(button => ({y:button.getBoundingClientRect().y,height:button.getBoundingClientRect().height}))));
          if (result.actionRows.some(row => row.some(action => action.y !== row[0].y || action.height < 44))) throw new Error('Desktop row actions must share one row and retain44px targets.');
        }
      }
      if (state === 'editor' && width < 700) {
        result.questionControls = await page.locator('.form-canvas-question').first().evaluate(question => {
          const type = question.querySelector('.cs-select-field__trigger').getBoundingClientRect();
          const value = question.querySelector('.cs-select-field__value').getBoundingClientRect();
          const required = question.querySelector('.form-question-required').getBoundingClientRect();
          const input = question.querySelector('.form-question-prompt');
          const canvas = document.createElement('canvas'); const ctx=canvas.getContext('2d'); ctx.font=getComputedStyle(input).font;
          return {typeWidth:type.width,typeHeight:type.height,typeBottom:type.bottom,valueRight:value.right,typeRight:type.right,requiredTop:required.top,requiredHeight:required.height,promptWidth:input.clientWidth,promptTextWidth:ctx.measureText(input.value).width};
        });
        const q=result.questionControls;
        if(q.typeWidth<q.valueRight-(q.typeRight-q.typeWidth)||q.typeHeight<44||q.requiredTop<q.typeBottom||q.requiredHeight<44||q.promptTextWidth>q.promptWidth) throw new Error('Selected question label/type/Required controls must fit without overlap and retain44px targets.');
      }
      await page.addScriptTag({content: axe});
      result.axe = await page.evaluate(async () => { const a = await axe.run(document, {runOnly:{type:'rule', values:['color-contrast']}}); return { violations:a.violations, incomplete:a.incomplete.map(item => ({id:item.id, nodes:item.nodes.length})) }; });
      if (result.axe.violations.length) throw new Error('Axe reports contrast violations.');
      if (errors.length) throw new Error(errors.join('; '));
      await page.screenshot({ path: path.join(out, 'after', `${name}.png`), fullPage: state === 'library', animations: 'disabled' });
      if (state === 'editor' && width < 700) {
        await page.locator('.form-canvas-question').first().evaluate(el => el.scrollIntoView({block:'center',behavior:'instant'}));
        await page.screenshot({ path:path.join(out,'after',`${name}-question.png`),fullPage:false,animations:'disabled' });
      }
      result.status = 'PASS';
    } catch (error) {
      result.status = 'FAIL'; result.error = error.message;
      await page.screenshot({path:path.join(out,'after',`${name}-failure.png`),fullPage:true}).catch(()=>{});
    }
    await writeFile(path.join(out,'after',`${name}.json`),JSON.stringify(result,null,2));
    results.push(result); console.log(`${name}: ${result.status}${result.error ? ` ${result.error}` : ''}`);
    await writeFile(path.join(out,process.env.RECEIPT || 'receipt.json'),JSON.stringify({generatedAt:new Date().toISOString(),base,browser:browser.version(),playwright:'1.55.0',sourceSha256,results},null,2));
    await context.close();
  }
} finally { await browser.close(); }
if (results.some(result => result.status !== 'PASS')) process.exitCode = 1;
