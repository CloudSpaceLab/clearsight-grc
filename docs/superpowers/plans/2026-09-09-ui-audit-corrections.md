# UI audit corrections implementation plan

> For agentic workers: use executing-plans and focused parallel agents for independent file groups. Preserve the existing working tree. No commits, deployment or external messages are part of this task.

**Goal:** Resolve the approved copy, contrast and duplication audit with accurate sector-neutral states and a simpler working interface.

**Architecture:** Preserve typed lifecycle states, verified command authority and current evidence reconciliation. Correct presentation at its source, use existing shared component contracts, and remove duplicate composition rather than merging distinct decisions. The detailed accepted decision brief and acceptance cases are in `docs/reviews/2026-09-09-copy-contrast-component-audit.md` and its four linked reports; the user authorized implementation with “Fix these”.

**Tech stack:** React, TypeScript, React Aria, CSS tokens, Go, Vitest, Playwright and axe. Node 24 is the supported local runtime.

## Baseline and ownership

- [x] Preserve a pre-edit source snapshot and the audit's 104 before-state renders.
- [x] Read the current standards, application architecture and relevant acceptance cases; retain the existing branch and uncommitted vendor work.
- [x] Delegate independent state-correctness, backend copy and contrast tasks with disjoint file ownership. Root owns frontend copy, composition, shared runtime error handling and integration.

## 1. Truth, attribution and requirement semantics (COPY-01–03)

Files: `ProgramCurrentPosition.tsx`, `TodayInterventions.tsx`, `MatterOutcomePanel.tsx`, `ProgramSetupWorkspace.tsx`, `ProgramRequirementsPanel.tsx`, `continuityCommands.ts` and focused tests.

- [x] Add failing cases for missing/stale calculation, absent/invalid schedule, reassigned/unavailable historical reviewer and a nonbank requirement with a different obligated party.
- [x] Render unknown counts as Unknown, stale calculations as Out of date, unknown schedule as Not scheduled; never infer a running calculation.
- [x] Remove current-assignee fallback from historical attribution and maker fallback from pending reviewer responsibility.
- [x] Require actual obligated party/action/object in authoring; preserve submitted values and verified command actor separately. Do not migrate old approved data silently.
- [x] Run affected tests and inspect changed authoring/status states.

## 2. Backend copy and current reviewer names (API-01–06)

Files: audited Go HTTP handlers, formcontract validation, onboarding definitions, communication renderers; evidence response assessment read models if necessary for scoped reviewer display names.

- [x] Neutralize runtime assessment/scope/errors and guide copy while preserving error codes and authorization.
- [x] Split distinct score-preview validation conditions; replace architecture narration with accurate task-specific recovery.
- [x] Shorten notification next-action labels; use Sample organization in universal preview without breaking saved placeholder contracts.
- [x] Provide scoped reviewer display names where the runtime already resolves exact reviewer identity; absent/ambiguous names remain unavailable. Never treat the historical reviewer as the current assignee.
- [x] Add meaningful error/identity regression cases and run affected Go packages.

## 3. Whole-workflow frontend copy and recovery (COPY-04–11)

Files: affected `web/src` React/API presenters, `http.ts`, `DocumentImportWorkspace.tsx`, `vendorDueDiligenceForm.ts`, copy-quality test and shared language documentation.

- [x] Change shared review labels, filters, actions, accessible names, receipt and retry copy together. Preferred pending state: Awaiting review; expanded detail only from the scoped named route.
- [x] Remove repeated review disclaimers; preserve Received versus Accepted versus Approved and the no-upload-needed indication.
- [x] Name response timestamps by their actual backend event. Remove Completed work from Forms Responses.
- [x] Remove Configure architecture narration and redundant shell/section headings. Preserve specialist settings that affect a real decision.
- [x] Fix premature refresh-success notices and normalize network errors without suppressing domain conflict/authority explanations or claiming a committed command failed.
- [x] Make new universal starter templates neutral without rewriting saved revisions.
- [x] Expand copy regression coverage recursively; classify fixtures/internal identifiers instead of broad substring bans.
- [x] Run affected workflow and copy regressions.

## 4. Contrast and style ownership (contrast C-01–06)

Files: shared field/feedback tokens and CSS, local response/activation CSS, audit harness; DESIGN token documentation updated by root.

- [x] Give placeholders explicit accessible theme colors; separate essential input borders from decorative dividers.
- [x] Correct composited status pairs on selected/tinted surfaces; remove response metadata selector overriding StatusBadge.
- [x] Correct activation explanation contrast.
- [x] Retain violations/incomplete/unsupported readings in the regression runner and check placeholders/boundaries explicitly.
- [x] Rebuild and re-render all 104 audit combinations, inspect affected pairs and focused fields; resolve confirmed failures rather than counting incomplete checks as passes.

## 5. Duplicate composition and mobile filters (component C01–C04, C10–C11)

Files: VendorsWorkspace, VendorDueDiligence, VendorFormsPanel, ResponsesView, ResponseAssessment, response filters, Forms navigation and builder inspector.

- [x] Remove the duplicate selected-vendor Request form action; keep batch scope and current assessment next step.
- [x] Consolidate automatic/reviewed result presentation, score explanation and response document action; preserve revision history and independent field decisions.
- [x] Filter uncovered checklist documents individually, tested for all/none/mixed and shared-artifact cases.
- [x] Put request/history and linked work in compact secondary disclosures with scoped counts; attention links expand their destination.
- [x] Replace stacked default response filters with an accessible Filters disclosure on narrow screens, visible active filters/reset and preserved query behavior.
- [x] Make Imports navigation direct, preserving prior routes and editor recovery; summarize builder assessment overview rather than listing every rubric.

## 6. Shared components and retired branches (component C05–C09)

- [x] Migrate audited vendor control groups onto existing shared fields/buttons, retaining specialized upload/signature controls.
- [x] Consolidate empty-state and overlay mechanics without merging sheet/dialog geometry or changing pending-command dismissal.
- [x] Share vendor filter and common score presenters without generic state humanization.
- [x] Remove confirmed obsolete Forms editor/quality panels and unused readiness branch after porting unique tests. Keep legacy Program lifecycle fixtures explicitly evidence-only.
- [x] Remove corresponding dead CSS and same-context duplicates; do not introduce another global patch layer.

## 7. Integration, review and completion

- [x] Update DESIGN, shared content standards, adoption/implementation records and state fixtures with actual delivered contracts.
- [x] Run TypeScript, frontend tests, production/evidence builds, runtime/UI contracts and affected Go tests.
- [x] Review spec coverage against every audit finding, then independent code quality; resolve findings and rerun affected checks.
- [x] Render desktop/mobile/light/dark, narrow320 and zoom/reflow, permissions/errors/conflicts/guide states for changed workspaces. Preserve before/after evidence.
- [x] Record results and exact remaining external limits in an acceptance receipt. No app-wide WCAG or hosted-deployment claim from local fixtures.

Acceptance: [correction receipt](../../acceptance/ui-audit-corrections.md). Full frontend 1,154 passed; final shared-select/workflow 211 passed; Go and builds passed. The receipt records rendered checks and the unavailable CGO/race gate.
