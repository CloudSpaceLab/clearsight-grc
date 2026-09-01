# Governed Form Advanced Scoring and Response Policies Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add deterministic advanced form scoring, score-aware completed-response search, and governed policies that idempotently create canonical Matters from qualifying poor responses.

**Architecture:** Extend the immutable form contract with a bounded score profile and calculate a generalized result in the existing response-submission transaction. Add a legal-entity-scoped completed-response read model, then attach typed form-response definitions to generic Automation Policy revisions and execute them from durable scored-response events through the canonical Matter trigger service.

**Tech Stack:** Go 1.24, PostgreSQL 17/pgx, append-only outbox/inbox runtime, React 19, TypeScript, Vite, Vitest, Testing Library, shared ClearSight UI contracts.

---

## File map

- `internal/formcontract` owns score-profile types, validation and deterministic evaluation.
- `internal/evidence` owns immutable response score persistence and completed-response reads.
- `internal/formpolicy` owns typed policy lifecycle, simulation and execution receipts.
- `internal/httpapi` binds verified identity/authority and exposes response/policy routes.
- `cmd/api` and `cmd/worker` wire synchronous policy governance and asynchronous execution.
- `web/src/components/forms` owns score authoring, response portfolio and policy workspaces.
- `migrations/000065_form_scoring_and_response_policies.*.sql` owns the schema change.

## Task 1: Add the bounded score-profile contract

**Files:**
- Modify: `internal/formcontract/model.go`
- Modify: `internal/formcontract/validation.go`
- Create: `internal/formcontract/advanced_scoring.go`
- Create: `internal/formcontract/advanced_scoring_test.go`

- [ ] **Step 1: Write the failing direction and cross-field test**

```go
func TestEvaluateScoreProfileUsesDirectionAndCrossFieldRule(t *testing.T) {
	profile := ScoreProfile{
		Version: "score-profile-v2", Mode: ScoringCompliance,
		Bands: DefaultConcernBands(),
		Contributions: []ScoreContribution{
			{ID: "cert", Weight: 70, Predicate: Predicate{FieldID: "certified", Operator: PredicateEquals, Values: []string{"Yes"}}, MatchPoints: 100, NonMatchPoints: 0, Missing: MissingIndeterminate},
			{ID: "expiry", Weight: 30, Predicate: Predicate{FieldID: "expires_on", Operator: PredicateDateOnOrAfter, Values: []string{"2026-09-01"}}, MatchPoints: 100, NonMatchPoints: 0, Missing: MissingIndeterminate},
		},
		Rules: []ScoreRule{{ID: "expired-cert", Predicate: Predicate{Operator: PredicateAnd, Children: []Predicate{{FieldID: "certified", Operator: PredicateEquals, Values: []string{"Yes"}}, {FieldID: "expires_on", Operator: PredicateDateBefore, Values: []string{"2026-09-01"}}}}, Effect: RuleEffect{Kind: EffectDisqualify}}},
	}
	result, err := EvaluateScoreProfile(profile, scoringContract(), TextAnswers(map[string]string{"certified": "Yes", "expires_on": "2026-08-31"}))
	if err != nil || result.RawScore == nil || result.AdverseScore == nil || result.Band != ConcernCritical {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}
```

Also test multi-select, numeric/date boundaries, hidden fields, `INDETERMINATE`/`EXCLUDE`/`ZERO`, depth 9, 101 rules, floor greater than cap, and exhaustive bands.

- [ ] **Step 2: Run the test and confirm failure**

Run: `go test ./internal/formcontract -run 'TestEvaluateScoreProfile|TestNormalizeScoreProfile' -count=1`

Expected: FAIL because score-profile types and evaluator are undefined.

- [ ] **Step 3: Add typed score-profile contracts**

Add `ScoreDirection`, `MissingScoreBehaviour`, `ConcernBand`, `Predicate`, `ScoreContribution`, `RuleEffect`, `ScoreRule`, `ScoreBandRange`, `ScoreProfile`, `ContributionResult`, `AdvancedRuleResult` and `AdvancedScoreResult`. Add `ScoreProfile *ScoreProfile` to `Contract`. Supported operators are bounded equality/membership, multi-select contains, numeric comparisons/ranges, date comparisons/ranges, answered, `AND`, `OR` and `NOT`. Effects are `CONTRIBUTION`, `FLOOR`, `CAP` and `DISQUALIFY`.

- [ ] **Step 4: Implement normalization and evaluation**

Implement `EvaluateScoreProfile(profile ScoreProfile, contract Contract, answers map[string]AnswerValue) (AdvancedScoreResult, error)`. Enforce 100 contributions, 100 rules, depth 8, 20 children/values, declared-order explanations and 0–100 points. Derive adverse score as raw for risk and `100-raw` for compliance; apply greatest floor, lowest cap, then disqualification.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/formcontract -count=1`

Expected: PASS.

Commit: `git commit -m "feat: add advanced form score profiles"` after staging only Task 1 files.

## Task 2: Persist generalized response scores atomically

**Files:**
- Modify: `internal/evidence/distribution.go`
- Modify: `internal/evidence/response_workspace_scoring.go`
- Modify: `internal/evidence/response_workspace_scoring_test.go`
- Modify: `internal/evidence/response_workspace_postgres_write.go`
- Modify: `internal/evidence/response_workspace_postgres_projection.go`
- Modify: `internal/evidence/distribution_response_revisions_postgres.go`
- Modify: `internal/evidence/response_workspace_memory.go`
- Create: `migrations/000065_form_scoring_and_response_policies.up.sql`
- Create: `migrations/000065_form_scoring_and_response_policies.down.sql`

- [ ] **Step 1: Write failing risk/compliance/unscored response tests**

```go
func TestBuildResponseRevisionStoresGeneralizedRiskScore(t *testing.T) {
	revision, err := buildResponseRevision(scoredWorkspaceRequest(formcontract.ScoringRisk), AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil || revision.Score == nil || revision.Score.Mode != formcontract.ScoringRisk || revision.Score.RawScore == nil || revision.Score.AdverseScore == nil {
		t.Fatalf("revision = %#v, err = %v", revision, err)
	}
}
```

- [ ] **Step 2: Run and confirm failure**

Run: `go test ./internal/evidence -run 'TestBuildResponseRevisionStoresGeneralized|TestResponseWorkspace.*Score' -count=1`

Expected: FAIL because `ResponseRevision.Score` does not exist.

- [ ] **Step 3: Add `ResponseScoreResult` and calculate every eligible mode**

Store mode, direction, raw/adverse score, band, coverage, final/state, profile version/checksum, evaluator version/time, contribution/rule results and bounded failure code. Preserve existing compliance fields as read compatibility; do not fabricate generalized meaning for legacy rows.

```go
type ResponseScoreState string
const (
	ResponseScoreNotConfigured ResponseScoreState = "NOT_CONFIGURED"
	ResponseScoreFinal ResponseScoreState = "FINAL"
	ResponseScoreProvisional ResponseScoreState = "PROVISIONAL"
	ResponseScoreFailed ResponseScoreState = "FAILED"
)
type ResponseScoreResult struct {
	Mode formcontract.ScoringMode `json:"mode"`
	Direction formcontract.ScoreDirection `json:"direction"`
	RawScore, AdverseScore *float64
	Band formcontract.ConcernBand `json:"band"`
	Coverage float64 `json:"coverage"`
	Final bool `json:"final"`
	State ResponseScoreState `json:"state"`
	ProfileVersion, ProfileChecksum, EvaluatorVersion, FailureCode string
	CalculatedAt time.Time `json:"calculated_at"`
	ContributionResults []formcontract.ContributionResult `json:"contribution_results"`
	RuleResults []formcontract.AdvancedRuleResult `json:"rule_results"`
}
```

- [ ] **Step 4: Add response columns and indexes in migration 65**

Add `score_mode`, `score_direction`, `raw_score`, `adverse_score`, `concern_band`, `score_state`, bounded `score_result` JSON, profile checksum and calculation time. Add current-response partial indexes for legal entity plus adverse/raw score. The down migration must guard against policy/execution data before removing dependencies.

- [ ] **Step 5: Make score/event/outbox one transaction**

Extend the existing workspace submit transaction to store the response revision and emit `FORM_RESPONSE_SCORED` with response revision ID, form revision and score state only. Inject an outbox failure in the PostgreSQL test and assert submission, response revision, event and outbox all remain absent.

- [ ] **Step 6: Run and commit**

Run: `go test ./internal/evidence ./internal/formcontract -count=1` and `go test -tags postgres ./internal/evidence -count=1`.

Expected: PASS; tagged integration skips only when `TEST_DATABASE_URL` is absent.

Commit: `git commit -m "feat: persist generalized response scores"`.

## Task 3: Add the completed-response query and API

**Files:**
- Create: `internal/evidence/completed_response.go`
- Create: `internal/evidence/completed_response_memory.go`
- Create: `internal/evidence/completed_response_postgres.go`
- Create: `internal/evidence/completed_response_test.go`
- Create: `internal/evidence/completed_response_postgres_integration_test.go`
- Create: `internal/httpapi/form_responses.go`
- Create: `internal/httpapi/form_responses_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `api/runtime.openapi.json`
- Modify: `migrations/000065_form_scoring_and_response_policies.up.sql`

- [ ] **Step 1: Write failing isolation/filter/cursor tests**

Create rows with equal scores across two legal entities. Query current High/Critical responses sorted by concern with limit 2. Assert entity-b never appears, score/time/ID order is stable and page two has no duplicate.

- [ ] **Step 2: Define the safe query model**

```go
type ResponseSort string
const (
	ResponseSortConcern ResponseSort = "CONCERN_DESC"
	ResponseSortNewest ResponseSort = "COMPLETED_DESC"
	ResponseSortRawAsc ResponseSort = "RAW_ASC"
	ResponseSortRawDesc ResponseSort = "RAW_DESC"
)
type CompletedResponseQuery struct {
	TenantID, LegalEntityID, FormTemplateID, SubjectType, SubjectID string
	FormTemplateVersion int64
	Modes []formcontract.ScoringMode
	Bands []formcontract.ConcernBand
	States []ResponseScoreState
	RawMinimum, RawMaximum, AdverseMinimum, AdverseMaximum *float64
	CompletedFrom, CompletedUntil *time.Time
	CurrentOnly bool
	Sort ResponseSort
	Cursor string
	Limit int
}
```

Validate enums/ranges, limit 1–100 and a cursor containing selected sort values plus response ID.

- [ ] **Step 3: Implement memory and PostgreSQL readers**

Join response revisions, distributions and exact form revisions. Apply tenant/legal-entity/restricted-subject scope before ordering and pagination. SQL performs sorting; the browser never receives a broad population for client sorting.

- [ ] **Step 4: Add verified-scope routes**

Register `GET /api/v1/forms/responses` and `GET /api/v1/forms/responses/{revision_id}`. Bind tenant/legal entity from verified identity, ignore scope query fields, return 422 for invalid filters, and exclude recipient addresses/routes.

- [ ] **Step 5: Add 10,000-row query-plan proof**

The tagged test seeds two entities, asserts the score partial index in `EXPLAIN (FORMAT JSON)` and applies a 500 ms warm-query budget.

- [ ] **Step 6: Run and commit**

Run: `go test ./internal/evidence ./internal/httpapi -count=1` and `go test -tags 'postgres postgresintegration' ./internal/evidence -run CompletedResponse -count=1`.

Commit: `git commit -m "feat: query completed form responses by score"`.

## Task 4: Replace the distribution-first Responses UI

**Files:**
- Modify: `web/src/formsDistributionApi.ts`
- Rewrite: `web/src/components/forms/ResponsesView.tsx`
- Create: `web/src/components/forms/ResponsesView.test.tsx`
- Create: `web/src/components/forms/responses-view.css`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`

- [ ] **Step 1: Write failing UI tests**

```tsx
it("requests completed responses by concern", async () => {
  render(<ResponsesView/>);
  expect(await screen.findByText("Vendor certification refresh")).toBeTruthy();
  expect(requests[0]).toContain("sort=CONCERN_DESC");
  expect(screen.getByText("42% compliance · Below required level")).toBeTruthy();
});
```

Also assert server filters, score-unavailable recovery and one mobile `Review response` action.

- [ ] **Step 2: Add typed API models and normalization**

Add `CompletedResponseQuery`, `CompletedResponseSummary`, `CompletedResponsePage` and `loadCompletedResponses`. Retain `loadResponseRevisions` for per-distribution audit history.

- [ ] **Step 3: Build the responsive portfolio**

Use shared `FilterBar`, `SelectField`, date `TextField`, `FilterChip`, `DataTable`, `StatusBadge`, `EmptyState`, `Notice`, `Button` and `FocusedSheet`. Default to `Needs attention first`. Show form, subject, completion, score mode/meaning, concern, coverage and one review action.

- [ ] **Step 4: Verify accessibility and commit**

Run `npm test -- ResponsesView Task11FormsViews`, `npm run typecheck` and `npm run check:ui-contracts` from `web`.

Expected: PASS with no raw select/button contract violations.

Commit: `git commit -m "feat: add score-aware response portfolio"`.

## Task 5: Add typed policy lifecycle and simulation

**Files:**
- Create: `internal/formpolicy/model.go`
- Create: `internal/formpolicy/validation.go`
- Create: `internal/formpolicy/repository.go`
- Create: `internal/formpolicy/memory.go`
- Create: `internal/formpolicy/simulation.go`
- Create: `internal/formpolicy/service.go`
- Create: `internal/formpolicy/service_test.go`

- [ ] **Step 1: Write failing lifecycle/simulation tests**

Test maker-checker separation, version conflict, active-form validation, preview age/checksum/high-water/count invalidation, score-direction predicates, shadow/enforce mode, dates and blast radius.

```go
func TestActivateRequiresFreshPreviewAndDistinctChecker(t *testing.T) {
	service := newPolicyTestService()
	draft := createPolicyFixture(t, service, "maker")
	preview := simulatePolicyFixture(t, service, draft, 3, 2)
	approved := approvePolicyFixture(t, service, draft, "checker", preview.ID)
	_, err := service.Activate(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}, approved.ID, approved.RecordVersion)
	if !errors.Is(err, ErrMakerChecker) { t.Fatalf("err = %v", err) }
}
```

- [ ] **Step 2: Define and validate policy records**

Define `Policy`, `Eligibility`, `MatterAction`, `BlastRadius`, `OutcomeContract`, `SimulationReceipt` and `ExecutionReceipt`. Reference generic Automation Policy ID/version but do not reuse AI policy definitions. Permit only approved title variables and action class `FORM_RESPONSE_CREATE_MATTER`.

```go
type Eligibility struct { FormTemplateID string; FormTemplateVersion int64; SubjectTypes []string; CurrentOnly bool; MinimumCoverage float64; Bands []formcontract.ConcernBand; RawBelow, RawAbove, AdverseAtLeast *float64 }
type MatterAction struct { Type string; Priority int; TitleTemplate, SummaryTemplate, RequestedHandling string }
type BlastRadius struct { PerRun, PerDay int }
type Policy struct { ID, TenantID, LegalEntityID, Code, Name, Purpose string; AutomationPolicyID string; AutomationPolicyVersion int64; Eligibility Eligibility; Action MatterAction; BlastRadius BlastRadius; Rollout RolloutMode; Status PolicyStatus; MakerID, CheckerID, Checksum string; EffectiveFrom, EffectiveUntil *time.Time; Version, RecordVersion int64 }
```

- [ ] **Step 3: Implement lifecycle and simulation**

Implement Create/Get/List/Simulate/Submit/Approve/Activate/Suspend/Rollback. Bind actors from verified service input. Require a matching preview less than 24 hours old and rerun impact counts before activation.

- [ ] **Step 4: Run and commit**

Run: `go test ./internal/formpolicy -count=1`.

Commit: `git commit -m "feat: govern form response policies"`.

## Task 6: Persist policies, previews and receipts

**Files:**
- Create: `internal/formpolicy/postgres.go`
- Create: `internal/formpolicy/postgres_integration_test.go`
- Modify: `migrations/000065_form_scoring_and_response_policies.up.sql`
- Modify: `migrations/000065_form_scoring_and_response_policies.down.sql`
- Modify: `internal/platform/database/schema_ownership_test.go`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: `docs/architecture/durable-schema-ownership.d/forms-distribution.md`

- [ ] **Step 1: Add schema tables**

Create typed definition, simulation, execution and adverse-episode tables. Definitions reference `(automation_policy_id, tenant_id)` and the exact entity/form. Executions are unique by tenant/policy revision/response revision. A partial unique index permits one open episode per policy code and subject.

```sql
CREATE TABLE form_response_policy_executions (
  id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL,
  automation_policy_id uuid NOT NULL, automation_policy_version bigint NOT NULL,
  response_revision_id uuid NOT NULL, state text NOT NULL,
  matter_id uuid, reason_code text NOT NULL DEFAULT '', created_at timestamptz NOT NULL,
  UNIQUE (tenant_id,automation_policy_id,automation_policy_version,response_revision_id),
  FOREIGN KEY (automation_policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id)
);
```

- [ ] **Step 2: Write failing PostgreSQL tests**

Assert cross-tenant references fail, concurrent updates conflict, malformed stored JSON fails closed, duplicate execution returns the stored receipt and open episodes remain unique.

- [ ] **Step 3: Implement repository transactions**

Use explicit column lists and `FOR UPDATE` for lifecycle changes. Store generic Automation Policy revision, typed definition, append-only event and outbox in one transaction.

- [ ] **Step 4: Update schema ownership, run and commit**

Run `go test ./internal/formpolicy ./internal/platform/database -count=1` and `go test -tags 'postgres postgresintegration' ./internal/formpolicy -count=1`.

Commit: `git commit -m "feat: persist governed response policies"`.

## Task 7: Expose authority-bound policy APIs

**Files:**
- Create: `internal/httpapi/form_policy_handlers.go`
- Create: `internal/httpapi/form_policy_routes.go`
- Create: `internal/httpapi/form_policy_handlers_test.go`
- Create: `internal/httpapi/form_score_preview.go`
- Create: `internal/httpapi/form_score_preview_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/command_lifecycle.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/main.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write route/actor tests**

Assert reads/writes/material commands are correctly classified; body tenant/entity/maker/checker/actor IDs are ignored; maker-checker and revoked authority fail closed.

- [ ] **Step 2: Wire `FormPolicies *formpolicy.Service`**

Construct memory/PostgreSQL repositories with completed-response reader, form reader and current authority resolver. Add the dependency to API construction.

- [ ] **Step 3: Implement approved routes**

Add list/create/get/simulate/submit/approve/activate/suspend/rollback. Use object type `FORM_RESPONSE_POLICY`, current owner/reviewer/authorizer routes, `noActorField` and verified legal-entity binding.

Also register the authenticated-write score preview against an exact form revision:

```go
write(http.MethodPost, "/api/v1/config/form-templates/{id}/score-preview", a.previewFormScore, nil)
```

`previewFormScore` must require a positive `form_template_version` in the body, load that verified tenant/legal-entity form revision, accept only bounded answer values, call the same `formcontract.EvaluateScoreProfile` used at completion, and store no response or score record.

- [ ] **Step 4: Verify error mapping, run and commit**

Map invalid preview/predicate to 422, conflict/maker-checker to 409, missing scope to 404, authority unavailable to 503 and unexpected storage failure to non-revealing 500.

Run: `go test ./internal/httpapi ./cmd/api -count=1`.

Commit: `git commit -m "feat: expose governed form policy commands"`.

## Task 8: Execute policies and deduplicate Matters

**Files:**
- Create: `internal/formpolicy/executor.go`
- Create: `internal/formpolicy/executor_test.go`
- Create: `internal/formpolicy/executor_postgres_test.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `internal/continuity/service.go` only if the transaction coordinator requires a narrow adapter

- [ ] **Step 1: Write failing execution tests**

```go
func TestExecutorReusesOpenAdverseEpisodeMatter(t *testing.T) {
	executor := newExecutorFixture(t, Enforce)
	first, err := executor.Handle(context.Background(), scoredEvent("response-1", 82))
	if err != nil || first.MatterID == "" || first.State != ExecutionApplied { t.Fatalf("first = %#v, err = %v", first, err) }
	second, err := executor.Handle(context.Background(), scoredEvent("response-2", 88))
	if err != nil || second.MatterID != first.MatterID || second.CreatedMatter { t.Fatalf("second = %#v, err = %v", second, err) }
}
```

Also test non-match, shadow, expired, blast suppression, route failure, replay, rollback and a new episode after verified closure.

- [ ] **Step 2: Implement effective-policy selection and receipts**

Evaluate only Active policies effective at response completion. Revalidate scope, current-response requirement, subject resolver and verified automation service principal. Store every decision state without answers or secure-route data.

- [ ] **Step 3: Apply Matter trigger in one material transaction**

Use `form-response-policy:{code}:{subject_type}:{subject_id}:{episode_id}`. Append receipt, open/reuse episode, apply/update canonical Matter, append events and enqueue outbox/maintenance together. Duplicate inbox delivery returns the existing receipt/Matter.

Rollback or compensation must never delete or silently close an applied Matter. Record the superseding policy revision and route an inappropriate policy-created Matter for authorized review with the execution receipt as its basis.

- [ ] **Step 4: Register worker consumer/reconciliation**

Consume `FORM_RESPONSE_SCORED` and retry pending/retryable failures in leased batches of 100. Configure event-driven execution plus 30-second reconciliation; log IDs and states only.

- [ ] **Step 5: Run and commit**

Run `go test ./internal/formpolicy ./internal/continuity ./cmd/worker -count=1` and `go test -tags postgres ./internal/formpolicy ./cmd/worker -count=1`.

Commit: `git commit -m "feat: create matters from poor form responses"`.

## Task 9: Add scoring authoring and policy workspaces

**Files:**
- Modify: `web/src/components/forms/formAuthoring.ts`
- Modify: `web/src/components/forms/formContractQuality.ts`
- Modify: `web/src/components/forms/FormFieldPropertyEditor.tsx`
- Modify: `web/src/components/forms/builder/FormInspector.tsx`
- Create: `web/src/components/forms/builder/AdvancedScoringEditor.tsx`
- Create: `web/src/components/forms/builder/AdvancedScoringEditor.test.tsx`
- Create: `web/src/formPoliciesApi.ts`
- Create: `web/src/components/forms/FormPoliciesView.tsx`
- Create: `web/src/components/forms/FormPoliciesView.test.tsx`
- Create: `web/src/components/forms/FormPolicyEditor.tsx`
- Create: `web/src/components/forms/FormPolicyEditor.test.tsx`
- Create: `web/src/components/forms/form-policies.css`
- Modify: `web/src/components/forms/FormsNavigation.tsx`
- Modify: `web/src/components/forms/FormsTabContent.tsx`

- [ ] **Step 1: Write failing builder/policy tests**

Assert bounded controls rather than JSON, server preview raw/adverse/coverage explanation, invalid weights block review, simulation precedes submit, maker cannot approve, shadow/enforced states are visible and linked Matters use canonical navigation.

- [ ] **Step 2: Round-trip profile and consume server preview**

Extend form authoring/revision inputs with `score_profile`. Client checks mirror limits but never calculate the authoritative completed score. Call `POST /api/v1/config/form-templates/{id}/score-preview` with the exact stored revision and render its result.

- [ ] **Step 3: Build `AdvancedScoringEditor`**

Use shared fields/buttons/cards/notices/sheet. Provide contribution rows, bounded AND/OR groups, effects and band ranges with one `Preview score` action. Never expose expression JSON or scripts.

- [ ] **Step 4: Build Policies tab and editor**

Add lazy-loaded `Policies` navigation. Show one dominant action for state: Simulate, Submit, Approve, Activate shadow, Enforce, Suspend or Roll back. Display population, excluded/restricted counts, deduplicated subjects, blast limits, preview age and receipts.

- [ ] **Step 5: Verify UI contracts and commit**

Run `npm test -- AdvancedScoringEditor FormPoliciesView FormPolicyEditor FormFieldPropertyEditor`, `npm run typecheck` and `npm run check:ui-contracts` from `web`.

Commit: `git commit -m "feat: add scoring and response policy workspaces"`.

## Task 10: Seed, prove and document the release

**Files:**
- Modify: `migrations/000065_form_scoring_and_response_policies.up.sql`
- Modify: `docs/product/governed-forms.md`
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `DESIGN.md`
- Modify: `web/ui-contract-migrations.json`
- Modify: `docs/design/ui-component-adoption.md`
- Create: `docs/evidence/form-scoring-policy-release.md`

- [ ] **Step 1: Seed the reference journey through ordinary tables**

Add clearly labelled sample vendor certification form/response/policy/Matter data through the same lifecycle and repositories as customer data. Add no static API branch or browser metric.

- [ ] **Step 2: Run complete automated verification**

Run `go test ./... -count=1`, `go test -tags postgres ./... -count=1`, `go vet ./...`, then from `web` run `npm test`, `npm run typecheck`, `npm run check:ui-contracts` and `npm run build`.

Expected: PASS. Record a missing `TEST_DATABASE_URL` as an integration skip, never as passed database execution.

- [ ] **Step 3: Run configured PostgreSQL/load proof**

Run `go test -tags 'postgres postgresintegration' ./internal/formcontract ./internal/evidence ./internal/formpolicy ./internal/httpapi -count=1`.

Expected: PASS with transaction atomicity, 10,000-response query plan, isolation and Matter dedupe.

- [ ] **Step 4: Render and inspect affected states**

Inspect Builder scoring, preview, Responses filters/detail, policy simulation/approval, shadow receipt and enforced Matter linkage at 320, 390, 768, 1280 and 1440 px in both themes. Confirm no dropdown shift, sidebar overlap or document overflow.

- [ ] **Step 5: Run the full acceptance journey**

Complete poor vendor certification response → score filter → policy receipt → one linked Matter → replay → second response without duplication → authorized remediation/outcome verification/closure → later poor response creates a new episode.

- [ ] **Step 6: Update docs and final verification**

Document score meaning, lifecycle, limits, freshness, schema ownership, UI adoption, evidence and deployed commit. Run `git diff --check` and inspect `git status --short`. Confirm no credentials, PEM paths, protected addresses, route tokens, static API data or unrelated files are staged.

- [ ] **Step 7: Commit release proof**

Commit: `git commit -m "test: prove governed form scoring policies"`.

## Execution checkpoints

- After Task 2, existing submission works with generalized scoring but no automatic Matter execution.
- After Task 4, users can query real responses by concern with no policy dependency.
- After Task 7, policy governance/simulation is available while Enforce remains unavailable until Task 8 is deployed.
- After Task 8, seeded environments remain in shadow mode until Task 10 proof passes.
- Never merge/deploy a partially enabled control, an unreviewed migration/down migration or required database proof that only skipped.
