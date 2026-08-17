# Monitoring Setup and Risk Scoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an authorized bank user create a Program, attach governed connected data or a reusable form, evaluate collected values with deterministic risk rules, and review the result entirely through the UI.

**Architecture:** Preserve Programs, Evidence Requests and Source Access as the authoritative domains. Add a focused `monitoring` domain for versioned form templates, Monitoring Checks and immutable results; reference exact source Binding and form versions instead of copying connector configuration. Build non-modal React setup workspaces over verified-identity command APIs, and keep risk observations separate from approved evidence assessments.

**Tech Stack:** Go 1.24+, PostgreSQL 18, `net/http`, React 19, TypeScript 7, Vite 8, Vitest 4, Testing Library, CSS design tokens.

---

## File map

### New backend files

- `internal/monitoring/model.go` — lifecycle, form, check, rule and result contracts.
- `internal/monitoring/scoring.go` — deterministic form and source evaluation.
- `internal/monitoring/service.go` — validation, authority-neutral domain orchestration and collection creation.
- `internal/monitoring/repository.go` — repository interface.
- `internal/monitoring/memory.go` — unit-test repository.
- `internal/monitoring/postgres.go` — tenant-scoped PostgreSQL persistence.
- `internal/monitoring/*_test.go` — lifecycle, scoring, concurrency and provenance tests.
- `internal/httpapi/monitoring_handlers.go` — form/check/result commands and reads.
- `migrations/000034_monitoring_setup.up.sql` and `.down.sql` — authoritative schema and request references.

### New frontend files

- `web/src/monitoringTypes.ts` — API response and command types.
- `web/src/monitoringApi.ts` — verified-context Program, form, source and monitoring commands.
- `web/src/components/ProgramSetupWorkspace.tsx` — resumable non-modal Program setup.
- `web/src/components/FormBuilder.tsx` — reusable form configuration and preview.
- `web/src/components/DataSourcesWorkspace.tsx` — source list, connection setup, schema selection and approval.
- `web/src/components/MonitoringChecks.tsx` — Program checks and latest results.
- `web/src/monitoring.css` — responsive layout using existing tokens.
- focused Vitest files next to each component.

### Existing files to modify

- `internal/evidence/model.go`, `service.go`, `postgres.go` — exact form-template and collection-period references.
- `internal/sourceaccess/catalog_service.go`, `catalog_types.go`, repository implementations — source revision lifecycle.
- `internal/httpapi/route_registry.go`, `server.go` — monitoring and source lifecycle routes.
- `cmd/api/services.go`, `services_memory.go`, `services_postgres.go`, `main.go` — service wiring.
- `cmd/worker/services_postgres.go` — submission-result evaluation worker registration.
- `web/src/App.tsx`, `AppViews.tsx`, `appRouting.ts`, `types.ts` — setup routes and Configure tabs.
- `web/src/components/ProgramsWorkspace.tsx` — create action and Monitoring section.
- fixture and context files containing prior sample-bank names.
- `AGENTS.md`, `DESIGN.md`, `api/runtime.openapi.json`, architecture/product docs and acceptance ledger.

## Task 1: Rename customer-visible sample organizations

**Files:**
- Modify: `deploy/scripts/seed-demo-foundation.sh`
- Modify: `internal/httpapi/handlers.go`
- Modify: `internal/httpapi/actor_read_handlers.go`
- Modify: `internal/bankverticals/model.go`
- Modify: `internal/bankverticals/seed.go`
- Modify: `internal/continuity/demo.go`
- Modify: affected backend and frontend tests
- Test: `web/src/copyQuality.test.ts`

- [ ] **Step 1: Add failing naming assertions**

Add tests asserting visible context uses `Clear Bank` and `Clear Bank Nigeria`, while durable IDs remain unchanged. Extend the copy regression with:

```ts
/ClearSight Demonstration Bank/i,
/Demonstration Bank Nigeria/i,
/\bDemo Bank(?: Nigeria)?\b/i,
```

- [ ] **Step 2: Run the focused tests and confirm failure**

Run:

```powershell
go test ./internal/httpapi ./internal/bankverticals ./internal/continuity
npm test -- --run src/copyQuality.test.ts src/App.test.tsx src/components/DemoAuthGate.test.tsx
```

Expected: failures name the prior organization strings.

- [ ] **Step 3: Replace visible fixture names safely**

Use `Clear Bank` and `Clear Bank Nigeria` in runtime context and sample domain fixtures. Keep `bank-demo`, `clearsight-demo` and UUIDs unchanged. In the deployment seed, update only rows that still have an owned prior name:

```sql
UPDATE tenants
SET name = 'Clear Bank'
WHERE id = '00000000-0000-4000-8000-000000000001'
  AND name IN ('ClearSight Demonstration Bank', 'Demo Bank', 'Clear Bank');

UPDATE legal_entities
SET name = 'Clear Bank Nigeria'
WHERE id = '00000000-0000-4000-8000-000000000002'
  AND name IN ('Demonstration Bank Nigeria', 'Demo Bank Nigeria', 'Clear Bank Nigeria');
```

This changes ClearSight-owned sample values and preserves any unrelated customer value.

- [ ] **Step 4: Run naming and deployment tests**

```powershell
go test ./internal/httpapi ./internal/bankverticals ./internal/continuity
npm test -- --run src/copyQuality.test.ts src/App.test.tsx src/components/DemoAuthGate.test.tsx
python -m unittest deploy.tests.deployment_config_test
```

- [ ] **Step 5: Commit**

```powershell
git add deploy internal web/src docs
git commit -m "fix(copy): use Clear Bank for sample organization"
```

## Task 2: Add deterministic monitoring contracts and scoring

**Files:**
- Create: `internal/monitoring/model.go`
- Create: `internal/monitoring/scoring.go`
- Create: `internal/monitoring/scoring_test.go`

- [ ] **Step 1: Write failing score tests**

Cover weighted results, threshold boundaries, critical override, missing required answers, invalid answer, stable ordering and source indeterminate states. The core contract is:

```go
type Evaluation struct {
    Score            *float64    `json:"score,omitempty"`
    Band             RiskBand    `json:"band"`
    Coverage         float64     `json:"coverage"`
    CriticalFailures []RuleResult `json:"critical_failures"`
    RuleResults      []RuleResult `json:"rule_results"`
}

func EvaluateForm(fields []FormField, answers map[string]string, thresholds Thresholds) (Evaluation, error)
func EvaluateSource(rules []SourceRule, resolution evidence.SourceResolution, thresholds Thresholds, now time.Time) (Evaluation, error)
```

- [ ] **Step 2: Verify tests fail because the package is absent**

```powershell
go test ./internal/monitoring
```

- [ ] **Step 3: Implement normalized validation and pure evaluation**

Implement inclusive, non-overlapping thresholds; integer weights and scores in `0..100`; allowed-choice validation; null score for incomplete required inputs; and Critical override. Sort stored rule results by configured rule order, then field ID for deterministic reconstruction.

- [ ] **Step 4: Run score tests**

```powershell
go test ./internal/monitoring -run 'TestEvaluate'
```

- [ ] **Step 5: Commit**

```powershell
git add internal/monitoring
git commit -m "feat(monitoring): add deterministic risk evaluation"
```

## Task 3: Persist form templates, checks and immutable results

**Files:**
- Create: `migrations/000034_monitoring_setup.up.sql`
- Create: `migrations/000034_monitoring_setup.down.sql`
- Create: `internal/monitoring/repository.go`
- Create: `internal/monitoring/memory.go`
- Create: `internal/monitoring/postgres.go`
- Create: `internal/monitoring/postgres_integration_test.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/evidence/postgres.go`

- [ ] **Step 1: Write repository contract tests**

Require tenant isolation, exact-version reads, one current active revision, optimistic conflict handling, immutable result insertion, keyset-bounded lists and request references:

```go
type Repository interface {
    CreateFormRevision(context.Context, FormTemplate) (FormTemplate, error)
    FormRevision(context.Context, string, string, int64) (FormTemplate, error)
    TransitionForm(context.Context, LifecycleTransition) (FormTemplate, error)
    CreateCheckRevision(context.Context, MonitoringCheck) (MonitoringCheck, error)
    CheckRevision(context.Context, string, string, int64) (MonitoringCheck, error)
    TransitionCheck(context.Context, LifecycleTransition) (MonitoringCheck, error)
    AppendResult(context.Context, MonitoringResult) (MonitoringResult, error)
    ListResults(context.Context, string, string, int) ([]MonitoringResult, error)
}
```

- [ ] **Step 2: Verify repository tests fail**

```powershell
go test -tags=postgres ./internal/monitoring
```

- [ ] **Step 3: Add schema**

Create `monitoring_form_templates`, `monitoring_checks` and `monitoring_results` with UUID tenant foreign keys, JSONB validation, lifecycle checks, version uniqueness and result idempotency. Add nullable `form_template_id`, `form_template_version`, `collection_period_start` and `collection_period_end` to `capture_requests`; require the ID/version pair together.

- [ ] **Step 4: Implement repositories**

All PostgreSQL reads filter tenant in SQL. Lifecycle transitions lock the current exact revision, verify expected version, demote any prior current revision and create or update the approved revision atomically. Results are insert-only and deduplicate on `(tenant_id, monitoring_check_id, input_reference_id, evaluator_version)`.

- [ ] **Step 5: Run migration and repository tests**

```powershell
go test -tags=postgres ./internal/monitoring ./internal/evidence
python -m unittest deploy.tests.deployment_config_test
```

- [ ] **Step 6: Commit**

```powershell
git add migrations internal/monitoring internal/evidence
git commit -m "feat(monitoring): persist forms checks and results"
```

## Task 4: Implement governed form and Monitoring Check services

**Files:**
- Create: `internal/monitoring/service.go`
- Create: `internal/monitoring/service_test.go`
- Modify: `internal/evidence/service.go`

- [ ] **Step 1: Write failing service tests**

Test draft creation, submit, maker-checker approval, rejection, pause, retire, revoked-authority input rejection at the HTTP boundary, exact active-version collection, ineligible recipient, score validation and cross-Program link rejection.

- [ ] **Step 2: Verify failure**

```powershell
go test ./internal/monitoring -run 'TestService'
```

- [ ] **Step 3: Implement service methods**

Expose explicit methods:

```go
CreateForm(context.Context, Actor, CreateFormInput) (FormTemplate, error)
TransitionForm(context.Context, Actor, TransitionInput) (FormTemplate, error)
StartCollection(context.Context, Actor, StartCollectionInput) (evidence.Request, error)
CreateCheck(context.Context, Actor, CreateCheckInput) (MonitoringCheck, error)
TransitionCheck(context.Context, Actor, TransitionInput) (MonitoringCheck, error)
EvaluateSubmission(context.Context, string, string) (MonitoringResult, error)
EvaluateSource(context.Context, Actor, EvaluateSourceInput) (MonitoringResult, error)
```

`StartCollection` copies the exact active form fields into the Evidence Request, stores the exact template version and reporting period, and uses the existing recipient eligibility checks. `EvaluateSubmission` loads the immutable submission and exact template/check versions; it never evaluates against a later draft.

- [ ] **Step 4: Run service tests**

```powershell
go test ./internal/monitoring ./internal/evidence
```

- [ ] **Step 5: Commit**

```powershell
git add internal/monitoring internal/evidence
git commit -m "feat(monitoring): govern forms and monitoring checks"
```

## Task 5: Add HTTP routes and verified authority guards

**Files:**
- Create: `internal/httpapi/monitoring_handlers.go`
- Create: `internal/httpapi/monitoring_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/server.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/main.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing route and identity tests**

Assert every new route is present once; reads require configuration permission; commands bind tenant and actor from verified identity; body actor/tenant values cannot override identity; approval enforces a different maker; and Program-linked commands use current authority resolution.

- [ ] **Step 2: Verify route tests fail**

```powershell
go test ./internal/httpapi -run 'TestMonitoring|TestRouteRegistry'
```

- [ ] **Step 3: Wire services and handlers**

Add `Monitoring *monitoring.Service` to dependencies. Use existing `withPermission`, `material` and `bindJSONIdentity` wrappers. Return stable error codes: `monitoring_invalid`, `monitoring_conflict`, `monitoring_forbidden`, `monitoring_not_found`, and `monitoring_unavailable`; do not return source definitions or secrets in errors.

- [ ] **Step 4: Synchronize route inventory**

Regenerate or update `api/runtime.openapi.json` with exact methods, paths, command metadata and permissions. Run route-inventory parity tests.

- [ ] **Step 5: Run HTTP and service tests**

```powershell
go test ./internal/httpapi ./internal/monitoring ./cmd/api
```

- [ ] **Step 6: Commit**

```powershell
git add internal/httpapi cmd/api api/runtime.openapi.json
git commit -m "feat(api): expose governed monitoring setup"
```

## Task 6: Add Program setup commands and non-modal UI

**Files:**
- Create: `web/src/monitoringTypes.ts`
- Create: `web/src/monitoringApi.ts`
- Create: `web/src/components/ProgramSetupWorkspace.tsx`
- Create: `web/src/components/ProgramSetupWorkspace.test.tsx`
- Create: `web/src/monitoring.css`
- Modify: `web/src/continuityCommands.ts`
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/AppViews.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`

- [ ] **Step 1: Write failing Program setup UI tests**

Render the setup route and assert: no dialog/modal role; labelled steps; scope choices; one primary action; create command payload; requirement/control link sequence; saved Program ID retained after a later failure; Back preserves input; unauthorized action is unavailable with an explanation; and successful review returns to the new Program.

- [ ] **Step 2: Verify UI tests fail**

```powershell
npm test -- --run src/components/ProgramSetupWorkspace.test.tsx
```

- [ ] **Step 3: Add typed command functions**

Implement `createProgram`, `addRequirement`, `determineApplicability`, `addControlObjective`, `addControlImplementation`, `linkRequirementControl`, `addEvidenceContract` and `transitionProgram`. Every command loads current context, sends tenant only as required by the existing API, and relies on the server to replace actor fields.

- [ ] **Step 4: Build the resumable workspace**

Use a route target such as `#programs/new`. Save the returned aggregate after every command. Store only non-material draft input in `sessionStorage` under a tenant-and-actor key; clear it after activation. Provide inline errors with retry for the failed step and preserve prior saved server state.

- [ ] **Step 5: Add responsive styles and run tests**

```powershell
npm test -- --run src/components/ProgramSetupWorkspace.test.tsx src/components/ProgramsProjectionTruth.test.tsx src/copyQuality.test.ts
npm run typecheck
```

- [ ] **Step 6: Commit**

```powershell
git add web/src
git commit -m "feat(web): add guided Program setup"
```

## Task 7: Build reusable form configuration and collection UI

**Files:**
- Create: `web/src/components/FormBuilder.tsx`
- Create: `web/src/components/FormBuilder.test.tsx`
- Create: `web/src/components/FormCollectionPanel.tsx`
- Create: `web/src/components/FormCollectionPanel.test.tsx`
- Modify: `web/src/monitoringApi.ts`
- Modify: `web/src/monitoringTypes.ts`
- Modify: `web/src/components/CapturePanel.tsx`

- [ ] **Step 1: Write failing builder and collection tests**

Cover add/remove/reorder, unique IDs, Yes/No defaults, required/critical settings, weights, answer scores, threshold validation, preview, draft save, approval state, eligible collection, respondent/deadline fields, immutable exact version and API error recovery.

- [ ] **Step 2: Verify failure**

```powershell
npm test -- --run src/components/FormBuilder.test.tsx src/components/FormCollectionPanel.test.tsx
```

- [ ] **Step 3: Implement the builder**

Use semantic fieldsets and labelled controls. Generate stable field IDs once and preserve them during reorder. Default Yes to score `0`, No to `100`, weight to `1`, and make critical opt-in. Display the computed example thresholds before save. Do not allow unsupported text/date/file fields to carry scores.

- [ ] **Step 4: Implement collection creation and response provenance**

Start collection from an Active form version only. Show the exact reporting period and respondent on the Evidence Request. Capture remains responsible only for submission; the result panel loads evaluation separately and displays Pending evaluation when no result exists yet.

- [ ] **Step 5: Run form, capture and accessibility tests**

```powershell
npm test -- --run src/components/FormBuilder.test.tsx src/components/FormCollectionPanel.test.tsx src/components/CapturePanel.test.tsx src/Accessibility.test.tsx src/copyQuality.test.ts
npm run typecheck
```

- [ ] **Step 6: Commit**

```powershell
git add web/src
git commit -m "feat(web): build and collect monitoring forms"
```

## Task 8: Implement governed Source Access lifecycle

**Files:**
- Modify: `internal/sourceaccess/catalog_types.go`
- Modify: `internal/sourceaccess/catalog_service.go`
- Modify: `internal/sourceaccess/catalog_memory.go`
- Modify: PostgreSQL catalog repository files
- Create: `internal/sourceaccess/catalog_lifecycle_test.go`
- Create: `internal/sourceaccess/catalog_lifecycle_postgres_integration_test.go`
- Modify: `internal/httpapi/source_catalog_handlers.go`
- Modify: `internal/httpapi/source_catalog_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`

- [ ] **Step 1: Write failing lifecycle tests**

Test DRAFT→PENDING_APPROVAL→ACTIVE, maker-checker separation, exact parent revalidation, schema fingerprint conflict, one active current revision, pause, retire, rejected draft, cross-tenant denial and preview unavailable before activation.

- [ ] **Step 2: Verify failure**

```powershell
go test ./internal/sourceaccess ./internal/httpapi -run 'TestCatalogLifecycle|TestSourceCatalogLifecycle'
```

- [ ] **Step 3: Implement lifecycle service and persistence**

Add an explicit `TransitionRevision` contract for Connection, View and Binding. Approval must lock the candidate and parent revisions, verify the submitter differs from the approver, re-run normalized validation and atomically set the approved revision current. Pause and retire preserve historical executability evidence but block new reads.

- [ ] **Step 4: Add identity-bound lifecycle endpoints**

Expose submit, approve, reject, pause and retire routes for each catalog object. Require `CONFIG_WRITE`; resolve reviewer/authorizer responsibility at execution; bind actors from verified identity; return scrubbed stable errors.

- [ ] **Step 5: Run source and HTTP suites**

```powershell
go test ./internal/sourceaccess ./internal/httpapi
go test -tags=postgres ./internal/sourceaccess
```

- [ ] **Step 6: Commit**

```powershell
git add internal/sourceaccess internal/httpapi api/runtime.openapi.json
git commit -m "feat(sourceaccess): govern source activation lifecycle"
```

## Task 9: Build Data Sources workspace

**Files:**
- Create: `web/src/components/DataSourcesWorkspace.tsx`
- Create: `web/src/components/DataSourcesWorkspace.test.tsx`
- Modify: `web/src/AppViews.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/monitoringApi.ts`
- Modify: `web/src/monitoringTypes.ts`
- Modify: `web/src/monitoring.css`

- [ ] **Step 1: Write failing source UI tests**

Cover API/PostgreSQL/file/webhook choices, public versus saved credential selection, fixed endpoint/path entry, test failure, bounded preview, schema field selection, stable identifier selection, submit, independent approval, pause/retire, unavailable permissions and no secret rendering.

- [ ] **Step 2: Verify failure**

```powershell
npm test -- --run src/components/DataSourcesWorkspace.test.tsx
```

- [ ] **Step 3: Implement Configure tabs and source workflow**

Keep existing routing/approval configuration under **Routing and approvals** and add **Data sources**. Translate domain terms to Connection, Data set and Available fields. Put code, revision, adapter and schema fingerprint under an optional specialist disclosure.

- [ ] **Step 4: Implement test, preview and approval states**

Block the next step after a failed inspect or preview. Show observed field names/types and a bounded value preview. Never display `secret_ref` after save; display only a credential label supplied by a server-owned safe descriptor.

- [ ] **Step 5: Run source, accessibility, copy and type tests**

```powershell
npm test -- --run src/components/DataSourcesWorkspace.test.tsx src/Accessibility.test.tsx src/copyQuality.test.ts
npm run typecheck
```

- [ ] **Step 6: Commit**

```powershell
git add web/src
git commit -m "feat(web): add governed data source setup"
```

## Task 10: Integrate Monitoring Checks and Program results

**Files:**
- Create: `web/src/components/MonitoringChecks.tsx`
- Create: `web/src/components/MonitoringChecks.test.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Modify: `web/src/components/ProgramSetupWorkspace.tsx`
- Modify: `web/src/monitoringApi.ts`
- Modify: `internal/today/service.go`
- Modify: projection/runtime worker files selected by the implementation
- Create: focused backend projection tests

- [ ] **Step 1: Write failing integrated tests**

Assert a Program can attach an active form or Binding; exact versions persist; latest result shows score, band, coverage, freshness, failed rules, owner and reviewer; Not assessed is used for missing/stale data; adverse result creates a trigger and reviewer task; no approved Evidence Assessment is synthesized; automatic Matter creation occurs only with an eligible Automation Policy.

- [ ] **Step 2: Verify failure**

```powershell
go test ./internal/monitoring ./internal/today ./internal/workflow
npm test -- --run src/components/MonitoringChecks.test.tsx src/components/ProgramSetupWorkspace.test.tsx
```

- [ ] **Step 3: Implement result projection and task generation**

Publish a deduplicated result-evaluated event after appending a result. The worker creates or updates the Program latest-result projection and a Reviewer task keyed by result ID. It applies `program.trigger.ingest` with result provenance. It records a recommendation unless an active Automation Policy explicitly permits Matter creation.

- [ ] **Step 4: Implement Program Monitoring UI**

Provide Add monitoring check, Start form collection, Evaluate source, Review result and Resolve source actions according to state and authority. One primary action is shown for the current user. Keep technical receipts in an expandable Evidence details section.

- [ ] **Step 5: Run integrated tests**

```powershell
go test ./internal/monitoring ./internal/continuity ./internal/today ./internal/workflow ./internal/httpapi
npm test -- --run src/components/MonitoringChecks.test.tsx src/components/ProgramSetupWorkspace.test.tsx src/components/ProgramsProjectionTruth.test.tsx src/copyQuality.test.ts
```

- [ ] **Step 6: Commit**

```powershell
git add internal web/src
git commit -m "feat(monitoring): connect results to Program review"
```

## Task 11: Add Mobile Banking acceptance fixture and end-to-end proof

**Files:**
- Modify: `internal/bankverticals/install_specs.go`
- Modify: `internal/bankverticals/seed.go`
- Create: `internal/integration/mobile_banking_monitoring_test.go`
- Create: `web/src/components/MobileBankingMonitoringJourney.test.tsx`
- Modify: deployment seed only where durable fixture IDs are required

- [ ] **Step 1: Write the failing end-to-end backend test**

Create the Mobile Banking Program, two requirements and controls, face-verification source/check, five-question form/check, collection and response. Assert exact face fields, a passing source result, a critical password-reset response, reviewer task, Program trigger and absence of an automatically approved Evidence Assessment.

- [ ] **Step 2: Verify failure**

```powershell
go test ./internal/integration -run TestMobileBankingMonitoring
```

- [ ] **Step 3: Add realistic Clear Bank reference data**

Use operational names, plausible owners and explicit sample-data scope. Do not present the fixture as a compliant conclusion. Ensure stable fixture IDs are inserted once and validated on rerun rather than rewriting approved history.

- [ ] **Step 4: Add the UI journey test**

Exercise create Program → add requirements/controls → attach source → build form → start collection → submit → review result. Assert no API IDs or JSON are required or shown in the routine flow.

- [ ] **Step 5: Run acceptance tests**

```powershell
go test ./internal/integration ./internal/bankverticals
npm test -- --run src/components/MobileBankingMonitoringJourney.test.tsx src/copyQuality.test.ts
```

- [ ] **Step 6: Commit**

```powershell
git add internal web/src deploy
git commit -m "feat(demo): add mobile banking monitoring journey"
```

## Task 12: Documentation, visual proof and release verification

**Files:**
- Modify: `README.md`
- Modify: `DESIGN.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/architecture/connected-source-access.md`
- Modify: `docs/product/respond-and-capture.md`
- Create: `docs/acceptance/monitoring-setup-and-risk-scoring.md`
- Modify: `AGENTS.md` if the delivered workflow reveals a new durable copy or scoring rule

- [ ] **Step 1: Synchronize documentation**

Document exact maturity: on-demand form collection is available; automatic recurring generation is not. Record the separation between result, trigger, reviewer task, Evidence Assessment and Matter. Update Source Access limitations after lifecycle/UI delivery.

- [ ] **Step 2: Run full backend verification**

```powershell
go test ./...
go test -tags=postgres ./internal/monitoring ./internal/sourceaccess ./internal/evidence ./internal/httpapi
```

- [ ] **Step 3: Run full frontend verification**

```powershell
npm test
npm run typecheck
npm run build
```

- [ ] **Step 4: Run deployment verification**

```powershell
python -m unittest deploy.tests.deployment_config_test
git diff --check
```

- [ ] **Step 5: Render material states**

Render Program setup, Form Builder, Data Sources, source failure, incomplete form result, Critical result and mobile layouts at 375px, 768px and 1440px. Verify keyboard focus, no blocking modal, one primary action, no horizontal overflow and no exposed technical IDs. Store required fixtures/screenshots using the repository UI review convention.

- [ ] **Step 6: Final copy audit**

Search customer-facing sources for demonstration/dummy organization names, implementation commentary, raw status codes, connector internals and unqualified automatic-compliance claims. Extend `web/src/copyQuality.test.ts` only for precise repeatable patterns.

- [ ] **Step 7: Commit and push**

```powershell
git add README.md DESIGN.md AGENTS.md docs api web internal migrations deploy cmd
git commit -m "docs(monitoring): document governed setup and scoring"
git push origin main
```

## Plan self-review

- Every approved design section maps to Tasks 1–12.
- The first three delivery slices remain independently testable; integrated behavior is deferred to Task 10 rather than faked in an earlier UI.
- Form collection is on demand; no recurring form generator is introduced.
- Risk results remain observations and cannot silently become Evidence Assessments.
- Source activation includes maker-checker and exact-parent validation before the UI presents a source as available.
- The plan contains no browser-managed plaintext credential store.
- All material UI controls map to an implemented command and have explicit unavailable states.
