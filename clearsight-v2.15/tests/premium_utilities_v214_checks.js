/* Actual utility interaction logic with controlled DOM nodes. These checks do
 * not certify native focus, inert support, screen readers or rendered layout. */
'use strict';
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function test(name,fn){try{fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function node(h,attributes={}){
 const attrs=new Map(Object.entries(attributes)),classes=new Set();
 return {attrs,hidden:false,disabled:false,inert:false,isConnected:true,dataset:{},textContent:'',innerHTML:'',
  setAttribute:(k,v)=>attrs.set(k,String(v)),getAttribute:k=>attrs.has(k)?attrs.get(k):null,removeAttribute:k=>attrs.delete(k),
  classList:{add:(...names)=>names.forEach(n=>classes.add(n)),remove:(...names)=>names.forEach(n=>classes.delete(n)),contains:n=>classes.has(n),toggle:(n,on)=>on?classes.add(n):classes.delete(n)},
  focus(){h.context.document.activeElement=this;},getClientRects:()=>[{}],querySelector:()=>null,querySelectorAll:()=>[],insertAdjacentHTML(pos,text){this.innerHTML+=text;},contains:()=>false};
}
function navigation(){
 const h=createRuntime();h.navigate('my-workspace');h.context.innerWidth=390;
 const nav=node(h),main=h.elements.get('#main-content'),header=node(h),opener=node(h),close=node(h),active=node(h,{href:'#/my-workspace','aria-current':'page'}),last=node(h,{href:'#/my-access'});
 main.inert=false;
 const children=[close,active,last];nav.querySelectorAll=()=>children;nav.querySelector=q=>q==='a[aria-current="page"]'?active:null;nav.contains=value=>value===nav||children.includes(value);
 h.elements.set('.sidebar',nav);h.elements.set('.topbar',header);h.elements.set('.topbar [data-action="menu"]',opener);
 h.context.document.body.classList=node(h).classList;h.elements.get('#app').classList=node(h).classList;
 return {h,nav,main,header,opener,close,active,last};
}
function key(name,shiftKey=false){return {key:name,shiftKey,prevented:false,stopped:false,preventDefault(){this.prevented=true;},stopImmediatePropagation(){this.stopped=true;}};}

test('Mobile navigation contains focus, exposes its state and restores the original background',()=>{
 const {h,nav,main,header,opener,close,active,last}=navigation(),before=h.run('recStable(S)');
 assert.equal(h.run('pu214OpenNav()'),true);assert.equal(h.context.document.activeElement,active);
 assert.equal(nav.getAttribute('role'),'dialog');assert.equal(nav.getAttribute('aria-modal'),'true');assert.equal(opener.getAttribute('aria-expanded'),'true');assert.equal(main.inert,true);assert.equal(header.inert,true);
 last.focus();const forward=key('Tab');h.context.ev=forward;h.run('pu214NavKeydown(ev)');assert.equal(forward.prevented,true);assert.equal(h.context.document.activeElement,close);
 const backward=key('Tab',true);h.context.ev=backward;h.run('pu214NavKeydown(ev)');assert.equal(h.context.document.activeElement,last);
 const escape=key('Escape');h.context.ev=escape;h.run('pu214NavKeydown(ev)');assert.equal(escape.stopped,true);assert.equal(main.inert,false);assert.equal(header.inert,false);assert.equal(nav.getAttribute('role'),null);assert.equal(opener.getAttribute('aria-expanded'),'false');assert.equal(h.context.document.activeElement,opener);assert.equal(h.run('recStable(S)'),before);
});

test('Navigation links keep their destination while releasing the drawer without a form render',()=>{
 const {h,nav,main,last}=navigation();h.run('window.PU214_RENDERS=0;render=()=>PU214_RENDERS++;pu214OpenNav();');const before=main.innerHTML,href=last.getAttribute('href');
 h.context.ev={target:{closest:()=>last}};h.run('pu214NavClick(ev)');assert.equal(h.run('CLEAR_SIGHT_PREMIUM_UTILITIES.drawerOpen()'),false);assert.equal(last.getAttribute('href'),href);assert.equal(nav.getAttribute('aria-modal'),null);assert.equal(main.inert,false);assert.equal(main.innerHTML,before);assert.equal(h.run('PU214_RENDERS'),0);
});

test('Desktop resize restores pre-existing inert/role attributes and does not focus a hidden opener',()=>{
 const {h,nav,main,header,opener}=navigation();main.inert=true;nav.setAttribute('role','navigation');nav.setAttribute('tabindex','0');let focused=0;opener.focus=()=>focused++;
 h.run('pu214OpenNav()');h.context.innerWidth=1440;h.run('pu214NavResize()');assert.equal(main.inert,true);assert.equal(header.inert,false);assert.equal(nav.getAttribute('role'),'navigation');assert.equal(nav.getAttribute('tabindex'),'0');assert.equal(focused,0);assert.equal(h.run('pu214OpenNav()'),false);
});

test('Mobile navigation cannot replace an open dialog and redirects stray focus while active',()=>{
 const {h,main,close}=navigation();h.elements.get('#dialog').open=true;h.elements.get('#dialog').innerHTML='Pending source decision';assert.equal(h.run('pu214OpenNav()'),false);assert.equal(main.inert,false);assert.equal(h.elements.get('#dialog').innerHTML,'Pending source decision');
 h.elements.get('#dialog').open=false;h.run('pu214OpenNav()');h.context.ev={target:main};h.run('pu214NavFocus(ev)');assert.equal(h.context.document.activeElement,close);h.run('pu214CloseNav()');
});

test('Search opened from the drawer retains a visible focus-return target',()=>{
 const {h,opener,main,header}=navigation();h.run('pu214OpenNav()');const event=key('k');event.ctrlKey=true;h.context.ev=event;h.run('pu214NavKeydown(ev)');
 assert.equal(event.stopped,true);assert.equal(h.run('CLEAR_SIGHT_PREMIUM_UTILITIES.drawerOpen()'),false);assert.equal(h.run('lastFocus'),opener);assert.equal(main.inert,false);assert.equal(header.inert,false);assert.equal(h.elements.get('#dialog').open,true);
 assert.match(h.elements.get('#dialog').innerHTML,/Find a workspace or record/);
});

function dirtySearch(){
 const h=createRuntime();h.navigate('my-workspace');
 const field=node(h);field.name='rationale';field.type='text';field.value='Unsaved independent decision';
 const foot=node(h),form=node(h);form.elements=[field];form.querySelector=q=>q==='.dialog-foot'?foot:q.startsWith('input')?field:null;
 h.elements.set('#modal-form',form);h.elements.set('#global-results',node(h));h.elements.set('#global-query',node(h));
 h.context.reviewForm=form;h.run("UI211_MODAL={id:91,form:reviewForm,callback:()=>false,baseline:'previous values',edited:true,busy:false,pending:null};modalSave=UI211_MODAL.callback;");
 const dialog=h.elements.get('#dialog');dialog.open=true;dialog.innerHTML='Review dialog with unsaved rationale';
 return {h,field,form,foot,dialog};
}
test('Search protects an unsaved decision and follows the existing Keep editing / Discard choice',()=>{
 const {h,field,dialog}=dirtySearch(),before=h.elements.get('#main-content').innerHTML;
 assert.equal(h.run('searchDialog()'),false);assert.equal(dialog.innerHTML,'Review dialog with unsaved rationale');assert.equal(h.run('typeof UI211_MODAL.pending'),'function');
 h.run("actions['ui211-keep-editing']()");assert.equal(h.run('UI211_MODAL.pending'),null);assert.equal(field.value,'Unsaved independent decision');
 h.run("searchDialog();actions['ui211-discard']()");assert.match(dialog.innerHTML,/Find a workspace or record/);assert.equal(h.run('UI211_MODAL.callback'),null);assert.equal(h.elements.get('#main-content').innerHTML,before);
});

test('Search never replaces a dialog while its asynchronous save is pending',()=>{
 const {h,dialog}=dirtySearch();h.run('UI211_MODAL.busy=true;');const before=dialog.innerHTML;
 assert.equal(h.run('searchDialog()'),false);assert.equal(dialog.innerHTML,before);assert.equal(h.run('UI211_MODAL.pending'),null);assert.equal(h.run('UI211_MODAL.busy'),true);
});

test('Search presentation retains exact role-scoped results, record URLs and canonical state',()=>{
 const h=createRuntime();h.run("S.governance.actorId='USR-01';");h.navigate('my-workspace?org=B-IK');const box=node(h);h.elements.set('#global-results',box);
 for(const query of ['','KRI','risk','USR-03','report']){
  h.context.query=query;h.run('pu214PriorGlobalResults(query)');const before=box.innerHTML,state=h.run('recStable(S)');h.run('globalResults(query)');assert.equal(box.innerHTML,before);assert.equal(h.run('recStable(S)'),state);
 }
});

test('Search live context uses plain text and announces the selected existing result',()=>{
 const h=createRuntime(),box=node(h),status=node(h),one=node(h),two=node(h);one.classList.add('selected');
 one.querySelector=q=>q==='strong'?{textContent:'A <script> named record'}:q==='small'?{textContent:'Risk · RSK-01'}:null;
 box.querySelectorAll=()=>[one,two];box.querySelector=()=>({textContent:'2 matching results · Ikeja branch'});
 h.elements.set('#global-results',box);h.elements.set('#pu214-search-status',status);
 h.run('pu214SearchStatus()');assert.equal(status.textContent,'2 matching results · Ikeja branch');
 h.run('pu214SearchStatus(true)');assert.equal(status.textContent,'A <script> named record · Risk · RSK-01 · 1 of 2');assert.equal(status.innerHTML,'');
});

test('Settings reading order moves the original nodes and route markers reset without state changes',()=>{
 const h=createRuntime();h.navigate('my-workspace');const main=h.elements.get('#main-content'),names=['admin','head','tabs','grid','preferences','footer'],items=Object.fromEntries(names.map(name=>[name,{name}]));let order=names.map(name=>items[name]);
 function move(item,index){order=order.filter(n=>n!==item);order.splice(index,0,item);}
 for(const item of Object.values(items))item.after=next=>{order=order.filter(n=>n!==next);order.splice(order.indexOf(item)+1,0,next);};
 main.prepend=item=>move(item,0);main.querySelector=q=>({':scope > .page-head':items.head,':scope > .tabs':items.tabs,':scope > .gv-settings-entry':items.admin,':scope > .pd213-settings':items.preferences}[q]||null);
 const before=h.run('recStable(S)');h.run("current={name:'settings',params:new URLSearchParams()};pu214Page();");assert.deepEqual(order.map(n=>n.name),['head','tabs','preferences','admin','grid','footer']);assert.equal(new Set(order).size,6);assert.equal(main.dataset.pu214,'settings');
 h.run("current={name:'risks',params:new URLSearchParams()};pu214Page();");assert.equal(main.dataset.pu214,undefined);assert.equal(h.run('recStable(S)'),before);
});

test('Document folder selection is exposed without changing availability or document state',()=>{
 const h=createRuntime();h.navigate('my-workspace');const main=h.elements.get('#main-content'),active=node(h),inactive=node(h),bulk=node(h);active.classList.add('active');inactive.disabled=true;
 main.querySelectorAll=q=>q.startsWith('.doc-folders')?[active,inactive]:q==='.doc-bulk'?[bulk]:[];
 const before=h.run('recStable(S)');h.run("current={name:'documents',params:new URLSearchParams()};pu214Page();");assert.equal(active.getAttribute('aria-pressed'),'true');assert.equal(inactive.getAttribute('aria-pressed'),'false');assert.equal(inactive.disabled,true);assert.equal(bulk.getAttribute('aria-label'),'Actions for selected documents');assert.equal(h.run('recStable(S)'),before);
});

const report={suite:'V2.14 shared utility interactions',runtime:'Actual source with controlled DOM/browser shells',browserLayoutValidated:false,results};
fs.writeFileSync(path.join(root,'tests/premium-utilities-v214-results.json'),JSON.stringify(report,null,2)+'\n');console.log(`${results.filter(r=>r.status==='PASS').length}/${results.length} utility interaction checks passed.`);if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
