from pathlib import Path
import json
from playwright.sync_api import sync_playwright
ROOT=Path(__file__).resolve().parents[1];OUT=ROOT/'screenshots/v2.7';OUT.mkdir(parents=True,exist_ok=True)
errors=[];files=[]
with sync_playwright() as pw:
 b=pw.chromium.launch(executable_path='/usr/bin/chromium',headless=True,args=['--no-sandbox'])
 p=b.new_page(viewport={'width':1440,'height':1050});p.on('pageerror',lambda e:errors.append(str(e)))
 p.set_content((ROOT/'clearsight-v2-demo.html').read_text())
 def go(route):p.evaluate('(r)=>location.hash="/"+r',route);p.wait_for_timeout(70)
 def action(a,extra=''):p.locator('[data-action="'+a+'"]'+extra).first.click();p.wait_for_timeout(70)
 def shot(name):
  p.evaluate("document.querySelector('#toast').textContent='';document.activeElement?.blur()")
  p.screenshot(path=str(OUT/name),full_page=True);files.append(name)
 def cp(task):
  if not p.locator('#copilot-root').count():action('cp-open')
  action('cp-task','[data-task="'+task+'"]')
 go('reports');cp('summary');shot('01-reports-copilot.png')
 go('risk?id=RSK-01');cp('draft');shot('02-risk-proposal.png')
 action('cp-review');shot('03-review-action.png');action('close-dialog')
 go('form-builder?form=FORM-TECH');cp('gaps');shot('04-form-checks.png')
 go('documents');cp('gaps');shot('05-document-gaps.png')
 go('document?id=DOC-SAMPLE-POLICY');cp('summary');shot('06-exact-version.png')
 go('program?id=PRG-104');cp('gaps');shot('07-program-gaps.png')
 p.set_viewport_size({'width':390,'height':844});go('reports');cp('summary');shot('08-mobile-report.png')
 p.set_viewport_size({'width':768,'height':1000});go('assessment?id=PRG-104');cp('summary');shot('09-tablet-assessment.png')
 p.set_viewport_size({'width':1440,'height':1000});action('theme');go('risk?id=RSK-01');cp('gaps');shot('10-dark-risk.png')
 b.close()
(ROOT/'tests/capture-v27-results.json').write_text(json.dumps({'version':'2.7','screenshots':files,'errors':errors},indent=2))
assert not errors,errors
print('Captured',len(files),'screenshots without page errors.')
