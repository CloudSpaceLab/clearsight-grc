import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import {inspectRenderedContrast} from '../../../../web/scripts/rendered-contrast.mjs';
const {chromium}=createRequire(import.meta.url)('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out=new URL('./select-value-interaction/',import.meta.url);await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true}),results=[];
try {for(const theme of ['light','dark'])for(const width of [390,320])for(const kind of ['priority','gallery']){
 const context=await browser.newContext({viewport:{width,height:844},colorScheme:theme,reducedMotion:'reduce'});
 await context.addInitScript(t=>localStorage.setItem('clearsight.theme',t),theme);
 const page=await context.newPage();page.setDefaultTimeout(9000);
 await page.goto(kind==='priority'?'http://127.0.0.1:4179/?tour=off&fixture=forms-response-history#forms':'http://127.0.0.1:4179/?tour=off&fixture=ui-component-gallery#ui-components',{waitUntil:'networkidle'});
 if(kind==='priority'){await page.getByRole('tab',{name:'Responses',exact:true}).click();await page.getByRole('button',{name:'Review Vendor due diligence review response'}).waitFor();}
 const field=page.locator('.cs-select-field').filter({has:page.locator('.cs-select-field__label',{hasText:kind==='priority'?'Priority':'Sample response status'})});
 const trigger=field.locator('.cs-select-field__trigger');await trigger.scrollIntoViewIfNeeded();await page.waitForTimeout(200);await trigger.click();await page.getByRole('listbox').waitFor();
 const guidance=kind==='priority'?'Highest adverse score, then most recent':'Recipients can no longer edit responses.';
 await page.getByRole('listbox').getByText(guidance,{exact:true}).waitFor();
 const id=`${kind}-${theme}-${width}`;await page.screenshot({path:new URL(`${id}-open.png`,out).pathname.replace(/^\/([A-Z]:)/,'$1')});
 if(kind==='gallery')await page.getByRole('option',{name:/Responses locked/}).click();else await page.keyboard.press('Escape');
 const value=await field.locator('.cs-select-field__value').innerText();
 if(value!==(kind==='priority'?'Needs attention first':'Responses locked'))throw new Error(`Unexpected selected text: ${value}`);
 await trigger.scrollIntoViewIfNeeded();await page.screenshot({path:new URL(`${id}-closed.png`,out).pathname.replace(/^\/([A-Z]:)/,'$1')});
 results.push({id,value,guidanceRetained:true,contrast:await page.evaluate(inspectRenderedContrast),layout:await page.evaluate(()=>({clientWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth}))});
 await context.close();
}}finally{await browser.close();await writeFile(new URL('receipt.json',out),JSON.stringify({generatedAt:new Date().toISOString(),results},null,2));}
