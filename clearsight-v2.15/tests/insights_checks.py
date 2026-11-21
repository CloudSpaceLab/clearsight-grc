"""V2.5 browser integration tests. Local synthetic data; no real services."""
from pathlib import Path
import json, time, traceback, math
from playwright.sync_api import sync_playwright
ROOT=Path(__file__).resolve().parents[1]
HTML=(ROOT/'clearsight-v2-demo.html').read_text()
RESULTS=[];ERRORS=[];LAYOUTS=0
PREFIX='''<script>window.TEST_STORE={};Object.defineProperty(window,'localStorage',{value:{getItem:k=>TEST_STORE[k]??null,setItem:(k,v)=>TEST_STORE[k]=String(v),removeItem:k=>delete TEST_STORE[k]}});</script>'''
def test(name,fn):
 t=time.monotonic()
 try:fn();RESULTS.append({'name':name,'status':'PASS','seconds':round(time.monotonic()-t,2)});print('PASS',name,flush=True)
 except Exception as ex:RESULTS.append({'name':name,'status':'FAIL','error':str(ex)});print('FAIL',name,str(ex),flush=True);traceback.print_exc()
def new(w=1440):
 p=b.new_page(viewport={'width':w,'height':1050});p.set_default_timeout(4000);p.on('pageerror',lambda e:ERRORS.append(str(e)));p.set_content(PREFIX+HTML);return p
def nav(p,r):p.evaluate('(r)=>location.hash="/"+r',r);p.wait_for_timeout(30)
def act(p,a,extra=''):p.locator('[data-action="'+a+'"]'+extra).first.click();p.wait_for_timeout(30)
def state(p):return p.evaluate('CLEAR_SIGHT_DEMO.getState()')
def stats(p):return p.evaluate('CLEAR_SIGHT_INSIGHTS.getKriStats()')
def scope(p,kind,val):p.locator('[data-org-scope="'+kind+'"]').select_option(val);p.wait_for_timeout(30)
def metric(p,m,part,val):p.locator('[data-kri-metric="'+m+'"][data-kri-part="'+part+'"]').fill(str(val))
def review(p,decision='Approve return',who='Ifeoma Nwosu',note='Independent review of the scoped synthetic return and its sources.'):
 f=p.locator('form[data-ix-submit="kri-review"]');f.locator('[name=reviewer]').select_option(who);f.locator('[name=decision]').select_option(decision);f.locator('[name=note]').fill(note);f.locator('button[type=submit]').click();p.wait_for_timeout(30)
def start(p):
 nav(p,'kri-returns');act(p,'ix-start-return','[data-unit="U-EN-CUSTOMER"]')
def report(p):return p.evaluate('CLEAR_SIGHT_INSIGHTS.getReport()')
with sync_playwright() as pw:
 b=pw.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox'])
 def header():
  p=new()
  for w in [390,768,1024,1440]:
   p.set_viewport_size({'width':w,'height':1000})
   for selected in [False,True]:
    if selected:scope(p,'unit','U-IK-PAY')
    elif p.locator('[data-action=cx-reset-scope]').count():act(p,'cx-reset-scope')
    rect=p.locator('.topbar').bounding_box();logo=p.locator('.topbar .brand').bounding_box();assert logo, 'Logo missing from header'
    assert p.locator('.cx-scopebar').count()==1 and p.locator('.topbar .cx-scopebar').count()==1
    fields=p.locator('.topbar [data-org-scope]');assert fields.count()==3
    for el in fields.all():
     r=el.bounding_box();assert r and r['y']>=rect['y'] and r['y']+r['height']<=rect['y']+rect['height']+1;assert r['x']>=logo['x']+logo['width']-2
    assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1')
   nav(p,'privacy');assert p.locator('.topbar [data-org-scope=unit]').input_value()=='U-IK-PAY'
  p.close()
 test('Scope and logo share a single header row at four widths; selections persist',header)
 def aggregation():
  p=new();s=stats(p);assert (s['expected'],s['approved'],s['known'],s['expectedValues'],s['breaches'])==(4,2,6,12,2)
  r=report(p);m=next(x for x in r['kri']['consolidation'] if x['metric']=='failedPayments');assert m['numerator']==170 and m['denominator']==60000 and math.isclose(m['value'],170/60000*100)
  assert m['observed']==2 and not m['complete'];assert all(x['unit']!='U-IK-CUSTOMER' for x in r['kri']['values']);assert all(x['sourceRef'] and x['submission'] and x['approvedBy'] for x in r['kri']['values'])
  vals=p.evaluate('''()=>{const m=ixCycle().metrics[0],q=ixCycle().metrics[2];return [ixBand(m,{value:0}),ixBand(m,{value:10}),ixBand(m,{value:11}),ixBand(m,{}),ixBand(m,{value:-1}),ixBand(q,{numerator:0,denominator:100}),ixBand(q,{numerator:1,denominator:0}),ixBand(q,{numerator:101,denominator:100})]}''');assert vals==['Within limit','Near limit','Breached','Unknown','Unknown','Within limit','Unknown','Unknown']
  scope(p,'branch','B-HO');assert stats(p)['expected']==0;nav(p,'reports?view=kri');assert 'Not in cycle' in p.locator('main').inner_text();p.close()
 test('Approved-only KRI consolidation, ratio-of-sums, strict boundaries and absent population',aggregation)
 def totals():
  p=new();original=p.evaluate('scopeTotals(ixLossPeriod())');sums={'gross':0,'recovered':0,'net':0,'actual':0,'near':0}
  for branch in ['B-IK','B-EN','B-HO']:
   scope(p,'branch',branch);t=p.evaluate('scopeTotals(ixLossPeriod())')
   for k in sums:sums[k]+=t[k]
  assert all(sums[k]==original[k] for k in sums)
  act(p,'cx-reset-scope');groups=p.evaluate('ixLossGroups()');assert sum(x['net'] for x in groups)==original['net'];assert original['net']==2450000000
  nav(p,'insight-detail?kind=category&id='+groups[0]['id']);detail=p.evaluate('CLEAR_SIGHT_INSIGHTS.getDetail()');assert sum(l['gross']-sum(x['amount'] for x in l['recoveries']) for l in detail['losses'])==groups[0]['net']
  nav(p,'reports?view=executive&period=90');r90=report(p);nav(p,'reports?view=executive&period=30');r30=report(p);assert r90['sourceRecords']['risks']==r30['sourceRecords']['risks'];assert r90['kri']==r30['kri'];assert len(r30['sourceRecords']['losses'])<len(r90['sourceRecords']['losses']);p.close()
 test('Branch totals, loss-driver source reconciliation and separate posture/loss/KRI clocks',totals)
 def lifecycle():
  p=new();base=state(p);start(p);act(p,'ix-return-submit');assert 'non-negative' in p.locator('#ix-return-error').inner_text();assert stats(p)['approved']==2
  act(p,'ix-return-sample');assert p.locator('[data-kri-metric=reconciliation][data-kri-part=value]').input_value()=='0';act(p,'ix-return-submit');r=state(p)['branchKri']['returns']['KRC-2026-10::U-EN-CUSTOMER'];assert r['status']=='Submitted' and len(r['submissions'])==1;assert p.locator('[data-kri-metric]').count()==0;assert stats(p)['approved']==2
  assert any(w.get('source')=='branch-kri' and w['ref']==r['id'] for w in p.evaluate('workItems()'))
  review(p,who=r['owner']);assert 'submitter cannot' in p.locator('.form-error').inner_text();review(p,'Request changes');metric(p,'complaints','value',7);act(p,'ix-return-submit');r=state(p)['branchKri']['returns'][r['id']];assert len(r['submissions'])==2 and r['submissions'][0]['values']['complaints']['value']==6 and r['submissions'][1]['values']['complaints']['value']==7
  review(p);s=state(p);assert stats(p)['approved']==3;assert s['branchKri']['returns'][r['id']]['status']=='Approved';assert len([x for x in s['tasks'] if x.get('kriEpisode')==r['id']])==1;assert s['indicators']==base['indicators'];assert s['programs'][0]['evaluations']==base['programs'][0]['evaluations'];assert not any(w.get('source')=='branch-kri' and w['ref']==r['id'] for w in p.evaluate('workItems()'))
  nav(p,'reports?view=kri');assert stats(p)['breaches']==3;p.close()
 test('Return submit/change/resubmit/independent approval, immutable versions and shared Work',lifecycle)
 def missing():
  p=new();start(p);act(p,'ix-return-sample');metric(p,'failedPayments','numerator',2001);act(p,'ix-return-submit');assert 'cannot exceed' in p.locator('#ix-return-error').inner_text();metric(p,'failedPayments','numerator',8);metric(p,'failedPayments','denominator',0);act(p,'ix-return-submit');assert 'positive' in p.locator('#ix-return-error').inner_text()
  p.locator('[data-kri-metric=failedPayments][data-kri-part=missing]').check();act(p,'ix-return-submit');assert 'explain why' in p.locator('#ix-return-error').inner_text();metric(p,'failedPayments','reason','Transaction volume source was unavailable during reporting.');act(p,'ix-return-submit');review(p);s=stats(p);assert s['approved']==3 and s['known']==8 and s['unknown']==1;assert next(x for x in report(p)['kri']['values'] if x['unit']=='U-EN-CUSTOMER' and x['metric']=='failedPayments')['value'] is None
  scope(p,'branch','B-IK');assert p.locator('h1').inner_text()=='Outside the selected scope';assert p.locator('[data-kri-metric]').count()==0;act(p,'cx-reset-scope');assert 'Enugu' in p.locator('main').inner_text();p.close()
 test('Missing is not zero; ratio validation; approved unknowns remain coverage gaps; scope guard',missing)
 def cycles():
  p=new();nav(p,'kri-returns');act(p,'ix-new-cycle');p.locator('[data-ix-cycle-draft=period]').fill('2026-10');act(p,'ix-cycle-step','[data-step="2"]');act(p,'ix-cycle-step','[data-step="3"]');assert 'already have' in p.locator('#ix-cycle-error').inner_text()
  act(p,'ix-cycle-step','[data-step="1"]');p.locator('[data-ix-cycle-draft=period]').fill('2026-11');act(p,'ix-cycle-step','[data-step="2"]');p.locator('[data-ix-cycle-draft=reviewer]').select_option('Amina Bello');act(p,'ix-cycle-step','[data-step="3"]');assert 'must be different' in p.locator('#ix-cycle-error').inner_text();p.locator('[data-ix-cycle-draft=reviewer]').select_option('Ifeoma Nwosu');pop=state(p)['branchKri']['draftCycle']['population'];scope(p,'unit','U-EN-CUSTOMER');assert state(p)['branchKri']['draftCycle']['population']==pop;act(p,'ix-cycle-step','[data-step="3"]');act(p,'ix-create-cycle');c=state(p)['branchKri']['cycles'][0];assert c['population']==pop and c['metrics'][0]['limit']==10;act(p,'ix-start-return','[data-unit="U-EN-CUSTOMER"]');act(p,'ix-return-sample');act(p,'ix-return-submit');assert 'period has not ended' in p.locator('#ix-return-error').inner_text();assert state(p)['branchKri']['returns'][c['id']+'::U-EN-CUSTOMER']['status']=='Draft';p.close()
 test('Cycle wizard pins unit population/definitions and prevents duplicates/self-review/early submission',cycles)
 def saved():
  p=new();scope(p,'branch','B-IK');nav(p,'reports?view=drivers&period=30&group=owner');p.locator('[data-ix-cycle]').select_option('KRC-2026-09');act(p,'ix-save-view');p.locator('#modal-form [name=name]').fill('Ikeja owner perspective');p.locator('#modal-form button[type=submit]').click();v=state(p)['savedInsightViews'][0];act(p,'cx-reset-scope');nav(p,'reports?view=library&period=365');act(p,'ix-open-view','[data-id="'+v['id']+'"]');assert p.locator('[data-org-scope=branch]').input_value()=='B-IK';assert report(p)['periods']['loss']['days']==30;assert report(p)['periods']['kri']['month']=='2026-09';assert 'group=owner' in p.url;p.close()
 test('Saved live view restores scope, periods, perspective and grouping',saved)
 def frozen():
  p=new();nav(p,'reports');act(p,'ix-new-report');snap=state(p)['reportDraft']['snapshot'];act(p,'ix-report-step','[data-step="2"]');p.locator('[data-ix-report-draft=commentary]').fill('Review missing branch returns before drawing bank-wide conclusions.');scope(p,'branch','B-IK');assert state(p)['reportDraft']['snapshot']==snap;act(p,'ix-report-step','[data-step="3"]');act(p,'ix-publish-report');pack=state(p)['reportPacks'][0];assert pack['snapshot']==snap;act(p,'cx-reset-scope');nav(p,'kri-return?id=KRC-2026-10%3A%3AU-IK-CUSTOMER');review(p);assert stats(p)['approved']==3;assert state(p)['reportPacks'][0]==pack;nav(p,'report-pack?id='+pack['id']);assert p.locator('h1').inner_text()==pack['name']
  p.evaluate('()=>{window.TEST_DOWNLOADS=[];download=(name,body,type)=>TEST_DOWNLOADS.push({name,body,type});}');act(p,'ix-export-pack');payload=p.evaluate('TEST_DOWNLOADS[0]');assert json.loads(payload['body'])['snapshot']['kri']['approved']==2;p.close()
 test('Guided management report captures fixed source/scope snapshot unaffected by later approvals',frozen)
 def export():
  p=new();nav(p,'kri-returns');p.evaluate('()=>{window.TEST_DOWNLOADS=[];download=(name,body,type)=>TEST_DOWNLOADS.push({name,body,type});}');act(p,'ix-export-kri');x=p.evaluate('TEST_DOWNLOADS[0]');assert 'denominator' in x['body'] and 'submission_id' in x['body'] and 'KRC-2026-10::U-IK-PAY::v1' in x['body'];assert len(x['body'].strip().splitlines())==7
  scope(p,'branch','B-IK');act(p,'ix-export-kri');x=p.evaluate('TEST_DOWNLOADS.at(-1)');assert len(x['body'].strip().splitlines())==4 and 'U-EN-' not in x['body'];p.close()
 test('KRI CSV payload is accepted-only, scoped and traceable to numerator/denominator/source/version',export)
 def views():
  global LAYOUTS
  p=new();routes=list(json.loads((ROOT/'docs/route-manifest.json').read_text()).values());routes+=['reports?view='+v for v in ['executive','locations','drivers','kri','assurance','dependencies','quality','library']];routes+=['kri-return?id=KRC-2026-10%3A%3AU-IK-CUSTOMER','kri-return?id=KRC-2026-10%3A%3AU-IK-PAY','insight-detail?kind=appetite','insight-detail?kind=actions','insight-detail?kind=category&id=Technology']
  for w in [390,768,1024,1440]:
   p.set_viewport_size({'width':w,'height':1000})
   for route in routes:
    nav(p,route);assert p.locator('h1').count()==1,(w,route,'heading');assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),(w,route,'overflow');LAYOUTS+=1
   scope(p,'unit','U-IK-PAY')
   for v in ['executive','locations','drivers','kri','assurance','dependencies','quality','library']:
    nav(p,'reports?view='+v);assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),(w,v,'scoped overflow');LAYOUTS+=1
   act(p,'cx-reset-scope')
  assert not ERRORS,ERRORS;p.close()
 test('All 50 routes plus insights/return/source states and scoped perspectives at four widths',views)
 def populated():
  global LAYOUTS
  p=new();nav(p,'kri-returns');act(p,'ix-new-cycle');act(p,'ix-new-report') if p.locator('[data-action=ix-new-report]').count() else None
  nav(p,'reports');act(p,'ix-new-report')
  for w in [390,768,1024,1440]:
   p.set_viewport_size({'width':w,'height':1000})
   for route,action in [('kri-cycle','ix-cycle-step'),('report-builder','ix-report-step')]:
    nav(p,route)
    for step in [1,2,3]:
     act(p,action,'[data-step="'+str(step)+'"]');assert p.locator('h1').count()==1;assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),(w,route,step);LAYOUTS+=1
   nav(p,'kri-returns');act(p,'ix-start-return','[data-unit="U-EN-CUSTOMER"]') if p.locator('[data-action=ix-start-return][data-unit="U-EN-CUSTOMER"]').count() else nav(p,'kri-return?id=KRC-2026-10%3A%3AU-EN-CUSTOMER');act(p,'ix-return-sample');assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1');LAYOUTS+=1
  act(p,'theme');nav(p,'reports?view=kri');assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1');p.close()
 test('Populated KRI and report wizards, editable return grid and dark mode at four widths',populated)
 b.close()
(ROOT/'tests/insights-results.json').write_text(json.dumps({'version':'2.7','tests':RESULTS,'layoutChecks':LAYOUTS,'pageErrors':ERRORS,'limitations':['Inline browser rendering with explicit Storage-API test double.','Export payloads checked; native download transport not exercised.','Synthetic records; no live backend or production authorization.']},indent=2))
print(json.dumps({'pass':sum(x['status']=='PASS' for x in RESULTS),'fail':sum(x['status']=='FAIL' for x in RESULTS),'layouts':LAYOUTS,'pageErrors':ERRORS}),flush=True)
