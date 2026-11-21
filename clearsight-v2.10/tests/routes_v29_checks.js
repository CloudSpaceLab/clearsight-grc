/* Source-level route/action integration. This is not a browser layout test. */
'use strict';
const fs=require('fs'),path=require('path'),assert=require('assert');
const {createRuntime,root}=require('./runtime_harness');
const h=createRuntime(),results=[];
const entries=JSON.parse(fs.readFileSync(path.join(root,'docs/route-manifest.json'),'utf8'));
const seedRoutes={
 program:'program?id=PRG-104',assessment:'assessment?id=PRG-104&target=Meridian%20Bank',vendor:'vendor?id=ENG-01',
 risk:'risk?id=RSK-01',loss:'loss?id=LOS-201',control:'control?id=CTL-01',
 'evidence-review':'evidence-review?id=EV-01','requirement-review':'requirement-review?id=PRG-104&target=Meridian%20Bank&requirement=R01',
 'kri-return':'kri-return?id=KRC-BRANCH-2026-10%3A%3AB-EN',
 'notification':'notification?id=NTF-MENTION-01','role-detail':'role-detail?id=ROLE-CONTRIBUTOR',
 'escalation-policy':'escalation-policy?id=ESC-KRI','indicator':'indicator?id=KRI-01','control-editor':'control-editor?id=CTL-01','risk-edit':'risk-edit?id=RSK-01'
};
for(const name of [...new Set(Object.values(entries))]){
 try{
  const html=h.navigate(seedRoutes[name]||name);
  assert(html.length>100,'No page content');assert(!/\bNaN\b/.test(html),'NaN shown in UI');assert(!html.includes('[object Object]'),'Object rendered as a label');
  for(const [,action] of html.matchAll(/data-action="([^"]+)"/g))assert(h.run(`typeof actions[${JSON.stringify(action)}]==='function'`),'Unwired action: '+action);
  for(const [,route] of html.matchAll(/href="#\/([^"?]+)/g))assert(h.run(`!!titles[${JSON.stringify(route)}]`),'Unknown route: '+route);
  results.push({route:name,status:'PASS',bytes:html.length});
 }catch(error){results.push({route:name,status:'FAIL',error:String(error)});console.error('FAIL',name,error.stack)}
}
for(const view of ['executive','locations','drivers','kri','assurance','dependencies','quality','library','loss-operations']){
 try{const html=h.navigate('reports?view='+view);assert(html.length>500);assert(!/\bNaN\b/.test(html));results.push({route:'reports?view='+view,status:'PASS'});}catch(error){results.push({route:'reports?view='+view,status:'FAIL',error:String(error)});console.error('FAIL report',view,error.stack)}
}
fs.writeFileSync(path.join(root,'tests/routes-v29-results.json'),JSON.stringify({runtime:'Node actual-source; browser DOM stubbed',scope:'Route functions, registered actions and navigation destinations; record-specific workflow tests are separate',results},null,2));
console.log(`${results.filter(x=>x.status==='PASS').length}/${results.length} route/action render checks passed.`);
if(results.some(x=>x.status==='FAIL'))process.exitCode=1;
