#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path
path = Path("web/scripts/review-ui-defects.mjs")
text = path.read_text()
old = 'const rationale = page.getByLabel("Rationale");'
new = 'const rationale = page.getByLabel("Rationale", { exact: true });'
if old not in text:
    raise SystemExit("Program rationale selector changed")
path.write_text(text.replace(old, new, 1))
PY

rm -f .github/workflows/ui-defect-selector-closeout.yml scripts/apply-ui-defect-selector-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add web/scripts/review-ui-defects.mjs .github/workflows/ui-defect-selector-closeout.yml scripts/apply-ui-defect-selector-closeout.sh
git commit -m "test(ui): resolve exact Program rationale"
git push origin HEAD:codex/issue-61-sourceaccess-t0
