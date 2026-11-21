/* Contextual copilot DESIGN PREVIEW. No model/API calls, no document-content
 * ingestion and no autonomous approval. All facts below are local read-model data.
 * Production must replace this adapter, not its domain command/authorization path. */
'use strict';
const CP={open:false,context:null,result:null,question:'',lastTask:'',notice:''};
function cpRef(type,record,detail='') {
  if(!record)return null;
  return {type,id:record.id,label:recName(type,record),route:REC_TYPES[type].route(record),detail};
}
function cpItem(text,refs=[],tone=''){return {text,refs:refs.filter(Boolean),tone};}
function cpSourceAdd(context,source) {
  if(!source)return;
  if(!context.sources.some(s=>s.type===source.type&&s.id===source.id&&s.route===source.route))context.sources.push(source);
}
function cpScopedSource(context,type,record,detail='') {
  if(!record||!recInScope(type,record))return null;
  const ref=cpRef(type,record,detail);cpSourceAdd(context,ref);return ref;
}
function cpTaskLabel(kind){return {summary:'Explain this view',gaps:'Find gaps',draft:'Draft follow-up'}[kind]||kind;}
function copilotContext() {
  const typeMap={document:'document',approval:'approval',program:'program',assessment:'program','requirement-review':'program',risk:'risk',loss:'loss',control:'control',vendor:'engagement',activity:'activity',dpia:'dpia','privacy-case':'privacyCase','kri-return':'kriReturn','form-response':'formResponse','report-pack':'report'};
  const key=current.name+'?'+current.params.toString(),type=typeMap[current.name],ctx={key,actor:recActor().id,scopeId:selectedScope(),scopeLabel:scopeLabel(),asOf:S.asOf,title:titles[current.name]||'Workspace',kind:'portfolio',facts:[],gaps:[],sources:[],actions:[],input:{},excluded:[],unavailable:false};
  const currentId=current.params.get('id');
  let record=type==='program'?prog():type==='formResponse'?faBinding():type==='approval'?S.governance.requests.find(x=>x.id===currentId):type?recRead(type,currentId):null;
  if(!record&&!currentId&&type==='risk')record=S.risks[0];
  if(!record&&!currentId&&type==='activity')record=jRecord('ROPA-01');
  if(!record&&!currentId&&type==='dpia')record=jDpia('DPIA-01');
  if(!record&&!currentId&&type==='privacyCase')record=recRead('privacyCase','PRIV-REQ-01');
  if(!record&&type==='kriReturn')record=ixCurrentReturn();
  if(!record&&type==='engagement'&&!currentId)record=S.engagements[0];
  if(current.blocked||S.scopeInvalid){ctx.unavailable=true;ctx.title='Context unavailable';ctx.hash=recHashText(recStable({key,scope:ctx.scopeId,blocked:true}));return ctx;}
  if(current.name==='documents'){
    const all=docRows(),included=all.filter(d=>d.classification!=='Restricted'),refs=included.map(d=>cpRef('document',d,docStatus(d)));
    ctx.kind='library';ctx.title='Document register';ctx.input={filters:clone(DOC_FILTERS),documents:included.map(d=>({id:d.id,title:d.title,version:d.currentVersionId,status:docStatus(d),owner:d.ownerId})),excludedCount:all.length-included.length};
    refs.forEach(ref=>cpSourceAdd(ctx,ref));
    const view={type:'view',id:'documents',label:'Filtered document register',route:'documents',detail:'Current location and filters'};cpSourceAdd(ctx,view);
    ctx.facts.push(cpItem(`${included.length} non-restricted documents match this view. ${included.filter(d=>docStatus(d)==='In review').length} are in review.`,[view]));
    for(const d of included){if(docHead(d)?.storage==='reference')ctx.gaps.push(cpItem(`${d.title}: the original file has not been supplied.`,[refs.find(r=>r.id===d.id)],'warning'));}
    if(all.length!==included.length)ctx.excluded.push(`${all.length-included.length} restricted documents were omitted from this context.`);
    ctx.excluded.push('Document contents are not read. Filters match the explorer; document approval is not evidence acceptance.');
  }else if(current.name==='form-builder'){
    const f=faForm(),ref={type:'form',id:f.id,label:f.name,route:'form-builder?form='+f.id,detail:`Draft revision ${f.revision}`},checks=faDefinitionErrors(f);
    ctx.kind='builder';ctx.title=f.name;ctx.input={id:f.id,revision:f.revision,status:f.status,fields:clone(f.fields),name:f.name};cpSourceAdd(ctx,ref);
    ctx.facts.push(cpItem(`${f.fields.length} fields: ${f.fields.filter(q=>q.mode==='integration').length} source-backed and ${f.fields.filter(q=>['combo_grid','repeat','matrix','multi_matrix'].includes(q.type)).length} grids or matrices.`,[ref]),cpItem('Published assignments retain their version. Changes here do not rewrite submitted assessments.',[ref]));
    checks.forEach(x=>ctx.gaps.push(cpItem(x,[ref],'warning')));
    for(const q of f.fields)if(q.mode==='integration'&&q.check?.assurance==='Heuristic only')ctx.gaps.push(cpItem(`${q.label}: this heuristic cannot establish the full compliance objective.`,[ref],'warning'));
    ctx.actions.push({label:'Preview respondent view',route:'form-response?form='+f.id});ctx.excluded.push('No questions, mappings or check settings are changed automatically.');
  }else if(type==='kriReturn'&&record){
    const cycle=ixCycle(record.cycle),snap=record.status==='Approved'?record.accepted:record.status==='Submitted'?record.submissions.at(-1):null,values=snap?.values||record.values,ref=cpRef('kriReturn',record,ixMonthName(cycle.period));
    ctx.kind='kri';ctx.title=recName('kriReturn',record);ctx.input={id:record.id,status:record.status,values:clone(values),definition:clone(cycle.metrics),accepted:record.accepted?.id};cpSourceAdd(ctx,ref);
    ctx.facts.push(cpItem(`${record.status} for ${ixMonthName(cycle.period)}. ${record.accepted?'An accepted snapshot contributes to management reporting.':'These values do not yet contribute to accepted reporting.'}`,[ref]));
    for(const m of cycle.metrics){const val=values[m.id],band=ixBand(m,val);ctx.facts.push(cpItem(`${m.name}: ${ixDisplay(m,val)}${m.type==='ratio'?'':' '+m.unit} · ${band.toLowerCase()}.`,[ref]));if(['Breached','Unknown'].includes(band))ctx.gaps.push(cpItem(`${m.name}: ${band==='Unknown'?'the observation is unavailable; it must not be treated as zero':'the unit threshold is breached; explain and assign follow-up'}.`,[ref],'warning'));}
    if(['Draft','Changes requested'].includes(record.status))ixReturnIssues(record).forEach(x=>ctx.gaps.push(cpItem(x,[ref],'warning')));
    ctx.excluded.push('Unit thresholds are not applied to bank-wide sums. No return is submitted or approved by Copilot.');
  }else if(type==='engagement'&&record){
    const ref=cpRef('engagement',record,orgName(record.orgScopeId)),programs=S.programs.filter(p=>p.targets.includes(record.id));ctx.kind='vendor';ctx.title=recName('engagement',record);ctx.input={id:record.id,status:record.status,criticality:record.criticality,renewal:record.renewal,owner:record.owner,programs:programs.map(p=>({id:p.id,posture:posture(p,record.id)}))};cpSourceAdd(ctx,ref);
    ctx.facts.push(cpItem(`${record.criticality} engagement · ${record.status.toLowerCase()}. Accountable owner: ${record.owner||'Unassigned'}.`,[ref]),cpItem(`${programs.length} programs assess this engagement. Other services from the same supplier are separate targets.`,[ref]));
    if(!programs.length)ctx.gaps.push(cpItem('No assessment program is assigned to this engagement. Review the required assurance scope.',[ref],'warning'));
    if(record.renewal&&record.renewal<S.asOf)ctx.gaps.push(cpItem('The recorded renewal date has passed. Verify contract status before taking action.',[ref],'warning'));
    programs.forEach(p=>{const ps=posture(p,record.id),source=cpRef('program',p,record.service);source.route+=`&target=${record.id}`;cpSourceAdd(ctx,source);ctx.facts.push(cpItem(`${p.name}: ${ps.reviewed}/${ps.all} applicable requirements reviewed for this engagement.`,[source]));});
    ctx.actions.push({label:'Open engagement assessments',route:'vendor?id='+record.id+'&tab=assessments'});
  }else if(current.name==='work'||current.name==='approvals'){
    const items=workItems().filter(x=>x.status==='Open'),ref={type:'view',id:'work',label:'Scoped work queue',route:'work',detail:ctx.scopeLabel};ctx.kind='work';ctx.title='Work and approvals';ctx.input={items:items.map(x=>({id:x.id,title:x.title,kind:x.kind,due:x.due,owner:x.owner,route:x.route}))};cpSourceAdd(ctx,ref);
    ctx.facts.push(cpItem(`${items.length} open items; ${items.filter(uxIsReviewItem).length} are reviews or decisions.`,[ref]));
    items.filter(x=>x.due&&x.due<S.asOf).forEach(x=>{const source={type:'work',id:x.id,label:x.title,route:x.route||'work',detail:'Due '+date(x.due)};cpSourceAdd(ctx,source);ctx.gaps.push(cpItem(`${x.title} is overdue. Owner: ${x.owner||'Unassigned'}.`,[source],'warning'));});
    ctx.excluded.push('Demo identities are not authenticated. Work remains in the source workflow; Copilot cannot complete a review.');
  }else if(type==='report'&&record){
    const snap=record.snapshot;ctx.kind='report';ctx.title=record.name;ctx.scopeLabel=snap.scope.label+' · published snapshot';ctx.input={reportId:record.id,snapshot:snap,sections:record.sections,commentary:record.commentary};
    const ref=cpRef('report',record,'Published source snapshot');cpSourceAdd(ctx,ref);
    ctx.facts.push(cpItem(`${snap.risk.aboveAppetite} of ${snap.risk.total} risks were above appetite at the captured date.`,[ref]),cpItem(`${ngn(snap.losses.net)} recorded net loss; ${snap.losses.events} actual loss events in the captured occurrence window.`,[ref]),cpItem(`${snap.kri.approved}/${snap.kri.expected} branch returns were approved for ${ixMonthName(snap.periods.kri.month)}.`,[ref]));
    ctx.gaps.push(cpItem('This is a historical pack. Later approvals and navigation changes must not rewrite its figures.',[ref]));
    ctx.excluded.push('Current live records are not substituted for the captured source snapshot.');
  }else if(type==='document'&&record||type==='approval'&&record){
    const request=type==='approval'?record:null,doc=request?.subject.type==='document'?recRead('document',request.subject.id):type==='document'?record:null;
    ctx.kind='document';ctx.title=request?.title||doc?.title||'Approval';
    if(doc?.classification==='Restricted'){
      ctx.input={id:doc.id,classification:'Restricted'};ctx.excluded.push('Restricted documents are excluded from this copilot preview, including their titles and contents.');ctx.title='Restricted document';
      ctx.facts.push(cpItem('No file information was supplied to the copilot. Use the document workspace for authorized review.'));
    }else if(doc){
      const selectedVersion=current.params.get('version'),v=request?docVersion(request.versionId):docVersion(selectedVersion)||docHead(doc),ref=cpRef('document',doc,`${docVersionLabel(v)} · ${docVersionStatus(v)}`);
      ref.route='document?id='+doc.id+'&version='+v.id;cpSourceAdd(ctx,ref);
      ctx.input={document:{id:doc.id,title:doc.title,classification:doc.classification,currentVersionId:doc.currentVersionId,archived:doc.archived},version:{id:v.id,number:v.number,fileName:v.fileName,size:v.size,mime:v.mime,sha256:v.sha256,storage:v.storage,uploadedBy:v.uploadedBy},request:request?{id:request.id,status:request.status,steps:clone(request.steps),snapshotHash:request.snapshotHash}:null};
      ctx.facts.push(cpItem(`${docVersionLabel(v)} is ${docVersionStatus(v).toLowerCase()}. Evidence acceptance and requirement satisfaction remain separate decisions.`,[ref]),cpItem(`The version has ${S.governance.links.filter(l=>l.documentVersionId===v.id).length} pinned record links.`,[ref]));
      if(v.storage==='reference')ctx.gaps.push(cpItem('Only a reference is present. The original file was not supplied.',[ref],'warning'));
      else ctx.facts.push(cpItem('A stored version and checksum are recorded. This preview has not read its content or verified malware status.',[ref]));
      if(v.id!==doc.currentVersionId)ctx.gaps.push(cpItem('A newer version exists; the selected record still refers to this exact historical version.',[ref],'warning'));
      if(request){const next=approvalNext(request);ctx.facts.push(cpItem(`Request state: ${request.status}. ${next?'Next: '+recUser(next.actorId)?.name+' ('+next.kind+').':'No pending review step.'}`,[ref]));}
      ctx.actions.push({label:'Open exact version',route:ref.route});ctx.excluded.push('File contents, attachments and secrets are not read or sent to a model.');
    }else{
      ctx.input={request:clone(record)};ctx.facts.push(cpItem('Review the captured report snapshot in the existing approval workspace. The copilot cannot approve or sign it.'));
    }
  }else if(type==='program'&&record){
    const t=selectedTarget(record),p=posture(record,t),ref=cpRef('program',record,'Target: '+targetName(t));ref.route=`program?id=${record.id}&target=${encodeURIComponent(t)}`;cpSourceAdd(ctx,ref);
    ctx.kind='program';ctx.title=record.name;ctx.input={programId:record.id,target:t,posture:p,evaluations:record.evaluations?.[t],excluded:record.excluded,due:record.due};
    ctx.facts.push(cpItem(`${p.reviewed}/${p.all} applicable requirements reviewed for ${targetName(t)}.`,[ref]),cpItem(`${p.pass} satisfied, ${p.partial} partially satisfied and ${p.failed} not satisfied.`,[ref]));
    for(const rid of activeReqs(record)){
      const result=evaluation(record,t,rid),req=findR(rid);if(result.status==='Satisfied')continue;
      const source={type:'requirement',id:record.id+'::'+t+'::'+rid,label:req.code+' · '+req.title,route:`requirement-review?id=${record.id}&target=${encodeURIComponent(t)}&requirement=${rid}`,detail:result.status};cpSourceAdd(ctx,source);
      ctx.gaps.push(cpItem(`${req.code}: ${result.status.toLowerCase()}. Review the objective and supporting evidence.`,[source],result.status==='Not satisfied'?'warning':''));
    }
    ctx.actions.push({label:'Open requirement register',route:ref.route+'&tab=requirements'});
  }else if(type==='risk'&&record){
    const ref=cpRef('risk',record,orgName(record.orgScopeId));cpSourceAdd(ctx,ref);const control=recRead('control',record.control),ls=S.losses.filter(l=>l.risk===record.id),totals=scopeTotals(ls);
    ctx.kind='risk';ctx.title=record.title;ctx.input={risk:clone(record),control:control?clone(control):null,losses:clone(ls)};
    ctx.facts.push(cpItem(`Effective residual rating ${score(record.residual)}/25; appetite ${record.appetite}. ${score(record.residual)>record.appetite?'The rating exceeds appetite.':'The rating is within appetite.'}`,[ref]),cpItem(`${ngn(totals.net)} lifetime recorded net loss across ${totals.actual} actual events. This is not the dashboard loss-period total.`,[ref]));
    if(control){const c=cpScopedSource(ctx,'control',control,control.effectiveness);ctx.facts.push(cpItem(`Linked control: ${control.title}. Recorded effectiveness: ${control.effectiveness}.`,[c]));if(control.effectiveness!=='Effective')ctx.gaps.push(cpItem('The linked control is not recorded as fully effective; review its evidence and treatment.',[c],'warning'));}
    if(score(record.residual)>record.appetite)ctx.gaps.push(cpItem('An appetite exception requires an accountable treatment decision, not an automatic risk-score adjustment.',[ref],'warning'));
    ctx.actions.push({label:'Open controls and assurance',route:'risk?id='+record.id+'&tab=assurance'});
  }else if(type==='formResponse'){
    const b=faBinding(),res=faResponse(),def=faDefinition(),ref={type:'formResponse',id:faResponseKey(),label:def.name,route:'form-response?'+current.params.toString(),detail:b?'Assigned assessment':'Designer preview'};cpSourceAdd(ctx,ref);
    ctx.kind='assessment';ctx.title=def.name;const errors=faErrors(def,res);ctx.input={binding:b?clone(b):null,status:res.status,definitionId:def.id,answers:clone(res.answers),observations:clone(res.observations),note:res.note};
    ctx.facts.push(cpItem(`${res.status}; ${b?'assigned to '+targetName(b.target):'unassigned designer preview, not compliance evidence'}.`,[ref]));
    for(const error of errors)ctx.gaps.push(cpItem(error,[ref],'warning'));
    const observations=Object.values(res.observations||{});ctx.facts.push(cpItem(`${observations.filter(o=>o.result==='Pass').length} passing, ${observations.filter(o=>o.result==='Fail').length} failing and ${observations.filter(o=>o.result==='Unknown').length} unknown technical observations.`,[ref]));
    ctx.excluded.push('No new checks were executed. A passing check is not a requirement conclusion.');
  }else if(['activity','dpia','privacyCase'].includes(type)&&record){
    const ref=cpRef(type,record,'Recorded state: '+record.state);cpSourceAdd(ctx,ref);ctx.kind='privacy';ctx.title=type==='privacyCase'?'Privacy case '+record.id:recName(type,record);
    const data=type==='privacyCase'?record:jData(record);ctx.input={id:record.id,state:record.state,revision:record.revision,owner:record.owner,reviewer:record.reviewer,nextReview:data.nextReview};
    ctx.facts.push(cpItem(`State: ${record.state}. Publication and processing legality are different decisions.`,[ref]));
    if(type==='activity'){const issues=jAllIssues(data);ctx.input.issues=issues;for(const issue of issues)ctx.gaps.push(cpItem(issue.message,[ref],'warning'));if(data.nextReview&&data.nextReview<S.asOf)ctx.gaps.push(cpItem('The recorded review date has passed.',[ref],'warning'));}
    else if(record.state==='In review'||record.state==='Awaiting review')ctx.gaps.push(cpItem('An independent decision is pending. Copilot cannot supply that decision or sign on behalf of a reviewer.',[ref]));
    ctx.excluded.push('No personal data or case attachments were read. No legal basis, regulatory deadline or DPIA approval is inferred.');
  }else{
    const report=reportData(),summary=ixSummary(),refs={risk:{type:'view',id:'risk-register',label:'Scoped risk register',route:'risks',detail:ctx.scopeLabel},loss:{type:'view',id:'loss-ledger',label:'Loss ledger',route:'losses?period='+scopePeriod(),detail:`${report.periods.loss.from}–${report.periods.loss.to}`},kri:{type:'view',id:'kri-returns',label:'Monthly branch returns',route:'kri-returns',detail:ixMonthName(ixCycle().period)},work:{type:'view',id:'work',label:'Work and approvals',route:'work',detail:ctx.scopeLabel}};
    Object.values(refs).forEach(ref=>cpSourceAdd(ctx,ref));
    ctx.input={scope:clone(S.scope),periods:report.periods,risk:report.risk,losses:report.losses,kri:report.kri,actions:report.actions,risks:summary.risks.map(r=>({id:r.id,residual:r.residual,appetite:r.appetite})),indicators:summary.ind.map(i=>({id:i.id,value:i.value,observed:i.observed,freshDays:i.freshDays})),tasks:summary.tasks.map(t=>({id:t.id,due:t.due,status:t.status}))};
    ctx.facts.push(cpItem(`${summary.above.length}/${summary.risks.length} current risks exceed appetite.`,[refs.risk]),cpItem(`${ngn(summary.loss.net)} recorded net loss (${ngn(summary.loss.gross)} gross less ${ngn(summary.loss.recovered)} recovered) in the ${scopePeriod()}-day occurrence window.`,[refs.loss]),cpItem(`${summary.kri.approved}/${summary.kri.expected} branch returns approved for ${ixMonthName(ixCycle().period)}; ${summary.kri.known}/${summary.kri.expectedValues} values known.`,[refs.kri]));
    for(const risk of summary.above){const ref=cpScopedSource(ctx,'risk',risk,'Above appetite');ctx.gaps.push(cpItem(`${risk.title}: residual ${score(risk.residual)} exceeds appetite ${risk.appetite}.`,[ref],'warning'));}
    if(summary.kri.approved<summary.kri.expected)ctx.gaps.push(cpItem(`${summary.kri.expected-summary.kri.approved} expected unit returns are not yet approved. Missing reporting is not a zero result.`,[refs.kri],'warning'));
    if(summary.overdue.length)ctx.gaps.push(cpItem(`${summary.overdue.length} recorded action${summary.overdue.length===1?' is':'s are'} overdue.`,[refs.work],'warning'));
    const uncertain=summary.ind.filter(i=>['Unknown','Needs refresh'].includes(indicatorState(i)));if(uncertain.length)ctx.gaps.push(cpItem(`${uncertain.length} continuous indicators have unknown or stale readings.`,[{type:'view',id:'indicators',label:'Continuous indicators',route:'indicators',detail:'Freshness and current readings'}],'warning'));
    ctx.excluded.push('No credit, market or liquidity feed is connected. Loss-collection completeness is not independently established.');
    ctx.actions.push({label:'Open source records',route:'reports?view=quality'});
  }
  // Text from records is evidence, never executable instructions. Content is escaped at render.
  for(const item of [...ctx.facts,...ctx.gaps])for(const ref of item.refs)cpSourceAdd(ctx,ref);
  ctx.hash=recHashText(recStable({key,actor:ctx.actor,scope:S.scope,asOf:S.asOf,input:ctx.input,facts:ctx.facts,gaps:ctx.gaps}));
  return ctx;
}
function copilotResult(task,question='') {
  const c=copilotContext();if(c.unavailable)return {task,heading:'No accessible context',items:[],sources:[],contextHash:c.hash};
  if(/\b(approve|sign off|sign this|delete|send email|publish|mark.*compliant)\b/i.test(question))return {task:'blocked',heading:'A human decision is required',items:[cpItem('Copilot does not approve, sign, delete, send or publish records. Open the assigned source workflow and review the exact version.')],sources:[],contextHash:c.hash};
  if(task==='summary')return {task,heading:'What the records show',items:c.facts,sources:c.sources,contextHash:c.hash};
  if(task==='gaps')return {task,heading:'Items to review',items:c.gaps.length?c.gaps:[cpItem('No additional gap was identified by this preview’s limited checks. This is not an assurance conclusion.')],sources:c.sources,contextHash:c.hash};
  if(task==='draft'){
    const ctx=recContext(),source=ctx&&['risk','control','program','engagement','loss','activity','dpia','evidence','kriReturn'].includes(ctx.type)?{type:ctx.type,id:ctx.id}:null;
    const primary=c.gaps[0]?.text||'Review the source records and document the outstanding decision.';
    return {task,heading:'Proposed follow-up',items:[cpItem(primary,c.gaps[0]?.refs||[])],sources:c.sources,contextHash:c.hash,proposal:{title:('Review '+(c.kind==='portfolio'?scopeLabel():c.title)).slice(0,180),description:primary,source,contextHash:c.hash,sources:c.sources.map(s=>({type:s.type,id:s.id,route:s.route}))}};
  }
  return {task:'unsupported',heading:'Choose a supported task',items:[cpItem('This local preview can explain the displayed records, identify configured gaps or draft a follow-up. It has no connected language model and does not search uploaded file contents.')],sources:[],contextHash:c.hash};
}
function cpSuggestionLabels(c) {
  if(c.kind==='builder')return [['summary','Explain this form'],['gaps','Check design readiness'],['draft','Draft follow-up']];
  if(c.kind==='kri')return [['summary','Explain this return'],['gaps','Check reporting gaps'],['draft','Draft follow-up']];
  if(c.kind==='library')return [['summary','Summarise this register'],['gaps','Find missing originals'],['draft','Draft follow-up']];
  if(c.kind==='document')return [['summary','Explain this version'],['gaps','Check review readiness'],['draft','Draft follow-up']];
  if(c.kind==='program')return [['summary','Summarise requirements'],['gaps','Find evidence gaps'],['draft','Draft follow-up']];
  if(c.kind==='assessment')return [['summary','Explain these results'],['gaps','Check submission readiness'],['draft','Draft follow-up']];
  if(c.kind==='privacy')return [['summary','Explain this record'],['gaps','Check review readiness'],['draft','Draft follow-up']];
  return [['summary','Explain this view'],['gaps','Find gaps'],['draft','Draft follow-up']];
}
function cpAnswerHTML(result,context) {
  if(!result)return `<div class="cp-empty"><h3>Choose a task for this view.</h3><p>Choose a task above. The answer uses this page’s scope and source records, not a separate set of figures.</p></div>`;
  const refs=result.sources||[],number=ref=>refs.findIndex(s=>s.type===ref.type&&s.id===ref.id&&s.route===ref.route)+1;
  return `<div class="cp-answer" aria-live="polite"><h3>${e(result.heading)}</h3><div class="cp-facts">${result.items.slice(0,8).map(item=>`<p class="${item.tone==='warning'?'cp-warning':''}">${e(item.text)} ${item.refs.map(ref=>`<a href="#/${e(ref.route)}" class="cp-cite" title="${e(ref.label)}" aria-label="Source ${number(ref)||''}: ${e(ref.label)}">[${number(ref)||'source'}]</a>`).join(' ')}</p>`).join('')}</div>${result.items.length>8?`<p class="small muted">Showing 8 of ${result.items.length} items. Open the source workspace for the full population.</p>`:''}${result.proposal?`<div class="cp-proposal"><label>Proposed title<input data-cp-title maxlength="180" value="${e(result.proposal.title)}"></label>${btn('Review proposed action','cp-review','','primary small')}<small>No record is changed until you confirm in the action form.</small></div>`:''}${context.actions.length&&!result.proposal?`<div class="row wrap">${context.actions.map(a=>link(e(a.label),a.route,'quiet small')).join('')}</div>`:''}</div>`;
}
function cpPanelHTML() {
  const c=CP.context,result=CP.result;if(!c)return '';
  return `<section id="copilot-root" class="cp-panel" aria-labelledby="cp-heading"><div class="cp-head"><div><div class="cp-title"><span class="cp-mark">${icon('form')}</span><h2 id="cp-heading">Copilot</h2><span class="cp-label">Design preview</span></div><p>${e(c.title)} · ${e(c.scopeLabel)}</p></div>${btn(icon('close'),'cp-close','aria-label="Close copilot"','icon-only quiet')}</div><div class="cp-suggestions" aria-label="Suggested copilot tasks">${cpSuggestionLabels(c).map(([id,label])=>btn(e(label),'cp-task',`data-task="${id}" aria-pressed="${result?.task===id}"`,result?.task===id?'small primary':'small')).join('')}</div>${CP.notice?`<div class="cp-context-update" role="status">${e(CP.notice)}</div>`:''}<div class="cp-body"><div>${cpAnswerHTML(result,c)}<form id="cp-question-form" class="cp-question"><label class="sr-only" for="cp-question">Question about this view</label><input id="cp-question" name="question" maxlength="600" value="${e(CP.question)}" placeholder="Ask about this view…" autocomplete="off"><button type="submit" class="btn small" aria-label="Ask copilot">${icon('arrow')}</button></form></div><aside class="cp-sources" aria-label="Copilot source records"><details class="cp-source-disclosure" ${innerWidth>1023?'open':''}><summary>Sources in this view (${(result?.sources||c.sources).length})</summary>${(result?.sources||c.sources).slice(0,6).map((s,i)=>`<a class="cp-source" href="#/${e(s.route)}"><span>${i+1}</span><div><strong>${e(s.label)}</strong><small>${e(s.detail||s.id)}</small></div>${icon('arrow')}</a>`).join('')||'<p class="small muted">No source records are included.</p>'}${(result?.sources||c.sources).length>6?`<p class="small muted">6 of ${(result?.sources||c.sources).length} source references shown.</p>`:''}</details><details><summary>Scope and limits</summary><p>As of ${date(c.asOf)}. No model or external service is connected; answers are generated by local preview rules.</p>${c.excluded.map(s=>`<p>${e(s)}</p>`).join('')}<p>No approvals, signatures or source changes are performed by the copilot.</p></details></aside></div><div class="cp-foot">Local design preview · no model connected · no file contents sent · drafts require human review</div></section>`;
}
function cpMount(refreshContext=true) {
  if(!CP.open||current.blocked)return;
  if(refreshContext){const next=copilotContext();if(CP.context&&CP.context.hash!==next.hash){CP.result=null;CP.question='';CP.notice='The scope, source record or acting user changed. Previous suggestions were cleared.';}CP.context=next;}
  const existing=$('#copilot-root');if(existing)existing.outerHTML=cpPanelHTML();else $('.page-head,.j-intro')?.insertAdjacentHTML('afterend',cpPanelHTML());
}
function cpRun(task,question='') {
  CP.context=copilotContext();CP.result=copilotResult(task,question);CP.lastTask=task;CP.question='';CP.notice='';cpMount(false);
  const title=$('#copilot-root .cp-answer h3');if(title){title.tabIndex=-1;title.focus({preventScroll:true});}
}
Object.assign(actions,{
 'cp-open':()=>{CP.open=!CP.open;CP.notice='';CP.context=copilotContext();if(CP.open){cpMount(false);$('#cp-question')?.focus({preventScroll:true});}else $('#copilot-root')?.remove();const t=$('[data-action="cp-open"]');t?.setAttribute('aria-expanded',String(CP.open));},
 'cp-close':()=>{CP.open=false;$('#copilot-root')?.remove();const t=$('[data-action="cp-open"]');t?.setAttribute('aria-expanded','false');t?.focus({preventScroll:true});},
 'cp-task':el=>cpRun(el.dataset.task),
 'cp-review':()=>{requireValue(CP.result?.proposal,'Generate a draft first.');requireValue(CP.result.contextHash===copilotContext().hash,'The source changed. Generate a new draft before continuing.');uxNewAction(clone(CP.result.proposal));}
});
document.addEventListener('input',ev=>{if(ev.target.matches('[data-cp-title]')&&CP.result?.proposal)CP.result.proposal.title=ev.target.value;if(ev.target.id==='cp-question')CP.question=ev.target.value;});
document.addEventListener('submit',ev=>{if(ev.target.id!=='cp-question-form')return;ev.preventDefault();const text=values(ev.target).question.trim();if(!text)return;const task=/draft|follow.?up|next action/i.test(text)?'draft':/gap|missing|stale|readiness|ready|issue|fail/i.test(text)?'gaps':/explain|summari[sz]e|overview|risk|loss|return|status|version/i.test(text)?'summary':'unsupported';cpRun(task,text);});
document.addEventListener('keydown',ev=>{if(ev.key==='Escape'&&!$('#dialog').open&&ev.target.closest?.('#copilot-root')){ev.preventDefault();actions['cp-close']();}});
const cpPriorRender=render;
render=function(){
  cpPriorRender();
  if(current.blocked){CP.open=false;CP.result=null;CP.context=null;return;}
  // Never place the assistant into a registration wizard or crowd the global scope header.
  const excluded=['adopt','activity-wizard','dpia-wizard','setup','case-intake','kri-cycle','document-upload','approval-request','settings','data-health','not-found'];
  const supported=['home','reports','risks','losses','indicators','kri-returns','kri-return','insight-detail','program','assessment','requirement-review','risk','vendor','activity','dpia','privacy-case','form-builder','form-response','document','documents','approval','approvals','work','report-pack'];
  if(excluded.includes(current.name)||!supported.includes(current.name)){CP.result=null;CP.context=null;return;}
  const next=copilotContext();if(CP.context&&CP.context.hash!==next.hash){CP.result=null;CP.question='';CP.notice='The scope, source record or acting user changed. Previous suggestions were cleared.';}CP.context=next;
  $('.page-head .actions,.j-intro .actions')?.insertAdjacentHTML('beforeend',btn(icon('form')+' Copilot','cp-open',`aria-expanded="${CP.open}" aria-controls="copilot-root"`,'copilot-trigger'));
  if(CP.open)cpMount(false);
};
window.CLEAR_SIGHT_COPILOT=Object.freeze({version:'2.7-design-preview',context:()=>clone(copilotContext()),result:()=>CP.result?clone(CP.result):null,hasModel:false});
// The final bootstrap runs after every workflow module is registered.
