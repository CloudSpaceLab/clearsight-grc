import {chromium} from '../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs';
import {formsEvidenceScenarios} from '../../../web/scripts/forms-evidence-scenarios.mjs';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const browser=await chromium.launch({headless:true});
try {
  const scenario=formsEvidenceScenarios.find(item=>item.name.startsWith('89-'));
  const context=await browser.newContext({viewport:scenario.viewport,colorScheme:'light',reducedMotion:'reduce',locale:'en-NG',timezoneId:'Africa/Lagos'});
  await context.addInitScript(()=>localStorage.setItem('clearsight.theme','light'));
  const page=await context.newPage();
  await page.goto('http://127.0.0.1:4188/?tour=off&fixture=forms-library-lifecycle#forms',{waitUntil:'networkidle'});
  await scenario.run(page);
  await page.screenshot({path:fileURLToPath(new URL('after/canonical-89.png',import.meta.url)),fullPage:true,animations:'disabled'});
  await writeFile(fileURLToPath(new URL('canonical-receipt.json',import.meta.url)),JSON.stringify({scenario:scenario.name,status:'PASS',browser:browser.version(),keyboard:'Enter opens named editor region; Back focuses original row edit button',generatedAt:new Date().toISOString()},null,2));
  console.log('Canonical scenario 89 passed.');
} finally {await browser.close()}
