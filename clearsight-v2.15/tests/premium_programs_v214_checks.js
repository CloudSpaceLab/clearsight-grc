/* Actual-source marker and field-preservation checks; no browser layout claim. */
'use strict';
const assert=require('assert'),fs=require('fs'),vm=require('vm');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function runtime(){const h=createRuntime();if(h.run('typeof CLEAR_SIGHT_PREMIUM_PROGRAMS')==='undefined')vm.runInContext(fs.readFileSync(root+'/assets/premium-programs.js','utf8'),h.context);h.run(`S.governance.actorId='USR-05';S.scope={branch:'',department:'',unit:''};`);return h;}
function check(name,fn){try{fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL',name,error.stack);}}

check('The eight-route marker resets without rewriting content, sibling markers or state',h=>{
 const main=h.elements.get('#main-content');main.innerHTML='<p>Existing source content</p>';main.dataset.pf213='response';main.dataset.pv214='sibling';
 const before=h.run('JSON.stringify(S)'),stored=[...h.storage];
 const routes=JSON.parse(h.run('JSON.stringify(CLEAR_SIGHT_PREMIUM_PROGRAMS.routes)'));
 assert.deepEqual(routes,['frameworks','adopt','programs','program','forms','form-builder','check-runs','setup']);
 for(const route of routes){h.run(`current={name:${JSON.stringify(route)},params:new URLSearchParams()};CLEAR_SIGHT_PREMIUM_PROGRAMS.refresh();`);assert.equal(main.dataset.pp214,route);assert.equal(main.innerHTML,'<p>Existing source content</p>');}
 h.run(`current={name:'work',params:new URLSearchParams()};CLEAR_SIGHT_PREMIUM_PROGRAMS.refresh();`);
 assert(!('pp214' in main.dataset));assert.equal(main.dataset.pf213,'response');assert.equal(main.dataset.pv214,'sibling');
 assert.equal(h.run('JSON.stringify(S)'),before);assert.deepEqual([...h.storage],stored);
});

check('Actual catalog, adoption, program, editor, history and setup routes retain source controls',h=>{
 const program=h.run('S.programs[0].id');
 const cases=[['frameworks','data-action="start-adoption"'],['adopt','data-draft="name"'],['programs','href="#/program?id='],['program?id='+program+'&tab=requirements','data-action="program-requirement"'],['forms','data-action="fa-create"'],['form-builder?form=FORM-OPS','data-cx-form-name'],['check-runs','Execution ledger']];
 for(const [route,expected]of cases){const html=h.navigate(route);assert.equal(h.elements.get('#main-content').dataset.pp214,route.split('?')[0]);assert(html.includes(expected),route);for(const [,action]of html.matchAll(/data-action="([^"]+)"/g))assert(h.run(`typeof actions[${JSON.stringify(action)}]==='function'`),route+': '+action);}
 h.run(`S.governance.actorId='USR-01';jStartOperation('risk');`);const route=h.context.location.hash.replace(/^#?\//,'');assert(route.startsWith('setup?'));
 const html=h.navigate(route);assert.equal(h.elements.get('#main-content').dataset.pp214,'setup');assert(html.includes('data-j-path="title"'));assert(html.includes('data-action="j-next"'));assert(html.includes('id="j-save-indicator"'));
 const before=h.run('JSON.stringify({forms:S.formStudio,adoption:S.draft,journeys:S.journeyDrafts,access:S.accessControl})');
 h.run('CLEAR_SIGHT_PREMIUM_PROGRAMS.refresh()');assert.equal(h.elements.get('#main-content').innerHTML,html);assert.equal(h.run('JSON.stringify({forms:S.formStudio,adoption:S.draft,journeys:S.journeyDrafts,access:S.accessControl})'),before);
});

check('Draft navigation and repeated module loading preserve field IDs, values and render identity',h=>{
 h.run(`gvState().directRules.push({id:'PP214-FORM-AUTHOR',userId:'USR-05',permission:'programs.edit',effect:'allow',scopeId:'',starts:'2026-01-01',ends:'',status:'Active',approvedBy:'USR-03',reason:'Synthetic form-author permission for shared definition preservation checks.'});re212FormDefinition();`);
 h.navigate('form-builder?form=FORM-OPS');h.run(`var ppForm=faForm();ppForm.fields[0].label='Preserved named respondent field';ppForm.description='Preserved authored introduction.';cxSave(ppForm);`);
 const form=h.run('JSON.stringify(ppForm)'),ids=JSON.parse(h.run('JSON.stringify(ppForm.fields.map(q=>q.id))'));
 let html=h.navigate('form-builder?form=FORM-OPS');assert(html.includes('Preserved named respondent field'));assert(html.includes('Preserved authored introduction.'));for(const id of ids)assert(html.includes('data-cx-card="'+id+'"'));
 assert.equal(h.run('JSON.stringify(ppForm)'),form);
 h.navigate('adopt');h.run(`draft().name='Preserved scoped adoption';draft().excluded[draft().reqs[0]]='Source applicability remains explicitly documented.';save();`);
 h.navigate('work');assert(!('pp214' in h.elements.get('#main-content').dataset));html=h.navigate('adopt');assert(html.includes('Preserved scoped adoption'));assert(html.includes('Source applicability remains explicitly documented.'));
 h.run('var ppPriorRender=render');const main=h.elements.get('#main-content'),beforeHTML=main.innerHTML,before=h.run('JSON.stringify(S)');
 vm.runInContext(fs.readFileSync(root+'/assets/premium-programs.js','utf8'),h.context);
 assert(h.run('render===ppPriorRender'));assert.equal(main.innerHTML,beforeHTML);assert.equal(h.run('JSON.stringify(S)'),before);
});

const report={suite:'V2.14 premium programs source preservation',browserLayoutValidated:false,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,results};
fs.writeFileSync(root+'/tests/premium-programs-v214-results.json',JSON.stringify(report,null,2));console.log(report.passed+'/'+results.length+' checks passed');if(report.failed)process.exitCode=1;
