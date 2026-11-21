/* Presentation contract checks using the actual application source handlers.
 * No browser layout, PDF pagination or CSS appearance claim is made. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict'),Module=require('node:module');
const root=path.resolve(__dirname,'..'),file=path.join(__dirname,'runtime_harness.js');
// Keep the common harness authoritative. An in-memory baseline omits only the
// new presentation module, including after the root adds it to the build list.
const baselineModule=new Module(file,module);baselineModule.filename=file;baselineModule.paths=module.paths;
const harnessSource=fs.readFileSync(file,'utf8');
baselineModule._compile(harnessSource.replace(/(?:'premium-reporting'|"premium-reporting")\s*,?/g,''),file);
const {createRuntime}=baselineModule.exports,adapter=fs.readFileSync(path.join(root,'assets/premium-reporting.js'),'utf8'),results=[];
function fresh(){const h=createRuntime();h.run("wf29Init();S.governance.actorId='USR-02';");return h;}
function load(h){h.run(adapter);}
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function stripWrapper(html,surface){const prefix=`<div class="pr213-report pr213-report--${surface}">`;assert.ok(html.startsWith(prefix),surface);assert.ok(html.endsWith('</div>'),surface);return html.slice(prefix.length,-6);}

test('Adapter loading does not render, mutate application state or replace source read models',()=>{
 const h=fresh();h.navigate('reports?org=B-IK&view=executive');
 h.run('window.PR213_BEFORE={state:recStable(S),read:reportData,frozen:brFrozenBody,loss:lrpCapture,response:arpCapture,render};');const before=h.elements.get('#main-content').innerHTML;load(h);
 assert.equal(h.run('recStable(S)'),h.run('PR213_BEFORE.state'));assert.equal(h.run('reportData===PR213_BEFORE.read'),true);assert.equal(h.run('brFrozenBody===PR213_BEFORE.frozen'),true);assert.equal(h.run('lrpCapture===PR213_BEFORE.loss'),true);assert.equal(h.run('arpCapture===PR213_BEFORE.response'),true);assert.equal(h.run('render===PR213_BEFORE.render'),true);assert.equal(h.elements.get('#main-content').innerHTML,before);
});

test('Dashboard and all report perspectives retain exact prior content, values, controls and routes',()=>{
 const h=fresh();h.run('window.PR213_ORIGINAL_HOME=homeView;window.PR213_ORIGINAL_REPORTS=reportsView;');load(h);
 for(const destination of ['home?org=B-IK','reports?view=executive&org=B-IK','reports?view=locations&org=D-HO-TECH','reports?view=drivers&org=all','reports?view=kri&org=B-IK','reports?view=assurance&org=D-HO-TECH','reports?view=dependencies&org=B-IK','reports?view=quality&org=B-IK','reports?view=library&org=all','reports?view=loss-operations&org=B-IK&currency=USD&basis=recognition&from=2026-10-01&to=2026-11-13','reports?view=responses&org=all&state=pending']){
  h.navigate(destination);const home=destination.startsWith('home'),before=h.run(home?'PR213_ORIGINAL_HOME()':'PR213_ORIGINAL_REPORTS()'),after=h.run(home?'homeView()':'reportsView()');assert.equal(stripWrapper(after,home?'dashboard':'insights'),before,destination);
 }
});

test('Detail routes retain their prior scope, currency, date and source checks',()=>{
 const h=fresh();h.run("window.PR213_ORIGINAL_ROUTES=Object.fromEntries(['loss-insights','assessment-insights','insight-detail'].map(name=>[name,CS_JOURNEYS[name]]));");load(h);
 for(const [route,surface]of [['loss-insights?org=B-IK&basis=reporting&from=2026-11-01&to=2026-11-13&currency=USD','losses'],['assessment-insights?org=all&state=pending','responses'],['insight-detail?org=B-IK&kind=appetite','sources']]){
  h.navigate(route);const name=route.split('?')[0],before=h.run(`PR213_ORIGINAL_ROUTES[${JSON.stringify(name)}]()`),after=h.run(`CS_JOURNEYS[${JSON.stringify(name)}]()`);assert.equal(stripWrapper(after,surface),before);
 }
});

test('Captured pack and read-only builder presentation leave the frozen renderer and snapshot unchanged',()=>{
 const h=fresh();h.navigate('reports?org=D-HO-TECH&view=assurance');h.run("window.PR213_PACK={id:'SYN-PR213',name:'Captured board review',audience:'Board risk committee',sections:['risks','losses','kri','assurance','actions'],commentary:'Retain the captured source and reporting limitations.',snapshot:clone(reportData())};S.reportPacks.push(PR213_PACK);S.reportDraft={...clone(PR213_PACK),step:3};window.PR213_FROZEN=brFrozenBody(PR213_PACK);window.PR213_CAPTURE=recStable(PR213_PACK);window.PR213_ORIGINAL_PACK=CS_JOURNEYS['report-pack'];window.PR213_ORIGINAL_BUILDER=CS_JOURNEYS['report-builder'];");load(h);
 h.navigate('report-pack?id=SYN-PR213&org=B-EN');assert.equal(stripWrapper(h.run("CS_JOURNEYS['report-pack']()"),'pack'),h.run('PR213_ORIGINAL_PACK()'));
 h.navigate('report-builder');assert.equal(stripWrapper(h.run("CS_JOURNEYS['report-builder']()"),'builder'),h.run('PR213_ORIGINAL_BUILDER()'));assert.equal(h.run('brFrozenBody(PR213_PACK)'),h.run('PR213_FROZEN'));assert.equal(h.run('recStable(PR213_PACK)'),h.run('PR213_CAPTURE'));
});

test('No-access and invalid-filter reporting states retain the original blocker and disabled controls',()=>{
 const h=fresh();h.run('window.PR213_REPORTS=reportsView;');load(h);
 h.navigate('loss-insights?org=B-IK&currency=ALL');assert.equal(h.run('current.blocked'),true);assert.ok(h.elements.get('#main-content').innerHTML.includes('supported reporting currency'));
 h.run("S.governance.actorId='USR-01';current={name:'reports',params:new URLSearchParams('org=all')};S.scope=sa212Scope('');scopeInit();");assert.equal(stripWrapper(h.run('reportsView()'),'insights'),h.run('PR213_REPORTS()'));assert.equal(h.run('current.blocked'),true);
});

test('Printing opens only reporting analysis and restores every prior disclosure state',()=>{
 const h=fresh();load(h);h.run('window.PR213_STATE=recStable(S);');
 const detail=(open,{builder=false,frozen=false}={})=>({open,closest:selector=>selector==='.br-builder'?builder?{}:null:selector==='.br-frozen-pack'?frozen?{}:null:null});
 const closed=detail(false),alreadyOpen=detail(true),editor=detail(false,{builder:true}),capturedPreview=detail(false,{builder:true,frozen:true}),unrelated=detail(false);
 let selector='';h.context.document.querySelectorAll=value=>{selector=value;return [closed,alreadyOpen,editor,capturedPreview];};
 h.run('pr213BeforePrint()');assert.ok(selector.startsWith('.pr213-report '));assert.equal(closed.open,true);assert.equal(alreadyOpen.open,true);assert.equal(capturedPreview.open,true);assert.equal(editor.open,false);assert.equal(unrelated.open,false);assert.equal(h.run('recStable(S)'),h.run('PR213_STATE'));
 h.run('pr213AfterPrint()');assert.equal(closed.open,false);assert.equal(alreadyOpen.open,true);assert.equal(capturedPreview.open,false);assert.equal(editor.open,false);assert.equal(unrelated.open,false);assert.equal(h.run('recStable(S)'),h.run('PR213_STATE'));
});

test('Repeated print-entry events do not overwrite original disclosure state or captured output',()=>{
 const h=fresh();h.navigate('reports?org=B-IK');h.run("window.PR213_PACK={name:'Print source',audience:'Board',sections:['kri'],snapshot:clone(reportData())};window.PR213_BODY=brFrozenBody(PR213_PACK);window.PR213_STATE=recStable(S);");load(h);
 const closed={open:false,closest:()=>null},open={open:true,closest:()=>null};h.context.document.querySelectorAll=()=>[closed,open];h.run('pr213BeforePrint();pr213BeforePrint();');assert.equal(closed.open,true);h.run('pr213AfterPrint();pr213AfterPrint();');assert.equal(closed.open,false);assert.equal(open.open,true);assert.equal(h.run('brFrozenBody(PR213_PACK)'),h.run('PR213_BODY'));assert.equal(h.run('recStable(S)'),h.run('PR213_STATE'));
});

test('Native print listeners are installed without triggering print or opening disclosures on load',()=>{
 const h=fresh(),events=[];let queried=0;h.context.addEventListener=(name,handler)=>events.push({name,handler});h.context.document.querySelectorAll=()=>{queried++;return [];};load(h);assert.deepEqual(events.map(v=>v.name),['beforeprint','afterprint']);assert.equal(queried,0);events[0].handler();events[1].handler();assert.equal(queried,1);
});

const report={suite:'V2.13 premium reporting presentation',runtime:'Actual application sources and controlled DOM disclosure nodes',browserLayoutValidated:false,pdfPaginationValidated:false,results,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length};fs.writeFileSync(path.join(root,'tests/premium-reporting-v213-results.json'),JSON.stringify(report,null,2));console.log(`${report.passed}/${results.length} presentation contract checks passed`);if(report.failed)process.exitCode=1;
