/* V2.11 operational interaction adapters. Existing domain commands, validators,
 * accepted snapshots and money/date helpers remain authoritative. No new store. */
'use strict';

const OP211_PRIOR={
 kri:window.CS_JOURNEYS['kri-return'],lossStep:l29CaptureStep,lossIntake:window.CS_JOURNEYS['loss-intake'],loss:window.CS_JOURNEYS.loss,
 rcsa:rcsaReviewView,controlTest:window.CS_JOURNEYS['control-test'],controlEditor:window.CS_JOURNEYS['control-editor'],riskEditor:window.CS_JOURNEYS['risk-edit'],risk:window.CS_JOURNEYS.risk,
 lossBody:lrpBody,concentration:lrpConcentrationPanel,age:lrpAgePanel,prospects:lrpProspectsPanel,reportIssues:brShowDraftIssues
};
const OP211_LOSS_FIELDS=[
 ['title','type','orgScopeId','owner','region','directorate','accountName','lossGroup','transactionReference','risk','transactionParticulars'],
 ['currency','gross','date','discoveredOn','recognizedOn','reportingPeriod','periodReason','sourceRef','evidenceVersionId'],
 ['rootCauseStatus','rootCause','actionPlan','actionOwner','actionDue','noActionReason','recoveryStatus','expectedAmount','recoveryTarget','recoveryNote'],
 ['reviewer','duplicateReason','correctionReason']
];
const op211LossStep=field=>Math.max(1,OP211_LOSS_FIELDS.findIndex(fields=>fields.includes(field))+1);
const op211Text=value=>value===null||value===undefined||value===''?'Not recorded':Array.isArray(value)?value.join(' · '):String(value);
const op211FocusButton=(label,selector,kind='quiet small')=>btn(label,'ui211-focus',`data-selector="${e(selector)}"`,kind);

// KRI review starts with actionable observations, while complete detail stays one click away.
function op211KriState(r){
 const c=ixCycle(r.cycle),snapshot=k28Snapshot(r),values=snapshot.values||{},editable=['Draft','Changes requested'].includes(r.status),issues=editable?k28Issues(r,true):[],byMetric=new Map();
 for(const issue of issues){if(!byMetric.has(issue.id))byMetric.set(issue.id,[]);byMetric.get(issue.id).push(issue.message);}
 const rows=c.metrics.map(m=>({m,value:values[m.id]||{},band:ixBand(m,values[m.id]||{}),issues:byMetric.get(m.id)||[]}));
 return {c,r,snapshot,editable,issues,rows,attention:rows.filter(row=>row.issues.length||['Breached','Unknown'].includes(row.band)),recorded:rows.filter(row=>!['Unknown','Not applicable'].includes(row.band)).length,notApplicable:rows.filter(row=>row.band==='Not applicable').length,unavailable:rows.filter(row=>row.band==='Unknown'&&row.value.missing).length,unanswered:rows.filter(row=>row.band==='Unknown'&&!row.value.missing).length};
}
function op211KriRoute(r,id){const c=ixCycle(r.cycle),m=c.metrics.find(m=>m.id===id);return ui211Route(ixReturnRoute(r.id),{section:m?.group||'review',focus:m?'#k28-'+m.id:'[data-k28-return="remarks"]'});}
function op211KriChecks(state){
 const items=state.issues.map(issue=>({message:issue.message,route:issue.id==='return'&&!/remarks/i.test(issue.message)?undefined:op211KriRoute(state.r,issue.id)}));
 return `<div class="op211-kri-readiness"><div class="op211-readiness-counts"><span><b>${state.recorded}</b> recorded</span><span><b>${state.notApplicable}</b> N/A</span><span><b>${state.unavailable}</b> unavailable</span><span><b>${state.unanswered}</b> unanswered</span></div>${items.length?ui211Issues(items,{title:'Resolve before submission'}):state.editable?`<p class="op211-ready">${icon('check')} No submission blockers. ${state.unavailable?'Unavailable observations retain their reasons and will remain reporting gaps.':'Confirm the source facts before submitting for independent review.'}</p>`:''}</div>`;
}
function op211KriAttention(state){
 if(!state.attention.length)return empty('No exceptions or unanswered observations in this return','Open All details to inspect every observation and its source before recording a decision.');
 return `<div class="op211-attention-register">${table(['Observation','Answer / availability','Attention / next action','Owner / due',''],state.attention.map(({m,value,band,issues})=>({cells:[`<strong>${e(m.name)}</strong><small>${e(state.c.groups?.find(g=>g.id===m.group)?.name||m.group)}</small>`,`${k28Badge(band)}<small>${e(ixDisplay(m,value))}${value.reason?'<br>'+e(value.reason):''}</small>`,issues.length?`<ul>${issues.map(message=>`<li>${e(message)}</li>`).join('')}</ul>`:band==='Breached'?e(value.detail?.action||'Review the exception and its treatment.'):e(value.missing?'Confirm the reason this observation is unavailable.':'Supply the observation or record why it is unavailable.'),band==='Breached'?`${e(value.detail?.owner||'Not assigned')}<small>${value.detail?.due?date(value.detail.due):'Due date not set'}</small>`:'—',link(issues.length?'Complete observation':'Inspect observation',op211KriRoute(state.r,m.id),'small')]})))}</div>`;
}
function op211KriNextContent(state){const first=state.issues.find(i=>i.id!=='return')?.id||state.attention[0]?.m.id,count=state.issues.length,primary=first?link(count?'Go to first item':'Inspect first exception / gap',op211KriRoute(state.r,first),'small'):'';return `<span>${count?`<strong>${count} submission ${count===1?'check':'checks'} to resolve</strong>`:`<strong>${state.recorded+state.notApplicable} of ${state.c.metrics.length} observations accounted for</strong>`}<small>${state.unavailable} unavailable · ${state.unanswered} unanswered · ${state.notApplicable} N/A</small></span><div class="row wrap">${primary}${link('Review whole return',ui211Route(ixReturnRoute(state.r.id),{section:'review'}),'quiet small')}</div>`;}
function op211KriView(){
 let html=OP211_PRIOR.kri();const r=ixCurrentReturn();if(!r||ixReturnGuard(r)||!k28Is(ixCycle(r.cycle)))return html;
 const state=op211KriState(r),c=state.c,section=current.params.get('section')||(c.groups||CS_BRANCH_REFERENCE.groups)[0].id,filter=current.params.get('filter')||'all',base=ixReturnRoute(r.id);
 if(section==='review'){
  html=html.replace(/<div class="k28-review-summary">[\s\S]*?(?=<div class="k28-section-heading">)/,`<div class="k28-review-summary" id="op211-kri-checks" aria-live="polite">${op211KriChecks(state)}</div>`);
  const focused=!current.params.has('filter')||filter==='attention';
  const controls=[['attention','Attention'],['all','All details'],['exceptions','Exceptions'],['gaps','Unknown / unavailable']].map(([id,label])=>link(label,ui211Route(base,{section:'review',filter:id}),'small '+((focused?id==='attention':filter===id)?'primary':'quiet'))).join('');
  html=html.replace(/<div class="k28-section-heading">[\s\S]*?<\/div><\/div>/,`<div class="k28-section-heading"><h2>${focused?'Review focus':'Return observations'}</h2><div class="row wrap">${controls}</div></div>`);
  if(focused){const cards=c.metrics.map((m,i)=>k28Question(m,state.snapshot.values,state.editable,i)).join('');html=html.replace(cards,op211KriAttention(state));}
 }else{
  const bar=`<div class="op211-kri-next" id="op211-kri-next">${op211KriNextContent(state)}</div>`;
  html=html.replace('<div id="ix-return-error" role="alert"></div>','<div id="ix-return-error" role="alert"></div>'+bar);
 }
 return html;
}
window.CS_JOURNEYS['kri-return']=op211KriView;

// Field errors navigate through the existing loss draft, saving the current step first.
function op211LossIssues(d,data=d.data){return l29Issues(data,{exclude:d.targetId||d.id,correction:d.kind==='correction'});}
function op211LossChecks(d,data=d.data){
 const all=op211LossIssues(d,data),issues=d.step===4?all:all.filter(issue=>op211LossStep(issue.field)===d.step);
 if(!issues.length)return `<p class="op211-ready">${icon('check')} ${d.step===4?'No capture checks remain. Submit when the source facts are ready for independent review.':'This section has no outstanding capture checks.'}</p>`;
 const list=`<ul>${issues.map(issue=>`<li><button type="button" data-action="op211-loss-field" data-id="${e(d.id)}" data-field="${e(issue.field)}"><span>${e(issue.message)}</span><small>${e(L29_STEPS[op211LossStep(issue.field)-1])}</small></button></li>`).join('')}</ul>`;
 return `<details class="op211-field-checks" ${d.step===4?'open':''}><summary>${issues.length} ${issues.length===1?'item':'items'} ${d.step===4?'to complete before submission':'to complete in this section'}</summary>${list}</details>`;
}
l29CaptureStep=function(d){
 let html=OP211_PRIOR.lossStep(d);if(d.step===4)html=html.replace(/<div class="l29-validation">[\s\S]*$/, '');
 return html+`<div id="op211-loss-checks" aria-live="polite">${op211LossChecks(d)}</div>`;
};
function op211LossField(id,field,form){
 const d=l29Draft(id);requireValue(OP211_LOSS_FIELDS.some(fields=>fields.includes(field)),'This capture field is unavailable.');
 requireValue(current.name==='loss-intake'&&current.params.get('id')===id,'Open the current loss draft before following its checks.');
 const source=form||document.querySelector(`[data-l29-form="capture"][data-id="${id}"]`);if(source?.dataset?.id)requireValue(source.dataset.id===id,'The displayed draft changed. Reopen the loss draft.');
 l29SaveDraft(id,source?values(source):{},op211LossStep(field));
 const route=ui211Route('loss-intake?id='+encodeURIComponent(id),{focus:`[name="${field}"]`});go(route);return route;
}
actions['op211-loss-field']=el=>{try{op211LossField(el.dataset.id,el.dataset.field,el.closest?.('form'));}catch(error){l29DialogError(error);}};

function op211CompareRows(rows,{before='Current record',after='Prepared change',title='Review the changes'}={}){
 if(!rows.length)return `<p class="op211-ready">No changed facts in this proposal.</p>`;
 return `<section class="op211-comparison" aria-label="${e(title)}"><div class="op211-comparison-heading"><strong>${e(title)}</strong><span>${rows.length} changed ${rows.length===1?'field':'fields'}</span></div>${table(['Field',before,after],rows.map(row=>({cells:[e(row.label),e(op211Text(row.before)),e(op211Text(row.after))]})))}</section>`;
}
const OP211_LOSS_COMPARE=[['title','Event title'],['type','Event classification'],['accountName','Account classification'],['lossGroup','Loss group'],['transactionReference','Transaction reference'],['transactionParticulars','Transaction particulars'],['orgScopeId','Accountable location'],['region','Regional bank'],['directorate','Directorate'],['risk','Linked risk'],['owner','Loss owner'],['date','Occurrence'],['recognizedOn','Recognition'],['reportingPeriod','Reporting month'],['periodReason','Period explanation'],['currency','Currency'],['gross','Gross booked amount'],['rootCauseStatus','Cause status'],['rootCause','Root cause'],['actionPlan','Corrective action'],['actionOwner','Action owner'],['actionDue','Action due'],['noActionReason','No-action rationale'],['sourceRef','Source reference'],['evidenceVersionId','Supporting document version'],['reviewer','Assigned reviewer']];
function op211LossChanges(before,after){
 const format=(key,value,record)=>key==='gross'?l29Money(value,record.currency):key==='orgScopeId'?orgName(value):value;
 const rows=OP211_LOSS_COMPARE.filter(([key])=>recStable(before[key]??'')!==recStable(after[key]??'')||key==='gross'&&before.currency!==after.currency).map(([key,label])=>({label,before:format(key,before[key],before),after:format(key,after[key],after)}));
 for(const [key,label]of [['status','Recovery assessment'],['expectedAmount','Expected recovery'],['targetDate','Recovery target'],['note','Recovery narrative']])if(recStable(before.recoverability?.[key]??'')!==recStable(after.recoverability?.[key]??'')||key==='expectedAmount'&&before.currency!==after.currency&&after.recoverability?.expectedAmount!=null)rows.push({label,before:key==='expectedAmount'&&before.recoverability?.[key]!=null?l29Money(before.recoverability[key],before.currency):before.recoverability?.[key],after:key==='expectedAmount'&&after.recoverability?.[key]!=null?l29Money(after.recoverability[key],after.currency):after.recoverability?.[key]});
 return rows;
}
function op211RecoveryQueue(l){
 const pending=l.recoveries.filter(r=>r.status==='Pending'||r.reversal?.status==='Submitted');if(!pending.length)return '';
 return panel('Recovery decisions to complete',`Only verified receipts and approved reversals change the balance. ${pending.length} pending ${pending.length===1?'item':'items'}.`,table(['Source / requested by','Amount','Pending decision',''],pending.map(r=>{const reversing=r.reversal?.status==='Submitted',maker=reversing?r.reversal.requestedBy:r.createdBy;return {cells:[`<strong>${e(r.sourceRef||r.id)}</strong><small>${e(recUser(maker)?.name||'Not recorded')} · ${e(r.id)}</small>`,l29Money(r.amount,l.currency),reversing?'Reversal approval':'Receipt verification',l29AllowedButton(reversing?'Review reversal':'Verify / reject',reversing?'l29-review-reverse':'l29-review-receipt',l,`data-id="${e(l.id)}" data-receipt="${e(r.id)}"`,'approve',maker)]};})),op211FocusButton('Full ledger','#op211-loss-recoveries'));
}
function op211DisableAction(html,action,reason){return html.replace(new RegExp('<button\\b[^>]*data-action="'+action+'"[^>]*>([\\s\\S]*?)<\\/button>','g'),(_,label)=>`<button type="button" class="btn small" disabled title="${e(reason)}">${label}</button>`);}
function op211LossView(){
 let html=OP211_PRIOR.loss();const l=S.losses.find(l=>l.id===current.params.get('id'));if(!l||!l29InScope(l))return html;
 const correction=lossFinanciallyIncluded(l)&&l.pendingRevision?.status==='Submitted',closure=l.closureRequest?.status==='Submitted',pending=l.recoveries.some(r=>r.status==='Pending'||r.reversal?.status==='Submitted');
 if(correction){const summary=l29Summary(l.pendingRevision.snapshot);html=html.replace(summary,op211CompareRows(op211LossChanges(l,l.pendingRevision.snapshot),{before:'Accepted revision '+l.revision,after:'Submitted correction',title:'What this correction would change'})+`<details class="op211-source-detail"><summary>Inspect the full submitted correction</summary>${summary}</details>`);}
 if(correction||closure)html=op211DisableAction(html,'add-recovery','Complete the pending '+(correction?'correction':'closure')+' review before adding a recovery.');
 if(correction||pending)html=op211DisableAction(html,'l29-close','Complete the pending correction and recovery decisions before requesting closure.');
 for(const [title,id]of [['Financial record & transaction context','facts'],['Recovery ledger','recoveries'],['Remediation & recovery pursuit','actions'],['Review history','history']])html=html.replace('<h2>'+title+'</h2>',`<h2 id="op211-loss-${id}" tabindex="-1">${title}</h2>`);
 const nav=`<nav class="op211-record-jumps" aria-label="Loss record sections">${[['Facts','facts'],['Recoveries','recoveries'],['Action plan','actions'],['History','history']].map(([label,id])=>op211FocusButton(label,'#op211-loss-'+id)).join('')}</nav>`;
 return html.replace(/(<div class="l29-actionbar">[\s\S]*?<\/div>)/,'$1'+nav+op211RecoveryQueue(l));
}
window.CS_JOURNEYS.loss=op211LossView;

// A saved first-line draft is shown against the effective rating without changing it.
function op211RcsaState(){
 const c=scopeItems('cycles').find(c=>c.id===current.params.get('id'))||scopeItems('cycles')[0];if(!c)return null;
 const rid=c.risks.includes(current.params.get('risk'))?current.params.get('risk'):c.risks[0],r=S.risks.find(r=>r.id===rid);if(!r)return null;
 const submitted=c.reviews[r.id]||{status:'Draft'},draft=!['Submitted','Approved'].includes(submitted.status)&&wf29State().rcsaDrafts[wf29RcsaKey(c,r)],v=draft?{...submitted,...draft}:submitted,control=recRead('control',r.control),pair=[Number(v.likelihood||r.residual[0]),Number(v.impact||r.residual[1])],valid=pair.every(n=>Number.isInteger(n)&&n>=1&&n<=5),hasProposal=!!draft||!!submitted.likelihood;
 const sourceChanged=submitted.status==='Submitted'&&((submitted.sourceBasis&&recStable(submitted.sourceBasis)!==recStable(wf29RiskBasis(r,control)))||(submitted.effectiveResidualAtSubmit&&recStable(submitted.effectiveResidualAtSubmit)!==recStable(r.residual)));
 return {c,r,submitted,v,draft:!!draft,control,pair:valid?pair:null,hasProposal,sourceChanged};
}
function op211RcsaCompare(state){
 const {r,v,pair,hasProposal,sourceChanged}=state,scoreNow=score(r.residual),proposed=hasProposal&&pair?score(pair):null,change=proposed===null?null:proposed-scoreNow,approved=v.status==='Approved';
 return `<div class="op211-rating-compare"><div><span>Effective residual now</span><strong>${scoreNow}<small> / 25</small></strong><p>Likelihood ${r.residual[0]} × impact ${r.residual[1]}</p><small>${scoreNow>r.appetite?'Above':'Within'} recorded appetite ≤ ${r.appetite}</small></div><div><span>${approved?'Accepted proposal':state.draft?'Saved first-line draft':'Submitted proposal'}</span><strong>${proposed??'—'}<small> / 25</small></strong><p>${proposed===null?'No proposal prepared':`Likelihood ${pair[0]} × impact ${pair[1]}`}</p><small>${proposed===null?'Complete the first-line assessment.':approved?'Historical accepted proposal; current effective rating is shown separately.':`${change===0?'No rating change':Math.abs(change)+' points '+(change>0?'higher':'lower')} · ${proposed>r.appetite?'still above':'within'} appetite if approved.`}</small></div></div>${sourceChanged?ui211Issues([{message:'The risk, control or effective rating changed after submission. Inspect the source and challenge this proposal for a refreshed assessment.',route:'risk?id='+r.id+'&tab=assurance'}],{title:'Source changed since submission'}):''}`;
}
rcsaReviewView=function(){const html=OP211_PRIOR.rcsa(),state=op211RcsaState();return state?html.replace(/<div class="compare-grid">[\s\S]*?<\/div><div class="row between mt16">/,`<div id="op211-rcsa-compare">${op211RcsaCompare(state)}</div><div class="row between mt16">`):html;};

function op211ControlComparison(c,proposal,status,digest){
 const changed=digest&&digest!==wf29ControlDigest(c),conclusion=proposal.recommendation||'Not selected',same=conclusion===c.effectiveness;
 return `<section class="op211-control-compare" aria-label="Effective and proposed control conclusion"><div><span>Effective now</span><strong>${e(c.effectiveness||'Not assessed')}</strong><small>${e(c.lastTestId||'Existing control record')}</small></div><span class="op211-compare-arrow" aria-hidden="true">${icon('arrow')}</span><div><span>${status==='Approved'?'Accepted test conclusion':status==='Submitted'?'Proposed on submission':'Proposed in this draft'}</span><strong>${e(conclusion)}</strong><small>${status==='Approved'?'A later accepted test may determine the current conclusion.':same?'No effectiveness change proposed.':'Only an accepted independent test changes effectiveness.'}</small></div></section>${changed?ui211Issues([{message:status==='Approved'?'The current implementation differs from this historical test. Its recorded acceptance is retained; inspect the current control for present assurance.':'The control implementation changed after this test was prepared. Inspect the current control and use a corrected test before approval.',route:'control?id='+c.id}],{title:status==='Approved'?'Historical test basis':'Test source needs refresh'}):''}`;
}
window.CS_JOURNEYS['control-test']=function(){
 const html=OP211_PRIOR.controlTest(),c=recRead('control',current.params.get('id'));if(!c||!recInScope('control',c))return html;
 const id=current.params.get('test'),t=id&&wf29State().controlTests.find(t=>t.id===id&&t.controlId===c.id),d=!id&&wf29Draft('control-test',c.id);if(!t&&!d)return html;
 const compare=`<div id="op211-control-compare">${op211ControlComparison(c,t||d.values,t?.status||'Draft',t?.definitionDigest||d.baseDigest)}</div>`;
 return t?html.replace('<div class="wf29-two">',compare+'<div class="wf29-two">'):html.replace('<div id="wf29-error"',compare+'<div id="wf29-error"');
};
function op211ContextChanges(kind,record,draft){
 const fields=kind==='risk'?[['title','Risk title'],['owner','Accountable owner'],['category','Category'],['control','Primary control'],['cause','Cause'],['event','Risk event'],['impact','Business consequence']]:[['title','Control name'],['owner','Accountable owner'],['orgScopeId','Implementation location'],['frequency','Operating frequency'],['execution','Execution'],['procedureRef','Procedure reference'],['objective','Control objective'],['description','Operating procedure'],['reqs','Requirement mappings'],['risks','Supported risks']];
 const value=(obj,key)=>key==='orgScopeId'&&!obj[key]?wf29CommonScope(wf29ControlScopes(record)):obj[key]??'';
 const format=(key,v)=>key==='control'?recRead('control',v)?.title||v:key==='orgScopeId'?orgName(v):Array.isArray(v)?v.map(id=>key==='reqs'?findR(id)?.code||id:recRead('risk',id)?.title||id).join(' · '):v;
 const comparable=v=>Array.isArray(v)?recStable([...v].sort()):recStable(v);
 return fields.filter(([key])=>comparable(value(record,key))!==comparable(value(draft,key))).map(([key,label])=>({label,before:format(key,value(record,key)),after:format(key,value(draft,key))}));
}
function op211EditorChanges(kind){const id=current.params.get('id'),record=recRead(kind,id),d=wf29Draft(kind,id);if(!record||!d)return '';const rows=op211ContextChanges(kind,record,d.values);return rows.length?`<details class="op211-editor-changes"><summary>Review ${rows.length} prepared ${rows.length===1?'change':'changes'} before saving</summary>${op211CompareRows(rows,{before:'Saved '+kind,after:'Draft edit',title:'Prepared changes'})}<p>${kind==='risk'?'Accepted risk ratings stay unchanged.':'Accepted test snapshots stay unchanged; a changed implementation may require a new test.'}</p></details>`:'';}
for(const [route,kind,prior]of [['risk-edit','risk',OP211_PRIOR.riskEditor],['control-editor','control',OP211_PRIOR.controlEditor]])window.CS_JOURNEYS[route]=()=>{const html=prior(),id=current.params.get('id');if(!recRead(kind,id)||!recInScope(kind,recRead(kind,id)))return html;return html.replace('<div id="wf29-error"',`<div id="op211-editor-changes">${op211EditorChanges(kind)}</div><div id="wf29-error"`);};
window.CS_JOURNEYS.risk=function(){
 const html=OP211_PRIOR.risk(),r=recRead('risk',current.params.get('id'));if(!r||!recInScope('risk',r)||!['','overview'].includes(current.params.get('tab')||''))return html;
 const cycles=scopeItems('cycles').filter(c=>c.risks.includes(r.id)),pending=cycles.filter(c=>c.reviews[r.id]?.status!=='Approved').sort((a,b)=>Number(b.reviews[r.id]?.status==='Submitted')-Number(a.reviews[r.id]?.status==='Submitted')||String(a.due).localeCompare(String(b.due)))[0],state=pending?.reviews[r.id]?.status||'Draft',review=state==='Submitted';
 const next=ui211Next({title:pending?review?'A rating proposal is ready for review':'Continue the current risk assessment':'Keep the risk context current',description:pending?`${r.title} retains its effective residual score of ${score(r.residual)} until an independent assessment is approved.`:'Update ownership, the causal statement or primary control; accepted ratings remain governed through RCSA.',owner:pending?(review?pending.reviewer:pending.owner):r.owner,due:pending?.due||'',state:pending?state:'',primary:pending?link(review?'Review proposal':'Continue assessment',ui211Route('rcsa-review',{id:pending.id,risk:r.id,mode:review?'review':null}),'primary'):link('Assessment history','risk?id='+r.id+'&tab=assurance','small'),secondary:btn('Edit risk context','wf29-risk-edit',`data-id="${e(r.id)}"`,'quiet small')});
 return html.replace(/<section class="panel pad mt16 wf29-next">[\s\S]*?<\/section>\s*$/,next);
};

// Live report chart drilldowns use IDs captured by the canonical read model.
function op211LossReportRoute(x,kind='',key=''){return ui211Route('loss-insights',{...x.config,inspect:kind,cohort:key,focus:kind?'#op211-loss-drill':null});}
function op211LossDrill(x,kind=current.params.get('inspect'),key=current.params.get('cohort')){
 const allowed={concentration:x.concentration,age:x.age,prospect:x.recoveryProspects.rows},rows=allowed[kind],group=rows?.find(row=>row.name===key);if(!group)return {valid:false,kind,key,records:[],label:'This source selection is no longer available in the displayed report.'};
 const ids=new Set(group.ids||[]);return {valid:true,kind,key,group,records:x.records.filter(record=>ids.has(record.id)),label:group.name};
}
function op211ChartLinks(html,x,rows,kind){for(const row of rows)html=html.replace('<div class="br-bar-label"><strong>'+e(row.name)+'</strong>',`<div class="br-bar-label"><a href="#/${e(op211LossReportRoute(x,kind,row.name))}" aria-label="${e('Inspect '+row.name+' source events')}">${e(row.name)}</a>`);return html;}
function op211LossDrillView(x){
 if(!current.params.get('inspect'))return '';const drill=op211LossDrill(x),clear=link('Clear source selection',op211LossReportRoute(x),'quiet small');
 const body=drill.valid?table(['Source loss / accountable owner','Occurrence / recognition','Gross','Unrecovered',''],drill.records.map(r=>({cells:[`<strong>${e(r.title)}</strong><small>${e(r.id)} · ${e(r.owner)} · ${e(r.branch)}</small>`,`${brDate(r.occurredOn)}<small>Recognised ${brDate(r.recognizedOn)}</small>`,lrpMoney(r.gross,x.currency),lrpMoney(r.unrecovered,x.currency),link('Open loss record','loss?id='+r.id,'small')]}))):notice(e(drill.label),'warning');
 return `<section id="op211-loss-drill" class="panel op211-report-drill" tabindex="-1"><div class="panel-head"><div><span class="eyebrow">Source drilldown · ${e(x.currency)}</span><h2>${e(drill.valid?drill.label:'Selection unavailable')}</h2><p>${drill.records.length} reviewed source events · ${e(x.basisLabel)} · ${date(x.config.from)}–${date(x.config.to)}</p></div><div class="row wrap">${clear}${drill.valid?btn('Export these events','op211-export-drill','','small'):''}</div></div>${body}<p class="op211-drill-note">This list is a subset of the displayed cohort. Board packs and the main cohort export retain the full report population.</p></section>`;
}
lrpBody=function(x,options={}){
 let html=OP211_PRIOR.lossBody(x,options);if(options.frozen)return html;
 html=html.replace(OP211_PRIOR.concentration(x,true),op211ChartLinks(OP211_PRIOR.concentration(x,true),x,x.concentration,'concentration'));
 html=html.replace(OP211_PRIOR.age(x),op211ChartLinks(OP211_PRIOR.age(x),x,x.age,'age'));
 html=html.replace(OP211_PRIOR.prospects(x),op211ChartLinks(OP211_PRIOR.prospects(x),x,x.recoveryProspects.rows,'prospect'));
 return html.replace(lrpContext(x,false),lrpContext(x,false)+op211LossDrillView(x));
};
actions['op211-export-drill']=()=>{const x=lrpCapture(),drill=op211LossDrill(x);requireValue(drill.valid,'Choose an available source group before exporting.');exportCSV('clearsight-loss-source-selection.csv',['snapshot','scope','currency','basis','window_from','window_to','selection','event_id','title','owner','occurrence','recognition','reporting_month','gross_minor','retained_recoveries_minor','unrecovered_minor'],drill.records.map(r=>[x.asOf,x.scopeLabel,x.currency,x.config.basis,x.config.from,x.config.to,drill.label,r.id,r.title,r.owner,r.occurredOn,r.recognizedOn,r.reportingPeriod,r.gross,r.recovered,r.unrecovered]));};

// Publication errors point to the exact draft field instead of leaving authors at Preview.
function op211ReportIssue(message,d=S.reportDraft){
 let step=1,selector='[data-ix-report-draft="name"]';
 if(/section/i.test(message))selector='[data-ix-report-section]';else if(/meeting/i.test(message))selector='[data-br-draft="meetingDate"]';else if(/coverage/i.test(message)){step=2;selector='[data-br-coverage]';}
 const match=message.match(/^Decision (\d+):/);if(match){step=2;const index=Number(match[1])-1,item=d?.decisions?.[index]||{},part=/owner/.test(message)?'owner':/target date/.test(message)?'due':/source ID/.test(message)?'sourceId':String(item.title||'').trim().length<5?'title':'rationale';selector=`[data-br-decision="${index}"][data-part="${part}"]`;}
 return {message,step,selector};
}
brShowDraftIssues=function(issues){const box=$('#ix-report-error');if(!box)return OP211_PRIOR.reportIssues(issues);box.innerHTML=`<section class="ui211-issues"><h3>Complete before publishing <span>${issues.length}</span></h3><ul>${issues.map(message=>{const item=op211ReportIssue(message);return `<li>${btn(e(message),'op211-report-field',`data-step="${item.step}" data-selector="${e(item.selector)}"`,'quiet')}</li>`;}).join('')}</ul></section>`;ui211Focus('#ix-report-error',$('#main-content'));};
actions['op211-report-field']=el=>{const d=brEnsureDraft(),step=Number(el.dataset.step);requireValue(d&&[1,2].includes(step),'The report draft is unavailable.');const allowed=brDraftIssues(d).map(message=>op211ReportIssue(message));requireValue(allowed.some(item=>item.step===step&&item.selector===el.dataset.selector),'This check no longer applies. Review the current report draft.');d.step=step;save();go(ui211Route('report-builder',{focus:el.dataset.selector}));};

// Existing input listeners have already written canonical draft state by this point.
function op211SetRegion(box,html){const previous=box.querySelector?.('details'),wasOpen=previous?.open;box.innerHTML=html;const next=box.querySelector?.('details');if(next&&typeof wasOpen==='boolean')next.open=wasOpen;}
function op211Refresh(event){
 const target=event.target,lossForm=target.closest?.('[data-l29-form="capture"]');
 if(lossForm){const d=S.lossRegister?.drafts[lossForm.dataset.id],box=$('#op211-loss-checks');if(d&&box&&d.createdBy===l29Actor().id)op211SetRegion(box,op211LossChecks(d,{...d.data,...values(lossForm)}));}
 if(current.name==='kri-return'&&target.matches?.('[data-k28-metric],[data-k28-return]')){const r=ixCurrentReturn();if(r&&k28Is(ixCycle(r.cycle))){const state=op211KriState(r),next=$('#op211-kri-next'),checks=$('#op211-kri-checks');if(next)next.innerHTML=op211KriNextContent(state);if(checks)checks.innerHTML=op211KriChecks(state);}}
 if(current.name==='rcsa-review'&&target.matches?.('[data-wf29-rcsa-field]')){const box=$('#op211-rcsa-compare'),state=op211RcsaState();if(box&&state)box.innerHTML=op211RcsaCompare(state);}
 if(current.name==='control-test'&&target.matches?.('[data-wf29-field]')){const c=recRead('control',current.params.get('id')),d=c&&wf29Draft('control-test',c.id),box=$('#op211-control-compare');if(c&&d&&box)box.innerHTML=op211ControlComparison(c,d.values,'Draft',d.baseDigest);}
 if(['risk-edit','control-editor'].includes(current.name)&&target.matches?.('[data-wf29-field],[data-wf29-array]')){const box=$('#op211-editor-changes');if(box)op211SetRegion(box,op211EditorChanges(current.name==='risk-edit'?'risk':'control'));}
}
document.addEventListener('input',op211Refresh);document.addEventListener('change',op211Refresh);
window.CLEAR_SIGHT_OPERATIONS_INTERACTIONS=Object.freeze({version:'2.11',kriState:op211KriState,lossIssues:op211LossIssues,lossFieldStep:op211LossStep,lossChanges:op211LossChanges,lossDrill:op211LossDrill,rcsaState:op211RcsaState});
