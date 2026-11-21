/* Actual-source capture, import and navigation checks. The browser shell is
   substituted by runtime_harness; this does not assert rendered visual quality. */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function test(name,fn){try{fn(createRuntime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.message});console.error('FAIL',name,error.stack);}}
function rawCsv(h,rows){const headers=h.run('CS_LOSS_REFERENCE.columns.map(c=>c[0])');return [headers,...rows].map(r=>r.map(v=>'"'+String(v??'').replace(/"/g,'""')+'"').join(',')).join('\r\n');}
const sourceRow=(overrides={})=>{const row=['CASH SHORTAGES','CASH SHORTAGES','Oct','10','Q4','100.50','Synthetic till difference, morning reconciliation','Ikeja','Lagos','South West','2026','4','','10/10/2026','11/10/2026','Cash count was reconciled to a recorded process exception.','Full recovery of the amount','Naira'];for(const [i,v] of Object.entries(overrides))row[Number(i)]=v;return row;};
function stage(h,rows){h.context.inputCsv=rawCsv(h,rows);h.run('var batch=l29StageImport(inputCsv);');return h.run('batch.id');}
function fillDraft(h){h.run(`var capture=l29NewDraft('RSK-04');Object.assign(capture.data,{title:'Synthetic payment reconciliation loss',accountName:'CASH SHORTAGES',lossGroup:'CASH SHORTAGES',transactionParticulars:'A synthetic payment reconciliation difference was booked after investigation.',transactionReference:'CAPTURE-001',region:'Lagos',directorate:'South West',gross:'120.75',date:'2026-11-10',recognizedOn:'2026-11-11',reportingPeriod:'2026-11',rootCauseStatus:'Confirmed',rootCause:'A reconciliation procedure exception produced the confirmed difference.',actionPlan:'',noActionReason:'The corrected entry and procedure were verified and no further action is required.',sourceRef:'JOURNAL-CAPTURE-001'});`);}

test('Source mapping captures all 18 fields without merging accounting dimensions',h=>{
 assert.equal(h.run('CS_LOSS_REFERENCE.columns.length'),18);assert.equal(h.run('new Set(CS_LOSS_REFERENCE.columns.map(c=>c[0])).size'),18);
 assert.equal(h.run('CS_LOSS_REFERENCE.columns.find(c=>c[0]==="ACCT_NAME")[1]'),'accountName');assert.equal(h.run('CS_LOSS_REFERENCE.columns.find(c=>c[0]==="LOSS GROUP")[1]'),'lossGroup');assert.equal(h.run('CS_LOSS_REFERENCE.columns.find(c=>c[0]==="DATE OF RECOGNITION")[1]'),'recognizedOn');
 assert.equal(h.run('scopeTotals().gross'),3650000000);assert.equal(h.run('scopeTotals().recovered'),1200000000);assert(h.run('S.losses.every(l=>!l.recognizedOn&&!l.reportingPeriod)'),'Legacy dates must not be invented');
});
test('Money precision, calendar dates and source serials are validated',h=>{
 assert.equal(h.run('l29Minor("8500000.99")'),850000099);for(const bad of ['1.234','-3.00','1e6','1,200','Infinity']){h.context.bad=bad;assert.throws(()=>h.run('l29Minor(bad)'));}
 assert.equal(h.run('l29SourceDate("10/11/2026")'),'2026-11-10');assert.equal(h.run('l29SourceDate("45971")'),'2025-11-10');assert.throws(()=>h.run('l29SourceDate("31/02/2026")'));assert.equal(h.run('l29SourceDate("2026-11-13")'),'2026-11-13');
});
test('Risk mapping is optional; acceptance controls inclusion and integrity checks',h=>{
 fillDraft(h);h.run('capture.data.risk="";var loss=l29SubmitDraft(capture.id);');assert.equal(h.run('loss.risk'),'');assert(!h.run('lossFinanciallyIncluded(loss)'));assert(!h.run('recIntegrity().problems.some(p=>p.code==="ORPHAN_LOSS"&&p.ref===loss.id)'));
 assert.throws(()=>h.run(`S.governance.actorId='USR-03';l29Review(loss.id,'Accept','The independent review confirms the recorded amount.','obsolete-submission');`),/changed|current/);
 h.run(`l29Review(loss.id,'Accept','The independent review confirms the recorded amount.',loss.submission.id);`);assert(h.run('lossFinanciallyIncluded(loss)'));assert.equal(h.run('loss.gross'),12075);
});
test('Period discrepancy, near miss and investigation exceptions require explicit handling',h=>{
 fillDraft(h);h.run(`capture.data.reportingPeriod='2026-10';`);assert(h.run('l29Issues(capture.data,{exclude:capture.id}).some(i=>i.field==="periodReason")'));h.run(`capture.data.periodReason='The adjustment belongs to the prior reporting close.';capture.data.type='Near miss';capture.data.gross='0';`);assert(h.run('l29Issues(capture.data,{exclude:capture.id}).some(i=>i.field==="recognizedOn")'));
 h.run(`capture.data.recognizedOn='';capture.data.rootCauseStatus='Under investigation';capture.data.actionPlan='';`);assert(h.run('l29Issues(capture.data,{exclude:capture.id}).some(i=>i.field==="actionPlan")'));
});
test('Saved drafts retain step values and reject edits by another preparer',h=>{
 fillDraft(h);h.run(`l29SaveDraft(capture.id,{transactionReference:'CAPTURE-SAVED',rootCause:'A saved root-cause narrative is retained while navigating.'},3);`);assert.equal(h.run('capture.step'),3);assert.equal(h.run('capture.data.transactionReference'),'CAPTURE-SAVED');h.run(`S.governance.actorId='USR-03';`);assert.throws(()=>h.run(`l29SaveDraft(capture.id,{gross:'999.00'});`),/preparer/);assert.equal(h.run('capture.data.gross'),'120.75');
});
test('CSV accepts quoted commas and exact source headers; malformed input stays out',h=>{
 stage(h,[sourceRow()]);assert.equal(h.run('batch.rows.length'),1);assert.equal(h.run('batch.rows[0].data.transactionParticulars'),'Synthetic till difference, morning reconciliation');assert.equal(h.run('batch.rows[0].data.reportingPeriod'),'2026-10');assert.equal(h.run('batch.rows[0].data.currency'),'NGN');assert.equal(h.run('batch.rows[0].data.recognizedOn'),'2026-10-11');assert.throws(()=>h.run('l29CsvRead("ACCT_NAME,AMOUNT\\nX,5")'),/Missing/);assert.throws(()=>h.run('l29CsvRead(inputCsv+"\\n\\\"unfinished")'),/unfinished/);
});
test('Repeated transactions across months are quarantined and inconsistent quarter data is visible',h=>{
 stage(h,[sourceRow(),sourceRow({2:'Nov',3:'11',11:'1'})]);assert.equal(h.run('batch.rows[0].status'),'Ready for draft');assert.equal(h.run('batch.rows[1].status'),'Duplicate candidate');assert(h.run('batch.rows[1].warnings.some(w=>w.includes("QTR NUMBER"))'));assert(h.run('batch.rows[1].warnings.some(w=>w.includes("recognition"))'));
 assert.throws(()=>h.run('l29CommitImport(batch.id,[2,3])'),/duplicate|mapping|errors/);assert.equal(h.run('S.losses.length'),5);assert.equal(h.run('Object.keys(S.lossRegister.drafts).length'),0);
});
test('Distinct import events require unique references and a retained explanation',h=>{
 stage(h,[sourceRow(),sourceRow({2:'Nov',3:'11'})]);h.run(`l29ResolveImportDuplicate(batch.id,3,'DISTINCT-SOURCE-003','This is a separately reconciled transaction on another journal line.');`);assert.equal(h.run('batch.rows[1].status'),'Ready for draft');h.run('var created=l29CommitImport(batch.id,[2,3]);');assert.equal(h.run('created.length'),2);assert.equal(h.run('S.losses.length'),5);assert.equal(h.run('Object.values(S.lossRegister.drafts).filter(d=>d.importId===batch.id).length'),2);assert.equal(h.run('S.lossRegister.drafts[created[1]].data.reportingPeriod'),'2026-11');
});
test('Unknown branches require explicit authorized mapping; imports do not verify recovery claims',h=>{
 stage(h,[sourceRow({7:'Source branch awaiting mapping',12:'12/10/2026'})]);assert.equal(h.run('batch.rows[0].status'),'Needs correction');assert.throws(()=>h.run('l29CommitImport(batch.id,[2])'));
 h.run(`l29MapImport(batch.id,{'Source branch awaiting mapping':'B-EN'});`);assert.equal(h.run('batch.rows[0].status'),'Needs correction');h.run(`l29MapImport(batch.id,{'Source branch awaiting mapping':'B-IK'});var created=l29CommitImport(batch.id,[2]);var imported=S.lossRegister.drafts[created[0]];`);assert.equal(h.run('imported.data.orgScopeId'),'B-IK');assert(h.run('imported.data.recoveryNote.includes("Full recovery")'));assert.equal(h.run('S.losses.length'),5);assert.equal(h.run('scopeTotals().recovered'),1200000000);assert.equal(h.run('imported.sourceColumns.BRANCH'),'Source branch awaiting mapping');
});
test('Imported provenance survives acceptance while narrative recovery stays unverified',h=>{
 stage(h,[sourceRow()]);h.run(`var created=l29CommitImport(batch.id,[2]);var imported=S.lossRegister.drafts[created[0]];Object.assign(imported.data,{noActionReason:'The error was corrected and the revised process was independently tested.',transactionReference:'IMPORT-ACCEPT-001'});var loss=l29SubmitDraft(imported.id);S.governance.actorId='USR-03';l29Review(loss.id,'Accept','The original source row and accounting amount were independently checked.');`);
 assert.equal(h.run('loss.importProvenance.row'),2);assert.equal(h.run('loss.importProvenance.sourceColumns["RECOVERY DATA"]'),'Full recovery of the amount');assert.equal(h.run('lossVerifiedRecoveries(loss)'),0);assert.equal(h.run('loss.gross'),10050);assert.equal(h.run('loss.recoveries.length'),0);
});
test('Source errors and unsupported currencies never create financial records',h=>{
 stage(h,[sourceRow({5:'100.999',17:'Unmapped currency',13:'31/02/2026'})]);assert.equal(h.run('batch.rows[0].status'),'Needs correction');assert(h.run('batch.rows[0].errors.length>=3'));assert.throws(()=>h.run('l29CommitImport(batch.id,[2])'));assert.equal(h.run('S.losses.length'),5);
});
test('Loss routes show useful locations and register every rendered action',h=>{
 fillDraft(h);const routes=['losses','loss?id=LOS-201','loss-import','loss-intake?id='+h.run('capture.id'),'loss-insights'];for(const route of routes){const html=h.navigate(route);assert(!html.includes('[object Object]'),route);assert(!html.includes('NaN'),route);for(const match of html.matchAll(/data-action="([^"]+)"/g)){h.context.actionName=match[1];assert(h.run('typeof actions[actionName]==="function"'),route+' missing action '+match[1]);}}
 assert(h.navigate('loss?id=LOS-201').includes('Ikeja / Operations / Payments'));assert(!h.navigate('loss?id=invalid-event').includes('Payment settlement error'));
});
const summary={suite:'Loss capture and workbook import V2.9',browserLayoutValidated:false,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length,results};fs.writeFileSync(path.join(root,'tests/loss-v29-results.json'),JSON.stringify(summary,null,2)+'\n');console.log(JSON.stringify({passed:summary.passed,failed:summary.failed}));if(summary.failed)process.exitCode=1;
