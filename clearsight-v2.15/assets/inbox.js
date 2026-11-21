/* ClearSight V2.8 — recipient-aware notification centre.
 * Local design simulation. Domain records remain the authority for resolution.
 * No email, push delivery, background scheduling or authentication is connected.
 */
'use strict';
const N_TYPES={mention:['Mentions','users'],program:['Programs','program'],quality:['Data quality','shield'],document:['Document expiry','file'],deadline:['Deadlines','clock'],approval:['Approvals','work'],escalation:['Escalations','risk']};
const N_VIEW_LABELS={inbox:'All notifications',unread:'Unread',actionable:'Needs action',mentions:'Mentions',snoozed:'Snoozed',archive:'Archived'};
let N_FILTER={view:'inbox',category:'',search:'',severity:'',source:''},N_SELECTED=new Set(),N_CONTEXT='';
const nActor=()=>recActor().id;
const nNow=()=>new Date().toISOString();
const nPerson=id=>recUser(id)?.name||id||'System';
const nUserId=value=>recUser(value)?.id||recUserId(value)||'';
const nDay=(value)=>String(value||'').slice(0,10);
function nPrefs(user=nActor()){
 const store=S.notificationsV28;
 return store.preferences[user]??={inApp:Object.fromEntries(Object.keys(N_TYPES).map(k=>[k,true])),email:'Daily digest',digestAt:'08:00',quietStart:'18:00',quietEnd:'08:00',weekdaysOnly:true};
}
function nMandatory(event){return event.type==='escalation'||event.type==='approval'||event.severity==='Critical'||(event.type==='deadline'&&event.due&&nDay(event.due)<S.asOf);}
function nReceipt(event,user=nActor()){
 event.receipts??={};
 return event.receipts[user]??={readAt:null,archivedAt:null,snoozedUntil:null,acknowledgedAt:null,delivered:nMandatory(event)||nPrefs(user).inApp[event.type]!==false};
}
function nRouteSubject(routeValue,sourceId){
 const [name,query='']=String(routeValue||'').replace(/^#\//,'').split('?'),p=new URLSearchParams(query),id=sourceId||p.get('id');
 const type={program:'program',assessment:'program','requirement-review':'program',risk:'risk',loss:'loss',vendor:'engagement',control:'control',document:'document','evidence-review':'evidence','kri-return':'kriReturn','rcsa-review':'rcsa',activity:'activity',dpia:'dpia','privacy-case':'privacyCase','report-pack':'report'}[name];
 return type&&id?{type,id}:null;
}
function nRecordFor(event){
 if(event.sourceType==='task')return S.tasks.find(t=>t.id===event.sourceId)||null;
 if(event.sourceType==='approval')return S.governance.requests.find(r=>r.id===event.sourceId)||null;
 const subject=event.subject||nRouteSubject(event.sourceRoute,event.sourceId);
 return subject?recRead(subject.type,subject.id):null;
}
function nScopeIds(event){
 const record=nRecordFor(event);
 if(event.sourceType==='task')return record?scopeIds(record,'tasks'):[];
 if(event.sourceType==='approval')return record?.orgScopeIds||[];
 const subject=event.subject||nRouteSubject(event.sourceRoute,event.sourceId);
 if(subject&&record)return recScope(subject.type,record);
 return event.orgScopeIds||(event.orgScopeId?[event.orgScopeId]:[]);
}
function nCanSee(event){
 if(S.scopeInvalid||!event.recipientIds.includes(nActor()))return false;
 const subject=event.subject||nRouteSubject(event.sourceRoute,event.sourceId),r=nRecordFor(event);
 if(subject?.type==='document'&&r?.classification==='Restricted'&&r.ownerId!==nActor())return false;
 if(subject?.type==='evidence'&&r?.documentVersionId){const d=recRead('document',docVersion(r.documentVersionId)?.documentId);if(d?.classification==='Restricted'&&d.ownerId!==nActor())return false;}
 const scope=selectedScope();
 if(scope){const ids=nScopeIds(event);if(!ids.length&&!event.shared)return false;if(ids.length&&!ids.some(id=>orgWithin(id,scope)||orgWithin(scope,id)))return false;}
 return nReceipt(event).delivered;
}
function nSourceAvailable(event){
 const r=nRecordFor(event);
 if(event.sourceType||event.subject||nRouteSubject(event.sourceRoute,event.sourceId))return !!r;
 const name=String(event.sourceRoute||'').split('?')[0];
 return !!titles[name];
}
function nResolved(event){
 const r=nRecordFor(event);
 if(event.sourceType==='task')return !!r&&r.status==='Completed';
 if(event.sourceType==='approval')return !!r&&r.status!=='Pending';
 const subject=event.subject||nRouteSubject(event.sourceRoute,event.sourceId);
 if(subject?.type==='kriReturn'&&r){
  const revision=event.sourceRevision;
  if(event.type==='approval')return r.status!=='Submitted'||(revision!=null&&Number(r.revision)!==Number(revision));
  if(event.type==='program'&&event.sourceState==='Changes requested')return r.status!=='Changes requested'||(revision!=null&&Number(r.revision)!==Number(revision));
  if(event.type==='quality'){
   const task=event.sourceTaskId?S.tasks.find(t=>t.id===event.sourceTaskId):event.kriObservation?S.tasks.find(t=>t.kriObservation===event.kriObservation):null;
   return !!task&&task.status==='Completed';
  }
 }
 if(subject?.type==='evidence'&&r){if(event.type==='document')return evQuality(r)==='Current';if(event.type==='quality'){const version=docVersion(r.documentVersionId);return !!version&&version.storage!=='reference';}if(event.type==='approval')return r.status!=='Submitted';}
 if(event.type==='mention')return !!event.threadResolvedAt;
 return !!event.resolvedAt;
}
function nStatus(event){
 const receipt=nReceipt(event);
 if(nResolved(event))return 'Source resolved';
 if(receipt.acknowledgedAt)return 'Acknowledged';
 return event.actionable?'Action required':'For information';
}
function nSourceRoute(event){
 if(event.sourceType==='task'){
  const t=nRecordFor(event);if(!t)return '';
  for(const type of ['risk','evidence','program','engagement','kriReturn']){const ref=recRead(type,t.ref);if(ref)return recUrl(type,ref.id);}
  return 'work';
 }
 return nSourceAvailable(event)?event.sourceRoute:'';
}
function nNormalize(event){
 const type=N_TYPES[event.type]?event.type:/(kri|exception|breach)/i.test(event.type||'')?'quality':'program';
 const suppliedRecipients=event.recipientIds||event.recipients||event.recipient||nActor(),recipientValues=Array.isArray(suppliedRecipients)?suppliedRecipients:[suppliedRecipients];
 const recipients=[...new Set(recipientValues.map(nUserId).filter(Boolean))];
 if(!recipients.length)throw Error('A notification needs a known recipient.');
 const routeValue=String(event.sourceRoute||'work').replace(/^#\//,'');
 if(!titles[routeValue.split('?')[0]])throw Error('Notification source route is unavailable.');
 const routeSubject=nRouteSubject(routeValue);
 const sourceType=event.sourceType||(routeValue.split('?')[0]==='work'&&S.tasks.some(t=>t.id===event.sourceId)?'task':undefined),sourceRecord=routeSubject?recRead(routeSubject.type,routeSubject.id):null;
 const sourceRevision=event.sourceRevision??(routeSubject?.type==='kriReturn'?sourceRecord?.revision:undefined),sourceState=event.sourceState??(routeSubject?.type==='kriReturn'?sourceRecord?.status:undefined);
 const severity={critical:'Critical',high:'High',medium:'Normal',normal:'Normal',warning:'Normal',low:'Low',info:'Low'}[String(event.severity||'Normal').toLowerCase()]||'Normal';
 if(routeSubject&&event.sourceId&&routeSubject.id!==String(event.sourceId))throw Error('Notification source ID does not match its route.');
 if(routeSubject&&event.subject&&(routeSubject.type!==event.subject.type||routeSubject.id!==event.subject.id))throw Error('Notification typed source does not match its route.');
 return {...event,sourceType,sourceRevision,sourceState,id:String(event.id||uid('NTF')),type,title:String(event.title||'Workspace update').slice(0,220),body:String(event.body||'').slice(0,4000),recipientIds:recipients,actor:nUserId(event.actor)||event.actor||'System',createdAt:event.createdAt||nNow(),sourceRoute:routeValue,subject:event.subject||nRouteSubject(routeValue,event.sourceId),severity,actionable:event.actionable??(sourceType==='task'||(routeSubject?.type==='kriReturn'&&sourceState==='Changes requested')||['quality','document','deadline','approval','escalation','mention'].includes(type)),receipts:event.receipts||{},thread:event.thread||[]};
}
function nPut(event){
 const normalized=nNormalize(event),items=S.notificationsV28.events,found=items.find(x=>x.id===normalized.id);
 if(found){const {receipts,thread,threadResolvedAt,threadResolvedBy,...update}=normalized;Object.assign(found,update);for(const id of found.recipientIds)nReceipt(found,id);return found;}
 items.unshift(normalized);for(const id of normalized.recipientIds)nReceipt(normalized,id);return normalized;
}
function inboxEmit(event){
 inboxInit(false);const result=nPut(event);save();return clone(result);
}
function nSync(){
 const base=S.asOf+'T08:00:00Z',oversight='USR-01';
 for(const task of S.tasks.filter(t=>t.status==='Open'&&t.due&&days(t.due,S.asOf)<=7)){
  const overdue=task.due<S.asOf,recipients=[oversight,nUserId(task.owner)].filter(Boolean);
  nPut({id:'NTF-TASK-'+task.id,type:'deadline',title:(overdue?'Overdue: ':'Due soon: ')+task.title,body:overdue?`The due date was ${date(task.due)}. Confirm the blocker, accountable owner and revised completion plan in the source action.`:`This action is due ${date(task.due)}. Check that its owner has the evidence and time needed to complete it.`,sourceType:'task',sourceId:task.id,sourceRoute:'work',recipientIds:recipients,orgScopeIds:scopeIds(task,'tasks'),severity:overdue?'High':'Normal',due:task.due,createdAt:base,actor:'Monitoring'});
  if(overdue&&days(S.asOf,task.due)>=2)nPut({id:'NTF-ESC-'+task.id,type:'escalation',title:'Escalation review: '+task.title,body:`This open action is ${days(S.asOf,task.due)} days overdue. This local example routes the unresolved item to the program owner for acknowledgement and a recovery plan. No message has been delivered externally.`,sourceType:'task',sourceId:task.id,sourceRoute:'work',recipientIds:[oversight],orgScopeIds:scopeIds(task,'tasks'),severity:'High',due:task.due,createdAt:base,actor:'Escalation preview'});
 }
 for(const ev of S.evidence){
  const recipients=[oversight,nUserId(ev.owner)].filter(Boolean),v=docVersion(ev.documentVersionId);
  if(ev.expires&&days(ev.expires,S.asOf)<=30){const expired=ev.expires<S.asOf;nPut({id:'NTF-EXP-'+ev.id,type:'document',title:(expired?'Expired document: ':'Document expires soon: ')+(v?.fileName||ev.name||ev.id),body:`${ev.question||'Documents'} · ${ev.program?recRead('program',ev.program)?.name||ev.program:'Independent evidence'}. Expiry ${date(ev.expires)}. ${expired?'Historical acceptance is retained; a fresh source is needed for current assurance.':'Arrange a replacement before the validity period ends.'}`,sourceId:ev.id,subject:{type:'evidence',id:ev.id},sourceRoute:'evidence-review?id='+encodeURIComponent(ev.id),recipientIds:recipients,severity:expired?'High':'Normal',due:ev.expires,createdAt:base,actor:'Document monitoring'});}
  if(ev.status==='Submitted')nPut({id:'NTF-EV-REVIEW-'+ev.id,type:'approval',title:'Document ready for review: '+(v?.fileName||ev.name||ev.id),body:'Inspect the source version and dates before accepting evidence. Evidence acceptance and the requirement conclusion are separate decisions.',subject:{type:'evidence',id:ev.id},sourceId:ev.id,sourceRoute:'evidence-review?id='+encodeURIComponent(ev.id),recipientIds:[ev.assignedReviewerId||nUserId(recRead('program',ev.program)?.reviewer)||'USR-03'],severity:'Normal',due:ev.expires,createdAt:ev.submittedAt||base,actor:ev.submittedById||nUserId(ev.owner)});
 }
 for(const request of S.governance.requests.filter(r=>r.status==='Pending')){
  const step=request.steps.find(s=>s.state==='Pending');if(!step)continue;
  nPut({id:'NTF-APR-'+request.id+'-'+step.id,type:'approval',title:'Review requested: '+request.title,body:request.message||'Inspect the exact submitted source version and record an independent decision.',sourceType:'approval',sourceId:request.id,sourceRoute:'approval?id='+encodeURIComponent(request.id),recipientIds:[step.actorId],orgScopeIds:request.orgScopeIds||[],shared:request.shared,severity:request.due<S.asOf?'High':'Normal',due:request.due,createdAt:request.createdAt,actor:request.requestedBy});
 }
 for(const p of S.programs.filter(p=>p.due&&days(p.due,S.asOf)<=21))nPut({id:'NTF-PROGRAM-DUE-'+p.id,type:'deadline',title:(p.due<S.asOf?'Program deadline passed: ':'Program deadline approaching: ')+p.name,body:`Assessment due ${date(p.due)}. Review respondent coverage, outstanding questions and evidence quality before submission.`,sourceId:p.id,subject:{type:'program',id:p.id},sourceRoute:'program?id='+encodeURIComponent(p.id),recipientIds:[nUserId(p.owner)].filter(Boolean),severity:p.due<S.asOf?'High':'Normal',due:p.due,createdAt:base,actor:'Program monitoring'});
 for(const activity of S.activity.slice(0,60)){const p=recRead('program',activity.ref);if(!p)continue;nPut({id:'NTF-ACTIVITY-'+activity.id,type:'program',title:'Program update: '+p.name,body:activity.text,sourceId:p.id,subject:{type:'program',id:p.id},sourceRoute:'program?id='+encodeURIComponent(p.id),recipientIds:[nUserId(p.owner),nUserId(p.reviewer)].filter(Boolean),actor:activity.actor,severity:'Normal',actionable:false,createdAt:activity.time||base});}
 for(const event of S.notificationsV28.events)for(const receipt of Object.values(event.receipts)){if(receipt.snoozedUntil&&receipt.snoozedUntil<=nNow())receipt.snoozedUntil=null;}
}
function inboxInit(sync=true){
 S.notificationsV28??={version:1,events:[],preferences:{},seeded:false};
 const store=S.notificationsV28;
 if(!store.seeded){
  const p=recRead('program','PRG-104'),ev=recRead('evidence','EV-01');
  if(p){
   nPut({id:'NTF-MENTION-01',type:'mention',title:'Ifeoma mentioned you in the access review',body:'@Amina Bello Please confirm the owner and target date for the privileged-access exception before we prepare the management update.',sourceId:p.id,subject:{type:'program',id:p.id},sourceRoute:'program?id='+p.id+'&target='+encodeURIComponent(p.targets[0])+'&tab=requirements',recipientIds:['USR-01'],actor:'USR-03',orgScopeIds:scopeIds(p,'programs'),severity:'Normal',createdAt:S.asOf+'T08:15:00Z',thread:[{id:'THREAD-SEED-01',actor:'USR-03',text:'@Amina Bello Please confirm the owner and target date for the privileged-access exception before we prepare the management update.',createdAt:S.asOf+'T08:15:00Z'}]});
   nPut({id:'NTF-PROGRAM-01',type:'program',title:'Your program is ready for a completeness review',body:'Bank assurance has respondent assignments and linked requirements. Review unassessed obligations, expired evidence and open actions before the next assessment deadline.',sourceId:p.id,subject:{type:'program',id:p.id},sourceRoute:'program?id='+p.id,recipientIds:['USR-01','USR-03'],actor:'Program monitoring',severity:'Low',actionable:false,createdAt:addDays(S.asOf,-1)+'T15:30:00Z'});
  }
  if(ev&&docVersion(ev.documentVersionId)?.storage==='reference')nPut({id:'NTF-QUALITY-01',type:'quality',title:'Original source file is missing for an observation',body:'The privileged-account extract has evidence metadata but no original uploaded file in this reference. Add the exact source file before relying on its contents in a new review. This is a source-availability defect, not a failed bank control.',sourceId:ev.id,subject:{type:'evidence',id:ev.id},sourceRoute:'evidence-review?id='+ev.id,recipientIds:['USR-01','USR-02'],actor:'Data quality monitoring',severity:'High',createdAt:S.asOf+'T07:20:00Z'});
  const ikeja=S.risks.find(r=>r.orgScopeId==='U-IK-PAY');
  if(ikeja)nPut({id:'NTF-MENTION-02',type:'mention',title:'Daniel mentioned you in the branch follow-up',body:'@Amina Bello Please review the payment exception and confirm whether the action plan addresses the recurring cause.',sourceId:ikeja.id,subject:{type:'risk',id:ikeja.id},sourceRoute:'risk?id='+ikeja.id,recipientIds:['USR-01'],actor:'USR-04',severity:'Normal',createdAt:addDays(S.asOf,-1)+'T11:30:00Z',thread:[{id:'THREAD-SEED-02',actor:'USR-04',text:'@Amina Bello Please review the payment exception and confirm whether the action plan addresses the recurring cause.',createdAt:addDays(S.asOf,-1)+'T11:30:00Z'}]});
  store.seeded=true;
 }
 if(sync)nSync();
 return store;
}
function nItems(){inboxInit();return S.notificationsV28.events.filter(nCanSee).sort((a,b)=>String(b.createdAt).localeCompare(String(a.createdAt)));}
function inboxUnreadCount(){return nItems().filter(x=>{const r=nReceipt(x);return !r.readAt&&!r.archivedAt&&!r.snoozedUntil;}).length;}
function nFiltered(){
 return nItems().filter(x=>{
  const r=nReceipt(x),view=N_FILTER.view;
  if(view==='archive'?!r.archivedAt:r.archivedAt)return false;
  if(view==='snoozed'?!r.snoozedUntil:view!=='archive'&&r.snoozedUntil)return false;
  if(view==='unread'&&r.readAt)return false;
  if(view==='actionable'&&(!x.actionable||nResolved(x)))return false;
  if(view==='mentions'&&x.type!=='mention')return false;
  if(N_FILTER.category&&x.type!==N_FILTER.category)return false;
  if(N_FILTER.severity&&x.severity!==N_FILTER.severity)return false;
  if(N_FILTER.source&&nSourceLabel(x)!==N_FILTER.source)return false;
  if(N_FILTER.search.trim()&&![x.title,x.body,nPerson(x.actor),x.sourceId,nScopeIds(x).map(orgName).join(' ')].join(' ').toLowerCase().includes(N_FILTER.search.trim().toLowerCase()))return false;
  return true;
 });
}
function nSourceLabel(event){if(event.sourceType==='task')return 'Work';if(event.sourceType==='approval')return 'Approval';const subject=event.subject||nRouteSubject(event.sourceRoute,event.sourceId);return subject?REC_TYPES[subject.type]?.label||'Workspace':(titles[String(event.sourceRoute).split('?')[0]]||'Workspace');}
function nTime(value){return value?recTimestamp(value):'Not set';}
function nMeta(event){const ids=nScopeIds(event);return ids.length?ids.map(orgName).join(' · '):event.shared?'Bank-wide':'Workspace';}
function nCategoryIcon(type){return icon(N_TYPES[type]?.[1]||'file');}
function nRow(event){
 const receipt=nReceipt(event),status=nStatus(event),unread=!receipt.readAt;
 return `<article class="n-row ${unread?'n-unread':''}" data-notification-id="${e(event.id)}"><label class="n-check"><span class="sr-only">Select ${e(event.title)}</span><input type="checkbox" data-n-select="${e(event.id)}" ${N_SELECTED.has(event.id)?'checked':''}></label><span class="n-kind n-kind-${e(event.type)}">${nCategoryIcon(event.type)}</span><a class="n-row-main" href="#/notification?id=${encodeURIComponent(event.id)}"><span class="n-row-top"><strong>${e(event.title)}</strong><span class="n-row-time">${e(nTime(event.createdAt))}</span></span><span class="n-snippet">${e(event.body)}</span><span class="n-row-meta"><span>${e(N_TYPES[event.type][0])}</span><span>${e(nSourceLabel(event))} · ${e(event.sourceId||'Workspace')}</span>${event.severity==='High'||event.severity==='Critical'?pill(event.severity,'red'):''}${event.actionable?pill(status,nResolved(event)?'green':receipt.acknowledgedAt?'blue':'amber'):''}${receipt.snoozedUntil?pill('Snoozed until '+nTime(receipt.snoozedUntil),'blue'):''}${unread?'<span class="n-unread-dot" aria-label="Unread"></span>':''}</span><span class="n-row-scope">${e(nMeta(event))}</span></a><div class="n-row-actions">${btn(icon(unread?'check':'eye'),'n-toggle-read',`data-id="${e(event.id)}" aria-label="${unread?'Mark read':'Mark unread'}: ${e(event.title)}" title="${unread?'Mark read':'Mark unread'}"`,'icon-only quiet small')}${btn(icon(receipt.archivedAt?'refresh':'download'),'n-archive',`data-id="${e(event.id)}" aria-label="${receipt.archivedAt?'Restore':'Archive'}: ${e(event.title)}" title="${receipt.archivedAt?'Restore to inbox':'Archive'}"`,'icon-only quiet small')}</div></article>`;
}
/* V2.15: show active refinements separately from the selected mailbox. */
function nRefinements(){return [N_FILTER.category&&['Category',N_TYPES[N_FILTER.category]?.[0]||N_FILTER.category],N_FILTER.severity&&['Priority',N_FILTER.severity],N_FILTER.source&&['Source',N_FILTER.source],N_FILTER.search.trim()&&['Search',N_FILTER.search.trim()]].filter(Boolean);}
function nRefinementSummary(){const filters=nRefinements();return filters.length?`<div class="n-active-filters"><span class="n-filter-label">Filtered by</span>${filters.map(([label,value])=>`<span class="n-filter-chip"><span>${e(label)}:</span> ${e(value)}</span>`).join('')}${btn('Clear filters','n-reset-refinements','','quiet small')}</div>`:'';}
function nResultText(rows){return `${rows.length} ${nRefinements().length?'matching ':''}notification${rows.length===1?'':'s'} · ${N_VIEW_LABELS[N_FILTER.view]||'All notifications'}`;}
function nList(rows=nFiltered()){
 if(rows.length)return rows.map(nRow).join('');
 if(nRefinements().length)return empty('No notifications match these filters','Clear the filters to see all notifications in '+(N_VIEW_LABELS[N_FILTER.view]||'this view')+'.',btn('Clear filters','n-reset-refinements','','small'));
 const states={unread:['No unread notifications','Read notifications can still need a decision or follow-up.',btn('View needs action','n-view','data-view="actionable"','small')],actionable:['No outstanding notification actions','Notifications requiring action will appear here. Your work queue may include other assigned work.',link('Open work queue','work','small')],mentions:['No mentions in this view','Source-linked discussions addressed to you will appear here.'],snoozed:['No snoozed notifications','Snoozed notifications return automatically at the time you selected.'],archive:['No archived notifications','Notifications you archive remain available here with their source history.'],inbox:['No notifications in this view','Notifications addressed to you in the selected location will appear here.']};
 const [title,description,action='']=states[N_FILTER.view]||states.inbox;return empty(title,description,action);
}
function nBulkBar(rows=nFiltered()){
 const ids=new Set(rows.map(x=>x.id));for(const id of N_SELECTED)if(!ids.has(id))N_SELECTED.delete(id);
 const selected=N_SELECTED.size,all=ids.size>0&&selected===ids.size,mixed=selected>0&&!all;
 return `<label class="n-select-all"><input type="checkbox" data-n-all data-n-mixed="${mixed}" ${all?'checked':''} ${ids.size?'':'disabled'} aria-checked="${mixed?'mixed':String(all)}" aria-label="Select all ${ids.size} visible notifications"><span>${selected?selected+' selected':rows.length+' notifications'}</span></label><div class="row wrap">${btn('Mark read','n-bulk',`data-operation="read" ${selected?'':'disabled'}`,'quiet small')}${btn(N_FILTER.view==='archive'?'Restore':'Archive','n-bulk',`data-operation="${N_FILTER.view==='archive'?'restore':'archive'}" ${selected?'':'disabled'}`,'quiet small')}${btn('Snooze','n-snooze',selected?'':'disabled','quiet small')}</div>`;
}
function nSyncSelection(){const all=$('[data-n-all]');if(all)all.indeterminate=all.dataset.nMixed==='true';}
function nRefreshList(){
 const rows=nFiltered(),list=$('#n-list');if(list)list.innerHTML=nList(rows);
 const bulk=$('#n-bulk');if(bulk)bulk.innerHTML=nBulkBar(rows);
 const summary=$('#n-refinements');if(summary)summary.innerHTML=nRefinementSummary();
 const status=$('#n-result-status');if(status)status.textContent=nResultText(rows);
 nSyncSelection();nUpdateChrome();
}
function nActorControl(){return `<label class="n-actor">Preview inbox for<select aria-label="Preview inbox for" data-n-actor>${options(REC_PEOPLE.map(x=>[x.id,x.name]),nActor())}</select></label>`;}
function inboxView(){
 inboxInit();const items=nItems(),rows=nFiltered(),active=items.filter(x=>{const r=nReceipt(x);return !r.archivedAt&&!r.snoozedUntil;}),unread=active.filter(x=>!nReceipt(x).readAt).length,needs=active.filter(x=>x.actionable&&!nResolved(x)).length,urgent=active.filter(x=>!nResolved(x)&&['High','Critical'].includes(x.severity)).length;
 const views=[['inbox','All notifications',active.length],['unread','Unread',unread],['actionable','Needs action',needs],['mentions','Mentions',active.filter(x=>x.type==='mention').length],['snoozed','Snoozed',items.filter(x=>nReceipt(x).snoozedUntil&&!nReceipt(x).archivedAt).length],['archive','Archived',items.filter(x=>nReceipt(x).archivedAt).length]];
 return header('Inbox','Mentions, deadlines and exceptions — with the source and the next step in one place.',link(icon('settings')+' Preferences','notification-preferences')+btn(icon('plus')+' Mention a colleague','n-new-mention','','primary'),'Your workspace')+`<section class="n-summary"><div><span class="n-summary-dot"></span><strong>${urgent} priority ${urgent===1?'item':'items'}</strong><span>${needs} need action · ${unread} unread</span></div>${nActorControl()}</section><div class="n-layout"><aside class="n-rail" aria-label="Inbox views"><div class="n-rail-section">${views.map(([key,label,count])=>`<button type="button" data-action="n-view" data-view="${key}" class="n-view ${N_FILTER.view===key?'active':''}" ${N_FILTER.view===key?'aria-current="page"':''}><span>${label}</span><b>${count}</b></button>`).join('')}</div><div class="n-rail-section"><h2>Categories</h2>${[['','All categories'],...Object.entries(N_TYPES).map(([k,v])=>[k,v[0]])].map(([key,label])=>`<button type="button" data-action="n-category" data-category="${key}" class="n-category ${N_FILTER.category===key?'active':''}" aria-pressed="${N_FILTER.category===key}">${key?nCategoryIcon(key):icon('layers')}<span>${label}</span></button>`).join('')}</div><p class="n-rail-note">Inbox actions are personal. Acknowledging or archiving a notification does not close its source issue.</p></aside><section class="panel n-main"><div class="n-filterbar"><label class="n-search">${icon('search')}<span class="sr-only">Search notifications</span><input type="search" data-n-search aria-describedby="n-result-status" value="${e(N_FILTER.search)}" placeholder="Search mentions, sources or people"></label><select data-n-severity aria-label="Notification priority">${options([['','Any priority'],['Critical','Critical'],['High','High'],['Normal','Normal'],['Low','Low']],N_FILTER.severity)}</select><select data-n-source aria-label="Notification source">${options([['','Any source'],...[...new Set(items.map(nSourceLabel))].sort().map(x=>[x,x])],N_FILTER.source)}</select></div><div id="n-refinements">${nRefinementSummary()}</div><div id="n-bulk" class="n-bulk">${nBulkBar(rows)}</div><p id="n-result-status" class="n-result-status" role="status" aria-live="polite" aria-atomic="true">${e(nResultText(rows))}</p><div id="n-list" class="n-list" aria-label="Notifications">${nList(rows)}</div></section></div><p class="n-simulation">Preview workspace · notifications and discussion replies stay in this browser. Source monitoring uses ${date(S.asOf)}; snoozes use the current clock.</p>`;
}
function nEvent(id){const event=S.notificationsV28.events.find(x=>x.id===id);requireValue(event&&nCanSee(event),'This notification is unavailable for the selected recipient or scope.');return event;}
function nThread(event){return (event.thread||[]).map(msg=>`<article class="n-message"><div class="n-avatar">${e(nPerson(msg.actor).split(/\s+/).map(n=>n[0]).slice(0,2).join(''))}</div><div><header><strong>${e(nPerson(msg.actor))}</strong><time>${e(nTime(msg.createdAt))}</time></header><p>${e(msg.text)}</p></div></article>`).join('');}
function nDetailView(){
 inboxInit();const event=S.notificationsV28.events.find(x=>x.id===current.params.get('id'));
 if(!event||!nCanSee(event)){current.blocked=true;return header('Notification unavailable','Choose the correct recipient and organizational scope, or return to your inbox.',link('Back to inbox','inbox','primary'));}
 const receipt=nReceipt(event);if(!receipt.readAt){receipt.readAt=nNow();save();}
 const source=nRecordFor(event),sourceRoute=nSourceRoute(event),status=nStatus(event),canThread=event.type==='mention',root=event.threadRootId?S.notificationsV28.events.find(x=>x.id===event.threadRootId)||event:event;
 return header(e(event.title),`${e(N_TYPES[event.type][0])} · ${e(nPerson(event.actor))} · ${e(nTime(event.createdAt))}`,link(icon('back')+' Inbox','inbox')+btn(receipt.archivedAt?'Restore to inbox':'Archive','n-archive',`data-id="${e(event.id)}"`),'Notification')+`<div class="n-detail-grid"><div class="stack"><section class="panel n-detail"><div class="row wrap">${pill(event.severity,event.severity==='High'||event.severity==='Critical'?'red':'blue')}${pill(status,nResolved(event)?'green':receipt.acknowledgedAt?'blue':'amber')}${nMandatory(event)?pill('Mandatory in-app','') :''}</div><p class="n-detail-body">${e(event.body)}</p><div class="n-next"><span class="eyebrow">Next step</span><h2>${nResolved(event)?'The source no longer needs this action':event.type==='mention'?'Reply with the decision or information needed':event.type==='escalation'?'Acknowledge ownership and review the recovery plan':event.type==='document'?'Refresh the evidence and submit it for review':event.type==='quality'?'Inspect and correct the source defect':event.type==='approval'?'Review the exact source and record your decision':'Review outstanding work with the accountable owner'}</h2><div class="actions wrap">${event.sourceType==='task'&&source?btn('Review action '+icon('arrow'),'n-open-task',`data-id="${e(event.id)}"`,'primary'):sourceRoute?link('Open '+e(nSourceLabel(event).toLowerCase())+' '+icon('arrow'),sourceRoute,'primary'):pill('Source unavailable','amber')}${event.actionable&&!receipt.acknowledgedAt&&!nResolved(event)?btn('Acknowledge','n-ack',`data-id="${e(event.id)}"`):''}${!nResolved(event)&&event.type!=='mention'?btn('Create follow-up','n-follow-up',`data-id="${e(event.id)}"`):''}</div>${receipt.acknowledgedAt?`<p class="field-help">Acknowledged by ${e(nPerson(nActor()))} · ${e(nTime(receipt.acknowledgedAt))}. The source record remains authoritative.</p>`:''}</div></section>${canThread?panel('Conversation','A local preview of a source-linked discussion.',`<div class="n-thread">${nThread(root)}</div>${root.threadResolvedAt?`<div class="n-thread-resolved">${icon('check')} Discussion resolved by ${e(nPerson(root.threadResolvedBy))} · ${e(nTime(root.threadResolvedAt))}</div>`:`<form class="n-reply" data-n-form="reply" data-id="${e(event.id)}">${area('reply','Your reply','','required minlength="3" maxlength="2000" placeholder="State the decision, evidence needed or next step…"')}${select('mention','Mention a colleague (optional)',[['','No additional mention'],...REC_PEOPLE.filter(p=>p.id!==nActor()).map(p=>[p.id,p.name])])}<div class="row wrap"><button type="submit" class="btn primary">Add reply</button>${btn('Resolve discussion','n-resolve-thread',`data-id="${e(event.id)}"`)}<span class="field-help">Saved locally; no message is sent.</span></div><div class="form-error" role="alert"></div></form>`}`):panel('Activity','Your notification history.',`<ul class="timeline"><li><strong>Notification created</strong><small>${e(nTime(event.createdAt))} · ${e(nPerson(event.actor))}</small></li><li><strong>Read by ${e(nPerson(nActor()))}</strong><small>${e(nTime(receipt.readAt))}</small></li>${receipt.acknowledgedAt?`<li><strong>Acknowledged</strong><small>${e(nTime(receipt.acknowledgedAt))}</small></li>`:''}${receipt.snoozedUntil?`<li><strong>Snoozed</strong><small>Returns to inbox ${e(nTime(receipt.snoozedUntil))}</small></li>`:''}${event.followUpId?`<li><strong>Follow-up created</strong><small>${e(event.followUpId)} · ${link('Open work queue','work','quiet small')}</small></li>`:''}</ul>`)}${event.type==='escalation'?notice('Acknowledgement records that you have seen this escalation. Timers and stop conditions are configured separately in '+link('Escalation matrix','escalation-matrix','quiet small')+'.','mini'):''}</div><aside class="stack"><section class="panel pad"><h2>Source context</h2><dl class="n-context"><div><dt>Source</dt><dd>${e(nSourceLabel(event))} · ${e(event.sourceId||'Workspace')}</dd></div><div><dt>Location</dt><dd>${e(nMeta(event))}</dd></div><div><dt>Accountable owner</dt><dd>${e(source?.owner||(source?.ownerId?nPerson(source.ownerId):'Not assigned'))}</dd></div><div><dt>Due / expiry</dt><dd>${event.due?date(nDay(event.due)):'No date attached'}</dd></div><div><dt>Recipient</dt><dd>${e(nPerson(nActor()))}</dd></div></dl>${sourceRoute?link('View source record',sourceRoute,'wide'):'<p class="field-help">The original record is no longer available. This notification retains its historical context.</p>'}</section><section class="panel pad"><h2>Manage notification</h2><div class="stack mt16">${btn('Mark unread','n-mark-unread',`data-id="${e(event.id)}"`,'wide')}${btn(receipt.snoozedUntil?'Change snooze':'Snooze','n-snooze',`data-id="${e(event.id)}"`,'wide')}${receipt.snoozedUntil?btn('Return to inbox now','n-unsnooze',`data-id="${e(event.id)}"`,'wide'):''}${link('Notification preferences','notification-preferences','wide')}</div><p class="field-help">Read, archive and snooze settings belong only to this recipient.</p></section></aside></div>`;
}
function nPreferencesView(){
 inboxInit();const prefs=nPrefs();
 return header('Notification preferences','Choose how future updates enter your inbox. Critical, overdue, escalation and approval items remain visible.',link(icon('back')+' Inbox','inbox'),'Personal settings')+`<form data-n-form="preferences" class="n-preferences"><section class="panel"><div class="panel-head"><div><h2>In-app notifications</h2><p>Preferences for ${e(nPerson(nActor()))}. Existing notification history is retained.</p></div></div><div class="n-pref-rows">${Object.entries(N_TYPES).map(([key,v])=>`<label class="n-pref-row"><span class="n-kind n-kind-${key}">${nCategoryIcon(key)}</span><span><strong>${e(v[0])}</strong><small>${{mention:'Direct mentions and discussion replies',program:'Program changes, submissions and respondent updates',quality:'Failed quality checks and missing source information',document:'Expiring or expired evidence and documents',deadline:'Upcoming deadlines; overdue items always remain enabled',approval:'Assigned approval and independent-review requests',escalation:'Policy-driven exceptions requiring your attention'}[key]}</small></span><input type="checkbox" name="inApp-${key}" ${prefs.inApp[key]||['approval','escalation'].includes(key)?'checked':''} ${['approval','escalation'].includes(key)?'disabled':''} aria-label="${e(v[0])} in-app notifications">${['approval','escalation'].includes(key)?'<b class="n-required">Required</b>':''}</label>`).join('')}</div></section><section class="panel pad"><h2>Delivery settings</h2><p class="small muted">Configuration preview only. Email delivery and background jobs are not connected.</p><div class="field-grid">${select('email','Email delivery',['Off','Daily digest','Immediate'],prefs.email)}${input('digestAt','Daily digest time · WAT',prefs.digestAt,'time','required')}${input('quietStart','Quiet hours start · WAT',prefs.quietStart,'time','required')}${input('quietEnd','Quiet hours end · WAT',prefs.quietEnd,'time','required')}</div><label class="checkbox-row mt16"><input type="checkbox" name="weekdaysOnly" ${prefs.weekdaysOnly?'checked':''}><span>Send non-urgent email digests on weekdays only</span></label><p class="field-help">Quiet hours defer optional immediate email to the next allowed time. Mandatory in-app alerts remain immediate. Daily digest preferences do not acknowledge or resolve an item.</p></section><div class="n-pref-footer"><div class="form-error" role="alert"></div><button type="submit" class="btn primary">Save preferences</button></div></form>`;
}
function nSnoozeDialog(ids){
 const actor=nActor(),events=ids.map(nEvent);requireValue(events.length,'Select at least one notification.');
 openDialog('Snooze '+events.length+' '+(events.length===1?'notification':'notifications'),'Snoozing changes only your inbox. Source deadlines and escalation clocks continue.',select('duration','Return to inbox',[['hour','In one hour'],['tomorrow','Tomorrow at 09:00 WAT'],['week','In seven days at 09:00 WAT']],'tomorrow'),'Snooze',form=>{requireValue(nActor()===actor,'The acting recipient changed. Reopen snooze for the intended inbox.');const duration=values(form).duration,now=new Date();let until;if(duration==='hour')until=new Date(now.getTime()+36e5);else{const watDay=new Date(now.getTime()+36e5).toISOString().slice(0,10),target=addDays(watDay,duration==='week'?7:1);until=new Date(target+'T08:00:00Z');}for(const event of events){requireValue(nCanSee(event),'A notification is outside the current scope.');const receipt=nReceipt(event);receipt.snoozedUntil=until.toISOString();receipt.archivedAt=null;}N_SELECTED.clear();save();go('inbox');toast('Snoozed until '+nTime(until.toISOString())+'.');});
}
function nFollowUp(event){
 requireValue(!nResolved(event),'The source is already resolved.');const actor=nActor();
 if(event.followUpId&&S.tasks.some(x=>x.id===event.followUpId)){go('work');toast('The existing follow-up is in Work & Approvals.');return;}
 const source=nRecordFor(event),scopes=nScopeIds(event);
 openDialog('Create a follow-up','Link an accountable action to the original notification and source.',`<div class="stack">${input('title','Action title',event.title,'text','required maxlength="180"')}${select('owner','Owner',REC_PEOPLE.map(p=>[p.id,p.name]),source?.ownerId||nActor())}${input('due','Due date',addDays(S.asOf,7),'date',`required min="${S.asOf}"`)}${select('scope','Accountable location',scopes.length?scopes.map(id=>[id,orgName(id)]):ORG.map(n=>[n.id,orgName(n.id)]),scopes[0]||selectedScope(),'required')}${area('outcome','Required outcome','','required minlength="10" maxlength="1500" placeholder="Describe the evidence or outcome needed for completion"')}</div>`,'Create follow-up',form=>{requireValue(nActor()===actor,'The acting user changed. Reopen the follow-up.');requireValue(nCanSee(event),'The notification is outside the current scope.');const v=values(form);requireValue(v.title.trim()&&v.outcome.trim().length>=10,'Enter a title and a clear required outcome.');requireValue(recUser(v.owner),'Choose a known owner.');validDate(v.due,'Due date',false);requireValue(v.due>=S.asOf,'The due date must be on or after the workspace snapshot.');requireValue(orgNode(v.scope)&&(!scopes.length||scopes.includes(v.scope)),'Choose a location belonging to this source.');const task={id:uid('ACT'),title:v.title.trim(),kind:'Remediation',owner:nPerson(v.owner),ownerId:v.owner,due:v.due,status:'Open',route:nSourceRoute(event)||'work',ref:event.sourceType==='task'?source.ref:event.sourceId||event.id,orgScopeId:v.scope,outcome:v.outcome.trim(),notificationId:event.id};S.tasks.unshift(task);event.followUpId=task.id;audit('Created notification follow-up: '+task.title,task.ref,nPerson(actor));nPut({id:'NTF-ASSIGNED-'+task.id,type:'program',title:'You have a new follow-up: '+task.title,body:task.outcome,sourceType:'task',sourceId:task.id,sourceRoute:'work',recipientIds:[v.owner],orgScopeId:v.scope,actor,due:task.due,actionable:true,severity:event.severity});save();render();toast('Follow-up created for '+nPerson(v.owner)+'.');});
}
function nMentionDialog(){
 const actor=nActor(),sources=['program','risk','evidence','kriReturn'].flatMap(type=>REC_TYPES[type].collection().filter(r=>recInScope(type,r)).map(r=>({type,id:r.id,label:REC_TYPES[type].label+' · '+recName(type,r),route:recUrl(type,r.id)})));
 requireValue(sources.length,'There are no source records in this scope.');
 const context=typeof recContext==='function'?recContext():null,defaultKey=context?sources.find(x=>x.type===context.type&&x.id===context.id):sources[0];
 openDialog('Mention a colleague','Create a source-linked discussion in the local preview inbox.',`<div class="stack">${select('source','Source record',sources.map(x=>[x.type+'::'+x.id,x.label]),defaultKey?defaultKey.type+'::'+defaultKey.id:sources[0].type+'::'+sources[0].id)}${select('recipient','Mention',REC_PEOPLE.filter(p=>p.id!==actor).map(p=>[p.id,p.name]))}${area('message','Message','','required minlength="5" maxlength="2000" placeholder="What decision, clarification or action do you need?"')}<p class="field-help">This preview saves the discussion locally. No message leaves the browser.</p></div>`,'Add mention',form=>{requireValue(nActor()===actor,'The acting user changed. Reopen the mention.');const v=values(form),source=sources.find(x=>x.type+'::'+x.id===v.source);requireValue(source&&recInScope(source.type,recRead(source.type,source.id)),'The source is no longer available in this scope.');requireValue(recUser(v.recipient)&&v.recipient!==actor,'Choose another known colleague.');requireValue(v.message.trim().length>=5,'Enter a clear message.');const event=nPut({id:uid('NTF'),type:'mention',title:nPerson(actor)+' mentioned you in '+recName(source.type,recRead(source.type,source.id)),body:'@'+nPerson(v.recipient)+' '+v.message.trim(),sourceId:source.id,subject:{type:source.type,id:source.id},sourceRoute:source.route,recipientIds:[v.recipient,actor],actor,createdAt:nNow(),thread:[{id:uid('MSG'),actor,text:'@'+nPerson(v.recipient)+' '+v.message.trim(),createdAt:nNow()}]});nReceipt(event,actor).readAt=nNow();audit('Added a local discussion mention',source.id,nPerson(actor));go('notification?id='+event.id);toast('Mention added to '+nPerson(v.recipient)+'’s preview inbox.');});
}
Object.assign(actions,{
 'n-view':el=>{N_FILTER.view=el.dataset.view;N_SELECTED.clear();render();},
 'n-category':el=>{N_FILTER.category=el.dataset.category;N_SELECTED.clear();render();},
 'n-clear-filters':()=>{N_FILTER={view:'inbox',category:'',search:'',severity:'',source:''};N_SELECTED.clear();render();},
 'n-reset-refinements':()=>{Object.assign(N_FILTER,{category:'',search:'',severity:'',source:''});N_SELECTED.clear();render();$('[data-n-search]')?.focus();},
 'n-toggle-read':el=>{const event=nEvent(el.dataset.id),r=nReceipt(event);r.readAt=r.readAt?null:nNow();save();render();},
 'n-mark-unread':el=>{nReceipt(nEvent(el.dataset.id)).readAt=null;save();go('inbox');},
 'n-archive':el=>{const r=nReceipt(nEvent(el.dataset.id));r.archivedAt=r.archivedAt?null:nNow();r.snoozedUntil=null;save();if(current.name==='notification')go('inbox');else render();},
 'n-ack':el=>{const event=nEvent(el.dataset.id);nReceipt(event).acknowledgedAt=nNow();audit('Acknowledged notification: '+event.title,event.sourceId,nPerson(nActor()));render();},
 'n-snooze':el=>nSnoozeDialog(el.dataset.id?[el.dataset.id]:[...N_SELECTED].filter(id=>nFiltered().some(x=>x.id===id))),
 'n-unsnooze':el=>{nReceipt(nEvent(el.dataset.id)).snoozedUntil=null;save();render();},
 'n-bulk':el=>{const ids=[...N_SELECTED].filter(id=>nFiltered().some(x=>x.id===id));requireValue(ids.length,'Select notifications in this view.');for(const id of ids){const receipt=nReceipt(nEvent(id));if(el.dataset.operation==='read')receipt.readAt=nNow();else if(el.dataset.operation==='archive'){receipt.archivedAt=nNow();receipt.snoozedUntil=null;}else if(el.dataset.operation==='restore')receipt.archivedAt=null;}N_SELECTED.clear();save();render();},
 'n-open-task':el=>{const event=nEvent(el.dataset.id),task=nRecordFor(event);requireValue(event.sourceType==='task'&&task,'The action is unavailable.');actions['task-detail']({dataset:{id:task.id}});},
 'n-follow-up':el=>nFollowUp(nEvent(el.dataset.id)),
 'n-new-mention':nMentionDialog,
 'n-resolve-thread':el=>{const event=nEvent(el.dataset.id),root=event.threadRootId?S.notificationsV28.events.find(x=>x.id===event.threadRootId)||event:event;requireValue(root.type==='mention','Only discussions can be resolved here.');const actor=nActor();openDialog('Resolve this discussion?','The source record and its approval state are unchanged.',area('outcome','Resolution summary','','required minlength="10" maxlength="1000"'),'Resolve discussion',form=>{requireValue(nActor()===actor&&nCanSee(event),'The acting user or scope changed.');const message=values(form).outcome.trim();requireValue(message.length>=10,'Record the discussion outcome.');root.thread.push({id:uid('MSG'),actor,text:'Discussion outcome: '+message,createdAt:nNow()});for(const x of S.notificationsV28.events.filter(x=>x.id===root.id||x.threadRootId===root.id)){x.threadResolvedAt=nNow();x.threadResolvedBy=actor;}audit('Resolved local discussion',root.sourceId,nPerson(actor));render();});}
});
document.addEventListener('change',ev=>{
 const el=ev.target;
 if(el.matches('[data-n-actor]')){requireValue(recUser(el.value),'Choose a known recipient.');S.governance.actorId=el.value;N_SELECTED.clear();save();render();}
 if(el.matches('[data-n-select]')){el.checked?N_SELECTED.add(el.dataset.nSelect):N_SELECTED.delete(el.dataset.nSelect);const bulk=$('#n-bulk');if(bulk)bulk.innerHTML=nBulkBar();nSyncSelection();}
 if(el.matches('[data-n-all]')){for(const event of nFiltered())el.checked?N_SELECTED.add(event.id):N_SELECTED.delete(event.id);nRefreshList();}
 if(el.matches('[data-n-severity]')){N_FILTER.severity=el.value;N_SELECTED.clear();nRefreshList();}
 if(el.matches('[data-n-source]')){N_FILTER.source=el.value;N_SELECTED.clear();nRefreshList();}
});
document.addEventListener('input',ev=>{if(ev.target.matches('[data-n-search]')){N_FILTER.search=ev.target.value;N_SELECTED.clear();nRefreshList();}});
document.addEventListener('submit',ev=>{
 const form=ev.target;if(!form.dataset.nForm)return;ev.preventDefault();
 try{
  const v=values(form);
  if(form.dataset.nForm==='preferences'){
   const prefs=nPrefs();
   requireValue(['Off','Daily digest','Immediate'].includes(v.email),'Choose an email delivery option.');
   for(const key of ['digestAt','quietStart','quietEnd'])requireValue(/^([01]\d|2[0-3]):[0-5]\d$/.test(v[key]),'Enter valid delivery times.');
   for(const key of Object.keys(N_TYPES))prefs.inApp[key]=['approval','escalation'].includes(key)||!!v['inApp-'+key];
   Object.assign(prefs,{email:v.email,digestAt:v.digestAt,quietStart:v.quietStart,quietEnd:v.quietEnd,weekdaysOnly:!!v.weekdaysOnly});audit('Updated personal notification preferences','notifications',nPerson(nActor()));toast('Preferences saved for future notifications. No email service is connected.');render();
  }else if(form.dataset.nForm==='reply'){
   const event=nEvent(form.dataset.id),root=event.threadRootId?S.notificationsV28.events.find(x=>x.id===event.threadRootId)||event:event,actor=nActor();requireValue(!root.threadResolvedAt,'This discussion has been resolved.');requireValue(v.reply.trim().length>=3,'Enter a reply.');requireValue(!v.mention||recUser(v.mention),'Choose a known colleague.');
   const text=(v.mention?'@'+nPerson(v.mention)+' ':'')+v.reply.trim(),createdAt=nNow();root.thread.push({id:uid('MSG'),actor,text,createdAt});
   const recipients=[...new Set([...(root.recipientIds||[]),nUserId(root.actor),v.mention].filter(id=>id&&id!==actor))];
   if(recipients.length)nPut({id:uid('NTF'),type:'mention',title:nPerson(actor)+' replied in '+root.title,body:text,sourceId:root.sourceId,subject:root.subject,sourceRoute:root.sourceRoute,orgScopeIds:nScopeIds(root),recipientIds:recipients,actor,threadRootId:root.id,createdAt});
   if(v.mention&&!root.recipientIds.includes(v.mention))root.recipientIds.push(v.mention);audit('Added a local discussion reply',root.sourceId,nPerson(actor));render();toast('Reply saved to the local discussion.');
  }
 }catch(err){const box=$('.form-error',form);if(box)box.textContent=err.message;else toast(err.message);}
});
function nUpdateChrome(){
 if(!S.notificationsV28||!S.governance)return;const count=inboxUnreadCount();
 document.querySelectorAll('[data-n-unread-count]').forEach(el=>{el.textContent=count;el.hidden=!count;});
 const bell=document.querySelector('[data-n-bell]');if(bell)bell.setAttribute('aria-label','Inbox, '+count+' unread notifications');
}
paths.bell='M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4';
Object.assign(titles,{inbox:'Inbox',notification:'Notification','notification-preferences':'Notification preferences'});
Object.assign(window.CS_JOURNEYS,{inbox:inboxView,notification:nDetailView,'notification-preferences':nPreferencesView});
Object.assign(window.CS_JOURNEY_GROUPS,{inbox:'inbox',notification:'inbox','notification-preferences':'inbox'});
const nPriorRender=render;
render=function(){
 inboxInit();const context=nActor()+'|'+selectedScope();if(context!==N_CONTEXT){N_SELECTED.clear();N_CONTEXT=context;}
 nPriorRender();
 const nav=$('.sidebar>nav');if(nav&&!nav.querySelector('[href="#/inbox"]'))nav.insertAdjacentHTML('afterbegin',`<a class="nav-link ${['inbox','notification','notification-preferences'].includes(current.name)?'active':''}" href="#/inbox" ${['inbox','notification','notification-preferences'].includes(current.name)?'aria-current="page"':''}>${icon('bell')}<span>Inbox</span><b class="count" data-n-unread-count></b></a>`);
 const profile=$('.topbar .profile');if(profile&&!$('[data-n-bell]'))profile.insertAdjacentHTML('beforebegin',`<a href="#/inbox" class="btn icon-only quiet n-bell" data-n-bell aria-label="Inbox">${icon('bell')}<span data-n-unread-count></span></a>`);
 if(['program','risk','evidence-review','kri-return'].includes(current.name)&&!current.blocked)$('.page-head .actions,.j-intro .actions')?.insertAdjacentHTML('beforeend',btn(icon('users')+' Mention','n-new-mention','','quiet'));
 nSyncSelection();nUpdateChrome();
};
window.CS_V28_EVENTS=Object.freeze({emit:inboxEmit,events:()=>clone(nItems()),unread:inboxUnreadCount});
window.CLEAR_SIGHT_INBOX=Object.freeze({version:'2.8-design-preview',getState:()=>clone(inboxInit()),getVisible:()=>clone(nItems()),getFiltered:()=>clone(nFiltered()),unread:inboxUnreadCount,isResolved:id=>{const event=S.notificationsV28.events.find(x=>x.id===id);return event?nResolved(event):null;}});
