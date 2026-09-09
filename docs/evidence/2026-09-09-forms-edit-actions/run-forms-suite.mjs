import {registerHooks} from 'node:module';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const output=fileURLToPath(new URL('full-forms/',import.meta.url));
process.env.PAGE_URL='http://127.0.0.1:4188';
process.env.UI_EVIDENCE_DIR=output;
await mkdir(output,{recursive:true});
await writeFile(`${output}/manifest.json`,JSON.stringify({generatedAt:new Date().toISOString(),baseURL:process.env.PAGE_URL,captures:[]},null,2));
registerHooks({resolve(specifier,context,nextResolve){
  if(specifier==='playwright') return {url:new URL('../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs',import.meta.url).href,shortCircuit:true};
  return nextResolve(specifier,context);
}});
await import('../../../web/scripts/capture-forms-evidence.mjs');
const result=JSON.parse(await readFile(`${output}/manifest.json`,'utf8'));
console.log(`${result.captures.length} Forms scenarios captured without failure.`);
