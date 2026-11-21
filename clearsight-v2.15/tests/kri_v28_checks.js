'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function test(name,fn){const h=createRuntime();try{fn(h);results.push({name,status:'PASS'});console.log('PASS '+name)}catch(error){results.push({name,status:'FAIL',error:String(error),stack:error.stack});console.error('FAIL '+name+' '+error.stack)}}
const ret="S.branchKri.returns['KRC-BRANCH-2026-10::B-EN']";
test('51 reference questions retain stable source columns and eight operational groups',h=>{
 assert.equal(h.run('CS_BRANCH_REFERENCE.metrics.length'),51);assert.equal(h.run('new Set(CS_BRANCH_REFERENCE.metrics.map(m=>m.sourceColumn)).size'),51);
 assert.equal(h.run('CS_BRANCH_REFERENCE.groups.length'),8);assert.equal(h.run('CS_BRANCH_REFERENCE.source.thresholdsProvided'),false);
 assert.equal(h.run('ixCycle().population.every(id=>orgNode(id).kind==="branch")'),true);
});
test('Empty, zero, unavailable and not-applicable answers remain distinct',h=>{
 assert.equal(h.run("ixMetricValue(ixCycle().metrics[0],{})"),null);assert.equal(h.run("ixMetricValue(ixCycle().metrics[0],{value:0})"),0);
 assert.equal(h.run("ixBand(ixCycle().metrics[0],{missing:true,value:0})"),'Unknown');assert.equal(h.run("ixBand(ixCycle().metrics[0],{na:true})"),'Not applicable');
 assert.equal(h.run('ixKriStats().known'),50);assert.equal(h.run('ixKriStats().notApplicable'),1);assert.equal(h.run('ixKriStats().unknown'),51);
});
test('Currency, decimal hours, Boolean polarity and contextual answers keep their types',h=>{
 assert.equal(h.run("ixDisplay(ixCycle().metrics.find(m=>m.id==='cashShort'),{value:125000})"),'₦1,250');
 assert.equal(h.run("ixMetricValue(ixCycle().metrics.find(m=>m.id==='powerHours'),{value:3.5})"),3.5);
 assert.equal(h.run("ixMetricValue(ixCycle().metrics.find(m=>m.id==='tellers'),{value:3.5})"),null);
 assert.equal(h.run("ixMetricValue(ixCycle().metrics.find(m=>m.id==='cctv'),{value:'Yes'})"),null);
 assert.equal(h.run("ixBand(ixCycle().metrics.find(m=>m.id==='cctv'),{value:false})"),'Breached');
 assert.equal(h.run("ixBand(ixCycle().metrics.find(m=>m.id==='fireIncident'),{value:true})"),'Breached');
 assert.equal(h.run("ixBand(ixCycle().metrics.find(m=>m.id==='lockerKeys'),{value:true})"),'Context only');
});
test('Inconsistent equipment and guard answers are blocked at submission',h=>{
 h.run(`${ret}.values.rotorFunctional.value=4;${ret}.values.cctv.value=true;${ret}.values.faultyCameras.value=1;${ret}.values.armedGuards.value=0;`);
 const messages=h.run(`k28Issues(${ret}).map(x=>x.message).join(' ')`);
 assert.match(messages,/rotor doors cannot exceed/);assert.match(messages,/CCTV status/);assert.match(messages,/24-hour armed surveillance/);
});
test('Conditional child drafts are preserved, never counted while inapplicable',h=>{
 h.run(`${ret}.values.cardPrinter.value=false;${ret}.values.cardReplacement={value:false,detail:{summary:'Replacement on order',reference:'PO-123'}};k28Normalize(${ret});${ret}.values.cardPrinter.value=true;k28Normalize(${ret});`);
 assert.equal(h.run(`${ret}.values.cardReplacement.na`),true);
 h.run(`${ret}.values.cardPrinter.value=false;k28Normalize(${ret});`);
 assert.equal(h.run(`${ret}.values.cardReplacement.value`),false);assert.equal(h.run(`${ret}.values.cardReplacement.detail.reference`),'PO-123');
 h.run(`${ret}.values.cardPrinter.missing=true;k28Normalize(${ret});`);assert.equal(h.run(`${ret}.values.cardReplacement.missing`),true);assert.equal(h.run(`${ret}.values.cardReplacement.na`),undefined);
});
test('Applicable replacement cannot be marked N/A; impossible action dates fail',h=>{
 h.run(`${ret}.values.cardPrinter.value=false;${ret}.values.cardReplacement={na:true,reason:'Trying to skip a relevant follow-up'};${ret}.values.inverter.detail.due='2026-02-31';`);
 const messages=h.run(`k28Issues(${ret}).map(x=>x.message).join(' ')`);assert.match(messages,/answer Yes or No/);assert.match(messages,/valid action due date/);
});
test('Branch aggregation uses integer minor units, adverse counts and context counts',h=>{
 assert.equal(h.run("ixAggregate(ixCycle(),ixCycle().metrics.find(m=>m.id==='cashShort'),['B-IK']).value"),125000);
 assert.equal(h.run("ixAggregate(ixCycle(),ixCycle().metrics.find(m=>m.id==='cctv'),['B-IK']).value"),1);
 h.run("S.branchKri.returns['KRC-BRANCH-2026-10::B-IK'].accepted.values.ewaste.value=true");
 assert.equal(h.run("ixAggregate(ixCycle(),ixCycle().metrics.find(m=>m.id==='ewaste'),['B-IK']).value"),1);
 assert.equal(h.run("ixAggregate(ixCycle(),ixCycle().metrics.find(m=>m.id==='ewaste'),['B-IK']).breaches"),0);
});
test('New cycles reject duplicate branch-months and preserve old metric snapshots',h=>{
 h.run("actions['ix-new-cycle']();Object.assign(S.branchKri.draftCycle,{period:'2026-10',population:['B-IK'],due:'2026-11-10'});");
 assert.match(h.run('k28CycleIssues(S.branchKri.draftCycle).join(" ")'),/already exists/);
 h.run("S.branchKri.referencePolicy[0].limit=1000;");assert.equal(h.run('ixCycle().metrics[0].limit'),0);
});
test('Reference source titles and input values are escaped in return HTML',h=>{
 h.run("S.governance.actorId='USR-04';"); // Enugu contributor reads their exact branch return.
 h.run(`${ret}.values.waterShortage.reason='<img src=x onerror=alert(1)> unavailable source';${ret}.remarks='<script>unsafe</script>';`);
 const html=h.navigate('kri-return?id=KRC-BRANCH-2026-10%3A%3AB-EN&section=review');assert(!html.includes('<script>unsafe'));assert(html.includes('&lt;script&gt;unsafe'));
});
test('The standalone route and every KRI section render through the complete module chain',h=>{
 h.run("S.governance.actorId='USR-04';");
 assert.match(h.navigate('kri-returns'),/51 source-linked questions/);
 for(const section of h.run('CS_BRANCH_REFERENCE.groups.map(g=>g.id).concat("review")'))assert(h.navigate('kri-return?id=KRC-BRANCH-2026-10%3A%3AB-EN&section='+section).includes('k28-return-layout'));
 assert(h.navigate('kri-library').includes('Reporting definitions'));
});
test('Missing and outside-scope return IDs do not silently substitute a branch',h=>{
 assert.match(h.navigate('kri-return?id=DOES-NOT-EXIST'),/Return not found|unavailable|not found/);
 h.run("S.scope={branch:'B-IK',department:'',unit:''};");assert.match(h.navigate('kri-return?id=KRC-BRANCH-2026-10%3A%3AB-EN'),/Scope mismatch|Outside|outside/);
});
fs.writeFileSync(path.join(root,'tests/kri-v28-results.json'),JSON.stringify({runtime:'Node actual-source; browser shell stubbed',results},null,2));
if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
