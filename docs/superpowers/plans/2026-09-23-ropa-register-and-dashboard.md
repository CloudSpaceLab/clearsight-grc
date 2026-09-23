# ROPA Register and Dashboard — Tranche 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a central register of processing activities with a projection-backed dashboard, satisfying Fidelity Bank Archer requirements #29 and the register foundation of #28.

**Architecture:** New `internal/ropa` Go package with its own `ProcessingActivity` aggregate, current rows plus immutable revisions and append-only events, and a separate summary projection that serves every dashboard tile. The dashboard never aggregates live; the register list uses keyset pagination with visibility applied in SQL before `LIMIT`.

**Tech Stack:** Go 1.24, PostgreSQL (raw SQL, no ORM), React + TypeScript + Vite, Playwright, `node:test` for script checks, Vitest for components.

**Spec:** `docs/superpowers/specs/2026-09-23-ropa-register-and-reporting-design.md`

**Out of scope this tranche:** report builder (#26), review reminders, consent, breach register, DPIA register. See Task 12.

---

## File structure

New Go package `internal/ropa`:

| File | Responsibility |
|---|---|
| `model.go` | `ProcessingActivity`, `Status`, `Event`, `Aggregate`, `Coverage`, `RegisterSummary` types |
| `service.go` | Domain rules: lifecycle transitions, closure validation, canonicalisation |
| `repository.go` | `Repository` and `SummaryRepository` interfaces + sentinel errors |
| `memory.go` | In-memory implementation for tests and local mode |
| `postgres.go` | Event-capable command repository: current row + event + outbox in one transaction |
| `current_postgres.go` | Current relational reads, no event replay |
| `summaries.go` | `ListFilter`, cursor encode/decode, `RegisterSummaryRepository` interface |
| `summaries_postgres.go` | Bounded keyset-paginated register list SQL |
| `projection.go` | `SummaryMaintainer` — leased batch refresh of `ropa_register_summary` |
| `projection_postgres.go` | PostgreSQL maintainer writing the summary snapshot |
| `replay.go` | Event replay and point-in-time reconstruction |
| `*_test.go` | Tests for each of the above |

New HTTP layer:

| File | Responsibility |
|---|---|
| `internal/httpapi/ropa_handlers.go` | Handlers for list, get, dashboard, create, update, transition |
| `internal/httpapi/ropa_routes.go` | `ropaRoutes()` returning `[]routeSpec` |
| `internal/httpapi/ropa_routes_test.go` | Route class, permission and OpenAPI parity assertions |

New migration pair:

| File | Responsibility |
|---|---|
| `migrations/000092_ropa_register.up.sql` | Current rows, children, revisions, events, summary, indexes, immutability triggers |
| `migrations/000092_ropa_register.down.sql` | Reverse, refusing when history exists |

New web:

| File | Responsibility |
|---|---|
| `web/src/components/RopaRegisterPage.tsx` | Register list with status/coverage strip and filters |
| `web/src/components/RopaActivityPage.tsx` | One activity: facts, children, history, next action |
| `web/src/components/RopaDashboardStrip.tsx` | Coverage/freshness strip — reused by both pages |
| `web/src/components/RopaRegisterPage.test.tsx` | Component tests |
| `web/src/ropaApi.ts` | Typed fetch client for the ROPA routes |

Modified:

| File | Change |
|---|---|
| `docs/architecture/durable-schema-ownership.md` | Ownership rows for the new tables (CI-enforced) |
| `internal/httpapi/route_catalog.go` | Append `ropa` routes in `productionRoutes()` |
| `internal/httpapi/server.go` | Add `Ropa` and `RopaSummary` dependencies |
| `cmd/api/services_postgres.go` | Wire PostgreSQL repositories and service |
| `cmd/api/services_memory.go` | Wire memory repositories and service |
| `cmd/worker/services_postgres.go` | Register the ROPA summary maintainer |
| `api/runtime.openapi.json` | New paths and schemas |
| `web/src/appRouting.ts` | `View` gains `"ropa"`; `RopaPage` union; `parseRoute`/`routeHash` |
| `web/src/App.tsx` | Sidebar entry, route dispatch |
| `docs/product/archer-requirements-gap.md` | Correct the overstated coverage claim |

---

## Task 1: Migration and schema ownership

**Files:**
- Create: `migrations/000092_ropa_register.up.sql`
- Create: `migrations/000092_ropa_register.down.sql`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Test: `internal/ropa/migration_test.go`

- [ ] **Step 1: Write the failing migration test**

Create `internal/ropa/migration_test.go`:

```go
package ropa_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const upFile = "../../migrations/000092_ropa_register.up.sql"
const downFile = "../../migrations/000092_ropa_register.down.sql"

func readMigration(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestUpMigrationIsASingleTransaction(t *testing.T) {
	body := readMigration(t, upFile)
	if got := strings.Count(body, "BEGIN;"); got != 1 {
		t.Fatalf("expected exactly one BEGIN; got %d", got)
	}
	if got := strings.Count(body, "COMMIT;"); got != 1 {
		t.Fatalf("expected exactly one COMMIT; got %d", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "BEGIN;") {
		t.Fatal("migration must open with BEGIN;")
	}
	if !strings.HasSuffix(strings.TrimSpace(body), "COMMIT;") {
		t.Fatal("migration must close with COMMIT;")
	}
}

func TestUpMigrationCreatesEveryOwnedTable(t *testing.T) {
	body := readMigration(t, upFile)
	for _, table := range []string{
		"ropa_processing_activities",
		"ropa_processing_activity_revisions",
		"ropa_processing_activity_data_categories",
		"ropa_processing_activity_recipients",
		"ropa_processing_activity_systems",
		"ropa_processing_activity_reviews",
		"ropa_events",
		"ropa_register_summary",
	} {
		if !strings.Contains(body, "CREATE TABLE "+table) {
			t.Errorf("migration must create %s", table)
		}
	}
}

func TestLegalEntityIsNotNullAndImmutable(t *testing.T) {
	body := readMigration(t, upFile)
	if !strings.Contains(body, "legal_entity_id uuid NOT NULL") {
		t.Error("legal_entity_id must be NOT NULL")
	}
	if !strings.Contains(body, "ropa_legal_entity_immutable") {
		t.Error("migration must install the legal-entity immutability trigger")
	}
}

func TestCurrentReadIndexesCoverTheKeyset(t *testing.T) {
	body := readMigration(t, upFile)
	for _, index := range []string{
		"ropa_register_keyset_idx",
		"ropa_lawful_basis_idx",
		"ropa_owner_idx",
	} {
		if !strings.Contains(body, index) {
			t.Errorf("migration must create index %s", index)
		}
	}
}

func TestDownMigrationRefusesWhenHistoryExists(t *testing.T) {
	body := readMigration(t, downFile)
	if !strings.Contains(body, "RAISE EXCEPTION") {
		t.Error("down migration must refuse to drop when revision history exists")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestUpMigration -v`
Expected: FAIL — the package directory does not exist.

- [ ] **Step 3: Write the up migration**

Create `migrations/000092_ropa_register.up.sql`:

```sql
BEGIN;

CREATE TABLE ropa_processing_activities (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
  code text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('NEW','OPEN','CLOSED')),
  purpose text NOT NULL DEFAULT '',
  lawful_basis text NOT NULL DEFAULT '',
  controller text NOT NULL DEFAULT '',
  processor text NOT NULL DEFAULT '',
  automated_decision_making boolean NOT NULL DEFAULT false,
  data_subject_categories text NOT NULL DEFAULT '',
  personal_data_categories text NOT NULL DEFAULT '',
  security_measures text NOT NULL DEFAULT '',
  retention_period text NOT NULL DEFAULT '',
  start_date date,
  end_date date,
  next_review_date date,
  owner_principal_id uuid REFERENCES principals(id),
  required_authority_principal_id uuid REFERENCES principals(id),
  program_id uuid,
  version bigint NOT NULL CHECK(version>0),
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  UNIQUE (id, tenant_id),
  UNIQUE (tenant_id, legal_entity_id, code),
  CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
CREATE UNIQUE INDEX ropa_activities_legal_entity_uk ON ropa_processing_activities(legal_entity_id,id);
CREATE INDEX ropa_register_keyset_idx ON ropa_processing_activities(tenant_id,legal_entity_id,status,next_review_date,id);
CREATE INDEX ropa_lawful_basis_idx ON ropa_processing_activities(tenant_id,legal_entity_id,lawful_basis);
CREATE INDEX ropa_owner_idx ON ropa_processing_activities(tenant_id,legal_entity_id,owner_principal_id);

CREATE TABLE ropa_processing_activity_revisions (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  version bigint NOT NULL CHECK(version>0),
  snapshot jsonb NOT NULL,
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,version),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE FUNCTION protect_ropa_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'ROPA processing activity history is immutable'; END; $$;
CREATE TRIGGER ropa_revision_immutable BEFORE UPDATE OR DELETE ON ropa_processing_activity_revisions FOR EACH ROW EXECUTE FUNCTION protect_ropa_revision();

CREATE TABLE ropa_processing_activity_data_categories (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  category text NOT NULL,
  sensitivity text NOT NULL DEFAULT 'UNCLASSIFIED'
    CHECK (sensitivity IN ('UNCLASSIFIED','DIRECT_PERSONAL','INDIRECT_PERSONAL','SENSITIVE_BY_NATURE','SENSITIVE_BY_LAW')),
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,category),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_data_categories_activity_idx ON ropa_processing_activity_data_categories(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_recipients (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  recipient text NOT NULL,
  recipient_kind text NOT NULL DEFAULT 'EXTERNAL'
    CHECK (recipient_kind IN ('INTERNAL','EXTERNAL','AUTHORITY')),
  transfer_basis text NOT NULL DEFAULT '',
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,recipient),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_recipients_activity_idx ON ropa_processing_activity_recipients(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_systems (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  system_name text NOT NULL,
  system_kind text NOT NULL DEFAULT 'APPLICATION'
    CHECK (system_kind IN ('APPLICATION','DATABASE','FILE','MANUAL','THIRD_PARTY')),
  PRIMARY KEY(tenant_id,legal_entity_id,activity_id,system_name),
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_systems_activity_idx ON ropa_processing_activity_systems(tenant_id,legal_entity_id,activity_id);

CREATE TABLE ropa_processing_activity_reviews (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  activity_id uuid NOT NULL,
  due_date date NOT NULL,
  completed_at timestamptz,
  outcome text CHECK (outcome IS NULL OR outcome IN ('CONFIRMED','REVISED','WITHDRAWN')),
  reviewer_principal_id uuid REFERENCES principals(id),
  created_at timestamptz NOT NULL,
  FOREIGN KEY(tenant_id,legal_entity_id,activity_id)
    REFERENCES ropa_processing_activities(tenant_id,legal_entity_id,id)
);
CREATE INDEX ropa_reviews_due_idx ON ropa_processing_activity_reviews(tenant_id,legal_entity_id,due_date);

CREATE TABLE ropa_events (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  legal_entity_id uuid NOT NULL,
  aggregate_type text NOT NULL CHECK (aggregate_type IN ('PROCESSING_ACTIVITY','REPORT_DEFINITION','REPORT_RUN')),
  aggregate_id uuid NOT NULL,
  aggregate_version bigint NOT NULL CHECK(aggregate_version>0),
  type text NOT NULL,
  payload jsonb NOT NULL,
  actor_type text NOT NULL CHECK (actor_type IN ('USER','SERVICE')),
  actor_id uuid,
  occurred_at timestamptz NOT NULL,
  UNIQUE (tenant_id,aggregate_type,aggregate_id,aggregate_version),
  FOREIGN KEY (tenant_id,legal_entity_id)
    REFERENCES legal_entities(tenant_id,id)
);
CREATE INDEX ropa_events_replay_idx ON ropa_events(tenant_id,aggregate_type,aggregate_id,aggregate_version);

CREATE TABLE ropa_register_summary (
  tenant_id uuid NOT NULL,
  legal_entity_id uuid NOT NULL,
  generated_at timestamptz NOT NULL,
  projection_version text NOT NULL,
  source_high_water timestamptz NOT NULL,
  population integer NOT NULL,
  excluded integer,
  unknown integer,
  counts jsonb NOT NULL,
  PRIMARY KEY(tenant_id,legal_entity_id),
  FOREIGN KEY (tenant_id,legal_entity_id) REFERENCES legal_entities(tenant_id,id)
);

CREATE FUNCTION protect_ropa_legal_entity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id THEN
    RAISE EXCEPTION 'ROPA processing activity legal entity is immutable';
  END IF;
  RETURN NEW;
END; $$;
CREATE TRIGGER ropa_legal_entity_immutable BEFORE UPDATE ON ropa_processing_activities
  FOR EACH ROW EXECUTE FUNCTION protect_ropa_legal_entity();

COMMIT;
```

- [ ] **Step 4: Write the down migration**

Create `migrations/000092_ropa_register.down.sql`:

```sql
BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM ropa_processing_activity_revisions) THEN
    RAISE EXCEPTION 'ROPA processing activity history exists; refusing to drop the register';
  END IF;
END $$;

DROP TABLE IF EXISTS ropa_register_summary;
DROP TABLE IF EXISTS ropa_events;
DROP TABLE IF EXISTS ropa_processing_activity_reviews;
DROP TABLE IF EXISTS ropa_processing_activity_systems;
DROP TABLE IF EXISTS ropa_processing_activity_recipients;
DROP TABLE IF EXISTS ropa_processing_activity_data_categories;
DROP TRIGGER IF EXISTS ropa_revision_immutable ON ropa_processing_activity_revisions;
DROP FUNCTION IF EXISTS protect_ropa_revision();
DROP TRIGGER IF EXISTS ropa_legal_entity_immutable ON ropa_processing_activities;
DROP FUNCTION IF EXISTS protect_ropa_legal_entity();
DROP TABLE IF EXISTS ropa_processing_activity_revisions;
DROP TABLE IF EXISTS ropa_processing_activities;

COMMIT;
```

- [ ] **Step 5: Register table ownership**

Append to `docs/architecture/durable-schema-ownership.md`:

```markdown
## T6 ROPA register and dashboard extension

Migration `000092_ropa_register` adds one authoritative processing-activity register and its
dashboard projection. `ropa_processing_activities` is the current row: legal-entity scoped,
immutable legal entity, versioned, and typed in every column the dashboard filters or sorts on.
`ropa_processing_activity_revisions` is immutable reconstruction truth. `ropa_events` is the
append-only domain ledger, unique per aggregate version. `ropa_processing_activity_data_categories`,
`ropa_processing_activity_recipients`, `ropa_processing_activity_systems` and
`ropa_processing_activity_reviews` are normalised children because each is independently governed
and queried. `ropa_register_summary` is a projection: a per-tenant and legal-entity dashboard
snapshot carrying coverage and freshness, and is never treated as authoritative state.

T6 creates no generic audit table, no ROPA-specific task system and no parallel authority model.
Review cycles reuse the existing Workflow Task timer class. Report definitions and report runs
reuse `ropa_events` and land in Tranche 2.
```

- [ ] **Step 6: Run the migration test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS — all five tests.

- [ ] **Step 7: Verify the schema-ownership CI check still passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/architecture/ -run TestDurableSchemaOwnership -v`
Expected: PASS. If the test name differs, run `go test ./internal/architecture/ -v` and confirm the ownership reconstruction test passes.

- [ ] **Step 8: Commit**

```bash
cd C:\dev\clearsight-grc
git add migrations/000092_ropa_register.up.sql migrations/000092_ropa_register.down.sql docs/architecture/durable-schema-ownership.md internal/ropa/migration_test.go
git commit -m "feat(ropa): processing activity register schema and ownership"
```

---

## Task 2: Domain model

**Files:**
- Create: `internal/ropa/model.go`
- Test: `internal/ropa/model_test.go`

- [ ] **Step 1: Write the failing model test**

Create `internal/ropa/model_test.go`:

```go
package ropa_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestProcessingActivityRoundTripsThroughJSON(t *testing.T) {
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	original := ropa.ProcessingActivity{
		ID:                         "activity-1",
		TenantID:                   "tenant-1",
		LegalEntityID:              "entity-1",
		Code:                       "PA-001",
		Name:                       "Customer onboarding",
		Description:                "Collects identity data at onboarding.",
		Status:                     ropa.StatusOpen,
		Purpose:                    "Onboard customers",
		LawfulBasis:                "Contract",
		Controller:                 "Fidelity Bank",
		Processor:                  "Internal operations",
		AutomatedDecisionMaking:    true,
		DataSubjectCategories:      "Customers",
		PersonalDataCategories:     "Name; Date of birth",
		SecurityMeasures:           "Encryption at rest",
		RetentionPeriod:            "7 years",
		StartDate:                  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:                    &end,
		NextReviewDate:             end,
		OwnerPrincipalID:           "owner-1",
		RequiredAuthorityPrincipalID: "authority-1",
		ProgramID:                  "program-1",
		Version:                    3,
		CreatedAt:                  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:                  time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ropa.ProcessingActivity
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != original.ID || decoded.Status != original.Status || decoded.Version != original.Version {
		t.Fatalf("round trip lost identity: %+v", decoded)
	}
	if decoded.EndDate == nil || !decoded.EndDate.Equal(end) {
		t.Fatalf("round trip lost end date")
	}
	if !decoded.AutomatedDecisionMaking {
		t.Fatal("round trip lost automated decision making")
	}
}

func TestStatusExposesOnlyTheThreeLifecycleValues(t *testing.T) {
	for _, status := range []ropa.Status{ropa.StatusNew, ropa.StatusOpen, ropa.StatusClosed} {
		if status.String() == "" {
			t.Fatalf("status %v must render a working label", status)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestProcessingActivity -v`
Expected: FAIL — `undefined: ropa.ProcessingActivity`.

- [ ] **Step 3: Implement the model**

Create `internal/ropa/model.go`:

```go
package ropa

import (
	"encoding/json"
	"fmt"
	"time"
)

// ProjectionVersion identifies the dashboard projection contract. A snapshot
// produced by a different version is reported stale rather than current.
const ProjectionVersion = "ropa-v1"

type Status string

const (
	StatusNew    Status = "NEW"
	StatusOpen   Status = "OPEN"
	StatusClosed Status = "CLOSED"
)

func (s Status) String() string {
	switch s {
	case StatusNew:
		return "Not started"
	case StatusOpen:
		return "In progress"
	case StatusClosed:
		return "Complete"
	default:
		return "Unknown"
	}
}

type ProcessingActivity struct {
	ID                           string     `json:"id"`
	TenantID                     string     `json:"tenant_id"`
	LegalEntityID                string     `json:"legal_entity_id"`
	Code                         string     `json:"code"`
	Name                         string     `json:"name"`
	Description                  string     `json:"description"`
	Status                       Status     `json:"status"`
	Purpose                      string     `json:"purpose"`
	LawfulBasis                  string     `json:"lawful_basis"`
	Controller                   string     `json:"controller"`
	Processor                    string     `json:"processor"`
	AutomatedDecisionMaking      bool       `json:"automated_decision_making"`
	DataSubjectCategories        string     `json:"data_subject_categories"`
	PersonalDataCategories       string     `json:"personal_data_categories"`
	SecurityMeasures             string     `json:"security_measures"`
	RetentionPeriod              string     `json:"retention_period"`
	StartDate                    *time.Time `json:"start_date,omitempty"`
	EndDate                      *time.Time `json:"end_date,omitempty"`
	NextReviewDate               *time.Time `json:"next_review_date,omitempty"`
	OwnerPrincipalID             string     `json:"owner_principal_id,omitempty"`
	RequiredAuthorityPrincipalID string     `json:"required_authority_principal_id,omitempty"`
	ProgramID                    string     `json:"program_id,omitempty"`
	Version                      int64      `json:"version"`
	CreatedAt                    time.Time  `json:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`

	DataCategories []DataCategory `json:"data_categories,omitempty"`
	Recipients     []Recipient    `json:"recipients,omitempty"`
	Systems        []System       `json:"systems,omitempty"`
	Reviews        []Review       `json:"reviews,omitempty"`
}

type DataCategory struct {
	Category    string `json:"category"`
	Sensitivity string `json:"sensitivity"`
}

type Recipient struct {
	Recipient     string `json:"recipient"`
	RecipientKind string `json:"recipient_kind"`
	TransferBasis string `json:"transfer_basis"`
}

type System struct {
	SystemName string `json:"system_name"`
	SystemKind string `json:"system_kind"`
}

type Review struct {
	ID                  string     `json:"id"`
	DueDate             time.Time  `json:"due_date"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	Outcome             string     `json:"outcome,omitempty"`
	ReviewerPrincipalID string     `json:"reviewer_principal_id,omitempty"`
}

type Event struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	LegalEntityID    string          `json:"legal_entity_id"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      string          `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ActorType        string          `json:"actor_type"`
	ActorID          string          `json:"actor_id,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
}

// Coverage separates what the dashboard counted from what it could not. A nil
// pointer means the value is unknown, not zero.
type Coverage struct {
	Population int  `json:"population"`
	Excluded   *int `json:"excluded,omitempty"`
	Unknown    *int `json:"unknown,omitempty"`
}

type Freshness string

const (
	FreshnessCurrent Freshness = "CURRENT"
	FreshnessStale   Freshness = "STALE"
)

type RegisterCounts struct {
	Total           int `json:"total"`
	New             int `json:"new"`
	Open            int `json:"open"`
	Closed          int `json:"closed"`
	ReviewOverdue   int `json:"review_overdue"`
	MissingBasis    int `json:"missing_lawful_basis"`
	MissingOwner    int `json:"missing_owner"`
	NoDataSubjects  int `json:"no_data_subjects"`
	Retired         int `json:"retired"`
}

type RegisterSummary struct {
	TenantID          string          `json:"-"`
	LegalEntityID     string          `json:"-"`
	GeneratedAt       time.Time       `json:"generated_at"`
	ProjectionVersion string          `json:"projection_version"`
	Freshness         Freshness       `json:"freshness"`
	SourceHighWater   time.Time       `json:"source_high_water"`
	Coverage          Coverage        `json:"coverage"`
	Counts            RegisterCounts  `json:"counts"`
}

type Aggregate struct {
	ProcessingActivity
	Events []Event `json:"events,omitempty"`
}

func (a Aggregate) IsRetired() bool {
	return a.EndDate != nil && !a.EndDate.IsZero()
}

func (a Aggregate) ReviewOverdue(now time.Time) bool {
	return a.NextReviewDate != nil && !a.NextReviewDate.IsZero() && a.NextReviewDate.Before(now)
}

func (a Aggregate) String() string {
	return fmt.Sprintf("%s (%s)", a.Name, a.Code)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/model.go internal/ropa/model_test.go
git commit -m "feat(ropa): processing activity domain model"
```

---

## Task 3: Domain service and validation rules

**Files:**
- Create: `internal/ropa/repository.go`
- Create: `internal/ropa/service.go`
- Test: `internal/ropa/service_test.go`

- [ ] **Step 1: Write the failing service test**

Create `internal/ropa/service_test.go`:

```go
package ropa_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func newService() *ropa.Service {
	return ropa.NewService(ropa.NewMemoryRepository(), ropa.NewMemorySummaryRepository())
}

func seedActivity(t *testing.T, svc *ropa.Service, mutate func(*ropa.CreateActivityInput)) ropa.ProcessingActivity {
	t.Helper()
	input := ropa.CreateActivityInput{
		TenantID:              "tenant-1",
		LegalEntityID:         "entity-1",
		Code:                  "PA-001",
		Name:                  "Customer onboarding",
		Purpose:               "Onboard customers",
		LawfulBasis:           "Contract",
		Controller:            "Fidelity Bank",
		Processor:             "Internal operations",
		DataSubjectCategories: "Customers",
		OwnerPrincipalID:      "owner-1",
		ActorID:               "actor-1",
	}
	if mutate != nil {
		mutate(&input)
	}
	activity, err := svc.CreateActivity(context.Background(), input)
	if err != nil {
		t.Fatalf("seed activity: %v", err)
	}
	return activity
}

func TestCreateRejectsMissingMandatoryFields(t *testing.T) {
	svc := newService()
	for name, mutate := range map[string]func(*ropa.CreateActivityInput){
		"code":          func(in *ropa.CreateActivityInput) { in.Code = "" },
		"name":          func(in *ropa.CreateActivityInput) { in.Name = "" },
		"legal entity":  func(in *ropa.CreateActivityInput) { in.LegalEntityID = "" },
		"controller":    func(in *ropa.CreateActivityInput) { in.Controller = "" },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateActivity(context.Background(), inputWith(mutate))
			if !errors.Is(err, ropa.ErrInvalid) {
				t.Fatalf("expected ErrInvalid for missing %s, got %v", name, err)
			}
		})
	}
}

func inputWith(mutate func(*ropa.CreateActivityInput)) ropa.CreateActivityInput {
	input := ropa.CreateActivityInput{
		TenantID:      "tenant-1",
		LegalEntityID: "entity-1",
		Code:          "PA-001",
		Name:          "Customer onboarding",
		Controller:    "Fidelity Bank",
		ActorID:       "actor-1",
	}
	if mutate != nil {
		mutate(&input)
	}
	return input
}

func TestCreateStoresActorAndStartsAtNew(t *testing.T) {
	svc := newService()
	activity := seedActivity(t, svc, nil)
	if activity.Status != ropa.StatusNew {
		t.Fatalf("expected NEW, got %s", activity.Status)
	}
	if activity.Version != 1 {
		t.Fatalf("expected version 1, got %d", activity.Version)
	}
}

func TestCreateRejectsDuplicateCodeInSameLegalEntity(t *testing.T) {
	svc := newService()
	seedActivity(t, svc, nil)
	_, err := svc.CreateActivity(context.Background(), inputWith(nil))
	if !errors.Is(err, ropa.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestSameCodeAllowedInDifferentLegalEntity(t *testing.T) {
	svc := newService()
	seedActivity(t, svc, nil)
	_, err := svc.CreateActivity(context.Background(), ropa.CreateActivityInput{
		TenantID:      "tenant-1",
		LegalEntityID: "entity-2",
		Code:          "PA-001",
		Name:          "Payments processing",
		Controller:    "Fidelity Bank",
		ActorID:       "actor-1",
	})
	if err != nil {
		t.Fatalf("code must be unique per legal entity, not tenant: %v", err)
	}
}

func TestClosureIsBlockedUntilRequiredFactsArePresent(t *testing.T) {
	svc := newService()
	activity := seedActivity(t, svc, func(in *ropa.CreateActivityInput) {
		in.LawfulBasis = ""
		in.OwnerPrincipalID = ""
		in.DataSubjectCategories = ""
	})
	_, err := svc.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:       "tenant-1",
		ActivityID:     activity.ID,
		ExpectedVersion: activity.Version,
		To:             ropa.StatusClosed,
		ActorID:        "actor-1",
	})
	if !errors.Is(err, ropa.ErrClosureBlocked) {
		t.Fatalf("expected ErrClosureBlocked, got %v", err)
	}
	blocked, err := svc.ClosureBlockers(context.Background(), activity.ID)
	if err != nil {
		t.Fatalf("closure blockers: %v", err)
	}
	want := map[string]bool{"lawful basis": true, "named owner": true, "data subject category": true}
	if len(blocked) != len(want) {
		t.Fatalf("expected %d blockers, got %v", len(want), blocked)
	}
	for _, reason := range blocked {
		if !want[reason] {
			t.Fatalf("unexpected blocker %q in %v", reason, blocked)
		}
	}
}

func TestClosureSucceedsWhenRequiredFactsArePresent(t *testing.T) {
	svc := newService()
	activity := seedActivity(t, svc, nil)
	closed, err := svc.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        "tenant-1",
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version,
		To:              ropa.StatusClosed,
		ActorID:         "actor-1",
	})
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closed.Status != ropa.StatusClosed {
		t.Fatalf("expected CLOSED, got %s", closed.Status)
	}
}

func TestStaleExpectedVersionConflicts(t *testing.T) {
	svc := newService()
	activity := seedActivity(t, svc, nil)
	_, err := svc.TransitionActivity(context.Background(), ropa.TransitionActivityInput{
		TenantID:        "tenant-1",
		ActivityID:      activity.ID,
		ExpectedVersion: activity.Version + 5,
		To:              ropa.StatusOpen,
		ActorID:         "actor-1",
	})
	if !errors.Is(err, ropa.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestRetiredActivityLeavesTheLiveRegisterButStaysReadable(t *testing.T) {
	svc := newService()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	retired := now.AddDate(0, 0, -1)
	activity := seedActivity(t, svc, func(in *ropa.CreateActivityInput) { in.EndDate = &retired })
	if _, err := svc.GetActivity(context.Background(), "tenant-1", activity.ID); err != nil {
		t.Fatalf("retired activity must stay readable: %v", err)
	}
	page, err := svc.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID: "tenant-1", LegalEntityID: "entity-1", Limit: 50, IncludeRetired: false,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range page.Rows {
		if row.ID == activity.ID {
			t.Fatal("retired activity must not appear on the live register")
		}
	}
}

func TestEndDateBeforeStartDateIsRejected(t *testing.T) {
	svc := newService()
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, -1)
	_, err := svc.CreateActivity(context.Background(), ropa.CreateActivityInput{
		TenantID: "tenant-1", LegalEntityID: "entity-1", Code: "PA-009", Name: "Bad dates",
		Controller: "Fidelity Bank", ActorID: "actor-1", StartDate: &start, EndDate: &end,
	})
	if !errors.Is(err, ropa.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestCreate -v`
Expected: FAIL — `undefined: ropa.Service`.

- [ ] **Step 3: Write the repository interfaces**

Create `internal/ropa/repository.go`:

```go
package ropa

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("processing activity not found")
	ErrVersionConflict = errors.New("processing activity version conflict")
	ErrDuplicate       = errors.New("processing activity code already exists in this legal entity")
	ErrInvalid         = errors.New("processing activity is not valid")
	ErrClosureBlocked  = errors.New("processing activity closure requirements are not met")
	ErrScopeMismatch   = errors.New("processing activity is outside the requested legal entity")
)

type Repository interface {
	CreateActivity(context.Context, ProcessingActivity, Event) (ProcessingActivity, error)
	GetActivity(context.Context, string, string) (ProcessingActivity, error)
	ApplyActivityEvent(context.Context, string, string, int64, Event) (int64, error)
	ActivityEvents(context.Context, string, string) ([]Event, error)
	ActivityByCode(context.Context, string, string, string) (ProcessingActivity, error)
}

type ListActivitiesFilter struct {
	TenantID        string
	LegalEntityID   string
	Status          Status
	LawfulBasis     string
	OwnerPrincipalID string
	Search          string
	IncludeRetired  bool
	Cursor          string
	Limit           int
}

type ActivityPage struct {
	Rows        []ProcessingActivity
	NextCursor  string
	HasMore     bool
}

type ActivityLister interface {
	ListActivities(context.Context, ListActivitiesFilter) (ActivityPage, error)
}

type SummaryRepository interface {
	LatestSummary(context.Context, string, string) (RegisterSummary, error)
	ReplaceSummary(context.Context, RegisterSummary) error
}
```

- [ ] **Step 4: Write the service**

Create `internal/ropa/service.go`:

```go
package ropa

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type CreateActivityInput struct {
	TenantID              string
	LegalEntityID         string
	Code                  string
	Name                  string
	Description           string
	Purpose               string
	LawfulBasis           string
	Controller            string
	Processor             string
	AutomatedDecisionMaking bool
	DataSubjectCategories string
	PersonalDataCategories string
	SecurityMeasures      string
	RetentionPeriod       string
	StartDate             *time.Time
	EndDate               *time.Time
	NextReviewDate        *time.Time
	OwnerPrincipalID      string
	RequiredAuthorityPrincipalID string
	ProgramID             string
	ActorID               string
}

type UpdateActivityInput struct {
	TenantID              string
	ActivityID            string
	ExpectedVersion       int64
	Description           *string
	Purpose               *string
	LawfulBasis           *string
	Controller            *string
	Processor             *string
	AutomatedDecisionMaking *bool
	DataSubjectCategories *string
	PersonalDataCategories *string
	SecurityMeasures      *string
	RetentionPeriod       *string
	NextReviewDate        *time.Time
	OwnerPrincipalID      *string
	RequiredAuthorityPrincipalID *string
	ActorID               string
}

type TransitionActivityInput struct {
	TenantID        string
	ActivityID      string
	ExpectedVersion int64
	To              Status
	EndDate         *time.Time
	ActorID         string
}

type Service struct {
	repository Repository
	lister     ActivityLister
	summaries  SummaryRepository
	Now        func() time.Time
}

func NewService(repository Repository, summaries SummaryRepository) *Service {
	service := &Service{repository: repository, summaries: summaries, Now: time.Now}
	if memory, ok := repository.(*MemoryRepository); ok {
		service.lister = memory
	}
	return service
}

// SetLister supplies the bounded list reader. Production uses a PostgreSQL
// lister because the command repository and the summary reader are separate
// connections with different query shapes.
func (s *Service) SetLister(lister ActivityLister) {
	if s != nil && lister != nil {
		s.lister = lister
	}
}

func (s *Service) CreateActivity(ctx context.Context, input CreateActivityInput) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	activity := ProcessingActivity{
		TenantID:                     strings.TrimSpace(input.TenantID),
		LegalEntityID:                strings.TrimSpace(input.LegalEntityID),
		Code:                         strings.TrimSpace(input.Code),
		Name:                         strings.TrimSpace(input.Name),
		Description:                  strings.TrimSpace(input.Description),
		Status:                       StatusNew,
		Purpose:                      strings.TrimSpace(input.Purpose),
		LawfulBasis:                  strings.TrimSpace(input.LawfulBasis),
		Controller:                   strings.TrimSpace(input.Controller),
		Processor:                    strings.TrimSpace(input.Processor),
		AutomatedDecisionMaking:      input.AutomatedDecisionMaking,
		DataSubjectCategories:        strings.TrimSpace(input.DataSubjectCategories),
		PersonalDataCategories:       strings.TrimSpace(input.PersonalDataCategories),
		SecurityMeasures:             strings.TrimSpace(input.SecurityMeasures),
		RetentionPeriod:              strings.TrimSpace(input.RetentionPeriod),
		StartDate:                    input.StartDate,
		EndDate:                      input.EndDate,
		NextReviewDate:               input.NextReviewDate,
		OwnerPrincipalID:             strings.TrimSpace(input.OwnerPrincipalID),
		RequiredAuthorityPrincipalID: strings.TrimSpace(input.RequiredAuthorityPrincipalID),
		ProgramID:                    strings.TrimSpace(input.ProgramID),
		Version:                      1,
		CreatedAt:                    s.now(),
		UpdatedAt:                    s.now(),
	}
	if err := validateActivity(activity); err != nil {
		return ProcessingActivity{}, err
	}
	if existing, err := s.repository.ActivityByCode(ctx, activity.TenantID, activity.LegalEntityID, activity.Code); err == nil && existing.ID != "" {
		return ProcessingActivity{}, ErrDuplicate
	} else if err != nil && !errorsIs(err, ErrNotFound) {
		return ProcessingActivity{}, err
	}
	event, err := s.newEvent(ctx, activity, "processing_activity.created", input.ActorID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	return s.repository.CreateActivity(ctx, activity, event)
}

func (s *Service) UpdateActivity(ctx context.Context, input UpdateActivityInput) (ProcessingActivity, error) {
	current, err := s.GetActivity(ctx, input.TenantID, input.ActivityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}
	if input.Description != nil { current.Description = strings.TrimSpace(*input.Description) }
	if input.Purpose != nil { current.Purpose = strings.TrimSpace(*input.Purpose) }
	if input.LawfulBasis != nil { current.LawfulBasis = strings.TrimSpace(*input.LawfulBasis) }
	if input.Controller != nil { current.Controller = strings.TrimSpace(*input.Controller) }
	if input.Processor != nil { current.Processor = strings.TrimSpace(*input.Processor) }
	if input.AutomatedDecisionMaking != nil { current.AutomatedDecisionMaking = *input.AutomatedDecisionMaking }
	if input.DataSubjectCategories != nil { current.DataSubjectCategories = strings.TrimSpace(*input.DataSubjectCategories) }
	if input.PersonalDataCategories != nil { current.PersonalDataCategories = strings.TrimSpace(*input.PersonalDataCategories) }
	if input.SecurityMeasures != nil { current.SecurityMeasures = strings.TrimSpace(*input.SecurityMeasures) }
	if input.RetentionPeriod != nil { current.RetentionPeriod = strings.TrimSpace(*input.RetentionPeriod) }
	if input.NextReviewDate != nil { current.NextReviewDate = input.NextReviewDate }
	if input.OwnerPrincipalID != nil { current.OwnerPrincipalID = strings.TrimSpace(*input.OwnerPrincipalID) }
	if input.RequiredAuthorityPrincipalID != nil { current.RequiredAuthorityPrincipalID = strings.TrimSpace(*input.RequiredAuthorityPrincipalID) }
	if err := validateActivity(current); err != nil {
		return ProcessingActivity{}, err
	}
	event, err := s.newEvent(ctx, current, "processing_activity.updated", input.ActorID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	version, err := s.repository.ApplyActivityEvent(ctx, current.TenantID, current.ID, input.ExpectedVersion, event)
	if err != nil {
		return ProcessingActivity{}, err
	}
	current.Version = version
	current.UpdatedAt = s.now()
	return current, nil
}

func (s *Service) TransitionActivity(ctx context.Context, input TransitionActivityInput) (ProcessingActivity, error) {
	current, err := s.GetActivity(ctx, input.TenantID, input.ActivityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}
	if !validTransition(current.Status, input.To) {
		return ProcessingActivity{}, ErrInvalid
	}
	if input.To == StatusClosed {
		if blockers := closureBlockers(current); len(blockers) > 0 {
			return ProcessingActivity{}, ErrClosureBlocked
		}
	}
	if input.To == StatusClosed && input.EndDate != nil {
		current.EndDate = input.EndDate
	}
	current.Status = input.To
	event, err := s.newEvent(ctx, current, "processing_activity.transitioned", input.ActorID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	version, err := s.repository.ApplyActivityEvent(ctx, current.TenantID, current.ID, input.ExpectedVersion, event)
	if err != nil {
		return ProcessingActivity{}, err
	}
	current.Version = version
	current.UpdatedAt = s.now()
	return current, nil
}

func (s *Service) GetActivity(ctx context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrNotFound
	}
	return s.repository.GetActivity(ctx, tenantID, activityID)
}

func (s *Service) ListActivities(ctx context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	if s == nil || s.lister == nil {
		return ActivityPage{}, ErrNotFound
	}
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.LegalEntityID = strings.TrimSpace(filter.LegalEntityID)
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return ActivityPage{}, ErrInvalid
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return s.lister.ListActivities(ctx, filter)
}

func (s *Service) RegisterSummary(ctx context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	if s == nil || s.summaries == nil {
		return RegisterSummary{}, ErrNotFound
	}
	summary, err := s.summaries.LatestSummary(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID))
	if err != nil {
		return RegisterSummary{}, err
	}
	summary.Freshness = FreshnessCurrent
	if summary.GeneratedAt.IsZero() || s.now().UTC().Sub(summary.GeneratedAt) > 15*time.Minute || summary.ProjectionVersion != ProjectionVersion {
		summary.Freshness = FreshnessStale
	}
	return summary, nil
}

func (s *Service) ClosureBlockers(ctx context.Context, activityID string) ([]string, error) {
	activity, err := s.GetActivity(ctx, "", activityID)
	if err != nil {
		return nil, err
	}
	return closureBlockers(activity), nil
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) newEvent(ctx context.Context, activity ProcessingActivity, eventType, actorID string) (Event, error) {
	payload, err := json.Marshal(activity)
	if err != nil {
		return Event{}, err
	}
	actorType := "USER"
	if strings.TrimSpace(actorID) == "" {
		actorType = "SERVICE"
	}
	return Event{
		ID:               newEventID(),
		TenantID:         activity.TenantID,
		LegalEntityID:    activity.LegalEntityID,
		AggregateType:    "PROCESSING_ACTIVITY",
		AggregateID:      activity.ID,
		AggregateVersion: activity.Version,
		Type:             eventType,
		Payload:          payload,
		ActorType:        actorType,
		ActorID:          strings.TrimSpace(actorID),
		OccurredAt:       s.now(),
	}, nil
}

func validateActivity(activity ProcessingActivity) error {
	if activity.TenantID == "" || activity.LegalEntityID == "" || activity.Code == "" || activity.Name == "" || activity.Controller == "" {
		return ErrInvalid
	}
	if activity.StartDate != nil && activity.EndDate != nil && activity.EndDate.Before(*activity.StartDate) {
		return ErrInvalid
	}
	return nil
}

func validTransition(from, to Status) bool {
	switch from {
	case StatusNew:
		return to == StatusOpen || to == StatusClosed
	case StatusOpen:
		return to == StatusNew || to == StatusClosed
	default:
		return false
	}
}

func closureBlockers(activity ProcessingActivity) []string {
	var blockers []string
	if strings.TrimSpace(activity.LawfulBasis) == "" {
		blockers = append(blockers, "lawful basis")
	}
	if strings.TrimSpace(activity.OwnerPrincipalID) == "" {
		blockers = append(blockers, "named owner")
	}
	if strings.TrimSpace(activity.DataSubjectCategories) == "" {
		blockers = append(blockers, "data subject category")
	}
	return blockers
}

func errorsIs(err, target error) bool {
	return err != nil && target != nil && (err == target || strings.Contains(err.Error(), target.Error()))
}
```

- [ ] **Step 5: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: FAIL — `undefined: ropa.NewMemoryRepository`.

- [ ] **Step 6: Implement the memory repository**

Create `internal/ropa/memory.go`:

```go
package ropa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu        sync.Mutex
	activities map[string]ProcessingActivity
	byCode     map[string]string
	events     map[string][]Event
	revisions  map[string][]ProcessingActivity
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		activities: map[string]ProcessingActivity{},
		byCode:     map[string]string{},
		events:     map[string][]Event{},
		revisions:  map[string][]ProcessingActivity{},
	}
}

func codeKey(tenantID, legalEntityID, code string) string {
	return tenantID + "|" + legalEntityID + "|" + code
}

func activityKey(tenantID, activityID string) string {
	return tenantID + "|" + activityID
}

func (m *MemoryRepository) CreateActivity(_ context.Context, activity ProcessingActivity, event Event) (ProcessingActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := codeKey(activity.TenantID, activity.LegalEntityID, activity.Code)
	if _, exists := m.byCode[key]; exists {
		return ProcessingActivity{}, ErrDuplicate
	}
	activity.ID = newActivityID()
	event.AggregateID = activity.ID
	m.activities[activityKey(activity.TenantID, activity.ID)] = activity
	m.byCode[key] = activity.ID
	m.events[activityKey(activity.TenantID, activity.ID)] = append(m.events[activityKey(activity.TenantID, activity.ID)], event)
	m.revisions[activityKey(activity.TenantID, activity.ID)] = append(m.revisions[activityKey(activity.TenantID, activity.ID)], activity)
	return activity, nil
}

func (m *MemoryRepository) GetActivity(_ context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	activity, ok := m.activities[activityKey(tenantID, activityID)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return activity, nil
}

func (m *MemoryRepository) ActivityByCode(_ context.Context, tenantID, legalEntityID, code string) (ProcessingActivity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byCode[codeKey(tenantID, legalEntityID, code)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	activity, ok := m.activities[activityKey(tenantID, id)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return activity, nil
}

func (m *MemoryRepository) ApplyActivityEvent(_ context.Context, tenantID, activityID string, expectedVersion int64, event Event) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := activityKey(tenantID, activityID)
	activity, ok := m.activities[key]
	if !ok {
		return 0, ErrNotFound
	}
	if activity.Version != expectedVersion || event.AggregateVersion != expectedVersion+1 {
		return 0, ErrVersionConflict
	}
	decoded, err := decodeActivity(event.Payload)
	if err != nil {
		return 0, err
	}
	decoded.ID = activity.ID
	decoded.TenantID = activity.TenantID
	decoded.LegalEntityID = activity.LegalEntityID
	decoded.Version = expectedVersion + 1
	decoded.CreatedAt = activity.CreatedAt
	m.activities[key] = decoded
	m.events[key] = append(m.events[key], event)
	m.revisions[key] = append(m.revisions[key], decoded)
	return decoded.Version, nil
}

func (m *MemoryRepository) ActivityEvents(_ context.Context, tenantID, activityID string) ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events[activityKey(tenantID, activityID)]...), nil
}

func (m *MemoryRepository) ListActivities(_ context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]ProcessingActivity, 0, len(m.activities))
	for _, activity := range m.activities {
		if activity.TenantID != filter.TenantID || activity.LegalEntityID != filter.LegalEntityID {
			continue
		}
		if !filter.IncludeRetired && activity.IsRetired() {
			continue
		}
		if filter.Status != "" && activity.Status != filter.Status {
			continue
		}
		if filter.LawfulBasis != "" && activity.LawfulBasis != filter.LawfulBasis {
			continue
		}
		if filter.OwnerPrincipalID != "" && activity.OwnerPrincipalID != filter.OwnerPrincipalID {
			continue
		}
		if filter.Search != "" && !matchesSearch(activity, filter.Search) {
			continue
		}
		rows = append(rows, activity)
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if left.Status != right.Status {
			return left.Status < right.Status
		}
		leftDate, rightDate := time.Time{}, time.Time{}
		if left.NextReviewDate != nil { leftDate = *left.NextReviewDate }
		if right.NextReviewDate != nil { rightDate = *right.NextReviewDate }
		if !leftDate.Equal(rightDate) {
			return leftDate.Before(rightDate)
		}
		return left.ID < right.ID
	})
	if filter.Cursor != "" {
		after, err := decodeCursor(filter.Cursor)
		if err != nil {
			return ActivityPage{}, ErrInvalid
		}
		filtered := rows[:0]
		for _, row := range rows {
			if cursorAfter(row, after) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	limit := filter.Limit
	if limit <= 0 { limit = 50 }
	page := ActivityPage{}
	if len(rows) > limit {
		page.Rows = rows[:limit]
		last := rows[limit-1]
		page.HasMore = true
		page.NextCursor = encodeCursor(last)
	} else {
		page.Rows = rows
	}
	return page, nil
}

func matchesSearch(activity ProcessingActivity, search string) bool {
	needle := strings.ToLower(search)
	return strings.Contains(strings.ToLower(activity.Name), needle) ||
		strings.Contains(strings.ToLower(activity.Code), needle) ||
		strings.Contains(strings.ToLower(activity.Purpose), needle)
}

type cursorPosition struct {
	Status         string    `json:"s"`
	NextReviewDate time.Time `json:"r"`
	ID             string    `json:"i"`
}

func encodeCursor(activity ProcessingActivity) string {
	position := cursorPosition{Status: string(activity.Status), ID: activity.ID}
	if activity.NextReviewDate != nil {
		position.NextReviewDate = *activity.NextReviewDate
	}
	encoded, _ := json.Marshal(position)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeCursor(cursor string) (cursorPosition, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return cursorPosition{}, err
	}
	var position cursorPosition
	if err := json.Unmarshal(raw, &position); err != nil {
		return cursorPosition{}, err
	}
	return position, nil
}

func cursorAfter(activity ProcessingActivity, position cursorPosition) bool {
	if string(activity.Status) != position.Status {
		return string(activity.Status) > position.Status
	}
	leftDate, rightDate := time.Time{}, position.NextReviewDate
	if activity.NextReviewDate != nil { leftDate = *activity.NextReviewDate }
	if !leftDate.Equal(rightDate) {
		return leftDate.After(rightDate)
	}
	return activity.ID > position.ID
}

func decodeActivity(payload []byte) (ProcessingActivity, error) {
	var activity ProcessingActivity
	if err := json.Unmarshal(payload, &activity); err != nil {
		return ProcessingActivity{}, err
	}
	return activity, nil
}

type MemorySummaryRepository struct {
	mu       sync.Mutex
	summaries map[string]RegisterSummary
}

func NewMemorySummaryRepository() *MemorySummaryRepository {
	return &MemorySummaryRepository{summaries: map[string]RegisterSummary{}}
}

func (m *MemorySummaryRepository) LatestSummary(_ context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	summary, ok := m.summaries[tenantID+"|"+legalEntityID]
	if !ok {
		return RegisterSummary{}, ErrNotFound
	}
	return summary, nil
}

func (m *MemorySummaryRepository) ReplaceSummary(_ context.Context, summary RegisterSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaries[summary.TenantID+"|"+summary.LegalEntityID] = summary
	return nil
}
```

Create `internal/ropa/ids.go`:

```go
package ropa

import "github.com/google/uuid"

func newActivityID() string { return uuid.Must(uuid.NewV7()).String() }
func newEventID() string    { return uuid.Must(uuid.NewV7()).String() }
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS.

If `github.com/google/uuid` is not already a dependency, use the identifier helper the repo already uses. Run `cd C:\dev\clearsight-grc; git grep -n "uuid.NewV7\|uuidv7" -- '*.go' | Select-Object -First 5` and reuse that package.

- [ ] **Step 8: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/repository.go internal/ropa/service.go internal/ropa/memory.go internal/ropa/ids.go internal/ropa/service_test.go
git commit -m "feat(ropa): processing activity service with closure validation"
```

---

## Task 4: Dashboard summary projection

**Files:**
- Create: `internal/ropa/projection.go`
- Test: `internal/ropa/projection_test.go`

- [ ] **Step 1: Write the failing projection test**

Create `internal/ropa/projection_test.go`:

```go
package ropa_test

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestMaintainProducesHonestCoverage(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	seedActivity(t, service, nil)
	seedActivity(t, service, func(in *ropa.CreateActivityInput) {
		in.Code = "PA-002"
		in.LawfulBasis = ""
	})

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary, err := summaries.LatestSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("latest summary: %v", err)
	}
	if summary.Counts.Total != 2 {
		t.Fatalf("expected 2 activities, got %d", summary.Counts.Total)
	}
	if summary.Counts.MissingBasis != 1 {
		t.Fatalf("expected 1 missing lawful basis, got %d", summary.Counts.MissingBasis)
	}
	if summary.Counts.New != 2 {
		t.Fatalf("expected 2 new activities, got %d", summary.Counts.New)
	}
	if summary.Coverage.Excluded == nil || summary.Coverage.Unknown == nil {
		t.Fatal("coverage must report excluded and unknown explicitly, not omit them")
	}
	if *summary.Coverage.Excluded != 0 || *summary.Coverage.Unknown != 0 {
		t.Fatalf("expected zero excluded/unknown, got %v and %v", *summary.Coverage.Excluded, *summary.Coverage.Unknown)
	}
	if summary.ProjectionVersion != ropa.ProjectionVersion {
		t.Fatalf("expected projection version %s, got %s", ropa.ProjectionVersion, summary.ProjectionVersion)
	}
}

func TestRetiredActivitiesAreCountedSeparately(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	retired := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	seedActivity(t, service, func(in *ropa.CreateActivityInput) { in.EndDate = &retired })

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary, err := summaries.LatestSummary(context.Background(), "tenant-1", "entity-1")
	if err != nil {
		t.Fatalf("latest summary: %v", err)
	}
	if summary.Counts.Retired != 1 {
		t.Fatalf("expected 1 retired activity, got %d", summary.Counts.Retired)
	}
}

func TestReviewOverdueIsCounted(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	overdue := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }
	seedActivity(t, service, func(in *ropa.CreateActivityInput) { in.NextReviewDate = &overdue })

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}); err != nil {
		t.Fatalf("maintain: %v", err)
	}
	summary, _ := summaries.LatestSummary(context.Background(), "tenant-1", "entity-1")
	if summary.Counts.ReviewOverdue != 1 {
		t.Fatalf("expected 1 overdue review, got %d", summary.Counts.ReviewOverdue)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestMaintain -v`
Expected: FAIL — `undefined: ropa.NewSummaryMaintainer`.

- [ ] **Step 3: Implement the maintainer**

Create `internal/ropa/projection.go`:

```go
package ropa

import (
	"context"
	"time"
)

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type SummaryMaintainer struct {
	repository Repository
	summaries  SummaryRepository
	service    *Service
	BatchSize  int
}

func NewSummaryMaintainer(repository Repository, summaries SummaryRepository, service *Service) *SummaryMaintainer {
	return &SummaryMaintainer{repository: repository, summaries: summaries, service: service, BatchSize: 500}
}

func (m *SummaryMaintainer) Maintain(ctx context.Context, scope Scope) error {
	if m == nil || m.repository == nil || m.summaries == nil {
		return ErrInvalid
	}
	counts := RegisterCounts{}
	population := 0
	cursor := ""
	now := time.Now().UTC()
	highWater := time.Time{}
	for {
		page, err := m.repositoryList(ctx, scope, cursor)
		if err != nil {
			return err
		}
		for _, activity := range page.Rows {
			population++
			counts.Total++
			switch activity.Status {
			case StatusNew:
				counts.New++
			case StatusOpen:
				counts.Open++
			case StatusClosed:
				counts.Closed++
			}
			if activity.IsRetired() {
				counts.Retired++
			}
			if activity.LawfulBasis == "" {
				counts.MissingBasis++
			}
			if activity.OwnerPrincipalID == "" {
				counts.MissingOwner++
			}
			if activity.DataSubjectCategories == "" {
				counts.NoDataSubjects++
			}
			if activity.ReviewOverdue(now) {
				counts.ReviewOverdue++
			}
			if activity.UpdatedAt.After(highWater) {
				highWater = activity.UpdatedAt
			}
		}
		if !page.HasMore {
			break
		}
		cursor = page.NextCursor
	}
	excluded := 0
	unknown := 0
	return m.summaries.ReplaceSummary(ctx, RegisterSummary{
		TenantID:          scope.TenantID,
		LegalEntityID:     scope.LegalEntityID,
		GeneratedAt:       now,
		ProjectionVersion: ProjectionVersion,
		SourceHighWater:   highWater,
		Coverage:          Coverage{Population: population, Excluded: &excluded, Unknown: &unknown},
		Counts:            counts,
	})
}

func (m *SummaryMaintainer) repositoryList(ctx context.Context, scope Scope, cursor string) (ActivityPage, error) {
	if lister, ok := m.repository.(ActivityLister); ok {
		return lister.ListActivities(ctx, ListActivitiesFilter{
			TenantID:       scope.TenantID,
			LegalEntityID:  scope.LegalEntityID,
			IncludeRetired: true,
			Cursor:         cursor,
			Limit:          m.BatchSize,
		})
	}
	return ActivityPage{}, ErrInvalid
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/projection.go internal/ropa/projection_test.go
git commit -m "feat(ropa): dashboard summary projection with explicit coverage"
```

---

## Task 5: PostgreSQL command repository and bounded reads

**Files:**
- Create: `internal/ropa/postgres.go`
- Create: `internal/ropa/current_postgres.go`
- Create: `internal/ropa/summaries_postgres.go`
- Test: `internal/ropa/portfolio_list_sql_test.go`
- Test: `internal/ropa/summaries_test.go`

- [ ] **Step 1: Write the failing SQL-shape test**

Create `internal/ropa/portfolio_list_sql_test.go`:

```go
package ropa_test

import (
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestListSQLIsBoundedAndNeverReplaysEvents(t *testing.T) {
	query := ropa.ListActivitiesSQL()
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		t.Error("register list query must be bounded with LIMIT")
	}
	if strings.Contains(query, "ropa_events") {
		t.Error("ordinary register list must not replay the event ledger")
	}
	if !strings.Contains(query, "tenant_id = $1") || !strings.Contains(query, "legal_entity_id = $2") {
		t.Error("register list must filter tenant and legal entity in SQL")
	}
}

func TestListSQLExcludesRetiredByDefault(t *testing.T) {
	query := ropa.ListActivitiesSQL()
	if !strings.Contains(query, "end_date IS NULL") {
		t.Error("live register must exclude retired activities by default")
	}
}

func TestSummarySQLGroupsWithoutLoadingBroadPopulations(t *testing.T) {
	query := ropa.RegisterSummarySQL()
	upper := strings.ToUpper(query)
	for _, fragment := range []string{"COUNT(", "GROUP BY", "tenant_id", "legal_entity_id"} {
		if !strings.Contains(upper, fragment) {
			t.Errorf("summary query must contain %s", fragment)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestListSQL -v`
Expected: FAIL — `undefined: ropa.ListActivitiesSQL`.

- [ ] **Step 3: Write the SQL builders**

Create `internal/ropa/summaries_postgres.go`:

```go
package ropa

import "fmt"

// ListActivitiesSQL returns the bounded register list. It selects only the
// requested page in a materialized CTE before assembling anything, and never
// touches the event ledger. The caller supplies filter values in the fixed
// order tenant, legal entity, status, lawful basis, owner, search, include
// retired, cursor tuple, limit.
func ListActivitiesSQL() string {
	return `
WITH page AS MATERIALIZED (
  SELECT id, tenant_id, legal_entity_id, code, name, description, status,
         purpose, lawful_basis, controller, processor,
         automated_decision_making, data_subject_categories,
         personal_data_categories, security_measures, retention_period,
         start_date, end_date, next_review_date, owner_principal_id,
         required_authority_principal_id, program_id, version,
         created_at, updated_at
  FROM ropa_processing_activities
  WHERE tenant_id = $1 AND legal_entity_id = $2
    AND ($3 = '' OR status = $3)
    AND ($4 = '' OR lawful_basis = $4)
    AND ($5 = '' OR owner_principal_id::text = $5)
    AND ($6 = '' OR name ILIKE '%' || $6 || '%' OR code ILIKE '%' || $6 || '%' OR purpose ILIKE '%' || $6 || '%')
    AND ($7 OR end_date IS NULL)
    AND (COALESCE(next_review_date, '0001-01-01'::date), id) > ($8::date, $9::uuid)
  ORDER BY status, COALESCE(next_review_date, '0001-01-01'::date), id
  LIMIT $10
)
SELECT * FROM page
`
}

// RegisterSummarySQL produces one dashboard snapshot row. It aggregates in the
// database so the service never loads the population into memory.
func RegisterSummarySQL() string {
	return `
INSERT INTO ropa_register_summary (
  tenant_id, legal_entity_id, generated_at, projection_version,
  source_high_water, population, excluded, unknown, counts
)
SELECT
  $1, $2, $3, $4,
  COALESCE(MAX(updated_at), $3),
  COUNT(*),
  0, 0,
  jsonb_build_object(
    'total', COUNT(*),
    'new', COUNT(*) FILTER (WHERE status = 'NEW'),
    'open', COUNT(*) FILTER (WHERE status = 'OPEN'),
    'closed', COUNT(*) FILTER (WHERE status = 'CLOSED'),
    'retired', COUNT(*) FILTER (WHERE end_date IS NOT NULL),
    'review_overdue', COUNT(*) FILTER (WHERE next_review_date IS NOT NULL AND next_review_date < $5),
    'missing_lawful_basis', COUNT(*) FILTER (WHERE lawful_basis = ''),
    'missing_owner', COUNT(*) FILTER (WHERE owner_principal_id IS NULL),
    'no_data_subjects', COUNT(*) FILTER (WHERE data_subject_categories = '')
  )
FROM ropa_processing_activities
WHERE tenant_id = $1 AND legal_entity_id = $2
ON CONFLICT (tenant_id, legal_entity_id) DO UPDATE SET
  generated_at = EXCLUDED.generated_at,
  projection_version = EXCLUDED.projection_version,
  source_high_water = EXCLUDED.source_high_water,
  population = EXCLUDED.population,
  excluded = EXCLUDED.excluded,
  unknown = EXCLUDED.unknown,
  counts = EXCLUDED.counts
`
}

func buildInsertActivitySQL() string {
	return `
INSERT INTO ropa_processing_activities (
  id, tenant_id, legal_entity_id, code, name, description, status, purpose,
  lawful_basis, controller, processor, automated_decision_making,
  data_subject_categories, personal_data_categories, security_measures,
  retention_period, start_date, end_date, next_review_date,
  owner_principal_id, required_authority_principal_id, program_id,
  version, created_at, updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25
)
ON CONFLICT (tenant_id, legal_entity_id, code) DO NOTHING
`
}

func buildAppendEventSQL() string {
	return `
INSERT INTO ropa_events (
  id, tenant_id, legal_entity_id, aggregate_type, aggregate_id,
  aggregate_version, type, payload, actor_type, actor_id, occurred_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
`
}

func buildRevisionSQL() string {
	return `
INSERT INTO ropa_processing_activity_revisions (
  tenant_id, legal_entity_id, activity_id, version, snapshot, recorded_at
) VALUES ($1,$2,$3,$4,$5,$6)
`
}

func buildOutboxSQL() string {
	return fmt.Sprintf(`
INSERT INTO outbox_events (id, topic, aggregate_type, aggregate_id, payload, occurred_at)
VALUES ($1,$2,$3,$4,$5,$6)
`, "ropa.event.v1")
}
```

- [ ] **Step 4: Write the command repository**

Create `internal/ropa/postgres.go`:

```go
//go:build postgres

package ropa

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) CreateActivity(ctx context.Context, activity ProcessingActivity, event Event) (ProcessingActivity, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return ProcessingActivity{}, err
	}
	defer tx.Rollback(ctx)

	if activity.ID == "" {
		activity.ID = newActivityID()
	}
	tag, err := tx.Exec(ctx, buildInsertActivitySQL(),
		activity.ID, activity.TenantID, activity.LegalEntityID, activity.Code,
		activity.Name, activity.Description, string(activity.Status), activity.Purpose,
		activity.LawfulBasis, activity.Controller, activity.Processor,
		activity.AutomatedDecisionMaking, activity.DataSubjectCategories,
		activity.PersonalDataCategories, activity.SecurityMeasures, activity.RetentionPeriod,
		activity.StartDate, activity.EndDate, activity.NextReviewDate,
		nullIfEmpty(activity.OwnerPrincipalID), nullIfEmpty(activity.RequiredAuthorityPrincipalID),
		nullIfEmpty(activity.ProgramID), activity.Version, activity.CreatedAt, activity.UpdatedAt,
	)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if tag.RowsAffected() == 0 {
		return ProcessingActivity{}, ErrDuplicate
	}
	if err := writeEventAndRevision(ctx, tx, activity, event); err != nil {
		return ProcessingActivity{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	return activity, nil
}

func (p *PostgresRepository) GetActivity(ctx context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	row := p.pool.QueryRow(ctx, getActivitySQL(), tenantID, activityID)
	activity, err := scanActivity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	return activity, err
}

func (p *PostgresRepository) ActivityByCode(ctx context.Context, tenantID, legalEntityID, code string) (ProcessingActivity, error) {
	row := p.pool.QueryRow(ctx, activityByCodeSQL(), tenantID, legalEntityID, code)
	activity, err := scanActivity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessingActivity{}, ErrNotFound
	}
	return activity, err
}

func (p *PostgresRepository) ApplyActivityEvent(ctx context.Context, tenantID, activityID string, expectedVersion int64, event Event) (int64, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var currentVersion int64
	err = tx.QueryRow(ctx, `SELECT version FROM ropa_processing_activities
	  WHERE id=$2::uuid AND tenant_id=$1::uuid FOR UPDATE`, tenantID, activityID).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if currentVersion != expectedVersion || event.AggregateVersion != expectedVersion+1 {
		return 0, ErrVersionConflict
	}
	activity, err := decodeActivity(event.Payload)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, updateActivitySQL(),
		activity.Description, activity.Purpose, activity.LawfulBasis, activity.Controller,
		activity.Processor, activity.AutomatedDecisionMaking, activity.DataSubjectCategories,
		activity.PersonalDataCategories, activity.SecurityMeasures, activity.RetentionPeriod,
		activity.NextReviewDate, nullIfEmpty(activity.OwnerPrincipalID),
		nullIfEmpty(activity.RequiredAuthorityPrincipalID), string(activity.Status),
		activity.EndDate, expectedVersion+1, activity.UpdatedAt,
		tenantID, activityID, expectedVersion,
	); err != nil {
		return 0, err
	}
	event.AggregateID = activityID
	if err := writeEventAndRevision(ctx, tx, activity, event); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return expectedVersion + 1, nil
}

func (p *PostgresRepository) ActivityEvents(ctx context.Context, tenantID, activityID string) ([]Event, error) {
	rows, err := p.pool.Query(ctx, activityEventsSQL(), tenantID, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var event Event
		var aggregateVersion int64
		if err := rows.Scan(&event.ID, &event.TenantID, &event.AggregateType, &event.AggregateID,
			&aggregateVersion, &event.Type, &event.Payload, &event.ActorType, &event.ActorID, &event.OccurredAt); err != nil {
			return nil, err
		}
		event.AggregateVersion = aggregateVersion
		event.LegalEntityID = ""
		events = append(events, event)
	}
	return events, rows.Err()
}

func writeEventAndRevision(ctx context.Context, tx pgx.Tx, activity ProcessingActivity, event Event) error {
	if event.ID == "" {
		event.ID = newEventID()
	}
	if _, err := tx.Exec(ctx, buildAppendEventSQL(), event.ID, event.TenantID, event.LegalEntityID,
		event.AggregateType, event.AggregateID, event.AggregateVersion, event.Type,
		event.Payload, event.ActorType, nullIfEmpty(event.ActorID), event.OccurredAt); err != nil {
		return err
	}
	snapshot, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, buildRevisionSQL(), activity.TenantID, activity.LegalEntityID,
		activity.ID, activity.Version, snapshot, event.OccurredAt); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, buildOutboxSQL(), newEventID(), "ropa.event.v1", event.AggregateType,
		event.AggregateID, event.Payload, event.OccurredAt)
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
```

- [ ] **Step 5: Write the current read repository**

Create `internal/ropa/current_postgres.go`:

```go
//go:build postgres

package ropa

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const activityColumns = `id, tenant_id, legal_entity_id, code, name, description, status, purpose,
  lawful_basis, controller, processor, automated_decision_making, data_subject_categories,
  personal_data_categories, security_measures, retention_period, start_date, end_date,
  next_review_date, owner_principal_id, required_authority_principal_id, program_id,
  version, created_at, updated_at`

func getActivitySQL() string {
	return `SELECT ` + activityColumns + ` FROM ropa_processing_activities
	  WHERE tenant_id=$1 AND id=$2::uuid`
}

func activityByCodeSQL() string {
	return `SELECT ` + activityColumns + ` FROM ropa_processing_activities
	  WHERE tenant_id=$1 AND legal_entity_id=$2 AND code=$3`
}

func activityEventsSQL() string {
	return `SELECT id, tenant_id, aggregate_type, aggregate_id, aggregate_version, type,
	  payload, actor_type, actor_id, occurred_at
	  FROM ropa_events WHERE tenant_id=$1 AND aggregate_id=$2::uuid
	  ORDER BY aggregate_version`
}

func updateActivitySQL() string {
	return `UPDATE ropa_processing_activities SET
	  description=$1, purpose=$2, lawful_basis=$3, controller=$4, processor=$5,
	  automated_decision_making=$6, data_subject_categories=$7, personal_data_categories=$8,
	  security_measures=$9, retention_period=$10, next_review_date=$11,
	  owner_principal_id=$12, required_authority_principal_id=$13, status=$14,
	  end_date=$15, version=$16, updated_at=$17
	  WHERE tenant_id=$18 AND id=$19::uuid AND version=$20`
}

type rowScanner interface{ Scan(...any) error }

func scanActivity(row rowScanner) (ProcessingActivity, error) {
	var activity ProcessingActivity
	var status string
	err := row.Scan(&activity.ID, &activity.TenantID, &activity.LegalEntityID, &activity.Code,
		&activity.Name, &activity.Description, &status, &activity.Purpose, &activity.LawfulBasis,
		&activity.Controller, &activity.Processor, &activity.AutomatedDecisionMaking,
		&activity.DataSubjectCategories, &activity.PersonalDataCategories, &activity.SecurityMeasures,
		&activity.RetentionPeriod, &activity.StartDate, &activity.EndDate, &activity.NextReviewDate,
		&activity.OwnerPrincipalID, &activity.RequiredAuthorityPrincipalID, &activity.ProgramID,
		&activity.Version, &activity.CreatedAt, &activity.UpdatedAt)
	if err != nil {
		return ProcessingActivity{}, err
	}
	activity.Status = Status(status)
	return activity, nil
}

type PostgresLister struct {
	pool *pgxpool.Pool
}

func NewPostgresLister(pool *pgxpool.Pool) *PostgresLister { return &PostgresLister{pool: pool} }

func (l *PostgresLister) ListActivities(ctx context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	status := ""
	if filter.Status != "" {
		status = string(filter.Status)
	}
	cursorDate := "0001-01-01"
	cursorID := "00000000-0000-0000-0000-000000000000"
	if filter.Cursor != "" {
		position, err := decodeCursor(filter.Cursor)
		if err != nil {
			return ActivityPage{}, ErrInvalid
		}
		if !position.NextReviewDate.IsZero() {
			cursorDate = position.NextReviewDate.Format("2006-01-02")
		}
		cursorID = position.ID
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := l.pool.Query(ctx, ListActivitiesSQL(), filter.TenantID, filter.LegalEntityID,
		status, filter.LawfulBasis, filter.OwnerPrincipalID, filter.Search,
		filter.IncludeRetired, cursorDate, cursorID, limit+1)
	if err != nil {
		return ActivityPage{}, err
	}
	defer rows.Close()
	page := ActivityPage{Rows: []ProcessingActivity{}}
	for rows.Next() {
		activity, err := scanActivity(rows)
		if err != nil {
			return ActivityPage{}, err
		}
		page.Rows = append(page.Rows, activity)
	}
	if err := rows.Err(); err != nil {
		return ActivityPage{}, err
	}
	if len(page.Rows) > limit {
		page.Rows = page.Rows[:limit]
		page.HasMore = true
		page.NextCursor = encodeCursor(page.Rows[limit-1])
	}
	return page, nil
}

var _ = errors.Is
var _ = pgx.ErrNoRows
var _ pgconn.CommandTag
```

- [ ] **Step 6: Run the SQL-shape test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS for the SQL-shape tests; the `postgres`-tagged files are excluded from this build and will be compiled in Task 6.

- [ ] **Step 7: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/summaries_postgres.go internal/ropa/postgres.go internal/ropa/current_postgres.go internal/ropa/portfolio_list_sql_test.go
git commit -m "feat(ropa): bounded PostgreSQL reads and transactional command repository"
```

---

## Task 6: HTTP routes and handlers

**Files:**
- Create: `internal/httpapi/ropa_handlers.go`
- Create: `internal/httpapi/ropa_routes.go`
- Create: `internal/httpapi/ropa_routes_test.go`
- Modify: `internal/httpapi/route_catalog.go:16-23`
- Modify: `internal/httpapi/server.go:44-103`

- [ ] **Step 1: Write the failing route test**

Create `internal/httpapi/ropa_routes_test.go`:

```go
package httpapi

import (
	"net/http"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestRopaRoutesAreRegisteredInProductionCatalog(t *testing.T) {
	api := newRouteTestAPI(t)
	registered := map[string]routeSpec{}
	for _, route := range api.productionRoutes() {
		registered[route.Pattern] = route
	}
	for _, pattern := range []string{
		"/api/v1/ropa/dashboard",
		"/api/v1/ropa/processing-activities",
		"/api/v1/ropa/processing-activities/{id}",
	} {
		if _, ok := registered[pattern]; !ok {
			t.Errorf("route %s must be in the production catalog", pattern)
		}
	}
}

func TestRopaWritesAreMaterialCommands(t *testing.T) {
	api := newRouteTestAPI(t)
	for _, route := range api.productionRoutes() {
		if route.Pattern != "/api/v1/ropa/processing-activities" || route.Method != http.MethodPost {
			continue
		}
		if route.Class != routeClassMaterial {
			t.Fatalf("create must be a material command, got %s", route.Class)
		}
		if route.Policy.ObjectType != "PROCESSING_ACTIVITY" {
			t.Fatalf("object type must be PROCESSING_ACTIVITY, got %s", route.Policy.ObjectType)
		}
		if route.Policy.Responsibility != authority.ResponsibilityOwner {
			t.Fatalf("responsibility must be owner, got %s", route.Policy.Responsibility)
		}
		if !route.Policy.BindLegalEntity {
			t.Fatal("create must bind the verified legal entity")
		}
		return
	}
	t.Fatal("create route not found")
}

func TestRopaReadsAreNotMaterialCommands(t *testing.T) {
	api := newRouteTestAPI(t)
	for _, route := range api.productionRoutes() {
		if route.Pattern == "/api/v1/ropa/dashboard" && route.Class == routeClassMaterial {
			t.Fatal("dashboard read must not be a material command")
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/httpapi/ -run TestRopa -v`
Expected: FAIL — ROPA routes not registered.

- [ ] **Step 3: Write the route specs**

Create `internal/httpapi/ropa_routes.go`:

```go
package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/authority"

func (a *API) ropaRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/ropa/dashboard", a.getRopaDashboard),
		read("/api/v1/ropa/processing-activities", a.listRopaProcessingActivities),
		material("/api/v1/ropa/processing-activities", "ropa.processing_activity.create", a.createRopaProcessingActivity, commandPolicy{
			ObjectType: "PROCESSING_ACTIVITY", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, BindLegalEntity: true,
		}),
		read("/api/v1/ropa/processing-activities/{id}", a.getRopaProcessingActivity),
		read("/api/v1/ropa/processing-activities/{id}/history", a.getRopaProcessingActivityHistory),
		material("/api/v1/ropa/processing-activities/{id}", "ropa.processing_activity.update", a.updateRopaProcessingActivity, commandPolicy{
			ObjectType: "PROCESSING_ACTIVITY", ObjectIDPath: "id",
			Responsibility: authority.ResponsibilityOwner, Materiality: 3,
		}),
		material("/api/v1/ropa/processing-activities/{id}/transition", "ropa.processing_activity.transition", a.transitionRopaProcessingActivity, commandPolicy{
			ObjectType: "PROCESSING_ACTIVITY", ObjectIDPath: "id",
			Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4,
		}),
	}
}
```

- [ ] **Step 4: Write the handlers**

Create `internal/httpapi/ropa_handlers.go`:

```go
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

type ropaActivityResponse struct {
	StateLabel      string                       `json:"state_label"`
	Activity        ropa.ProcessingActivity       `json:"activity"`
	ClosureBlockers []string                     `json:"closure_blockers,omitempty"`
}

func (a *API) getRopaDashboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	summary, err := a.Ropa.RegisterSummary(r.Context(), actor.TenantID, actor.LegalEntityID)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) listRopaProcessingActivities(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := a.Ropa.ListActivities(r.Context(), ropa.ListActivitiesFilter{
		TenantID:        actor.TenantID,
		LegalEntityID:   actor.LegalEntityID,
		Status:          ropa.Status(r.URL.Query().Get("status")),
		LawfulBasis:     r.URL.Query().Get("lawful_basis"),
		OwnerPrincipalID: r.URL.Query().Get("owner_id"),
		Search:          r.URL.Query().Get("search"),
		IncludeRetired:  r.URL.Query().Get("include_retired") == "true",
		Cursor:          r.URL.Query().Get("cursor"),
		Limit:           limit,
	})
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) getRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	activity, err := a.Ropa.GetActivity(r.Context(), actor.TenantID, r.PathValue("id"))
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ropaActivityResponse{
		StateLabel:      activity.Status.String(),
		Activity:        activity,
		ClosureBlockers: a.ropaClosureBlockers(r, activity),
	})
}

func (a *API) getRopaProcessingActivityHistory(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	events, err := a.ropaActivityEvents(r.Context(), actor.TenantID, r.PathValue("id"))
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

type createRopaActivityRequest struct {
	Code                         string  `json:"code"`
	Name                         string  `json:"name"`
	Description                  string  `json:"description"`
	Purpose                      string  `json:"purpose"`
	LawfulBasis                  string  `json:"lawful_basis"`
	Controller                   string  `json:"controller"`
	Processor                    string  `json:"processor"`
	AutomatedDecisionMaking      bool    `json:"automated_decision_making"`
	DataSubjectCategories        string  `json:"data_subject_categories"`
	PersonalDataCategories       string  `json:"personal_data_categories"`
	SecurityMeasures             string  `json:"security_measures"`
	RetentionPeriod              string  `json:"retention_period"`
	OwnerPrincipalID             string  `json:"owner_principal_id"`
	RequiredAuthorityPrincipalID string  `json:"required_authority_principal_id"`
	ProgramID                    string  `json:"program_id"`
	// Deliberately ignored: tenant_id, legal_entity_id, actor_id and version
	// all come from verified request context, never the body.
	TenantID     string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	ActorID      string `json:"actor_id"`
}

func (a *API) createRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var body createRopaActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeROPAError(w, ropa.ErrInvalid)
		return
	}
	activity, err := a.Ropa.CreateActivity(r.Context(), ropa.CreateActivityInput{
		TenantID:                     actor.TenantID,
		LegalEntityID:                actor.LegalEntityID,
		Code:                         body.Code,
		Name:                         body.Name,
		Description:                  body.Description,
		Purpose:                      body.Purpose,
		LawfulBasis:                  body.LawfulBasis,
		Controller:                   body.Controller,
		Processor:                    body.Processor,
		AutomatedDecisionMaking:      body.AutomatedDecisionMaking,
		DataSubjectCategories:        body.DataSubjectCategories,
		PersonalDataCategories:       body.PersonalDataCategories,
		SecurityMeasures:             body.SecurityMeasures,
		RetentionPeriod:              body.RetentionPeriod,
		OwnerPrincipalID:             body.OwnerPrincipalID,
		RequiredAuthorityPrincipalID: body.RequiredAuthorityPrincipalID,
		ProgramID:                    body.ProgramID,
		ActorID:                      actor.PrincipalID,
	})
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ropaActivityResponse{StateLabel: activity.Status.String(), Activity: activity})
}

type transitionRopaActivityRequest struct {
	To              string `json:"to"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
}

func (a *API) transitionRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var body transitionRopaActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeROPAError(w, ropa.ErrInvalid)
		return
	}
	activity, err := a.Ropa.TransitionActivity(r.Context(), ropa.TransitionActivityInput{
		TenantID:        actor.TenantID,
		ActivityID:      r.PathValue("id"),
		ExpectedVersion: body.ExpectedVersion,
		To:              ropa.Status(body.To),
		ActorID:         actor.PrincipalID,
	})
	if err != nil {
		writeROPAError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ropaActivityResponse{StateLabel: activity.Status.String(), Activity: activity})
}

func (a *API) ropaClosureBlockers(r *http.Request, activity ropa.ProcessingActivity) []string {
	if activity.Status == ropa.StatusClosed {
		return nil
	}
	var blockers []string
	if activity.LawfulBasis == "" {
		blockers = append(blockers, "lawful basis")
	}
	if activity.OwnerPrincipalID == "" {
		blockers = append(blockers, "named owner")
	}
	if activity.DataSubjectCategories == "" {
		blockers = append(blockers, "data subject category")
	}
	return blockers
}

func (a *API) ropaActivityEvents(ctx context.Context, tenantID, activityID string) ([]ropa.Event, error) {
	return a.RopaEventsReader.ActivityEvents(ctx, tenantID, activityID)
}

func writeROPAError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ropa.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Processing activity not found."})
	case errors.Is(err, ropa.ErrVersionConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "This processing activity changed since you opened it. Review the current record and try again."})
	case errors.Is(err, ropa.ErrDuplicate):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "A processing activity with this code already exists in this legal entity."})
	case errors.Is(err, ropa.ErrClosureBlocked):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Complete the lawful basis, named owner and data subject category before closing this processing activity."})
	case errors.Is(err, ropa.ErrInvalid):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "This processing activity is not valid. Check the code, name and controller."})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "No change was made. Try again."})
	}
}
```

Reconcile helper names with the file you are editing: if `actorFromRequest`, `writeJSON` and the context type have different names in `internal/httpapi`, use the existing ones. Confirm with `cd C:\dev\clearsight-grc; git grep -n "func actorFrom\|func writeJSON\|actor :=" -- internal/httpapi | Select-Object -First 6`.

- [ ] **Step 5: Register the routes in the catalog**

Modify `internal/httpapi/route_catalog.go` so the ROPA specs join the aggregate:

```go
func (a *API) productionRoutes() []routeSpec {
	base := a.routes()
	distributions := a.formDistributionRoutes()
	communications := a.formCommunicationRoutes()
	proposals := a.formProposalRoutes()
	policies := a.formPolicyRoutes()
	activity := a.activityRoutes()
	gatewayTransports := a.aiGatewayTransportRoutes()
	ropa := a.ropaRoutes()
	routes := make([]routeSpec, 0, len(base)+len(distributions)+len(communications)+len(proposals)+len(policies)+len(activity)+len(gatewayTransports)+len(ropa))
	routes = append(routes, base...)
	routes = append(routes, distributions...)
	routes = append(routes, communications...)
	routes = append(routes, proposals...)
	routes = append(routes, policies...)
	routes = append(routes, activity...)
	routes = append(routes, gatewayTransports...)
	routes = append(routes, ropa...)
	return routes
}
```

- [ ] **Step 6: Add the server dependencies**

Modify `internal/httpapi/server.go` to add fields alongside the existing services:

```go
	Ropa            *ropa.Service
	RopaEventsReader ropa.Repository
```

and the matching import:

```go
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
```

- [ ] **Step 7: Run the route test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/httpapi/ -run TestRopa -v`
Expected: PASS.

- [ ] **Step 8: Update the OpenAPI contract**

Add `/api/v1/ropa/dashboard`, `/api/v1/ropa/processing-activities` and
`/api/v1/ropa/processing-activities/{id}` paths plus their schemas to
`api/runtime.openapi.json`, matching the shapes the handlers return.

Run: `cd C:\dev\clearsight-grc; go test ./internal/httpapi/ -run TestRuntimeContract -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/httpapi/ropa_handlers.go internal/httpapi/ropa_routes.go internal/httpapi/ropa_routes_test.go internal/httpapi/route_catalog.go internal/httpapi/server.go api/runtime.openapi.json
git commit -m "feat(ropa): HTTP routes for the processing activity register and dashboard"
```

---

## Task 7: API and worker composition

**Files:**
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`

- [ ] **Step 1: Wire the PostgreSQL API composition**

In `cmd/api/services_postgres.go`, construct the ROPA repository, lister and summary
repository from the existing pool and pass them to the API:

```go
	ropaRepository := ropa.NewPostgresRepository(pool)
	ropaLister := ropa.NewPostgresLister(pool)
	ropaSummaries := ropa.NewPostgresSummaryRepository(pool)
	ropaService := ropa.NewService(ropaRepository, ropaSummaries)
	ropaService.SetLister(ropaLister)
```

Add `internal/ropa/projection_postgres.go` with:

```go
//go:build postgres

package ropa

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSummaryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSummaryRepository(pool *pgxpool.Pool) *PostgresSummaryRepository {
	return &PostgresSummaryRepository{pool: pool}
}

func (p *PostgresSummaryRepository) LatestSummary(ctx context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	row := p.pool.QueryRow(ctx, latestSummarySQL(), tenantID, legalEntityID)
	var summary RegisterSummary
	var counts []byte
	if err := row.Scan(&summary.GeneratedAt, &summary.ProjectionVersion, &summary.SourceHighWater,
		&summary.Coverage.Population, &summary.Coverage.Excluded, &summary.Coverage.Unknown, &counts); err != nil {
		return RegisterSummary{}, ErrNotFound
	}
	if err := decodeCounts(counts, &summary.Counts); err != nil {
		return RegisterSummary{}, err
	}
	summary.TenantID = tenantID
	summary.LegalEntityID = legalEntityID
	return summary, nil
}

func (p *PostgresSummaryRepository) ReplaceSummary(ctx context.Context, summary RegisterSummary) error {
	_, err := p.pool.Exec(ctx, RegisterSummarySQL(), summary.TenantID, summary.LegalEntityID,
		summary.GeneratedAt, summary.ProjectionVersion, time.Now().UTC())
	return err
}

func latestSummarySQL() string {
	return `SELECT generated_at, projection_version, source_high_water, population, excluded, unknown, counts
	  FROM ropa_register_summary WHERE tenant_id=$1 AND legal_entity_id=$2`
}
```

- [ ] **Step 2: Wire the memory API composition**

In `cmd/api/services_memory.go`:

```go
	ropaRepository := ropa.NewMemoryRepository()
	ropaSummaries := ropa.NewMemorySummaryRepository()
	ropaService := ropa.NewService(ropaRepository, ropaSummaries)
```

- [ ] **Step 3: Register the worker maintainer**

In `cmd/worker/services_postgres.go`, add the ROPA summary maintainer to the existing
worker class list, following the pattern used by the oversight maintainer:

```go
	ropaMaintainer := ropa.NewSummaryMaintainer(ropaRepository, ropaSummaries, ropaService)
```

Give the maintainer a `Maintain(ctx, ropa.Scope{...})` call on the same lease loop that
drives the other projections, and log scope, generated count and duration on completion.

- [ ] **Step 4: Build and vet**

Run: `cd C:\dev\clearsight-grc; go build ./... ; go vet ./internal/ropa/ ./internal/httpapi/`
Expected: no errors.

- [ ] **Step 5: Run the full Go suite**

Run: `cd C:\dev\clearsight-grc; go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd C:\dev\clearsight-grc
git add cmd/api/services_postgres.go cmd/api/services_memory.go cmd/worker/services_postgres.go internal/ropa/projection_postgres.go
git commit -m "feat(ropa): wire API and worker composition"
```

---

## Task 8: Web routing and navigation

**Files:**
- Modify: `web/src/appRouting.ts:1`
- Modify: `web/src/appRouting.test.ts`
- Modify: `web/src/App.tsx:393`
- Create: `web/src/ropaApi.ts`

- [ ] **Step 1: Write the failing routing test**

Append to `web/src/appRouting.test.ts`:

```ts
  it("parses the ROPA register and activity routes", () => {
    expect(parseRoute("#ropa")).toEqual({ view: "ropa", target: { ropaPage: "register" } });
    expect(parseRoute("#ropa/activity/activity-1")).toEqual({
      view: "ropa",
      target: { ropaPage: "register", ropaActivityID: "activity-1" },
    });
  });

  it("round-trips ROPA route hashes", () => {
    expect(routeHash("ropa", { ropaPage: "register" }, "work")).toBe("#ropa");
    expect(routeHash("ropa", { ropaPage: "register", ropaActivityID: "activity-1" }, "work")).toBe("#ropa/activity/activity-1");
  });
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc\web; npx vitest run src/appRouting.test.ts`
Expected: FAIL — `"ropa"` is not in the `View` union.

- [ ] **Step 3: Extend the routing types**

Modify `web/src/appRouting.ts:1`:

```ts
export type View = "today" | "oversight" | "programs" | "forms" | "vendors" | "ropa" | "work" | "people" | "imports" | "explore" | "configure";
export type RopaPage = "register" | "reports";
```

Add `"ropa"` to the `WorkspaceTarget` union:

```ts
  | { ropaPage: RopaPage; ropaActivityID?: string }
```

Extend `parseRoute`:

```ts
  if (view === "ropa") {
    const [sub, identifier] = rest.split("/");
    if (sub === "activity" && identifier) {
      return { view: "ropa", target: { ropaPage: "register", ropaActivityID: decodeURIComponent(identifier) } };
    }
    return { view: "ropa", target: { ropaPage: (sub as RopaPage) || "register" } };
  }
```

Extend `routeHash`:

```ts
  if (view === "ropa") {
    if (target.ropaActivityID) return `#ropa/activity/${encodeURIComponent(target.ropaActivityID)}`;
    return target.ropaPage === "reports" ? "#ropa/reports" : "#ropa";
  }
```

- [ ] **Step 4: Add the API client**

Create `web/src/ropaApi.ts`:

```ts
import type { ProcessingActivity, RegisterSummary } from "./ropaTypes";

export type ActivityPage = {
  rows: ProcessingActivity[];
  next_cursor?: string;
  has_more: boolean;
};

export async function fetchDashboard(signal?: AbortSignal): Promise<RegisterSummary> {
  const response = await fetch("/api/v1/ropa/dashboard", { signal });
  if (!response.ok) throw new Error("The processing activity dashboard is unavailable.");
  return response.json() as Promise<RegisterSummary>;
}

export async function listProcessingActivities(
  params: { status?: string; search?: string; cursor?: string; include_retired?: boolean },
  signal?: AbortSignal,
): Promise<ActivityPage> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.search) query.set("search", params.search);
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.include_retired) query.set("include_retired", "true");
  const response = await fetch(`/api/v1/ropa/processing-activities?${query.toString()}`, { signal });
  if (!response.ok) throw new Error("The processing activity register could not be loaded.");
  return response.json() as Promise<ActivityPage>;
}

export async function fetchProcessingActivity(id: string, signal?: AbortSignal) {
  const response = await fetch(`/api/v1/ropa/processing-activities/${encodeURIComponent(id)}`, { signal });
  if (!response.ok) throw new Error("This processing activity could not be loaded.");
  return response.json();
}
```

Create `web/src/ropaTypes.ts`:

```ts
export type ProcessingActivityStatus = "NEW" | "OPEN" | "CLOSED";

export type ProcessingActivity = {
  id: string;
  code: string;
  name: string;
  description: string;
  status: ProcessingActivityStatus;
  purpose: string;
  lawful_basis: string;
  controller: string;
  processor: string;
  automated_decision_making: boolean;
  data_subject_categories: string;
  personal_data_categories: string;
  security_measures: string;
  retention_period: string;
  start_date?: string;
  end_date?: string;
  next_review_date?: string;
  owner_principal_id?: string;
  required_authority_principal_id?: string;
  program_id?: string;
  version: number;
  updated_at: string;
};

export type RegisterCounts = {
  total: number;
  new: number;
  open: number;
  closed: number;
  review_overdue: number;
  missing_lawful_basis: number;
  missing_owner: number;
  no_data_subjects: number;
  retired: number;
};

export type RegisterSummary = {
  generated_at: string;
  projection_version: string;
  freshness: "CURRENT" | "STALE";
  source_high_water: string;
  coverage: { population: number; excluded?: number; unknown?: number };
  counts: RegisterCounts;
};
```

- [ ] **Step 5: Add the sidebar entry**

In `web/src/App.tsx`, alongside the existing sidebar navigation items, add a ROPA entry
using the same item shape:

```tsx
  { view: "ropa", label: "Processing activities", icon: "clipboard" },
```

- [ ] **Step 6: Run the routing test to verify it passes**

Run: `cd C:\dev\clearsight-grc\web; npx vitest run src/appRouting.test.ts`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd C:\dev\clearsight-grc
git add web/src/appRouting.ts web/src/appRouting.test.ts web/src/App.tsx web/src/ropaApi.ts web/src/ropaTypes.ts
git commit -m "feat(ropa): register routes, navigation and typed API client"
```

---

## Task 9: Register and activity screens

**Files:**
- Create: `web/src/components/RopaDashboardStrip.tsx`
- Create: `web/src/components/RopaRegisterPage.tsx`
- Create: `web/src/components/RopaActivityPage.tsx`
- Create: `web/src/components/RopaRegisterPage.test.tsx`

- [ ] **Step 1: Write the failing component test**

Create `web/src/components/RopaRegisterPage.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RopaDashboardStrip } from "./RopaDashboardStrip";

const current = {
  generated_at: "2026-06-01T10:00:00Z",
  projection_version: "ropa-v1",
  freshness: "CURRENT" as const,
  source_high_water: "2026-06-01T09:58:00Z",
  coverage: { population: 120, excluded: 0, unknown: 0 },
  counts: {
    total: 120, new: 20, open: 80, closed: 20,
    review_overdue: 4, missing_lawful_basis: 3, missing_owner: 1,
    no_data_subjects: 2, retired: 6,
  },
};

describe("RopaDashboardStrip", () => {
  it("shows the population and the exception counts", () => {
    render(<RopaDashboardStrip summary={current} />);
    expect(screen.getByText("120")).toBeInTheDocument();
    expect(screen.getByText(/4/)).toBeInTheDocument();
  });

  it("labels a stale projection instead of presenting it as current", () => {
    render(<RopaDashboardStrip summary={{ ...current, freshness: "STALE" }} />);
    expect(screen.getByText(/may be out of date/i)).toBeInTheDocument();
  });

  it("shows unknown coverage rather than a zero", () => {
    render(<RopaDashboardStrip summary={{ ...current, coverage: { population: 120 } }} />);
    expect(screen.getByText(/unknown/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc\web; npx vitest run src/components/RopaRegisterPage.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Build the coverage strip**

Create `web/src/components/RopaDashboardStrip.tsx`:

```tsx
import type { RegisterSummary } from "../ropaTypes";

function formatTime(value: string): string {
  return new Date(value).toLocaleString();
}

export function RopaDashboardStrip({ summary }: { summary: RegisterSummary }) {
  const { counts, coverage, freshness } = summary;
  const tiles = [
    { label: "Processing activities", value: counts.total, tone: "neutral" as const },
    { label: "In progress", value: counts.open, tone: "info" as const },
    { label: "Review overdue", value: counts.review_overdue, tone: counts.review_overdue > 0 ? "warning" as const : "success" as const },
    { label: "Missing lawful basis", value: counts.missing_lawful_basis, tone: counts.missing_lawful_basis > 0 ? "warning" as const : "success" as const },
    { label: "No named owner", value: counts.missing_owner, tone: counts.missing_owner > 0 ? "warning" as const : "success" as const },
  ];
  return (
    <section className="ropa-summary" aria-label="Processing activity overview">
      {tiles.map((tile) => (
        <article key={tile.label} className={`metric-card metric-card--${tile.tone}`}>
          <p className="metric-card__label">{tile.label}</p>
          <p className="metric-card__value">{tile.value}</p>
        </article>
      ))}
      <p className="ropa-summary__coverage">
        Register checked: {coverage.population} processing activities
        {coverage.excluded === undefined ? ", excluded unknown" : `, ${coverage.excluded} excluded`}
        {coverage.unknown === undefined ? ", unavailable unknown" : `, ${coverage.unknown} unavailable`}.
      </p>
      {freshness === "STALE" ? (
        <p className="notice notice--warning" role="status">
          These counts may be out of date. Last refreshed {formatTime(summary.generated_at)}. Check the register for the latest record before relying on these totals.
        </p>
      ) : (
        <p className="ropa-summary__freshness">Last refreshed {formatTime(summary.generated_at)}.</p>
      )}
    </section>
  );
}
```

- [ ] **Step 4: Build the register page**

Create `web/src/components/RopaRegisterPage.tsx`:

```tsx
import { useEffect, useState } from "react";
import { DataTable, FilterBar, StatusBadge } from "../../src/components";
import { listProcessingActivities } from "../ropaApi";
import type { ProcessingActivity, RegisterSummary } from "../ropaTypes";
import { RopaDashboardStrip } from "./RopaDashboardStrip";

export function RopaRegisterPage({ summary }: { summary?: RegisterSummary }) {
  const [rows, setRows] = useState<ProcessingActivity[]>([]);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    listProcessingActivities({ status: status || undefined, search: search || undefined }, controller.signal)
      .then((page) => { setRows(page.rows ?? []); setError(null); })
      .catch((cause: unknown) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof Error ? cause.message : "The processing activity register could not be loaded.");
      })
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, [status, search]);

  return (
    <div className="workspace ropa-workspace">
      <header className="workspace__header">
        <h1>Processing activities</h1>
        <p>Every activity that collects, uses or shares personal data, with its owner, lawful basis and review date.</p>
      </header>
      {summary ? <RopaDashboardStrip summary={summary} /> : null}
      <FilterBar
        search={{ label: "Search processing activities", value: search, onChange: setSearch }}
        selects={[{ label: "Status", value: status, onChange: setStatus, options: [
          { value: "", label: "All statuses" },
          { value: "NEW", label: "Not started" },
          { value: "OPEN", label: "In progress" },
          { value: "CLOSED", label: "Complete" },
        ] }]}
        resultCount={rows.length}
      />
      {error ? <p className="notice notice--error" role="alert">{error}</p> : null}
      <DataTable
        caption="Processing activities"
        rows={rows}
        rowKey={(row) => row.id}
        columns={[
          { key: "name", header: "Processing activity", render: (row) => (
            <span><strong>{row.name}</strong><br /><small>{row.code}</small></span>
          ) },
          { key: "purpose", header: "Purpose", render: (row) => row.purpose || "Not recorded" },
          { key: "lawful_basis", header: "Lawful basis", render: (row) => row.lawful_basis || "Not recorded" },
          { key: "owner", header: "Owner", render: (row) => row.owner_principal_id || "No named owner" },
          { key: "next_review_date", header: "Next review", render: (row) => row.next_review_date ?? "Not scheduled" },
          { key: "status", header: "Status", render: (row) => <StatusBadge tone="neutral">{row.status === "NEW" ? "Not started" : row.status === "OPEN" ? "In progress" : "Complete"}</StatusBadge> },
        ]}
        loading={loading}
      />
    </div>
  );
}
```

Reconcile the component imports with the real export surface. Run
`cd C:\dev\clearsight-grc; git grep -n "export function DataTable\|export function FilterBar\|export function StatusBadge" -- web/src | Select-Object -First 6`
and use the actual paths and prop names.

- [ ] **Step 5: Build the activity page**

Create `web/src/components/RopaActivityPage.tsx`:

```tsx
import { useEffect, useState } from "react";
import { fetchProcessingActivity } from "../ropaApi";
import type { ProcessingActivity } from "../ropaTypes";

export function RopaActivityPage({ activityID }: { activityID: string }) {
  const [activity, setActivity] = useState<ProcessingActivity | null>(null);
  const [blockers, setBlockers] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    fetchProcessingActivity(activityID, controller.signal)
      .then((payload) => {
        setActivity(payload.activity as ProcessingActivity);
        setBlockers(payload.closure_blockers ?? []);
      })
      .catch((cause: unknown) => {
        if (cause instanceof DOMException && cause.name === "AbortError") return;
        setError(cause instanceof Error ? cause.message : "This processing activity could not be loaded.");
      });
    return () => controller.abort();
  }, [activityID]);

  if (error) return <p className="notice notice--error" role="alert">{error}</p>;
  if (!activity) return <p className="notice">Loading this processing activity…</p>;

  return (
    <div className="workspace ropa-activity">
      <header className="workspace__header">
        <h1>{activity.name}</h1>
        <p>{activity.code}</p>
      </header>
      {blockers.length > 0 ? (
        <div className="notice notice--warning" role="status">
          <p>This processing activity cannot be closed yet. Complete these first:</p>
          <ul>{blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul>
        </div>
      ) : null}
      <dl className="detail-grid">
        <div><dt>Purpose</dt><dd>{activity.purpose || "Not recorded"}</dd></div>
        <div><dt>Lawful basis</dt><dd>{activity.lawful_basis || "Not recorded"}</dd></div>
        <div><dt>Controller</dt><dd>{activity.controller}</dd></div>
        <div><dt>Processor</dt><dd>{activity.processor || "Not recorded"}</dd></div>
        <div><dt>Data subjects</dt><dd>{activity.data_subject_categories || "Not recorded"}</dd></div>
        <div><dt>Personal data</dt><dd>{activity.personal_data_categories || "Not recorded"}</dd></div>
        <div><dt>Retention</dt><dd>{activity.retention_period || "Not recorded"}</dd></div>
        <div><dt>Security measures</dt><dd>{activity.security_measures || "Not recorded"}</dd></div>
        <div><dt>Automated decision making</dt><dd>{activity.automated_decision_making ? "Yes" : "No"}</dd></div>
        <div><dt>Next review</dt><dd>{activity.next_review_date ?? "Not scheduled"}</dd></div>
      </dl>
    </div>
  );
}
```

- [ ] **Step 6: Run the component test to verify it passes**

Run: `cd C:\dev\clearsight-grc\web; npx vitest run src/components/RopaRegisterPage.test.tsx`
Expected: PASS.

- [ ] **Step 7: Run the copy-quality and component suites**

Run: `cd C:\dev\clearsight-grc\web; npm test -- copyQuality src/components`
Expected: PASS. Every visible string must name a business object and state a condition,
source or next action.

- [ ] **Step 8: Commit**

```bash
cd C:\dev\clearsight-grc
git add web/src/components/RopaDashboardStrip.tsx web/src/components/RopaRegisterPage.tsx web/src/components/RopaActivityPage.tsx web/src/components/RopaRegisterPage.test.tsx
git commit -m "feat(ropa): register list, coverage strip and activity record screens"
```

---

## Task 10: Demo fixture data

**Files:**
- Modify: `internal/ropa/demo.go` (create)
- Test: `internal/ropa/demo_test.go`

- [ ] **Step 1: Write the failing demo test**

Create `internal/ropa/demo_test.go`:

```go
package ropa_test

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestDemoInstallIsLabelledAndIdempotent(t *testing.T) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)

	if _, err := ropa.InstallDemo(context.Background(), service); err != nil {
		t.Fatalf("install demo: %v", err)
	}
	page, err := service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID: "bank-demo", LegalEntityID: "entity-demo", Limit: 50,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) < 4 {
		t.Fatalf("expected at least 4 sample processing activities, got %d", len(page.Rows))
	}
	for _, row := range page.Rows {
		if row.Code == "" || row.Name == "" || row.Controller == "" {
			t.Fatalf("sample data must be operationally plausible: %+v", row)
		}
	}
	if _, err := ropa.InstallDemo(context.Background(), service); err != nil {
		t.Fatalf("install demo must be idempotent: %v", err)
	}
	page, _ = service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID: "bank-demo", LegalEntityID: "entity-demo", Limit: 50,
	})
	if len(page.Rows) < 4 {
		t.Fatalf("second install changed the population: %d", len(page.Rows))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -run TestDemo -v`
Expected: FAIL — `undefined: ropa.InstallDemo`.

- [ ] **Step 3: Implement the demo installer**

Create `internal/ropa/demo.go`:

```go
package ropa

import (
	"context"
	"errors"
	"strings"
	"time"
)

// DemoTenant and DemoLegalEntity are the non-production sample scope. Sample
// rows are labelled as sample data and never imply a connected bank is compliant.
const (
	DemoTenant      = "bank-demo"
	DemoLegalEntity = "entity-demo"
)

type demoSeed struct {
	code                   string
	name                   string
	description            string
	purpose                string
	lawfulBasis            string
	processor              string
	dataSubjectCategories  string
	personalDataCategories string
	retentionPeriod        string
	securityMeasures       string
	owner                  string
	status                 Status
	reviewOffsetDays       int
	retired                bool
}

var demoSeeds = []demoSeed{
	{
		code: "PA-CUSTOMER-001", name: "Customer account opening",
		description: "Collects and verifies identity documents when a customer opens an account.",
		purpose: "Open and verify customer accounts", lawfulBasis: "Contract",
		dataSubjectCategories: "Customers", personalDataCategories: "Name; Date of birth; Address",
		retentionPeriod: "7 years after account closure", securityMeasures: "Encryption at rest and in transit",
		owner: "owner-retail-demo", status: StatusOpen, reviewOffsetDays: 120,
	},
	{
		code: "PA-LOAN-002", name: "Loan application assessment",
		description: "Evaluates applicant financial information to decide loan terms.",
		purpose: "Assess loan applications", lawfulBasis: "Legal obligation",
		processor: "Credit scoring service", dataSubjectCategories: "Customers; Prospective customers",
		personalDataCategories: "Income; Employment history; Credit history",
		retentionPeriod: "7 years after loan closure", securityMeasures: "Role-based access; Encryption at rest",
		owner: "owner-credit-demo", status: StatusOpen, reviewOffsetDays: -14,
	},
	{
		code: "PA-MARKETING-003", name: "Customer marketing preferences",
		description: "Stores consent and channel preferences used to send product communications.",
		purpose: "Send approved product communications", lawfulBasis: "Consent",
		dataSubjectCategories: "Customers", personalDataCategories: "Email address; Telephone number; Consent record",
		retentionPeriod: "Consent duration plus 2 years", securityMeasures: "Encryption at rest",
		owner: "owner-marketing-demo", status: StatusNew, reviewOffsetDays: 200,
	},
	{
		code: "PA-LEGACY-ARCHIVE-004", name: "Archived customer records migration",
		description: "Historic extract retained for regulatory retrieval; processing has ceased.",
		purpose: "Regulatory retrieval", lawfulBasis: "Legal obligation",
		dataSubjectCategories: "Customers", personalDataCategories: "Name; Account number; Transaction history",
		retentionPeriod: "10 years", securityMeasures: "Read-only archive storage",
		owner: "owner-records-demo", status: StatusClosed, retired: true,
	},
}

func InstallDemo(ctx context.Context, service *Service) error {
	if service == nil {
		return ErrInvalid
	}
	now := time.Now().UTC()
	for _, seed := range demoSeeds {
		input := CreateActivityInput{
			TenantID:              DemoTenant,
			LegalEntityID:         DemoLegalEntity,
			Code:                  seed.code,
			Name:                  seed.name,
			Description:           seed.description,
			Purpose:               seed.purpose,
			LawfulBasis:           seed.lawfulBasis,
			Controller:            "Fidelity Bank (sample data)",
			Processor:             seed.processor,
			DataSubjectCategories: seed.dataSubjectCategories,
			PersonalDataCategories: seed.personalDataCategories,
			SecurityMeasures:      seed.securityMeasures,
			RetentionPeriod:       seed.retentionPeriod,
			OwnerPrincipalID:      seed.owner,
			ActorID:               "user-demo",
		}
		review := now.AddDate(0, 0, seed.reviewOffsetDays)
		input.NextReviewDate = &review
		if seed.retired {
			retired := now.AddDate(0, -1, 0)
			input.EndDate = &retired
		}
		activity, err := service.CreateActivity(ctx, input)
		if err != nil {
			if errors.Is(err, ErrDuplicate) {
				continue
			}
			return err
		}
		if seed.status != StatusNew {
			if _, err := service.TransitionActivity(ctx, TransitionActivityInput{
				TenantID:        DemoTenant,
				ActivityID:      activity.ID,
				ExpectedVersion: activity.Version,
				To:              seed.status,
				ActorID:         "user-demo",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func IsDemoScope(tenantID, legalEntityID string) bool {
	return strings.TrimSpace(tenantID) == DemoTenant && strings.TrimSpace(legalEntityID) == DemoLegalEntity
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd C:\dev\clearsight-grc; go test ./internal/ropa/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/demo.go internal/ropa/demo_test.go
git commit -m "feat(ropa): labelled sample processing activities for non-production"
```

---

## Task 11: Scale and performance verification

**Files:**
- Create: `internal/ropa/load_test.go`
- Test: `internal/ropa/load_test.go`

- [ ] **Step 1: Write the load test**

Create `internal/ropa/load_test.go`:

```go
//go:build load

package ropa_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

// TestRegisterListAtTargetVolume proves the bounded keyset read stays inside the
// agreed budget at the target population for a very large bank.
func TestRegisterListAtTargetVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	const population = 100_000
	for i := 0; i < population; i++ {
		review := now.AddDate(0, 0, i%365)
		if _, err := service.CreateActivity(context.Background(), ropa.CreateActivityInput{
			TenantID: "tenant-1", LegalEntityID: "entity-1",
			Code: fmt.Sprintf("PA-%06d", i), Name: fmt.Sprintf("Processing activity %d", i),
			Controller: "Fidelity Bank", LawfulBasis: "Contract",
			DataSubjectCategories: "Customers", OwnerPrincipalID: "owner-1",
			NextReviewDate: &review, ActorID: "seed",
		}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	start := time.Now()
	page, err := service.ListActivities(context.Background(), ropa.ListActivitiesFilter{
		TenantID: "tenant-1", LegalEntityID: "entity-1", Limit: 50,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Rows) != 50 {
		t.Fatalf("expected a 50-row page, got %d", len(page.Rows))
	}
	if elapsed > 750*time.Millisecond {
		t.Fatalf("register list took %s at %d activities, budget is 750ms", elapsed, population)
	}
	t.Logf("register list p50-equivalent at %d activities: %s", population, elapsed)
}
```

- [ ] **Step 2: Run the load test**

Run: `cd C:\dev\clearsight-grc; go test -tags load ./internal/ropa/ -run TestRegisterListAtTargetVolume -v -timeout 30m`
Expected: PASS, with a logged duration under 750 ms.

- [ ] **Step 3: Record the result**

Append to `docs/quality/performance-test-plan.md`:

```markdown
## ROPA register volume

Target population: 100,000 processing activities in one legal entity, with the
bounded register list and dashboard projection.

| Read | Budget | Result |
|---|---|---|
| Register list, first page of 50 | p95 under 750 ms | see the `TestRegisterListAtTargetVolume` run log |
| Dashboard summary read from projection | p95 under 500 ms | projection read is a single row lookup by tenant and legal entity |

The dashboard never aggregates the live population on request. The register list
never replays the event ledger. Both budgets are enforced by re-running this
test whenever the ROPA read path changes.
```

- [ ] **Step 4: Commit**

```bash
cd C:\dev\clearsight-grc
git add internal/ropa/load_test.go docs/quality/performance-test-plan.md
git commit -m "test(ropa): 100k-activity register read budget"
```

---

## Task 12: Correct the requirements gap analysis

**Files:**
- Modify: `docs/product/archer-requirements-gap.md`

- [ ] **Step 1: Update the Data Privacy rows**

Replace the #28 and #29 rows so the covered/implemented claim matches the code. The
current text claims a central ROPA population is covered; only a Program requirement
and an evidence contract with a JSON population label exist.

New #28 row:

```markdown
| 28 | Departments complete and update ROPA through automated workflows with reminders and review cycles | Gap | — | No processing-activity register exists. Tranche 1 adds the register, validation rules and a stored review date; automated review cycles and standing reminders remain to be built on the existing Workflow Task timer class. |
```

New #29 row:

```markdown
| 29 | Centralized ROPA repository, automated workflows, real-time dashboards, validation rules, audit trails, role-based access, collaboration tools, notifications and review mechanisms | Gap | `UC-PRIV-01` | Tranche 1 delivers the centralized register, a projection-backed dashboard with explicit coverage and freshness, validation rules on closure, immutable revisions and role-scoped command access. Automated workflows, collaboration tooling and notifications remain to be built. |
```

Update the #26 row to reference Tranche 2:

```markdown
| 26 | Data Privacy team generates and shares reports directly with stakeholders from within the tool | Partial | `UC-REPORT-01` | Governed report/export is a documented Pilot use case. A configurable exception and compliance report builder scoped to a Program, Matter or legal entity is planned for Tranche 2; no in-tool reporting surface is implemented yet. |
```

- [ ] **Step 2: Verify the traceability chain still holds**

Run: `cd C:\dev\clearsight-grc; git grep -n "UC-PRIV-01\|UC-REPORT-01" -- docs/product/use-case-catalogue.md`
Expected: both use-case IDs still exist, so the new status values are traceable.

- [ ] **Step 3: Commit**

```bash
cd C:\dev\clearsight-grc
git add docs/product/archer-requirements-gap.md
git commit -m "docs: correct ROPA coverage claim in the Archer gap analysis"
```

---

## Definition of done

- [ ] `go build ./...` and `go vet ./...` clean
- [ ] `go test ./...` passes, including migration, domain, SQL-shape and route tests
- [ ] `cd web; npm test` passes, including `copyQuality`
- [ ] `npm run typecheck` and `npm run check:runtime-truth` pass
- [ ] `npm run check:ui-contracts` passes
- [ ] Durable schema ownership reconstruction passes with the six new tables
- [ ] OpenAPI parity test passes
- [ ] 100k-activity register load test within budget
- [ ] Register, activity and dashboard screens rendered at 1440x900, 768 and 390 widths
- [ ] A restricted or cross-entity activity is excluded before pagination
- [ ] Client-supplied tenant, legal entity and actor fields are overwritten from verified identity
- [ ] Material commands fail closed when the authority route is missing, ambiguous or unavailable
- [ ] `archer-requirements-gap.md` reflects the implemented state

## Explicitly deferred

- Report builder, maker-checker report definitions and protected report downloads (#26)
- Review reminders and escalation on overdue reviews (#28 remainder)
- Collaboration tooling and notifications (#29 remainder)
- Consent, breach register, DPIA register and PIA screening (#21, #22, #24, #30)
- Event and revision partitioning, once either exceeds 50M rows or 12 months of retention
