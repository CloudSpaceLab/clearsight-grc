# Oversight, Hierarchy and Role-Accurate Today Implementation Plan

> **Execution rule:** implement in bounded tranches with failing tests first. Each tranche must be independently reviewable, renderable and deployable.

**Goal:** Stabilize the Form Builder, remove duplicate owner actions, add governed reporting hierarchy and manager reassignment, make Today accurate for every role, and deliver a freshness-labelled CRO/GRC oversight workspace.

**Architecture:** Keep identity, authority, Workflow, outbox and continuity boundaries. Reuse effective-dated positions and parent relationships; add governed configuration revisions rather than direct mutations. Extend Today from canonical actor work plus capability-scoped operational exceptions. Build oversight as an authorization-scoped projection, not browser aggregation.

**Tech stack:** Go 1.25, PostgreSQL/pgx, React 19, TypeScript, React Aria components, Vitest/Testing Library, Playwright/render fixtures, SMTP outbox delivery.

---

## Tranche A — Workspace containment and one owner action

### Task 1: Prove and fix Form Builder pane containment

**Files:**
- Modify: `web/src/form-builder-workspace.css`
- Modify: `web/src/form-builder-responsive.css`
- Modify: `web/src/components/FormBuilder.test.tsx`
- Modify: `web/src/uiReviewFixtures.tsx`
- Modify: `docs/quality/rendered-ui-evidence.md`

- [ ] Add a failing structural test asserting the outline shell owns sticky/overflow behavior and the inner navigation does not.
- [ ] Add a long-outline fixture with the reusable-section disclosure expanded.
- [ ] Run `npm test -- src/components/FormBuilder.test.tsx` and confirm red.
- [ ] Move sticky height/overflow to `.form-builder-outline-shell`; make outline content normal flow; preserve responsive-sheet overrides.
- [ ] Run focused tests and `npm run review:ui`.
- [ ] Inspect light/dark at 1440, 1024, 768, 390 and 320/200%; record and correct the highest-impact defect.

### Task 2: Render one owner-change control per operation

**Files:**
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/MatterCurrentHandoff.tsx`
- Modify: `web/src/components/MatterDetailsPanel.tsx`
- Modify: `web/src/components/matterHandoff.ts`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.tsx`
- Modify: relevant Program workspace tests

- [ ] Add failing tests for `matter.assign` dominant/non-dominant states and assert exactly one **Change issue owner** button.
- [ ] Add equivalent Program owner tests.
- [ ] Lift owner-sheet intent to the record workspace or pass an explicit suppression/open callback contract; do not programmatically click a second DOM control.
- [ ] Keep one control in the handoff when assignment is dominant and one in details otherwise.
- [ ] Run focused Matter/Program tests, UI-contract checks and accessibility tests.

---

## Tranche B — Governed organization hierarchy and reassignment

### Task 3: Add versioned organization configuration records

**Files:**
- Create: `migrations/0000XX_organization_configuration_revisions.up.sql`
- Create: `migrations/0000XX_organization_configuration_revisions.down.sql`
- Create: `internal/organization/model.go`
- Create: `internal/organization/repository.go`
- Create: `internal/organization/postgres.go`
- Create: `internal/organization/service.go`
- Create: unit and PostgreSQL integration tests
- Modify: `docs/architecture/durable-schema-ownership.md`

- [ ] Write failing schema tests for effective-dated revision, position snapshot, parent edge, occupant/deputy change, maker/checker, rationale, status and version.
- [ ] Add bounded indexes for tenant/entity/status/effective time and active position traversal.
- [ ] Implement draft, preview, submit, approve, schedule, activate and rollback transitions.
- [ ] Reject self-approval, cross-entity references, cycles, excessive depth, inactive principals, duplicate active codes and stale expected versions.
- [ ] Commit authoritative row changes, append-only governance decision, outbox event and required activation timer in one transaction.
- [ ] Run migration rollback/reapply and PostgreSQL integration gates.

### Task 4: Add hierarchy impact preview and simulation

**Files:**
- Create: `internal/organization/impact.go`
- Create: `internal/organization/impact_postgres.go`
- Create: `internal/organization/impact_test.go`
- Modify: `internal/authority` integration where current positions are resolved

- [ ] Write failing tests for descendants, active assignments, routing gaps, escalation changes, vacant positions, affected delegations and bounded counts.
- [ ] Implement exact indexed traversal with a hard depth and result cap.
- [ ] Return population checked, truncated/unknown counts and source version.
- [ ] Ensure preview never grants authority or mutates active organization state.

### Task 5: Permit narrowly governed manager reassignment

**Files:**
- Create: `internal/authority/reassignment.go`
- Create: `internal/authority/reassignment_postgres.go`
- Create: unit and PostgreSQL integration tests
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: command lifecycle tests

- [ ] Add failing tests for current owner, direct manager, higher ancestor, deputy/delegate and governed emergency actor.
- [ ] Add failing denial tests for peer, inactive/vacant ancestor, cycle, stale revision, cross-tenant/entity, protected visibility, conflict, revoked role and authority outage.
- [ ] Resolve the verified actor's current active position and bounded ancestor chain at command time.
- [ ] Authorize only approved owner-change command classes; keep execution/review/authorization/signing permissions unchanged.
- [ ] Revalidate replacement candidate through current scoped authority.
- [ ] Record hierarchy/policy version and reason in the existing material command event.

### Task 6: Preserve event-backed assignment notification

**Files:**
- Modify: existing staff-assignment notification consumer and tests
- Modify: Matter/Program/Action assignment event payload builders as necessary

- [ ] Add failing tests proving manager-initiated Matter, Program and Action handoffs produce the same safe canonical event.
- [ ] Verify one redacted receipt per event/recipient/kind, including replay and ambiguous SMTP outcomes.
- [ ] Include record reference, responsibility, reason, due date, next action and authenticated deep link.
- [ ] Do not include recipient addresses or links in logs/events/API output.

### Task 7: Build the Configure hierarchy workspace

**Files:**
- Create: `web/src/organizationApi.ts`
- Create: `web/src/components/configure/PeopleInventory.tsx`
- Create: `web/src/components/configure/PositionsRolesWorkspace.tsx`
- Create: `web/src/components/configure/ReportingLinesWorkspace.tsx`
- Create: `web/src/components/configure/EscalationRoutesWorkspace.tsx`
- Create: `web/src/components/configure/OrganizationChangeHistory.tsx`
- Modify: `web/src/components/configure/PeopleAccessSection.tsx`
- Modify: `web/src/configure-workspace.css`
- Modify: `web/src/identity-access.css`
- Modify: API route registry/OpenAPI and handler tests

- [ ] Add failing component tests for navigation, search, hierarchy/table equivalence, orphan/cycle state and empty/unavailable states.
- [ ] Add focused sheets for position/reporting edits with reason and effective date.
- [ ] Add impact preview, maker/checker approval, scheduled activation and rollback states.
- [ ] Keep directory people read-only and distinguish workspace access from material authority.
- [ ] Migrate all new controls to shared design-system components.
- [ ] Render populated, empty, stale, partial, draft, checker and rollback fixtures in both themes and responsive sizes.

---

## Tranche C — Role-accurate Today

### Task 8: Remove static Today fallback behavior

**Files:**
- Modify: `web/src/App.tsx`
- Modify: `web/src/staticDemo.ts`
- Modify: `internal/today/service.go`
- Modify: composition and tests under `cmd/api`

- [x] Add failing tests proving a failed Today request produces unavailable state even in demo presentation.
- [x] Remove `fallbackItems` from the browser.
- [x] Ensure deployed demo Today reads normal seeded database records through the same actor-scoped projection.
- [x] Prevent `today.DemoItems()` and fixture-specific Today arrays from runtime composition; keep isolated unit fixtures only where explicitly constructed by tests.

### Task 9: Project stored work for every operational role

**Files:**
- Modify: `internal/today/workflow_projection.go`
- Create: `internal/today/operational_attention.go`
- Create: `internal/today/operational_attention_postgres.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: Today API/model tests

- [x] Add failing role-matrix tests for performer, owner, reviewer, challenger, authorizer, signatory, transmitter, acknowledgement recorder, manager and CRO/GRC oversight.
- [x] Keep personal Today limited to currently assigned/routed work; oversight visibility alone must not create a Today item.
- [ ] Deduplicate exact target/responsibility/step, sort overdue/materiality/deadline and return generation/source health.
- [ ] Recheck exact source visibility before `LIMIT`; use keyset continuation when the cap is reached.

### Task 10: Add System Administrator operational attention

**Files:**
- Modify: `internal/today/operational_attention_postgres.go`
- Modify: `internal/operations` and identity/routing repositories where bounded summaries are needed
- Modify: `web/src/components/TodayInterventions.tsx`
- Modify: Today tests and fixtures

- [ ] Add failing tests for actionable inactive/stale source, provisioning failure, unresolved routing, failed timer, pending independent approval and scheduled activation.
- [x] Include only exceptions for which the verified actor holds the current capability/responsibility.
- [x] Give every item a real configuration/operations target and executable or explanatory recovery action.
- [x] Ensure System Administrator does not receive unrelated risk approvals.
- [ ] Show the population/source checked and generation time in empty/partial states.

---

## Tranche D — CRO/GRC oversight

### Task 11: Define the oversight projection schema and contracts

**Files:**
- Create: migration for oversight projection/high-water rows
- Create: `internal/oversight/model.go`
- Create: `internal/oversight/repository.go`
- Create: `internal/oversight/projection.go`
- Create: tests and schema ownership entry

- [x] Write failing contracts for scope, category, criticality, state, aging, owner/function, source versions, unknown counts and freshness.
- [x] Add deterministic cycle-time and estimate cohort semantics with minimum sample and confidence class.
- [x] Prohibit static/demo metrics and unknown-to-zero coercion.
- [ ] Define cardinality, retention, partition/index and recomputation bounds.

### Task 12: Build and maintain the projection

**Files:**
- Create: projection worker/consumer under `internal/oversight`
- Modify: worker composition and runtime maintenance
- Add PostgreSQL integration and recovery tests

- [ ] Consume canonical Program/Matter/Action/Workflow/escalation/outcome events idempotently.
- [x] Maintain source high-water marks and deduplicated bounded recomputation jobs.
- [ ] Test replay, out-of-order events, retry, stale source, deletion/retirement, legal-entity isolation and recovery.
- [ ] Validate query plans and reference workload targets.

### Task 13: Add authorization-scoped oversight API

**Files:**
- Add `internal/httpapi/oversight_handlers.go`
- Modify canonical route registry and runtime OpenAPI
- Add API tests

- [x] Require verified identity and exact tenant/entity scope.
- [x] Scope CRO/GRC data through current capability and visibility rules; filter restricted records in the repository query.
- [ ] Provide bounded filters, keyset drilldowns, freshness and partial-coverage metadata.
- [x] Fail closed on scope/authority failure and never load broad populations for browser filtering.

### Task 14: Build the oversight workspace

**Files:**
- Create: `web/src/oversightApi.ts`
- Create: `web/src/components/oversight/OversightWorkspace.tsx`
- Create: intervention, risk pressure, resolution outlook, operating performance and improvement components
- Create: token-owned chart/table CSS and tests
- Modify navigation and role visibility

- [ ] Add failing tests for current/stale/partial/no-data/restricted states and filter-to-drilldown continuity.
- [x] Lead with intervention list; use ordered category bars, accessible trends and confidence ranges with table alternatives.
- [x] Show median/p75, sample sizes and exclusions; do not create a composite employee score.
- [ ] Add authority-scoped individual drilldown with workload/reassignment/blocked context.
- [ ] Render and inspect all required theme/viewport fixtures.

---

## Final verification and delivery

- [ ] Run affected focused tests after every task.
- [ ] Run `npm test`, `npm run typecheck`, `npm run check:copy`, `npm run check:ui-contracts`, `npm run review:ui` and `npm run build`.
- [ ] Run `go test ./...`, `go test -race ./...`, `go test -tags postgres ./...`, `go vet ./...` and migration rollback/reapply.
- [ ] Run `git diff --check` and secret/static-demo scans.
- [ ] Inspect exact-head rendered evidence and correct the highest-impact defect before re-rendering.
- [ ] Request exact-head code review, merge only when green, deploy `main`, verify readiness reports the merge SHA, and run actor-specific Today/hierarchy/oversight acceptance on the hosted application.
