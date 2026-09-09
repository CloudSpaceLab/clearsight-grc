import {createRequire} from 'node:module';import {writeFile} from 'node:fs/promises';import path from 'node:path';
const require=createRequire(import.meta.url),{chromium}=require('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out=path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Z]:)/,'$1'));const browser=await chromium.launch({headless:true});const results=[];
for(const theme of ['light','dark']){
 const context=await browser.newContext({viewport:{width:1440,height:900},colorScheme:theme,reducedMotion:'reduce'});await context.addInitScript(t=>localStorage.setItem('clearsight.theme',t),theme);const page=await context.newPage();await page.goto(`http://127.0.0.1:4179/?tour=off&fixture=ui-component-gallery#ui-components`,{waitUntil:'networkidle'});
 const input=page.locator('input.cs-field__control:not(:disabled)').first();await input.scrollIntoViewIfNeeded();await page.screenshot({path:path.join(out,`gallery-fields-${theme}.png`)});
 const raw=await input.evaluate(e=>{const s=getComputedStyle(e),p=getComputedStyle(e.parentElement),ph=getComputedStyle(e,'::placeholder');return{html:e.outerHTML,color:s.color,background:s.backgroundColor,border:s.borderColor,parentBackground:p.backgroundColor,placeholderColor:ph.color,placeholderOpacity:ph.opacity}});
 await input.focus();await page.keyboard.press('Tab');await page.keyboard.press('Shift+Tab');await page.screenshot({path:path.join(out,`gallery-fields-${theme}-focus.png`)});
 const focus=await input.evaluate(e=>{const s=getComputedStyle(e);return{outline:s.outline,outlineOffset:s.outlineOffset,border:s.borderColor,boxShadow:s.boxShadow,focusVisible:e.matches(':focus-visible')}});
 const styles=await input.evaluate(e=>{const matches=[];function walk(rules){for(const r of rules){if(r.cssRules){walk(r.cssRules);continue}try{if(r.selectorText&&(e.matches(r.selectorText)||r.selectorText.includes('placeholder'))&&(r.style?.cssText.includes('color')||r.style?.cssText.includes('outline')))matches.push({selector:r.selectorText,style:r.style.cssText})}catch{}}}for(const s of document.styleSheets){try{walk(s.cssRules)}catch{}}return matches});
 results.push({theme,raw,focus,styles});await context.close();
}
await writeFile(path.join(out,'focus.json'),JSON.stringify(results,null,2));console.log(JSON.stringify(results.map(({theme,raw,focus})=>({theme,raw,focus}))));await browser.close();
