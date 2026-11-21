/* Actual-source integration harness for restricted environments without a
 * browser process. DOM rendering/layout is deliberately not claimed here. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),vm=require('node:vm'),assert=require('node:assert/strict');
const ROOT=path.resolve(__dirname,'..'),results=[];
const fakeElement=()=>({innerHTML:'',textContent:'',dataset:{},style:{},open:false,hidden:false,value:'',classList:{add(){},remove(){},toggle(){},contains(){return false;}},addEventListener(){},setAttribute(){},removeAttribute(){},getAttribute(){return null;},append(){},prepend(){},remove(){},insertAdjacentHTML(){},focus(){},scrollIntoView(){},close(){},showModal(){},querySelector(){return null;},querySelectorAll(){return[];},closest(){return null;},matches(){return false;}});
const elements=new Map(),body=fakeElement(),store={};body.dataset.start='reports';
const context={console,URL,URLSearchParams,TextEncoder,TextDecoder,Blob,Date,Math,JSON,Map,Set,Array,Object,String,Number,Boolean,RegExp,Promise,Intl,parseInt,parseFloat,isNaN,Error,Uint8Array,ArrayBuffer,structuredClone,crypto:require('node:crypto').webcrypto,navigator:{},location:{hash:'#/reports'},localStorage:{getItem:k=>store[k]??null,setItem:(k,v)=>store[k]=String(v),removeItem:k=>delete store[k]},document:{body,documentElement:fakeElement(),addEventListener(){},querySelector:s=>{if(!elements.has(s))elements.set(s,fakeElement());return elements.get(s);},querySelectorAll(){return[];},createElement:fakeElement},setTimeout:()=>0,clearTimeout(){},requestAnimationFrame(){},scrollTo(){},scrollY:0,innerWidth:1440,addEventListener(){}};
context.window=context;vm.createContext(context);
const files=['seed','app','forms','experience','journeys','insights','records','document-store','documents','usability','copilot','kri-schema','branch-kri'];
for(const f of files){const p=path.join(ROOT,'assets',f+'.js');if(!fs.existsSync(p))continue;let text=fs.readFileSync(p,'utf8');if(f==='copilot')text=text.replace('recordsInit();docSeed();render();','');vm.runInContext(text,context,{filename:f+'.js'});}
const run=src=>vm.runInContext(src,context),json=src=>JSON.parse(JSON.stringify(run(src)));
run('scopeInit();faInit();journeyInit();ixInit();recordsInit();docSeed();render=()=>{};current=route();values=form=>form;');
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS',name);}catch(err){results.push({name,status:'FAIL',error:err.stack});console.error('FAIL',name,err.message);}}


function revise(mutator=''){
 run("S.scope={branch:'',department:'',unit:''};location.hash='#/kri-return?id=KRC-BRANCH-2026-10%3A%3AB-IK';current=route();var kr=ixCurrentReturn();var ks=clone(kr.accepted);ks.revision=kr.revision+1;ks.id=kr.id+'::v'+ks.revision;ks.submittedBy=kr.owner;ks.submittedAt=S.asOf;delete ks.reviewedAt;delete ks.reviewedBy;delete ks.reviewNote;");
 if(mutator)run(mutator);
 run("kr.revision=ks.revision;kr.submissions.push(ks);kr.status='Submitted';");
}
const decision="{reviewer:'Ifeoma Nwosu',decision:'Approve return',note:'Independent review of exact source observations and corrective commitments.'}";
test('V2.8 acceptance creates one shared decision receipt and versioned remediation actions',()=>{
 const before=run('S.governance.receipts.length');revise();run('ixReviewReturn('+decision+')');
 assert.equal(run('S.governance.receipts.length'),before+1);
 assert.equal(run("S.governance.receipts.at(-1).subject.type"),'kriReturn');
 assert.equal(run('kr.status'),'Approved');assert.equal(run('kr.accepted.revision'),2);
 assert.equal(run("S.tasks.filter(t=>t.kriObservation?.startsWith(kr.id+'::')).length"),4);
 assert.ok(run("S.tasks.filter(t=>t.kriObservation?.startsWith(kr.id+'::')).every(t=>t.sourceRevision===2&&t.ownerId)"));
});
test('Accepted correction updates one open task and retains its previous accountability',()=>{
 run("var originalTask=S.tasks.find(t=>t.kriObservation===kr.id+'::cashShort');var taskId=originalTask.id;");
 revise("ks.values.cashShort.detail.owner='Daniel Mensah';ks.values.cashShort.detail.due='2026-11-25';ks.values.cashShort.detail.action='Reconcile the corrected loss schedule and independently verify closure.';");run('ixReviewReturn('+decision+')');
 assert.equal(run("S.tasks.filter(t=>t.kriObservation===kr.id+'::cashShort').length"),1);
 assert.equal(run('originalTask.id'),run('taskId'));assert.equal(run('originalTask.owner'),'Daniel Mensah');assert.equal(run('originalTask.sourceRevision'),3);
 assert.equal(run('originalTask.sourceHistory.at(-1).previous.owner'),'Amina Bello');
});
test('A closed action remains intact when a later accepted breach creates another episode',()=>{
 run("originalTask.status='Completed';var oldClosed=JSON.stringify(originalTask);");
 revise("ks.values.cashShort.detail.action='Investigate recurrence and reconcile the new cash discrepancy.';");run('ixReviewReturn('+decision+')');
 assert.equal(run("S.tasks.filter(t=>t.kriObservation===kr.id+'::cashShort').length"),2);
 assert.equal(run('JSON.stringify(originalTask)'),run('oldClosed'));
 assert.equal(run("S.tasks.find(t=>t.kriObservation===kr.id+'::cashShort'&&t.status==='Open').previousActionId"),run('taskId'));
});
test('A nonbreaching correction flags the open action for closure verification without closing it',()=>{
 revise('ks.values.cashShort.value=0;');run('ixReviewReturn('+decision+')');
 assert.equal(run("S.tasks.find(t=>t.kriObservation===kr.id+'::cashShort'&&t.status==='Open').sourceObservationChanged.condition"),'Within limit');
});
test('Malformed submission provenance fails before mutation or a review receipt',()=>{
 const receipts=run('S.governance.receipts.length'),accepted=run('JSON.stringify(kr.accepted)');
 revise("ks.definitionVersion='WRONG-DEFINITION';");
 assert.throws(()=>run('ixReviewReturn('+decision+')'),/definition does not match/);
 assert.equal(run('S.governance.receipts.length'),receipts);assert.equal(run('JSON.stringify(kr.accepted)'),accepted);
});
test('Legacy KRI review keeps exactly one original shared decision receipt',()=>{
 run("location.hash='#/kri-return?id=KRC-2026-10%3A%3AU-IK-CUSTOMER';current=route();");
 const before=run('S.governance.receipts.length');
 run("ixReviewReturn({reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'Please reconcile the recorded source and resubmit the exact revision.'})");
 assert.equal(run('S.governance.receipts.length'),before+1);
});
test('Recording a continuous indicator appends traceable dated history without inventing legacy dates',()=>{
 run("var indicator=S.indicators[0];var legacyHistoryLength=indicator.history.length;var datedHistoryLength=indicator.observations?.length||0;observeIndicator(indicator.id);modalSave({value:'3.5',observed:S.asOf,note:'Measured by independent source at the snapshot date.'});");
 assert.equal(run('indicator.history.length'),run('legacyHistoryLength')+1);
 assert.equal(run('indicator.observations.length'),run('datedHistoryLength')+1);
 assert.equal(run('indicator.observations.at(-1).date'),run('S.asOf'));
 assert.equal(run('indicator.observations.at(-1).value'),3.5);
 assert.ok(run('indicator.observations.at(-1).sourceRef.length')>10);
});
const result={suite:'kri-v28-review-integration',validation:'Actual domain functions in Node VM; DOM shell only',checks:results,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length};fs.writeFileSync(path.join(ROOT,'tests/kri-review-v28-results.json'),JSON.stringify(result,null,2));console.log(JSON.stringify({passed:result.passed,failed:result.failed}));process.exitCode=result.failed?1:0;
