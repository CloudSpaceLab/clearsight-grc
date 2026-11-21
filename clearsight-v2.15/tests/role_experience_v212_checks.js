/* Actual-source role experience checks. Browser/DOM shell only is substituted. */
'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const {createRuntime,root}=require('./runtime_harness');
const results=[],cases=[],test=(name,fn)=>cases.push({name,fn});
function runtime(){const h=createRuntime();h.run("wf29Init();current={name:'my-workspace',params:new URLSearchParams()};values=f=>f.values||f;render=()=>{};");return h;}
function actor(h,id){h.run(`S.governance.actorId=${JSON.stringify(id)};`);}
function scope(h,id){h.run(`{const p=orgPath(${JSON.stringify(id)});S.scope={branch:p.find(n=>n.kind==='branch')?.id||'',department:p.find(n=>n.kind==='department')?.id||'',unit:p.find(n=>n.kind==='unit')?.id||''};S.scopeInvalid=false;}`);}
function deny(h,permission,scopeId='U-IK-PAY',userId='USR-01'){h.run(`gvState().directRules.push({id:uid('RE212-DENY'),userId:${JSON.stringify(userId)},permission:${JSON.stringify(permission)},effect:'deny',scopeId:${JSON.stringify(scopeId)},starts:S.asOf,ends:'',status:'Active',reason:'A precise restriction for this synthetic role check.'});`);}
function testDraft(h){h.run("wf29StartTest('CTL-03');Object.assign(wf29Draft('control-test','CTL-03').values,{tester:'Amina Bello',reviewer:'Ifeoma Nwosu',population:'100',sample:'20',exceptions:'0',method:'A reproducible sample from the full scoped payment population.',results:'Every sampled payment retained the documented dual authorization.',evidenceRef:'RE212-WORKPAPER-01',recommendation:'Effective'});");}
function submitTest(h){testDraft(h);h.run("wf29SubmitTest('CTL-03');var RE212_TEST=wf29State().controlTests.at(-1);");}
function decision(h,reviewer='Ifeoma Nwosu',choice='Approve test'){return h.run(`wf29TestDecision({dataset:{id:RE212_TEST.id},values:{reviewer:${JSON.stringify(reviewer)},decision:${JSON.stringify(choice)},note:'The source population, sample and result have been independently reviewed.'}})`);}

test('Exact capabilities preserve scope, active dates, explicit denies and actual actor requirements',h=>{
 assert(h.run("re212Capability('risks.edit',{scopeId:'U-IK-PAY'}).allowed"));assert(!h.run("re212Capability('risks.edit',{scopeId:'U-HO-SEC'}).allowed"));assert(!h.run("re212Capability('risks.edit',{scopeId:''}).allowed"));
 assert(!h.run("re212Capability('documents.export',{scopeId:'U-IK-PAY'}).allowed"));assert(!h.run("re212Capability('documents.export',{scopeId:'U-EN-REC',userId:'USR-04'}).allowed"));
 actor(h,'USR-03');assert.throws(()=>h.run("re212Require('risks.edit',{scopeId:'U-IK-PAY',userId:'USR-01'})"),/No active grant/);
 assert(!h.run("re212Capability('risks.approve',{scopeId:'U-IK-PAY',makerId:'USR-03'}).allowed"));assert(!h.run("re212Capability('unknown.edit',{scopeId:''}).allowed"));assert(!h.run("re212Capability('risks.approve',{scopeId:'U-IK-PAY',userId:''}).allowed"));
});
test('Coverage reports partial subtrees without turning an ancestor grant into a child-deny bypass',h=>{
 scope(h,'B-IK');assert(h.run("re212Coverage('risks.edit').full"));deny(h,'risks.edit');const c=JSON.parse(h.run("JSON.stringify(re212Coverage('risks.edit'))"));assert(c.any);assert(!c.full);assert(c.deniedScopeIds.includes('U-IK-PAY'));assert(!c.roots.includes('B-IK'));assert(c.roots.includes('D-IK-RETAIL'));
 assert(!h.run("re212Capability('risks.edit',{scopeIds:['U-IK-CUSTOMER','U-IK-PAY']}).allowed"));scope(h,'');assert(h.run("re212Coverage('risks.edit').any"));assert(!h.run("re212Coverage('risks.edit').full"));
});
test('Shared references use scoped read coverage while shared mutation and export remain exact',h=>{
 h.run("var RE_DOC={id:'DOC-RE212-SHARED',title:'Shared policy',scopeMode:'shared',orgScopeId:'',classification:'Restricted',ownerId:'USR-05',archived:false};S.governance.documents.push(RE_DOC);var RE_SCOPED={id:'DOC-RE212-SCOPED',title:'Scoped instruction',scopeMode:'scoped',orgScopeId:'U-IK-PAY',classification:'Restricted',ownerId:'USR-05',archived:false};S.governance.documents.push(RE_SCOPED);");scope(h,'B-IK');
 assert(h.run("re212CanViewSubject('document',RE_DOC.id).allowed"));assert(h.run("re212CanViewSubject('document',RE_SCOPED.id).allowed"));assert(!h.run("re212Capability('documents.edit',{subject:{type:'document',id:RE_DOC.id}}).allowed"));assert(!h.run("re212Capability('documents.export',{subject:{type:'document',id:RE_DOC.id}}).allowed"));
 deny(h,'documents.view','B-IK');assert(!h.run("re212CanViewSubject('document',RE_DOC.id).allowed"));assert(!h.run("re212CanViewSubject('document',RE_SCOPED.id).allowed"));
});
test('Missing subject scope and invalid navigation never widen to whole-organization permission',h=>{
 actor(h,'USR-03');h.run("S.governance.documents.push({id:'DOC-RE212-NOSCOPE',title:'Missing location',scopeMode:'scoped',ownerId:'USR-03'});");assert(!h.run("re212CanViewSubject('document','DOC-RE212-NOSCOPE').allowed"));assert(!h.run("re212Capability('documents.view',{scopeIds:[]}).allowed"));h.run('S.scopeInvalid=true;');assert(!h.run("re212Coverage('documents.view').any"));assert(!h.run("re212Capability('documents.view',{scopeId:''}).allowed"));
});
test('Role lenses derive from current grants and verified ownership, never from title text',h=>{
 assert(h.run("re212Context().lenses.some(l=>l.id==='contributor')"));assert(h.run("re212Context().lenses.some(l=>l.id==='owner')"));assert(!h.run("re212Context().lenses.some(l=>l.id==='administrator')"));h.run("gvRole('ROLE-CONTRIBUTOR').name='Board chief administrator';");assert(!h.run("re212Context().lenses.some(l=>l.id==='leadership'||l.id==='administrator')"));
 actor(h,'USR-05');assert(h.run("re212Context().lenses.some(l=>l.id==='administrator')"));assert(!h.run("re212Context().lenses.some(l=>l.id==='contributor')"));
 actor(h,'USR-03');assert(h.run("re212Context().lenses.some(l=>l.id==='reviewer')"));assert(h.run("re212Context().lenses.some(l=>l.id==='leadership')"));
});
test('Switching the lens preserves all live access and rejects unavailable responsibilities',h=>{
 const before=h.run('recStable(S.accessControl)');h.run("actions['re212-lens']({dataset:{lens:'owner'}});");assert.equal(h.run('re212Context().lens.id'),'owner');assert.equal(h.run('recStable(S.accessControl)'),before);assert.throws(()=>h.run("actions['re212-lens']({dataset:{lens:'administrator'}})"),/not available/);assert(!h.run("re212Capability('access.manage',{scopeId:''}).allowed"));
 scope(h,'B-EN');assert(!h.run("re212Context().lenses.some(x=>x.id==='contributor')"));assert.notEqual(h.run('re212Context().lens.id'),'owner');
});
test('Navigation and cached search follow actor capability and exact source location',h=>{
 const nav=h.run('re212Nav().map(x=>x[0])');assert(nav.includes('my-workspace'));assert(nav.includes('programs'));assert(!nav.includes('roles'));assert(!nav.includes('reports'));
 assert.equal(h.run("ui211SearchRows('RSK-01').filter(x=>x.id==='RSK-01').length"),0);assert.equal(h.run("ui211SearchRows('RSK-04')[0].route"),'risk?id=RSK-04');
 actor(h,'USR-03');h.run('var RE_CACHED=ui211SearchCatalog();');assert(h.run("RE_CACHED.some(x=>x.id==='RSK-01')"));actor(h,'USR-01');assert.equal(h.run("ui211SearchRows('RSK-01',RE_CACHED).filter(x=>x.id==='RSK-01').length"),0);
});
test('A reporting-only person receives leadership workspaces without operational write surfaces',h=>{
 h.run("gvState().assignments=gvState().assignments.filter(a=>a.userId!=='USR-04');gvState().assignments.push({id:'RE212-BOARD',userId:'USR-04',roleId:'ROLE-BOARD',scopeId:'',starts:S.asOf,ends:'',status:'Active'});S.governance.actorId='USR-04';");const nav=h.run('re212Nav().map(x=>x[0])');assert(nav.includes('reports'));assert(nav.includes('home'));assert(!nav.includes('risks'));assert(!nav.includes('documents'));assert(!h.run("re212ActionDecision('new-risk').allowed"));assert.equal(h.run('re212Context().lens.id'),'leadership');
});
test('Named program ownership and responding remain legitimate and exact denies still win',h=>{
 assert(h.run("atCanManage({programId:'PRG-104',target:'Meridian Bank'})"));assert(!h.run("re212Capability('programs.edit',{scopeId:'U-HO-SEC'}).allowed"));
 h.run("var RE_ASSIGN=atAssign({kind:'canonical',programId:'PRG-104',target:'Meridian Bank',respondentId:'USR-02',reviewerId:'USR-03',due:'2026-11-30',reason:'An explicitly named synthetic response assignment.'});S.governance.actorId='USR-02';");assert(h.run('atCanRespond(atAssignmentRaw(RE_ASSIGN.id))'));
 deny(h,'programs.submit','U-HO-SEC','USR-02');assert(!h.run('atCanRespond(atAssignmentRaw(RE_ASSIGN.id))'));
 actor(h,'USR-01');deny(h,'programs.edit','U-HO-SEC');assert(!h.run("atCanManage({programId:'PRG-104',target:'Meridian Bank'})"));assert.throws(()=>h.run("CLEAR_SIGHT_ASSESSMENTS.assign({kind:'canonical',programId:'PRG-104',target:'Meridian Bank',respondentId:'USR-04',reviewerId:'USR-03',due:'2026-11-30'})"),/actual program owner/);
});
test('New operational drafts start in a permitted location and record the actual preparer',h=>{
 h.run("jStartOperation('risk');var RE_DRAFT=Object.values(S.journeyDrafts).at(-1);");assert.equal(h.run('RE_DRAFT.data.owner'),'Amina Bello');assert(h.run("orgWithin(RE_DRAFT.data.orgScopeId,'B-IK')"));h.run('wf29StartControl();');assert.equal(h.run("wf29Draft('control','new').values.owner"),'Amina Bello');assert(h.run("orgWithin(wf29Draft('control','new').values.orgScopeId,'B-IK')"));actor(h,'USR-03');assert.throws(()=>h.run('wf29StartControl()'),/No active/);
});
test('Direct control commands bind the actual tester and assigned independent reviewer',h=>{
 testDraft(h);actor(h,'USR-04');const before=h.run('recStable(wf29State().controlTests)');assert.throws(()=>h.run("wf29SubmitTest('CTL-03')"),/No active grant|test preparer/);assert.equal(h.run('recStable(wf29State().controlTests)'),before);
 actor(h,'USR-01');h.run("wf29SubmitTest('CTL-03');var RE212_TEST=wf29State().controlTests.at(-1);");assert.throws(()=>decision(h,'Amina Bello'),/own submission|explicit deny/);actor(h,'USR-03');assert.throws(()=>decision(h,'Chidi Okafor'),/working identity/);decision(h);assert.equal(h.run('RE212_TEST.status'),'Approved');assert.equal(h.run("recRead('control','CTL-03').effectiveness"),'Effective');
});
test('Control moves and edits recheck original plus proposed scopes and revoked grants',h=>{
 h.run("wf29StartControl('CTL-03');var RE_EDIT=wf29Draft('control','CTL-03');RE_EDIT.values.orgScopeId='U-HO-SEC';");const original=h.run("recStable(recRead('control','CTL-03'))");assert.throws(()=>h.run("wf29SaveControl('CTL-03')"),/No active grant/);assert.equal(h.run("recStable(recRead('control','CTL-03'))"),original);
 h.run("RE_EDIT.values.orgScopeId='U-IK-PAY';gvState().assignments.find(a=>a.id==='GVA-02').status='Revoked';");assert.throws(()=>h.run("wf29SaveControl('CTL-03')"),/No active grant/);assert.equal(h.run("recStable(recRead('control','CTL-03'))"),original);
});
test('RCSA draft and decision check the real first-line and assigned review identities',h=>{
 h.run("var RE_CYCLE={id:'RCSA-RE212',name:'Scoped synthetic cycle',owner:'Amina Bello',reviewer:'Ifeoma Nwosu',due:'2026-11-30',risks:['RSK-04'],orgScopeIds:['U-IK-PAY'],reviews:{}};S.cycles.push(RE_CYCLE);wf29SaveRcsaDraft(RE_CYCLE.id,'RSK-04',{rationale:'An accountable first-line assessment draft.',likelihood:'2',impact:'3'});");actor(h,'USR-02');assert.throws(()=>h.run("wf29SaveRcsaDraft(RE_CYCLE.id,'RSK-04',{rationale:'Another person cannot rewrite the named first-line proposal.'})"),/first-line assessment owner/);
 h.run("RE_CYCLE.reviews['RSK-04']={status:'Submitted',submittedBy:'Amina Bello',revision:1};");assert.throws(()=>h.run("rcsaDecision({dataset:{cycle:RE_CYCLE.id,risk:'RSK-04'},values:{reviewer:'Ifeoma Nwosu',decision:'Challenge',note:'The proposed rating needs a refreshed evidence basis.'}})"),/assigned reviewer/);actor(h,'USR-03');h.run("rcsaDecision({dataset:{cycle:RE_CYCLE.id,risk:'RSK-04'},values:{reviewer:'Ifeoma Nwosu',decision:'Challenge',note:'The proposed rating needs a refreshed evidence basis.'}})");assert.equal(h.run("RE_CYCLE.reviews['RSK-04'].status"),'Challenge requested');
});
test('KRI submission rechecks authorization after confirmation and cannot select an impersonated reviewer',h=>{
 actor(h,'USR-04');h.run("current={name:'kri-return',params:new URLSearchParams('id=KRC-BRANCH-2026-10%3A%3AB-EN')};var RE_RETURN=ixCurrentReturn();ixSubmitReturn();");assert(h.run("typeof modalSave==='function'"));deny(h,'kri.submit','B-EN','USR-04');const before=h.run('RE_RETURN.submissions.length');assert.throws(()=>h.run('modalSave({values:{attest:true}})'),/explicit deny/);assert.equal(h.run('RE_RETURN.submissions.length'),before);
 h.run("gvState().directRules=gvState().directRules.filter(d=>d.permission!=='kri.submit');ixSubmitReturn();modalSave({values:{attest:true}});");assert.equal(h.run('RE_RETURN.status'),'Submitted');actor(h,'USR-02');assert.throws(()=>h.run("ixReviewReturn({values:{reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'Please clarify the source of the unavailable observation.'}})"),/assigned reviewer/);actor(h,'USR-03');h.run("ixReviewReturn({values:{reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'Please clarify the source of the unavailable observation.'}})");assert.equal(h.run('RE_RETURN.status'),'Changes requested');
});
test('Privacy decisions preserve source independence and block name-only reviewer switching',h=>{
 h.run("var RE_CASE={id:'CASE-RE212',title:'Synthetic privacy case',orgScopeId:'U-IK-PAY',owner:'Amina Bello',reviewer:'Ifeoma Nwosu',type:'Rights request',state:'Awaiting review',activity:'',history:[],deadline:'2026-11-30'};S.privacyCases.push(RE_CASE);");assert.throws(()=>h.run("jCaseDecision({dataset:{id:RE_CASE.id},values:{reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'The response package requires further documented investigation.'}})"),/explicit deny|assigned reviewer/);assert.equal(h.run('RE_CASE.state'),'Awaiting review');actor(h,'USR-03');h.run("jCaseDecision({dataset:{id:RE_CASE.id},values:{reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'The response package requires further documented investigation.'}})");assert.equal(h.run('RE_CASE.state'),'Changes requested');
});
test('A branch grant cannot publish a shared form or a whole-organization program',h=>{
 scope(h,'U-IK-PAY');assert.throws(()=>h.run('faCreate()'),/No active grant/);scope(h,'');const count=h.run('S.programs.length');assert.throws(()=>h.run('publishReview()'),/No active grant/);assert.equal(h.run('S.programs.length'),count);
});
test('Legacy privacy cases without a named reviewer stay reviewable under exact independent authority',h=>{
 h.run("var RE_LEGACY={id:'CASE-RE212-LEGACY',title:'Legacy case',orgScopeId:'U-IK-PAY',owner:'Amina Bello',type:'Rights request',state:'Awaiting review',activity:'',history:[],deadline:'2026-11-30'};S.privacyCases.push(RE_LEGACY);S.governance.actorId='USR-03';");assert(h.run("!!re212FormCheck({dataset:{jSubmit:'case-decision',id:RE_LEGACY.id}})"));h.run("jCaseDecision({dataset:{id:RE_LEGACY.id},values:{reviewer:'Ifeoma Nwosu',decision:'Request changes',note:'The previous case requires a documented correction before closure.'}})");assert.equal(h.run('RE_LEGACY.state'),'Changes requested');
});
test('Action visibility and draft guards use the same real capability as their commits',h=>{
 assert(h.run("re212ActionDecision('wf29-control-edit',{dataset:{id:'CTL-03'}}).allowed"));assert(!h.run("re212ActionDecision('wf29-control-edit',{dataset:{id:'CTL-01'}}).allowed"));actor(h,'USR-05');assert(!h.run("re212ActionDecision('new-control').allowed"));assert(h.run("re212ActionDecision('gv-new-role').allowed"));
 actor(h,'USR-01');submitTest(h);assert.throws(()=>h.run("re212FormCheck({dataset:{wf29Submit:'control-decision',id:RE212_TEST.id}})"),/assigned reviewer/);actor(h,'USR-03');assert(h.run("!!re212FormCheck({dataset:{wf29Submit:'control-decision',id:RE212_TEST.id}})"));
});
test('Other people can inspect an action but only its actual owner may complete or reopen it',h=>{
 h.run("S.tasks.push({id:'TASK-RE212',title:'Synthetic accountable task',owner:'Amina Bello',ownerId:'USR-01',orgScopeId:'U-IK-PAY',status:'Open',kind:'Remediation',due:'2026-11-20',ref:'',route:'work',outcome:'Confirm the documented branch process correction.'});");actor(h,'USR-03');h.run("taskDetail('TASK-RE212');");assert.equal(h.run('modalSave'),null);assert(h.elements.get('#dialog').innerHTML.includes('read-only'));assert.equal(h.run("S.tasks.find(t=>t.id==='TASK-RE212').status"),'Open');actor(h,'USR-01');h.run("taskDetail('TASK-RE212');modalSave({values:{note:'The owner inspected the corrected process and recorded the exact outcome.'}})");assert.equal(h.run("S.tasks.find(t=>t.id==='TASK-RE212').completedBy"),'Amina Bello');
});
test('All role views render real registered actions, escaped source names and useful access explanations',h=>{
 h.run("REC_PEOPLE.find(x=>x.id==='USR-01').name='<script>unsafe actor</script>';current={name:'my-workspace',params:new URLSearchParams()};");for(const route of ['my-workspace','my-access']){h.run(`current={name:${JSON.stringify(route)},params:new URLSearchParams()};`);const html=h.run('window.CS_JOURNEYS[current.name]()');assert(html.includes('<h1>'));assert(!html.includes('<script>unsafe'));assert(html.includes('&lt;script&gt;'));assert(!/\bundefined\b|\bNaN\b|\[object Object\]/.test(html));for(const [,action]of html.matchAll(/data-action="([^"]+)"/g))assert.equal(h.run(`typeof actions[${JSON.stringify(action)}]`),'function');}
 h.run("current={name:'roles',params:new URLSearchParams()};");assert(h.run('re212RouteGuard().includes("Access is unavailable")'));
});

(async()=>{for(const {name,fn}of cases){try{await fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.stack||String(error)});console.error('FAIL',name,error.stack||error);}}const report={suite:'V2.12 actual-source internal role experience',runtime:'Node; DOM/browser shell substituted',browserLayoutValidated:false,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,results};fs.writeFileSync(path.join(root,'tests/role-experience-v212-results.json'),JSON.stringify(report,null,2)+'\n');console.log(report.passed+'/'+results.length+' role checks passed');if(report.failed)process.exitCode=1;})();
