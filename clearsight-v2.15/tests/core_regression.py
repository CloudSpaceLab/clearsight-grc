"""V2.4 retained core workflows. Superseded drawer tests are not executed; their replacements are in journey_checks.py.
Requires Python Playwright + Chromium. No external URL or dependencies at runtime.
The managed test browser blocks navigation/storage; render with set_content.
Persistence checks use an explicitly labelled Storage-API test double.
"""
from pathlib import Path
import json, re, os, sys, time
from playwright.sync_api import sync_playwright, expect
ROOT = Path(__file__).resolve().parents[1]
HTML = (ROOT / 'clearsight-v2-demo.html').read_text()
ROUTES = json.loads((ROOT / 'docs/route-manifest.json').read_text())
RESULT = []
ERRORS = []

def record(name, fn):
    if not any((name.startswith(x) for x in ['Adoption:', 'Vendor adoption:', 'RCSA:', 'Unknown-source', 'Mobile navigation', 'CSV/JSON'])):
        return
    start = time.monotonic()
    try:
        fn()
        RESULT.append({'test': name, 'status': 'PASS', 'seconds': round(time.monotonic() - start, 2)})
        print('PASS', name, flush=True)
    except Exception as err:
        RESULT.append({'test': name, 'status': 'FAIL', 'error': str(err), 'seconds': round(time.monotonic() - start, 2)})
        print('FAIL', name, str(err), flush=True)
        import traceback
        traceback.print_exc()

def state(page):
    return page.evaluate('CLEAR_SIGHT_DEMO.getState()')

def nav(page, route):
    page.evaluate('(r)=>{location.hash="/"+r}', route)
    page.wait_for_timeout(35)

def action(page, name, id=None):
    loc = page.locator(f'[data-action="{name}"]' + (f'[data-id="{id}"]' if id else ''))
    loc.first.click()

def submit(page):
    page.locator('#modal-form button[type=submit]').click()

def close(page):
    if page.locator('#dialog').evaluate('(x)=>x.open'):
        action(page, 'close-dialog')

def field(page, name, value):
    q = page.locator(f'#modal-form [name="{name}"]')
    if q.evaluate('(x)=>x.tagName') == 'SELECT':
        q.select_option(value)
    else:
        q.fill(value)

def newpage(browser, width=1440, height=1000, storage=None):
    page = browser.new_page(viewport={'width': width, 'height': height}, accept_downloads=True)
    page.set_default_timeout(6000)
    page.on('pageerror', lambda err: ERRORS.append(str(err)))
    prefix = ''
    if storage is not None:
        prefix = '<script>const STORE=' + json.dumps(storage) + ';Object.defineProperty(window,"localStorage",{value:{getItem:k=>STORE[k]??null,setItem:(k,v)=>STORE[k]=String(v),removeItem:k=>delete STORE[k]}});window.TEST_STORE=STORE;</script>'
    page.set_content(prefix + HTML, wait_until='load')
    return page
with sync_playwright() as p:
    browser = p.chromium.launch(executable_path=os.environ.get('CHROMIUM_PATH', '/usr/bin/chromium'), headless=True, args=['--no-sandbox'])

    def adoption_test():
        page = newpage(browser, storage={})
        nav(page, 'frameworks')
        action(page, 'start-adoption', 'FW-BANK')
        page.locator('[data-draft=name]').fill('Enterprise baseline · browser test')
        canonical = page.evaluate('CLEAR_SIGHT_DEMO.getCanonical()')
        action(page, 'tailor', 'R03')
        field(page, 'override', 'Review all privileged access every 30 days.')
        submit(page)
        page.locator('[data-requirement=R08]').click()
        field(page, 'reason', 'Excluded in this test scope with documented justification.')
        submit(page)
        action(page, 'wizard-step', '2')
        action(page, 'question-rule', 'Q05')
        field(page, 'freshDays', '15')
        submit(page)
        action(page, 'test-draft')
        expect(page.locator('text=Cloud fixture: source unavailable → Unknown.')).to_be_visible()
        action(page, 'wizard-step', '3')
        page.locator('[data-draft=owner]').select_option('Sarah Adams')
        action(page, 'assign-requirement', 'R03')
        field(page, 'mode', 'override')
        page.locator('#modal-form input[name=units]').uncheck() if page.locator('#modal-form input[name=units]').count() == 1 else None
        for el in page.locator('#modal-form input[name=units]').all():
            el.uncheck()
        page.locator('#modal-form input[value="Information Security"]').check()
        submit(page)
        action(page, 'publish-review')
        submit(page)
        data = state(page)
        pg = data['programs'][0]
        assert pg['name'] == 'Enterprise baseline · browser test'
        assert len(pg['reqs']) == 8 and len(pg['excluded']) == 1
        assert pg['rules']['Q05']['freshDays'] == 15
        assert pg['assignments']['R03'] == ['Information Security']
        assert pg['evaluations'] == {'Meridian Bank': {}}
        assert page.evaluate('CLEAR_SIGHT_DEMO.getCanonical()') == canonical
        report = page.evaluate('CLEAR_SIGHT_DEMO.getReport()')['programs'][0]['targets'][0]
        assert report['all'] == 7 and report['pass'] == 0 and (report['unassessed'] == 7)
        nav(page, 'assessment?id=' + pg['id'])
        assert page.locator('#q-Q06').count() == 1
        stored = page.evaluate('TEST_STORE')
        page.close()
        page = newpage(browser, storage=stored)
        assert state(page)['programs'][0]['id'] == pg['id']
        page.close()
    record('Adoption: tailoring, N/A, smart rules, assignment, publication, source integrity, serialization', adoption_test)

    def multi_target_test():
        page = newpage(browser)
        nav(page, 'frameworks')
        action(page, 'select-framework', 'FW-VENDOR')
        action(page, 'start-adoption', 'FW-VENDOR')
        page.locator('[data-target=ENG-02]').check()
        action(page, 'wizard-step', '3')
        action(page, 'publish-review')
        submit(page)
        pg = state(page)['programs'][0]
        assert pg['scopeType'] == 'vendor' and len(pg['targets']) == 2
        assert pg['evaluations'] == {'ENG-01': {}, 'ENG-02': {}}
        targets = page.evaluate('CLEAR_SIGHT_DEMO.getReport()')['programs'][0]['targets']
        assert all((x['all'] == 5 and x['pass'] == 0 for x in targets))
        page.locator('[data-program-target]').select_option('ENG-02')
        assert 'ENG-02' in page.evaluate('location.hash')
        assert state(page)['programs'][1]['evaluations']['Meridian Bank']['R01']['status'] == 'Satisfied'
        page.close()
    record('Vendor adoption: independent target populations, no bank score transfer', multi_target_test)

    def rcsa_test():
        page = newpage(browser)
        nav(page, 'rcsa-review?id=RCSA-01&risk=RSK-01')
        old = state(page)['risks'][0]['residual']
        form = page.locator('[data-submit=rcsa-submit]')
        form.locator('[name=likelihood]').select_option('2')
        form.locator('[name=impact]').select_option('3')
        form.locator('[name=rationale]').fill('Evidence supports the proposed residual score after independent control testing.')
        form.locator('button[type=submit]').click()
        assert state(page)['risks'][0]['residual'] == old
        nav(page, 'rcsa-review?id=RCSA-01&risk=RSK-01&mode=review')
        form = page.locator('[data-submit=rcsa-decision]')
        form.locator('[name=reviewer]').select_option('Chidi Okafor')
        form.locator('[name=note]').fill('Reviewed the evidence and challenged the control scope.')
        form.locator('button[type=submit]').click()
        expect(page.locator('.form-error')).to_contain_text('Self-approval')
        form.locator('[name=reviewer]').select_option('Ifeoma Nwosu')
        form.locator('[name=decision]').select_option('Challenge')
        form.locator('button[type=submit]').click()
        assert state(page)['cycles'][0]['reviews']['RSK-01']['status'] == 'Challenge requested'
        assert state(page)['risks'][0]['residual'] == old
        nav(page, 'rcsa-review?id=RCSA-01&risk=RSK-01')
        page.locator('[data-submit=rcsa-submit] [name=rationale]').fill('Expanded evidence confirms the proposed lower residual exposure in this scope.')
        page.locator('[data-submit=rcsa-submit] button[type=submit]').click()
        nav(page, 'rcsa-review?id=RCSA-01&risk=RSK-01&mode=review')
        page.locator('[name=note]').fill('Independently verified current accepted evidence and the assessment rationale.')
        page.locator('[data-submit=rcsa-decision] button[type=submit]').click()
        data = state(page)
        assert data['risks'][0]['residual'] == [2, 3]
        assert data['cycles'][0]['reviews']['RSK-01']['status'] == 'Approved'
        assert len(data['cycles'][0]['reviews']['RSK-01']['history']) == 1
        assert data['programs'][0]['evaluations']['Meridian Bank']['R03']['status'] == 'Partially satisfied'
        page.close()
    record('RCSA: proposal isolation, self-approval rejection, challenge/resubmit and accepted rating', rcsa_test)

    def fixtures_filters():
        page = newpage(browser)
        nav(page, 'integrations')
        before = state(page)['programs']
        action(page, 'run-fixtures')
        assert any((x['quality'] == 'Unknown' for x in state(page)['fixtureRuns']))
        assert state(page)['programs'] == before
        nav(page, 'risks')
        action(page, 'matrix-filter', '3,4')
        assert page.locator('tr[data-search]:visible').count() == 2
        page.locator('[data-org-scope=branch]').select_option('B-HO')
        action(page, 'matrix-filter', '3,4')
        assert page.locator('tr[data-search]:visible').count() == 1
        page.locator('[data-filter]').fill('not-a-risk')
        assert page.locator('tr[data-search]:visible').count() == 0
        expect(page.locator('#no-match')).to_be_visible()
        nav(page, 'losses')
        page.locator('[data-loss-period]').select_option('30')
        page.wait_for_timeout(70)
        assert page.locator('tr[data-search]').count() == len(page.evaluate('CLEAR_SIGHT_EXPERIENCE.lossIds()'))
        page.close()
    record('Unknown-source fixture isolation and combined table/matrix/period filters', fixtures_filters)

    def keyboard_theme():
        page = newpage(browser, width=390, height=844)
        action(page, 'menu')
        expect(page.locator('.sidebar')).to_be_visible()
        page.keyboard.press('Escape')
        page.keyboard.press('Control+k')
        expect(page.locator('#dialog')).to_be_visible()
        expect(page.locator('#global-query')).to_be_focused()
        page.locator('#global-query').fill('Atlas')
        assert page.locator('#global-results .list-row').count() == 3
        page.keyboard.press('Escape')
        assert not page.locator('#dialog').evaluate('(x)=>x.open')
        action(page, 'theme')
        assert page.locator('html').get_attribute('data-theme') == 'dark'
        assert not page.evaluate('document.documentElement.scrollWidth>innerWidth')
        page.close()
    record('Mobile navigation, command search, Escape focus behavior and dark mode', keyboard_theme)

    def exports():
        page = newpage(browser)
        nav(page, 'losses')
        page.evaluate('() => { window.EXPORTED=[]; window.NativeBlob=Blob; window.Blob=class extends NativeBlob { constructor(parts,opt){super(parts,opt);EXPORTED.push({parts,type:opt?.type});}}; HTMLAnchorElement.prototype.click=function(){}; }')
        action(page, 'export-losses')
        payload = page.evaluate('EXPORTED.at(-1).parts.join("")')
        assert 'LOS-201' in payload and '850000000' in payload and ('U-IK-PAY' in payload)
        nav(page, 'reports')
        action(page, 'report-export')
        report = json.loads(page.evaluate('EXPORTED.at(-1).parts.join("")'))
        assert report['synthetic'] and report['losses']['gross'] - report['losses']['recoveries'] == report['losses']['net']
        page.close()
    record('CSV/JSON export payloads and reconciliation (download transport not exercised)', exports)
    browser.close()
report = {'runner': 'Python Playwright + Chromium; inline document rendering', 'native_navigation': 'Managed browser blocks URL navigation; file:// and HTTP serving were not exercised', 'persistence': 'Storage-API test double exercises serialization/recreation; native file-origin storage not tested', 'external_requests': 'None required', 'results': RESULT, 'javascript_errors': ERRORS, 'passed': sum((x['status'] == 'PASS' for x in RESULT)), 'failed': sum((x['status'] == 'FAIL' for x in RESULT))}
(ROOT / 'tests/core-regression-results.json').write_text(json.dumps(report, indent=2))
print(json.dumps({'passed': report['passed'], 'failed': report['failed'], 'errors': ERRORS}, indent=2))
sys.exit(bool(report['failed'] or ERRORS))
