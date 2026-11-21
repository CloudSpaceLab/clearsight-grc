/* Actual handlers and document readers; substituted DOM/history shell.
 * Native history, focus and visual acceptance are not claimed. */
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path');
const {createRuntime,root}=require('./runtime_harness');
const results=[];
function runtime(){const h=createRuntime();h.context.history={state:{retained:'route-state'},replaceState(state,title,url){this.state=state;h.context.location.hash=url;}};return h;}
async function test(name,fn){try{await fn(runtime());results.push({name,status:'PASS'});console.log('PASS',name);}catch(error){results.push({name,status:'FAIL',error:error.message});console.error('FAIL',name,error.stack);}}
function change(h,key,value){const fn=h.events.change.find(f=>f.toString().includes("el.dataset.docFilter!=='search'"));assert(fn,'Canonical document change handler');fn({target:{dataset:{docFilter:key},value,matches:s=>s==='[data-doc-filter]'}});}
function pair(h,before,after){h.context.beforeText=before;h.context.afterText=after;h.run(`var docNavBefore=docVersion('DOC-SAMPLE-KRI-V1');Object.assign(docNavBefore,{sampleText:beforeText,size:new TextEncoder().encode(beforeText).length,sha256:recHashText(beforeText)});var docNavAfter={...clone(docNavBefore),id:'DOC-NAV-V2',number:2,sampleText:afterText,size:new TextEncoder().encode(afterText).length,sha256:recHashText(afterText)};S.governance.versions.push(docNavAfter);`);}
function comparison(h){const el={dataset:{dm212Comparison:'DOC-SAMPLE-KRI-V1',current:'DOC-NAV-V2'},innerHTML:'',isConnected:true};h.context.docNavElement=el;return el;}
function wordBlob(paragraphs){
 const name=Buffer.from('word/document.xml'),data=Buffer.from('<w:document xmlns:w="x"><w:body>'+paragraphs.map(text=>'<w:p><w:r><w:t>'+text+'</w:t></w:r></w:p>').join('')+'</w:body></w:document>'),local=Buffer.alloc(30),central=Buffer.alloc(46),end=Buffer.alloc(22);
 local.writeUInt32LE(0x04034b50);local.writeUInt16LE(20,4);local.writeUInt32LE(data.length,18);local.writeUInt32LE(data.length,22);local.writeUInt16LE(name.length,26);
 central.writeUInt32LE(0x02014b50);central.writeUInt16LE(20,4);central.writeUInt16LE(20,6);central.writeUInt32LE(data.length,20);central.writeUInt32LE(data.length,24);central.writeUInt16LE(name.length,28);
 end.writeUInt32LE(0x06054b50);end.writeUInt16LE(1,8);end.writeUInt16LE(1,10);end.writeUInt32LE(central.length+name.length,12);end.writeUInt32LE(local.length+name.length+data.length,16);
 return new Blob([local,name,data,central,name,end]);
}

(async()=>{
await test('Folder and archive selections persist before render and retain route history metadata',h=>{
 h.navigate('documents?docFolder=F-COMPLIANCE&docType=txt&docStatus=Draft&docOwner=me');const state=h.run('recStable(S)');
 h.run("actions['doc-folder']({dataset:{id:'F-REPORTS'}})");assert.equal(h.run('DOC_FILTERS.folder'),'F-REPORTS');assert(h.context.location.hash.includes('docFolder=F-REPORTS'));
 h.run("actions['doc-archived']()");assert(h.run('DOC_FILTERS.archived'));assert.equal(h.run('DOC_FILTERS.folder'),'');assert.equal(h.run('DOC_FILTERS.status'),'');assert(h.context.location.hash.includes('docArchived=1'));assert(h.context.location.hash.includes('docOwner=me'));
 h.run("actions['doc-folder']({dataset:{id:''}})");assert(!h.run('DOC_FILTERS.archived'));assert(!h.context.location.hash.includes('docArchived'));assert.equal(h.context.history.state.retained,'route-state');assert.equal(h.run('recStable(S)'),state);
});
await test('Repeated format, state and sort changes survive route restoration; clean destinations reset omitted filters',h=>{
 h.navigate('documents?docType=txt&docStatus=Draft&docSort=name');
 for(const [key,value,param]of [['type','pdf','docType'],['status','Approved','docStatus'],['sort','updated','docSort']]){change(h,key,value);assert.equal(h.run('DOC_FILTERS.'+key),value);const p=new URLSearchParams(h.context.location.hash.split('?')[1]);assert.equal(p.get(param),key==='sort'?null:value);}
 change(h,'type','');assert.equal(h.run('DOC_FILTERS.type'),'');assert(!h.context.location.hash.includes('docType='));
 h.navigate('documents');assert.equal(h.run('JSON.stringify(DOC_FILTERS)'),JSON.stringify({search:'',folder:'',type:'',status:'',sort:'updated',archived:false}));
});
await test('Document titles pin the same exact version and filtered return destination as Preview',h=>{
 const html=h.navigate('documents?docOwner=me&docType=txt&docSort=name'),title=html.match(/class="doc-name" href="#\/([^\"]+)"/);assert(title);
 const route=title[1].replace(/&amp;/g,'&'),p=new URLSearchParams(route.split('?')[1]);assert(p.get('version'));assert.equal(p.get('returnTo'),'documents?docOwner=me&docType=txt&docSort=name');
 const detail=h.navigate(route);assert(detail.includes('Back to results'));assert(detail.includes('returnTo='));
 h.navigate(p.get('returnTo'));assert.equal(h.run('DOC_FILTERS.type'),'txt');assert.equal(h.run('DOC_FILTERS.sort'),'name');assert.equal(h.run('dm212Filters().owner'),'me');
});
await test('Comparison finds later lines and long-line changes while bounding rendered excerpts',h=>{
 const before=Array.from({length:700},(_,i)=>'unchanged '+i).join('\n');h.context.diffBefore=before;h.context.diffAfter=before.replace('unchanged 550','changed 550');
 let diff=h.run('dm212Diff(diffBefore,diffAfter)');assert.equal(diff.prefix,550);assert.equal(diff.before,'unchanged 550');assert.equal(diff.after,'changed 550');assert(!diff.truncated);
 h.context.diffBefore='a'.repeat(40000)+'before';h.context.diffAfter='a'.repeat(40000)+'after';diff=h.run('dm212Diff(diffBefore,diffAfter)');assert(diff.leadingCharacters>0);assert(diff.before.endsWith('before'));assert(diff.after.endsWith('after'));assert(diff.before.length<=30000);
 h.context.diffBefore=Array.from({length:800},(_,i)=>'old '+i).join('\n');h.context.diffAfter=Array.from({length:800},(_,i)=>'new '+i).join('\n');diff=h.run('dm212Diff(diffBefore,diffAfter)');assert(diff.truncated);assert(diff.before.split('\n').length<=500);assert(diff.after.length<=30000);
});
await test('Exact-version comparison displays late changes safely without mutating source records',async h=>{
 const before=Array.from({length:700},(_,i)=>'same '+i).join('\n');pair(h,before,before.replace('same 610','<script>changed</script>'));const el=comparison(h),state=h.run('recStable(S)');await h.run('dm212LoadComparison(docNavElement)');
 assert(el.innerHTML.includes('610 leading'));assert(el.innerHTML.includes('&lt;script&gt;changed&lt;/script&gt;'));assert(!el.innerHTML.includes('<script>'));assert(el.innerHTML.includes('comparison selection'));assert.equal(h.run('recStable(S)'),state);
});
await test('Matching bounded previews explicitly disclose that later original text was not compared',async h=>{
 pair(h,'a'.repeat(300000)+'old original ending','a'.repeat(300000)+'new original ending');const el=comparison(h);await h.run('dm212LoadComparison(docNavElement)');
 assert(el.innerHTML.includes('readable preview text matches'));assert(el.innerHTML.includes('first 300,000 bytes'));assert(el.innerHTML.includes('outside those excerpts are not compared'));
 const paragraphs=Array.from({length:2501},(_,i)=>'Paragraph '+i);h.context.docNavWordBefore=wordBlob(paragraphs);paragraphs[2500]='A changed final paragraph';h.context.docNavWordAfter=wordBlob(paragraphs);
 await h.run(`(async()=>{for(const [v,blob]of [[docNavBefore,docNavWordBefore],[docNavAfter,docNavWordAfter]])Object.assign(v,{fileName:'source.docx',mime:DOC_MIME.docx,size:blob.size,sha256:recSHA256(new Uint8Array(await blob.arrayBuffer())),storage:await DocumentBytes.put(v.id,blob)});})()`);
 await h.run('dm212LoadComparison(docNavElement)');assert(el.innerHTML.includes('readable preview text matches'));assert(el.innerHTML.includes('2500 of 2501 paragraphs'));assert(el.innerHTML.includes('first 2,500 paragraphs'));assert(el.innerHTML.includes('later paragraphs are not displayed or compared'));assert(!el.innerHTML.includes('300,000 bytes'));assert(!el.innerHTML.includes('A changed final paragraph'));
});
await test('Comparison refuses a selected version changed during asynchronous text extraction',async h=>{
 pair(h,'earlier text','updated text');const el=comparison(h);h.run(`var docNavRead=dm212Readable;dm212Readable=async(v,b)=>{const model=await docNavRead(v,b);if(v.id==='DOC-NAV-V2')docVersion(v.id).note='Changed during extraction';return model;};`);await h.run('dm212LoadComparison(docNavElement)');assert(el.innerHTML.includes('selected version changed'));assert(!el.innerHTML.includes('dm212-compare-text'));
});
const report={suite:'Document navigation and comparison V2.15',browserLayoutValidated:false,passed:results.filter(x=>x.status==='PASS').length,failed:results.filter(x=>x.status==='FAIL').length,results};
fs.writeFileSync(path.join(root,'tests/document-navigation-v215-results.json'),JSON.stringify(report,null,2)+'\n');console.log(report.passed+'/'+results.length+' checks passed');if(report.failed)process.exitCode=1;
})();
