import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { after, before, test } from 'node:test';
import { inspectRenderedContrast } from './rendered-contrast.mjs';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
let browser;
before(async () => { browser = await chromium.launch({ headless: true }); });
after(async () => { await browser?.close(); });

test('accessibility runner reports every matrix state after contrast failures', async () => {
  const output = await mkdtemp(path.join(tmpdir(), 'clearsight-contrast-'));
  const server = createServer((request, response) => {
    response.setHeader('Content-Type', 'text/html');
    response.end(`<html lang="en"><title>Contrast regression fixture</title><style>
      body { background: white; color: black; } input { background: white; color: black; }
      input::placeholder { color: color(srgb .7 .7 .7); opacity: 1; }
    </style><main>${['Today', 'Programs', 'Work', 'Imports', 'Configuration'].map((heading) => `<h1>${heading}</h1>`).join('')}
    <label>Search records<input placeholder="Search records"></label></main></html>`);
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const child = spawn(process.execPath, ['scripts/review-ui-accessibility.mjs'], {
      cwd: fileURLToPath(new URL('..', import.meta.url)),
      env: { ...process.env, PAGE_URL: `http://127.0.0.1:${server.address().port}`, UI_EVIDENCE_DIR: output },
      stdio: 'ignore',
    });
    const code = await new Promise((resolve, reject) => { child.once('error', reject); child.once('exit', resolve); });
    assert.equal(code, 1);
    const report = JSON.parse(await readFile(path.join(output, 'accessibility.json'), 'utf8'));
    assert.equal(report.scenarios.length, 28);
    assert.ok(report.scenarios.every((scenario) => scenario.status === 'FAIL' && Array.isArray(scenario.incomplete) && scenario.contrast.some((reading) => reading.kind === 'placeholder' && reading.outcome === 'FAIL')));
  } finally {
    await new Promise((resolve) => server.close(resolve));
    assert.equal(path.dirname(path.resolve(output)), path.resolve(tmpdir()));
    assert.ok(path.basename(output).startsWith('clearsight-contrast-'));
    await rm(output, { recursive: true, force: true });
  }
});

test('modern-color placeholders, opaque masking and control classification retain their distinct outcomes', async () => {
  const page = await browser.newPage();
  await page.setContent(`<style>
    body { background: white; } input { height: 40px; background: white; border: 1px solid #ddd; }
    input::placeholder { color: color(srgb .5 .5 .5 / .5); opacity: 1; }
    .gradient { background: linear-gradient(white, black); padding: 10px; }
    .opaque { background: white; } .transparent { background: transparent; }
    .group { opacity: .5; }
  </style>
  <input id="essential" class="cs-field__control" placeholder="Search records">
  <input id="inactive" disabled placeholder="Unavailable">
  <button id="decorative" style="border:1px solid #ddd">Open records</button>
  <div class="gradient"><input id="masked" placeholder="Opaque field"><input id="uncertain" class="transparent" placeholder="Gradient field"></div>
  <div class="group"><input id="group" placeholder="Faded group"></div>`);
  const readings = await page.evaluate(inspectRenderedContrast);
  const find = (selector, kind = 'placeholder') => readings.find((r) => r.selector === selector && r.kind === kind);
  assert.equal(find('#essential').outcome, 'FAIL');
  assert.ok(find('#essential').ratio < 2);
  assert.equal(find('#inactive').outcome, 'INACTIVE');
  assert.equal(find('#essential', 'control-border').outcome, 'FAIL');
  assert.equal(find('#decorative', 'control-border').outcome, 'REVIEW');
  assert.deepEqual(find('#masked').uncertain, []);
  assert.equal(find('#uncertain').outcome, 'INCOMPLETE');
  assert.equal(find('#group').outcome, 'INCOMPLETE');
  await page.close();
});

test('shared field and badge contracts pass on plain and selected surfaces in both themes', async () => {
  const files = ['design-system/tokens/primitives.css', 'design-system/tokens/semantic.css', 'design-system/tokens/components.css', 'design-system/base.css', 'design-system/components/fields.css', 'design-system/components/feedback.css', 'ui-preferences.css'];
  const css = (await Promise.all(files.map((file) => readFile(new URL('../src/' + file, import.meta.url), 'utf8')))).join('\n');
  const page = await browser.newPage();
  for (const theme of ['light', 'dark']) {
    await page.setContent(`<html data-theme="${theme}"><head><style>
      @layer base { ::placeholder { color: color-mix(in oklab, currentcolor 50%, transparent); } }
      ${css}
      body { padding: 20px; background: var(--cs-bg-surface-1); }
      section { padding: 20px; background: color-mix(in srgb, var(--cs-accent-cyan) 10%, var(--cs-bg-surface-2)); }
    </style></head><body>
      <input class="cs-field__control" placeholder="Search obligations">
      <label class="cs-checkbox-field" data-selected><span class="cs-checkbox-field__box">✓</span><span>Include received records</span></label>
      <section><input class="cs-search-field__control" placeholder="Search records">
      ${['neutral', 'info', 'success', 'warning', 'error', 'unknown'].map((tone) => `<span class="cs-status-badge cs-tone--${tone}">${tone} status</span>`).join('')}</section>
    </body></html>`);
    const readings = await page.evaluate(inspectRenderedContrast);
    const failures = readings.filter((r) => r.outcome === 'FAIL' && (r.kind === 'placeholder' || r.essential || r.selector.includes('cs-status-badge')));
    assert.deepEqual(failures, [], `${theme} contrast: ${JSON.stringify(failures)}`);
  }
  await page.close();
});
