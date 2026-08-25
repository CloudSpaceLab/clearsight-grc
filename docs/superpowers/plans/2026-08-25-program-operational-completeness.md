# Program Operational Completeness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make ongoing Programs fully maintainable from the Programs workspace, including ownership, requirements, applicability, safeguards, evidence expectations, assessments, monitoring and linked issues.

**Architecture:** Reuse the actor-scoped operation and participant pattern delivered by the Matter plan. Add explicit versioned Program detail/assignment and requirement-supersession commands, expose all existing continuity Program commands through focused React panels, and extend the material-command coverage gate so Program capabilities cannot remain API-only.

**Tech Stack:** Go 1.25 standard HTTP, PostgreSQL/pgx event projections and Program-status queue, React 19, TypeScript 7, Vite 8, Vitest/Testing Library, existing Monitoring components and semantic CSS tokens.

---

## File map

- Create `internal/continuity/program_edits.go` and `program_edits_test.go`: Program detail, owner and requirement-supersession commands.
- Modify `internal/continuity/replay.go`, `memory.go`, `postgres.go`: new Program event reconstruction/projections.
- Create `internal/continuity/program_edits_postgres_integration_test.go`: transactional state/event/outbox/status-queue proof.
- Create `internal/httpapi/program_operations.go` and `program_operations_test.go`: bounded actor-scoped operations and participant explanations.
- Modify `internal/httpapi/continuity_handlers.go`, `command_lifecycle.go`, `route_registry.go`, related tests and `api/runtime.openapi.json`: commands and reads.
- Create `web/src/programOperationsApi.ts` and tests: typed Program command clients.
- Create `web/src/components/ProgramRecordWorkspace.tsx`: dedicated Program orchestration.
- Create `web/src/components/ProgramCurrentPosition.tsx`: calculated state, freshness, reasons and dominant action.
- Create `web/src/components/ProgramDetailsPanel.tsx`: owner, authority, function, jurisdiction, scope and effective dates.
- Create `web/src/components/ProgramRequirementsPanel.tsx`: requirement and applicability maintenance.
- Create `web/src/components/ProgramSafeguardsPanel.tsx`: objectives, implementations and coverage links.
- Create `web/src/components/ProgramEvidencePanel.tsx`: evidence expectations, assessments and monitoring composition.
- Create `web/src/components/ProgramIssuesPanel.tsx`: open/create exact linked Matter paths.
- Create `web/src/components/ProgramRecordWorkspace.test.tsx`: complete Program owner/reviewer journey.
- Modify `web/src/components/ProgramsWorkspace.tsx`, `ProgramLifecycleControls.tsx`, `MonitoringSetup.tsx`, `AppViews.tsx`, `types.ts`, `continuityCommands.ts`: dedicated routing and shared updates.
- Create `web/src/program-record.css`; modify `web/src/main.tsx`: responsive style registration.
- Modify `web/src/operationalCoverage.ts`, its test and the UI flow manifest: Program material-command coverage.
- Modify root/product/quality docs and rendered evidence scripts.

### Task 1: Add versioned Program details and ownership commands

**Files:**
- Create: `internal/continuity/program_edits.go`
- Create: `internal/continuity/program_edits_test.go`
- Modify: `internal/continuity/replay.go`

- [ ] **Step 1: Write failing domain tests**

```go
func TestUpdateAndAssignProgram(t *testing.T) {
	service := NewService(NewMemoryRepository())
	program := mustCreateProgram(t, service, CreateProgramInput{
		TenantID:"bank", LegalEntityID:"entity", Code:"NDPA", Name:"Data protection",
		Type:"PRIVACY", OwningFunction:"Data Protection Office", OwnerPrincipalID:"owner-1",
		AuthorityPrincipalID:"approver-1", Scope:json.RawMessage(`{"business_lines":["Retail"]}`),
		EffectiveFrom: time.Now().UTC(),
	})
	updated, err := service.UpdateProgramDetails(t.Context(), UpdateProgramDetailsInput{
		TenantID:"bank", ProgramID:program.Program.ID, ExpectedVersion:program.Program.Version,
		Name:"Nigeria data protection", OwningFunction:"Data Protection Office", Jurisdiction:"Nigeria",
		Scope:json.RawMessage(`{"business_lines":["Retail","Corporate"]}`),
		EffectiveFrom:program.Program.EffectiveFrom, ActorID:"owner-1", Rationale:"Confirm the approved operating scope.",
	})
	if err != nil { t.Fatal(err) }
	assigned, err := service.AssignProgram(t.Context(), AssignProgramInput{
		TenantID:"bank", ProgramID:program.Program.ID, ExpectedVersion:updated.Program.Version,
		OwnerPrincipalID:"owner-2", ActorID:"owner-1", Rationale:"Move accountability to the current DPO position.",
	})
	if err != nil { t.Fatal(err) }
	if assigned.Program.OwnerPrincipalID != "owner-2" { t.Fatalf("owner not changed: %#v", assigned.Program) }
}
```

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/continuity -run TestUpdateAndAssignProgram -count=1`

Expected: FAIL because the inputs and methods are undefined.

- [ ] **Step 3: Implement commands and events**

Add `UpdateProgramDetailsInput`, `AssignProgramInput`, `EventProgramDetailsUpdated` and `EventProgramOwnerChanged`. Require actor, rationale, optimistic version, non-empty name/function, valid effective dates and non-empty new owner. Preserve code, type, legal entity, status, creation time and authority route.

- [ ] **Step 4: Add replay cases and pass tests**

Decode both new event payloads as full `Program` values and replace the current Program.

Run: `go test ./internal/continuity -run 'TestUpdateAndAssignProgram|TestProgram' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit Program edits**

```bash
git add internal/continuity/program_edits.go internal/continuity/program_edits_test.go internal/continuity/replay.go
git commit -m "feat(continuity): add versioned Program maintenance"
```

### Task 2: Add requirement supersession without overwriting history

**Files:**
- Modify: `internal/continuity/program_edits.go`
- Modify: `internal/continuity/program_edits_test.go`
- Modify: `internal/continuity/replay.go`

- [ ] **Step 1: Write the failing supersession test**

```go
func TestSupersedeRequirementPreservesPriorVersion(t *testing.T) {
	service, program, prior := programWithRequirementFixture(t)
	updated, err := service.SupersedeRequirement(t.Context(), SupersedeRequirementInput{
		TenantID:"bank", ProgramID:program.Program.ID, RequirementID:prior.ID,
		ExpectedVersion:program.Program.Version, Code:prior.Code, Title:"Submit the annual compliance return",
		Statement:"Submit the current annual return with approved source-linked evidence.",
		SourceID:prior.SourceID, SourceAnchor:"GAID 2025 · annual return · paragraph 4",
		EffectiveFrom:time.Now().UTC(), ActorID:"owner", Rationale:"The approved directive changed the evidence requirement.",
	})
	if err != nil { t.Fatal(err) }
	if len(updated.Requirements) != 2 { t.Fatalf("history lost: %#v", updated.Requirements) }
	if updated.Requirements[0].Status != RequirementSuperseded || updated.Requirements[1].Status != RequirementApproved { t.Fatalf("wrong versions: %#v", updated.Requirements) }
}
```

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/continuity -run TestSupersedeRequirementPreservesPriorVersion -count=1`

Expected: FAIL.

- [ ] **Step 3: Implement one atomic supersession event**

Define `RequirementSupersession{Prior Requirement, Replacement Requirement, Rationale string}` and `EventRequirementSuperseded`. Validate the prior requirement belongs to the Program and is current, the replacement has source anchor/code/title/statement/effective date, and the dates do not overlap incorrectly. Apply one Program event/version containing both values.

- [ ] **Step 4: Replay and pass tests**

Replay must upsert the superseded prior and append/upsert the replacement. Run:

`go test ./internal/continuity -run 'TestSupersedeRequirementPreservesPriorVersion|TestRequirement' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit supersession**

```bash
git add internal/continuity/program_edits.go internal/continuity/program_edits_test.go internal/continuity/replay.go
git commit -m "feat(continuity): preserve Program requirement supersession"
```

### Task 3: Persist Program edits and queue state refresh atomically

**Files:**
- Modify: `internal/continuity/memory.go`
- Modify: `internal/continuity/postgres.go`
- Create: `internal/continuity/program_edits_postgres_integration_test.go`

- [ ] **Step 1: Write the PostgreSQL transaction test**

For each new event, assert the Program/requirement row change, one continuity event, one outbox event and one deduplicated Program-status job are committed together. Force the status-job insert to fail in a transaction fixture and assert no authoritative/event/outbox change committed.

- [ ] **Step 2: Verify failure**

Run: `go test -tags 'postgres postgresintegration' ./internal/continuity -run TestProgramEditsAreAtomic -count=1`

Expected: FAIL because projections reject new events.

- [ ] **Step 3: Extend projections**

Group Program detail/owner events with the current Program update statement. For supersession, update the prior requirement status/effective-until and insert the replacement in the same transaction. Queue Program state from the same transaction before commit; do not call an asynchronous best-effort refresh after the command.

- [ ] **Step 4: Run continuity tests**

Run: `go test ./internal/continuity && go test -tags postgres ./internal/continuity`

Expected: PASS.

- [ ] **Step 5: Commit persistence**

```bash
git add internal/continuity/memory.go internal/continuity/postgres.go internal/continuity/program_edits_postgres_integration_test.go
git commit -m "feat(continuity): persist Program maintenance atomically"
```

### Task 4: Expose Program commands through verified routes

**Files:**
- Modify: `internal/httpapi/continuity_handlers.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: `internal/httpapi/continuity_handlers_test.go`
- Modify: `internal/httpapi/matter_command_authority_test.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing route/identity tests**

Test:

```text
POST /api/v1/programs/{id}/details                         program.details.update
POST /api/v1/programs/{id}/assignment                      program.assign
POST /api/v1/programs/{id}/requirements/{requirement_id}/supersede program.requirement.supersede
```

Assert verified actor binding and that candidate owner is a governed assignment subject, not the command actor.

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/httpapi -run 'TestProgramEditRoutes|TestProgramAssignmentAuthority' -count=1`

Expected: FAIL.

- [ ] **Step 3: Add handlers, routes and lifecycle policy**

Use `ACCOUNTABLE_OWNER` for details/supersession, and require the current owner route for reassignment. Validate replacement owners against current candidate principals and the actor legal entity. Register all as `MATERIAL_COMMAND`.

- [ ] **Step 4: Update the runtime contract and pass tests**

Run: `go test ./internal/httpapi -run 'TestProgram|TestRuntime' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit Program HTTP commands**

```bash
git add internal/httpapi api/runtime.openapi.json
git commit -m "feat(api): expose governed Program maintenance commands"
```

### Task 5: Add bounded actor-scoped Program operations

**Files:**
- Create: `internal/httpapi/program_operations.go`
- Create: `internal/httpapi/program_operations_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing operation tests**

For Program Owner, CRO/authorizer and Internal Auditor/reviewer fixtures, assert current participant labels, `can_act`, candidate assignments, current lifecycle targets and reasons. No-route returns unavailable operations with no invented chain.

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/httpapi -run TestProgramOperations -count=1`

Expected: FAIL.

- [ ] **Step 3: Implement `GET /api/v1/programs/{id}/operations`**

Return the same `MatterOperation` shape renamed to shared `RecordOperation`. Resolve the exact current Program for these responsibilities/commands: owner for details/requirements/safeguards/evidence definition, authorizer for applicability/status, reviewer for assessment/review. Include only current candidate principals and current aggregate version.

- [ ] **Step 4: Add route/contract and pass tests**

Run: `go test ./internal/httpapi -run 'TestProgramOperations|TestProgramReview' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit operations**

```bash
git add internal/httpapi/program_operations.go internal/httpapi/program_operations_test.go internal/httpapi/route_registry.go api/runtime.openapi.json
git commit -m "feat(api): explain current Program actions and owners"
```

### Task 6: Add typed Program web clients

**Files:**
- Create: `web/src/programOperationsApi.ts`
- Create: `web/src/programOperationsApi.test.ts`
- Modify: `web/src/types.ts`
- Modify: `web/src/continuityCommands.ts`

- [ ] **Step 1: Write failing exact-payload tests**

Cover detail update, assignment, supersession and every existing Program command: requirement add, applicability, objective, implementation, control link, evidence expectation, assessment and lifecycle transition. Assert actor/approver/reviewer fields are absent from client input.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- programOperationsApi.test.ts`

Expected: FAIL.

- [ ] **Step 3: Implement typed clients**

Reuse the shared scoped command helper. Export human-facing input types; translate typed form values to the current canonical API JSON only in this module.

- [ ] **Step 4: Pass tests and typecheck**

Run: `cd web && npm test -- programOperationsApi.test.ts && npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit clients**

```bash
git add web/src/programOperationsApi.ts web/src/programOperationsApi.test.ts web/src/types.ts web/src/continuityCommands.ts
git commit -m "feat(web): add typed Program operation clients"
```

### Task 7: Build the dedicated Program record shell

**Files:**
- Create: `web/src/components/ProgramRecordWorkspace.tsx`
- Create: `web/src/components/ProgramCurrentPosition.tsx`
- Create: `web/src/components/ProgramRecordWorkspace.test.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Modify: `web/src/AppViews.tsx`
- Create: `web/src/program-record.css`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write failing shell tests**

Assert target routes show dedicated Program heading, owner, current calculated state/freshness, reasons and one dominant action. List routes retain portfolio search/cards. A stale projection must say it is updating and identify assessed/current versions.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx`

Expected: FAIL.

- [ ] **Step 3: Implement shell and current position**

Load aggregate, review digest and operations together. Preserve current status reasons and `ProgramReviewDigest`. Do not allow a lifecycle form to replace calculated status. Provide exact linked issue navigation.

- [ ] **Step 4: Add responsive styles and pass tests**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx ProgramsProjectionTruth.test.tsx ExactTargetWorkspaces.test.tsx && npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit shell**

```bash
git add web/src/components/ProgramRecordWorkspace.tsx web/src/components/ProgramCurrentPosition.tsx web/src/components/ProgramRecordWorkspace.test.tsx web/src/components/ProgramsWorkspace.tsx web/src/AppViews.tsx web/src/program-record.css web/src/main.tsx
git commit -m "feat(web): add dedicated Program workspace"
```

### Task 8: Add Program details, requirements and applicability

**Files:**
- Create: `web/src/components/ProgramDetailsPanel.tsx`
- Create: `web/src/components/ProgramRequirementsPanel.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing owner/requirement tests**

Cover edit function/jurisdiction/scope, change eligible owner, add source-anchored requirement, supersede current requirement, authorizer applicability decision, and permission-limited read-only views with responsible-person explanations.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx -t 'details|requirement|applicability'`

Expected: FAIL.

- [ ] **Step 3: Implement focused panels**

Use task-specific forms and current candidate labels. Applicability asks “Does this apply?”, scope, rationale and effective date. Supersession shows prior source/version and requires replacement source anchor; it never edits prior text in place.

- [ ] **Step 4: Pass tests, copy and accessibility gates**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx copyQuality.test.ts Accessibility.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit panels**

```bash
git add web/src/components/ProgramDetailsPanel.tsx web/src/components/ProgramRequirementsPanel.tsx web/src/components/ProgramRecordWorkspace.tsx web/src/components/ProgramRecordWorkspace.test.tsx
git commit -m "feat(web): make Program scope and requirements operable"
```

### Task 9: Add safeguards and requirement coverage

**Files:**
- Create: `web/src/components/ProgramSafeguardsPanel.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing safeguard tests**

Cover objective creation, implementation owner/status/scope, requirement-to-implementation linking, visible uncovered requirements and permission-limited state.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx -t safeguard`

Expected: FAIL.

- [ ] **Step 3: Implement safeguard forms**

Show “Safeguard” in business UI while sending canonical control-objective/implementation commands. Present only Program-owned requirements/objectives/implementations in selects. Prevent duplicate links client-side for convenience while leaving server validation authoritative.

- [ ] **Step 4: Pass tests**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx -t safeguard`

Expected: PASS.

- [ ] **Step 5: Commit safeguards**

```bash
git add web/src/components/ProgramSafeguardsPanel.tsx web/src/components/ProgramRecordWorkspace.tsx web/src/components/ProgramRecordWorkspace.test.tsx
git commit -m "feat(web): expose Program safeguard coverage"
```

### Task 10: Add evidence expectations, assessments, monitoring and linked issues

**Files:**
- Create: `web/src/components/ProgramEvidencePanel.tsx`
- Create: `web/src/components/ProgramIssuesPanel.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.tsx`
- Modify: `web/src/components/ProgramRecordWorkspace.test.tsx`
- Modify: `web/src/components/MonitoringSetup.tsx`

- [ ] **Step 1: Write failing evidence/issue journey tests**

Cover evidence expectation definition with acceptable sources/freshness/coverage/independence, reviewer assessment with conclusion/coverage/basis/validity, monitoring setup composition, and exact creation/opening of a linked Matter from a current gap.

- [ ] **Step 2: Verify failure**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx -t 'evidence|assessment|monitoring|issue'`

Expected: FAIL.

- [ ] **Step 3: Implement evidence and issue panels**

Keep monitoring observations separate from evidence assessments. Label assessment status freshness and source. Compose the existing `MonitoringSetup` inside the evidence panel without duplicating its state. `ProgramIssuesPanel` uses `MatterSetupWorkspace` with the Program preselected and opens the created exact Matter route.

- [ ] **Step 4: Pass workflow tests**

Run: `cd web && npm test -- ProgramRecordWorkspace.test.tsx MonitoringSetup.test.tsx MatterSetupWorkspace.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit evidence and issue UX**

```bash
git add web/src/components/ProgramEvidencePanel.tsx web/src/components/ProgramIssuesPanel.tsx web/src/components/ProgramRecordWorkspace.tsx web/src/components/ProgramRecordWorkspace.test.tsx web/src/components/MonitoringSetup.tsx
git commit -m "feat(web): complete Program evidence and issue handling"
```

### Task 11: Extend operational coverage and rendered proof

**Files:**
- Modify: `web/src/operationalCoverage.ts`
- Modify: `web/src/operationalCoverage.test.ts`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`
- Modify: `web/scripts/capture-program-review-evidence.mjs`
- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `web/src/program-record.css`

- [ ] **Step 1: Add failing Program coverage expectations**

Require UI entries for Program create/details/assign/transition, requirement add/supersede, applicability, safeguard objective/implementation/link, evidence definition/assessment, review and supported monitoring operations.

- [ ] **Step 2: Verify and complete the manifest**

Run: `cd web && npm test -- operationalCoverage.test.ts`

Expected before entries: FAIL. Expected after entries: PASS.

- [ ] **Step 3: Render all required Program states**

Run: `cd web && npm run build && npm run review:ui`

Expected: renders for Program Owner, authorizer and reviewer at 1440px, 1024px/200% replacement and 390px in light/dark presentation.

- [ ] **Step 4: Inspect, repair and rerender**

Check owner/authority clarity, status freshness, dense requirement/safeguard rows, focused-form hierarchy, mobile overflow and action reachability. Fix the highest-impact defect and rerun its exact fixture.

- [ ] **Step 5: Commit coverage and visual proof**

```bash
git add web/src/operationalCoverage.ts web/src/operationalCoverage.test.ts web/scripts/review-ui-flow-manifest.mjs web/scripts/capture-program-review-evidence.mjs web/scripts/capture-ui-evidence.mjs web/src/program-record.css
git commit -m "test(web): prove complete Program operations"
```

### Task 12: Synchronize documentation and run full release verification

**Files:**
- Modify: `README.md`
- Modify: `DESIGN.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/quality/program-matter-acceptance-tests.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/engineering/ui-use-case-acceptance-matrix.md`

- [ ] **Step 1: Update executable scope and maturity**

Record complete Program maintenance, actor-scoped operations, requirement supersession, visible responsibility and operational command coverage. Remove any boundary that claims supported Program commands require API work.

- [ ] **Step 2: Run backend verification**

Run: `go test ./... && go test -tags postgres ./... && go vet ./...`

Expected: PASS.

- [ ] **Step 3: Run frontend verification**

Run: `cd web && npm test && npm run typecheck && npm run build && npm run review:ui`

Expected: PASS.

- [ ] **Step 4: Check repository consistency**

Run: `git diff --check && git status --short`

Expected: no whitespace errors and only intended files.

- [ ] **Step 5: Commit Program completion**

```bash
git add README.md DESIGN.md docs/implementation-plan.md docs/quality/program-matter-acceptance-tests.md docs/quality/rendered-ui-evidence.md docs/engineering/ui-use-case-acceptance-matrix.md
git commit -m "docs: record complete Program operating workflow"
```
