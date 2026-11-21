/* Actual-source V2.13 presentation checks. DOM shell only; no layout claim. */
'use strict';
const assert=require('assert'),fs=require('fs'),vm=require('vm');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function runtime(){const h=createRuntime();h.run(`S.governance.actorId='USR-01';S.scope={branch:'',department:'',unit:''};`);return h;}
function fixture(h){h.run(`S.tasks.push({id:'PW213-TASK',title:'PW213 accountable action',owner:'Amina Bello',ownerId:'USR-01',orgScopeId:'U-IK-PAY',due:'2026-11-01',status:'Open',kind:'Remediation',outcome:'Confirm the source finding and record the accountable recovery outcome.',source:'task',ref:'',route:'work'});nPut({id:'PW213-NOTICE',type:'deadline',title:'PW213 accountable source',body:'Inspect the exact action before recording completion.',sourceType:'task',sourceId:'PW213-TASK',sourceRoute:'work',recipientIds:['USR-01'],orgScopeId:'U-IK-PAY',actor:'USR-03',due:'2026-11-01',createdAt:'2026-11-01T10:00:00Z',actionable:true,severity:'High'});`);}
function targets(html){return [...html.matchAll(/(?:href|data-action|data-id|data-n-select|data-n-form|data-gi-filter|data-gv-form|data-doc-actor|data-n-actor)="[^"]*"/g)].map(m=>m[0]);}
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL',name,error.stack);}}

test('All thirteen decorated routes retain their actual source/action targets and registered commands',()=>{
 const h=runtime();fixture(h);
 const views=[['my-workspace','','re212WorkspaceView'],['my-access','','re212AccessView'],['work','owner=all&lane=open','giWorkView'],['work-item','id=PW213-TASK','giWorkItemView'],['inbox','','inboxView'],['notification','id=PW213-NOTICE','nDetailView'],['notification-preferences','','nPreferencesView'],['roles','','gvRolesView'],['role-detail','id=ROLE-BOARD','gvRoleDetail'],['role-change-review','','giRoleReviewView'],['access-review','state=all','giAccessView'],['escalation-matrix','','gvMatrixView'],['escalation-policy','id=ESC-KRI','gvPolicyDetail']];
 assert.equal(h.run('CLEAR_SIGHT_PREMIUM_WORKSPACES.routes.length'),views.length);
 for(const [route,query,fn]of views){
  h.run(`current={name:${JSON.stringify(route)},params:new URLSearchParams(${JSON.stringify(query)})};`);
  if(route==='role-change-review')h.run(`gvState().draftRole.description='A specific revised reporting responsibility.';gvState().draftRole.reason='Clarify the documented observer responsibility after the access review.';`);
  const before=h.run(fn+'()'), html=h.run('window.CS_JOURNEYS[current.name]()');
  assert(html.includes('data-pw-route="'+route+'"'),route);assert.equal((html.match(/pw213-workspace/g)||[]).length,1,route);assert(!html.includes('<main'),route);
  assert.deepEqual(targets(html),targets(before),route+' target preservation');
  for(const [,action]of html.matchAll(/data-action="([^"]+)"/g))assert(h.run(`typeof actions[${JSON.stringify(action)}]==='function'`),action);
  assert(!/undefined|NaN|\[object Object\]/.test(html),route);
  if(route==='escalation-matrix'){
   const table=html.match(/<table[^>]*pw213-policies-table[\s\S]*?<\/table>/)[0];
   const policyRows=[...table.matchAll(/<tbody>([\s\S]*?)<\/tbody>/g)].flatMap(m=>[...m[1].matchAll(/<tr>([\s\S]*?)<\/tr>/g)]);assert(policyRows.length);
   for(const [,cells]of policyRows)assert.deepEqual([...cells.matchAll(/data-label="([^"]+)"/g)].map(m=>m[1]),['Policy / signal','Scope','Escalation levels','Clock / stop','Action']);
   assert.equal((table.match(/<th scope="col">/g)||[]).length,5);
  }
 }
});
test('Work filters retain owner, query and kind while the selected queue is announced',()=>{
 const h=runtime();fixture(h);h.run(`current={name:'work',params:new URLSearchParams('owner=all&lane=overdue&q=PW213&kind=Remediation')};`);
 const html=h.run('window.CS_JOURNEYS.work()'),nav=html.match(/<nav class="gi-lanes"[\s\S]*?<\/nav>/)[0];
 assert.equal((nav.match(/aria-current="page"/g)||[]).length,1);assert(nav.match(/href="[^"]*lane=overdue[^"]*"[^>]*aria-current="page"/));
 for(const [,href]of nav.matchAll(/href="#\/([^"]+)"/g)){const p=new URLSearchParams(href.split('?')[1]);assert.equal(p.get('owner'),'all');assert.equal(p.get('q'),'PW213');assert.equal(p.get('kind'),'Remediation');}
 assert(html.includes('href="#/work-item?id=PW213-TASK"'));assert(html.includes('aria-live="polite" aria-atomic="true"'));
 assert(html.includes('<span class="sr-only">Next action</span>'));
});
test('Primary queue tables have exact heading references without changing their rows',()=>{
 const h=runtime();fixture(h);h.run(`current={name:'work',params:new URLSearchParams('owner=all&q=PW213')};`);
 const old=h.run('giWorkView()'),html=h.run('window.CS_JOURNEYS.work()');
 const label=html.match(/<table[^>]*aria-labelledby="([^"]+)"/)[1];assert(html.includes('<h2 id="'+label+'">Accountable work</h2>'));
 assert.deepEqual(html.match(/<tbody>[\s\S]*?<\/tbody>/g),old.match(/<tbody>[\s\S]*?<\/tbody>/g));
});
test('An empty personal queue cannot label a subsequent owned-program table',()=>{
 const h=runtime();h.run(`current={name:'my-workspace',params:new URLSearchParams()};re212WorkspaceRows=()=>[];`);
 const html=h.run('window.CS_JOURNEYS["my-workspace"]()');
 assert(html.includes('No open actions assigned to you'));assert(html.includes('Programs you are accountable for'));assert(html.includes('<table'));
 assert(!html.includes('aria-labelledby="pw213-my-workspace-table-0"'));
});
test('Notification accessibility names remain unique and escape source content',()=>{
 const h=runtime();fixture(h);h.run(`var pwEvent=nEvent('PW213-NOTICE');pwEvent.title='Quarterly $& <board> "review"';var pwSecond={...pwEvent,id:'PW213 / second'};`);
 const html=h.run('nRow(pwEvent)+nRow(pwSecond)'), ids=[...html.matchAll(/<strong id="([^"]+)"/g)].map(m=>m[1]);
 assert.equal(new Set(ids).size,2);for(const id of ids)assert(html.includes('aria-labelledby="'+id+'"'));
 assert(html.includes('Quarterly $&amp; &lt;board&gt; &quot;review&quot;'));assert(!html.includes('<board>'));
 assert.equal((html.match(/role="group"/g)||[]).length,2);assert(html.includes('aria-label="Actions for Quarterly $&amp; &lt;board&gt; &quot;review&quot;"'));
});
test('Reading and acknowledging a notification never presents its open source as resolved',()=>{
 const h=runtime();fixture(h);h.run(`var pwEvent=nEvent('PW213-NOTICE');nReceipt(pwEvent).readAt=nNow();nReceipt(pwEvent).acknowledgedAt=nNow();`);
 const before=h.run('JSON.stringify(S.tasks)'),html=h.run('nRow(pwEvent)');
 assert(html.includes('data-pw-source-state="attention"'));assert(html.includes('Acknowledged'));assert(!html.includes('n-unread"'));
 assert.equal(h.run('JSON.stringify(S.tasks)'),before);assert(!h.run('nResolved(pwEvent)'));
 h.run(`S.tasks.find(t=>t.id==='PW213-TASK').status='Completed';`);
 assert(h.run('nRow(pwEvent)').includes('data-pw-source-state="resolved"'));
});
test('The notification next action still opens the exact task and rejects a different recipient',()=>{
 const h=runtime();fixture(h);const html=h.run('nRow(nEvent("PW213-NOTICE"))');
 assert(html.includes('data-action="gi-inbox-source" data-id="PW213-NOTICE"'));assert(html.includes('data-action="n-toggle-read"'));assert(html.includes('data-action="n-archive"'));
 h.run(`giOpenNotificationSource('PW213-NOTICE');`);assert(h.context.location.hash.includes('work-item?id=PW213-TASK'));
 h.run(`S.governance.actorId='USR-04';`);assert.throws(()=>h.run(`giOpenNotificationSource('PW213-NOTICE')`),/unavailable/);
});
test('Presentation does not change effective access, saved policies or current source records',()=>{
 const h=runtime();fixture(h);h.run(`current={name:'roles',params:new URLSearchParams()};gvRolesView();`);
 const before=h.run('JSON.stringify({access:S.accessControl,tasks:S.tasks,governance:S.governance})');h.run('window.CS_JOURNEYS.roles()');
 assert.equal(h.run('JSON.stringify({access:S.accessControl,tasks:S.tasks,governance:S.governance})'),before);
 h.run(`current={name:'escalation-matrix',params:new URLSearchParams()};gvMatrixView();`);const policies=h.run('JSON.stringify(gvState().policies)');
 h.run('window.CS_JOURNEYS["escalation-matrix"]()');assert.equal(h.run('JSON.stringify(gvState().policies)'),policies);
 assert(!h.run(`gvDecision('USR-01','documents.export','U-IK-PAY').allowed`));
});
test('Repeated module loading is inert and never creates a second main landmark',()=>{
 const h=runtime();fixture(h);h.run(`current={name:'work',params:new URLSearchParams('owner=all&q=PW213')};`);
 const before=h.run('window.CS_JOURNEYS.work()');vm.runInContext(fs.readFileSync(root+'/assets/premium-workspaces.js','utf8'),h.context);
 assert.equal(h.run('window.CS_JOURNEYS.work()'),before);
 h.navigate('work?owner=all&q=PW213');const shell=h.elements.get('#app').innerHTML;assert.equal((shell.match(/<main\b/g)||[]).length,1);
});
const report={suite:'V2.13 actual-source premium workspace presentation',browserLayoutValidated:false,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length,results};
fs.writeFileSync(root+'/tests/premium-workspaces-v213-results.json',JSON.stringify(report,null,2));console.log(report.passed+'/'+results.length+' checks passed');if(report.failed)process.exitCode=1;
