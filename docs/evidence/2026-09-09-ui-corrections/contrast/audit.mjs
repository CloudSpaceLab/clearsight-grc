import {createRequire} from 'node:module';
import {inspectRenderedContrast as inspect} from '../../../../web/scripts/rendered-contrast.mjs';
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import path from 'node:path';
const require=createRequire(import.meta.url);
const {chromium}=require('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out=process.env.UI_EVIDENCE_DIR ? path.resolve(process.env.UI_EVIDENCE_DIR) : path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Z]:)/,'$1'));
const widths=(process.env.AUDIT_WIDTHS ?? '1440,390').split(',').map(Number);
const scale=Number(process.env.AUDIT_SCALE ?? '1');
const axeSource=await readFile('C:/dev/clearsight-grc/web/node_modules/axe-core/axe.min.js','utf8');
const browser=await chromium.launch({headless:true});
const bases=[
 ['today','','#today'],['today-empty','today-empty','#today'],['today-unavailable','today-unavailable','#today'],
 ['programs','','#programs'],['program-detail','','#programs/program-ndpa/overview'],
 ['issues','','#work/matters'],['issue-detail','matter-overdue-action','#work/matters/matter-gaid-change'],['evidence','','#work/evidence'],['imports','','#imports'],
 ['forms-library','forms-library-lifecycle','#forms'],['forms-builder','forms-weights-valid','#forms','builder'],['forms-sent','forms-distribution-history','#forms','sent'],['forms-response','forms-response-history','#forms','response'],
 ['vendors','vendor-collection','#vendors'],['vendor-checklist','vendor-collection','#vendors','checklist'],['document-select','vendor-collection','#vendors','select'],['document-review','vendor-collection','#vendors','review'],
 ['configure','','#configure'],['configure-authority','','#configure/authority'],['capture','forms-vendor-held-actions','/capture'],
 ['gallery','ui-component-gallery','#ui-components'],['gallery-select','ui-component-gallery','#ui-components','gallery-select'],['field-review','field-assessment-review',''],['field-builder','field-assessment-builder',''],['capture-received','vendor-capture-received',''],['capture-all-held','vendor-capture-all-held','']
];
if(process.env.AUDIT_EXTRA) bases.push(['forms-response-list','forms-response-history','#forms','response-list'],['forms-response-filters','forms-response-history','#forms','response-filters'],['forms-response-unavailable','forms-response-history','#forms','response-error'],['configure-data','','#configure/data']);
const results=[];await mkdir(out,{recursive:true});
try{for(const [name,fixture,route,action] of bases.filter(b=>!process.env.AUDIT_CASES||process.env.AUDIT_CASES.split(',').includes(b[0])))for(const theme of ['light','dark'])for(const width of widths){
 const id=`${name}-${theme}-${width}`,context=await browser.newContext({viewport:{width,height:scale===2?450:width>=1024?900:844},deviceScaleFactor:scale,colorScheme:theme,reducedMotion:'reduce',hasTouch:width<=390});
 await context.addInitScript(t=>{localStorage.setItem('clearsight.theme',t);localStorage.setItem('clearsight.density','comfortable')},theme);
 const page=await context.newPage();page.setDefaultTimeout(9000);const errors=[];page.on('pageerror',e=>errors.push(e.message));let result={id,name,fixture,route,action,theme,width,scale,generatedAt:new Date().toISOString(),errors};
 try{
  const url=route.startsWith('/')?`${process.env.PAGE_URL ?? 'http://127.0.0.1:4179'}${route}?tour=off&fixture=${fixture}#form_access=task22-${fixture}`:`${process.env.PAGE_URL ?? 'http://127.0.0.1:4179'}/?tour=off&fixture=${fixture}${route}`;
  await page.goto(url,{waitUntil:'networkidle'});await page.evaluate(()=>document.fonts.ready);
  if(action==='response-error')await page.evaluate(()=>{const fixtureFetch=globalThis.fetch;globalThis.__assessmentFailures=0;globalThis.fetch=(input,init)=>{const url=typeof input==='string'?input:input.url??input.toString();if(/\/api\/v1\/forms\/responses\/[^/]+\/assessment(?:\?|$)/.test(url)){globalThis.__assessmentFailures++;return Promise.resolve(new Response(JSON.stringify({message:'Response assessment is temporarily unavailable. Try again.'}),{status:503,headers:{'Content-Type':'application/json'}}))}return fixtureFetch(input,init)}});
  if(action==='builder'){await page.getByRole('button',{name:'Open Compliance scoring review'}).click();await page.getByRole('button',{name:'Edit draft',exact:true}).click();await page.getByLabel('Form canvas').waitFor()}
  if(action==='sent')await page.getByRole('tab',{name:'Sent forms',exact:true}).click();
  if(action==='response-list'||action==='response-filters'){await page.getByRole('tab',{name:'Responses',exact:true}).click();await page.getByRole('button',{name:'Review Vendor due diligence review response'}).waitFor();if(action==='response-filters')await page.locator('.forms-response-filters > summary').click();result.filterLayout=await page.evaluate(()=>({open:document.querySelector('.forms-response-filters').open,resultsTop:document.querySelector('.forms-responses__results').getBoundingClientRect().top,filterHeight:document.querySelector('.forms-response-filters').getBoundingClientRect().height}))}
  if(action==='response'||action==='response-error'){await page.getByRole('tab',{name:'Responses',exact:true}).click();await page.getByRole('button',{name:'Review Vendor due diligence review response'}).click();await page.locator('.response-assessment').waitFor();await page.getByText('Loading assessment…',{exact:true}).waitFor({state:'hidden'});}
  if(['checklist','select','review'].includes(action)){
   await page.getByRole('button',{name:/Acme Processing Limited.*Card transaction processing/}).click();await page.locator('.vendor-checklist').waitFor();await page.locator('.vendor-checklist').scrollIntoViewIfNeeded();
   if(action==='select'){await page.getByRole('button',{name:'Missing 1',exact:true}).click();await page.getByRole('button',{name:'Use existing document',exact:true}).click();await page.getByRole('dialog',{name:/Use existing document for/}).waitFor()}
   if(action==='review'){await page.locator('.vendor-checklist').getByRole('article',{name:'ISO 27001 assurance',exact:true}).getByRole('button',{name:'Review document',exact:true}).click();await page.getByRole('dialog',{name:'Review document',exact:true}).waitFor()}
  }
  if(action==='gallery-select'){const trigger=page.getByRole('button',{name:/Sample response status/});await trigger.scrollIntoViewIfNeeded();await page.waitForTimeout(150);await trigger.click();await page.getByRole('listbox').waitFor()}
  if(action==='response-error'){await page.getByRole('button',{name:'Reload assessment',exact:true}).waitFor();await page.getByRole('button',{name:'Reload assessment',exact:true}).scrollIntoViewIfNeeded();result.simulatedAssessmentFailures=await page.evaluate(()=>globalThis.__assessmentFailures);if(!result.simulatedAssessmentFailures)throw new Error('Assessment failure fixture did not run')}
  await page.screenshot({path:path.join(out,id+'.png'),fullPage:true});
  await page.addScriptTag({content:axeSource});
  result.axe=await page.evaluate(async()=>{const a=await axe.run(document,{runOnly:{type:'rule',values:['color-contrast']}});return{version:axe.version,violations:a.violations,incomplete:a.incomplete,passes:a.passes.map(p=>({id:p.id,nodes:p.nodes.length})),inapplicable:a.inapplicable.map(p=>p.id)}});
  result.measurements=await page.evaluate(inspect);
  result.assetURLs=await page.evaluate(()=>performance.getEntriesByType("resource").map(entry=>entry.name).filter(url=>url.includes("/assets/")));
  const fixtureErrors=await page.getByText(/Static stakeholder demo does not implement/).allTextContents();if(fixtureErrors.length)errors.push(...fixtureErrors.map(text=>'fixture: '+text));
  result.layout=await page.evaluate(()=>({clientWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth}));
  result.headings=await page.locator('h1,h2').allTextContents();result.status='CAPTURED';
  if(name==='gallery'){
   const field=page.getByLabel('Sample vendor name',{exact:true});await field.scrollIntoViewIfNeeded();await page.screenshot({path:path.join(out,`gallery-fields-${theme}-${width}.png`)});await field.focus();await page.keyboard.press('Tab');await page.keyboard.press('Shift+Tab');const focused=await page.evaluate(()=>{const e=document.activeElement,s=getComputedStyle(e);return{tag:e.tagName,text:e.textContent?.trim(),focusVisible:e.matches(':focus-visible'),outline:s.outline,boxShadow:s.boxShadow,background:s.backgroundColor,color:s.color}});result.keyboardFocus=focused;await page.screenshot({path:path.join(out,id+'-focus.png'),fullPage:true});
  }
 }catch(e){result.status='ERROR';result.error=e.message;await page.screenshot({path:path.join(out,id+'-error.png'),fullPage:true}).catch(()=>{})}
 await writeFile(path.join(out,id+'.json'),JSON.stringify(result,null,2));results.push({id,status:result.status,error:result.error,errors,violations:result.axe?.violations.reduce((n,v)=>n+v.nodes.length,0),incomplete:result.axe?.incomplete.reduce((n,v)=>n+v.nodes.length,0)});console.log(JSON.stringify(results.at(-1)));await context.close();
 await writeFile(path.join(out,'summary.json'),JSON.stringify({generatedAt:new Date().toISOString(),results},null,2));
}}finally{await browser.close()}
