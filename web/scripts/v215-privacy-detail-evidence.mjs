import { createRequire } from 'node:module'; const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
import { mkdir,writeFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
const out='../docs/evidence/2026-11-22-privacy-details'; await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.PLAYWRIGHT_BROWSER_PATH ? {executablePath:process.env.PLAYWRIGHT_BROWSER_PATH} : {})}); const results=[];
for (const width of [1440,768,390]) {
 const page=await browser.newPage({viewport:{width,height:900},reducedMotion:'reduce'}); const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.route('**/api/v1/**',route=>{const u=new URL(route.request().url());const value=u.pathname==='/api/v1/context'?{tenant:{id:'bank'},legal_entity:{id:'entity'},actor:{id:'owner'}}:{activity:{id:'activity-1',code:'PA-01',name:'Customer records',status:'OPEN',version:2,purpose:'Provide services',lawful_basis:'Contract',owner_display_name:'Ada',data_categories:[{category:'Biometric data',sensitivity:'SENSITIVE_BY_LAW'}],recipients:[{recipient:'Cloud processor',recipient_kind:'EXTERNAL',country_code:'GB',is_cross_border:true,transfer_basis:'STANDARD_CONTRACT_CLAUSES'}],systems:[{system_name:'Customer platform',system_kind:'APPLICATION'}],reviews:[{id:'review-1',due_date:'2026-02-01',completed_at:'2026-01-31T00:00:00Z',outcome:'CONFIRMED',reviewer_principal_id:'reviewer-1',reviewer_display_name:'Chika Nwosu'}]},state_label:'Open'};return route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(value)});});
 await page.goto('http://127.0.0.1:43124/privacy-detail.html');await page.getByText('Chika Nwosu').waitFor();const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);assert.equal(overflow,false);assert.deepEqual(errors,[]);await page.screenshot({path:`${out}/privacy-${width}.png`,fullPage:true});results.push({width,overflow,errors});await page.close();
}await browser.close();await writeFile(`${out}/manifest.json`,JSON.stringify({synthetic:true,scope:'isolated component; mocked API',results},null,2));

