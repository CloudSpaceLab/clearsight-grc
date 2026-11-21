/* Node checks of actual notification/domain logic. Minimal DOM shell, no domain stubs.
   These do not replace browser layout or interaction verification. */
'use strict';
const fs=require('fs'),vm=require('vm'),path=require('path'),assert=require('assert');
const root=path.resolve(__dirname,'..'),listeners={},results=[];
class ModelFormData{constructor(form){this.data=form?._values||{}}*[Symbol.iterator](){yield* Object.entries(this.data)}get(name){return this.data[name]??null}getAll(name){const value=this.data[name];return value==null?[]:Array.isArray(value)?value:[value]}}
const context={FormData:ModelFormData,console,TextEncoder,TextDecoder,URL,URLSearchParams,Blob,Intl,Date,JSON,Map,Set,Math,Number,String,Object,Array,Error,Promise,setTimeout,clearTimeout,crypto:require('crypto').webcrypto,location:{hash:''},localStorage:{getItem(){return null},setItem(){}},requestAnimationFrame(){},scrollTo(){},innerWidth:1440,scrollY:0,document:{body:{dataset:{start:'home'}},documentElement:{dataset:{}},addEventListener(type,fn){(listeners[type]??=[]).push(fn)},querySelector(q){return q==='#dialog'?{addEventListener(){}}:null},querySelectorAll(){return []},getElementById(){return null}}};
context.window=context;context.addEventListener=()=>{};vm.createContext(context);
for(const file of ['seed','app','forms','experience','journeys','insights','records','document-store','documents','usability','copilot','kri-schema','branch-kri','governance','inbox','board-reports']){
 let source=fs.readFileSync(path.join(root,'assets',file+'.js'),'utf8').replace('recordsInit();docSeed();render();','');
 vm.runInContext(source,context,{filename:file+'.js'});
}
vm.runInContext('scopeInit();recordsInit();docSeed();gvInit();inboxInit();render=function(){};',context);
const evaluate=code=>vm.runInContext(code,context);
function check(name,fn){try{fn();results.push({test:name,status:'PASS'});console.log('PASS',name)}catch(error){results.push({test:name,status:'FAIL',error:String(error)});console.error('FAIL',name,error)}}
check('Inbox derives records and builds its complete list/detail views',()=>{
 assert(evaluate('nItems().length')>=10);
 assert(evaluate('inboxView().includes("All notifications")'));
 evaluate("current={name:'notification',params:new URLSearchParams('id=NTF-MENTION-01')};");
 assert(evaluate('nDetailView().includes("Conversation")'));
 assert(evaluate("!!nReceipt(nEvent('NTF-MENTION-01')).readAt"));
});
check('Repeated monitoring deduplicates stable events and retains read receipts',()=>{
 const before=evaluate('S.notificationsV28.events.length');evaluate('nItems();nItems();');
 assert.equal(evaluate('S.notificationsV28.events.length'),before);
 assert(evaluate("!!nReceipt(nEvent('NTF-MENTION-01')).readAt"));
});
check('Recipients have isolated personal history on shared events',()=>{
 evaluate("nReceipt(nEvent('NTF-PROGRAM-01')).readAt=nNow();S.governance.actorId='USR-03';");
 assert(!evaluate("nItems().some(x=>x.id==='NTF-MENTION-01')"));
 assert.equal(evaluate("nReceipt(nEvent('NTF-PROGRAM-01')).readAt"),null);
 assert(evaluate("nItems().some(x=>x.type==='approval')"));
 evaluate("S.governance.actorId='USR-01'");
});
check('Scope filters both notification text and direct detail access',()=>{
 evaluate("S.scope={branch:'B-IK',department:'',unit:''};scopeInit();");
 assert(!evaluate("nItems().some(x=>x.id==='NTF-MENTION-01')"));
 assert(evaluate("nItems().every(x=>nScopeIds(x).some(id=>orgWithin(id,'B-IK')||orgWithin('B-IK',id)))"));
 assert(evaluate('nDetailView().includes("Notification unavailable")'));
 evaluate("S.scope={branch:'',department:'',unit:''};scopeInit();");
});
check('Invalid stored scope does not widen to bank-wide notifications',()=>{
 evaluate("S.scope={branch:'B-UNKNOWN',department:'',unit:''};scopeInit();");
 assert.equal(evaluate('nItems().length'),0);
 evaluate("S.scope={branch:'',department:'',unit:''};scopeInit();");
});
check('Snoozes use time and resume without modifying the source',()=>{
 evaluate("nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil='2099-01-01T00:00:00Z';");
 assert(!evaluate("nFiltered().some(x=>x.id==='NTF-MENTION-01')"));
 evaluate("N_FILTER.view='snoozed';");assert(evaluate("nFiltered().some(x=>x.id==='NTF-MENTION-01')"));
 evaluate("nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil='2000-01-01T00:00:00Z';nItems();N_FILTER.view='inbox';");
 assert.equal(evaluate("nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil"),null);
});
check('Future optional preferences do not suppress critical events or existing history',()=>{
 evaluate("nPrefs().inApp.program=false;CS_V28_EVENTS.emit({id:'TEST-OPTIONAL',type:'program',title:'Optional',sourceRoute:'program?id=PRG-104',sourceId:'PRG-104',recipient:'USR-01'});CS_V28_EVENTS.emit({id:'TEST-CRITICAL',type:'program',title:'Critical',sourceRoute:'program?id=PRG-104',sourceId:'PRG-104',recipient:'USR-01',severity:'Critical'});");
 assert(!evaluate("nItems().some(x=>x.id==='TEST-OPTIONAL')"));
 assert(evaluate("nItems().some(x=>x.id==='TEST-CRITICAL')"));
 assert(evaluate("nItems().some(x=>x.id==='NTF-PROGRAM-01')"));
});
check('Source references are exact; mismatched route IDs are rejected',()=>{
 assert.equal(evaluate("nSourceRoute(nEvent('NTF-TASK-ACT-03'))"),'evidence-review?id=EV-02');
 assert.throws(()=>evaluate("CS_V28_EVENTS.emit({type:'program',title:'Invalid',sourceRoute:'program?id=PRG-104',sourceId:'PRG-105',recipient:'USR-01'})"),/does not match/);
});
check('Acknowledgement is distinct from resolution; completion is source-owned',()=>{
 assert.equal(evaluate("nResolved(nEvent('NTF-ESC-ACT-03'))"),false);
 evaluate("nReceipt(nEvent('NTF-ESC-ACT-03')).acknowledgedAt=nNow()");
 assert.equal(evaluate("nResolved(nEvent('NTF-ESC-ACT-03'))"),false);
 evaluate("S.tasks.find(x=>x.id==='ACT-03').status='Completed'");
 assert.equal(evaluate("nResolved(nEvent('NTF-ESC-ACT-03'))"),true);
});
check('Program activity creates a notification linked to the actual program',()=>{
 evaluate("nPrefs().inApp.program=true;audit('Assessment submitted for Meridian Bank','PRG-104','Chidi Okafor');nItems();");
 assert(evaluate("nItems().some(x=>x.id.startsWith('NTF-ACTIVITY-')&&x.sourceId==='PRG-104')"));
});
check('Notification titles, bodies and thread content are escaped in HTML',()=>{
 evaluate("CS_V28_EVENTS.emit({id:'TEST-ESCAPE',type:'mention',title:'<script>alert(1)</script>',body:'<img onerror=alert(2)>',sourceRoute:'program?id=PRG-104',sourceId:'PRG-104',recipient:'USR-01',thread:[{id:'MSG-X',actor:'USR-03',text:'<script>evil()</script>',createdAt:nNow()}]});current={name:'notification',params:new URLSearchParams('id=TEST-ESCAPE')};");
 const html=evaluate('nDetailView()');assert(html.includes('&lt;script&gt;'));assert(!html.includes('<script>'));
});
check('Integrated source modules build board, governance and KRI views without conflicts',()=>{
 evaluate("S.governance.actorId='USR-01';S.scope={branch:'',department:'',unit:''};scopeInit();current={name:'kri-returns',params:new URLSearchParams()};");
 assert(evaluate('k28Overview().includes("Branch")'));
 assert(evaluate('gvRolesView().length')>1000);
 assert(evaluate('gvMatrixView().length')>1000);
 evaluate("current={name:'reports',params:new URLSearchParams()};");
 assert(evaluate('reportsView().length')>1000);
 assert(evaluate('reportData().board.kriDefinition.metrics.length')>=50);
});
check('Lowercase priorities retain their meaning across KRI and board events',()=>{
 evaluate("CS_V28_EVENTS.emit({id:'TEST-HIGH-CASE',type:'kri',title:'Breach',sourceRoute:'kri-return?id='+encodeURIComponent(ixReturnKey('KRC-BRANCH-2026-10','B-IK')),sourceId:ixReturnKey('KRC-BRANCH-2026-10','B-IK'),recipient:'Amina Bello',severity:'high'});CS_V28_EVENTS.emit({id:'TEST-MED-CASE',type:'program',title:'Update',sourceRoute:'work',recipient:'USR-01',severity:'medium'});");
 assert.equal(evaluate("S.notificationsV28.events.find(x=>x.id==='TEST-HIGH-CASE').severity"),'High');
 assert.equal(evaluate("S.notificationsV28.events.find(x=>x.id==='TEST-MED-CASE').severity"),'Normal');
});
check('Actual KRI submission emits a revision-bound approval and review resolves it',()=>{
 evaluate("S.governance.actorId='USR-04';current={name:'kri-return',params:new URLSearchParams('id='+encodeURIComponent(ixReturnKey('KRC-BRANCH-2026-10','B-EN')))};openDialog=(title,subtitle,body,label,callback)=>{globalThis.MODEL_DIALOG={title,body,callback}};toast=()=>{};ixSubmitReturn();");
 assert(evaluate('!!MODEL_DIALOG.callback'));
 evaluate("MODEL_DIALOG.callback({_values:{attest:'on'}});");
 const eventId=evaluate("S.notificationsV28.events.find(x=>x.id.startsWith('kri-submit-')&&x.sourceId===ixReturnKey('KRC-BRANCH-2026-10','B-EN')).id");
 context.eventId=eventId;
 assert.equal(evaluate('S.notificationsV28.events.find(x=>x.id===eventId).sourceRevision'),1);
 assert.equal(evaluate('nResolved(S.notificationsV28.events.find(x=>x.id===eventId))'),false);
 evaluate("S.governance.actorId='USR-03';ixReviewReturn({_values:{reviewer:'Ifeoma Nwosu',decision:'Approve return',note:'Inspection references and exception follow-up responsibilities checked independently.'}});");
 assert.equal(evaluate('nResolved(S.notificationsV28.events.find(x=>x.id===eventId))'),true);
 assert(evaluate("S.notificationsV28.events.some(x=>x.id.startsWith('kri-review-')&&x.sourceState==='Approved'&&!x.actionable)"));
});
check('Accepted KRI exception remains open until its own remediation task is completed',()=>{
 evaluate("S.governance.actorId='USR-04';");
 const key=evaluate("S.notificationsV28.events.find(x=>x.id.startsWith('kri-exception-')&&x.sourceId===ixReturnKey('KRC-BRANCH-2026-10','B-EN')).id");
 context.exceptionId=key;
 assert.equal(evaluate('nResolved(S.notificationsV28.events.find(x=>x.id===exceptionId))'),false);
 const linked=evaluate('S.notificationsV28.events.find(x=>x.id===exceptionId).sourceTaskId||S.notificationsV28.events.find(x=>x.id===exceptionId).kriObservation');
 assert(linked,'KRI emitter must provide sourceTaskId or kriObservation for exact closure');
 evaluate("{const event=S.notificationsV28.events.find(x=>x.id===exceptionId),task=event.sourceTaskId?S.tasks.find(t=>t.id===event.sourceTaskId):S.tasks.find(t=>t.kriObservation===event.kriObservation);task.status='Completed';}");
 assert.equal(evaluate('nResolved(S.notificationsV28.events.find(x=>x.id===exceptionId))'),true);
});
check('Changes-requested KRI notification resolves when a correction is resubmitted',()=>{
 evaluate("{const r=ixCurrentReturn();r.status='Changes requested';CS_V28_EVENTS.emit({id:'TEST-KRI-CORRECTION',type:'program',title:'Changes requested',sourceRoute:ixReturnRoute(r.id),sourceId:r.id,recipient:r.owner,sourceRevision:r.revision,sourceState:r.status});}");
 assert(evaluate("S.notificationsV28.events.find(x=>x.id==='TEST-KRI-CORRECTION').actionable"));
 assert(!evaluate("nResolved(S.notificationsV28.events.find(x=>x.id==='TEST-KRI-CORRECTION'))"));
 evaluate("ixSubmitReturn();MODEL_DIALOG.callback({_values:{attest:'on'}});");
 assert(evaluate("nResolved(S.notificationsV28.events.find(x=>x.id==='TEST-KRI-CORRECTION'))"));
});
check('Board-style work notification links to the actual task and its completion',()=>{
 evaluate("S.governance.actorId='USR-01';S.tasks.push({id:'ACT-INTEGRATION',title:'Board follow-up',owner:'Amina Bello',ownerId:'USR-01',due:S.asOf,status:'Open',kind:'Remediation',ref:'RSK-01',orgScopeId:'U-HO-SEC'});CS_V28_EVENTS.emit({id:'TEST-BOARD-WORK',type:'program',title:'New exception follow-up',sourceRoute:'work',sourceId:'ACT-INTEGRATION',orgScopeId:'U-HO-SEC',recipient:'Amina Bello',severity:'warning'});");
 assert.equal(evaluate("S.notificationsV28.events.find(x=>x.id==='TEST-BOARD-WORK').sourceType"),'task');
 assert(evaluate("S.notificationsV28.events.find(x=>x.id==='TEST-BOARD-WORK').actionable"));
 assert.equal(evaluate("nSourceRoute(nEvent('TEST-BOARD-WORK'))"),'risk?id=RSK-01');
 evaluate("S.tasks.find(x=>x.id==='ACT-INTEGRATION').status='Completed';");
 assert(evaluate("nResolved(nEvent('TEST-BOARD-WORK'))"));
});
fs.writeFileSync(path.join(root,'tests/inbox-v28-model-results.json'),JSON.stringify({kind:'Node domain-model checks; browser DOM mocked',checks:results},null,2));
if(results.some(x=>x.status==='FAIL'))process.exitCode=1;
