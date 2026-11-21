/* ClearSight 2.8. Source-linked branch returns and practical indicator workspaces.
 * One observation model: accepted snapshots feed existing reporting functions.
 * UI policy checks are demonstrations, not an authentication boundary.
 */
'use strict';
const K28_OLD={init:ixInit,value:ixMetricValue,band:ixBand,display:ixDisplay,aggregate:ixAggregate,stats:ixKriStats,returns:ixKriReturnsView,returnView:ixReturnView,issues:ixReturnIssues,submit:ixSubmitReturn,review:ixReviewReturn,cycleView:ixCycleView};
const k28Is=c=>c?.templateVersion==='2.8';
const k28DateOk=d=>typeof d==='string'&&/^\d{4}-\d{2}-\d{2}$/.test(d)&&Number.isFinite(Date.parse(d+'T00:00:00Z'))&&new Date(d+'T00:00:00Z').toISOString().slice(0,10)===d;
const k28Value=(m,v={})=>{
 if(v.missing||v.na)return null;
 if(m.type==='boolean')return typeof v.value==='boolean'?(v.value?1:0):null;
 if(m.type==='decimal')return typeof v.value==='number'&&Number.isFinite(v.value)&&v.value>=0?v.value:null;
 return K28_OLD.value(m,v);
};
ixMetricValue=k28Value;
ixBand=function(m,v={}){
 if(v.na)return 'Not applicable';
 const n=ixMetricValue(m,v);if(n===null)return 'Unknown';
 if(m.context)return 'Context only';
 if(m.type==='boolean')return v.value===m.adverse?'Breached':'Within limit';
 return K28_OLD.band(m,v);
};
ixDisplay=function(m,v={}){
 if(v.na)return 'N/A';if(ixMetricValue(m,v)===null)return '—';
 if(m.type==='boolean')return v.value?'Yes':'No';
 if(m.type==='currency')return ngn(v.value);
 return K28_OLD.display(m,v)+(m.type==='decimal'?' hours':'');
};
ixAggregate=function(c,m,units){
 if(!k28Is(c))return K28_OLD.aggregate(c,m,units);
 const accepted=units.map(unit=>({unit,value:ixAccepted(c,unit)?.values[m.id]}));
 const applicable=accepted.filter(x=>x.value?.na).length;
 const xs=accepted.filter(x=>ixMetricValue(m,x.value)!==null);
 const breaches=xs.filter(x=>ixBand(m,x.value)==='Breached').length;
 const affirmative=xs.filter(x=>x.value.value===true).length;
 const booleanCount=m.context?affirmative:breaches;
 const value=!xs.length?null:m.type==='boolean'?booleanCount:xs.reduce((n,x)=>n+ixMetricValue(m,x.value),0);
 return {value,numerator:m.type==='boolean'?booleanCount:null,denominator:m.type==='boolean'?xs.length:null,affirmative:m.type==='boolean'?affirmative:null,observed:xs.length,expected:units.length,notApplicable:applicable,applicableExpected:units.length-applicable,breaches,near:0,complete:xs.length+applicable===units.length&&units.length>0,unitIds:xs.map(x=>x.unit),aggregation:m.aggregation};
};
ixKriStats=function(c=ixCycle(),scope=selectedScope()){
 if(!k28Is(c))return K28_OLD.stats(c,scope);
 const units=ixExpected(c,scope),rs=units.map(u=>S.branchKri.returns[ixReturnKey(c.id,u)]),accepted=units.filter(u=>ixAccepted(c,u));
 const values=accepted.flatMap(unit=>c.metrics.map(m=>({unit,m,value:ixAccepted(c,unit).values[m.id]||{},band:ixBand(m,ixAccepted(c,unit).values[m.id])})));
 const known=values.filter(x=>!['Unknown','Not applicable'].includes(x.band)).length,notApplicable=values.filter(x=>x.band==='Not applicable').length,expectedValues=units.length*c.metrics.length;
 return {expected:units.length,approved:accepted.length,submitted:rs.filter(r=>r?.status==='Submitted').length,drafts:rs.filter(r=>r&&['Draft','Changes requested'].includes(r.status)).length,missing:rs.filter(r=>!r).length,known,notApplicable,completedValues:known+notApplicable,applicableExpected:expectedValues-notApplicable,expectedValues,breaches:values.filter(x=>x.band==='Breached').length,near:0,unknown:expectedValues-known-notApplicable,values};
};
function k28SeedValues(branch){
 const vals={};
 for(const m of CS_BRANCH_REFERENCE.metrics){
  const v={value:m.type==='boolean'?(m.context?false:!m.adverse):0,sourceRef:'SYNTHETIC-INSPECTION-OCT26',comment:''};
  if(m.id==='tellers')v.value=3;if(m.id==='printers')v.value=3;if(m.id==='civilGuards')v.value=6;if(m.id==='armedGuards')v.value=2;
  if(['rotorInstalled','rotorFunctional'].includes(m.id))v.value=2;
  if(m.id==='cardReplacement'){delete v.value;v.na=true;v.reason='The card printer is fully functional; replacement is not required.';}
  vals[m.id]=v;
 }
 const adverse=branch==='B-IK'?{cashShort:125000,cctv:false,faultyCameras:2,powerHours:3.5}:{fireDrill:false,inverter:false};
 for(const [id,value] of Object.entries(adverse)){
  vals[id].value=value;vals[id].comment='Synthetic branch inspection identified an exception requiring follow-up.';
  vals[id].detail={summary:id==='cashShort'?'Till shortage reconciled to two transactions.':id==='powerHours'?'Generator downtime interrupted branch operations.':'Inspection identified a control deficiency.',impact:'Branch operations affected; no additional loss confirmed.',owner:branch==='B-IK'?'Amina Bello':'Daniel Mensah',due:'2026-11-18',action:'Complete repair or reconciliation and attach the verification record.',status:'Open',evidence:'SYNTHETIC-INSPECTION-OCT26',occurredOn:'2026-10-29',count:id==='faultyCameras'?'2':'',reference:'',documentVersionId:''};
 }
 return vals;
}
ixInit=function(){
 K28_OLD.init();
 if(S.branchKri.referenceVersion==='2.8')return;
 const c={id:'KRC-BRANCH-2026-10',period:'2026-10',title:'Branch operational return · October 2026',population:['B-IK','B-EN'],metrics:clone(CS_BRANCH_REFERENCE.metrics),groups:clone(CS_BRANCH_REFERENCE.groups),definitionVersion:CS_BRANCH_REFERENCE.version,templateVersion:'2.8',reviewer:'Ifeoma Nwosu',owners:{'B-IK':'Amina Bello','B-EN':'Daniel Mensah'},programId:'',due:'2026-11-10',source:clone(CS_BRANCH_REFERENCE.source),hierarchy:{'B-IK':{directorate:'Lagos & South-West',region:'Lagos'},'B-EN':{directorate:'South',region:'South-East'}}};
 S.branchKri.cycles.unshift(c);
 for(const unit of c.population){
  const id=ixReturnKey(c.id,unit),r={id,cycle:c.id,unit,owner:c.owners[unit],status:unit==='B-IK'?'Approved':'Draft',values:k28SeedValues(unit),sourceRef:'SYNTHETIC-INSPECTION-OCT26',remarks:'Monthly branch inspection; exception owners and dates are recorded below.',revision:unit==='B-IK'?1:0,submissions:[],history:[],meta:clone(c.hierarchy[unit])};
  if(unit==='B-IK'){
   const snapshot={id:id+'::v1',revision:1,values:clone(r.values),sourceRef:r.sourceRef,remarks:r.remarks,meta:clone(r.meta),submittedBy:r.owner,submittedAt:'2026-11-08',definitionVersion:c.definitionVersion,unit,cycle:c.id};
   r.submissions=[snapshot];r.accepted={...clone(snapshot),reviewedBy:c.reviewer,reviewedAt:'2026-11-09',reviewNote:'Synthetic inspection reviewed. Follow-ups remain open.'};
   r.history=[{date:'2026-11-08',text:'Illustrative return submitted.',actor:r.owner},{date:'2026-11-09',text:'Illustrative return approved; exception follow-up remains open.',actor:c.reviewer}];
  }else{
   delete r.values.waterShortage.value;r.values.waterShortage.missing=true;r.values.waterShortage.reason='Facilities inspection log is still being collated.';
   r.history=[{date:'2026-11-08',text:'Working return prepared with synthetic answers.',actor:r.owner}];
  }
  S.branchKri.returns[id]=r;
  if(r.accepted)for(const m of c.metrics.filter(m=>ixBand(m,r.accepted.values[m.id])==='Breached')){
   const d=r.accepted.values[m.id].detail,key=r.id+'::'+m.id;
   if(!S.tasks.some(t=>t.kriObservation===key))S.tasks.unshift({id:'ACT-BRANCH-'+m.id,title:m.name+' · '+orgNode(unit).name,kind:'KRI remediation',owner:d.owner,ownerId:recUserId(d.owner),due:d.due,status:'Open',ref:r.id,orgScopeId:unit,kriObservation:key,sourceRevision:r.accepted.revision,description:d.action,route:ixReturnRoute(r.id)+'&section='+m.group});
  }
 }
 S.branchKri.referenceVersion='2.8';S.branchKri.referencePolicy=clone(CS_BRANCH_REFERENCE.metrics);S.insightCycle=c.id;save();
};
function k28Snapshot(r){return r.status==='Approved'?r.accepted:r.status==='Submitted'?r.submissions.at(-1):r;}
function k28Conditional(m,vals){return !m.condition||vals[m.condition.field]?.value===m.condition.equals;}
function k28Normalize(r){
 const c=ixCycle(r.cycle);
 for(const m of c.metrics){if(!m.condition)continue;const upstream=r.values[m.condition.field];
  const known=typeof upstream?.value==='boolean'&&!upstream.missing&&!upstream.na;
  if(!known||!k28Conditional(m,r.values)){
   if(r.values[m.id]&&!r.values[m.id].conditional){r.conditionalDrafts??={};r.conditionalDrafts[m.id]=clone(r.values[m.id]);}
   r.values[m.id]=known?{na:true,conditional:true,reason:'Not required because '+c.metrics.find(x=>x.id===m.condition.field).name.toLowerCase()+' is Yes.'}:{missing:true,conditional:'pending',reason:'Cannot assess until the card printer status is known.'};
  }else if(r.values[m.id]?.conditional){r.values[m.id]=clone(r.conditionalDrafts?.[m.id]||{});if(r.values[m.id].na)r.values[m.id]={};}
 }
}
function k28Issues(r,forSubmit=true){
 const c=ixCycle(r.cycle),issues=[];
 const add=(id,message)=>issues.push({id,message});
 if(forSubmit&&c.period>=S.asOf.slice(0,7))add('return','The monthly period has not ended. Save this as a draft.');
 for(const m of c.metrics){
  const v=r.values[m.id]||{};
  if(m.condition){
   const parent=r.values[m.condition.field]||{},known=typeof parent.value==='boolean'&&!parent.missing&&!parent.na;
   if(!known&&!v.missing)add(m.id,m.name+': keep this unavailable until the original printer status is known.');
   if(known&&parent.value===m.condition.equals&&v.na)add(m.id,m.name+': answer Yes or No because the original printer is faulty.');
   if(known&&parent.value!==m.condition.equals&&!v.na)add(m.id,m.name+': this follow-up is not applicable when the printer is functional.');
  }
  if(v.na||v.missing){if(!v.reason||v.reason.trim().length<10)add(m.id,m.name+': explain why this is '+(v.na?'not applicable.':'unavailable.'));continue;}
  if(ixMetricValue(m,v)===null){add(m.id,m.name+': enter a valid '+(m.type==='boolean'?'Yes or No answer.':m.type==='currency'?'NGN amount.':m.type==='decimal'?'number of hours.':'whole-number count.'));continue;}
  if(!(v.sourceRef||r.sourceRef||'').trim())add(m.id,m.name+': add a source reference or the common inspection reference.');
  if(ixBand(m,v)==='Breached'){
   const d=v.detail||{};
   if((d.summary||'').trim().length<10)add(m.id,m.name+': describe the exception.');
   if((d.impact||'').trim().length<5)add(m.id,m.name+': describe the operational or financial impact.');
   if(!CS_SEED.people.includes(d.owner))add(m.id,m.name+': assign an accountable owner.');
   if(!k28DateOk(d.due))add(m.id,m.name+': set a valid action due date.');
   if((d.action||'').trim().length<10)add(m.id,m.name+': record the action required.');
   if((d.evidence||v.sourceRef||r.sourceRef||'').trim().length<3)add(m.id,m.name+': link the inspection, incident or evidence reference.');
   if(d.occurredOn&&(!k28DateOk(d.occurredOn)||d.occurredOn>S.asOf))add(m.id,m.name+': the occurrence date is invalid or in the future.');
  }
 }
 const value=id=>ixMetricValue(c.metrics.find(m=>m.id===id),r.values[id]);
 if(value('rotorFunctional')!==null&&value('rotorInstalled')!==null&&value('rotorFunctional')>value('rotorInstalled'))add('rotorFunctional','Working rotor doors cannot exceed installed rotor doors.');
 if(value('cctv')!==null&&value('faultyCameras')!==null&&((value('cctv')===1&&value('faultyCameras')>0)||(value('cctv')===0&&value('faultyCameras')===0)))add('faultyCameras','CCTV status and the faulty-camera count conflict. Reconcile both answers.');
 if(value('armed24')===1&&value('armedGuards')===0)add('armed24','24-hour armed surveillance cannot be confirmed with zero armed guards.');
 if(forSubmit&&(r.remarks||'').trim().length<5)add('return','Add the CCO / CCM review remarks.');
 return issues;
}
ixReturnIssues=function(r){return k28Is(ixCycle(r.cycle))?k28Issues(r).map(x=>x.message):K28_OLD.issues(r);};
function k28Counts(c,vals){
 const all=c.metrics.map(m=>({m,v:vals[m.id]||{},band:ixBand(m,vals[m.id])}));
 return {all,complete:all.filter(x=>x.band!=='Unknown').length,exceptions:all.filter(x=>x.band==='Breached'),na:all.filter(x=>x.band==='Not applicable').length,missing:all.filter(x=>x.band==='Unknown').length};
}
function k28Tone(band){return band==='Breached'?'red':band==='Unknown'?'amber':band==='Within limit'?'green':'';}
function k28Badge(band){return pill(band==='Breached'?'Exception':band==='Within limit'?'No exception':band,k28Tone(band));}
function k28Nav(active){return tabLinks([['Branch returns','kri-returns','returns'],['Continuous indicators','indicators','indicators'],['Question library','kri-library','library'],['Branch insights','reports?view=kri','insights']],active);}
function k28Overview(){
 const c=ixCycle();if(!k28Is(c))return K28_OLD.returns();
 const stats=ixKriStats(c),expected=ixExpected(c);
 const rows=expected.map(unit=>{
  const r=S.branchKri.returns[ixReturnKey(c.id,unit)],snap=r?k28Snapshot(r):null,counts=k28Counts(c,snap?.values||{}),hier=c.hierarchy?.[unit]||{};
  return {cells:[`<strong>${e(orgNode(unit)?.name||unit)}</strong><small>${e(hier.directorate||'Directorate not assigned')} · ${e(hier.region||'Region not assigned')}</small>`,e(c.owners[unit]),pill(r?.status||'Not started'),`<strong>${counts.complete} / ${c.metrics.length}</strong><div class="k28-progress"><i style="width:${counts.complete/c.metrics.length*100}%"></i></div>`,`${counts.exceptions.length?pill(counts.exceptions.length+' exceptions','red'):pill('No recorded exceptions')}<small>${counts.missing} unknown · ${counts.na} N/A</small>`,`${date(c.due)}${c.due<S.asOf&&r?.status!=='Approved'?'<small class="cx-red">Overdue</small>':''}`,r?link(r.status==='Submitted'?'Review return':r.status==='Approved'?'Open return':'Continue return',ixReturnRoute(r.id),'small primary'):btn('Start return','ix-start-return',`data-cycle="${c.id}" data-unit="${unit}"`,'small primary')],attrs:`data-search="${e(orgName(unit)+' '+c.owners[unit]+' '+(r?.status||'not started'))}"`};
 });
 return header('Branch KRI returns','Collect the operational facts. Explain exceptions. Track accountable action.',btn(icon('plus')+' New reporting cycle','ix-new-cycle','','primary'),'Operational risk')+k28Nav('returns')+
 `<div class="k28-hero"><div><span class="eyebrow">${e(c.definitionVersion)} · ${c.metrics.length} source-linked questions</span><h2>Every exception needs an explanation and an owner.</h2><p>Cash, security, fire safety, facilities, customer service, conduct, environment and people — in one branch return.</p></div><div class="k28-hero-control">${ixCycleSelect()}${link('View branch insights','reports?view=kri','small')}</div></div>`+
 metricRow([metric('Accepted returns',`${stats.approved} <span class="muted small">/ ${stats.expected}</span>`,'Only accepted snapshots feed management reporting','check'),metric('Awaiting review',stats.submitted,'Submitted returns remain separate','work'),metric('Accepted exceptions',stats.breaches,`${stats.known} measured answers · ${stats.notApplicable} N/A`,'risk'),metric('Reporting gaps',stats.unknown,'Unapproved or unavailable answers','clock')])+
 tools('Find a branch, owner or return status')+panel('Reporting population','One accountable return per branch and month. Corrections retain the prior accepted version.',expected.length?table(['Branch / region','Accountable owner','State','Completeness','Current return findings','Due',''],rows):empty('No branches in this scope','Branch-wide returns are shown at branch or whole-bank scope. Select a branch in the header.'))+noMatch+
 `<div class="k28-bottom-grid">${panel('What happens after submission?','',`<ol class="k28-flow"><li><strong>Validate</strong><span>Typed answers, conditional questions and cross-checks identify contradictions.</span></li><li><strong>Review</strong><span>An independent reviewer accepts or requests specific corrections.</span></li><li><strong>Act</strong><span>Accepted exceptions create accountable actions and in-app notifications.</span></li></ol>`)}${panel('Question and policy provenance','',`<div class="pad"><p>Mapped from <strong>Branch KRI .xlsx</strong>, sheet “Branch KRI November ’25”, columns D–BB. The workbook supplies questions, not approved thresholds.</p><p>Exception rules shown here are illustrative and frozen per cycle.</p>${link('Inspect the question library','kri-library','small')}${btn('Export accepted answers','k28-export','','small')}</div>`)}</div>`;
}
function k28DetailFields(m,v,edit){
 const d=v.detail||{};
 if(!edit)return `<div class="k28-exception-read"><dl>${[['What happened',d.summary],['Impact',d.impact],['Accountable owner',d.owner],['Action due',d.due?date(d.due):'Not set'],['Required action',d.action],['Document / incident reference',d.evidence||v.sourceRef],['Occurrence date',d.occurredOn?date(d.occurredOn):'Not recorded'],['Additional context',d.context]].map(([a,b])=>`<div><dt>${a}</dt><dd>${e(b||'Not recorded')}</dd></div>`).join('')}</dl>${d.documentVersionId&&docVersion(d.documentVersionId)?link('Open linked evidence',`document?id=${e(docVersion(d.documentVersionId).documentId)}&version=${e(d.documentVersionId)}`,'small'):''}</div>`;
 const a=(field)=>`data-k28-metric="${m.id}" data-k28-detail="${field}"`;
 return `<div class="k28-exception-fields"><div class="k28-exception-title">${icon('risk')}<strong>Explain this exception</strong><span>Included in the review and board report</span></div><div class="field-grid">${area('summary','What happened?',d.summary||'',a('summary')+' placeholder="Affected asset, event or control failure"')}${area('impact','Operational / financial impact',d.impact||'',a('impact')+' placeholder="Service disruption, people affected or exposure"')}${select('owner','Accountable owner',[['','Assign owner'],...CS_SEED.people],d.owner||'',a('owner'))}${input('due','Action due date',d.due||'','date',a('due'))}<div class="full">${area('action','Required corrective action',d.action||'',a('action')+' placeholder="State the deliverable and how closure will be verified"')}</div>${input('evidence','Evidence / incident / service-ticket reference',d.evidence||'','text',a('evidence'))}${input('occurredOn','Occurrence or inspection date',d.occurredOn||'','date',a('occurredOn'))}<div class="full">${area('context','Additional context (where applicable)',d.context||'',a('context')+' placeholder="Affected assets, number of people, root cause, location, loss record or mitigation"')}</div></div><div class="row wrap">${btn(icon('file')+' Link document evidence','k28-link-evidence',`data-id="${m.id}"`,'small')}${d.documentVersionId?pill('Document version linked','blue'):''}<small>Keep personal and sensitive incident details in the restricted source record.</small></div></div>`;
}
function k28Question(m,vals,edit,idx){
 const v=vals[m.id]||{},band=ixBand(m,v),available=!(v.na||v.missing),attr=`data-k28-metric="${m.id}"`,normal=v.na?'na':v.missing?'missing':'reported';
 let inputHtml='';
 if(!edit)inputHtml=`<div class="k28-answer-read">${ixDisplay(m,v)}<small>${e(v.reason||'')}</small></div>`;
 else if(v.conditional)inputHtml=`<p class="k28-conditional">${icon('info')}${e(v.reason)}</p>`;
 else if(!available)inputHtml=`<label>Reason ${v.na?'not applicable':'unavailable'}<textarea ${attr} data-k28-field="reason" placeholder="Explain the reason; this is visible to the reviewer">${e(v.reason||'')}</textarea></label>`;
 else if(m.type==='boolean')inputHtml=`<div class="k28-boolean" role="group" aria-label="${e(m.name)}">${[[true,'Yes'],[false,'No']].map(([value,label])=>`<button type="button" data-action="k28-answer" data-id="${m.id}" data-value="${value}" class="${v.value===value?'selected':''}" aria-pressed="${v.value===value}">${label}</button>`).join('')}</div>`;
 else inputHtml=`<label class="k28-number"><span>${m.type==='currency'?'Amount (NGN)':m.type==='decimal'?'Hours':'Count'}</span><input type="number" min="0" step="${['currency','decimal'].includes(m.type)?'0.01':'1'}" ${attr} data-k28-field="value" aria-label="${e(m.name)}" value="${e(v.rawValue??(v.value===undefined?'':m.type==='currency'?v.value/100:v.value))}" placeholder="Enter a value"></label>`;
 return `<article class="k28-question ${band==='Breached'?'exception':''}" id="k28-${m.id}" data-k28-card="${m.id}"><div class="k28-question-top"><span class="k28-question-number">${String(idx+1).padStart(2,'0')}</span><div class="grow"><h3>${e(m.name)}</h3><small>${e(m.type==='currency'?'NGN · monthly value':m.periodBasis||'Monthly return')}${m.context?' · contextual information':''}</small></div><span data-k28-status>${k28Badge(band)}</span></div><div class="k28-answer-row">${inputHtml}${edit&&!v.conditional?`<label class="k28-availability">Availability<select ${attr} data-k28-field="availability" aria-label="${e(m.name)} availability">${options([['reported','Reported'],['missing','Unavailable'],['na','Not applicable']],normal)}</select></label>`:''}</div><details class="k28-question-help"><summary>Question, source and interpretation</summary><p>${e(m.sourceQuestion||m.help)}</p><small>Workbook column ${e(m.sourceColumn||'Custom')} · ${e(m.policyBasis||'Cycle-specific policy')}</small>${edit&&available?`<label>Observation source (optional if the common source applies)<input ${attr} data-k28-field="sourceRef" value="${e(v.sourceRef||'')}" placeholder="Inspection register / ticket / evidence reference"></label>`:`<p>${e(v.sourceRef||'Uses the common source reference')}</p>`}</details><div data-k28-exception-slot data-active="${band==='Breached'}">${band==='Breached'?k28DetailFields(m,v,edit):''}</div></article>`;
}
function k28RefreshAnswer(id){
 const r=ixCurrentReturn(),c=ixCycle(r.cycle),m=c.metrics.find(m=>m.id===id),card=$('[data-k28-card="'+id+'"]');if(!m||!card)return;
 const v=r.values[id]||{},band=ixBand(m,v),badge=$('[data-k28-status]',card),slot=$('[data-k28-exception-slot]',card);
 if(badge)badge.innerHTML=k28Badge(band);card.classList.toggle('exception',band==='Breached');
 if(slot&&slot.dataset.active!==String(band==='Breached')){slot.innerHTML=band==='Breached'?k28DetailFields(m,v,true):'';slot.dataset.active=String(band==='Breached');}
 const count=k28Counts(c,r.values),progress=$('.k28-section-nav .k28-return-progress');
 if(progress){const spans=progress.querySelectorAll(':scope > span');if(spans[0])spans[0].innerHTML='<strong>'+count.complete+' of '+c.metrics.length+'</strong> answered';if(spans[1])spans[1].textContent=count.exceptions.length+' exceptions · '+count.missing+' unavailable';const fill=progress.querySelector('.k28-progress i');if(fill)fill.style.width=count.complete/c.metrics.length*100+'%';}
}
function k28RenderReturn(){
 const r=ixCurrentReturn(),guard=ixReturnGuard(r);if(guard)return guard;const c=ixCycle(r.cycle);if(!k28Is(c))return K28_OLD.returnView();
 const snap=k28Snapshot(r),vals=snap.values,edit=['Draft','Changes requested'].includes(r.status),counts=k28Counts(c,vals),groups=c.groups||CS_BRANCH_REFERENCE.groups;
 const section=current.params.get('section')||groups[0].id,filter=current.params.get('filter')||'all';
 const visible=c.metrics.filter(m=>(section==='review'||m.group===section)&&(filter!=='exceptions'||ixBand(m,vals[m.id])==='Breached')&&(filter!=='gaps'||ixBand(m,vals[m.id])==='Unknown'));
 const base=ixReturnRoute(r.id),meta=snap.meta||r.meta||c.hierarchy?.[r.unit]||{},issues=edit?k28Issues(r,false):[];
 const progress=`<div class="k28-return-progress"><span><strong>${counts.complete} of ${c.metrics.length}</strong> answered</span><span>${counts.exceptions.length} exceptions · ${counts.missing} unavailable</span><div class="k28-progress"><i style="width:${counts.complete/c.metrics.length*100}%"></i></div></div>`;
 let decision='';
 if(r.status==='Submitted')decision=panel('Independent review','Review the submitted revision and its exception actions before deciding.',`<form data-ix-submit="kri-review" class="pad"><div class="field-grid">${select('reviewer','Reviewer',CS_SEED.people,c.reviewer)}${select('decision','Decision',['Approve return','Request changes'])}<div class="full">${area('note','Review rationale / corrections required','','required minlength="15"')}</div></div><div class="form-error" role="alert"></div><button class="btn primary" type="submit">Record review decision</button></form>`);
 else if(r.status==='Approved')decision=notice(`Accepted revision ${r.accepted.revision} · ${e(r.accepted.reviewedBy)} · ${date(r.accepted.reviewedAt)}. Exception actions can remain open after the return is accepted.`);
 const review=section==='review'?`<div class="k28-review-summary">${panel('Review the complete return','Cross-check answers and assess exceptions before submission.',`<div class="pad">${progress}${issues.length?notice(`${issues.length} items need attention before submission.`,'warning'):notice('All field and consistency checks pass. Confirm the facts and submit for independent review.')}${issues.length?`<ul class="k28-issue-list">${issues.map(i=>`<li><a href="#/${base}&section=${e(c.metrics.find(m=>m.id===i.id)?.group||'review')}">${e(i.message)}</a></li>`).join('')}</ul>`:''}</div>`)}</div>`:'';
 const context=edit?`<section class="panel k28-context"><div class="field-grid">${input('sourceRef','Common inspection / source reference',r.sourceRef||'','text','data-k28-return="sourceRef"')}${input('region','Region',meta.region||'','text','data-k28-meta="region"')}${input('directorate','Directorate',meta.directorate||'','text','data-k28-meta="directorate"')}<div class="full">${area('remarks','CCO / CCM remarks',r.remarks||'','data-k28-return="remarks" placeholder="Summarize branch conditions, material changes and unresolved matters"')}</div></div></section>`:`<div class="k28-read-context"><span><strong>Source</strong> ${e(snap.sourceRef||'Recorded per question')}</span><span><strong>CCO / CCM remarks</strong> ${e(snap.remarks||'Not recorded')}</span></div>`;
 return header(`${e(orgNode(r.unit)?.name||r.unit)} · Branch KRI`,`${ixMonthName(c.period)} · ${e(meta.directorate||'Directorate not set')} / ${e(meta.region||'Region not set')}`,link('All returns','kri-returns')+' '+(edit?btn('Save draft','ix-return-save')+' '+btn('Review & submit','k28-review','','primary'):r.status==='Approved'?btn('Start correction','k28-correction','','quiet'):'')+' '+btn('Export return','k28-export-return','','quiet'),'Branch operations')+
 `<div class="k28-return-meta">${pill(r.status)}<span>Owner <strong>${e(r.owner)}</strong></span><span>Due <strong>${date(c.due)}</strong></span><span>${e(c.definitionVersion)} · revision ${r.revision||'draft'}</span></div>${r.status==='Changes requested'?notice(e(r.history.at(-1)?.text||'Please review the requested corrections.'),'warning'):''}${r.accepted&&edit?notice('The last accepted revision remains in reporting while this correction is prepared.','mini'):''}<div id="ix-return-error" role="alert"></div>`+
 `<div class="k28-return-layout"><aside class="k28-section-nav">${progress}<nav aria-label="Return sections">${groups.map(g=>{const fields=c.metrics.filter(m=>m.group===g.id),done=fields.filter(m=>ixBand(m,vals[m.id])!=='Unknown').length,n=fields.filter(m=>ixBand(m,vals[m.id])==='Breached').length;return `<a class="${section===g.id?'active':''}" href="#/${base}&section=${g.id}" ${section===g.id?'aria-current="page"':''}><span>${e(g.name)}${n?`<em>${n}</em>`:''}</span><small>${done}/${fields.length}</small></a>`;}).join('')}<a class="${section==='review'?'active':''}" href="#/${base}&section=review"><span>${icon('check')} Review & submit</span></a></nav></aside><div class="k28-return-body">${section==='review'?context+review:section===groups[0].id?context:''}<div class="k28-section-heading"><h2>${section==='review'?'All observations':e(groups.find(g=>g.id===section)?.name||'Section')}</h2><div class="row">${[['all','All'],['exceptions','Exceptions'],['gaps','Gaps']].map(([id,label])=>link(label,base+'&section='+section+'&filter='+id,'small '+(filter===id?'primary':'quiet'))).join('')}</div></div>${visible.map(m=>k28Question(m,vals,edit,c.metrics.indexOf(m))).join('')||empty('No observations in this filter','Choose All to see the questions in this section.')}${section==='review'?decision+(edit?`<div class="k28-submit-bar"><p>Submit the current revision for independent review. Drafts do not change accepted reporting.</p>${btn('Submit for review','ix-return-submit','','primary')}</div>`:''):''}<div class="k28-section-footer">${link('Return overview','kri-returns','quiet')}${section!=='review'?link(section===groups.at(-1).id?'Review return':'Next section '+icon('arrow'),base+'&section='+(groups[groups.findIndex(g=>g.id===section)+1]?.id||'review'),'primary'):btn('Print return','print','','quiet')}</div>${panel('Version and review history','Accepted snapshots remain unchanged by a correction.',`<ul class="timeline">${r.history.slice().reverse().map(h=>`<li><strong>${e(h.text)}</strong><small>${date(h.date)} · ${e(h.actor)}</small></li>`).join('')}</ul>`)}</div></div>`;
}
function k28Emit(event){if(window.CS_V28_EVENTS?.emit)window.CS_V28_EVENTS.emit(event);}
function k28DocumentAllowed(d,unit){return d&&!d.archived&&recInScope('document',d,unit)&&(d.classification!=='Restricted'||d.ownerId===recActor().id);}
ixSubmitReturn=function(){
 const r=ixCurrentReturn();if(!r||!k28Is(ixCycle(r.cycle)))return K28_OLD.submit();
 requireValue(ixMatch(r.unit)&&['Draft','Changes requested'].includes(r.status),'This return is locked or outside the selected scope.');k28Normalize(r);
 const issues=k28Issues(r);if(issues.length){$('#ix-return-error').innerHTML=notice('<strong>Submission needs attention</strong><ul>'+issues.map(i=>'<li>'+e(i.message)+'</li>').join('')+'</ul>','warning');$('#ix-return-error').scrollIntoView({block:'start'});return;}
 openDialog('Submit this branch return?',`${orgName(r.unit)} · ${ixMonthName(ixCycle(r.cycle).period)}`,`<p>${k28Counts(ixCycle(r.cycle),r.values).exceptions.length} exception observations will be sent for independent review. The submitted revision will be locked.</p><label class="checkbox-row"><input type="checkbox" name="attest" required>I have reviewed the answers, sources and exception actions.</label>`,'Submit for review',form=>{
  requireValue(values(form).attest,'Confirm that you reviewed the return.');requireValue(['Draft','Changes requested'].includes(r.status),'This return has already changed.');requireValue(!k28Issues(r).length,'The return changed. Review its validation issues.');
  const c=ixCycle(r.cycle),revision=r.submissions.length+1;
  const materialized=clone(r.values);for(const v of Object.values(materialized))if(!v.sourceRef)v.sourceRef=r.sourceRef;
  r.submissions.push({id:r.id+'::v'+revision,revision,values:materialized,sourceRef:r.sourceRef,remarks:r.remarks,meta:clone(r.meta||{}),definitionVersion:c.definitionVersion,unit:r.unit,cycle:r.cycle,submittedBy:r.owner,submittedAt:S.asOf});r.revision=revision;r.status='Submitted';r.history.push({date:S.asOf,text:'Submitted revision '+revision+' for independent review.',actor:r.owner});audit('Branch KRI submitted',r.id,r.owner);
  k28Emit({id:'kri-submit-'+r.id+'-'+revision,type:'approval',title:'Branch return ready for review',body:orgName(r.unit)+' · '+ixMonthName(c.period),sourceRoute:ixReturnRoute(r.id)+'&section=review',sourceId:r.id,sourceRevision:revision,orgScopeId:r.unit,severity:'Normal',recipient:c.reviewer,actor:r.owner});go(ixReturnRoute(r.id)+'&section=review');
 });
};
const k28ReviewDecision=docWrapDecision(function(form){
 const r=ixCurrentReturn(),v=values(form),c=ixCycle(r.cycle),submitted=r.submissions?.at(-1);
 requireValue(r.status==='Submitted'&&ixMatch(r.unit),'This revision is no longer awaiting review.');
 requireValue(submitted&&submitted.values&&typeof submitted.values==='object'&&!Array.isArray(submitted.values),'The submitted snapshot is unavailable or malformed. Restore the source before review.');
 requireValue(submitted.id===r.id+'::v'+r.revision&&submitted.revision===r.revision&&submitted.cycle===r.cycle&&submitted.unit===r.unit&&submitted.definitionVersion===c.definitionVersion,'The submitted revision, location or definition does not match this return. Restore the source before review.');
 requireValue(CS_SEED.people.includes(submitted.submittedBy),'The submitted revision has no known contributor. Restore its provenance before review.');
 requireValue(CS_SEED.people.includes(v.reviewer)&&v.reviewer!==r.owner&&v.reviewer!==submitted.submittedBy,'Choose an independent reviewer who did not submit this return.');
 requireValue((v.note||'').trim().length>=15,'Record a substantive rationale or the corrections required.');requireValue(['Approve return','Request changes'].includes(v.decision),'Choose a valid decision.');
 if(v.decision==='Approve return'){
  const s=clone(submitted);requireValue(!k28Issues({...r,...s},false).length,'The submitted values contain unresolved consistency or required-field issues. Request changes.');
  for(const value of Object.values(s.values))if(value?.detail?.documentVersionId)requireValue(docVersion(value.detail.documentVersionId),'A pinned evidence version is unavailable. Restore that version or request a correction.');
  r.accepted={...s,reviewedBy:v.reviewer,reviewedAt:S.asOf,reviewNote:v.note};r.status='Approved';
  for(const m of c.metrics.filter(m=>ixBand(m,s.values[m.id])==='Breached')){
   const d=s.values[m.id].detail||{},key=r.id+'::'+m.id,episodes=S.tasks.filter(t=>t.kriObservation===key);
   // A correction updates its open action. Closed episodes remain immutable;
   // a newly accepted breach starts another episode with an explicit predecessor.
   let task=episodes.find(t=>t.status!=='Completed'&&t.status!=='Cancelled');
   const next={title:m.name+' · '+orgNode(r.unit).name,kind:'KRI remediation',owner:d.owner,ownerId:recUserId(d.owner),due:d.due,ref:r.id,orgScopeId:r.unit,kriObservation:key,sourceRevision:s.revision,description:d.action,route:ixReturnRoute(r.id)+'&section='+m.group};
   if(task){
    task.sourceHistory??=[];task.sourceHistory.push({at:S.asOf,actor:v.reviewer,reason:'Accepted corrected return revision '+s.revision,previous:{owner:task.owner,due:task.due,description:task.description,sourceRevision:task.sourceRevision},next:{owner:next.owner,due:next.due,description:next.description,sourceRevision:next.sourceRevision}});
    Object.assign(task,next);
   }else{
    const previous=episodes.slice().sort((a,b)=>(b.sourceRevision||0)-(a.sourceRevision||0))[0];
    task={...next,id:uid('ACT'),status:'Open',episode:episodes.length+1,previousActionId:previous?.id||'',sourceHistory:[{at:S.asOf,actor:v.reviewer,reason:'Created from accepted return revision '+s.revision}]};S.tasks.unshift(task);
   }
   k28Emit({id:'kri-exception-'+key+'-'+s.revision,type:'kri',title:m.name+' · '+orgNode(r.unit).name,body:d.summary+' Action: '+d.action,sourceRoute:ixReturnRoute(r.id)+'&section='+m.group,sourceId:r.id,sourceRevision:s.revision,actionId:task.id,sourceTaskId:task.id,kriObservation:key,actionable:true,orgScopeId:r.unit,severity:m.severity==='High'?'high':'medium',recipient:d.owner,actor:v.reviewer,due:d.due});
  }
  // An accepted correction is not evidence of action completion. Retain any
  // now-nonbreaching open action and flag that closure still needs verification.
  for(const task of S.tasks.filter(t=>t.kriObservation?.startsWith(r.id+'::')&&t.status!=='Completed'&&t.status!=='Cancelled')){
   const metricId=task.kriObservation.slice((r.id+'::').length),m=c.metrics.find(x=>x.id===metricId);
   if(m&&ixBand(m,s.values[metricId])!=='Breached')task.sourceObservationChanged={revision:s.revision,condition:ixBand(m,s.values[metricId]),at:S.asOf,note:'Accepted correction no longer records this breach. Independently verify the action before closure.'};
   else if(m)delete task.sourceObservationChanged;
  }
 }else r.status='Changes requested';
 r.history.push({date:S.asOf,text:v.decision+': '+v.note,actor:v.reviewer});audit('Branch KRI '+r.status,r.id,v.reviewer);
 k28Emit({id:'kri-review-'+r.id+'-'+r.revision+'-'+r.status,type:'program',title:'Branch return '+r.status.toLowerCase(),body:v.note,sourceRoute:ixReturnRoute(r.id)+'&section=review',sourceId:r.id,sourceRevision:r.revision,actionable:r.status==='Changes requested',sourceState:r.status,orgScopeId:r.unit,severity:r.status==='Changes requested'?'medium':'low',recipient:r.owner,actor:v.reviewer});render();toast('Independent review recorded.');
},()=>({type:'kriReturn',id:ixCurrentReturn()?.id}));
ixReviewReturn=function(form){
 const r=ixCurrentReturn();if(!r||!k28Is(ixCycle(r.cycle)))return K28_OLD.review(form);
 return k28ReviewDecision(form);
};
function k28Library(){
 const metrics=S.branchKri.referencePolicy||CS_BRANCH_REFERENCE.metrics;
 return header('Branch KRI question library','Source-linked definitions, conditional questions and explicit exception rules.',btn('Create reporting cycle','ix-new-cycle','','primary'),'KRI configuration')+k28Nav('library')+notice('The workbook provides 51 questions and no approved risk thresholds. The rules below are illustrative. Changes apply to new cycles; existing cycles retain their definitions.','mini')+tools('Search question, type or source column')+panel('Reporting definitions','Counts, NGN values, hours and Yes/No answers retain different meanings.',table(['Question / workbook column','Type','Interpretation','Follow-up',''],metrics.map(m=>({attrs:`data-search="${e(m.name+' '+m.type+' '+m.sourceColumn+' '+m.group)}"`,cells:[`<strong>${e(m.name)}</strong><small>${e(m.sourceColumn)} · ${e(CS_BRANCH_REFERENCE.groups.find(g=>g.id===m.group)?.name)}</small>`,e(m.type==='currency'?'NGN amount':m.type==='boolean'?'Yes / No':m.type==='decimal'?'Hours':'Count'),m.context?pill('Context only'):m.type==='boolean'?`Exception when <strong>${m.adverse?'Yes':'No'}</strong>`:`Exception above <strong>${m.type==='currency'?ngn(m.limit):m.limit}</strong>`,m.condition?'Only if card printer is faulty':'Owner, due date and action for exceptions',btn('Inspect rule','k28-policy',`data-id="${m.id}"`,'small')]}))))+noMatch;
}
function k28CycleView(){
 const d=S.branchKri.draftCycle;if(!d?.templateVersion)return K28_OLD.cycleView();const step=d.step;
 let body='';
 if(step===1)body=`<h2>Choose the period and deadline</h2><p>The full Branch KRI question set is pinned when the cycle is created.</p><div class="field-grid">${input('name','Cycle name',d.name,'text','data-k28-cycle="name"')}${input('period','Reporting month',d.period,'month','data-k28-cycle="period"')}${input('due','Submission due date',d.due,'date','data-k28-cycle="due"')}${select('programId','Program context (optional)',[['','Standalone branch return'],...S.programs.map(p=>[p.id,p.name])],d.programId,'data-k28-cycle="programId"')}</div><div class="k28-template-preview">${icon('form')}<div><strong>Branch operational KRI · 51 questions</strong><p>8 sections · typed answers · conditional follow-ups · independent review</p></div>${pill('BRANCH-OPS/2.0','blue')}</div>`;
 if(step===2)body=`<h2>Assign the reporting branches</h2><p>Each branch gets one return. Assign accountable owners separately from the reviewer.</p>${select('reviewer','Independent reviewer',CS_SEED.people,d.reviewer,'data-k28-cycle="reviewer"')}<div class="k28-population">${ORG.filter(n=>n.kind==='branch').map(n=>`<div class="k28-population-row"><label class="checkbox-row"><input type="checkbox" data-k28-population="${n.id}" ${d.population.includes(n.id)?'checked':''}>${e(n.name)}</label>${select('owner-'+n.id,'Return owner',CS_SEED.people,d.owners[n.id]||'Amina Bello',`data-k28-owner="${n.id}"`)}${input('region-'+n.id,'Region',d.hierarchy[n.id]?.region||'','text',`data-k28-hierarchy="region" data-unit="${n.id}"`)}${input('directorate-'+n.id,'Directorate',d.hierarchy[n.id]?.directorate||'','text',`data-k28-hierarchy="directorate" data-unit="${n.id}"`)}</div>`).join('')}</div>`;
 if(step===3)body=`<h2>Review the cycle</h2><div class="k28-cycle-review"><div><span>Reporting month</span><strong>${e(d.period)}</strong></div><div><span>Deadline</span><strong>${date(d.due)}</strong></div><div><span>Branches</span><strong>${d.population.length}</strong></div><div><span>Reviewer</span><strong>${e(d.reviewer)}</strong></div></div>${table(['Branch','Directorate / region','Owner'],d.population.map(id=>({cells:[e(orgName(id)),`${e(d.hierarchy[id]?.directorate||'Not set')} / ${e(d.hierarchy[id]?.region||'Not set')}`,e(d.owners[id])]})))}${notice('Creating a cycle creates expected return slots. It does not prefill values or approve any return.','mini')}`;
 return header('Create branch reporting cycle','Define the month, assign branch owners and confirm the question set.',link('Save & exit','kri-returns'),'KRI setup')+ixWizardSteps(['Period & definition','Branches & reviewers','Review'],step,'k28-cycle-step')+`<section class="panel ix-wizard"><div class="pad">${body}<div id="ix-cycle-error" role="alert"></div></div><footer>${step>1?btn('Back','k28-cycle-step',`data-step="${step-1}"`):btn('Discard cycle','k28-discard-cycle','','quiet')}<span>Step ${step} of 3</span>${step<3?btn('Continue','k28-cycle-step',`data-step="${step+1}"`,'primary'):btn('Create reporting cycle','k28-create-cycle','','primary')}</footer></section>`;
}
function k28CycleIssues(d,step=3){
 const issues=[];if((d.name||'').trim().length<3)issues.push('Name the reporting cycle.');if(!/^\d{4}-(0[1-9]|1[0-2])$/.test(d.period)||d.period>S.asOf.slice(0,7))issues.push('Choose a valid reporting month no later than the scenario month.');
 const endOfMonth=/^\d{4}-(0[1-9]|1[0-2])$/.test(d.period)?new Date(Date.UTC(Number(d.period.slice(0,4)),Number(d.period.slice(5,7)),0)).toISOString().slice(0,10):'';
 if(!k28DateOk(d.due)||!endOfMonth||d.due<=endOfMonth)issues.push('Set a valid submission deadline after the reporting month ends.');
 if(step>=2){if(!d.population.length)issues.push('Select at least one branch.');for(const id of d.population){if(orgNode(id)?.kind!=='branch')issues.push('Invalid reporting branch.');if(d.owners[id]===d.reviewer)issues.push(orgName(id)+': the reviewer cannot be the return owner.');if(!CS_SEED.people.includes(d.owners[id]))issues.push('Assign a known owner.');if(!d.hierarchy[id]?.directorate?.trim()||!d.hierarchy[id]?.region?.trim())issues.push(orgName(id)+': capture directorate and region.');}if(!CS_SEED.people.includes(d.reviewer))issues.push('Assign a known independent reviewer.');
  if(S.branchKri.cycles.some(c=>k28Is(c)&&c.period===d.period&&c.population.some(id=>d.population.includes(id))))issues.push('A branch return already exists for this period and definition family. Continue that cycle or choose another month.');
 }return issues;
}
function k28ExportReturn(r){
 const c=ixCycle(r.cycle),snapshot=k28Snapshot(r);exportCSV('branch-kri-'+r.unit+'-'+c.period+'.csv',['period','branch','revision','state','column','question','type','value','availability','source','exception','impact','owner','due','action','evidence','remarks'],c.metrics.map(m=>{const v=snapshot.values[m.id]||{},d=v.detail||{};return [c.period,orgName(r.unit),snapshot.revision||r.revision,r.status,m.sourceColumn,m.name,m.type,ixMetricValue(m,v)===null?'':m.type==='currency'?v.value/100:m.type==='boolean'?(v.value?'Yes':'No'):v.value,v.na?'Not applicable':v.missing?'Unavailable':ixMetricValue(m,v)===null?'Not answered':'Reported',v.sourceRef||snapshot.sourceRef,d.summary,d.impact,d.owner,d.due,d.action,d.evidence,snapshot.remarks];}));
}
const k28StartReturn=actions['ix-start-return'];
actions['ix-start-return']=el=>{
 const c=S.branchKri.cycles.find(c=>c.id===el.dataset.cycle);requireValue(c,'Reporting cycle not found.');
 k28StartReturn(el);const r=S.branchKri.returns[ixReturnKey(c.id,el.dataset.unit)];
 if(r&&k28Is(c)){r.meta??=clone(c.hierarchy?.[r.unit]||{});r.sourceRef??='';r.remarks??='';k28Normalize(r);save();}
};
Object.assign(actions,{
 'ix-return-submit':()=>ixSubmitReturn(),
 'k28-review':()=>go(ixReturnRoute(ixCurrentReturn().id)+'&section=review'),
 'k28-answer':el=>{const r=ixCurrentReturn();requireValue(r&&ixMatch(r.unit)&&['Draft','Changes requested'].includes(r.status),'This return is not editable.');r.values[el.dataset.id]??={};r.values[el.dataset.id].value=el.dataset.value==='true';delete r.values[el.dataset.id].rawValue;k28Normalize(r);save();const y=scrollY;render();scrollTo(0,y);},
 'k28-correction':()=>{const r=ixCurrentReturn();requireValue(r?.status==='Approved'&&ixMatch(r.unit),'Only an accepted in-scope return can be corrected.');openDialog('Start a correction?','The accepted revision remains in reports until the correction is reviewed.',area('reason','Reason for correction','','required minlength="10"'),'Start correction',form=>{const reason=values(form).reason.trim();requireValue(reason.length>=10,'Explain why the return requires correction.');r.values=clone(r.accepted.values);r.status='Draft';r.history.push({date:S.asOf,text:'Correction started: '+reason,actor:recActor().name});audit('Started KRI correction',r.id,recActor().name);render();});},
 'k28-link-evidence':el=>{const r=ixCurrentReturn();requireValue(r&&['Draft','Changes requested'].includes(r.status)&&ixMatch(r.unit),'Return is locked.');const docs=(S.governance?.documents||[]).filter(d=>k28DocumentAllowed(d,r.unit));openDialog('Link document evidence','The exact version is retained in this observation.',select('document','Document', [['','Choose a document'],...docs.map(d=>[d.id,d.title])],'','required'),'Link version',form=>{const d=docs.find(d=>d.id===values(form).document);requireValue(k28DocumentAllowed(d,r.unit),'Choose an accessible document.');const ver=docHead(d);requireValue(ver,'Document has no available version.');r.values[el.dataset.id]??={};r.values[el.dataset.id].detail??={};docLinkVersion(ver.id,{type:'kriReturn',id:r.id},'metric:'+el.dataset.id);r.values[el.dataset.id].detail.documentVersionId=ver.id;r.values[el.dataset.id].detail.evidence=d.title+' · v'+ver.number;save();render();});},
 'k28-policy':el=>{const m=S.branchKri.referencePolicy.find(m=>m.id===el.dataset.id);requireValue(m,'Definition not found.');openDialog(m.name,'Changes apply to future cycles only.',`<p>${e(m.sourceQuestion)}</p><p class="small">Source column ${e(m.sourceColumn)} · ${e(m.policyBasis)}</p>${m.context?notice('This is contextual information. No numeric appetite is inferred.','mini'):m.type==='boolean'?select('adverse','Exception when the answer is',[['true','Yes'],['false','No']],String(m.adverse)):input('limit',m.type==='currency'?'Exception above (NGN)':'Exception above ('+m.unit+')',m.type==='currency'?m.limit/100:m.limit,'number','min="0" step="'+(m.type==='currency'||m.type==='decimal'?'0.01':'1')+'" required')}${select('severity','Exception priority',['Medium','High'],m.severity)}${area('policyBasis','Policy basis / approval reference',m.policyBasis,'required minlength="10"')}`,'Save future-cycle rule',form=>{const v=values(form);requireValue(v.policyBasis.trim().length>=10,'Explain the policy basis.');if(!m.context){if(m.type==='boolean')m.adverse=v.adverse==='true';else{const n=m.type==='currency'?minorUnits(v.limit):Number(v.limit);requireValue(Number.isFinite(n)&&n>=0&&(m.type==='decimal'||Number.isSafeInteger(n)),'Enter a valid non-negative threshold.');m.limit=n;m.warning=n;}}m.severity=v.severity;m.policyBasis=v.policyBasis.trim();audit('Updated future branch KRI definition',m.id,recActor().name);render();});},
 'ix-new-cycle':()=>{if(!S.branchKri.draftCycle)S.branchKri.draftCycle={templateVersion:'2.8',name:'Branch operational return',period:S.asOf.slice(0,7),due:addDays(S.asOf,20),step:1,population:[],owners:Object.fromEntries(ORG.filter(n=>n.kind==='branch').map(n=>[n.id,n.id==='B-EN'?'Daniel Mensah':'Amina Bello'])),hierarchy:clone(ixCycle('KRC-BRANCH-2026-10').hierarchy||{}),reviewer:'Ifeoma Nwosu',programId:''};save();go('kri-cycle');},
 'k28-cycle-step':el=>{const d=S.branchKri.draftCycle,n=Number(el.dataset.step);requireValue(d&&[1,2,3].includes(n),'Invalid cycle step.');const issues=n>d.step?k28CycleIssues(d,n-1):[];if(issues.length){$('#ix-cycle-error').innerHTML=notice(issues.map(e).join('<br>'),'warning');return;}d.step=n;save();render();},
 'k28-discard-cycle':()=>openDialog('Discard this cycle draft?','Existing reporting cycles are unaffected.','<p>The uncreated cycle will be removed.</p>','Discard draft',()=>{S.branchKri.draftCycle=null;save();go('kri-returns');}),
 'k28-create-cycle':()=>{const d=S.branchKri.draftCycle,issues=k28CycleIssues(d);if(issues.length){$('#ix-cycle-error').innerHTML=notice(issues.map(e).join('<br>'),'warning');return;}const id=uid('KRC');S.branchKri.cycles.unshift({...clone(d),id,title:d.name+' · '+ixMonthName(d.period),metrics:clone(S.branchKri.referencePolicy),groups:clone(CS_BRANCH_REFERENCE.groups),definitionVersion:CS_BRANCH_REFERENCE.version,source:clone(CS_BRANCH_REFERENCE.source)});S.branchKri.draftCycle=null;S.insightCycle=id;audit('Created branch KRI reporting cycle',id,recActor().name);go('kri-returns');},
 'k28-export-return':()=>k28ExportReturn(ixCurrentReturn()),
 'k28-export':()=>{const c=ixCycle(),rows=ixExpected(c).flatMap(unit=>{const a=ixAccepted(c,unit);return a?c.metrics.map(m=>{const v=a.values[m.id]||{},d=v.detail||{};return [c.period,orgName(unit),a.revision,m.name,m.type,ixDisplay(m,v),ixBand(m,v),d.summary,d.impact,d.owner,d.due,d.action,a.reviewedBy];}):[[c.period,orgName(unit),'','Return not accepted','','','Unknown','','','','','','']];});exportCSV('clearsight-accepted-branch-kri.csv',['period','branch','revision','question','type','answer','condition','exception','impact','owner','due','action','reviewer'],rows);}
});
document.addEventListener('input',ev=>{
 const el=ev.target,r=ixCurrentReturn();if(el.matches('[data-k28-cycle]')&&S.branchKri.draftCycle){S.branchKri.draftCycle[el.dataset.k28Cycle]=el.value;save();return;}
 if(el.matches('[data-k28-hierarchy]')&&S.branchKri.draftCycle){const d=S.branchKri.draftCycle;d.hierarchy[el.dataset.unit]??={};d.hierarchy[el.dataset.unit][el.dataset.k28Hierarchy]=el.value;save();return;}
 if(!r||!['Draft','Changes requested'].includes(r.status)||!ixMatch(r.unit))return;
 if(el.matches('[data-k28-return]')){r[el.dataset.k28Return]=el.value;save();}
 if(el.matches('[data-k28-meta]')){r.meta??={};r.meta[el.dataset.k28Meta]=el.value;save();}
 if(el.matches('[data-k28-metric]')&&!el.matches('[data-k28-field="availability"]')){
  const m=ixCycle(r.cycle).metrics.find(m=>m.id===el.dataset.k28Metric);if(!m)return;const v=r.values[m.id]??={};
  if(el.dataset.k28Detail){v.detail??={};v.detail[el.dataset.k28Detail]=el.value;}
  else if(el.dataset.k28Field==='value'){
   const raw=el.value;v.rawValue=raw;delete v.value;
   if(raw.trim()){try{const n=m.type==='currency'?minorUnits(raw):Number(raw);if(Number.isFinite(n)&&n>=0&&(m.type==='decimal'||Number.isSafeInteger(n))){v.value=n;delete v.rawValue;}}catch(_) {}}
  }else v[el.dataset.k28Field]=el.value;
  save();
 }
});
document.addEventListener('change',ev=>{
 const el=ev.target,d=S.branchKri.draftCycle;
 if(el.matches('[data-k28-cycle]')&&d){d[el.dataset.k28Cycle]=el.value;save();}
 if(el.matches('[data-k28-owner]')&&d){d.owners[el.dataset.k28Owner]=el.value;save();}
 if(el.matches('[data-k28-population]')&&d){d.population=el.checked?[...new Set([...d.population,el.dataset.k28Population])]:d.population.filter(id=>id!==el.dataset.k28Population);save();}
 const r=ixCurrentReturn();if(!r||!['Draft','Changes requested'].includes(r.status)||!ixMatch(r.unit))return;
 if(el.matches('[data-k28-field="availability"]')){const v=r.values[el.dataset.k28Metric]??={};delete v.missing;delete v.na;if(el.value==='missing')v.missing=true;if(el.value==='na')v.na=true;k28Normalize(r);save();const y=scrollY;render();scrollTo(0,y);}
 else if(el.matches('[data-k28-field="value"]')){k28Normalize(r);k28RefreshAnswer(el.dataset.k28Metric);}
});
// Continuous indicators retain their original domain but gain a full investigation page.
const k28OldIndicators=indicatorsView;
indicatorsView=function(){return k28OldIndicators().replace(riskTabs('indicators'),k28Nav('indicators')).replace(/<strong>([^<]+)<\/strong><small>(IND-[^<]+)<\/small>/g,(full,title,meta)=>`<a href="#/indicator?id=${encodeURIComponent(meta.split(' · ')[0])}"><strong>${title}</strong></a><small>${meta}</small>`);};
function k28IndicatorView(){
 const x=S.indicators.find(i=>i.id===current.params.get('id'));if(!x||!scopeMatches(x,'indicators'))return header('Indicator unavailable','Choose an indicator within the selected scope.')+empty('Indicator not found in this scope','Return to the indicator register.',link('Indicators','indicators'));
 const r=S.risks.find(r=>r.id===x.risk),history=x.observations||[],vals=history.map(v=>v.value),max=Math.max(x.threshold||0,...vals,1);
 return header(e(x.title),`${e(x.id)} · ${e(x.type)} · ${e(orgName(scopeIds(x,'indicators')[0]))}`,link('Indicators','indicators')+' '+btn('Record observation','observe-indicator',`data-id="${x.id}"`,'primary'),'Indicator investigation')+metricRow([metric('Current observation',x.value===null?'—':num(x.value)+' '+e(x.unit),date(x.observed)),metric('Condition',pill(indicatorState(x)),'Freshness is assessed before the threshold'),metric('Appetite',`${x.direction==='above'?'≤':'≥'} ${x.threshold} ${e(x.unit)}`,'Warning at '+x.warning+' '+x.unit),metric('Owner',e(x.owner||r?.owner||'Unassigned'),r?e(r.title):'Risk link not set','users')])+panel('Observation history','Dated observations only; undated legacy samples are not plotted.',history.length?`<div class="k28-history-chart" role="img" aria-label="Dated indicator values">${history.map(v=>`<div><strong>${num(v.value)}</strong><i style="height:${Math.max(2,v.value/max*110)}px"></i><small>${date(v.date||v.observed)}</small></div>`).join('')}</div>${table(['Observation date','Value','Source / rationale'],history.map(v=>({cells:[date(v.date||v.observed),num(v.value)+' '+e(x.unit),e(v.sourceRef||v.note||'Not recorded')]})))}`:empty('No dated history','Record an observation to establish a traceable series.'))+panel('Response and accountability','Use the linked risk for treatment decisions and the inbox for follow-up.',`<div class="pad">${r?link('Open linked risk','risk?id='+r.id,'primary'):''} ${link('Review action queue','work')} ${link('Escalation policy','escalation-matrix')}</div>`);
}
const k28RenderBefore=render;
render=function(){k28RenderBefore();if(current.name==='indicators'){
 const cells=$$('tbody tr');for(const row of cells){const button=row.querySelector('[data-action="observe-indicator"]');if(button&&!row.querySelector('[data-k28-open]')){const a=document.createElement('a');a.className='btn quiet small';a.dataset.k28Open='';a.href='#/indicator?id='+encodeURIComponent(button.dataset.id);a.textContent='Investigate';button.parentElement.prepend(a);}}
 }};
Object.assign(window.CS_JOURNEYS,{'kri-returns':k28Overview,'kri-return':k28RenderReturn,'kri-cycle':k28CycleView,'kri-library':k28Library,'indicator':k28IndicatorView});
Object.assign(window.CS_JOURNEY_GROUPS,{'kri-library':'risks','indicator':'risks'});
Object.assign(titles,{'kri-library':'KRI question library','indicator':'Indicator investigation'});
window.CLEAR_SIGHT_KRI=Object.freeze({version:'2.8',reference:()=>clone(CS_BRANCH_REFERENCE),validate:id=>k28Issues(S.branchKri.returns[id]),getReturn:id=>clone(S.branchKri.returns[id]),getStats:()=>clone(ixKriStats()),getValues:id=>clone(k28Snapshot(S.branchKri.returns[id]).values)});
