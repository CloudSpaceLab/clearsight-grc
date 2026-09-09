import { chromium } from '../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs';
import {writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const browser=await chromium.launch({headless:true});
const captures=[];
try {
  for(const theme of ['light','dark']) for(const width of [1440,390,320]) {
    const context=await browser.newContext({viewport:{width,height:width===1440?900:width===390?844:800},colorScheme:theme,reducedMotion:'reduce',locale:'en-NG',timezoneId:'Africa/Lagos'});
    await context.addInitScript(theme=>localStorage.setItem('clearsight.theme',theme),theme);
    const page=await context.newPage();
    await page.goto('http://127.0.0.1:4187/?tour=off&fixture=forms-library-lifecycle#forms',{waitUntil:'networkidle'});
    await page.getByRole('button',{name:'Open Customer complaint review draft',exact:true}).waitFor();
    for(const state of ['library','detail','editor']) {
      if(state==='detail') {
        await page.getByRole('button',{name:'Open Customer complaint review draft',exact:true}).click();
        await page.waitForFunction(()=>{const r=document.querySelector('[aria-label="Close form detail"]')?.getBoundingClientRect();return r&&r.x>=0&&r.right<=innerWidth&&r.y>=0&&r.bottom<=innerHeight});
      }
      if(state==='editor') {
        await page.getByRole('dialog',{name:'Selected form template'}).getByRole('button',{name:'Edit draft',exact:true}).click();
        await page.getByLabel('Form canvas').waitFor();
      }
      const name=`before/${state}-${theme}-${width}.png`;
      await page.screenshot({path:fileURLToPath(new URL(name,import.meta.url)),fullPage:state==='library',animations:'disabled'});
      captures.push({name,theme,width,state,fixture:'forms-library-lifecycle'});
    }
    await context.close();
  }
} finally {await browser.close()}
await writeFile(fileURLToPath(new URL('before/receipt.json',import.meta.url)),JSON.stringify({source:'Preserved web/dist-evidence build, copied before rebuilding the changed source. Exact source revision of this local build is not certified.',browser:browser.version(),captures},null,2));
console.log(`${captures.length} before screenshots captured.`);
