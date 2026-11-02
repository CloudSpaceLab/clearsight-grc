import { createRequire } from 'node:module';
import { createHash } from 'node:crypto';
import { readFile, writeFile, readdir } from 'node:fs/promises';
import path from 'node:path';
const {chromium}=createRequire(import.meta.url)('/tmp/clearsight-v2-tools/node_modules/playwright');
const root='/tmp/people-v2-baseline-source';
const out='/workspace/scratch/992dce7dae5c/people-evidence/before-canonical';
const digest=(v)=>createHash('sha256').update(v).digest('hex');
async function files(dir){let values=[];for(const f of await readdir(dir,{withFileTypes:true})){const p=path.join(dir,f.name);values.push(...(f.isDirectory()?await files(p):[p]));}return values;}
const build=[];for(const file of (await files('/tmp/people-v2-canonical-baseline-dist')).sort())build.push({path:path.relative('/tmp/people-v2-canonical-baseline-dist',file),sha256:digest(await readFile(file))});
const inputs=[];for(const file of ['web/src/evidenceMain.tsx','web/src/peopleAccessEvidence.ts']) inputs.push({path:file,sha256:digest(await readFile(path.join(root,file)))});
const captures=[]; const browser=await chromium.launch({headless:true});
try {
for(const view of ['organization','directory','authority'])for(const theme of ['light','dark'])for(const viewport of [{width:1440,height:900},{width:1280,height:800},{width:1024,height:768},{width:768,height:900},{width:390,height:844}]){
 const context=await browser.newContext({viewport,colorScheme:theme,reducedMotion:'reduce',locale:'en-NG',timezoneId:'Africa/Lagos',hasTouch:viewport.width<=1024});
 await context.addInitScript(theme=>{localStorage.setItem('clearsight.theme',theme);localStorage.setItem('clearsight.density','comfortable')},theme);
 const page=await context.newPage();const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('http://127.0.0.1:4180/?tour=off&fixture=people-v2-current#configure/'+(view==='authority'?'authority':'access'),{waitUntil:'networkidle'});
 await page.getByRole('heading',{name:'Configuration',exact:true}).waitFor();
 if(view==='directory'){await page.getByRole('tab',{name:'Directory groups & access',exact:true}).click();await page.getByText('Amina Yusuf',{exact:true}).first().waitFor();}
 else if(view==='organization'){await page.getByRole('heading',{name:'Positions & roles',exact:true}).waitFor({state:'attached'});}
 else await page.getByRole('heading',{name:'Governance policies and delegations',exact:true}).waitFor();
 await page.evaluate(()=>{window.scrollTo(0,0);document.querySelector('.main')?.scrollTo(0,0)});
 const geometry=await page.evaluate(view=>{const box=e=>{if(!e)return null;const b=e.getBoundingClientRect();return {x:b.x,y:b.y,width:b.width,height:b.height,bottom:b.bottom,right:b.right}};const exact=(selector,text)=>[...document.querySelectorAll(selector)].find(e=>e.textContent?.trim()===text);return {documentWidth:document.documentElement.clientWidth,scrollWidth:document.documentElement.scrollWidth,heading:box(document.querySelector('.configure-header')),firstPerson:box(exact('strong','Amina Yusuf')),firstPolicy:box(exact('h4','Material risk approval')),directory:box(exact('h3','People & groups')),localTabs:[...document.querySelectorAll('.identity-access-tabs [role=tab]')].map(e=>e.textContent?.trim()),peopleVisible:[...document.querySelectorAll('.identity-directory-columns>div:first-child .identity-mini-row strong')].map(e=>e.textContent),reads:window.peopleAccessEvidence?.reads,commands:window.peopleAccessEvidence?.commands}},view);
 const name=`before-${view}-${theme}-${viewport.width}.png`;await page.screenshot({path:path.join(out,name),fullPage:false,animations:'disabled',caret:'hide'});
 captures.push({name,view,theme,viewport,geometry,errors,sha256:digest(await readFile(path.join(out,name)))});await context.close();console.log(name,geometry.firstPerson?.y??geometry.firstPolicy?.y??'no-first-record');
}
}finally{await browser.close()}
await writeFile(path.join(out,'manifest.json'),JSON.stringify({runtimeRevision:'ea84283f913755be6dc1b8e366282469745b3e9a',runtimeTree:'28f0b24ef1da098b46bfd345c466cecd79051675',fixture:'people-v2-current',fixtureNote:'Old runtime plus isolated evidence transport; no customer runtime changes.',build,buildDigest:digest(JSON.stringify(build)),inputs,inputDigest:digest(JSON.stringify(inputs)),captures},null,2));
