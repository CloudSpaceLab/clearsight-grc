/* ClearSight V2.11 · focused assurance interactions.
 * This adapter derives next actions from the canonical stores. It does not own
 * response, document, approval or onboarding records and does not decide them. */
'use strict';

const ax211Attr=value=>String(value??'').replace(/\\/g,'\\\\').replace(/"/g,'\\"');
const ax211Here=()=>current.name+(current.params.toString()?'?'+current.params.toString():'');
function ax211ReplacePanel(html,title,replacement){
 const marker='<h2>'+title+'</h2>',at=html.indexOf(marker);if(at<0)return html;
 const start=html.lastIndexOf('<section',at);if(start<0)return html;
 const re=/<\/?section\b[^>]*>/g;re.lastIndex=start;let depth=0,m;
 while((m=re.exec(html))){depth+=m[0].startsWith('</')?-1:1;if(!depth)return html.slice(0,start)+replacement+html.slice(re.lastIndex);}
 return html;
}
function ax211ReviewAllowed(a,s){
 return !!s&&s.status==='Awaiting review'&&a.currentSubmissionId===s.id&&!a.draftOpen&&recActor().id===(s.assignedReviewerId||s.reviewerId)&&atReviewerEligibility(a.programId,a.target,recActor().id,s.submittedById,s.respondent.id).allowed;
}
function ax211AssignmentNext(value){
 const a=atAssignment(typeof value==='string'?value:value.id),z=atStatus(a.id),s=atSubmissionRaw(a.currentSubmissionId),respond=atCanRespond(a),manage=atCanManage(a),ownerId=a.respondent.kind==='external'?a.coordinatorId:a.respondent.id;
 const out={assignmentId:a.id,state:z.state,due:a.due,ownerId,route:atResponseRoute(a.id),label:'View assigned response',description:'Assigned to '+atName(ownerId)+'.',action:'',mine:false};
 if(!a.respondent.id)return {...out,ownerId:recUserId(atProgram(a.programId,a.target).owner),label:manage?'Assign a person':'View unassigned response',action:manage?'at-assign-edit':'',mine:manage,description:'Name the respondent and an independent reviewer before work can start.'};
 if(a.draftOpen){const returned=s?.status==='Changes requested';return {...out,label:respond?(returned?'Make requested changes':'Continue response'):'View assigned draft',mine:respond,description:returned?(s.review?.note||'Address the reviewer’s comments in this response cycle.'):'Waiting for '+atName(ownerId)+' to complete and submit this response.'};}
 if(s?.status==='Awaiting review'){const can=ax211ReviewAllowed(a,s);return {...out,ownerId:s.assignedReviewerId||s.reviewerId,route:atReviewRoute(s.id),label:can?'Review response':'View submitted response',mine:can,description:'Cycle '+s.cycle+' is awaiting '+atName(s.assignedReviewerId||s.reviewerId)+'.'};}
 if(s?.status==='Accepted'){
  if(!z.ready)return {...out,route:atReviewRoute(s.id),label:'Inspect accepted source',description:z.basisCurrent===false?'The program basis changed. A new response is needed before new conclusions.':z.integrity?.reason||'Inspect the current source before continuing.'};
  const rid=atMappedRequirements(s).find(id=>!s.requirementDecisions?.[id]&&evaluation(atProgram(a.programId,a.target),a.target,id).status!=='Satisfied');
  const eligible=atReviewerEligibility(a.programId,a.target,recActor().id,s.submittedById,s.respondent.id).allowed;
  if(rid&&eligible)return {...out,route:atRequirementRoute(s,rid),ownerId:recActor().id,label:'Review requirement conclusion',mine:true,description:'The response is accepted. '+findR(rid).code+' still needs a requirement conclusion.'};
  return {...out,route:atReviewRoute(s.id),label:'Inspect accepted response',description:'Current response accepted by '+(s.review?.reviewerName||'the independent reviewer')+'.'};
 }
 return {...out,route:s?atReviewRoute(s.id):out.route,label:s?'Inspect response history':'View assigned response',description:s?.legacy?'Historic attribution is retained. Start a named response cycle to collect current facts.':out.description};
}
function ax211NextAction(next,style='small'){
 return next.action?btn(next.label,next.action,`data-id="${e(next.assignmentId||next.id)}"${next.itemId?` data-item="${e(next.itemId)}"`:''}`,style):link(next.label,next.route,style);
}
function ax211AssignmentFilters(rows){
 if(!((current.name==='program'&&current.params.get('tab')==='assessments')||(current.name==='forms'&&current.params.get('tab')==='assignments')))return {rows,html:''};
 const mode=current.params.get('responsibility')||'all',mine=rows.filter(a=>ax211AssignmentNext(a).mine),pending=rows.filter(a=>atStatus(a.id).state==='Awaiting review');
 const html=`<nav class="ax211-queue-tabs" aria-label="Assignment responsibility">${[['all','All assignments',rows.length],['mine','My next actions',mine.length],['review','Awaiting review',pending.length]].map(([key,label,n])=>link(e(label)+' <span>'+n+'</span>',ui211Route(ax211Here(),{responsibility:key==='all'?null:key}),key===mode?'active small':'quiet small')).join('')}</nav>`;
 return {rows:mode==='mine'?mine:mode==='review'?pending:rows,html};
}
atAssignmentTable=function(assignments){
 const filtered=ax211AssignmentFilters(assignments),rows=filtered.rows;
 return filtered.html+(rows.length?table(['Named person','Assessment / organizational target','Current responsibility','Reviewer / due',''],rows.map(a=>{const z=atStatus(a.id),next=ax211AssignmentNext(a);return {attrs:`data-search="${e([a.respondent.name,atName(a.coordinatorId),atTitle(a),targetName(a.target),atName(z.reviewerId),z.state].join(' '))}"`,cells:[atPersonCell(a.respondent)+(a.coordinatorId?`<small>Captured by ${e(atName(a.coordinatorId))}</small>`:''),`<strong>${e(atTitle(a))}</strong><small>${e(targetName(a.target))} · ${a.kind==='form'?e(a.formVersionId):'Program assessment'} · cycle ${a.cycle||1}</small>`,`${pill(z.state,atTone(z.state))}<small>${e(next.description)}</small>`,`<strong>${e(atName(z.reviewerId))}</strong><small>Due ${date(a.due)}</small>`,`<div class="ax211-row-actions">${ax211NextAction(next,next.mine?'primary small':'small')}${link('History',atHistoryRoute(a),'quiet small')}${atCanManage(a)&&(a.draftOpen||atSubmissionRaw(a.currentSubmissionId)?.legacy)&&next.action!=='at-assign-edit'?btn('Assignment','at-assign-edit',`data-id="${e(a.id)}"`,'quiet small'):''}</div>`]};})):empty(assignments.length?'No assignments in this view':'No named response assignments',assignments.length?'Choose All assignments to see the other people and review responsibilities.':'Assign a person for this program target. The organizational target remains separate from the person.'));
};
const ax211ProgramView=programView;
programView=function(){let html=ax211ProgramView();if((current.params.get('tab')||'overview')!=='overview')return html;const p=prog(),target=selectedTarget(p),rows=atInit().assignments.filter(a=>a.programId===p.id&&a.target===target&&atVisible(a)),next=rows.map(ax211AssignmentNext).sort((a,b)=>Number(b.mine)-Number(a.mine)||(a.due||'9999').localeCompare(b.due||'9999'))[0],all='program?id='+p.id+'&target='+encodeURIComponent(target)+'&tab=assessments';
 const body=next?`<div class="pad ax211-program-next"><strong>${e(next.description)}</strong><dl><div><dt>Responsible person</dt><dd>${e(atName(next.ownerId))}</dd></div><div><dt>Response due</dt><dd>${date(next.due)}</dd></div><div><dt>Current state</dt><dd>${pill(next.state,atTone(next.state))}</dd></div></dl>${ax211NextAction(next,'primary wide')}${link('All named assignments',all,'quiet small wide')}</div>`:`<div class="pad"><p>Name the people who will provide this target’s responses and their independent reviewers.</p>${link('Assign people to this target',all,'primary wide')}</div>`;
 html=ax211ReplacePanel(html,'Next assessment',panel('Next response responsibility','Current named assignments for this organizational target.',body));if(next)html=html.replace(link('Open assessment '+icon('arrow'),'assessment?id='+p.id+'&target='+encodeURIComponent(target),'primary'),ax211NextAction(next,'primary'));return html;
};

/* A field-level read model uses the same source validators as submission. The
 * existing atOpenSubmit/atSubmit remain authoritative for identity and mutation. */
const ax211BasicControl=faBasicControl;
faBasicControl=function(type,value,attr,choices,disabled){let html=ax211BasicControl(type,value,attr,choices,disabled);if(type!=='evidence'||disabled)return html;const get=name=>String(attr||'').match(new RegExp('data-'+name+'="([^"]+)"'))?.[1]||'',field=get('fa-answer')||get('fa-cell');if(!field)return html;return html.replace('data-action="doc-form-file"',`data-action="doc-form-file" data-ax211-field="${field}"${get('row')?` data-ax211-row="${get('row')}"`:''}${get('col')?` data-ax211-col="${get('col')}"`:''}`);};
function ax211FormSelector(field,message,response,definition){
 const id=ax211Attr(field.id),base='#fa-field-'+field.id;
 if(field.columns){
  const rows=response.answers[field.id]||field.rows.map(r=>({...r,cells:{}}));
  for(const row of rows)for(const column of field.columns){
   if(column.mode==='integration'||column.type==='calculated')continue;
   const value=row.cells?.[column.id],prefix=field.label+' / '+row.label+' / '+column.label+':',range=message.startsWith(column.label+':')&&column.type==='number'&&value!==undefined&&value!==''&&(!Number.isFinite(Number(value))||Number(value)<(column.min??-Infinity)||Number(value)>(column.max??Infinity)),documentError=column.type==='evidence'&&message.startsWith(field.label+' / '+column.label+':')&&value&&!docVersion(value);
   if(message.startsWith(prefix)||range||documentError)return column.type==='evidence'?`[data-ax211-field="${id}"][data-ax211-row="${ax211Attr(row.id)}"][data-ax211-col="${ax211Attr(column.id)}"]`:`[data-fa-cell="${id}"][data-row="${ax211Attr(row.id)}"][data-col="${ax211Attr(column.id)}"]`;
  }
 }
 if(['matrix','multi_matrix'].includes(field.type)){const index=field.rows.findIndex((_,i)=>field.type==='matrix'?!response.answers[field.id]?.[i]:!response.answers[field.id]?.[i]?.length);if(index>=0)return `[data-fa-matrix="${id}"][data-row="${index}"]`;}
 if(field.mode==='calculated'||field.type==='calculated'){const dependency=(field.depends||[]).map(id=>definition?.fields.find(f=>f.id===id)).find(f=>f&&f.mode==='human'&&(response.answers[f.id]==null||response.answers[f.id]===''));return dependency?ax211FormSelector(dependency,'',response,definition):base;}
 if(field.type==='evidence')return `[data-ax211-field="${id}"]`;
 return `[data-fa-answer="${id}"]`;
}
function ax211ResponseIssues(value){
 const a=atAssignment(typeof value==='string'?value:value.id),p=atProgram(a.programId,a.target),response=atResponse(a),issues=[];
 if(!a.draftOpen)return issues;
 if(a.kind==='form'){
  const def=faVersion(a.formVersionId)?.definition;if(!def)return [{message:'The pinned form version is unavailable.',selector:'.at-accountability',kind:'source'}];
  for(const f of def.fields.filter(f=>faVisible(f,response.answers)))for(const message of faErrors({...def,fields:[f]},response))issues.push({message,selector:ax211FormSelector(f,message,response,def),fieldId:f.id,kind:'answer'});
  if(faChecks(def,response.answers).some(c=>!response.observations?.[c.key]||response.observations[c.key].result==='Unknown')&&String(response.note||'').trim().length<12)issues.push({message:'Explain the missing or uncertain technical checks (at least 12 characters).',selector:'[data-fa-note]',kind:'explanation'});
 }else for(const q of uniqueQ(activeReqs(p))){
  const value=String(response.answers[q.id]??'').trim(),selector=`[data-answer="${ax211Attr(q.id)}"]`;let problem='';
  try{requireValue(value,q.id+': answer “'+q.title+'”.');if(q.type==='date'){validDate(value,q.id+' observation date');requireValue(days(S.asOf,value)<=policy(p,q).freshDays,q.id+': refresh the observation before submitting.');}if(q.type==='number')bounded(value,0,1e9,q.id+' value');if(q.type==='choice')requireValue(['Yes','Partially','No'].includes(value),q.id+': choose a valid answer.');}catch(error){problem=error.message;}
  if(problem)issues.push({message:problem,selector,fieldId:q.id,kind:'answer'});
  if(policy(p,q).evidence&&!qualifiedEvidence(p,a.target,q).length)issues.push({message:q.id+': link a current document for '+q.title+'.',selector:`[data-action="add-evidence"][data-question="${ax211Attr(q.id)}"]`,fieldId:q.id,kind:'document'});
 }
 return issues;
}
function ax211QuestionRows(a){
 const response=atResponse(a),p=atProgram(a.programId,a.target),issues=ax211ResponseIssues(a),questions=a.kind==='form'?faVersion(a.formVersionId).definition.fields.filter(f=>faVisible(f,response.answers)).map(f=>({id:f.id,label:f.label,selector:'#fa-field-'+f.id,mode:f.mode})):uniqueQ(activeReqs(p)).map(q=>({id:q.id,label:q.title,selector:'#q-'+q.id,mode:'human'}));
 return questions.map(q=>({...q,issues:issues.filter(x=>x.fieldId===q.id)}));
}
function ax211ResponseNavigator(a){
 const rows=ax211QuestionRows(a),issues=ax211ResponseIssues(a),pending=rows.filter(q=>q.issues.length),issueFields=new Set(pending.map(q=>q.id));
 return `<div class="ax211-response-nav" id="ax211-response-nav"><div class="ax211-readiness"><strong>${pending.length?pending.length+' question'+(pending.length===1?'':'s')+' need attention':issues.length?'Add the check explanation':'Required answers complete'}</strong><small>${rows.length-issueFields.size} / ${rows.length} questions have no answer or document blockers. Review is a separate decision.</small></div><div id="ax211-response-issues">${ui211Issues(issues,{title:'Complete before submitting'})}</div><details ${pending.length?'':'open'}><summary>Jump to any question (${rows.length})</summary><nav aria-label="Response questions">${rows.map(q=>`<button type="button" data-action="ui211-focus" data-selector="${e(q.selector)}"><span>${e(q.id)} · ${e(q.label)}</span><small>${q.issues.length?'Needs attention':q.mode==='integration'?'Source observation':q.mode==='calculated'?'Calculated':'No answer blockers'}</small></button>`).join('')}</nav></details></div>`;
}
function ax211RefreshResponseNavigator(){
 if(!['assessment','form-response'].includes(current.name))return;
 try{const a=atRouteAssignment(current.name==='form-response'?'form':'canonical'),node=$('#ax211-response-nav');if(a&&node&&a.draftOpen&&atCanRespond(a))node.outerHTML=ax211ResponseNavigator(a);}catch{/* The original view explains unavailable assignments. */}
}
const ax211AssessmentView=assessmentView;
assessmentView=function(){let html=ax211AssessmentView(),a;try{a=atRouteAssignment();}catch{return html;}if(!a?.draftOpen||!atCanRespond(a))return html;
 html=html.replace(/<nav class="question-nav pad"[\s\S]*?<\/nav>/,ax211ResponseNavigator(a));
 return html.replace('<h2>Response progress</h2>','<h2>Answers saved</h2>');
};
const ax211FormResponseView=formResponseView;
formResponseView=function(){let html=ax211FormResponseView();if(!faBinding())return html;const a=atRouteAssignment('form');if(!a.draftOpen||!atCanRespond(a)||faBinding().id!==a.bindingId)return html;const response=atResponse(a),checks=faChecks(faVersion(a.formVersionId).definition,response.answers),results=checks.map(c=>response.observations?.[c.key]?.result||'Unknown'),body=ax211ResponseNavigator(a)+`<div class="pad ax211-submit-area">${checks.length?`<p class="small muted">Source checks: ${results.filter(x=>x==='Pass').length} pass · ${results.filter(x=>x==='Fail').length} fail · ${results.filter(x=>x==='Unknown').length} missing or unknown. Failed checks remain available for review.</p><label>Explain uncertain / unavailable checks<textarea data-fa-note placeholder="Required when source checks are missing, stale or unknown.">${e(response.note||'')}</textarea></label>`:''}<div class="fa-form-error" id="fa-response-error" role="alert"></div>${btn('Submit for independent review','fa-submit','','primary wide mt16')}</div>`;return ax211ReplacePanel(html,'Assessment status',panel('Response checklist','Select an item to go directly to the answer, document or explanation.',body));};
const ax211OpenSubmit=atOpenSubmit;
atOpenSubmit=function(a){
 if(a?.draftOpen&&atCanRespond(a)){const issues=ax211ResponseIssues(a);if(issues.length){ax211RefreshResponseNavigator();ui211Focus(issues[0].selector);toast(issues.length+' item'+(issues.length===1?'':'s')+' to complete. Select an item in the response checklist.');return false;}}
 return ax211OpenSubmit(a);
};
for(const type of ['input','change'])document.addEventListener(type,event=>{if(event.target.matches?.('[data-answer],[data-fa-note]'))ax211RefreshResponseNavigator();});

/* Reviewer choices reflect the exact target and person currently selected in
 * the assignment dialog; the canonical assign command validates them again. */
function ax211AssignmentReviewerChoices(programId,target,respondentId,coordinatorId=''){
 return atEligibleReviewers(programId,target,coordinatorId||respondentId,respondentId);
}
function ax211RefreshReviewers(form){
 if(!form?.dataset.ax211Assignment)return;const scope=form.querySelector('[name="scope"]')?.value||'',cut=scope.indexOf('::');if(cut<0)return;
 const respondentId=form.querySelector('[name="respondentId"]')?.value||form.dataset.ax211Respondent||'',coordinatorId=form.querySelector('[name="coordinatorId"]')?.value||'',field=form.querySelector('[name="reviewerId"]');if(!field)return;
 let choices;try{choices=ax211AssignmentReviewerChoices(scope.slice(0,cut),scope.slice(cut+2),respondentId,coordinatorId);}catch{choices=[];}
 const chosen=choices.some(p=>p.id===field.value)?field.value:'';field.innerHTML=options([['','Choose an eligible reviewer'],...choices.map(p=>[p.id,p.name+' · '+p.kind])],chosen);field.value=chosen;
}
const ax211OpenAssign=atOpenAssign;
atOpenAssign=function(config={}){const result=ax211OpenAssign(config),form=$('#modal-form');if(form){form.dataset.ax211Assignment='true';form.dataset.ax211Respondent=config.assignmentId?atAssignment(config.assignmentId).respondent.id||'':'';ax211RefreshReviewers(form);}return result;};
document.addEventListener('change',event=>{const field=event.target,form=field.closest?.('form[data-ax211-assignment]');if(form&&['scope','respondentId','coordinatorId'].includes(field.name))ax211RefreshReviewers(form);});

function ax211RevisionChanges(s){
 s=atSubmission(typeof s==='string'?s:s.id).s;
 const previous=atSubmissionRaw(s.previousSubmissionId);if(!previous)return null;const before=previous.sourceSnapshot,after=s.sourceSnapshot,keys=(a,b)=>[...new Set([...Object.keys(a||{}),...Object.keys(b||{})])].filter(key=>recStable(a?.[key])!==recStable(b?.[key]));
 const oldDocs=new Map((before.documentVersions||[]).map(v=>[v.id,v])),newDocs=new Map((after.documentVersions||[]).map(v=>[v.id,v]));
 return {previousId:previous.id,answers:keys(before.answers,after.answers),observations:keys(before.observations,after.observations),addedDocuments:[...newDocs.keys()].filter(id=>!oldDocs.has(id)),removedDocuments:[...oldDocs.keys()].filter(id=>!newDocs.has(id)),documentContextChanged:recStable(before.documents||[])!==recStable(after.documents||[]),noteChanged:(before.note||'')!==(after.note||''),captureChanged:(s.captureReference||'')!==(previous.captureReference||'')};
}
const ax211Differences=atDifferences;
atDifferences=function(s){const change=ax211RevisionChanges(s);if(!change)return '';const old=ax211Differences(s);return `<details class="ax211-revision-summary panel" id="ax211-review-changes"><summary><strong>Changes since cycle ${atSubmissionRaw(change.previousId).cycle}</strong><span>${change.answers.length} changed answer${change.answers.length===1?'':'s'} · ${change.observations.length} changed observation${change.observations.length===1?'':'s'} · ${change.addedDocuments.length} added / ${change.removedDocuments.length} removed document versions</span></summary>${change.documentContextChanged||change.noteChanged||change.captureChanged?`<p class="pad">${[change.documentContextChanged?'Document review context changed.':'',change.noteChanged?'The respondent explanation changed.':'',change.captureChanged?'The external capture reference changed.':''].filter(Boolean).map(e).join(' ')}</p>`:''}${old}</details>`;};
const ax211ReviewView=atReviewView;
atReviewView=function(){let html=ax211ReviewView(),ctx;try{ctx=atSubmission(current.params.get('submission'));}catch{return html;}
 for(const [label,id]of [['Complete submitted answers','ax211-review-answers'],['Documents','ax211-review-documents'],['Independent response review','ax211-review-decision']])html=html.replace('<h2>'+label+'</h2>',`<h2 id="${id}" tabindex="-1">${label}</h2>`);
 const changes=ax211RevisionChanges(ctx.s),jump=`<nav class="ax211-review-jumps" aria-label="Review sections">${[['Answers','#ax211-review-answers'],['Documents','#ax211-review-documents'],...(changes?[['Changes ('+changes.answers.length+' answer'+(changes.answers.length===1?'':'s')+')','#ax211-review-changes summary']]:[]),[ax211ReviewAllowed(ctx.a,ctx.s)?'Record review decision':'Review outcome','#ax211-review-decision']].map(([label,selector])=>btn(label,'ui211-focus',`data-selector="${selector}"`,'quiet small')).join('')}</nav>`;
 return html.replace('<div class="at-review-layout">',jump+'<div class="at-review-layout">');
};
window.CS_JOURNEYS['assessment-review']=()=>atReviewView();

/* Program adoption now hands off to named assignments instead of the generic
 * overview. No people or reviews are created implicitly by this presentation. */
function ax211AdoptionIssues(){return draftErrors(draft()).map(message=>{const first=/program name|assessment target|applicable requirement|rationale/.test(message),selector=/program name/.test(message)?'[data-draft="name"]':/assessment target/.test(message)?'[data-target]':/applicable requirement|rationale/.test(message)?'[data-action="tailor"]':/owner/.test(message)&&!/different/.test(message)?'[data-draft="owner"]':/respondent unit/.test(message)?'[data-respondent]':/due date/.test(message)?'[data-draft="due"]':'[data-draft="reviewer"]';return {message,step:first?1:3,selector};});}
function ax211StepIssues(items,action,id=''){
 return `<section class="ui211-issues ax211-step-issues" id="ax211-step-issues"><h3>Complete before submitting <span>${items.length}</span></h3><ul>${items.map(item=>`<li>${btn(e(item.message),action,`data-step="${item.step}" data-selector="${e(item.selector)}"${id?` data-id="${e(id)}"`:''}`,'quiet')}</li>`).join('')}</ul></section>`;
}
const ax211AdoptView=adoptView;
adoptView=function(){let html=ax211AdoptView();const issues=ax211AdoptionIssues();html=html.replace('Accountability & respondents','Accountability & participating teams').replace('Targets define what is assessed. Respondents define who supplies the answers.','Targets define what is assessed. Named people are assigned after the program is published.').replace('<h3 class="mt16">Respondent units</h3>','<h3 class="mt16">Participating teams</h3>').replace('Multiple contributors may respond. Review is a separate action, not a consequence of submitting a form.','Publishing is followed by named respondent assignments. Each person then has a separate response and independent review.');if(wizardStep===3&&issues.length)html=html.replace(notice(draftErrors(draft()).map(e).join('<br>'),'warning'),ax211StepIssues(issues,'ax211-adoption-fix'));return html;};
const ax211PublishReview=publishReview;
publishReview=function(){const issues=ax211AdoptionIssues();if(issues.length){wizardStep=3;render();ui211Focus('#ax211-step-issues');return false;}const existing=new Set(S.programs.map(p=>p.id)),result=ax211PublishReview(),commit=modalSave;if(commit)modalSave=form=>{const finish=value=>{if(value===false)return false;const created=S.programs.find(p=>!existing.has(p.id));if(created){atInit();go('program?id='+created.id+'&target='+encodeURIComponent(created.targets[0])+'&tab=assessments');toast('Program published. Assign the people who will respond.');}return value;};const value=commit(form);return value?.then?value.then(finish):finish(value);};return result;};
actions['publish-review']=()=>publishReview();
actions['ax211-adoption-fix']=el=>{const items=ax211AdoptionIssues(),item=items.find(x=>String(x.step)===String(el.dataset.step)&&x.selector===el.dataset.selector);requireValue(item,'This adoption issue is no longer current.');wizardStep=item.step;save();render();ui211Focus(item.selector);};

/* Document queues and destinations use contextual review records and exact
 * version links. Reading a file or opening a review never completes it. */
function ax211DocumentQueue(mode='all'){
 d210Init();const all=S.evidence.filter(x=>!S.scopeInvalid&&recInScope('evidence',x)&&x.status==='Submitted');
 return all.filter(x=>mode==='mine'?d210ReviewEligibility(x).allowed:mode==='prepared'?x.submittedById===recActor().id:true).sort((a,b)=>String(a.expires||'9999').localeCompare(String(b.expires||'9999'))||String(a.submittedAt||'').localeCompare(String(b.submittedAt||'')));
}
const ax211DocumentRows=d210ReviewRows;
d210ReviewRows=function(rows){let html=ax211DocumentRows(rows);for(const x of rows){const label=x.status==='Submitted'&&d210ReviewEligibility(x).allowed?'Review this submission':x.status==='Changes requested'?'View requested changes':'Inspect submission';html=html.replace(link('Open document',d210ReviewRoute(x),'quiet small'),link(label,d210ReviewRoute(x),'small'));}return html;};
const ax211DocumentsView=documentsView;
documentsView=function(){let html=ax211DocumentsView();if(current.params.get('view')!=='pending')return html;const mode=['mine','prepared'].includes(current.params.get('queue'))?current.params.get('queue'):'all',rows=ax211DocumentQueue(mode),tabs=`<nav class="ax211-queue-tabs" aria-label="Document review responsibility">${[['all','All pending'],['mine','My review decisions'],['prepared','Submitted by me']].map(([key,label])=>link(e(label)+' <span>'+ax211DocumentQueue(key).length+'</span>',ui211Route(ax211Here(),{queue:key==='all'?null:key}),key===mode?'active small':'quiet small')).join('')}</nav>`;return ax211ReplacePanel(html,'Document review queue',panel('Document review queue',rows.length+' exact submissions in this view. Review authority and the pinned file version are checked again when deciding.',tabs+(rows.length?d210ReviewRows(rows):empty('No pending documents in this view','Choose All pending to see other review responsibilities.'))));};
function ax211DocumentResponse(x){
 const assignments=atInit().assignments.filter(a=>atVisible(a)&&a.programId===x.program&&a.target===x.target),submitted=assignments.map(a=>atSubmissionRaw(a.currentSubmissionId)).filter(s=>s?.sourceSnapshot.documents?.some(d=>d.id===x.id));
 if(submitted.length===1)return {label:'Open linked response',route:atReviewRoute(submitted[0].id)};
 const editable=assignments.filter(a=>a.kind==='canonical'&&a.draftOpen&&atCanRespond(a));if(editable.length===1)return {label:'Return to this question',route:ui211Route(atResponseRoute(editable[0].id),{question:x.question})};
 return {label:'Program assessments',route:'program?id='+x.program+'&target='+encodeURIComponent(x.target)+'&tab=assessments'};
}
const ax211DocumentReviews=d210DocumentReviews;
d210DocumentReviews=function(d,v){let html=ax211DocumentReviews(d,v);const id=current.params.get('review'),x=id&&d210Contexts(d.id).find(x=>x.id===id);if(!x||x.documentVersionId!==v.id)return html;
 const response=x.program&&ax211DocumentResponse(x),old=x.program&&link('Open assessment','assessment?id='+x.program+'&target='+encodeURIComponent(x.target),'quiet small');if(response&&old)html=html.replace(old,link(response.label,response.route,'quiet small'));
 const next=ax211DocumentQueue('mine').find(y=>y.id!==x.id),controls=link('My pending document reviews','documents?view=pending&queue=mine','quiet small')+(x.status==='Submitted'&&d210ReviewEligibility(x).allowed?btn('Go to review decision','ui211-focus','data-selector="[data-d210-submit] [name=decision]"','small'):'')+(next?link('Review next document',d210ReviewRoute(next),'small'):'');
 return html.replace('<div class="d210-actions">','<div class="d210-actions">'+controls);
};

/* Vendor blockers are actionable records, not another readiness calculation. */
function ax211VendorBlockers(id){
 const d=voDossier(typeof id==='string'?id:id.id),readiness=voReadiness(d),blocks=[];
 if(d.state!=='Due diligence')return [{kind:'intake',title:'Complete the intake decision',description:d.returnNote||'Confirm the business scope and approve the diligence plan.',ownerId:d.state==='Owner confirmation'?d.ownerId:d.state==='Changes requested'?d.submission.submittedBy:d.riskReviewerId,route:voRoute(d.id),label:'Open intake decision',mine:false}];
 for(const row of voAssignmentRows(d).filter(row=>!row.status.ready)){if(!row.assignment){blocks.push({kind:'response',title:row.title,state:row.status.state,description:'This named assignment is unavailable. Review the service’s assignment records before continuing.',ownerId:d.ownerId,recordId:row.id,route:voRoute(d.id)+'&tab=assessments',label:'Inspect service assignments',mine:d.ownerId===recActor().id});continue;}const next=ax211AssignmentNext(row.assignment);blocks.push({...next,kind:'response',title:row.title,description:next.description,recordId:row.id});}
 for(const item of d.documents){const status=voDocumentStatus(d,item);if(status.ready)continue;const request=S.governance.requests.find(r=>r.id===item.approvalRequestId),canProvide=[item.coordinatorId,item.assignee.id].includes(recActor().id),pending=status.state==='Awaiting review'&&request?.status==='Pending',reviewer=pending?approvalNext(request)?.actorId:item.reviewerId,needsSource=['Document requested','Source unavailable','Original file required','New version available','Expired','Validity required'].includes(status.state),action=canProvide&&!pending?(needsSource?'vo-document-link':'vo-document-review'):'';
  blocks.push({kind:'document',title:item.title,state:status.state,description:status.reason,due:item.due,ownerId:pending?reviewer:item.coordinatorId||item.assignee.id,mine:pending?reviewer===recActor().id:canProvide,action,id:d.id,itemId:item.id,recordId:item.id,label:pending?(reviewer===recActor().id?'Review document':'View document review'):action?(needsSource?'Choose current document':'Prepare document review'):'View document request',route:pending?'approval?id='+request.id:voRoute(d.id)+'&tab=documents'});
 }
 for(const summary of readiness.programs){const p=recRead('program',summary.id);if(!p||!flTargetVisible(p,d.engagementId))continue;for(const rid of activeReqs(p)){const state=evaluation(p,d.engagementId,rid);if(state.status==='Satisfied')continue;const sources=atAcceptedSources(p.id,d.engagementId,rid),source=sources.find(s=>!s.requirementDecisions?.[rid])||sources[0],decided=source?.requirementDecisions?.[rid],eligible=source&&!decided&&atReviewerEligibility(p.id,d.engagementId,recActor().id,source.submittedById,source.respondent.id).allowed;blocks.push({kind:'requirement',title:findR(rid).code+' · '+findR(rid).title,state:state.status,description:decided?'The recorded conclusion needs follow-up. Address the finding and submit a new response before another conclusion.':source?'Review the accepted response against this requirement.':'A current accepted response is needed before a new conclusion can be recorded.',ownerId:source?.assignedReviewerId||source?.reviewerId||recUserId(p.reviewer),mine:!!eligible,recordId:rid,label:eligible?'Review requirement':decided?'Inspect recorded conclusion':'Inspect requirement',route:source?atRequirementRoute(source,rid):'requirement-review?id='+p.id+'&target='+encodeURIComponent(d.engagementId)+'&requirement='+rid});}}
 return blocks.sort((a,b)=>Number(b.mine)-Number(a.mine)||(a.due||'9999').localeCompare(b.due||'9999'));
}
function ax211VendorBlockList(blocks){return `<ol class="ax211-blocker-list" id="ax211-vendor-blockers">${blocks.map(b=>`<li><div><strong>${e(b.title)}</strong><small>${e(b.description)}</small><span>${e(atName(b.ownerId))}${b.due?' · due '+date(b.due):''}</span></div><div>${b.state?pill(b.state):''}${ax211NextAction(b,b.mine?'primary small':'small')}</div></li>`).join('')}</ol>`;}
const ax211VendorDossierView=voDossierView;
voDossierView=function(){let html=ax211VendorDossierView(),d;try{d=voDossier(current.params.get('id'));}catch{return html;}if(d.state!=='Due diligence'||recRead('engagement',d.engagementId)?.status==='Active')return html;const blocks=ax211VendorBlockers(d);if(!blocks.length)return html;const next=blocks[0],card=ui211Next({title:next.mine?'Your next diligence task':'Next diligence responsibility',description:next.title+' — '+next.description,owner:atName(next.ownerId),due:next.due,state:next.state,primary:ax211NextAction(next,'primary'),secondary:btn('See remaining work ('+blocks.length+')','ax211-vendor-work',`data-id="${e(d.id)}"`,'quiet')});html=html.replace(/<section class="vo-next panel">[\s\S]*?<\/section>/,card);const tab=current.params.get('tab')||'overview';if(tab==='overview'){html=ax211ReplacePanel(html,'Activation conditions','');html=html.replace('<aside>','<aside>'+panel('Remaining due diligence','Each item opens its exact response, document review or requirement. Readiness is still checked by the source workflow.',ax211VendorBlockList(blocks)));}return html;};
const ax211VendorAssessmentsPanel=voAssessmentsPanel;
voAssessmentsPanel=function(d){return ax211ReplacePanel(ax211VendorAssessmentsPanel(d),'Named assessment assignments',panel('Named assessment assignments','Each row opens the next responsibility for the current actor.',atAssignmentTable(voAssignmentRows(d).map(row=>row.assignment).filter(Boolean))));};
actions['ax211-vendor-work']=el=>{const d=voDossier(el.dataset.id);if(current.name==='vendor-onboarding'&&current.params.get('id')===d.id&&!current.params.get('tab'))ui211Focus('#ax211-vendor-blockers');else go(voRoute(d.id)+'&attention=remaining');};
window.CS_JOURNEYS['vendor-onboarding']=()=>voDossierView();

function ax211VendorIntakeIssues(d){
 const field=(name,step)=>({selector:`[name="${ax211Attr(name)}"]`,step});
 return voIssues(d.values,{engagementId:d.engagementId,preparerId:d.createdBy}).map(message=>{
  let target;
  for(const q of d.values.questionnaires){if(message.includes(q.title)){target=field(message.includes('deadline')?'q_due_'+q.key:message.includes('respondent type')?'q_type_'+q.key:message.includes('internal respondent')?'q_person_'+q.key:'q_review_'+q.key,3);break;}}
  if(!target)for(const item of d.values.documents){if(message.toLowerCase().includes(item.title.toLowerCase())){target=field(message.includes('deadline')?'doc_due_'+item.id:message.startsWith('Keep')?'doc_required_'+item.id:message.startsWith('Explain')?'doc_reason_'+item.id:message.includes('reviewer')?'doc_review_'+item.id:'doc_person_'+item.id,3);break;}}
  if(!target)for(const r of d.values.requirements){if(message.includes(findR(r.id)?.code)&&message.includes('does not apply')){target=field('reason_'+r.id,3);break;}}
  if(!target){const map=[[/legal name/,'supplierName',1],[/already exists|existing supplier/,'supplierId',1],[/supplier.*country/,'country',1],[/external person/,'contactName',1],[/respondent’s role/,'contactRole',1],[/valid email/,'contactEmail',1],[/service being onboarded|already has this service/,'service',2],[/service boundary/,'description',2],[/business need/,'businessPurpose',2],[/data, systems/,'dataContext',2],[/whether personal data/,'personalData',2],[/Choose service criticality/,'criticality',2],[/criticality and operational/,'criticalityReason',2],[/risk reviewer/,'riskReviewerId',2],[/location|submission authority/,'orgScopeId',2],[/internal owner/,'ownerId',2],[/internal response lead/,'responseLeadId',2],[/Owner confirmation deadline/,'ownerDue',3],[/Risk review deadline|Risk review must follow/,'reviewDue',3],[/Proposed start/,'proposedStart',2],[/periodic review/,'renewal',2],[/vendor assurance questionnaire/,'q_type_FORM-VENDOR',3],[/baseline requirement response/,'q_review_baseline',3],[/applicable requirement|baseline changed/,'req_'+d.values.requirements[0]?.id,3],[/document assignee/,'doc_type_'+d.values.documents[0]?.id,3]];const match=map.find(([pattern])=>pattern.test(message));target=match?field(match[1],match[2]):{step:3,selector:'.vo-capture'};}
  return {message,...target};
 });
}
const ax211VendorIntakeStep=voIntakeStep;
voIntakeStep=function(d){const html=ax211VendorIntakeStep(d);if(d.step!==4)return html;const issues=ax211VendorIntakeIssues(d);return issues.length?html.replace(/<section class="vo-validation">[\s\S]*?<\/section>/,ax211StepIssues(issues,'ax211-intake-fix',d.id)):html;};
actions['ax211-intake-fix']=el=>{const d=voDraft(el.dataset.id),issue=ax211VendorIntakeIssues(d).find(x=>String(x.step)===String(el.dataset.step)&&x.selector===el.dataset.selector);requireValue(issue,'This intake issue is no longer current.');const form=$('form[data-vo-form="intake"]');voSaveDraft(d.id,form?values(form):{},issue.step);render();ui211Focus(issue.selector);};

const ax211Render=render;
render=function(){const value=ax211Render();if(current.name==='assessment'&&current.params.get('question')){try{const a=atRouteAssignment(),q=current.params.get('question');if(a&&uniqueQ(activeReqs(atProgram(a.programId,a.target))).some(x=>x.id===q))ui211Focus('#q-'+q);}catch{/* Unavailable route is already rendered. */}}if(current.name==='vendor-onboarding'&&current.params.get('attention')==='remaining')ui211Focus('#ax211-vendor-blockers');return value;};

window.CLEAR_SIGHT_ASSURANCE_UX={version:'2.11',assignmentNext:ax211AssignmentNext,responseIssues:ax211ResponseIssues,reviewerChoices:ax211AssignmentReviewerChoices,revisionChanges:ax211RevisionChanges,documentQueue:ax211DocumentQueue,vendorBlockers:ax211VendorBlockers,adoptionIssues:ax211AdoptionIssues,intakeIssues:ax211VendorIntakeIssues};
