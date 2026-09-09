import {chromium} from '../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs';
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const output=path.dirname(fileURLToPath(import.meta.url));
const base=process.env.PAGE_URL || 'http://127.0.0.1:4188';
const states=process.env.STATES?.split(',') || ['empty','gaps','incomplete','reused','expired','partially-replaced','awaiting-review','satisfactory','conditional','adverse','unavailable','pagination','restricted','freshness-unknown'];
const integrated=process.env.INTEGRATED==='1';
const axe=await readFile(path.resolve(output,'../../../web/node_modules/axe-core/axe.min.js'),'utf8');
const sourceSha256={};
for(const file of ['web/src/components/VendorComplianceOverview.tsx','web/src/components/vendor-compliance-overview.css','web/src/components/VendorsWorkspace.tsx','web/src/vendors.css','web/src/vendorComplianceEvidence.tsx','web/src/vendorFormsApi.ts','web/dist-evidence/index.html']) sourceSha256[file]=createHash('sha256').update(await readFile(path.resolve(output,'../../..',file))).digest('hex');
const browser=await chromium.launch({headless:true});const results=[];
await mkdir(path.join(output,'after'),{recursive:true});
try {
  for(const theme of ['light','dark']) for(const width of [1440,390,320]) for(const state of states) {
    const name=`${integrated?'app-':''}${state}-${theme}-${width}`;
    const context=await browser.newContext({viewport:{width,height:width===1440?900:width===390?844:800},colorScheme:theme,reducedMotion:'reduce',locale:'en-NG',timezoneId:'Africa/Lagos',hasTouch:width<700});
    await context.addInitScript(theme=>{localStorage.setItem('clearsight.theme',theme);localStorage.setItem('clearsight.density','comfortable')},theme);
    const page=await context.newPage();page.setDefaultTimeout(12000);const errors=[];
    page.on('pageerror',error=>errors.push(error.message));
    page.on('console',message=>{if(message.type()==='error')errors.push(message.text())});
    const result={name,state,theme,width,integrated,errors};
    try {
      await page.goto(`${base}/?tour=off&fixture=vendor-compliance-${integrated?'app-':''}${state}#vendors`,{waitUntil:'networkidle'});
      if(integrated) await page.getByRole('button',{name:/Acme Processing Limited/}).first().click();
      const overview=page.getByRole('region',{name:'Vendor compliance',exact:true});
      await overview.waitFor();
      await overview.getByText('Loading vendor forms…',{exact:true}).waitFor({state:'hidden'});
      const status=overview.locator('.vendor-compliance__heading .cs-status-badge');
      const expected={empty:'Not assessed',gaps:'Action required',incomplete:'Action required',reused:'Awaiting review',expired:'Action required','partially-replaced':'Outdated','awaiting-review':'Awaiting review',satisfactory:'Satisfactory',conditional:'Conditional',adverse:'Unsatisfactory',unavailable:'Unavailable',pagination:'Unknown',restricted:'Not assessed','freshness-unknown':'Unknown'}[state];
      await overview.getByText(expected,{exact:true}).first().waitFor();
      const expectedAction=state==='empty'?'Request form':['gaps','expired','awaiting-review'].includes(state)?'Review response':['satisfactory','conditional','adverse','pagination','freshness-unknown'].includes(state)?'Review compliance':'Manage forms';
      await overview.locator('.vendor-compliance__heading').getByRole('button',{name:expectedAction,exact:true}).waitFor();
      if(state==='gaps') {
        if(integrated&&width===1440)await page.getByText('No assessed concern recorded',{exact:true}).waitFor();
        for(const label of ['ISO 27001 certificate','PCI DSS attestation','Contractual audit rights','Vulnerability and penetration testing']) await overview.getByText(label,{exact:true}).waitFor();
        if(await overview.getByText('Not met',{exact:true}).count()!==2) throw new Error('Both configured rule failures must remain visible.');
        await overview.getByText('2 checks awaiting review',{exact:true}).waitFor();
        const incomplete=await overview.locator('.vendor-compliance__counts > div').first().locator('dd').textContent();
        if(incomplete!=='0') throw new Error('Submitted failures must not become an incomplete form.');
      }
      if(state==='freshness-unknown')await overview.getByText('1 not checked',{exact:true}).waitFor();
      if(state==='reused') {await overview.getByText('3 documents received',{exact:true}).waitFor();if(await overview.getByText('Missing',{exact:true}).count())throw new Error('Reused current evidence was described as missing.');}
      if(state==='partially-replaced')await overview.getByText('Partly replaced · Review required',{exact:true}).waitFor();
      if(state==='unavailable' && await overview.locator('.vendor-compliance__counts dd').count())throw new Error('Unavailable counts must not appear as zero.');
      if(['empty','incomplete','awaiting-review','expired','partially-replaced','unavailable','pagination','restricted'].includes(state) && await status.textContent()==='Satisfactory')throw new Error('Uncertain or incomplete state received a favourable assessment.');
      if(state==='restricted') {
        if((await page.locator('body').textContent()).includes('Restricted treasury review'))throw new Error('Restricted sample record leaked into the UI.');
        if(await overview.locator('.vendor-compliance__forms > li').count()!==1)throw new Error('The overview must display only the authorized population.');
      }
      await page.evaluate(()=>document.fonts.ready);
      result.geometry=await overview.evaluate(el=>({pageWidth:document.documentElement.clientWidth,pageScrollWidth:document.documentElement.scrollWidth,scrollY:window.scrollY,componentWidth:el.clientWidth,componentScrollWidth:el.scrollWidth,buttons:[...el.querySelectorAll('button')].filter(b=>b.getClientRects().length).map(b=>({label:b.getAttribute('aria-label')||b.textContent,x:b.getBoundingClientRect().x,right:b.getBoundingClientRect().right,height:b.getBoundingClientRect().height}))}));
      const g=result.geometry;
      if(g.pageScrollWidth>g.pageWidth+1||g.componentScrollWidth>g.componentWidth+1)throw new Error('Vendor overview overflows horizontally.');
      if(g.buttons.some(b=>b.x<0||b.right>width+1||b.height<44))throw new Error('Vendor overview action is outside the viewport or smaller than44px.');
      await page.addScriptTag({content:axe});
      result.axe=await page.evaluate(async()=>{const a=await axe.run(document,{runOnly:{type:'rule',values:['color-contrast']}});return{violations:a.violations,incomplete:a.incomplete.map(item=>({id:item.id,nodes:item.nodes.length}))}});
      if(result.axe.violations.length)throw new Error('Axe reports a contrast violation.');
      await page.screenshot({path:path.join(output,'after',`${name}.png`),fullPage:true,animations:'disabled'});
      if(integrated) await page.screenshot({path:path.join(output,'after',`${name}-viewport.png`),fullPage:false,animations:'disabled'});
      result.initialViewport=await overview.evaluate(el=>({overviewTop:el.getBoundingClientRect().top,firstRequirementTop:el.querySelector('.vendor-compliance__items')?.getBoundingClientRect().top??null,firstRequirementBottom:el.querySelector('.vendor-compliance__items > li')?.getBoundingClientRect().bottom??null,navigationTop:document.querySelector('.mobile-nav')?.getBoundingClientRect().top??null}));
      if(integrated&&state==='gaps'&&result.initialViewport.firstRequirementBottom>(width<700?result.initialViewport.navigationTop:900))throw new Error('The initial viewport does not expose the first failed requirement above navigation.');
      if(state==='gaps') {
        const review=overview.getByRole('button',{name:'Review Sample · Card processing service checks',exact:true});
        const primary=overview.getByRole('button',{name:'Review response',exact:true});
        await primary.focus();await page.keyboard.press('Enter');
        const sheet=page.getByRole('dialog',{name:'Review Sample · Card processing service checks',exact:true});await sheet.waitFor();
        await sheet.getByText('Does the contract include audit rights?',{exact:true}).waitFor();
        const calls=await page.evaluate(()=>window.vendorComplianceEvidenceReads);
        if(!calls.some(path=>path==='/api/v1/forms/responses/sample-compliance-response/assessment'))throw new Error('Response review did not read the exact selected response.');
        await page.screenshot({path:path.join(output,'after',`${name}-review.png`),fullPage:false,animations:'disabled'});
        await page.keyboard.press('Escape');await sheet.waitFor({state:'hidden'});
        await page.waitForFunction(()=>document.activeElement?.textContent==='Review response');
        await review.focus();await page.keyboard.press('Enter');await sheet.waitFor();await page.keyboard.press('Escape');await sheet.waitFor({state:'hidden'});
        await page.waitForFunction(()=>document.activeElement?.getAttribute('aria-label')==='Review Sample · Card processing service checks');
        result.keyboardReview='Primary and row actions opened the exact response; Escape restored each originating action.';
      }
      if(!integrated && state==='pagination') {
        const initial=await overview.locator('.vendor-compliance__forms > li').count();
        await overview.getByRole('button',{name:'Load more forms',exact:true}).click();
        await overview.getByText('Sample · Privacy and data protection review',{exact:true}).waitFor();
        if(initial!==1||await overview.locator('.vendor-compliance__forms > li').count()!==2)throw new Error('Pagination did not append the second authorized page.');
        const calls=await page.evaluate(()=>window.vendorComplianceEvidenceReads);
        if(!calls.some(path=>path.includes('cursor=sample-overview-page-2')))throw new Error('Pagination did not send the stored cursor.');
        result.pagination='Second page appended with exact cursor.';
        await page.screenshot({path:path.join(output,'after',`${name}-page-2.png`),fullPage:true,animations:'disabled'});
      }
      if(!integrated && state==='unavailable') {
        await overview.getByRole('button',{name:'Retry',exact:true}).click();
        await overview.getByText('No forms requested',{exact:true}).waitFor();
        result.retry='Reloaded the authorized empty population after recovery.';
        await page.screenshot({path:path.join(output,'after',`${name}-recovered.png`),fullPage:true,animations:'disabled'});
      }
      if(!integrated && ['empty','satisfactory','incomplete','reused'].includes(state)) {
        const action=state==='incomplete'?overview.getByRole('button',{name:/^Open .* request$/}):state==='reused'?overview.getByRole('button',{name:/^Review evidence for /}):overview.locator('.vendor-compliance__heading').getByRole('button');
        await action.focus();await page.keyboard.press('Enter');
        const target=state==='empty'?'Request vendor form':state==='incomplete'?'Vendor form request':'Vendor due diligence';
        await page.getByRole('region',{name:'Sample destination'}).getByRole('heading',{name:target,exact:true}).waitFor();
        result.navigation=`Opened ${target} from the visible action.`;
      }
      if(errors.length)throw new Error(errors.join('; '));
      result.status='PASS';
    } catch(error) {result.status='FAIL';result.error=error.message;await page.screenshot({path:path.join(output,'after',`${name}-failure.png`),fullPage:true}).catch(()=>{})}
    results.push(result);console.log(`${name}: ${result.status}${result.error?' '+result.error:''}`);
    await writeFile(path.join(output,'after',`${name}.json`),JSON.stringify(result,null,2));
    await writeFile(path.join(output,process.env.RECEIPT || (integrated?'integrated-receipt.json':'receipt.json')),JSON.stringify({generatedAt:new Date().toISOString(),base,browser:browser.version(),sourceSha256,results},null,2));
    await context.close();
  }
} finally {await browser.close()}
if(results.some(item=>item.status==='FAIL'))process.exitCode=1;
