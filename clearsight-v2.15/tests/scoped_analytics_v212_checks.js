/* V2.12 actual-source reporting checks. The harness substitutes the DOM shell;
 * these are scope, permission, context and immutability checks, not visual QA. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
const json=(h,s)=>JSON.parse(JSON.stringify(h.run(s)));
const fresh=()=>{const h=createRuntime();h.run("wf29Init();S.governance.actorId='USR-02';");return h;};
function grant(h,permission,scopeId='',userId='USR-02',effect='allow'){
 h.run(`S.accessControl.directRules.push(${JSON.stringify({id:'SYN-'+permission+'-'+scopeId+'-'+effect,userId,permission,scopeId,effect,status:'Active',starts:'2026-01-01',ends:'',reason:'Synthetic scoped-analytics fixture only',approvedBy:'USR-05'})});`);
}
function navigateHash(h){return h.navigate(h.context.location.hash.replace(/^#?\//,''));}
function saveView(h,name='Synthetic saved view'){
 h.run("window.SA212_OPEN=openDialog;window.SA212_CB=null;openDialog=(title,description,body,label,cb)=>SA212_CB=cb;actions['ix-save-view']();");
 h.run(`SA212_CB({name:${JSON.stringify(name)}});openDialog=SA212_OPEN;`);return json(h,'S.savedInsightViews[0]');
}
function test(name,fn){try{const h=fresh();fn(h);results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}

test('Explicit branch, department and unit routes resolve exact canonical descendant populations',h=>{
 h.navigate('reports?org=D-IK-OPS&view=locations');const c=json(h,'CLEAR_SIGHT_SCOPED_ANALYTICS.context()');
 assert.deepEqual(c.scope,{branch:'B-IK',department:'D-IK-OPS',unit:''});assert.deepEqual(c.descendantScopeIds,['D-IK-OPS','U-IK-PAY']);assert.equal(c.kind,'department');assert.equal(c.access.full,true);
 h.navigate('home?org=U-EN-REC');assert.deepEqual(json(h,'CLEAR_SIGHT_SCOPED_ANALYTICS.context().descendantScopeIds'),['U-EN-REC']);assert.equal(h.run('selectedScope()'),'U-EN-REC');
});
test('Invalid organizational locations fail closed and never broaden to whole organization',h=>{
 const html=h.navigate('reports?org=NO-SUCH-BRANCH');assert.equal(h.run('S.scopeInvalid'),true);assert.equal(h.run('current.blocked'),true);assert.equal(h.run('sa212Context().valid'),false);assert.ok(!html.includes('Recorded net loss'));
 h.navigate('reports?org=');assert.equal(h.run('S.scopeInvalid'),true);h.navigate('reports?org=B-IK');assert.equal(h.run('S.scopeInvalid'),false);assert.equal(h.run('selectedScope()'),'B-IK');
});
test('Partial whole-organization access offers explicit accessible subtrees without silently slicing aggregates',h=>{
 grant(h,'reports.view','B-IK','USR-01');h.run("S.governance.actorId='USR-01';");const html=h.navigate('reports?org=all');
 assert.equal(h.run('current.blocked'),true);assert.ok(html.includes('not available in full'));assert.ok(html.includes('org=B-IK'));assert.deepEqual(json(h,'sa212Context().access.roots'),['B-IK']);assert.ok(!html.includes('Current residual risk distribution'));
 assert.ok(h.navigate('reports?org=B-IK').includes('Current residual risk distribution'));assert.equal(h.run('current.blocked'),false);
});
test('A descendant denial prevents broader cohorts and global contextual sources despite a root allow',h=>{
 grant(h,'reports.view','U-IK-PAY','USR-02','deny');h.navigate('reports?org=B-IK');const c=json(h,'sa212Context()');assert.equal(c.access.full,false);assert.ok(c.access.deniedScopeIds.includes('U-IK-PAY'));assert.ok(!c.access.roots.includes('B-IK'));
 h.navigate('reports?org=D-HO-TECH');assert.equal(h.run("sa212Root('reports.view')"),true);assert.equal(h.run("sa212SourceAccess([''])"),false);const x=json(h,'reportData()');assert.ok(!x.programs.some(p=>p.id==='PRG-104'));
});
test('A union of branch grants is not exact-root authority for global records',h=>{
 for(const id of ['B-HO','B-IK','B-EN'])grant(h,'reports.view',id,'USR-01');h.run("S.governance.actorId='USR-01';");h.navigate('reports?org=all');assert.equal(h.run('sa212Context().access.full'),true);assert.equal(h.run("sa212Root('reports.view')"),false);assert.equal(h.run("sa212SourceAccess([''])"),false);
});
test('Bank-wide assurance remains broader context when a local department is selected',h=>{
 const html=h.navigate('reports?org=D-HO-TECH&view=assurance'),x=json(h,'reportData()'),bank=x.scopedAnalytics.programTargets.find(t=>t.programId==='PRG-104');
 assert.equal(bank.cohort,'context');assert.equal(bank.boundary,'Whole-organization assessment');assert.ok(html.includes('Broader assurance context'));assert.ok(html.includes('No branch compliance percentage'));assert.ok(!h.run('brAssuranceChart(reportData())').includes('Bank assurance'));
 assert.ok(html.includes('Control effectiveness'));assert.ok(html.includes('Open action ownership'));assert.ok(html.includes('Document review quality'));
});
test('Multi-target supplier programs include only targets relevant to the selected local cohort',h=>{
 h.run("const p=S.programs.find(p=>p.id==='PRG-105');p.targets.push('ENG-03');p.orgScopeIds=['U-HO-INFRA','U-IK-PAY'];");h.navigate('reports?org=B-IK&view=assurance');
 const p=json(h,"reportData().programs.find(p=>p.id==='PRG-105')");assert.deepEqual(p.targets.map(t=>t.id),['ENG-03']);assert.ok(!h.run('sa212AssuranceTable(reportData())').includes('Core banking hosting'));
});
test('Branch-only KRI collection is not reported as missing department or unit responses',h=>{
 let html=h.navigate('reports?org=B-IK&view=locations');assert.ok(html.includes('Not collected at this level'));assert.ok(html.includes('NGN'));assert.equal(h.run('ixKriStats(ixCycle(),"B-IK").expected'),1);
 html=h.navigate('home?org=U-IK-PAY');assert.equal(h.run('ixKriStats(ixCycle(),selectedScope()).expected'),0);assert.ok(html.includes('neither a missing response nor an assurance conclusion'));
});
test('Report and source links preserve organization, period, KRI cycle, loss basis and currency',h=>{
 h.navigate('loss-insights?org=B-IK&period=30&currency=USD&basis=recognition&from=2026-11-01&to=2026-11-13&group=branch');
 const route=h.run("sa212Route('document?id=DOC-001&tab=assessment&review=EV-001')"),p=new URLSearchParams(route.split('?')[1]);
 assert.equal(p.get('org'),'B-IK');assert.equal(p.get('period'),'30');assert.equal(p.get('currency'),'USD');assert.equal(p.get('basis'),'recognition');assert.equal(p.get('id'),'DOC-001');assert.equal(p.get('review'),'EV-001');assert.equal(p.get('cycle'),h.run('ixCycle().id'));
 const lossRoute=h.run("op211LossReportRoute(lrpCapture(),'age','0–30 days')");assert.equal(new URLSearchParams(lossRoute.split('?')[1]).get('org'),'B-IK');
});
test('The canonical organizational selector updates stale URL context instead of being undone by render',h=>{
 h.navigate('reports?org=B-IK&view=locations&period=30');h.run("scopeChange('department','D-EN-OPS')");assert.equal(h.run('selectedScope()'),'D-EN-OPS');assert.equal(new URLSearchParams(h.context.location.hash.split('?')[1]).get('org'),'D-EN-OPS');
 navigateHash(h);assert.equal(h.run('selectedScope()'),'D-EN-OPS');assert.equal(h.run('scopePeriod()'),30);
});
test('Loss filter transitions make the inherited scope and resolved defaults portable in the URL',h=>{
 h.navigate('loss-insights?org=B-IK&currency=USD');h.navigate('loss-insights?basis=recognition&currency=NGN');const p=new URLSearchParams(h.context.location.hash.split('?')[1]);
 assert.equal(p.get('org'),'B-IK');assert.equal(p.get('basis'),'recognition');assert.equal(p.get('currency'),'NGN');assert.ok(p.get('from'));assert.ok(p.get('to'));assert.ok(p.get('cycle'));
});
test('Saved loss views restore exact scope, explicit dates, currency and grouping after navigation changes',h=>{
 h.navigate('loss-insights?org=B-IK&period=30&currency=USD&basis=recognition&from=2026-11-01&to=2026-11-13&group=branch');const v=saveView(h);h.navigate('reports?org=B-EN&period=365');h.run(`actions['ix-open-view']({dataset:{id:${JSON.stringify(v.id)}}})`);navigateHash(h);
 assert.equal(h.run('selectedScope()'),'B-IK');assert.equal(h.run('scopePeriod()'),30);assert.deepEqual(json(h,'lrpConfig()'),{basis:'recognition',from:'2026-11-01',to:'2026-11-13',currency:'USD',group:'branch'});
});
test('Saved response views retain the respondent and review-state filters',h=>{
 h.navigate('assessment-insights?org=all&state=pending');const v=saveView(h,'Synthetic response queue');const p=new URLSearchParams(v.analyticsContext.route.split('?')[1]);assert.equal(p.get('view'),'responses');assert.equal(p.get('state'),'pending');assert.equal(p.get('org'),'all');
});
test('Saved views recheck access and hierarchy before changing navigation state',h=>{
 h.navigate('reports?org=B-IK');const v=saveView(h);h.navigate('reports?org=B-EN');grant(h,'reports.view','B-IK','USR-02','deny');assert.throws(()=>h.run(`actions['ix-open-view']({dataset:{id:${JSON.stringify(v.id)}}})`),/denied|access|grant/i);assert.equal(h.run('selectedScope()'),'B-EN');
 h.run("S.accessControl.directRules=S.accessControl.directRules.filter(r=>r.effect!=='deny');S.savedInsightViews[0].scope.department='D-EN-OPS';");assert.throws(()=>h.run(`actions['ix-open-view']({dataset:{id:${JSON.stringify(v.id)}}})`),/hierarchy/);assert.equal(h.run('selectedScope()'),'B-EN');
});
test('Report CSV exports include portable scope, currency, date and cycle metadata',h=>{
 h.navigate('loss-insights?org=B-IK&currency=NGN&basis=occurrence&from=2026-01-01&to=2026-11-13');h.run("window.SA212_DOWNLOADS=[];download=(name,body,type)=>SA212_DOWNLOADS.push({name,body,type});actions['lrp-export-cohort']();actions['lrp-export-receipts']();");
 const outputs=json(h,'SA212_DOWNLOADS');assert.equal(outputs.length,2);for(const o of outputs){assert.ok(o.body.includes('report_scope_id'));assert.ok(o.body.includes('report_window_from'));assert.ok(o.body.includes('report_currency'));assert.ok(o.body.includes('report_view_route'));}
 assert.ok(outputs[0].body.includes('B-IK'));assert.ok(outputs[0].body.includes('2026-01-01'));
});
test('Current permissions guard raw data, selected-event exports and printing after a rendered view',h=>{
 h.navigate('loss-insights?org=B-IK&currency=NGN&from=2026-01-01&to=2026-11-13');grant(h,'reports.export','B-IK','USR-02','deny');
 for(const action of ['report-export','br-export-brief','lrp-export-cohort','lrp-export-receipts','op211-export-drill','print'])assert.throws(()=>h.run(`actions[${JSON.stringify(action)}]()`),/denied|access|grant/i,action);
 const html=h.navigate('loss-insights?org=B-IK');assert.match(html,/<button[^>]*data-action="lrp-export-cohort"[^>]*disabled/);
});
test('New captured reports remain byte-stable and older packs gain no live scope annex',h=>{
 h.navigate('reports?org=D-HO-TECH&view=assurance');h.run("window.SA212_PACK={name:'Synthetic captured pack',audience:'Board',sections:['risks','losses','assurance'],snapshot:clone(reportData())};window.SA212_OLD=clone(SA212_PACK);delete SA212_OLD.snapshot.scopedAnalytics;");
 const captured=h.run('JSON.stringify(SA212_PACK)'),body=h.run('brFrozenBody(SA212_PACK)'),old=h.run('brFrozenBody(SA212_OLD)');h.run("S.losses[0].gross+=777;S.programs[0].name='Changed live program';S.scope=sa212Scope('B-EN');S.lossPeriod=365;");
 assert.equal(h.run('JSON.stringify(SA212_PACK)'),captured);assert.equal(h.run('brFrozenBody(SA212_PACK)'),body);assert.equal(h.run('brFrozenBody(SA212_OLD)'),old);assert.ok(!old.includes('Saved reporting boundary'));
});
test('Captured contextual boundaries and current child denies govern pack access independently of navigation scope',h=>{
 h.navigate('reports?org=D-HO-TECH');h.run("window.SA212_SNAPSHOT=clone(reportData());S.reportPacks.push({id:'SYN-PACK',name:'Synthetic captured scope',audience:'Board',sections:['assurance'],snapshot:SA212_SNAPSHOT});");
 h.navigate('report-pack?id=SYN-PACK&org=B-EN');assert.equal(h.run('current.blocked'),false);assert.equal(h.run('sa212SnapshotAccess(SA212_SNAPSHOT).allowed'),true);
 grant(h,'reports.view','U-IK-PAY','USR-02','deny');assert.equal(h.run('sa212SnapshotAccess(SA212_SNAPSHOT).allowed'),false);
});
test('Read-only reporting grants cannot create, edit or publish a draft and the builder exposes no editable fields',h=>{
 const html=h.navigate('reports?org=B-IK'),initial=h.run('JSON.stringify(S.reportDraft)');assert.match(html,/<button[^>]*data-action="ix-new-report"[^>]*disabled/);assert.throws(()=>h.run('ixNewReport()'),/grant|access/i);assert.equal(h.run('JSON.stringify(S.reportDraft)'),initial);
 h.run("S.reportDraft={name:'Synthetic draft',audience:'Board',sections:['risks'],snapshot:clone(reportData()),decisions:[],step:1};");const before=h.run('JSON.stringify(S.reportDraft)'),builder=h.navigate('report-builder');assert.ok(builder.includes('Report preparation · read only'));assert.ok(!builder.includes('data-ix-report-draft'));
 for(const action of ['br-add-decision','ix-report-step','ix-publish-report'])assert.throws(()=>h.run(`actions[${JSON.stringify(action)}]({dataset:{step:'2'}})`),/grant|access/i);assert.equal(h.run('JSON.stringify(S.reportDraft)'),before);
});
test('Legacy draft inputs are stopped in capture phase after author access is revoked',h=>{
 grant(h,'reports.edit','B-IK');h.navigate('reports?org=B-IK');h.run('ixNewReport()');grant(h,'reports.edit','B-IK','USR-02','deny');const before=h.run('JSON.stringify(S.reportDraft)');
 h.run("window.SA212_EVENT={target:{matches:()=>true},preventDefault(){this.prevented=true},stopImmediatePropagation(){this.stopped=true}};sa212DraftInputGuard(SA212_EVENT);");assert.equal(h.run('SA212_EVENT.stopped'),true);assert.equal(h.run('SA212_EVENT.prevented'),true);assert.equal(h.run('JSON.stringify(S.reportDraft)'),before);
});
test('Report authors require a separate submission grant and publishing preserves the captured source',h=>{
 grant(h,'reports.edit','B-IK');h.navigate('reports?org=B-IK');h.run("ixNewReport();Object.assign(S.reportDraft,{name:'Synthetic author report',meetingDate:'2026-11-20',coverageAcknowledged:true,decisions:[]});");const captured=h.run('JSON.stringify(S.reportDraft.snapshot)');assert.throws(()=>h.run("actions['ix-publish-report']()"),/grant|access/i);grant(h,'reports.submit','B-IK');h.run("actions['ix-publish-report']()");assert.equal(h.run('S.reportDraft'),null);assert.equal(h.run('JSON.stringify(S.reportPacks[0].snapshot)'),captured);
});
test('Report-only observers cannot create operational follow-up actions',h=>{
 h.navigate('reports?org=B-IK');const key=h.run("brDecisionItems(reportData()).find(v=>v.sourceType==='risk').key"),count=h.run('S.tasks.length');assert.throws(()=>h.run(`actions['br-followup']({dataset:{key:${JSON.stringify(key)}}})`),/grant|access/i);assert.equal(h.run('S.tasks.length'),count);assert.match(h.navigate('reports?org=B-IK'),/<button[^>]*data-action="br-followup"[^>]*disabled/);
});
test('Report controls and all perspectives render actual registered routes without invalid data',h=>{
 for(const route of ['home?org=B-IK','reports?view=locations&org=D-HO-TECH','reports?view=assurance&org=D-HO-TECH','reports?view=quality&org=B-IK','reports?view=library&org=all','loss-insights?org=U-IK-PAY','assessment-insights?org=all']){
  const html=h.navigate(route);assert.ok(!/\bNaN\b|\bundefined\b|\[object Object\]/.test(html),route);for(const [,action]of html.matchAll(/data-action="([^"]+)"/g))assert.equal(h.run(`typeof actions[${JSON.stringify(action)}]`),'function',action);for(const [,name]of html.matchAll(/href="#\/([^"?]+)/g))assert.ok(h.run(`!!titles[${JSON.stringify(name)}]`),name);
 }
});
test('Invalid loss filters remain editable error views and unsupported period/cycle URLs fail closed',h=>{
 let html=h.navigate('loss-insights?org=B-IK&currency=ALL');assert.equal(h.run('current.blocked'),true);assert.ok(html.includes('supported reporting currency'));assert.ok(html.includes('loss-report-filters'));assert.match(html,/<button[^>]*data-action="ix-new-report"[^>]*disabled/);
 html=h.navigate('reports?org=B-IK&period=999');assert.equal(h.run('current.blocked'),true);assert.ok(html.includes('supported 30-'));html=h.navigate('reports?org=B-IK&cycle=NONE');assert.equal(h.run('current.blocked'),true);assert.ok(html.includes('cycle does not exist'));
});
const report={suite:'V2.12 scoped analytics',runtime:'Actual application sources with a minimal DOM shell',browserLayoutValidated:false,results,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length};fs.writeFileSync(path.join(root,'tests/scoped-analytics-v212-results.json'),JSON.stringify(report,null,2));console.log(`${report.passed}/${results.length} scoped analytics checks passed`);if(report.failed)process.exitCode=1;
