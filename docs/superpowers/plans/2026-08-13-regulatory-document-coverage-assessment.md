# Regulatory Document Coverage Assessment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically compare source-anchored regulatory document obligations with current Programs and actor-visible Matters, calculate defensible estimated and verified coverage, and guide reviewers through governed updates in a clean workflow.

**Architecture:** Add an isolated `documentcoverage` domain between document extraction and Continuity. It ranks candidate requirement matches with transparent signals, derives positive coverage exclusively from an exported authoritative Continuity requirement-chain evaluator, persists versioned assessments, and resolves Matter context per actor at read time. The React import workspace renders the assessment through a compact summary and progressively disclosed review sheet.

**Tech Stack:** Go 1.25+, PostgreSQL 18, pgx v5, existing durable outbox worker, React 19, TypeScript 7, Vitest, Testing Library, axe-core, CSS custom properties.

---

## File structure

### New backend files

- `internal/continuity/requirement_coverage.go`: authoritative per-requirement applicability, implementation, and evidence-chain projection.
- `internal/continuity/requirement_coverage_test.go`: current-chain boundary tests.
- `internal/documentimport/obligation.go`: deterministic structured parsing for source-backed obligation candidates.
- `internal/documentimport/obligation_test.go`: modality, actor, action, citation, date, eligibility, and deduplication tests.
- `internal/documentcoverage/model.go`: assessment, metric, match, review, suggestion, and API projection types.
- `internal/documentcoverage/matcher.go`: scope gates, token/citation normalization, ranking, and score explanations.
- `internal/documentcoverage/evaluator.go`: maps ranked candidates onto authoritative requirement coverage and generates classifications/suggestions.
- `internal/documentcoverage/repository.go`: persistence interface.
- `internal/documentcoverage/memory.go`: deterministic memory repository.
- `internal/documentcoverage/postgres.go`: PostgreSQL repository and durable recompare queue.
- `internal/documentcoverage/service.go`: processing, reads, reviews, Matter projection, staleness, and governed suggestion application.
- `internal/documentcoverage/matcher_test.go`, `evaluator_test.go`, `service_test.go`: isolated domain tests.
- `internal/documentcoverage/postgres_integration_test.go`: tenant, version, queue, and persistence tests.
- `internal/httpapi/document_coverage_handlers.go`: actor-scoped coverage endpoints.
- `internal/httpapi/document_coverage_handlers_test.go`: route, identity, version, visibility, and authority tests.
- `migrations/000029_document_coverage_assessments.up.sql`: normalized coverage storage and constraints.
- `migrations/000029_document_coverage_assessments.down.sql`: exact rollback.

### Modified backend files

- `internal/documentimport/model.go`: attach structured obligation data to proposals without breaking existing JSON.
- `internal/documentimport/analyzer.go`: populate obligation structure and deduplicate by normalized obligation fingerprint.
- `internal/httpapi/server.go`: inject the coverage service.
- `internal/httpapi/document_import_handlers.go`: start synchronous coverage in memory mode after extraction.
- `internal/httpapi/route_registry.go`: register read, review, recompare, and governed apply endpoints.
- `cmd/api/services.go`, `cmd/api/services_postgres.go`, `cmd/api/services_memory.go`, `cmd/api/main.go`: construct the coverage service and inject it into the HTTP API.
- `cmd/worker/services_postgres.go`: publish extraction first and coverage second for the same durable import event.

### New web files

- `web/src/documentCoverageTypes.ts`: typed coverage contract.
- `web/src/documentCoverageApi.ts`: load, review, recompare, and suggestion-apply calls.
- `web/src/components/DocumentCoveragePanel.tsx`: compact summary, metrics, queues, and state recovery.
- `web/src/components/CoverageReviewSheet.tsx`: focused source/match review.
- `web/src/components/DocumentCoveragePanel.test.tsx`: interaction, responsive semantics, and accessibility tests.
- `web/src/document-coverage.css`: restrained responsive coverage UI.

### Modified web files

- `web/src/components/DocumentImportWorkspace.tsx`: load/poll coverage and place it before raw proposals.
- `web/src/components/DocumentImportWorkspace.test.tsx`: end-to-end workspace states.
- `web/src/main.tsx`: import the coverage stylesheet.

## Task 1: Export authoritative requirement-chain truth

**Files:**
- Create: `internal/continuity/requirement_coverage.go`
- Create: `internal/continuity/requirement_coverage_test.go`

- [ ] **Step 1: Write failing tests for complete, missing-control, and stale-evidence chains**

Add table-driven tests that construct one `ProgramAggregate` and assert the exact public projection:

```go
func TestCurrentRequirementCoverage(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	aggregate := ProgramAggregate{
		Program: Program{ID: "program-1", Status: ProgramActive},
		Requirements: []Requirement{{ID: "req-1", Status: RequirementApproved, EffectiveFrom: now.Add(-time.Hour)}},
		Applicability: []Applicability{{RequirementID: "req-1", Status: ApplicabilityApplicable, EffectiveFrom: now.Add(-time.Hour)}},
		ControlImplementations: []ControlImplementation{{ID: "control-1", Status: ImplementationImplemented, EffectiveFrom: now.Add(-time.Hour)}},
		RequirementControlLinks: []RequirementControlLink{{RequirementID: "req-1", ImplementationID: "control-1"}},
		EvidenceContracts: []EvidenceContract{{ID: "contract-1", RequirementID: "req-1", ControlImplementationID: "control-1", Status: EvidenceContractActive, FreshnessMinutes: 60, MinimumCoverage: .9}},
		EvidenceAssessments: []EvidenceAssessment{{ContractID: "contract-1", Conclusion: EvidenceSupported, Coverage: 1, AssessedAt: now.Add(-time.Minute)}},
	}
	got := CurrentRequirementCoverage(aggregate, now)["req-1"]
	if !got.Applicable || !got.ControlImplemented || !got.EvidenceSupported || !got.Complete {
		t.Fatalf("expected complete chain, got %#v", got)
	}
}
```

Add separate assertions that removing the link makes `ControlImplemented` and `Complete` false, and moving `AssessedAt` two hours back makes `EvidenceSupported` and `Complete` false.

- [ ] **Step 2: Run the focused test and confirm the API is absent**

Run: `go test ./internal/continuity -run TestCurrentRequirementCoverage -count=1`

Expected: FAIL with `undefined: CurrentRequirementCoverage`.

- [ ] **Step 3: Implement the focused public projection by reusing Continuity's current rules**

Define:

```go
type RequirementCoverage struct {
	RequirementID       string              `json:"requirement_id"`
	Applicability       ApplicabilityStatus `json:"applicability"`
	Applicable          bool                `json:"applicable"`
	ControlImplemented  bool                `json:"control_implemented"`
	EvidenceSupported   bool                `json:"evidence_supported"`
	Complete            bool                `json:"complete"`
	ControlIDs          []string            `json:"control_ids"`
	EvidenceContractIDs []string            `json:"evidence_contract_ids"`
	Reasons             []string            `json:"reasons"`
}

func CurrentRequirementCoverage(aggregate ProgramAggregate, at time.Time) map[string]RequirementCoverage
```

Inside the function, use the existing package-private `effectiveAt`, `effectiveEvidenceContracts`, and `boundedAssessmentValidity` helpers. Select the latest effective applicability, require `ApplicabilityApplicable`, and require at least one effective `ImplementationImplemented` linked control. Evidence is supported only when at least one effective contract applies to the requirement or its implemented controls **and every such contract** has a latest assessment that is `EvidenceSupported`, meets `MinimumCoverage`, and remains before its bounded validity time. Sort identifier and reason slices before returning so snapshots hash deterministically.

- [ ] **Step 4: Run Continuity tests**

Run: `go test ./internal/continuity -run 'TestCurrentRequirementCoverage|TestDeriveProgramState' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the authoritative projection**

```bash
git add internal/continuity/requirement_coverage.go internal/continuity/requirement_coverage_test.go
git commit -m "feat(continuity): expose requirement coverage truth"
```

## Task 2: Produce structured obligation candidates

**Files:**
- Create: `internal/documentimport/obligation.go`
- Create: `internal/documentimport/obligation_test.go`
- Modify: `internal/documentimport/model.go`
- Modify: `internal/documentimport/analyzer.go`

- [ ] **Step 1: Write failing parser and analyzer tests**

Use realistic regulatory sentences and assert structure plus source provenance:

```go
func TestParseObligation(t *testing.T) {
	got := ParseObligation("A data controller must notify the Commission within 72 hours under section 41.", "REQUIREMENT_CANDIDATE")
	if !got.Eligible || got.Modality != "MUST" || got.Actor != "data controller" || got.Action != "notify" {
		t.Fatalf("unexpected obligation: %#v", got)
	}
	if !slices.Contains(got.Citations, "section 41") || !slices.Contains(got.Dates, "within 72 hours") {
		t.Fatalf("missing source structure: %#v", got)
	}
}

func TestAnalyzeBoundedAttachesObligationAndPage(t *testing.T) {
	result := AnalyzeBounded([]Section{{ID: "page-7", Page: 7, Text: "A data controller must notify the Commission within 72 hours under section 41."}}, 10)
	if result.Proposals[0].Obligation == nil || result.Proposals[0].Anchor.Page != 7 {
		t.Fatalf("expected structured page-backed obligation: %#v", result.Proposals[0])
	}
}
```

Also assert that a definition and a generic risk sentence are `Eligible=false`, and that repeated whitespace/case variants share one fingerprint.

- [ ] **Step 2: Run tests and verify they fail before the new types exist**

Run: `go test ./internal/documentimport -run 'TestParseObligation|TestAnalyzeBoundedAttachesObligation' -count=1`

Expected: FAIL with undefined parser/type errors.

- [ ] **Step 3: Add the backward-compatible JSON model**

Add to `model.go`:

```go
type Obligation struct {
	Fingerprint string   `json:"fingerprint"`
	Eligible    bool     `json:"eligible"`
	Modality    string   `json:"modality,omitempty"`
	Actor       string   `json:"actor,omitempty"`
	Action      string   `json:"action,omitempty"`
	Object      string   `json:"object,omitempty"`
	Citations   []string `json:"citations"`
	Dates       []string `json:"dates"`
	Topics      []string `json:"topics"`
	Uncertainty []string `json:"uncertainty"`
}
```

Add `Obligation *Obligation \`json:"obligation,omitempty"\`` to `Proposal` so old stored proposals continue to decode.

- [ ] **Step 4: Implement bounded deterministic parsing**

In `obligation.go`, compile regexes once for mandatory/prohibitive modals, `section|article|regulation|paragraph` citations, and bounded time expressions. Normalize Unicode whitespace and lowercase matching text, retain the original statement in `Proposal.Statement`, derive the actor from text before the modal, derive the first verb and remaining object after the modal, remove a fixed stop-word set from topics, sort/deduplicate arrays, and hash the canonical fields for `Fingerprint`.

Only `REQUIREMENT_CANDIDATE`, `DEADLINE_CANDIDATE`, and `CONTROL_EXPECTATION` are eligible. Missing actor/action adds an uncertainty reason but does not invent a value. Authority references and risk signals remain review context with `Eligible=false`.

- [ ] **Step 5: Populate obligations and deduplicate eligible candidates by fingerprint**

In `AnalyzeBounded`, call `ParseObligation(statement, kind)`, attach the pointer, and use `obligation.Fingerprint` as the duplicate key when it is non-empty. Preserve all existing proposal IDs, source anchors, limits, and statuses.

- [ ] **Step 6: Run analyzer, PDF, and resource-limit tests**

Run: `go test ./internal/documentimport -count=1`

Expected: PASS, including existing Poppler page-anchor tests.

- [ ] **Step 7: Commit structured extraction**

```bash
git add internal/documentimport/model.go internal/documentimport/analyzer.go internal/documentimport/obligation.go internal/documentimport/obligation_test.go
git commit -m "feat(imports): structure regulatory obligations"
```

## Task 3: Build explainable scope matching and coverage evaluation

**Files:**
- Create: `internal/documentcoverage/model.go`
- Create: `internal/documentcoverage/matcher.go`
- Create: `internal/documentcoverage/evaluator.go`
- Create: `internal/documentcoverage/matcher_test.go`
- Create: `internal/documentcoverage/evaluator_test.go`

- [ ] **Step 1: Define failing truth-boundary tests**

Cover these cases with exact expected classifications and counts:

```go
func TestEvaluateRejectsCrossJurisdictionKeywordOverlap(t *testing.T) {
	doc := candidate("federal-reserve", "The bank must maintain cybersecurity controls.", "US")
	program := programSnapshot("NDPA-2023", "Nigeria", "PRIVACY", completeRequirement("req-1", "Maintain cybersecurity controls"))
	got := Evaluate([]Candidate{doc}, []ProgramSnapshot{program}, fixedNow)
	if got.Metrics.EstimatedVerified.Numerator != 0 || got.Candidates[0].Classification != ClassificationGap {
		t.Fatalf("cross-jurisdiction text must not count: %#v", got)
	}
}

func TestEvaluateUsesCompleteChainOnly(t *testing.T) {
	got := Evaluate([]Candidate{candidate("ndpa", "A data controller must retain processing records.", "Nigeria")}, []ProgramSnapshot{programSnapshot("NDPA-2023", "Nigeria", "PRIVACY", completeRequirement("req-1", "Data controller must retain processing records"))}, fixedNow)
	if got.Metrics.EstimatedVerified.Numerator != 1 || got.Candidates[0].Classification != ClassificationPartialMatch {
		t.Fatalf("strong unreviewed match should estimate but not verify: %#v", got)
	}
}
```

Also test 0.85/0.55 thresholds, hard legal-entity conflicts, missing controls, stale evidence, zero denominator, open Matter absence from metrics, deterministic ordering, and visible score components.

- [ ] **Step 2: Run the package test before creating implementation files**

Run: `go test ./internal/documentcoverage -count=1`

Expected: FAIL because the package/types do not exist.

- [ ] **Step 3: Define stable domain types**

Create enums and JSON types in `model.go`:

```go
type AssessmentStatus string
const (
	AssessmentPending AssessmentStatus = "PENDING"
	AssessmentComparing AssessmentStatus = "COMPARING"
	AssessmentReady AssessmentStatus = "READY"
	AssessmentPartial AssessmentStatus = "PARTIAL"
	AssessmentFailed AssessmentStatus = "FAILED"
)

type ViewStatus string
const (
	ViewPending ViewStatus = "PENDING"
	ViewComparing ViewStatus = "COMPARING"
	ViewReady ViewStatus = "READY"
	ViewPartial ViewStatus = "PARTIAL"
	ViewFailed ViewStatus = "FAILED"
	ViewStale ViewStatus = "STALE"
)

type Classification string
const (
	ClassificationVerified Classification = "VERIFIED_COVERAGE"
	ClassificationNoEvidence Classification = "MAPPED_NO_CURRENT_EVIDENCE"
	ClassificationControlGap Classification = "MAPPED_CONTROL_GAP"
	ClassificationPartialMatch Classification = "PARTIAL_MATCH"
	ClassificationGap Classification = "GAP"
	ClassificationNeedsReview Classification = "NEEDS_REVIEW"
	ClassificationNotApplicable Classification = "NOT_APPLICABLE"
)

type CountMetric struct { Numerator int `json:"numerator"`; Denominator int `json:"denominator"` }
type Metrics struct {
	EstimatedVerified CountMetric `json:"estimated_verified"`
	Verified           CountMetric `json:"verified"`
	RequirementMapped  CountMetric `json:"requirement_mapped"`
	ControlImplemented CountMetric `json:"control_implemented"`
	EvidenceSupported  CountMetric `json:"evidence_supported"`
}
```

Define `Candidate`, `ProgramSnapshot`, `RequirementTarget`, `ScoreComponent`, `Match`, `ReviewDecision`, `Suggestion`, persisted `Assessment`, and actor-filtered `View`. `ViewStatus` may be `STALE`; persisted `AssessmentStatus` may not. Include source anchor, target IDs/versions, analyzer/matcher versions, document SHA, snapshot hash, review actor/time/reason, applied-result identifiers, and a cursor page with `items` plus `next_cursor`.

- [ ] **Step 4: Implement scope gates and weighted ranking**

`MatchCandidate` must reject explicit tenant, legal-entity, jurisdiction, regulator, regulated-party, retired-Program, and not-applicable conflicts. For eligible targets, calculate exactly:

```go
score := citationScore*0.35 + actionObjectTopicScore*0.30 + scopeScore*0.20 + cadenceThresholdScore*0.15
```

Return each named component, positive rationale, and conflicts. Normalize with Unicode lowercase, punctuation stripping, token deduplication, and a fixed regulatory stop-word set. Sort by descending score, then Program code, requirement code, and IDs. Retain at most five targets per candidate.

- [ ] **Step 5: Implement deterministic classification and metrics**

`Evaluate` must include every eligible candidate in the denominator until a reviewed `NOT_APPLICABLE` decision exists. A strong unreviewed match can enter `EstimatedVerified` only when the authoritative target chain is complete. Reviewed accepted mappings populate mapped/control/evidence metrics and become `VERIFIED_COVERAGE` only for a complete chain. Rejected/ambiguous candidates remain gaps or needs-review. Generate one suggestion per unresolved candidate using the ordered types `LINK_REQUIREMENT`, `ADD_REQUIREMENT`, `CREATE_MATTER`, `CREATE_PROGRAM`.

- [ ] **Step 6: Run domain tests and race checks**

Run: `go test -race ./internal/documentcoverage -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the matching engine**

```bash
git add internal/documentcoverage/model.go internal/documentcoverage/matcher.go internal/documentcoverage/evaluator.go internal/documentcoverage/matcher_test.go internal/documentcoverage/evaluator_test.go
git commit -m "feat(coverage): compare obligations with program truth"
```

## Task 4: Add versioned service, reviews, staleness, and actor-filtered Matter context

**Files:**
- Create: `internal/documentcoverage/repository.go`
- Create: `internal/documentcoverage/memory.go`
- Create: `internal/documentcoverage/service.go`
- Create: `internal/documentcoverage/service_test.go`

- [ ] **Step 1: Write failing service tests**

Use `documentimport.MemoryRepository`, `continuity.MemoryRepository`, and a coverage memory repository to prove:

- assessing an extracted document creates `READY` version 1;
- reviewing with an obsolete expected version returns `ErrVersionConflict`;
- accepting a complete target changes verified numerator from 0 to 1;
- a Program-version/snapshot-hash change returns `STALE` without erasing the prior result;
- two actors receive the same Program metrics but only their visible Matters;
- no Matter title or ID is stored in the shared assessment;
- failed/unsupported extraction returns a truthful non-ready view without invented candidates.

Use the public service contract:

```go
type ReviewInput struct {
	TenantID string
	DocumentID string
	ExpectedVersion int64
	ReviewerID string
	Decisions []DecisionInput
}

type ReadInput struct {
	TenantID string
	LegalEntityID string
	DocumentID string
	PrincipalID string
}
```

- [ ] **Step 2: Run the service tests and confirm the missing repository failure**

Run: `go test ./internal/documentcoverage -run 'TestService' -count=1`

Expected: FAIL with undefined service/repository symbols.

- [ ] **Step 3: Implement repository and clone-safe memory persistence**

The repository interface must expose `BeginVersion`, `CompleteVersion`, `Current`, `Review`, `MarkFailed`, and `QueueRecompare`. `BeginVersion` idempotently creates or claims the document/SHA/analyzer/matcher/snapshot tuple as `COMPARING`; `CompleteVersion` transactionally replaces its bounded candidates, matches, suggestions, and metrics and marks it `READY` or `PARTIAL`. `Review` takes an expected version, appends immutable review records, recalculates metrics, and increments the assessment version atomically. Deep-clone every nested slice and `json.RawMessage` in memory mode.

- [ ] **Step 4: Implement processing and staleness**

`Service.Process(ctx, tenant, documentID)` loads the document, exits idempotently if the same document SHA, analyzer version, matcher version, and Program snapshot hash already have a ready assessment, otherwise calls `BeginVersion`, builds the result, and calls `CompleteVersion`. A retry claims the same tuple rather than creating duplicate history. Program snapshots come from `continuity.Service.ListPrograms`; use `continuity.CurrentRequirementCoverage` for every target.

`Service.Get` recomputes only the current snapshot hash to flag staleness, loads Matters through `ListMatters`, filters with `continuity.MatterVisibleTo`, ranks visible context against the candidate, and returns it without modifying stored assessment data.

- [ ] **Step 5: Implement explicit review semantics**

Accept only `ACCEPT_MATCH`, `REJECT_MATCH`, and `NOT_APPLICABLE`. An accepted match requires a target ID from the stored ranked set. `NOT_APPLICABLE` requires a non-empty reason. Reject selections over 50 candidates. Re-evaluate classifications and metrics using the persisted target versions; if the live target version differs, return `ErrStaleAssessment` and require recompare.

- [ ] **Step 6: Run all memory service tests**

Run: `go test -race ./internal/documentcoverage -count=1`

Expected: PASS.

- [ ] **Step 7: Commit service behavior**

```bash
git add internal/documentcoverage/repository.go internal/documentcoverage/memory.go internal/documentcoverage/service.go internal/documentcoverage/service_test.go
git commit -m "feat(coverage): persist reviewed coverage assessments"
```

## Task 5: Persist assessments in PostgreSQL and run them durably after extraction

**Files:**
- Create: `migrations/000029_document_coverage_assessments.up.sql`
- Create: `migrations/000029_document_coverage_assessments.down.sql`
- Create: `internal/documentcoverage/postgres.go`
- Create: `internal/documentcoverage/postgres_integration_test.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/services_memory.go`

- [ ] **Step 1: Write the migration with tenant-safe normalized tables**

Create these tables inside one transaction:

```sql
CREATE TABLE document_coverage_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid REFERENCES legal_entities(id),
    document_id uuid NOT NULL REFERENCES document_imports(id) ON DELETE CASCADE,
    document_sha256 text NOT NULL CHECK (document_sha256 ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('PENDING','COMPARING','READY','PARTIAL','FAILED')),
    analyzer_version text NOT NULL,
    matcher_version text NOT NULL,
    scoring_policy_version text NOT NULL,
    program_snapshot_hash text NOT NULL CHECK (program_snapshot_hash ~ '^[0-9a-f]{64}$'),
    metrics jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metrics)='object'),
    limitations jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(limitations)='array'),
    failure_message text NOT NULL DEFAULT '',
    assessed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (tenant_id, document_id, document_sha256, analyzer_version, matcher_version, program_snapshot_hash)
);
```

Add `document_coverage_candidates`, `document_coverage_matches`, `document_coverage_reviews`, and `document_coverage_suggestions` with composite foreign keys that include `assessment_id` and `tenant_id`; JSON columns must have object/array checks; review rows are append-only; suggestion status is constrained to `PROPOSED`, `DISMISSED`, `APPLIED`, `FAILED`. Add tenant/document, assessment/ordinal, target, and current-suggestion indexes. The down migration drops the five tables in dependency order inside one transaction.

- [ ] **Step 2: Add PostgreSQL integration tests before the repository exists**

Test assessment round-trip, immutable review history, optimistic conflict, tenant isolation, cascade cleanup, duplicate processing idempotence, and an outbox row for `DocumentCoverageComparisonRequested`. Use the repository package's existing `TEST_DATABASE_URL` convention and cleanup only the test tenant.

- [ ] **Step 3: Run the integration test and confirm it fails**

Run: `go test -tags 'postgres postgresintegration' ./internal/documentcoverage -run TestPostgres -count=1`

Expected: FAIL because `NewPostgresRepository` is undefined or migration 29 is unapplied.

- [ ] **Step 4: Implement transactional PostgreSQL persistence**

`BeginVersion` inserts or claims the unique assessment tuple as `COMPARING`. `CompleteVersion` inserts candidates, bounded matches, and suggestions and marks the assessment terminal in one transaction. `Review` locks the current assessment `FOR UPDATE`, verifies version and tenant, appends review rows, updates candidate dispositions and metrics, and increments the version. `QueueRecompare` inserts an outbox event with aggregate type `DOCUMENT_IMPORT` and event type `DocumentCoverageComparisonRequested`; use the existing outbox retry/dead-letter mechanism.

- [ ] **Step 5: Wire durable publisher ordering**

Construct one coverage service with the document and Continuity services. In the worker publisher list, keep `documentService` immediately before `coverageService`:

```go
publisher := workflowruntime.NewCompositePublisher(
	 sourceHealth, actionWork, lifecycleWork, escalationWork,
	 documentService,
	 coverageService,
	 workflowruntime.LogPublisher{Logger: logger},
)
```

`coverageService.Publish` handles both the existing `DocumentImportProcessingRequested` event and `DocumentCoverageComparisonRequested`. On the first event, the preceding publisher has already made extraction durable. If coverage fails, the outbox retries; document processing remains idempotent and coverage retries independently.

Wire the same PostgreSQL coverage repository/service into `serviceSet` and the API. In memory API mode use the memory coverage repository; Task 6 starts processing from the import handler after synchronous extraction.

- [ ] **Step 6: Apply migrations locally and run repository/worker tests**

Run:

```bash
psql $env:TEST_DATABASE_URL -v ON_ERROR_STOP=1 -f migrations/000029_document_coverage_assessments.up.sql
go test -tags postgres ./cmd/api ./cmd/worker ./internal/documentcoverage -count=1
go test -tags 'postgres postgresintegration' ./internal/documentcoverage -count=1
```

Expected: all commands PASS.

- [ ] **Step 7: Verify rollback and reapply**

Run:

```bash
psql $env:TEST_DATABASE_URL -v ON_ERROR_STOP=1 -f migrations/000029_document_coverage_assessments.down.sql
psql $env:TEST_DATABASE_URL -v ON_ERROR_STOP=1 -f migrations/000029_document_coverage_assessments.up.sql
```

Expected: both commands exit 0.

- [ ] **Step 8: Commit durable storage and wiring**

```bash
git add migrations/000029_document_coverage_assessments.* internal/documentcoverage/postgres.go internal/documentcoverage/postgres_integration_test.go cmd/api/services.go cmd/api/services_postgres.go cmd/api/services_memory.go cmd/worker/services_postgres.go
git commit -m "feat(coverage): process assessments durably"
```

## Task 6: Expose actor-scoped review and governed suggestion endpoints

**Files:**
- Create: `internal/httpapi/document_coverage_handlers.go`
- Create: `internal/httpapi/document_coverage_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/document_import_handlers.go`
- Modify: `internal/httpapi/route_registry.go`
- Modify: `cmd/api/main.go`
- Modify: `internal/documentcoverage/service.go`
- Modify: `internal/documentcoverage/service_test.go`

- [ ] **Step 1: Write failing HTTP contract and security tests**

Prove these exact outcomes:

- unauthenticated reads return 401;
- cross-tenant query/path access returns 403 or tenant-scoped 404;
- a ready assessment returns metrics, source anchors, and only visible Matter context;
- more than 25 findings return a stable `next_cursor` and the next request neither duplicates nor skips a candidate;
- a document still extracting with no assessment returns a synthesized `PENDING` view instead of a generic 404;
- stale assessment returns 200 with `status: STALE` and prior metrics;
- obsolete review version returns 409;
- missing not-applicable reason returns 422;
- recompare returns 202;
- suggestion apply without material authority returns the command guard's denial;
- applied `CREATE_MATTER` suggestion creates a draft/triage Matter linked to the target Program and records the resulting Matter ID;
- restricted Matter data is absent from both JSON and serialized response bytes.

- [ ] **Step 2: Run the focused handler tests**

Run: `go test ./internal/httpapi -run DocumentCoverage -count=1`

Expected: FAIL because handlers and routes are missing.

- [ ] **Step 3: Register explicit routes and dependency**

Add `Coverage *documentcoverage.Service` to `Dependencies` and register:

```go
read("/api/v1/document-imports/{id}/coverage", a.getDocumentCoverage),
write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/review", a.reviewDocumentCoverage, nil),
write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/recompare", a.recompareDocumentCoverage, nil),
write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/suggestions/{suggestion_id}/apply", a.applyDocumentCoverageSuggestion, nil),
```

- [ ] **Step 4: Implement handlers with stable error semantics**

Bind tenant, legal entity, and principal exclusively from `identity.Require`. Inject `services.Coverage` into `httpapi.Dependencies` in `cmd/api/main.go`. Decode review decisions with a maximum of 50. Bound reads to 25 findings by default and 100 maximum; use an opaque encoded `(ordinal,candidate_id)` cursor with deterministic ordering. Map an unknown document to 404, a known still-processing document with no assessment to a synthesized `PENDING` view, invalid decisions to 422, version/stale conflict to 409, accepted recompare to 202, authority denial through the existing material command wrapper, and internal failures to a non-sensitive 500 message. After a memory-mode import returns already extracted, call `Coverage.Process`; PostgreSQL imports remain pending and are handled by the worker.

- [ ] **Step 5: Implement governed suggestion application**

Load the suggestion first, construct its target command payload, and dispatch the execution through `a.command` with the **existing** command name and policy. Use `program.requirement.add` for `ADD_REQUIREMENT`, `matter.create` for `CREATE_MATTER`, and `program.create` for `CREATE_PROGRAM`; `LINK_REQUIREMENT` delegates to coverage review and makes no Continuity mutation. Keep the route itself an authenticated write because the material command is selected only after the server has loaded the immutable suggestion type. Add a regression test proving every mutating suggestion passes through the guard and no branch calls Continuity directly before authorization.

Support these typed operations:

- `LINK_REQUIREMENT`: records an accepted coverage mapping; no Continuity mutation.
- `ADD_REQUIREMENT`: calls `continuity.Service.AddRequirement` with `RequirementDraft`, document ID as source ID, exact page anchor, and current Program expected version.
- `CREATE_MATTER`: calls `continuity.Service.CreateMatter` with `MatterRegulatoryChange` or `MatterControlGap`, document/candidate source IDs, source quote in known facts, and the selected Program/requirement/control link.
- `CREATE_PROGRAM`: calls `continuity.Service.CreateProgram`; it always produces `ProgramDraft`.

The material route binds the actor; the service re-loads the current suggestion and target versions immediately before the command. Persist `APPLIED` plus resulting object type/ID on success. Persist `FAILED` with a bounded non-secret recovery message on command failure. Never activate a Program or approve a requirement automatically.

- [ ] **Step 6: Run authorization, visibility, and route registry tests**

Run: `go test ./internal/httpapi ./internal/documentcoverage -run 'DocumentCoverage|Route' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the API slice**

```bash
git add internal/httpapi/document_coverage_handlers.go internal/httpapi/document_coverage_handlers_test.go internal/httpapi/server.go internal/httpapi/document_import_handlers.go internal/httpapi/route_registry.go internal/documentcoverage/service.go internal/documentcoverage/service_test.go cmd/api/main.go
git commit -m "feat(api): review document coverage safely"
```

## Task 7: Add typed browser API contracts

**Files:**
- Create: `web/src/documentCoverageTypes.ts`
- Create: `web/src/documentCoverageApi.ts`

- [ ] **Step 1: Define the exact TypeScript contract**

Mirror backend enums and expose discriminated types:

```ts
export type CoverageStatus = "PENDING" | "COMPARING" | "READY" | "PARTIAL" | "FAILED" | "STALE";
export type CoverageClassification = "VERIFIED_COVERAGE" | "MAPPED_NO_CURRENT_EVIDENCE" | "MAPPED_CONTROL_GAP" | "PARTIAL_MATCH" | "GAP" | "NEEDS_REVIEW" | "NOT_APPLICABLE";
export type CountMetric = { numerator: number; denominator: number };
export type CoverageMetrics = {
  estimated_verified: CountMetric;
  verified: CountMetric;
  requirement_mapped: CountMetric;
  control_implemented: CountMetric;
  evidence_supported: CountMetric;
};
export type CoverageReviewDecision =
  | { candidate_id: string; decision: "ACCEPT_MATCH"; match_id: string; reason?: string }
  | { candidate_id: string; decision: "REJECT_MATCH"; reason?: string }
  | { candidate_id: string; decision: "NOT_APPLICABLE"; reason: string };
```

Define source anchor, score component, match, candidate, Matter context, suggestion, assessment version/freshness, review progress, limitations, and `next_cursor`. Do not use `any`.

- [ ] **Step 2: Implement four request functions**

```ts
export const loadDocumentCoverage = (documentID: string, cursor = "") => {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return requestJSON<DocumentCoverage>(apiBase, `/api/v1/document-imports/${encodeURIComponent(documentID)}/coverage${query}`);
};

export const reviewDocumentCoverage = (documentID: string, expectedVersion: number, decisions: CoverageReviewDecision[]) =>
  requestJSON<DocumentCoverage>(apiBase, `/api/v1/document-imports/${encodeURIComponent(documentID)}/coverage/review`, {
    method: "POST", body: JSON.stringify({ expected_version: expectedVersion, decisions }),
  });
```

Add `recompareDocumentCoverage` using POST and `applyCoverageSuggestion` using the suggestion apply path.

- [ ] **Step 3: Run TypeScript typecheck**

Run: `npm run typecheck --prefix web`

Expected: PASS.

- [ ] **Step 4: Commit the client contract**

```bash
git add web/src/documentCoverageTypes.ts web/src/documentCoverageApi.ts
git commit -m "feat(web): type document coverage API"
```

## Task 8: Build the compact progressive review experience

**Files:**
- Create: `web/src/components/DocumentCoveragePanel.tsx`
- Create: `web/src/components/CoverageReviewSheet.tsx`
- Create: `web/src/components/DocumentCoveragePanel.test.tsx`
- Create: `web/src/document-coverage.css`
- Modify: `web/src/components/DocumentImportWorkspace.tsx`
- Modify: `web/src/components/DocumentImportWorkspace.test.tsx`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write failing component tests for the approved hierarchy**

Mock the typed API and assert:

- the heading says `Estimated coverage` before review and `Verified document coverage` after review;
- every metric displays numerator and denominator;
- the default queue is `Needs attention` when gaps exist;
- raw scoring is absent until `Why this match?` expands;
- opening a finding shows exact quote and `Page 7`;
- accepting a match posts the current assessment version;
- only strong conflict-free matches can be selected for bounded bulk acceptance;
- stale results retain prior metrics and make `Refresh comparison` the only primary action;
- zero denominator says `Not enough information` and renders no percentage;
- failed comparison has a retry and never says non-compliant;
- axe reports no WCAG A/AA violations with contrast disabled only in jsdom;
- switching selected documents discards coverage state from the prior document.

- [ ] **Step 2: Run the component tests before implementation**

Run: `npm test --prefix web -- DocumentCoveragePanel.test.tsx`

Expected: FAIL because the components do not exist.

- [ ] **Step 3: Implement the compact summary and action queues**

`DocumentCoveragePanel` must render one headline card, three supporting metrics, three labelled queue buttons (`Covered`, `Needs attention`, `Needs review`), and one primary action. Format percentages as `Math.round(numerator / denominator * 100)` only when the denominator is positive. Always render `numerator / denominator` beside the label. Use `<progress>` plus visible text; do not use a donut chart or color alone. When `next_cursor` exists, show a secondary `Load more findings` action that appends the next page without changing the active queue or scroll position.

Processing copy follows the durable stages: `Extracting document`, `Comparing with Programs`, `Ready for review`. Preserve the last durable result during polling and announce status through `aria-live="polite"`.

- [ ] **Step 4: Implement the focused review sheet**

Use the existing `FocusedSheet` interaction pattern. On desktop, show the exact source quote/page and proposed Program chain in two columns; below 760px stack them. The source header includes a `View in document` action that expands the existing extracted section and moves focus to the anchored page text. Put `Why this match?` in a native `<details>`. Render one recommended decision first, alternative actions as secondary buttons, and require a reason input only for not-applicable. Restore focus to the triggering row when the sheet closes.

- [ ] **Step 5: Integrate coverage before raw proposal details**

In `DocumentImportWorkspace`, load coverage after the selected document reaches extracted terminal state, poll while coverage is `PENDING` or `COMPARING`, and pass recoverable errors to the panel. Place the panel before limitations, generic proposal cards, and extracted sections. Keep raw proposals and source reconstruction under collapsed secondary disclosure so expert traceability remains available without clutter.

- [ ] **Step 6: Apply restrained responsive styling**

Use existing semantic variables, an 8px spacing rhythm, neutral surfaces, blue/cyan primary focus, text plus icons for status, 44px minimum controls, visible focus rings, and 150-200ms opacity/border transitions. Add `prefers-reduced-motion` behavior. At 375px use one column and a full-width review sheet; at 1024px use the bounded two-column evidence/match layout. No decorative gradients, metric pulses, emoji icons, or horizontal scrolling.

- [ ] **Step 7: Run UI tests, accessibility checks, typecheck, and build**

Run:

```bash
npm test --prefix web -- DocumentCoveragePanel.test.tsx DocumentImportWorkspace.test.tsx Accessibility.test.tsx
npm run typecheck --prefix web
npm run build --prefix web
```

Expected: all commands PASS.

- [ ] **Step 8: Commit the user experience**

```bash
git add web/src/components/DocumentCoveragePanel.tsx web/src/components/CoverageReviewSheet.tsx web/src/components/DocumentCoveragePanel.test.tsx web/src/components/DocumentImportWorkspace.tsx web/src/components/DocumentImportWorkspace.test.tsx web/src/document-coverage.css web/src/main.tsx
git commit -m "feat(web): guide regulatory coverage review"
```

## Task 9: Prove accuracy, security, deployment, and the live demo workflow

**Files:**
- Modify: `internal/documentcoverage/evaluator_test.go`
- Modify: `internal/httpapi/document_coverage_handlers_test.go`
- Modify: `docs/product/continuous-compliance-and-autonomy.md`

- [ ] **Step 1: Add a fixed regulatory accuracy corpus**

Add table-driven fixtures for:

1. an NDPC/GAID obligation that should strongly rank the seeded `NDPA-2023` requirement;
2. an NDPC obligation with a requirement match but missing current evidence;
3. an NDPC obligation with no requirement, producing an add-requirement or Matter suggestion;
4. Federal Reserve cybersecurity text against only the Nigeria privacy Program, producing no match;
5. Bank of England operational-resilience text against only the Nigeria privacy Program, producing no match;
6. a descriptive passage that is not an eligible obligation.

Assert exact classification, suggestion type, match threshold band, metric numerator/denominator, and positive source page for every fixture.

- [ ] **Step 2: Add privacy regression assertions**

Serialize an assessment as two actors. Assert equal metric JSON, absence of restricted Matter ID/title for the unauthorized actor, presence for the authorized actor, and no difference in candidate count or denominator.

- [ ] **Step 3: Document product semantics**

Update `docs/product/continuous-compliance-and-autonomy.md` with the four metric definitions, estimated-versus-verified distinction, reviewed not-applicable denominator rule, open-Matter non-credit rule, and the exact statement that document coverage is decision support rather than legal compliance.

- [ ] **Step 4: Run the complete verification matrix**

Run:

```powershell
gofmt -w (rg --files cmd internal -g '*.go')
go test -race ./...
go test -tags postgres ./...
go vet ./...
npm test --prefix web
npm run typecheck --prefix web
npm run build --prefix web
python deploy/tests/deployment_config_test.py
git diff --check
```

Expected: every command PASS and `git diff --check` prints nothing.

- [ ] **Step 5: Commit the accuracy corpus and semantics**

```bash
git add internal/documentcoverage/evaluator_test.go internal/httpapi/document_coverage_handlers_test.go docs/product/continuous-compliance-and-autonomy.md
git commit -m "test(coverage): prove regulatory comparison boundaries"
```

- [ ] **Step 6: Push main and monitor CI/deployment**

Run:

```bash
git push origin main
gh run watch --exit-status "$(gh run list --workflow CI --branch main --limit 1 --json databaseId --jq '.[0].databaseId')"
gh run watch --exit-status "$(gh run list --workflow deploy-demo.yml --branch main --limit 1 --json databaseId --jq '.[0].databaseId')"
```

Expected: CI and demo deployment complete successfully for the pushed SHA.

- [ ] **Step 7: Validate official documents end to end on the deployed demo**

Use official regulator-domain URLs discovered at execution time. Download one searchable NDPC/GAID PDF, the previously validated Federal Reserve PDF, and the Bank of England PDF; verify `%PDF-`, MIME type, size below 20 MiB, and SHA-256 before upload. Sign in through `/api/v1/demo/login`, upload through `/api/v1/document-imports`, poll the returned document ID, then poll `/coverage`.

Expected live outcomes:

- all three uploads reach `EXTRACTED` with positive page anchors;
- the NDPC/GAID document produces plausible partial coverage against `NDPA-2023`, with at least one mapped requirement and truthful control/evidence state;
- representative accepted mappings recalculate verified counts from 0 to the accepted complete-chain count;
- at least one actionable existing-Program or Matter suggestion is prepared but not automatically applied;
- Federal Reserve and Bank of England samples show zero Nigeria privacy verified coverage and no strong cross-jurisdiction match;
- two demo roles see identical Program metrics while Matter context/actions respect visibility and authority;
- the default UI remains compact and account switching still unmounts actor-scoped state.

- [ ] **Step 8: Record live evidence and final state**

Capture the deployed SHA, CI run URL, deploy run URL, source URLs, uploaded document IDs, file digests, extraction section/proposal counts, coverage numerators/denominators, suggestion types, and role-visibility results in the task handoff. If any expected outcome fails, keep the task open, add a focused regression test, fix the defect, rerun the full matrix, redeploy, and repeat the live validation.
