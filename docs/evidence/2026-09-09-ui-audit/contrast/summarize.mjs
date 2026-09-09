import {readdir,readFile,writeFile} from 'node:fs/promises';
const dir=new URL('.',import.meta.url);const all=[];for(const f of await readdir(dir)){if(!/-(1440|390)\.json$/.test(f))continue;all.push(JSON.parse(await readFile(new URL(f,dir),'utf8')))}
const violations=all.flatMap(r=>(r.axe?.violations??[]).flatMap(v=>v.nodes.map(n=>({view:r.id,selector:n.target.join(' '),html:n.html,...n.any[0]?.data}))));
const readings=all.flatMap(r=>(r.measurements??[]).filter(m=>!m.disabled).map(m=>({view:r.id,...m})));
const aggregate={counts:{captures:all.length,errors:all.filter(r=>r.status==='ERROR').map(r=>({id:r.id,error:r.error})),axeViolations:violations.length,axeIncomplete:all.reduce((n,r)=>n+(r.axe?.incomplete??[]).reduce((n,v)=>n+v.nodes.length,0),0),textReadings:readings.filter(r=>r.kind==='text').length},violations,placeholders:readings.filter(r=>r.kind==='placeholder'),textBelowThreshold:readings.filter(r=>r.kind==='text'&&r.ratio<r.minimum),borderBelowThreshold:readings.filter(r=>r.kind==='control-border'&&r.ratio<3)};
await writeFile(new URL('aggregate.json',dir),JSON.stringify(aggregate,null,2));console.log(JSON.stringify(aggregate.counts));console.log('Violations',JSON.stringify(violations));console.log('Placeholders',JSON.stringify(aggregate.placeholders.map(({view,selector,text,foreground,background,ratio})=>({view,selector,text,foreground,background,ratio}))));

