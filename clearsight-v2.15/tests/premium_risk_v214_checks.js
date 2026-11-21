/* Focused V2.14 presentation checks against actual integrated sources.
 * The harness substitutes the DOM shell. It does not validate browser layout. */
'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),vm=require('node:vm');
const {createRuntime,root}=require('./runtime_harness');
const results=[],j=JSON.stringify;
function runtime(){const h=createRuntime();if(!h.context.CLEAR_SIGHT_PREMIUM_RISK)vm.runInContext(fs.readFileSync(path.join(root,'assets/premium-risk.js'),'utf8'),h.context,{filename:'premium-risk.js'});h.run('wf29Init();');return h;}
function actor(h,id){h.run(`S.governance.actorId=${j(id)}`);}
function test(name,fn){try{fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL',name,error.stack);}}
function prepare(h){actor(h,'USR-01');h.run("wf29StartRiskEdit('RSK-04');wf29StartControl('CTL-03');wf29StartTest('CTL-03');");}
function unchanged(h){const main=h.elements.get('#main-content'),html=main.innerHTML,state=h.run('recStable(S)'),route=h.run('current.name+"?"+current.params.toString()'),storage=j([...h.storage]);h.run('pr214Refresh()');assert.equal(main.innerHTML,html,'Source markup changed');assert.equal(h.run('recStable(S)'),state,'Source records changed');assert.equal(h.run('current.name+"?"+current.params.toString()'),route,'Route context changed');assert.equal(j([...h.storage]),storage,'Stored state changed');return html;}
const routes=[
 ['risks','USR-03','class="heatmap"'],
 ['risk?id=RSK-04','USR-03','j-dataflow'],
 ['risk-edit?id=RSK-04','USR-01','data-wf29-draft="risk:RSK-04"'],
 ['rcsa','USR-03','cycle-card'],
 ['rcsa-review?id=RCSA-02&risk=RSK-04','USR-01','data-submit="rcsa-submit"'],
 ['indicators','USR-03','Key indicators'],
 ['indicator?id=KRI-01','USR-03','Observation history'],
 ['losses','USR-03','data-l29-form="filters"'],
 ['controls','USR-03','Implemented controls'],
 ['control?id=CTL-03','USR-03','How the control operates'],
 ['control-editor?id=CTL-03','USR-01','data-wf29-draft="control:CTL-03"'],
 ['control-test?id=CTL-03','USR-01','data-wf29-draft="control-test:CTL-03"'],
 ['kri-returns','USR-03','Reporting population'],
 ['kri-library','USR-03','Reporting definitions'],
 ['vendors','USR-03','Service engagements'],
 ['risk?id=RSK-04&tab=assurance','USR-03','Control implementation'],
 ['rcsa-review?id=RCSA-02&risk=RSK-04&mode=review','USR-03','No submitted assessment'],
 ['vendors?tab=onboarding','USR-03','vo-onboarding-steps']
];

test('All 15 operational routes and 18 variants retain source markup, state and query context',h=>{
 prepare(h);for(const [route,id,signature]of routes){actor(h,id);const html=h.navigate(route);assert(html.includes(signature),route+' did not open its source view');assert.equal(h.elements.get('#main-content').dataset.pr214,route.split('?')[0],route);unchanged(h);}
 assert.equal(new Set(routes.map(r=>r[0].split('?')[0])).size,15);
});

test('Route changes reset the marker and preserve existing capture and review presentation families',h=>{
 prepare(h);actor(h,'USR-03');h.navigate('risks');
 for(const [route,family]of [['loss?id=LOS-201','loss'],['vendor?id=ENG-01','vendor'],['document?id=DOC-SAMPLE-POLICY','document'],['kri-return?id=KRC-BRANCH-2026-10%3A%3AB-IK','kri'],['my-workspace',null],['reports',null]]){
  h.navigate(route);assert(!('pr214'in h.elements.get('#main-content').dataset),route);if(family)assert.equal(h.elements.get('#main-content').dataset.pf213,family,route);
 }
 actor(h,'USR-01');h.navigate('control-test?id=CTL-03');assert.equal(h.elements.get('#main-content').dataset.pr214,'control-test');assert.equal(h.elements.get('#main-content').dataset.pf213,'review');
});

test('Risk, RCSA and control forms keep native draft keys, fields and registered commands',h=>{
 prepare(h);actor(h,'USR-01');const before=h.run("recStable({risk:recRead('risk','RSK-04'),control:recRead('control','CTL-03'),drafts:wf29State().drafts})");
 for(const [route,marker,field]of [
  ['risk-edit?id=RSK-04','data-wf29-draft="risk:RSK-04"','data-wf29-field="cause"'],
  ['control-editor?id=CTL-03','data-wf29-draft="control:CTL-03"','data-wf29-field="objective"'],
  ['control-test?id=CTL-03','data-wf29-draft="control-test:CTL-03"','data-wf29-field="reviewer"'],
  ['rcsa-review?id=RCSA-02&risk=RSK-04','data-wf29-rcsa="true"','name="rationale"']
 ]){const html=h.navigate(route);assert(html.includes(marker),route);assert(html.includes(field),route);for(const [,action]of html.matchAll(/data-action="([^"]+)"/g))assert.equal(h.run(`typeof actions[${j(action)}]`),'function',action);unchanged(h);}
 assert.equal(h.run("recStable({risk:recRead('risk','RSK-04'),control:recRead('control','CTL-03'),drafts:wf29State().drafts})"),before);
});

test('Heatmap bands, dated observation scales and accepted loss summaries retain exact source values',h=>{
 actor(h,'USR-03');const html=h.navigate('risks'),cells=[...html.matchAll(/<button class="heat-cell ([^"]*)"[^>]*data-id="(\d),(\d)"[^>]*aria-label="([^"]*)">([\s\S]*?)<\/button>/g)];assert.equal(cells.length,25);
 const risks=JSON.parse(h.run('JSON.stringify(scopeItems("risks"))'));
 for(const [,level,l,i,label,body]of cells){const score=Number(l)*Number(i),count=risks.filter(r=>r.residual[0]===Number(l)&&r.residual[1]===Number(i)).length;assert.equal(level,score>9?'high':score>4?'mid':'');assert.equal(label,`Likelihood ${l}, impact ${i}, ${count} risks`);assert(body.includes(`<small>${score}</small>`));if(count)assert(body.includes(`<span>${count}</span>`));}
 unchanged(h);assert(html.includes('Likelihood · low → high'));assert(html.includes('Impact · low → high'));
 // A dated test fixture is explicit; undated legacy samples remain excluded.
 h.run("var prFixture=S.indicators.find(x=>x.id==='KRI-01');prFixture.observations=[{date:'2026-10-01',value:2,sourceRef:'Fixture observation one'},{date:'2026-11-01',value:4,sourceRef:'Fixture observation two'}];");
 const indicator=h.navigate('indicator?id=KRI-01'),expected=JSON.parse(h.run('JSON.stringify(prFixture.observations.map(o=>Math.max(2,o.value/Math.max(prFixture.threshold||0,4,1)*110)))'));
 assert.deepEqual([...indicator.matchAll(/<i style="height:([\d.]+)px"><\/i>/g)].map(m=>Number(m[1])),expected);assert(indicator.includes('Fixture observation one'));assert(indicator.includes('Fixture observation two'));unchanged(h);
 const losses=h.navigate('losses');assert(losses.includes('Accepted gross loss'));assert(losses.includes('Verified recoveries'));assert(losses.includes('Unrecovered loss'));assert(losses.includes('data-l29-form="filters"'));unchanged(h);
});

test('Unauthorized, out-of-scope and unavailable records retain the canonical blocked states',h=>{
 actor(h,'USR-01');const denied=h.navigate('control-editor?id=CTL-01');assert(denied.includes('Access is unavailable for this location'));assert(!denied.includes('data-wf29-draft='));unchanged(h);
 actor(h,'USR-03');h.run("S.scope={branch:'B-EN',department:'',unit:''};S.scopeInvalid=false;");const scoped=h.navigate('risk?id=RSK-04');assert(scoped.includes('Outside the selected scope'));assert(!scoped.includes('Incorrect payment authorization'));unchanged(h);
 h.run("S.scope={branch:'',department:'',unit:''};S.scopeInvalid=false;");const missing=h.navigate('indicator?id=NOT-A-RECORD');assert(missing.includes('source record is unavailable'));assert(!missing.includes('data-action="observe-indicator"'));unchanged(h);
});

test('Accessible region names add keyboard entry without replacing existing roles or labels',h=>{
 function node(attrs={}){return {attrs:{...attrs},hasAttribute(k){return Object.hasOwn(this.attrs,k)},setAttribute(k,v){this.attrs[k]=v}};}
 actor(h,'USR-03');h.navigate('risks');const main=h.elements.get('#main-content'),query=main.querySelectorAll,toolbar=node(),actions=node({'aria-label':'Existing source actions',role:'group'}),matrix=node(),comparison=node({'aria-labelledby':'original-compare-heading'}),history=node({role:'img','aria-label':'Dated indicator values',tabindex:'-1'});
 main.querySelectorAll=s=>s==='.toolbar,form.l29-filters'?[toolbar]:s==='.wf29-actions'?[actions]:s==='.heatmap-layout'?[matrix]:s==='.op211-comparison'?[comparison]:s==='.k28-history-chart'?[history]:[];
 try{unchanged(h);assert.deepEqual(toolbar.attrs,{role:'group','aria-label':'Risk register filters'});assert.equal(actions.attrs['aria-label'],'Existing source actions');assert.deepEqual(matrix.attrs,{role:'region','aria-label':'Residual risk matrix: likelihood by impact',tabindex:'0'});assert.equal(comparison.attrs['aria-labelledby'],'original-compare-heading');assert(!('aria-label'in comparison.attrs));assert.equal(comparison.attrs.tabindex,'0');assert.deepEqual(history.attrs,{role:'img','aria-label':'Dated indicator values',tabindex:'-1'});}finally{main.querySelectorAll=query;}
});

test('Presentation API and stylesheet stay scoped, with normal-flow actions and touch-size rules',h=>{
 const api=h.context.CLEAR_SIGHT_PREMIUM_RISK;assert.deepEqual(Object.keys(api).sort(),['presentationOnly','refresh','routes','version']);assert.equal(api.version,'2.14');assert(api.presentationOnly);assert.equal(api.routes.length,15);assert(Object.isFrozen(api));assert(Object.isFrozen(api.routes));
 const css=fs.readFileSync(path.join(root,'assets/premium-risk.css'),'utf8');for(const line of css.split('\n').filter(l=>l.includes('{')&&!/^\s*@/.test(l)))assert(line.includes('#main-content[data-pr214'),line);
 assert(!/@import|url\s*\(/i.test(css));assert(!/position\s*:\s*(?:sticky|fixed)/i.test(css));assert(css.includes('@media(min-width:1024px) and (pointer:fine)'));assert(css.includes('@media(max-width:767px)'));assert(css.includes('@media(pointer:coarse)'));assert(css.includes('font-size:16px;min-height:44px'));
 assert(css.includes('.op211-comparison .table{display:table;min-width:620px'));const barStyle=css.match(/\.k28-history-chart i\{([^}]+)\}/)?.[1];assert(barStyle);assert(!/(?:^|;)\s*height\s*:/.test(barStyle));
});

test('Live document labels are consistent while canonical evidence keys and source text remain intact',h=>{
 actor(h,'USR-01');const rcsa=h.navigate('rcsa-review?id=RCSA-02&risk=RSK-04');assert(rcsa.includes('Supporting document'));assert(rcsa.includes('Choose a document'));assert(rcsa.includes('No current documents map to this implementation.'));assert(rcsa.includes('name="evidence"'));assert(/href="#\/evidence"[^>]*>Documents<\/a>/.test(rcsa));assert(!rcsa.includes('Evidence workspace'));
 actor(h,'USR-03');const question=h.run("recStable(findQ('Q02'))");h.run("inspectQuestion('Q02')");const inspector=h.elements.get('#dialog').innerHTML;assert(inspector.includes('Default document requirement'));assert(inspector.includes('<strong>Required</strong>'));assert.equal(h.run("recStable(findQ('Q02'))"),question);
 for(const route of ['rcsa-review?id=RCSA-02&risk=RSK-03','rcsa-review?id=RCSA-03&risk=RSK-06']){const locked=h.navigate(route);assert(locked.includes('<span>Supporting document</span>'));assert(!locked.includes('<span>Evidence</span>'));}
 const builder=h.navigate('form-builder?form=FORM-VENDOR');assert(builder.includes('Document reference'));assert.equal(h.run('FA_TYPES.evidence'),'Document reference');assert.equal(h.run('CX_TYPES.evidence'),'Document reference');assert.equal(h.run("faForm().fields.find(f=>f.id==='report').type"),'evidence');assert(builder.includes('Independent assurance report reference'),'Published source question wording must remain intact');assert(builder.includes('Attach or reference supporting evidence'),'Contextual explanation remains intact');
 const loss=h.run("recStable(S.losses.find(l=>l.id==='LOS-201'))"),source=h.run("l29Summary({...S.losses.find(l=>l.id==='LOS-201'),evidenceVersionId:'VERSION-UNAVAILABLE'})");assert(source.includes('<b>Source document</b>'));assert(source.includes('Document version VERSION-UNAVAILABLE is unavailable to the current reviewer.'));assert.equal(h.run("recStable(S.losses.find(l=>l.id==='LOS-201'))"),loss);
});

const report={suite:'V2.14 premium operational risk presentation',runtime:'Actual integrated application sources with a substituted DOM shell',browserLayoutValidated:false,assignedRoutes:15,routeVariants:routes.length,results,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length};
fs.writeFileSync(path.join(root,'tests/premium-risk-v214-results.json'),JSON.stringify(report,null,2)+'\n');console.log(`${report.passed}/${results.length} premium risk checks passed; ${routes.length} route variants`);if(report.failed)process.exitCode=1;
