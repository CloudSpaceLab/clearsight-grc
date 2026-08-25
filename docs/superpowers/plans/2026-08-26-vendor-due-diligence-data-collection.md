# Vendor Due-Diligence and Data-Collection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver an enterprise-ready vendor due-diligence workflow that composes the existing vendor, Matter, form, Evidence Request, invitation, capture and worker foundations into one secure, prefilled collection and review experience.

**Architecture:** Extend the shared form and capture contracts first, then add a narrow third-party assessment composition layer. Assessment start commits its authoritative state, event, outbox and deduplicated setup job atomically; the existing worker provisions one canonical review Matter. A later verified send command passes the recipient directly to the protected Evidence Request boundary, creates or reuses one origin-keyed request and issues the existing request-scoped invitation without putting raw recipient data in third-party state or events.

**Tech Stack:** Go 1.25, PostgreSQL 18/pgx, existing command guard/runtime/outbox/inbox, React 19, TypeScript 7, Vite 8, Vitest/Testing Library, Playwright UI review.

**Source:** [`2026-08-26-vendor-due-diligence-data-collection-design.md`](../specs/2026-08-26-vendor-due-diligence-data-collection-design.md), GitHub issue #80 and the repository customer-facing copy gate.

---

## File map

### Shared form and capture

- Create `internal/formcontract/model.go`, `validation.go`, `validation_test.go`, `visibility.go`, `visibility_test.go`, `scoring.go`, `scoring_test.go`: dependency-neutral form types, normalization, bounded conditional rules, visibility and reusable scoring.
- Modify `internal/monitoring/model.go`, `service.go`, `service_test.go`: make Monitoring Form Templates the authoritative versioned owner while consuming the shared contract.
- Modify `internal/evidence/model.go`, `field_validation.go`, `field_validation_test.go`, `service.go`: request snapshots, typed-answer validation, origin identity and drafts.
- Create `internal/evidence/draft.go`, `draft_test.go`, `draft_postgres.go`, `draft_postgres_integration_test.go`: request-scoped optimistic drafts.
- Create `internal/evidence/invitation_delivery.go`, `invitation_delivery_test.go`: optional protected delivery interface and redacted receipt.
- Modify `internal/evidence/memory.go`, `postgres.go`, `recipient_memory.go`, `recipient_postgres.go`: origin lookup and presentation/draft persistence.
- Modify `internal/httpapi/evidence_handlers.go`, `evidence_handlers_test.go`, `route_registry.go`: external draft read/save routes.
- Modify `web/src/types.ts`, `captureApi.ts`, `captureApi.test.ts`: typed form, presentation and draft contracts.
- Split `web/src/components/CapturePanel.tsx` into focused renderer components under `web/src/components/capture/`.
- Modify `web/src/components/FormBuilder.tsx`, related tests and monitoring CSS: enterprise section/field builder and preview.
- Create migrations `000036_shared_form_capture_contract.up.sql` and `.down.sql`.

### Third-party assessment composition

- Create `internal/thirdparty/assessment_model.go`, `assessment_repository.go`, `assessment_service.go`, `assessment_service_test.go`.
- Create `internal/thirdparty/assessment_memory.go`, `assessment_postgres.go`, `assessment_postgres_integration_test.go`.
- Create `internal/thirdparty/assessment_provisioner.go`, `assessment_provisioner_test.go`.
- Modify `internal/continuity/model.go`, `labels.go` and tests: canonical `VENDOR_REVIEW` Matter type.
- Modify `cmd/worker/services.go`, `services_memory.go`, `services_postgres.go`: register one bounded assessment setup maintainer on the existing runtime.
- Modify `cmd/api/services.go`, `services_memory.go`, `services_postgres.go`, `main.go`: compose assessment dependencies.
- Create `internal/httpapi/third_party_assessment_handlers.go`, `third_party_assessment_handlers_test.go` and register routes.
- Create `web/src/vendorAssessmentTypes.ts`, `vendorAssessmentApi.ts`, `vendorAssessmentApi.test.ts`.
- Create `web/src/components/VendorDueDiligence.tsx`, `VendorDueDiligence.test.tsx` and extend `vendors.css`.
- Create migrations `000037_third_party_due_diligence.up.sql` and `.down.sql`.

### Synchronization and proof

- Modify `api/runtime.openapi.json`, `docs/architecture/durable-schema-ownership.md`, `docs/implementation-plan.md`, `README.md`, issue #80 and rendered UI fixtures.
- Modify `web/src/copyQuality.test.ts` only for reliable new semantic anti-patterns discovered during the full workflow review.

---

### Task 1: Define and validate the shared typed-form contract

**Files:**
- Create: `internal/formcontract/model.go`
- Create: `internal/formcontract/validation.go`
- Create: `internal/formcontract/validation_test.go`
- Modify: `internal/monitoring/model.go`
- Modify: `internal/monitoring/service.go`
- Modify: `internal/evidence/model.go`

- [ ] **Step 1: Write failing contract tests**

Add table-driven tests in package `formcontract` for every explicit type, type-relevant constraint, section reference, field/section cardinality and compatibility alias:

```go
func TestNormalizeRejectsConstraintForWrongType(t *testing.T) {
	_, err := Normalize(Contract{Sections: []Section{{ID: "company", Title: "Company"}}, Fields: []Field{{ID: "website", SectionID: "company", Label: "Website", Type: "url", Constraints: Constraints{Maximum: floatPtr(10)}}}})
	if !errors.Is(err, ErrInvalid) { t.Fatalf("expected invalid contract, got %v", err) }
}

func floatPtr(value float64) *float64 { return &value }

func TestNormalizeKeepsLegacyTypesReadable(t *testing.T) {
	got, err := Normalize(Contract{Sections: []Section{{ID: "review", Title: "Review"}}, Fields: []Field{{ID: "note", SectionID: "review", Label: "Note", Type: "text"}, {ID: "value", SectionID: "review", Label: "Value", Type: "number"}}})
	if err != nil { t.Fatal(err) }
	if got.Fields[0].Type != "short_text" || got.Fields[1].Type != "decimal" { t.Fatalf("unexpected aliases %#v", got.Fields) }
}
```

- [ ] **Step 2: Run the tests and confirm RED**

Run: `go test ./internal/formcontract -run 'TestNormalize' -count=1`

Expected: compilation fails because the package and contract do not exist.

- [ ] **Step 3: Implement the bounded contract**

Define the exact shared types:

```go
type PresentationMode string
const (PresentationClassic PresentationMode = "CLASSIC"; PresentationWizard PresentationMode = "WIZARD"; PresentationAutomatic PresentationMode = "AUTOMATIC")
type Presentation struct { DefaultMode PresentationMode `json:"default_mode"`; AllowModeSwitch bool `json:"allow_mode_switch"` }
type Section struct { ID string `json:"id"`; Title string `json:"title"`; Help string `json:"help,omitempty"`; Condition *VisibilityCondition `json:"condition,omitempty"` }
type Constraints struct { MinLength, MaxLength, MinSelections, MaxSelections, MaxFiles *int; Minimum, Maximum, Step *float64; Currency string; MinDate, MaxDate string; MaxFileBytes *int64 }
type VisibilityCondition struct { FieldID string `json:"field_id"`; Operator ConditionOperator `json:"operator"`; Values []string `json:"values,omitempty"` }
```

Allow only the field types and limits from the approved design. Normalize `text` to `short_text`, `number` to `decimal`, `yes_no` to a two-choice selection, IDs/labels/options to trimmed values and presentation to `AUTOMATIC` when omitted. Reject more than 20 sections, 200 fields, 50 choices, invalid date/numeric bounds and constraints unrelated to the selected type.

- [ ] **Step 4: Use one validator in Monitoring and Evidence**

Have `monitoring.Service.CreateFormTemplate` call `formcontract.Normalize`. Store the normalized exact fields/presentation. Define `monitoring.TemplateField` as an alias of `formcontract.Field`; embed `formcontract.Field` in `evidence.Field` beside request-only bindings/resolutions. This keeps one validation/type contract without creating a Monitoring↔Evidence import cycle.

- [ ] **Step 5: Run package tests and commit**

Run:

```powershell
go test ./internal/formcontract ./internal/monitoring ./internal/evidence -count=1
go test -tags postgres ./internal/formcontract ./internal/monitoring ./internal/evidence -count=1
```

Expected: PASS.

Commit: `feat(forms): add typed shared form contract`

---

### Task 2: Add bounded visibility and authoritative typed-answer validation

**Files:**
- Create: `internal/formcontract/visibility.go`
- Create: `internal/formcontract/visibility_test.go`
- Create: `internal/formcontract/scoring.go`
- Create: `internal/formcontract/scoring_test.go`
- Modify: `internal/evidence/field_validation.go`
- Modify: `internal/evidence/field_validation_test.go`
- Modify: `internal/monitoring/scoring.go`
- Modify: `internal/monitoring/scoring_test.go`

- [ ] **Step 1: Write failing visibility and validation tests**

Cover cycles, dependency on later fields, depth over five, hidden required fields, hidden submitted values, email/telephone/URL formats, integer/decimal/percentage/currency bounds, multi-select cardinality, attestation and file limits:

```go
func TestValidateAnswersRejectsHiddenAnswer(t *testing.T) {
	request := Request{TenantID:"bank", Fields: []Field{
		{Field:formcontract.Field{ID:"handles_data", Label:"Handles customer data", Type:"yes_no", Required:true}},
		{Field:formcontract.Field{ID:"data_location", Label:"Data location", Type:"short_text", Required:true, Condition:&formcontract.VisibilityCondition{FieldID:"handles_data", Operator:formcontract.ConditionEquals, Values:[]string{"Yes"}}},
	}}
	no, location := "No", "Lagos"
	err := service.validateAnswers(ctx, request, map[string]formcontract.AnswerValue{"handles_data":{Text:&no}, "data_location":{Text:&location}})
	if err == nil || !strings.Contains(err.Error(), "not requested for the current answers") { t.Fatalf("unexpected error %v", err) }
}
```

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/monitoring ./internal/evidence -run 'Test.*(Visibility|Hidden|Typed|Constraint)' -count=1`

Expected: failures for missing visibility evaluator and new type validation.

- [ ] **Step 3: Implement deterministic visibility**

Implement `formcontract.VisibleFields(contract formcontract.Contract, answers map[string]formcontract.AnswerValue) ([]formcontract.Field, error)`. Evaluate conditions in declared order after validating an acyclic earlier-field graph. `ANSWERED`, `EQUALS`, `NOT_EQUALS`, `IN` and `NOT_IN` are the only operators. Hidden answers are rejected at draft and submit boundaries.

- [ ] **Step 4: Implement typed answer values and shared scoring input**

Represent single values, arrays and artifact references without comma-delimited encoding:

```go
type AnswerValue struct { Text *string `json:"text,omitempty"`; Values []string `json:"values,omitempty"`; ArtifactIDs []string `json:"artifact_ids,omitempty"`; Document *DocumentAnswer `json:"document,omitempty"` }
```

Define `AnswerValue` in `internal/formcontract`. Update evidence validation and provenance to use `map[string]formcontract.AnswerValue`. Move/adapt the existing pure scoring evaluator into `internal/formcontract` and keep compatibility aliases in Monitoring; Monitoring and later assessment review call the same evaluator.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/formcontract ./internal/monitoring ./internal/evidence -count=1`

Expected: PASS.

Commit: `feat(forms): enforce conditional typed responses`

---

### Task 3: Persist presentation snapshots, origin keys and server drafts

**Files:**
- Create: `migrations/000036_shared_form_capture_contract.up.sql`
- Create: `migrations/000036_shared_form_capture_contract.down.sql`
- Create: `internal/evidence/draft.go`
- Create: `internal/evidence/draft_postgres.go`
- Create: `internal/evidence/draft_postgres_integration_test.go`
- Modify: `internal/evidence/repository.go`
- Modify: `internal/evidence/memory.go`
- Modify: `internal/evidence/postgres.go`
- Modify: `internal/evidence/recipient_postgres.go`
- Modify: `docs/architecture/durable-schema-ownership.md`

- [ ] **Step 1: Write failing PostgreSQL tests**

Prove exact origin reuse, one draft per active session, optimistic conflict, tenant/request/session foreign-key integrity, expiry/revocation denial and deletion/inaccessibility after submission.

```go
func TestSaveDraftRequiresCurrentSessionAndVersion(t *testing.T) {
	value := "security@vendor.example"
	answers := map[string]formcontract.AnswerValue{"security_contact":{Text:&value}}
	first, err := service.SaveDraft(ctx, sessionToken, SaveDraftInput{ExpectedVersion:0, PresentationMode:"WIZARD", Answers:answers})
	if err != nil || first.Version != 1 { t.Fatalf("save %#v %v", first, err) }
	_, err = service.SaveDraft(ctx, sessionToken, SaveDraftInput{ExpectedVersion:0, Answers:answers})
	if !errors.Is(err, ErrVersionConflict) { t.Fatalf("expected conflict, got %v", err) }
}
```

- [ ] **Step 2: Run and confirm RED or SKIP**

Run: `go test -tags "postgres postgresintegration" ./internal/evidence -run 'Test.*(Draft|Origin)' -count=1`

Expected with `TEST_DATABASE_URL`: FAIL before migration/repository implementation. Without it: explicit SKIP.

- [ ] **Step 3: Add migration 000036**

Add `presentation jsonb NOT NULL DEFAULT '{"default_mode":"AUTOMATIC","allow_mode_switch":false}'` to `monitoring_form_templates`; replace the existing 50-field JSON check with the approved 1–200 bound; add immutable `presentation`, `origin_type`, `origin_id`, `origin_version` to `capture_requests`; create a partial unique `(tenant_id,origin_type,origin_id,origin_version)` index when origin is present.

Create `capture_response_drafts` with `(id,tenant_id,request_id,session_id,answers,presentation_mode,version,created_at,updated_at)`, composite tenant/request/session foreign keys, one active row per session and bounded JSON object checks. The down migration reverses only 000036 changes in dependency order.

- [ ] **Step 4: Implement origin lookup and draft repository**

Add:

```go
type DraftStore interface { GetDraft(context.Context, string, string, string) (ResponseDraft, error); SaveDraft(context.Context, SaveDraftRecord) (ResponseDraft, error); DeleteDraft(context.Context, string, string, string) error }
type OriginRequestStore interface { GetRequestByOrigin(context.Context, string, RequestOrigin) (Request, error) }
```

All reads include tenant, request and session scope before returning a row. Save uses `WHERE version=$expected` and returns `ErrVersionConflict` on a stale revision.

- [ ] **Step 5: Classify schema and run tests**

Add `capture_response_drafts` as protected authoritative in-progress response state; update existing form/request ownership rows with presentation and origin semantics.

Run:

```powershell
go test -tags postgres ./internal/evidence ./internal/platform/database -count=1
go test -tags "postgres postgresintegration" ./internal/evidence -count=1
```

Expected: compile/unit PASS; database tests PASS when configured, otherwise SKIP with the missing variable identified.

Commit: `feat(evidence): persist request origins and response drafts`

---

### Task 4: Expose request-scoped autosave and resume APIs

**Files:**
- Modify: `internal/evidence/service.go`
- Create: `internal/evidence/draft_test.go`
- Modify: `internal/httpapi/evidence_handlers.go`
- Modify: `internal/httpapi/evidence_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing service and HTTP tests**

Test `GET /api/v1/evidence/session/draft` and `PUT /api/v1/evidence/session/draft` with bearer-session authentication only. Cover invalid fields, hidden answers, expired/revoked sessions, request version change, conflict and absence without enumeration.

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/evidence ./internal/httpapi -run 'Test.*Draft' -count=1`

Expected: route/service methods are missing.

- [ ] **Step 3: Implement the service boundary**

`SaveDraft` loads the current session and request, verifies the request remains open, validates visible typed answers without requiring every required field, normalizes the permitted presentation mode and writes by expected draft version. `LoadDraft` returns an empty version-zero draft when no row exists.

- [ ] **Step 4: Implement handlers and OpenAPI parity**

The handler derives tenant, request and session exclusively from the bearer capability. It never accepts them from JSON/query values. Map conflicts to 409, closed access to non-enumerating 401, invalid answers to 422 and repository failure to 503 with concise recovery copy.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/evidence ./internal/httpapi -count=1`

Expected: PASS, including runtime route/OpenAPI parity.

Commit: `feat(evidence): add secure response autosave`

---

### Task 5: Build one reusable Classic/Wizard capture renderer

**Files:**
- Create: `web/src/components/capture/CaptureFieldControl.tsx`
- Create: `web/src/components/capture/CaptureForm.tsx`
- Create: `web/src/components/capture/CaptureForm.test.tsx`
- Create: `web/src/components/capture/useCaptureDraft.ts`
- Create: `web/src/components/capture/useCaptureDraft.test.tsx`
- Modify: `web/src/components/CapturePanel.tsx`
- Modify: `web/src/types.ts`
- Modify: `web/src/captureApi.ts`
- Modify: `web/src/captureApi.test.ts`
- Modify: `web/src/capture-inputs.css`

- [ ] **Step 1: Write failing renderer tests**

Cover native input selection, relevant limits, conditional visibility, Classic section index, Wizard progress/back/continue, review, mode switching, 500ms idle autosave, resume and conflict retention.

```tsx
it("uses semantic controls rather than text inputs", () => {
  const contract: CaptureFormContract = {
    presentation: { default_mode: "CLASSIC", allow_mode_switch: true }, sections: [{ id: "company", title: "Company" }],
    fields: [
      { id: "email", section_id: "company", label: "Security contact email", type: "email", required: true },
      { id: "expiry", section_id: "company", label: "Certificate expiry", type: "date", required: true },
      { id: "value", section_id: "company", label: "Annual transaction value", type: "decimal", required: true, constraints: { maximum: 1000000000 } },
    ],
  };
  const draft: CaptureDraft = { answers: {}, presentation_mode: "CLASSIC", version: 0 };
  render(<CaptureForm contract={contract} draft={draft} mode="CLASSIC" onDraft={vi.fn()}/>);
  expect(screen.getByLabelText("Security contact email")).toHaveAttribute("type", "email");
  expect(screen.getByLabelText("Certificate expiry")).toHaveAttribute("type", "date");
  expect(screen.getByRole("spinbutton", { name: "Annual transaction value" })).toHaveAttribute("max", "1000000000");
});
```

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- --run src/components/capture/CaptureForm.test.tsx src/components/capture/useCaptureDraft.test.tsx`

Expected: missing components/hooks.

- [ ] **Step 3: Implement field controls and visibility**

Map each contract type to a native or established component. Never use a generic text input for email, telephone, URL, numeric, date, choice, attestation or upload fields. Apply `min`, `max`, `step`, `minLength`, `maxLength`, `accept`, `multiple`, `inputMode` and accessible descriptions from the contract.

- [ ] **Step 4: Implement Classic, Wizard and Automatic modes**

Classic renders all visible sections with an index. Wizard renders one section, progress and Back/Continue, saving before navigation. Automatic follows the documented threshold. A permitted mode switch changes rendering only and retains the same answers/draft version.

- [ ] **Step 5: Implement autosave and recovery**

`useCaptureDraft` loads the server draft, merges current source-prefilled answers only when no draft answer exists, debounces saves for 500ms, flushes before section changes and preserves local values on 409/503. Status text is exactly `Saving`, `Saved`, `Could not save` or `Access ended` with the appropriate recovery control.

- [ ] **Step 6: Run frontend gates and commit**

Run:

```powershell
npm test
npm run typecheck
npm run build
```

Expected: PASS.

Commit: `feat(web): add typed classic and wizard capture`

---

### Task 6: Upgrade the shared form builder

**Files:**
- Modify: `web/src/components/FormBuilder.tsx`
- Create: `web/src/components/FormBuilder.test.tsx` if no focused test exists; otherwise extend the existing test.
- Modify: `web/src/monitoringTypes.ts`
- Modify: `web/src/monitoringApi.ts`
- Modify: `web/src/monitoringApi.test.ts`
- Modify: `web/src/monitoring.css`

- [ ] **Step 1: Write failing authoring tests**

Prove section add/reorder, field-type selection, relevant constraint editor, conditional dependency limited to earlier fields, Classic/Wizard preview and an exact normalized payload. Prove the old Yes/No shortcut still creates a valid `yes_no` field rather than remaining the only builder capability.

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- --run src/components/FormBuilder.test.tsx src/monitoringApi.test.ts`

Expected: current builder cannot author sections/types/presentation.

- [ ] **Step 3: Implement the section-and-field builder**

Keep one dominant **Save draft** action. Use a field-type select and show only relevant settings. Use direct copy such as `Response type`, `Required response`, `Show this question when`, `Accepted files` and `Response limits`; do not expose JSON, internal enum explanations or product narration.

- [ ] **Step 4: Add preview without a parallel renderer**

Render preview through the same `CaptureForm` component in a non-submitting preview mode. Authors switch Classic/Wizard preview without changing the stored template default.

- [ ] **Step 5: Run frontend gates and commit**

Run: `npm test && npm run typecheck && npm run build`

Expected: PASS.

Commit: `feat(web): add enterprise form authoring`

---

### Task 7: Define the third-party assessment domain

**Files:**
- Create: `internal/thirdparty/assessment_model.go`
- Create: `internal/thirdparty/assessment_repository.go`
- Create: `internal/thirdparty/assessment_service.go`
- Create: `internal/thirdparty/assessment_service_test.go`
- Create: `internal/thirdparty/assessment_memory.go`
- Modify: `internal/continuity/model.go`
- Modify: `internal/continuity/labels.go`
- Modify: related continuity tests

- [ ] **Step 1: Write failing assessment tests**

Cover verified actor/scope, one current onboarding episode, exact template version, state transitions, no client-supplied reviewer, completion conclusions and relationship/version checks.

```go
func TestStartAssessmentReusesCurrentEpisode(t *testing.T) {
	actor := Actor{TenantID:"bank", LegalEntityID:"entity", PrincipalID:"owner"}
	relationshipID := "relationship-1"
	future := time.Now().UTC().Add(14*24*time.Hour)
	first, err := service.StartAssessment(ctx, actor, relationshipID, StartAssessmentInput{RelationshipVersion:1, FormTemplateID:"form-1", FormTemplateVersion:3, ReviewDueAt:future})
	if err != nil { t.Fatal(err) }
	second, err := service.StartAssessment(ctx, actor, relationshipID, StartAssessmentInput{RelationshipVersion:1, FormTemplateID:"form-1", FormTemplateVersion:3, ReviewDueAt:future})
	if err != nil || second.ID != first.ID { t.Fatalf("duplicate episode %#v %v", second, err) }
}
```

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/thirdparty -run 'Test.*Assessment' -count=1`

Expected: assessment types/services missing.

- [ ] **Step 3: Implement model and state machine**

Define statuses and conclusions exactly as the specification. `StartAssessment` accepts only relationship/template versions and review target date; tenant/legal entity/actor come from `Actor`. Add `MatterVendorReview MatterType = "VENDOR_REVIEW"` while retaining `VENDOR_DEFICIENCY` for findings.

- [ ] **Step 4: Implement deterministic memory behavior**

Use a mutex, stable episode index and optimistic versions. Relationship reads remain legal-entity scoped. No assessment command may update the relationship status or vendor organization identity.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/thirdparty ./internal/continuity -count=1`

Expected: PASS.

Commit: `feat(thirdparty): define due diligence assessments`

---

### Task 8: Add assessment schema and atomic start persistence

**Files:**
- Create: `migrations/000037_third_party_due_diligence.up.sql`
- Create: `migrations/000037_third_party_due_diligence.down.sql`
- Create: `internal/thirdparty/assessment_postgres.go`
- Create: `internal/thirdparty/assessment_postgres_integration_test.go`
- Modify: `docs/architecture/durable-schema-ownership.md`

- [ ] **Step 1: Write failing integration tests**

Prove one transaction contains the assessment, `third_party_events`, outbox and setup job; concurrent starts dedupe; stale relationship/version and cross-entity access write nothing; keyset lists filter before limit; point-in-time event versions reconstruct status.

- [ ] **Step 2: Run and confirm RED or SKIP**

Run: `go test -tags "postgres postgresintegration" ./internal/thirdparty -run 'TestPostgres.*Assessment' -count=1`

Expected with database: FAIL before migration. Without database: explicit SKIP.

- [ ] **Step 3: Add migration 000037**

Create:

- `third_party_assessments` with composite tenant/legal-entity/relationship keys, exact template FK, stable episode key, status/conclusion checks and scoped keyset indexes;
- `third_party_assessment_matter_links` with link kind `REVIEW` or `DEFICIENCY` and unique assessment/matter;
- `third_party_assessment_request_links` with purpose `INITIAL` or `CLARIFICATION`, ordered sequence, exact request ID and one current link;
- `third_party_documents` referencing assessment, relationship and existing capture artifact, with validity/evidence-class/status checks;
- `third_party_assessment_jobs` with dedupe key, state, lease fencing, attempts and safe payload containing identifiers only.

Extend `third_party_events.aggregate_type` check for `THIRD_PARTY_ASSESSMENT`. Do not store recipient addresses, answers, artifact contents or reviewer notes in job/outbox payloads.

- [ ] **Step 4: Implement transactional repository methods**

`StartAssessment` locks the scoped relationship/version and exact active template version, inserts/reuses the episode, event, outbox and one READY setup job, then commits. Return the committed assessment. List/get/update queries always include tenant and legal entity before status/cursor/limit.

- [ ] **Step 5: Update schema ownership and run tests**

Run:

```powershell
go test -tags postgres ./internal/thirdparty ./internal/platform/database -count=1
go test -tags "postgres postgresintegration" ./internal/thirdparty -count=1
```

Expected: PASS with configured database; otherwise tagged integration SKIP is reported.

Commit: `feat(thirdparty): persist assessment episodes`

---

### Task 9: Provision one canonical review Matter through the existing worker

**Files:**
- Create: `internal/thirdparty/assessment_provisioner.go`
- Create: `internal/thirdparty/assessment_provisioner_test.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`

- [ ] **Step 1: Write failing retry/dedupe tests**

Use recording fakes for the canonical Matter service. Prove a crash after Matter creation reuses the same `TriggerKey`, links one Matter and completes one job; a Matter-service failure releases/retries the job; terminal failure remains visible without changing assessment status to ready.

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/thirdparty -run 'TestAssessmentProvisioner' -count=1`

Expected: provisioner missing.

- [ ] **Step 3: Implement a bounded maintainer**

`AssessmentProvisioner.Maintain(ctx, now, limit)` claims leased jobs, calls `continuity.Service.CreateMatter` with `Type: VENDOR_REVIEW` and `TriggerKey: "thirdparty-assessment:"+assessment.ID`, links the returned Matter and changes `SETUP_PENDING` to `READY_TO_SEND` in one third-party transaction. It never creates Evidence Requests and never receives recipient data.

- [ ] **Step 4: Register on the existing worker runtime**

Add one named work class, `third-party-assessment-setup`, to existing worker composition for memory and PostgreSQL. Reuse runtime polling, batch, health and shutdown; do not add a process or scheduler.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./cmd/worker ./internal/thirdparty ./internal/continuity -count=1`

Expected: PASS.

Commit: `feat(worker): provision vendor review matters`

---

### Task 10: Create and send the secure vendor request idempotently

**Files:**
- Modify: `internal/thirdparty/assessment_service.go`
- Modify: `internal/thirdparty/assessment_repository.go`
- Modify: `internal/thirdparty/assessment_memory.go`
- Modify: `internal/thirdparty/assessment_postgres.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/repository.go`
- Create: `internal/httpapi/third_party_assessment_handlers.go`
- Create: `internal/httpapi/third_party_assessment_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/server.go`
- Modify: API composition files and `api/runtime.openapi.json`
- Modify: `internal/platform/config/config.go`, `config_test.go` and `.env.example`

- [ ] **Step 1: Write failing send tests**

Cover current owner authority, raw address absence from assessment/event/outbox/job, exact request origin, known facts/prefill, invitation expiry cap, interrupted request/link/invitation recovery, recipient replacement revocation, optional protected delivery and truthful partial outcome.

```go
func TestSendAssessmentRequestDoesNotPersistRawRecipientOutsideEvidence(t *testing.T) {
	repo := &recordingAssessmentRepository{}
	service := newAssessmentServiceForTest(repo)
	result, err := service.SendRequest(ctx, actor, assessmentID, SendRequestInput{ExpectedVersion:2, Audience:"security@vendor.example", Deadline:future})
	if err != nil { t.Fatal(err) }
	if result.State != SendCollecting { t.Fatalf("unexpected result %#v", result) }
	stored, err := json.Marshal([]any{repo.assessments, repo.events, repo.jobs, repo.outbox})
	if err != nil { t.Fatal(err) }
	if bytes.Contains(stored, []byte("security@vendor.example")) { t.Fatalf("recipient leaked into third-party storage: %s", stored) }
}
```

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/thirdparty ./internal/httpapi -run 'Test.*SendAssessment' -count=1`

Expected: command/routes missing.

- [ ] **Step 3: Implement the orchestration command**

The command requires `READY_TO_SEND`, verified actor, current assessment version and Matter link. It calls `evidence.CreateRequest` with subject `VENDOR_RELATIONSHIP`, exact template/version, immutable origin `{THIRD_PARTY_ASSESSMENT, assessmentID, 1}`, request facts and bindings. Replays use `GetRequestByOrigin`. Link the request ID before calling existing `IssueInvitation`.

Return a structured outcome:

```go
type SendRequestOutcome struct { Assessment Assessment `json:"assessment"`; Request evidence.Request `json:"request"`; Invitation *evidence.IssuedInvitation `json:"invitation,omitempty"`; State string `json:"state"`; Recovery string `json:"recovery,omitempty"` }
```

If request/link committed but invitation failed, return `REQUEST_READY_INVITATION_NOT_ISSUED` without claiming email delivery. A retry issues against the same request.

Add a shared `evidence.InvitationDelivery` interface invoked synchronously after invitation creation with the raw address and one-time link. Its receipt stores provider reference/status/time only. Add an optional `CAPTURE_PUBLIC_BASE_URL`; require absolute HTTPS outside local development and construct the link with `url.Values{"capture_invite": []string{issued.Token}}` rather than the request Host header. Default composition has no delivery adapter and returns `LINK_CREATED_EMAIL_NOT_SENT` with the authorized copyable URL; delivery failure returns the same safe link plus recovery. Do not put the address or token in an outbox event or logs.

- [ ] **Step 4: Register verified material routes**

Add:

```go
material("/api/v1/vendors/{id}/assessments", "thirdparty.assessment.start", a.startVendorAssessment, ownerPolicy)
read("/api/v1/vendors/{id}/assessments/current", a.getCurrentVendorAssessment)
material("/api/v1/vendor-assessments/{id}/send-request", "thirdparty.assessment.send_request", a.sendVendorAssessmentRequest, ownerPolicy)
```

Handlers use route IDs and verified identity only. API errors use concise operational copy and never echo addresses or tokens.

- [ ] **Step 5: Run API/security tests and commit**

Run:

```powershell
go test ./internal/thirdparty ./internal/evidence ./internal/httpapi ./cmd/api -count=1
go test -tags postgres ./... -count=1
```

Expected: PASS.

Commit: `feat(thirdparty): send secure due diligence requests`

---

### Task 11: React to submissions and implement authorized review

**Files:**
- Modify: `internal/thirdparty/assessment_service.go`
- Modify: `internal/thirdparty/assessment_postgres.go`
- Create: `internal/thirdparty/assessment_consumer.go`
- Create: `internal/thirdparty/assessment_consumer_test.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `internal/httpapi/third_party_assessment_handlers.go`
- Modify: `internal/httpapi/third_party_assessment_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`

- [ ] **Step 1: Write failing submission/review tests**

Prove a capture-submitted event advances one linked assessment once; external actors cannot read review data; reviewer identity is verified; document validation references an artifact for the exact request; deficiency creation returns one canonical Matter; clarification creates the next request sequence without replacing history; completion requires current review state, no unresolved clarification and an allowed conclusion.

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/thirdparty ./internal/httpapi -run 'Test.*(Submission|Review|Document|Deficiency|Conclusion)' -count=1`

Expected: handlers/consumer missing.

- [ ] **Step 3: Add a submission outbox consumer**

Extend evidence final submission to emit a safe `EvidenceRequestSubmitted` outbox event in its transaction. `AssessmentConsumer.Publish` uses inbox dedupe, resolves request origin and advances the assessment from `COLLECTING` to `SUBMITTED`. No answers enter the event.

- [ ] **Step 4: Implement reviewer commands**

Add material commands for start review, validate/reject a document, request clarification, create deficiency and complete assessment. Clarification creates a new origin-keyed Evidence Request at `sequence=current+1` and retains every prior request link. Use existing `continuity.Service.CreateMatter` with `VENDOR_DEFICIENCY`, assessment/relationship IDs in bounded scope and stable trigger key. Persist only the link in third-party state. Use the shared form evaluator for provisional score; keep score distinct from reviewer conclusion.

- [ ] **Step 5: Register review routes**

```text
GET  /api/v1/vendor-assessments/{id}
POST /api/v1/vendor-assessments/{id}/review/start
POST /api/v1/vendor-assessments/{id}/documents/{artifact_id}/validate
POST /api/v1/vendor-assessments/{id}/clarifications
POST /api/v1/vendor-assessments/{id}/deficiencies
POST /api/v1/vendor-assessments/{id}/complete
```

The scoped GET composes the assessment, ordered request links, exact current submission summary, answer provenance, artifacts, validated documents and Matter links for the internal review UI. It uses bounded exact-ID reads from the existing evidence service and never exposes invitation/session secrets. Route policies use reviewer/owner responsibilities as defined by current authority. Body-supplied reviewer/assessor IDs are ignored.

- [ ] **Step 6: Run tests and commit**

Run: `go test ./internal/thirdparty ./internal/evidence ./internal/continuity ./internal/httpapi ./cmd/worker -count=1`

Expected: PASS.

Commit: `feat(thirdparty): review vendor due diligence`

---

### Task 12: Deliver the Vendors workspace due-diligence experience

**Files:**
- Create: `web/src/vendorAssessmentTypes.ts`
- Create: `web/src/vendorAssessmentApi.ts`
- Create: `web/src/vendorAssessmentApi.test.ts`
- Create: `web/src/components/VendorDueDiligence.tsx`
- Create: `web/src/components/VendorDueDiligence.test.tsx`
- Modify: `web/src/components/VendorsWorkspace.tsx`
- Modify: `web/src/components/VendorsWorkspace.test.tsx`
- Modify: `web/src/vendors.css`
- Modify: `web/src/staticDemo.ts`

- [ ] **Step 1: Write failing workflow tests**

Cover every state and dominant action, start preview, provisioning, send preview, partial invitation recovery, collecting status, submitted review, provenance, document validation, deficiency handoff and conclusion. Assert there is exactly one enabled primary action for the current state.

```tsx
it("offers one next action for a ready assessment", async () => {
  const relationship = { vendor: { id: "vendor-1", legal_name: "Acme Processing Limited" }, relationship: { id: "relationship-1", service_name: "Card processing" } } as VendorRelationshipAggregate;
  const assessment = { id: "assessment-1", relationship_id: "relationship-1", status: "READY_TO_SEND", version: 2 } as VendorAssessment;
  render(<VendorDueDiligence relationship={relationship} assessment={assessment}/>);
  expect(screen.getByRole("button", { name: "Send due diligence request" })).toBeEnabled();
  expect(screen.getAllByRole("button", { name: /Start|Send|Review|Complete/ })).toHaveLength(1);
});
```

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- --run src/components/VendorDueDiligence.test.tsx src/components/VendorsWorkspace.test.tsx`

Expected: component/API missing.

- [ ] **Step 3: Implement typed API and state component**

Keep due diligence within the selected relationship detail. Use state-driven actions: `Start due diligence`, `View setup status`, `Send due diligence request`, `Review request status`, `Review vendor response`, `Record assessment conclusion`, `View completed assessment`. No second vendor dashboard.

- [ ] **Step 4: Implement enterprise-ready content**

Every heading names the task/record; every button names the result; supporting text states status, consequence or recovery once. Remove demo/prototype/AI narration, slogans, vague reassurance, redundant paragraphs and internal implementation terms. External copy never exposes `Matter`, projection, outbox, binding IDs, risk score or review notes.

- [ ] **Step 5: Add deterministic fixtures and responsive layout**

Add explicitly sample-labelled fixtures for ready, collecting, submitted, partial delivery, source degradation and completed states. Desktop retains register/detail context; mobile uses the existing focused record/back behavior. Forms and review content reflow at 390px and 320px with no horizontal overflow.

- [ ] **Step 6: Run frontend gates and commit**

Run:

```powershell
npm test
npm run typecheck
npm run build
```

Expected: PASS.

Commit: `feat(web): add vendor due diligence workflow`

---

### Task 13: Perform full copy, rendered, security and recovery acceptance

**Files:**
- Modify: `web/src/copyQuality.test.ts` only for a reliable regression pattern found during review.
- Modify: UI evidence scripts/manifests under `web/scripts/`.
- Modify: `README.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `api/runtime.openapi.json`
- Modify: issue #80 body/checklist.

- [ ] **Step 1: Audit the complete customer-visible workflow**

Review Vendors, start/send, builder, invitation entry, Classic/Wizard response, autosave, validation, receipt, review, document validation, deficiency, empty/error/conflict/delivery states and API errors. For each string verify it names an object/task, states status/context/consequence/recovery or identifies the next result. Rewrite bloated or narrational copy as a complete flow, not phrase-by-phrase substitution.

- [ ] **Step 2: Run backend verification**

```powershell
go test ./... -count=1
go test -tags postgres ./... -count=1
go test -tags "postgres postgresintegration" ./... -count=1
go vet ./...
```

Expected: PASS. If `TEST_DATABASE_URL` is absent, record the exact skipped database proof and do not claim it ran.

- [ ] **Step 3: Run frontend and copy verification**

```powershell
Set-Location web
npm test
npm run typecheck
npm run build
```

Expected: PASS with all customer-visible strings included in copy-quality coverage.

- [ ] **Step 4: Render and inspect required states**

Run the deterministic static app and `npm run review:ui`. Capture/inspect desktop 1440×900, tablet 1024×768, mobile 390×844, 320×800 and 200% zoom proxy for:

- due-diligence start and ready-to-send;
- Classic and Wizard external forms;
- autosave failure and resume;
- conditional fields and typed validation;
- invitation expired/revoked/delivery failure;
- submitted internal review and document validation;
- source-degraded and optimistic-conflict states.

Fix the highest-impact defect and rerun the affected capture before completion. Confirm no console application errors, overlap, inaccessible control, duplicate primary action or horizontal overflow.

- [ ] **Step 5: Synchronize truth and issue progress**

Document only executable scope: due-diligence collection and review are implemented; activation, continuation, reassessment and exit remain open. Keep `UC-TPRM-01` at Expansion until fail-closed activation acceptance is executable. Update issue #80 without checking incomplete tranche boxes.

- [ ] **Step 6: Final diff and commit**

```powershell
git diff --check
git status --short
git diff --stat
```

Exclude `.codex-tmp`, local Office files, Playwright caches and diagnostic-only screenshots.

Commit: `docs: verify vendor due diligence workflow`

---

## Completion boundary

This plan is complete only when a verified bank owner can start one due-diligence episode, provision one canonical review Matter, issue one secure origin-keyed vendor request, collect a resumable typed response in Classic or Wizard mode, review provenance and documents, create canonical deficiencies and record an authorized assessment conclusion. It must remain impossible to infer that submission, document upload, assessment completion or a satisfactory conclusion activates or approves the vendor relationship.
