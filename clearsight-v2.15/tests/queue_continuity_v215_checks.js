/* Actual-source V2.15 queue checks. Controlled DOM shell; no browser acceptance claim. */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const cases=[],results=[];
function test(name,fn){cases.push({name,fn});}
function runtime(){const h=createRuntime();h.run(`render=()=>{};current={name:'inbox',params:new URLSearchParams()};S.governance.actorId='USR-01';inboxInit();nItems();`);return h;}

test('Filtered empty inbox explains hidden results and escapes active search context',h=>{
 const before=h.run('JSON.stringify(S.tasks)');
 h.run(`N_FILTER.search='<script>missing</script>';N_FILTER.severity='Critical';giInboxPrefs().due='overdue';`);
 const html=h.run('inboxView()');
 assert(html.includes('No notifications match these filters'));
 assert(!html.includes('You’re up to date'));
 assert(html.includes('&lt;script&gt;missing&lt;/script&gt;'));
 assert(html.includes('Deadline:</span> Overdue action'));
 assert(html.includes('0 matching notifications · All notifications'));
 assert.equal(h.run('JSON.stringify(S.tasks)'),before);
});

test('Clear refinements retains mailbox, sorting, recipient, scope and receipts',h=>{
 h.run(`N_FILTER={view:'archive',category:'deadline',search:'missing',severity:'High',source:'Work'};giInboxPrefs().sort='newest';giInboxPrefs().due='overdue';`);
 const before=h.run(`JSON.stringify({scope:S.scope,actor:nActor(),receipts:S.notificationsV28.events.map(x=>[x.id,x.receipts]),tasks:S.tasks})`);
 h.run(`actions['n-reset-refinements']();`);
 assert.equal(h.run('N_FILTER.view'),'archive');
 assert.equal(h.run('nRefinements().length'),0);
 assert.equal(h.run('giInboxPrefs().sort'),'newest');
 assert.equal(h.run(`JSON.stringify({scope:S.scope,actor:nActor(),receipts:S.notificationsV28.events.map(x=>[x.id,x.receipts]),tasks:S.tasks})`),before);
 h.run(`N_FILTER.view='snoozed';N_FILTER.search='missing';actions['n-reset-refinements']();`);
 assert.equal(h.run('N_FILTER.view'),'snoozed');
 assert(h.run('nList([])').includes('No snoozed notifications'));
 h.run(`N_FILTER.view='unread';`);
 assert(h.run('nList([])').includes('Read notifications can still need a decision'));
});

test('Partial and empty selections expose correct native select-all state and prune stale IDs',h=>{
 h.run(`var qiRows=nFiltered().slice(0,2);N_SELECTED.add(qiRows[0].id);N_SELECTED.add('out-of-scope-id');`);
 assert.equal(h.run('qiRows.length'),2);
 const html=h.run('nBulkBar(qiRows)');
 assert(html.includes('aria-checked="mixed"'));
 assert(html.includes('data-n-mixed="true"'));
 assert.equal(h.run('N_SELECTED.size'),1);
 const input={dataset:{nMixed:'true'},indeterminate:false};h.elements.set('[data-n-all]',input);
 h.run('nSyncSelection()');assert.equal(input.indeterminate,true);
 h.run('N_SELECTED.add(qiRows[1].id)');assert(h.run('nBulkBar(qiRows)').includes('aria-checked="true"'));
 input.dataset.nMixed='false';h.run('nSyncSelection()');assert.equal(input.indeterminate,false);
 const empty=h.run('nBulkBar([])');assert(empty.includes('disabled aria-checked="false"'));assert.equal(h.run('N_SELECTED.size'),0);
});

test('Inbox refresh filters once and announces concise counts without announcing the entire list',h=>{
 for(const selector of ['#n-list','#n-bulk','#n-refinements','#n-result-status'])h.elements.set(selector,{innerHTML:'',textContent:''});
 h.run(`var qiFilterCalls=0,qiPrior=nFiltered;nFiltered=function(){qiFilterCalls++;return qiPrior();};N_FILTER.search='   ';nRefreshList();`);
 assert.equal(h.run('qiFilterCalls'),1);
 assert(h.elements.get('#n-result-status').textContent.includes('notifications · All notifications'));
 assert.equal(h.run('nRefinements().length'),0);
 const html=h.run('inboxView()');
 assert(html.includes('id="n-result-status" class="n-result-status" role="status" aria-live="polite" aria-atomic="true"'));
 assert(!/id="n-list"[^>]*aria-live/.test(html));
 assert(h.elements.get('#n-list').innerHTML.includes('data-notification-id'));
});

test('Work detail round-trips exact queue filters through bounded same-workspace parameters',h=>{
 h.run(`S.tasks.push({id:'QI-ACTION',title:'QI scoped task',owner:'Amina Bello',ownerId:'USR-01',orgScopeId:'U-IK-PAY',due:'2026-11-01',status:'Open',kind:'Remediation',outcome:'Review the operational outcome.',ref:'',route:'work'});current={name:'work',params:new URLSearchParams({owner:'all',lane:'overdue',kind:'Remediation',q:'QI & "task" #42'})};`);
 const before=h.run('JSON.stringify(S.tasks)'),route=h.run(`giWorkItemRoute('QI-ACTION')`),params=new URLSearchParams(route.split('?')[1]);
 assert.equal(params.get('wsearch'),'QI & "task" #42');
 h.context.qiRoute=route;h.run(`current={name:'work-item',params:new URLSearchParams(qiRoute.split('?')[1])};`);
 const back=h.run('giWorkBackRoute()'),backParams=new URLSearchParams(back.split('?')[1]);
 assert.equal(back.split('?')[0],'work');
 assert.equal(backParams.get('owner'),'all');assert.equal(backParams.get('lane'),'overdue');assert.equal(backParams.get('kind'),'Remediation');assert.equal(backParams.get('q'),'QI & "task" #42');
 assert(h.run('giWorkItemView()').includes('Back to work'));
 assert(h.run('giWorkItemView()').includes('href="#/'+back+'"'));
 assert.equal(h.run('JSON.stringify(S.tasks)'),before);
});

test('Malformed queue return data cannot change route, expand scope or expose unavailable actions',h=>{
 h.run(`current={name:'work-item',params:new URLSearchParams({id:'missing',wowner:'https://example.invalid',wlane:'javascript:alert(1)',wkind:'x'.repeat(500),wsearch:'y'.repeat(800),return:'https://example.invalid',scope:'B-EN'})};`);
 const before=h.run('JSON.stringify(S.scope)'),back=h.run('giWorkBackRoute()'),p=new URLSearchParams(back.split('?')[1]);
 assert.equal(back.split('?')[0],'work');assert.equal(p.get('owner'),'me');assert.equal(p.get('lane'),'open');assert.equal(p.get('kind').length,120);assert.equal(p.get('q').length,200);assert(!p.has('return'));assert(!p.has('scope'));
 assert(h.run('giWorkItemView()').includes('unavailable'));assert.equal(h.run('JSON.stringify(S.scope)'),before);
 h.run(`current={name:'work-item',params:new URLSearchParams('id=ACT-03')};`);assert.equal(h.run('giWorkBackRoute()'),'work');
});

for(const {name,fn}of cases){try{fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.stack||String(error)});console.error('FAIL',name,error.stack||error);}}
const report={suite:'V2.15 inbox refinement and work queue continuity',browserLayoutValidated:false,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,results};
fs.writeFileSync(path.join(root,'tests/queue-continuity-v215-results.json'),JSON.stringify(report,null,2)+'\n');
console.log(`${report.passed}/${cases.length} queue checks passed`);if(report.failed)process.exitCode=1;
