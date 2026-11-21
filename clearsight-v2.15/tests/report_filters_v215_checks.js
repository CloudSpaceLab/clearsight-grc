/* Actual report handlers with a controlled filter-form DOM. These checks do
 * not certify native layout, focus announcements or browser printing. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {createRuntime,root}=require('./runtime_harness');
const results=[],json=(h,s)=>JSON.parse(JSON.stringify(h.run(s)));
function fresh(){const h=createRuntime();h.run("wf29Init();S.governance.actorId='USR-02';");return h;}
function test(name,fn){try{fn(fresh());results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function mount(h,kind='loss'){
 const id=kind==='loss'?'loss-report-filters':'assessment-report-filters',config=json(h,kind==='loss'?'lrpConfig()':'arpConfig()'),keys=kind==='loss'?['basis','from','to','currency']:['program','person','reviewer','state'];
 const status={textContent:''},revert={hidden:true},apply={focused:false,focus(){this.focused=true}},fields=keys.map(name=>({name,value:config[name]||'',focused:false,focus(){this.focused=true}}));
 const form={id,dataset:{sa215Applied:JSON.stringify(Object.fromEntries(keys.map(k=>[k,config[k]||'']))),sa215Valid:'true'},elements:fields,querySelector:q=>q==='[data-sa215-filter-status]'?status:q==='[data-action="sa215-revert-filters"]'?revert:q==='[type="submit"]'?apply:null};
 const exportButton={dataset:{action:kind==='loss'?'lrp-export-cohort':'arp-export-assignments'},disabled:false},restrictedButton={dataset:{action:'ix-new-report'},disabled:true},unrelated={dataset:{action:'unrelated-action'},disabled:false},group={dataset:{},disabled:false,value:config.group||'',matches:q=>q==='[data-lrp-group]',hasAttribute:n=>n==='data-lrp-group'};
 h.elements.set('#'+id,form);h.context.document.querySelectorAll=q=>q==='#main-content [data-action],#main-content [data-lrp-group]'?[exportButton,restrictedButton,unrelated,group]:[];h.context.SA215_FORM=form;h.context.SA215_GROUP=group;
 return {form,fields,status,revert,apply,exportButton,restrictedButton,unrelated,group,field:name=>fields.find(f=>f.name===name)};
}
function hashRoute(h){return h.context.location.hash.replace(/^#?\//,'');}

test('Edited loss filters leave the displayed cohort intact and block report output until apply or revert',h=>{
 const html=h.navigate('loss-insights?org=B-IK&period=30&basis=occurrence&from=2026-11-01&to=2026-11-13&currency=NGN');assert.ok(html.includes('data-sa215-applied='));assert.ok(html.includes('role="status"'));assert.ok(html.includes('Reset loss view'));
 const dom=mount(h),before=h.run('recStable(S)'),cohort=h.run('recStable({...lrpCapture(),capturedAt:null})');dom.field('currency').value='USD';h.run('sa215RefreshReportFilters()');
 assert.equal(dom.form.dataset.sa215Pending,'true');assert.ok(dom.status.textContent.includes('Apply view'));assert.equal(dom.revert.hidden,false);assert.equal(dom.exportButton.disabled,true);assert.equal(dom.restrictedButton.disabled,true);assert.equal(dom.unrelated.disabled,false);
 for(const name of ['lrp-export-cohort','lrp-export-receipts','ix-save-view','ix-new-report','print'])assert.throws(()=>h.run(`actions[${JSON.stringify(name)}]()`),/Apply the changed report filters/);
 assert.throws(()=>h.run('ixNewReport()'),/Apply the changed report filters/);assert.equal(dom.apply.focused,true);assert.equal(h.run('recStable(S)'),before);assert.equal(h.run('recStable({...lrpCapture(),capturedAt:null})'),cohort);
});
test('Revert restores exact applied form values and original control availability without rendering',h=>{
 h.navigate('loss-insights?org=B-IK&basis=recognition&currency=NGN');const dom=mount(h),before=h.elements.get('#main-content').innerHTML;dom.field('basis').value='reporting';dom.field('from').value='2026-01-01';h.run('sa215RefreshReportFilters()');
 h.run("actions['sa215-revert-filters']({dataset:{form:'loss-report-filters'}})");assert.equal(dom.field('basis').value,'recognition');assert.equal(dom.field('from').value,h.run('lrpConfig().from'));assert.equal(dom.field('basis').focused,true);assert.equal(dom.form.dataset.sa215Pending,'false');assert.equal(dom.revert.hidden,true);assert.equal(dom.exportButton.disabled,false);assert.equal(dom.restrictedButton.disabled,true);assert.equal(h.elements.get('#main-content').innerHTML,before);
});
test('Named-response edits are isolated from applied identities and reset preserves organization and clocks',h=>{
 h.navigate('assessment-insights?org=all&period=30&state=pending');const dom=mount(h,'response'),before=h.run('recStable(S)'),applied=json(h,'arpCapture()');dom.field('state').value='ready';h.run('sa215RefreshReportFilters()');assert.throws(()=>h.run("actions['arp-export-assignments']()"),/Apply the changed report filters/);assert.throws(()=>h.run("actions['arp-export-history']()"),/Apply the changed report filters/);assert.deepEqual(json(h,'arpCapture().config'),applied.config);
 const cycle=h.run('ixCycle().id');h.run("actions['sa215-reset-filters']({dataset:{form:'assessment-report-filters'}})");const p=new URLSearchParams(hashRoute(h).split('?')[1]);assert.equal(p.get('org'),'all');assert.equal(p.get('period'),'30');assert.equal(p.get('cycle'),cycle);for(const key of ['program','person','reviewer','state'])assert.equal(p.has(key),false);assert.equal(h.run('recStable(S)'),before);
});
test('Loss reset uses stated defaults and clears stale drilldowns while preserving exact scope and period',h=>{
 h.navigate('loss-insights?org=U-IK-PAY&period=30&basis=recognition&from=2026-10-01&to=2026-11-01&currency=USD&group=branch&inspect=age&cohort=0%E2%80%9330+days');mount(h);const cycle=h.run('ixCycle().id'),sources=h.run('recStable(S.losses)');h.run("actions['sa215-reset-filters']({dataset:{form:'loss-report-filters'}})");
 const p=new URLSearchParams(hashRoute(h).split('?')[1]);assert.equal(p.get('org'),'U-IK-PAY');assert.equal(p.get('period'),'30');assert.equal(p.get('cycle'),cycle);assert.equal(p.get('basis'),'occurrence');assert.equal(p.get('currency'),'NGN');assert.equal(p.get('group'),'lossGroup');assert.equal(p.get('from'),h.run('addDays(S.asOf,-29)'));assert.equal(p.get('to'),h.run('S.asOf'));assert.equal(p.has('inspect'),false);assert.equal(p.has('cohort'),false);assert.deepEqual(json(h,'S.lossReportPreferences'),json(h,'lrpDefaults()'));assert.equal(h.run('recStable(S.losses)'),sources);
});
test('Invalid response recovery actually clears inherited filters rather than returning to the same error',h=>{
 const html=h.navigate('assessment-insights?org=B-IK&period=30&program=NO-SUCH-PROGRAM&person=invalid&reviewer=missing&state=pending');assert.equal(h.run('current.blocked'),true);
 const match=html.match(/href="#\/([^\"]+)"[^>]*>Clear response filters<\/a>/);assert.ok(match);const route=match[1].replace(/&amp;/g,'&'),p=new URLSearchParams(route.split('?')[1]);assert.equal(p.get('org'),'B-IK');assert.equal(p.get('period'),'30');for(const key of ['program','person','reviewer','state'])assert.equal(p.has(key),false);h.navigate(route);assert.equal(h.run('current.blocked'),false);assert.equal(h.run('selectedScope()'),'B-IK');
});
test('Grouping cannot silently discard edited dates and current permission guards still run after revert',h=>{
 h.navigate('loss-insights?org=B-IK&group=lossGroup');const dom=mount(h);dom.field('from').value='2026-01-01';dom.group.value='branch';h.run("window.SA215_EVENT={target:SA215_GROUP,preventDefault(){this.prevented=true},stopImmediatePropagation(){this.stopped=true}};sa215GuardGrouping(SA215_EVENT)");assert.equal(h.run('SA215_EVENT.prevented'),true);assert.equal(h.run('SA215_EVENT.stopped'),true);assert.equal(dom.group.value,'lossGroup');assert.equal(dom.field('from').value,'2026-01-01');
 h.run("actions['sa215-revert-filters']({dataset:{form:'loss-report-filters'}});S.accessControl.directRules.push({id:'SYN-REPORT-DENY',userId:'USR-02',permission:'reports.export',scopeId:'B-IK',effect:'deny',status:'Active',starts:'2026-01-01',ends:'',reason:'Synthetic permission-change fixture',approvedBy:'USR-05'});");assert.throws(()=>h.run("actions['lrp-export-cohort']()"),/denied|access|grant/i);
});
test('Captured report output remains unchanged and has no live filter-edit controls',h=>{
 h.navigate('reports?org=B-IK');h.run("window.SA215_PACK={name:'Synthetic frozen pack',audience:'Board',sections:['losses','assurance'],snapshot:clone(reportData())};window.SA215_FROZEN=brFrozenBody(SA215_PACK);window.SA215_CAPTURE=recStable(SA215_PACK);");h.navigate('loss-insights?org=B-IK');const dom=mount(h);dom.field('currency').value='USD';h.run('sa215RefreshReportFilters()');assert.equal(h.run('brFrozenBody(SA215_PACK)'),h.run('SA215_FROZEN'));assert.equal(h.run('recStable(SA215_PACK)'),h.run('SA215_CAPTURE'));assert.ok(!h.run('SA215_FROZEN').includes('data-sa215-applied'));
});

const report={suite:'V2.15 applied reporting filters',runtime:'Actual application handlers and controlled filter-form DOM',browserLayoutValidated:false,results,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length};fs.writeFileSync(path.join(root,'tests/report-filters-v215-results.json'),JSON.stringify(report,null,2));console.log(`${report.passed}/${results.length} report filter checks passed`);if(report.failed)process.exitCode=1;
