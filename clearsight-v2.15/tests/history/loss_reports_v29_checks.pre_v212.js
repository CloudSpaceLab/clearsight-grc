/* Actual-source loss reporting integration checks. Synthetic test records only.
 * Minimal DOM stubs exercise render functions and commands; no browser-layout
 * or production-authorization conformance is implied. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {createRuntime,root:ROOT}=require('./runtime_harness');
const h=createRuntime(),run=h.run,results=[];
const json=s=>JSON.parse(JSON.stringify(run(s)));
const base={type:'Actual loss',currency:'NGN',workflowState:'Accepted',status:'Open',owner:'Amina Bello',risk:'RSK-04',orgScopeId:'U-IK-PAY',unit:'Payment operations',accountName:'CASH SHORTAGES',lossGroup:'CASH SHORTAGES',transactionParticulars:'Synthetic investigation event for reporting checks',transactionReference:'',rootCause:'A synthetic reconciliation control failed.',region:'Lagos region',directorate:'South-West directorate',recoverability:{status:'Under assessment',expectedAmount:null,targetDate:'',note:''},recoveries:[],revision:1,versions:[],history:[]};
const fixture=[
 {...base,id:'TEST-A',title:'Synthetic cash variance',date:'2026-10-02',recognizedOn:'2026-11-01',reportingPeriod:'2026-11',gross:100000,transactionReference:'SYN-A',recoverability:{status:'Recovery pursued',expectedAmount:60000,targetDate:'2026-11-10',note:'Awaiting confirmed receipt evidence.'},recoveries:[{id:'REC-A',amount:30000,date:'2026-11-03',currency:'NGN',status:'Verified',sourceRef:'SYN-RECEIPT-A',verifiedBy:'USR-02'},{id:'REC-PENDING',amount:20000,date:'2026-11-06',currency:'NGN',status:'Pending'},{id:'REC-MIXED',amount:10000,date:'2026-11-06',currency:'USD',status:'Verified'},{id:'REC-FUTURE',amount:10000,date:'2026-12-01',currency:'NGN',status:'Verified'},{id:'REC-BEFORE',amount:10000,date:'2026-09-30',currency:'NGN',status:'Verified'}]},
 {...base,id:'TEST-B',title:'Synthetic legacy service dispute',date:'2026-05-01',recognizedOn:'2026-11-05',reportingPeriod:'2026-10',gross:200000,accountName:'LEGAL EXPENSES',lossGroup:'LEGAL LOSSES',orgScopeId:'U-HO-INFRA',owner:'Daniel Mensah',transactionReference:'SYN-B',rootCause:'',recoverability:{status:'Recovery pursued',expectedAmount:250000,targetDate:'2026-11-09',note:'Management estimate needs updating.'},recoveries:[{id:'REC-B',amount:50000,date:'2026-10-08',currency:'NGN',status:'Verified',sourceRef:'SYN-RECEIPT-B',verifiedBy:'USR-02',reversal:{id:'REV-B',date:'2026-11-08',status:'Approved',reason:'Synthetic receipt allocation corrected after independent review.',reviewedBy:'USR-02'}}]},
 {...base,id:'TEST-MISSING',title:'Synthetic event awaiting recognition metadata',date:'2026-11-02',recognizedOn:'',reportingPeriod:'2026-11',gross:50000,transactionReference:'SYN-M'},
 {...base,id:'TEST-USD',title:'Synthetic foreign-currency loss',date:'2026-11-01',recognizedOn:'2026-11-02',reportingPeriod:'2026-11',currency:'USD',gross:10000,transactionReference:'SYN-USD',recoveries:[{id:'REC-USD',amount:2000,date:'2026-11-04',currency:'USD',status:'Verified',sourceRef:'SYN-USD-R'}]},
 {...base,id:'TEST-PENDING',title:'Synthetic unaccepted loss',date:'2026-11-02',recognizedOn:'2026-11-03',reportingPeriod:'2026-11',gross:900000,workflowState:'Submitted',transactionReference:'SYN-P'},
 {...base,id:'TEST-CLOSED',title:'Synthetic fully recovered closed event',date:'2026-11-04',recognizedOn:'2026-11-04',reportingPeriod:'2026-11',gross:50000,workflowState:'Closed',status:'Closed',transactionReference:'SYN-C',recoveries:[{id:'REC-C',amount:50000,date:'2026-11-06',currency:'NGN',status:'Verified',sourceRef:'SYN-C-R'}]},
 {...base,id:'TEST-NEAR',title:'Synthetic prevented transaction',date:'2026-11-05',recognizedOn:'',reportingPeriod:'2026-11',gross:0,type:'Near miss',transactionReference:'SYN-N'},
 {...base,id:'TEST-CORRECTION',title:'Synthetic accepted loss under correction',date:'2026-10-01',recognizedOn:'2026-11-02',reportingPeriod:'2026-11',gross:10000,transactionReference:'SYN-CORR',pendingRevision:{status:'Submitted',snapshot:{gross:990000,recognizedOn:'2026-11-10'}}}
];
run(`S.losses=${JSON.stringify(fixture)};S.lossReportPreferences={basis:'recognition',from:'2026-11-01',to:'2026-11-13',currency:'NGN',group:'lossGroup'};current={name:'loss-insights',params:new URLSearchParams()};`);
const capture=(overrides={})=>json(`lrpCapture({...lrpConfig(),...${JSON.stringify(overrides)}})`);
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS',name);}catch(e){results.push({name,status:'FAIL',error:e.stack});console.error('FAIL',name,e.message);}}

test('Occurrence, recognition and reporting-month cohorts remain distinct',()=>{
 const ids=basis=>json(`lossReportRows({basis:${JSON.stringify(basis)},from:'2026-11-01',to:'2026-11-13',currency:'NGN'}).map(r=>r.id)`);
 assert.deepEqual(ids('occurrence').sort(),['TEST-MISSING','TEST-CLOSED','TEST-NEAR'].sort());
 assert.deepEqual(ids('recognition').sort(),['TEST-A','TEST-B','TEST-CLOSED','TEST-CORRECTION'].sort());
 assert.deepEqual(ids('reporting').sort(),['TEST-A','TEST-MISSING','TEST-CLOSED','TEST-NEAR','TEST-CORRECTION'].sort());
 assert.ok(!ids('recognition').includes('TEST-PENDING'));
});
test('Mixed currencies, unaccepted records and pending corrections never pollute financial totals',()=>{
 const ngn=capture(),usd=capture({currency:'USD'});
 assert.equal(ngn.totals.gross,360000);assert.equal(ngn.totals.recovered,80000);assert.equal(ngn.totals.unrecovered,280000);
 assert.equal(usd.totals.gross,10000);assert.equal(usd.totals.recovered,2000);assert.equal(usd.totals.unrecovered,8000);
 assert.equal(ngn.confidence.pendingCorrections,1);assert.equal(ngn.confidence.pendingExcluded,1);assert.deepEqual(ngn.confidence.otherCurrencies,['USD']);
 assert.equal(ngn.records.find(r=>r.id==='TEST-CORRECTION').gross,10000);
 assert.equal(ngn.confidence.basisMissing,1);
});
test('Cash inflows and approved reversals use their own date window and preserve historical receipts',()=>{
 const x=capture();assert.equal(x.cash.inflows,80000);assert.equal(x.cash.reversed,50000);assert.equal(x.cash.netMovement,30000);
 assert.deepEqual(x.cash.receipts.map(r=>r.id).sort(),['REC-A','REC-C']);assert.deepEqual(x.cash.reversals.map(r=>r.receiptId),['REC-B']);
 const october=capture({from:'2026-10-01',to:'2026-10-31'});assert.equal(october.cash.inflows,50000);assert.equal(october.cash.reversed,0);
 const reversalOnly=capture({from:'2026-11-08',to:'2026-11-09'});assert.equal(reversalOnly.cash.inflows,0);assert.equal(reversalOnly.cash.reversed,50000);assert.equal(reversalOnly.cash.netMovement,-50000);
 assert.equal(run('lossVerifiedRecoveries(S.losses.find(l=>l.id==="TEST-B"),"2026-11-07")'),50000);
 assert.equal(run('lossVerifiedRecoveries(S.losses.find(l=>l.id==="TEST-B"),"2026-11-08")'),0);
});
test('Age, concentration and date-basis charts reconcile including partial reporting-month windows',()=>{
 for(const basis of ['occurrence','recognition','reporting']){const x=capture({basis});assert.equal(x.concentration.reduce((n,v)=>n+v.gross,0),x.totals.gross);assert.equal(x.age.reduce((n,v)=>n+v.amount,0),x.totals.unrecovered);assert.equal(x.series.reduce((n,v)=>n+v.gross,0),x.totals.gross);}
 const midmonth=capture({basis:'reporting',from:'2026-11-08',to:'2026-11-10'});assert.equal(midmonth.totals.gross,210000);assert.equal(midmonth.series.reduce((n,v)=>n+v.gross,0),210000);
 assert.equal(midmonth.cash.reversed,50000);assert.equal(midmonth.cash.inflows,0);
});
test('Source account classifications, loss groups and risk categories are not conflated',()=>{
 const byAccount=capture({group:'accountName'}),byGroup=capture({group:'lossGroup'}),record=byGroup.records.find(r=>r.id==='TEST-B');
 assert.equal(record.accountName,'LEGAL EXPENSES');assert.equal(record.lossGroup,'LEGAL LOSSES');assert.ok(record.riskCategory);
 assert.ok(byAccount.concentration.some(x=>x.name==='LEGAL EXPENSES'));assert.ok(byGroup.concentration.some(x=>x.name==='LEGAL LOSSES'));
});
test('Recovery estimates and reporting discrepancies remain visible without changing booked balances',()=>{
 const x=capture();assert.equal(x.recoveryProspects.estimatedAmount,310000);assert.deepEqual(x.recoveryProspects.estimatesAboveBalance,['TEST-B']);assert.equal(x.recoveryProspects.targetsOverdue.length,2);assert.equal(x.confidence.periodDiffersFromRecognition,1);assert.equal(x.totals.unrecovered,280000);
 assert.ok(run('lrpAgenda(lrpCapture(),false)').includes('Reconcile the recovery estimate'));
});
test('Repeated-period rows are flagged as candidates and never silently deduplicated',()=>{
 run('(()=>{const row=clone(S.losses.find(l=>l.id==="TEST-A"));row.id="TEST-DUPLICATE";row.reportingPeriod="2026-10";row.recoveries=[];S.losses.push(row);})()');
 const x=capture();assert.equal(x.confidence.duplicateCandidates.length,1);assert.deepEqual(x.confidence.duplicateCandidates[0].sort(),['TEST-A','TEST-DUPLICATE']);assert.equal(x.totals.gross,460000);
 run('S.losses=S.losses.filter(l=>l.id!=="TEST-DUPLICATE");');
});
test('Explicit scopes remain precise, and invalid periods, dates or currencies fail closed',()=>{
 const scoped=json('lossReportRows({basis:"recognition",from:"2026-11-01",to:"2026-11-13",scope:"B-IK",currency:"NGN"}).map(l=>l.id)');assert.ok(!scoped.includes('TEST-B'));assert.ok(scoped.includes('TEST-A'));
 assert.throws(()=>run('lossReportRows({scope:"FAKE"})'),/not valid/);
 assert.throws(()=>run('lossReportRows({basis:"anything"})'),/Choose occurrence/);
 assert.throws(()=>run('lossReportRows({currency:"ALL"})'),/supported reporting currency/);
 assert.throws(()=>run('lossReportRows({from:"2026-11-31"})'),/valid start/);
 assert.throws(()=>run('lossReportRows({to:"2026-12-01"})'),/beyond/);
});
test('All reporting lenses and empty states render through the actual source functions',()=>{
 for(const basis of ['occurrence','recognition','reporting'])for(const currency of ['NGN','USD','ZAR']){const x=capture({basis,currency}),html=run(`lrpBody(${JSON.stringify(x)},{frozen:false})`);assert.ok(html.includes('Recognition'));assert.ok(!html.includes('NaN'));assert.ok(!html.includes('[object Object]'));assert.ok(!html.includes('undefined'));}
 run('current={name:"reports",params:new URLSearchParams("view=loss-operations")};');assert.ok(run('reportsView()').includes('Operational loss insights'));
 run('current={name:"loss-insights",params:new URLSearchParams("currency=ALL")};current.blocked=false;');assert.ok(run('lrpView()').includes('supported reporting currency'));assert.equal(run('current.blocked'),true);
 run('current={name:"loss-insights",params:new URLSearchParams()};current.blocked=false;');
});
test('New board annexes are immutable and old report packs gain no live-data annex',()=>{
 run('window.LRP_PACK={name:"Synthetic loss committee pack",audience:"Board risk committee",sections:["losses"],commentary:"Investigate delayed recognition and recovery evidence.",snapshot:clone(reportData())};window.LRP_OLD_PACK=clone(LRP_PACK);delete LRP_OLD_PACK.snapshot.operationalLoss;');
 const pack=json('LRP_PACK'),html=run('brFrozenBody(LRP_PACK)'),oldHtml=run('brFrozenBody(LRP_OLD_PACK)');
 assert.ok(html.includes('Operational-loss annex'));assert.ok(!oldHtml.includes('Operational-loss annex'));
 run('S.losses[0].gross+=99999;S.losses[0].recoveries[0].amount+=10000;S.lossReportPreferences.currency="USD";S.scope={branch:"B-EN",department:"",unit:""};');
 assert.deepEqual(json('LRP_PACK'),pack);assert.equal(run('brFrozenBody(LRP_PACK)'),html);assert.equal(run('brFrozenBody(LRP_OLD_PACK)'),oldHtml);
 run(`S.losses=${JSON.stringify(fixture)};S.scope={branch:'',department:'',unit:''};S.lossReportPreferences.currency='NGN';`);
});
test('Cohort and receipt exports preserve dates, separate currencies, original account fields and reversal identity',()=>{
 run('window.LRP_DOWNLOADS=[];download=(name,body,type)=>LRP_DOWNLOADS.push({name,body,type});actions["lrp-export-cohort"]();actions["lrp-export-receipts"]();');
 const [cohort,receipts]=json('LRP_DOWNLOADS');assert.ok(cohort.body.includes('reporting_month'));assert.ok(cohort.body.includes('account_name'));assert.ok(cohort.body.includes('gross_minor'));assert.ok(!cohort.body.includes('TEST-USD'));
 assert.ok(receipts.body.includes('Reversal'));assert.ok(receipts.body.includes('REC-B'));assert.ok(receipts.body.includes('REC-A'));assert.ok(!receipts.body.includes('REC-PENDING'));
});

test('Near misses are not applicable to financial recognition and are not reported as missing dates',()=>{
 const x=capture({basis:'recognition'});assert.equal(x.totals.near,0);assert.equal(x.confidence.basisNotApplicable,1);assert.deepEqual(x.confidence.basisNotApplicableIds,['TEST-NEAR']);assert.deepEqual(x.confidence.basisMissingIds,['TEST-MISSING']);
 assert.ok(run('lrpContext(lrpCapture())').includes('this basis is not applicable to them'));
 assert.ok(!run('lrpLedger(lrpCapture(),{live:true})').includes('loss?id=TEST-NEAR'));
});

function integrated(){const rt=createRuntime();rt.run(`S.losses=${JSON.stringify(fixture)};S.asOf='2026-11-13';S.lossPeriod=90;S.scope={branch:'',department:'',unit:''};S.lossReportPreferences={basis:'recognition',from:'2026-11-01',to:S.asOf,currency:'NGN',group:'lossGroup'};`);return rt;}
const data=(rt,source)=>JSON.parse(JSON.stringify(rt.run(source)));

test('Combined management read models reconcile reviewed NGN totals and exclude foreign and unaccepted records',()=>{
 const rt=integrated();rt.navigate('reports?view=executive');const x=data(rt,'reportData()');
 assert.equal(x.losses.gross,210000);assert.equal(x.losses.recoveries,80000);assert.equal(x.losses.net,130000);assert.equal(x.losses.events,4);assert.equal(x.losses.nearMisses,1);
 assert.deepEqual(x.sourceRecords.losses.map(l=>l.id).sort(),['TEST-A','TEST-MISSING','TEST-CLOSED','TEST-NEAR','TEST-CORRECTION'].sort());
 assert.equal(x.board.lossTrend.reduce((n,v)=>n+v.net,0),130000);assert.equal(x.board.lossDrivers.reduce((n,v)=>n+v.net,0),130000);assert.equal(x.board.locations.rows.reduce((n,v)=>n+v.loss.net,0),130000);
 assert.equal(x.operationalLoss.config.basis,'occurrence');assert.equal(x.operationalLoss.currency,'NGN');assert.equal(x.operationalLoss.totals.unrecovered,130000);
 rt.run('S.scope={branch:"B-IK",department:"",unit:""};');const scoped=data(rt,'reportData()');assert.ok(scoped.sourceRecords.losses.every(l=>l.orgScopeId==='U-IK-PAY'));assert.equal(scoped.losses.net,130000);
});

test('Loss trend and adjacent comparison offsets are days and partition every supported period exactly once',()=>{
 const rt=integrated(),offsets=[-1,0,14,15,29,30,44,45,59,60,74,75,89,90,91,179,180,364,365,729];
 rt.run(`S.losses=${JSON.stringify(offsets)}.map((d,i)=>({...${JSON.stringify(base)},id:'RANGE-'+i,title:'Synthetic dated loss '+d,date:addDays(S.asOf,-d),recognizedOn:addDays(S.asOf,-d),reportingPeriod:addDays(S.asOf,-d).slice(0,7),gross:(i+1)*100,recoveries:[]}));S.losses.push({...clone(S.losses[1]),id:'RANGE-USD',currency:'USD',gross:99999999},{...clone(S.losses[1]),id:'RANGE-SUBMITTED',workflowState:'Submitted',gross:99999999});`);
 for(const period of [30,90,365]){
  rt.run(`current={name:'reports',params:new URLSearchParams('view=executive&period=${period}')};S.lossPeriod=${period};window.LRP_PERIOD_CALLS=[];window.LRP_REAL_PERIOD=ixLossPeriod;ixLossPeriod=(...args)=>{LRP_PERIOD_CALLS.push(args);return LRP_REAL_PERIOD(...args);};`);
  const html=rt.run('ixTrend()'),calls=data(rt,'LRP_PERIOD_CALLS');rt.run('ixLossPeriod=LRP_REAL_PERIOD;');
  const size=Math.ceil(period/6),expected=[];for(let i=5;i>=0;i--){const from=i*size,to=Math.min(period,(i+1)*size);if(from<period)expected.push([to-from,'',from]);}assert.deepEqual(calls,expected);
  const x=data(rt,'reportData()'),gross=offsets.reduce((n,d,i)=>n+(d>=0&&d<period?(i+1)*100:0),0),prior=offsets.reduce((n,d,i)=>n+(d>=period&&d<2*period?(i+1)*100:0),0);
  assert.equal(x.losses.gross,gross);assert.equal(x.board.lossTrend.reduce((n,v)=>n+v.gross,0),gross);assert.equal(x.board.priorLoss.gross,prior);
  for(const bin of x.board.lossTrend){const wanted=data(rt,`S.losses.filter(l=>l.workflowState==='Accepted'&&l.currency==='NGN'&&l.date>=${JSON.stringify(bin.from)}&&l.date<=${JSON.stringify(bin.to)}).reduce((n,l)=>n+l.gross,0)`);assert.equal(bin.gross,wanted);assert.ok(html.includes(`from=${bin.from}&to=${bin.to}`));}
 }
});

test('Risk monetary summaries use reviewed NGN while linked rows preserve state and original currencies',()=>{
 const rt=integrated(),html=rt.navigate('risk?id=RSK-04');assert.ok(html.includes('Lifetime reviewed gross · NGN'));assert.ok(html.includes(rt.run('ngn(410000)')));assert.ok(html.includes(rt.run('ngn(330000)')));assert.ok(html.includes('data-action="wf29-risk-edit"'),'The risk context editor remains reachable');assert.ok(html.includes('Continue assessment')||html.includes('Review proposal')||html.includes('Assessment history'),'The current assessment handoff remains reachable');
 const rows=rt.navigate('risk?id=RSK-04&tab=losses').split('<tr'),usd=rows.find(s=>s.includes('TEST-USD')),pending=rows.find(s=>s.includes('TEST-PENDING'));
 assert.ok(usd.includes('USD'));assert.ok(usd.includes(rt.run('l29Money(10000,"USD")')));assert.ok(usd.includes('currency reported separately'));assert.ok(!usd.includes(rt.run('ngn(10000)')));assert.ok(pending.includes('Proposed record; excluded from reviewed totals'));
 const ctx=data(rt,'copilotContext()');assert.equal(ctx.input.lossPopulation.currency,'NGN');assert.ok(ctx.input.losses.every(l=>l.currency==='NGN'&&['Accepted','Closed'].includes(l.workflowState)));assert.ok(ctx.facts.some(f=>f.text.includes('reviewed actual events')&&f.text.includes(rt.run('lrpMoney(330000,"NGN")'))));
 rt.run('S.risks.find(r=>r.id==="RSK-04").orgScopeId="U-IK-PAY";S.scope={branch:"B-IK",department:"",unit:""};');const scoped=rt.navigate('risk?id=RSK-04&tab=losses');assert.ok(!scoped.includes('TEST-B'));assert.ok(!data(rt,'copilotContext().input.losses').some(l=>l.id==='TEST-B'));
});

test('Copilot follows the visible loss currency, date basis, search and workflow filters with stable context hashes',()=>{
 const rt=integrated();rt.navigate('reports?view=loss-operations&basis=recognition&from=2026-11-01&to=2026-11-13&currency=USD');const ctx=data(rt,'copilotContext()');
 assert.equal(ctx.kind,'loss-report');assert.equal(ctx.input.operationalLoss.currency,'USD');assert.equal(ctx.input.operationalLoss.config.basis,'recognition');assert.equal(ctx.input.operationalLoss.totals.gross,10000);assert.ok(!('capturedAt' in ctx.input.operationalLoss));assert.equal(rt.run('copilotContext().hash'),ctx.hash);assert.ok(ctx.facts.some(f=>f.text.includes('USD')&&f.text.includes('date of recognition')));
 rt.navigate('losses?currency=NGN&basis=recognition&from=2026-11-01&to=2026-11-13&state=Submitted&q=unaccepted');const register=data(rt,'copilotContext()');assert.equal(register.kind,'loss-register');assert.equal(register.input.filters.q,'unaccepted');assert.equal(register.input.filters.state,'Submitted');assert.deepEqual(register.input.records.map(l=>l.id),['TEST-PENDING']);assert.equal(register.input.totals.gross,0);assert.ok(register.gaps.some(f=>f.text.includes('1 visible records are unaccepted')));
 rt.navigate('loss-insights?currency=USD&basis=reporting&from=2026-11-08&to=2026-11-10');assert.equal(rt.run('copilotContext().input.operationalLoss.config.basis'),'reporting');assert.notEqual(rt.run('copilotContext().hash'),ctx.hash);
});

test('Published annex explanations use captured figures and old packs gain no live financial context',()=>{
 const rt=integrated();rt.navigate('reports?view=loss-operations&currency=USD&basis=recognition&from=2026-11-01&to=2026-11-13');rt.run('S.reportPacks.push({id:"REPORT-LRP",name:"Synthetic frozen loss report",audience:"Board",sections:["losses"],commentary:"Review synthetic recovery evidence.",snapshot:clone(reportData())});const legacy=clone(S.reportPacks.at(-1));legacy.id="REPORT-LEGACY";delete legacy.snapshot.operationalLoss;S.reportPacks.push(legacy);current={name:"report-pack",params:new URLSearchParams("id=REPORT-LRP")};');
 const snapshot=data(rt,'S.reportPacks.find(p=>p.id==="REPORT-LRP").snapshot'),facts=data(rt,'copilotContext().facts'),body=rt.run('brFrozenBody(S.reportPacks.find(p=>p.id==="REPORT-LRP"))');assert.ok(facts.some(f=>f.text.startsWith('Captured annex: USD')));
 rt.run('S.losses.find(l=>l.id==="TEST-USD").gross=900000;S.losses.find(l=>l.id==="TEST-USD").recoveries[0].amount=4000;S.lossReportPreferences.currency="ZAR";S.scope={branch:"B-EN",department:"",unit:""};');
 assert.deepEqual(data(rt,'S.reportPacks.find(p=>p.id==="REPORT-LRP").snapshot'),snapshot);assert.deepEqual(data(rt,'copilotContext().facts'),facts);assert.equal(rt.run('brFrozenBody(S.reportPacks.find(p=>p.id==="REPORT-LRP"))'),body);
 rt.run('current={name:"report-pack",params:new URLSearchParams("id=REPORT-LEGACY")};');assert.ok(!data(rt,'copilotContext().facts').some(f=>f.text.startsWith('Captured annex:')));assert.ok(!rt.run('brFrozenBody(S.reportPacks.find(p=>p.id==="REPORT-LEGACY"))').includes('Operational-loss annex'));
});

test('Legacy detail entry points reach scoped workspaces and all new report controls remain wired',()=>{
 const rt=integrated();rt.run('riskDetail("RSK-04")');assert.equal(rt.context.location.hash.replace(/^#/,''),'/risk?id=RSK-04');rt.run('lossDetail("TEST-USD")');assert.equal(rt.context.location.hash.replace(/^#/,''),'/loss?id=TEST-USD');rt.run('actions["risk-detail"]({dataset:{id:"RSK-04"}})');assert.equal(rt.context.location.hash.replace(/^#/,''),'/risk?id=RSK-04');rt.run('actions["loss-detail"]({dataset:{id:"TEST-A"}})');assert.equal(rt.context.location.hash.replace(/^#/,''),'/loss?id=TEST-A');
 for(const route of ['loss-insights','reports?view=loss-operations','loss-insights?currency=ZAR','loss-insights?basis=recognition','risk?id=RSK-04&tab=losses']){const html=rt.navigate(route);assert.ok(html.length>500);assert.ok(!/\bNaN\b|\[object Object\]|\bundefined\b/.test(html));for(const [,action] of html.matchAll(/data-action="([^"]+)"/g))assert.equal(rt.run(`typeof actions[${JSON.stringify(action)}]`),'function',action);for(const [,dest] of html.matchAll(/href="#\/([^"?]+)/g))assert.ok(rt.run(`!!titles[${JSON.stringify(dest)}]`),dest);}
 rt.run('lossDetail()');assert.equal(rt.context.location.hash.replace(/^#/,''),'/loss?id=unavailable');assert.ok(rt.navigate('loss?id=unavailable').includes('record-unavailable'));
});

const report={suite:'loss-reports-v2.9',validation:'Actual JavaScript sources with minimal DOM stubs; browser layout not tested',checks:results,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length};fs.writeFileSync(path.join(ROOT,'tests/loss-reports-v29-results.json'),JSON.stringify(report,null,2));console.log(JSON.stringify({passed:report.passed,failed:report.failed}));process.exitCode=report.failed?1:0;
