/* Actual-source presentation contracts. DOM nodes are controlled fixtures;
 * these checks do not claim browser layout, focus or print pagination testing. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict'),Module=require('node:module');
const root=path.resolve(__dirname,'..'),file=path.join(__dirname,'runtime_harness.js');
const baseline=new Module(file,module);baseline.filename=file;baseline.paths=module.paths;
baseline._compile(fs.readFileSync(file,'utf8').replace(/(?:'premium-privacy'|"premium-privacy")\s*,?/g,''),file);
const {createRuntime}=baseline.exports,adapter=fs.readFileSync(path.join(root,'assets/premium-privacy.js'),'utf8'),results=[];
function fresh(){const h=createRuntime();h.run("wf29Init();S.governance.actorId='USR-03';");return h;}
function load(h){h.run(adapter);}
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function plain(value){return JSON.parse(JSON.stringify(value));}
function node(attrs={},extra={}){
 const attributes={...attrs};return {attributes,hasAttribute:name=>Object.hasOwn(attributes,name),setAttribute:(name,value)=>attributes[name]=value,querySelector:()=>null,closest:()=>null,...extra};
}

test('Loading changes only the render presentation chain, with no state, route or command changes',()=>{
 const h=fresh();h.navigate('privacy-case?id=PRIV-REQ-01');
 h.run("window.PV214_BEFORE={state:recStable(S),privacy:privacyView,integrations:integrationsView,source:CS_JOURNEYS.activity,dpia:CS_JOURNEYS.dpia,caseView:CS_JOURNEYS['privacy-case'],submit:jSubmitActivity,review:jActivityDecision,dpiaReview:jDpiaDecision,caseReview:jCaseDecision,caseUpdate:jCaseUpdate,runCheck:actions['pv211-run-check'],draftSubmit:actions['j-submit'],render};");
 const html=h.elements.get('#main-content').innerHTML,route=h.context.location.hash;load(h);
 assert.equal(h.run('recStable(S)'),h.run('PV214_BEFORE.state'));
 assert.equal(h.elements.get('#main-content').innerHTML,html);assert.equal(h.context.location.hash,route);
 assert.equal(h.run("privacyView===PV214_BEFORE.privacy&&integrationsView===PV214_BEFORE.integrations&&CS_JOURNEYS.activity===PV214_BEFORE.source&&CS_JOURNEYS.dpia===PV214_BEFORE.dpia&&CS_JOURNEYS['privacy-case']===PV214_BEFORE.caseView"),true);
 assert.equal(h.run("jSubmitActivity===PV214_BEFORE.submit&&jActivityDecision===PV214_BEFORE.review&&jDpiaDecision===PV214_BEFORE.dpiaReview&&jCaseDecision===PV214_BEFORE.caseReview&&jCaseUpdate===PV214_BEFORE.caseUpdate&&actions['pv211-run-check']===PV214_BEFORE.runCheck&&actions['j-submit']===PV214_BEFORE.draftSubmit"),true);
 assert.equal(h.run('render===PV214_BEFORE.render'),false);
});

test('All ten route families and privacy subviews retain exact source content, controls and values',()=>{
 const h=fresh();h.run('window.PV214_BASE_RENDER=render;');load(h);h.run('window.PV214_PRESENTATION_RENDER=render;');
 const routes=[
  'privacy','privacy?tab=register','privacy?tab=dpias','privacy?tab=sharing','privacy?tab=cases','privacy?tab=drafts',
  'activity?id=ROPA-01','activity?id=ROPA-01&tab=flow','activity?id=ROPA-01&tab=basis','activity?id=ROPA-01&tab=screening','activity?id=ROPA-01&tab=review','activity?id=ROPA-02&tab=history','activity-version?id=ROPA-02&revision=1',
  'dpia?id=DPIA-01','dpia?id=DPIA-01&tab=actions','dpia?id=DPIA-01&tab=decision','dpia?id=DPIA-02&tab=history',
  'privacy-case?id=PRIV-REQ-01','privacy-case?id=PRIV-INC-01','privacy-case?id=PRIV-REQ-01&tab=decision','privacy-case?id=PRIV-REQ-01&tab=history',
  'activity-wizard','dpia-wizard','case-intake','integrations','data-health'
 ];
 for(const destination of routes){
  h.run('render=PV214_BASE_RENDER');const before=h.navigate(destination);
  h.run('render=PV214_PRESENTATION_RENDER');const after=h.navigate(destination);
  assert.equal(after,before,destination);assert.ok(h.elements.get('#main-content').dataset.pv214,destination);
  assert.ok(!after.includes('[object Object]'),destination);
 }
 h.run("S.dpias.find(x=>x.id==='DPIA-01').state='In review';S.privacyCases.find(x=>x.id==='PRIV-INC-01').state='Awaiting review';");
 for(const destination of ['dpia?id=DPIA-01&tab=decision','privacy-case?id=PRIV-INC-01&tab=decision']){
  h.run('render=PV214_BASE_RENDER');const before=h.navigate(destination);assert.match(before,/data-j-submit="(?:dpia|case)-decision"/);
  h.run('render=PV214_PRESENTATION_RENDER');assert.equal(h.navigate(destination),before,destination);
 }
});

test('Activity, DPIA and both case-intake journeys preserve exact inputs and saved source data across all steps',()=>{
 const h=fresh();h.run("S.governance.actorId='USR-01';jStartActivity();jStartDpia('ROPA-01');jNewCase('Rights request');jNewCase('Privacy incident');window.PV214_BASE_RENDER=render;");
 const drafts=plain(h.run('Object.values(S.journeyDrafts).map(x=>({id:x.id,kind:x.kind}))'));load(h);h.run('window.PV214_PRESENTATION_RENDER=render;');
 const config={activity:['activity-wizard',6],dpia:['dpia-wizard',5],case:['case-intake',3]};
 for(const draft of drafts){const [route,count]=config[draft.kind];for(let step=1;step<=count;step++){
  const destination=route+'?draft='+draft.id+'&step='+step;h.run('render=PV214_BASE_RENDER');const before=h.navigate(destination),state=h.run('recStable(S.journeyDrafts)');
  h.run('render=PV214_PRESENTATION_RENDER');const after=h.navigate(destination);
  assert.equal(after,before,destination);assert.equal(h.run('recStable(S.journeyDrafts)'),state,destination);assert.equal(h.elements.get('#main-content').dataset.pv214,'capture');
  assert.match(after,/data-action="j-save-exit"/);assert.match(after,/data-action="j-step"/);
 }}
});

test('Refresh adds only missing accessible names and keyboard scroll access; existing semantics are retained',()=>{
 const h=fresh();load(h);const main=h.elements.get('#main-content'),heading={textContent:'Purposes & supporting basis'};
 const table=node({}, {querySelector:selector=>selector==='.j-edit-table'?{}:null,closest:()=>({querySelector:()=>heading})});
 const preserved=node({role:'group','aria-labelledby':'original-title',tabindex:'-1'}, {querySelector:()=>({}),closest:()=>null});
 const otherTable=node();const footer=node(),caseActions=node(),decision=node(),namedDecision=node({'aria-label':'Existing review name'});
 const regions={'.j-table-scroll':[table,preserved,otherTable],'.j-actions':[footer],'.pv211-case-actions':[caseActions],'form[data-j-submit="activity-decision"]':[decision,namedDecision]};
 main.querySelectorAll=selector=>regions[selector]||[];
 h.run("current={name:'activity-wizard',params:new URLSearchParams()};window.PV214_STATE=recStable(S);pv214Refresh();pv214Refresh();");
 assert.deepEqual(table.attributes,{role:'region','aria-label':'Purposes & supporting basis — table',tabindex:'0'});
 assert.deepEqual(preserved.attributes,{role:'group','aria-labelledby':'original-title',tabindex:'-1'});assert.deepEqual(otherTable.attributes,{});
 assert.deepEqual(footer.attributes,{role:'group','aria-label':'Privacy draft actions'});assert.equal(caseActions.attributes['aria-label'],'Case preparation actions');
 assert.deepEqual(decision.attributes,{'aria-label':'Independent processing record review'});assert.equal(namedDecision.attributes['aria-label'],'Existing review name');
 assert.equal(h.run('recStable(S)'),h.run('PV214_STATE'));
});

test('Navigation cleans only its own route marker and preserves other premium adapters and render result',()=>{
 const h=fresh(),main=h.elements.get('#main-content');main.dataset.pf213='review';main.dataset.pp214='program';
 h.run("window.PV214_CALLS=[];render=function(...args){PV214_CALLS.push(args);return 'prior-render-result';};");load(h);
 h.run("current={name:'dpia',params:new URLSearchParams()}");assert.equal(h.run("render('retained-argument')"),'prior-render-result');assert.deepEqual(plain(h.run('PV214_CALLS')),[['retained-argument']]);assert.equal(main.dataset.pv214,'record');
 let queries=0;main.querySelectorAll=()=>{queries++;return [];};h.run("current={name:'reports',params:new URLSearchParams()};pv214Refresh()");assert.equal(queries,0);assert.equal(main.dataset.pv214,undefined);assert.equal(main.dataset.pf213,'review');assert.equal(main.dataset.pp214,'program');
});

test('Historical versions, pending facts and independent decisions remain untouched by presentation refresh',()=>{
 const h=fresh();h.navigate('activity-version?id=ROPA-02&revision=1');load(h);h.run('window.PV214_STATE=recStable(S);');
 for(const route of ['privacy','activity','activity-wizard','activity-version','dpia','dpia-wizard','privacy-case','case-intake','integrations','data-health']){
  h.context.pv214TestRoute=route;h.run('current={name:pv214TestRoute,params:new URLSearchParams()};pv214Refresh()');
 }
 assert.equal(h.run('recStable(S)'),h.run('PV214_STATE'));
 h.run("current={name:'activity-version',params:new URLSearchParams('id=ROPA-02&revision=1')};");assert.match(h.run("CS_JOURNEYS['activity-version']()"),/This snapshot is not changed by a new draft/);
});

test('Exact Documents labels and factual integration quality remain backed by canonical source renderers',()=>{
 const h=fresh();load(h);const dpia=h.navigate('dpia?id=DPIA-01&tab=actions'),health=h.navigate('data-health');
 assert.match(dpia,/Actions &amp; documents|Actions & documents/);assert.doesNotMatch(dpia,/Actions &amp; evidence|Actions & evidence/);assert.match(health,/<dt>Assessment documents<\/dt>/);assert.doesNotMatch(health,/<dt>Evidence<\/dt>/);
 h.run('S.fixtureRuns=[];');let html=h.navigate('integrations');assert.equal((html.match(/>Not run<\/span>/g)||[]).length,3);assert.match(html,/Local fixtures only/);assert.match(html,/No checks run yet/);
 h.run("window.PV214_PROGRAM_STATE=recStable(S.programs);fixtureRun('Cloud fixture');");html=h.navigate('integrations');
 assert.ok(html.includes(h.run('e(S.fixtureRuns[0].quality)')));assert.ok(html.includes(h.run('e(S.fixtureRuns[0].observation)')));assert.ok(html.includes(h.run('e(recTimestamp(S.fixtureRuns[0].time))')));assert.equal(h.run('recStable(S.programs)'),h.run('PV214_PROGRAM_STATE'));
});

const report={suite:'V2.14 premium privacy and monitoring presentation',runtime:'Actual application sources with controlled DOM semantic nodes',browserLayoutValidated:false,pdfPaginationValidated:false,results,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length};
fs.writeFileSync(path.join(root,'tests/premium-privacy-v214-results.json'),JSON.stringify(report,null,2)+'\n');console.log(report.passed+'/'+results.length+' presentation contract checks passed.');if(report.failed)process.exitCode=1;
