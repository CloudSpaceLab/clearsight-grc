import {createRequire} from 'node:module';
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import path from 'node:path';
const require=createRequire(import.meta.url);
const {chromium}=require('C:/Users/Son/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const out=path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Z]:)/,'$1'));
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
function inspect(){
 const colorCanvas=document.createElement('canvas');colorCanvas.width=1;colorCanvas.height=1;const colorContext=colorCanvas.getContext('2d',{willReadFrequently:true});const colorCache=new Map();
 const parse=s=>{if(!s)return null;if(!colorCache.has(s)){colorContext.clearRect(0,0,1,1);colorContext.fillStyle=s;colorContext.fillRect(0,0,1,1);const c=[...colorContext.getImageData(0,0,1,1).data];colorCache.set(s,[...c.slice(0,3),c[3]/255])}return [...colorCache.get(s)]};
 const over=(f,b)=>[0,1,2].map(i=>f[i]*f[3]+b[i]*(1-f[3])).concat(1);
 const luminance=c=>c.slice(0,3).map(v=>v/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4).reduce((s,v,i)=>s+v*[.2126,.7152,.0722][i],0);
 const contrast=(a,b)=>(Math.max(luminance(a),luminance(b))+.05)/(Math.min(luminance(a),luminance(b))+.05);
 const hex=c=>'#'+c.slice(0,3).map(v=>Math.round(v).toString(16).padStart(2,'0')).join('');
 const sel=e=>e.id?'#'+CSS.escape(e.id):e.tagName.toLowerCase()+[...e.classList].map(c=>'.'+CSS.escape(c)).join('');
 function background(e){let chain=[],uncertain=[],group=[];for(let n=e;n;n=n.parentElement){const s=getComputedStyle(n);chain.unshift(s);if(+s.opacity!==1)group.push('element/group opacity');if(s.filter!=='none')group.push('filter')};let color=[255,255,255,1];for(const s of chain){const c=parse(s.backgroundColor);if(c){if(c[3]===1)uncertain=[];color=over(c,color)}if(s.backgroundImage!=='none')uncertain.push('background image')}return{color,uncertain:[...new Set([...uncertain,...group])]}}
 const measurements=[];
 for(const e of document.querySelectorAll('body *')){
  const s=getComputedStyle(e),r=e.getBoundingClientRect();if(!r.width||!r.height||s.visibility==='hidden'||s.display==='none'||e.closest('[aria-hidden="true"], [inert]'))continue;
  const disabled=e.matches(':disabled,[aria-disabled="true"]')||!!e.closest(':disabled,[aria-disabled="true"]');
  const bg=background(e);const text=[...e.childNodes].filter(n=>n.nodeType===3).map(n=>n.textContent).join(' ').trim();
  const base={selector:sel(e),text:text.slice(0,120),disabled,background:hex(bg.color),uncertain:bg.uncertain,box:{x:r.x,y:r.y,width:r.width,height:r.height},fontSize:s.fontSize,fontWeight:s.fontWeight};
  if(text){const fg=parse(s.color);if(fg){const large=parseFloat(s.fontSize)>=24||(parseFloat(s.fontSize)>=18.66&&+s.fontWeight>=700);measurements.push({...base,kind:'text',foreground:hex(over(fg,bg.color)),ratio:contrast(over(fg,bg.color),bg.color),minimum:large?3:4.5})}}
  if(e.matches('input[placeholder],textarea[placeholder]')&&!e.value){const p=getComputedStyle(e,'::placeholder'),fg=parse(p.color);if(fg){fg[3]*=+p.opacity;measurements.push({...base,text:e.getAttribute('placeholder'),kind:'placeholder',foreground:hex(over(fg,bg.color)),ratio:contrast(over(fg,bg.color),bg.color),minimum:4.5})}}
  if(e.matches('input,textarea,select,button,[role="button"],[role="checkbox"],[role="combobox"]')){
   const adjacent=background(e.parentElement);const border=parse(s.borderTopColor);if(border&&parseFloat(s.borderTopWidth)>0&&s.borderTopStyle!=='none')measurements.push({...base,kind:'control-border',foreground:hex(over(border,adjacent.color)),background:hex(adjacent.color),ratio:contrast(over(border,adjacent.color),adjacent.color),minimum:3,interpretation:'Review whether this boundary is necessary to identify the control; low ratio alone is not a WCAG failure.'});
  }
 }
 return measurements;
}
const results=[];await mkdir(out,{recursive:true});
try{for(const [name,fixture,route,action] of bases.filter(b=>!process.env.AUDIT_CASES||process.env.AUDIT_CASES.split(',').includes(b[0])))for(const theme of ['light','dark'])for(const width of [1440,390]){
 const id=`${name}-${theme}-${width}`,context=await browser.newContext({viewport:{width,height:width===1440?900:844},colorScheme:theme,reducedMotion:'reduce',hasTouch:width===390});
 await context.addInitScript(t=>{localStorage.setItem('clearsight.theme',t);localStorage.setItem('clearsight.density','comfortable')},theme);
 const page=await context.newPage();page.setDefaultTimeout(9000);const errors=[];page.on('pageerror',e=>errors.push(e.message));let result={id,name,fixture,route,action,theme,width,errors};
 try{
  const url=route.startsWith('/')?`http://127.0.0.1:4179${route}?tour=off&fixture=${fixture}#form_access=task22-${fixture}`:`http://127.0.0.1:4179/?tour=off&fixture=${fixture}${route}`;
  await page.goto(url,{waitUntil:'networkidle'});await page.evaluate(()=>document.fonts.ready);
  if(action==='builder'){await page.getByRole('button',{name:'Open Compliance scoring review'}).click();await page.getByRole('button',{name:'Edit draft',exact:true}).click();await page.getByLabel('Form canvas').waitFor()}
  if(action==='sent')await page.getByRole('tab',{name:'Sent forms',exact:true}).click();
  if(action==='response'){await page.getByRole('tab',{name:'Responses',exact:true}).click();await page.getByRole('button',{name:'Review Vendor due diligence review response'}).click()}
  if(['checklist','select','review'].includes(action)){
   await page.getByRole('button',{name:/Acme Processing Limited.*Card transaction processing/}).click();await page.locator('.vendor-checklist').waitFor();await page.locator('.vendor-checklist').scrollIntoViewIfNeeded();
   if(action==='select'){await page.getByRole('button',{name:'Missing 1',exact:true}).click();await page.getByRole('button',{name:'Use existing document',exact:true}).click();await page.getByRole('dialog',{name:/Use existing document for/}).waitFor()}
   if(action==='review'){await page.locator('.vendor-checklist').getByRole('article',{name:'ISO 27001 assurance',exact:true}).getByRole('button',{name:'Review document',exact:true}).click();await page.getByRole('dialog',{name:'Review document',exact:true}).waitFor()}
  }
  if(action==='gallery-select'){const trigger=page.getByRole('button',{name:/Sample response status/});await trigger.scrollIntoViewIfNeeded();await page.waitForTimeout(150);await trigger.click();await page.getByRole('listbox').waitFor()}
  await page.screenshot({path:path.join(out,id+'.png'),fullPage:true});
  await page.addScriptTag({content:axeSource});
  result.axe=await page.evaluate(async()=>{const a=await axe.run(document,{runOnly:{type:'rule',values:['color-contrast']}});return{version:axe.version,violations:a.violations,incomplete:a.incomplete,passes:a.passes.map(p=>({id:p.id,nodes:p.nodes.length})),inapplicable:a.inapplicable.map(p=>p.id)}});
  result.measurements=await page.evaluate(inspect);
  result.headings=await page.locator('h1,h2').allTextContents();result.status='CAPTURED';
  if(name==='gallery'){
   await page.keyboard.press('Tab');const focused=await page.evaluate(()=>{const e=document.activeElement,s=getComputedStyle(e);return{tag:e.tagName,text:e.textContent?.trim(),outline:s.outline,boxShadow:s.boxShadow,background:s.backgroundColor,color:s.color}});result.keyboardFocus=focused;await page.screenshot({path:path.join(out,id+'-focus.png'),fullPage:true});
  }
 }catch(e){result.status='ERROR';result.error=e.message;await page.screenshot({path:path.join(out,id+'-error.png'),fullPage:true}).catch(()=>{})}
 await writeFile(path.join(out,id+'.json'),JSON.stringify(result,null,2));results.push({id,status:result.status,error:result.error,errors,violations:result.axe?.violations.reduce((n,v)=>n+v.nodes.length,0),incomplete:result.axe?.incomplete.reduce((n,v)=>n+v.nodes.length,0)});console.log(JSON.stringify(results.at(-1)));await context.close();
 await writeFile(path.join(out,'summary.json'),JSON.stringify({generatedAt:new Date().toISOString(),results},null,2));
}}finally{await browser.close()}
