import {createRequire} from 'node:module';
import {mkdir, writeFile, readFile} from 'node:fs/promises';
import path from 'node:path';
const require=createRequire(import.meta.url);
const {chromium}=require(process.env.PLAYWRIGHT_MODULE ?? 'C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const base=process.env.PAGE_URL ?? 'http://127.0.0.1:4187';
const out=path.dirname(new URL(import.meta.url).pathname.replace(/^\/(?:([A-Za-z]:))/, '$1'));
const axe=await readFile(path.resolve('web/node_modules/axe-core/axe.min.js'),'utf8');
const browser=await chromium.launch({headless:true});
const captures=[];
await mkdir(out,{recursive:true});
async function tab(page,name,label='Vendor section') {
 const t=page.getByRole('tablist',{name:label==='Response section'?'Response sections':'Vendor sections',exact:true}).getByRole('tab',{name,exact:true});
 if(await t.isVisible()) await t.click();
 else {const control=page.getByRole('button',{name:new RegExp(label)});await control.scrollIntoViewIfNeeded();await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));await control.click();await page.getByRole('option',{name,exact:true}).click();}
}
async function save(page,id,theme,width,checks=[]) {
 if(['checklist','reused-pending-review'].includes(id))await page.locator('.vendor-checklist').evaluate(element=>window.scrollBy(0,element.getBoundingClientRect().top-88));
 await page.evaluate(()=>document.fonts.ready);
 await page.addScriptTag({content:axe});
 const violations=await page.evaluate(async()=> (await globalThis.axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa']}})).violations.map(v=>({id:v.id,impact:v.impact,nodes:v.nodes.map(n=>({target:n.target,summary:n.failureSummary}))})));
 const metrics=await page.evaluate(()=>({width:innerWidth,scrollWidth:document.documentElement.scrollWidth}));
 const file=`${id}-${theme}-${width}.png`;
 await page.screenshot({path:path.join(out,file),fullPage:false});
 captures.push({file,id,theme,width,checks,metrics,violations,passed:!violations.length&&metrics.scrollWidth<=width+1});
 console.log(`${file}: ${captures.at(-1).passed?'PASS':'FAIL'}`);
}
try {
 for(const theme of ['light','dark']) for(const width of [1440,390]) {
  const context=await browser.newContext({viewport:{width,height:900},colorScheme:theme,hasTouch:width<800,reducedMotion:'reduce'});
  await context.addInitScript(t=>localStorage.setItem('clearsight.theme',t),theme);
  const page=await context.newPage();page.setDefaultTimeout(10000);
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  try {
   await page.goto(`${base}/?tour=off&fixture=forms-response-history#forms?section=responses`,{waitUntil:'networkidle'});
   await page.getByRole('button',{name:'Review Vendor due diligence review response',exact:true}).waitFor();
   await save(page,'responses',theme,width,['Submitted response stays separate from review and score.']);
   await page.getByRole('button',{name:'Review Vendor due diligence review response',exact:true}).click();
   for(const section of ['Answers','Documents','Review','History']) {
    await tab(page,section,'Response section');
    await save(page,`response-${section.toLowerCase()}`,theme,width,['Response tabs use a compact select on mobile; documents use the selected revision.']);
   }
   await page.goto(`${base}/?tour=off&fixture=field-assessment-policy`,{waitUntil:'networkidle'});
   await page.getByRole('heading',{name:'Create a response policy',exact:true}).waitFor();
   await save(page,'policy-editor',theme,width,['Approved form, result basis, issue limits and outcome checks remain editable.']);
   await page.goto(`${base}/?tour=off#forms?section=policies`,{waitUntil:'networkidle'});
   await page.getByRole('button',{name:'Retry loading policies',exact:true}).waitFor();
   await page.evaluate(()=>{
    const previous=window.fetch.bind(window);
    window.fetch=async(input,init)=>{
     const url=new URL(typeof input==='string'?input:input instanceof URL?input.href:input.url,location.origin);
     const json=value=>new Response(JSON.stringify(value),{headers:{'Content-Type':'application/json'}});
     if(url.pathname==='/api/v1/config/form-response-policies')return json({items:[{id:'release-sample-policy',code:'SAMPLE-ASSURANCE',name:'Sample · Review assurance gaps',purpose:'Review certification gaps for payment processing services.',version:2,record_version:4,status:'ACTIVE',rollout:'ACTIVE',checker_id:'sample-checker',approved_simulation_id:'sample-simulation',approved_at:'2026-09-08T12:00:00Z',eligibility:{form_template_id:'form-vendor-due-diligence',form_template_version:2,result_basis:'BANK_ASSESSED',bands:['HIGH','CRITICAL']},action:{title_template:'Review assurance gap',requested_handling:'Confirm the affected service and remediation deadline.'},blast_radius:{per_run:10,per_day:50},outcome_contract:{check_after_minutes:1440}}]});
     if(url.pathname==='/api/v1/config/form-response-policies/release-sample-policy/executions')return json({items:[{id:'sample-execution',state:'APPLIED',result_basis:'BANK_ASSESSED',assessment_version:2,created_at:'2026-09-08T12:00:00Z'}]});
     if(url.pathname.includes('/release-sample-policy/executions/'))return json({targets:[{type:'FORM_RESPONSE',id:'sample-response',title:'Sample · Vendor assurance review'}]});
     return previous(input,init);
    };
   });
   await page.getByRole('button',{name:'Retry loading policies',exact:true}).click();
   await page.getByRole('button',{name:'View result',exact:true}).waitFor();
   await save(page,'policy-detail',theme,width,['Synthetic approved policy fixture separates historical approval simulation from current counts.']);
   await page.getByRole('button',{name:'View result',exact:true}).click();
   await page.getByRole('dialog',{name:'Policy result',exact:true}).waitFor();
   await save(page,'policy-result',theme,width,['Synthetic execution result provides the permitted response link.']);
   await page.goto(`${base}/?tour=off&fixture=vendor-form-assessment#vendors/vendor-relationship-payments`,{waitUntil:'networkidle'});
   await page.getByRole('heading',{name:'Vendors',exact:true}).waitFor();
   for(const section of ['Overview','Forms','Documents','Due diligence','History']) {
    await tab(page,section);
    const panel=page.getByRole('tabpanel');if(await panel.count())await panel.scrollIntoViewIfNeeded();
    await save(page,`vendor-${section.toLowerCase().replaceAll(' ','-')}`,theme,width,['Vendor sections retain separate evidence collection, assessment and history.']);
   }
   await page.goto(`${base}/?tour=off&fixture=vendor-collection#vendors`,{waitUntil:'networkidle'});
   await page.getByRole('button',{name:/Acme Processing Limited.*Card transaction processing/}).click();
   await tab(page,'Due diligence');
   await page.locator('.vendor-checklist').waitFor();
   await page.locator('.vendor-checklist').scrollIntoViewIfNeeded();
   await save(page,'checklist',theme,width,['Missing, Pending review and Accepted are separately counted.']);
   await page.getByRole('button',{name:'Missing 1',exact:true}).click();
   await page.getByRole('button',{name:'Use existing document',exact:true}).click();
   const dialog=page.getByRole('dialog',{name:/Use existing document for/});
   await dialog.getByRole('row').filter({hasText:'Security test report.pdf'}).click();
   await dialog.getByRole('button',{name:'Choose this document',exact:true}).click();
   await dialog.getByLabel('Reason for reuse').fill('Sample review: covers this service and assessment period.');
   await save(page,'reuse-document',theme,width,['Existing evidence is selected with source context and a recorded reason.']);
   await dialog.getByRole('button',{name:'Use this document',exact:true}).click();
   await dialog.waitFor({state:'hidden'});
   await page.getByRole('button',{name:'Missing 0',exact:true}).waitFor();
   await page.getByRole('button',{name:'Awaiting review 2',exact:true}).waitFor();
   await page.getByRole('button',{name:'All items 5',exact:true}).click();
   await page.locator('.vendor-checklist').scrollIntoViewIfNeeded();
   await save(page,'reused-pending-review',theme,width,['Reused evidence removes the vendor request but still requires review.']);
   if(errors.length)throw Error(errors.join('\n'));
  } catch(error) {captures.push({id:'scenario-error',theme,width,passed:false,error:String(error)});console.log(String(error));await page.screenshot({path:path.join(out,`failure-${theme}-${width}.png`)});}
  finally {await context.close();}
 }
} finally {await browser.close();await writeFile(path.join(out,'manifest.json'),JSON.stringify({generatedAt:new Date().toISOString(),base,source:'Local isolated evidence build; HEAD 470ad3c2e90e84750c43985ddd835ee69be36d84 plus integrated origin/main and staged audit fixes. Not hosted/live data.',captures},null,2));}
if(captures.some(c=>!c.passed))process.exitCode=1;
