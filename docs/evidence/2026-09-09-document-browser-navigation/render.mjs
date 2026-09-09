import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const {chromium}=createRequire(import.meta.url)('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out=path.dirname(fileURLToPath(import.meta.url));await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true}),results=[];
try {for(const theme of ['light','dark'])for(const width of [1440,390,320]){
 const context=await browser.newContext({viewport:{width:1440,height:900},colorScheme:theme,reducedMotion:'reduce'});
 await context.addInitScript(t=>localStorage.setItem('clearsight.theme',t),theme);
 const page=await context.newPage();page.setDefaultTimeout(10000);
 await page.goto(`${process.env.PAGE_URL ?? 'http://127.0.0.1:4187'}/?tour=off&fixture=forms-response-history#forms`,{waitUntil:'networkidle'});
 await page.getByRole('tab',{name:'Documents',exact:true}).click();
 const host=page.locator('.document-browser');await host.waitFor();await page.waitForTimeout(400);
 await page.setViewportSize({width,height:width<760?844:900});await page.evaluate(()=>document.fonts.ready);
 const id=`documents-${theme}-${width}`;
 await host.screenshot({path:path.join(out,`${id}-all.png`)});
 if(width>760){await page.getByRole('navigation',{name:'File types'}).getByRole('button',{name:'PDF files',exact:true}).click();}
 else{await host.locator('.document-kind-selector button').click();await page.getByRole('option',{name:'PDF files',exact:true}).click();}
 await page.waitForTimeout(400);
 const geometry=await host.evaluate(e=>({
  navVisible:getComputedStyle(e.querySelector('.document-kinds')).display!=='none',
  selectorVisible:getComputedStyle(e.querySelector('.document-kind-selector')).display!=='none',
  buttons:[...e.querySelectorAll('.document-kinds > button')].map(b=>{const s=b.querySelector('.document-kind-label');return{label:s.children[1].textContent,selected:b.getAttribute('aria-pressed'),height:b.getBoundingClientRect().height,iconX:s.children[0].getBoundingClientRect().x,labelX:s.children[1].getBoundingClientRect().x,labelHeight:s.children[1].getBoundingClientRect().height}}),
  clientWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth,
  selectedText:e.querySelector('.document-kind-selector .cs-select-field__value').textContent,
 }));
 assert.equal(geometry.scrollWidth,geometry.clientWidth);assert.equal(geometry.selectedText,'PDF files');
 assert.equal(geometry.buttons.filter(b=>b.selected==='true').length,1);
 if(width>760){assert.equal(geometry.navVisible,true);assert.equal(geometry.selectorVisible,false);assert.equal(new Set(geometry.buttons.map(b=>b.iconX)).size,1);assert.equal(new Set(geometry.buttons.map(b=>b.labelX)).size,1);assert.ok(geometry.buttons.every(b=>b.height>=44));assert.equal(new Set(geometry.buttons.map(b=>b.labelHeight)).size,1);}
 else {assert.equal(geometry.navVisible,false);assert.equal(geometry.selectorVisible,true);}
 await host.screenshot({path:path.join(out,`${id}-pdf.png`)});
 if(width>760){const button=page.getByRole('navigation',{name:'File types'}).getByRole('button',{name:'PDF files',exact:true});await button.focus();await page.keyboard.press('Tab');await page.keyboard.press('Shift+Tab');geometry.keyboardFocus=await button.evaluate(e=>({visible:e.matches(':focus-visible'),outline:getComputedStyle(e).outline}));assert.equal(geometry.keyboardFocus.visible,true);await host.screenshot({path:path.join(out,`${id}-focus.png`)});}
 results.push({id,...geometry});await context.close();
}}finally{await browser.close();await writeFile(path.join(out,'receipt.json'),JSON.stringify({generatedAt:new Date().toISOString(),results},null,2));}
