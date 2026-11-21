"""V2.4 browser workflow tests. Synthetic data. No remote services."""
from pathlib import Path
import json,time,traceback,os
from playwright.sync_api import sync_playwright
ROOT=Path(__file__).resolve().parents[1]; HTML=(ROOT/'clearsight-v2-demo.html').read_text().replace('data-start="reports"','data-start="privacy"'); RESULTS=[]; ERRORS=[]
def test(name,fn):
 if os.environ.get('TEST_FILTER') and os.environ['TEST_FILTER'].lower() not in name.lower():return
 t=time.monotonic()
 try:fn();RESULTS.append({'name':name,'status':'PASS','seconds':round(time.monotonic()-t,2)});print('PASS',name,flush=True)
 except Exception as ex:
  status='BLOCKED' if 'ERR_BLOCKED_BY_ADMINISTRATOR' in str(ex) else 'FAIL'
  RESULTS.append({'name':name,'status':status,'error':str(ex)});print(status,name,str(ex),flush=True)
  if status=='FAIL':traceback.print_exc()
def new(store=None,w=1440):
 p=b.new_page(viewport={'width':w,'height':1050},accept_downloads=True);p.set_default_timeout(4500);p.on('pageerror',lambda e:ERRORS.append(str(e)))
 prefix='<script>window.TEST_STORE='+json.dumps(store or {})+';Object.defineProperty(window,"localStorage",{value:{getItem:k=>TEST_STORE[k]??null,setItem:(k,v)=>TEST_STORE[k]=String(v),removeItem:k=>delete TEST_STORE[k]}});</script>'
 p.set_content(prefix+HTML);return p
def nav(p,r):p.evaluate('(r)=>location.hash="/"+r',r);p.wait_for_timeout(35)
def act(p,name,extra=''):p.locator('[data-action="'+name+'"]'+extra).first.click();p.wait_for_timeout(35)
def state(p):return p.evaluate('CLEAR_SIGHT_DEMO.getState()')
def jf(p,path,val):
 el=p.locator('[data-j-path="'+path+'"]');(el.select_option if el.evaluate('(e)=>e.tagName')=='SELECT' else el.fill)(str(val));p.wait_for_timeout(20)
def nf(p,name,val,root='form[data-j-submit]'):
 el=p.locator(root+' [name="'+name+'"]');(el.select_option if el.evaluate('(e)=>e.tagName')=='SELECT' else el.fill)(str(val))
def dec(p):p.locator('form[data-j-submit] button[type="submit"]').click();p.wait_for_timeout(40)
def modal(p,name,val):nf(p,name,val,'#modal-form')
def msubmit(p):p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(40)
def close(p):
 if p.locator('#dialog').evaluate('(d)=>d.open'):act(p,'close-dialog')
def err(p):return p.locator('.j-error').all_text_contents()
def advance(p):act(p,'j-next')
def draft_id(p):return p.evaluate('new URLSearchParams(location.hash.split("?")[1]).get("draft")')
with sync_playwright() as pw:
 b=pw.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox'])
 def blank_guard():
  p=new();nav(p,'privacy');act(p,'new-processing');assert not p.locator('#dialog').evaluate('(d)=>d.open');advance(p);assert 'Name the processing activity' in ''.join(err(p));assert 'step=1' in p.url
  jf(p,'name','Saved incomplete activity');act(p,'j-save-exit');assert p.locator('text=Saved incomplete activity').count()>0
  store=p.evaluate('TEST_STORE');p.close();p=new(store);nav(p,'privacy?tab=register');p.get_by_role('link',name='Resume',exact=True).click();assert p.locator('[data-j-path=name]').input_value()=='Saved incomplete activity';p.close()
 test('ROPA incomplete-step validation, full-page creation, save/resume and storage reconstruction',blank_guard)
 def register():
  p=new();nav(p,'privacy');act(p,'new-processing');jf(p,'name','Branch reconciliation record');jf(p,'orgScopeId','U-IK-PAY');jf(p,'description','Reconcile customer payment instructions with posted transactions.');advance(p)
  p.locator('[data-j-array=subjects][value="Customers"]').check();p.locator('[data-j-array=dataCategories][value="Account / transaction data"]').check();jf(p,'collection','Customer payment instructions and core banking postings.');jf(p,'volume','5000 daily transactions');jf(p,'purposes.0.purpose','Payment reconciliation');jf(p,'purposes.0.basis','Contract');jf(p,'purposes.0.reference','DEMO-CONTRACT-001');advance(p)
  jf(p,'systems.0.name','Reconciliation service');advance(p)
  for k,v in {'dataset':'Transaction records','trigger':'End of reconciliation cycle','period':'3 years under illustrative policy','disposal':'Approved deletion after hold checks','reference':'RET-DEMO-04'}.items():jf(p,'retentions.0.'+k,v)
  p.locator('[data-j-array=controlIds][value=CTL-05]').check();jf(p,'notice','NOTICE-DEMO-04');jf(p,'rightsRoute','DPO request desk');advance(p)
  for k in ['digital','profiling','sensitive','monitoring','vulnerable','innovation']:jf(p,'screening.'+k,'Yes' if k=='digital' else 'No')
  jf(p,'screeningContext','Digital financial activity; DPO to review GAID requirements.');advance(p);act(p,'j-submit');a=state(p)['processing'][0];assert a['state']=='In review' and not a['published'] and a['revision']==0
  aid=a['id'];assert any(x.get('ref')==aid for x in p.evaluate('workItems()'))
  nf(p,'reviewer','Sarah Adams');nf(p,'decision','Publish register revision');nf(p,'rationale','Reviewed the processing record and confirmed the assessment requirement.');dec(p);assert 'cannot approve' in ''.join(err(p))
  nf(p,'reviewer','Ifeoma Nwosu');dec(p);a=state(p)['processing'][0];assert a['published'] and a['revision']==1 and a['screeningDecision']=='DPIA required';assert not any(x.get('ref')==aid and x.get('source')=='privacy' for x in p.evaluate('workItems()'))
  assert state(p)['programs'][0]['evaluations']==p.evaluate('CS_SEED.programs[0].evaluations');p.close()
 test('ROPA full registration, source-linked data, independent review and no compliance auto-pass',register)
 def revision():
  p=new();nav(p,'activity?id=ROPA-02');base=next(x for x in state(p)['processing'] if x['id']=='ROPA-02');act(p,'j-edit-activity');jf(p,'name','Transaction monitoring · revised');assert next(x for x in state(p)['processing'] if x['id']=='ROPA-02')['name']==base['name']
  act(p,'j-step','[data-step="6"]');act(p,'j-submit');x=next(x for x in state(p)['processing'] if x['id']=='ROPA-02');assert x['name']==base['name'] and x['pending']['data']['name'].endswith('revised')
  nf(p,'decision','Request changes');nf(p,'rationale','Clarify the data boundary and response obligations before publication.');dec(p);nav(p,'activity?id=ROPA-02');act(p,'j-edit-activity');jf(p,'description','Revised monitoring boundary with human review and explicit escalation limits.');act(p,'j-step','[data-step="6"]');act(p,'j-submit');nf(p,'decision','Publish register revision');nf(p,'rationale','Revised boundary, notice and purpose have been independently reviewed.');dec(p)
  x=next(x for x in state(p)['processing'] if x['id']=='ROPA-02');assert x['revision']==2 and x['revisions'][0]['data']['name']==base['name'];assert 'pending' not in x
  nav(p,'activity-version?id=ROPA-02&revision=1');assert p.locator('h1').inner_text()==base['name'];p.close()
 test('Published ROPA revision isolation, change request, resubmission and historical snapshot',revision)
 def role():
  p=new();nav(p,'privacy');act(p,'new-processing');jf(p,'name','Payroll processing service');jf(p,'description','Process payroll on another controller’s instructions.');jf(p,'orgScopeId','U-HO-COMP');jf(p,'role','Processor');advance(p);assert 'Identify the controller' in ''.join(err(p));jf(p,'controllerName','Synthetic parent organisation');jf(p,'controllerContact','DIRECTORY-CONTROLLER-01');advance(p);assert 'step=2' in p.url
  id=draft_id(p);p.locator('[data-org-scope=unit]').select_option('U-EN-REC');assert state(p)['journeyDrafts'][id]['data']['orgScopeId']=='U-HO-COMP';p.close()
 test('Controller/processor conditional fields and pinned draft scope',role)
 def dpia():
  p=new();nav(p,'dpia?id=DPIA-01');act(p,'j-edit-dpia');act(p,'j-step','[data-step="3"]');jf(p,'risks.0.residualLikelihood','4');jf(p,'risks.0.residualSeverity','5');act(p,'j-step','[data-step="5"]');jf(p,'recommendation','Submit for independent review with unresolved measures and elevated residual risk.');act(p,'j-submit');nf(p,'decision','Approve assessment');nf(p,'note','Reviewed the assessment and proposed privacy safeguards.');dec(p);assert 'High residual' in ''.join(err(p));assert any(t.get('privacyMeasure','').startswith('DPIA-01::') for t in state(p)['tasks'])
  nf(p,'decision','Request changes');dec(p);nav(p,'dpia?id=DPIA-01');act(p,'j-edit-dpia');act(p,'j-step','[data-step="3"]');jf(p,'risks.0.residualLikelihood','1');jf(p,'risks.0.residualSeverity','4');advance(p);jf(p,'risks.0.implemented','Verified');jf(p,'risks.0.evidence','EV-SYNTHETIC-VERIFIED-01');jf(p,'externalAction','Recorded as complete');jf(p,'externalRef','SYNTHETIC-FILING-REF-01');advance(p);jf(p,'recommendation','Verified mitigations and recorded external obligations support independent assessment approval.');act(p,'j-submit');nf(p,'reviewer','Sarah Adams');nf(p,'decision','Approve assessment');nf(p,'note','Verified measures and external obligations considered in the reviewed assessment.');dec(p);assert 'cannot approve' in ''.join(err(p));nf(p,'reviewer','Ifeoma Nwosu');dec(p)
  x=next(x for x in state(p)['dpias'] if x['id']=='DPIA-01');assert x['state']=='Approved';assert all(t['status']=='Completed' for t in state(p)['tasks'] if t.get('privacyMeasure','').startswith('DPIA-01::'));p.close()
 test('DPIA risks, remediation creation, high-risk blocking, independent approval and verified action reconciliation',dpia)
 def rights():
  p=new();nav(p,'privacy-case?id=PRIV-REQ-01');nf(p,'outcome','Prepared a synthetic customer access response with reviewed exclusions.');nf(p,'evidenceRef','PACKAGE-DEMO-01');nf(p,'status','Submit for independent review');dec(p);assert 'Confirm identity' in ''.join(err(p));nf(p,'identity','Verified');nf(p,'identityRef','IDV-DEMO-01');dec(p);nf(p,'reviewer','Sarah Adams');nf(p,'decision','Close case');nf(p,'note','Reviewed the requested scope and response before recorded closure.');nf(p,'deliveryRef','DELIVERY-DEMO-01');dec(p);assert 'cannot independently approve' in ''.join(err(p));nf(p,'reviewer','Ifeoma Nwosu');dec(p);assert state(p)['privacyCases'][0]['state']=='Closed';p.close()
 test('Rights-request identity and disclosure-review gates, independent closure',rights)
 def incident():
  p=new();nav(p,'privacy?tab=cases');act(p,'j-new-case','[data-kind="Privacy incident"]');jf(p,'title','Synthetic privacy event');jf(p,'description','Potentially misdirected extract; scope of exposure is not yet established.');advance(p);jf(p,'deadline','2026-11-14');jf(p,'deadlineBasis','Internal triage checkpoint; DPO must assess statutory clock separately.');jf(p,'containment','Access revoked and audit trail preserved for investigation.');jf(p,'harm','Possible exposure of financial information; recipient access remains unknown.');advance(p);act(p,'j-submit');x=state(p)['privacyCases'][0];assert x['state']=='Triage' and x['notify']=='Pending determination'
  nf(p,'outcome','Investigation found no access outside the authorised purpose; evidence was retained.');nf(p,'evidenceRef','EV-INCIDENT-DEMO-01');nf(p,'status','Submit for independent review');dec(p);nf(p,'decision','Close case');nf(p,'note','Reviewed notification applicability and the investigation evidence.');nf(p,'externalRef','NON-NOTIFICATION-LEGAL-DEMO-01');dec(p);assert 'notification determination' in ''.join(err(p));nf(p,'notify','Not required — reviewed basis');dec(p);assert state(p)['privacyCases'][0]['state']=='Closed';p.close()
 test('Incident intake, explicit awareness and target basis, notification-decision gate',incident)
 def riskloss():
  p=new();nav(p,'risks');p.locator('[data-org-scope=unit]').select_option('U-IK-PAY');act(p,'new-risk');jf(p,'title','Synthetic payment control gap');jf(p,'cause','A missing review in the settlement process');jf(p,'event','Incorrect instruction released');jf(p,'impact','Financial loss and customer delay');advance(p);jf(p,'control','CTL-03');advance(p);act(p,'j-submit');r=state(p)['risks'][0];assert r['orgScopeId']=='U-IK-PAY' and p.locator('h1').inner_text()==r['title'];assert not p.locator('#dialog').evaluate('(d)=>d.open')
  act(p,'new-loss');jf(p,'title','Synthetic settlement error');jf(p,'note','Incorrect instruction caused a confirmed booked loss.');advance(p);jf(p,'gross','123.45');advance(p);act(p,'j-submit');l=state(p)['losses'][0];assert l['gross']==12345 and l['risk']==r['id'];act(p,'add-recovery');modal(p,'amount','200');modal(p,'note','TEST-RECOVERY');msubmit(p);assert 'cannot exceed' in p.locator('#modal-error').inner_text();modal(p,'amount','23.45');msubmit(p);l=state(p)['losses'][0];assert sum(x['amount'] for x in l['recoveries'])==2345;assert p.locator('.j-facts').inner_text().count('₦100')==1;p.close()
 test('Risk and loss setup wizards, scoped records, exact kobo arithmetic and bounded recovery',riskloss)
 def vendor():
  p=new();nav(p,'vendors');before=len(state(p)['vendors']);act(p,'new-vendor');jf(p,'supplier',state(p)['vendors'][0]['id']);advance(p);jf(p,'service','Synthetic archive hosting');jf(p,'data','Encrypted document archive with restricted retrieval.');advance(p);jf(p,'orgScopeId','U-HO-INFRA');advance(p);act(p,'j-submit');s=state(p);assert len(s['vendors'])==before;assert s['engagements'][0]['status']=='Onboarding' and s['engagements'][0]['orgScopeId']=='U-HO-INFRA';assert not p.locator('#dialog').evaluate('(d)=>d.open');p.close()
 test('Vendor onboarding reuses supplier identity and separates activation from registration',vendor)
 def cycle():
  p=new();nav(p,'rcsa');p.locator('[data-org-scope=branch]').select_option('B-IK');act(p,'new-cycle');jf(p,'name','Ikeja scope freeze');advance(p);jf(p,'reviewer','Chidi Okafor');advance(p);assert 'must be different' in ''.join(err(p));jf(p,'reviewer','Ifeoma Nwosu');advance(p);act(p,'j-submit');c=state(p)['cycles'][0];assert c['risks']==['RSK-04','RSK-05'];assert len(c['frozenPopulation'])==2;assert c['reviews']=={};p.close()
 test('RCSA setup freezes risk/control population and enforces independent responsibility',cycle)
 def scope_preserve():
  p=new();nav(p,'privacy?tab=register');p.locator('[data-filter]').fill('digital');p.locator('a[href="#/activity?id=ROPA-01"]').first.click();p.wait_for_timeout(30);nav(p,'privacy?tab=register');assert p.locator('[data-filter]').input_value()=='digital';p.locator('[data-org-scope=branch]').select_option('B-EN');assert p.locator('tr[data-search]').count()==2;nav(p,'activity?id=ROPA-01');assert p.locator('h1').inner_text()=='Outside the selected scope';act(p,'cx-reset-scope');assert p.locator('h1').inner_text()=='Digital account opening';p.close()
 test('Privacy scope isolation, descendant filtering and register return-context preservation',scope_preserve)
 def evidence():
  p=new();nav(p,'evidence-review?id=EV-02');x=next(x for x in state(p)['evidence'] if x['id']=='EV-02');assert not p.locator('#dialog').evaluate('(d)=>d.open')
  # Seed may have historical acceptance; make another submitted metadata record via existing UI.
  nav(p,'assessment?id=PRG-104');act(p,'fill-assessment');msubmit(p);act(p,'assessment-submit');msubmit(p)
  ev=next(x for x in state(p)['evidence'] if x['program']=='PRG-104' and x['status']!='Accepted' and x['expires']>='2026-11-13')
  nav(p,'evidence-review?id='+ev['id']);nf(p,'reviewer',ev['owner']);nf(p,'decision','Accept');nf(p,'note','Reviewed the synthetic metadata in the correct program and target.');dec(p);assert 'contributor cannot' in ''.join(err(p));nf(p,'reviewer','Ifeoma Nwosu' if ev['owner']!='Ifeoma Nwosu' else 'Daniel Mensah');dec(p);assert next(x for x in state(p)['evidence'] if x['id']==ev['id'])['status']=='Accepted'
  nav(p,'requirement-review?id=PRG-104&target=Meridian%20Bank&requirement=R03');assert p.locator('form[data-j-submit=requirement-decision]').count()==1;assert not p.locator('#dialog').evaluate('(d)=>d.open');p.close()
 test('Evidence review becomes a full workspace and retains independent acceptance guards',evidence)
 def partial_case():
  p=new();nav(p,'privacy?tab=cases');p.locator('[data-org-scope=unit]').select_option('U-EN-REC');act(p,'j-new-case','[data-kind="Privacy incident"]');assert p.locator('[data-j-path=activity]').input_value()==''
  jf(p,'title','Unclassified privacy incident');jf(p,'description','An exposure may have occurred; the responsible system is not identified yet.');advance(p);jf(p,'deadline','2026-11-14');jf(p,'deadlineBasis','Internal triage checkpoint; legal applicability is pending.');jf(p,'containment','Restrict access and preserve investigation logs.');jf(p,'harm','Unknown pending investigation.');advance(p);act(p,'j-submit');c=state(p)['privacyCases'][0];assert c['activity']=='' and c['orgScopeId']=='U-EN-REC';assert p.locator('h1').inner_text()=='Unclassified privacy incident'
  dec(p);assert state(p)['privacyCases'][0]['state']=='In progress' and state(p)['privacyCases'][0]['outcome']=='';p.close()
 test('Unclassified incident intake without a ROPA prerequisite and partial investigation save',partial_case)
 def requirement():
  p=new();nav(p,'assessment?id=PRG-104');act(p,'fill-assessment');msubmit(p);act(p,'assessment-submit');msubmit(p)
  nav(p,'requirement-review?id=PRG-104&target=Meridian%20Bank&requirement=R01');nf(p,'note','Reviewed ownership, accountability and evidence for this exact requirement.');nf(p,'decision','Satisfied');dec(p);assert 'accepted' in ''.join(err(p)).lower()
  qs=p.evaluate('findR("R01").questions');evs=[x for x in state(p)['evidence'] if x['program']=='PRG-104' and x['question'] in qs and x['status']!='Accepted' and x['expires']>='2026-11-13']
  for ev in evs:
   nav(p,'evidence-review?id='+ev['id']);nf(p,'decision','Accept');nf(p,'note','Reviewed scoped synthetic evidence metadata and its stated provenance.');dec(p)
  nav(p,'requirement-review?id=PRG-104&target=Meridian%20Bank&requirement=R01');owner=p.evaluate('getAssessment(prog(),selectedTarget(prog())).submittedBy');nf(p,'reviewer',owner);nf(p,'decision','Satisfied');nf(p,'note','Reviewed the exact requirement objective against current accepted evidence.');dec(p);assert 'own submission' in ''.join(err(p));nf(p,'reviewer','Ifeoma Nwosu' if owner!='Ifeoma Nwosu' else 'Daniel Mensah');dec(p);pg=state(p)['programs'][0];assert pg['evaluations']['Meridian Bank']['R01']['status']=='Satisfied';assert p.locator('form[data-j-submit=requirement-decision]').count()==0;p.close()
 test('Full-page requirement review: evidence readiness, self-approval guard and immutable submitted decision',requirement)
 def ropa_export():
  p=new();nav(p,'privacy?tab=register');p.evaluate('() => {window.EXPORTED=[];download=(name,text,mime)=>EXPORTED.push({name,text,mime});}');act(p,'j-export-ropa');out=p.evaluate('EXPORTED[0]');assert 'ROPA-01' in out['text'] and 'purpose_and_basis' in out['text'];p.locator('[data-org-scope=branch]').select_option('B-EN');act(p,'j-export-ropa');out=p.evaluate('EXPORTED[1]');assert 'ROPA-01' not in out['text'] and 'ROPA-04' in out['text'];p.close()
 test('ROPA export payload reflects current scope and uses canonical register records',ropa_export)
 def draft_layout():
  p=new();nav(p,'activity?id=ROPA-02');act(p,'j-edit-activity');aid=draft_id(p);nav(p,'dpia?id=DPIA-01');act(p,'j-edit-dpia');did=draft_id(p)
  nav(p,'privacy?tab=cases');act(p,'j-new-case','[data-kind="Privacy incident"]');cid=draft_id(p);routes=[f'activity-wizard?draft={aid}&step={i}' for i in range(1,7)]+[f'dpia-wizard?draft={did}&step={i}' for i in range(1,6)]+[f'case-intake?draft={cid}&step={i}' for i in range(1,4)]
  for page,action,count in [('vendors','new-vendor',4),('risks','new-risk',3),('losses','new-loss',3),('rcsa','new-cycle',3)]:
   nav(p,page);act(p,action);sid=draft_id(p);routes += [f'setup?draft={sid}&step={i}' for i in range(1,count+1)]
  checks=0
  for width in [390,768,1024,1440]:
   p.set_viewport_size({'width':width,'height':1000})
   for r in routes:
    nav(p,r);assert p.locator('h1').count()==1 and p.locator('.cx-scopebar').is_visible();assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),(width,r);assert p.locator('.j-actions').count()==1;checks+=1
  assert checks==108;p.close()
 test('27 populated wizard steps × 4 widths: accessible heading, persistent scope and no page overflow',draft_layout)
 def nativefile():
  p=b.new_page(accept_downloads=True);p.set_default_timeout(5000);p.goto((ROOT/'clearsight-v2-demo.html').as_uri());assert p.locator('h1').inner_text()=='Privacy & data';nav(p,'privacy?tab=register')
  with p.expect_download() as info:act(p,'j-export-ropa')
  dl=info.value;path=ROOT/'tests/actual-ropa-export.csv';dl.save_as(path);assert 'ROPA-01' in path.read_text();p.locator('[data-org-scope=unit]').select_option('U-IK-PAY');p.reload();p.wait_for_timeout(50);assert p.locator('[data-org-scope=unit]').input_value()=='U-IK-PAY';p.close()
 test('Native file opening, real browser storage reload and CSV download',nativefile)
 b.close()
report={'results':RESULTS,'passed':sum(r['status']=='PASS' for r in RESULTS),'failed':sum(r['status']=='FAIL' for r in RESULTS),'blocked':sum(r['status']=='BLOCKED' for r in RESULTS),'javascript_errors':ERRORS,'browser':'Chromium'}
(ROOT/'tests/journey-results.json').write_text(json.dumps(report,indent=2));print(json.dumps({k:report[k] for k in ['passed','failed','javascript_errors']},indent=2));raise SystemExit(bool(report['failed'] or ERRORS))
