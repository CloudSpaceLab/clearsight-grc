#!/usr/bin/env python3
"""Build dependency-free ClearSight HTML entry points from shared source assets."""
from pathlib import Path
import json
ROOT = Path(__file__).resolve().parent
ROUTES = {
    'index': 'my-workspace', 'dashboard':'home', 'framework-library': 'frameworks', 'framework-adoption': 'adopt',
    'programs': 'programs', 'program-workspace': 'program', 'assessment-workspace': 'assessment',
    'forms': 'forms', 'form-builder':'form-builder', 'form-response':'form-response', 'check-runs':'check-runs', 'vendors': 'vendors', 'vendor-engagement': 'vendor',
    'risks': 'risks', 'rcsa': 'rcsa', 'rcsa-review': 'rcsa-review',
    'key-indicators': 'indicators', 'losses': 'losses', 'controls': 'controls',
    'evidence': 'evidence', 'privacy': 'privacy', 'work-approvals': 'work',
    'reports': 'reports', 'integrations': 'integrations', 'settings': 'settings', 'processing-activity':'activity', 'processing-registration':'activity-wizard', 'processing-version':'activity-version', 'privacy-impact-assessment':'dpia', 'privacy-impact-wizard':'dpia-wizard', 'privacy-case':'privacy-case', 'privacy-case-intake':'case-intake', 'guided-setup':'setup', 'risk-workspace':'risk', 'loss-workspace':'loss', 'control-workspace':'control', 'evidence-review':'evidence-review', 'requirement-review':'requirement-review', 'branch-kri-returns':'kri-returns', 'branch-kri-return':'kri-return', 'kri-cycle-setup':'kri-cycle', 'insight-detail':'insight-detail', 'report-builder':'report-builder', 'report-pack':'report-pack', 'documents':'documents', 'document-workspace':'document', 'document-upload':'document-upload', 'approval-inbox':'approvals', 'approval-review':'approval', 'approval-request':'approval-request', 'data-checks':'data-health'
}
ROUTES.update({'kri-question-library':'kri-library','indicator-workspace':'indicator','roles-permissions':'roles','role-detail':'role-detail','access-review':'access-review','escalation-matrix':'escalation-matrix','escalation-policy':'escalation-policy','inbox':'inbox','notification-detail':'notification','notification-preferences':'notification-preferences'})

ROUTES.update({'loss-intake':'loss-intake','loss-import':'loss-import','loss-insights':'loss-insights','control-editor':'control-editor','control-test':'control-test','vendor-review':'vendor-review','risk-editor':'risk-edit'})

ROUTES.update({'assessment-history':'assessment-history','assessment-response-review':'assessment-review','assessment-insights':'assessment-insights','vendor-onboarding':'vendor-onboarding','vendor-intake':'vendor-intake','vendor-requests':'vendor-requests'})
ROUTES.update({'work-item':'work-item','role-change-review':'role-change-review','my-workspace':'my-workspace','my-access':'my-access','external-assessments':'external-assessments','external-assessment-request':'external-assessment-request'})

HEAD = '''<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark"><meta name="description" content="ClearSight V2.15 connected interactive prototype. Synthetic records and local-only workflows.">
<link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Crect width='32' height='32' rx='9' fill='%23246ff5'/%3E%3Cpath d='M22 10a9 9 0 1 0 0 12' fill='none' stroke='white' stroke-width='3'/%3E%3C/svg%3E">
<title>ClearSight V2.15 — Interactive Demo</title>'''
BODY = '''<div id="app"></div><div id="toast" class="toast" role="status" aria-live="polite"></div><dialog id="dialog"></dialog>
<noscript><h1>ClearSight V2 interactive demo</h1><p>Enable JavaScript to use this local-only prototype. All records are synthetic.</p></noscript>'''

def html(route: str, inline: bool=False) -> str:
    css_names=['styles','forms','experience','journeys','insights','documents','usability','copilot','branch-kri','governance','inbox','board-reports','loss-register','workflow-completion','flow-polish','loss-reporting','assessment-tracking','vendor-onboarding','documents-unified','assessment-reporting','interaction-core','assurance-interactions','governance-interactions','operations-interactions','privacy-interactions','role-experience','scoped-analytics','document-management','external-assessments','premium-foundation','premium-workspaces','premium-forms','premium-reporting','premium-programs','premium-risk','premium-privacy','premium-utilities']
    js_names=['app','forms','experience','journeys','insights','records','document-store','documents','usability','copilot','kri-schema','branch-kri','governance','inbox','board-reports','loss-schema','loss-register','workflow-completion','flow-polish','loss-reporting','assessment-tracking','vendor-onboarding','documents-unified','assessment-reporting','interaction-core','assurance-interactions','governance-interactions','operations-interactions','privacy-interactions','role-experience','scoped-analytics','document-management','external-assessments','premium-foundation','premium-workspaces','premium-forms','premium-reporting','premium-programs','premium-risk','premium-privacy','premium-utilities','bootstrap']
    css='\n'.join((ROOT/'assets'/f'{name}.css').read_text() for name in css_names)
    seed=(ROOT/'assets/seed.js').read_text()
    app='\n'.join((ROOT/'assets'/f'{name}.js').read_text() for name in js_names)
    styles='<style>\n'+css+'\n</style>' if inline else ''.join(f'<link rel="stylesheet" href="assets/{name}.css">' for name in css_names)
    scripts=('<script>\n'+seed.replace('</script','<\\/script')+'\n</script><script>\n'+app.replace('</script','<\\/script')+'\n</script>') if inline else '<script src="assets/seed.js"></script>'+''.join(f'<script src="assets/{name}.js"></script>' for name in js_names)
    return HEAD+styles+'</head><body data-start="'+route+'">'+BODY+scripts+'</body></html>\n'

if __name__ == '__main__':
    for file, route in ROUTES.items():
        (ROOT/(file+'.html')).write_text(html(route), encoding='utf-8')
    (ROOT/'clearsight-v2-demo.html').write_text(html('my-workspace', True), encoding='utf-8')
    (ROOT/'docs/route-manifest.json').write_text(json.dumps(ROUTES, indent=2))
    print(f'Built {len(ROUTES)} module entry points and one standalone connected demo.')
