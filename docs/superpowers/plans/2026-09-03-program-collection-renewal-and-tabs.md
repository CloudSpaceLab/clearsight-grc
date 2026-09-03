# Program Collection Renewal and Tabs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add governed response expiry, pre-expiry renewal, bounded reminders, latest-respondent visibility and an accessible tabbed Program workspace without creating parallel form, workflow, scheduling or notification systems.

**Architecture:** Store the versioned collection policy on the Program-linked form Monitoring Check. Preserve Evidence Requests and submissions as immutable records, add origin-keyed request lineage for idempotent renewal, and use one focused leased monitoring work class inside the existing worker. Expose one bounded collection-summary read model to a unified Monitoring view and split Program detail into URL-addressable sections with a mobile selector replacement.

**Tech Stack:** Go 1.25, pgx/PostgreSQL 18, the existing runtime outbox/inbox and worker, React 19, TypeScript 7, Vite 8, Vitest/Testing Library/axe and Playwright rendered-evidence scripts.

---

## Execution preflight

The worktree already contains uncommitted submitted-evidence work in `internal/evidence/model.go`, `internal/monitoring/*`, `web/src/components/FormBuilder.tsx`, `web/src/components/MonitoringSetup.tsx`, related tests/styles and monitoring documentation. Those edits belong to the user and overlap this plan.

Before Task 1, run:

```powershell
git status --short
git diff -- internal/evidence/model.go internal/monitoring web/src/components/FormBuilder.tsx web/src/components/MonitoringSetup.tsx web/src/monitoringTypes.ts web/src/monitoring.css
```

Record the current diff in the issue log. Do not reset, discard, reformat wholesale or include unrelated files in feature commits. Merge every task below into the submitted-evidence changes at their current state.

## File structure

New focused units:

- `internal/monitoring/collection_policy.go` — policy validation and calendar/reminder calculations.
- `internal/monitoring/collection_cycle.go` — cycle, recipient route, summary and persistence contracts.
- `internal/monitoring/collection_memory.go` / `collection_postgres.go` — operational persistence.
- `internal/monitoring/collection_consumer.go` — submission event to next cycle.
- `internal/monitoring/collection_maintainer.go` — renewal and reminder execution.
- `internal/evidence/request_origin.go` / `request_origin_postgres.go` — idempotent request origin.
- `internal/evidence/previous_response.go` — compatible prior-response prefill.
- `web/src/components/ProgramDetailSections.tsx` — Program section navigation.
- `web/src/components/CollectionPolicyForm.tsx` — policy entry.
- `web/src/components/CollectionRecord.tsx` — unified collection state.
- `migrations/000076_program_collection_renewal.up.sql` / `.down.sql` — schema, sequenced after the current mainline migration set.

Existing monitoring, evidence, HTTP, worker, React routing, capture, fixture, CSS and documentation files change only where listed by each task.

---

### Task 1: Record the UI decision brief and before-state baseline

**Files:**
- Create: `docs/design/program-collection-renewal-tabs-decision-brief.md`
- Create: `docs/screenshots/program-collection-renewal/README.md`
- Create: `docs/screenshots/program-collection-renewal/before-monitoring.png`
- Modify: `docs/superpowers/issues/2026-09-03-program-collection-renewal-and-tabs.md`

- [ ] **Step 1: Preserve the supplied baseline**

```powershell
New-Item -ItemType Directory -Force docs/screenshots/program-collection-renewal | Out-Null
Copy-Item -LiteralPath 'C:\Users\Son\AppData\Local\Temp\codex-clipboard-c7ef6dba-f9ad-40f9-b2e9-c003556d2af9.png' -Destination 'docs/screenshots/program-collection-renewal/before-monitoring.png'
Get-FileHash docs/screenshots/program-collection-renewal/before-monitoring.png -Algorithm SHA256
```

Record the SHA-256, dimensions, source date and statement that the image is the unchanged before-state.

- [ ] **Step 2: Write the compact decision brief**

Include product job, primary object/action, six sections, desktop tabs, mobile/200% selector, one dominant action, no-policy/current/renewal/expired/blocked states, sensitive-data boundary and required rendered viewports. Do not introduce a new token, illustration style or navigation system.

- [ ] **Step 3: Verify and commit only the baseline**

```powershell
git diff --check -- docs/design/program-collection-renewal-tabs-decision-brief.md docs/screenshots/program-collection-renewal/README.md docs/superpowers/issues/2026-09-03-program-collection-renewal-and-tabs.md
git add docs/design/program-collection-renewal-tabs-decision-brief.md docs/screenshots/program-collection-renewal/README.md docs/screenshots/program-collection-renewal/before-monitoring.png docs/superpowers/issues/2026-09-03-program-collection-renewal-and-tabs.md
git commit -m "docs: baseline collection renewal workspace"
```

Expected: diff check exits 0 and the commit contains only four listed paths.

---

### Task 2: Add and validate the versioned collection policy

**Files:**
- Create: `internal/monitoring/collection_policy.go`
- Create: `internal/monitoring/collection_policy_test.go`
- Modify: `internal/monitoring/model.go`
- Modify: `internal/monitoring/service.go`
- Modify: `internal/monitoring/service_test.go`
- Modify: `internal/monitoring/repository.go`
- Modify: `internal/monitoring/memory.go`

- [ ] **Step 1: Write failing policy tests**

Add tests for: 12 months defaulting to 30 days/3 reminders; 1 month rejecting a 30-day window; 1–5 reminders accepted; 6 rejected; form checks requiring policy; source checks rejecting it; and existing active form checks remaining readable with no policy.

```go
func TestNormalizeCollectionPolicyDefaults(t *testing.T) {
	got, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 12})
	if err != nil { t.Fatal(err) }
	want := CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3}
	if got != want { t.Fatalf("policy = %#v, want %#v", got, want) }
}

func TestNormalizeCollectionPolicyRejectsWindowBeyondOneMonth(t *testing.T) {
	_, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 1, RenewalWindowDays: 30, ReminderCount: 3})
	if !errors.Is(err, ErrInvalid) { t.Fatalf("expected invalid policy, got %v", err) }
}
```

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/monitoring -run 'Test.*CollectionPolicy' -count=1
```

Expected: FAIL because `CollectionPolicy` is undefined.

- [ ] **Step 3: Implement the minimal policy model**

```go
type CollectionPolicy struct {
	ValidityMonths int `json:"validity_months"`
	RenewalWindowDays int `json:"renewal_window_days"`
	ReminderCount int `json:"reminder_count"`
}
```

Add `CollectionPolicy *CollectionPolicy` to `MonitoringCheck` and `CreateCheckInput`. `normalizeCollectionPolicy` requires 1–120 months, defaults zero window/count to 30/3, accepts 1–90 days and 1–5 reminders, and requires `RenewalWindowDays <= ValidityMonths*28-1`. Require policy for newly created `FORM` checks and reject it for `SOURCE` checks.

- [ ] **Step 4: Add governed revision tests and implementation**

Add `UpdateCollectionPolicyInput{ID, ExpectedVersion, Policy}`. Revising an active check creates a non-current `DRAFT` at `version+1`, leaves the old active revision current, and uses existing submit/approve maker-checker transitions. Add repository method `ReviseCheck` so current-version lock and draft insert are atomic in PostgreSQL and mutex-protected in memory.

- [ ] **Step 5: Run and commit**

```powershell
go test ./internal/monitoring -run 'Test.*(CollectionPolicy|ReviseCheck|MakerChecker)' -count=1
git add internal/monitoring/collection_policy.go internal/monitoring/collection_policy_test.go internal/monitoring/model.go internal/monitoring/service.go internal/monitoring/service_test.go internal/monitoring/repository.go internal/monitoring/memory.go
git commit -m "feat(monitoring): define collection renewal policy"
```

Expected: tests PASS. Preserve the existing `UseAsEvidence` and `SubmissionEvidence` edits in these files.

---

### Task 3: Persist policy, request origin and collection cycles

**Files:**
- Create: `migrations/000076_program_collection_renewal.up.sql`
- Create: `migrations/000076_program_collection_renewal.down.sql`
- Create: `internal/monitoring/collection_cycle.go`
- Create: `internal/monitoring/collection_memory.go`
- Create: `internal/monitoring/collection_postgres.go`
- Create: `internal/monitoring/collection_postgres_integration_test.go`
- Modify: `internal/monitoring/postgres.go`
- Modify: `internal/monitoring/repository.go`
- Modify: `internal/monitoring/schema_test.go`
- Modify: `internal/evidence/model.go`
- Modify: `docs/architecture/durable-schema-ownership.md`

- [ ] **Step 1: Write failing schema and integration tests**

Require the three policy columns, four request-lineage columns, bounded `previous_responses` JSONB, `monitoring_collection_cycles`, a due partial index, a Program summary index and a unique request-origin index. PostgreSQL tests prove tenant isolation, legacy-null policy, policy round-trip, origin uniqueness, lease fencing, due ordering and limit-after-scope.

- [ ] **Step 2: Run and confirm RED or explicit SKIP**

```powershell
go test ./internal/monitoring -run TestMonitoringMigrationIncludesCollectionRenewal -count=1
go test -tags "postgres postgresintegration" ./internal/monitoring -run TestPostgresCollection -count=1
```

Expected: schema test FAIL before migration; integration FAIL with configured PostgreSQL or explicit SKIP without `TEST_DATABASE_URL`.

- [ ] **Step 3: Add migration 000076**

Add nullable `validity_months`, `renewal_window_days`, `reminder_count` to `monitoring_checks`, constrained as an all-null legacy group or valid all-present group for `FORM`. Reuse the shared `origin_type`, `origin_id` and `origin_version` capture contract from migration `000036_shared_form_capture_contract`; add nullable `predecessor_request_id` and non-null `previous_responses jsonb DEFAULT '{}'` to `capture_requests`, with monitoring-lineage, same-table predecessor and bounded JSON-object checks. The existing partial unique `(tenant_id,origin_type,origin_id,origin_version)` index provides idempotency.

Create `monitoring_collection_cycles` with tenant/Program/check IDs, policy/check version, current/predecessor request, latest submission, expiry/open/next timestamps, reminder progress, recipient route, delivery state, state, lease, attempts, safe error and timestamps. Its state check permits only `SCHEDULED`, `CLAIMED`, `AWAITING_RESPONSE`, `COMPLETE`, `CANCELLED`, `BLOCKED`, `FAILED`. Store no raw address, answers, artifact content or invitation token.

- [ ] **Step 4: Implement persistence contracts**

Add create/upsert, exact get, claim due, complete action, retry/terminal failure, cancel-by-check and Program-summary methods in memory/PostgreSQL. Recipient routes are either `INTERNAL_PRINCIPAL` with principal ID or `EXTERNAL_CONTACT` with opaque contact ref and safe hint.

- [ ] **Step 5: Round-trip policy in existing check queries**

Update `insertCheckRevision`, `checkSelect` and `scanCheck` together. Do not drop or replace current submitted-evidence fields.

- [ ] **Step 6: Run and commit**

```powershell
go test ./internal/monitoring -count=1
go test -tags postgres ./internal/monitoring ./internal/platform/database -count=1
go test -tags "postgres postgresintegration" ./internal/monitoring -run TestPostgresCollection -count=1
git add migrations/000076_program_collection_renewal.up.sql migrations/000076_program_collection_renewal.down.sql internal/monitoring/collection_cycle.go internal/monitoring/collection_memory.go internal/monitoring/collection_postgres.go internal/monitoring/collection_postgres_integration_test.go internal/monitoring/postgres.go internal/monitoring/repository.go internal/monitoring/schema_test.go internal/evidence/model.go docs/architecture/durable-schema-ownership.md
git commit -m "feat(monitoring): persist collection renewal cycles"
```

Expected: unit/postgres-build PASS; integration PASS or explicit environment SKIP.

---

### Task 4: Make Evidence Request origin creation idempotent

**Files:**
- Create: `internal/evidence/request_origin.go`
- Create: `internal/evidence/request_origin_test.go`
- Create: `internal/evidence/request_origin_postgres.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/evidence/repository.go`
- Modify: `internal/evidence/memory.go`
- Modify: `internal/evidence/postgres.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/service_test.go`

- [ ] **Step 1: Write failing origin tests**

Test same-origin replay returning the same request, changed immutable input returning `ErrVersionConflict`, cross-tenant origins remaining isolated, and predecessor requiring the same tenant/subject/origin with sequence exactly one lower.

```go
func TestCreateRequestReusesExactOrigin(t *testing.T) {
	service := newOriginService(t)
	input := validRequestInput()
	input.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 2}
	first, err := service.CreateRequest(context.Background(), input)
	if err != nil { t.Fatal(err) }
	second, err := service.CreateRequest(context.Background(), input)
	if err != nil || second.ID != first.ID { t.Fatalf("second = %#v, err = %v", second, err) }
}
```

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/evidence -run 'TestCreateRequest.*Origin' -count=1
```

Expected: FAIL because request origin is missing.

- [ ] **Step 3: Implement exact origin**

Add `RequestOrigin{Type, ID, Sequence}`, constant `MONITORING_COLLECTION`, `Origin *RequestOrigin` and `PredecessorRequestID` to request/input types. Add exact `GetRequestByOrigin`. `CreateRequest` returns an existing request only when its immutable fingerprint matches; otherwise it returns `ErrVersionConflict`. PostgreSQL handles unique conflict by exact origin read; memory maintains an origin index under its current mutex.

- [ ] **Step 4: Run and commit**

```powershell
go test ./internal/evidence -count=1
go test -tags postgres ./internal/evidence -count=1
git add internal/evidence/request_origin.go internal/evidence/request_origin_test.go internal/evidence/request_origin_postgres.go internal/evidence/model.go internal/evidence/repository.go internal/evidence/memory.go internal/evidence/postgres.go internal/evidence/service.go internal/evidence/service_test.go
git commit -m "feat(evidence): add idempotent request lineage"
```

Expected: PASS including recipient, invitation, source-prefill and submitted-evidence tests.

---

### Task 5: Add previous-response prefill and provenance

**Files:**
- Create: `internal/evidence/previous_response.go`
- Create: `internal/evidence/previous_response_test.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/evidence/service.go`
- Modify: `internal/evidence/source_bindings.go`
- Modify: `internal/evidence/field_validation.go`
- Modify: `internal/evidence/field_validation_test.go`

- [ ] **Step 1: Write failing compatibility tests**

Use predecessor text, date, choice, file and signature fields. Assert compatible scalar answers copy with request/submission/date provenance, removed/new fields do not copy, and file/photo/signature IDs never become successor defaults.

```go
func TestBuildPreviousResponsePrefillCopiesOnlyCompatibleScalars(t *testing.T) {
	previous := Request{ID: "request-1", Fields: []Field{{ID: "name", Type: "text"}, {ID: "reviewed", Type: "date"}, {ID: "certificate", Type: "file"}}}
	submission := Submission{ID: "submission-1", SubmittedAt: time.Date(2026, 8, 14, 10, 32, 0, 0, time.UTC), Answers: map[string]string{"name": "Acme Processing Limited", "reviewed": "2026-08-14", "certificate": "artifact-1"}}
	next := []Field{{ID: "name", Type: "text"}, {ID: "reviewed", Type: "date"}, {ID: "certificate", Type: "file"}, {ID: "new_owner", Type: "text", Required: true}}
	got := BuildPreviousResponsePrefill(previous, submission, next)
	if len(got) != 2 || got["name"].Value != "Acme Processing Limited" || got["reviewed"].PreviousSubmissionID != submission.ID { t.Fatalf("prefill = %#v", got) }
}
```

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/evidence -run 'TestBuildPreviousResponsePrefill|TestPreviousResponseProvenance' -count=1
```

Expected: FAIL because predecessor prefill is absent.

- [ ] **Step 3: Implement bounded prior-response data**

Add `PreviousResponseValue{Value, PreviousRequestID, PreviousSubmissionID, PreviousSubmittedAt}` and `PreviousResponses map[string]PreviousResponseValue` to request/input, persisted as bounded JSONB. Add `PRIOR_RESPONSE_PREFILLED` answer origin. A respondent change retains predecessor IDs/value while using `RESPONDENT_CORRECTED`. Current governed source prefill wins when both source and previous response exist.

- [ ] **Step 4: Reuse current validation**

Include only values accepted by the successor field type/options. Keep normal required-field submission validation. Never include file, photo or signature answers.

- [ ] **Step 5: Run and commit**

```powershell
go test ./internal/evidence -count=1
go test ./internal/monitoring -run 'TestService.*SubmissionEvidence' -count=1
git add internal/evidence/previous_response.go internal/evidence/previous_response_test.go internal/evidence/model.go internal/evidence/service.go internal/evidence/source_bindings.go internal/evidence/field_validation.go internal/evidence/field_validation_test.go
git commit -m "feat(evidence): attribute renewal response prefill"
```

Expected: PASS; submitted-evidence projection remains intact.

---

### Task 6: Project submissions into deterministic renewal cycles

**Files:**
- Create: `internal/monitoring/collection_consumer.go`
- Create: `internal/monitoring/collection_consumer_test.go`
- Modify: `internal/monitoring/collection_policy.go`
- Modify: `internal/monitoring/collection_cycle.go`
- Modify: `internal/monitoring/collection_memory.go`
- Modify: `internal/monitoring/collection_postgres.go`
- Modify: `internal/monitoring/service.go`
- Modify: `internal/monitoring/service_test.go`
- Modify: `cmd/worker/services_postgres.go`

- [ ] **Step 1: Write failing calendar tests**

Test 31 January + 1 month = last valid February day; 29 February + 12 months = 28 February; counts 1–5 produce ordered unique reminders before expiry; renewal opening equals expiry minus window.

```go
func TestCollectionDatesClampMonthEnd(t *testing.T) {
	submitted := time.Date(2027, 1, 31, 10, 32, 0, 0, time.UTC)
	expires, opens, reminders := CollectionDates(submitted, CollectionPolicy{ValidityMonths: 1, RenewalWindowDays: 20, ReminderCount: 3})
	want := time.Date(2027, 2, 28, 10, 32, 0, 0, time.UTC)
	if !expires.Equal(want) || !opens.Equal(want.AddDate(0, 0, -20)) || len(reminders) != 3 { t.Fatalf("dates = %s %s %#v", expires, opens, reminders) }
}
```

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/monitoring -run 'TestCollectionDates|TestCollectionConsumer' -count=1
```

Expected: FAIL because calculation/consumer are missing.

- [ ] **Step 3: Implement calendar/reminder arithmetic**

Implement clamped month addition explicitly; do not use bare `time.AddDate` for invalid month-end dates. Reminder `n` occurs at `open + window*n/(count+1)` using integer duration division. The initial renewal request is not a reminder.

- [ ] **Step 4: Implement inbox-idempotent consumer**

Update `StartCollection` first: resolve the exact active form check and policy, create the initial request with origin `{MONITORING_COLLECTION, check.ID, 1}`, and retain only a stable internal-principal or opaque external-contact route. Then `CollectionConsumer.Publish` accepts `EvidenceResponseSubmitted`, reads the safe `submission_id`, exact request/origin/check and upserts one cycle. It records inbox after the idempotent write. A paused/retired check closes prior cycle and schedules nothing. Non-monitoring origins are ignored. Answers never enter event/cycle.

- [ ] **Step 5: Register and commit**

```powershell
go test ./internal/monitoring ./internal/evidence ./cmd/worker -count=1
git add internal/monitoring/collection_policy.go internal/monitoring/collection_cycle.go internal/monitoring/collection_consumer.go internal/monitoring/collection_consumer_test.go internal/monitoring/collection_memory.go internal/monitoring/collection_postgres.go internal/monitoring/service.go internal/monitoring/service_test.go cmd/worker/services_postgres.go
git commit -m "feat(worker): schedule collection renewal from submissions"
```

Expected: duplicate delivery creates one cycle.

---

### Task 7: Create renewal requests and deliver bounded reminders

**Files:**
- Create: `internal/monitoring/collection_maintainer.go`
- Create: `internal/monitoring/collection_maintainer_test.go`
- Modify: `internal/monitoring/collection_cycle.go`
- Modify: `internal/monitoring/collection_memory.go`
- Modify: `internal/monitoring/collection_postgres.go`
- Modify: `internal/evidence/model.go`
- Modify: `internal/evidence/service.go`
- Modify: `cmd/worker/services.go`
- Modify: `cmd/worker/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`
- Modify: `.env.example`

- [ ] **Step 1: Write failing maintainer tests**

Cover one successor, crash/retry origin reuse, scalar prefill, internal assignment, external route/delivery receipt, missing adapter = `BLOCKED`, exact 1/3/5 reminders, submission/cancellation/pause/retirement/reassignment stop, bounded terminal failure.

```go
func TestCollectionMaintainerReusesSuccessorAfterInterruption(t *testing.T) {
	fixture := renewalFixture(t)
	maintainer := fixture.maintainer
	maintainer.afterRequest = func() error { return errors.New("simulated interruption") }
	if _, err := maintainer.Maintain(context.Background(), fixture.openAt, 10); err == nil { t.Fatal("expected interruption") }
	maintainer.afterRequest = nil
	if _, err := maintainer.Maintain(context.Background(), fixture.openAt.Add(time.Minute), 10); err != nil { t.Fatal(err) }
	if got := fixture.requestsForOrigin(2); got != 1 { t.Fatalf("successor count = %d", got) }
}
```

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/monitoring -run TestCollectionMaintainer -count=1
```

Expected: FAIL because maintainer/dispatcher are absent.

- [ ] **Step 3: Add the narrow delivery boundary**

```go
type CollectionDispatcher interface {
	ValidateRoute(context.Context, string, RecipientRoute) error
	DispatchRequest(context.Context, evidence.Request, RecipientRoute) (DeliveryReceipt, error)
	DispatchReminder(context.Context, evidence.Request, RecipientRoute, int) (DeliveryReceipt, error)
}
```

Internal dispatch verifies canonical assignment and returns `ASSIGNED`. External dispatch resolves an opaque contact ref, issues a request-scoped invitation and returns `DELIVERED` only with provider receipt. Missing resolver/adapter returns `ErrDeliveryUnavailable`. Do not add contact CRUD, SMTP, notification templates or a notification center.

- [ ] **Step 4: Implement renewal/reminder actions**

At renewal opening re-read Program/check/form/recipient/authority, create or reuse sequence `n+1`, attach predecessor and compatible prefill, set deadline to expiry, dispatch, then schedule the first reminder. Each reminder re-reads the request and stops when submitted/cancelled/expired/not assigned. After configured reminders, expiry marks collection attention potentially expired; it does not change risk/evidence status.

- [ ] **Step 5: Register one worker class and fail-closed config**

Use `monitoring-collection-renewal`, batch 50, existing worker supervision. Validate public HTTPS capture base outside development; never accept a raw default recipient. Default missing external adapter is blocked, not sent.

- [ ] **Step 6: Run and commit**

```powershell
go test ./internal/monitoring ./internal/evidence ./internal/platform/config ./cmd/worker -count=1
go test -tags postgres ./internal/monitoring ./internal/evidence ./cmd/worker -count=1
git add internal/monitoring/collection_maintainer.go internal/monitoring/collection_maintainer_test.go internal/monitoring/collection_cycle.go internal/monitoring/collection_memory.go internal/monitoring/collection_postgres.go internal/evidence/model.go internal/evidence/service.go cmd/worker/services.go cmd/worker/services_memory.go cmd/worker/services_postgres.go internal/platform/config/config.go internal/platform/config/config_test.go .env.example
git commit -m "feat(worker): renew program collection requests"
```

Expected: PASS. PCR-06 stays operationally blocked until a real deployment adapter is exercised.

---

### Task 8: Expose policy commands and bounded collection summaries

**Files:**
- Modify: `internal/httpapi/monitoring_handlers.go`
- Modify: `internal/httpapi/monitoring_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `internal/httpapi/route_registry_test.go`
- Modify: `internal/httpapi/runtime_contract_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write failing handler/security tests**

Cover policy defaults, source-policy rejection, expected-version policy revision, body tenant/actor ignored, cross-tenant not found, and summary with latest submission, safe respondent label/hint, expiry, reminder progress, delivery and freshness.

- [ ] **Step 2: Run and confirm RED**

```powershell
go test ./internal/httpapi -run 'Test.*(CollectionPolicy|CollectionSummary)' -count=1
```

Expected: FAIL because routes are absent.

- [ ] **Step 3: Add focused routes**

```text
POST /api/v1/monitoring-checks/{id}/collection-policy
GET  /api/v1/programs/{id}/collection-summaries
```

The write uses current verified identity and monitoring configuration authority. The read uses one Program-scoped query capped at 100; no N+1 or broad in-memory authorization.

- [ ] **Step 4: Implement summary response**

Return check ID, latest request, latest submission time, permitted respondent label/hint, expiry/opening, currency state, active deadline, reminders sent/count, delivery state, safe error, generated time and source version. Unknown values remain absent.

- [ ] **Step 5: Update executable contract, run and commit**

```powershell
go test ./internal/httpapi ./cmd/api -count=1
go test ./internal/httpapi -run 'TestRuntimeContract|TestRouteRegistry' -count=1
git add internal/httpapi/monitoring_handlers.go internal/httpapi/monitoring_handlers_test.go internal/httpapi/route_registry.go internal/httpapi/route_registry_test.go internal/httpapi/runtime_contract_test.go internal/httpapi/server.go cmd/api/services.go cmd/api/services_memory.go cmd/api/services_postgres.go api/runtime.openapi.json
git commit -m "feat(api): expose collection renewal state"
```

Expected: routes and runtime contract match.

---

### Task 9: Add typed policy setup and unified Monitoring records

**Files:**
- Create: `web/src/components/CollectionPolicyForm.tsx`
- Create: `web/src/components/CollectionPolicyForm.test.tsx`
- Create: `web/src/components/CollectionRecord.tsx`
- Create: `web/src/components/CollectionRecord.test.tsx`
- Modify: `web/src/monitoringTypes.ts`
- Modify: `web/src/monitoringApi.ts`
- Modify: `web/src/components/MonitoringSetup.tsx`
- Modify: `web/src/components/MonitoringSetup.test.tsx`
- Modify: `web/src/monitoring.css`

- [ ] **Step 1: Write failing API/form tests**

Assert `collection_policy: { validity_months: 12, renewal_window_days: 30, reminder_count: 3 }`. Render labels **Response expires after**, **Renewal starts**, **Reminders during renewal**; reject one month/30 days; preserve values after API failure; one primary **Add collection to Program**.

- [ ] **Step 2: Run and confirm RED**

```powershell
Set-Location web
npm test -- --run src/components/CollectionPolicyForm.test.tsx src/components/MonitoringSetup.test.tsx
```

Expected: FAIL because components/types/API are missing.

- [ ] **Step 3: Implement types/API/form**

Add `CollectionPolicy`, `CollectionSummary`, `updateCollectionPolicy`, `loadCollectionSummaries`. Use bounded numeric inputs and preview: **Responses will be renewed 30 days before they reach 12 months old. The initial request is followed by up to 3 reminders.**

- [ ] **Step 4: Write failing unified-record tests**

One form plus linked check must render once. Assert question count/status/validity, renewal text, **Last submitted 14 Aug 2026 at 10:32 by Vendor security contact**, and **Expires 14 Aug 2027**. Cover no policy, no response, renewal due, potentially expired, awaiting response and blocked.

- [ ] **Step 5: Implement grouping and partial failure**

Approved unlinked forms appear only in setup. Form-linked checks use `CollectionRecord`; source checks remain separate. Load summaries independently; summary failure renders **Collection dates unavailable** with local retry. Preserve the in-progress `SubmittedEvidence` result section.

- [ ] **Step 6: Run and commit**

```powershell
npm test -- --run src/components/CollectionPolicyForm.test.tsx src/components/CollectionRecord.test.tsx src/components/MonitoringSetup.test.tsx src/copyQuality.test.ts
npm run typecheck
npm run build
Set-Location ..
git add web/src/components/CollectionPolicyForm.tsx web/src/components/CollectionPolicyForm.test.tsx web/src/components/CollectionRecord.tsx web/src/components/CollectionRecord.test.tsx web/src/monitoringTypes.ts web/src/monitoringApi.ts web/src/components/MonitoringSetup.tsx web/src/components/MonitoringSetup.test.tsx web/src/monitoring.css
git commit -m "feat(web): show collection expiry and reminders"
```

Expected: tests/type/build PASS.

---

### Task 10: Replace long Program detail with accessible sections

**Files:**
- Create: `web/src/components/ProgramDetailSections.tsx`
- Create: `web/src/components/ProgramDetailSections.test.tsx`
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/appRouting.test.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/AppViews.tsx`
- Modify: `web/src/components/ProgramsWorkspace.tsx`
- Modify: `web/src/components/ExactTargetWorkspaces.test.tsx`
- Modify: `web/src/Accessibility.test.tsx`
- Modify: `web/src/continuity.css`
- Modify: `web/src/interventions.css`
- Modify: `web/src/product-finish.css`
- Modify: `web/src/ui-preferences.css`

- [ ] **Step 1: Write failing route tests**

Extend `WorkspaceTarget` with `programSection`. Assert `#programs/program-1/monitoring` parses to Program + monitoring and routeHash emits `#programs/program-1/history`. Unknown section falls back to `overview` without losing ID.

- [ ] **Step 2: Write failing tabs/selector tests**

Assert six names, one visible `tabpanel`, correct labels/controls, ArrowLeft/Right/Home/End behavior and section callback. Under mobile fixture/class assert labelled combobox **Program section** replaces tablist.

- [ ] **Step 3: Run and confirm RED**

```powershell
Set-Location web
npm test -- --run src/appRouting.test.ts src/components/ProgramDetailSections.test.tsx src/components/ExactTargetWorkspaces.test.tsx
```

Expected: FAIL because section routing/component is absent.

- [ ] **Step 4: Implement fixed section contract**

```ts
export const programSections = [
  { id: "overview", label: "Overview" },
  { id: "requirements-controls", label: "Requirements & controls" },
  { id: "monitoring", label: "Monitoring" },
  { id: "evidence-results", label: "Evidence & results" },
  { id: "issues-actions", label: "Issues & actions" },
  { id: "history", label: "History" },
] as const;
```

Keep Program identity/stale notice above navigation. Overview receives digest/lifecycle/details; requirements/control content goes to Requirements & controls; MonitoringSetup to Monitoring; evidence expectations/results to Evidence & results; linked work to Issues & actions; reconstruction to History. Unavailable backend sections show bounded empty state, not enabled dead controls.

- [ ] **Step 5: Wire URL state and responsive replacement**

Pass section/callback App → ProgramsView → ProgramsWorkspace and use existing `navigate`/`routeHash` so Back/Forward work. Do not mutate hash in the detail component. Desktop/tablet use tabs; mobile and 200%-proxy use native selector without horizontal scrolling; selector change focuses section heading; reduced motion remains honored.

- [ ] **Step 6: Run and commit**

```powershell
npm test -- --run src/appRouting.test.ts src/components/ProgramDetailSections.test.tsx src/components/ExactTargetWorkspaces.test.tsx src/Accessibility.test.tsx src/components/ProgramsProjectionTruth.test.tsx
npm run typecheck
Set-Location ..
git add web/src/components/ProgramDetailSections.tsx web/src/components/ProgramDetailSections.test.tsx web/src/appRouting.ts web/src/appRouting.test.ts web/src/App.tsx web/src/AppViews.tsx web/src/components/ProgramsWorkspace.tsx web/src/components/ExactTargetWorkspaces.test.tsx web/src/Accessibility.test.tsx web/src/continuity.css web/src/interventions.css web/src/product-finish.css web/src/ui-preferences.css
git commit -m "feat(web): organize Program detail into sections"
```

Expected: tests/type PASS; stale projection truth unchanged.

---

### Task 11: Show previous-response provenance in Capture

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/components/CapturePanel.tsx`
- Create: `web/src/components/CapturePanel.previous-response.test.tsx`
- Modify: `web/src/capture-inputs.css`
- Modify: `web/src/copyQuality.test.ts`

- [ ] **Step 1: Write failing Capture tests**

Render previous text/date/choice values; assert initial values and **From the response submitted on 14 Aug 2026**. Change one and assert **Changed by you · previous response was [value]**. File/signature remain empty.

- [ ] **Step 2: Run and confirm RED**

```powershell
Set-Location web
npm test -- --run src/components/CapturePanel.previous-response.test.tsx
```

Expected: FAIL because browser contract/renderer are absent.

- [ ] **Step 3: Implement request type, initialization and labels**

Add `previous_responses` map containing value, predecessor request/submission and submitted time. Initialize compatible non-file answers after source-prefill resolution; current governed source prefill wins when both exist. Review summary distinguishes unchanged prior response from respondent correction.

- [ ] **Step 4: Run and commit**

```powershell
npm test -- --run src/components/CapturePanel.previous-response.test.tsx src/components/CapturePanel.source-bindings.test.tsx src/components/CapturePanel.test.tsx src/copyQuality.test.ts
npm run typecheck
npm run build
Set-Location ..
git add web/src/types.ts web/src/components/CapturePanel.tsx web/src/components/CapturePanel.previous-response.test.tsx web/src/capture-inputs.css web/src/copyQuality.test.ts
git commit -m "feat(web): show renewal response provenance"
```

Expected: tests/type/build PASS.

---

### Task 12: Add deterministic fixtures and rendered acceptance

**Files:**
- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/staticDemo.test.ts`
- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `web/scripts/capture-program-review-evidence.mjs`
- Modify: `docs/screenshots/program-collection-renewal/README.md`
- Modify: `docs/quality/rendered-ui-evidence.md`

- [ ] **Step 1: Add failing fixture tests**

Create sample-labelled fixtures for no policy, current response with respondent/time, renewal due with 1/3 reminders, potentially expired, awaiting response, external delivery blocked and long names. Assert dates/counts are stored fixture values, not derived from viewer now.

- [ ] **Step 2: Run and confirm RED**

```powershell
Set-Location web
npm test -- --run src/staticDemo.test.ts src/components/CollectionRecord.test.tsx
```

Expected: FAIL until fixtures/routes exist.

- [ ] **Step 3: Extend capture matrix**

Capture `#programs/program-ndpa/overview`, `/monitoring`, `/evidence-results` at 1440×900 light/dark, 1024×768, 390×844, 320×800 and 200% proxy. Include tab focus, mobile selector, blocked delivery and long content.

- [ ] **Step 4: Build, render, inspect and repair**

```powershell
npm run build
npm run review:ui
```

Inspect every PNG for scan order, section, last respondent/time, expiry, dominant action, long labels, themes, focus and overflow. Fix highest-impact defect, rerun affected capture and record before/after hash in README.

- [ ] **Step 5: Commit**

```powershell
Set-Location ..
git add web/src/staticDemo.ts web/src/staticDemo.test.ts web/scripts/capture-ui-evidence.mjs web/scripts/capture-program-review-evidence.mjs docs/screenshots/program-collection-renewal/README.md docs/quality/rendered-ui-evidence.md
git commit -m "test(ui): verify collection renewal workspace"
```

Expected: capture scripts exit 0 without browser application error or horizontal overflow.

---

### Task 13: Synchronize product truth and issue status

**Files:**
- Modify: `README.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/architecture/governance-runtime.md`
- Modify: `docs/acceptance/monitoring-setup-and-risk-scoring.md`
- Modify: `docs/superpowers/issues/2026-09-03-program-collection-renewal-and-tabs.md`

- [ ] **Step 1: Update capability truth**

Document approved policy, latest respondent/expiry, new immutable successor, attributed scalar prefill, bounded stop-aware reminders, receipt-backed delivery and responsive Program sections. State that expiry changes attention, not compliance/risk. Remove “recurring schedules are not implemented” only after executable proof; retain external-adapter limitation.

- [ ] **Step 2: Update issue evidence**

For PCR-01…PCR-10 record commit, command, result and render. Keep PCR-06 `Blocked` when no real deployment adapter was exercised; fake tests do not close it.

- [ ] **Step 3: Check and commit**

```powershell
rg -n "Recurring schedules are not implemented|automatic weekly form generation|No expiry period set|Response potentially expired" README.md docs web/src
git diff --check
git add README.md docs/implementation-plan.md docs/architecture/governance-runtime.md docs/acceptance/monitoring-setup-and-risk-scoring.md docs/superpowers/issues/2026-09-03-program-collection-renewal-and-tabs.md
git commit -m "docs: record collection renewal capability"
```

Expected: obsolete claims removed/narrowed, real limitation preserved, diff check exits 0.

---

### Task 14: Run final repository verification

**Files:**
- Modify only exact files required to fix failures caused by this feature.

- [ ] **Step 1: Run focused backend gates**

```powershell
go test ./internal/monitoring ./internal/evidence ./internal/httpapi ./cmd/api ./cmd/worker -count=1
go test -tags postgres ./internal/monitoring ./internal/evidence ./internal/httpapi ./cmd/api ./cmd/worker -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full backend gates**

```powershell
go test ./... -count=1
go test -tags postgres ./... -count=1
go test -tags "postgres postgresintegration" ./... -count=1
go vet ./...
```

Expected: PASS. If `TEST_DATABASE_URL` is absent, record exact SKIPs and do not claim integration ran.

- [ ] **Step 3: Run full frontend gates**

```powershell
Set-Location web
npm test
npm run typecheck
npm run build
npm run review:ui
```

Expected: PASS without test, type, build, copy, browser-error or overflow failure.

- [ ] **Step 4: Check requirements line by line**

Confirm proof for expiry months; defaults 30/3 and range 1–5; last respondent/time; expiry; pre-expiry successor; immutable predecessor; attributed scalar prefill; file/signature exclusion; stop conditions; receipt-backed delivery; six sections; keyboard tabs; mobile/200% selector; partial failure isolation.

- [ ] **Step 5: Inspect final diff**

```powershell
Set-Location ..
git diff --check
git status --short
git diff --stat
git log --oneline -15
```

Exclude `.codex-tmp`, Office files, Playwright caches and diagnostic root screenshots. Never use `git add -A` in this worktree.

- [ ] **Step 6: Commit verification-only fixes if any**

Stage the explicit paths shown by `git status --short`, rerun their affected verification, then:

```powershell
git commit -m "test: complete collection renewal acceptance"
```

---

## Completion boundary

The implementation is complete when an authorized Program user can configure validity/reminders, see who last submitted and when, see calculated expiry/current renewal state, and use responsive accessible Program sections. A committed response schedules one idempotent successor before expiry; prior scalar answers are attributed and require confirmation; files/signatures are not copied; reminders are bounded/cancel correctly; and delivery state is supported by assignment/provider receipt.

If no real external resolver/delivery adapter is configured and exercised, internal renewal may be complete but external automatic resend remains explicitly blocked in PCR-06 and must not be described as delivered production capability.
