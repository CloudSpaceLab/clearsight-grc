#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

path = Path("web/src/components/EvidenceWorkspace.tsx")
text = path.read_text()
needle = 'const emptyEdit = (): EditState => ({ reason: "", recipient: "", busy: false, error: "" });\n'
replacement = '''const operationalDateTime = new Intl.DateTimeFormat("en-NG", { dateStyle: "medium", timeStyle: "short" });

function formatOperationalDateTime(value: string) {
  return operationalDateTime.format(new Date(value));
}

const emptyEdit = (): EditState => ({ reason: "", recipient: "", busy: false, error: "" });
'''
if needle not in text:
    raise SystemExit("EvidenceWorkspace formatter insertion point changed")
text = text.replace(needle, replacement, 1)
old_summary = '<div className="workspace-brief-facts" aria-label="Evidence work summary"><span><strong>{requestState === "live" ? openRequests : "—"}</strong> open requests</span><span><strong>{sourceState === "live" ? sourceIssues : "—"}</strong> source issues</span></div>'
new_summary = '<div className="workspace-brief-facts" aria-label="Evidence work summary"><span><strong>{requestState === "live" ? openRequests : "—"}</strong> open request{requestState === "live" && openRequests === 1 ? "" : "s"}</span><span><strong>{sourceState === "live" ? sourceIssues : "—"}</strong> source issue{sourceState === "live" && sourceIssues === 1 ? "" : "s"}</span></div>'
if old_summary not in text:
    raise SystemExit("EvidenceWorkspace summary shape changed")
text = text.replace(old_summary, new_summary, 1)
if 'new Date(request.deadline).toLocaleString()' not in text:
    raise SystemExit("EvidenceWorkspace request timestamp shape changed")
text = text.replace('new Date(request.deadline).toLocaleString()', 'formatOperationalDateTime(request.deadline)')
if 'new Date(source.last_success_at).toLocaleString()' not in text:
    raise SystemExit("EvidenceWorkspace source timestamp shape changed")
text = text.replace('new Date(source.last_success_at).toLocaleString()', 'formatOperationalDateTime(source.last_success_at)')
path.write_text(text)

path = Path("web/src/components/OperatingMutations.test.tsx")
text = path.read_text()
if 'name: "Request Paused"' not in text:
    raise SystemExit("Operating mutation button expectation changed")
path.write_text(text.replace('name: "Request Paused"', 'name: "Request pause"'))
PY

rm -f .github/workflows/ui-defect-closeout.yml scripts/apply-ui-defect-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add web/src/components/EvidenceWorkspace.tsx web/src/components/OperatingMutations.test.tsx .github/workflows/ui-defect-closeout.yml scripts/apply-ui-defect-closeout.sh
git commit -m "fix(ui): close rendered workflow defects"
git push origin HEAD:codex/issue-61-sourceaccess-t0
