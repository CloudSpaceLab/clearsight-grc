/* Actual-source integration harness for restricted environments without a
 * browser process. DOM rendering/layout is deliberately not claimed here. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),vm=require('node:vm'),assert=require('node:assert/strict');
const ROOT=path.resolve(__dirname,'..'),results=[];
const fakeElement=()=>({innerHTML:'',textContent:'',dataset:{},style:{},open:false,hidden:false,value:'',classList:{add(){},remove(){},toggle(){},contains(){return false;}},addEventListener(){},setAttribute(){},removeAttribute(){},getAttribute(){return null;},append(){},prepend(){},remove(){},insertAdjacentHTML(){},focus(){},scrollIntoView(){},close(){},showModal(){},querySelector(){return null;},querySelectorAll(){return[];},closest(){return null;},matches(){return false;}});
const elements=new Map(),body=fakeElement(),store={};body.dataset.start='reports';
const context={console,URL,URLSearchParams,TextEncoder,TextDecoder,Blob,Date,Math,JSON,Map,Set,Array,Object,String,Number,Boolean,RegExp,Promise,Intl,parseInt,parseFloat,isNaN,Error,Uint8Array,ArrayBuffer,structuredClone,crypto:require('node:crypto').webcrypto,navigator:{},location:{hash:'#/reports'},localStorage:{getItem:k=>store[k]??null,setItem:(k,v)=>store[k]=String(v),removeItem:k=>delete store[k]},document:{body,documentElement:fakeElement(),addEventListener(){},querySelector:s=>{if(!elements.has(s))elements.set(s,fakeElement());return elements.get(s);},querySelectorAll(){return[];},createElement:fakeElement},setTimeout:()=>0,clearTimeout(){},requestAnimationFrame(){},scrollTo(){},scrollY:0,innerWidth:1440,addEventListener(){}};
context.window=context;vm.createContext(context);
const files=['seed','app','forms','experience','journeys','insights','records','document-store','documents','usability','copilot','kri-schema','branch-kri','board-reports'];
for(const f of files){const p=path.join(ROOT,'assets',f+'.js');if(!fs.existsSync(p))continue;let text=fs.readFileSync(p,'utf8');if(f==='copilot')text=text.replace('recordsInit();docSeed();render();','');vm.runInContext(text,context,{filename:f+'.js'});}
const run=src=>vm.runInContext(src,context),json=src=>JSON.parse(JSON.stringify(run(src)));
run('scopeInit();faInit();journeyInit();ixInit();recordsInit();docSeed();render=()=>{};current=route();');
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS',name);}catch(err){results.push({name,status:'FAIL',error:err.stack});console.error('FAIL',name,err.message);}}

test('Current, prior and concentration loss read models reconcile to the source ledger',()=>{
 const x=json('reportData()'),t=json('scopeTotals(ixLossPeriod())');
 assert.equal(x.losses.net,t.net);
 assert.equal(x.board.lossTrend.reduce((n,b)=>n+b.net,0),t.net);
 assert.equal(x.board.lossDrivers.reduce((n,b)=>n+b.net,0),t.net);
 assert.equal(x.board.locations.rows.reduce((n,b)=>n+b.loss.net,0),t.net);
 assert.equal(x.board.priorLoss.net,run('scopeTotals(ixLossPeriod(scopePeriod(),selectedScope(),scopePeriod())).net'));
});
test('Actual report renderers cover every perspective and source drill-down without runtime errors',()=>{
 for(const view of ['executive','locations','drivers','kri','assurance','dependencies','quality','library']){
  run(`current={name:'reports',params:new URLSearchParams('view=${view}')};`);const html=run('reportsView()');assert.ok(html.includes('Reports &amp; insights')||html.includes('Reports & insights'));assert.ok(!html.includes('NaN'));assert.ok(!html.includes('[object Object]'));
 }
 for(const kind of ['appetite','actions','category']){run(`current={name:'insight-detail',params:new URLSearchParams('kind=${kind}&id=Information%20security')};`);assert.ok(run('brDetailView()').length>500);}
});
test('Monetary values retain integer-kobo conversion and booleans are exception counts',()=>{
 assert.equal(run('brMetricFormat({type:"currency",unit:"NGN"},125000)'),run('ngn(125000)'));
 assert.equal(run('brMetricFormat({type:"boolean",aggregation:"exceptions"},3)'),'3 exceptions');
 assert.equal(run('brMetricFormat({type:"decimal",unit:"hours"},3.5)'),'3.5 hours');
 const data=json('reportData()');if(data.board.kriDefinition.templateVersion==='2.8'){
  assert.equal(data.kri.expectedValues,102);assert.equal(data.kri.approved,1);
  const cash=data.kri.consolidation.find(x=>x.metric==='cashShort');assert.equal(cash.value,125000);
  const html=run('brKriDisplay(reportData())');assert.ok(html.includes('Context only'));assert.ok(html.includes('₦1,250'));
  assert.ok(!html.includes('0 Yes / No'));
  const adverseNo=json('brDecisionItems(reportData()).find(v=>v.key.endsWith(":cctv"))');assert.ok(adverseNo&&adverseNo.why.includes('No.'));
  assert.ok(!adverseNo.why.includes('0 exceptions'),'Individual adverse Boolean answers must show No, not an aggregate exception count');
 }
});
test('KRI history excludes incompatible definitions and holds the exact reporting cohort',()=>{
 const before=json('reportData().board.kriTrend');
 run('S.branchKri.cycles.unshift({...clone(ixCycle()),id:"BAD-DEFINITION",definitionVersion:"different/1",period:"2026-07"});');
 const after=json('reportData().board.kriTrend');assert.deepEqual(before,after);
 run('S.branchKri.cycles=S.branchKri.cycles.filter(c=>c.id!=="BAD-DEFINITION");');
});
test('Boolean polarity, context classification, conditional applicability and policy changes break comparability',()=>{
 const original=json('reportData().board.kriTrend');
 for(const [field,value] of [['adverse',false],['context',true],['condition',{field:'cardPrinter',equals:false}],['policyBasis','Revised treatment and applicability policy']]){
  run(`(()=>{const c=clone(ixCycle());c.id='CHANGED-${field}';c.period='2026-07';const m=c.metrics.find(v=>v.type==='boolean'&&v.adverse===true&&!v.context)||c.metrics[0];m[${JSON.stringify(field)}]=${JSON.stringify(value)};S.branchKri.cycles.unshift(c);})()`);
  assert.deepEqual(json('reportData().board.kriTrend'),original,field+' must be part of the comparable definition');
  run(`S.branchKri.cycles=S.branchKri.cycles.filter(c=>c.id!=='CHANGED-${field}');`);
 }
});
test('Approved N/A is excluded from applicable coverage and the full indicator register stays grouped',()=>{
 run('scopeChange("branch","B-IK");');const x=json('reportData()'),html=run('brQualityPanel(reportData())');
 const applicable=x.kri.applicableExpected??x.kri.expectedValues-(x.kri.notApplicable||0);
 assert.ok(html.includes(`<b>${x.kri.known}/${applicable}</b>`));
 assert.ok(html.includes(`<b>${x.kri.notApplicable||0}/${x.kri.expectedValues}</b>`));
 assert.ok(html.includes('0 KRI observations unknown or unreported'));
 const register=run('brKriDisplay(reportData())');assert.ok(register.includes('51 indicators in 8 categories'));
 assert.equal((register.match(/class="br-kri-group"/g)||[]).length,8);
 assert.ok((register.match(/<details class="br-kri-group" open/g)||[]).length<8);
 assert.equal((run('brKriDisplay(reportData(),{frozen:true})').match(/<details class="br-kri-group" open/g)||[]).length,8);
 run('S.scope={branch:"",department:"",unit:""};');
});
test('Selected scope and loss occurrence periods remain independent of KRI and risk state',()=>{
 run('current={name:"reports",params:new URLSearchParams("view=executive")};');
 const before=json('reportData()');run('S.lossPeriod=30;');const after=json('reportData()');
 assert.deepEqual(before.kri,after.kri);assert.deepEqual(before.sourceRecords.risks,after.sourceRecords.risks);assert.equal(after.periods.loss.days,30);
 run('scopeChange("branch","B-IK");');const scoped=json('reportData()');assert.equal(scoped.board.scopeId,'B-IK');
 assert.ok(scoped.sourceRecords.risks.every(r=>run(`orgWithin(${JSON.stringify(r.orgScopeId)},'B-IK')`)));
 assert.ok(scoped.kri.values.every(v=>run(`orgWithin(${JSON.stringify(v.unit)},'B-IK')`)));
 run('S.scope={branch:"",department:"",unit:""};S.lossPeriod=90;');
});
test('Board draft requires coverage acknowledgement, valid dates and captured source references',()=>{
 run('current={name:"reports",params:new URLSearchParams()};ixNewReport();brEnsureDraft();');
 assert.ok(json('brDraftIssues(S.reportDraft)').some(s=>s.includes('acknowledge')));
 run('S.reportDraft.coverageAcknowledged=true;S.reportDraft.meetingDate="2026-13-01";');assert.ok(json('brDraftIssues(S.reportDraft)').some(s=>s.includes('valid meeting')));
 run('S.reportDraft.meetingDate="2026-02-29";');assert.ok(json('brDraftIssues(S.reportDraft)').some(s=>s.includes('valid meeting')));
 assert.equal(run('brValidDate("2028-02-29")'),true);assert.equal(run('brValidDate("2026-04-31")'),false);
 run('S.reportDraft.meetingDate="2026-11-27";actions["br-use-suggestion"]({dataset:{index:"0"}});');
 assert.equal(run('S.reportDraft.decisions.length'),1);assert.deepEqual(json('brDraftIssues(S.reportDraft)'),[]);
 run('S.reportDraft.decisions[0].sourceId="OUTSIDE-CAPTURE";');assert.ok(json('brDraftIssues(S.reportDraft)').some(s=>s.includes('captured records')));
 run('S.reportDraft.decisions[0].sourceId="";');
 run('S.reportDraft.decisions[0].due="2026-11-31";');assert.ok(json('brDraftIssues(S.reportDraft)').some(s=>s.includes('valid target date')));
 run('S.reportDraft.decisions[0].due="2026-11-27";');
});
test('Publishing freezes the complete board snapshot and render output survives subsequent changes',()=>{
 const snapshot=json('S.reportDraft.snapshot');run('actions["ix-publish-report"]();');const pack=json('S.reportPacks[0]');assert.deepEqual(pack.snapshot,snapshot);
 const html=run('brFrozenBody(S.reportPacks[0])');
 run('S.risks[0].title="LIVE CHANGE MUST NOT ALTER REPORT";S.losses[0].gross+=100000;S.tasks[0].due="2020-01-01";scopeChange("branch","B-EN");');
 assert.deepEqual(json('S.reportPacks[0]'),pack);assert.equal(run('brFrozenBody(S.reportPacks[0])'),html);
 assert.ok(html.includes('Decisions requested by management'));assert.ok(html.includes('Current risk distribution'));assert.ok(!html.includes('NaN'));
 run('S.scope={branch:"",department:"",unit:""};');
});
test('Assigning a report exception creates a normal scoped source-linked task and rejects duplication',()=>{
 run('current={name:"reports",params:new URLSearchParams()};window.TEST_DIALOG=null;openDialog=(title,subtitle,html,label,callback)=>{TEST_DIALOG={title,subtitle,html,label,callback};};values=x=>x;');
 const candidate=json('brDecisionItems(reportData()).find(v=>v.sourceType&&v.sourceType!=="task"&&!v.taskId)');assert.ok(candidate);
 run(`actions['br-followup']({dataset:{key:${JSON.stringify(candidate.key)}}});`);assert.ok(run('TEST_DIALOG'));
 run('TEST_DIALOG.callback({title:"Investigate management reporting exception",owner:"Amina Bello",due:"2026-11-27",description:"Confirm the business impact and complete the agreed corrective action."});');
 const task=json('S.tasks.find(t=>t.boardDecisionKey)');assert.ok(task);assert.equal(task.ref,candidate.sourceId);assert.equal(task.sourceRef.id,task.ref);assert.ok(task.orgScopeId);
 assert.throws(()=>run('TEST_DIALOG.callback({title:"Duplicate source action",owner:"Amina Bello",due:"2026-11-27",description:"Confirm the business impact and complete the agreed corrective action."});'),/already exists/);
 assert.equal(run('S.tasks.filter(t=>t.boardDecisionKey).length'),1);
});
test('Briefing CSV exports source keys, owners, due dates and stated management questions',()=>{
 run('window.TEST_DOWNLOADS=[];download=(name,body,type)=>TEST_DOWNLOADS.push({name,body,type});actions["br-export-brief"]();');const payload=json('TEST_DOWNLOADS[0]');assert.ok(payload.body.includes('management_question'));assert.ok(payload.body.includes('record_id'));assert.ok(payload.body.includes('source_route'));assert.ok(payload.body.includes('Amina Bello'));
});

const result={suite:'board-reports-v2.8-actual-source',validation:'Node vm with minimal DOM stubs; no browser layout claim',checks:results,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length};fs.writeFileSync(path.join(ROOT,'tests/board-reports-v28-model-results.json'),JSON.stringify(result,null,2));console.log(JSON.stringify({passed:result.passed,failed:result.failed}));process.exitCode=result.failed?1:0;
