# Forms section resumption implementation plan

> **For agentic workers:** Use subagent-driven-development with test-driven development and spec then quality review. Preserve the unrelated modified presentation PNG.

**Goal:** Keep the selected Forms peer section on direct load, reload and Back/Forward without changing existing template URLs or storing protected document data.

**Architecture:** Extend the existing Forms location helpers and workspace history listener. Reuse FormsNavigation and the existing section readers. No new router, data store, dependency or backend command.

**Decision brief:** [Forms section resumption](../../design/2026-09-08-forms-section-resume.md). Execution owner: #200, IGX-00; release receipt: #147.

## Task 1 — location behavior and regression tests

Files: `web/src/components/forms/formsLocation.ts`, `web/src/components/FormsWorkspace.tsx`, `web/src/components/FormsWorkspace.location.test.tsx`, optionally a focused `formsLocation.test.ts`.

1. Add failing tests for validated section slugs (`templates`, `sent-forms`, `responses`, `documents`, `policies`, `imports`, `communications`), missing/invalid fallback, initial Documents selection, hashchange/popstate synchronization, no duplicate history, transient-state resets on actual changes only and legacy template/filter compatibility.
2. Extend location helpers to read and write the selected section using `section`. Omit it for the default Templates section. Preserve the template target and existing template query while changing tabs; do not reinterpret path IDs. Avoid duplicate history for active-tab activation.
3. Initialize the workspace tab from the location and synchronize it on browser history events. Reuse the existing transient-state cleanup for actual changes; repeated same-section events retain an active editor.
4. Run the new tests red then green, existing Forms tests, typecheck and copy regression. Self-review and commit only owned code/tests.

## Task 2 — browser acceptance and documentation

Files: existing `web/scripts/forms-evidence-scenarios.mjs` and its node checks if needed; this plan, the brief, `docs/README.md`, `docs/implementation-plan.md` and existing design navigation contract as appropriate.

1. Capture the current hosted before-state without material commands or real mail.
2. Extend the existing document scenarios with reload and real Back/Forward assertions, selected tab/panel relationship and fresh document reads; keep deterministic evidence isolated from customer builds.
3. Run web tests, typecheck/build, runtime-truth and UI-contract checks, full rendered review and inspect affected renders at desktop and narrow widths in both themes.
4. Record evidence and residual scope; no broad issue closure.

## Task 3 — review, merge and release

1. Independent spec review then code-quality review; resolve findings and repeat relevant checks.
2. Commit and push, create focused PR, pass exact-head CI/UI gates and merge using the existing authorized workflow.
3. Deploy the exact merged revision through the normal pipeline. Keep SMTP advisory but required security/API/worker checks blocking.
4. Recheck hosted section reload/history behavior, record #147/#200 receipts, and leave broader workstreams open.

## Execution receipt

- Baseline: 2 selected test files / 3 tests passed before changes (Forms location compatibility and copy quality).
- Isolated branch: `codex/forms-section-resume` from `af45e4dbac2c24956fd6ea57d25783090b25e609`.
- Hosted before-state: six light/dark checks at 1440/390/320px reproduced Documents resetting to Templates on reload at `af45e4d`; GET-only after demo sign-in. Local receipt and paired screenshots: `C:/Users/Son/AppData/Local/Temp/clearsight-forms-resume-before-20260908/receipt.json`.
- Browser regression red: the extended existing Documents scenario failed against the baseline evidence bundle because the selected section was absent from the URL. The evidence capability node check also failed before the scenario was registered, then passed.
- Implementation, review and release: pending.
