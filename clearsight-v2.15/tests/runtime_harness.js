/* Actual-source JavaScript harness. Only the DOM/browser shell is substituted.
   This validates calculations, routing and commands, not layout or browser APIs. */
'use strict';
const fs=require('fs'),vm=require('vm'),path=require('path');
const root=path.resolve(__dirname,'..');
function createRuntime({initialStorage=[]}={}){
 const events={},elements=new Map(),storage=new Map(initialStorage),errors=[];
 const element=()=>({innerHTML:'',textContent:'',hidden:false,open:false,value:'',dataset:{},classList:{add(){},remove(){},toggle(){},contains(){return false}},addEventListener(){},setAttribute(){},removeAttribute(){},insertAdjacentHTML(position,html){this.innerHTML+=html},querySelector(){return null},querySelectorAll(){return []},scrollIntoView(){},focus(){},showModal(){this.open=true},close(){this.open=false},append(){},prepend(){},remove(){}});
 for(const id of ['#app','#main-content','#dialog','#toast','#ix-return-error','#ix-cycle-error','#modal-error','.page-head p'])elements.set(id,element());
 const doc={body:{dataset:{start:'kri-returns'},classList:{toggle(){}}},documentElement:{dataset:{}},activeElement:null,title:'',addEventListener(type,fn){(events[type]??=[]).push(fn)},querySelector(q){return elements.get(q)||null},querySelectorAll(){return []},getElementById(id){return elements.get('#'+id)||null},createElement:element};
 const context={console,TextEncoder,TextDecoder,URL,URLSearchParams,Blob,Intl,Date,JSON,Map,Set,Math,Number,String,Object,Array,Error,Promise,Uint8Array,ArrayBuffer,DataView,DecompressionStream,ReadableStream,atob,btoa,crypto:require('crypto').webcrypto,document:doc,location:{hash:''},localStorage:{getItem:k=>storage.get(k)||null,setItem:(k,v)=>storage.set(k,String(v)),removeItem:k=>storage.delete(k)},setTimeout:()=>0,clearTimeout(){},requestAnimationFrame(){},scrollTo(){},innerWidth:1440,innerHeight:1000,scrollY:0,addEventListener(){},print(){},FormData:class {constructor(obj){this.obj=obj||{}}[Symbol.iterator](){return Object.entries(this.obj)[Symbol.iterator]()}}};
 context.window=context;context.window.scrollTo=()=>{};vm.createContext(context);
 const files=['seed','app','forms','experience','journeys','insights','records','document-store','documents','usability','copilot','kri-schema','branch-kri','governance','inbox','board-reports','loss-schema','loss-register','workflow-completion','flow-polish','loss-reporting','assessment-tracking','vendor-onboarding','documents-unified','assessment-reporting','interaction-core','assurance-interactions','governance-interactions','operations-interactions','privacy-interactions','role-experience','scoped-analytics','document-management','external-assessments','premium-foundation','premium-workspaces','premium-forms','premium-reporting','premium-programs','premium-risk','premium-privacy','premium-utilities'];
 for(const file of files.filter(file=>fs.existsSync(path.join(root,'assets',file+'.js'))))vm.runInContext(fs.readFileSync(path.join(root,'assets',file+'.js'),'utf8'),context,{filename:file+'.js'});
 const run=source=>vm.runInContext(source,context);
 run('recordsInit();docSeed();ixInit();gvInit();inboxInit();lossInit();');
 function navigate(route){context.location.hash='#/'+route;run('render()');return elements.get('#main-content').innerHTML;}
 return {context,run,navigate,elements,events,storage,errors};
}
module.exports={createRuntime,root};
