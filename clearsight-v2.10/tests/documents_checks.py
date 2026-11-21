"""V2.6 shared documents, file bytes, approval integrity and UI integration checks.
Browser storage is a labelled test double; bytes exercise the real memory fallback.
"""
from pathlib import Path
import json, time, traceback, hashlib
from playwright.sync_api import sync_playwright, expect
ROOT=Path(__file__).resolve().parents[1]
HTML=(ROOT/'clearsight-v2-demo.html').read_text()
RESULTS=[]; ERRORS=[]; LAYOUTS=0
PREFIX='''<script>window.TEST_STORE=__STORE__;Object.defineProperty(window,'localStorage',{value:{getItem:k=>TEST_STORE[k]??null,setItem:(k,v)=>TEST_STORE[k]=String(v),removeItem:k=>delete TEST_STORE[k]}});</script>'''
def test(name,fn):
 t=time.monotonic()
 try:fn();RESULTS.append({'name':name,'status':'PASS','seconds':round(time.monotonic()-t,2)});print('PASS',name,flush=True)
 except Exception as ex:RESULTS.append({'name':name,'status':'FAIL','error':str(ex)});print('FAIL',name,str(ex),flush=True);traceback.print_exc()
def new(width=1440,store=None):
 p=b.new_page(viewport={'width':width,'height':1000},accept_downloads=True);p.set_default_timeout(6000);p.on('pageerror',lambda e:ERRORS.append(str(e)));p.set_content(PREFIX.replace('__STORE__',json.dumps(store or {}))+HTML);p.wait_for_timeout(60);return p
def nav(p,r):p.evaluate('(r)=>location.hash="/"+r',r);p.wait_for_timeout(70)
def act(p,a,extra=''):p.locator('[data-action="'+a+'"]'+extra).first.click();p.wait_for_timeout(70)
def data(p):return p.evaluate('CLEAR_SIGHT_DOCUMENTS.getData()')
def state(p):return p.evaluate('CLEAR_SIGHT_DEMO.getState()')
def upload(p,name='test-evidence.txt',content=b'Test document contents\nVersion one.\n',scope=None):
 nav(p,'documents');act(p,'doc-upload')
 if scope:p.locator('[data-doc-meta=scope]').select_option(scope)
 p.locator('#doc-file-input').set_input_files({'name':name,'mimeType':'text/plain','buffer':content})
 p.wait_for_function('DOC_UPLOAD.rows[0].state!=="Reading"')
 assert p.evaluate('DOC_UPLOAD.rows[0].state')=='Ready',p.evaluate('DOC_UPLOAD.rows[0].message')
 act(p,'doc-save-upload');p.wait_for_function('current.name==="document"')
 return data(p)['documents'][0]
def request(p,docid='DOC-SAMPLE-POLICY',template='review-sign'):
 nav(p,'document?id='+docid);act(p,'approval-new')
 p.locator('[data-approval-draft=template]').select_option(template)
 p.locator('[data-approval-draft=reviewer]').select_option('USR-02')
 if template=='review-sign':p.locator('[data-approval-draft=signer]').select_option('USR-03')
 p.locator('[data-approval-draft=note]').fill('Review the source and confirm the controls described.')
 p.locator('#approval-request-form button[type=submit]').click();p.wait_for_function('current.name==="approval"')
 return data(p)['requests'][-1]
def decide(p,user,decision='Approve',signature=None):
 p.locator('[data-doc-actor]').select_option(user)
 f=p.locator('#approval-decision-form');f.locator('[name=decision]').select_option(decision);f.locator('[name=note]').fill('Reviewed the exact source version and its supporting context.')
 if signature is not None:f.locator('[name=signature]').fill(signature);f.locator('[name=intent]').check()
 f.locator('button[type=submit]').click();p.wait_for_timeout(100)
with sync_playwright() as pw:
 b=pw.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox'])
 def integrity():
  p=new();assert p.locator('h1').inner_text()=='Documents';assert p.evaluate('recIntegrity().problems')==[]
  for text in ['', 'abc', 'A'*100000,'\u20a6 financial record']:
   assert p.evaluate('(x)=>recHashText(x)',text)==hashlib.sha256(text.encode()).hexdigest()
  assert len(data(p)['documents'])==9
  p.evaluate("S.risks[0].owner='Sarah Adams'; save();")
  assert p.evaluate("S.risks[0].ownerId")=='USR-05'
  store=p.evaluate('TEST_STORE');p.close();p=new(store=store)
  assert len(data(p)['documents'])==9 and p.evaluate('recIntegrity().problems')==[]
  assert p.evaluate('S.risks[0].owner')=='Sarah Adams'
  p.evaluate("S.governance.versions.find(v=>v.id==='DOC-EV-01-V1').fileName='renamed-source.csv';render();")
  nav(p,'evidence');assert 'renamed-source.csv' in p.locator('body').inner_text()
  assert p.evaluate("S.evidence.find(x=>x.id==='EV-01').name")=='renamed-source.csv';p.close()
 test('Canonical identities, source-backed evidence labels, stable hashes and metadata reconstruction',integrity)
 def uploads():
  p=new();d=upload(p,content=b'Sample text <img src=x onerror="window.PWN=1">\nOnly text.');v=next(v for v in data(p)['versions'] if v['id']==d['currentVersionId'])
  assert v['sha256']==hashlib.sha256(b'Sample text <img src=x onerror="window.PWN=1">\nOnly text.').hexdigest()
  expect(p.locator('.doc-text-preview')).to_contain_text('<img src=x');assert not p.evaluate('window.PWN||false')
  assert v['storage']=='session';assert 'sampleText' not in v
  nav(p,'documents');act(p,'doc-upload');p.locator('#doc-file-input').set_input_files([
   {'name':'valid.csv','mimeType':'text/csv','buffer':b'name,value\n"One, two",3\n'},
   {'name':'fake.pdf','mimeType':'application/pdf','buffer':b'not a PDF'},
   {'name':'script.js.txt','mimeType':'text/plain','buffer':b'not accepted'},
   {'name':'duplicate.txt','mimeType':'text/plain','buffer':b'name,value\n"One, two",3\n'}])
  p.wait_for_function('DOC_UPLOAD.rows.every(x=>x.state!=="Reading")')
  assert p.evaluate('DOC_UPLOAD.rows.map(x=>x.state)')==['Ready','Error','Error','Error']
  act(p,'doc-save-upload');expect(p.locator('.doc-preview table')).to_contain_text('One, two');p.close()
 test('Real file upload, partial batch errors, duplicate contents, SHA-256 and safe CSV/text preview',uploads)
 def invalid():
  p=new();result=p.evaluate('''()=>{const cases=[['zero.txt',new Uint8Array(0)],['bad.png',new TextEncoder().encode('wrong')],['script.txt',new TextEncoder().encode('<html>bad</html>')],['bin.txt',new Uint8Array([0,1,2])],['archive.docx',new Uint8Array([80,75,3,4,1,2,3])]];return cases.map(([name,bytes])=>{try{docValidateFile({name,size:bytes.length},bytes);return false}catch(e){return !!e.message}})}''');assert all(result)
  assert p.evaluate("(()=>{try{docValidateFile({name:'large.txt',size:21*1024*1024},new Uint8Array());return false}catch(e){return e.message.includes('20 MB')}})()")
  p.close()
 test('Empty, oversized, disguised, binary-text and invalid Office uploads are rejected',invalid)
 def signing():
  p=new();r=request(p);assert len([x for x in p.evaluate('workItems()') if x.get('source')=='document-approval'])==1
  p.locator('[data-doc-actor]').select_option('USR-03');assert p.locator('#approval-decision-form').count()==0
  decide(p,'USR-02');r=data(p)['requests'][-1];assert r['status']=='Pending' and r['steps'][0]['state']=='Approved'
  decide(p,'USR-03','Sign off','Wrong person');assert 'full name' in p.locator('#approval-decision-error').inner_text()
  decide(p,'USR-03','Sign off','Ifeoma Nwosu');r=data(p)['requests'][-1];assert r['status']=='Completed'
  receipts=[x for x in data(p)['receipts'] if x.get('requestId')==r['id']];assert len(receipts)==2 and receipts[1]['typedName']=='Ifeoma Nwosu'
  assert all(x['snapshotHash']==r['snapshotHash'] for x in receipts)
  assert p.evaluate("docStatus(recRead('document','DOC-SAMPLE-POLICY'))")=='Signed off'
  assert not [x for x in p.evaluate('workItems()') if x.get('source')=='document-approval'];assert p.evaluate('recIntegrity().problems')==[]
  p.close()
 test('Sequential independent review and explicit typed sign-off; one shared Work item; immutable receipts',signing)
 def selfreview():
  p=new();nav(p,'document?id=DOC-SAMPLE-POLICY');act(p,'approval-new');p.locator('[data-approval-draft=reviewer]').select_option('USR-01');p.locator('#approval-request-form button[type=submit]').click();p.wait_for_timeout(70)
  assert 'cannot approve' in p.locator('#approval-request-error').inner_text();assert not data(p)['requests'];p.close()
 test('Requester and file contributor cannot be assigned to approve their own submission',selfreview)
 def replacement():
  p=new();r=request(p);old=r['versionId'];nav(p,'program?id=PRG-104');act(p,'doc-attach');p.locator('#modal-form [name=version]').select_option(old);p.locator('#modal-form button[type=submit]').click()
  nav(p,'document?id=DOC-SAMPLE-POLICY');act(p,'doc-replace');p.locator('[data-doc-meta=note]').fill('Updated review guidance for the next review cycle.')
  p.locator('#doc-file-input').set_input_files({'name':'procedure-v2.txt','mimeType':'text/plain','buffer':b'Revised procedure version two.\n'});p.wait_for_function('DOC_UPLOAD.rows[0].state==="Ready"');act(p,'doc-save-upload');p.wait_for_function('current.name==="document"')
  assert data(p)['requests'][-1]['status']=='Superseded'
  d=next(x for x in data(p)['documents'] if x['id']=='DOC-SAMPLE-POLICY');assert d['currentVersionId']!=old
  assert any(x['documentVersionId']==old and x['subject']['id']=='PRG-104' for x in data(p)['links'])
  assert p.evaluate("docStatus(recRead('document','DOC-SAMPLE-POLICY'))")=='Draft'
  nav(p,'approval?id='+r['id']);p.locator('[data-doc-actor]').select_option('USR-02');assert not p.locator('#approval-decision-form').count();p.close()
 test('Replacement supersedes pending approval without retargeting earlier document links',replacement)
 def approved_old():
  p=new();r=request(p,template='review');decide(p,'USR-02');old=r['versionId']
  nav(p,'program?id=PRG-104');act(p,'doc-attach');p.locator('#modal-form [name=version]').select_option(old);p.locator('#modal-form button[type=submit]').click()
  nav(p,'document?id=DOC-SAMPLE-POLICY');act(p,'doc-replace');p.locator('[data-doc-meta=note]').fill('Updated control instructions and responsibilities.');p.locator('#doc-file-input').set_input_files({'name':'procedure-v2.txt','mimeType':'text/plain','buffer':b'Version two content distinct.\n'});p.wait_for_function('DOC_UPLOAD.rows[0].state==="Ready"');act(p,'doc-save-upload');p.wait_for_function('current.name==="document"')
  nav(p,'program?id=PRG-104');expect(p.locator('.doc-attachments')).to_contain_text('Approved');expect(p.locator('.doc-attachments')).to_contain_text('Newer version available');assert p.evaluate("docStatus(recRead('document','DOC-SAMPLE-POLICY'))")=='Draft';p.close()
 test('Pinned attachments retain their version-specific approval instead of inheriting the latest state',approved_old)
 def missing_bytes():
  p=new();d=upload(p);store=p.evaluate('TEST_STORE');p.close();p=new(store=store);nav(p,'document?id='+d['id']);expect(p.locator('.doc-preview')).to_contain_text('File unavailable in this browser');act(p,'approval-new');p.locator('#approval-request-form button[type=submit]').click();p.wait_for_timeout(100);assert 'File bytes are unavailable' in p.locator('#approval-request-error').inner_text();assert not data(p)['requests'];p.close()
 test('Session-only byte loss remains explicit and blocks approval after metadata reconstruction',missing_bytes)
 def linked_evidence():
  p=new();nav(p,'assessment?id=PRG-104');act(p,'add-evidence','[data-question=Q05]');p.locator('#modal-form [name=version]').select_option('DOC-SAMPLE-POLICY-V1');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(70)
  ev=state(p)['evidence'][0];assert ev['documentVersionId']=='DOC-SAMPLE-POLICY-V1' and ev['question']=='Q05'
  nav(p,'evidence-review?id='+ev['id']);expect(p.locator('.doc-text-preview')).to_contain_text('Access review procedure');f=p.locator('[data-j-submit=evidence-decision]');f.locator('[name=reviewer]').select_option('Ifeoma Nwosu');f.locator('[name=decision]').select_option('Accept');f.locator('[name=note]').fill('Reviewed the source and accepted its use for this question.');f.locator('button[type=submit]').click();p.wait_for_timeout(80)
  ev=next(x for x in state(p)['evidence'] if x['id']==ev['id']);assert ev['status']=='Accepted';assert p.evaluate("docStatus(recRead('document','DOC-SAMPLE-POLICY'))")=='Draft'
  assert any(x['subject']['id']==ev['id'] and x['sourceVersion']=='DOC-SAMPLE-POLICY-V1' for x in data(p)['receipts']);p.close()
 test('Assessment uploads and evidence review use the same file version without auto-approving compliance',linked_evidence)
 def scope_export():
  p=new();p.locator('[data-org-scope=branch]').select_option('B-IK');rows=p.evaluate('CLEAR_SIGHT_DOCUMENTS.query()');assert all(x['scopeMode']=='shared' or x['orgScopeId'].startswith('U-IK') for x in rows)
  p.locator('[data-doc-filter=search]').fill('procedure');p.wait_for_timeout(250);rows=p.evaluate('CLEAR_SIGHT_DOCUMENTS.query()');assert len(rows)==1
  p.evaluate('()=>{window.EXPORTS=[];download=(name,data,mime)=>EXPORTS.push({name,data,mime});}');act(p,'doc-export');csv=p.evaluate('EXPORTS[0].data');assert 'DOC-SAMPLE-POLICY' in csv and 'DOC-SAMPLE-KRI' not in csv
  rejected=p.evaluate("(()=>{try{docLinkVersion('DOC-SAMPLE-KRI-V1',{type:'engagement',id:'ENG-01'});return false}catch(e){return e.message.includes('scope')}})()");assert rejected;p.close()
 test('Scope, search and exports use the same query; cross-location links fail validation',scope_export)
 def form_file():
  p=new();forms=state(p)['formStudio']['forms'];f=next(x for x in forms if any(q['type']=='evidence' for q in x['fields']));nav(p,'form-response?form='+f['id']);act(p,'doc-form-file');p.locator('#modal-form [name=version]').select_option('DOC-SAMPLE-POLICY-V1');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(70)
  response=state(p)['formStudio']['responses'][f['id']+'::preview'];assert 'DOC-SAMPLE-POLICY-V1' in response['answers'].values();expect(p.locator('.doc-evidence-input').first).to_contain_text('access-review-procedure.txt')
  p.evaluate('faResponse().status="Submitted";render();');assert p.locator('[data-action=doc-form-file]').count()==0
  failed=p.evaluate("(()=>{try{docLinkVersion('DOC-SAMPLE-POLICY-V1',{type:'formResponse',id:faResponseKey()},'changed');return false}catch(e){return true}})()");assert failed;p.close()
 test('Form evidence fields use a controlled document picker and submitted response links are locked',form_file)
 def report_output():
  p=new();nav(p,'reports');act(p,'ix-new-report');act(p,'ix-report-step','[data-step="2"]');p.locator('[data-ix-report-draft=commentary]').fill('Review branch returns and source documents before management decisions.');act(p,'ix-report-step','[data-step="3"]');act(p,'ix-publish-report');pack=state(p)['reportPacks'][0];act(p,'doc-save-report');p.wait_for_function('current.name==="document"');assert any(l['subject']=={'type':'report','id':pack['id']} for l in data(p)['links']);expect(p.locator('.doc-text-preview')).to_contain_text(pack['id']);assert state(p)['reportPacks'][0]==pack;assert p.evaluate('recIntegrity().problems')==[];p.close()
 test('Published report output is saved as a versioned document without changing its source snapshot',report_output)
 def dispositions():
  p=new();r=request(p,template='review');decide(p,'USR-02','Request changes');assert data(p)['requests'][-1]['status']=='Changes requested'
  p.locator('[data-doc-actor]').select_option('USR-01');r2=request(p,template='review');decide(p,'USR-02','Reject');assert data(p)['requests'][-1]['status']=='Rejected'
  p.locator('[data-doc-actor]').select_option('USR-01');r3=request(p,template='review');act(p,'approval-withdraw');p.locator('#modal-form [name=reason]').fill('Withdrawn while the document owner prepares a new source version.');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(70)
  assert data(p)['requests'][-1]['status']=='Cancelled';assert len(data(p)['receipts'])==2;assert len(data(p)['requests'])==3;assert p.evaluate('recIntegrity().problems')==[];p.close()
 test('Change requests, rejection and withdrawal retain prior decisions and separate request episodes',dispositions)
 def tamper_and_duplicate():
  p=new();r=request(p,template='review');p.locator('[data-doc-actor]').select_option('USR-02');p.evaluate('S.governance.requests.at(-1).snapshot.size+=1');decide(p,'USR-02');assert 'review data changed' in p.locator('#approval-decision-error').inner_text();assert not data(p)['receipts'];assert p.evaluate('recIntegrity().problems.some(x=>x.code==="CHANGED_SNAPSHOT")')
  p.evaluate('S.governance.requests.at(-1).snapshot.size-=1');decide(p,'USR-02');assert data(p)['requests'][-1]['status']=='Completed';assert len(data(p)['receipts'])==1
  p.evaluate("S.governance.actorId='USR-01';render()");nav(p,'document?id=DOC-SAMPLE-POLICY');act(p,'approval-new');assert len(data(p)['requests'])==1;assert 'completed approval' in p.locator('#toast').inner_text();p.close()
 test('Changed approval snapshots and repeated approval attempts cannot create accepted decisions',tamper_and_duplicate)
 def archive_move():
  p=new();r=request(p,template='review');decide(p,'USR-02');nav(p,'program?id=PRG-104');act(p,'doc-attach');p.locator('#modal-form [name=version]').select_option('DOC-SAMPLE-POLICY-V1');p.locator('#modal-form button[type=submit]').click();before=data(p)
  nav(p,'document?id=DOC-SAMPLE-POLICY');act(p,'doc-move');p.locator('#modal-form [name=folder]').select_option('F-OPERATIONS');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(60);act(p,'doc-archive');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(60)
  after=data(p);assert after['versions']==before['versions'] and after['links']==before['links'] and after['receipts']==before['receipts'];assert next(d for d in after['documents'] if d['id']=='DOC-SAMPLE-POLICY')['archived'];act(p,'doc-archive');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(60);assert p.evaluate("docStatus(recRead('document','DOC-SAMPLE-POLICY'))")=='Approved';p.close()
 test('Folder moves and archive/restore preserve file versions, source links and completed reviews',archive_move)
 def attach_submit(p,form_id):
  nav(p,'forms');act(p,'fa-attach','[data-id="'+form_id+'"]')
  target=p.evaluate("recRead('program','PRG-104').targets[0]");p.locator('#modal-form [name=scope]').select_option('PRG-104::'+target)
  for sel in p.locator('#modal-form select[name^="slot-"]').all():
   value=sel.locator('option').nth(1).get_attribute('value');sel.select_option(value)
  p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(80)
  if form_id=='FORM-TECH':act(p,'fa-run')
  else:act(p,'fa-example')
  act(p,'fa-submit');p.locator('#modal-form [name=respondent]').select_option('Amina Bello');p.locator('#modal-form button[type=submit]').click();p.wait_for_timeout(100)
  binding=p.evaluate('faBinding()');return binding,target
 def form_to_requirement():
  global LAYOUTS
  p=new();bnd,target=attach_submit(p,'FORM-OPS');before=state(p)['formStudio']['responses'][bnd['id']]['snapshot']
  assert p.evaluate('workItems().some(x=>x.source==="form-review")')
  route='requirement-review?id=PRG-104&target='+target+'&requirement=R03&binding='+bnd['id'];nav(p,route)
  f=p.locator('#source-requirement-form');f.locator('[name=reviewer]').select_option('Amina Bello');f.locator('[name=decision]').select_option('Satisfied');f.locator('[name=note]').fill('Reviewed the complete objective and the submitted control assessment.');f.locator('[name=coverage]').check();f.locator('button[type=submit]').click();p.wait_for_timeout(80);expect(p.locator('#source-review-error')).to_contain_text('cannot review')
  f.locator('[name=reviewer]').select_option('Ifeoma Nwosu');f.locator('[name=coverage]').uncheck();f.locator('button[type=submit]').click();p.wait_for_timeout(80);expect(p.locator('#source-review-error')).to_contain_text('Confirm full objective coverage');f.locator('[name=coverage]').check();f.locator('button[type=submit]').click();p.wait_for_timeout(100)
  out=p.evaluate('(t)=>evaluation(recRead("program","PRG-104"),t,"R03")',target);assert out['status']=='Satisfied' and out['source']['bindingId']==bnd['id'];assert state(p)['formStudio']['responses'][bnd['id']]['snapshot']==before
  for w in [390,768,1024,1440]:p.set_viewport_size({'width':w,'height':1000});nav(p,route);assert p.locator('h1').count()==1 and p.evaluate('document.documentElement.scrollWidth<=innerWidth+1');LAYOUTS+=1
  nav(p,'program?id=PRG-104&target='+target+'&tab=requirements');expect(p.locator('body')).to_contain_text('Satisfied');assert p.evaluate('recIntegrity().problems')==[];p.close()
 test('Custom form submissions feed canonical requirement evaluations, independent review and shared Work',form_to_requirement)
 def technical_requirement():
  global LAYOUTS
  p=new();bnd,target=attach_submit(p,'FORM-TECH');prior=p.evaluate('(t)=>evaluation(recRead("program","PRG-104"),t,"R05")',target);route='requirement-review?id=PRG-104&target='+target+'&requirement=R05&binding='+bnd['id'];nav(p,route);f=p.locator('#source-requirement-form');f.locator('[name=reviewer]').select_option('Ifeoma Nwosu');f.locator('[name=decision]').select_option('Satisfied');f.locator('[name=coverage]').check();f.locator('[name=note]').fill('Reviewed configured metadata, but a length check alone is only a heuristic.');f.locator('button[type=submit]').click();p.wait_for_timeout(80);expect(p.locator('#source-review-error')).to_contain_text('heuristic');assert p.evaluate('(t)=>evaluation(recRead("program","PRG-104"),t,"R05")',target)==prior
  f.locator('[name=decision]').select_option('Partially satisfied');f.locator('button[type=submit]').click();p.wait_for_timeout(80);assert p.evaluate('(t)=>evaluation(recRead("program","PRG-104"),t,"R05").status',target)=='Partially satisfied'
  for w in [390,768,1024,1440]:p.set_viewport_size({'width':w,'height':1000});nav(p,route);assert p.locator('h1').count()==1 and p.evaluate('document.documentElement.scrollWidth<=innerWidth+1');LAYOUTS+=1
  p.close()
 test('Technical source uncertainty and heuristics cannot silently produce satisfied requirements',technical_requirement)
 def submission_isolation():
  p=new();bnd,target=attach_submit(p,'FORM-OPS');nav(p,'requirement-review?id=PRG-105&target=ENG-01&requirement=R03&binding='+bnd['id']);assert p.locator('#source-requirement-form').count()==0
  nav(p,'requirement-review?id=PRG-104&target='+target+'&requirement=R03&binding='+bnd['id']);p.evaluate('(id)=>S.formStudio.responses[id].snapshot.answers.controls[0]="tampered"',bnd['id']);f=p.locator('#source-requirement-form');f.locator('[name=reviewer]').select_option('Ifeoma Nwosu');f.locator('[name=note]').fill('Attempted review after source mutation must not be accepted.');f.locator('button[type=submit]').click();p.wait_for_timeout(80);expect(p.locator('#source-review-error')).to_contain_text('source changed');assert not state(p)['formStudio']['responses'][bnd['id']].get('requirementReviews');p.close()
 test('Assessment-to-requirement bridge rejects cross-target sources and changed submitted snapshots',submission_isolation)

 def layout():
  global LAYOUTS
  p=new();r=request(p);nav(p,'documents');act(p,'doc-upload');p.locator('#doc-file-input').set_input_files({'name':'layout.txt','mimeType':'text/plain','buffer':b'Layout preview.\n'});p.wait_for_function('DOC_UPLOAD.rows[0].state==="Ready"')
  routes=['documents','document-upload','document?id=DOC-SAMPLE-POLICY','document?id=DOC-SAMPLE-POLICY&tab=versions','document?id=DOC-SAMPLE-POLICY&tab=links','document?id=DOC-SAMPLE-POLICY&tab=approvals','document?id=DOC-SAMPLE-KRI','approvals?tab=all','approvals?tab=records','approval?id='+r['id'],'data-health']
  for w in [390,768,1024,1440]:
   p.set_viewport_size({'width':w,'height':1000})
   for route in routes:
    nav(p,route);assert p.locator('h1').count()==1,(w,route);assert p.evaluate("[...document.querySelectorAll('.doc-name')].every(el=>el.getBoundingClientRect().right<=el.closest('td').getBoundingClientRect().right+1)"),(w,route,'file-name cell overflow');assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1'),(w,route,'overflow');LAYOUTS+=1
  p.locator('[data-action=theme]').click();nav(p,'document?id=DOC-SAMPLE-POLICY');assert p.evaluate('document.documentElement.scrollWidth<=innerWidth+1');p.close()
 test('Document explorer, previews, version history and populated approval/upload flows at four widths',layout)
 b.close()
(ROOT/'tests/documents-results.json').write_text(json.dumps({'version':'2.7','tests':RESULTS,'layouts':LAYOUTS,'pageErrors':ERRORS},indent=2))
print(json.dumps({'passed':sum(x['status']=='PASS' for x in RESULTS),'failed':sum(x['status']=='FAIL' for x in RESULTS),'layouts':LAYOUTS,'errors':ERRORS}),flush=True)
