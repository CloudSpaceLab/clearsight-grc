// Read-only source inventory. Run from repository root: node docs/evidence/2026-09-09-ui-audit/copy-inventory.cjs
const fs = require('fs');
const path = require('path');
const root = 'web/src';
const files = [];
function walk(dir) { for (const entry of fs.readdirSync(dir, {withFileTypes:true})) { const p = path.join(dir, entry.name); if(entry.isDirectory()) { if(entry.name !== 'test') walk(p); } else if(/\.(ts|tsx)$/.test(p) && !/\.test\.|\.d\.ts$|testSetup/.test(p)) files.push(p.replaceAll('\\','/')); } }
walk(root);
const rows = [];
for(const file of files.sort()) {
 const source = fs.readFileSync(file,'utf8');
 source.split(/\r?\n/).forEach((value,index)=>{
  if (/^\s*(import|export type|type |interface |\/\/|\*)/.test(value)) return;
  const candidate=/>\s*[A-Za-z][^<>]*</.test(value) || /(?:label|title|description|placeholder|aria-label|alt|population|action|errorMessage)[=:]/.test(value) || /["'`][A-Za-z][A-Za-z ,;:.-]+ [A-Za-z]/.test(value);
  if(candidate) rows.push({file,line:index+1,value:value.trim()});
 });
}
const output='docs/evidence/2026-09-09-ui-audit';
fs.writeFileSync(`${output}/frontend-copy-candidates.json`,JSON.stringify({method:'Heuristic source-line inventory: JSX text, known text props, or literals with natural-language spacing. Not an AST or unique-rendered-string count. Excludes test and declaration files. Includes reference and fixture code; may include code/comments and miss dynamic strings.',files,rows},null,2));
fs.writeFileSync(`${output}/frontend-copy-candidates.txt`,rows.map(r=>`${r.file}:${r.line}\t${r.value}`).join('\n'));
const covered=files.filter(f=>/^web\/src\/[^/]+\.(ts|tsx)$/.test(f)||/^web\/src\/components\/[^/]+\.(ts|tsx)$/.test(f)||/^web\/src\/components\/forms\/[^/]+\.(ts|tsx)$/.test(f));
console.log(JSON.stringify({sourceFiles:files.length,filesWithCandidates:new Set(rows.map(r=>r.file)).size,candidateOccurrences:rows.length,uniqueCandidateValues:new Set(rows.map(r=>r.value)).size,copyQualityGlobCoveredFiles:covered.length,copyQualityGlobExcludedFiles:files.filter(f=>!covered.includes(f)).length,bankCandidates:rows.filter(r=>/\bbank(?:ing|'s)?\b/i.test(r.value)).length,bankCandidateFiles:new Set(rows.filter(r=>/\bbank(?:ing|'s)?\b/i.test(r.value)).map(r=>r.file)).size},null,2));
