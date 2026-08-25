# Matter Operational Completeness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every supported Matter journey executable from Work, including visible ownership, fact correction, assignment, action management, outcome capture and lifecycle completion.

**Architecture:** Add object-specific Matter update events and commands to the existing continuity aggregate, then expose them through the current verified-identity material-command stack. Add a bounded actor-scoped operation read so the UI can show the responsible participant and valid action without duplicating authority state. Replace target-record accordion expansion with a dedicated Matter workspace while retaining the existing list for portfolio scanning.

**Tech Stack:** Go 1.25 standard HTTP, PostgreSQL/pgx event projections and outbox, React 19, TypeScript 7, Vite 8, Vitest/Testing Library, existing CSS semantic tokens.

---

## File map

- Create `internal/continuity/matter_edits.go`: validated object-specific detail, context, owner and Action update commands.
- Create `internal/continuity/matter_edits_test.go`: memory-domain tests for changes, reconstruction and conflicts.
- Modify `internal/continuity/replay.go`: new event constants and replay cases.
- Modify `internal/continuity/memory.go`: current projection updates for the new events.
- Modify `internal/continuity/postgres.go`: transactional PostgreSQL projection updates for the new events.
- Create `internal/continuity/matter_edits_postgres_integration_test.go`: row/event/outbox atomicity tests.
- Create `internal/httpapi/record_operations.go`: shared actor-scoped operation and participant response types.
- Create `internal/httpapi/matter_operations.go`: Matter current operation and participant projection.
- Create `internal/httpapi/matter_operations_test.go`: operation visibility, routing and fail-closed tests.
- Modify `internal/httpapi/continuity_handlers.go`, `command_lifecycle.go`, `route_registry.go`: command handlers, lifecycle policy and routes.
- Modify `internal/httpapi/continuity_handlers_test.go`, `command_lifecycle_test.go`, `matter_command_authority_test.go`: identity binding and route behavior.
- Modify `internal/workflow/matter_action_projector_postgres.go`: consume Action update/assignment events.
- Modify `internal/workflow/matter_action_projector_postgres_integration_test.go`: prove reassignment convergence.
- Create `web/src/matterOperationsApi.ts`: typed Matter operation reads and command clients.
- Create `web/src/components/MatterRecordWorkspace.tsx`: dedicated record orchestration.
- Create `web/src/components/MatterCurrentHandoff.tsx`: dominant action and responsibility.
- Create `web/src/components/MatterDetailsPanel.tsx`: details, owner and Program linking.
- Create `web/src/components/MatterInformationPanel.tsx`: facts, missing information and contradictions.
- Create `web/src/components/MatterActionsPanel.tsx`: add/edit/assign/transition Actions.
- Create `web/src/components/MatterOutcomePanel.tsx`: define checks and record independent results.
- Create `web/src/components/MatterDecisionResponsePanel.tsx`: decision and response creation/transitions.
- Create `web/src/components/MatterRecordWorkspace.test.tsx`: complete cross-role component journey.
- Modify `web/src/components/MattersWorkspace.tsx`: route target records to the dedicated workspace.
- Modify `web/src/AppViews.tsx`, `web/src/types.ts`, `web/src/continuityCommands.ts`: navigation, types and clients.
- Create `web/src/matter-record.css`: responsive operational layout using existing semantic tokens.
- Create `web/src/operationalCoverage.ts` and `web/src/operationalCoverage.test.ts`: material command → UI coverage gate.
- Modify `api/runtime.openapi.json`, `docs/implementation-plan.md`, `docs/engineering/ui-use-case-acceptance-matrix.md`, `docs/quality/program-matter-acceptance-tests.md`, `DESIGN.md`: synchronized contracts and evidence.

### Task 1: Add versioned Matter detail and information commands

**Files:**
- Create: `internal/continuity/matter_edits.go`
- Create: `internal/continuity/matter_edits_test.go`
- Modify: `internal/continuity/replay.go`

- [ ] **Step 1: Write failing domain tests for detail correction and missing-information resolution**

```go
func TestUpdateMatterDetailsAndResolveMissingFact(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter := mustCreateMatter(t, service, CreateMatterInput{
		TenantID: "bank", Type: MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		KnownFacts: json.RawMessage(`{"filing_channel":"licensed DPCO"}`),
		MissingFacts: json.RawMessage(`["final DPCO checklist"]`),
	})
	updated, err := service.UpdateMatterDetails(t.Context(), UpdateMatterDetailsInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Annual return filing", Summary: "Update the filing process and evidence ownership.",
		Priority: 4, DueAt: matter.Matter.DueAt, ActorID: "owner", Rationale: "Clarify the approved scope.",
	})
	if err != nil { t.Fatal(err) }
	resolved, err := service.ChangeMatterContext(t.Context(), ChangeMatterContextInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: updated.Matter.Version,
		Kind: MatterContextResolveMissing, Key: "final_dpco_checklist", Label: "final DPCO checklist",
		Value: json.RawMessage(`"Checklist v3"`), Rationale: "DPCO approved version 3.",
		EvidenceReferences: json.RawMessage(`["artifact-checklist-v3"]`), ActorID: "owner",
	})
	if err != nil { t.Fatal(err) }
	var facts map[string]any
	_ = json.Unmarshal(resolved.Matter.KnownFacts, &facts)
	if facts["final_dpco_checklist"] != "Checklist v3" { t.Fatalf("fact missing: %#v", facts) }
	if string(resolved.Matter.MissingFacts) != "[]" { t.Fatalf("missing fact not resolved: %s", resolved.Matter.MissingFacts) }
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal/continuity -run TestUpdateMatterDetailsAndResolveMissingFact -count=1`

Expected: FAIL because `UpdateMatterDetailsInput` and `ChangeMatterContextInput` are undefined.

- [ ] **Step 3: Add command inputs, validation and event types**

Define in `matter_edits.go`:

```go
type MatterContextChangeKind string

const (
	MatterContextAddFact MatterContextChangeKind = "ADD_FACT"
	MatterContextCorrectFact MatterContextChangeKind = "CORRECT_FACT"
	MatterContextRetireFact MatterContextChangeKind = "RETIRE_FACT"
	MatterContextAddMissing MatterContextChangeKind = "ADD_MISSING"
	MatterContextResolveMissing MatterContextChangeKind = "RESOLVE_MISSING"
	MatterContextAddContradiction MatterContextChangeKind = "ADD_CONTRADICTION"
	MatterContextResolveContradiction MatterContextChangeKind = "RESOLVE_CONTRADICTION"
)

type UpdateMatterDetailsInput struct {
	TenantID string; MatterID string; ExpectedVersion int64
	Title string; Summary string; Priority int; DueAt *time.Time
	Scope json.RawMessage; ActorID string; Rationale string
}

type ChangeMatterContextInput struct {
	TenantID string; MatterID string; ExpectedVersion int64
	Kind MatterContextChangeKind; Key string; Label string; Value json.RawMessage
	Rationale string; EvidenceReferences json.RawMessage; ActorID string
}
```

Add `EventMatterDetailsUpdated` and `EventMatterContextChanged` in `replay.go`. Each service method must load with `matterForMutation`, validate non-empty actor/rationale, normalize JSON, mutate a copy of the current `Matter`, and call `applyMatterValue` with the new event. `ChangeMatterContext` must use decoded `map[string]json.RawMessage` and `[]string` values and apply each change kind explicitly; it must reject duplicate missing labels, unsupported scalar JSON and absent keys.

- [ ] **Step 4: Add replay cases and run domain tests**

Handle both new events exactly like `EventMatterStateChanged`: decode the full `Matter` payload and replace `aggregate.Matter`.

Run: `go test ./internal/continuity -run 'TestUpdateMatterDetailsAndResolveMissingFact|TestMatter' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the domain commands**

```bash
git add internal/continuity/matter_edits.go internal/continuity/matter_edits_test.go internal/continuity/replay.go
git commit -m "feat(continuity): add versioned Matter corrections"
```

### Task 2: Add Matter and Action assignment/update commands

**Files:**
- Modify: `internal/continuity/matter_edits.go`
- Modify: `internal/continuity/matter_edits_test.go`
- Modify: `internal/continuity/replay.go`

- [ ] **Step 1: Write failing tests for owner and Action updates**

```go
func TestAssignMatterAndUpdateAction(t *testing.T) {
	service, matter := editableMatterFixture(t)
	action := matter.Actions[0]
	assigned, err := service.AssignMatter(t.Context(), AssignMatterInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: "owner-2", ActorID: "owner-1", Rationale: "Move accountability to privacy operations.",
	})
	if err != nil { t.Fatal(err) }
	updated, err := service.UpdateAction(t.Context(), UpdateActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID,
		ExpectedVersion: assigned.Matter.Version, Title: action.Title,
		Description: "Assign every annual-return section and attach its source.",
		DueAt: action.DueAt, ActorID: "owner-2", Rationale: "Clarify the required evidence.",
	})
	if err != nil { t.Fatal(err) }
	assignedAction, err := service.AssignAction(t.Context(), AssignActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID,
		ExpectedVersion: updated.Matter.Version, OwnerPrincipalID: "performer-2",
		ActorID: "owner-2", Rationale: "Assign the active privacy operations owner.",
	})
	if err != nil { t.Fatal(err) }
	if assignedAction.Matter.OwnerPrincipalID != "owner-2" || assignedAction.Actions[0].OwnerPrincipalID != "performer-2" {
		t.Fatalf("assignments not updated: %#v", assignedAction)
	}
}
```

- [ ] **Step 2: Verify the focused test fails**

Run: `go test ./internal/continuity -run TestAssignMatterAndUpdateAction -count=1`

Expected: FAIL with undefined assignment/update types.

- [ ] **Step 3: Implement explicit assignment and Action update commands**

Add `AssignMatterInput`, `UpdateActionInput` and `AssignActionInput`, plus `EventMatterOwnerChanged`, `EventActionUpdated` and `EventActionAssigned`. `AssignMatter` and `AssignAction` must reject empty/new-equal owners and require rationale. `UpdateAction` changes only title, description and due date; it preserves owner/status/implemented time, requires title/description/rationale, rejects closed/cancelled Matters and increments only the Action version once.

- [ ] **Step 4: Add replay cases and pass tests**

`EventMatterOwnerChanged` replaces `aggregate.Matter`; `EventActionUpdated` and `EventActionAssigned` use `upsertAction`.

Run: `go test ./internal/continuity -run 'TestAssignMatterAndUpdateAction|TestUpdateMatter' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit assignment commands**

```bash
git add internal/continuity/matter_edits.go internal/continuity/matter_edits_test.go internal/continuity/replay.go
git commit -m "feat(continuity): add governed Matter assignments"
```

### Task 3: Persist the new events atomically in memory and PostgreSQL

**Files:**
- Modify: `internal/continuity/memory.go`
- Modify: `internal/continuity/postgres.go`
- Create: `internal/continuity/matter_edits_postgres_integration_test.go`

- [ ] **Step 1: Write a PostgreSQL integration test**

The test must execute one detail update, one context resolution and one Action reassignment, then assert the current rows and exactly one continuity/outbox event per command:

```go
for _, eventType := range []string{EventMatterDetailsUpdated, EventMatterContextChanged, EventActionUpdated, EventActionAssigned} {
	var continuityCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matterID, eventType).Scan(&continuityCount); err != nil { t.Fatal(err) }
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matterID, eventType).Scan(&outboxCount); err != nil { t.Fatal(err) }
	if continuityCount != 1 || outboxCount != 1 { t.Fatalf("%s not atomic: %d/%d", eventType, continuityCount, outboxCount) }
}
```

- [ ] **Step 2: Run the integration test and verify failure**

Run: `go test -tags 'postgres postgresintegration' ./internal/continuity -run TestMatterEditsPersistEventAndOutbox -count=1`

Expected: FAIL because projections reject the new event types.

- [ ] **Step 3: Extend memory and PostgreSQL projections**

In `memory.go`, group the three full-Matter events with `EventMatterStateChanged`. Group `EventActionUpdated` and `EventActionAssigned` with Action upsert handling.

In `postgres.go`, route the full-Matter events to the existing `UPDATE matters` statement. Route `EventActionUpdated` and `EventActionAssigned` to a statement that updates `title`, `description`, `owner_principal_id`, `due_at`, `status`, `implemented_at`, `updated_at` and `version` under tenant/matter/action scope.

- [ ] **Step 4: Run memory and PostgreSQL tests**

Run: `go test ./internal/continuity && go test -tags postgres ./internal/continuity`

Expected: PASS.

- [ ] **Step 5: Commit projections**

```bash
git add internal/continuity/memory.go internal/continuity/postgres.go internal/continuity/matter_edits_postgres_integration_test.go
git commit -m "feat(continuity): persist Matter corrections atomically"
```

### Task 4: Expose verified material-command routes

**Files:**
- Modify: `internal/httpapi/continuity_handlers.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: `internal/httpapi/continuity_handlers_test.go`
- Modify: `internal/httpapi/command_lifecycle_test.go`
- Modify: `internal/httpapi/matter_command_authority_test.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing route and identity-binding tests**

Test these routes and canonical command names:

```text
POST /api/v1/matters/{id}/details                 matter.details.update
POST /api/v1/matters/{id}/context-changes         matter.context.change
POST /api/v1/matters/{id}/assignment              matter.assign
POST /api/v1/matters/{id}/actions/{action_id}     matter.action.update
POST /api/v1/matters/{id}/actions/{action_id}/assignment matter.action.assign
```

For assignment, assert the body `actor_id` is overwritten by verified identity while `owner_principal_id` remains the requested subject only after current-route eligibility succeeds.

- [ ] **Step 2: Run focused HTTP tests and verify failure**

Run: `go test ./internal/httpapi -run 'TestMatterEditRoutes|TestMatterAssignmentAuthority' -count=1`

Expected: FAIL because routes are absent.

- [ ] **Step 3: Add handlers and material routes**

Use owner responsibility/materiality floors:

```go
material("/api/v1/matters/{id}/details", "matter.details.update", a.updateMatterDetails, commandPolicy{ObjectType:"MATTER", Responsibility:authority.ResponsibilityOwner, Materiality:2}),
material("/api/v1/matters/{id}/context-changes", "matter.context.change", a.changeMatterContext, commandPolicy{ObjectType:"MATTER", Responsibility:authority.ResponsibilityOwner, Materiality:2}),
material("/api/v1/matters/{id}/assignment", "matter.assign", a.assignMatter, commandPolicy{ObjectType:"MATTER", Responsibility:authority.ResponsibilityOwner, Materiality:3}),
material("/api/v1/matters/{id}/actions/{action_id}", "matter.action.update", a.updateMatterAction, commandPolicy{ObjectType:"MATTER", Responsibility:authority.ResponsibilityOwner, Materiality:2}),
material("/api/v1/matters/{id}/actions/{action_id}/assignment", "matter.action.assign", a.assignMatterAction, commandPolicy{ObjectType:"MATTER", Responsibility:authority.ResponsibilityOwner, Materiality:3}),
```

Handlers decode the exact input, overwrite route IDs and call the domain service. Extend `lifecycleCommandPolicy` so assignment/update targets must be visible to the Matter and allowed by a current `ACCOUNTABLE_OWNER` resolution. Never bind `owner_principal_id` as an actor field.

- [ ] **Step 4: Synchronize the executable API contract and pass tests**

Add the five paths to `api/runtime.openapi.json` as `MATERIAL_COMMAND`. Run:

`go test ./internal/httpapi -run 'TestMatter|TestRuntime' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit HTTP commands**

```bash
git add internal/httpapi api/runtime.openapi.json
git commit -m "feat(api): expose governed Matter maintenance commands"
```

### Task 5: Add bounded Matter operation and participant reads

**Files:**
- Create: `internal/httpapi/record_operations.go`
- Create: `internal/httpapi/matter_operations.go`
- Create: `internal/httpapi/matter_operations_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing operation projection tests**

Build a fixture where CRO views a Matter owned by Program Owner and an outcome check assigned to Internal Auditor. Assert the response shows all three names, `can_act=false` for the CRO-owned Action transition, and the correct assignment explanation. Add no-route and restricted-visibility tests.

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/httpapi -run TestMatterOperations -count=1`

Expected: FAIL because `/operations` is absent.

- [ ] **Step 3: Implement `GET /api/v1/matters/{id}/operations`**

Return:

```go
type RecordOperation struct {
	Command string `json:"command"`
	SubresourceID string `json:"subresource_id,omitempty"`
	Label string `json:"label"`
	Responsibility string `json:"responsibility"`
	CanAct bool `json:"can_act"`
	AssignedTo *authority.Principal `json:"assigned_to,omitempty"`
	Candidates []authority.Principal `json:"candidates,omitempty"`
	Reason string `json:"reason"`
	AllowedTargets []string `json:"allowed_targets,omitempty"`
}
```

Load exactly one visible Matter, derive lifecycle work with `continuity.CompileMatterWork`, resolve each current responsibility through `a.deps.Authority`, filter candidates with `MatterVisibleTo`, and add stored Action/Matter owners even when the viewer cannot act. Authority failure returns a safe read with unavailable operations and no fabricated assignee.

- [ ] **Step 4: Add route/contract and pass tests**

Register an authenticated read and update the runtime contract. Run:

`go test ./internal/httpapi -run 'TestMatterOperations|TestWorkflowTaskRead' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit operation reads**

```bash
git add internal/httpapi/record_operations.go internal/httpapi/matter_operations.go internal/httpapi/matter_operations_test.go internal/httpapi/route_registry.go api/runtime.openapi.json
git commit -m "feat(api): explain current Matter ownership and actions"
```

### Task 6: Reproject Action reassignment to the new performer

**Files:**
- Modify: `internal/workflow/matter_action_projector_postgres.go`
- Modify: `internal/workflow/matter_action_projector_postgres_integration_test.go`

- [ ] **Step 1: Write the failing reassignment projection test**

Publish `ACTION_ADDED`, then `ACTION_ASSIGNED` with a different owner. Assert one workflow/task remains, its `principal_id` is the new owner, the old owner receives no active task and the task version increments.

- [ ] **Step 2: Verify failure**

Run: `go test -tags 'postgres postgresintegration' ./internal/workflow -run TestMatterActionProjectorReassignsCurrentTask -count=1`

Expected: FAIL because `ACTION_ASSIGNED` is ignored.

- [ ] **Step 3: Consume the update event**

Add `continuity.EventActionUpdated` and `continuity.EventActionAssigned` to the accepted event list. Keep the existing inbox receipt, workflow/task upsert and context generation so edits and reassignment are idempotent and do not create parallel task state.

- [ ] **Step 4: Run projector tests**

Run: `go test -tags postgres ./internal/workflow -run MatterActionProjector -count=1`

Expected: PASS.

- [ ] **Step 5: Commit projection convergence**

```bash
git add internal/workflow/matter_action_projector_postgres.go internal/workflow/matter_action_projector_postgres_integration_test.go
git commit -m "feat(workflow): reproject reassigned Matter actions"
```

### Task 7: Add typed web clients and operation state

**Files:**
- Create: `web/src/matterOperationsApi.ts`
- Create: `web/src/matterOperationsApi.test.ts`
- Modify: `web/src/types.ts`
- Modify: `web/src/continuityCommands.ts`

- [ ] **Step 1: Write failing client payload tests**

Assert exact paths and payloads for details, context change, assignment, Action update, Action add, outcome definition, result recording and lifecycle transition. Verify no client payload sends `actor_id`, reviewer ID, approver ID or tenant from browser state.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- matterOperationsApi.test.ts`

Expected: FAIL because the module is absent.

- [ ] **Step 3: Implement typed operation/client contracts**

Define `MatterOperation`, `MatterOperationsResponse`, `MatterDetailsInput`, `MatterContextChangeInput`, `MatterAssignmentInput`, `MatterActionUpdateInput` and the command functions. Reuse the existing `command<T>` helper by exporting it from `continuityCommands.ts` as an internal named helper; do not create another tenant/identity binding layer.

- [ ] **Step 4: Run client tests and typecheck**

Run: `cd web && npm test -- matterOperationsApi.test.ts && npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit typed clients**

```bash
git add web/src/matterOperationsApi.ts web/src/matterOperationsApi.test.ts web/src/types.ts web/src/continuityCommands.ts
git commit -m "feat(web): add typed Matter operation clients"
```

### Task 8: Build the dedicated Matter record shell and current handoff

**Files:**
- Create: `web/src/components/MatterRecordWorkspace.tsx`
- Create: `web/src/components/MatterCurrentHandoff.tsx`
- Create: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `web/src/components/MattersWorkspace.tsx`
- Modify: `web/src/AppViews.tsx`
- Create: `web/src/matter-record.css`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write failing rendering tests**

Test that a target route renders a dedicated heading, owner name, due date, current blocker and exactly one dominant action. A CRO fixture must see “Assigned to Program Owner” without an enabled “Update action” button. The list route must retain search and cards.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx`

Expected: FAIL because the record workspace does not exist.

- [ ] **Step 3: Implement the dedicated shell**

`MatterRecordWorkspace` loads aggregate and operations together, owns reload/conflict state, and passes one `onUpdated` callback to panels. `MatterCurrentHandoff` selects the first current operation matching `next_action`, otherwise shows the responsible explanation without inventing a control. `MattersWorkspace` renders the dedicated component when `targetID` is present and a “Back to issues and changes” action clears the route.

- [ ] **Step 4: Add responsive CSS and pass tests**

Use semantic variables only. At `max-width: 900px` and 200% replacement, use one column; at `max-width: 560px`, focused forms occupy the full content width and actions remain at least 44px.

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx ExactTargetWorkspaces.test.tsx && npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit the record shell**

```bash
git add web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterCurrentHandoff.tsx web/src/components/MatterRecordWorkspace.test.tsx web/src/components/MattersWorkspace.tsx web/src/AppViews.tsx web/src/matter-record.css web/src/main.tsx
git commit -m "feat(web): add dedicated Matter workspace"
```

### Task 9: Add issue detail, owner and information forms

**Files:**
- Create: `web/src/components/MatterDetailsPanel.tsx`
- Create: `web/src/components/MatterInformationPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing interaction tests**

Cover: edit filing channel; resolve “final DPCO evidence checklist”; add a missing item; correct core due date; change owner from an eligible candidate; permission-limited viewer sees values and owner but no edit controls; conflict preserves input and offers reload.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx -t 'information|owner|details'`

Expected: FAIL because panels are absent.

- [ ] **Step 3: Implement focused forms**

Use labeled inputs for title, summary, priority, due date and affected area. Render facts as rows with `Edit`, missing items with `Add information`, contradictions with `Resolve`. The missing-information form requires a business label, scalar answer, rationale and optional evidence-reference lines; submit one `RESOLVE_MISSING` command. Candidate assignment uses operation-provided display labels only.

- [ ] **Step 4: Run tests, accessibility and copy gates**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx copyQuality.test.ts Accessibility.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit details and information UX**

```bash
git add web/src/components/MatterDetailsPanel.tsx web/src/components/MatterInformationPanel.tsx web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterRecordWorkspace.test.tsx
git commit -m "feat(web): make Matter details and facts operable"
```

### Task 10: Add complete Action management

**Files:**
- Create: `web/src/components/MatterActionsPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `web/src/components/MatterWorkCommandPanel.tsx`

- [ ] **Step 1: Write failing Action journey tests**

Assert owner/deadline are always visible; authorized owner can add/edit/reassign; current performer can start/block/resume/implement; implementation leaves the outcome check pending; a non-performer sees the assignee and explanation.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx -t Action`

Expected: FAIL.

- [ ] **Step 3: Implement `MatterActionsPanel`**

Render each Action with title, description, owner label, due date and human state. Use operation entries keyed by `subresource_id` for enabled transitions and candidate assignment. Reuse `MatterWorkCommand` only for command execution logic that remains useful; move display and form ownership to the new panel so tasks are not silently absent for other viewers.

- [ ] **Step 4: Pass Action and existing mutation tests**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx OperatingMutations.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit Action UX**

```bash
git add web/src/components/MatterActionsPanel.tsx web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterRecordWorkspace.test.tsx web/src/components/MatterWorkCommandPanel.tsx
git commit -m "feat(web): complete Matter action handling"
```

### Task 11: Add decisions, responses, lifecycle and outcome checks

**Files:**
- Create: `web/src/components/MatterDecisionResponsePanel.tsx`
- Create: `web/src/components/MatterOutcomePanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `web/src/components/MatterWorkCommandPanel.tsx`

- [ ] **Step 1: Write failing cross-role tests**

Cover Draft → initial review, proposal/review/authorization, response preparation where applicable, define outcome check, performer implementation, independent Internal Auditor pass/fail/inconclusive, closure blocked before pass and enabled after pass.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx -t 'lifecycle|decision|response|outcome'`

Expected: FAIL.

- [ ] **Step 3: Implement operation-driven governed forms**

Support `matter.transition` in `MatterWorkCommand`. `MatterDecisionResponsePanel` renders only type-relevant decision/response forms and preserves append-only history. `MatterOutcomePanel` defines contracts using typed fields (expected outcome, linked Action, observation minutes, baseline/threshold key-value rows, failure response, reviewer candidate) and records results with observation/evidence-reference lines and rationale.

- [ ] **Step 4: Run component, copy and accessibility tests**

Run: `cd web && npm test -- MatterRecordWorkspace.test.tsx OperatingMutations.test.tsx copyQuality.test.ts Accessibility.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit governed completion flows**

```bash
git add web/src/components/MatterDecisionResponsePanel.tsx web/src/components/MatterOutcomePanel.tsx web/src/components/MatterRecordWorkspace.tsx web/src/components/MatterRecordWorkspace.test.tsx web/src/components/MatterWorkCommandPanel.tsx
git commit -m "feat(web): complete Matter decision and outcome flows"
```

### Task 12: Add the operational command coverage gate

**Files:**
- Create: `web/src/operationalCoverage.ts`
- Create: `web/src/operationalCoverage.test.ts`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`
- Modify: `docs/engineering/ui-use-case-acceptance-matrix.md`

- [ ] **Step 1: Write a failing coverage test**

The test reads declared Matter material commands and asserts each has a component entry and tested state. Required entries are the 15 commands listed in the design specification.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- operationalCoverage.test.ts`

Expected: FAIL until the manifest is complete.

- [ ] **Step 3: Add the manifest and flow-review integration**

Use a typed constant:

```ts
export const matterOperationalCoverage = {
  "matter.create": { surface: "MattersWorkspace", states: ["list", "empty"] },
  "matter.details.update": { surface: "MatterDetailsPanel", states: ["open"] },
  "matter.context.change": { surface: "MatterInformationPanel", states: ["open"] },
  "matter.assign": { surface: "MatterDetailsPanel", states: ["open"] },
  "matter.transition": { surface: "MatterCurrentHandoff", states: ["open", "ready_to_close"] },
  "matter.link": { surface: "MatterDetailsPanel", states: ["open"] },
  "matter.decision.record": { surface: "MatterDecisionResponsePanel", states: ["decision"] },
  "matter.action.add": { surface: "MatterActionsPanel", states: ["open"] },
  "matter.action.update": { surface: "MatterActionsPanel", states: ["open"] },
  "matter.action.assign": { surface: "MatterActionsPanel", states: ["open"] },
  "matter.action.transition": { surface: "MatterActionsPanel", states: ["planned", "in_progress", "blocked"] },
  "matter.response.add": { surface: "MatterDecisionResponsePanel", states: ["response"] },
  "matter.response.transition": { surface: "MatterDecisionResponsePanel", states: ["response"] },
  "matter.outcome.define": { surface: "MatterOutcomePanel", states: ["open"] },
  "matter.outcome.record": { surface: "MatterOutcomePanel", states: ["ready"] },
} as const;
```

- [ ] **Step 4: Pass the coverage gate**

Run: `cd web && npm test -- operationalCoverage.test.ts && node scripts/review-ui-flow-manifest.mjs`

Expected: PASS.

- [ ] **Step 5: Commit the regression gate**

```bash
git add web/src/operationalCoverage.ts web/src/operationalCoverage.test.ts web/scripts/review-ui-flow-manifest.mjs docs/engineering/ui-use-case-acceptance-matrix.md
git commit -m "test(web): prevent API-only Matter commands"
```

### Task 13: Render and repair the affected Work states

**Files:**
- Modify: `web/scripts/capture-operating-mutations-evidence.mjs`
- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `web/src/operating-mutations.css` or `web/src/matter-record.css`
- Create: rendered files under the existing ignored evidence output directory

- [ ] **Step 1: Extend deterministic fixtures**

Add CRO read-only ownership, Program Owner edit/action, Internal Auditor outcome and mobile conflict fixtures to the existing evidence page/manifest.

- [ ] **Step 2: Run the UI review at required viewports**

Run: `cd web && npm run build && npm run review:ui`

Expected: successful build and rendered review output for 1440px, 1024px/200% replacement and 390px in light/dark presentation.

- [ ] **Step 3: Inspect every affected render and record the highest-impact defect**

Check dominant action prominence, owner visibility, form density, overflow, focus order, status contrast and mobile save/cancel reachability. Record the one highest-impact defect in the review output.

- [ ] **Step 4: Fix and rerun the affected render**

Apply the smallest semantic CSS/component correction and rerun `npm run review:ui`. Expected: the recorded defect is absent in the replacement render.

- [ ] **Step 5: Commit rendered-proof changes**

```bash
git add web/scripts/capture-operating-mutations-evidence.mjs web/scripts/capture-ui-evidence.mjs web/src/matter-record.css web/src/operating-mutations.css
git commit -m "fix(web): finish responsive Matter operations"
```

### Task 14: Synchronize docs and run release verification

**Files:**
- Modify: `README.md`
- Modify: `DESIGN.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/quality/program-matter-acceptance-tests.md`
- Modify: `docs/quality/rendered-ui-evidence.md`

- [ ] **Step 1: Update executable scope and acceptance evidence**

Document the complete Matter workspace, operation read projection, visible responsibility rule, command coverage gate and retained boundary that Programs are completed in the companion plan.

- [ ] **Step 2: Run Go verification**

Run: `go test ./... && go test -tags postgres ./... && go vet ./...`

Expected: PASS with no package failures.

- [ ] **Step 3: Run web verification**

Run: `cd web && npm test && npm run typecheck && npm run build && npm run review:ui`

Expected: PASS with no Vitest, axe, copy, TypeScript, build or rendered-flow failures.

- [ ] **Step 4: Check repository consistency**

Run: `git diff --check && git status --short`

Expected: no whitespace errors; only intended files are modified.

- [ ] **Step 5: Commit Matter completion**

```bash
git add README.md DESIGN.md docs/implementation-plan.md docs/quality/program-matter-acceptance-tests.md docs/quality/rendered-ui-evidence.md
git commit -m "docs: record complete Matter operating workflow"
```
