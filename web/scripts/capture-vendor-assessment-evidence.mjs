import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
const require=createRequire(import.meta.url);
const {chromium}=require(process.env.PLAYWRIGHT_MODULE??'playwright');
const base=process.env.PAGE_URL??'http://127.0.0.1:4178';
const out=path.resolve(process.env.UI_EVIDENCE_DIR??'../.codex-tmp/vendor-assessment-evidence');
await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true});
const results=[];
const selectedCases=process.env.UI_CASES?.split(',');
const widths=(process.env.UI_WIDTHS??'1440,390,320').split(',').map(Number);
const themes=(process.env.UI_THEMES??'light,dark').split(',');
const deviceScaleFactor=Number(process.env.UI_DEVICE_SCALE??1);
const height=Number(process.env.UI_HEIGHT??900);
const cases=[
 ['vendor-overview','vendor-form-assessment','#vendors/vendor-relationship-payments','Vendors'],
 ['vendor-forms','vendor-form-assessment','#vendors/vendor-relationship-payments','Forms and responses'],
 ['vendor-empty','vendor-form-assessment-empty','#vendors/vendor-relationship-payments','Forms and responses'],
 ['vendor-error','vendor-form-assessment-error','#vendors/vendor-relationship-payments','Forms and responses'],
 ['field-authoring','field-assessment-builder','','Sample vendor security review'],
 ['bank-review','field-assessment-review','','Bank assessment'],
 ['policy-setup','field-assessment-policy','','Create a response policy'],
 ['vendor-partial','vendor-form-assessment-partial','#vendors/vendor-relationship-payments','Forms and responses'],
 ['vendor-history','vendor-form-assessment','#vendors/vendor-relationship-payments','Response history'],
 ['vendor-history-review','vendor-form-assessment','#vendors/vendor-relationship-payments','Bank assessment'],
 ['bank-review-saved','vendor-form-assessment','#vendors/vendor-relationship-payments','Bank assessment'],
 ['bank-review-poor','vendor-form-assessment','#vendors/vendor-relationship-payments','Bank assessment'],
];
try{
 for(const theme of themes)for(const width of widths)for(const [name,fixture,route,heading] of cases.filter(([name])=>!selectedCases||selectedCases.includes(name))){
  const context=await browser.newContext({viewport:{width,height},deviceScaleFactor,colorScheme:theme,reducedMotion:'reduce'});
  await context.addInitScript(t=>localStorage.setItem('clearsight.theme',t),theme);
  const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto(`${base}/?tour=off&fixture=${fixture}${route}`,{waitUntil:'networkidle'});
  const checks=[];
  if(name.startsWith('vendor-history')){
   await page.getByRole('button',{name:'View response history',exact:true}).click();
   await page.getByRole('button',{name:'Load earlier responses',exact:true}).click();
   await page.getByRole('heading',{name:'Sample · Earlier certification review',exact:true}).waitFor();
   checks.push('Scoped history loaded current and earlier response revisions through the next-page cursor.');
   if(name==='vendor-history-review'){
    await page.getByRole('button',{name:'Review Sample · Earlier certification review revision 1',exact:true}).click();
    await page.getByText('Historical response. Review the current submission',{exact:false}).waitFor();
    if(await page.getByRole('button',{name:'Save bank assessment',exact:true}).count())throw Error('Historical response offers a save command');
    if(await page.getByRole('button',{name:/Bank judgement for/}).count())throw Error('Historical response offers judgement input');
    checks.push('Historical response retains saved judgement and exposes no judgement input or save command.');
   }
  }
  if(name==='bank-review-saved'||name==='bank-review-poor'){
   await page.getByRole('button',{name:'Review Sample · Payment-service security evidence response',exact:true}).click();
   const judgement=page.getByRole('button',{name:/Bank judgement for Independent vulnerability/});
   await judgement.scrollIntoViewIfNeeded();
   await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
   await judgement.click();
   await page.getByRole('option',{name:/Scope or remediation evidence incomplete/}).click();
   await page.getByLabel('Rationale for Independent vulnerability test report',{exact:true}).fill('Sample review: the submitted test excludes the payment processing service.');
   await page.getByRole('button',{name:'Save bank assessment',exact:true}).click();
   await page.getByText('Bank assessment saved. Respondent answers remain as submitted.',{exact:true}).waitFor();
   await page.getByText('1 of 1 required fields reviewed',{exact:true}).waitFor();
   checks.push('Rubric judgement and rationale saved; required review count moved from 0 to 1.');
   if(name==='bank-review-poor'){
    const filter=page.getByRole('button',{name:/Assessment fields/});
    await filter.scrollIntoViewIfNeeded();
    await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
    await filter.click();
    await page.getByRole('option',{name:'Poor results',exact:true}).click();
    await page.getByText('Showing automatic or reviewed fields with at least',{exact:false}).waitFor();
    const assessment=page.getByRole('region',{name:'Bank assessment',exact:true});
    if(await assessment.getByRole('article').count()!==2)throw Error('Poor filter failed to retain automatic and bank concerns');
    if(await assessment.getByRole('article',{name:'Service changes since the previous review',exact:true}).count())throw Error('Unscored field remains in poor filter');
    checks.push('Poor-results filter retains the automatic 80-point and bank 80-point concerns and excludes the unscored field.');
    await filter.focus();await filter.press('ArrowDown');
    await page.getByRole('option',{name:'Poor results',exact:true}).waitFor();
    await page.keyboard.press('Escape');
    await page.getByRole('listbox').waitFor({state:'hidden'});
    checks.push('Keyboard ArrowDown opens field filters; Escape closes the list without changing the selected result.');
   }
  }
  const target=page.getByRole('heading',{name:heading,exact:true}).first();
  if(name==='field-authoring'){
   await page.locator('.form-builder').waitFor();
   if(await page.getByRole('button',{name:'Settings',exact:true}).isVisible())await page.getByRole('button',{name:'Settings',exact:true}).click();
   await page.locator('.field-assessment-editor:visible').first().waitFor();
  }else await target.waitFor();
  if(name==='bank-review-saved')await page.getByText('Bank assessment saved. Respondent answers remain as submitted.',{exact:true}).scrollIntoViewIfNeeded();
  else if(name==='vendor-partial')await page.getByText('Later submissions replaced some answers.',{exact:false}).scrollIntoViewIfNeeded();
  else if(name.startsWith('vendor-')&&name!=='vendor-overview')await target.scrollIntoViewIfNeeded();
  await page.evaluate(()=>document.fonts.ready);
  const metrics=await page.evaluate(()=>({viewport:innerWidth,scrollWidth:document.documentElement.scrollWidth}));
  await page.addScriptTag({path:require.resolve('axe-core/axe.min.js')});
  const accessibility=await page.evaluate(async()=>{const r=await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa']}});return r.violations.map(v=>({id:v.id,impact:v.impact,nodes:v.nodes.map(n=>n.target)}))});
  const file=`${name}-${theme}-${width}.png`;await page.screenshot({path:path.join(out,file),fullPage:name==='field-authoring'&&width>=1280||name==='bank-review'});
  results.push({file,fixture,theme,width,height,deviceScaleFactor,metrics,errors,accessibility,checks});await context.close();
 }
}finally{await browser.close();await writeFile(path.join(out,'manifest.json'),JSON.stringify(results,null,2));}
const failures=results.filter(r=>r.errors.length||r.metrics.scrollWidth>r.width||r.accessibility.some(v=>v.impact==='critical'||v.impact==='serious'));
if(failures.length)throw Error(JSON.stringify(failures,null,2));
console.log(`Verified ${results.length} vendor and assessment renders; manifest: ${path.join(out,'manifest.json')}`);
