# ClearSight V1 Resolution Board Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Close issue #128 with one canonical V1 contract, persisted reference truth, completed real SMTP and Program/Matter workflows, governed organization routing, explainable scoring/oversight and exact-head hosted acceptance.

**Architecture:** Preserve Programs, Matters, Evidence Requests, form distributions, response workspaces, third-party relationships, authority routes, workflow tasks, outbox/inbox and projections as the only canonical foundations. Each workstream removes a proven gap without adding parallel truth. Material mutations re-evaluate verified identity and current authority and commit authoritative state, append-only event, outbox fact and required maintenance work together.

**Tech Stack:** Go 1.24 modular monolith, PostgreSQL migrations/repositories, React 19 + TypeScript + Vite, Vitest/Testing Library/axe, Playwright/Chromium evidence scripts, GitHub Actions, Docker Compose deployment, SMTP STARTTLS.

---

## Dependency board

| Order | Issue | Outcome | Depends on |
| --- | --- | --- | --- |
| 1 | #137 | strict canonical HTTP/capture contracts; no unreleased compatibility | merged #133–#136 baseline |
| 2 | #138 | all interactive demo truth persisted through normal APIs | #137 contract shape |
| 3 | #141 | reporting hierarchy and one governed handoff control | current identity/authority foundations |
| 4 | #142 | maker-checker escalation configuration and runtime | #141 |
| 5 | #143 | scoring and automatic Matter policy acceptance | #137; reworked non-duplicate parts of PR #129 |
| 6 | #139 | registration, address verification, activation and certification SMTP journeys | #137, #141 |
| 7 | #140 | approved forms remediate existing Program issues | #137; current form/Matter foundations |
| 8 | #144 | complete oversight history and explainable estimates | canonical lifecycle data from #139/#140/#143 |
| 9 | #145 | role-accurate Today from current assigned/operational state | #138, #141, #142 |
| 10 | #146 | hosted operational data repair and count reconciliation | #141/#145 where assignment/routing is involved |
| 11 | #147 | exact-head release and hosted closure | all prior issues |

Issues #13, #57, #74 and #80 remain broader capability roadmaps. Only the V1 slices explicitly linked above block #128. Do not close those umbrella issues merely because this board passes.

## Task 1: Freeze the board and reconcile overlapping pull requests

**Files:**
- Modify: GitHub issue #128
- Review: GitHub PR #129
- Review: GitHub PR #124

- [ ] Add issues #137–#147 to the parent #128 checklist with the dependency order above.
- [ ] Comment on PR #129 that merged #130 supersedes its runtime-context work, the branch conflicts with current main, and backend formatting currently fails in `cmd/seed-bank-reference/form_policy_authority_seed.go`.
- [ ] Extract only non-duplicated scoring/policy changes from PR #129 into a new branch based on current `origin/main`; close #129 after the replacement is linked.
- [ ] Ask PR #124's author to split the document-import extraction fix and demo-menu change into titled, current-main PRs; close #124 if no unique current change remains.
- [ ] Keep #128 as the release board and avoid adding implementation detail that belongs in child issues.

## Task 2: Remove alternate V1 contracts (#137)

**Files:**
- Modify: `internal/evidence/distribution.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/evidence/completed_response.go`
- Modify: `internal/formcontract/model.go`
- Modify: `internal/httpapi/form_distribution_list.go`
- Modify: `internal/httpapi/form_distribution_handlers.go`
- Delete: `internal/evidence/draft_compatibility.go`
- Delete: `internal/evidence/draft_compatibility_test.go`
- Modify: `web/src/formsDistributionApi.ts`
- Modify: `web/src/formsDistributionApi.test.ts`
- Modify: `web/src/captureInvitationBrowser.ts`
- Modify: `web/src/components/ExternalCaptureApp.test.tsx`
- Test: `internal/httpapi/*contract*_test.go`

- [ ] Write failing handler-level JSON tests for distribution list/detail, recipient, workspace and response revision canonical fields.
- [ ] Add explicit safe HTTP DTOs or complete JSON tags and stop writing untagged aggregates.
- [ ] Remove browser PascalCase fallbacks and prove PascalCase input is not silently accepted.
- [ ] Replace `AnswerValue.UnmarshalJSON` scalar acceptance with structured-only decoding and add rejection tests.
- [ ] Remove the dead draft compatibility facade and current composition references.
- [ ] Remove `capture_invite` query discovery and retired session-storage cleanup; retain fragment `form_access` only.
- [ ] Audit every registered `WriteJSON` and governed strict-decoder route using a checked script/test, then fix all same-class findings in the tranche.
- [ ] Run `gofmt -w` on changed Go files.
- [ ] Run `go test ./internal/evidence ./internal/formcontract ./internal/httpapi -count=1`.
- [ ] Run `npm --prefix web test -- formsDistributionApi ExternalCaptureApp`.
- [ ] Run `go test -tags postgres ./internal/evidence ./internal/httpapi -count=1`.

## Task 3: Move all demo truth to persisted reference data (#138)

**Files:**
- Delete: `web/src/staticDemoBootstrap.ts`
- Delete or test-isolate: `web/src/staticDemo.ts`
- Delete or test-isolate: `web/src/staticExternalCapture.ts`
- Modify: `web/src/main.tsx`
- Modify: `web/src/evidenceMain.tsx`
- Modify: `internal/workflow/memory.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/seed-bank-reference/main.go`
- Modify: `web/scripts/runtime-fixture-boundary.nodecheck.mjs`
- Test: `internal/runtimecontext/architecture_test.go`

- [ ] Write a failing customer-bundle test proving no deployable module can intercept `/api/*`.
- [ ] Route the stakeholder deployment to the normal API/worker and a dedicated reference PostgreSQL database.
- [ ] Install reference identities, authority, Programs, Matters, Forms, responses, policies, vendors and history through idempotent canonical service commands.
- [ ] Remove `workflow.DemoTasks()` and project Today from installed workflow records.
- [ ] Keep deterministic visual fixtures only in the evidence build, visibly labelled and unreachable from the customer entry.
- [ ] Start with an empty database and verify truthful empty states; run the installer and verify the same API returns stored reference journeys.
- [ ] Restart API/worker/browser and prove state persists without local/session storage truth.
- [ ] Run `go test ./cmd/api ./cmd/seed-bank-reference ./internal/runtimecontext ./internal/today ./internal/workflow -count=1`.
- [ ] Run `npm --prefix web run check:ui-contracts && npm --prefix web run build`.

## Task 4: Complete hierarchy and escalation (#141, #142)

**Files:**
- Modify: `internal/access/*`
- Modify: `internal/authority/*`
- Modify: `internal/workflow/matter_escalation_postgres.go`
- Modify: `internal/httpapi/identity_access_handlers.go`
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: `web/src/components/access/*`
- Modify: `web/src/components/MatterDetailsPanel.tsx`
- Modify: `web/src/components/MatterActionsPanel.tsx`
- Test: `internal/httpapi/command_lifecycle_test.go`
- Test: `internal/workflow/*escalation*_test.go`

- [ ] Write failing tests for current-assignee and reporting-manager handoff, cycle/cross-entity rejection, vacancy and revoked occupant.
- [ ] Implement versioned reporting-line configuration with simulation, impact preview and maker-checker activation.
- [ ] Consolidate issue/Action ownership into one contextual handoff component.
- [ ] Persist assignment and protected notification outbox fact atomically; make notification retry separate from assignment truth.
- [ ] Write failing three-level OVERDUE sequence tests including replay, completion cancellation, no-route and stale directory state.
- [ ] Build the escalation editor on existing routing-policy revisions and runtime resolver.
- [ ] Verify escalation candidates remain a subset of current authority/visibility eligibility.
- [ ] Run `go test ./internal/access ./internal/authority ./internal/workflow ./internal/httpapi -count=1`.
- [ ] Run `go test -tags postgres ./internal/access ./internal/workflow ./internal/httpapi -count=1`.
- [ ] Run the affected access/Matter component tests and UI evidence manifest.

## Task 5: Rebase and finish scoring policies (#143)

**Files:**
- Modify: `internal/formcontract/advanced_scoring.go`
- Modify: `internal/evidence/completed_response.go`
- Modify: `internal/formpolicy/*`
- Modify: `cmd/seed-bank-reference/scoring_acceptance.go`
- Modify: `cmd/seed-bank-reference/form_policy_acceptance.go`
- Modify: `web/src/components/forms/FormPoliciesView.tsx`
- Modify: `web/src/components/forms/FormPolicyEditor.tsx`
- Test: `internal/formpolicy/*test.go`

- [ ] Create a current-main branch and cherry-pick/reimplement only the unique scoring and policy-selector changes from PR #129.
- [ ] Write failing integration tests for good, borderline, poor and unavailable scores submitted through distribution access.
- [ ] Prove response filters and sorts execute bounded server queries with stable cursors.
- [ ] Select only approved active form revisions in policy authoring.
- [ ] Simulate, independently approve and activate the poor-score policy.
- [ ] Prove one adverse episode/Matter, replay reuse, good-score no-op, safe unavailable-score behavior, authorized closure and later new episode.
- [ ] Run `go test ./internal/formcontract ./internal/evidence ./internal/formpolicy -count=1`.
- [ ] Run `go test -tags postgres ./internal/evidence ./internal/formpolicy -count=1`.
- [ ] Run `npm --prefix web test -- FormPoliciesView FormPolicyEditor`.

## Task 6: Implement vendor activation and both SMTP journeys (#139)

**Files:**
- Follow: `docs/superpowers/specs/2026-09-02-governed-vendor-activation-and-address-verification-design.md`
- Modify: `internal/thirdparty/*`
- Modify: `internal/evidence/invitation_delivery.go`
- Modify: `internal/httpapi/third_party_*`
- Modify: `internal/httpapi/vendor_*`
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/VendorWorkPanel.tsx`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `cmd/worker/*`

- [ ] Write the implementation plan for the already approved activation/address-verification design before code.
- [ ] Add versioned activation policy, proposal/simulation/approval/activation services and exact route registrations.
- [ ] Create/reuse one address-verification Matter/Action/request after onboarding submission.
- [ ] Use the hierarchy handoff command for verifier assignment and authenticated staff deep-link notification.
- [ ] Apply staff response to Action implementation; require independent verification and explicit closure.
- [ ] Fail closed until activation policy, assessment, Decisions, address verification, deficiencies and current authority pass.
- [ ] Restrict certification refresh to ACTIVE relationships.
- [ ] Run the two real SMTP journeys with safe hosted evidence and negative expiry/revoke/replay/audience/reassignment cases.

## Task 7: Implement linked-form issue remediation (#140)

**Files:**
- Follow: `docs/superpowers/specs/2026-09-02-program-matter-linked-form-remediation-design.md`
- Create: `internal/continuity/matter_form_remediation*.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/record_operations.go`
- Modify: `web/src/components/MatterInformationPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramIssuesPanel.tsx`
- Test: `internal/continuity/*matter_form*_test.go`
- Test: `web/src/components/MatterRecordWorkspace.test.tsx`

- [ ] Write failing domain tests for immutable binding validation, overlapping missing-item mappings and exact-version conflicts.
- [ ] Add the smallest authoritative binding schema and update durable-schema ownership.
- [ ] Implement propose/activate/send/read routes through current identity and material command guards.
- [ ] Reuse distribution/capture and display prior linked requests before send.
- [ ] Apply immutable response revisions idempotently and schedule the named outcome check.
- [ ] Replace duplicate mapped-item forms with the current request/review/verification action.
- [ ] Prove partial, poor, stale, revoked, replayed, failed-verification and authority-change paths.
- [ ] Run Go unit/PostgreSQL tests, affected TypeScript tests and the Program/Matter rendered journey.

## Task 8: Complete oversight and Today truth (#144, #145)

**Files:**
- Modify: `internal/oversight/*`
- Modify: `internal/today/*`
- Modify: `internal/workflow/*projector*`
- Modify: `web/src/components/oversight/*`
- Modify: `web/src/components/Today*`
- Modify: `docs/architecture/operational-read-models.md`

- [ ] Define duration and performance semantics for reassigned, returned, blocked and reopened Matters in executable tests.
- [ ] Detect missing lifecycle history and report unknown/excluded populations instead of calculating shortened durations.
- [ ] Filter restricted records before counts, samples, estimates and pagination.
- [ ] Project Today only from current actor-visible tasks/Actions/Decisions/requests/verification/routing or platform recovery state.
- [ ] Distinguish assigned, eligible shared queue, manager oversight and administrative recovery.
- [ ] Verify role matrices for System Administrator, GRC Administrator, CRO, Program Owner, reviewer, assignee and respondent.
- [ ] Run projection rebuild/replay tests and compare a hand-calculated history sample with the stored snapshot.

## Task 9: Repair hosted operational state (#146)

**Files:**
- Modify only after root cause: affected domain/worker/configuration files
- Update: operational runbook and regression test for each discovered failure class

- [ ] Export a redacted inventory of terminal outbox jobs with safe IDs, types, attempts and failure codes.
- [ ] Trace each to its originating aggregate/event and fix the common root cause before retry.
- [ ] Retry or compensate through the governed operation while retaining terminal history.
- [ ] Resolve unassigned issues through current routes or record explicit no-route recovery.
- [ ] Re-plan the GAID issue with owner, rationale and due date.
- [ ] Rebuild projections and record before/after Today/Oversight/Configure counts and high-water marks.

## Task 10: Exact-head release closure (#147)

**Files:**
- Modify: `README.md`
- Modify: `DESIGN.md`
- Modify: `docs/README.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/engineering/ui-use-case-acceptance-matrix.md`
- Modify: relevant product, architecture, ownership and acceptance documents

- [ ] Run `gofmt` and confirm `git diff --check` is clean.
- [ ] Run `go vet ./...`.
- [ ] Run `go test -race ./... -count=1`.
- [ ] Run `go test -tags postgres ./... -count=1` against a clean current schema.
- [ ] Run `npm --prefix web run typecheck`.
- [ ] Run `npm --prefix web test`.
- [ ] Run `npm --prefix web run check:ui-contracts`.
- [ ] Run `npm --prefix web run build`.
- [ ] Run `npm --prefix web run review:ui` and inspect every materially affected desktop/tablet/mobile, light/dark and density state.
- [ ] Run deployment configuration/readiness tests without printing secrets.
- [ ] Merge only the exact green head, deploy the same revision to API/worker/web and verify revision parity.
- [ ] Execute all #147 hosted journeys and attach safe evidence to #128.
- [ ] Close child issues only from their own acceptance evidence, then close #128 when every blocking child is complete.
