/* Functional checks for display preferences. Layout and native focus still
 * require a real browser; the actual application source is loaded here. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function plain(value){return JSON.parse(JSON.stringify(value));}
function fresh(){const h=createRuntime();h.navigate('my-workspace');return h;}

test('Missing or malformed display preferences fall back safely without mutating records',()=>{
 const h=fresh();h.run("delete S.displayPreferences;window.PD213_BEFORE=recStable(S);");
 assert.deepEqual(plain(h.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'light',density:'comfortable'});
 assert.equal(h.run('recStable(S)'),h.run('PD213_BEFORE'));
 h.run("S.displayPreferences={density:'unknown'};S.theme='unknown';window.PD213_BEFORE=recStable(S);pd213ApplyAttributes();");
 assert.deepEqual(plain(h.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'light',density:'comfortable'});
 assert.equal(h.run('recStable(S)'),h.run('PD213_BEFORE'));
 assert.deepEqual(plain(h.context.document.documentElement.dataset),{theme:'light',density:'comfortable'});
});

test('Changing appearance preserves the active route, unsaved form, modal and all domain state',()=>{
 const h=fresh();h.context.location.hash='#/loss-intake?org=B-IK&step=2';
 h.elements.get('#main-content').innerHTML='<form><textarea>Unsubmitted account of the event</textarea></form>';
 const markup=h.elements.get('#main-content').innerHTML;
 h.run("S.scope={branch:'B-IK',department:'',unit:''};window.PD213_STATE=clone(S);window.PD213_RENDER=0;render=()=>PD213_RENDER++;UI211_MODAL={edited:true,baseline:'keep-original',form:{}};");
 h.run("CLEAR_SIGHT_PREMIUM.apply({theme:'dark',density:'compact'})");
 assert.equal(h.run('PD213_RENDER'),0);assert.equal(h.elements.get('#main-content').innerHTML,markup);
 assert.equal(h.context.location.hash,'#/loss-intake?org=B-IK&step=2');
 assert.equal(h.run("UI211_MODAL.edited&&UI211_MODAL.baseline==='keep-original'"),true);
 assert.equal(h.run("const after=clone(S);delete after.theme;delete after.displayPreferences;const before=clone(PD213_STATE);delete before.theme;delete before.displayPreferences;recStable(after)===recStable(before)"),true);
 assert.deepEqual(plain(h.context.document.documentElement.dataset),{theme:'dark',density:'compact'});
});

test('Preferences survive reload and supported V2.12/V2.13/V2.14 state migrates without replacing its scope or drafts',()=>{
 const h=fresh();h.run("S.scope={branch:'B-EN',department:'',unit:''};S.premiumTestDraft={summary:'Retain this unsent working entry'};CLEAR_SIGHT_PREMIUM.apply({theme:'dark',density:'compact'});");
 const reloaded=createRuntime({initialStorage:[...h.storage]});
 assert.deepEqual(plain(reloaded.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'dark',density:'compact'});
 assert.equal(reloaded.run('S.premiumTestDraft.summary'),'Retain this unsent working entry');assert.equal(reloaded.run('S.scope.branch'),'B-EN');
 const old=JSON.parse(h.storage.get(h.run('KEY')));
 for(const key of ['clearsight-v2.14-demo-20261121','clearsight-v2.13-demo-20261121']){
  const prior=createRuntime({initialStorage:[[key,JSON.stringify(old)]]});
  assert.deepEqual(plain(prior.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'dark',density:'compact'});
  assert.equal(prior.run('S.premiumTestDraft.summary'),'Retain this unsent working entry');assert.equal(prior.run('S.scope.branch'),'B-EN');
 }
 delete old.displayPreferences;
 const migrated=createRuntime({initialStorage:[['clearsight-v2.12-demo-20261121',JSON.stringify(old)]]});
 assert.deepEqual(plain(migrated.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'dark',density:'comfortable'});
 assert.equal(migrated.run('S.premiumTestDraft.summary'),'Retain this unsent working entry');assert.equal(migrated.run('S.scope.branch'),'B-EN');
});

test('Invalid preferences are rejected atomically before state or persisted records change',()=>{
 const h=fresh(),before=h.run('recStable(S)'),stored=[...h.storage];
 for(const source of ['null',"{theme:'sepia',density:'compact'}","{theme:'dark',density:'tiny'}","{density:'compact'}"])
  assert.throws(()=>h.run('CLEAR_SIGHT_PREMIUM.apply('+source+')'),/valid theme and display density/);
 assert.equal(h.run('recStable(S)'),before);assert.deepEqual([...h.storage],stored);
});

test('Existing theme action preserves density and does not rerender or reset ongoing work',()=>{
 const h=fresh();h.run("CLEAR_SIGHT_PREMIUM.apply({theme:'light',density:'compact'});window.PD213_RENDER=0;render=()=>PD213_RENDER++;");
 const markup=h.elements.get('#main-content').innerHTML;h.run('actions.theme()');
 assert.deepEqual(plain(h.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'dark',density:'compact'});
 assert.equal(h.run('PD213_RENDER'),0);assert.equal(h.elements.get('#main-content').innerHTML,markup);
 h.run('actions.theme()');assert.equal(h.run('S.theme'),'light');
});

test('Display command protects an existing dialog and its pending changes',()=>{
 const h=fresh();h.elements.get('#dialog').open=true;h.elements.get('#dialog').innerHTML='Unsaved review decision';
 h.run('window.PD213_OPENED=0;openDialog=()=>PD213_OPENED++;');
 assert.equal(h.run("actions['pd213-display']()"),false);assert.equal(h.run('PD213_OPENED'),0);
 assert.equal(h.elements.get('#dialog').innerHTML,'Unsaved review decision');
});

test('Display form commits explicit selected values through the existing save callback',()=>{
 const h=fresh();const markup=h.elements.get('#main-content').innerHTML;
 h.context.FormData=class {constructor(form){this.form=form;}get(name){return this.form[name]??null;}};
 h.run("actions['pd213-display']();modalSave({pfTheme:'dark',pfDensity:'compact'});");
 assert.deepEqual(plain(h.run('CLEAR_SIGHT_PREMIUM.preferences()')),{theme:'dark',density:'compact'});
 assert.equal(h.elements.get('#main-content').innerHTML,markup);
 assert.throws(()=>h.run("modalSave({pfTheme:'dark',pfDensity:'unsupported'})"),/valid theme/);
});

test('Applied appearance updates accessible chrome context and settings text in place',()=>{
 const h=fresh(),attributes={},control={setAttribute:(name,value)=>attributes[name]=value},summary={textContent:''};
 h.context.document.querySelectorAll=selector=>selector==='.topbar [data-action="pd213-display"]'?[control]:selector==='[data-pd213-summary]'?[summary]:[];
 h.run("CLEAR_SIGHT_PREMIUM.apply({theme:'dark',density:'compact'})");
 assert.equal(attributes['aria-label'],'Display preferences');assert.match(attributes.title,/Dark · Compact/);assert.match(summary.textContent,/Dark appearance · Compact density/);
 h.context.document.querySelectorAll=()=>[];
 h.run("current={name:'settings',params:new URLSearchParams('tab=demo')}");assert.match(h.run('settingsView()'),/Dark appearance · Compact density/);
 h.run("current={name:'settings',params:new URLSearchParams('tab=history')}");assert.doesNotMatch(h.run('settingsView()'),/data-pd213-summary/);
});

const report={suite:'V2.13 display preference interactions',runtime:'Actual source with substituted browser/DOM shell',browserLayoutValidated:false,results};
fs.writeFileSync(path.join(root,'tests/premium-foundation-v213-results.json'),JSON.stringify(report,null,2)+'\n');
console.log(`${results.filter(r=>r.status==='PASS').length}/${results.length} display interaction checks passed.`);
if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
