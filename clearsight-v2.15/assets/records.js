/* Shared record contracts for V2.6.
 * This is a local design prototype. References are typed IDs; authorization is not
 * simulated by hiding UI. Domain decisions remain owned by their original modules.
 */
'use strict';
const REC_PEOPLE = CS_SEED.people.map((name, index) => ({id:'USR-'+String(index+1).padStart(2,'0'),name}));
const REC_FOLDERS = [
  {id:'F-COMPLIANCE',name:'Compliance'}, {id:'F-OPERATIONS',name:'Operations'},
  {id:'F-VENDORS',name:'Third parties'}, {id:'F-PRIVACY',name:'Privacy'},
  {id:'F-REPORTS',name:'Management reports'}
];
const REC_TYPES = {
  program:{collection:()=>S.programs,label:'Program',route:r=>'program?id='+r.id},
  risk:{collection:()=>S.risks,label:'Risk',route:r=>'risk?id='+r.id},
  loss:{collection:()=>S.losses,label:'Loss event',route:r=>'loss?id='+r.id},
  control:{collection:()=>S.controls,label:'Control',route:r=>'control?id='+r.id},
  engagement:{collection:()=>S.engagements,label:'Vendor engagement',route:r=>'vendor?id='+r.id},
  activity:{collection:()=>S.processing,label:'Processing activity',route:r=>'activity?id='+r.id},
  dpia:{collection:()=>S.dpias||[],label:'DPIA',route:r=>'dpia?id='+r.id},
  privacyCase:{collection:()=>S.privacyCases||[],label:'Privacy case',route:r=>'privacy-case?id='+r.id},
  evidence:{collection:()=>S.evidence,label:'Documents',route:r=>'evidence-review?id='+r.id},
  kriReturn:{collection:()=>Object.values(S.branchKri?.returns||{}),label:'KRI return',route:r=>'kri-return?id='+encodeURIComponent(r.id)},
  rcsa:{collection:()=>S.cycles,label:'RCSA cycle',route:r=>'rcsa-review?id='+r.id},
  formResponse:{collection:()=>Object.entries(S.formStudio?.responses||S.formEngine?.responses||S.formSystem?.responses||{}).map(([id,r])=>({...r,id})),label:'Form response',route:r=>'form-response?binding='+encodeURIComponent(r.id)},
  report:{collection:()=>S.reportPacks||[],label:'Management report',route:r=>'report-pack?id='+r.id},
  document:{collection:()=>S.governance?.documents||[],label:'Document',route:r=>'document?id='+r.id}
};
function recRead(type,id){return REC_TYPES[type]?.collection().find(x=>x.id===id)||null;}
function recName(type,r){
  if(!r)return 'Record unavailable';
  if(type==='engagement')return targetName(r.id);
  if(type==='kriReturn')return (ixCycle(r.cycle)?.title||r.cycle)+' · '+orgName(r.unit);
  if(type==='evidence')return docVersion(r.documentVersionId)?.fileName||r.name||r.id;
  return r.title||r.name||r.id;
}
function recUrl(type,id){const x=recRead(type,id);return x?REC_TYPES[type].route(x):'';}
function recUser(id){return REC_PEOPLE.find(x=>x.id===id)||null;}
function recUserId(name){return REC_PEOPLE.find(x=>x.name===name)?.id||'';}
function recActor(){return recUser(S.governance.actorId)||REC_PEOPLE[0];}
function recScope(type,r){
  if(!r)return [];
  if(type==='kriReturn')return [r.unit];
  if(type==='report'){const id=r.snapshot?.scopeId||r.snapshot?.scope?.unit||r.snapshot?.scope?.department||r.snapshot?.scope?.branch||r.snapshot?.scope?.id||'';return id?[id]:[];}
  if(type==='formResponse'){
    const b=faInit().bindings.find(b=>b.id===r.id),p=b&&recRead('program',b.programId);
    if(!p)return [];const engagement=S.engagements.find(x=>x.id===b.target);return engagement?scopeIds(engagement,'engagements'):scopeIds(p,'programs');
  }
  if(type==='document')return r.orgScopeId?[r.orgScopeId]:[];
  return scopeIds(r,({activity:'processing',engagement:'engagements',program:'programs',risk:'risks',loss:'losses',control:'controls',rcsa:'cycles',evidence:'evidence'})[type]||type);
}
function recInScope(type,r,scope=selectedScope()){
  if(!scope)return true;
  if(type==='document'&&r.scopeMode==='shared')return true;
  return recScope(type,r).some(id=>orgWithin(id,scope)||orgWithin(scope,id));
}
function recRef(type,id){const x=recRead(type,id);return {type,id,name:recName(type,x),route:x?REC_TYPES[type].route(x):'',scopeIds:recScope(type,x),available:!!x};}
function recContext(){
  if(current.blocked)return null;
  const map={program:'program',assessment:'program',vendor:'engagement',risk:'risk',loss:'loss',control:'control',activity:'activity',dpia:'dpia','privacy-case':'privacyCase','kri-return':'kriReturn','rcsa-review':'rcsa','report-pack':'report','form-response':'formResponse'};
  const type=map[current.name];if(!type)return null;
  let id=current.params.get('id');
  if(type==='program')id=prog()?.id;
  if(type==='formResponse'){const b=faBinding();if(!b)return null;id=b.id;}
  if(!id&&type==='engagement')id=S.engagements[0]?.id;
  const r=recRead(type,id);return r?{type,id,record:r}:null;
}
function recTimestamp(value){
  if(!value)return '—';const d=new Date(value);if(Number.isNaN(d.getTime()))return 'Invalid date';
  return new Intl.DateTimeFormat('en-GB',{dateStyle:'medium',timeStyle:'short',timeZone:'Africa/Lagos'}).format(d)+' WAT';
}
function recBytes(size){if(size==null)return 'Size not supplied';return size<1024?size+' B':size<1048576?(size/1024).toFixed(1)+' KB':(size/1048576).toFixed(1)+' MB';}
function recStable(value){
  if(value===null||typeof value!=='object')return JSON.stringify(value);
  if(Array.isArray(value))return '['+value.map(x=>recStable(x===undefined?null:x)).join(',')+']';
  return '{'+Object.keys(value).filter(k=>value[k]!==undefined).sort().map(k=>JSON.stringify(k)+':'+recStable(value[k])).join(',')+'}';
}
// Portable SHA-256 for local/file origins where SubtleCrypto is unavailable.
// Integrity checksum only: this does NOT authenticate a signer or make a digital signature.
function recSHA256(bytes){
  const K=[0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2];
  const H=[0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19];
  const data=new Uint8Array(Math.ceil((bytes.length+9)/64)*64);data.set(bytes);data[bytes.length]=128;
  const dv=new DataView(data.buffer);dv.setUint32(data.length-8,Math.floor(bytes.length/0x20000000));dv.setUint32(data.length-4,bytes.length*8>>>0);
  const ro=(x,n)=>(x>>>n)|(x<<(32-n)),w=new Int32Array(64);
  for(let off=0;off<data.length;off+=64){
    for(let i=0;i<16;i++)w[i]=dv.getInt32(off+i*4);
    for(let i=16;i<64;i++){const a=w[i-15],b=w[i-2];w[i]=(w[i-16]+(ro(a,7)^ro(a,18)^(a>>>3))+w[i-7]+(ro(b,17)^ro(b,19)^(b>>>10)))|0;}
    let [a,b,c,d,ee,f,g,h]=H;
    for(let i=0;i<64;i++){const t1=(h+(ro(ee,6)^ro(ee,11)^ro(ee,25))+((ee&f)^(~ee&g))+K[i]+w[i])|0,t2=((ro(a,2)^ro(a,13)^ro(a,22))+((a&b)^(a&c)^(b&c)))|0;h=g;g=f;f=ee;ee=(d+t1)|0;d=c;c=b;b=a;a=(t1+t2)|0;}
    [a,b,c,d,ee,f,g,h].forEach((v,i)=>H[i]=(H[i]+v)|0);
  }
  return H.map(x=>(x>>>0).toString(16).padStart(8,'0')).join('');
}
const recHashText=text=>recSHA256(new TextEncoder().encode(text));
function recEvent(kind,ref,message,actor=recActor().id,detail={}){
  const item={id:uid('EVT'),kind,ref:clone(ref),message,actorId:actor,at:new Date().toISOString(),...detail};
  S.governance.events.push(item);save();return item;
}
function recDecision(type,id,actorName,decision,note,snapshot,sourceVersion=''){
  const receipt={id:uid('DEC'),subject:{type,id},actorId:recUserId(actorName),actorName,decision,note,at:new Date().toISOString(),sourceVersion,snapshotHash:recHashText(recStable(snapshot)),snapshot:clone(snapshot),method:'Internal review · demo identity'};
  S.governance.receipts.push(receipt);save();return receipt;
}
function recordsInit(){
  ixInit();journeyInit();
  S.governance??={schema:1,actorId:'USR-01',documents:[],versions:[],links:[],requests:[],receipts:[],events:[],folders:clone(REC_FOLDERS)};
  const g=S.governance;
  {
    for(const ev of S.evidence.filter(ev=>!ev.documentVersionId)){
      const id='DOC-'+ev.id,vid=id+'-V1';
      g.documents.push({id,title:ev.name,folderId:ev.target?.startsWith('ENG')?'F-VENDORS':'F-COMPLIANCE',classification:'Internal',orgScopeId:ev.orgScopeId||'',scopeMode:ev.orgScopeId?'scoped':'shared',ownerId:recUserId(ev.owner),currentVersionId:vid,archived:false,createdAt:'2026-11-01T09:00:00Z'});
      g.versions.push({id:vid,documentId:id,number:1,fileName:ev.name,mime:'',size:ev.size||0,sha256:null,storage:'reference',uploadedBy:recUserId(ev.owner),createdAt:ev.observed+'T09:00:00Z',note:'Earlier demo evidence reference; original file was not supplied.'});
      ev.documentVersionId=vid;
      g.links.push({id:'LINK-'+ev.id,documentVersionId:vid,subject:{type:'evidence',id:ev.id},anchor:ev.question,createdBy:recUserId(ev.owner),createdAt:ev.observed+'T09:00:00Z'});
      if(ev.program)g.links.push({id:'LINK-P-'+ev.id,documentVersionId:vid,subject:{type:'program',id:ev.program},anchor:ev.question,createdBy:recUserId(ev.owner),createdAt:ev.observed+'T09:00:00Z'});
      if(ev.target?.startsWith('ENG'))g.links.push({id:'LINK-E-'+ev.id,documentVersionId:vid,subject:{type:'engagement',id:ev.target},anchor:'',createdBy:recUserId(ev.owner),createdAt:ev.observed+'T09:00:00Z'});
    }
    g.migratedEvidence=true;
  }
  // Compatibility accessors avoid competing file names/sizes on old evidence screens.
  // Persistence contains only the reference and evidence-specific evaluation metadata.
  for(const ev of S.evidence){
    if(!ev.documentVersionId)continue;
    const v=g.versions.find(v=>v.id===ev.documentVersionId);if(!v)continue;
    for(const [key,getter] of Object.entries({name:()=>docVersion(ev.documentVersionId)?.fileName||'File unavailable',size:()=>docVersion(ev.documentVersionId)?.size||0})){Object.defineProperty(ev,key,{configurable:true,enumerable:true,get:getter});}
    Object.defineProperty(ev,'toJSON',{configurable:true,value(){const out={};for(const key of Object.keys(this))if(!['name','size'].includes(key))out[key]=this[key];return out;}});
  }
  for(const type of ['program','risk','loss','control','engagement','activity','dpia','privacyCase','evidence','kriReturn','rcsa'])for(const row of REC_TYPES[type].collection()){
    for(const nameKey of ['owner','reviewer','submittedBy']){
      const idKey=nameKey+'Id',known=recUserId(row[nameKey]);
      if(!row[idKey]&&known)row[idKey]=known;
      if(!recUser(row[idKey]))continue;
      Object.defineProperty(row,nameKey,{configurable:true,enumerable:true,get(){return recUser(this[idKey])?.name||'';},set(name){const id=recUserId(name);if(!id)throw Error('Choose a known person for '+nameKey+'.');this[idKey]=id;}});
    }
  }
  for(const [id,res] of Object.entries(faInit().responses))if(res.status==='Submitted'&&res.snapshot&&res.definitionSnapshot&&!res.submissionHash){
    res.submissionHash=submissionDigest(faInit().bindings.find(b=>b.id===id),res);
    res.checksumRecordedOnImport=true; // Local import checksum, not prior signature evidence.
  }
  REC_TYPES.formResponse.collection=()=>Object.entries(faInit().responses).map(([id,r])=>({...r,id}));
}
function recIntegrity(){
  const problems=[],g=S.governance;
  const check=(condition,code,ref,message)=>{if(!condition)problems.push({code,ref,message});};
  for(const [type,cfg]of Object.entries(REC_TYPES)){const ids=new Set();for(const r of cfg.collection()){check(!ids.has(r.id),'DUPLICATE_ID',r.id,type+' has a duplicate identity.');ids.add(r.id);for(const id of recScope(type,r))check(!!orgNode(id),'INVALID_SCOPE',r.id,'Unknown organization scope '+id);}}
  for(const d of g.documents){const v=docVersion(d.currentVersionId);check(v&&v.documentId===d.id,'INVALID_HEAD',d.id,'Document head does not identify its own version.');check(g.folders.some(f=>f.id===d.folderId),'MISSING_FOLDER',d.id,'Folder is unavailable.');}
  for(const v of g.versions){check(!!recRead('document',v.documentId),'ORPHAN_VERSION',v.id,'The parent document does not exist.');check(v.storage==='reference'||/^[0-9a-f]{64}$/.test(v.sha256||''),'MISSING_HASH',v.id,'Stored bytes need a content checksum.');}
  for(const l of g.links){check(!!docVersion(l.documentVersionId),'ORPHAN_LINK',l.id,'Linked version is missing.');check(!!recRead(l.subject.type,l.subject.id),'INVALID_TARGET',l.id,'Attachment target is unavailable.');}
  for(const x of S.evidence)if(x.documentVersionId)check(!!docVersion(x.documentVersionId),'MISSING_EVIDENCE_FILE',x.id,'Evidence points to an unavailable version.');
  for(const l of S.losses){check(!l.risk||!!recRead('risk',l.risk),'ORPHAN_LOSS',l.id,'The linked risk is unavailable.');check(Number.isSafeInteger(l.gross)&&l.gross>=0,'INVALID_MONEY',l.id,'Gross amount must be non-negative integer minor units.');check(recoveries(l)<=l.gross,'OVER_RECOVERY',l.id,'Recovered amount exceeds gross.');}
  for(const b of faInit().bindings){const p=recRead('program',b.programId);check(p&&p.targets.includes(b.target),'INVALID_ASSESSMENT_TARGET',b.id,'The bound target is unavailable.');check(!!faVersion(b.formVersionId),'MISSING_FORM_VERSION',b.id,'Published form version is unavailable.');}
  for(const [id,res]of Object.entries(faInit().responses))if(res.status==='Submitted')check(res.snapshot&&res.definitionSnapshot&&res.submissionHash===submissionDigest(faInit().bindings.find(b=>b.id===id),res),'CHANGED_ASSESSMENT_SNAPSHOT',id,'Submitted form data is missing or changed.');
  for(const r of g.requests){check(recHashText(recStable(r.snapshot))===r.snapshotHash,'CHANGED_SNAPSHOT',r.id,'Requested approval content has changed.');check(r.steps.every(s=>s.actorId!==r.requestedBy),'SELF_APPROVAL',r.id,'Submitter also appears as an approver.');}
  return {asOf:S.asOf,checkedAt:new Date().toISOString(),problems,counts:{documents:g.documents.length,versions:g.versions.length,links:g.links.length,requests:g.requests.length},scope:clone(S.scope)};
}
