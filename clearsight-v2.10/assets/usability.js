/* V2.7: shared navigation/command safeguards. No new domain stores. */
'use strict';
function uxIsReviewItem(item) {
  return item.status === 'Open' && /review|approval|sign[ -]?off|decision|challenge/i.test(item.kind || '');
}
function uxUnavailable(message, back='home', heading='Record not available') {
  current.blocked = true;
  return header(heading, message) + `<section id="record-unavailable" class="panel">${empty('Check the source link', 'No other record has been substituted. Your organizational scope has not changed.', link('Back to workspace', back, 'primary'))}</section>`;
}
function uxRouteGuard() {
  if (S.scopeInvalid) {
    current.blocked = true;
    return header('Saved scope is unavailable', 'A stored location was removed or its branch, department and unit do not agree.') +
      `<section id="record-unavailable" class="panel">${empty('Choose a valid scope', 'The view has not been widened to the whole bank. Select a valid location in the header, or explicitly clear all locations.', btn('Clear all locations', 'cx-reset-scope', '', 'primary'))}</section>`;
  }
  if (current.name === 'not-found') return uxUnavailable('This page does not exist in the design reference.');
  const routes = {
    program:['program','programs'], assessment:['program','programs'], 'requirement-review':['program','programs'],
    vendor:['engagement','vendors'], risk:['risk','risks'], loss:['loss','losses'], control:['control','controls'],
    activity:['activity','privacy?tab=register'], 'activity-version':['activity','privacy?tab=register'],
    dpia:['dpia','privacy?tab=dpias'], 'privacy-case':['privacyCase','privacy?tab=cases'],
    'evidence-review':['evidence','evidence'], 'rcsa-review':['rcsa','rcsa'], document:['document','documents'],
    'kri-return':['kriReturn','kri-returns'], 'report-pack':['report','reports?view=library']
  };
  const cfg=routes[current.name], requestedId=current.params.get('id');
  if (cfg && requestedId && !recRead(cfg[0], requestedId)) return uxUnavailable('The requested '+REC_TYPES[cfg[0]].label.toLowerCase()+' could not be found.', cfg[1]);
  if(cfg&&requestedId&&cfg[0]!=='report'&&!recInScope(cfg[0],recRead(cfg[0],requestedId)))return uxUnavailable('This record is outside the selected location. Clear scope to open its original context.',cfg[1],'Outside the selected scope');
  if (current.name==='document' && requestedId) {
    const d=recRead('document',requestedId),vid=current.params.get('version'),v=vid&&docVersion(vid);
    if(!docVisible(d))return uxUnavailable('This document is outside the selected location.','documents','Outside the selected scope');
    if(vid&&(!v||v.documentId!==d.id))return uxUnavailable('This file version does not belong to the requested document.','documents','Version not available');
  }
  if (current.name==='approval' && requestedId) { const request=S.governance.requests.find(r=>r.id===requestedId);if(!request)return uxUnavailable('The approval request could not be found.','approvals');if(!approvalVisible(request))return uxUnavailable('This request belongs to a different location.','approvals','Outside the selected scope'); }
  if (['program','assessment','requirement-review'].includes(current.name)) {
    const p=prog(), t=current.params.get('target'), rid=current.params.get('requirement');
    if(t?.startsWith('ENG')&&p?.targets.includes(t)&&!recInScope('engagement',recRead('engagement',t)))return uxUnavailable('The selected engagement is outside the navigation scope.','programs','Outside the selected scope');
    if(t && !p?.targets.includes(t)) return uxUnavailable('This assessment target is not part of the requested program.', 'programs');
    if(rid && !p?.reqs.includes(rid)) return uxUnavailable('This requirement is not part of the requested program.', 'programs');
    const bid=current.params.get('binding');
    if(bid && !faInit().bindings.some(b=>b.id===bid&&b.programId===p.id&&b.target===selectedTarget(p))) return uxUnavailable('The submitted form does not belong to this program and target.', 'programs');
  }
  if (current.name==='rcsa-review' && current.params.has('risk')) {
    const c=recRead('rcsa',requestedId), r=recRead('risk',current.params.get('risk'));
    if(!r||!c?.risks.includes(r.id)) return uxUnavailable('This risk is not in the frozen cycle population.', 'rcsa');
    if(!scopeMatches(r,'risks')) return uxUnavailable('This risk is outside the selected scope. Clear scope to open its original assessment.', 'rcsa', 'Outside the selected scope');
  }
  if (['form-builder','form-response'].includes(current.name)) {
    const id=current.params.get('form'), bid=current.params.get('binding');
    if(id&&!faInit().forms.some(f=>f.id===id)) return uxUnavailable('This form could not be found.','forms');
    if(bid){
      const b=faInit().bindings.find(b=>b.id===bid),p=b&&recRead('program',b.programId);
      if(!b||!p||!p.targets.includes(b.target)||!faVersion(b.formVersionId))return uxUnavailable('The form assignment or its pinned version is unavailable.','forms');
      if(!recInScope('formResponse',{id:b.id}))return uxUnavailable('This assignment belongs to a different location. Clear scope to open the original target.','forms','Outside the selected scope');
    }
  }
  return '';
}
function uxWorkAction(item) {
  const canonical=workItems().find(x=>x.id===item.id);
  return canonical?.source&&canonical.source!=='task'
    ?link('Open review',canonical.route,'small')
    :btn('Review','task-detail',`data-id="${e(item.id)}"`,'small');
}
function uxActionSources() {
  const types=['risk','control','program','engagement','loss','activity','dpia','evidence','kriReturn'];
  return types.flatMap(type=>REC_TYPES[type].collection().filter(r=>recInScope(type,r)).map(r=>({type,id:r.id,name:recName(type,r),scope:recScope(type,r)})));
}
function uxNewAction(proposal=null) {
  const fixedScope=selectedScope(),context=recContext(),sources=uxActionSources();
  const proposedSource=proposal?.source||context, sourceKey=proposedSource?proposedSource.type+'::'+proposedSource.id:'';
  const initialSource=sources.some(s=>s.type+'::'+s.id===sourceKey)?sourceKey:'';
  const scope=fixedScope||(proposedSource?recScope(proposedSource.type,recRead(proposedSource.type,proposedSource.id))[0]:'')||'';
  const locations=ORG.filter(n=>!fixedScope||orgWithin(n.id,fixedScope));
  openDialog(proposal?'Review proposed action':'Create action', 'Confirm the accountable location, owner and due date before saving.',
    `<div class="stack">${input('title','Action title',proposal?.title||'','text','required maxlength="200"')}
      ${select('owner','Accountable owner',CS_SEED.people,proposal?.owner||recActor().name)}
      ${select('scope','Accountable location',[['','Choose location'],...locations.map(n=>[n.id,orgName(n.id)])],scope,'required')}
      ${input('due','Due date',addDays(S.asOf,7),'date',`required min="${S.asOf}"`)}
      ${select('source','Related record',[['','No related record'],...sources.map(s=>[s.type+'::'+s.id,REC_TYPES[s.type].label+' · '+s.name])],initialSource)}
      ${area('description','Work to be done',proposal?.description||'','maxlength="2000"')}
      ${proposal?notice('Copilot draft. Saving creates a work item only; it does not approve a document, change a risk rating or satisfy a requirement.','mini'):''}
    </div>`, 'Create action', form=>{
      const x=values(form),title=x.title.trim();requireValue(title.length>=3,'Enter an action title of at least three characters.');
      requireValue(recUserId(x.owner),'Choose a known accountable owner.');validDate(x.due,'Due date',false);requireValue(x.due>=S.asOf,'The due date cannot precede the workspace date.');
      requireValue(orgNode(x.scope)&&(!fixedScope||orgWithin(x.scope,fixedScope)),'Choose a location within the original scope.');
      let src=null;
      if(x.source){const [type,id]=x.source.split('::');const r=recRead(type,id);requireValue(r&&recInScope(type,r,fixedScope),'The related record is no longer available in the original scope.');requireValue(recScope(type,r).some(s=>orgWithin(s,x.scope)||orgWithin(x.scope,s)),'The related record and accountable location do not match.');src={type,id};}
      if(proposal?.contextHash)requireValue(proposal.contextHash===copilotContext().hash,'The source context changed. Regenerate this proposal before creating the action.');
      const z={id:uid('ACT'),title,description:x.description.trim(),kind:'Remediation',owner:x.owner,ownerId:recUserId(x.owner),due:x.due,status:'Open',orgScopeId:x.scope,ref:src?.id||'',sourceRef:src,route:src?recUrl(src.type,src.id):'work'};
      if(proposal)z.origin={type:'copilot-proposal',contextHash:proposal.contextHash,sources:clone(proposal.sources||[]),reviewedBy:recActor().id};
      S.tasks.unshift(z);audit('Created action '+z.title,z.ref,recActor().name);render();toast('Action created for '+orgName(x.scope)+'.');
    });
}
function uxShowError(box,message) {
  if(!box){toast(message);return;}
  box.hidden=false;box.textContent=message;box.setAttribute('role','alert');box.setAttribute('tabindex','-1');box.focus();box.scrollIntoView({block:'nearest'});
}
actions['new-action']=()=>uxNewAction();
Object.assign(titles,{'not-found':'Page not found'});
Object.assign(window.CS_JOURNEYS,{'not-found':()=>uxUnavailable('This page is not available in the design reference.')});
// Reuse the existing route/list memory. There is intentionally no second filter store.
const uxPriorRender=render;
render=function(){
  uxPriorRender();
  if(DOC_UPLOAD?.busy&&current.name==='document-upload')for(const el of $$('[data-doc-meta],#doc-file-input,[data-action="doc-browse"],[data-action="doc-upload-cancel"]'))el.disabled=true;
  if(current.blocked)return;
  if(current.name==='work')$('.page-head p').textContent='Assigned reviews, actions and due dates. Demo identities do not enforce access permissions.';
};
window.CLEAR_SIGHT_USABILITY=Object.freeze({version:'2.7-demo',routeBlocked:()=>!!current.blocked,isReview:uxIsReviewItem});

const uxPriorScopeRouteGuard=scopeRouteGuard;scopeRouteGuard=function(){const result=uxPriorScopeRouteGuard();if(result)current.blocked=true;return result;};
const uxPriorJourneyGuard=jScopeDenied;jScopeDenied=function(...args){const result=uxPriorJourneyGuard(...args);if(result)current.blocked=true;return result;};
const uxPriorKriGuard=ixReturnGuard;ixReturnGuard=function(...args){const result=uxPriorKriGuard(...args);if(result)current.blocked=true;return result;};
