#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

path = Path("web/src/staticDemo.ts")
text = path.read_text()
needle = '  if (pathname === `/api/v1/programs/${programID}`) return clone(programDetail) as T;\n'
block = '''  if (pathname === `/api/v1/programs/${programID}`) return clone(programDetail) as T;
  if (pathname === `/api/v1/programs/${programID}/transition` && method === "POST") {
    const input = parseBody(init) as { expected_version?: number; to?: string; rationale?: string };
    if (input.expected_version !== program.version) throw new StaticDemoHTTPError(409, "version_conflict", "The Program changed before the status update was recorded.");
    const target = input.to ?? "";
    const transitions: Record<string, string[]> = { DRAFT: ["ACTIVE", "RETIRED"], ACTIVE: ["PAUSED", "RETIRED"], PAUSED: ["ACTIVE", "RETIRED"] };
    if (!(transitions[program.status] ?? []).includes(target)) throw new StaticDemoHTTPError(409, "transition_invalid", `The Program cannot move from ${program.status} to ${target || "an empty status"}.`);
    if (!input.rationale?.trim()) throw new StaticDemoHTTPError(400, "rationale_required", "A rationale is required for the Program status change.");
    program.status = target;
    program.version += 1;
    program.updated_at = now;
    programSummary.program_version = program.version;
    programSummary.projection_stale = true;
    return clone({ ...programDetail, program }) as T;
  }
'''
if needle not in text:
    raise SystemExit("static Program route insertion point changed")
path.write_text(text.replace(needle, block, 1))

path = Path("web/src/components/OperatingMutationsEvidencePage.tsx")
text = path.read_text()
old = 'import type { MatterAggregate, ProgramAggregate, WorkflowTask } from "../types";\n'
new = 'import { useState } from "react";\nimport type { MatterAggregate, ProgramAggregate, WorkflowTask } from "../types";\n'
if old not in text:
    raise SystemExit("Operating mutation import shape changed")
text = text.replace(old, new, 1)
old = 'export function OperatingMutationsEvidencePage() {\n  return <main className="operating-evidence-page">\n'
new = 'export function OperatingMutationsEvidencePage() {\n  const [currentProgram, setCurrentProgram] = useState<ProgramAggregate>(program);\n  return <main className="operating-evidence-page">\n'
if old not in text:
    raise SystemExit("Operating mutation component shape changed")
text = text.replace(old, new, 1)
old = '<ProgramLifecycleControls aggregate={program} onUpdated={() => undefined}/>'
new = '<ProgramLifecycleControls aggregate={currentProgram} onUpdated={setCurrentProgram}/>'
if old not in text:
    raise SystemExit("Operating mutation Program control shape changed")
path.write_text(text.replace(old, new, 1))

path = Path("web/src/staticDemo.test.ts")
text = path.read_text()
needle = '''  it("can deterministically exercise permission and conflict fixtures", async () => {
'''
block = '''  it("executes a governed Program transition and rejects stale replays", async () => {
    const { StaticDemoHTTPError, staticDemoRequest } = await demo();
    const current = await staticDemoRequest<{ program: { status: string; version: number } }>("/api/v1/programs/program-ndpa");
    const updated = await staticDemoRequest<{ program: { status: string; version: number } }>("/api/v1/programs/program-ndpa/transition", {
      method: "POST",
      body: JSON.stringify({ expected_version: current.program.version, to: "PAUSED", rationale: "Pause while ownership is corrected." }),
    });

    expect(updated.program.status).toBe("PAUSED");
    expect(updated.program.version).toBe(current.program.version + 1);
    await expect(staticDemoRequest("/api/v1/programs/program-ndpa/transition", {
      method: "POST",
      body: JSON.stringify({ expected_version: current.program.version, to: "RETIRED", rationale: "Stale replay." }),
    })).rejects.toMatchObject({ status: 409, code: "version_conflict" } satisfies Partial<InstanceType<typeof StaticDemoHTTPError>>);
  });

  it("can deterministically exercise permission and conflict fixtures", async () => {
'''
if needle not in text:
    raise SystemExit("static demo test insertion point changed")
path.write_text(text.replace(needle, block, 1))

path = Path("web/scripts/review-ui-defects.mjs")
text = path.read_text()
old = '''    await page.getByText("Program status updated.", { exact: true }).waitFor({ state: "visible" });
    const calls = await delayedFetchCalls(page);
'''
new = '''    await page.getByText("Program status updated.", { exact: true }).waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Request activation" }).waitFor({ state: "visible" });
    const calls = await delayedFetchCalls(page);
'''
if old not in text:
    raise SystemExit("Program transition defect assertion changed")
path.write_text(text.replace(old, new, 1))
PY

(
  cd web
  npm run typecheck
  npm test
)

rm -f .github/workflows/ui-program-transition-closeout.yml scripts/apply-ui-program-transition-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add web/src/staticDemo.ts \
  web/src/staticDemo.test.ts \
  web/src/components/OperatingMutationsEvidencePage.tsx \
  web/scripts/review-ui-defects.mjs \
  .github/workflows/ui-program-transition-closeout.yml \
  scripts/apply-ui-program-transition-closeout.sh
git commit -m "fix(ui): execute Program status flow end to end"
git push origin HEAD:codex/issue-61-sourceaccess-t0
