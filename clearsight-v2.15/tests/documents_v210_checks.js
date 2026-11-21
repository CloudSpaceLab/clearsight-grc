/* Unified Documents workspace: actual-source command/view tests.
   File verification uses synthetic bytes, not browser upload automation. */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
async function test(name,fn){try{await fn(createRuntime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.message});console.error('FAIL',name,error.stack);}}
function pending(h){h.run(`S.governance.actorId='USR-04';var docReview=docEvidenceAdd('DOC-SAMPLE-POLICY-V1',{programId:'PRG-104',target:'Meridian Bank',question:'Q01',observed:S.asOf});var sourceVersion=docVersion(docReview.documentVersionId);var sourceDocument=recRead('document',sourceVersion.documentId);`);}
function reviewer(h){h.run('S.governance.actorId=docReview.assignedReviewerId;');}

(async()=>{
await test('Documents and the former Evidence entry share one canonical library',h=>{
 const count=h.run('S.governance.documents.length'),all=h.navigate('documents'),reviews=h.navigate('evidence');assert(all.includes('Documents'));assert(reviews.includes('Document review queue'));assert(!reviews.includes('<h1>Evidence'));assert.equal(h.run('S.governance.documents.length'),count);assert.equal(h.run('current.name'),'documents');
});
await test('Legacy review links open the exact document version and assessment context',h=>{
 h.run("S.governance.actorId='USR-03';"); // Authorized reader for the Head Office source.
 const html=h.navigate('evidence-review?id=EV-01');assert.equal(h.run('current.name'),'document');assert.equal(h.run('current.params.get("review")'),'EV-01');assert(html.includes('Assessment context'));assert(!html.includes('metadata-only record'));assert(!html.includes('Evidence library'));
 h.run(`evidenceDetail('EV-01');`);assert(h.context.location.hash.includes('document?id=DOC-EV-01'));assert(h.context.location.hash.includes('review=EV-01'));
});
await test('Multiple assessment reviews reference one stored document and file version',h=>{
 const docs=h.run('S.governance.documents.length'),versions=h.run('S.governance.versions.length');pending(h);h.run(`var secondReview=docEvidenceAdd('DOC-SAMPLE-POLICY-V1',{programId:'PRG-104',target:'Meridian Bank',question:'Q02',observed:S.asOf});`);assert.notEqual(h.run('docReview.id'),h.run('secondReview.id'));assert.equal(h.run('S.governance.documents.length'),docs);assert.equal(h.run('S.governance.versions.length'),versions);assert.equal(h.run('d210Contexts("DOC-SAMPLE-POLICY").filter(x=>[docReview.id,secondReview.id].includes(x.id)).length'),2);
});
await test('Linking submitter is recorded separately from original file uploader',h=>{
 pending(h);assert.equal(h.run('sourceVersion.uploadedBy'),'USR-01');assert.equal(h.run('docReview.submittedById'),'USR-04');assert(h.run('Boolean(docReview.submittedAt)'));assert.equal(h.run('docReview.documentReviewHistory[0].actorId'),'USR-04');const html=h.navigate(h.run('d210ReviewRoute(docReview)'));assert(html.includes('Daniel Mensah'));assert(html.includes('Amina Bello'));assert(html.includes('File uploaded by'));assert(html.includes('Submitted by'));
});
await test('Actual submitter cannot impersonate a reviewer or approve their own document',async h=>{
 pending(h);const count=h.run('S.governance.receipts.length');await assert.rejects(()=>h.run(`d210Review(docReview.id,{decision:'Accept',note:'A different name does not establish a separate acting reviewer.',reviewer:'Ifeoma Nwosu'});`));assert.equal(h.run('docReview.status'),'Submitted');assert.equal(h.run('S.governance.receipts.length'),count);
});
await test('Assigned eligible reviewer verifies bytes and records attributable decision history',async h=>{
 pending(h);reviewer(h);const expected=h.run('recActor().id'),count=h.run('S.governance.receipts.length');await h.run(`d210Review(docReview.id,{decision:'Accept',note:'The exact file and assessment context were independently checked.'},docReview.documentVersionId);`);assert.equal(h.run('docReview.status'),'Accepted');assert.equal(h.run('docReview.review.byId'),expected);assert.equal(h.run('docReview.documentReviewHistory.at(-1).actorId'),expected);assert.equal(h.run('S.governance.receipts.length'),count+1);assert.equal(h.run('evaluation(recRead("program","PRG-104"),"Meridian Bank","R01").status'),'Satisfied');
});
await test('Archived or corrupt source cannot partially accept a document',async h=>{
 pending(h);reviewer(h);const count=h.run('S.governance.receipts.length');h.run('sourceDocument.archived=true;');await assert.rejects(()=>h.run(`d210Review(docReview.id,{decision:'Accept',note:'The archived source is unavailable for a current acceptance.'});`));assert.equal(h.run('docReview.status'),'Submitted');h.run(`sourceDocument.archived=false;sourceVersion.sampleText+='tampered';`);await assert.rejects(()=>h.run(`d210Review(docReview.id,{decision:'Accept',note:'The corrupt content must fail its exact version checksum.'});`),/checksum/);assert.equal(h.run('docReview.status'),'Submitted');assert.equal(h.run('S.governance.receipts.length'),count);
});
await test('Unavailable document can be returned for correction and reaches the actual submitter',async h=>{
 pending(h);reviewer(h);h.run('sourceDocument.archived=true;');await h.run(`d210Review(docReview.id,{decision:'Request changes',note:'Please restore and submit the current signed source document.'});`);assert.equal(h.run('docReview.status'),'Changes requested');assert(h.run('S.notificationsV28.events.some(x=>x.id===docReview.returnNotificationId&&x.actionable&&x.recipientIds.includes("USR-04"))'));assert.equal(h.run('workItems().find(x=>x.ref===docReview.id&&x.source==="evidence-correction").owner'),'Daniel Mensah');assert(h.run('workItems().find(x=>x.ref===docReview.id&&x.source==="evidence-correction").route.includes("document?id=")'));
});
await test('Review route rejects a contextual review belonging to another document',h=>{
 pending(h);reviewer(h);const html=h.navigate('document?id=DOC-SAMPLE-KRI&tab=assessment&review='+h.run('docReview.id'));assert(html.includes('does not belong'));assert(!html.includes('The exact file and assessment context'));assert(!html.includes('data-d210-submit="review"'));
});
await test('Legacy records retain unknown submitting identity without invented timestamps',h=>{
 h.run("S.governance.actorId='USR-03';");h.navigate('evidence-review?id=EV-01');assert(h.run('!S.evidence.find(x=>x.id==="EV-01").submittedAt'));assert(h.run('!S.evidence.find(x=>x.id==="EV-01").submittedById'));assert(h.elements.get('#main-content').innerHTML.includes('Legacy submitter not recorded'));
});
await test('Expiry prevents current acceptance while preserving source and history',async h=>{
 pending(h);h.run(`docReview.expires='2026-11-01';`);reviewer(h);await assert.rejects(()=>h.run(`d210Review(docReview.id,{decision:'Accept',note:'An expired version cannot become current by changing review state.'});`),/Refresh|dates/);assert.equal(h.run('docReview.status'),'Submitted');assert.equal(h.run('docReview.documentReviewHistory.length'),1);
});
await test('All visible document commands and canonical routes are registered',h=>{
 for(const route of ['documents','documents?view=pending','documents?view=refresh','documents?view=reviews','evidence-review?id=EV-01']){const html=h.navigate(route);assert(!html.includes('NaN'));assert(!html.includes('[object Object]'));for(const [,action] of html.matchAll(/data-action="([^"]+)"/g)){h.context.actionName=action;assert(h.run('typeof actions[actionName]==="function"'),action);}for(const [,name] of html.matchAll(/href="#\/([^"?]+)/g)){h.context.routeName=name;assert(h.run('Boolean(titles[routeName])'),name);}}
});
const summary={suite:'Documents consolidation V2.10',browserLayoutValidated:false,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,results};fs.writeFileSync(path.join(root,'tests/documents-v210-results.json'),JSON.stringify(summary,null,2)+'\n');console.log(summary.passed+'/'+results.length+' checks passed.');if(summary.failed)process.exitCode=1;
})();
