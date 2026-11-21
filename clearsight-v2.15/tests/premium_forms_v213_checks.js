/* Focused V2.13 presentation assertions against the integrated source runtime.
 * The DOM shell is substituted. These are not browser layout or device checks. */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const h=createRuntime(),j=JSON.stringify,results=[];
function test(name,fn){
 try{fn();results.push({name,status:'PASS'});console.log('PASS',name);}
 catch(error){results.push({name,status:'FAIL',error:error.message});console.error('FAIL',name,error.stack);}
}
const routes=[
 ['program?id=PRG-104&tab=assessments','USR-01','response'],
 ['assessment?id=PRG-104&target=Meridian%20Bank','USR-01','response'],
 ['form-response?form=FORM-VENDOR','USR-03','response'],
 ['form-builder?form=FORM-VENDOR','USR-03','response'],
 ['kri-return?id=KRC-BRANCH-2026-10%3A%3AB-IK','USR-01','kri'],
 ['kri-cycle?id=KRC-BRANCH-2026-10','USR-03','kri'],
 ['loss?id=LOS-201','USR-03','loss'],
 ['loss-import','USR-01','loss'],
 ['vendor?id=ENG-01','USR-03','vendor'],
 ['vendor-requests?id=ENG-01','USR-03','vendor'],
 ['document?id=DOC-SAMPLE-POLICY','USR-03','document'],
 ['document?id=DOC-SAMPLE-POLICY&tab=details','USR-03','document'],
 ['document?id=DOC-SAMPLE-POLICY&tab=versions','USR-03','document'],
 ['document-upload','USR-01','document'],
 ['requirement-review?id=PRG-104&target=Meridian%20Bank&requirement=R01','USR-03','review'],
 ['external-assessments','USR-05','external']
];

test('Sixteen actual route variants receive only the correct presentation family',()=>{
 for(const [route,actor,family] of routes){
  h.run(`S.governance.actorId=${j(actor)}`);h.navigate(route);
  assert.equal(h.elements.get('#main-content').dataset.pf213,family,route);
  const html=h.elements.get('#main-content').innerHTML,stored=h.run('recStable(S)');
  h.run('pf213Refresh()');
  assert.equal(h.elements.get('#main-content').innerHTML,html,route+' field markup changed');
  assert.equal(h.run('recStable(S)'),stored,route+' domain state changed');
 }
});

test('Route transitions clear the family instead of restyling unrelated pages',()=>{
 h.navigate('my-workspace');assert(!('pf213' in h.elements.get('#main-content').dataset));
 h.navigate('document?id=DOC-SAMPLE-POLICY');assert.equal(h.elements.get('#main-content').dataset.pf213,'document');
 h.navigate('reports');assert(!('pf213' in h.elements.get('#main-content').dataset));
});

test('Saved vendor and loss step forms keep native names, draft keys and submission controls',()=>{
 h.run("S.governance.actorId='USR-01';var pfLoss=l29NewDraft();var pfVendor=voStart()");
 for(const [route,fieldMarker,family] of [
  ['loss-intake?id='+h.run('pfLoss.id'),'data-l29-form="capture"','loss'],
  ['vendor-intake?id='+h.run('pfVendor.id'),'data-vo-form="intake"','vendor']
 ]){
  const html=h.navigate(route);assert(html.includes(fieldMarker),route);assert(html.includes('name="intent"'),route);
  assert.equal(h.elements.get('#main-content').dataset.pf213,family);
  const before=h.run('recStable(S)');h.run('pf213Refresh()');assert.equal(h.run('recStable(S)'),before);
 }
});

test('Recipient decoration leaves the entire original HTML, manifest, CSP and runtime unchanged',()=>{
 h.context.pfPayload={manifest:{title:'Example source <not markup>',requestId:'PF213',recipient:{name:'Named Person'}},manifestHash:'unchanged',referenceFiles:[]};
 const original=h.run('pf213OriginalRecipientHTML(pfPayload)'),styled=h.run('ex212RecipientHTML(pfPayload)'),marker='<style id="pf213-recipient-style">',start=styled.indexOf(marker),end=styled.indexOf('</style>',start)+8;
 assert(start>0);assert.equal(styled.slice(0,start)+styled.slice(end),original);
 assert.equal((styled.match(/<script/g)||[]).length,2);assert(styled.includes("connect-src 'none'"));
 const css=styled.slice(start+marker.length,end-8);assert(!/@import|url\s*\(|<script|<\/style/i.test(css));assert(css.includes('--blue:#245dd8'));
});

test('Existing accessible labels are retained and scrolling preview can receive keyboard focus',()=>{
 function node(attrs={}){return {attrs:{...attrs},hasAttribute(k){return Object.hasOwn(this.attrs,k)},setAttribute(k,v){this.attrs[k]=v}};}
 const preview=node(),bar=node({'aria-label':'Original source action label',role:'group'}),nav=node({'aria-label':'Existing question navigation'}),main=h.elements.get('#main-content'),query=main.querySelectorAll;
 main.querySelectorAll=s=>s==='.doc-preview'?[preview]:s==='.l29-footer'?[bar]:s==='.ax211-response-nav nav'?[nav]:[];
 try{
  h.navigate('document?id=DOC-SAMPLE-POLICY');
  assert.equal(preview.attrs.tabindex,'0');assert.equal(preview.attrs.role,'region');assert.equal(preview.attrs['aria-label'],'Selected document preview');
  assert.equal(bar.attrs['aria-label'],'Original source action label');assert.equal(nav.attrs['aria-label'],'Existing question navigation');
 }finally{main.querySelectorAll=query;}
});

test('Presentation API exposes no record commands or operational store',()=>{
 assert.deepEqual(Object.keys(h.context.CLEAR_SIGHT_PREMIUM_FORMS).sort(),['family','refresh','version']);
 assert.equal(h.context.CLEAR_SIGHT_PREMIUM_FORMS.version,'2.13');
});

const report={
 runtime:'Actual integrated V2.13 JavaScript with a substituted DOM shell; no browser layout or device claim',
 scope:'Six focused presentation assertions, including sixteen route variants and exact recipient HTML equivalence after removing only the added stylesheet',
 routeVariants:routes.length,
 results
};
fs.writeFileSync(path.join(root,'tests/premium-forms-v213-results.json'),JSON.stringify(report,null,2)+'\n');
console.log(JSON.stringify({passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,routeVariants:routes.length}));
if(results.some(x=>x.status==='FAIL'))process.exitCode=1;
