"""V2.4 retained direct-form and scope regression tests; deterministic fixtures, inline Chromium, storage test double."""
from pathlib import Path
import json, time, os, traceback
from playwright.sync_api import sync_playwright
ROOT = Path(__file__).resolve().parents[1]
HTML = (ROOT / 'clearsight-v2-demo.html').read_text()
RESULTS = []
ERRORS = []

def run(name, fn):
    filt = os.environ.get('TEST_FILTER', '').lower()
    if filt and (filt.startswith('!') and filt[1:] in name.lower() or (not filt.startswith('!') and filt not in name.lower())):
        return
    if name.startswith('New risk and loss capture') or name.startswith('24 routes'):
        return
    start = time.monotonic()
    try:
        fn()
        RESULTS.append({'name': name, 'status': 'PASS', 'seconds': round(time.monotonic() - start, 2)})
        print('PASS', name, flush=True)
    except Exception as e:
        RESULTS.append({'name': name, 'status': 'FAIL', 'error': str(e)})
        print('FAIL', name, str(e), flush=True)
        traceback.print_exc()

def new(store=None):
    p = b.new_page(viewport={'width': 1440, 'height': 1000})
    p.set_default_timeout(5000)
    p.on('pageerror', lambda e: ERRORS.append(str(e)))
    prefix = '<script>const STORE=' + json.dumps(store or {}) + ';Object.defineProperty(window,"localStorage",{value:{getItem:k=>STORE[k]??null,setItem:(k,v)=>STORE[k]=String(v),removeItem:k=>delete STORE[k]}});window.TEST_STORE=STORE;</script>'
    p.set_content(prefix + HTML)
    return p

def nav(p, r):
    p.evaluate('(r)=>location.hash="/"+r', r)
    p.wait_for_timeout(25)

def act(p, a, id=None):
    p.locator(f'[data-action="{a}"]' + (f'[data-id="{id}"]' if id else '')).first.click()

def submit(p):
    p.locator('#modal-form button[type=submit]').click()

def close(p):
    if p.locator('#dialog').evaluate('(d)=>d.open'):
        act(p, 'close-dialog')

def field(p, name, value):
    el = p.locator('#modal-form [name="' + name + '"]')
    (el.select_option if el.evaluate('(el)=>el.tagName') == 'SELECT' else el.fill)(value)

def scope(p, kind, id):
    p.locator(f'[data-org-scope={kind}]').select_option(id)

def totals(p):
    return p.evaluate('CLEAR_SIGHT_EXPERIENCE.totals()')

def state(p):
    return p.evaluate('CLEAR_SIGHT_DEMO.getState()')

def forms(p):
    return p.evaluate('CLEAR_SIGHT_FORMS.getState()')

def blur(p, selector, value):
    p.locator(selector).fill(value)
    p.locator(selector).press('Tab')
with sync_playwright() as pw:
    b = pw.chromium.launch(executable_path='/usr/bin/chromium', headless=True, args=['--no-sandbox'])

    def reconciliation():
        p = new()
        nav(p, 'risks')
        assert totals(p) == {'gross': 3650000000, 'recovered': 1200000000, 'net': 2450000000, 'actual': 4, 'near': 1, 'total': 5}
        assert p.locator('[data-loss-total=net]').inner_text() == '₦24,500,000'
        scope(p, 'branch', 'B-HO')
        assert totals(p)['net'] == 1200000000
        assert p.locator('tr[data-search]').count() == 3
        scope(p, 'department', 'D-HO-TECH')
        assert p.locator('tr[data-search]').count() == 2
        scope(p, 'unit', 'U-HO-SEC')
        assert totals(p)['net'] == 0
        assert p.locator('tr[data-search]').count() == 1
        act(p, 'cx-clear-scope', 'unit')
        assert totals(p)['net'] == 1200000000
        act(p, 'cx-clear-scope', 'department')
        assert p.locator('tr[data-search]').count() == 3
        act(p, 'cx-reset-scope')
        scope(p, 'unit', 'U-IK-PAY')
        assert state(p)['scope'] == {'branch': 'B-IK', 'department': 'D-IK-OPS', 'unit': 'U-IK-PAY'}
        assert totals(p)['net'] == 550000000 and totals(p)['near'] == 1
        act(p, 'cx-reset-scope')
        assert totals(p)['net'] == 2450000000
        p.close()
    run('Hierarchical selection, exact scope totals, descendant resets and child-to-parent resolution', reconciliation)

    def drill():
        p = new()
        nav(p, 'risks')
        act(p, 'cx-drill-scope', 'B-IK')
        assert p.locator('h2').filter(has_text='Losses by department').count() == 1
        act(p, 'cx-drill-scope', 'D-IK-OPS')
        assert p.locator('h2').filter(has_text='Losses by unit').count() == 1
        act(p, 'cx-drill-scope', 'U-IK-PAY')
        assert p.locator('h2').filter(has_text='Linked loss events').count() == 1
        p.locator('[data-cx-period]').select_option('30')
        nav(p, 'losses')
        assert len(p.evaluate('CLEAR_SIGHT_EXPERIENCE.lossIds()')) == 2
        assert p.locator('[data-loss-period]').input_value() == '30'
        nav(p, 'reports')
        report = p.evaluate('CLEAR_SIGHT_DEMO.getReport()')
        assert report['losses']['net'] == 550000000 and report['lossPeriod']['days'] == 30
        store = p.evaluate('TEST_STORE')
        p.close()
        p = new(store)
        assert state(p)['scope']['unit'] == 'U-IK-PAY'
        nav(p, 'risks')
        assert totals(p)['net'] == 550000000
        p.close()
    run('Loss drill-down, period/navigation consistency and persisted scope serialization', drill)

    def zero():
        p = new()
        nav(p, 'risks')
        scope(p, 'unit', 'U-EN-CUSTOMER')
        assert totals(p)['net'] == 0
        assert p.locator('[data-loss-total=net]').inner_text() == '₦0'
        assert p.locator('[data-scope-risk-count]').inner_text() == '0'
        nav(p, 'rcsa')
        assert p.locator('.cycle-card').count() == 0
        nav(p, 'program?id=PRG-104')
        assert p.locator('h1').inner_text() == 'Outside the selected scope'
        assert p.evaluate('current.blocked') and p.locator('.copilot-trigger').count() == 0
        nav(p, 'form-builder?form=FORM-OPS')
        assert p.locator('.cx-context-note').count() == 1
        assert p.locator('[data-cx-label]').count() == len(forms(p)['forms'][0]['fields'])
        p.close()
    run('Zero-loss and zero-risk scopes, out-of-scope detail guard, shared template exception', zero)

    def create():
        p = new()
        nav(p, 'forms')
        act(p, 'fa-create')
        field(p, 'name', 'Monthly branch review')
        submit(p)
        f = forms(p)['forms'][0]
        id = f['id']
        assert f['fields'] == []
        act(p, 'cx-add')
        act(p, 'cx-add-type', 'boolean')
        q = forms(p)['forms'][0]['fields'][0]['id']
        blur(p, f'[data-cx-label="{q}"]', 'Has reconciliation been completed?')
        p.locator(f'[data-cx-required="{q}"]').check()
        act(p, 'cx-add')
        act(p, 'cx-add-type', 'matrix')
        grid = forms(p)['forms'][0]['fields'][1]['id']
        blur(p, f'[data-cx-row="{grid}"][data-index="0"]', 'Cash controls')
        blur(p, f'[data-cx-option="{grid}"][data-index="0"]', 'Effective')
        act(p, 'cx-row-add', grid)
        assert len(forms(p)['forms'][0]['fields'][1]['rows']) == 3
        act(p, 'cx-option-add', grid)
        assert len(forms(p)['forms'][0]['fields'][1]['options']) == 5
        blur(p, '[data-cx-form-name]', 'Monthly branch review · revised')
        act(p, 'fa-publish', id)
        submit(p)
        assert forms(p)['forms'][0]['status'] == 'Published'
        nav(p, 'form-response?form=' + id)
        assert p.locator('[data-fa-answer="' + q + '"]').count() == 2
        assert p.locator('[data-fa-matrix="' + grid + '"]').count() == 15
        p.close()
    run('Create a blank form, edit questions/grid directly, publish and preview', create)

    def column():
        p = new()
        nav(p, 'form-builder?form=FORM-OPS')
        original = forms(p)['versions'][0]['definition']['fields'][4]['columns'][1]['label']
        blur(p, '[data-cx-column-label=register][data-column=design]', 'Control design')
        p.locator('[data-action=cx-column][data-id=register][data-column=design]').click()
        field(p, 'choices', 'Effective\nPartial\nIneffective\nNot tested')
        submit(p)
        f = forms(p)['forms'][0]
        assert f['fields'][4]['columns'][1]['label'] == 'Control design'
        assert len(f['fields'][4]['columns'][1]['options']) == 4
        assert forms(p)['versions'][0]['definition']['fields'][4]['columns'][1]['label'] == original
        assert f['status'] == 'Draft'
        act(p, 'cx-column-add', 'register')
        field(p, 'label', 'Accountable owner')
        submit(p)
        assert forms(p)['forms'][0]['fields'][4]['columns'][-1]['label'] == 'Accountable owner'
        p.close()
    run('Direct table headings, one-column editor and published-version isolation', column)

    def source():
        p = new()
        nav(p, 'form-builder?form=FORM-TECH')
        act(p, 'cx-source', 'response_time')
        assert not p.locator('details.cx-details').get_attribute('open')
        field(p, 'threshold', '4000')
        act(p, 'cx-test-source')
        assert 'Pass' in p.locator('#cx-source-result').inner_text()
        submit(p)
        assert forms(p)['forms'][1]['fields'][1]['check']['threshold'] == '4000'
        nav(p, 'form-response?form=FORM-TECH')
        act(p, 'fa-run')
        res = forms(p)['runs'][0]['results']
        assert all((x['result'] == 'Pass' for x in res.values()))
        p.locator('#fa-scenario').select_option('sparse')
        act(p, 'fa-run')
        assert forms(p)['runs'][0]['results']['availability']['result'] == 'Unknown'
        p.locator('#fa-scenario').select_option('unavailable')
        act(p, 'fa-run')
        assert all((x['result'] == 'Unknown' for x in forms(p)['runs'][0]['results'].values()))
        p.close()
    run('Focused connected-data setup, sample test, saved threshold and unknown-source semantics', source)

    def source_cancel():
        p = new()
        nav(p, 'form-builder?form=FORM-VENDOR')
        orig = forms(p)['forms'][2]['fields'][2]['columns'][1]
        p.locator('[data-action=cx-column][data-id=service_grid][data-column=owner]').click()
        field(p, 'label', 'Temporary change')
        act(p, 'cx-column-source', 'service_grid')
        close(p)
        assert forms(p)['forms'][2]['fields'][2]['columns'][1] == orig
        p.locator('[data-action=cx-column][data-id=service_grid][data-column=owner]').click()
        act(p, 'cx-column-source', 'service_grid')
        act(p, 'cx-source-mode', 'integration')
        field(p, 'kind', 'api')
        submit(p)
        col = forms(p)['forms'][2]['fields'][2]['columns'][1]
        assert col['mode'] == 'integration' and col['check']['kind'] == 'api'
        p.close()
    run('Per-column automation and cancel-safe staged column edits', source_cancel)

    def rules():
        p = new()
        nav(p, 'form-builder?form=FORM-OPS')
        act(p, 'cx-rules', 'exception_note')
        assert p.locator('[name=when]').input_value() == 'exceptions'
        field(p, 'equals', 'No')
        submit(p)
        assert next(f for f in forms(p)['forms'][0]['fields'] if f['id']=='exception_note')['condition']['equals'] == 'No'
        nav(p, 'form-response?form=FORM-OPS')
        act(p, 'fa-example')
        assert p.locator('[data-fa-answer=exception_note]').count() == 0
        p.locator('[data-fa-answer=exceptions][value=No]').check()
        assert p.locator('[data-fa-answer=exception_note]').count() == 1
        assert p.locator('output.fa-score').all_text_contents() == ['4', '4']
        p.close()
    run('Progressive question rules, conditional visibility and existing calculated grid answers', rules)

    def grids_mobile():
        p = new()
        for width in [390, 768, 1024, 1440]:
            p.set_viewport_size({'width': width, 'height': 900})
            nav(p, 'form-builder?form=FORM-OPS')
            p.locator('#cx-q-register').scroll_into_view_if_needed()
            assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
            p.locator('[data-action=cx-column][data-id=register][data-column=design]').click()
            assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
            close(p)
            nav(p, 'form-builder?form=FORM-TECH')
            act(p, 'cx-source', 'availability')
            assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
            close(p)
            act(p, 'theme')
            assert p.locator('html').get_attribute('data-theme') == 'dark'
            act(p, 'theme')
        p.close()
    run('Grid, column editor, technical-source dialog and dark theme across four widths', grids_mobile)
    b.close()
report = {'results': RESULTS, 'javascript_errors': ERRORS, 'passed': sum((x['status'] == 'PASS' for x in RESULTS)), 'failed': sum((x['status'] == 'FAIL' for x in RESULTS)), 'browser': 'Chromium', 'origin': 'inline set_content; storage test double', 'scope_route_viewport_combinations': 0}
(ROOT / ('tests/experience-regression-results-' + os.environ.get('TEST_FILTER', 'full').replace(' ', '-') + '.json')).write_text(json.dumps(report, indent=2))
print(json.dumps({k: report[k] for k in ['passed', 'failed', 'javascript_errors']}, indent=2))
raise SystemExit(bool(report['failed'] or ERRORS))
