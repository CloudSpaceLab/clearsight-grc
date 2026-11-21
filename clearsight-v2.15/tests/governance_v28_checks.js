/* Node model and render-contract tests. These do not replace browser layout QA. */
'use strict';
const fs=require('fs'),vm=require('vm'),assert=require('assert'),path=require('path');
const root=path.resolve(__dirname,'..'),seed=JSON.parse(fs.readFileSync(path.join(root,'assets/seed.json'),'utf8'));
const source=fs.readFileSync(path.join(root,'assets/governance.js'),'utf8');
const orgSource=fs.readFileSync(path.join(root,'assets/experience.js'),'utf8').split('const ORG = ')[1].split('function scopeInit(){')[0];
const context=vm.createContext({console,URLSearchParams,Date,JSON,Number,Object,Array,String,Set,Map,Math});
const helpers=`
let S=${JSON.stringify(seed)};S.scope={branch:'',department:'',unit:''};S.governance={actorId:'USR-05'};
const ORG = ${orgSource}
const REC_PEOPLE=S.people.map((name,index)=>({id:'USR-'+String(index+1).padStart(2,'0'),name}));
const clone=x=>JSON.parse(JSON.stringify(x));const e=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let seq=0,modalSave=null;const uid=p=>p+'-'+(++seq),save=()=>{},audit=()=>{},go=r=>{current={name:r.split('?')[0],params:new URLSearchParams(r.split('?')[1]||'')}};
const selectedScope=()=>S.scope.unit||S.scope.department||S.scope.branch||'';
const fail=m=>{throw new Error(m)},requireValue=(b,m)=>{if(!b)throw new Error(m)},bounded=(v,a,b,n)=>{const x=Number(v);requireValue(Number.isInteger(x)&&x>=a&&x<=b,n+' invalid');return x};
const days=(a,b)=>Math.floor((Date.parse(a+'T00:00:00Z')-Date.parse(b+'T00:00:00Z'))/86400000),addDays=(x,n)=>new Date(Date.parse(x+'T00:00:00Z')+n*86400000).toISOString().slice(0,10);
const validDate=(v,label)=>{requireValue(/^\\d{4}-\\d{2}-\\d{2}$/.test(v)&&new Date(v+'T00:00:00Z').toISOString().slice(0,10)===v,label+' invalid');return v};
const recActor=()=>REC_PEOPLE[4],recTimestamp=x=>new Date(x).toISOString(),date=x=>x||'—';
const icon=x=>'<svg data-icon="'+e(x)+'"></svg>',btn=(t,a,extra='',c='')=>'<button type="button" class="'+c+'" data-action="'+a+'" '+extra+'>'+t+'</button>',link=(t,r,c='')=>'<a class="'+c+'" href="#/'+r+'">'+t+'</a>',pill=(t,c='')=>'<span class="'+c+'">'+e(t)+'</span>',panel=(t,s,b)=>'<section><h2>'+t+'</h2><p>'+s+'</p>'+b+'</section>',header=(t,s,a='',eb='')=>'<header><small>'+eb+'</small><h1>'+t+'</h1><p>'+s+'</p>'+a+'</header>',notice=(t)=>'<aside>'+t+'</aside>',empty=(t,d)=>'<h3>'+t+'</h3><p>'+d+'</p>',metric=(t,v,n)=>'<div>'+t+':'+v+' '+n+'</div>',metricRow=x=>x.join('');
const values=f=>f,toast=()=>{},download=()=>{},openDialog=(title,subtitle,body,label,cb)=>{modalSave=cb};
let current={name:'roles',params:new URLSearchParams()};const actions={},titles={},document={addEventListener:()=>{}},window={CS_JOURNEYS:{},CS_JOURNEY_GROUPS:{}};let settingsView=()=>'',render=()=>{};const $=()=>null;
`;
vm.runInContext(helpers,context);vm.runInContext(source,context);
const run=s=>vm.runInContext(s,context),results=[];
function test(name,fn){try{run('delete S.accessControl;gvInit();');fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:String(error)});console.error('FAIL '+name+' '+error.stack);}}
test('Scoped grants, missing grants, self-approval and expired assignments remain distinct',()=>{
 assert.equal(run("gvDecision('USR-01','kri.submit','U-IK-PAY').allowed"),true);
 assert.equal(run("gvDecision('USR-01','kri.submit','U-EN-REC').allowed"),false);
 assert.equal(run("gvDecision('USR-01','kri.approve','U-IK-PAY').allowed"),false);
 assert.equal(run("gvDecision('USR-03','kri.approve','U-IK-PAY').allowed"),true);
 assert.equal(run("gvDecision('USR-03','kri.approve','U-IK-PAY',S.asOf,{makerId:'USR-03'}).allowed"),false);
 assert.equal(run("gvDecision('USR-04','documents.export','U-EN-REC').allowed"),false);
 assert.equal(run("gvDecision('USR-04','documents.export','U-EN-REC','2026-11-09').allowed"),true);
});
test('Explicit deny overrides all role grants in its scope, and unknown IDs fail closed',()=>{
 run("gvState().assignments.push({id:'TEST',userId:'USR-01',roleId:'ROLE-AUDITOR',scopeId:'B-IK',starts:'2026-01-01',ends:'',status:'Active'})");
 assert.equal(run("gvDecision('USR-01','documents.export','U-IK-PAY').allowed"),false);
 assert.equal(run("gvDecision('USR-01','reports.export','U-IK-PAY').allowed"),true);
 assert.equal(run("gvDecision('USR-99','kri.view','U-IK-PAY').allowed"),false);
 assert.equal(run("gvDecision('USR-03','kri.view','INVALID').allowed"),false);
});
test('Role lifecycle preserves drafts, rejects SoD conflict and records an accepted policy',()=>{
 run("gvNewRole();Object.assign(gvState().draftRole,{name:'Branch document observer',description:'Read scoped document records for review.',reason:'Provide controlled document review for branch supervisors.'});gvState().draftRole.permissions={'documents.view':'allow','documents.edit':'allow','documents.approve':'allow'}");
 assert.throws(()=>run('gvSaveRole()'),/preparation and approval/);
 assert.equal(run('gvState().roles.length'),6);
 run("delete gvState().draftRole.permissions['documents.approve'];gvSaveRole()");
 assert.equal(run('gvState().roles.length'),7);assert.equal(run('gvState().audit.length'),1);
});
test('Protected system role and final active bank administrator cannot be changed or removed',()=>{
 run("gvDraftRole('ROLE-ADMIN')");assert.throws(()=>run('gvSaveRole()'),/Protected/);
 assert.throws(()=>run("gvKeepAdmin('GVA-01')"),/at least one active/);
 run("gvState().assignments.push({...clone(gvState().assignments[0]),id:'ADMIN-2',userId:'USR-02'})");
 run("gvKeepAdmin('GVA-01')");
});
test('Future-dated overlapping SoD assignments are rejected, disjoint locations are permitted',()=>{
 assert.ok(run("gvConflictFor({id:'NEW',userId:'USR-01',roleId:'ROLE-REVIEWER',scopeId:'B-IK',starts:'2026-12-01',ends:'2027-01-01',status:'Active'}).length")>0);
 assert.equal(run("gvConflictFor({id:'NEW',userId:'USR-01',roleId:'ROLE-REVIEWER',scopeId:'B-EN',starts:'2026-12-01',ends:'2027-01-01',status:'Active'}).length"),0);
});
test('Assignment form rejects role scope widening and duplicate dated assignments',()=>{
 run("gvRole('ROLE-CONTRIBUTOR').scopeRoot='B-IK';gvAssignDialog()");
 assert.throws(()=>run("modalSave({userId:'USR-04',roleId:'ROLE-CONTRIBUTOR',scopeId:'',starts:'2026-11-13',ends:'',reason:'Wide assignment should be rejected.'})"),/exceeds/);
 assert.throws(()=>run("modalSave({userId:'USR-01',roleId:'ROLE-CONTRIBUTOR',scopeId:'B-IK',starts:'2026-11-13',ends:'',reason:'Duplicate assignment should be rejected.'})"),/overlapping assignment/);
});
test('Direct rules require independent authorized reviewer and catch descendant SoD conflicts',()=>{
 run('gvDirectDialog()');
 assert.throws(()=>run("modalSave({userId:'USR-01',approvedBy:'USR-01',permission:'kri.approve',effect:'allow',scopeId:'',starts:'2026-11-13',ends:'2026-12-01',reason:'Independent approval validation.'})"),/cannot independently/);
 assert.throws(()=>run("modalSave({userId:'USR-01',approvedBy:'USR-03',permission:'kri.approve',effect:'allow',scopeId:'',starts:'2026-11-13',ends:'2026-12-01',reason:'Authorized reviewer validation.'})"),/access-management/);
 assert.throws(()=>run("modalSave({userId:'USR-01',approvedBy:'USR-05',permission:'kri.approve',effect:'allow',scopeId:'',starts:'2026-11-13',ends:'2026-12-01',reason:'Organization grant cannot override branch separation.'})"),/conflicts with preparation/);
});
test('Calendar hours retain 24/7 time while business hours skip the weekend in WAT',()=>{
 assert.equal(run("gvBusinessTime(new Date('2026-11-13T15:00:00Z'),4).toISOString()"),'2026-11-16T10:00:00.000Z');
 assert.equal(run("gvBusinessTime(new Date('2026-11-14T09:00:00Z'),0).toISOString()"),'2026-11-16T07:00:00.000Z');
 assert.equal(run("gvBusinessTime(new Date('2026-11-13T05:00:00Z'),1).toISOString()"),'2026-11-13T08:00:00.000Z');
 assert.equal(run("gvSimulate({...gvState().simulator,at:'2026-11-13T16:00'}).levels[1].due"),'2026-11-13T19:00:00.000Z');
});
test('Escalation resolves actual role holders and applies the fallback for missing owner',()=>{
 assert.equal(run("gvSimulate({...gvState().simulator,ownerId:''}).levels[0].recipients[0].id"),'USR-03');
 assert.equal(run("gvSimulate({...gvState().simulator,ownerId:''}).levels[0].fallback"),'ROLE-REVIEWER');
 run("gvState().assignments.find(a=>a.roleId==='ROLE-REVIEWER').status='Revoked'");
 assert.equal(run("gvSimulate({...gvState().simulator,ownerId:''}).levels[0].status"),'Unassigned');
});
test('Acknowledgement and resolution stop only the configured future levels',()=>{
 assert.equal(run("gvSimulate({...gvState().simulator,ackAfter:'1'}).levels[1].status"),'Scheduled');
 assert.equal(run("gvSimulate({...gvState().simulator,resolvedAfter:'1'}).levels[1].status"),'Stopped');
 assert.equal(run("gvSimulate({...gvState().simulator,policyId:'ESC-QUALITY',ackAfter:'1'}).levels[1].status"),'Stopped');
 assert.equal(run("gvSimulate({...gvState().simulator,policyId:'ESC-QUALITY',ackAfter:'1'}).levels[0].status"),'Scheduled');
});
test('Policy scope, severity and active state are applied before scheduling',()=>{
 assert.throws(()=>run("gvSimulate({...gvState().simulator,severity:'Low'})"),/below/);
 run("gvState().policies[0].scopeId='B-EN'");assert.throws(()=>run("gvSimulate(gvState().simulator)"),/does not cover/);
 run("gvState().policies[0].status='Inactive'");assert.throws(()=>run("gvSimulate(gvState().simulator)"),/active/);
});
test('Policy matching prefers the specific location and highest matching severity',()=>{
 run("gvState().policies.push({...clone(gvState().policies[0]),id:'ESC-LOCAL',scopeId:'B-IK',severity:'High'})");
 assert.equal(run("gvMatchPolicy('kri-breach','Critical','U-IK-PAY').id"),'ESC-LOCAL');
 assert.equal(run("gvMatchPolicy('kri-breach','Critical','U-EN-REC').id"),'ESC-KRI');
 assert.equal(run("gvMatchPolicy('kri-breach','Medium','U-IK-PAY')"),null);
});
test('Policy save validates increasing offsets and fallback then creates a revision',()=>{
 run("gvPolicyDraft('ESC-KRI');gvState().draftPolicy.reason='Improve material breach response and accountable follow-up.';gvState().draftPolicy.levels[1].after=0");
 assert.throws(()=>run('gvSavePolicy()'),/later than/);
 run("gvState().draftPolicy.levels[1].after=6;gvSavePolicy()");
 assert.equal(run("gvState().policies[0].version"),2);assert.equal(run("gvState().policies[0].levels[1].after"),6);
});
test('All governance routes render complete action markup and permission matrix labels',()=>{
 for(const [route,id] of [['roles',''],['role-detail','ROLE-CONTRIBUTOR'],['access-review',''],['escalation-matrix',''],['escalation-policy','ESC-KRI']]){
  const html=run(`current={name:${JSON.stringify(route)},params:new URLSearchParams(${JSON.stringify(id?'id='+id:'')})};window.CS_JOURNEYS[current.name]()`);
  assert.ok(html.includes('<h1>'));assert.ok(!html.includes('undefined'));assert.ok(!html.includes('NaN'));
  for(const [,action] of html.matchAll(/data-action="([^"]+)"/g))assert.ok(run(`typeof actions[${JSON.stringify(action)}]`)==='function',action+' is wired');
 }
 const matrix=run("gvPermissionMatrix(gvRole('ROLE-CONTRIBUTOR'))");assert.equal((matrix.match(/data-gv-permission=/g)||[]).length,48);assert.equal((matrix.match(/aria-label=/g)||[]).length,49);
});
fs.writeFileSync(path.join(root,'tests/governance-v28-results.json'),JSON.stringify({suite:'governance-v28-model-and-render-contract',browserLayoutValidated:false,results},null,2)+'\n');
if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
