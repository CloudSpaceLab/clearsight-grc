#!/usr/bin/env python3
"""Rebuild and run the authoritative actual-source suite inventory.

Requires Python 3 and a modern Node runtime (the delivery used Node 24).
This does not launch a browser and does not update MANIFEST.json.
"""
from __future__ import annotations
import concurrent.futures
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
SUMMARY = ROOT / 'tests/validation-summary.json'

def command(args):
    return subprocess.run(args, cwd=ROOT, capture_output=True, text=True)

def run_suite(item):
    result = command(['node', str(ROOT / 'tests' / item['script'])])
    data = json.loads((ROOT / 'tests' / item['results']).read_text())
    rows = data.get('results', data.get('checks', []))
    passed = sum(row.get('status') == 'PASS' for row in rows)
    failed = len(rows) - passed
    return {**item, 'passed': passed, 'failed': failed}, result

def main():
    summary = json.loads(SUMMARY.read_text())
    built = command([sys.executable, str(ROOT / 'build.py')])
    if built.returncode:
        print(built.stderr, file=sys.stderr)
        return 1
    print(built.stdout.strip(), flush=True)
    suites = summary['modelWorkflowChecks']['suites']
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        runs = list(pool.map(run_suite, suites))
    failed_commands = 0
    for result, process in runs:
        print(f"{result['script']}: {result['passed']} passed, {result['failed']} failed", flush=True)
        if process.returncode:
            failed_commands += 1
            print((process.stderr or process.stdout)[-6500:], file=sys.stderr)
    summary['modelWorkflowChecks'] = {
        'passed': sum(row['passed'] for row, _ in runs),
        'failed': sum(row['failed'] for row, _ in runs),
        'suites': [row for row, _ in runs],
    }
    routes = command(['node', str(ROOT / 'tests/routes_v215_checks.js')])
    if routes.returncode:
        failed_commands += 1
        print(routes.stderr[-6500:], file=sys.stderr)
    print(routes.stdout.strip(), flush=True)
    rows = json.loads((ROOT / 'tests/routes-v215-results.json').read_text())['results']
    summary['routeActionChecks'] = {
        'passed': sum(row['status'] == 'PASS' for row in rows),
        'failed': sum(row['status'] != 'PASS' for row in rows),
        'entryRoutes': sum(row['kind'] == 'entry' for row in rows),
        'reportPerspectives': sum(row['kind'] == 'report perspective' for row in rows),
        'additionalInteractionViews': sum(row['kind'] == 'interaction variant' for row in rows),
        'roleEntryViews': sum(row['kind'] == 'role entry' for row in rows),
        'blockedRoleBoundaries': sum(row['kind'] == 'blocked role boundary' for row in rows),
    }
    compile_source = r"""
const fs=require('fs'),path=require('path'),vm=require('vm');
const files=fs.readdirSync('assets').filter(n=>n.endsWith('.js'));
for(const file of files)new vm.Script(fs.readFileSync(path.join('assets',file),'utf8'),{filename:file});
const html=fs.readFileSync('clearsight-v2-demo.html','utf8'),blocks=[...html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)];
if(blocks.length!==2)throw Error('Expected two standalone script blocks');
blocks.forEach((block,i)=>new vm.Script(block[1],{filename:'standalone-'+i}));
console.log(JSON.stringify({javascriptModules:files.length,standaloneScriptBlocks:blocks.length,status:'PASS'}));
"""
    compiled = command(['node', '-e', compile_source])
    if compiled.returncode:
        failed_commands += 1
        summary['compilation'] = {'status': 'FAIL', 'error': compiled.stderr[-2500:]}
        print(compiled.stderr[-2500:], file=sys.stderr)
    else:
        summary['compilation'] = json.loads(compiled.stdout)
    entries = json.loads((ROOT / 'docs/route-manifest.json').read_text())
    reference_count = 0
    for name in entries:
        html = (ROOT / (name + '.html')).read_text()
        refs = re.findall(r'(?:src|href)="(assets/[^"?#]+)"', html)
        for ref in refs:
            if not (ROOT / ref).is_file():
                raise ValueError('Missing module asset: ' + ref)
        reference_count += len(refs)
    standalone = ROOT / 'clearsight-v2-demo.html'
    html = standalone.read_text()
    external = bool(re.search(r'<script\b[^>]*\bsrc=|<link\b[^>]*\brel="stylesheet"', html))
    if external:
        raise ValueError('Standalone must not reference external JS/CSS')
    summary['build'] = {
        'entryPoints': len(entries),
        'linkedAssetReferences': reference_count,
        'standaloneBytes': standalone.stat().st_size,
        'externalStandaloneJsCss': external,
        'defaultRoute': re.search(r'<body data-start="([^"]+)"', html).group(1),
    }
    summary['runtime'] = 'Actual application JavaScript; ' + command(['node', '--version']).stdout.strip() + ' with substituted browser/DOM shells'
    SUMMARY.write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps({'modelWorkflowChecks': summary['modelWorkflowChecks']['passed'], 'modelFailures': summary['modelWorkflowChecks']['failed'], 'routeActionChecks': summary['routeActionChecks']['passed'], 'routeFailures': summary['routeActionChecks']['failed'], 'compilation': summary['compilation']['status'], 'build': summary['build']}, indent=2))
    return 1 if failed_commands or summary['modelWorkflowChecks']['failed'] or summary['routeActionChecks']['failed'] else 0

if __name__ == '__main__':
    raise SystemExit(main())
