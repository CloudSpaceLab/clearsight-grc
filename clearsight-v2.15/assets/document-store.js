/* File bytes live outside the application JSON. IndexedDB is optional; memory
 * fallback is explicit. Only synthetic examples should be used in this prototype. */
'use strict';
const DocumentBytes=(()=>{
  const memory=new Map();let database=null,available=false,initPromise;
  function open(){if(initPromise)return initPromise;initPromise=new Promise(resolve=>{
    try{const req=indexedDB.open('clearsight-v26-files',1);
      req.onupgradeneeded=()=>req.result.createObjectStore('bytes');
      req.onsuccess=()=>{database=req.result;available=true;resolve(true);};
      req.onerror=()=>resolve(false);req.onblocked=()=>resolve(false);
    }catch(_){resolve(false);}
  });return initPromise;}
  async function put(id,bytes){
    await open();const blob=bytes instanceof Blob?bytes:new Blob([bytes]);
    if(available){try{await new Promise((resolve,reject)=>{const t=database.transaction('bytes','readwrite');t.objectStore('bytes').put(blob,id);t.oncomplete=resolve;t.onerror=()=>reject(t.error);t.onabort=()=>reject(t.error);});return 'browser';}catch(_){available=false;}}
    memory.set(id,blob);return 'session';
  }
  async function get(id){if(memory.has(id))return memory.get(id);await open();if(!available)return null;try{return await new Promise(resolve=>{const req=database.transaction('bytes').objectStore('bytes').get(id);req.onsuccess=()=>resolve(req.result||null);req.onerror=()=>resolve(null);});}catch(_){return null;}}
  async function remove(id){memory.delete(id);await open();if(!available)return;await new Promise(resolve=>{const t=database.transaction('bytes','readwrite');t.objectStore('bytes').delete(id);t.oncomplete=resolve;t.onerror=resolve;});}
  async function clear(){memory.clear();await open();if(!available)return;await new Promise(resolve=>{const t=database.transaction('bytes','readwrite');t.objectStore('bytes').clear();t.oncomplete=resolve;t.onerror=resolve;});}
  return {open,put,get,remove,clear,mode:()=>available?'browser':'session'};
})();
const DOC_MIME={pdf:'application/pdf',png:'image/png',jpg:'image/jpeg',jpeg:'image/jpeg',txt:'text/plain',csv:'text/csv',docx:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',xlsx:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'};
function docVersion(id){return S.governance?.versions.find(v=>v.id===id)||null;}
function docHead(d){return docVersion(d.currentVersionId);}
function docValidateFile(file,bytes){
  requireValue(file.size>0,'Empty files cannot be added.');
  requireValue(file.size<=20*1024*1024,'The per-file limit is 20 MB.');
  requireValue(file.name.length<=180&&!/[\x00-\x1f\x7f/\\]/.test(file.name),'Use a filename of 180 characters or fewer without control or path characters.');
  const extension=file.name.split('.').at(-1).toLowerCase();
  requireValue(Object.hasOwn(DOC_MIME,extension),'Allowed formats: PDF, PNG, JPG, TXT, CSV, DOCX and XLSX.');
  requireValue(!/\.(exe|js|html?|svg|bat|cmd|ps1|vbs|scr|com)(\.|$)/i.test(file.name),'Executable or active-content filenames are not accepted.');
  const sig=[...bytes.slice(0,12)],prefix=new TextDecoder().decode(bytes.slice(0,512));
  if(extension==='pdf')requireValue(prefix.startsWith('%PDF-'),'This file does not have a PDF header.');
  if(extension==='png')requireValue(sig.slice(0,8).join(',')==='137,80,78,71,13,10,26,10','This file does not have a PNG signature.');
  if(['jpg','jpeg'].includes(extension))requireValue(sig[0]===255&&sig[1]===216&&sig[2]===255,'This file does not have a JPEG signature.');
  if(['docx','xlsx'].includes(extension)){
    requireValue(sig[0]===80&&sig[1]===75&&sig[2]===3&&sig[3]===4,'This file is not an Office ZIP container.');
    // Inspect central-directory entry names without decompressing untrusted data.
    const dv=new DataView(bytes.buffer,bytes.byteOffset,bytes.byteLength),names=[];
    for(let i=0;i+46<bytes.length;i++)if(dv.getUint32(i,true)===0x02014b50){const n=dv.getUint16(i+28,true),extra=dv.getUint16(i+30,true),comment=dv.getUint16(i+32,true);if(i+46+n<=bytes.length)names.push(new TextDecoder().decode(bytes.slice(i+46,i+46+n)));i+=45+n+extra+comment;}
    requireValue(names.includes('[Content_Types].xml')&&names.includes(extension==='docx'?'word/document.xml':'xl/workbook.xml'),'The Office package is missing its expected document entries.');
    requireValue(!names.some(n=>/vbaProject|activeX|\.exe$/i.test(n)),'Macro or active-content Office packages are not supported.');
  }
  if(['txt','csv'].includes(extension)){
    requireValue(!bytes.includes(0),'Binary content is not accepted as text.');
    let text;try{text=new TextDecoder('utf-8',{fatal:true}).decode(bytes);}catch(_){throw Error('Save text and CSV files as UTF-8.');}
    requireValue(!/^\s*(<!doctype\s+html|<html|<script|<svg)/i.test(text),'HTML, SVG and script documents are not supported.');
  }
  return {extension,mime:DOC_MIME[extension]};
}
function docParseCSV(text,limit=101){
  const rows=[];let row=[],cell='',quoted=false;
  for(let i=0;i<text.length;i++){const c=text[i];if(c==='"'){if(quoted&&text[i+1]==='"'){cell+='"';i++;}else quoted=!quoted;}else if(c===','&&!quoted){row.push(cell);cell='';}else if((c==='\n'||c==='\r')&&!quoted){if(c==='\r'&&text[i+1]==='\n')i++;row.push(cell);rows.push(row);row=[];cell='';if(rows.length>=limit)return rows;}else cell+=c;}
  if(cell.length||row.length){row.push(cell);rows.push(row);}return rows;
}
function docBlobDownload(blob,name){const url=URL.createObjectURL(blob),a=document.createElement('a');a.download=name;a.href=url;document.body.append(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(url),3000);}
