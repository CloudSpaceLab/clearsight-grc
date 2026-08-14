#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path
path = Path("web/scripts/review-ui-defects.mjs")
text = path.read_text()
old = '''    await submit.click();
    if (!(await submit.isDisabled())) throw new Error("Program status action is not disabled while the command is in flight");
    await submit.evaluate((element) => element.click());
    await page.getByText("Program status updated.", { exact: true }).waitFor({ state: "visible" });
'''
new = '''    await submit.click();
    const saving = page.getByRole("button", { name: "Recording…" });
    await saving.waitFor({ state: "visible" });
    if (!(await saving.isDisabled())) throw new Error("Program status action is not disabled while the command is in flight");
    await saving.evaluate((element) => element.click());
    await page.getByText("Program status updated.", { exact: true }).waitFor({ state: "visible" });
'''
if old not in text:
    raise SystemExit("Program in-flight assertion changed")
text = text.replace(old, new, 1)
old = '''    await submit.click();
    if (!(await submit.isDisabled())) throw new Error("Evidence submission is not disabled while the request is in flight");
    await submit.evaluate((element) => element.click());
    await page.getByRole("heading", { name: "Response submitted" }).waitFor({ state: "visible" });
'''
new = '''    await submit.click();
    const saving = page.getByRole("button", { name: "Submitting…" });
    await saving.waitFor({ state: "visible" });
    if (!(await saving.isDisabled())) throw new Error("Evidence submission is not disabled while the request is in flight");
    await saving.evaluate((element) => element.click());
    await page.getByRole("heading", { name: "Response submitted" }).waitFor({ state: "visible" });
'''
if old not in text:
    raise SystemExit("Evidence in-flight assertion changed")
path.write_text(text.replace(old, new, 1))
PY

rm -f .github/workflows/ui-defect-busy-closeout.yml scripts/apply-ui-defect-busy-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add web/scripts/review-ui-defects.mjs .github/workflows/ui-defect-busy-closeout.yml scripts/apply-ui-defect-busy-closeout.sh
git commit -m "test(ui): inspect committed busy controls"
git push origin HEAD:codex/issue-61-sourceaccess-t0
