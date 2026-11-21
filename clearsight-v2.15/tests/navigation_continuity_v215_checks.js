/* Actual shared handlers with controlled DOM nodes. Native focus, history and
 * keyboard acceptance require a browser; this suite does not claim them. */
'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
async function test(name,fn){try{await fn();results.push({name,status:'PASS'});console.log('PASS '+name);}catch(error){results.push({name,status:'FAIL',error:error.stack});console.error('FAIL '+name+'\n'+error.stack);}}
function node(h,tag='BUTTON',attributes={}){
 const attrs=new Map(Object.entries(attributes)),classes=new Set();
 const n={tagName:tag,isConnected:true,disabled:false,hidden:false,inert:false,children:[],parent:null,dataset:{},textContent:'',innerHTML:'',
  get attributes(){return [...attrs].map(([name,value])=>({name,value}));},
  getAttribute:k=>attrs.has(k)?attrs.get(k):null,setAttribute:(k,v)=>attrs.set(k,String(v)),removeAttribute:k=>attrs.delete(k),
  classList:{add:(...keys)=>keys.forEach(k=>classes.add(k)),remove:(...keys)=>keys.forEach(k=>classes.delete(k)),contains:k=>classes.has(k),toggle:(k,on)=>on?classes.add(k):classes.delete(k)},
  focus(options){h.context.document.activeElement=n;n.focusOptions=options;n.focusCalls=(n.focusCalls||0)+1;},getClientRects:()=>[{}],
  querySelector(q){return q==='h1'?n.children.find(c=>c.tagName==='H1')||null:null;},querySelectorAll(q){return ['a[href],button,input,select,textarea,[tabindex]','[data-ui211-search]'].includes(q)?n.children:[];},
  contains(other){return other===n||n.children.includes(other);},
  closest(q){if(q==='form')return n.form||null;if(q.includes('#dialog')&&n.inDialog)return h.elements.get('#dialog');return null;},
  scrollIntoView(){},insertAdjacentHTML(position,html){n.innerHTML+=html;},prepend(){},append(){}};
 for(const [name,value] of attrs)if(name.startsWith('data-'))n.dataset[name.slice(5).replace(/-([a-z])/g,(_,c)=>c.toUpperCase())]=value;
 return n;
}
function page(h,controls=[]){
 const main=node(h,'MAIN',{id:'main-content'}),heading=node(h,'H1');main.children=[heading,...controls];h.elements.set('#main-content',main);return {main,heading};
}
function event(key='Escape'){return {key,prevented:false,stopped:false,preventDefault(){this.prevented=true;},stopImmediatePropagation(){this.stopped=true;}};}

(async()=>{
await test('A completed modal save returns to the same record control after its workspace is rebuilt',()=>{
 const h=createRuntime();h.navigate('my-workspace');const original=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'}),{main}=page(h,[original]);original.focus();
 h.run('openDialog("Inspect A","","", "Save",()=>true)');
 const other=node(h,'BUTTON',{'data-action':'inspect','data-id':'B'}),replacement=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'});main.children=[other,replacement];original.isConnected=false;
 h.run('closeDialog()');assert.equal(h.context.document.activeElement,replacement);assert.equal(other.focusCalls,undefined);assert.equal(replacement.focusOptions.preventScroll,true);assert.equal(h.run('UI211_MODAL'),null);
});

await test('Removed or ambiguous controls fall back to the page heading; another actor never inherits the focus target',()=>{
 for(const mode of ['removed','duplicate','actor','hidden']){
  const h=createRuntime();h.navigate('my-workspace');const original=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'}),{main,heading}=page(h,[original]);original.focus();h.run('openDialog("A","","", "Save",()=>true)');
  const a=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'});main.children=[heading,...(mode==='removed'?[]:mode==='duplicate'?[a,node(h,'BUTTON',{'data-action':'inspect','data-id':'A'})]:[a])];original.isConnected=false;
  if(mode==='actor')h.run("S.governance.actorId=recActor().id==='USR-01'?'USR-03':'USR-01';");
  if(mode==='hidden'){original.isConnected=true;original.getClientRects=()=>[];a.getClientRects=()=>[];}
  h.run('closeDialog()');assert.equal(h.context.document.activeElement,heading);assert.equal(a.focusCalls,undefined);
 }
});

await test('A continuation dialog retains the original workspace return target',()=>{
 const h=createRuntime();h.navigate('my-workspace');const trigger=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'}),{main}=page(h,[trigger]);trigger.focus();h.run('openDialog("First","","", "Continue",()=>false)');
 const modalButton=node(h);modalButton.inDialog=true;modalButton.focus();h.run('openDialog("Second","","", "Apply",()=>true)');
 const replacement=node(h,'BUTTON',{'data-action':'inspect','data-id':'A'});main.children=[replacement];trigger.isConnected=false;modalButton.isConnected=false;
 h.run('closeDialog()');assert.equal(h.context.document.activeElement,replacement);
});

await test('Escape uses the same close lifecycle and retains dirty entries until discard',()=>{
 const h=createRuntime();h.navigate('my-workspace');const trigger=node(h,'BUTTON',{'data-action':'inspect'});page(h,[trigger]);trigger.focus();h.run('openDialog("Review","","", "Save",()=>true)');
 const clean=event();h.context.ev=clean;h.run('ui215CancelModal(ev)');assert(clean.prevented&&clean.stopped);assert.equal(h.elements.get('#dialog').open,false);assert.equal(h.run('UI211_MODAL'),null);assert.equal(h.run('modalSave'),null);assert.equal(h.context.document.activeElement,trigger);
 const field={name:'rationale',type:'text',value:'Existing rationale'},form={elements:[field],dataset:{},querySelector(){return null;}};h.elements.set('#modal-form',form);h.run('openDialog("Review","","", "Save",()=>true)');field.value='Still being edited';h.run('UI211_MODAL.edited=true');h.context.ev=event();h.run('ui215CancelModal(ev)');assert.equal(h.elements.get('#dialog').open,true);assert.equal(field.value,'Still being edited');assert.equal(h.run('typeof UI211_MODAL.pending'),'function');
 h.run('actions["ui211-discard"]()');assert.equal(h.elements.get('#dialog').open,false);assert.equal(h.run('UI211_MODAL'),null);
});

await test('Back-style navigation cannot replace source context while an asynchronous modal save is in flight',async()=>{
 const h=createRuntime();h.navigate('roles');const main=h.elements.get('#main-content'),before=main.innerHTML;
 const form={elements:[],dataset:{},setAttribute(){},querySelector(){return null;},querySelectorAll(){return [];}};h.elements.set('#modal-form',form);h.context.testForm=form;
 h.context.history={state:{known:'retained'},replaceState(state,title,url){assert.equal(state.known,'retained');h.context.location.hash=url;}};
 h.run('var completeSave,saveCalls=0;openDialog("Access change","","","Save",()=>{saveCalls++;return new Promise(resolve=>completeSave=resolve);})');
 const saving=h.run('ui211SubmitModal(testForm)');
 for(const destination of ['#/work','']){
  h.context.location.hash=destination;h.context.ev=event();assert.equal(h.run('ui211HashChange(ev)'),true);assert.equal(h.context.location.hash,'#/roles');assert(h.context.ev.stopped);assert.equal(h.elements.get('#dialog').open,true);assert.equal(main.innerHTML,before);assert.equal(h.run('UI211_MODAL.pending'),null);assert.equal(h.run('saveCalls'),1);
 }
 h.run('completeSave(true)');await saving;assert.equal(h.elements.get('#dialog').open,false);assert.equal(h.context.location.hash,'#/roles');
});

await test('Returning to an original no-hash entry protects dirty edits and uses its default route only after discard',()=>{
 const h=createRuntime();h.navigate('roles');h.context.document.body.dataset.start='my-workspace';
 const field={name:'rationale',type:'text',value:'Original'},form={elements:[field],dataset:{},querySelector(){return null;}};h.elements.set('#modal-form',form);
 h.context.history={state:null,replaceState(state,title,url){h.context.location.hash=url;}};
 h.run('openDialog("Edit","","","Save",()=>true)');field.value='Uncommitted';h.run('UI211_MODAL.edited=true');h.context.location.hash='';h.context.ev=event();
 assert.equal(h.run('ui211HashChange(ev)'),true);assert.equal(h.context.location.hash,'#/roles');assert.equal(field.value,'Uncommitted');
 h.run('actions["ui211-discard"]()');assert.equal(h.context.location.hash,'/my-workspace');assert.equal(h.elements.get('#dialog').open,false);
});

await test('Search starts upward at the last result, downward at the first, and leaves IME composition alone',()=>{
 const h=createRuntime();h.navigate('my-workspace');const box=node(h,'DIV'),rows=[0,1,2].map(i=>node(h,'A',{href:'#/risk?id=R'+i}));box.children=rows;h.elements.set('#global-results',box);
 const handler=h.events.keydown.find(fn=>fn.toString().includes('ui211SearchIndex'));assert(handler);
 const key=k=>Object.assign(event(k),{target:{id:'global-query'}});
 h.run('ui211SearchIndex=-1');handler(key('ArrowUp'));assert(rows[2].classList.contains('selected'));assert.equal(h.run('ui211SearchIndex'),2);
 h.run('ui211SearchIndex=-1');handler(key('ArrowDown'));assert(rows[0].classList.contains('selected'));
 const before=h.context.location.hash,composing=key('Enter');composing.isComposing=true;handler(composing);assert.equal(h.context.location.hash,before);assert.equal(composing.prevented,false);
 const announce=h.events.keydown.find(fn=>fn.toString().includes('pu214SearchStatus'));assert(announce);h.run('var announcements=0;pu214SearchStatus=()=>announcements++;');const candidate=key('ArrowDown');candidate.isComposing=true;announce(candidate);assert.equal(h.run('announcements'),0);
});

await test('Route changes focus the new page without changing its remembered scroll or taking focus from an explicit workflow',()=>{
 const h=createRuntime();h.navigate('my-workspace');const frames=[];h.context.requestAnimationFrame=fn=>frames.push(fn);const {heading}=page(h);h.context.document.activeElement=h.context.document.body;
 const scroll=[];h.context.scrollTo=(x,y)=>scroll.push([x,y]);h.navigate('risks');while(frames.length)frames.shift()();assert.equal(h.context.document.activeElement,heading);assert.equal(heading.focusOptions.preventScroll,true);
 const explicit=node(h,'INPUT',{id:'intentional-field'});h.navigate('controls');explicit.focus();while(frames.length)frames.shift()();assert.equal(h.context.document.activeElement,explicit);
});

await test('An inline render restores the unique original field and a stale deferred render never steals focus',()=>{
 const h=createRuntime();h.navigate('my-workspace');const frames=[];h.context.requestAnimationFrame=fn=>frames.push(fn);
 const original=node(h,'INPUT',{'data-answer':'Q-01',name:'answer'}),{main}=page(h,[original]);original.focus();h.run('render()');
 const replacement=node(h,'INPUT',{'data-answer':'Q-01',name:'answer'});main.children=[replacement];original.isConnected=false;h.context.document.activeElement=h.context.document.body;
 while(frames.length)frames.shift()();assert.equal(h.context.document.activeElement,replacement);
 h.navigate('risks');h.navigate('controls');const explicit=node(h,'BUTTON',{id:'focused-after-navigation'});explicit.focus();while(frames.length)frames.shift()();assert.equal(h.context.document.activeElement,explicit);
});

await test('Correction deep links focus again after revisiting and ignore callbacks for a route already left',()=>{
 const h=createRuntime();h.navigate('my-workspace');const frames=[];h.context.requestAnimationFrame=fn=>frames.push(fn);h.run('var focusHits=0;ui211Focus=()=>{focusHits++;return true;}');
 h.navigate('my-workspace?focus=%5Bname%3D%22answer%22%5D');while(frames.length)frames.shift()();assert.equal(h.run('focusHits'),1);
 h.navigate('risks');while(frames.length)frames.shift()();h.navigate('my-workspace?focus=%5Bname%3D%22answer%22%5D');while(frames.length)frames.shift()();assert.equal(h.run('focusHits'),2);
 h.navigate('risks');h.navigate('my-workspace?focus=%5Bname%3D%22answer%22%5D');h.navigate('controls');while(frames.length)frames.shift()();assert.equal(h.run('focusHits'),2);
});

const report={suite:'V2.15 navigation, modal and keyboard continuity',runtime:'Actual source with controlled DOM/browser shells',browserLayoutValidated:false,results};
fs.writeFileSync(path.join(root,'tests/navigation-continuity-v215-results.json'),JSON.stringify(report,null,2)+'\n');
console.log(results.filter(r=>r.status==='PASS').length+'/'+results.length+' navigation checks passed.');if(results.some(r=>r.status==='FAIL'))process.exitCode=1;
})();
