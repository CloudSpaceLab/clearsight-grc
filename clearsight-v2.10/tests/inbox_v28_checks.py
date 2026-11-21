"""Meaningful local-flow checks for the V2.8 notification centre.
Uses the same no-network Chromium/set_content harness as existing suites.
"""
from pathlib import Path
import json, os, time
from playwright.sync_api import sync_playwright, expect
ROOT=Path(__file__).resolve().parents[1]
HTML=(ROOT/'clearsight-v2-demo.html').read_text()
if 'window.CS_V28_EVENTS=' not in HTML:
    HTML=HTML.replace('recordsInit();docSeed();render();', (ROOT/'assets/inbox.js').read_text()+'\nrecordsInit();docSeed();render();')
    HTML=HTML.replace('</head>','<style>'+(ROOT/'assets/inbox.css').read_text()+'</style></head>')
PREFIX='<script>const STORE={};Object.defineProperty(window,"localStorage",{value:{getItem:k=>STORE[k]??null,setItem:(k,v)=>STORE[k]=String(v),removeItem:k=>delete STORE[k]}});window.TEST_STORE=STORE;</script>'
RESULT=[]
with sync_playwright() as p:
    browser=p.chromium.launch(executable_path=os.environ.get('CHROMIUM_PATH','/usr/bin/chromium'),headless=True,args=['--no-sandbox'])
    page=browser.new_page(viewport={'width':1440,'height':1100})
    page.set_default_timeout(6000)
    errors=[]
    page.on('pageerror',lambda err:errors.append(str(err)))
    page.set_content(HTML.replace('<head>','<head>'+PREFIX,1))
    def nav(route):
        page.evaluate('(route)=>{location.hash="/"+route;render()}',route)
        page.wait_for_timeout(40)
    def test(name,fn):
        try:
            fn();RESULT.append({'test':name,'status':'PASS'});print('PASS',name,flush=True)
        except Exception as err:
            RESULT.append({'test':name,'status':'FAIL','error':str(err)});print('FAIL',name,str(err),flush=True)
    def initial():
        nav('inbox')
        expect(page.locator('h1')).to_have_text('Inbox')
        assert page.locator('.n-row').count()>5
        assert page.evaluate('inboxUnreadCount()')>0
        assert not errors, errors
    test('Recipient inbox renders all source categories and header unread badge',initial)
    def reads():
        nav('notification?id=NTF-MENTION-01')
        assert page.evaluate("!!nReceipt(nEvent('NTF-MENTION-01')).readAt")
        page.locator('[data-action="n-mark-unread"]').click()
        page.wait_for_timeout(50)
        assert not page.evaluate("nReceipt(nEvent('NTF-MENTION-01')).readAt")
        assert page.locator('.n-row[data-notification-id="NTF-MENTION-01"]').count()==1
    test('Opening marks read; Mark unread returns to inbox and preserves state',reads)
    def search_scope():
        page.locator('[data-n-search]').fill('access review')
        assert page.locator('.n-row').count()>=1
        assert page.locator('.n-row').count()<page.evaluate('CLEAR_SIGHT_INBOX.getVisible().length')
        page.locator('[data-n-search]').fill('')
        page.evaluate("scopeChange('branch','B-IK')")
        rows=page.evaluate('CLEAR_SIGHT_INBOX.getVisible().map(x=>({id:x.id,scopes:nScopeIds(x)}))')
        assert all(any(s.startswith('B-IK') or '-IK-' in s for s in row['scopes']) for row in rows),rows
        assert not page.locator('.n-row[data-notification-id="NTF-MENTION-01"]').count()
        nav('notification?id=NTF-MENTION-01')
        expect(page.locator('h1')).to_have_text('Notification unavailable')
        page.evaluate("scopeChange('branch','')")
        nav('inbox')
    test('Search and organizational scope apply to list and direct detail routes',search_scope)
    def recipient():
        page.locator('[data-n-actor]').select_option('USR-03')
        ids=page.evaluate('CLEAR_SIGHT_INBOX.getVisible().map(x=>x.id)')
        assert 'NTF-MENTION-01' not in ids
        assert any(x.startswith('NTF-EV-REVIEW') for x in ids)
        page.locator('[data-n-actor]').select_option('USR-01')
    test('Recipient switch isolates personal notifications and approval queue',recipient)
    def snooze():
        nav('notification?id=NTF-MENTION-01')
        page.locator('[data-action="n-snooze"]').click()
        page.locator('#modal-form [name="duration"]').select_option('hour')
        page.locator('#modal-form button[type="submit"]').click()
        page.wait_for_timeout(50)
        assert page.evaluate("!!nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil")
        assert not page.locator('.n-row[data-notification-id="NTF-MENTION-01"]').count()
        page.locator('[data-action="n-view"][data-view="snoozed"]').click()
        assert page.locator('.n-row[data-notification-id="NTF-MENTION-01"]').count()==1
        page.evaluate("nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil='2000-01-01T00:00:00Z';render()")
        assert not page.evaluate("nReceipt(nEvent('NTF-MENTION-01')).snoozedUntil")
        page.locator('[data-action="n-view"][data-view="inbox"]').click()
    test('Snooze hides an item until due and then resumes automatically on render',snooze)
    def archive():
        page.locator('[data-action="n-archive"][data-id="NTF-MENTION-01"]').click()
        assert page.evaluate("!!nReceipt(nEvent('NTF-MENTION-01')).archivedAt")
        page.locator('[data-action="n-view"][data-view="archive"]').click()
        assert page.locator('.n-row[data-notification-id="NTF-MENTION-01"]').count()==1
        page.locator('[data-action="n-archive"][data-id="NTF-MENTION-01"]').click()
        assert not page.evaluate("nReceipt(nEvent('NTF-MENTION-01')).archivedAt")
        page.locator('[data-action="n-view"][data-view="inbox"]').click()
        page.locator('[data-n-all]').check()
        page.locator('[data-action="n-bulk"][data-operation="read"]').click()
        assert page.evaluate('inboxUnreadCount()')==0
    test('Archive/restore and bulk read persist recipient-only state',archive)
    def reply():
        nav('notification?id=NTF-MENTION-01')
        page.locator('[name="reply"]').fill('Chidi will provide a revised extract and closure evidence tomorrow.')
        page.locator('[name="mention"]').select_option('USR-02')
        page.locator('form[data-n-form="reply"] button[type="submit"]').click()
        assert page.evaluate("S.notificationsV28.events.find(x=>x.id==='NTF-MENTION-01').thread.length")>=2
        assert page.evaluate("S.notificationsV28.events.some(x=>x.threadRootId==='NTF-MENTION-01'&&x.recipientIds.includes('USR-02'))")
        page.locator('[data-action="n-resolve-thread"]').click()
        page.locator('#modal-form [name="outcome"]').fill('The exception owner and required source evidence are now agreed.')
        page.locator('#modal-form button[type="submit"]').click()
        assert page.evaluate("!!S.notificationsV28.events.find(x=>x.id==='NTF-MENTION-01').threadResolvedAt")
        assert page.evaluate("S.programs.find(x=>x.id==='PRG-104').evaluations['Meridian Bank']['R03'].status")=='Partially satisfied'
    test('Replies notify mentioned colleagues; discussion resolution preserves source decision',reply)
    def followup():
        nav('notification?id=NTF-QUALITY-01')
        original=page.evaluate('S.tasks.length')
        page.locator('[data-action="n-follow-up"]').click()
        page.locator('#modal-form [name="owner"]').select_option('USR-02')
        page.locator('#modal-form [name="outcome"]').fill('Upload the original privileged account extract and retain the source version.')
        page.locator('#modal-form button[type="submit"]').click()
        assert page.evaluate('S.tasks.length')==original+1
        task=page.evaluate("S.tasks.find(t=>t.notificationId==='NTF-QUALITY-01')")
        assert task['ownerId']=='USR-02' and task['orgScopeId']=='U-HO-SEC'
        page.locator('[data-action="n-follow-up"]').click()
        page.wait_for_timeout(50)
        assert page.evaluate('S.tasks.length')==original+1
        assert page.evaluate('current.name')=='work'
    test('Follow-up creates one assigned scoped task; repeat opens existing work',followup)
    def preferences():
        nav('notification-preferences')
        page.locator('[name="inApp-program"]').uncheck()
        page.locator('[name="email"]').select_option('Off')
        page.locator('form[data-n-form="preferences"] button[type="submit"]').click()
        assert not page.evaluate('nPrefs().inApp.program')
        assert page.locator('[name="inApp-escalation"]').is_disabled()
        page.evaluate("CS_V28_EVENTS.emit({id:'TEST-OPTIONAL',type:'program',title:'Optional update',sourceRoute:'program?id=PRG-104',sourceId:'PRG-104',recipient:'USR-01'});CS_V28_EVENTS.emit({id:'TEST-CRITICAL',type:'program',title:'Critical update',severity:'Critical',sourceRoute:'program?id=PRG-104',sourceId:'PRG-104',recipient:'USR-01'});")
        ids=page.evaluate('CLEAR_SIGHT_INBOX.getVisible().map(x=>x.id)')
        assert 'TEST-OPTIONAL' not in ids and 'TEST-CRITICAL' in ids
        assert 'NTF-PROGRAM-01' in ids
    test('Preferences suppress future optional updates and retain mandatory alerts/history',preferences)
    def source_resolution():
        page.evaluate("S.tasks.find(x=>x.id==='ACT-03').status='Completed';render()")
        assert page.evaluate("CLEAR_SIGHT_INBOX.isResolved('NTF-ESC-ACT-03')")
        nav('notification?id=NTF-ESC-ACT-03')
        assert page.get_by_text('Source resolved',exact=True).count()==1
    test('Escalation resolution follows completion in the source domain',source_resolution)
    def responsive():
        nav('inbox')
        page.set_viewport_size({'width':390,'height':844})
        assert page.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
        page.screenshot(path=str(ROOT/'screenshots/inbox-v28-mobile.png'),full_page=True)
        page.set_viewport_size({'width':1440,'height':1100})
        page.screenshot(path=str(ROOT/'screenshots/inbox-v28-desktop.png'),full_page=True)
        assert not errors,errors
    test('Mobile and desktop layouts remain within viewport without runtime errors',responsive)
    browser.close()
(ROOT/'tests/inbox-v28-results.json').write_text(json.dumps({'checks':RESULT,'pageErrors':errors},indent=2))
if any(x['status']=='FAIL' for x in RESULT):raise SystemExit(1)
