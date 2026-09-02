# Oversight History and Data Completeness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Seed reconstructable, realistically timed reference Matter histories through canonical commands and make oversight expose truthful populations, freshness, cycle ranges, SLA, reassignment, return, and reopen context.

**Architecture:** Extend the non-production bank reference installer with a deterministic timeline factory that creates and transitions Matters through the existing continuity service over the same repository. Seed five comparable completed vendor reviews plus varied additional categories, using normal action, assignment, decision, verification, reopen, and closure commands. Enhance the bounded PostgreSQL oversight projection to derive missing history measures from authoritative rows and append-only events; never insert snapshot metrics or pre-closed Matter rows.

**Tech Stack:** Go 1.26, pgx/PostgreSQL 18, existing continuity event store/projections, React 19, TypeScript 7, Vitest 4.

---

## File map

- Create `internal/bankverticals/oversight_history.go` and `internal/bankverticals/oversight_history_test.go`.
- Modify `internal/bankverticals/service.go`, `install_service.go`, `model.go`, and `install_test.go` for deterministic timeline composition and idempotent verification.
- Modify `cmd/seed-bank-reference/main.go` to provide clocked continuity services over one PostgreSQL repository and run both continuity and oversight maintainers.
- Modify `cmd/api/services_memory.go` only to install reference history through the same domain service when explicit demo/reference fixtures are requested.
- Modify `internal/oversight/model.go`, `postgres.go`, `continuity_projection.go`, and focused tests for complete history measures.
- Extend `internal/oversight/postgres_integration_test.go` with seed-to-snapshot acceptance.
- Modify `web/src/oversightApi.ts`, `web/src/components/oversight/OversightWorkspace.tsx`, its tests, and `web/src/oversight.css` for history/freshness presentation.
- Update `docs/quality/acceptance-tests.md` and `.github/workflows/ci.yml`.

### Task 1: Define the reference-history cohort and deterministic timeline contract

- [ ] **Step 1: Write failing installer tests**

Create `internal/bankverticals/oversight_history_test.go`. Use one memory continuity repository and a factory that returns `continuity.NewServiceWithClock(repo, func() time.Time { return at })` for each timeline instant.

```go
func TestInstallSampleCreatesReconstructableOversightHistory(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	repo := continuity.NewMemoryRepository()
	base := continuity.NewServiceWithClock(repo, func() time.Time { return now })
	service := NewService(base, referenceEvidence(t, now))
	service.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
		return continuity.NewServiceWithClock(repo, func() time.Time { return at })
	})
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now

	if _, err := service.InstallSample(t.Context(), config); err != nil { t.Fatal(err) }
	matters, err := base.ListMatters(continuity.WithTrustedSystemEntityScope(t.Context(), config.TenantID, config.LegalEntityID), config.TenantID, "", 200)
	if err != nil { t.Fatal(err) }
	assertReferenceHistoryCohort(t, matters, now)
}
```

The assertion must require:

- at least five `VENDOR_REVIEW` Matters closed in the last 90 days so resolution estimates meet the current minimum;
- at least two additional closed Matter types;
- more than one owner and a reviewer distinct from each action owner;
- at least one on-time, one overdue, one reassigned, one returned decision, and one reopened Matter;
- non-zero and varied `created_at`, `implemented_at`, `observed_at`, and `closed_at` intervals;
- `scope.sample=true`, `scope.reference_data=true`, stable journey code, and operationally plausible titles.

- [ ] **Step 2: Run the focused test and confirm failure**

Run: `go test ./internal/bankverticals -run 'OversightHistory' -count=1`

Expected: compilation fails because reference timeline configuration and cohort installer do not exist.

- [ ] **Step 3: Add the non-production timeline factory**

In `internal/bankverticals/service.go` add:

```go
type ReferenceTimelineFactory func(time.Time) *continuity.Service

func (s *Service) ConfigureReferenceTimeline(factory ReferenceTimelineFactory) {
	s.referenceTimeline = factory
}
```

Reject history installation when the factory returns nil. The factory is configured only by the reference installer and deterministic memory fixture composition; it is never exposed through HTTP and never changes the production API service clock.

- [ ] **Step 4: Define stable history specifications**

In `oversight_history.go`, define immutable specs rather than metric values:

```go
type referenceMatterHistory struct {
	Key, Title, Summary string
	Type continuity.MatterType
	Priority int
	OwnerID, ActionOwnerID string
	OpenedBefore, WorkStartedAfter, ImplementedAfter, VerifiedAfter, ClosedAfter time.Duration
	DueAfter time.Duration
	ReassignTo string
	ReturnDecision, Reopen bool
}
```

Use five vendor-review durations such as 30h, 42h, 55h, 74h, and 96h plus a control-gap and an exception history. These are event timings, not precomputed metrics. At least one due date must precede close, and at least one must follow close.

- [ ] **Step 5: Commit the contract slice**

```bash
git add internal/bankverticals
git commit -m "test: define reference oversight history"
```

### Task 2: Seed complete Matter histories through canonical commands

- [ ] **Step 1: Implement one history using only continuity services**

For each spec, lookup by stable `TriggerKey` first. When absent, call the service returned by the timeline factory at each event instant:

```go
matter, err := at(openedAt).CreateMatter(ctx, continuity.CreateMatterInput{
	TenantID: config.TenantID, LegalEntityID: config.LegalEntityID,
	Type: spec.Type, Priority: spec.Priority, Title: spec.Title, Summary: spec.Summary,
	Scope: mustJSON(map[string]any{"sample": true, "reference_data": true, "journey_code": "OVERSIGHT_HISTORY", "history_key": spec.Key}),
	TriggerType: "REFERENCE_HISTORY", TriggerKey: "reference:oversight:" + spec.Key,
	KnownFacts: mustJSON(map[string]any{"source": "ClearSight reference history", "as_of": config.Now.Format("2006-01-02")}),
	MissingFacts: mustJSON([]string{}), Contradictions: mustJSON([]string{}),
	OwnerPrincipalID: spec.OwnerID, RequiredAuthority: "REVIEWER", DueAt: timePointer(openedAt.Add(spec.DueAfter)), ActorID: config.ActorID,
})
```

Then use `TransitionMatter`, `AddDecision`/`RecordDecisionLifecycle`, `AddAction`, `AssignMatter` or `AssignAction`, `TransitionAction`, `AddVerificationContract`, `RecordVerificationResult`, and `TransitionMatter(MatterClosed)` in valid order. The outcome reviewer must match the contract authority and differ from the action owner.

- [ ] **Step 2: Add returned-decision and reassignment histories**

For the return case, record `DecisionProposed`, `DecisionInReview`, `DecisionReturned`, a new `DecisionProposed`, and `DecisionApproved` with distinct actors and rationales. For reassignment, call `AssignMatter` with `ReassignmentBasis: "Planned reference-data absence handoff"`; never overwrite owner columns directly.

- [ ] **Step 3: Add the reopened history**

After a valid closure, call `TransitionMatter` from `CLOSED` to `ASSESSMENT` with a recorded rationale, proceed through `VERIFICATION`, record a new current verification result if the contract/result freshness rules require it, and close again. Assert `reopen_count == 1`.

- [ ] **Step 4: Install history from `InstallSample`**

Call `ensureOversightHistory` after the existing regulatory, authority-request, and finding journeys. If no reference timeline factory is configured, return a clear installation error rather than silently omitting history when the installer claims completeness.

- [ ] **Step 5: Run memory installer tests**

Run:

```bash
go test ./internal/bankverticals -run 'InstallSample|OversightHistory' -count=1
go test ./internal/continuity -run 'MatterAt|Closure|Owner|Decision' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the canonical history seed**

```bash
git add internal/bankverticals cmd/api/services_memory.go
git commit -m "feat: seed canonical oversight history"
```

### Task 3: Make repeated installation verify rather than skip

- [ ] **Step 1: Add failing partial-history recovery tests**

Test these interrupted states separately:

- Matter created but still in triage;
- action created but not implemented;
- verification contract exists without a result;
- first closure exists but required reopen is absent;
- expected owner assignment event is absent;
- same trigger key belongs to a non-reference Matter.

The first five must resume through canonical commands. The collision must fail closed.

- [ ] **Step 2: Implement full-history verification**

When `MatterByTriggerKey` finds an existing reference record, validate scope markers, type, title, owner route, expected action origin key, decision lifecycle, active verification contract, independent current pass, closure, reopen count, and event timing order. Do not report `already seeded` merely because the final status is `CLOSED`.

- [ ] **Step 3: Prove idempotency**

Call `InstallSample` twice and compare counts of Matters, actions, contracts, results, decisions, matter events, and outbox events. No count may change on the complete second run.

```go
before := referenceHistoryCounts(t, repo)
if _, err := service.InstallSample(t.Context(), config); err != nil { t.Fatal(err) }
after := referenceHistoryCounts(t, repo)
if !reflect.DeepEqual(before, after) { t.Fatalf("second install changed history: before=%#v after=%#v", before, after) }
```

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/bankverticals -run 'OversightHistory|InstallSample' -count=1
git add internal/bankverticals
git commit -m "fix: verify complete reference history"
```

### Task 4: Populate missing oversight history measures

- [ ] **Step 1: Add failing projection tests**

Extend `internal/oversight/postgres_integration_test.go` to seed the reference cohort, run `oversight.Maintainer`, and assert:

```go
if len(snapshot.Estimates) == 0 || snapshot.Estimates[0].SampleSize < 5 { t.Fatalf("estimates = %#v", snapshot.Estimates) }
if snapshot.Performance[0].MeasurementSamples == 0 { t.Fatalf("performance = %#v", snapshot.Performance) }
if snapshot.Performance[0].Reassigned == nil || *snapshot.Performance[0].Reassigned == 0 { t.Fatalf("reassigned = %#v", snapshot.Performance[0]) }
if snapshot.Performance[0].Returned == nil || *snapshot.Performance[0].Returned == 0 { t.Fatalf("returned = %#v", snapshot.Performance[0]) }
```

Also assert `Coverage.Population`, `Excluded`, `Unknown`, `GeneratedAt`, `ProjectionVersion`, and the `matters`, `actions`, `workflow_tasks`, `verification_results`, and `continuity_events` high-water marks.

- [ ] **Step 2: Extend the model without inventing a composite score**

Add `Returned *int 'json:"returned,omitempty"'` to `oversight.Performance`. Retain `Reassigned`; do not add an employee rank, rating, trend adjective, or blended performance score.

- [ ] **Step 3: Derive reassignment and return counts from append-only history**

In `internal/oversight/postgres.go`, extend the owner-performance query with bounded CTEs over included Matters in the selected legal entity/90-day period:

```sql
reassignments AS (
  SELECT m.owner_principal_id,count(*) reassigned
  FROM matters m JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid
    AND e.event_type='MATTER_OWNER_CHANGED' AND e.occurred_at>=$3
  GROUP BY m.owner_principal_id
), returns AS (
  SELECT m.owner_principal_id,count(*) returned
  FROM matters m JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid
    AND e.event_type='DECISION_ADDED' AND e.occurred_at>=$3
    AND e.payload->>'status'='RETURNED'
  GROUP BY m.owner_principal_id
)
```

Confirm the actual `continuity_events` payload shape and adjust the JSON path in the test-driven implementation. Keep tenant/legal-entity/scope filters on every CTE.

- [ ] **Step 4: Add matter-event high-water and sparse suppression tests**

Add `continuity_events` to `SourceHighWater`. Preserve the existing `HAVING count(*)>=5` estimate rule. Test a four-item category and assert no range is returned, rather than a persuasive fallback.

- [ ] **Step 5: Keep memory fixtures honest**

Update `FromMatterAggregates` to calculate only measures available from aggregates: completion counts and duration estimates from actual `CreatedAt`/`ClosedAt`. Leave return/reassignment unknown when event history is not supplied; do not set them to zero.

- [ ] **Step 6: Run projection tests and commit**

```bash
go test ./internal/oversight -count=1
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/oversight
git add internal/oversight
git commit -m "feat: complete oversight history measures"
```

### Task 5: Compose the reference CLI and prove seed-to-snapshot acceptance

- [ ] **Step 1: Configure the timeline factory in the CLI**

In `cmd/seed-bank-reference/main.go`, retain one `continuity.NewPostgresRepository(pool)` and configure:

```go
installer.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
	service := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return at.UTC() })
	service.ConfigureEvidenceSourceValidator(evidenceService)
	return service
})
```

Use a deterministic `seed.Now` captured once for the whole invocation. Do not read `time.Now()` separately for each history command.

- [ ] **Step 2: Run both maintainers after authoritative installation**

After continuity projection maintenance, construct `oversight.NewPostgresRepository(pool)` and run `oversight.Maintainer` at the seed instant plus one bounded refresh slot. Then load the exact legal-entity snapshot and validate the expected reference population, estimate sample, and high-water values before printing success.

- [ ] **Step 3: Keep CLI output redacted and operational**

Output IDs, counts, timestamps, journey codes, snapshot version, coverage, and sample sizes. Do not output directory contacts, invitation data, evidence contents, or secure URLs.

- [ ] **Step 4: Add PostgreSQL CLI-composition acceptance**

Add a tagged test under `cmd/seed-bank-reference` or `internal/bankverticals` that applies migrations, installs twice, runs maintainers, and reconstructs every reference Matter with `MatterAt` and event history. Assert no direct insert path exists in `oversight_history.go` by architecture test: it may not contain `INSERT INTO matters`, `INSERT INTO oversight_snapshots`, or `UPDATE matters`.

- [ ] **Step 5: Run and commit**

```bash
go test -tags postgres ./cmd/seed-bank-reference
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/bankverticals ./internal/oversight
git add cmd/seed-bank-reference internal/bankverticals internal/oversight
git commit -m "feat: install and verify oversight reference history"
```

### Task 6: Present completeness and history context in oversight UI

- [ ] **Step 1: Add failing component/API tests**

Extend `web/src/oversightApi.ts` with optional `returned`. In `OversightWorkspace.test.tsx`, assert:

- coverage explicitly names population, exclusions, and unknowns;
- freshness names generated time and projection version;
- source high-water details are accessible from a `Data freshness` disclosure;
- estimates show sample size and range without a promised date;
- sparse categories show the existing minimum-five empty state;
- performance shows current load, completed/measured, cycle time, SLA, blocked, reopened, reassigned, and returned facts;
- no row is described as a rank or employee score;
- intervention actions open the exact Matter ID.

- [ ] **Step 2: Run and confirm failure**

Run: `cd web && npm test -- OversightWorkspace.test.tsx oversightApi.test.ts`

- [ ] **Step 3: Implement the minimal UI changes**

Add a compact freshness/coverage disclosure and rename the final performance column to `Workflow history`. Render unknown optional values as `Unknown`, never zero. Keep the explanatory copy: measures describe work conditions and recorded delivery history, not employee ranking.

- [ ] **Step 4: Verify rendered states**

Run:

```bash
cd web
npm test -- OversightWorkspace.test.tsx oversightApi.test.ts copyQuality.test.ts
npm run typecheck
npm run check:ui-contracts
npm run review:ui
```

Inspect populated, sparse, stale, empty, and unavailable states at desktop/mobile, light/dark, keyboard, and 200% reflow. Fix the highest-impact issue and re-render.

- [ ] **Step 5: Commit the UI slice**

```bash
git add web DESIGN.md
git commit -m "fix: expose oversight data completeness"
```

### Task 7: Add the reference-history acceptance gate to CI

- [ ] **Step 1: Add an explicit CI step after migrations**

Run the seed-to-snapshot tagged test in the serialized PostgreSQL acceptance section. The test must verify at least five comparable completed Matters, idempotency, history reconstruction, sparse suppression, and exact projection metadata.

- [ ] **Step 2: Run the full gates**

```bash
gofmt -w internal/bankverticals internal/oversight cmd/seed-bank-reference
go test -race ./...
go test -tags postgres ./...
go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/...
go vet ./...
cd web
npm run typecheck
npm test
npm run build
npm run review:ui
```

- [ ] **Step 3: Update acceptance documentation**

Document the exact cohort definition, minimum sample, selected period, exclusions, high-water sources, repeat-run behavior, and the fact that reference history is labelled sample data and not evidence that the connected bank is compliant.

- [ ] **Step 4: Commit CI/documentation**

```bash
git add .github/workflows/ci.yml docs/quality/acceptance-tests.md docs/architecture/operational-read-models.md
git commit -m "ci: verify oversight history completeness"
```

### Task 8: Install and verify hosted reference history

- [ ] **Step 1: Run installer before the new release is marked current**

On the non-production host, run the exact deployed `seed-bank-reference` binary with existing protected tenant/legal-entity/principal IDs. Do not pass or print email or SMTP settings.

- [ ] **Step 2: Verify stored records and projection**

Confirm the output reports complete idempotent history, then use bounded SQL/API reads to verify Matter counts, event counts, closure intervals, projection version, coverage, high-water marks, sample sizes, and ranges.

- [ ] **Step 3: Verify the hosted UI and drilldowns**

Open Oversight as CRO/GRC Administrator. Confirm populated history measures, sparse suppression, freshness, and exact Matter drilldowns reconstruct the displayed timelines. Re-run as a user without oversight permission and confirm the API fails closed.

- [ ] **Step 4: Record the exact remainder**

Record any unavailable representative production volume, load/performance proof, object scanning, or broader historical cohort limitations. Do not imply reference records represent bank performance.
