/* Shared interaction primitives. Domain commands remain the source of decisions. */
'use strict';

function ui211Route(value,patch={}){
 const [name,query='']=String(value||'').replace(/^#\//,'').split('?'),p=new URLSearchParams(query);
 for(const [key,v] of Object.entries(patch)){if(v===null||v===undefined||v==='')p.delete(key);else p.set(key,String(v));}
 return name+(p.toString()?'?'+p.toString():'');
}
function ui211Focus(selector,root){
 root??=($('#dialog')?.open?$('#dialog'):$('#main-content'))||document;
 let target;try{target=root.querySelector?.(selector);if(!target&&root.matches?.(selector))target=root;}catch{return false;}
 if(!target)return false;
 for(let p=target.parentElement;p&&p!==root;p=p.parentElement)if(p.tagName==='DETAILS')p.open=true;
 const editable='input:not([type="hidden"]):not([disabled]),select:not([disabled]),textarea:not([disabled]),button:not([disabled]),a[href]';
 const field=target.matches?.(editable)?target:target.querySelector?.(editable)||target;
 if(!field.matches?.(editable))field.setAttribute?.('tabindex','-1');
 target.scrollIntoView?.({block:'center',behavior:'auto'});field.focus?.({preventScroll:true});return true;
}
function ui211Issues(items,{title='Complete before submitting',id=''}={}){
 if(!items.length)return '';
 return `<section class="ui211-issues" ${id?`id="${e(id)}"`:''} aria-label="${e(title)}"><h3>${e(title)} <span>${items.length}</span></h3><ul>${items.map(item=>`<li>${item.route?`<a href="#/${e(item.route)}">${e(item.message)}</a>`:item.selector?`<button type="button" data-action="ui211-focus" data-selector="${e(item.selector)}">${e(item.message)}</button>`:e(item.message)}</li>`).join('')}</ul></section>`;
}
function ui211Next({title,description='',owner='',due='',state='',primary='',secondary=''}){
 return `<section class="ui211-next"><div class="ui211-next-copy"><h2>${e(title)}</h2>${description?`<p>${e(description)}</p>`:''}${owner||due||state?`<div class="ui211-next-meta">${owner?`<span>${icon('users')}${e(owner)}</span>`:''}${due?`<span>${icon('clock')}${e(date(due))}</span>`:''}${state?pill(state):''}</div>`:''}</div><div class="ui211-next-actions">${primary}${secondary}</div></section>`;
}

// Track only the current modal's unsaved presentation fields, never another domain draft.
let UI211_MODAL=null,ui211ModalSequence=0,ui211FieldSequence=0;
// Focus belongs to the existing control, even when a successful save rebuilds
// its workspace. Match stable attributes, never an ordinal or another record.
function ui215FocusContext(){return (current.name||'')+'?'+(current.params?.toString()||'')+'|'+selectedScope()+'|'+recActor().id;}
function ui215FocusIdentity(node){
 if(!node)return null;
 const attributes=Array.from(node.attributes||[]).filter(a=>['id','name','href','type'].includes(a.name)||a.name.startsWith('data-')).map(a=>[a.name,a.value]);
 return attributes.length?{tag:node.tagName,attributes}:null;
}
function ui215FocusReference(node=document.activeElement){
 const area=['#main-content','.topbar','.sidebar'].find(selector=>$(selector)?.contains?.(node));
 if(!area)return null;
 const identity=ui215FocusIdentity(node);if(!identity)return null;
 const form=node.closest?.('form');
 return {area,identity,form:form?ui215FocusIdentity(form):null,context:ui215FocusContext()};
}
function ui215FocusMatches(node,identity){return !!(node&&identity&&node.tagName===identity.tag&&identity.attributes.every(([name,value])=>node.getAttribute?.(name)===value));}
function ui215FocusTarget(reference){
 if(!reference||reference.context!==ui215FocusContext())return null;
 const root=$(reference.area);if(!root)return null;
 const nodes=Array.from(root.querySelectorAll('a[href],button,input,select,textarea,[tabindex]')).filter(node=>
  ui215FocusMatches(node,reference.identity)&&(!reference.form||ui215FocusMatches(node.closest?.('form'),reference.form))&&
  !node.disabled&&!node.hidden&&node.isConnected!==false&&!node.closest?.('[hidden],[inert]')&&(!node.getClientRects||node.getClientRects().length));
 return nodes.length===1?nodes[0]:null;
}
function ui215PageFocus(){
 const main=$('#main-content'),heading=main?.querySelector('h1'),target=heading||main;
 if(!target||target.isConnected===false||main.inert)return false;
 if(heading)heading.setAttribute('tabindex','-1');
 target.focus?.({preventScroll:true});return true;
}
function ui211FormFingerprint(form){
 if(!form)return '';
 const fields=Array.from(form.elements||[]).filter(x=>!['button','submit','reset'].includes(x.type));
 return JSON.stringify(fields.map(x=>[x.name||x.id||'',x.type||'',x.type==='checkbox'||x.type==='radio'?Boolean(x.checked):x.type==='file'?Array.from(x.files||[]).map(f=>[f.name,f.size,f.lastModified]):x.multiple?Array.from(x.selectedOptions||[]).map(o=>o.value):x.value]));
}
function ui211ModalDirty(){return !!(UI211_MODAL?.edited&&UI211_MODAL.form&&ui211FormFingerprint(UI211_MODAL.form)!==UI211_MODAL.baseline);}
const ui211PriorOpenDialog=openDialog;
openDialog=function(...args){
 const returnFocus=ui215FocusReference()||UI211_MODAL?.returnFocus||null;
 ui211PriorOpenDialog(...args);const form=$('#modal-form'),dialog=$('#dialog');
 UI211_MODAL={id:++ui211ModalSequence,form,callback:modalSave,baseline:ui211FormFingerprint(form),edited:false,busy:false,pending:null,returnFocus,originHash:location.hash||'#/'+(current.name||'home')+(current.params?.toString()?'?'+current.params.toString():'')};
 if(form&&modalSave){form.dataset.ui211Guard='true';form.noValidate=true;const foot=$('.dialog-foot',form);foot?.insertAdjacentHTML('afterbegin','<span class="ui211-form-state" role="status" aria-live="polite">Changes apply when you save.</span>');}
 dialog?.setAttribute?.('aria-busy','false');
};
const ui211PriorCloseDialog=closeDialog;
closeDialog=function(){
 if(UI211_MODAL?.busy)return false;
 const state=UI211_MODAL,wasOpen=$('#dialog')?.open,original=lastFocus;UI211_MODAL=null;ui211PriorCloseDialog();
 if(wasOpen&&(!original?.isConnected||original.disabled||original.closest?.('#dialog,[hidden],[inert]')||(original.getClientRects&&!original.getClientRects().length))){
  const target=ui215FocusTarget(state?.returnFocus);if(target)target.focus({preventScroll:true});else ui215PageFocus();
 }
 return true;
};
function ui211DiscardPrompt(next){
 const state=UI211_MODAL;if(!state)return false;
 if(state.busy){toast('Wait for the current operation to finish.');return true;}
 if(!ui211ModalDirty())return false;
 state.pending=next;
 const form=state.form;let region=$('#ui211-discard',form);
 if(!region){$('.dialog-foot',form)?.insertAdjacentHTML('beforebegin',`<section id="ui211-discard" class="ui211-discard" tabindex="-1" role="alert"><div><strong>Keep your unsaved changes?</strong><p>Your changes are still in this form.</p></div><div>${btn('Keep editing','ui211-keep-editing','','primary')}${btn('Discard changes','ui211-discard','','quiet danger')}</div></section>`);region=$('#ui211-discard',form);}
 region?.focus?.();return true;
}
function ui211ModalStatus(state,text){const label=state?.form?.querySelector?.('.ui211-form-state');if(label)label.textContent=text;}
function ui211FieldLabel(field){return String(field.labels?.[0]?.childNodes?.[0]?.textContent||field.getAttribute?.('aria-label')||field.name||'Required field').trim().replace(/\s+/g,' ').slice(0,100);}
function ui211InvalidFields(form){
 const issues=[],radioGroups=new Set();
 for(const field of Array.from(form.elements||[])){
  if(field.disabled||['button','submit','reset','hidden'].includes(field.type))continue;
  const empty=field.required&&(field.type==='checkbox'?!field.checked:field.type==='radio'?!Array.from(form.elements||[]).some(x=>x.name===field.name&&x.checked):field.type==='file'?!field.files?.length:!String(field.value||'').trim());
  if(empty||field.validity?.valid===false){if(field.type==='radio'&&radioGroups.has(field.name))continue;if(field.type==='radio')radioGroups.add(field.name);field.id||='ui211-field-'+(UI211_MODAL?.id||0)+'-'+(++ui211FieldSequence);field.setAttribute?.('aria-invalid','true');issues.push({message:ui211FieldLabel(field)+': '+(empty?'complete this field.':field.validationMessage||'check this value.'),selector:'[id="'+field.id.replace(/\\/g,'\\\\').replace(/"/g,'\\"')+'"]'});}
 }
 return issues;
}
function ui211FormError(form,message,issues=[]){
 const box=form?.querySelector?.('#modal-error')||form?.querySelector?.('.form-error,.j-error')||$('#modal-error');
 if(!box){toast(message);return;}
 box.hidden=false;box.className='form-error ui211-error';box.setAttribute?.('role','alert');box.setAttribute?.('tabindex','-1');
 if(issues.length)box.innerHTML=ui211Issues(issues,{title:message});else box.textContent=message;
 box.focus?.();box.scrollIntoView?.({block:'nearest'});
}
async function ui211SubmitModal(form){
 const state=UI211_MODAL||{id:++ui211ModalSequence,form,callback:modalSave,busy:false};
 if(state.busy||!modalSave)return false;
 state.form=form;state.callback=modalSave;UI211_MODAL=state;
 const issues=ui211InvalidFields(form);if(issues.length){ui211FormError(form,'Check these fields',issues);return false;}
 const buttons=Array.from(form.querySelectorAll?.('button[type="submit"]')||[]),previous=buttons.map(x=>({disabled:x.disabled,text:x.textContent}));
 const body=form.querySelector?.('.dialog-body'),priorInert=body?.inert||false;
 const error=form.querySelector?.('#modal-error');if(error){error.hidden=true;error.textContent='';}
 state.busy=true;form.setAttribute?.('aria-busy','true');$('#dialog')?.setAttribute?.('aria-busy','true');ui211ModalStatus(state,'Saving…');
 buttons.forEach(x=>{x.disabled=true;x.textContent='Saving…';});
 try{
  const pending=state.callback(form);if(pending&&typeof pending.then==='function'&&body)body.inert=true;
  const result=await pending;if(body)body.inert=priorInert;
  if(UI211_MODAL!==state)return result; // A successful command may open its next dialog.
  state.busy=false;
  if(result!==false){state.edited=false;state.baseline=ui211FormFingerprint(form);closeDialog();}else ui211ModalStatus(state,'Review the next step above.');
  return result;
 }catch(error){
  if(body)body.inert=priorInert;
  if(UI211_MODAL===state){state.busy=false;ui211FormError(form,error.message||'The change could not be saved.');ui211ModalStatus(state,'Not saved. Your entries are still here.');}
  return false;
 }finally{
  if(body)body.inert=priorInert;
  form.setAttribute?.('aria-busy','false');if(UI211_MODAL===state)$('#dialog')?.setAttribute?.('aria-busy','false');
  buttons.forEach((x,i)=>{if(x.isConnected!==false){x.disabled=previous[i].disabled;if(x.textContent==='Saving…')x.textContent=previous[i].text;}});
 }
}
const UI211_PENDING_ACTIONS=new WeakSet();
function ui211RunAction(fn,el){
 if(el&&UI211_PENDING_ACTIONS.has(el))return;
 const report=error=>{if($('#dialog')?.open)ui211FormError($('#modal-form'),error.message||'This action could not be completed.');else toast(error.message||'This action could not be completed.');};
 try{
  const result=fn(el);if(!result||typeof result.then!=='function')return result;
  const disabled=el?.disabled;if(el){UI211_PENDING_ACTIONS.add(el);el.disabled=true;el.setAttribute?.('aria-busy','true');}
  return Promise.resolve(result).catch(report).finally(()=>{if(el){UI211_PENDING_ACTIONS.delete(el);el.disabled=disabled;el.removeAttribute?.('aria-busy');}});
 }catch(error){report(error);}
}

actions['ui211-focus']=el=>ui211Focus(el.dataset.selector);
actions['ui211-keep-editing']=()=>{const state=UI211_MODAL;if(!state)return;state.pending=null;$('#ui211-discard',state.form)?.remove?.();const field=state.form.querySelector?.('input:not([type="hidden"]),select,textarea');field?.focus?.();};
actions['ui211-discard']=()=>{const state=UI211_MODAL;if(!state||state.busy)return;const next=state.pending;state.edited=false;closeDialog();next?.();};
document.addEventListener('click',ev=>{
 if(!$('#dialog')?.open||!UI211_MODAL?.callback)return;
 const action=ev.target.closest?.('[data-action]'),anchor=ev.target.closest?.('a[href]'),href=anchor?.getAttribute?.('href');
 const closing=action?.dataset.action==='close-dialog',navigating=href?.startsWith('#/');if(!closing&&!navigating)return;
 const next=navigating?()=>go(href.slice(2)):()=>{};
 if(ui211DiscardPrompt(next)){ev.preventDefault();ev.stopImmediatePropagation?.();}
},true);
function ui215CancelModal(ev){
 ev.preventDefault();ev.stopImmediatePropagation?.();
 if(!ui211DiscardPrompt(()=>{}))closeDialog();
}
$('#dialog')?.addEventListener?.('cancel',ui215CancelModal,true);
function ui211HashChange(ev){
 const state=UI211_MODAL;if(!state?.callback||(!state.busy&&!ui211ModalDirty())||!$('#dialog')?.open)return false;
 const intended=location.hash;if(intended===state.originHash||!window.history?.replaceState)return false;
 try{window.history.replaceState(window.history.state,'',state.originHash||'#/home');}catch{return false;}
 ev.stopImmediatePropagation?.();
 if(state.busy){toast('The current change is still saving. Stay on this page until it finishes.');return true;}
 ui211DiscardPrompt(()=>{if(intended.startsWith('#/'))go(intended.slice(2));else if(!intended)go(document.body.dataset.start||'home');else location.hash=intended;});return true;
}
window.addEventListener('hashchange',ui211HashChange,true);
for(const name of ['input','change'])document.addEventListener(name,ev=>{
 const state=UI211_MODAL;if(state?.form&&ev.target.closest?.('#modal-form')===state.form){state.edited=true;ui211ModalStatus(state,ui211ModalDirty()?'Unsaved changes':'No unsaved changes');}
 if(ev.target.getAttribute?.('aria-invalid')==='true')ev.target.removeAttribute?.('aria-invalid');
});
window.addEventListener('beforeunload',ev=>{if(UI211_MODAL?.busy||ui211ModalDirty()){ev.preventDefault();ev.returnValue='';}});

// Keep the original list-memory model; provide one obvious way to clear its filters.
const ui211PriorFilteredRows=filteredRows;
filteredRows=function(){ui211PriorFilteredRows();const reset=$('[data-action="ui211-clear-filters"]');if(reset)reset.hidden=!Object.values(filters).some(Boolean);const count=$('#filter-count');if(count){count.setAttribute?.('role','status');count.setAttribute?.('aria-live','polite');}};
actions['ui211-clear-filters']=()=>{filters={search:'',unit:'',matrix:''};for(const node of $$('[data-filter],[data-unit-filter]'))node.value='';$$('.heat-cell.selected').forEach(x=>x.classList.remove('selected'));filteredRows();$('[data-filter]')?.focus?.();};

const UI211_WORKSPACES=[['My work','work','Reviews, corrections and assigned actions'],['Inbox','inbox','Mentions and program notifications'],['Programs','programs','Requirements and named assessments'],['Forms','forms','Questionnaires and assignments'],['Documents','documents','Files, versions and reviews'],['Vendors','vendors','Service engagements and due diligence'],['Vendor onboarding','vendors?tab=onboarding','Draft intakes and onboarding dossiers'],['Risk register','risks','Risk ownership and treatments'],['Loss register','losses','Events, reviews and recovery'],['Branch KRI returns','kri-returns','Monthly operational returns'],['Indicators','indicators','Continuous observations'],['Controls','controls','Control design and tests'],['RCSA','rcsa','Risk and control assessments'],['Privacy','privacy','Processing, impact assessments and cases'],['Reports','reports','Insights and board packs'],['Roles & permissions','roles','Access and role changes'],['Escalation matrix','escalation-matrix','Routing and escalation policy'],['Framework library','frameworks','Adopt a framework'],['Integrations','integrations','Source checks and monitoring'],['Settings','settings','Workspace preferences and activity']];
function ui211SearchCatalog(){
 const rows=UI211_WORKSPACES.map(([name,route,note])=>({name,route,note,kind:'Workspace',id:''}));
 for(const type of ['program','engagement','risk','loss','control','document','activity','dpia','privacyCase','kriReturn']){
  const config=REC_TYPES[type];if(!config)continue;
  for(const record of config.collection().filter(x=>recInScope(type,x)&&(type!=='document'||docVisible(x)))){
   const route=recUrl(type,record.id);if(!route||route.includes('undefined'))continue;
   rows.push({name:recName(type,record),kind:config.label,id:record.id,route,note:record.owner||(record.ownerId?d210Name(record.ownerId):'')});
  }
 }
 return rows.map(x=>({...x,search:(x.name+' '+x.kind+' '+x.id+' '+x.note).toLowerCase()}));
}
function ui211SearchRows(query,catalog=ui211SearchCatalog()){
 const q=String(query||'').trim().toLowerCase(),tokens=q.split(/\s+/).filter(Boolean);
 return catalog.filter(x=>tokens.every(t=>x.search.includes(t))).map((x,index)=>({...x,rank:!q?(x.kind==='Workspace'?0:3):x.id.toLowerCase()===q||x.name.toLowerCase()===q?0:x.name.toLowerCase().startsWith(q)?1:x.kind==='Workspace'?3:2,index})).sort((a,b)=>a.rank-b.rank||a.index-b.index).slice(0,18);
}
let UI211_SEARCH=[],ui211SearchIndex=-1;
globalResults=function(query){
 const rows=ui211SearchRows(query,UI211_SEARCH.length?UI211_SEARCH:ui211SearchCatalog()),box=$('#global-results');if(!box)return;
 ui211SearchIndex=-1;
 box.innerHTML=rows.length?`<div class="ui211-search-meta">${rows.length} ${query?'matching results':'starting points'} · ${e(scopeLabel())}</div>${rows.map((r,i)=>`<a class="ui211-search-result" id="ui211-search-${i}" href="#/${e(r.route)}" data-ui211-search="${i}"><span>${icon(r.kind==='Workspace'?'layers':'file')}</span><span><strong>${e(r.name)}</strong><small>${e(r.kind)}${r.id?' · '+e(r.id):''}${r.note?' · '+e(r.note):''}</small></span>${icon('arrow')}</a>`).join('')}`:empty('No matching records','Try the record ID, a shorter name or another location.');
};
searchDialog=function(){UI211_SEARCH=ui211SearchCatalog();openDialog('Find a workspace or record','Search names, record IDs and owners in the selected location.',`<label class="ui211-search-input"><span class="sr-only">Search workspaces and scoped records</span>${icon('search')}<input id="global-query" placeholder="Search by name, ID or owner…" aria-controls="global-results" autocomplete="off" autofocus></label><p class="ui211-search-hint">↑ ↓ to choose · Enter to open · Esc to close</p><div id="global-results" role="region" aria-label="Search results"></div>`,'',null,'ui211-search');globalResults('');$('#global-query')?.focus?.();};
document.addEventListener('keydown',ev=>{
 if(ev.isComposing||ev.target.id!=='global-query'||!['ArrowDown','ArrowUp','Enter'].includes(ev.key))return;
 const links=$$('[data-ui211-search]',$('#global-results'));if(!links.length)return;ev.preventDefault();
 if(ev.key==='Enter'){if(ui211SearchIndex<0)ui211SearchIndex=0;const href=links[ui211SearchIndex].getAttribute('href');closeDialog();go(href.slice(2));return;}
 ui211SearchIndex=ui211SearchIndex<0?(ev.key==='ArrowDown'?0:links.length-1):(ui211SearchIndex+(ev.key==='ArrowDown'?1:-1)+links.length)%links.length;
 links.forEach((x,i)=>{x.classList.toggle('selected',i===ui211SearchIndex);if(i===ui211SearchIndex)x.setAttribute('aria-current','true');else x.removeAttribute('aria-current');});
 links[ui211SearchIndex].scrollIntoView?.({block:'nearest'});
});

let ui211LastFocusRoute='';
function ui211Enhance(){
 const root=$('#main-content');if(!root)return;
 const toolbar=$('[data-filter]',root)?.closest?.('.toolbar');if(toolbar&&!$('[data-action="ui211-clear-filters"]',toolbar))toolbar.insertAdjacentHTML('beforeend',btn('Clear filters','ui211-clear-filters','hidden','quiet small'));filteredRows();
 for(const wrap of $$('.table-wrap',root)){if(wrap.scrollWidth>wrap.clientWidth+2){wrap.setAttribute?.('tabindex','0');wrap.setAttribute?.('role','region');wrap.setAttribute?.('aria-label',wrap.closest?.('section')?.querySelector?.('h2')?.textContent||'Scrollable records');}else{wrap.removeAttribute?.('tabindex');wrap.removeAttribute?.('role');wrap.removeAttribute?.('aria-label');}}
 const selector=current.params?.get('focus'),key=ui215FocusContext(),changed=key!==ui211LastFocusRoute;
 ui211LastFocusRoute=key;
 if(selector&&changed)requestAnimationFrame(()=>{if(key===ui215FocusContext()&&root.isConnected!==false)ui211Focus(selector,root);});
}
const ui211PriorRender=render;render=function(){ui211PriorRender();if(!current.blocked)ui211Enhance();};
window.CLEAR_SIGHT_INTERACTIONS=Object.freeze({version:'2.11',route:ui211Route,issues:ui211Issues,focus:ui211Focus,next:ui211Next,search:ui211SearchRows,submitModal:ui211SubmitModal,modalDirty:ui211ModalDirty});
