"""Board-report integration checks: real rendered UI and canonical read models.
Run after build.py. For isolated module development this also composes the new
report layer onto the previous standalone build without modifying that build.
"""
from pathlib import Path
import json, time, traceback
from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[1]
HTML = (ROOT / 'clearsight-v2-demo.html').read_text()
if 'window.CLEAR_SIGHT_BOARD=' not in HTML:
    HTML = HTML.replace('</head>', '<style>' + (ROOT / 'assets/board-reports.css').read_text() + '</style></head>')
    HTML = HTML.replace('</body>', '<script>' + (ROOT / 'assets/board-reports.js').read_text().replace('</script', '<\\/script') + '\nrender();</script></body>')
PREFIX = '''<script>window.TEST_STORE={};Object.defineProperty(window,'localStorage',{value:{getItem:k=>TEST_STORE[k]??null,setItem:(k,v)=>TEST_STORE[k]=String(v),removeItem:k=>delete TEST_STORE[k]}});</script>'''
RESULTS = []
ERRORS = []

def check(name, fn):
    started = time.monotonic()
    try:
        fn()
        RESULTS.append({'name': name, 'status': 'PASS', 'seconds': round(time.monotonic() - started, 2)})
        print('PASS', name, flush=True)
    except Exception as exc:
        RESULTS.append({'name': name, 'status': 'FAIL', 'error': str(exc)})
        print('FAIL', name, str(exc), flush=True)
        traceback.print_exc()

def new(width=1440):
    p = browser.new_page(viewport={'width': width, 'height': 1100})
    p.set_default_timeout(5000)
    p.on('pageerror', lambda error: ERRORS.append(str(error)))
    p.set_content(PREFIX + HTML)
    return p

def nav(p, route):
    p.evaluate('(r)=>location.hash="/"+r', route)
    p.wait_for_timeout(50)

def act(p, action, extra=''):
    p.locator('[data-action="' + action + '"]' + extra).first.click()
    p.wait_for_timeout(50)

def snapshot(p):
    return p.evaluate('CLEAR_SIGHT_BOARD.snapshot()')

with sync_playwright() as pw:
    browser = pw.chromium.launch(executable_path='/usr/bin/chromium', headless=True, args=['--no-sandbox'])

    def reconcile():
        p = new()
        nav(p, 'reports')
        x = snapshot(p)
        canonical = p.evaluate('scopeTotals(ixLossPeriod())')
        assert x['losses']['net'] == canonical['net']
        assert sum(v['net'] for v in x['board']['lossTrend']) == canonical['net']
        assert sum(v['net'] for v in x['board']['lossDrivers']) == canonical['net']
        assert sum(v['loss']['net'] for v in x['board']['locations']['rows']) == canonical['net']
        assert x['risk']['aboveAppetite'] == len([r for r in x['sourceRecords']['risks'] if r['residual'][0] * r['residual'][1] > r['appetite']])
        assert p.locator('.br-heat-cell').count() == 25
        assert p.locator('.br-decision').count() > 0
        p.close()
    check('Charts reconcile exactly with canonical losses and current appetite records', reconcile)

    def scope_and_clocks():
        p = new()
        nav(p, 'reports?view=drivers&period=90')
        before = snapshot(p)
        nav(p, 'reports?view=drivers&period=30')
        after = snapshot(p)
        assert before['sourceRecords']['risks'] == after['sourceRecords']['risks']
        assert before['kri'] == after['kri']
        assert after['periods']['loss']['days'] == 30
        assert after['board']['priorLoss']['from'] != before['board']['priorLoss']['from']
        p.locator('[data-org-scope=branch]').select_option('B-IK')
        p.wait_for_timeout(50)
        scoped = snapshot(p)
        assert all(p.evaluate('(id)=>orgWithin(id,"B-IK")', r['orgScopeId']) for r in scoped['sourceRecords']['risks'])
        assert all(p.evaluate('(id)=>orgWithin(id,"B-IK")', v['unit']) for v in scoped['kri']['values'])
        assert scoped['board']['scopeId'] == 'B-IK'
        p.close()
    check('Scope remains pinned to source records and loss windows do not change current risk or KRI clocks', scope_and_clocks)

    def frozen_report():
        p = new()
        nav(p, 'reports')
        act(p, 'ix-new-report')
        original = p.evaluate('clone(S.reportDraft.snapshot)')
        act(p, 'ix-report-step', '[data-step="2"]')
        act(p, 'ix-report-step', '[data-step="3"]')
        assert 'acknowledge' in p.locator('#ix-report-error').inner_text()
        p.locator('[data-ix-report-draft=commentary]').fill('Confirm the risk treatment owner and close the reporting gaps before making a bank-wide assurance statement.')
        p.locator('[data-br-coverage]').check()
        act(p, 'br-use-suggestion')
        assert p.locator('[data-br-decision][data-part=title]').count() == 1
        act(p, 'ix-report-step', '[data-step="3"]')
        assert p.locator('.br-frozen-pack').count() == 1
        act(p, 'ix-publish-report')
        pack = p.evaluate('clone(S.reportPacks[0])')
        assert pack['snapshot'] == original
        old_text = p.locator('.br-frozen-pack').inner_text()
        p.evaluate('()=>{S.risks[0].title="Changed live risk title";S.losses[0].gross+=100;S.tasks.push({id:"LIVE-ONLY",title:"Live-only action",owner:"Amina Bello",due:"2026-11-01",status:"Open",orgScopeId:"U-IK-PAY"});save();render();}')
        assert p.locator('.br-frozen-pack').inner_text() == old_text
        p.locator('[data-org-scope=branch]').select_option('B-EN')
        p.wait_for_timeout(50)
        assert p.evaluate('clone(S.reportPacks[0])') == pack
        assert p.locator('.br-frozen-pack').inner_text() == old_text
        p.evaluate('()=>{window.TEST_DOWNLOADS=[];download=(name,body,type)=>TEST_DOWNLOADS.push({name,body,type});}')
        act(p, 'ix-export-pack')
        exported = json.loads(p.evaluate('TEST_DOWNLOADS[0].body'))
        assert exported == pack
        assert exported['decisions'][0]['owner'] and exported['decisions'][0]['due']
        p.close()
    check('Board authoring captures structured decisions and published figures never refresh from live records', frozen_report)

    def followup():
        p = new()
        nav(p, 'reports')
        act(p, 'br-followup')
        form = p.locator('#modal-form')
        form.locator('[name=title]').fill('Confirm the operational risk treatment recovery plan')
        form.locator('[name=due]').fill('2026-11-27')
        form.locator('button[type=submit]').click()
        p.wait_for_timeout(50)
        tasks = p.evaluate('S.tasks.filter(t=>t.boardDecisionKey)')
        assert len(tasks) == 1
        t = tasks[0]
        assert t['sourceRef']['id'] == t['ref'] and t['orgScopeId']
        assert t['title'] == 'Confirm the operational risk treatment recovery plan'
        nav(p, 'work')
        assert t['title'] in p.locator('main').inner_text()
        p.close()
    check('An exception follow-up becomes one ordinary scoped Work record with its source link', followup)

    def definitions_and_missing():
        p = new()
        nav(p, 'reports?view=kri')
        x = snapshot(p)
        known = x['kri']['known']
        assert known <= x['kri']['expectedValues']
        assert all(v['definitionVersion'] == x['periods']['kri']['definitionVersion'] for v in x['kri']['values'])
        p.evaluate('()=>{S.branchKri.cycles.unshift({...clone(ixCycle()),id:"DIFFERENT-DEFINITION",definitionVersion:"incompatible/1",period:"2026-07"});}')
        assert all(v['id'] != 'DIFFERENT-DEFINITION' for v in snapshot(p)['board']['kriTrend'])
        assert p.evaluate('brMetricFormat({type:"currency",unit:"NGN"},125000)') == p.evaluate('ngn(125000)')
        assert p.evaluate('brMetricFormat({type:"boolean",aggregation:"exceptions"},3)') == '3 exceptions'
        p.close()
    check('KRI report preserves missingness, definition identity, monetary minor units and exception counts', definitions_and_missing)

    def responsive():
        p = new()
        views = ['executive','locations','drivers','kri','assurance','dependencies','quality','library']
        routes = ['reports?view=' + v for v in views] + ['insight-detail?kind=appetite','insight-detail?kind=actions']
        for width in [390,768,1440]:
            p.set_viewport_size({'width': width, 'height': 1100})
            for route in routes:
                nav(p, route)
                assert p.locator('h1').inner_text()
                assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'), (width, route)
        p.evaluate('()=>{S.theme="dark";render();}')
        nav(p, 'reports')
        assert p.locator('.br-brief').is_visible()
        p.close()
    check('Every report perspective and source drilldown renders without overflow at mobile, tablet and desktop widths', responsive)

    def print_report():
        p = new()
        nav(p, 'reports')
        act(p, 'ix-new-report')
        act(p, 'ix-report-step', '[data-step="2"]')
        p.locator('[data-br-coverage]').check()
        act(p, 'ix-report-step', '[data-step="3"]')
        act(p, 'ix-publish-report')
        p.emulate_media(media='print')
        assert not p.locator('.sidebar').is_visible()
        assert p.locator('.br-frozen-pack').is_visible()
        assert p.locator('.br-heatmap').is_visible()
        p.close()
    check('Print layout keeps board charts and tables while removing navigation and action controls', print_report)

    check('No browser runtime errors across reporting flows', lambda: (_ for _ in ()).throw(AssertionError(ERRORS)) if ERRORS else None)
    browser.close()

out = {'suite':'board-reports-v2.8','checks':RESULTS,'browser_errors':ERRORS,'passed':sum(r['status']=='PASS' for r in RESULTS),'failed':sum(r['status']=='FAIL' for r in RESULTS)}
(ROOT/'tests/board-reports-v28-results.json').write_text(json.dumps(out,indent=2))
print(json.dumps({'passed':out['passed'],'failed':out['failed']}))
raise SystemExit(1 if out['failed'] else 0)
