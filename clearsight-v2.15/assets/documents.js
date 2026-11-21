/* V2.6 document workspaces, uploads, version-pinned links and internal sign-off.
 * No external services. All permissions and identities here are demonstration UI.
 */
'use strict';
let DOC_FILTERS={search:'',folder:'',type:'',status:'',sort:'updated',archived:false};
let DOC_SELECTED=new Set(),DOC_UPLOAD=null,DOC_APPROVAL_DRAFT=null,DOC_PREVIEW_URLS=[];
const DOC_ACCEPT=Object.keys(DOC_MIME).map(x=>'.'+x).join(',');
const docUserOptions=()=>REC_PEOPLE.map(p=>[p.id,p.name]);
const docFolderName=id=>S.governance.folders.find(f=>f.id===id)?.name||'Folder unavailable';
const docVersionLabel=v=>v?'v'+v.number:'Version unavailable';
function docVersionStatus(v){
  if(!v)return 'Unavailable';
  if(v.storage==='reference')return 'Reference only';
  const r=S.governance.requests.filter(r=>r.subject.type==='document'&&r.subject.id===v.documentId&&r.versionId===v.id).at(-1);
  if(!r)return 'Draft';
  return r.status==='Completed'?(r.steps.some(s=>s.kind==='signoff')?'Signed off':'Approved'):r.status==='Pending'?'In review':['Cancelled','Superseded'].includes(r.status)?'Draft':r.status;
}
function docStatus(d){return d.archived?'Archived':docVersionStatus(docHead(d));}

function docStatePill(status){return pill(status,['Approved','Signed off','Completed'].includes(status)?'green':['In review','Pending','Draft'].includes(status)?'blue':['Changes requested','Reference only','Superseded'].includes(status)?'amber':status==='Rejected'?'red':'');}
function docScopeLabel(d){return d.scopeMode==='shared'?'Bank-wide':orgName(d.orgScopeId)||'Unassigned';}
function docVisible(d,scope=selectedScope()){return recInScope('document',d,scope);}
function docRows(filter=DOC_FILTERS){
  let rows=S.governance.documents.filter(d=>docVisible(d)&&!!d.archived===!!filter.archived);
  if(filter.folder)rows=rows.filter(d=>d.folderId===filter.folder);
  if(filter.type)rows=rows.filter(d=>docHead(d)?.fileName?.split('.').at(-1).toLowerCase()===filter.type);
  if(filter.status)rows=rows.filter(d=>docStatus(d)===filter.status);
  if(filter.search){const q=filter.search.toLocaleLowerCase();rows=rows.filter(d=>[d.title,d.id,docHead(d)?.fileName,recUser(d.ownerId)?.name,docFolderName(d.folderId)].join(' ').toLocaleLowerCase().includes(q));}
  return [...rows].sort((a,b)=>filter.sort==='name'?a.title.localeCompare(b.title):String(docHead(b)?.createdAt||b.createdAt).localeCompare(String(docHead(a)?.createdAt||a.createdAt)));
}
function docLinks(type,id){return S.governance.links.filter(l=>l.subject.type===type&&l.subject.id===id);}
function docCanLink(d,subject){
  if(d.archived||!recRead(subject.type,subject.id))return false;
  if(d.scopeMode==='shared')return true;
  const scopes=recScope(subject.type,recRead(subject.type,subject.id));
  return scopes.some(id=>orgWithin(id,d.orgScopeId)||orgWithin(d.orgScopeId,id));
}
function docLinkVersion(versionId,subject,anchor='',allowLocked=false){
  const v=docVersion(versionId),d=v&&recRead('document',v.documentId);
  requireValue(v&&d,'Select an existing document version.');
  requireValue(allowLocked||!docLocked(subject.type,recRead(subject.type,subject.id)||{}),'This record is locked. Create a revision before changing its attachments.');
  requireValue(docCanLink(d,subject),'This document is not available to the target scope. Use a bank-wide document or a matching location.');
  const old=S.governance.links.find(l=>l.documentVersionId===versionId&&l.subject.type===subject.type&&l.subject.id===subject.id&&(l.anchor||'')===anchor);
  if(old)return old;
  const l={id:uid('LINK'),documentVersionId:versionId,subject:{type:subject.type,id:subject.id},anchor,createdBy:recActor().id,createdAt:new Date().toISOString()};
  S.governance.links.push(l);recEvent('link',subject,'Linked '+v.fileName+' · v'+v.number,recActor().id,{documentVersionId:versionId});return l;
}
function docEvidenceContext(config,d){
  const {programId,target,question,observed}=config,p=recRead('program',programId),q=findQ(question);
  requireValue(p&&p.targets.includes(target),'Select a valid program target.');
  requireValue(q&&uniqueQ(activeReqs(p)).some(x=>x.id===question),'The question is not applicable to this program.');
  validDate(observed,'Observation date');
  requireValue(d&&docCanLink(d,{type:'program',id:p.id}),'The document does not match this program scope.');
  if(target.startsWith('ENG'))requireValue(docCanLink(d,{type:'engagement',id:target}),'The document does not match this vendor engagement.');
  return {p,q,target,observed};
}
function docEvidenceAdd(versionId,config){
  const v=docVersion(versionId),d=v&&recRead('document',v.documentId);
  requireValue(v,'Select a document version.');
  const {p,q,target,observed}=docEvidenceContext(config,d);
  const existing=S.evidence.find(x=>x.documentVersionId===versionId&&x.program===p.id&&x.target===target&&x.question===q.id);
  if(existing)return existing;
  const ev={id:uid('EV'),documentVersionId:versionId,program:p.id,target,question:q.id,observed,expires:addDays(observed,policy(p,q).freshDays),owner:recUser(v.uploadedBy)?.name||'Unknown contributor',status:'Submitted',orgScopeId:d.orgScopeId||undefined};
  S.evidence.unshift(ev);
  // Evidence is linked to its program once; acceptance remains a distinct decision.
  docLinkVersion(versionId,{type:'evidence',id:ev.id},q.id);docLinkVersion(versionId,{type:'program',id:p.id},q.id);
  if(target.startsWith('ENG'))docLinkVersion(versionId,{type:'engagement',id:target},q.id);
  recordsInit();audit('Document submitted: '+v.fileName,ev.id,recActor().name);return ev;
}
function docTable(rows,{selection=true}={}){
  if(!rows.length)return empty('No documents match','Change the filters or add a document.',btn(icon('plus')+' Upload files','doc-upload','','primary'));
  return table([...(selection?['<input type="checkbox" data-doc-select-all aria-label="Select all displayed documents">']:[]),'Document','Location / owner','Version','State',''],rows.map(d=>{
    const v=docHead(d);return {attrs:`data-doc-row="${d.id}"`,cells:[...(selection?[`<input type="checkbox" data-doc-select="${d.id}" ${DOC_SELECTED.has(d.id)?'checked':''} aria-label="Select ${e(d.title)}">`]:[]),`<a class="doc-name" href="#/document?id=${d.id}"><span class="doc-filetype">${e((v?.fileName.split('.').at(-1)||'file').toUpperCase())}</span><span><strong>${e(d.title)}</strong><small>${e(d.id)} · ${recBytes(v?.size)} · ${e(docFolderName(d.folderId))}</small></span></a>`,`<strong>${e(docScopeLabel(d))}</strong><small>${e(recUser(d.ownerId)?.name||'Unassigned')}</small>`,`<strong>${docVersionLabel(v)}</strong><small>${v?.storage==='reference'?'No file supplied':v?.storage==='session'?'Session file':v?.storage==='sample'?'Sample file':'Stored in browser'}</small>`,docStatePill(docStatus(d)),link('Open','document?id='+d.id,'quiet small')]};
  }));
}
function documentsView(){
  const rows=docRows(),visible=S.governance.documents.filter(d=>docVisible(d)&&!d.archived),selected=[...DOC_SELECTED].filter(id=>rows.some(d=>d.id===id));
  return header('Documents','Files, versions and their linked records.',btn(icon('plus')+' Upload files','doc-upload','','primary'),'Documents')+
    (typeof d210Navigation==='function'?d210Navigation('library'):'')+
    `<div class="doc-summary-strip"><span><strong>${visible.length}</strong> documents</span><span><strong>${visible.filter(d=>docStatus(d)==='In review').length}</strong> in review</span><span><strong>${visible.filter(d=>['Approved','Signed off'].includes(docStatus(d))).length}</strong> approved</span>${link('Approval inbox','approvals','quiet small')}</div>`+
    `<div class="doc-explorer"><nav class="doc-folders" aria-label="Document folders"><h2>Folders</h2>${btn('All documents','doc-folder','data-id=""',!DOC_FILTERS.folder&&!DOC_FILTERS.archived?'active':'')}${S.governance.folders.map(f=>btn(icon('program')+e(f.name),'doc-folder',`data-id="${f.id}"`,DOC_FILTERS.folder===f.id?'active':'')).join('')}${btn(icon('plus')+' New folder','doc-new-folder','','quiet small')}<hr class="divider">${btn('Archived','doc-archived','',DOC_FILTERS.archived?'active':'')}${link('Data checks','data-health','quiet small')}</nav><section class="panel doc-results"><div class="doc-toolbar"><label class="search">${icon('search')}<input aria-label="Search documents" data-doc-filter="search" value="${e(DOC_FILTERS.search)}" placeholder="Search name, ID or owner"></label><select data-doc-filter="type" aria-label="File format">${options([['','All formats'],...Object.keys(DOC_MIME).map(x=>[x,x.toUpperCase()])],DOC_FILTERS.type)}</select><select data-doc-filter="status" aria-label="Document state">${options([['','All states'],'Draft','In review','Approved','Signed off','Changes requested','Rejected','Reference only'],DOC_FILTERS.status)}</select><select data-doc-filter="sort" aria-label="Sort documents">${options([['updated','Recently updated'],['name','Name A–Z']],DOC_FILTERS.sort)}</select>${btn(icon('download'),'doc-export','aria-label="Export displayed document register"','icon-only quiet')}</div><div class="doc-list-meta"><span>${rows.length} ${rows.length===1?'document':'documents'} · ${e(DOC_FILTERS.archived?'Archive':docFolderName(DOC_FILTERS.folder)==='Folder unavailable'?'All folders':docFolderName(DOC_FILTERS.folder))}</span><span>Scope: ${e(selectedScope()?orgName(selectedScope()):'All locations')}</span></div><div id="doc-bulk" ${selected.length?'':'hidden'} class="doc-bulk"><strong>${selected.length} selected</strong>${btn('Move','doc-bulk-move','','small')}${btn(DOC_FILTERS.archived?'Restore':'Archive','doc-bulk-archive','','small')}${btn('Clear','doc-clear-selected','','quiet small')}</div><div id="doc-table">${docTable(rows)}</div></section></div>`;
}
function docStartUpload({subject=null,versionFor=null,evidence=null,field=null}={}){
  if(DOC_UPLOAD?.rows.length){go('document-upload');toast('Finish or discard the existing upload before starting another.');return;}
  const d=versionFor&&recRead('document',versionFor),subjectScopes=subject?recScope(subject.type,recRead(subject.type,subject.id)):[],scope=subject?(subjectScopes.length===1?subjectScopes[0]:''):selectedScope();
  requireValue(!d||!d.archived,'Restore this document before uploading a version.');
  DOC_UPLOAD={actorId:recActor().id,rows:[],subject:subject&&{type:subject.type,id:subject.id},versionFor,scopeLocked:!!d||!!subject&&subjectScopes.length===1,baseVersionId:d?.currentVersionId||'',evidence,field,busy:false,meta:{folder:d?.folderId||DOC_FILTERS.folder||'F-COMPLIANCE',owner:d?.ownerId||recActor().id,classification:d?.classification||'Internal',scope:d?.orgScopeId||scope,scopeMode:d?.scopeMode||(scope?'scoped':'shared'),note:''},returnRoute:current.name+'?'+current.params.toString()};
  go('document-upload');
}
function docUploadView(){
  const u=DOC_UPLOAD;if(!u)return header('Upload files','Choose where the files belong.',link('Documents','documents'))+empty('No upload in progress','Start a new upload.',btn('Choose files','doc-upload','','primary'));
  const existing=u.versionFor&&recRead('document',u.versionFor),ready=u.rows.filter(r=>r.state==='Ready');
  return `<div class="crumb"><a href="#/documents">Documents</a> / ${existing?'New version':'Upload'}</div>`+header(existing?'Upload a new version':'Upload files',existing?existing.title:'Add files once. Link exact versions wherever they are needed.',btn('Cancel','doc-upload-cancel'),'Documents')+
    `<div class="doc-upload-layout"><section class="panel"><div class="doc-drop" data-doc-drop tabindex="0" role="button" aria-label="Choose or drop files"><span class="doc-upload-icon">${icon('plus')}</span><h2>${existing?'Drop the replacement file here':'Drop files here'}</h2><p>or ${btn('Browse files','doc-browse','','quiet')}</p><small>PDF, Office, images, text and CSV · 20 MB per file · 100 MB per batch</small><input id="doc-file-input" type="file" ${existing||u.field?'':'multiple'} accept="${DOC_ACCEPT}" hidden></div><div class="doc-upload-queue" aria-live="polite">${u.rows.length?u.rows.map(r=>`<div class="doc-queue-row"><span class="doc-filetype">${e(r.file.name.split('.').at(-1).toUpperCase())}</span><div class="grow"><strong>${e(r.file.name)}</strong><small>${recBytes(r.file.size)} · ${e(r.message||r.state)}</small>${r.duplicate?link('Open existing','document?id='+r.duplicate,'quiet small'):''}</div>${pill(r.state,r.state==='Error'?'red':r.state==='Ready'?'green':'amber')}${!u.busy?btn(icon('close'),'doc-remove-upload',`data-id="${r.id}" aria-label="Remove ${e(r.file.name)}"`,'icon-only quiet small'):''}</div>`).join(''):'<p class="pad muted">Selected files will appear here for review.</p>'}</div></section><section class="panel pad"><h2>File details</h2><div class="stack mt16">${select('folder','Folder',S.governance.folders.map(f=>[f.id,f.name]),u.meta.folder,'data-doc-meta="folder"')}${select('owner','Document owner',docUserOptions(),u.meta.owner,`data-doc-meta="owner" ${existing?'disabled':''}`)}${select('classification','Classification',['Internal','Confidential','Restricted'],u.meta.classification,`data-doc-meta="classification" ${existing?'disabled':''}`)}${select('scope','Accountable location',[['','Bank-wide'],...ORG.map(n=>[n.id,orgName(n.id)])],u.meta.scope,`data-doc-meta="scope" ${u.scopeLocked?'disabled':''}`)}${existing?area('note','What changed?',u.meta.note,'data-doc-meta="note" required minlength="10"'):''}${u.subject?`<div class="doc-context"><small>Attach to</small><strong>${e(recName(u.subject.type,recRead(u.subject.type,u.subject.id)))}</strong><span>${e(REC_TYPES[u.subject.type].label)} · ${e(u.subject.id)}</span></div>`:''}${u.evidence?`<div class="doc-context"><small>Evidence for ${e(u.evidence.question)}</small><strong>${e(findQ(u.evidence.question)?.title)}</strong><span>${e(targetName(u.evidence.target))}</span></div>`:''}<p class="field-help">The location is fixed for this upload. Header scope changes do not move the files.</p></div><div class="doc-check-note">${icon('shield')}<span>Format checks only. No malware scanner is connected. Use sample files, not customer data.</span></div><div id="doc-upload-error" class="form-error" role="alert"></div>${btn(u.busy?'Saving…':existing?'Save new version':'Add '+ready.length+' '+(ready.length===1?'file':'files'),'doc-save-upload',!ready.length||u.busy?'disabled':'','primary wide mt16')}</section></div>`;
}
async function docStage(files){
  const u=DOC_UPLOAD;requireValue(u&&!u.busy,'Start an upload first.');
  const incoming=[...files];requireValue(!(u.versionFor||u.field)||u.rows.length+incoming.length===1,u.field?'This answer accepts one file. Remove the current selection before choosing another.':'Choose exactly one replacement file.');
  requireValue(u.rows.length+incoming.length<=20,'Add no more than 20 files in one batch.');
  requireValue(u.rows.reduce((s,r)=>s+r.file.size,0)+incoming.reduce((s,f)=>s+f.size,0)<=100*1024*1024,'The batch limit is 100 MB.');
  for(const file of incoming){
    const row={id:uid('UP'),file,state:'Reading',message:'Checking file format…'};u.rows.push(row);if(current.name==='document-upload')render();
    try{
      requireValue(file.size<=20*1024*1024,'The per-file limit is 20 MB.');const bytes=new Uint8Array(await file.arrayBuffer());if(DOC_UPLOAD!==u||!u.rows.includes(row))return;const format=docValidateFile(file,bytes);
      const sha256=recSHA256(bytes),same=u.rows.find(r=>r!==row&&r.sha256===sha256&&r.state==='Ready');
      const existing=S.governance.versions.find(v=>v.sha256===sha256&&(()=>{const d=recRead('document',v.documentId);return d&&!d.archived&&d.orgScopeId===u.meta.scope;})());
      if(same)throw Error('This batch already contains identical file contents.');
      if(existing){row.duplicate=existing.documentId;throw Error('Identical contents already exist in this location. Link the existing document instead.');}
      Object.assign(row,{bytes,sha256,format,state:'Ready',message:'Format checked · SHA-256 recorded'});
    }catch(err){row.state='Error';row.message=err.message;}
    if(current.name==='document-upload')render();
  }
}
async function docCommitUpload(){
  const u=DOC_UPLOAD;requireValue(u&&!u.busy,'An upload is already being saved.');requireValue(u.actorId===recActor().id,'The acting user changed. Discard this batch and start under the intended user.');requireValue(!u.rows.some(r=>r.state==='Reading'),'File validation is still in progress.');requireValue(!(u.versionFor||u.field)||u.rows.filter(r=>r.state==='Ready').length===1,'This destination accepts exactly one file.');
  const ready=u.rows.filter(r=>r.state==='Ready');requireValue(ready.length,'Choose at least one valid file.');
  requireValue(recUser(u.meta.owner),'Choose a document owner.');requireValue(S.governance.folders.some(f=>f.id===u.meta.folder),'Choose an existing folder.');
  requireValue(!u.meta.scope||orgNode(u.meta.scope),'Choose a valid location.');
  const existing=u.versionFor&&recRead('document',u.versionFor);
  if(existing){requireValue(existing.currentVersionId===u.baseVersionId,'Another version was added. Start the replacement again.');requireValue(!existing.archived,'The document has been archived.');requireValue(u.meta.note.trim().length>=10,'Describe what changed in the replacement file.');}
  if(u.subject){requireValue(recRead(u.subject.type,u.subject.id),'The attachment target no longer exists.');const pseudo={scopeMode:u.meta.scope?'scoped':'shared',orgScopeId:u.meta.scope,archived:false};requireValue(docCanLink(pseudo,u.subject),'The selected location does not match the target.');}
  if(u.evidence)docEvidenceContext(u.evidence,{scopeMode:u.meta.scope?'scoped':'shared',orgScopeId:u.meta.scope,archived:false});
  if(u.field)requireValue(faInit().responses[u.field.responseKey]?.status!=='Submitted','The assessment was submitted; its attachments are locked.');
  if(typeof dm212CheckUpload==='function')dm212CheckUpload(u);
  u.busy=true;render();const created=[];
  try{
    for(const row of ready){const id=existing?.id||uid('DOC'),vid=uid('FILE');const storage=await DocumentBytes.put(vid,new Blob([row.bytes],{type:row.format.mime}));created.push({row,id,vid,storage});}
    requireValue(DOC_UPLOAD===u&&u.actorId===recActor().id,'The upload context changed during saving. No file was committed.');
    if(typeof dm212CheckUpload==='function')dm212CheckUpload(u);
    if(u.field)requireValue(faInit().responses[u.field.responseKey]?.status!=='Submitted','The assessment was submitted while the file was being saved.');
    if(existing)requireValue(existing.currentVersionId===u.baseVersionId&&!existing.archived,'The document changed while the file was saved. Try again.');
    if(u.subject)requireValue(!docLocked(u.subject.type,recRead(u.subject.type,u.subject.id)),'The target was submitted while the file was being saved.');
    if(u.evidence)docEvidenceContext(u.evidence,{scopeMode:u.meta.scope?'scoped':'shared',orgScopeId:u.meta.scope,archived:false});
    const at=new Date().toISOString();
    for(const {row,id,vid,storage}of created){
      const d=existing||{id,title:row.file.name,folderId:u.meta.folder,classification:u.meta.classification,orgScopeId:u.meta.scope,scopeMode:u.meta.scope?'scoped':'shared',ownerId:u.meta.owner,archived:false,createdAt:at};
      const version={id:vid,documentId:id,number:existing?Math.max(...S.governance.versions.filter(v=>v.documentId===id).map(v=>v.number))+1:1,fileName:row.file.name,mime:row.format.mime,size:row.file.size,sha256:row.sha256,storage,uploadedBy:recActor().id,createdAt:at,note:u.meta.note.trim()||'Initial upload'};
      S.governance.versions.push(version);d.currentVersionId=vid;
      if(!existing)S.governance.documents.unshift(d);
      else for(const req of S.governance.requests.filter(r=>r.subject.type==='document'&&r.subject.id===id&&r.status==='Pending')){req.status='Superseded';req.closedAt=at;recEvent('approval',req.subject,'Pending review superseded by a new file version',recActor().id,{requestId:req.id});}
      if(u.subject)docLinkVersion(vid,u.subject,u.field?.anchor||'');
      if(u.evidence)docEvidenceAdd(vid,u.evidence);
      if(u.field)docSetFormEvidence(vid,u.field);
      recEvent('upload',{type:'document',id},'Added '+row.file.name+' · v'+version.number,recActor().id,{documentVersionId:vid,sha256:version.sha256});
    }
    save();const next=(u.subject||u.field||u.evidence)?u.returnRoute:created.length===1?'document?id='+created[0].id:'documents';DOC_UPLOAD=null;go(next);toast(created.length+' '+(created.length===1?'file added.':'files added.')+(created.some(x=>x.storage==='session')?' Files are held in this tab only.':''));
  }catch(err){for(const x of created)if(!docVersion(x.vid))await DocumentBytes.remove(x.vid);u.busy=false;render();throw err;}
}
function docAttachmentRows(type,id,readOnly=false){
  const rows=docLinks(type,id);return rows.length?table(['Document','Linked version','Review',''],rows.map(l=>{
    const v=docVersion(l.documentVersionId),d=v&&recRead('document',v.documentId);return {cells:[`<strong>${e(d?.title||'Document unavailable')}</strong><small>${e(l.anchor||'Supporting document')}</small>`,`<strong>${docVersionLabel(v)}</strong><small>${v&&d&&d.currentVersionId!==v.id?'Newer version available':'Current version'}</small>`,v?docStatePill(docVersionStatus(v)):'—',v?link('Open',`document?id=${d.id}&version=${v.id}`,'small'):''+(readOnly?'':'')]};
  })):empty('No linked documents','Attach an existing document or upload a new file.');
}
function docAttachments(type,id,readOnly=false){
  return `<section class="panel doc-attachments" data-attachments-for="${e(type+':'+id)}"><div class="panel-head"><div><h2>Documents</h2><p>${readOnly?'Versions captured with this record.':'Linked versions stay fixed when a document is updated.'}</p></div>${!readOnly?btn(icon('link')+' Attach document','doc-attach',`data-type="${type}" data-id="${e(id)}"`,'small'):''}</div>${docAttachmentRows(type,id,readOnly)}</section>`;
}
function docAttachDialog(subject,field=null){
  requireValue(recRead(subject.type,subject.id),'The target record is unavailable.');
  if(field)requireValue(faInit().responses[field.responseKey]?.status!=='Submitted','The response has been submitted.');
  const docs=S.governance.documents.filter(d=>docCanLink(d,subject));
  openDialog('Attach a document',recName(subject.type,recRead(subject.type,subject.id)),`<div class="stack">${select('version','Document version',[['','Select a document'],...docs.map(d=>[d.currentVersionId,d.title+' · '+docVersionLabel(docHead(d))+' · '+docScopeLabel(d)])])}${input('anchor','Purpose / reference',field?.anchor||'','text','maxlength="160"')}<p class="field-help">This links the selected version. It does not copy the file or approve its contents.</p>${btn(icon('plus')+' Upload a new file','doc-attach-upload',`data-type="${subject.type}" data-id="${e(subject.id)}"`,'quiet')}</div>`,'Attach',form=>{
    const x=values(form);requireValue(x.version,'Select a document.');
    if(field)requireValue(faInit().responses[field.responseKey]?.status!=='Submitted','The response has been submitted.');
    docLinkVersion(x.version,subject,field?.anchor||x.anchor.trim());if(field)docSetFormEvidence(x.version,field);save();render();toast('Document linked.');
  });
  $('#dialog')._documentField=field;
}
function docView(){
  const d=recRead('document',current.params.get('id'));
  if(!d||!docVisible(d))return header('Document unavailable','The document is missing or outside the selected scope.',link('Documents','documents'));
  const v=docVersion(current.params.get('version')||d.currentVersionId);if(!v||v.documentId!==d.id)return header('Version unavailable','This version does not belong to this document.',link('Current document','document?id='+d.id));
  const tab=current.params.get('tab')||'overview',url=`document?id=${d.id}&version=${v.id}`,versions=S.governance.versions.filter(x=>x.documentId===d.id).sort((a,b)=>b.number-a.number),links=S.governance.links.filter(l=>versions.some(v=>v.id===l.documentVersionId)),requests=S.governance.requests.filter(r=>r.subject.type==='document'&&r.subject.id===d.id),historical=v.id!==d.currentVersionId;
  let body='';
  if(tab==='overview')body=`<div class="doc-detail-layout"><section class="panel"><div class="panel-head"><div><h2>Preview</h2><p>${e(v.fileName)} · ${recBytes(v.size)}</p></div>${v.storage!=='reference'?btn(icon('download')+' Download','doc-download',`data-id="${v.id}"`,'small'):''}</div><div class="doc-preview" data-doc-preview="${v.id}"><p class="pad muted">Loading file…</p></div></section><section class="panel pad"><h2>Document details</h2><dl class="doc-facts"><div><dt>Document ID</dt><dd>${e(d.id)}</dd></div><div><dt>Version</dt><dd>${docVersionLabel(v)}${historical?' · historical':''}</dd></div><div><dt>Owner</dt><dd>${e(recUser(d.ownerId)?.name||'Unassigned')}</dd></div><div><dt>Location</dt><dd>${e(docScopeLabel(d))}</dd></div><div><dt>Classification</dt><dd>${e(d.classification)}</dd></div><div><dt>Folder</dt><dd>${e(docFolderName(d.folderId))}</dd></div><div><dt>Added by</dt><dd>${e(recUser(v.uploadedBy)?.name||'Unknown')}</dd></div><div><dt>Added</dt><dd>${recTimestamp(v.createdAt)}</dd></div><div><dt>Change note</dt><dd>${e(v.note)}</dd></div><div><dt>Assessment reviews</dt><dd>${typeof d210ReviewCount==='function'?link(String(d210ReviewCount(d.id)),url+'&tab=assessment'):0}</dd></div><div><dt>File checks</dt><dd>${v.storage==='reference'?'No file available':'Format checked · not malware-scanned'}</dd></div></dl><details class="doc-checksum"><summary>File integrity</summary><p>SHA-256 of the original bytes</p><code>${e(v.sha256||'No checksum — reference only')}</code></details><hr class="divider"><div class="row wrap">${btn('Move to folder','doc-move',`data-id="${d.id}"`,'small')}${btn(d.archived?'Restore':'Archive','doc-archive',`data-id="${d.id}"`,'quiet small')}</div></section></div>`;
  else if(tab==='versions')body=panel('Version history','Existing links and decisions keep their original version.',table(['Version','Original file','Added by','Added','Change note',''],versions.map(x=>({cells:[docVersionLabel(x)+(x.id===d.currentVersionId?' · current':''),e(x.fileName),e(recUser(x.uploadedBy)?.name||'Unknown'),recTimestamp(x.createdAt),e(x.note),link('Open',`document?id=${d.id}&version=${x.id}`,'small')]}))));
  else if(tab==='links')body=panel('Linked records','Each link identifies a specific file version.',links.length?table(['Record','Type','File version','Reference',''],links.map(l=>{const ref=recRef(l.subject.type,l.subject.id);return {cells:[e(ref.name),e(REC_TYPES[l.subject.type]?.label||l.subject.type),docVersionLabel(docVersion(l.documentVersionId)),e(l.anchor||'—'),ref.route?link('Open record',ref.route,'small'):'Unavailable']};})):empty('Not linked','Attach this document from a program, vendor, risk or assessment.'));
  else if(tab==='assessment')body=typeof d210DocumentReviews==='function'?d210DocumentReviews(d,v):empty('Document reviews','No contextual review module is loaded.');
  else if(tab==='approvals')body=panel('Approval history','A completed review applies only to its captured version.',requests.length?approvalTable(requests):empty('No approval requests','Request review of the current file version.',!d.archived&&!historical&&v.storage!=='reference'?btn('Request approval','approval-new',`data-type="document" data-id="${d.id}"`,'primary'):''));
  else body=panel('Activity','',docTimeline(S.governance.events.filter(x=>x.ref.type==='document'&&x.ref.id===d.id)));
  return `<div class="crumb"><a href="#/documents">Documents</a> / ${e(docFolderName(d.folderId))}</div>`+header(e(d.title),`${d.id} · ${docVersionLabel(v)} · ${e(docScopeLabel(d))}`,!d.archived?(!historical&&v.storage!=='reference'?btn('Request approval','approval-new',`data-type="document" data-id="${d.id}"`,'primary')+' ':'')+btn('Upload new version','doc-replace',`data-id="${d.id}"`):'','Document')+`<div class="doc-record-status">${typeof d210DocumentState==='function'?d210DocumentState(d,v,historical,tab):(historical?pill('Historical version','amber')+docStatePill(docVersionStatus(v)):docStatePill(docStatus(d)))}${pill(d.classification)}${v.storage==='session'?pill('File available in this tab only','amber'):''}${historical?link('Open current version','document?id='+d.id,'quiet small'):''}</div>`+tabLinks([['Overview',url,'overview'],['Versions',url+'&tab=versions','versions'],['Linked records',url+'&tab=links','links'],['Approvals',url+'&tab=approvals','approvals'],['Assessment reviews',url+'&tab=assessment','assessment'],['Activity',url+'&tab=activity','activity']],tab)+body;
}
function docTimeline(events){return events.length?`<ol class="doc-timeline">${[...events].reverse().map(x=>`<li><strong>${e(x.message||x.decision)}</strong><p>${e(x.note||'')}</p><small>${e(recUser(x.actorId)?.name||x.actorName||'System')} · ${recTimestamp(x.at)}</small></li>`).join('')}</ol>`:empty('No activity yet','Changes to this document will appear here.');}
async function docLoadPreviews(){
  for(const el of $$('[data-doc-preview]')){
    const v=docVersion(el.dataset.docPreview);if(!v)continue;
    if(v.storage==='reference'){el.innerHTML=empty('Original file not supplied','This is an imported reference. Upload the source as a new version to preview or review its contents.');continue;}
    let blob=await DocumentBytes.get(v.id);
    if(!blob&&v.sampleText)blob=new Blob([v.sampleText],{type:v.mime});
    if(!el.isConnected)continue;
    if(!blob){el.innerHTML=empty('File unavailable in this browser','Its metadata remains, but the file bytes are not available. Re-upload the source; do not approve from metadata alone.');continue;}
    const bytes=new Uint8Array(await blob.arrayBuffer());
    if(!el.isConnected)continue;
    if(recSHA256(bytes)!==v.sha256){el.innerHTML=notice('The file checksum does not match its recorded version. Preview and approval are blocked.','warning');continue;}
    if(v.mime.startsWith('text/')){
      const text=new TextDecoder().decode(bytes.slice(0,300000));
      if(v.mime==='text/csv'){const rows=docParseCSV(text);el.innerHTML=rows.length?`<div class="doc-preview-caption">First ${Math.min(rows.length-1,100)} data rows · values are displayed as text</div>`+table(rows[0].slice(0,16).map(e),rows.slice(1,101).map(r=>({cells:rows[0].slice(0,16).map((_,i)=>e(r[i]||''))}))):empty('Empty CSV','No rows to display.');}
      else el.innerHTML=`<pre class="doc-text-preview">${e(text)}${bytes.length>300000?'\n\n[Preview truncated]':''}</pre>`;
    }else if(v.mime.startsWith('image/')){const url=URL.createObjectURL(blob);DOC_PREVIEW_URLS.push(url);el.innerHTML=`<img class="doc-image-preview" src="${url}" alt="Preview of ${e(v.fileName)}">`;}
    else if(v.mime==='application/pdf')el.innerHTML=`<div class="empty">${icon('file')}<h3>PDF document</h3><p>Open the original in the browser’s PDF viewer. The file has not been malware-scanned.</p>${btn('Open PDF preview','doc-open-pdf',`data-id="${v.id}"`,'primary')}${btn('Download original','doc-download',`data-id="${v.id}"`,'small')}</div>`;
    else el.innerHTML=empty('Office document','Download the original for review. An Office preview converter is not connected.',btn('Download original','doc-download',`data-id="${v.id}"`,'primary'));
  }
}
async function docVerifiedBytes(v){const b=await DocumentBytes.get(v.id)||(v.sampleText?new Blob([v.sampleText],{type:v.mime}):null);requireValue(b,'File bytes are unavailable. Upload the source before approving or downloading.');requireValue(recSHA256(new Uint8Array(await b.arrayBuffer()))===v.sha256,'The file checksum does not match the recorded version.');return b;}

/* Approval requests bind the material under review, not a mutable page URL. */
function approvalSnapshot(subject){
  const r=recRead(subject.type,subject.id);requireValue(r,'The source record is unavailable.');
  if(subject.type==='document'){
    const v=docHead(r);requireValue(v&&v.sha256,'Upload the original file before requesting approval.');requireValue(!r.archived,'Restore the document before requesting approval.');
    return {subject:{type:'document',id:r.id},title:r.title,versionId:v.id,version:v.number,fileName:v.fileName,sha256:v.sha256,size:v.size,mime:v.mime,classification:r.classification,orgScopeId:r.orgScopeId,scopeMode:r.scopeMode,uploadedBy:v.uploadedBy};
  }
  requireValue(subject.type==='report','Use the source workflow to review this record.');
  return {subject:{type:'report',id:r.id},title:r.name,report:clone(r)};
}
function approvalCurrent(request){
  try{return recHashText(recStable(approvalSnapshot(request.subject)))===request.snapshotHash;}catch(_){return false;}
}
function approvalScope(r){return r.orgScopeIds||[];}
function approvalVisible(r){const scope=selectedScope();return !scope||r.shared||approvalScope(r).some(id=>orgWithin(id,scope)||orgWithin(scope,id));}
function approvalNext(r){return r.steps.find(s=>s.state==='Pending');}
function approvalStart(subject){
  const snapshot=approvalSnapshot(subject),r=recRead(subject.type,subject.id),author=subject.type==='document'?docHead(r).uploadedBy:r.createdById||recUserId(r.createdBy||r.owner||'');
  requireValue(!S.governance.requests.some(x=>x.subject.type===subject.type&&x.subject.id===subject.id&&x.status==='Pending'),'This version already has an approval request. Open its Approvals tab.');
  requireValue(!S.governance.requests.some(x=>x.subject.type===subject.type&&x.subject.id===subject.id&&x.status==='Completed'&&x.snapshotHash===recHashText(recStable(snapshot))),'This version already has a completed approval. Upload a revision for another review.');
  const people=REC_PEOPLE.filter(p=>p.id!==recActor().id&&p.id!==author);
  DOC_APPROVAL_DRAFT={subject:clone(subject),snapshot,snapshotHash:recHashText(recStable(snapshot)),requestedBy:recActor().id,authorId:author,returnRoute:current.name+'?'+current.params.toString(),reviewer:people[0]?.id||'',signer:people[1]?.id||'',template:'review-sign',due:addDays(S.asOf,7),note:''};
  go('approval-request');
}
function approvalRequestView(){
  const d=DOC_APPROVAL_DRAFT;if(!d)return header('Request approval','Choose a document or a published report.',link('Documents','documents'));
  return `<div class="crumb"><a href="#/${e(d.returnRoute)}">${e(d.snapshot.title)}</a> / Request approval</div>`+header('Request approval','Choose the reviewers and confirm the exact version.',btn('Cancel','approval-cancel-draft'),'Review')+
  `<form id="approval-request-form" class="doc-request-layout"><section class="panel pad"><h2>Routing</h2><div class="stack mt16">${select('template','Workflow',[['review','Review only'],['review-sign','Review, then sign off']],d.template,'data-approval-draft="template"')}${select('reviewer','Reviewer',docUserOptions(),d.reviewer,'data-approval-draft="reviewer"')}${d.template==='review-sign'?select('signer','Final sign-off',docUserOptions(),d.signer,'data-approval-draft="signer"'):''}${input('due','Due date',d.due,'date',`data-approval-draft="due" required min="${S.asOf}"`)}${area('note','Message to reviewers',d.note,'data-approval-draft="note" maxlength="1500" placeholder="What should the reviewer check?"')}</div><div class="doc-routing">${pill('1 · Review','blue')}${d.template==='review-sign'?icon('arrow')+pill('2 · Sign-off'):''}</div><div class="form-error" id="approval-request-error" role="alert"></div><button type="submit" class="btn primary">Send for review</button></section><section class="panel pad"><h2>What is being approved</h2><dl class="doc-facts"><div><dt>Record</dt><dd>${e(d.snapshot.title)}</dd></div><div><dt>ID</dt><dd>${e(d.subject.id)}</dd></div><div><dt>Version</dt><dd>${d.snapshot.version?'v'+d.snapshot.version:'Published report snapshot'}</dd></div><div><dt>Requested by</dt><dd>${e(recUser(d.requestedBy)?.name)}</dd></div></dl><details class="doc-checksum"><summary>Snapshot checksum</summary><code>${e(d.snapshotHash)}</code></details><div class="doc-check-note">${icon('info')}<span>Reviewers act in order. The requester and file contributor cannot approve their own submission.</span></div><p class="field-help">Sign-off records a named acknowledgement in this demo. It is not a certificate-based digital signature.</p></section></form>`;
}
async function approvalSend(){
  const d=DOC_APPROVAL_DRAFT;requireValue(d,'Start an approval request.');
  requireValue(d.requestedBy===recActor().id,'The acting user changed. Start a new request under the intended requester.');
  const snapshot=approvalSnapshot(d.subject);requireValue(recHashText(recStable(snapshot))===d.snapshotHash,'The source changed. Review the new version before requesting approval.');
  const ids=d.template==='review-sign'?[d.reviewer,d.signer]:[d.reviewer];requireValue(ids.every(id=>recUser(id)),'Choose known reviewers.');
  requireValue(new Set(ids).size===ids.length,'Use different people for review and final sign-off.');
  requireValue(!ids.includes(d.requestedBy)&&!ids.includes(d.authorId),'The requester and file contributor cannot approve their own submission.');
  validDate(d.due,'Due date',false);requireValue(d.due>=S.asOf,'The due date cannot precede the workspace date.');
  requireValue(!S.governance.requests.some(r=>r.subject.type===d.subject.type&&r.subject.id===d.subject.id&&r.status==='Pending'),'A review is already pending for this record.');
  if(d.subject.type==='document')await docVerifiedBytes(docVersion(d.snapshot.versionId));
  // Recheck after asynchronous byte access, so a replacement or user switch cannot race a request.
  requireValue(d.requestedBy===recActor().id,'The acting user changed during verification. Start the request again.');
  requireValue(recHashText(recStable(approvalSnapshot(d.subject)))===d.snapshotHash,'The source changed while the file was being checked.');
  requireValue(!S.governance.requests.some(r=>r.subject.type===d.subject.type&&r.subject.id===d.subject.id&&r.status==='Pending'),'An approval request was already created.');
  const r=recRead(d.subject.type,d.subject.id),id=uid('APR'),scope=recScope(d.subject.type,r);
  const request={id,subject:clone(d.subject),title:d.snapshot.title,versionId:d.snapshot.versionId||'',snapshot:clone(d.snapshot),snapshotHash:d.snapshotHash,requestedBy:d.requestedBy,authorId:d.authorId,createdAt:new Date().toISOString(),due:d.due,message:d.note.trim(),status:'Pending',orgScopeIds:scope,shared:d.subject.type==='document'?r.scopeMode==='shared':!scope.length,steps:ids.map((actorId,i)=>({id:id+'-S'+(i+1),order:i+1,kind:i===1?'signoff':'review',actorId,state:'Pending',receiptId:null}))};
  S.governance.requests.push(request);recEvent('approval',request.subject,'Requested review'+(ids.length===2?' and sign-off':''),request.requestedBy,{requestId:id});
  DOC_APPROVAL_DRAFT=null;go('approval?id='+id);toast('Approval requested. No email was sent.');
}
function approvalTable(rows){return table(['Request','Version / next person','Due','State',''],rows.map(r=>({cells:[`<strong>${e(r.title)}</strong><small>${e(r.id)} · ${e(REC_TYPES[r.subject.type]?.label)}</small>`,`<strong>${r.snapshot.version?'v'+r.snapshot.version:'Report snapshot'}</strong><small>${r.status==='Pending'?e(recUser(approvalNext(r)?.actorId)?.name||'No pending step'):e(recUser(r.requestedBy)?.name||'Unknown')}</small>`,date(r.due),docStatePill(r.status),link('Open review','approval?id='+r.id,'small')]})));}
function approvalsView(){
  const tab=current.params.get('tab')||'mine',all=S.governance.requests.filter(approvalVisible),mine=all.filter(r=>r.status==='Pending'&&approvalNext(r)?.actorId===recActor().id);
  const rows=tab==='mine'?mine:tab==='sent'?all.filter(r=>r.requestedBy===recActor().id):tab==='completed'?all.filter(r=>r.status!=='Pending'):all;
  const domainRows=workItems().filter(x=>x.source!=='document-approval'&&/review|approval|decision/i.test(x.kind));
  return header('Approvals','Review the record, confirm the version and record a decision.',link('Work queue','work'),'Work')+docActorControl()+tabLinks([['Assigned to me ('+mine.length+')','approvals?tab=mine','mine'],['Requested by me','approvals?tab=sent','sent'],['All document reviews','approvals?tab=all','all'],['Completed','approvals?tab=completed','completed'],['Record reviews','approvals?tab=records','records']],tab)+(tab==='records'?panel('Record reviews','These decisions retain their domain-specific checks.',domainRows.length?table(['Record','Review','Owner','Due',''],domainRows.map(x=>({cells:[`<strong>${e(x.title)}</strong><small>${e(x.ref||x.id)}</small>`,e(x.kind),e(x.owner),date(x.due),link('Review',x.route,'small')]}))):empty('No record reviews pending','Submitted assessments and records appear here.')):panel(tab==='mine'?'Your pending reviews':'Document and report requests','',rows.length?approvalTable(rows):empty('No requests in this view','Open a document to request review, or switch the demo user to inspect another person’s queue.')));
}
function docActorControl(){return `<div class="doc-actor"><label>Demo user <select aria-label="Demo user" data-doc-actor>${options(docUserOptions(),recActor().id)}</select></label><span>Role simulation, not authentication</span></div>`;}
function approvalView(){
  const r=S.governance.requests.find(r=>r.id===current.params.get('id'));
  if(!r||!approvalVisible(r))return header('Approval unavailable','The request is missing or outside the selected scope.',link('Approval inbox','approvals'));
  const next=approvalNext(r),isCurrent=approvalCurrent(r),can=r.status==='Pending'&&next?.actorId===recActor().id&&isCurrent,receipts=S.governance.receipts.filter(x=>x.requestId===r.id);
  const original=r.snapshot.versionId?`<section class="panel"><div class="panel-head"><div><h2>Version under review</h2><p>${e(r.snapshot.fileName)} · v${r.snapshot.version}</p></div>${btn('Download original','doc-download',`data-id="${r.snapshot.versionId}"`,'small')}</div><div class="doc-preview" data-doc-preview="${r.snapshot.versionId}"></div></section>`:panel('Report snapshot','The report below is fixed at the time of the request.',`<div class="pad">${link('Open published report',recUrl('report',r.subject.id),'small')}<pre class="doc-text-preview">${e(JSON.stringify(r.snapshot.report?.snapshot||r.snapshot.report,null,2))}</pre></div>`);
  const steps=`<ol class="doc-approval-steps">${r.steps.map((s,i)=>`<li class="${s.state==='Approved'?'done':s.id===next?.id&&r.status==='Pending'?'current':''}"><b>${s.state==='Approved'?icon('check'):i+1}</b><div><strong>${s.kind==='signoff'?'Final sign-off':'Review'}</strong><small>${e(recUser(s.actorId)?.name)} · ${e(s.state)}</small></div></li>`).join('')}</ol>`;
  const controls=can?`<form id="approval-decision-form" data-id="${r.id}" data-step="${next.id}"><div class="stack">${select('decision','Decision',[[next.kind==='signoff'?'Sign off':'Approve',next.kind==='signoff'?'Sign off':'Approve'],['Request changes','Request changes'],['Reject','Reject']],next.kind==='signoff'?'Sign off':'Approve')}${area('note','Review note','','required minlength="10" placeholder="Record the basis for your decision."')}${next.kind==='signoff'?`${input('signature','Type your full name','','text','autocomplete="off"')}<label class="checkbox-row"><input type="checkbox" name="intent" value="confirmed"> I have reviewed this exact version and intend to record my sign-off.</label>`:''}</div><p class="field-help">Recorded as ${e(recActor().name)}. File approval does not establish requirement compliance.</p><div id="approval-decision-error" class="form-error" role="alert"></div><button type="submit" class="btn primary wide">${next.kind==='signoff'?'Record decision / sign-off':'Record decision'}</button></form>`:`<p>${r.status==='Pending'?!isCurrent?'The source changed. This request cannot be approved.':`Awaiting ${e(recUser(next?.actorId)?.name)}. Switch the demo user to continue.`:`Request ${e(r.status.toLowerCase())}. Decisions cannot be edited.`}</p>`;
  return `<div class="crumb"><a href="#/approvals">Approvals</a> / ${e(r.id)}</div>`+header(e(r.title),r.id+' · '+(r.snapshot.version?'Version '+r.snapshot.version:'Published snapshot'),btn('Export decision record','approval-export',`data-id="${r.id}"`),'Approval')+docActorControl()+`<div class="doc-record-status">${docStatePill(r.status)}<span>Requested by ${e(recUser(r.requestedBy)?.name)} · Due ${date(r.due)}</span></div>`+steps+`<div class="doc-review-layout"><div>${original}</div><section class="panel pad"><h2>${can?'Your decision':'Review status'}</h2>${r.message?`<p class="doc-request-message">${e(r.message)}</p>`:''}${controls}<hr class="divider"><details class="doc-checksum"><summary>Version binding</summary><code>${e(r.snapshotHash)}</code><p>SHA-256 of the captured review data. This checksum is not a digital signature.</p></details>${r.status==='Pending'&&r.requestedBy===recActor().id?btn('Withdraw request','approval-withdraw',`data-id="${r.id}"`,'quiet danger mt16'):''}</section></div>${panel('Decision record','Named internal acknowledgements under demo identities.',docTimeline(receipts))}`;
}
async function approvalDecide(form){
  const id=form.dataset.id,stepId=form.dataset.step,r=S.governance.requests.find(r=>r.id===id),v=values(form);
  requireValue(r&&r.status==='Pending','This request is no longer awaiting a decision.');
  const step=approvalNext(r),actor=recActor();
  requireValue(step?.id===stepId&&step.actorId===actor.id,'Only the assigned person can complete the next step.');
  requireValue(actor.id!==r.requestedBy&&actor.id!==r.authorId,'You cannot approve your own submission.');
  requireValue(recHashText(recStable(r.snapshot))===r.snapshotHash,'The captured review data changed. This request cannot be approved.');
  requireValue(approvalCurrent(r),'The source version changed. Request a new review.');
  requireValue(v.note.trim().length>=10,'Provide a review note of at least 10 characters.');
  const accepted=step.kind==='signoff'?'Sign off':'Approve';requireValue([accepted,'Request changes','Reject'].includes(v.decision),'Choose a valid decision.');
  if(v.decision==='Sign off'){requireValue(v.signature.trim()===actor.name,'Type your full name exactly as shown for the acting user.');requireValue(v.intent==='confirmed','Confirm your intention to sign off this version.');}
  if(r.subject.type==='document')await docVerifiedBytes(docVersion(r.versionId));
  // Reject double submissions and changes occurring while bytes were verified.
  requireValue(actor.id===recActor().id,'The acting user changed during verification. Review again under the intended identity.');
  requireValue(r.status==='Pending'&&approvalNext(r)?.id===stepId&&approvalCurrent(r),'This request changed during verification. Reload the review.');
  const receipt={id:uid('DEC'),requestId:r.id,stepId:step.id,subject:clone(r.subject),sourceVersion:r.versionId,snapshotHash:r.snapshotHash,actorId:actor.id,actorName:actor.name,at:new Date().toISOString(),decision:v.decision,note:v.note.trim(),intent:v.decision==='Sign off'?'I reviewed this exact version and intend to record my sign-off.':'',typedName:v.decision==='Sign off'?v.signature.trim():'',method:v.decision==='Sign off'?'Typed internal acknowledgement · demo identity':'Internal review · demo identity'};
  S.governance.receipts.push(receipt);step.state=v.decision===accepted?'Approved':v.decision==='Reject'?'Rejected':'Changes requested';step.receiptId=receipt.id;
  if(v.decision==='Reject')r.status='Rejected';else if(v.decision==='Request changes')r.status='Changes requested';else if(r.steps.every(s=>s.state==='Approved'))r.status='Completed';
  if(r.status!=='Pending')r.closedAt=receipt.at;
  recEvent('decision',r.subject,v.decision+' · '+r.title,actor.id,{requestId:r.id,receiptId:receipt.id});render();toast(v.decision==='Sign off'?'Internal sign-off recorded.':'Decision recorded.');
}
function docDataHealthView(){
  const x=recIntegrity(),refs=S.governance.versions.filter(v=>v.storage==='reference').length,sessions=S.governance.versions.filter(v=>v.storage==='session').length;
  return header('Data checks','Reference integrity and known coverage gaps.',btn('Export checks','doc-export-health'),'Workspace')+metricRow([metric('Reference errors',x.problems.length,'IDs, scope, version links and approval snapshots','link'),metric('Documents',x.counts.documents,x.counts.versions+' versions','file'),metric('Reference-only files',refs,'Original bytes were not supplied','info'),metric('Session files',sessions,'May become unavailable when this tab closes','clock')])+panel('Integrity results','Checks run over the same records used by screens and exports.',x.problems.length?table(['Code','Record','Issue'],x.problems.map(p=>({cells:[e(p.code),e(p.ref),e(p.message)]}))):'<div class="pad"><strong>No broken references found.</strong><p class="muted mb0">This checks local record consistency, not the truth or completeness of bank data.</p></div>')+panel('Shared contracts','',`<div class="pad"><dl class="doc-facts"><div><dt>Organization scope</dt><dd>Canonical branch, department and unit IDs; no ownership changes from header filters.</dd></div><div><dt>Money</dt><dd>Integer kobo for NGN. Gross, recoveries and net reuse the loss ledger.</dd></div><div><dt>Documents</dt><dd>One document identity, append-only file versions, typed record links.</dd></div><div><dt>Assessment documents</dt><dd>Program, target and question context linked to an exact document version. Freshness and acceptance are separate.</dd></div><div><dt>Reviews</dt><dd>Document requests and domain decisions share the Work inbox. Domain acceptance rules remain authoritative.</dd></div><div><dt>Exports</dt><dd>Live queries use the current scope and filters; published reports retain their captured population.</dd></div><div><dt>Local storage</dt><dd>Metadata in localStorage; file bytes in IndexedDB or an explicitly labelled session fallback.</dd></div></dl></div>`);
}

/* Integration adapters. A file is not copied into a vendor, question or report. */
function docFormField(attr){
  const get=key=>attr.match(new RegExp(key+'="([^"]+)"'))?.[1]||'';
  const fieldId=get('data-fa-answer')||get('data-fa-cell'),rowId=get('data-row'),columnId=get('data-col');
  return {responseKey:faResponseKey(),fieldId,rowId,columnId,anchor:[fieldId,rowId,columnId].filter(Boolean).join(' / ')};
}
function docSetFormEvidence(versionId,field){
  const response=faInit().responses[field.responseKey];requireValue(response&&response.status!=='Submitted','The assessment is read-only.');
  const v=docVersion(versionId),d=v&&recRead('document',v.documentId),subject={type:'formResponse',id:field.responseKey};
  requireValue(d&&docCanLink(d,subject),'Select a document version available to this assessment scope.');
  let row;
  if(field.rowId){const binding=faInit().bindings.find(x=>x.id===field.responseKey),definition=binding?faVersion(binding.formVersionId)?.definition:faInit().forms.find(x=>field.responseKey===x.id+'::preview');const def=definition?.fields.find(x=>x.id===field.fieldId);requireValue(def,'The field was removed.');const rows=response.answers[field.fieldId]||clone(def.rows);row=rows.find(x=>x.id===field.rowId);requireValue(row,'The table row was removed.');requireValue(def.columns?.some(c=>c.id===field.columnId),'The column was removed.');response.answers[field.fieldId]=rows;}
  docLinkVersion(versionId,subject,field.anchor);
  if(row){row.cells??={};row.cells[field.columnId]=versionId;}else response.answers[field.fieldId]=versionId;
  save();
}
const docOldBasicControl=faBasicControl;
faBasicControl=function(type,value,attr,choices=[],disabled=false){
  if(type!=='evidence')return docOldBasicControl(type,value,attr,choices,disabled);
  const v=docVersion(value),field=docFormField(attr),payload=e(JSON.stringify(field));
  return `<div class="doc-evidence-input">${v?link(icon('file')+e(v.fileName)+' · v'+v.number,`document?id=${v.documentId}&version=${v.id}`,'quiet small'):`<span class="muted">${value?'Unlinked reference: '+e(value):'No document attached'}</span>`}${!disabled?btn(icon('link')+(v?' Change':' Attach document'),'doc-form-file',`data-field="${payload}"`,'small'):''}</div>`;
};
const docOldFormErrors=faErrors;
faErrors=function(def,res){
  const errors=docOldFormErrors(def,res);
  for(const f of def.fields){if(!faVisible(f,res.answers)||f.mode==='integration')continue;
    if(f.type==='evidence'&&res.answers[f.id]&&!docVersion(res.answers[f.id]))errors.push(f.label+': attach a document from the library.');
    if(f.columns){for(const row of res.answers[f.id]||[])for(const c of f.columns)if(c.type==='evidence'&&row.cells?.[c.id]&&!docVersion(row.cells[c.id]))errors.push(f.label+' / '+c.label+': select a document version.');}
  }return errors;
};
const docOldFaExample=faExample;
faExample=function(){docOldFaExample();const def=faDefinition(),res=faResponse(),version=S.governance.documents.find(d=>d.id==='DOC-SAMPLE-POLICY')?.currentVersionId;if(!version||res.status==='Submitted')return;for(const f of def.fields)if(f.type==='evidence'){res.answers[f.id]=version;docLinkVersion(version,{type:'formResponse',id:faResponseKey()},f.id);}save();render();};
const docOldFaSubmit=faSubmit;
faSubmit=function(){docOldFaSubmit();if(!modalSave)return;const before=modalSave,key=faResponseKey();modalSave=form=>{const result=before(form),r=faInit().responses[key];if(r.status==='Submitted'){r.documentVersionIds=[...new Set(docLinks('formResponse',key).map(l=>l.documentVersionId))];r.snapshot.documentVersionIds=clone(r.documentVersionIds);r.submissionHash=submissionDigest(faInit().bindings.find(b=>b.id===key),r);save();render();}return result;};};
function docEvidenceDialog(qid=''){
  const p=['program','assessment'].includes(current.name)?prog():S.programs[0],target=['program','assessment'].includes(current.name)?selectedTarget(p):p.targets[0],choices=S.programs.flatMap(z=>z.targets.map(t=>[z.id+'::'+t,z.name+' / '+targetName(t)]));
  function renderDialog(pid=p.id,t=target,q=qid){const pg=recRead('program',pid),questions=uniqueQ(activeReqs(pg)),docs=S.governance.documents.filter(d=>docCanLink(d,{type:'program',id:pid}));
    openDialog('Link document to assessment','Choose a file and confirm what it supports.',`<div class="stack">${select('scope','Program & target',choices,pid+'::'+t,'data-doc-evidence-scope')}${select('question','Assessment question',questions.map(x=>[x.id,x.id+' · '+x.title]),questions.some(x=>x.id===q)?q:questions[0]?.id)}${input('observed','Observation date',S.asOf,'date',`required max="${S.asOf}"`)}${select('version','Existing document',[['','Choose a document'],...docs.map(d=>[d.currentVersionId,d.title+' · '+docVersionLabel(docHead(d))])])}${btn(icon('plus')+' Upload a file instead','doc-evidence-upload','','quiet')}<p class="field-help">Freshness comes from the question policy. The review records whether this version supports the selected assessment.</p></div>`,'Link document',form=>{const x=values(form),[programId,tar]=x.scope.split('::');requireValue(x.version,'Choose a document or upload a file.');docEvidenceAdd(x.version,{programId,target:tar,question:x.question,observed:x.observed});render();toast('Evidence submitted for review.');});
    $('#dialog')._evidenceRebuild=renderDialog;
  }renderDialog();
}
addEvidence=docEvidenceDialog;
evidenceTable=function(items,compact=false){return panel('Documents',compact?'':'Files linked to assessment questions.',table(['Document / assessment','Observed','Freshness','Acceptance',''],items.map(x=>{const v=docVersion(x.documentVersionId);return {attrs:`data-search="${e((v?.fileName||x.name)+' '+x.target+' '+x.owner)}"`,cells:[`<strong>${e(v?.fileName||x.name)}</strong><small>${e(targetName(x.target))} · ${e(x.question||'General evidence')}${v?' · v'+v.number:''}</small>`,date(x.observed),`${pill(evQuality(x))}<small>Expires ${date(x.expires)}</small>`,pill(x.status),btn('Review','evidence-detail',`data-id="${x.id}"`,'small')]};})));};
evidenceView=function(){const rows=scopeItems('evidence');return header('Documents','Review supporting files in their assessment context.',link('Document library','documents')+' '+btn(icon('plus')+' Link document','add-evidence','','primary'),'Programs')+metricRow([metric('Document reviews',rows.length,'In the selected scope','file'),metric('Current and accepted',rows.filter(x=>x.status==='Accepted'&&evQuality(x)==='Current').length,'Reviewed and within the freshness period','check'),metric('Needs refresh',rows.filter(x=>evQuality(x)!=='Current').length,'Original review remains in history','clock'),metric('Awaiting review',rows.filter(x=>x.status==='Submitted').length,'Pending an acceptance decision','work')])+tools('Search documents')+evidenceTable(rows)+noMatch;};
const docOldEvidenceView=jEvidenceReview;
jEvidenceReview=function(){const html=docOldEvidenceView(),x=recRead('evidence',current.params.get('id')||S.evidence[0]?.id),v=x&&docVersion(x.documentVersionId);return html.replace(/<div class="j-evidence-placeholder">[\s\S]*?<\/div>/,v?`<section class="panel"><div class="panel-head"><h2>Source document · v${v.number}</h2>${link('Document history',`document?id=${v.documentId}&version=${v.id}`,'small')}</div><div class="doc-preview" data-doc-preview="${v.id}"></div></section>`:'');};
window.CS_JOURNEYS['evidence-review']=()=>jEvidenceReview();

// Decision receipts are a common audit representation, not a second approval state.
function docWrapDecision(original,resolver){return function(form){
  const ref=resolver(form),r=ref&&recRead(ref.type,ref.id),snapshot=r?clone(r):null,v=values(form);
  const result=original(form);
  if(ref&&snapshot){const after=recRead(ref.type,ref.id);if(recStable(snapshot)!==recStable(after)){const actor=v.reviewer||v.approver||v.respondent||recActor().name;recDecision(ref.type,ref.id,actor,v.decision||'Review completed',v.note||v.rationale||'',{record:snapshot,documents:docLinks(ref.type,ref.id).map(l=>({linkId:l.id,versionId:l.documentVersionId,sha256:docVersion(l.documentVersionId)?.sha256||null}))},snapshot.documentVersionId||String(snapshot.revision||''));}}
  return result;
};}
jEvidenceDecision=docWrapDecision(jEvidenceDecision,f=>({type:'evidence',id:f.dataset.id}));
jActivityDecision=docWrapDecision(jActivityDecision,f=>({type:'activity',id:f.dataset.id}));
jDpiaDecision=docWrapDecision(jDpiaDecision,f=>({type:'dpia',id:f.dataset.id}));
jCaseDecision=docWrapDecision(jCaseDecision,f=>({type:'privacyCase',id:f.dataset.id}));
jRequirementDecision=docWrapDecision(jRequirementDecision,()=>({type:'program',id:prog().id}));
ixReviewReturn=docWrapDecision(ixReviewReturn,()=>({type:'kriReturn',id:ixCurrentReturn()?.id}));
rcsaDecision=docWrapDecision(rcsaDecision,()=>({type:'rcsa',id:current.params.get('id')}));

const docOldWorkItems=workItems;
workItems=function(){
  let rows=docOldWorkItems();if(!S.governance)return rows;
  rows=rows.flatMap(x=>{if(x.kind!=='Evidence review')return [x];const ev=recRead('evidence',x.ref);return ev?.status==='Submitted'?[{...x,source:'evidence-review',route:'evidence-review?id='+ev.id}]:[];});
  for(const ev of scopeItems('evidence'))if(ev.status==='Submitted'&&!rows.some(x=>x.ref===ev.id&&x.kind==='Evidence review'))rows.push({id:'EVIDENCE-REVIEW-'+ev.id,ref:ev.id,title:'Review evidence · '+(docVersion(ev.documentVersionId)?.fileName||ev.name),kind:'Evidence review',owner:recRead('program',ev.program)?.reviewer||'Ifeoma Nwosu',due:ev.expires,status:'Open',source:'evidence-review',route:'evidence-review?id='+ev.id,orgScopeId:ev.orgScopeId});
  for(const r of S.governance.requests)if(r.status==='Pending'&&approvalVisible(r)){const step=approvalNext(r);rows.push({id:r.id,ref:r.subject.id,title:(step?.kind==='signoff'?'Sign off · ':'Review · ')+r.title,kind:step?.kind==='signoff'?'Document sign-off':'Document approval',owner:recUser(step?.actorId)?.name||'Unassigned',due:r.due,status:'Open',source:'document-approval',route:'approval?id='+r.id,orgScopeIds:r.orgScopeIds});}
  return rows;
};
function docLocked(type,r){
  if(type==='formResponse')return r.status==='Submitted';
  if(type==='kriReturn')return ['Submitted','Approved'].includes(r.status);
  if(['activity','dpia','privacyCase'].includes(type))return ['In review','Active','Approved','Closed','Awaiting review'].includes(r.state);
  if(type==='report')return true;
  return false;
}
async function docSaveReport(id){
  const p=recRead('report',id);requireValue(p,'The report is unavailable.');
  const content=p.name+'\nManagement report snapshot\n\n'+JSON.stringify(p,null,2),hash=recHashText(content);
  const existing=S.governance.versions.find(v=>v.sha256===hash);if(existing){go('document?id='+existing.documentId);return;}
  const did=uid('DOC'),vid=uid('FILE'),at=new Date().toISOString(),scope=recScope('report',p)[0]||'',name=p.name.replace(/[^a-zA-Z0-9 _.-]/g,'').slice(0,130)+'.txt',blob=new Blob([content],{type:'text/plain'}),storage=await DocumentBytes.put(vid,blob);
  S.governance.documents.unshift({id:did,title:p.name,folderId:'F-REPORTS',classification:'Confidential',orgScopeId:scope,scopeMode:scope?'scoped':'shared',ownerId:recActor().id,currentVersionId:vid,archived:false,createdAt:at});
  S.governance.versions.push({id:vid,documentId:did,number:1,fileName:name,mime:'text/plain',size:blob.size,sha256:hash,storage,uploadedBy:recActor().id,createdAt:at,note:'Generated from published report '+id});
  docLinkVersion(vid,{type:'report',id},'',true);recEvent('generated',{type:'document',id:did},'Saved published report '+id);go('document?id='+did);
}
function docSeed(){
  if(S.governance.samplesCreated)return;
  const samples=[
    {id:'DOC-SAMPLE-POLICY',title:'Access review procedure',folder:'F-COMPLIANCE',name:'access-review-procedure.txt',scope:'',mime:'text/plain',text:'MERIDIAN BANK — SAMPLE DOCUMENT\nAccess review procedure\nVersion 1.0\n\n1. The control owner prepares the account population.\n2. A reviewer confirms that access remains appropriate.\n3. Exceptions receive an owner, due date and closure evidence.\n4. The final sign-off identifies the reviewed extract version.\n\nThis document is synthetic and is not a bank policy.\n'},
    {id:'DOC-SAMPLE-KRI',title:'Ikeja payment observations · October',folder:'F-OPERATIONS',name:'ikeja-payment-observations.csv',scope:'U-IK-PAY',mime:'text/csv',text:'Metric,Numerator,Denominator,Period,Source\nFailed payments,120,10000,2026-10,Synthetic example\nUnresolved reconciliation items,14,,2026-10,Synthetic example\nAged complaints,2,,2026-10,Synthetic example\n'},
    {id:'DOC-SAMPLE-VENDOR',title:'Hosting service review notes',folder:'F-VENDORS',name:'hosting-service-review.txt',scope:'U-HO-INFRA',mime:'text/plain',text:'SAMPLE — HOSTING SERVICE REVIEW\nEngagement: Core banking hosting\n\nReview scope\n- Access administration\n- Recovery exercise evidence\n- Incident notification contacts\n\nOpen item: obtain the latest recovery exercise report.\n\nThese are synthetic notes, not a provider certification.\n'}
  ];
  for(const x of samples){const vid=x.id+'-V1',text=x.text;S.governance.documents.push({id:x.id,title:x.title,folderId:x.folder,classification:'Internal',orgScopeId:x.scope,scopeMode:x.scope?'scoped':'shared',ownerId:'USR-01',currentVersionId:vid,archived:false,createdAt:'2026-11-13T08:00:00Z'});S.governance.versions.push({id:vid,documentId:x.id,number:1,fileName:x.name,mime:x.mime,size:new TextEncoder().encode(text).length,sha256:recHashText(text),storage:'sample',sampleText:text,uploadedBy:'USR-01',createdAt:'2026-11-13T08:00:00Z',note:'Synthetic example file'});}
  docLinkVersion('DOC-SAMPLE-VENDOR-V1',{type:'engagement',id:'ENG-01'});
  S.governance.samplesCreated=true;save();
}
function docMoveDialog(ids){
  const ds=ids.map(id=>recRead('document',id));requireValue(ds.every(Boolean),'A selected document is unavailable.');
  openDialog('Move '+ds.length+' '+(ds.length===1?'document':'documents'),'File versions, scope and linked records remain unchanged.',select('folder','Destination folder',S.governance.folders.map(f=>[f.id,f.name]),ds[0]?.folderId),'Move',form=>{const folder=values(form).folder;requireValue(S.governance.folders.some(f=>f.id===folder),'Choose an existing folder.');for(const d of ds){d.folderId=folder;recEvent('move',{type:'document',id:d.id},'Moved to '+docFolderName(folder));}DOC_SELECTED.clear();render();toast('Folder updated.');});
}
function docArchiveDialog(ids){
  const docs=ids.map(id=>recRead('document',id));requireValue(docs.every(Boolean),'A selected document is unavailable.');const restore=docs.every(d=>d.archived);
  requireValue(restore||!S.governance.requests.some(r=>r.status==='Pending'&&r.subject.type==='document'&&ids.includes(r.subject.id)),'Withdraw the pending approval before archiving this document.');
  openDialog(restore?'Restore documents?':'Archive documents?',restore?'Documents will reappear in the active library.':'Existing version links and decision records are retained.',`<p>${docs.length} ${docs.length===1?'document':'documents'} selected. No file bytes are deleted.</p>`,restore?'Restore':'Archive',()=>{for(const d of docs){d.archived=!restore;recEvent('archive',{type:'document',id:d.id},restore?'Restored document':'Archived document');}DOC_SELECTED.clear();render();});
}
function docAsync(fn){return (...args)=>Promise.resolve().then(()=>fn(...args)).catch(err=>{const box=$('#doc-upload-error')||$('#approval-request-error')||$('#approval-decision-error')||$('#modal-error');uxShowError(box,err.message);});}
Object.assign(actions,{
  'doc-upload':()=>docStartUpload(),
  'doc-replace':el=>docStartUpload({versionFor:el.dataset.id}),
  'doc-browse':()=>$('#doc-file-input').click(),
  'doc-save-upload':docAsync(docCommitUpload),
  'doc-remove-upload':el=>{requireValue(!DOC_UPLOAD.busy,'Wait for the current save to finish.');DOC_UPLOAD.rows=DOC_UPLOAD.rows.filter(x=>x.id!==el.dataset.id);render();},
  'doc-upload-cancel':()=>{requireValue(!DOC_UPLOAD?.busy,'The files are being saved. This upload cannot be discarded during the save.');const route=DOC_UPLOAD?.returnRoute||'documents';if(DOC_UPLOAD?.rows.length)openDialog('Discard selected files?','Nothing will be uploaded.',`<p>Remove the ${DOC_UPLOAD.rows.length} selected files from this upload?</p>`,'Discard',()=>{DOC_UPLOAD=null;go(route);});else{DOC_UPLOAD=null;go(route);}},
  'doc-folder':el=>{DOC_FILTERS.folder=el.dataset.id;DOC_FILTERS.archived=false;DOC_SELECTED.clear();window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT?.rememberFilters?.();render();},
  'doc-archived':()=>{DOC_FILTERS.archived=true;DOC_FILTERS.folder='';DOC_FILTERS.status='';DOC_SELECTED.clear();window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT?.rememberFilters?.();render();},
  'doc-clear-selected':()=>{DOC_SELECTED.clear();render();},
  'doc-move':el=>docMoveDialog([el.dataset.id]),
  'doc-bulk-move':()=>docMoveDialog([...DOC_SELECTED].filter(id=>docRows().some(d=>d.id===id))),
  'doc-archive':el=>docArchiveDialog([el.dataset.id]),
  'doc-bulk-archive':()=>docArchiveDialog([...DOC_SELECTED].filter(id=>docRows().some(d=>d.id===id))),
  'doc-new-folder':()=>openDialog('New folder','Folders organize documents; they do not change scope or access.',input('name','Folder name','','text','required maxlength="60"'),'Create folder',form=>{const name=values(form).name.trim();requireValue(name.length>0,'Enter a folder name.');requireValue(!S.governance.folders.some(f=>f.name.toLowerCase()===name.toLowerCase()),'That folder already exists.');const id=uid('FOLDER');S.governance.folders.push({id,name});save();DOC_FILTERS.folder=id;DOC_FILTERS.archived=false;window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT?.rememberFilters?.();render();}),
  'doc-download':docAsync(async el=>{const v=docVersion(el.dataset.id);requireValue(v,'File version not found.');docBlobDownload(await docVerifiedBytes(v),v.fileName);}),
  'doc-open-pdf':docAsync(async el=>{const v=docVersion(el.dataset.id);requireValue(v?.mime==='application/pdf','This is not a PDF document.');const b=await docVerifiedBytes(v),url=URL.createObjectURL(new Blob([b],{type:'application/pdf'}));window.open(url,'_blank','noopener,noreferrer');setTimeout(()=>URL.revokeObjectURL(url),300000);}),
  'doc-attach':el=>{const subject={type:el.dataset.type,id:el.dataset.id};requireValue(!docLocked(subject.type,recRead(subject.type,subject.id)),'This record is locked. Create a revision before changing its attachments.');docAttachDialog(subject);},
  'doc-attach-upload':el=>{const field=$('#dialog')._documentField||null,subject={type:el.dataset.type,id:el.dataset.id};closeDialog();docStartUpload({subject,field});},
  'doc-form-file':el=>{const f=JSON.parse(el.dataset.field);docAttachDialog({type:'formResponse',id:f.responseKey},f);},
  'doc-evidence-upload':()=>{const v=values($('#modal-form')),[programId,target]=v.scope.split('::');validDate(v.observed,'Observation date');const evidence={programId,target,question:v.question,observed:v.observed};closeDialog();docStartUpload({subject:{type:'program',id:programId},evidence});},
  'doc-export':()=>{const rows=docRows();exportCSV('clearsight-documents.csv',['Document ID','Title','Version ID','Version','Original filename','Bytes','SHA-256','Scope ID','Scope','Owner ID','Owner','Folder','Classification','State','Storage'],rows.map(d=>{const v=docHead(d);return[d.id,d.title,v.id,v.number,v.fileName,v.size,v.sha256||'',d.orgScopeId||'',docScopeLabel(d),d.ownerId,recUser(d.ownerId)?.name,docFolderName(d.folderId),d.classification,docStatus(d),v.storage];}));},
  'doc-export-health':()=>download('clearsight-data-checks.json',JSON.stringify(recIntegrity(),null,2)),
  'doc-save-report':docAsync(el=>docSaveReport(el.dataset.id)),
  'approval-new':el=>approvalStart({type:el.dataset.type,id:el.dataset.id}),
  'approval-cancel-draft':()=>{const url=DOC_APPROVAL_DRAFT?.returnRoute||'approvals';DOC_APPROVAL_DRAFT=null;go(url);},
  'approval-withdraw':el=>{const r=S.governance.requests.find(r=>r.id===el.dataset.id);requireValue(r?.status==='Pending'&&r.requestedBy===recActor().id,'Only the requester can withdraw a pending review.');openDialog('Withdraw this request?','Earlier decisions remain in the audit record.',area('reason','Reason','','required minlength="10"'),'Withdraw',form=>{requireValue(r.status==='Pending','The request has already closed.');const note=values(form).reason.trim();requireValue(note.length>=10,'Provide a reason.');r.status='Cancelled';r.closedAt=new Date().toISOString();recEvent('withdraw',r.subject,note,recActor().id,{requestId:r.id});render();});},
  'approval-export':el=>{const r=S.governance.requests.find(r=>r.id===el.dataset.id);requireValue(r&&approvalVisible(r),'Request unavailable.');download('clearsight-'+r.id+'-decision-record.json',JSON.stringify({schema:'clearsight.internal-approval/1',demonstration:true,signatureType:'Named internal acknowledgement; no signing key or certificate',request:r,decisions:S.governance.receipts.filter(x=>x.requestId===r.id)},null,2));},
  'doc-help':()=>openDialog('About this workspace','Local demonstration · synthetic records only.',`<div class="stack"><p>Uploads are stored in this browser when IndexedDB is available. Otherwise file bytes exist only in the current tab. Metadata uses separate browser storage.</p><p>Checks validate format, size and a SHA-256 checksum. There is no malware scanning, authentication, remote storage, email delivery or legally certified signing service.</p><p>Internal sign-off captures a name, intention, timestamp and exact version under a selectable demo identity. The checksum is not a digital signature.</p><p>Production must enforce authorization, scan files, protect audit records and verify signers on the server. Do not use this prototype for customer data.</p>${link('Data checks','data-health','small')}</div>`)
});
let docFilterTimer;
document.addEventListener('input',ev=>{
  const el=ev.target;
  if(el.matches('[data-doc-filter="search"]')){DOC_FILTERS.search=el.value;window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT?.rememberFilters?.();clearTimeout(docFilterTimer);docFilterTimer=setTimeout(()=>{const value=DOC_FILTERS.search;render();const x=$('[data-doc-filter="search"]');x?.focus();x?.setSelectionRange(value.length,value.length);},160);}
  if(el.matches('[data-doc-meta]')&&DOC_UPLOAD&&!DOC_UPLOAD.busy){DOC_UPLOAD.meta[el.dataset.docMeta]=el.value;if(el.dataset.docMeta==='scope')DOC_UPLOAD.meta.scopeMode=el.value?'scoped':'shared';}
  if(el.matches('[data-approval-draft]')&&DOC_APPROVAL_DRAFT)DOC_APPROVAL_DRAFT[el.dataset.approvalDraft]=el.value;
});
document.addEventListener('change',ev=>{
  const el=ev.target;
  try{
    if(el.matches('[data-doc-filter]')&&el.dataset.docFilter!=='search'){DOC_FILTERS[el.dataset.docFilter]=el.value;DOC_SELECTED.clear();window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT?.rememberFilters?.();render();}
    if(el.matches('[data-doc-select]')){el.checked?DOC_SELECTED.add(el.dataset.docSelect):DOC_SELECTED.delete(el.dataset.docSelect);docRefreshBulk();}
    if(el.matches('[data-doc-select-all]')){for(const d of docRows())el.checked?DOC_SELECTED.add(d.id):DOC_SELECTED.delete(d.id);for(const x of $$('[data-doc-select]'))x.checked=el.checked;docRefreshBulk();}
    if(el.id==='doc-file-input')docAsync(()=>docStage(el.files))();
    if(el.matches('[data-doc-meta]')&&DOC_UPLOAD&&!DOC_UPLOAD.busy){DOC_UPLOAD.meta[el.dataset.docMeta]=el.value;if(el.dataset.docMeta==='scope')DOC_UPLOAD.meta.scopeMode=el.value?'scoped':'shared';}
    if(el.matches('[data-doc-actor]')){requireValue(recUser(el.value),'Unknown user.');S.governance.actorId=el.value;save();render();}
    if(el.matches('[data-approval-draft]')&&DOC_APPROVAL_DRAFT){DOC_APPROVAL_DRAFT[el.dataset.approvalDraft]=el.value;if(el.dataset.approvalDraft==='template')render();}
    if(el.matches('[data-doc-evidence-scope]')){const [id,target]=el.value.split('::');$('#dialog')._evidenceRebuild?.(id,target,'');}
  }catch(err){toast(err.message);}
});
function docRefreshBulk(){const ids=[...DOC_SELECTED].filter(id=>docRows().some(d=>d.id===id));const el=$('#doc-bulk');if(el){el.hidden=!ids.length;$('strong',el).textContent=ids.length+' selected';}}
document.addEventListener('submit',ev=>{
  if(ev.target.id==='approval-request-form'){ev.preventDefault();docAsync(approvalSend)();}
  if(ev.target.id==='approval-decision-form'){ev.preventDefault();docAsync(approvalDecide)(ev.target);}
});
document.addEventListener('dragover',ev=>{const z=ev.target.closest('[data-doc-drop]');if(z){ev.preventDefault();z.classList.add('dragging');}});
document.addEventListener('dragleave',ev=>ev.target.closest('[data-doc-drop]')?.classList.remove('dragging'));
document.addEventListener('drop',ev=>{const z=ev.target.closest('[data-doc-drop]');if(z){ev.preventDefault();z.classList.remove('dragging');docAsync(()=>docStage(ev.dataTransfer.files))();}});
document.addEventListener('keydown',ev=>{if(ev.target.matches('[data-doc-drop]')&&['Enter',' '].includes(ev.key)){ev.preventDefault();$('#doc-file-input')?.click();}});

// Make all surviving domain screens use the same shared attachment surface.
Object.assign(window.CS_JOURNEYS,{documents:documentsView,document:docView,'document-upload':docUploadView,approvals:approvalsView,approval:approvalView,'approval-request':approvalRequestView,'data-health':docDataHealthView});
Object.assign(window.CS_JOURNEY_GROUPS,{documents:'documents',document:'documents','document-upload':'documents',approvals:'work',approval:'work','approval-request':'work','data-health':'settings'});
Object.assign(titles,{documents:'Documents',document:'Document', 'document-upload':'Upload files',approvals:'Approvals',approval:'Approval review','approval-request':'Request approval','data-health':'Data checks'});
const docOldPublishReport=actions['ix-publish-report'];
actions['ix-publish-report']=()=>{const count=S.reportPacks.length;docOldPublishReport();if(S.reportPacks.length>count){S.reportPacks[0].createdById=recActor().id;save();}};
const docPriorReportData=reportData;
reportData=function(){const out=docPriorReportData();if(S.governance)out.documents={schema:'clearsight.documents/1',scopeId:selectedScope(),active:S.governance.documents.filter(d=>!d.archived&&docVisible(d)).length,pendingApprovals:S.governance.requests.filter(r=>r.status==='Pending'&&approvalVisible(r)).length,referenceOnly:S.governance.documents.filter(d=>!d.archived&&docVisible(d)&&docHead(d)?.storage==='reference').length};return out;};
// Keep operational copy short; detailed explanations belong in help and source views.
const docOriginalHeader=header;
const DOC_COPY={
 'Framework library':'Choose a baseline and tailor it to the assessment scope.',
 'Compliance programs':'Track requirements, assessments and review outcomes.',
 'Vendor relationships':'Suppliers, service engagements and assurance reviews.',
 'Risk & losses':'Risk ratings, control effectiveness and recorded losses.',
 'Key indicators':'Monitor thresholds, trends and missing readings.',
 'Work & approvals':'Reviews, actions and due dates.',
 'Reports & insights':'Compare locations, losses, indicators and outstanding actions.',
 'Workspace settings':'Preferences, activity and data checks.',
 'Branch KRI returns':'Collect, review and compare monthly returns.',
 'Privacy & data':'Processing records, impact assessments, requests and incidents.',
 'Create KRI reporting cycle':'Set the reporting period, units and reviewers.'
};
header=function(title,subtitle,action='',eyebrow='',crumb=''){const label=['Decision intelligence','Enterprise control tower','Assessment instruments'].includes(eyebrow)?'':eyebrow;return docOriginalHeader(title,DOC_COPY[title]??subtitle,action,label.toLowerCase()===title.toLowerCase()?'':label,crumb);};
actions['fa-example']=()=>faExample();actions['fa-submit']=()=>faSubmit();
const docPriorRender=render;
render=function(){
  recordsInit();docSeed();for(const url of DOC_PREVIEW_URLS)URL.revokeObjectURL(url);DOC_PREVIEW_URLS=[];
  docPriorRender();
  const context=recContext();
  if(context&&!$('.doc-attachments'))$('.page-footer')?.insertAdjacentHTML('beforebegin',docAttachments(context.type,context.id,docLocked(context.type,context.record)));
  if(!current.blocked&&current.name==='report-pack')$('.page-head .actions')?.insertAdjacentHTML('afterbegin',btn('Request approval','approval-new',`data-type="report" data-id="${current.params.get('id')}"`)+' '+btn('Save to documents','doc-save-report',`data-id="${current.params.get('id')}"`));
  if(current.name==='work')$('.page-head .actions')?.insertAdjacentHTML('afterbegin',link('Approval inbox','approvals'));
  if(current.name==='settings')$('.page-head .actions')?.insertAdjacentHTML('afterbegin',link('Data checks','data-health'));
  const profile=$('.topbar .profile');if(profile)profile.innerHTML=`<div class="avatar">${e(recActor().name.split(' ').map(n=>n[0]).join(''))}</div><div><strong>${e(recActor().name)}</strong><small>Demo user</small></div>`;
  const footer=$('.page-footer');if(footer)footer.innerHTML=`<span>Demo · synthetic data · ${date(S.asOf)}</span><span>${storageOK?'Metadata saved':'Metadata: session only'} · ${btn('About this demo','doc-help','','quiet small')}</span>`;
  docLoadPreviews().catch(err=>toast('Preview unavailable: '+err.message));
};
const docOldReset=actions.reset;
actions.reset=()=>{docOldReset();const callback=modalSave;modalSave=form=>{docAsync(async()=>{await DocumentBytes.clear();DOC_SELECTED.clear();DOC_FILTERS={search:'',folder:'',type:'',status:'',sort:'updated',archived:false};DOC_UPLOAD=null;DOC_APPROVAL_DRAFT=null;callback(form);closeDialog();})();return false;};};
window.CLEAR_SIGHT_DOCUMENTS=Object.freeze({version:'2.6-demo',getData:()=>clone(S.governance),query:()=>clone(docRows()),checks:()=>clone(recIntegrity()),getReceipts:()=>clone(S.governance.receipts),storageMode:()=>DocumentBytes.mode()});
// Submitted custom forms are alternative sources for the existing requirement
// evaluation, not a second compliance store. Designer previews never qualify.
function submissionDigest(b,res){return recHashText(recStable({bindingId:b?.id||'',formVersionId:b?.formVersionId||'',definition:res.definitionSnapshot,snapshot:res.snapshot,submittedAt:res.submittedAt,submittedBy:res.submittedBy}));}
function requirementFormSources(p,t,rid){if(!p.targets.includes(t))return [];return faInit().bindings.filter(b=>b.programId===p.id&&b.target===t).map(b=>{const res=faInit().responses[b.id],def=res?.definitionSnapshot;return {b,res,def,fields:def?.fields.filter(f=>(f.mapping||[]).includes(rid)&&faVisible(f,res.snapshot?.answers||{}))||[]};}).filter(x=>x.res?.status==='Submitted'&&x.res.snapshot&&x.fields.length);}
function requirementSourceURL(p,t,rid,b){return `requirement-review?id=${p.id}&target=${encodeURIComponent(t)}&requirement=${rid}&binding=${encodeURIComponent(b.id)}`;}
function submissionIssues(src){
  const {b,res,def,fields}=src,snap=res.snapshot,errors=faErrors({...def,fields},{...res,answers:snap.answers,observations:snap.observations});
  if(fields.every(f=>f.mappingCoverage==='supporting'))errors.push('These fields provide supporting evidence only; a complete-objective assessment is still needed.');
  if(!res.submissionHash||res.submissionHash!==submissionDigest(b,res))errors.push('The submission checksum is missing or changed.');
  for(const f of fields){if(days(S.asOf,res.submittedAt.slice(0,10))>(f.freshDays||90))errors.push(f.label+': answer is outside its freshness policy.');}
  for(const x of faChecks({...def,fields},snap.answers)){
    const o=snap.observations[x.key];
    if(!o||o.result!=='Pass')errors.push(x.label+': '+(o?.result||'Not collected'));
    if(x.check.assurance==='Heuristic only')errors.push(x.label+': a heuristic is not sufficient assurance.');
    if(o?.observedAt&&Date.now()-new Date(o.observedAt).getTime()>x.check.freshness*1000)errors.push(x.label+': source is stale.');
  }
  for(const id of snap.documentVersionIds||[]){const v=docVersion(id);if(!v||v.storage==='reference')errors.push('A linked document has no original file: '+id);}
  return [...new Set(errors)];
}
const submissionPriorReview=jRequirementReview;
jRequirementReview=function(){
  const p=prog(),t=selectedTarget(p),rid=current.params.get('requirement')||p.reqs[0],r=findR(rid),sources=requirementFormSources(p,t,rid),bid=current.params.get('binding');
  if(!bid){const original=submissionPriorReview();if(!sources.length)return original;const picker=panel('Submitted assessment sources','These forms refer to this requirement and the same target. Select the source to review.',table(['Submission','Version / respondent',''],sources.map(x=>({cells:[e(x.def.name),`${e(x.b.formVersionId)}<small>${e(x.res.submittedBy)} · ${recTimestamp(x.res.submittedAt)}</small>`,link('Review this source',requirementSourceURL(p,t,rid,x.b),'small')]}))));return original.replace('<div class="j-review-grid">',picker+'<div class="j-review-grid mt16">');}
  const src=sources.find(x=>x.b.id===bid),denied=jScopeDenied(p,'programs','programs');if(denied)return denied;
  if(!src||!r||!activeReqs(p).includes(rid))return header('Assessment source unavailable','Use a submitted form assigned to this program, target and requirement.',link('Back to program','program?id='+p.id));
  const {b,res,def,fields}=src,problems=submissionIssues(src),done=res.requirementReviews?.[rid],outcome=evaluation(p,t,rid),back=`program?id=${p.id}&target=${encodeURIComponent(t)}&tab=requirements`;
  return `<div class="crumb"><a href="#/${back}">${e(p.name)}</a> / ${e(r.code)}</div>`+header(r.title,`${r.code} · ${targetName(t)}`,link('Back to requirements',back),'Requirement review')+jFacts([['Current conclusion',pill(outcome.status)],['Source',e(def.name)],['Form version',e(b.formVersionId)],['Submitted by',e(res.submittedBy)]])+`<div class="j-review-grid"><div><section class="panel pad"><h2>Requirement</h2><p>${e(p.overrides[rid]||r.source)}</p><p class="field-help">The form links ${fields.length} field${fields.length===1?'':'s'} to this objective. The reviewer must assess whether that is sufficient coverage.</p>${link('Open complete submission','form-response?binding='+b.id,'small')}</section><section class="fa-canvas mt16">${fields.map((f,i)=>faRenderField(f,res.snapshot.answers,res.snapshot.observations,false,false,i,true)).join('')}</section>${docAttachments('formResponse',b.id,true)}</div><div><section class="panel pad"><h2>Record conclusion</h2>${problems.length?notice('<strong>Not ready for a satisfied conclusion.</strong><br>'+problems.map(e).join('<br>'),'mini mt16'):notice('Submitted values are available. Confirm objective coverage before recording a conclusion.','mini mt16')}${done?`<p class="mt16">This submitted source has already been reviewed for this requirement.</p>${pill(done.status)}`:`<form id="source-requirement-form" data-binding="${b.id}" data-requirement="${rid}"><div class="stack mt16">${select('reviewer','Reviewer',J_PEOPLE,p.reviewer)}${select('decision','Conclusion',['Satisfied','Partially satisfied','Not satisfied'],'Partially satisfied')}${area('note','Evaluation rationale','','required minlength="20"')}<label class="check-label"><input name="coverage" type="checkbox" value="confirmed">I reviewed the full requirement, the linked fields and supporting documents. They cover the complete objective.</label><p class="field-help">Required for Satisfied. Document approval alone does not establish compliance.</p></div><div id="source-review-error" class="form-error" role="alert"></div><button type="submit" class="btn primary mt16">Record conclusion</button></form>`}</section></div></div>`;
};
window.CS_JOURNEYS['requirement-review']=()=>jRequirementReview();
const submissionPriorResponseView=formResponseView;
formResponseView=function(){const html=submissionPriorResponseView(),b=faBinding(),p=b&&recRead('program',b.programId);if(!p)return html;const ids=activeReqs(p).filter(id=>faDefinition().fields.some(f=>(f.mapping||[]).includes(id)));if(!ids.length)return html;const res=faResponse(),out=ids.map(id=>`<div>${e(findR(id).code)} · ${pill(evaluation(p,b.target,id).status)}${res.status==='Submitted'?link('Review',requirementSourceURL(p,b.target,id,b),'quiet small'):''}</div>`).join('');return html.replace('<div>Requirement conclusion: <strong>Not evaluated here</strong></div>','<div><strong>Program conclusions</strong><p class="field-help">From the program’s requirement evaluations, not form completion.</p>'+out+'</div>');};

async function submissionDecide(form){
  const p=prog(),t=selectedTarget(p),rid=form.dataset.requirement,src=requirementFormSources(p,t,rid).find(x=>x.b.id===form.dataset.binding),v=values(form);
  requireValue(src&&activeReqs(p).includes(rid)&&scopeMatches(p,'programs'),'The source does not belong to this scoped requirement.');
  const {b,res}=src;requireValue(!res.requirementReviews?.[rid],'This source already has a recorded conclusion.');
  requireValue(recUserId(v.reviewer)&&v.reviewer!==res.submittedBy,'The respondent cannot review their own submission.');
  requireValue(res.submissionHash===submissionDigest(b,res),'The submitted source changed. It cannot be evaluated.');
  requireValue(['Satisfied','Partially satisfied','Not satisfied'].includes(v.decision),'Choose a valid conclusion.');requireValue(v.note.trim().length>=20,'Explain the evaluation in at least 20 characters.');
  if(v.decision==='Satisfied'){
    const issues=submissionIssues(src);requireValue(!issues.length,issues.join(' '));requireValue(v.coverage==='confirmed','Confirm full objective coverage before recording Satisfied.');
    for(const id of res.snapshot.documentVersionIds||[]){const dv=docVersion(id);requireValue(dv.uploadedBy!==recUserId(v.reviewer),'A file contributor cannot independently accept their own supporting document.');await docVerifiedBytes(dv);}
  }
  requireValue(!res.requirementReviews?.[rid]&&res.submissionHash===submissionDigest(b,res),'The review state changed; reload the source.');
  const result={status:v.decision,date:S.asOf,reviewer:v.reviewer,note:v.note.trim(),evidence:[],documentVersionIds:clone(res.snapshot.documentVersionIds||[]),source:{type:'formResponse',bindingId:b.id,formVersionId:b.formVersionId,submissionHash:res.submissionHash,fieldIds:src.fields.map(x=>x.id)},coverageConfirmed:v.coverage==='confirmed'};
  p.evaluationHistory??={};p.evaluationHistory[t]??={};p.evaluationHistory[t][rid]??=[];if(p.evaluations?.[t]?.[rid])p.evaluationHistory[t][rid].push(clone(p.evaluations[t][rid]));
  p.evaluations[t]??={};p.evaluations[t][rid]=result;res.requirementReviews??={};res.requirementReviews[rid]=clone(result);
  recDecision('program',p.id,v.reviewer,v.decision,v.note.trim(),{requirementId:rid,target:t,source:clone(res.snapshot),definition:clone(res.definitionSnapshot),evaluation:clone(result)},b.formVersionId);
  audit(`${findR(rid).code}: ${v.decision} from ${b.formVersionId}`,p.id,v.reviewer);save();render();toast('Requirement conclusion recorded.');
}
const submissionPriorWork=workItems;
workItems=function(){const rows=submissionPriorWork();for(const b of faInit().bindings){const p=recRead('program',b.programId),res=faInit().responses[b.id];if(!p||!p.targets.includes(b.target)||!scopeMatches(p,'programs')||res?.status!=='Submitted')continue;const ids=activeReqs(p).filter(rid=>!res.requirementReviews?.[rid]&&requirementFormSources(p,b.target,rid).some(s=>s.b.id===b.id));if(ids.length)rows.push({id:'FORM-REVIEW-'+b.id,ref:p.id,kind:'Assessment review',title:'Review '+res.definitionSnapshot.name+' · '+targetName(b.target),owner:p.reviewer,due:p.due,status:'Open',source:'form-review',route:requirementSourceURL(p,b.target,ids[0],b),orgScopeIds:recScope('program',p)});}return rows;};
document.addEventListener('submit',ev=>{if(ev.target.id==='source-requirement-form'){ev.preventDefault();submissionDecide(ev.target).catch(err=>{const el=$('#source-review-error');if(el)el.textContent=err.message;else toast(err.message);});}});

// Bootstrap follows the shared usability and copilot modules.
