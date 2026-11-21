/* V2.15 shared utility interactions. Domain sources and action permissions
 * remain authoritative. No polling, extra data store or startup render. */
'use strict';

let PU214_NAV=null;
function pu214Mobile(){return window.matchMedia?window.matchMedia('(max-width: 767px)').matches:window.innerWidth<=767;}
function pu214RestoreAttribute(node,name,value){if(value===null||value===undefined)node.removeAttribute(name);else node.setAttribute(name,value);}
function pu214NavFocusables(nav){
 return Array.from(nav.querySelectorAll('a[href],button:not([disabled]),[tabindex="0"]')).filter(node=>!node.hidden&&!node.disabled&&(!node.getClientRects||node.getClientRects().length));
}
function pu214CloseNav(restoreFocus=true){
 const state=PU214_NAV;if(!state)return false;
 PU214_NAV=null;
 for(const item of state.inert)item.node.inert=item.value;
 pu214RestoreAttribute(state.nav,'role',state.role);
 pu214RestoreAttribute(state.nav,'aria-modal',state.modal);
 pu214RestoreAttribute(state.nav,'tabindex',state.tabindex);
 state.opener?.setAttribute('aria-expanded','false');
 $('#app')?.classList.remove('nav-open');
 document.body.classList.remove('pu214-nav-lock');
 if(restoreFocus&&state.opener?.isConnected!==false&&pu214Mobile())state.opener?.focus({preventScroll:true});
 return true;
}
function pu214OpenNav(){
 if(PU214_NAV||!pu214Mobile()||$('#dialog')?.open)return false;
 const nav=$('.sidebar'),opener=$('.topbar [data-action="menu"]');if(!nav)return false;
 const inert=[$('#main-content'),$('.topbar')].filter(Boolean).map(node=>({node,value:!!node.inert}));
 PU214_NAV={nav,opener,inert,role:nav.getAttribute('role'),modal:nav.getAttribute('aria-modal'),tabindex:nav.getAttribute('tabindex')};
 $('#app')?.classList.add('nav-open');document.body.classList.add('pu214-nav-lock');
 nav.setAttribute('role','dialog');nav.setAttribute('aria-modal','true');nav.setAttribute('tabindex','-1');
 opener?.setAttribute('aria-expanded','true');for(const item of inert)item.node.inert=true;
 (nav.querySelector('a[aria-current="page"]')||pu214NavFocusables(nav)[0]||nav).focus();
 return true;
}
function pu214NavKeydown(ev){
 if(!PU214_NAV)return;
 if(ev.key==='Escape'){ev.preventDefault();ev.stopImmediatePropagation?.();pu214CloseNav();return;}
 if((ev.metaKey||ev.ctrlKey)&&String(ev.key).toLowerCase()==='k'){
  ev.preventDefault();ev.stopImmediatePropagation?.();pu214CloseNav();searchDialog();return;
 }
 if(ev.key!=='Tab')return;
 const nav=PU214_NAV.nav,items=pu214NavFocusables(nav),first=items[0]||nav,last=items.at(-1)||nav,active=document.activeElement;
 if(!nav.contains(active)||active===nav||(ev.shiftKey&&active===first)||(!ev.shiftKey&&active===last)){
  ev.preventDefault();(ev.shiftKey?last:first).focus();
 }
}
function pu214NavClick(ev){
 if(!PU214_NAV)return;
 const link=ev.target.closest?.('a[href]');
 if(link&&PU214_NAV.nav.contains(link)&&link.getAttribute('href')?.startsWith('#/'))pu214CloseNav(false);
}
function pu214NavFocus(ev){
 if(PU214_NAV&&!PU214_NAV.nav.contains(ev.target))(pu214NavFocusables(PU214_NAV.nav)[0]||PU214_NAV.nav).focus();
}
function pu214NavResize(){if(PU214_NAV&&!pu214Mobile())pu214CloseNav(false);}
actions.menu=pu214OpenNav;
const pu214PriorMenuClose=actions['menu-close'];
actions['menu-close']=()=>{if(!pu214CloseNav())pu214PriorMenuClose();};
document.addEventListener('keydown',pu214NavKeydown,true);
document.addEventListener('click',pu214NavClick,true);
document.addEventListener('focusin',pu214NavFocus,true);
window.addEventListener('resize',pu214NavResize,{passive:true});

function pu214Chrome(){
 const nav=$('.sidebar'),opener=$('.topbar [data-action="menu"]');
 if(nav){
  nav.id='pu214-primary-navigation';
  if(!nav.querySelector('.pu214-nav-close'))nav.insertAdjacentHTML('afterbegin',btn(icon('close'),'menu-close','aria-label="Close navigation"','icon-only quiet pu214-nav-close'));
 }
 if(opener){opener.setAttribute('aria-controls','pu214-primary-navigation');opener.setAttribute('aria-expanded','false');opener.setAttribute('aria-haspopup','dialog');}
 $('.mobile-shade')?.setAttribute('tabindex','-1');
}
const pu214PriorShell=shell;
shell=function(...args){pu214CloseNav(false);const result=pu214PriorShell.apply(this,args);pu214Chrome();return result;};

function pu214SearchStatus(selected=false){
 const box=$('#global-results'),status=$('#pu214-search-status');if(!box||!status)return;
 const rows=Array.from(box.querySelectorAll('[data-ui211-search]'));
 const currentRow=selected?rows.find(row=>row.classList.contains('selected')):null;
 if(currentRow){
  const name=currentRow.querySelector('strong')?.textContent||currentRow.textContent;
  const detail=currentRow.querySelector('small')?.textContent||'';
  status.textContent=name+(detail?' · '+detail:'')+' · '+(rows.indexOf(currentRow)+1)+' of '+rows.length;
 }else status.textContent=box.querySelector('.ui211-search-meta')?.textContent||'No results in the current scope. Try another name, record ID or owner.';
}
const pu214PriorGlobalResults=globalResults;
globalResults=function(...args){const result=pu214PriorGlobalResults.apply(this,args);pu214SearchStatus();return result;};
function pu214SearchEnhance(){
 const query=$('#global-query'),box=$('#global-results'),hint=$('.ui211-search-hint');if(!query||!box)return;
 query.setAttribute('type','search');
 if(hint){hint.id='pu214-search-help';query.setAttribute('aria-describedby','pu214-search-help');}
 if(!$('#pu214-search-status'))box.insertAdjacentHTML('beforebegin','<p id="pu214-search-status" class="sr-only" role="status" aria-live="polite" aria-atomic="true"></p>');
 pu214SearchStatus();
}
const pu214PriorSearchDialog=searchDialog;
searchDialog=function(...args){
 const open=()=>{pu214CloseNav();const result=pu214PriorSearchDialog.apply(this,args);pu214SearchEnhance();return result;};
 if($('#dialog')?.open&&UI211_MODAL?.busy){toast('Wait for the current operation to finish before opening search.');return false;}
 if($('#dialog')?.open&&UI211_MODAL?.callback&&ui211DiscardPrompt(open))return false;
 return open();
};
// The existing result selector runs first and retains its native link behavior.
document.addEventListener('keydown',ev=>{if(!ev.isComposing&&ev.target?.id==='global-query'&&['ArrowDown','ArrowUp'].includes(ev.key))pu214SearchStatus(true);});

const PU214_ROUTES=Object.freeze({settings:'settings',documents:'documents',evidence:'documents',approvals:'approvals'});
function pu214Page(){
 const main=$('#main-content');if(!main)return;
 const kind=PU214_ROUTES[current.name];if(kind)main.dataset.pu214=kind;else delete main.dataset.pu214;
 if(current.name==='settings'){
  // Move existing nodes, retaining focus targets and handlers. The H1 and tabs
  // precede administrative shortcuts in both visual and reading order.
  const head=main.querySelector(':scope > .page-head'),tabs=main.querySelector(':scope > .tabs'),admin=main.querySelector(':scope > .gv-settings-entry'),preferences=main.querySelector(':scope > .pd213-settings');
  if(head&&tabs){main.prepend(head);head.after(tabs);if(admin)tabs.after(admin);if(preferences)tabs.after(preferences);}
 }
 if(kind==='documents'){
  for(const folder of main.querySelectorAll('.doc-folders button[data-action="doc-folder"]'))folder.setAttribute('aria-pressed',String(folder.classList.contains('active')));
  for(const bar of main.querySelectorAll('.doc-bulk')){bar.setAttribute('role','group');bar.setAttribute('aria-label','Actions for selected documents');}
 }
}
const pu214PriorRender=render;
let pu215FocusSequence=0;
render=function(...args){
 const priorContext=ui215FocusContext(),hadPage=!!current.name,active=document.activeElement,reference=ui215FocusReference(active);
 const result=pu214PriorRender.apply(this,args);pu214Page();
 const context=ui215FocusContext(),sequence=++pu215FocusSequence;
 if(hadPage&&!$('#dialog')?.open&&!current.params?.get('focus')&&(priorContext!==context||reference))requestAnimationFrame(()=>{
  if(sequence!==pu215FocusSequence||context!==ui215FocusContext()||$('#dialog')?.open||PU214_NAV)return;
  // Respect focus deliberately placed by a workflow or by the user meanwhile.
  const focused=document.activeElement;
  if(focused&&focused!==active&&focused.isConnected!==false&&focused!==document.body&&focused!==document.documentElement)return;
  if(priorContext!==context){ui215PageFocus();return;}
  if(active?.isConnected!==false)return;
  const target=ui215FocusTarget(reference);if(target)target.focus({preventScroll:true});else if(reference.area==='#main-content')ui215PageFocus();
 });
 return result;
};
window.CLEAR_SIGHT_PREMIUM_UTILITIES=Object.freeze({version:'2.15',routes:PU214_ROUTES,refresh:pu214Page,drawerOpen:()=>!!PU214_NAV});
