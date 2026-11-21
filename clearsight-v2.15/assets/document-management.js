/* ClearSight V2.12 · document reading, ownership and lifecycle.
 * Canonical metadata/versions remain in S.governance; original bytes remain in
 * DocumentBytes. Browser policy demonstrations are not server authorization. */
'use strict';

const DM212_LIMITS=Object.freeze({file:20*1024*1024,batch:20*1024*1024,files:10,expanded:8*1024*1024,entries:2000});
const dm212BeforeVisible=docVisible;
const dm212BeforeRows=docRows;
const dm212BeforeLibrary=documentsView;
const dm212BeforeDocument=docView;
const dm212BeforeStartUpload=docStartUpload;
const dm212BeforeVerifiedBytes=docVerifiedBytes;
const dm212BeforeReviewQueue=ax211DocumentQueue;
ax211DocumentQueue=function(mode){return dm212BeforeReviewQueue(mode).filter(x=>docVisible(recRead('document',docVersion(x.documentVersionId)?.documentId)));};
const dm212Scope=d=>d?.scopeMode==='shared'?['']:d?.orgScopeId?[d.orgScopeId]:[];
const dm212Here=()=>current.name+(current.params.toString()?'?'+current.params.toString():'');
function dm212Decision(d,action='view'){
 if(!d||S.scopeInvalid||!dm212BeforeVisible(d))return {allowed:false,reason:'This document is unavailable in the selected location.'};
 const role=window.CLEAR_SIGHT_ROLE_EXPERIENCE;
 if(action==='view'&&role?.canViewSubject)return role.canViewSubject('document',d.id);
 const scopes=dm212Scope(d);if(!scopes.length)return {allowed:false,reason:'An accountable document location has not been recorded.'};
 if(role?.capability)return role.capability('documents.'+action,{scopeIds:scopes});
 const checks=scopes.map(scope=>gvDecision(recActor().id,'documents.'+action,scope));
 return {allowed:checks.every(x=>x.allowed),reason:checks.find(x=>!x.allowed)?.reason||'The active document permission covers this location.',checks};
}
function dm212Require(d,action='view'){const check=dm212Decision(d,action);requireValue(check.allowed,check.reason);return d;}
docVisible=function(d,scope=selectedScope()){return dm212BeforeVisible(d,scope)&&dm212Decision(d,'view').allowed;};
function dm212Fingerprint(d){return recHashText(recStable({document:d,version:docHead(d)}));}
function dm212Lifecycle(d){
 if(d.archived)return {state:'Archived',tone:'',due:d.validThrough||'',reason:'Retained version history and links.'};
 if(d.validThrough&&d.validThrough<S.asOf)return {state:'Expired',tone:'red',due:d.validThrough,reason:'Document validity has elapsed.'};
 if(d.reviewOn&&d.reviewOn<S.asOf)return {state:'Review overdue',tone:'red',due:d.reviewOn,reason:'The owner must review this document.'};
 const due=[d.validThrough,d.reviewOn].filter(Boolean).sort()[0];
 if(due&&due<=addDays(S.asOf,30))return {state:'Due within 30 days',tone:'amber',due,reason:'Review or replace before the recorded date.'};
 if(!d.validThrough&&!d.reviewOn)return {state:'Dates not set',tone:'',due:'',reason:'No document review or validity date has been recorded.'};
 return {state:'Current dates',tone:'green',due,reason:'Within the document management dates. Assessment-specific validity remains separate.'};
}
function dm212Filters(){
 const p=current.params;
 return {owner:p.get('docOwner')||'',classification:p.get('docClass')||'',lifecycle:p.get('docLife')||'',linked:p.get('docLinked')||''};
}
docRows=function(filter=DOC_FILTERS){
 let rows=dm212BeforeRows({...filter,search:''});if(filter.search){const q=filter.search.toLocaleLowerCase();rows=rows.filter(d=>[d.title,d.id,docHead(d)?.fileName,recUser(d.ownerId)?.name,docFolderName(d.folderId),...(d.tags||[])].join(' ').toLocaleLowerCase().includes(q));}if(current.name!=='documents')return rows;
 const f=dm212Filters();
 if(f.owner)rows=rows.filter(d=>d.ownerId===(f.owner==='me'?recActor().id:f.owner));
 if(f.classification)rows=rows.filter(d=>d.classification===f.classification);
 if(f.lifecycle)rows=rows.filter(d=>dm212Lifecycle(d).state===f.lifecycle);
 if(f.linked)rows=rows.filter(d=>(S.governance.links.some(l=>docVersion(l.documentVersionId)?.documentId===d.id))===(f.linked==='yes'));
 return rows;
};
function dm212FilterBar(){
 const f=dm212Filters(),visible=S.governance.documents.filter(d=>docVisible(d)&&!d.archived),owners=[...new Set(visible.map(d=>d.ownerId))];
 return `<section class="panel dm212-filter-panel" aria-label="Document responsibility and lifecycle filters"><div class="dm212-filters">${select('docOwner','Owner',[['','All owners'],['me','Owned by me'],...owners.map(id=>[id,recUser(id)?.name||id])],f.owner,'data-dm212-filter="docOwner"')}${select('docClass','Classification',[['','All classifications'],'Internal','Confidential','Restricted'],f.classification,'data-dm212-filter="docClass"')}${select('docLife','Document dates',[['','All dates'],'Expired','Review overdue','Due within 30 days','Dates not set','Current dates'],f.lifecycle,'data-dm212-filter="docLife"')}${select('docLinked','Linked records',[['','All documents'],['yes','Linked to a record'],['no','Not yet linked']],f.linked,'data-dm212-filter="docLinked"')}</div><div class="dm212-filter-caption"><span>${e(selectedScope()?orgName(selectedScope()):'All permitted locations')} · file approval and assessment review remain separate decisions</span>${btn('Clear document filters','dm212-clear-filters','','quiet small')}</div></section>`;
}
function dm212RestoreFilters(){
 const p=current.params;if(current.name!=='documents')return;
 for(const [key,param]of Object.entries({search:'docSearch',folder:'docFolder',type:'docType',status:'docStatus',sort:'docSort'}))DOC_FILTERS[key]=p.get(param)|| (key==='sort'?'updated':'');
 DOC_FILTERS.archived=p.get('docArchived')==='1';
}
function dm212RememberFilters(){
 if(current.name!=='documents')return;
 const patch={docSearch:DOC_FILTERS.search,docFolder:DOC_FILTERS.folder,docType:DOC_FILTERS.type,docStatus:DOC_FILTERS.status,docSort:DOC_FILTERS.sort==='updated'?'':DOC_FILTERS.sort,docArchived:DOC_FILTERS.archived?'1':''};
 const route=ui211Route(dm212Here(),patch);if(window.history?.replaceState){window.history.replaceState(window.history.state,'','#/'+route);current.params=new URLSearchParams(route.split('?')[1]||'');}
}
documentsView=function(){dm212RestoreFilters();const view=current.params.get('view')||'library';if(view!=='library'){
 const all=S.evidence.filter(x=>recInScope('evidence',x)&&docVisible(recRead('document',docVersion(x.documentVersionId)?.documentId))),active=all.filter(x=>x.status!=='Superseded'),pending=active.filter(x=>x.status==='Submitted'),refresh=active.filter(x=>x.status==='Changes requested'||evQuality(x)!=='Current'),mode=['mine','prepared'].includes(current.params.get('queue'))?current.params.get('queue'):'all',rows=view==='pending'?ax211DocumentQueue(mode):view==='refresh'?refresh:all,tabs=view==='pending'?`<nav class="ax211-queue-tabs" aria-label="Document review responsibility">${[['all','All pending'],['mine','My review decisions'],['prepared','Submitted by me']].map(([key,label])=>link(e(label)+' <span>'+ax211DocumentQueue(key).length+'</span>',ui211Route(dm212Here(),{queue:key==='all'?null:key}),key===mode?'active small':'quiet small')).join('')}</nav>`:'';
 return header('Documents','Review an exact file version in its assessment context.',btn('Upload documents','doc-upload','','small'))+d210Identity()+d210Navigation(view)+metricRow([metric('Unique documents',new Set(all.map(x=>docVersion(x.documentVersionId)?.documentId)).size,'Readable documents in this scope','file'),metric('Awaiting review',pending.length,'Assessment review decisions','work'),metric('Need attention',refresh.length,'Refresh or correction required','clock')])+panel('Document review queue',rows.length+' permitted review records · source files and contextual decisions are separate.',tabs+d210ReviewRows(rows));
 }let html=dm212BeforeLibrary();const marker='<div class="doc-explorer">';html=html.replace(marker,dm212FilterBar()+marker);return dm212ControlHTML(html);};
const dm212BeforeTable=docTable;
docTable=function(rows,config={}){
 let html=dm212BeforeTable(rows,config);
 for(const d of rows){const v=docHead(d),life=dm212Lifecycle(d),route=ui211Route('document?id='+encodeURIComponent(d.id)+'&version='+encodeURIComponent(v?.id||''),{returnTo:current.name==='documents'?dm212Here():null});
  html=html.replace('class="doc-name" href="#/document?id='+d.id+'"','class="doc-name" href="#/'+e(route)+'"');
  html=html.replace(link('Open','document?id='+d.id,'quiet small'),`<div class="dm212-row-actions">${link('Preview',route,'small')}${link('Details',route+'&tab=details','quiet small')}</div>`);
  html=html.replace(docStatePill(docStatus(d))+'</td>',docStatePill(docStatus(d))+`<small class="dm212-life">${e(life.state)}${life.due?' · '+date(life.due):''}</small></td>`);
 }
 return html;
};
function dm212ControlHTML(html,d=null){
 const mapping={'doc-download':'export','doc-open-pdf':'view','doc-export':'export','doc-replace':'edit','doc-move':'edit','doc-archive':'edit','dm212-edit':'edit','doc-bulk-move':'edit','doc-bulk-archive':'edit','doc-new-folder':'manage'};
 return html.replace(/<button\b[^>]*data-action="([^"]+)"[^>]*>[\s\S]*?<\/button>/g,(whole,action)=>{
  if(!mapping[action])return whole;
  let decision;if(d)decision=dm212Decision(d,mapping[action]);else if(action==='doc-export'){const rows=docRows();decision={allowed:rows.length>0&&rows.every(x=>dm212Decision(x,'export').allowed),reason:'Export requires permission for every displayed document.'};}
  else if(action==='doc-new-folder')decision=gvDecision(recActor().id,'documents.manage','');
  else {const rows=[...DOC_SELECTED].map(id=>recRead('document',id)).filter(Boolean);decision={allowed:rows.length>0&&rows.every(x=>dm212Decision(x,mapping[action]).allowed),reason:'Select documents you can manage.'};}
  if(decision.allowed)return whole;
  return whole.replace(/<button\b/,'<button disabled aria-disabled="true" title="'+e(decision.reason)+'"');
 });
}
function dm212Details(d,v){const lifecycle=dm212Lifecycle(d),edit=dm212Decision(d,'edit');return `<div class="dm212-detail-grid"><section class="panel pad"><div class="row between wrap"><div><h2>Ownership and review dates</h2><p>Keep the document useful and assign the next review.</p></div>${edit.allowed?btn('Edit details','dm212-edit',`data-id="${e(d.id)}"`,'small'):pill('Read only')}</div><dl class="doc-facts"><div><dt>Title</dt><dd>${e(d.title)}</dd></div><div><dt>Owner</dt><dd>${e(recUser(d.ownerId)?.name||'Unassigned')}</dd></div><div><dt>Location</dt><dd>${e(docScopeLabel(d))}</dd></div><div><dt>Classification</dt><dd>${e(d.classification)}</dd></div><div><dt>Folder</dt><dd>${e(docFolderName(d.folderId))}</dd></div><div><dt>Review by</dt><dd>${d.reviewOn?date(d.reviewOn):'Not set'}</dd></div><div><dt>Valid through</dt><dd>${d.validThrough?date(d.validThrough):'Not set'}</dd></div><div><dt>Tags</dt><dd>${e((d.tags||[]).join(', ')||'None')}</dd></div></dl>${pill(lifecycle.state,lifecycle.tone)}<p class="muted">${e(lifecycle.reason)}</p><small>These document management dates do not change a previous file approval or an assessment's observation and validity dates.</small></section><section class="panel pad"><h2>Exact file and provenance</h2><dl class="doc-facts"><div><dt>Version</dt><dd>${docVersionLabel(v)}${v.id===d.currentVersionId?' · current':' · historical'}</dd></div><div><dt>Original filename</dt><dd>${e(v.fileName)}</dd></div><div><dt>Added by</dt><dd>${e(recUser(v.uploadedBy)?.name||'Not recorded')}</dd></div><div><dt>Added</dt><dd>${recTimestamp(v.createdAt)}</dd></div><div><dt>Size</dt><dd>${recBytes(v.size)}</dd></div><div><dt>Change note</dt><dd>${e(v.note||'Not recorded')}</dd></div>${v.externalSource?`<div><dt>Supplied by</dt><dd>${e(v.externalSource.respondentName)}<small>External response · ${e(v.externalSource.requestId)}<br>Captured by ${e(recUser(v.uploadedBy)?.name||'Not recorded')}</small></dd></div>`:''}</dl><details class="doc-checksum"><summary>Verify exact file</summary><code>${e(v.sha256||'No checksum recorded')}</code></details><p class="muted">Original bytes are checked before preview, download and acceptance.</p><div class="row wrap">${btn('Move to folder','doc-move',`data-id="${e(d.id)}"`,'small')}${btn(d.archived?'Restore':'Archive','doc-archive',`data-id="${e(d.id)}"`,'quiet small')}</div></section></div>`;}
function dm212VersionSelect(d,v,compare=false){const versions=S.governance.versions.filter(x=>x.documentId===d.id).sort((a,b)=>b.number-a.number);return `<div class="dm212-version-bar"><label>Viewing version<select data-dm212-version="${e(d.id)}">${options(versions.map(x=>[x.id,'v'+x.number+' · '+date(x.createdAt)+(x.id===d.currentVersionId?' · current':'')]),v.id)}</select></label><span>${e(v.note||'No change note')}</span>${versions.length>1?link('Compare versions','document?id='+d.id+'&version='+v.id+'&tab=compare&compare='+encodeURIComponent(versions.find(x=>x.id!==v.id).id),'small'):''}</div>`;}
function dm212CompareView(d,v){const other=docVersion(current.params.get('compare')),versions=S.governance.versions.filter(x=>x.documentId===d.id&&x.id!==v.id);if(!other||other.documentId!==d.id||other.id===v.id)return empty('Choose another version','Compare two retained versions of this document.',versions.length?link('Compare with v'+versions[0].number,'document?id='+d.id+'&version='+v.id+'&tab=compare&compare='+versions[0].id,'primary'):'');return panel('Compare retained versions','Metadata differences and readable text. Original files and all decisions remain unchanged.',`<div class="dm212-compare-picker">${select('compare','Compare against',versions.map(x=>[x.id,'v'+x.number+' · '+x.fileName]),other.id,'data-dm212-compare="'+e(d.id)+'"')}</div>`+table(['Detail','v'+other.number,'v'+v.number],[['File',other.fileName,v.fileName],['Added by',recUser(other.uploadedBy)?.name||'Not recorded',recUser(v.uploadedBy)?.name||'Not recorded'],['Added',other.createdAt,v.createdAt],['Size',recBytes(other.size),recBytes(v.size)],['Change note',other.note,v.note],['Checksum',other.sha256,v.sha256]].map(([name,a,b])=>({cells:[e(name),e(a||'—'),`<span class="${a!==b?'dm212-changed':''}">${e(b||'—')}</span>`]})))+`<div data-dm212-comparison="${e(other.id)}" data-current="${e(v.id)}" class="dm212-comparison"><p class="pad muted">Loading readable text for comparison…</p></div>`);}
docView=function(){
 const d=recRead('document',current.params.get('id'));if(!d||!docVisible(d))return header('Document unavailable','The file is missing, outside this location, or unavailable to your current role.',link('Documents','documents'));
 const v=docVersion(current.params.get('version')||d.currentVersionId);if(!v||v.documentId!==d.id)return dm212BeforeDocument();
 const tab=current.params.get('tab')||'overview';let html;
 if(['details','compare'].includes(tab)){const old=current.params.get('tab');current.params.set('tab','versions');html=dm212BeforeDocument();current.params.set('tab',old);html=ax211ReplacePanel(html,'Version history',tab==='details'?dm212Details(d,v):dm212CompareView(d,v));}
 else html=dm212BeforeDocument();
 const url='document?id='+d.id+'&version='+v.id;
 html=html.replace('</nav>',link('Details',url+'&tab=details',tab==='details'?'active':'')+ (S.governance.versions.filter(x=>x.documentId===d.id).length>1?link('Compare',url+'&tab=compare'+(current.params.get('compare')?'&compare='+encodeURIComponent(current.params.get('compare')):''),tab==='compare'?'active':''):'')+'</nav>');
 html=html.replace('<div class="doc-record-status">',dm212VersionSelect(d,v)+'<div class="doc-record-status">');
 if(tab==='overview')html=html.replace('<h2>Preview</h2>','<h2>Document preview</h2>').replace('<div class="doc-preview"',`<div class="dm212-preview-tools">${btn('Expand preview','dm212-expand-preview','','quiet small')}<span>Exact version ${e(docVersionLabel(v))}</span></div><div class="doc-preview dm212-preview"`);
 if(tab==='overview')html=html.replace('<h2>Document details</h2>','<div class="row between wrap"><h2>Document details</h2>'+link('Manage details',url+'&tab=details','quiet small')+'</div>');
 const back=current.params.get('returnTo');if(back&&/^documents(?:\?|$)/.test(back)&&back.length<=2500){html=html.replace('<a href="#/documents">Documents</a>','<a href="#/'+e(back)+'">Back to results</a>');html=html.replace(/href="#\/(document\?[^"]*)"/g,(whole,value)=>'href="#/'+e(ui211Route(value.replace(/&amp;/g,'&'),{returnTo:back}))+'"');}
 return dm212ControlHTML(html,d);
};
window.CS_JOURNEYS.documents=()=>documentsView();window.CS_JOURNEYS.document=()=>docView();

/* Read permission, actor, source and organization scope are checked again after
 * asynchronous byte verification. A metadata reference never becomes content. */
async function dm212ReadVersion(versionId,{purpose='view'}={}){
 requireValue(['view','export'].includes(purpose),'Unknown document read purpose.');
 const v=docVersion(versionId),d=v&&recRead('document',v.documentId);requireValue(v&&d,'The document version is unavailable.');dm212Require(d,purpose);
 requireValue(v.storage!=='reference','Original file not supplied. Upload a verified source version.');
 const actor=recActor().id,scope=selectedScope(),basis=recStable({v,d});const blob=await dm212BeforeVerifiedBytes(v);
 requireValue(actor===recActor().id&&scope===selectedScope()&&basis===recStable({v:docVersion(versionId),d:recRead('document',d.id)}),'The actor, location or document changed during file verification. Reopen the document.');dm212Require(d,purpose);return blob;
}
docVerifiedBytes=async function(v){return dm212ReadVersion(v?.id,{purpose:'view'});};

/* Small bounded Office text/table previews. No external conversion, formula
 * execution, embedded objects or document HTML is accepted. */
function dm212XML(value){const s=String(value);requireValue(!/<!DOCTYPE|<!ENTITY/i.test(s),'Document XML declarations are unsupported.');return s;}
function dm212Decode(s){return String(s).replace(/&(amp|lt|gt|quot|apos|#x[0-9a-f]+|#\d+);/gi,(m,x)=>{const known={amp:'&',lt:'<',gt:'>',quot:'"',apos:"'"};if(known[x])return known[x];const n=x.toLowerCase().startsWith('#x')?parseInt(x.slice(2),16):Number(x.slice(1));return n>0&&n<=0x10ffff&&!(n>=0xd800&&n<=0xdfff)?String.fromCodePoint(n):'�';});}
function dm212Attr(s,key){const m=s.match(new RegExp('(?:^|\\s)'+key+'\\s*=\\s*([\"\x27])([\\s\\S]*?)\\1'));return m?dm212Decode(m[2]):'';}
function dm212TextNodes(xml){dm212XML(xml);return [...xml.matchAll(/<(?:[\w.-]+:)?t\b[^>]*>([\s\S]*?)<\/(?:[\w.-]+:)?t\s*>/g)].map(x=>dm212Decode(x[1].replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g,'$1'))).join('');}
function dm212WordPreview(xml){
 dm212XML(xml);const paragraphs=[...xml.matchAll(/<(?:[\w.-]+:)?p\b[^>]*>([\s\S]*?)<\/(?:[\w.-]+:)?p\s*>/g)],limit=2500;
 const parts=paragraphs.slice(0,limit).map(x=>dm212TextNodes(x[1].replace(/<(?:[\w.-]+:)?(?:tab|br)\b[^>]*\/?\s*>/g,'<t> </t>')));
 return {text:parts.join('\n'),shown:parts.length,total:paragraphs.length,truncated:paragraphs.length>limit};
}
function dm212WordText(xml){return dm212WordPreview(xml).text;}
function dm212Zip(bytes){
 requireValue(bytes.length<=DM212_LIMITS.file&&bytes.length>=22,'Unsupported or oversized Office package.');const dv=new DataView(bytes.buffer,bytes.byteOffset,bytes.byteLength);let end=-1;
 for(let i=bytes.length-22;i>=Math.max(0,bytes.length-65557);i--)if(dv.getUint32(i,true)===0x06054b50&&i+22+dv.getUint16(i+20,true)===bytes.length){end=i;break;}
 requireValue(end>=0,'The Office package directory is unavailable.');const count=dv.getUint16(end+10,true),size=dv.getUint32(end+12,true),start=dv.getUint32(end+16,true);
 requireValue(dv.getUint16(end+4,true)===0&&dv.getUint16(end+6,true)===0&&count<=DM212_LIMITS.entries&&start+size<=end,'Split, ZIP64 or oversized Office packages are unsupported.');const entries=new Map();let pos=start,total=0;
 for(let n=0;n<count;n++){
  requireValue(pos+46<=end&&dv.getUint32(pos,true)===0x02014b50,'Invalid Office package directory.');const flags=dv.getUint16(pos+8,true),method=dv.getUint16(pos+10,true),compressed=dv.getUint32(pos+20,true),expanded=dv.getUint32(pos+24,true),nl=dv.getUint16(pos+28,true),extra=dv.getUint16(pos+30,true),comment=dv.getUint16(pos+32,true),offset=dv.getUint32(pos+42,true);requireValue(pos+46+nl+extra+comment<=end,'Invalid Office package entry.');
  const name=new TextDecoder('utf-8',{fatal:true}).decode(bytes.slice(pos+46,pos+46+nl));requireValue(!entries.has(name)&&!name.startsWith('/')&&!name.split('/').includes('..')&&!name.includes('\\'),'Unsafe or repeated Office package path.');total+=expanded;requireValue(total<=40*1024*1024&&expanded<=DM212_LIMITS.expanded&&!(flags&1)&&[0,8].includes(method),'Encrypted or oversized Office contents cannot be previewed.');
  entries.set(name,{method,compressed,expanded,offset});pos+=46+nl+extra+comment;
 }
 async function read(name){const x=entries.get(name);if(!x)return null;const p=x.offset;requireValue(p+30<=bytes.length&&dv.getUint32(p,true)===0x04034b50,'Invalid Office file entry.');const begin=p+30+dv.getUint16(p+26,true)+dv.getUint16(p+28,true);requireValue(begin+x.compressed<=start,'Invalid Office file boundary.');const raw=bytes.slice(begin,begin+x.compressed);let out;
  if(x.method===0)out=raw;
  else {requireValue(typeof DecompressionStream==='function','This browser cannot read compressed Office previews. Download the original if permitted.');const reader=new Blob([raw]).stream().pipeThrough(new DecompressionStream('deflate-raw')).getReader(),chunks=[];let length=0;try{while(true){const part=await reader.read();if(part.done)break;length+=part.value.length;requireValue(length<=x.expanded&&length<=DM212_LIMITS.expanded,'Expanded Office contents exceed the preview limit.');chunks.push(part.value);}}finally{reader.cancel().catch(()=>{});}out=new Uint8Array(length);let at=0;for(const c of chunks){out.set(c,at);at+=c.length;}}
  requireValue(out.length===x.expanded,'Office package size check failed.');return dm212XML(new TextDecoder('utf-8',{fatal:true}).decode(out));
 }
 return {entries,read};
}
function dm212Sheet(xml,strings=[]){
 dm212XML(xml);const rows=[];let omitted=false;
 for(const row of xml.matchAll(/<(?:[\w.-]+:)?row\b[^>]*>([\s\S]*?)<\/(?:[\w.-]+:)?row\s*>/g)){
  if(rows.length>=100){omitted=true;break;}const cells=Array(16).fill('');
  for(const cell of row[1].matchAll(/<(?:[\w.-]+:)?c\b([^>]*)>([\s\S]*?)<\/(?:[\w.-]+:)?c\s*>/g)){
   const ref=dm212Attr(cell[1],'r'),col=ref.match(/^[A-Z]+/)?.[0]||'';let index=0;for(const c of col)index=index*26+c.charCodeAt(0)-64;index--;
   if(index<0||index>=16){omitted=true;continue;}const type=dm212Attr(cell[1],'t'),value=cell[2].match(/<(?:[\w.-]+:)?v\b[^>]*>([\s\S]*?)<\/(?:[\w.-]+:)?v\s*>/)?.[1]||'',formula=/<(?:[\w.-]+:)?f\b/.test(cell[2]);
   cells[index]=type==='s'?(strings[Number(value)]??'[Shared value unavailable]'):type==='inlineStr'?dm212TextNodes(cell[2]):dm212Decode(value);
   if(formula)cells[index]=(cells[index]||'[No cached value]')+' [cached formula result]';
  }
  rows.push(cells);
 }
 return {rows,omitted};
}
async function dm212Readable(v,blob){
 const bytes=new Uint8Array(await blob.arrayBuffer());
 if(v.mime?.startsWith('text/'))return {kind:v.mime==='text/csv'?'csv':'text',text:new TextDecoder().decode(bytes.slice(0,300000)),truncated:bytes.length>300000};
 const ext=v.fileName?.split('.').at(-1).toLowerCase();if(!['docx','xlsx'].includes(ext))return null;
 const zip=dm212Zip(bytes);
 if(ext==='docx'){const xml=await zip.read('word/document.xml');requireValue(xml,'The Word document body is unavailable.');const preview=dm212WordPreview(xml);return {kind:'office-text',text:preview.text,truncated:preview.truncated,note:`Text preview · ${preview.shown} of ${preview.total} paragraphs${preview.truncated?' · limited to the first 2,500 paragraphs; later paragraphs are not displayed or compared':''}. Layout, embedded objects, images, tracked changes and annotations are not reproduced.`};}
 const ss=await zip.read('xl/sharedStrings.xml'),strings=ss?[...ss.matchAll(/<(?:[\w.-]+:)?si\b[^>]*>([\s\S]*?)<\/(?:[\w.-]+:)?si\s*>/g)].map(x=>dm212TextNodes(x[1])):[],workbook=await zip.read('xl/workbook.xml'),rels=await zip.read('xl/_rels/workbook.xml.rels');requireValue(workbook&&rels,'The workbook sheet references are unavailable.');
 const relationships=new Map([...rels.matchAll(/<(?:[\w.-]+:)?Relationship\b([^>]*)\/?\s*>/g)].filter(m=>dm212Attr(m[1],'TargetMode')!=='External').map(m=>[dm212Attr(m[1],'Id'),dm212Attr(m[1],'Target')])),sheets=[];
 for(const m of [...workbook.matchAll(/<(?:[\w.-]+:)?sheet\b([^>]*)\/?\s*>/g)].slice(0,4)){
  const name=dm212Attr(m[1],'name'),target=relationships.get(dm212Attr(m[1],'r:id'));if(!target)continue;
  const path=target.startsWith('/xl/')?target.slice(1):'xl/'+target;requireValue(!path.split('/').includes('..')&&!path.includes('\\')&&!/^[a-z]+:/i.test(target),'Unsupported workbook sheet path.');const xml=await zip.read(path);if(xml)sheets.push({name,...dm212Sheet(xml,strings)});
 }
 requireValue(sheets.length,'No readable worksheet was found.');return {kind:'sheets',sheets,text:sheets.map(s=>s.name+'\n'+s.rows.map(r=>r.join('\t')).join('\n')).join('\n\n'),note:'Up to 4 sheets, 100 rows and 16 columns each. Cached values only; formulas are never evaluated. Number formats, charts and embedded objects are not reproduced.'};
}
async function dm212PreviewElement(el){
 const v=docVersion(el.dataset.docPreview);if(!v)return;const d=recRead('document',v.documentId),actor=recActor().id,scope=selectedScope(),basis=recStable({v,d});
 try{
  const blob=await dm212ReadVersion(v.id),model=await dm212Readable(v,blob);
  if(el.isConnected===false)return;requireValue(actor===recActor().id&&scope===selectedScope()&&basis===recStable({v:docVersion(v.id),d:recRead('document',d.id)}),'The reading context or file changed. Reopen the document.');dm212Require(d);
  if(model?.kind==='sheets')el.innerHTML=`<div class="doc-preview-caption">${e(model.note)}</div>`+model.sheets.map(s=>`<section class="dm212-sheet"><h3>${e(s.name)}</h3>${table(Array.from({length:16},(_,i)=>String.fromCharCode(65+i)),s.rows.map(r=>({cells:r.map(e)})))}${s.omitted?'<p class="small pad">Additional cells are outside this preview.</p>':''}</section>`).join('');
  else if(model?.kind==='csv'){const rows=docParseCSV(model.text);el.innerHTML=`<div class="doc-preview-caption">First 100 data rows and 16 columns · values displayed as text${model.truncated?' · file excerpt':''}</div>`+(rows.length?table(rows[0].slice(0,16).map(e),rows.slice(1,101).map(r=>({cells:rows[0].slice(0,16).map((_,i)=>e(r[i]||''))}))):empty('No rows','No readable CSV rows.'));}
  else if(model)el.innerHTML=(model.note?`<div class="doc-preview-caption">${e(model.note)}</div>`:'')+`<pre class="doc-text-preview">${e(model.text)}${model.truncated?'\n\n[Preview excerpt ends]':''}</pre>`;
  else if(v.mime?.startsWith('image/')){const url=URL.createObjectURL(blob);DOC_PREVIEW_URLS.push(url);el.innerHTML=`<img class="doc-image-preview" src="${e(url)}" alt="${e(v.fileName)} · ${e(docVersionLabel(v))}">`;}
  else if(v.mime==='application/pdf'){const url=URL.createObjectURL(blob);DOC_PREVIEW_URLS.push(url);el.innerHTML=`<iframe class="dm212-pdf" src="${e(url)}" title="${e(v.fileName)} · ${e(docVersionLabel(v))}" sandbox="" referrerpolicy="no-referrer"></iframe><p class="doc-preview-caption">If the embedded PDF viewer is unavailable, ${dm212Decision(d,'export').allowed?btn('download the original','doc-download',`data-id="${e(v.id)}"`,'quiet small'):'ask an authorized document owner for a readable copy'}.</p>`;}
  else el.innerHTML=empty('Preview unavailable','This file format has no local preview.',dm212Decision(d,'export').allowed?btn('Download original','doc-download',`data-id="${e(v.id)}"`,'small'):'');
  for(const wrapper of el.querySelectorAll('.table-wrap')){wrapper.setAttribute('tabindex','0');wrapper.setAttribute('aria-label','Scrollable document table preview');}
 }catch(error){if(el.isConnected!==false)el.innerHTML=empty('Preview unavailable',e(error.message),v?.storage!=='reference'&&d&&dm212Decision(d,'export').allowed?btn('Download original','doc-download',`data-id="${e(v.id)}"`,'small'):'');}
}
function dm212Diff(a,b){
 // Compare all text the bounded reader supplied; limit only the rendered excerpt.
 const old=String(a).split('\n'),next=String(b).split('\n');let prefix=0;
 while(prefix<old.length&&prefix<next.length&&old[prefix]===next[prefix])prefix++;
 let suffix=0;while(suffix<old.length-prefix&&suffix<next.length-prefix&&old[old.length-1-suffix]===next[next.length-1-suffix])suffix++;
 const before=old.slice(prefix,old.length-suffix).join('\n'),after=next.slice(prefix,next.length-suffix).join('\n');
 let common=0;while(common<before.length&&common<after.length&&before[common]===after[common])common++;
 const leadingCharacters=Math.max(0,common-160),excerpt=value=>value.slice(leadingCharacters,leadingCharacters+30000).split('\n').slice(0,500).join('\n'),oldText=excerpt(before),newText=excerpt(after);
 return {same:String(a)===String(b),prefix,suffix,before:oldText,after:newText,leadingCharacters,truncated:oldText.length+leadingCharacters<before.length||newText.length+leadingCharacters<after.length};
}
async function dm212LoadComparison(el){
 try{
  const a=docVersion(el.dataset.dm212Comparison),b=docVersion(el.dataset.current);requireValue(a&&b&&a.documentId===b.documentId,'Choose versions of one document.');
  const actor=recActor().id,scope=selectedScope(),basis=recStable([a,b]),blobs=await Promise.all([dm212ReadVersion(a.id),dm212ReadVersion(b.id)]),read=await Promise.all([dm212Readable(a,blobs[0]),dm212Readable(b,blobs[1])]);
  if(el.isConnected===false)return;requireValue(actor===recActor().id&&scope===selectedScope()&&basis===recStable([docVersion(a.id),docVersion(b.id)]),'The reading context or selected version changed. Reopen the comparison.');dm212Require(recRead('document',a.documentId));
  if(!read.every(Boolean)){el.innerHTML=notice('This format has no text comparison. Open the exact original versions side by side to review visual changes.','mini');return;}
  const diff=dm212Diff(read[0].text,read[1].text),limits=[...new Set(read.map(x=>x.note).filter(Boolean))];
  if(read.some(x=>x.truncated&&['text','csv'].includes(x.kind)))limits.push('Only the first 300,000 bytes of each text file are available to this preview. Changes outside those excerpts are not compared.');
  const summary=diff.same?'The readable preview text matches. Original checksums and visual content may still differ.':`Changed text excerpt · ${diff.prefix} leading and ${diff.suffix} trailing unchanged lines omitted${diff.leadingCharacters?' · '+diff.leadingCharacters+' unchanged characters omitted within the first changed line':''}${diff.truncated?' · displayed excerpts are limited to 500 lines and 30,000 characters per version':''}.`;
  el.innerHTML=notice(summary,'mini')+limits.map(note=>`<p class="pad small muted">${e(note)}</p>`).join('')+`<div class="dm212-compare-text"><section><h3>v${a.number} · comparison selection</h3><pre tabindex="0" aria-label="Comparison version text">${e(diff.same?read[0].text.slice(0,30000):diff.before||'[No text in this version’s changed region]')}</pre></section><section><h3>v${b.number} · viewing</h3><pre tabindex="0" aria-label="Viewed version text">${e(diff.same?'No text changes in this preview':diff.after||'[No text in this version’s changed region]')}</pre></section></div>`;
 }catch(error){if(el.isConnected!==false)el.innerHTML=notice(e(error.message),'warning');}
}
docLoadPreviews=async function(){await Promise.all([...$$('[data-doc-preview]')].map(dm212PreviewElement));await Promise.all([...$$('[data-dm212-comparison]')].map(dm212LoadComparison));};

function dm212Metadata(id,data,{expected,actorId}={}){
 const d=recRead('document',id);dm212Require(d,'edit');if(actorId)requireValue(actorId===recActor().id,'The acting user changed. Reopen document details.');if(expected)requireValue(expected===dm212Fingerprint(d),'The document changed. Reopen its details before saving.');
 const title=String(data.title||'').trim(),reason=String(data.reason||'').trim(),tags=String(data.tags||'').split(',').map(x=>x.trim()).filter(Boolean);
 requireValue(title.length>=3&&title.length<=180,'Use a title of 3–180 characters.');requireValue(recUser(data.ownerId),'Choose a named document owner.');requireValue(S.governance.folders.some(f=>f.id===data.folderId),'Choose an existing folder.');requireValue(['Internal','Confidential','Restricted'].includes(data.classification),'Choose a valid classification.');requireValue(tags.length<=10&&tags.every(x=>x.length<=32),'Use at most 10 tags of 32 characters each.');requireValue(reason.length>=10&&reason.length<=2000,'Describe the reason for this change in 10–2,000 characters.');
 for(const key of ['reviewOn','validThrough'])if(data[key])validDate(data[key],key==='reviewOn'?'Review by':'Valid through',false);
 requireValue(!data.reviewOn||!data.validThrough||data.reviewOn<=data.validThrough,'Schedule the document review on or before its validity ends.');
 const before=clone(d);Object.assign(d,{title,ownerId:data.ownerId,folderId:data.folderId,classification:data.classification,reviewOn:data.reviewOn||'',validThrough:data.validThrough||'',tags:[...new Set(tags)],metadataUpdatedAt:new Date().toISOString(),metadataUpdatedBy:recActor().id});
 recEvent('metadata',{type:'document',id},reason,recActor().id,{before,after:clone(d)});save();return d;
}
function dm212Edit(id){const d=recRead('document',id);dm212Require(d,'edit');const expected=dm212Fingerprint(d),actorId=recActor().id;openDialog('Edit document details',d.title,`<div class="field-grid">${input('title','Document title',d.title,'text','required minlength="3" maxlength="180"')}${select('ownerId','Accountable owner',docUserOptions(),d.ownerId)}${select('folderId','Folder',S.governance.folders.map(f=>[f.id,f.name]),d.folderId)}${select('classification','Classification',['Internal','Confidential','Restricted'],d.classification)}${input('reviewOn','Review by',d.reviewOn||'','date')}${input('validThrough','Valid through',d.validThrough||'','date')}${input('tags','Tags (comma separated)',(d.tags||[]).join(', '),'text','maxlength="330"')}${area('reason','Reason for changes','','required minlength="10" maxlength="2000"')}</div><p class="field-help">The accountable location and exact file versions stay fixed. Classification labels describe handling; access follows the active role permissions.</p>`,'Save details',form=>{dm212Metadata(id,values(form),{expected,actorId});render();toast('Document owner, details and review dates saved.');});}
function dm212CheckUpload(u){
 requireValue(u&&u.actorId===recActor().id&&!S.scopeInvalid,'The upload actor or location is unavailable.');
 const existing=u.versionFor&&recRead('document',u.versionFor),scope=u.meta.scope||'';requireValue(!scope||orgNode(scope),'Choose a valid document location.');
 if(existing){dm212Require(existing,'edit');requireValue(existing.orgScopeId===u.meta.scope&&existing.scopeMode===u.meta.scopeMode,'A new version must retain the existing document location.');}
 else {const decision=gvDecision(recActor().id,'documents.edit',scope);requireValue(decision.allowed,decision.reason);requireValue(!selectedScope()||scope&&orgWithin(scope,selectedScope()),'Choose a file location within the current scope.');}
 const basis=recStable({meta:u.meta,subject:u.subject,field:u.field,evidence:u.evidence,baseVersionId:u.baseVersionId,actorId:u.actorId,scope:selectedScope()});
 if(u.busy)requireValue(u.dm212SavingBasis===basis,'The upload details or selected location changed during saving. Review the batch again.');else u.dm212SavingBasis=basis;
 for(const row of u.rows.filter(x=>x.state==='Ready'))requireValue(recSHA256(row.bytes)===row.sha256,'A staged file changed. Remove it and choose the original again.');
 return true;
}
docStartUpload=function(config={}){
 const existing=config.versionFor&&recRead('document',config.versionFor),scopes=config.subject?recScope(config.subject.type,recRead(config.subject.type,config.subject.id)):[],scope=existing?existing.orgScopeId||'':scopes.length===1?scopes[0]:selectedScope();
 if(existing)dm212Require(existing,'edit');
 const allowed=gvDecision(recActor().id,'documents.edit',scope).allowed;
 if(!allowed&&!config.versionFor&&!config.subject){const choices=['',...ORG.map(n=>n.id)].filter(id=>gvDecision(recActor().id,'documents.edit',id).allowed&&(!selectedScope()||id&&orgWithin(id,selectedScope())));requireValue(choices.length,'Your role has no document upload location in this scope.');openDialog('Choose the upload location','The files will belong to this location.',select('scope','Accountable location',choices.map(id=>[id,id?orgName(id):'Bank-wide']),choices[0],'required'),'Choose files',form=>{const chosen=values(form).scope;requireValue(choices.includes(chosen)&&gvDecision(recActor().id,'documents.edit',chosen).allowed,'The selected upload location is no longer permitted.');dm212BeforeStartUpload(config);DOC_UPLOAD.meta.scope=chosen;DOC_UPLOAD.meta.scopeMode=chosen?'scoped':'shared';DOC_UPLOAD.scopeLocked=true;render();});return;}
 requireValue(allowed,'Your role cannot add a document in this location.');dm212BeforeStartUpload(config);dm212CheckUpload(DOC_UPLOAD);
};
function dm212Archive(ids,restore,{expected,actorId,reason}={}){
 requireValue(ids.length&&new Set(ids).size===ids.length,'Choose documents to update.');requireValue(!actorId||actorId===recActor().id,'The acting user changed. Reopen the archive action.');requireValue(String(reason||'').trim().length>=10,'Record a reason for the archive or restoration.');
 const docs=ids.map(id=>dm212Require(recRead('document',id),'edit'));for(const d of docs){requireValue(!expected||expected[d.id]===dm212Fingerprint(d),'A selected document changed. Reopen the archive action.');requireValue(d.archived===!!restore,'The archive state changed. Reopen the selection.');}
 if(!restore){requireValue(!S.governance.requests.some(r=>r.status==='Pending'&&r.subject.type==='document'&&ids.includes(r.subject.id)),'Withdraw the pending file approval before archiving.');requireValue(!S.evidence.some(x=>x.status==='Submitted'&&ids.includes(docVersion(x.documentVersionId)?.documentId)),'Complete or return the pending assessment document review before archiving.');}
 for(const d of docs){d.archived=!restore;recEvent('archive',{type:'document',id:d.id},(restore?'Restored: ':'Archived: ')+reason,recActor().id);}save();return docs;
}
function dm212ReportCaptureCheck(id){
 const p=recRead('report',id);requireValue(p,'The published report is unavailable.');
 const scope=recScope('report',p)[0]||'';
 if(window.CLEAR_SIGHT_SCOPED_ANALYTICS){for(const permission of ['reports.view','reports.export']){const decision=CLEAR_SIGHT_SCOPED_ANALYTICS.snapshotAccess(p.snapshot,permission);requireValue(decision.allowed,decision.reason);}}
 else for(const permission of ['reports.view','reports.export'])requireValue(gvDecision(recActor().id,permission,scope).allowed,'The report cannot be exported with the current permission.');
 requireValue(!selectedScope()||scope&&orgWithin(scope,selectedScope()),'The report belongs to another or broader location. Choose its captured location before saving it to Documents.');
 const edit=gvDecision(recActor().id,'documents.edit',scope);requireValue(edit.allowed,edit.reason);return {p,scope};
}
docSaveReport=async function(id){
 const {p,scope}=dm212ReportCaptureCheck(id),actor=recActor().id,viewScope=selectedScope(),basis=recStable(p),content=p.name+'\nManagement report snapshot\n\n'+JSON.stringify(p,null,2),hash=recHashText(content),existing=S.governance.versions.find(v=>v.sha256===hash&&(()=>{const d=recRead('document',v.documentId);return d&&!d.archived&&docVisible(d)&&d.orgScopeId===scope;})());
 if(existing){go('document?id='+existing.documentId+'&version='+existing.id);return existing;}
 const did=uid('DOC'),vid=uid('FILE'),blob=new Blob([content],{type:'text/plain'});let stored=false;
 try{const storage=await DocumentBytes.put(vid,blob);stored=true;requireValue(actor===recActor().id&&viewScope===selectedScope()&&basis===recStable(recRead('report',id)),'The report, actor or selected location changed while the source was saved.');dm212ReportCaptureCheck(id);const now=new Date().toISOString(),name=p.name.replace(/[^a-zA-Z0-9 _.-]/g,'').slice(0,130)+'.txt',d={id:did,title:p.name,folderId:'F-REPORTS',classification:'Confidential',orgScopeId:scope,scopeMode:scope?'scoped':'shared',ownerId:actor,currentVersionId:vid,archived:false,createdAt:now},v={id:vid,documentId:did,number:1,fileName:name,mime:'text/plain',size:blob.size,sha256:hash,storage,uploadedBy:actor,createdAt:now,note:'Generated from published report '+id};S.governance.documents.unshift(d);S.governance.versions.push(v);docLinkVersion(vid,{type:'report',id},'',true);recEvent('generated',{type:'document',id:did},'Saved published report '+id,actor);save();go('document?id='+did);return v;
 }catch(error){if(stored&&!docVersion(vid))await DocumentBytes.remove(vid);throw error;}
};
docArchiveDialog=function(ids){const docs=ids.map(id=>dm212Require(recRead('document',id),'edit')),restore=docs.every(d=>d.archived),actorId=recActor().id,expected=Object.fromEntries(docs.map(d=>[d.id,dm212Fingerprint(d)]));requireValue(docs.length&&docs.every(d=>d.archived===restore),'Select either active or archived documents.');openDialog(restore?'Restore documents':'Archive documents','Retained links, versions and decision history remain available.',`<p>${docs.length} document${docs.length===1?'':'s'} selected.</p>${area('reason','Reason','','required minlength="10" maxlength="2000"')}`,restore?'Restore':'Archive',form=>{dm212Archive(ids,restore,{expected,actorId,reason:values(form).reason});DOC_SELECTED.clear();render();});};
docMoveDialog=function(ids){const docs=ids.map(id=>dm212Require(recRead('document',id),'edit')),actorId=recActor().id,basis=Object.fromEntries(docs.map(d=>[d.id,dm212Fingerprint(d)]));requireValue(docs.length,'Select documents to move.');openDialog('Move documents','File locations, versions and links stay fixed.',select('folder','Destination folder',S.governance.folders.map(f=>[f.id,f.name]),docs[0].folderId),'Move',form=>{requireValue(recActor().id===actorId,'The acting user changed. Reopen the move action.');const folder=values(form).folder;requireValue(S.governance.folders.some(f=>f.id===folder),'Choose an existing folder.');for(const d of docs){dm212Require(d,'edit');requireValue(basis[d.id]===dm212Fingerprint(d),'A selected document changed. Reopen the move action.');}for(const d of docs){d.folderId=folder;recEvent('move',{type:'document',id:d.id},'Moved to '+docFolderName(folder),actorId);}DOC_SELECTED.clear();save();render();});};

/* Recipient supplied documents are captured as originals, with external and
 * importing identities kept separate. No approval is created by importing. */
async function dm212ImportExternalFiles(config){
 const {assignmentId,requestId,respondent,files,sourceCheck}=config,a=atAssignment(assignmentId),actorId=recActor().id,scopes=atScope(atProgram(a.programId,a.target),a.target),scope=selectedScope();
 requireValue(a.respondent.kind==='external'&&a.coordinatorId===actorId&&atCanRespond(a),'Only the assigned internal coordinator can capture these documents.');requireValue(!config.expectedActorId||config.expectedActorId===actorId,'The importing user changed.');requireValue(typeof sourceCheck==='function'&&sourceCheck()===true,'The external request is no longer current.');requireValue(typeof requestId==='string'&&requestId.length<=120,'A current external request is required.');requireValue(scopes.length===1,'An external upload needs one exact accountable location.');requireValue(Array.isArray(files)&&files.length<=DM212_LIMITS.files,'A response accepts at most 10 documents.');
 const check=()=>{requireValue(recActor().id===actorId&&selectedScope()===scope&&!S.scopeInvalid,'The importing actor or location changed.');const fresh=atAssignment(assignmentId);requireValue(fresh.coordinatorId===actorId&&atCanRespond(fresh)&&recStable(atScope(atProgram(fresh.programId,fresh.target),fresh.target))===recStable(scopes),'The assignment or accountable location changed.');const permission=gvDecision(actorId,'documents.edit',scopes[0]);requireValue(permission.allowed,permission.reason);requireValue(sourceCheck()===true,'The external request or response source changed.');};check();
 const validated=[],ids=new Set();let total=0;
 for(const f of files){requireValue(typeof f.fileId==='string'&&f.fileId.length<=100&&!ids.has(f.fileId),'Every supplied file needs a unique response file identifier.');ids.add(f.fileId);requireValue(typeof f.base64==='string'&&f.base64.length<=Math.ceil(DM212_LIMITS.file/3)*4&&f.base64.length%4===0&&!/[^A-Za-z0-9+/=]/.test(f.base64)&&!f.base64.slice(0,-2).includes('=')&&!/=.$/.test(f.base64.replace(/==$/,'')),'Invalid encoded document bytes.');const binary=atob(f.base64),bytes=Uint8Array.from(binary,c=>c.charCodeAt(0));total+=bytes.length;requireValue(total<=DM212_LIMITS.batch,'The response document batch must be 20 MB or smaller.');const format=docValidateFile({name:f.fileName,size:bytes.length},bytes),hash=recSHA256(bytes);requireValue(!f.mime||f.mime===format.mime,'The file type does not match the claimed format.');requireValue(!f.sha256||f.sha256===hash,'The supplied document checksum does not match its bytes.');validated.push({source:f,bytes,format,hash,id:uid('DOC'),vid:uid('FILE')});}
 const stored=[];
 try{
  for(const item of validated){item.storage=await DocumentBytes.put(item.vid,new Blob([item.bytes],{type:item.format.mime}));stored.push(item);}check();
  const now=new Date().toISOString(),folder=S.governance.folders.some(f=>f.id==='F-COMPLIANCE')?'F-COMPLIANCE':S.governance.folders[0]?.id;requireValue(folder,'Create a document folder before importing files.');
  for(const item of stored){const d={id:item.id,title:item.source.fileName,folderId:folder,classification:'Confidential',orgScopeId:scopes[0],scopeMode:'scoped',ownerId:actorId,archived:false,createdAt:now,currentVersionId:item.vid},v={id:item.vid,documentId:item.id,number:1,fileName:item.source.fileName,mime:item.format.mime,size:item.bytes.length,sha256:item.hash,storage:item.storage,uploadedBy:actorId,createdAt:now,note:'Supplied with external assessment response',externalSource:{requestId,assignmentId,respondentId:a.respondent.id,respondentName:a.respondent.name,respondentEmail:a.respondent.email||'',fileId:item.source.fileId}};S.governance.documents.unshift(d);S.governance.versions.push(v);recEvent('external-upload',{type:'document',id:d.id},'Captured document from '+a.respondent.name,actorId,{documentVersionId:v.id,requestId,assignmentId});}
  save();return stored.map(x=>({fileId:x.source.fileId,documentId:x.id,documentVersionId:x.vid,fileName:x.source.fileName,sha256:x.hash}));
 }catch(error){for(const item of stored)if(!docVersion(item.vid))await DocumentBytes.remove(item.vid);throw error;}
}
async function dm212RollbackExternalFiles({requestId,documentVersionIds}){
 const versions=documentVersionIds.map(docVersion);requireValue(versions.every(v=>v?.externalSource?.requestId===requestId&&v.uploadedBy===recActor().id),'Only this importing user can withdraw unlinked documents from this request.');
 for(const v of versions){requireValue(!S.governance.links.some(l=>l.documentVersionId===v.id)&&!S.evidence.some(x=>x.documentVersionId===v.id)&&!S.governance.requests.some(r=>r.versionId===v.id),'A linked or reviewed imported file cannot be rolled back.');requireValue(S.governance.versions.filter(x=>x.documentId===v.documentId).length===1,'A revised document cannot be rolled back.');}
 const ids=new Set(versions.map(v=>v.id)),documents=new Set(versions.map(v=>v.documentId));S.governance.versions=S.governance.versions.filter(v=>!ids.has(v.id));S.governance.documents=S.governance.documents.filter(d=>!documents.has(d.id));for(const id of documents)recEvent('external-import-rollback',{type:'document',id},'Unlinked capture rolled back for '+requestId);save();for(const id of ids)await DocumentBytes.remove(id);
}
Object.assign(actions,{
 'dm212-clear-filters':()=>{DOC_FILTERS={search:'',folder:'',type:'',status:'',sort:'updated',archived:false};DOC_SELECTED.clear();go(ui211Route(dm212Here(),{docOwner:null,docClass:null,docLife:null,docLinked:null,docSearch:null,docFolder:null,docType:null,docStatus:null,docArchived:null,docSort:null}));},
 'dm212-edit':el=>dm212Edit(el.dataset.id),
 'dm212-expand-preview':el=>{const layout=el.closest('.doc-detail-layout'),panel=layout?.querySelector('.doc-preview')?.closest('.panel');if(panel){panel.classList.toggle('dm212-expanded');const expanded=panel.classList.contains('dm212-expanded');layout.classList.toggle('dm212-wide',expanded);el.textContent=expanded?'Restore split view':'Expand preview';el.setAttribute('aria-pressed',String(expanded));}},
 'doc-download':async el=>{const v=docVersion(el.dataset.id);requireValue(v,'File version unavailable.');const blob=await dm212ReadVersion(v.id,{purpose:'export'});docBlobDownload(blob,v.fileName);},
 'doc-save-report':el=>docSaveReport(el.dataset.id),
 'doc-open-pdf':async el=>{const v=docVersion(el.dataset.id);requireValue(v?.mime==='application/pdf','Choose a PDF version.');const blob=await dm212ReadVersion(v.id),url=URL.createObjectURL(blob);DOC_PREVIEW_URLS.push(url);window.open(url,'_blank','noopener,noreferrer');},
 'doc-export':()=>{const rows=docRows();requireValue(rows.length,'No displayed documents to export.');rows.forEach(d=>dm212Require(d,'export'));exportCSV('clearsight-document-register.csv',['View scope','Document ID','Title','Version ID','Version','Filename','File scope','Owner','Classification','File approval','Document dates','Review by','Valid through','Tags','SHA-256'],rows.map(d=>{const v=docHead(d);return [selectedScope()?orgName(selectedScope()):'All permitted locations',d.id,d.title,v.id,v.number,v.fileName,docScopeLabel(d),recUser(d.ownerId)?.name||'',d.classification,docStatus(d),dm212Lifecycle(d).state,d.reviewOn||'',d.validThrough||'',(d.tags||[]).join(';'),v.sha256||''];}));}
});
const dm212NewFolder=actions['doc-new-folder'];actions['doc-new-folder']=()=>{requireValue(gvDecision(recActor().id,'documents.manage','').allowed,'Organization-wide document management permission is required to create a shared folder.');const actor=recActor().id;dm212NewFolder();const callback=modalSave;modalSave=form=>{requireValue(actor===recActor().id&&gvDecision(actor,'documents.manage','').allowed,'The acting user or document management permission changed.');return callback(form);};};
document.addEventListener('change',event=>{const el=event.target;try{
 if(el.matches?.('[data-dm212-filter]')){DOC_SELECTED.clear();go(ui211Route(dm212Here(),{[el.dataset.dm212Filter]:el.value}));}
 if(el.matches?.('[data-dm212-version]')){go(ui211Route(dm212Here(),{version:el.value,review:null,compare:null}));}
 if(el.matches?.('[data-dm212-compare]'))go(ui211Route(dm212Here(),{compare:el.value}));
}catch(error){toast(error.message);}});
const dm212PriorRender=render;render=function(){dm212PriorRender();for(const element of $$('[data-action="doc-save-report"]')){try{dm212ReportCaptureCheck(element.dataset.id);}catch(error){element.disabled=true;element.setAttribute('aria-disabled','true');element.title=error.message;}}};
window.CLEAR_SIGHT_DOCUMENT_MANAGEMENT=Object.freeze({version:'2.12',decision:dm212Decision,readVersion:dm212ReadVersion,metadata:dm212Metadata,lifecycle:dm212Lifecycle,rememberFilters:dm212RememberFilters,archive:dm212Archive,importExternalFiles:dm212ImportExternalFiles,rollbackExternalFiles:dm212RollbackExternalFiles,readable:dm212Readable});
