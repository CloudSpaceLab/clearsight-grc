import {registerHooks} from 'node:module';
import {mkdir,writeFile,readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {formsEvidenceScenarios} from '../../../web/scripts/forms-evidence-scenarios.mjs';
const only=process.env.ONLY_SCENARIO;
const start=process.env.FROM_SCENARIO;
const selected=only?formsEvidenceScenarios.filter(s=>s.name.startsWith(only)):formsEvidenceScenarios.slice(formsEvidenceScenarios.findIndex(s=>s.name.startsWith(start)));
if(!selected.length || (!only&&!start))throw new Error('Specify a valid ONLY_SCENARIO or FROM_SCENARIO prefix.');

const output=fileURLToPath(new URL(`${only?'full-forms-targeted':'full-forms-continuation'}/`,import.meta.url));
process.env.PAGE_URL='http://127.0.0.1:4188'; process.env.UI_EVIDENCE_DIR=output;
await mkdir(output,{recursive:true});
await writeFile(`${output}/manifest.json`,JSON.stringify({generatedAt:new Date().toISOString(),baseURL:process.env.PAGE_URL,selection:selected.map(s=>s.name),captures:[]},null,2));
registerHooks({resolve(specifier,context,nextResolve){if(specifier==='playwright')return {url:new URL('../../../.codex-tmp/playwright-ci/node_modules/playwright/index.mjs',import.meta.url).href,shortCircuit:true};return nextResolve(specifier,context)},load(url,context,nextLoad){ const result=nextLoad(url,context);if(url.endsWith('/web/scripts/capture-forms-evidence.mjs')) {const source=typeof result.source==='string'?result.source:new TextDecoder().decode(result.source);return {...result,source:source.replace('import { formsEvidenceScenarios } from', 'import { formsEvidenceScenarios as allFormsEvidenceScenarios } from').replace('const baseURL =',`const formsEvidenceScenarios = allFormsEvidenceScenarios.filter(s=>${JSON.stringify(selected.map(s=>s.name))}.includes(s.name));\nconst baseURL =`)};}return result;}});
await import('../../../web/scripts/capture-forms-evidence.mjs');
const result=JSON.parse(await readFile(`${output}/manifest.json`,'utf8'));
console.log(JSON.stringify({captured:result.captures.length,metrics:result.captures.filter(r=>r.name.startsWith('116-')).map(r=>r.scenario_metrics)},null,2));
