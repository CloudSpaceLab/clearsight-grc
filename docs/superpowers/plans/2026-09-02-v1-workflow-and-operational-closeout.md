# V1 Workflow and Operational Closeout Implementation Plan

> **Execution:** Apply each task test-first on a current-main worktree. Do not retain unreleased compatibility paths or introduce parallel task, evidence, notification, form, authority, or reporting stores.

**Goal:** Close #139, #140, #144, #145, and #146 with reconstructable domain state, real SMTP journeys, role-accurate operational reads, and exact-head hosted evidence.

**Architecture:** Extend the existing third-party relationship, Matter/Action/verification, form distribution/capture, workflow projection, outbox, Today, and Oversight boundaries. Material commands resolve verified identity and current authority, update authoritative rows, append events, enqueue protected notifications, and schedule projections in one transaction. Submission, implementation, verification, closure, and relationship activation stay distinct.

## Task 1: Govern relationship activation and address-verification setup (#139)

**Files:**
- Create: `internal/thirdparty/activation_policy*.go`
- Create: `internal/thirdparty/relationship_activation*.go`
- Create: `internal/thirdparty/address_verification*.go`
- Modify: `internal/thirdparty/postgres.go`
- Modify: `internal/thirdparty/repository.go`
- Modify: `internal/httpapi/third_party_handlers.go`
- Modify: `internal/httpapi/route_registry.go`

- [ ] Write failing domain tests for policy proposal, simulation, independent approval, effective selection, rollback, and tenant/entity isolation.
- [ ] Write failing activation tests for every policy gate, stale versions, self-approval, and atomic receipt/event/outbox persistence.
- [ ] Implement immutable policy versions and exact effective-time lookup.
- [ ] Implement onboarding-submission setup that creates or reuses one address-verification Matter, Action, verification contract, and internal Evidence Request.
- [ ] Reuse the governed Action handoff and protected assignment notification; reassignment revokes prior request access.
- [ ] Apply staff submission as Action implementation only; require independent verification and explicit Matter closure.
- [ ] Remove the duplicate external address-verification Vendor Work option and reject certification refresh unless the relationship is ACTIVE.

## Task 2: Expose the vendor journey in one workspace (#139)

**Files:**
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Modify: `web/src/components/VendorWorkPanel.tsx`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: affected API clients and tests

- [ ] Add failing component tests for the state-dependent dominant actions and activation gate explanations.
- [ ] Show the exact current request, assignee, evidence-review, verification, closure, activation, and certification action without duplicate owner or evidence controls.
- [ ] Prove desktop/mobile, light/dark, keyboard, 200% reflow, and axe states.

## Task 3: Bind approved forms to existing Matters (#140)

**Files:**
- Create: `internal/continuity/matter_form_remediation*.go`
- Modify: `internal/continuity/postgres.go`
- Modify: `internal/httpapi/matter_operations.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `web/src/components/MatterInformationPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramIssuesPanel.tsx`

- [ ] Write failing tests for immutable binding versions, exact Matter/form/field/missing-item validation, overlap, stale authority, and subject isolation.
- [ ] Create/reuse one standard form distribution for the active binding and show prior requests before send.
- [ ] Apply one immutable submitted response idempotently to only mapped missing items and schedule the named outcome check.
- [ ] Keep partial, poor, stale, revoked, replayed, mismatched, and failed-verification responses open with one corrective action.
- [ ] Replace mapped-item `Add information` forms with the current request/review/check/close action; retain direct correction only for unmapped items.

## Task 4: Make Today canonical and role-accurate (#145)

**Files:**
- Delete production use of: `internal/workflow.DemoTasks()`
- Modify: `cmd/api/services_memory.go`
- Modify: `internal/workflow/*projector*.go`
- Modify: `cmd/api/today_service.go`
- Modify: `internal/today/*`
- Modify: affected Today UI/tests

- [ ] Write a failing architecture test that rejects production composition of hardcoded workflow tasks.
- [ ] Project current Matter tasks, Actions, Decisions, Evidence Requests, verification, routing failures, and operational recovery from stored records.
- [ ] Distinguish assigned work, eligible queues, manager oversight, and administrator recovery.
- [ ] Re-resolve scope, visibility, delegation, absence, revocation, and current action authorization before list and launch.
- [ ] Suppress terminal/superseded work and expose projection lag truthfully.

## Task 5: Complete Oversight history and estimates (#144)

**Files:**
- Modify: `internal/oversight/*`
- Modify: `internal/workflow/*projector*.go`
- Modify: `web/src/components/oversight/*`
- Modify: `docs/architecture/operational-read-models.md`

- [ ] Write executable duration semantics for reassigned, returned, blocked, and reopened examples.
- [ ] Detect missing lifecycle anchors and report unknown/excluded populations instead of shortened durations.
- [ ] Attribute actor time separately from blocked, shared-process, and reassignment time.
- [ ] Filter restricted records before aggregation and small-sample suppression.
- [ ] Show period, sample size, confidence, exclusions, unknowns, projection version, and source high-water marks.
- [ ] Prove bounded rebuild/replay and hand-calculate representative histories.

## Task 6: Repair hosted operations from governed commands (#146)

**Files:**
- Modify only files implicated by the redacted failure inventory
- Modify: operational runbook and regression tests for each confirmed failure class

- [ ] Capture safe before-state IDs, event types, failure codes, attempt counts, unassigned work, overdue work, and projection marks.
- [ ] Trace every terminal job to its originating aggregate and repair the root cause before retry/compensation.
- [ ] Resolve unassigned and overdue records through current route/handoff commands with rationale; never patch/delete rows to improve counts.
- [ ] Rebuild projections and compare Today/Oversight/Configure against the authoritative post-repair state.

## Task 7: Exact-head acceptance and closure

- [ ] Run focused unit, PostgreSQL, API contract, TypeScript, copy, rendered-state, accessibility, and clean-schema tests after each task.
- [ ] Run repository-wide Go race/vet and web typecheck/test/build/UI-contract gates.
- [ ] Merge only the exact green head and deploy the same revision to API, worker, and web.
- [ ] Execute both real SMTP journeys and negative expiry, revoke, replay, audience, reassignment, and delivery-failure cases without exposing secrets.
- [ ] Attach safe IDs, versions, states, timestamps, counts, and source high-water marks to each issue; close only after its own acceptance checklist is proven.
