/* ClearSight V2.12 · internal responsibilities and scoped capabilities.
 * Role lenses only arrange existing work. The canonical access policy, named
 * assignments and domain source/version checks remain authoritative. This local
 * synthetic model does not provide an authenticated server boundary. */
'use strict';

const RE212_MODULES={program:'programs',formResponse:'programs',engagement:'vendors',activity:'vendors',dpia:'vendors',privacyCase:'vendors',risk:'risks',loss:'risks',control:'risks',rcsa:'risks',kriReturn:'kri',indicator:'kri',document:'documents',evidence:'documents',report:'reports'};
const re212Own=(x,key)=>Object.prototype.hasOwnProperty.call(x||{},key);
const re212Id=value=>recUser(value)?.id||recUserId(value)||'';
const re212Identity=()=>({id:recActor().id,name:recActor().name});
function re212ScopeList(type,record){
 if(!record)return [];
 if(type==='document')return record.scopeMode==='shared'?['']:record.orgScopeId?[record.orgScopeId]:[];
 if(type==='indicator')return scopeIds(record,'indicators');
 if(type==='report'){const snapshot=record.snapshot||{};if(snapshot.scopeId===''||snapshot.scope?.id===''||snapshot.scope&&['branch','department','unit'].every(k=>snapshot.scope[k]===''))return [''];}
 return [...new Set(recScope(type,record))];
}
function re212Subject(type,id){return type==='indicator'?S.indicators.find(x=>x.id===id):recRead(type,id);}
function re212Denied(reason,more={}){return {allowed:false,partial:false,reason,trace:[],checks:[],scopeIds:[],...more};}
function re212Capability(permission,options={}){
 gvInit();
 const userId=options.userId===undefined?recActor().id:options.userId,subject=options.subject;
 if(!GV_MODULES.some(([m])=>GV_ACTIONS.some(([a])=>permission===m+'.'+a)))return re212Denied('Unknown capability.');
 if(!recUser(userId))return re212Denied('The working identity is unavailable.');
 let scopeIds;
 if(re212Own(options,'scopeIds'))scopeIds=Array.isArray(options.scopeIds)?options.scopeIds:[];
 else if(re212Own(options,'scopeId'))scopeIds=[options.scopeId];
 else if(subject){const record=re212Subject(subject.type,subject.id);if(!record)return re212Denied('The source record is unavailable.');scopeIds=re212ScopeList(subject.type,record);}
 else scopeIds=[selectedScope()];
 scopeIds=[...new Set(scopeIds)];
 if(!scopeIds.length||scopeIds.some(id=>id!==''&&!orgNode(id)))return re212Denied('The accountable location is missing or invalid.',{scopeIds});
 if(S.scopeInvalid)return re212Denied('The selected organizational scope is invalid. Choose a valid location.',{scopeIds});
 const checks=scopeIds.map(scopeId=>({scopeId,...gvDecision(userId,permission,scopeId,S.asOf,{makerId:options.makerId})}));
 const allowed=checks.every(x=>x.allowed),partial=!allowed&&checks.some(x=>x.allowed),first=checks.find(x=>!x.allowed);
 return {allowed,partial,permission,userId,scopeIds,checks,trace:checks.flatMap(check=>check.trace.map(trace=>({...trace,scopeId:check.scopeId}))),reason:allowed?'Current grants cover '+permission+' at every accountable location.':(first?.scopeId?orgName(first.scopeId)+': ':'')+(first?.reason||'No active grant covers this action.')};
}
function re212Coverage(permission,options={}){
 const scopeId=re212Own(options,'scopeId')?options.scopeId:selectedScope(),userId=options.userId===undefined?recActor().id:options.userId;
 if(S.scopeInvalid||scopeId!==''&&!orgNode(scopeId)||!recUser(userId))return {any:false,full:false,scopeId,allowedScopeIds:[],deniedScopeIds:[],roots:[],checks:[],reason:'The identity or selected organizational scope is invalid.'};
 const ids=ORG.filter(n=>!scopeId||orgWithin(n.id,scopeId)).map(n=>n.id),checks=ids.map(id=>({scopeId:id,...re212Capability(permission,{scopeId:id,userId})}));
 const allowedScopeIds=checks.filter(x=>x.allowed).map(x=>x.scopeId),deniedScopeIds=checks.filter(x=>!x.allowed).map(x=>x.scopeId),allowed=new Set(allowedScopeIds);
 const complete=ids.filter(id=>allowed.has(id)&&ids.filter(child=>orgWithin(child,id)).every(child=>allowed.has(child)));
 const roots=complete.filter(id=>!complete.some(parent=>parent!==id&&orgWithin(id,parent)));
 return {any:allowedScopeIds.length>0,full:ids.length>0&&!deniedScopeIds.length,scopeId,userId,allowedScopeIds,deniedScopeIds,roots,checks,reason:!allowedScopeIds.length?'No active '+permission+' grant covers this location.':deniedScopeIds.length?'Access covers part of this location; denied and unassigned areas remain excluded from permission.':'Current grants cover every location in this scope.'};
}
function re212Require(permission,options={}){const decision=re212Capability(permission,{...options,userId:recActor().id});requireValue(decision.allowed,decision.reason);return decision;}
function re212NoDeny(permission,scopeIds,userId=recActor().id){const check=re212Capability(permission,{scopeIds,userId});return check.scopeIds.length>0&&!S.scopeInvalid&&check.checks.length===check.scopeIds.length&&!check.trace.some(x=>x.effect==='deny');}
function re212CanViewSubject(type,id){
 const record=re212Subject(type,id),permission=(RE212_MODULES[type]||'')+'.view';
 if(!record||!RE212_MODULES[type])return re212Denied('The source record is unavailable.');
 if(type==='report'&&window.CLEAR_SIGHT_SCOPED_ANALYTICS?.snapshotAccess)return window.CLEAR_SIGHT_SCOPED_ANALYTICS.snapshotAccess(record.snapshot,'reports.view');
 if(!recInScope(type,record))return re212Denied('The source is outside the selected organizational location.');
 if(type==='document'&&record.scopeMode==='shared'){
  const coverage=re212Coverage('documents.view');
  return {allowed:coverage.any,partial:coverage.any&&!coverage.full,reason:coverage.any?'This shared reference is readable within a location covered by your document-view access. Shared-document changes and exports require their own whole-organization authority.':coverage.reason,scopeIds:coverage.allowedScopeIds,trace:coverage.checks.flatMap(x=>x.trace),checks:coverage.checks,sharedReference:true};
 }
 return re212Capability(permission,{subject:{type,id}});
}
function re212State(){S.roleExperience??={version:1,lenses:{}};S.roleExperience.lenses??={};return S.roleExperience;}
function re212OwnedPrograms(){const actor=recActor().id;return S.programs.filter(p=>re212Id(p.owner)===actor&&recInScope('program',p)&&re212ScopeList('program',p).length&&re212NoDeny('programs.edit',re212ScopeList('program',p)));}
function re212Assignments(){return typeof atInit==='function'?atInit().assignments.filter(a=>atVisible(a)&&!a.retired):[];}
function re212Context(){
 const actor=re212Identity(),coverage=Object.fromEntries(GV_MODULES.flatMap(([m])=>GV_ACTIONS.map(([a])=>[m+'.'+a,re212Coverage(m+'.'+a)])));
 const assignments=re212Assignments(),mine=assignments.filter(a=>(a.respondent?.kind==='external'?a.coordinatorId:a.respondent?.id)===actor.id&&atCanRespond(a)),ownedPrograms=re212OwnedPrograms();
 const currentRoles=gvState().assignments.filter(a=>a.userId===actor.id&&gvActive(a)&&gvRole(a.roleId)?.status==='Active'&&(!selectedScope()||gvOverlap(a.scopeId,selectedScope()))).map(a=>({id:a.id,roleId:a.roleId,name:gvRole(a.roleId).name,scopeId:a.scopeId,scopeLabel:gvScope(a.scopeId),ends:a.ends||'',reviewDue:a.reviewDue}));
 const lenses=[];
 if(['programs','kri','risks','documents','vendors'].some(m=>coverage[m+'.edit'].any||coverage[m+'.submit'].any)||mine.length)lenses.push({id:'contributor',label:'Capture & respond',description:'Your assigned responses, corrections and record preparation.'});
 if(ownedPrograms.length)lenses.push({id:'owner',label:'Program ownership',description:'Named respondents, overdue work and requirement decisions for programs you own.'});
 if(['programs','kri','risks','documents','vendors'].some(m=>coverage[m+'.approve'].any)||assignments.some(a=>a.reviewerId===actor.id&&atReviewerEligibility(a.programId,a.target,actor.id).allowed))lenses.push({id:'reviewer',label:'Independent review',description:'Submitted work awaiting your independent decision.'});
 if(coverage['reports.view'].any)lenses.push({id:'leadership',label:'Leadership reporting',description:'Scoped dashboards, source insight and published management reports.'});
 if(coverage['access.manage'].any)lenses.push({id:'administrator',label:'Access administration',description:'Review grants, compare changes and maintain escalation routing.'});
 if(!lenses.length)lenses.push({id:'reader',label:'Read & follow up',description:'Inspect available records and follow your assigned work.'});
 const available=lenses.length>1?[{id:'combined',label:'All responsibilities',description:'One queue across your active responsibilities.'},...lenses]:lenses;
 const preferred=re212State().lenses[actor.id],lens=available.find(x=>x.id===preferred)||available[0];
 return {actor,scopeId:selectedScope(),scopeLabel:scopeLabel(),asOf:S.asOf,coverage,currentRoles,lenses:available,lens,ownedPrograms,assignments:mine};
}

window.CLEAR_SIGHT_ROLE_EXPERIENCE=Object.freeze({version:'2.12',identity:re212Identity,capability:re212Capability,coverage:re212Coverage,require:re212Require,canViewSubject:re212CanViewSubject,context:re212Context});

/* Bind previously name-only commands to the working identity. These checks run
 * at commit as well as at the visible control; existing version, evidence and
 * independent-decision validations run unchanged inside the original command. */
function re212NeedSubject(permission,type,id,options={}){
 const record=re212Subject(type,id);
 requireValue(record&&recInScope(type,record),'The source record is unavailable in the selected location.');
 re212Require(permission,{...options,scopeIds:re212ScopeList(type,record)});
 return record;
}
function re212NeedActor(named,role='assigned owner'){
 requireValue(re212Id(named)===recActor().id,'Continue as the '+role+' ('+(recUser(named)?.name||named||'not assigned')+'). Selecting another name does not change your working identity.');
}
function re212NeedReviewer(form,named,permission,scopes,maker){
 if(named)re212NeedActor(named,'assigned reviewer');
 requireValue(re212Id(values(form).reviewer)===recActor().id,'The recorded reviewer must match your working identity.');
 return re212Require(permission,{scopeIds:scopes,makerId:re212Id(maker)});
}
function re212Dialog(check,open){
 check();const actorId=recActor().id,scopeId=selectedScope(),before=modalSave,result=open();
 if(modalSave&&modalSave!==before){const commit=modalSave;modalSave=function(form){requireValue(recActor().id===actorId&&selectedScope()===scopeId,'The working identity or location changed. Reopen this action in the correct context.');check(form);return commit(form);};}
 return result;
}
const RE212_PRIOR={saveControl:wf29SaveControl,submitTest:wf29SubmitTest,testDecision:wf29TestDecision,saveRisk:wf29SaveRisk,rcsaDraft:wf29SaveRcsaDraft,rcsaSubmit,rcsaDecision,kriSubmit:ixSubmitReturn,kriReview:ixReviewReturn,observe:observeIndicator,operation:jSubmitOperation,activitySubmit:jSubmitActivity,activityDecision:jActivityDecision,dpiaSubmit:jSubmitDpia,dpiaDecision:jDpiaDecision,caseUpdate:jCaseUpdate,caseDecision:jCaseDecision,publishReview,atCanManage,atCanRespond,atAssign,formCreate:faCreate,formPublish:faPublish,formSettings:faSettings,formAdd:faAdd,formColumns:faColumns,formApply:faApplySettings};
wf29SaveControl=function(id){const d=wf29Draft('control',id);requireValue(d,'Control draft unavailable.');const scopes=[d.values.orgScopeId];if(id!=='new'){const c=recRead('control',id);requireValue(c&&recInScope('control',c),'This control is outside the selected location.');scopes.push(...re212ScopeList('control',c));}re212Require('risks.edit',{scopeIds:scopes});return RE212_PRIOR.saveControl(id);};
wf29SubmitTest=function(id){const c=re212NeedSubject('risks.submit','control',id),d=wf29Draft('control-test',id);requireValue(d,'Test draft unavailable.');re212NeedActor(d.values.tester,'test preparer');const reviewer=re212Capability('risks.approve',{scopeIds:re212ScopeList('control',c),userId:re212Id(d.values.reviewer),makerId:recActor().id});requireValue(reviewer.allowed,'Choose an independent reviewer with current control-review access. '+reviewer.reason);return RE212_PRIOR.submitTest(id);};
wf29TestDecision=function(form){const test=wf29State().controlTests.find(x=>x.id===form.dataset.id);requireValue(test,'Test not found.');const c=re212NeedSubject('risks.approve','control',test.controlId,{makerId:re212Id(test.tester)});re212NeedReviewer(form,test.reviewer,'risks.approve',re212ScopeList('control',c),test.tester);return RE212_PRIOR.testDecision(form);};
wf29SaveRisk=function(id){re212NeedSubject('risks.edit','risk',id);return RE212_PRIOR.saveRisk(id);};
wf29SaveRcsaDraft=function(cycleId,riskId,values){const c=recRead('rcsa',cycleId);requireValue(c,'RCSA cycle unavailable.');re212NeedActor(c.owner,'first-line assessment owner');re212NeedSubject('risks.edit','risk',riskId);return RE212_PRIOR.rcsaDraft(cycleId,riskId,values);};
rcsaSubmit=function(form){const {c,r}=wf29RcsaContext(form.dataset.cycle,form.dataset.risk);re212NeedActor(c.owner,'first-line assessment owner');re212NeedSubject('risks.submit','risk',r.id);const reviewer=re212Capability('risks.approve',{scopeIds:re212ScopeList('risk',r),userId:re212Id(c.reviewer),makerId:recActor().id});requireValue(reviewer.allowed,'The assigned RCSA reviewer needs current independent review access. '+reviewer.reason);return RE212_PRIOR.rcsaSubmit(form);};
rcsaDecision=function(form){const {c,r}=wf29RcsaContext(form.dataset.cycle,form.dataset.risk);re212NeedReviewer(form,c.reviewer,'risks.approve',re212ScopeList('risk',r),c.reviews[r.id]?.submittedBy);return RE212_PRIOR.rcsaDecision(form);};
function re212KriPrepare(permission='kri.edit',record=ixCurrentReturn()) {requireValue(record&&ixMatch(record.unit),'Choose a return in the selected location.');re212NeedActor(record.owner,'branch-return owner');return re212Require(permission,{scopeId:record.unit});}
ixSubmitReturn=function(){const r=ixCurrentReturn();return re212Dialog(()=>{requireValue(ixCurrentReturn()?.id===r?.id,'The return context changed. Reopen the source.');re212KriPrepare('kri.submit',r);},()=>RE212_PRIOR.kriSubmit());};
ixReviewReturn=function(form){const r=ixCurrentReturn();requireValue(r&&ixMatch(r.unit),'Choose a return in the selected location.');const c=ixCycle(r.cycle),submission=r.submissions?.at(-1);re212NeedReviewer(form,c.reviewer,'kri.approve',[r.unit],submission?.submittedBy||r.owner);return RE212_PRIOR.kriReview(form);};
observeIndicator=function(id){return re212Dialog(()=>re212NeedSubject('kri.edit','indicator',id),()=>RE212_PRIOR.observe(id));};
jSubmitOperation=function(){const d=jDraft();if(['risk','cycle'].includes(d?.kind)){const scopes=d.kind==='risk'?[d.data.orgScopeId]:d.data.risks.map(id=>recRead('risk',id)?.orgScopeId);re212Require('risks.edit',{scopeIds:scopes});re212NeedActor(d.data.owner,d.kind==='risk'?'initial risk owner':'cycle owner');if(d.kind==='cycle'){const reviewer=re212Capability('risks.approve',{scopeIds:scopes,userId:re212Id(d.data.reviewer),makerId:recActor().id});requireValue(reviewer.allowed,'Assign an independent reviewer with access to the complete cycle population. '+reviewer.reason);}}return RE212_PRIOR.operation();};
// Keep the superseded dialog entry points on the same scoped setup command.
newRisk=()=>jStartOperation('risk');newCycle=()=>jStartOperation('cycle');
function re212PrivacySubmit(kind){const d=jDraft();requireValue(d?.data,'The saved privacy draft is unavailable.');re212NeedActor(d.data.owner,'privacy record owner');re212Require('vendors.submit',{scopeId:d.data.orgScopeId});if(d.recordId)re212NeedSubject('vendors.submit',kind==='activity'?'activity':'dpia',d.recordId);const reviewer=re212Capability('vendors.approve',{scopeId:d.data.orgScopeId,userId:re212Id(d.data.reviewer),makerId:recActor().id});requireValue(reviewer.allowed,'Assign an independent privacy reviewer with access to this location. '+reviewer.reason);}
jSubmitActivity=function(){re212PrivacySubmit('activity');return RE212_PRIOR.activitySubmit();};
jSubmitDpia=function(){re212PrivacySubmit('dpia');return RE212_PRIOR.dpiaSubmit();};
jActivityDecision=function(form){const x=re212NeedSubject('vendors.approve','activity',form.dataset.id),d=jData(x);re212NeedReviewer(form,d.reviewer,'vendors.approve',re212ScopeList('activity',x),x.submittedBy||d.owner);return RE212_PRIOR.activityDecision(form);};
jDpiaDecision=function(form){const x=re212NeedSubject('vendors.approve','dpia',form.dataset.id),d=jData(x);re212NeedReviewer(form,d.reviewer,'vendors.approve',re212ScopeList('dpia',x),x.submittedBy||d.owner);return RE212_PRIOR.dpiaDecision(form);};
jCaseUpdate=function(form){const c=re212NeedSubject(values(form).status==='Submit for independent review'?'vendors.submit':'vendors.edit','privacyCase',form.dataset.id);re212NeedActor(c.owner,'case owner');return RE212_PRIOR.caseUpdate(form);};
jCaseDecision=function(form){const c=re212NeedSubject('vendors.approve','privacyCase',form.dataset.id);re212NeedReviewer(form,c.reviewer,'vendors.approve',re212ScopeList('privacyCase',c),c.owner);return RE212_PRIOR.caseDecision(form);};
function re212AdoptionScopes(){const d=S.draft;return d?.scopeType==='vendor'?d.targets.flatMap(id=>re212ScopeList('engagement',recRead('engagement',id))):[selectedScope()];}
publishReview=function(){return re212Dialog(()=>re212Require('programs.edit',{scopeIds:re212AdoptionScopes()}),()=>RE212_PRIOR.publishReview());};

/* A verified named assignment is a domain entitlement, not a role title. Keep
 * that entitlement while honoring an explicit restriction at its exact target. */
atCanManage=function(a){return !!a&&re212NoDeny('programs.edit',atScope(recRead('program',a.programId),a.target))&&RE212_PRIOR.atCanManage(a);};
atCanRespond=function(a){return !!a&&re212NoDeny('programs.edit',atScope(recRead('program',a.programId),a.target))&&re212NoDeny('programs.submit',atScope(recRead('program',a.programId),a.target))&&RE212_PRIOR.atCanRespond(a);};
atAssign=function(config){requireValue(atCanManage({programId:config.programId,target:config.target}),'The actual program owner or eligible risk reviewer must manage this exact assignment, and explicit target restrictions must permit it.');return RE212_PRIOR.atAssign(config);};
window.CLEAR_SIGHT_ASSESSMENTS=Object.freeze({...window.CLEAR_SIGHT_ASSESSMENTS,assign:atAssign});

// The form catalogue is shared across the organization; target-specific
// responses keep their separate named-assignment permissions.
function re212FormDefinition(){return re212Require('programs.edit',{scopeId:''});}
faCreate=function(){return re212Dialog(re212FormDefinition,()=>RE212_PRIOR.formCreate());};
faPublish=function(id){return re212Dialog(re212FormDefinition,()=>RE212_PRIOR.formPublish(id));};
faSettings=function(id){return re212Dialog(re212FormDefinition,()=>RE212_PRIOR.formSettings(id));};
faAdd=function(){return re212Dialog(re212FormDefinition,()=>RE212_PRIOR.formAdd());};
faColumns=function(id){return re212Dialog(re212FormDefinition,()=>RE212_PRIOR.formColumns(id));};
faApplySettings=function(form){re212FormDefinition();return RE212_PRIOR.formApply(form);};

/* Discoverability follows capabilities, while the responsibility lens only
 * changes the order and emphasis of the workspace below. */
const RE212_ROUTES={
 home:'reports',frameworks:'programs',adopt:'programs',programs:'programs',program:'programs',assessment:'programs','assessment-review':'programs','assessment-history':'programs','requirement-review':'programs',forms:'programs','form-builder':'programs','form-response':'programs','check-runs':'programs','external-assessments':'programs','external-assessment-request':'programs',
 vendors:'vendors',vendor:'vendors','vendor-intake':'vendors','vendor-onboarding':'vendors','vendor-review':'vendors','vendor-requests':'vendors',privacy:'vendors',activity:'vendors','activity-version':'vendors','activity-wizard':'vendors',dpia:'vendors','dpia-wizard':'vendors','privacy-case':'vendors','case-intake':'vendors',
 risks:'risks',risk:'risks','risk-edit':'risks',losses:'risks',loss:'risks','loss-capture':'risks','loss-import':'risks',controls:'risks',control:'risks','control-editor':'risks','control-test':'risks',rcsa:'risks','rcsa-review':'risks',setup:'risks',
 indicators:'kri',indicator:'kri','kri-returns':'kri','kri-return':'kri','kri-cycle':'kri','kri-library':'kri',
 documents:'documents',document:'documents',evidence:'documents','evidence-review':'documents','document-requests':'documents','document-request':'documents',
 reports:'reports','report-builder':'reports','report-pack':'reports','loss-insights':'reports','insight-detail':'reports',
 roles:'access','role-detail':'access','role-change-review':'access','access-review':'access','escalation-matrix':'escalation','escalation-policy':'escalation'
};
const RE212_SUBJECT_ROUTES={program:'program',risk:'risk','risk-edit':'risk',loss:'loss',control:'control','control-editor':'control','control-test':'control',vendor:'engagement','vendor-review':'engagement','vendor-requests':'engagement',activity:'activity','activity-version':'activity',dpia:'dpia','privacy-case':'privacyCase','kri-return':'kriReturn',indicator:'indicator',document:'document','evidence-review':'evidence','report-pack':'report'};
function re212NamedProgramAccess(programId,target=''){
 const p=recRead('program',programId);if(!p||!recInScope('program',p))return false;
 const scopes=target?atScope(p,target):re212ScopeList('program',p);if(!scopes.length||!re212NoDeny('programs.view',scopes))return false;
 if(re212Id(p.owner)===recActor().id)return true;
 return re212Assignments().some(a=>a.programId===programId&&(!target||a.target===target)&&[a.respondent?.id,a.coordinatorId,a.reviewerId].includes(recActor().id));
}
function re212RouteDecision(route){
 const [name,query='']=String(route||'').replace(/^#?\//,'').split('?'),params=new URLSearchParams(query);
 if(['my-workspace','my-access','work','work-item','inbox','notification','notification-preferences','settings','not-found'].includes(name))return {allowed:true,reason:'Personal workspace.'};
 let module=RE212_ROUTES[name];
 if(name==='setup'){const draft=S.journeyDrafts?.[params.get('draft')];module=draft?.kind==='vendor'?'vendors':draft?.kind==='loss'||draft?.kind==='risk'||draft?.kind==='cycle'?'risks':module;}
 if(!module)return {allowed:true,reason:'The source workflow supplies its access checks.'};
 if(['program','assessment','requirement-review'].includes(name)&&re212NamedProgramAccess(params.get('id'),params.get('target')||''))return {allowed:true,reason:'Verified named program responsibility.'};
 if(['assessment-review','assessment-history'].includes(name)){const submission=params.get('submission')&&atSubmissionRaw(params.get('submission')),assignment=atAssignmentRaw(params.get('assignment'))||submission&&atAssignmentRaw(submission.assignmentId);if(assignment&&re212NamedProgramAccess(assignment.programId,assignment.target))return {allowed:true,reason:'Verified named assessment responsibility.'};}
 if(name==='form-response'&&params.has('binding')){const binding=faInit().bindings.find(b=>b.id===params.get('binding')),a=binding&&atAssignmentRaw(binding.trackingAssignmentId);if(a&&re212NamedProgramAccess(a.programId,a.target))return {allowed:true,reason:'Verified named response responsibility.'};}
 const type=RE212_SUBJECT_ROUTES[name],id=params.get('id');
 if(type&&id&&id!=='new')return re212CanViewSubject(type,id);
 const access=re212Coverage(module+'.view');
 if(module==='programs'&&!access.any&&(re212OwnedPrograms().length||re212Assignments().some(a=>[a.respondent?.id,a.coordinatorId,a.reviewerId].includes(recActor().id)&&re212NoDeny('programs.view',a.scopeIds))))return {allowed:true,reason:'Verified named program work is available.'};
 return {allowed:access.any,partial:access.any&&!access.full,reason:access.reason};
}
function re212Nav(){
 const rows=[['my-workspace','My workspace','home'],['home','Scope dashboard','chart'],['inbox','Inbox','bell'],['work','Work & approvals','work'],['programs','Programs','program'],['forms','Forms & assessments','form'],['vendors','Vendors','vendors'],['risks','Risk & losses','risk'],['privacy','Privacy','lock'],['documents','Documents','file'],['reports','Reports & insights','chart'],['frameworks','Framework library','layers'],['roles','Roles & access','shield'],['escalation-matrix','Escalation matrix','clock']];
 return rows.filter(([route])=>re212RouteDecision(route).allowed);
}
function re212ScopeHint(coverage){return coverage.full?'Complete selected scope':coverage.any?'Access in part of this scope':'Unavailable in this scope';}
function re212ReadOnly(reason,title='Read-only for your current access'){
 return `<section class="re212-restriction" role="status"><span>${icon('lock')}</span><div><strong>${e(title)}</strong><p>${e(reason)}</p></div>${link('Inspect my access','my-access','quiet small')}</section>`;
}
function re212RouteGuard(){const check=re212RouteDecision(current.name+(current.params.size?'?'+current.params.toString():''));if(check.allowed)return '';current.blocked=true;return header('Access is unavailable for this location',e(check.reason),link('My workspace','my-workspace','primary'),'Current role access')+re212ReadOnly('Use the organizational selector to choose a location covered by your active grants, or ask an access administrator to review the specific permission. A responsibility lens does not change access.','Keep the selected location explicit');}
const re212PriorUxGuard=uxRouteGuard;
uxRouteGuard=function(){return re212PriorUxGuard()||re212RouteGuard();};

shell=function(){
 const groups={adopt:'frameworks',program:'programs',assessment:'programs',vendor:'vendors',rcsa:'risks','rcsa-review':'risks',indicators:'risks',losses:'risks',controls:'risks',evidence:'documents',integrations:'settings','form-builder':'forms','form-response':'forms','check-runs':'forms'},active=(window.CS_JOURNEY_GROUPS||{})[current.name]||groups[current.name]||current.name;
 const actor=re212Identity(),count=giWorkRows({owner:'me',lane:'open'}).length,nav=re212Nav(),unread=typeof inboxUnreadCount==='function'?inboxUnreadCount():0;
 $('#app').innerHTML=`<a href="#main-content" class="skip">Skip to main content</a><button class="mobile-shade" data-action="menu-close" aria-label="Close navigation"></button><aside class="sidebar" aria-label="Primary navigation"><div class="ix-sidebar-bank"><span class="bank-logo">M</span><div><strong>Meridian Bank</strong><small>Synthetic workspace</small></div></div><div class="nav-label">Your workspace</div><nav>${nav.map(([id,title,ic])=>`<a class="nav-link ${active===id?'active':''}" href="#/${id}" ${active===id?'aria-current="page"':''}>${icon(ic)}<span>${title}</span>${id==='work'?`<b class="count" aria-label="${count} open actions assigned to you">${count}</b>`:id==='inbox'?`<b class="count" data-n-unread-count ${unread?'':'hidden'}>${unread}</b>`:''}</a>`).join('')}</nav><div class="side-foot"><a class="nav-link ${current.name==='my-access'?'active':''}" href="#/my-access">${icon('shield')}<span>My access</span></a><a class="nav-link ${active==='settings'?'active':''}" href="#/settings">${icon('settings')}<span>Settings</span></a><div class="small faint re212-build">V2.15 · Demo workspace</div></div></aside><div class="workspace"><header class="topbar">${btn(icon('menu'),'menu','aria-label="Open navigation"','icon-only mobile-menu')}<a class="brand" href="#/my-workspace"><div class="logo-mark"><i class="a"></i><i class="b"></i><i class="c"></i></div><div><strong>ClearSight</strong><small>GOVERNANCE · RISK · COMPLIANCE</small></div></a>${scopeBar()}<div class="grow"></div><button class="search-trigger" data-action="search" aria-label="Search available workspaces and records">${icon('search')}<span>Search ClearSight</span><kbd>⌘ K</kbd></button>${btn(icon('sun'),'theme','aria-label="Toggle light or dark theme"','icon-only quiet')}<a class="profile re212-profile" href="#/my-access" title="Inspect access for ${e(actor.name)}"><div class="avatar">${e(actor.name.split(/\s+/).map(n=>n[0]).slice(0,2).join(''))}</div><div><strong>${e(actor.name)}</strong><small>Working identity · local demo</small></div></a></header><main id="main-content" class="page" tabindex="-1"></main></div>`;
};
function re212WorkDecision(row){
 if(row.source==='task'){const task=S.tasks.find(t=>t.id===row.id),subject=task&&flTaskSource(task);if(subject)return re212CanViewSubject(subject.type,subject.id);return {allowed:!!task,reason:'Assigned action.'};}
 return re212RouteDecision(row.route);
}
function re212WorkspaceRows(context,rows=workItems()){
 const all=giWorkRows({owner:'me',lane:'open'},rows),lens=context.lens.id;
 if(lens==='reviewer')return all.filter(uxIsReviewItem);
 if(lens==='contributor')return all.filter(row=>!uxIsReviewItem(row));
 if(lens==='owner'){const programIds=new Set(context.ownedPrograms.map(p=>p.id));return all.filter(row=>programIds.has(row.ref)||/assignment|conclusion|assessment|response/i.test(row.kind));}
 return all;
}
function re212QuickActions(context){
 const c=context.coverage,choices=[];
 if(['combined','contributor'].includes(context.lens.id)){
  if(c['kri.submit'].any)choices.push({title:'Prepare branch returns',description:'Open the return assigned to you, including saved corrections.',route:'kri-returns',icon:'form'});
  if(c['risks.edit'].any)choices.push({title:'Maintain risks & controls',description:'Edit in an authorized location; independent review controls rating changes.',route:'risks',icon:'risk'});
  if(c['vendors.edit'].any)choices.push({title:'Onboard a supplier',description:'Named respondents, required documents and an accountable service owner.',route:'vendors?tab=onboarding',icon:'vendors'});
  if(c['documents.view'].any)choices.push({title:'Manage documents',description:'Open files, versions, requests and exact contextual reviews.',route:'documents',icon:'file'});
 }
 if(['combined','reviewer'].includes(context.lens.id))choices.push({title:'Independent review queue',description:'Only source decisions close submitted reviews.',route:'work?owner=me&lane=reviews',icon:'shield'});
 if(['combined','leadership'].includes(context.lens.id)&&c['reports.view'].any)choices.push({title:'Review scoped insights',description:'The selected bank, branch, department or unit stays explicit in every view.',route:'home',icon:'chart'},{title:'Management reports',description:'Explore live findings or inspect a published, frozen board pack.',route:'reports',icon:'chart'});
 if(['combined','administrator'].includes(context.lens.id)&&c['access.manage'].any)choices.push({title:'Review people’s access',description:'Inspect expiring grants and compare a proposed access change before applying.',route:'access-review?state=overdue',icon:'users'},{title:'Review role policy',description:'Effective access reflects grant scope, active dates and explicit denies.',route:'roles',icon:'shield'});
 if(['combined','administrator','leadership'].includes(context.lens.id)&&c['escalation.manage'].any)choices.push({title:'Test escalation routing',description:'Resolve actual recipients, timing, fallback and stop conditions.',route:'escalation-matrix',icon:'clock'});
 if(!choices.length)choices.push({title:'Inspect available records',description:'Search the workspaces and sources available to your current identity.',action:'search',icon:'search'});
 return choices.filter(x=>!x.route||re212RouteDecision(x.route).allowed);
}
function re212WorkspaceView(){
 const context=re212Context(),rows=re212WorkspaceRows(context),shown=rows.slice(0,10),quick=re212QuickActions(context),isOwner=['combined','owner'].includes(context.lens.id)&&context.ownedPrograms.length;
 const queue=shown.length?table(['Your next action','Source / responsibility','Due',''],shown.map(row=>{const due=giDue(row.due),access=re212WorkDecision(row);return {cells:[`<strong>${e(row.title)}</strong>${!access.allowed?`<small class="re212-blocked">${e(access.reason)}</small>`:''}`,`${e(giWorkType(row))}<small>${e(row.ref||row.id)}</small>`,pill(due.label,due.tone),access.allowed?link(giWorkAction(row),row.source==='task'?'work-item?id='+encodeURIComponent(row.id):row.route,'small'):link('Inspect access','my-access','quiet small')]};})):empty('No open actions assigned to you in this view','Your selected responsibility and organizational location remain explicit. You can inspect the scoped team queue or another responsibility.',link('Open work queue','work?owner=me','small'));
 const owned=isOwner?panel('Programs you are accountable for','Ownership comes from the program record. Assignment and approval permissions are checked at the exact assessment target.',table(['Program','Target / due','Response progress',''],context.ownedPrograms.map(p=>{const a=re212Assignments().filter(a=>a.programId===p.id),pending=a.filter(a=>!atStatus(a.id).ready).length;return {cells:[`<strong>${e(p.name)}</strong><small>${e(p.id)}</small>`,`${p.targets.length} organizational ${p.targets.length===1?'target':'targets'}<small>${date(p.due)}</small>`,`${a.length} named ${a.length===1?'response':'responses'}<small>${pending} need follow-up</small>`,link(a.length?'Manage named responses':'Assign respondents','program?id='+p.id+'&target='+encodeURIComponent(p.targets[0])+'&tab=assessments','small')]};}))):'';
 const roles=context.currentRoles.length?context.currentRoles.map(r=>`<span>${e(r.name)} <small>· ${e(r.scopeLabel)}</small></span>`).join(''):'<span>No active role assignment intersects this location.</span>';
 return header('My workspace','Start with the work assigned to '+e(context.actor.name)+' in '+e(context.scopeLabel)+'.',link('My access','my-access','quiet')+link('Inbox','inbox','small'),'Your responsibilities')+`<section class="re212-context"><div>${giActorControl()}<p>Working identity is a local demonstration choice.</p></div><div><span class="eyebrow">Active role assignments</span><div class="re212-role-list">${roles}</div></div></section><nav class="re212-lenses" aria-label="Responsibility lens">${context.lenses.map(l=>btn(e(l.label),'re212-lens',`data-lens="${l.id}" aria-pressed="${l.id===context.lens.id}"`,l.id===context.lens.id?'active':'quiet')).join('')}</nav><p class="re212-lens-note">${e(context.lens.description)} <span>Changing this view never grants access.</span></p><div class="re212-layout"><section class="stack" aria-label="Assigned responsibilities">${panel('Your next actions',rows.length+' open '+(rows.length===1?'action':'actions')+' · oldest deadlines first',queue,rows.length>10?link('All '+rows.length+' assigned actions','work?owner=me','small'):'')}${owned}</section><aside class="re212-shortcuts" aria-label="Available workspaces">${quick.map(item=>`<article class="panel re212-shortcut"><span class="re212-shortcut-icon">${icon(item.icon)}</span><h2>${e(item.title)}</h2><p>${e(item.description)}</p>${item.route?link('Open '+icon('arrow'),item.route,'quiet small'):btn('Search available records','search','','quiet small')}</article>`).join('')}</aside></div>`;
}
function re212AccessView(){
 const context=re212Context(),permission=current.params.get('permission')||'programs.view',scopeId=current.params.has('scope')?current.params.get('scope'):selectedScope(),detail=re212Capability(permission,{scopeId});
 const matrix=GV_MODULES.map(([module,label])=>({cells:[`<strong>${e(label.replace(' & evidence',''))}</strong>`,...GV_ACTIONS.map(([action])=>{const c=context.coverage[module+'.'+action],route='my-access?permission='+module+'.'+action+'&scope='+encodeURIComponent(selectedScope());return `<a class="re212-cap ${c.full?'allowed':c.any?'partial':'denied'}" href="#/${route}" aria-label="${e(label+' '+action+': '+re212ScopeHint(c))}">${c.full?'Allowed':c.any?'Partial':'—'}</a>`;})]}));
 const restrictions=gvState().directRules.filter(r=>r.userId===context.actor.id&&gvActive(r)&&(!selectedScope()||gvOverlap(r.scopeId,selectedScope())));
 return header('My access',e(context.actor.name)+' · active grants, explicit restrictions and named work.',link('My workspace','my-workspace'),'Working identity')+giActorControl()+panel('Capabilities in the selected location',e(context.scopeLabel)+'. Partial means only some constituent locations are covered. Source commands still evaluate their exact scope.',table(['Module',...GV_ACTIONS.map(([,label])=>label)],matrix))+`<div class="re212-access-grid">${panel('Why this action is available or blocked','Inspect the exact permission and location without changing the working identity.',`<form data-re212-access class="pad"><div class="field-grid">${select('permission','Capability',GV_MODULES.flatMap(([m,label])=>GV_ACTIONS.map(([a,title])=>[m+'.'+a,label+' · '+title])),permission)}${select('scope','Exact location',[['','Whole organization'],...ORG.map(n=>[n.id,orgName(n.id)])],scopeId)}</div><button class="btn small mt16" type="submit">Inspect decision</button></form><div class="re212-decision">${pill(detail.allowed?'Allowed':'Unavailable',detail.allowed?'green':'amber')}<p>${e(detail.reason)}</p>${detail.trace.length?table(['Policy source','Effect','Assignment scope'],detail.trace.map(t=>({cells:[e(t.source),pill(t.effect,t.effect==='deny'?'red':t.effect==='allow'?'green':''),e(t.scope)]}))):'<p class="small muted">No active assignment or direct rule matches this exact action and location.</p>'}</div>`)}${panel('Active assignments and restrictions','Dates are evaluated at the workspace snapshot: '+date(S.asOf)+'.',`<div class="pad">${context.currentRoles.length?context.currentRoles.map(r=>`<div class="re212-access-row"><strong>${e(r.name)}</strong><span>${e(r.scopeLabel)}</span><small>${r.ends?'Ends '+date(r.ends):'No end date'} · review ${date(r.reviewDue)}</small></div>`).join(''):notice('No active role assignment covers this selected location.','mini')}${restrictions.map(r=>`<div class="re212-access-row"><strong>${e(r.permission)} · ${e(r.effect)}</strong><span>${e(gvScope(r.scopeId))}</span><small>${e(r.reason)}</small></div>`).join('')}<p class="small muted">Named program ownership, respondent assignments and action ownership retain their own source checks. They do not grant unrelated record editing, file exports or access administration.</p></div>`)}</div>`;
}
Object.assign(titles,{'my-workspace':'My workspace','my-access':'My access'});
Object.assign(window.CS_JOURNEYS,{'my-workspace':re212WorkspaceView,'my-access':re212AccessView});
Object.assign(window.CS_JOURNEY_GROUPS,{'my-workspace':'my-workspace','my-access':'my-workspace'});
actions['re212-lens']=el=>{const context=re212Context();requireValue(context.lenses.some(x=>x.id===el.dataset.lens),'This responsibility is not available in the current identity and location.');re212State().lenses[context.actor.id]=el.dataset.lens;save();render();};
document.addEventListener('submit',event=>{if(!event.target.matches?.('[data-re212-access]'))return;event.preventDefault();const v=values(event.target);go('my-access?permission='+encodeURIComponent(v.permission)+'&scope='+encodeURIComponent(v.scope));});

const re212PriorSearchCatalog=ui211SearchCatalog,re212PriorSearchRows=ui211SearchRows;
ui211SearchCatalog=function(){const rows=re212PriorSearchCatalog().filter(row=>re212RouteDecision(row.route).allowed);rows.unshift({name:'My workspace',route:'my-workspace',note:'Assigned work and current responsibilities',kind:'Workspace',id:'',search:'my workspace assigned work current responsibilities'});return rows;};
// Recheck cached results after identity/grant changes as well as at navigation.
ui211SearchRows=function(query,catalog=ui211SearchCatalog()){return re212PriorSearchRows(query,catalog.filter(row=>re212RouteDecision(row.route).allowed));};

function re212Try(check){try{const value=check();return value?.allowed!==undefined?value:{allowed:true,reason:'Current authority permits this action.'};}catch(error){return re212Denied(error.message);}}
function re212NeedAny(permission){const c=re212Coverage(permission);requireValue(c.any,c.reason);return c;}
function re212InitialLocation(permission){const c=re212Coverage(permission);return c.allowedScopeIds.find(id=>id===selectedScope()&&orgNode(id)?.kind==='unit')||c.allowedScopeIds.find(id=>orgNode(id)?.kind==='unit')||c.allowedScopeIds[0]||'';}
function re212TaskCheck(id){const task=S.tasks.find(t=>t.id===id);requireValue(task,'The action is unavailable.');re212NeedActor(task.ownerId||task.owner,'action owner');const source=flTaskSource(task),record=source&&recRead(source.type,source.id);if(source)requireValue(record&&recInScope(source.type,record),'The related record is unavailable in this location. Open the source before completing this action.');const module=RE212_MODULES[source?.type]||'risks',scopes=[...new Set([...scopeIds(task,'tasks'),...(record?re212ScopeList(source.type,record):[])])];requireValue(scopes.length&&re212NoDeny(module+'.edit',scopes),'An explicit restriction or missing action location prevents completion. Inspect the source or ask the owner to correct its scope.');return task;}
const re212PriorTask=taskDetail,re212PriorReopen=flReopenTask;
taskDetail=function(id){const item=workItems().find(x=>x.id===id);requireValue(item,'This work item is unavailable in the current location.');if(item.source&&item.source!=='task')return re212PriorTask(id);const check=re212Try(()=>re212TaskCheck(id));if(!check.allowed){const task=S.tasks.find(t=>t.id===id),source=task&&flTaskSource(task),record=source&&recRead(source.type,source.id);return openDialog(item.title,'Action details · read-only',`<dl class="re212-task-facts"><dt>Owner</dt><dd>${e(item.owner)}</dd><dt>Deadline</dt><dd>${date(item.due)}</dd><dt>State</dt><dd>${e(item.status)}</dd><dt>Required outcome</dt><dd>${e(item.outcome||item.description||'Open the source to inspect the required action.')}</dd></dl>${re212ReadOnly(check.reason)}${record?link('Open source',recUrl(source.type,source.id),'small'):''}`,'',null);}return re212Dialog(()=>re212TaskCheck(id),()=>re212PriorTask(id));};
flReopenTask=function(id){return re212Dialog(()=>re212TaskCheck(id),()=>re212PriorReopen(id));};

const re212PriorStartOperation=jStartOperation,re212PriorStartControl=wf29StartControl,re212PriorStartTest=wf29StartTest,re212PriorStartRisk=wf29StartRiskEdit,re212PriorStartActivity=jStartActivity,re212PriorStartDpia=jStartDpia,re212PriorNewCase=jNewCase,re212PriorSubmitCase=jSubmitCase;
function re212NewJourney(open,permission,patch){const before=new Set(Object.keys(S.journeyDrafts||{})),value=open(),draft=Object.values(S.journeyDrafts||{}).find(d=>!before.has(d.id));if(draft){patch(draft);save();}return value;}
jStartOperation=function(kind,extra={}){if(!['risk','cycle'].includes(kind))return re212PriorStartOperation(kind,extra);re212NeedAny('risks.edit');return re212NewJourney(()=>re212PriorStartOperation(kind,extra),'risks.edit',d=>{d.data.owner=recActor().name;if(kind==='risk')d.data.orgScopeId=re212InitialLocation('risks.edit');});};
wf29StartControl=function(id='new'){if(id==='new')re212NeedAny('risks.edit');else re212NeedSubject('risks.edit','control',id);const had=!!wf29Draft('control',id),result=re212PriorStartControl(id);if(id==='new'&&!had){const draft=wf29Draft('control',id);if(draft){draft.values.owner=recActor().name;draft.values.orgScopeId=re212InitialLocation('risks.edit');save();}}return result;};
wf29StartTest=function(id){re212NeedSubject('risks.edit','control',id);return re212PriorStartTest(id);};
wf29StartRiskEdit=function(id){re212NeedSubject('risks.edit','risk',id);return re212PriorStartRisk(id);};
jStartActivity=function(id,startStep=1){if(id)re212NeedSubject('vendors.edit','activity',id);else re212NeedAny('vendors.edit');return re212NewJourney(()=>re212PriorStartActivity(id,startStep),'vendors.edit',d=>{if(!id){d.data.owner=recActor().name;d.data.orgScopeId=re212InitialLocation('vendors.edit');}});};
jStartDpia=function(activityId,dpiaId){const record=dpiaId?re212NeedSubject('vendors.edit','dpia',dpiaId):re212NeedSubject('vendors.edit','activity',activityId);return re212NewJourney(()=>re212PriorStartDpia(activityId,dpiaId),'vendors.edit',d=>{if(!dpiaId)d.data.owner=recActor().name;});};
jNewCase=function(type){re212NeedAny('vendors.edit');return re212NewJourney(()=>re212PriorNewCase(type),'vendors.edit',d=>{d.data.owner=recActor().name;d.data.orgScopeId=re212InitialLocation('vendors.edit');if(d.data.activity&&recRead('activity',d.data.activity)?.orgScopeId!==d.data.orgScopeId)d.data.activity='';});};
jSubmitCase=function(){const d=jDraft();requireValue(d?.data,'Privacy case draft unavailable.');re212NeedActor(d.data.owner,'case owner');const scopeId=d.data.activity?recRead('activity',d.data.activity)?.orgScopeId:d.data.orgScopeId;re212Require('vendors.submit',{scopeId});const reviewer=re212Capability('vendors.approve',{scopeId,userId:re212Id(d.data.reviewer),makerId:recActor().id});requireValue(reviewer.allowed,'Assign an independent reviewer with current privacy review access. '+reviewer.reason);return re212PriorSubmitCase();};

function re212WorkflowDraftCheck(kind,id){
 if(kind==='control'){const d=wf29Draft(kind,id);if(id==='new')return re212NeedAny('risks.edit');return re212NeedSubject('risks.edit','control',id);}
 if(kind==='risk')return re212NeedSubject('risks.edit','risk',id);
 if(kind==='control-test'){const c=re212NeedSubject('risks.edit','control',id),d=wf29Draft(kind,id);if(d)re212NeedActor(d.values.tester,'test preparer');return c;}
 if(kind==='vendor-review'){const t=recRead('engagement',id),d=wf29Draft(kind,id);requireValue(t&&voCanPrepare(recActor().id,t.orgScopeId,t,'vendors.edit'),'Your current identity cannot prepare this service review.');if(d)re212NeedActor(d.values.requestedBy,'review preparer');return t;}
}
function re212JourneyDraftCheck(){const d=jDraft();requireValue(d,'The saved setup draft is unavailable.');if(['risk','cycle'].includes(d.kind))return re212NeedAny('risks.edit');if(['activity','dpia','case'].includes(d.kind)){if(d.recordId)re212NeedSubject('vendors.edit',d.kind==='activity'?'activity':d.kind==='dpia'?'dpia':'privacyCase',d.recordId);else re212NeedAny('vendors.edit');if(d.data.owner)re212NeedActor(d.data.owner,'privacy draft owner');}return d;}
const RE212_ACTION_CHECKS={
 'new-risk':()=>re212NeedAny('risks.edit'),'new-cycle':()=>re212NeedAny('risks.edit'),'new-control':()=>re212NeedAny('risks.edit'),
 'wf29-control-edit':el=>re212NeedSubject('risks.edit','control',el.dataset.id),'wf29-control-test':el=>re212NeedSubject('risks.edit','control',el.dataset.id),'wf29-risk-edit':el=>re212NeedSubject('risks.edit','risk',el.dataset.id),'wf29-risk-save':el=>re212NeedSubject('risks.edit','risk',el.dataset.id),
 'wf29-control-save':el=>{const d=wf29Draft('control',el.dataset.id);requireValue(d,'Control draft unavailable.');return re212Require('risks.edit',{scopeIds:[d.values.orgScopeId,...(el.dataset.id!=='new'?re212ScopeList('control',recRead('control',el.dataset.id)):[])]});},
 'wf29-test-submit':el=>{re212NeedSubject('risks.submit','control',el.dataset.id);re212NeedActor(wf29Draft('control-test',el.dataset.id)?.values.tester,'test preparer');},
 'wf29-test-correct':el=>{const t=wf29State().controlTests.find(t=>t.id===el.dataset.id);requireValue(t,'The test is unavailable.');re212NeedSubject('risks.edit','control',t.controlId);re212NeedActor(t.tester,'test preparer');},
 'wf29-discard':el=>{const [kind,id]=el.dataset.key.split(':');return re212WorkflowDraftCheck(kind,id);},
 'wf29-save-exit':()=>re212WorkflowDraftCheck(current.name,current.params.get('id')),
 'wf29-rcsa-exit':el=>{re212NeedSubject('risks.edit','risk',el.dataset.risk);re212NeedActor(recRead('rcsa',el.dataset.cycle)?.owner,'first-line assessment owner');},
 'ix-start-return':el=>{const c=ixCycle(el.dataset.cycle);re212Require('kri.edit',{scopeId:el.dataset.unit});re212NeedActor(c?.owners[el.dataset.unit],'branch-return owner');},
 'ix-return-sample':()=>re212KriPrepare(),'ix-return-save':()=>re212KriPrepare(),'ix-return-submit':()=>re212KriPrepare('kri.submit'),'k28-answer':()=>re212KriPrepare(),'k28-correction':()=>re212KriPrepare(),'k28-link-evidence':()=>re212KriPrepare(),
 'ix-new-cycle':()=>re212NeedAny('kri.edit'),'ix-create-cycle':()=>re212Require('kri.edit',{scopeIds:S.branchKri.draftCycle?.population||[]}),
 'k28-policy':()=>re212Require('kri.manage',{scopeId:''}),'observe-indicator':el=>re212NeedSubject('kri.edit','indicator',el.dataset.id),
 'publish-review':()=>re212Require('programs.edit',{scopeIds:re212AdoptionScopes()}),
 'fa-create':re212FormDefinition,'fa-settings':re212FormDefinition,'fa-add':re212FormDefinition,'fa-add-type':re212FormDefinition,'fa-columns':re212FormDefinition,'fa-move':re212FormDefinition,'fa-remove':re212FormDefinition,'fa-publish':re212FormDefinition,'fa-export':()=>re212Require('programs.export',{scopeId:''}),
 'new-processing':()=>re212NeedAny('vendors.edit'),'j-edit-activity':el=>re212NeedSubject('vendors.edit','activity',el.dataset.id),'j-start-dpia':el=>re212NeedSubject('vendors.edit','activity',el.dataset.id),'j-edit-dpia':el=>re212NeedSubject('vendors.edit','dpia',el.dataset.id),'j-new-case':()=>re212NeedAny('vendors.edit'),'fl-reopen-case':el=>{const c=re212NeedSubject('vendors.edit','privacyCase',el.dataset.id);re212NeedActor(c.owner,'case owner');},
 'j-submit':re212JourneyDraftCheck,'j-save-exit':re212JourneyDraftCheck,'j-add-row':re212JourneyDraftCheck,'j-remove-row':re212JourneyDraftCheck,
 'gv-new-role':()=>re212Require('access.manage',{scopeId:''}),'gv-clone-role':()=>re212Require('access.manage',{scopeId:''}),'gv-save-role':()=>re212Require('access.manage',{scopeId:''}),'gi-apply-role':()=>re212Require('access.manage',{scopeId:''}),'gv-discard-role':()=>re212Require('access.manage',{scopeId:''}),
 'gv-assign-role':()=>re212Require('access.manage'),'gv-direct-rule':()=>re212Require('access.manage'),'gv-review-assignment':el=>re212Require('access.manage',{scopeId:gvState().assignments.find(a=>a.id===el.dataset.id)?.scopeId}),
 'gv-revoke-assignment':el=>re212Require('access.manage',{scopeId:gvState().assignments.find(a=>a.id===el.dataset.id)?.scopeId}),'gi-renew-access':el=>re212Require('access.manage',{scopeId:gvState().assignments.find(a=>a.id===el.dataset.id)?.scopeId}),'gv-revoke-direct':el=>re212Require('access.manage',{scopeId:gvState().directRules.find(d=>d.id===el.dataset.id)?.scopeId}),
 'gv-export-access':()=>{re212Require('access.export',{scopeId:''});requireValue(re212Coverage('access.export',{scopeId:''}).full,'The access export includes organization-wide policy and audit records. Complete export coverage is required.');},
 'gv-new-policy':()=>re212Require('escalation.manage'),'gv-save-policy':()=>re212Require('escalation.manage',{scopeId:gvState().draftPolicy?.scopeId}),
 'gv-add-level':()=>re212Require('escalation.manage',{scopeId:gvState().draftPolicy?.scopeId}),'gv-remove-level':()=>re212Require('escalation.manage',{scopeId:gvState().draftPolicy?.scopeId}),'gv-discard-policy':()=>re212Require('escalation.manage',{scopeId:gvState().draftPolicy?.scopeId}),
 'fl-task-reopen':el=>re212TaskCheck(el.dataset.id)
};
function re212ActionDecision(action,element={dataset:{}}){const check=RE212_ACTION_CHECKS[action];return check?re212Try(()=>check(element)):null;}
// Update captured function references before adding a shared check to the action.
Object.assign(actions,{'fa-create':()=>faCreate(),'fa-add':()=>faAdd(),'ix-return-submit':()=>ixSubmitReturn(),'publish-review':()=>publishReview(),'new-risk':()=>newRisk(),'new-cycle':()=>newCycle()});
for(const [action,check]of Object.entries(RE212_ACTION_CHECKS)){const original=actions[action];if(!original)continue;actions[action]=element=>re212Dialog(()=>check(element||{dataset:{}}),()=>original(element));}

function re212FormCheck(form){
 const d=form.dataset||{},id=d.id;
 if(d.wf29Submit==='control-decision'){const t=wf29State().controlTests.find(t=>t.id===id);requireValue(t,'Test unavailable.');re212NeedActor(t.reviewer,'assigned reviewer');return re212NeedSubject('risks.approve','control',t.controlId,{makerId:re212Id(t.tester)});}
 if(d.submit==='rcsa-submit'||d.submit==='rcsa-decision'){const c=recRead('rcsa',d.cycle),r=recRead('risk',d.risk);requireValue(c&&r,'RCSA source unavailable.');const review=d.submit==='rcsa-decision';re212NeedActor(review?c.reviewer:c.owner,review?'assigned reviewer':'first-line assessment owner');return re212NeedSubject('risks.'+(review?'approve':'submit'),'risk',r.id,{makerId:review?re212Id(c.reviews[r.id]?.submittedBy):''});}
 if(d.ixSubmit==='kri-review'){const r=ixCurrentReturn();requireValue(r,'Return unavailable.');re212NeedActor(ixCycle(r.cycle).reviewer,'assigned reviewer');return re212Require('kri.approve',{scopeId:r.unit,makerId:re212Id(r.submissions?.at(-1)?.submittedBy||r.owner)});}
 if(['activity-decision','dpia-decision','case-decision','case-update'].includes(d.jSubmit)){const type=d.jSubmit.startsWith('activity')?'activity':d.jSubmit.startsWith('dpia')?'dpia':'privacyCase',r=recRead(type,id);requireValue(r,'Privacy source unavailable.');const data=type==='privacyCase'?r:jData(r),review=d.jSubmit!=='case-update';if(!review||data.reviewer||type!=='privacyCase')re212NeedActor(review?data.reviewer:data.owner,review?'assigned reviewer':'case owner');return re212NeedSubject('vendors.'+(review?'approve':'edit'),type,id,{makerId:review?re212Id(r.submittedBy||data.owner):''});}
 if(form.id==='fa-field-settings')return re212FormDefinition();
 if(d.wf29Draft){const [kind,recordId]=d.wf29Draft.split(':');return re212WorkflowDraftCheck(kind,recordId);}
 return null;
}
function re212Disable(element,decision){
 if(!element||!decision||decision.allowed)return;
 element.dataset.re212Restricted='true';element.disabled=true;element.setAttribute?.('aria-disabled','true');element.setAttribute?.('title',decision.reason);
}
function re212Enhance(){
 if(current.blocked)return;
 const notices=[];
 for(const control of $$('[data-action]',$('#main-content'))){const check=re212ActionDecision(control.dataset.action,control);if(check&&!check.allowed){re212Disable(control,check);notices.push(check.reason);}}
 for(const form of $$('form',$('#main-content'))){const check=re212Try(()=>re212FormCheck(form));if(!check.allowed){form.insertAdjacentHTML?.('afterbegin',re212ReadOnly(check.reason));for(const control of $$('input,textarea,select,button[type="submit"]',form))re212Disable(control,check);notices.push(check.reason);}else if(form.matches?.('[data-wf29-submit="control-decision"],[data-submit="rcsa-decision"],[data-ix-submit="kri-review"],[data-j-submit="activity-decision"],[data-j-submit="dpia-decision"],[data-j-submit="case-decision"]')){const actorField=form.querySelector?.('select[name="reviewer"]');if(actorField){actorField.innerHTML=`<option value="${e(recActor().name)}">${e(recActor().name)} · your working identity</option>`;actorField.setAttribute?.('aria-label','Recorded reviewer: '+recActor().name);}}}
 if(current.name==='kri-return'&&['Draft','Changes requested'].includes(ixCurrentReturn()?.status)){const check=re212Try(()=>re212KriPrepare());if(!check.allowed){for(const control of $$('[data-k28-metric],[data-k28-return],[data-k28-meta],[data-ix-metric],[data-ix-detail]'))re212Disable(control,check);notices.push(check.reason);}}
 if(['activity-wizard','dpia-wizard','case-intake'].includes(current.name)){const check=re212Try(re212JourneyDraftCheck);if(!check.allowed){for(const control of $$('[data-j-path],[data-j-array]'))re212Disable(control,check);notices.push(check.reason);}}
 if(['role-detail','escalation-policy'].includes(current.name)){const role=current.name==='role-detail',check=re212Capability(role?'access.manage':'escalation.manage',{scopeId:role?'':gvState().draftPolicy?.scopeId});if(!check.allowed){for(const control of $$(role?'[data-gv-role],[data-gv-permission]':'[data-gv-policy],[data-gv-level]'))re212Disable(control,check);notices.push(check.reason);}}
 if(notices.length&&!$('#re212-current-access')){const reasons=[...new Set(notices)],summary=reasons.length===1?reasons[0]:reasons.slice(0,2).join(' ');$('.page-head,.j-intro')?.insertAdjacentHTML?.('afterend',`<div id="re212-current-access">${re212ReadOnly(summary,'Some changes are unavailable for your current identity')}</div>`);}
}
// Capture mutable draft events before legacy listeners can write to their store.
for(const name of ['input','change'])document.addEventListener(name,event=>{const el=event.target;if(!el.matches)return;let check=null;
 if(current.name==='kri-return'&&el.matches('[data-k28-metric],[data-k28-return],[data-k28-meta],[data-ix-metric],[data-ix-detail]'))check=()=>re212KriPrepare();
 const workflow=el.closest?.('[data-wf29-draft]');if(workflow&&el.matches('[data-wf29-field],[data-wf29-array]'))check=()=>re212FormCheck(workflow);
 if(el.matches('[data-j-path],[data-j-array]'))check=re212JourneyDraftCheck;
 if(current.name==='form-builder'&&el.closest?.('#fa-field-settings'))check=re212FormDefinition;
 if(el.matches('[data-gv-role],[data-gv-permission]'))check=()=>re212Require('access.manage',{scopeId:''});
 if(el.matches('[data-gv-policy],[data-gv-level]'))check=()=>re212Require('escalation.manage',{scopeId:gvState().draftPolicy?.scopeId});
 if(!check)return;const result=re212Try(check);if(!result.allowed){event.preventDefault();event.stopImmediatePropagation?.();toast(result.reason);render();}
},true);
const re212PriorRender=render;
render=function(){re212PriorRender();re212Enhance();};
