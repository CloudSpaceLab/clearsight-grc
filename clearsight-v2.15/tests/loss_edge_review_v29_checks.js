/* Independent actual-source review of V2.9 loss commands and financial state.
 * Uses the shared DOM-shell harness. No browser/layout validation is claimed.
 */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function runtime(){const h=createRuntime();h.run('render=()=>{};values=f=>f.values||f;');return h;}
function test(name,fn){try{fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.message});console.error('FAIL',name,'\n ',error.message);}}
function submitted(h,extra=''){
 h.run(`S.governance.actorId='USR-01';var edgeDraft=l29NewDraft('RSK-04');Object.assign(edgeDraft.data,{
 title:'Independent settlement loss',accountName:'CASH SHORTAGES',lossGroup:'CASH SHORTAGES',
 transactionParticulars:'A payment settlement variance was reconciled in the branch ledger.',
 transactionReference:'EDGE-2026-001',region:'Lagos Mainland',directorate:'Operations',
 gross:'100.00',date:'2026-11-03',recognizedOn:'2026-11-04',reportingPeriod:'2026-11',
 rootCauseStatus:'Confirmed',rootCause:'The payment register was reconciled after a settlement mismatch.',
 actionPlan:'',noActionReason:'The corrected procedure was tested and no further action is required.',
 sourceRef:'EDGE-SOURCE-001'});`);
 if(extra)h.run(extra);
 h.run('var edgeLoss=l29SubmitDraft(edgeDraft.id);');
}
function accepted(h,extra=''){submitted(h,extra);h.run(`S.governance.actorId='USR-03';l29Review(edgeLoss.id,'Accept','The source amount and event classification were independently checked.');S.governance.actorId='USR-01';`);}
function correction(h,extra=''){h.run(`S.governance.actorId='USR-01';var correctionDraft=l29NewDraft('',edgeLoss.id);correctionDraft.data.correctionReason='Correct the recorded amount and preserve the original accepted source facts.';`);if(extra)h.run(extra);}
function receipt(h,extra=''){h.run(`S.governance.actorId='USR-01';var receiptData={amount:'25.00',date:'2026-11-10',sourceRef:'EDGE-RECEIPT-001',note:'Partial recovery received in the settlement account.'};`);if(extra)h.run(extra);h.run('var edgeReceipt=l29AddReceipt(edgeLoss.id,receiptData);');}
function sourceDocument(h){h.run(`var edgeVersion=S.governance.versions.find(v=>l29EvidenceAllowed(v.id,'U-IK-PAY'));var edgeDocument=recRead('document',edgeVersion.documentId);`);assert(h.run('Boolean(edgeDocument)'));}

test('Only accepted losses and independently verified receipts enter financial totals',h=>{
 const before=h.run('scopeTotals().gross');submitted(h);assert.equal(h.run('scopeTotals().gross'),before);assert.equal(h.run('lossFinanciallyIncluded(edgeLoss)'),false);
 assert.throws(()=>h.run(`l29Review(edgeLoss.id,'Accept','An author must not approve the loss they prepared.');`));
 h.run(`S.governance.actorId='USR-03';l29Review(edgeLoss.id,'Accept','The source amount and transaction facts have been independently verified.');S.governance.actorId='USR-01';`);
 assert.equal(h.run('scopeTotals().gross'),before+10000);receipt(h);assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),0);
 assert.throws(()=>h.run(`l29ReviewReceipt(edgeLoss.id,edgeReceipt.id,'Verify','The preparer cannot verify their own submitted recovery.');`));
 h.run(`S.governance.actorId='USR-03';l29ReviewReceipt(edgeLoss.id,edgeReceipt.id,'Verify','The receipt matches the independently checked settlement record.');`);
 assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),2500);assert.equal(h.run('lossVerifiedRecoveries(edgeLoss,"2026-11-09")'),0);
});

test('Pending corrections retain accepted figures and acceptance preserves prior revision',h=>{
 accepted(h);const original=h.run('edgeLoss.gross'),revision=h.run('edgeLoss.revision');correction(h,`correctionDraft.data.gross='125.00';`);h.run('l29SubmitDraft(correctionDraft.id);');
 assert.equal(h.run('edgeLoss.gross'),original);assert.equal(h.run('edgeLoss.revision'),revision);
 h.run(`S.governance.actorId='USR-03';l29Review(edgeLoss.id,'Accept','Verified the adjustment against the approved accounting source.');`);
 assert.equal(h.run('edgeLoss.gross'),12500);assert.equal(h.run('edgeLoss.versions.at(-1).snapshot.gross'),original);assert.equal(h.run('edgeLoss.revision'),revision+1);
});

test('An identical transaction reference cannot be justified as a distinct loss',h=>{
 accepted(h);h.run(`var secondDraft=l29NewDraft('RSK-04');Object.assign(secondDraft.data,l29RecordData(edgeLoss),{title:'A supposedly separate loss event',duplicateReason:'This is asserted to be another event but it reuses the identical transaction reference.'});`);
 assert.throws(()=>h.run('l29SubmitDraft(secondDraft.id);'),/reference|duplicate/i);
 assert.equal(h.run('S.losses.filter(l=>l.transactionReference==="EDGE-2026-001").length'),1);
});

test('Unavailable loss evidence cannot partially accept a submission',h=>{
 sourceDocument(h);submitted(h,'edgeDraft.data.evidenceVersionId=edgeVersion.id;');h.run(`edgeDocument.archived=true;S.governance.actorId='USR-03';`);const receipts=h.run('S.governance.receipts.length');
 assert.throws(()=>h.run(`l29Review(edgeLoss.id,'Accept','The linked evidence must remain available before acceptance.');`));
 assert.equal(h.run('edgeLoss.workflowState'),'Submitted');assert.equal(h.run('edgeLoss.submission.status'),'Submitted');assert.equal(h.run('S.governance.receipts.length'),receipts);assert.equal(h.run('lossFinanciallyIncluded(edgeLoss)'),false);
});

test('Unavailable receipt evidence cannot partially verify or block rejection',h=>{
 accepted(h);sourceDocument(h);receipt(h,'receiptData.evidenceVersionId=edgeVersion.id;');h.run(`edgeDocument.archived=true;S.governance.actorId='USR-03';`);const receipts=h.run('S.governance.receipts.length');
 assert.throws(()=>h.run(`l29ReviewReceipt(edgeLoss.id,edgeReceipt.id,'Verify','Evidence availability must be checked before updating money.');`));
 assert.equal(h.run('edgeReceipt.status'),'Pending');assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),0);assert.equal(h.run('S.governance.receipts.length'),receipts);
 h.run(`l29ReviewReceipt(edgeLoss.id,edgeReceipt.id,'Reject','The submitted evidence is unavailable and the receipt cannot be verified.');`);
 assert.equal(h.run('edgeReceipt.status'),'Rejected');assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),0);
});

test('A correction opened before closure cannot be submitted during closure review',h=>{
 accepted(h);correction(h);h.run(`l29ClosureRequest(edgeLoss.id,{reason:'No further financial investigation is required for this event.',residualReason:'The remaining amount is recorded and accepted as a final unrecovered loss.',openActionReason:''});`);
 assert.throws(()=>h.run('l29SubmitDraft(correctionDraft.id);'),/clos|review/i);assert.equal(h.run('edgeLoss.closureRequest.status'),'Submitted');assert.equal(h.run('Boolean(edgeLoss.pendingRevision?.status==="Submitted")'),false);
});

test('Correction approval must still cover the proposed destination scope',h=>{
 accepted(h);correction(h,`Object.assign(correctionDraft.data,{orgScopeId:'U-IK-CUSTOMER',risk:'',title:'Settlement loss at corrected location'});`);h.run(`l29SubmitDraft(correctionDraft.id);S.accessControl.directRules.push({id:'EDGE-DENY-DESTINATION',userId:'USR-03',permission:'risks.approve',effect:'deny',scopeId:'U-IK-CUSTOMER',starts:'2026-01-01',ends:'',status:'Active',reason:'Destination review responsibility was withdrawn after submission'});S.governance.actorId='USR-03';`);
 assert(h.run('l29Permission("approve",edgeLoss).allowed'));assert(!h.run('l29Permission("approve",edgeLoss.pendingRevision.snapshot).allowed'));
 assert.throws(()=>h.run(`l29Review(edgeLoss.id,'Accept','The reviewer no longer has approval authority at the proposed location.');`));assert.equal(h.run('edgeLoss.orgScopeId'),'U-IK-PAY');assert.equal(h.run('edgeLoss.pendingRevision.status'),'Submitted');
});

test('Reversal preserves original receipt and changes totals only from its approved date',h=>{
 accepted(h);receipt(h);h.run(`S.governance.actorId='USR-03';l29ReviewReceipt(edgeLoss.id,edgeReceipt.id,'Verify','The received amount was independently matched to the bank source.');S.governance.actorId='USR-01';l29RequestReversal(edgeLoss.id,edgeReceipt.id,{date:'2026-11-12',reason:'Settlement reference REV-001 confirms the original receipt was returned.'});`);assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),2500);
 h.run(`S.governance.actorId='USR-03';l29ReviewReversal(edgeLoss.id,edgeReceipt.id,'Approve','Verified the returned settlement against the independent source reference.');`);
 assert.equal(h.run('lossVerifiedRecoveries(edgeLoss)'),0);assert.equal(h.run('lossVerifiedRecoveries(edgeLoss,"2026-11-11")'),2500);assert.equal(h.run('edgeLoss.recoveries.length'),1);assert.equal(h.run('lossRecoveryReversalRows(edgeLoss)[0].amount'),2500);
 h.run(`S.governance.actorId='USR-01';l29ClosureRequest(edgeLoss.id,{reason:'The investigation is complete and the financial treatment is final.',residualReason:'Retain the entire unrecovered amount as the final event loss.',openActionReason:''});S.governance.actorId='USR-03';l29ReviewClosure(edgeLoss.id,'Approve','The residual financial treatment and investigation outcome were reviewed.');`);
 assert.equal(h.run('edgeLoss.workflowState'),'Closed');assert(h.run('scopeLosses().some(l=>l.id===edgeLoss.id)'));assert.throws(()=>h.run(`l29AddReceipt(edgeLoss.id,{amount:'10.00',date:S.asOf,sourceRef:'LATE-REC-001',note:'Late recovery must require a reopened event.'});`));
});

test('Legacy loss setup commands cannot insert a malformed unreviewed loss',h=>{
 const before=h.run('S.losses.length');h.run(`jStartOperation('loss',{risk:'RSK-04'});current=route();`);
 assert.equal(h.run('current.name.replace(/^\\//,"")'),'loss-intake');assert.equal(h.run('S.losses.length'),before);
 h.run(`var legacyDraft={id:'LEGACY-EDGE-DRAFT',kind:'loss',step:3,data:{title:'Legacy setup event',risk:'RSK-04',orgScopeId:'U-IK-PAY',date:'2026-11-03',type:'Actual loss',gross:'99.00',note:'This legacy draft must enter the reviewed capture workflow.'}};S.journeyDrafts[legacyDraft.id]=legacyDraft;current={name:'setup',params:new URLSearchParams('draft='+legacyDraft.id+'&step=3')};`);
 try{h.run('jSubmitOperation();');}catch(error){assert(/legacy|capture|review|draft|submit/i.test(error.message),error.message);}
 assert.equal(h.run('S.losses.length'),before);assert(h.run('Boolean(S.lossRegister.drafts[legacyDraft.id])||Boolean(S.journeyDrafts[legacyDraft.id])'));
});

test('Completed loss actions keep their earlier accountability when a correction changes the action',h=>{
 accepted(h,`Object.assign(edgeDraft.data,{actionPlan:'Reconcile the payment control exception and verify the amended procedure.',actionOwner:'Amina Bello',actionDue:'2026-12-01',noActionReason:''});`);
 h.run(`var priorTask=S.tasks.find(t=>t.lossEvent===edgeLoss.id);Object.assign(priorTask,{status:'Completed',completed:'2026-11-13',completedBy:'Amina Bello',note:'The original payment reconciliation action was completed.'});var priorCompletion=clone(priorTask);`);
 correction(h,`Object.assign(correctionDraft.data,{actionPlan:'Run a new sample test after the separately identified settlement exception.',actionOwner:'Chidi Okafor',actionDue:'2026-12-15'});`);h.run(`l29SubmitDraft(correctionDraft.id);S.governance.actorId='USR-03';l29Review(edgeLoss.id,'Accept','The additional corrective action requires a separately accountable work episode.');`);
 assert(h.run(`S.tasks.some(t=>t.lossEvent===edgeLoss.id&&t.status==='Open'&&t.owner==='Chidi Okafor')`));
 assert(h.run(`(priorTask.status==='Completed'&&priorTask.owner===priorCompletion.owner&&priorTask.due===priorCompletion.due)||[...(priorTask.completionHistory||[]),...(priorTask.sourceHistory||[])].some(x=>x.owner===priorCompletion.owner&&x.due===priorCompletion.due)`),'Prior completion owner and deadline must survive correction');
});

const report={suite:'Independent V2.9 loss command edge review',browserLayoutValidated:false,passed:results.filter(r=>r.status==='PASS').length,failed:results.filter(r=>r.status==='FAIL').length,results};
fs.writeFileSync(path.join(root,'tests/loss-edge-review-v29-results.json'),JSON.stringify(report,null,2));
console.log(JSON.stringify(report,null,2));
if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
