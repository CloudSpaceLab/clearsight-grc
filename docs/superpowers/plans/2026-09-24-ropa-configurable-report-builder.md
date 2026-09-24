# Configurable Exception and Compliance Report Builder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a bank's privacy function define, approve, run and download a configurable exception/compliance report over its processing activities, scoped to the whole legal entity, one Program or one Matter.

**Architecture:** A new `internal/reporting` package owns two aggregates. `REPORT_DEFINITION` is maker-checker governed configuration (propose → submit → approve → activate → retire) with immutable revisions, effective dating, a checksum and rollback. `REPORT_RUN` is an immutable, bounded execution receipt that renders a filtered population into a versioned object-storage artefact with a manifest and SHA-256, downloadable through the same protected path as audit exports. The filter vocabulary is a closed, server-enforced allow-list of indexed dimensions; an unknown field, operator or value is rejected before any query runs, so no user-supplied filter can force a full-table scan.

**Tech Stack:** Go 1.24, PostgreSQL 16 via `pgx/v5`, `internal/evidence.ObjectStore`, `internal/authority` command policies, React 19 + TypeScript + Vite, Vitest, Playwright.

**Requirements:** Fidelity Bank Archer workbook #26; NDPA Articles 13 and 48, Schedule 2 ("Maintain ROPA and compliance evidence"). Reference data, not legal advice.

**Related:** `docs/superpowers/specs/2026-09-23-ropa-register-and-reporting-design.md` (approved design, "Reports (#26, Tranche 2)"), `docs/product/archer-requirements-gap.md`, `docs/architecture/durable-schema-ownership.md`.

---

## Scope and non-goals

**In scope**

- A governed report definition: propose, submit for review, review, activate, reject, retire, with immutable revisions, effective dating, checksum and rollback.
- Four datasets: every processing activity; only processing activities carrying an open exception; every Program; and every Matter carrying an open exception or an overdue obligation.
- Three scopes: the whole legal entity, one Program, or one Matter.
- A bounded, **asynchronous** report run that produces a CSV or NDJSON artefact plus a JSON manifest in object storage, with row-count, byte and time bounds.
- A protected download path that re-authorises on every download, verifies artefact integrity, and records the download.
- A web workspace at `#ropa/reports` for defining, reviewing, activating, running and downloading.

**Not in scope**

- Cross-legal-entity or tenant-wide reports. A report is bound to exactly one legal entity.
- Row-level streaming of restricted activities to a shared surface. The artefact is purpose-bound and expires.
- Scheduled or recurring reports. #28's review-cycle timer work owns scheduling; this tranche has a run-now action only.
- Editing a report without going back through maker-checker. There is no direct edit path.
- Report authoring by anyone other than a routed proposer. There is no hard-coded approver.
- Reusing the Forms advanced-filter component unchanged. It is hard-coded to Forms fields, statuses and copy; its registry must be generalised, not its component reused as-is.
- Reusing `routing_policies` as a generic configuration catalogue. Its revision table is consumed by `refresh_effective_authority_routes` and its payload validators assume routing rules.

---

## File structure

| File | Responsibility |
|---|---|
| `migrations/000093_report_builder.up.sql` / `.down.sql` | Report definition, revision and run tables; `matter_id` on activities; indexes; immutability triggers |
| `internal/reporting/model.go` | `ReportDefinition`, `ReportDefinitionRevision`, `ReportRun`, `DecisionRecord`, states, datasets, scopes, formats |
| `internal/reporting/filter.go` | The closed filter vocabulary: field allow-list, operator allow-list, value normaliser, node/depth budget, `ReportFilterSQL` |
| `internal/reporting/exceptions.go` | The single SQL predicate and column projection for the exception dataset, proven equal to `ropa.closureBlockers` |
| `internal/reporting/service.go` | Definition lifecycle, single `ValidateTransitionForWrite` gate, run creation, bounded rendering, manifest, download authorisation |
| `internal/reporting/repository.go` | Repository interfaces and scope types |
| `internal/reporting/memory.go` | In-memory repository and object store composition for demo and tests |
| `internal/reporting/postgres.go` | PostgreSQL repository: transactions, optimistic locking, immutable-revision writes, keyset run reads |
| `internal/reporting/report_postgres.go` | The bounded keyset report query |
| `internal/reporting/worker.go` | Worker-class maintainer that runs queued report runs with lease and bounded retry |
| `internal/httpapi/reporting_routes.go` | Route specs with command policies |
| `internal/httpapi/reporting_handlers.go` | Request decoding, forged-scope overwrite, response shaping, protected download |
| `web/src/reportingTypes.ts` | Wire types mirroring the Go JSON exactly |
| `web/src/reportingApi.ts` | Typed client |
| `web/src/components/ReportingPage.tsx` | The `#ropa/reports` workspace |
| `web/src/components/ReportFilterEditor.tsx` | Allow-list-driven filter builder |
| `web/src/reportingEvidence.ts` | Static review transport so the workspace renders in `review:ui` |

---

## Reuse points this plan depends on

| Reused | Location | Reused for |
|---|---|---|
| Governed-export envelope | `internal/activity/export.go:20-32`, `internal/activity/export_postgres.go` | Run receipt shape, limits, manifest, SHA-256, retention |
| `evidence.ObjectStore` | `internal/evidence/store.go:16-20` | Artefact and manifest storage |
| Protected download | `internal/httpapi/audit_export_handlers.go:107-146` | Re-authorise on download, no-store headers, download audit |
| Maker-checker | `internal/governance/model.go:48-85`, `internal/governance/postgres.go:128-172` | Current + immutable revision shape, transition transaction |
| Filter allow-list | `internal/monitoring/form_library_advanced.go:10-13,41-120` | Node/depth budget, normalisation, rejection before query |
| ROPA scope | `internal/ropa/repository.go:46-60` | `ActivityScope{TenantID, LegalEntityID}`, keyset pages |
| ROPA closure blockers | `internal/ropa/service.go:779-803` | The single definition of an open exception |
| Command policies | `internal/httpapi/route_registry.go:85-89` | `commandPolicy` with distinct `Responsibility` per action |
| Worker class | `cmd/worker/services_postgres.go:40,206-237` | `ConfigureClass` + `AddMaintainerClass` |
| Reserved event vocabulary | `migrations/000092_ropa_register.up.sql:162` | `aggregate_type IN (...,'REPORT_DEFINITION','REPORT_RUN')` already permits both |

---

## Task 1: Migration 000093 — report schema and matter linkage

**Files:**
- Create: `migrations/000093_report_builder.up.sql`
- Create: `migrations/000093_report_builder.down.sql`
- Test: `internal/reporting/migration_test.go`

- [ ] **Step 1: Write the failing test**

`internal/reporting/migration_test.go` must read the migration file and assert the structural guarantees, mirroring the house pattern in `internal/ropa/children_sql_test.go`:

```go
func TestMigrationDeclaresReportDefinitionSchema(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000093_report_builder.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(up)
	for _, required := range []string{
		"CREATE TABLE report_definitions",
		"CREATE TABLE report_definition_revisions",
		"CREATE TABLE report_runs",
		"report_definitions_owner_fk",
		"report_runs_definition_fk",
		"ALTER TABLE ropa_processing_activities ADD COLUMN matter_id",
		"ropa_matter_idx",
		"report_definition_revisions_immutable",
		"report_runs_generation_guard",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration is missing %q", required)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/reporting/ -run TestMigrationDeclaresReportDefinitionSchema -count=1`
Expected: FAIL — `read migration` with "file does not exist".

- [ ] **Step 3: Write the migration**

`migrations/000093_report_builder.up.sql`:

```sql
BEGIN;

-- A processing activity may be linked to the Matter that raised the change.
-- Additive and nullable so existing rows and the Tranche 1 keyset index stay valid.
ALTER TABLE ropa_processing_activities
    ADD COLUMN matter_id uuid;
ALTER TABLE ropa_processing_activities
    ADD CONSTRAINT ropa_activities_matter_tenant_fk
    FOREIGN KEY (matter_id, tenant_id) REFERENCES matters(id, tenant_id);
CREATE INDEX ropa_matter_idx
    ON ropa_processing_activities(tenant_id, legal_entity_id, matter_id);

Create TABLE report_definitions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{2,47}$'),
    name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 3 AND 120),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 1000),
    dataset text NOT NULL CHECK (dataset IN (
        'PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS','PROGRAMS','MATTER_EXCEPTIONS')),
    scope_kind text NOT NULL CHECK (scope_kind IN ('LEGAL_ENTITY','PROGRAM','MATTER')),
    scope_ref uuid,
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    filter jsonb NOT NULL DEFAULT '{"kind":"group","operator":"and","children":[]}'::jsonb
        CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_REVIEW','REVIEWED','ACTIVE','RETIRED')),
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version >= 1),
    checksum text NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    maker_id uuid NOT NULL,
    checker_id uuid,
    reviewer_id uuid,
    reviewer_note text NOT NULL DEFAULT '' CHECK (char_length(reviewer_note) <= 1000),
    effective_from timestamptz,
    effective_until timestamptz,
    submitted_at timestamptz,
    approved_at timestamptz,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    CONSTRAINT report_definitions_entity_tenant_fk
        FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT report_definitions_program_scope_fk
        FOREIGN KEY (scope_ref, tenant_id) REFERENCES programs(id, tenant_id),
    CONSTRAINT report_definitions_maker_tenant_fk
        FOREIGN KEY (maker_id, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definitions_checker_tenant_fk
        FOREIGN KEY (checker_id, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definitions_reviewer_tenant_fk
        FOREIGN KEY (reviewer_id, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definitions_scope_shape_ck CHECK (
        (scope_kind='LEGAL_ENTITY' AND scope_ref IS NULL) OR
        (scope_kind IN ('PROGRAM','MATTER') AND scope_ref IS NOT NULL)),
    -- The approved design keeps responsibility, review and authorization
    -- distinct, so maker, reviewer and authorizer must be three principals.
    CONSTRAINT report_definitions_three_way_separation_ck CHECK (
        (reviewer_id IS NULL OR (reviewer_id <> maker_id AND (checker_id IS NULL OR reviewer_id <> checker_id)))
        AND (checker_id IS NULL OR checker_id <> maker_id)),
    CONSTRAINT report_definitions_effective_ck CHECK (effective_until IS NULL OR effective_from IS NULL OR effective_until > effective_from),
    CONSTRAINT report_definitions_status_ck CHECK (
        (status='DRAFT') OR
        (status='PENDING_REVIEW' AND submitted_at IS NOT NULL) OR
        (status='REVIEWED' AND reviewer_id IS NOT NULL) OR
        (status='ACTIVE' AND reviewer_id IS NOT NULL AND checker_id IS NOT NULL
                 AND approved_at IS NOT NULL AND effective_from IS NOT NULL) OR
        (status='RETIRED' AND retired_at IS NOT NULL)),
    CONSTRAINT report_definitions_scope_kind_ck CHECK (scope_kind <> 'MATTER' OR scope_ref IS NOT NULL)
);
CREATE UNIQUE INDEX report_definitions_scope_code_uq
    ON report_definitions(tenant_id, legal_entity_id, code) WHERE status <> 'RETIRED';
CREATE INDEX report_definitions_tenant_time_idx
    ON report_definitions(tenant_id, legal_entity_id, created_at DESC, id DESC);

CREATE TABLE report_definition_revisions (
    definition_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    version integer NOT NULL CHECK (version >= 1),
    base_version integer NOT NULL CHECK (base_version >= 0),
    dataset text NOT NULL,
    scope_kind text NOT NULL,
    scope_ref uuid,
    format text NOT NULL,
    filter jsonb NOT NULL,
    checksum text NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    maker_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    reviewed_by uuid,
    reviewed_at timestamptz,
    approved_by uuid,
    approved_at timestamptz,
    decision text NOT NULL DEFAULT 'PROPOSED'
        CHECK (decision IN ('PROPOSED','REVIEWED','APPROVED','REJECTED','RETIRED')),
    decision_note text NOT NULL DEFAULT '' CHECK (char_length(decision_note) <= 1000),
    PRIMARY KEY (definition_id, version),
    CONSTRAINT report_definition_revisions_entity_fk
        FOREIGN KEY (definition_id, tenant_id, legal_entity_id)
        REFERENCES report_definitions(id, tenant_id, legal_entity_id),
    CONSTRAINT report_definition_revisions_maker_tenant_fk
        FOREIGN KEY (maker_id, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definition_revisions_reviewer_tenant_fk
        FOREIGN KEY (reviewed_by, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definition_revisions_approver_tenant_fk
        FOREIGN KEY (approved_by, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT report_definition_revisions_three_way_separation_ck CHECK (
        (reviewed_by IS NULL OR (reviewed_by <> maker_id AND (approved_by IS NULL OR reviewed_by <> approved_by)))
        AND (approved_by IS NULL OR approved_by <> maker_id)),
    CONSTRAINT report_definition_revisions_decision_ck CHECK (
        (decision='PROPOSED'  AND reviewed_by IS NULL AND reviewed_at IS NULL AND approved_by IS NULL AND approved_at IS NULL) OR
        (decision='REVIEWED'  AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND approved_by IS NULL AND approved_at IS NULL) OR
        (decision='APPROVED'  AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND approved_by IS NOT NULL AND approved_at IS NOT NULL) OR
        (decision='REJECTED'  AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL) OR
        (decision='RETIRED'))
);
-- A revision is writable only while its decision is still moving forward, and
-- only for the decision columns. Content columns are immutable from the moment
-- the revision is inserted, so an approval can never cover edited content.
CREATE FUNCTION report_definition_revisions_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    step integer;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'report definition revisions cannot be deleted';
    END IF;
    IF NEW.definition_id   IS DISTINCT FROM OLD.definition_id
       OR NEW.tenant_id      IS DISTINCT FROM OLD.tenant_id
       OR NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id
       OR NEW.version        IS DISTINCT FROM OLD.version
       OR NEW.base_version   IS DISTINCT FROM OLD.base_version
       OR NEW.dataset        IS DISTINCT FROM OLD.dataset
       OR NEW.scope_kind     IS DISTINCT FROM OLD.scope_kind
       OR NEW.scope_ref      IS DISTINCT FROM OLD.scope_ref
       OR NEW.format         IS DISTINCT FROM OLD.format
       OR NEW.filter         IS DISTINCT FROM OLD.filter
       OR NEW.checksum       IS DISTINCT FROM OLD.checksum
       OR NEW.maker_id       IS DISTINCT FROM OLD.maker_id
       OR NEW.created_at     IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'report definition revision content is immutable';
    END IF;
    step := CASE OLD.decision WHEN 'PROPOSED' THEN 1 WHEN 'REVIEWED' THEN 2 ELSE 3 END;
    IF CASE NEW.decision WHEN 'PROPOSED' THEN 1 WHEN 'REVIEWED' THEN 2 ELSE 3 END < step THEN
        RAISE EXCEPTION 'report definition revision decisions cannot move backwards';
    END IF;
    IF OLD.decision IN ('APPROVED','REJECTED','RETIRED') THEN
        RAISE EXCEPTION 'report definition revision % is already decided', OLD.version;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER report_definition_revisions_immutable
    BEFORE UPDATE OR DELETE ON report_definition_revisions
    FOR EACH ROW EXECUTE FUNCTION report_definition_revisions_immutable();

CREATE TABLE report_runs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    definition_id uuid NOT NULL,
    definition_version integer NOT NULL CHECK (definition_version >= 1),
    requested_by_ref text NOT NULL,
    as_of timestamptz NOT NULL,
    filter jsonb NOT NULL CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    dataset text NOT NULL,
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    status text NOT NULL CHECK (status IN ('QUEUED','RUNNING','READY','FAILED')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 5),
    row_count integer NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    data_object_key text,
    data_sha256 text,
    manifest_object_key text,
    manifest_sha256 text,
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    completed_at timestamptz,
    expires_at timestamptz NOT NULL,
    CONSTRAINT report_runs_definition_fk
        FOREIGN KEY (definition_id, tenant_id, legal_entity_id)
        REFERENCES report_definitions(id, tenant_id, legal_entity_id),
    CONSTRAINT report_runs_entity_tenant_fk
        FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT report_runs_expiry_ck CHECK (expires_at > created_at),
    CONSTRAINT report_runs_generation_guard CHECK (
        (status='READY') = (data_object_key IS NOT NULL AND data_sha256 IS NOT NULL
                            AND manifest_object_key IS NOT NULL AND manifest_sha256 IS NOT NULL
                            AND completed_at IS NOT NULL)),
    CONSTRAINT report_runs_failure_ck CHECK (status <> 'FAILED' OR failure_code IS NOT NULL)
);
CREATE INDEX report_runs_tenant_time_idx ON report_runs(tenant_id, legal_entity_id, created_at DESC, id DESC);
CREATE INDEX report_runs_definition_idx ON report_runs(tenant_id, definition_id, created_at DESC, id DESC);
CREATE INDEX report_runs_expiry_idx ON report_runs(expires_at, id);
CREATE INDEX report_runs_queue_idx ON report_runs(status, created_at, id) WHERE status IN ('QUEUED','RUNNING');

-- A run may only enter READY once it holds a real artefact, and a terminal row
-- may never be re-opened. This is the database-level half of the generation guard.
CREATE FUNCTION report_runs_generation_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.status IN ('READY','FAILED') AND OLD.status IN ('READY','FAILED') THEN
    RAISE EXCEPTION 'report run % is already terminal', OLD.id;
  END IF;
  IF NEW.status = 'READY' AND (NEW.data_object_key IS NULL OR NEW.manifest_object_key IS NULL) THEN
    RAISE EXCEPTION 'report run % cannot be READY without artefacts', NEW.id;
  END IF;
  IF NEW.attempt_count > 5 THEN
    RAISE EXCEPTION 'report run % exhausted its retry budget', NEW.id;
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER report_runs_generation_guard
    BEFORE UPDATE ON report_runs
    FOR EACH ROW EXECUTE FUNCTION report_runs_generation_guard();

COMMIT;
```

`migrations/000093_report_builder.down.sql` must refuse to erase populated report history rather than dropping it silently:

```sql
BEGIN;
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM report_runs LIMIT 1) THEN
    RAISE EXCEPTION 'report runs exist; refusing to erase report history';
  END IF;
  IF EXISTS (SELECT 1 FROM report_definition_revisions LIMIT 1) THEN
    RAISE EXCEPTION 'report definition revisions exist; refusing to erase report history';
  END IF;
END;
$$;
DROP INDEX IF EXISTS report_runs_queue_idx;
DROP TRIGGER IF EXISTS report_runs_generation_guard ON report_runs;
DROP FUNCTION IF EXISTS report_runs_generation_guard();
DROP TABLE IF EXISTS report_runs;
DROP TRIGGER IF EXISTS report_definition_revisions_immutable ON report_definition_revisions;
DROP FUNCTION IF EXISTS report_definition_revisions_immutable();
DROP TABLE IF EXISTS report_definition_revisions;
DROP TABLE IF EXISTS report_definitions;
DROP INDEX IF EXISTS ropa_matter_idx;
ALTER TABLE ropa_processing_activities DROP CONSTRAINT IF EXISTS ropa_activities_matter_tenant_fk;
ALTER TABLE ropa_processing_activities DROP COLUMN IF EXISTS matter_id;
COMMIT;
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/reporting/ -run TestMigrationDeclaresReportDefinitionSchema -count=1`
Expected: PASS.

- [ ] **Step 5: Verify the migration applies to real PostgreSQL**

Run: `go test ./internal/reporting/ -tags postgres -run TestReportMigrationRoundTrip -count=1`
Expected: PASS. The test applies `000093` up, exercises an insert/approve/run cycle, then applies down and asserts the refusal path. Real DDL is the only proof that the composite foreign keys resolve.

- [ ] **Step 6: Commit**

```bash
git add migrations/000093_report_builder.up.sql migrations/000093_report_builder.down.sql internal/reporting/migration_test.go
git commit -m "feat(reporting): add the report definition and run schema

Report definitions are maker-checker governed current rows with immutable
revisions. Runs are bounded receipts that may only become READY while holding
a real artefact. A nullable matter_id on processing activities makes Matter
scoped reports possible without weakening the Tranche 1 keyset index."
```

---

## Task 2: The filter allow-list

This is the security-critical task. A user-supplied filter must never reach SQL as a free-form fragment, and must never be able to select a non-indexed combination.

**Files:**
- Create: `internal/reporting/filter.go`
- Test: `internal/reporting/filter_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestNormalizeReportFilterRejectsUnknownField(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: "secret_column", Operator: "is", Value: "x",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unknown field to be rejected, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsUnknownOperator(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: ReportFieldStatus, Operator: "matches_regex", Value: "OPEN",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unknown operator to be rejected, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsNodeAndDepthBudget(t *testing.T) {
	deep := &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
		{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
				{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
					{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN"},
				}},
			}},
		}},
	}}
	if _, err := NormalizeReportFilter(deep); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected the depth budget to be enforced, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsValueOutsideTheFieldVocabulary(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "MADE_UP",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unlisted status value to be rejected, got %v", err)
	}
}

func TestReportFilterSQLBindsEveryValueAsAParameter(t *testing.T) {
	expression, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "open"},
			{Kind: "condition", Field: ReportFieldName, Operator: "contains", Value: "'; DROP TABLE x; --"},
		},
	})
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}
	fragment, args, err := ReportFilterSQL(expression, 3)
	if err != nil {
		t.Fatalf("build SQL: %v", err)
	}
	if strings.Contains(fragment, "DROP TABLE") || strings.Contains(fragment, "open") {
		t.Fatalf("filter SQL must carry no literal user value: %q", fragment)
	}
	if len(args) != 2 {
		t.Fatalf("expected both values to be bound parameters, got %d", len(args))
	}
	if args[0] != "OPEN" || args[1] != "'; DROP TABLE x; --" {
		t.Fatalf("unexpected bound arguments: %#v", args)
	}
}

func TestReportFilterSQLRejectsAnUnboundFieldEvenIfNormalisationWasSkipped(t *testing.T) {
	// Defence in depth: SQL generation validates the field itself rather than
	// trusting that NormalizeReportFilter ran first.
	if _, _, err := ReportFilterSQL(&ReportFilterExpression{
		Kind: "condition", Field: "owner_principal_id || password", Operator: "is", Value: "x",
	}, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected SQL generation to reject an unbound field, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/reporting/ -run "TestNormalizeReportFilter|TestReportFilterSQL" -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `internal/reporting/filter.go`**

```go
package reporting

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

const (
	// A report filter is a bounded allow-list expression, never free-form SQL.
	maxReportFilterNodes = 12
	maxReportFilterDepth = 3
	maxReportFilterValue = 200
)

// ReportFilterField is a field a report may be filtered on. The set is closed:
// every entry maps to a column covered by a migration 000092 or 000093 index.
type ReportFilterField string

const (
	ReportFieldStatus          ReportFilterField = "status"
	ReportFieldLawfulBasis     ReportFilterField = "lawful_basis"
	ReportFieldOwner           ReportFilterField = "owner_principal_id"
	ReportFieldProgram         ReportFilterField = "program_id"
	ReportFieldMatter          ReportFilterField = "matter_id"
	ReportFieldAutomated       ReportFilterField = "automated_decision_making"
	ReportFieldCrossBorder     ReportFilterField = "cross_border_transfer"
	ReportFieldReviewOverdue   ReportFilterField = "review_overdue"
	ReportFieldMissingBasis    ReportFilterField = "missing_lawful_basis"
	ReportFieldMissingOwner    ReportFilterField = "missing_owner"
	ReportFieldMissingSubjects ReportFilterField = "missing_data_subjects"
	ReportFieldName            ReportFilterField = "name"
)

// ReportFilterFieldVocabulary is published to the web workspace so the builder
// can only offer filters the server will accept.
var ReportFilterFieldVocabulary = []ReportFilterFieldDefinition{
	{Field: ReportFieldStatus, Label: "Processing activity status", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldLawfulBasis, Label: "Lawful basis", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldOwner, Label: "Named owner", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldProgram, Label: "Related program", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldMatter, Label: "Related issue or change", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldAutomated, Label: "Automated decision making", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldCrossBorder, Label: "Cross-border transfer", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldReviewOverdue, Label: "Review overdue", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldMissingBasis, Label: "Lawful basis not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingOwner, Label: "Owner not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingSubjects, Label: "Data subject categories not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldName, Label: "Activity name contains", Operators: []string{"contains"}, Indexed: false},
}

type ReportFilterFieldDefinition struct {
	Field     ReportFilterField `json:"field"`
	Label     string            `json:"label"`
	Operators []string          `json:"operators"`
	Indexed   bool              `json:"indexed"`
}

// ReportFilterExpression is a bounded condition tree. Kind is "condition" or
// "group"; a group joins children with and/or and holds no field or value.
type ReportFilterExpression struct {
	Kind     string                 `json:"kind"`
	Field    ReportFilterField      `json:"field,omitempty"`
	Operator string                 `json:"operator"`
	Value    string                 `json:"value,omitempty"`
	Children []ReportFilterExpression `json:"children,omitempty"`
}

// filterSQLFragment is the only place a field becomes a SQL fragment. A field
// absent from this switch can never reach a query, whatever the caller sends.
func filterSQLFragment(field ReportFilterField, operator string) (string, error) {
	switch field {
	case ReportFieldStatus:
		if operator != "is" {
			return "", fmt.Errorf("status supports only the is operator")
		}
		return "a.status = $%d", nil
	case ReportFieldLawfulBasis:
		if operator != "is" {
			return "", fmt.Errorf("lawful basis supports only the is operator")
		}
		return "a.lawful_basis = $%d", nil
	case ReportFieldOwner:
		switch operator {
		case "is":
			return "a.owner_principal_id = $%d::uuid", nil
		case "is_not":
			return "(a.owner_principal_id IS NULL OR a.owner_principal_id <> $%d::uuid)", nil
		}
	case ReportFieldProgram:
		switch operator {
		case "is":
			return "a.program_id = $%d::uuid", nil
		case "is_not":
			return "(a.program_id IS NULL OR a.program_id <> $%d::uuid)", nil
		}
	case ReportFieldMatter:
		switch operator {
		case "is":
			return "a.matter_id = $%d::uuid", nil
		case "is_not":
			return "(a.matter_id IS NULL OR a.matter_id <> $%d::uuid)", nil
		}
	case ReportFieldAutomated:
		if operator != "is" {
			return "", fmt.Errorf("automated decision making supports only the is operator")
		}
		return "a.automated_decision_making = $%d::boolean", nil
	case ReportFieldCrossBorder:
		if operator != "is" {
			return "", fmt.Errorf("cross-border transfer supports only the is operator")
		}
		return "EXISTS (SELECT 1 FROM ropa_processing_activity_recipients rc WHERE rc.tenant_id=a.tenant_id AND rc.legal_entity_id=a.legal_entity_id AND rc.activity_id=a.id AND rc.is_cross_border = $%d::boolean)", nil
	case ReportFieldReviewOverdue:
		if operator != "is" {
			return "", fmt.Errorf("review overdue supports only the is operator")
		}
		return "(a.next_review_date IS NOT NULL AND a.next_review_date < $%d::timestamptz)", nil
	case ReportFieldMissingBasis:
		if operator != "is" {
			return "", fmt.Errorf("missing lawful basis supports only the is operator")
		}
		return "(btrim(a.lawful_basis) = '' AND $%d::boolean)", nil
	case ReportFieldMissingOwner:
		if operator != "is" {
			return "", fmt.Errorf("missing owner supports only the is operator")
		}
		return "(a.owner_principal_id IS NULL AND $%d::boolean)", nil
	case ReportFieldMissingSubjects:
		if operator != "is" {
			return "", fmt.Errorf("missing data subject categories supports only the is operator")
		}
		return "(btrim(a.data_subject_categories) = '' AND $%d::boolean)", nil
	case ReportFieldName:
		if operator != "contains" {
			return "", fmt.Errorf("activity name supports only the contains operator")
		}
		return "a.name ILIKE '%' || $%d || '%'", nil
	}
	return "", fmt.Errorf("field %q is not available for report filtering", field)
}
```

Add to the same file:

```go
// NormalizeReportFilter validates a filter against the allow-list before it can
// reach SQL. It returns a nil expression for an empty filter so an unfiltered
// report renders the whole scoped population rather than nothing.
func NormalizeReportFilter(expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	if expression == nil {
		return nil, nil
	}
	if isEmptyReportFilter(expression) {
		return nil, nil
	}
	nodes := 0
	normalized, err := normalizeReportFilterNode(*expression, 1, &nodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func isEmptyReportFilter(expression *ReportFilterExpression) bool {
	return expression.Kind == "group" && len(expression.Children) == 0
}

func normalizeReportFilterNode(expression ReportFilterExpression, depth int, nodes *int) (ReportFilterExpression, error) {
	*nodes++
	if *nodes > maxReportFilterNodes || depth > maxReportFilterDepth {
		return ReportFilterExpression{}, fmt.Errorf("report filters are limited to %d conditions and %d levels", maxReportFilterNodes, maxReportFilterDepth)
	}
	expression.Kind = strings.ToLower(strings.TrimSpace(expression.Kind))
	expression.Operator = strings.ToLower(strings.TrimSpace(expression.Operator))
	switch expression.Kind {
	case "condition":
		if len(expression.Children) != 0 {
			return ReportFilterExpression{}, fmt.Errorf("report filter conditions cannot contain children")
		}
		if _, err := filterSQLFragment(expression.Field, expression.Operator); err != nil {
			return ReportFilterExpression{}, err
		}
		value, err := normalizeReportFilterValue(expression.Field, expression.Value)
		if err != nil {
			return ReportFilterExpression{}, err
		}
		expression.Value = value
		return expression, nil
	case "group":
		if expression.Field != "" || expression.Value != "" {
			return ReportFilterExpression{}, fmt.Errorf("report filter groups cannot carry a field or value")
		}
		if expression.Operator != "and" && expression.Operator != "or" {
			return ReportFilterExpression{}, fmt.Errorf("report filter groups join conditions with and or or")
		}
		if len(expression.Children) == 0 {
			return ReportFilterExpression{}, fmt.Errorf("report filter groups need at least one condition")
		}
		children := make([]ReportFilterExpression, 0, len(expression.Children))
		for _, child := range expression.Children {
			normalized, err := normalizeReportFilterNode(child, depth+1, nodes)
			if err != nil {
				return ReportFilterExpression{}, err
			}
			children = append(children, normalized)
		}
		expression.Children = children
		return expression, nil
	default:
		return ReportFilterExpression{}, fmt.Errorf("report filter nodes are conditions or groups")
	}
}

func normalizeReportFilterValue(field ReportFilterField, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("report filter values cannot be empty")
	}
	if len([]rune(value)) > maxReportFilterValue {
		return "", fmt.Errorf("report filter values are limited to %d characters", maxReportFilterValue)
	}
	switch field {
	case ReportFieldStatus:
		value = strings.ToUpper(value)
		if !ropa.ValidStatus(ropa.Status(value)) {
			return "", fmt.Errorf("%q is not a recorded processing activity status", value)
		}
		return value, nil	case ReportFieldAutomated, ReportFieldCrossBorder, ReportFieldReviewOverdue,
		ReportFieldMissingBasis, ReportFieldMissingOwner, ReportFieldMissingSubjects:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", fmt.Errorf("%s is recorded as true or false", field)
		}
		return strconv.FormatBool(parsed), nil
	case ReportFieldOwner, ReportFieldProgram, ReportFieldMatter:
		if !isUUID(value) {
			return "", fmt.Errorf("%s must be a recorded identifier", field)
		}
		return strings.ToLower(value), nil
	}
	return value, nil
}

// ReportFilterSQL renders the validated expression into a parameterised
// fragment starting at the given position. The returned arguments are in
// fragment order, so the caller appends them directly to its query arguments.
func ReportFilterSQL(expression *ReportFilterExpression, nextPosition int) (string, []any, error) {
	if expression == nil {
		return "TRUE", nil, nil
	}
	args := make([]any, 0, maxReportFilterNodes)
	position := nextPosition
	fragment, err := renderReportFilterNode(expression, &position, &args)
	if err != nil {
		return "", nil, err
	}
	return fragment, args, nil
}

// renderReportFilterNode advances position for every bound value, so the first
// value is $nextPosition and each subsequent one is the next number after it.
func renderReportFilterNode(expression *ReportFilterExpression, position *int, args *[]any) (string, error) {
	if expression == nil {
		return "TRUE", nil
	}
	switch strings.ToLower(strings.TrimSpace(expression.Kind)) {
	case "condition":
		fragment, err := filterSQLFragment(expression.Field, strings.ToLower(strings.TrimSpace(expression.Operator)))
		if err != nil {
			return "", err
		}
		*args = append(*args, expression.Value)
		bound := *position
		*position++
		return fmt.Sprintf(fragment, bound), nil
	case "group":
		parts := make([]string, 0, len(expression.Children))
		for i := range expression.Children {
			part, err := renderReportFilterNode(&expression.Children[i], position, args)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		joiner := " AND "
		if strings.EqualFold(strings.TrimSpace(expression.Operator), "or") {
			joiner = " OR "
		}
		return "(" + strings.Join(parts, joiner) + ")", nil
	default:
		return "", fmt.Errorf("report filter nodes are conditions or groups")
	}
}
```

Add `isUUID` to the same file:

```go
func isUUID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 36 {
		return false
	}
	for i, r := range trimmed {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}
```

- [ ] **Step 4: Export the register's status vocabulary rather than restating it**

The filter must accept exactly the statuses the register records. Add to `internal/ropa/model.go`, directly below `validStatus`'s counterpart:

```go
// ValidStatus lets a consumer validate against the register's own status
// vocabulary instead of copying the list, which is how a filter would come to
// offer a status the register cannot store.
func ValidStatus(status Status) bool { return validStatus(status) }
```

`internal/ropa/service.go` already has `validStatus`; move or re-export it as needed so the build is clean. Add `TestValidStatusMatchesTheRecordedVocabulary` to `internal/ropa/model_test.go`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/reporting/ -run "TestNormalizeReportFilter|TestReportFilterSQL" -count=1` and `go test ./internal/ropa/ -run TestValidStatus -count=1`
Expected: both PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/reporting/filter.go internal/reporting/filter_test.go internal/ropa/model.go internal/ropa/model_test.go
git commit -m "feat(reporting): enforce a closed filter vocabulary

A report filter is user input, so the field set, operator set and value
vocabulary are all closed and validated before any query runs. Every value is
bound as a parameter, and SQL generation re-checks the field itself so a
skipped normalisation step cannot inject a fragment. The status vocabulary
comes from the register rather than a copied list."
```

---

## Task 3: The exception dataset, proven equal to the register's blockers

The report and the register must not be able to disagree about what an exception is. This task makes that a tested property rather than a convention.

**Files:**
- Create: `internal/reporting/exceptions.go`
- Test: `internal/reporting/exceptions_test.go`
- Test: `internal/reporting/exceptions_postgres_test.go` (build tag `postgres`)

- [ ] **Step 1: Write the failing parity test**

```go
func TestExceptionPredicateMatchesRegisterClosureBlockers(t *testing.T) {
	// The register decides closure with ropa.ClosureBlockers. The report's
	// exception dataset must select exactly the activities that register would
	// refuse to close, or the same bank gets two different answers.
	cases := []struct {
		name     string
		activity ropa.ProcessingActivity
		excepted bool
	}{
		{"complete", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, false},
		{"missing lawful basis", ropa.ProcessingActivity{
			OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"blank lawful basis is missing", ropa.ProcessingActivity{
			LawfulBasis: "   ", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"missing owner", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"missing data subjects", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"no review at all", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
		}, true},
		{"review without completion", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{DueDate: time.Now()}},
		}, true},
		{"withdrawn review does not count", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "WITHDRAWN"}},
		}, true},
		{"blank outcome does not count", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "  "}},
		}, true},
		{"revised review counts", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "REVISED"}},
		}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			byRegister := len(ropa.ClosureBlockersForTest(testCase.activity)) > 0
			byPredicate := ExceptionPredicate(testCase.activity)
			if byRegister != testCase.excepted || byPredicate != byRegister {
				t.Fatalf("register says excepted=%v, predicate says %v, want %v",
					byRegister, byPredicate, testCase.excepted)
			}
		})
	}
}

func ptr(value time.Time) *time.Time { return &value }
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/reporting/ -run TestExceptionPredicateMatchesRegisterClosureBlockers -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Export the register's own predicate rather than restating it**

Add to `internal/ropa/service.go`, directly above `closureBlockers`:

```go
// ClosureBlockersForTest exposes the register's closure decision to other
// packages so the report's exception predicate can be proven equal to it. A
// second implementation of this rule is how a register and its report would
// start disagreeing.
func ClosureBlockersForTest(activity ProcessingActivity) []string { return closureBlockers(activity) }
```

- [ ] **Step 4: Write `internal/reporting/exceptions.go`**

```go
package reporting

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

// ExceptionBlocker describes one missing fact, in the same words the register
// shows the operator, so a report row and a register row read identically.
type ExceptionBlocker struct {
	Field   string `json:"field"`
	Missing string `json:"missing"`
}

// ExceptionBlockers returns the missing facts for one activity, delegating to
// the register so the two surfaces cannot diverge.
func ExceptionBlockers(activity ropa.ProcessingActivity) []ExceptionBlocker {
	names := ropa.ClosureBlockersForTest(activity)
	blockers := make([]ExceptionBlocker, 0, len(names))
	for _, name := range names {
		blockers = append(blockers, ExceptionBlocker{Field: blockerField(name), Missing: name})
	}
	return blockers
}

func blockerField(name string) string {
	switch name {
	case "lawful basis":
		return string(ReportFieldMissingBasis)
	case "named owner":
		return string(ReportFieldMissingOwner)
	case "data subject category":
		return string(ReportFieldMissingSubjects)
	case "completed review":
		return string(ReportFieldReviewOverdue)
	default:
		return strings.ReplaceAll(name, " ", "_")
	}
}

// ExceptionPredicate is the in-process form of the SQL predicate below. The
// parity test pins the two together.
func ExceptionPredicate(activity ropa.ProcessingActivity) bool {
	return len(ropa.ClosureBlockersForTest(activity)) > 0
}

// ExceptionPredicateSQL is the database form. Each clause mirrors one branch of
// ropa.closureBlockers exactly.
const ExceptionPredicateSQL = `(
  btrim(a.lawful_basis) = ''
  OR a.owner_principal_id IS NULL
  OR btrim(a.data_subject_categories) = ''
  OR NOT EXISTS (
       SELECT 1 FROM ropa_processing_activity_reviews r
       WHERE r.tenant_id=a.tenant_id AND r.legal_entity_id=a.legal_entity_id AND r.activity_id=a.id
         AND r.completed_at IS NOT NULL AND btrim(r.outcome) IN ('CONFIRMED','REVISED'))
)`

// ExceptionColumnSQL lists, per row, which facts are missing. The expressions
// match ExceptionPredicateSQL one for one.
const ExceptionColumnSQL = `ARRAY_REMOVE(ARRAY[
  CASE WHEN btrim(a.lawful_basis) = '' THEN 'Lawful basis not recorded' END,
  CASE WHEN a.owner_principal_id IS NULL THEN 'Owner not recorded' END,
  CASE WHEN btrim(a.data_subject_categories) = '' THEN 'Data subject categories not recorded' END,
  CASE WHEN NOT EXISTS (
       SELECT 1 FROM ropa_processing_activity_reviews r
       WHERE r.tenant_id=a.tenant_id AND r.legal_entity_id=a.legal_entity_id AND r.activity_id=a.id
         AND r.completed_at IS NOT NULL AND btrim(r.outcome) IN ('CONFIRMED','REVISED'))
       THEN 'No completed review' END], NULL)`
```

- [ ] **Step 5: Add a PostgreSQL test that proves the SQL agrees with the Go predicate**

`internal/reporting/exceptions_postgres_test.go` (tag `postgres`) inserts one activity per case from the parity table, runs `SELECT id FROM ropa_processing_activities a WHERE ` + `ExceptionPredicateSQL`, and asserts the returned set equals the Go-derived set. Static reasoning about a `NOT EXISTS` subquery is not proof; this is.

- [ ] **Step 6: Run both tests**

Run: `go test ./internal/reporting/ -run TestExceptionPredicate -count=1` and `go test ./internal/reporting/ -tags postgres -run TestExceptionPredicateSQLAgreesWithGo -count=1`
Expected: both PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/reporting/exceptions.go internal/reporting/exceptions_test.go internal/reporting/exceptions_postgres_test.go internal/ropa/service.go
git commit -m "feat(reporting): derive report exceptions from the register's own rule

A report that disagreed with the register about what is incomplete would give
one bank two different answers. The in-process predicate delegates to
ropa.closureBlockers, and a PostgreSQL test pins the SQL form to the Go form."
```

---

## Task 4: Domain model and repository interfaces

**Files:**
- Create: `internal/reporting/model.go`
- Create: `internal/reporting/repository.go`
- Test: `internal/reporting/model_test.go`

- [ ] **Step 1: Write the failing checksum and shape tests**

```go
func TestDefinitionChecksumIsStableAndCoversEveryGovernedField(t *testing.T) {
	base := ReportDefinition{
		TenantID: "t", LegalEntityID: "e", Code: "ROPA-EXCEPTIONS",
		Dataset: DatasetProcessingActivityExceptions, ScopeKind: ScopeLegalEntity,
		Format: FormatCSV, CurrentVersion: 1, MakerID: "maker-1",
		Filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN"},
		}},
	}
	first := base.Checksum()
	if len(first) != 64 {
		t.Fatalf("expected a SHA-256 hex checksum, got %q", first)
	}
	if first != base.Checksum() {
		t.Fatal("checksum must be stable for identical content")
	}
	for _, mutate := range []func(*ReportDefinition){
		func(d *ReportDefinition) { d.Dataset = DatasetProcessingActivities },
		func(d *ReportDefinition) { d.ScopeKind = ScopeProgram },
		func(d *ReportDefinition) { d.ScopeRef = "program-1" },
		func(d *ReportDefinition) { d.Format = FormatNDJSON },
		func(d *ReportDefinition) { d.Filter.Children[0].Value = "CLOSED" },
		func(d *ReportDefinition) { d.MakerID = "maker-2" },
		func(d *ReportDefinition) { d.CurrentVersion = 2 },
	} {
		changed := base
		changed.Filter = cloneFilter(base.Filter)
		mutate(&changed)
		if changed.Checksum() == first {
			t.Fatalf("checksum did not change after mutating a governed field")
		}
	}
}

func TestDefinitionStatusVocabularyIsClosed(t *testing.T) {
	for _, value := range []string{"DRAFT", "PENDING_REVIEW", "ACTIVE", "RETIRED"} {
		if !validDefinitionStatus(DefinitionStatus(value)) {
			t.Fatalf("status %q should be valid", value)
		}
	}
	if validDefinitionStatus("APPROVED") || validDefinitionStatus("") {
		t.Fatal("unlisted statuses must be rejected")
	}
}

// cloneFilter deep-copies so a mutation in the checksum table cannot leak into
// the next case and make the test pass for the wrong reason.
func cloneFilter(expression *ReportFilterExpression) *ReportFilterExpression {
	if expression == nil {
		return nil
	}
	raw, err := json.Marshal(expression)
	if err != nil {
		panic(err)
	}
	var clone ReportFilterExpression
	if err := json.Unmarshal(raw, &clone); err != nil {
		panic(err)
	}
	return &clone
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/reporting/ -run "TestDefinitionChecksum|TestDefinitionStatusVocabulary" -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `internal/reporting/model.go`**

Declare the package errors first, so every task below can use them:

```go
package reporting

import "errors"

var (
	// ErrInvalid covers a rejected request, filter, definition or transition.
	ErrInvalid = errors.New("reporting: invalid request")
	// ErrNotFound is returned for a definition or run outside the caller's scope.
	ErrNotFound = errors.New("reporting: not found")
	// ErrConflict is returned for a stale expected version or a moved scope.
	ErrConflict = errors.New("reporting: conflicting change")
	// ErrClosureBlocked is returned when a definition cannot advance because a
	// governed precondition is unmet. It is distinct from ErrInvalid so the API
	// can explain which precondition failed.
	ErrClosureBlocked = errors.New("reporting: blocked by a governed precondition")
)
```

Then define the closed vocabularies and the structs:

```go
type DefinitionStatus string

const (
	DefinitionDraft        DefinitionStatus = "DRAFT"
	DefinitionPendingReview DefinitionStatus = "PENDING_REVIEW"
	DefinitionActive       DefinitionStatus = "ACTIVE"
	DefinitionRetired      DefinitionStatus = "RETIRED"
)

func validDefinitionStatus(status DefinitionStatus) bool {
	switch status {
	case DefinitionDraft, DefinitionPendingReview, DefinitionActive, DefinitionRetired:
		return true
	}
	return false
}

type RunStatus string

const (
	RunQueued  RunStatus = "QUEUED"
	RunRunning RunStatus = "RUNNING"
	RunReady   RunStatus = "READY"
	RunFailed  RunStatus = "FAILED"
)

type ReportDataset string

const (
	DatasetProcessingActivities         ReportDataset = "PROCESSING_ACTIVITIES"
	DatasetProcessingActivityExceptions ReportDataset = "PROCESSING_ACTIVITY_EXCEPTIONS"
	DatasetPrograms                      ReportDataset = "PROGRAMS"
	DatasetMatterExceptions              ReportDataset = "MATTER_EXCEPTIONS"
)

type ReportScopeKind string

const (
	ScopeLegalEntity ReportScopeKind = "LEGAL_ENTITY"
	ScopeProgram      ReportScopeKind = "PROGRAM"
	ScopeMatter       ReportScopeKind = "MATTER"
)

type ReportFormat string

const (
	FormatCSV   ReportFormat = "CSV"
	FormatNDJSON ReportFormat = "NDJSON"
)
```

Add `ReportDefinition` (one field per `report_definitions` column, snake_case JSON tags), `ReportDefinitionRevision` (one field per `report_definition_revisions` column), `DecisionRecord` (actor, action, note, checksum seen, timestamp) and `ReportRun` (one field per `report_runs` column).

Then the checksum:

```go
// Checksum binds every governed field of a definition. Approval is granted
// against this value, so a definition that changes after approval must produce
// a different checksum or the approval would silently cover new content.
func (d ReportDefinition) Checksum() string {
	filter, _ := json.Marshal(d.Filter)
	payload := strings.Join([]string{
		d.TenantID, d.LegalEntityID, d.Code, d.Name, d.Description,
		string(d.Dataset), string(d.ScopeKind), d.ScopeRef, string(d.Format),
		string(filter), strconv.Itoa(d.CurrentVersion), d.MakerID,
	}, "\x1f")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
```

Also add the manifest type and the run limits:

```go
// Manifest is written beside every artefact. It states what was asked for,
// what was actually read, and whether the population was complete, so a
// reader never has to infer completeness from a row count.
type Manifest struct {
	Schema             string                `json:"schema"`
	GeneratedAt        time.Time             `json:"generated_at"`
	AsOf               time.Time             `json:"as_of"`
	Source             SourceBoundary        `json:"source"`
	DefinitionCode     string                `json:"definition_code"`
	DefinitionVersion  int                   `json:"definition_version"`
	DefinitionChecksum string                `json:"definition_checksum"`
	Dataset            ReportDataset         `json:"dataset"`
	ScopeKind          ReportScopeKind       `json:"scope_kind"`
	ScopeRef           string                `json:"scope_ref,omitempty"`
	RowCount           int                   `json:"row_count"`
	PopulationComplete bool                  `json:"population_complete"`
	Filter             *ReportFilterExpression `json:"filter,omitempty"`
	Coverage           ManifestCoverage      `json:"coverage"`
	DataSHA256         string                `json:"data_sha256"`
	RetentionUntil     time.Time             `json:"retention_until"`
}

// ManifestCoverage is never a persuasive number. A count the run could not
// establish is omitted, so a reader sees its absence rather than a zero.
type ManifestCoverage struct {
	Population int  `json:"population"`
	Excluded   *int `json:"excluded,omitempty"`
	Unknown    *int `json:"unknown,omitempty"`
}

const (
	// The bounded envelope is the one the approved design names
	// (docs/superpowers/specs/2026-09-23-ropa-register-and-reporting-design.md):
	// 100-row pages, 10,000-row ceiling, 32 MiB, 7-day retention, matching
	// internal/activity/export.go. Do not enlarge it without a design change.
	ReportRunPageSize  = 100
	MaxReportRunRows   = 10_000
	MaxReportRunBytes  = int64(32 << 20)
	ReportRunRetention = 7 * 24 * time.Hour
	MaxReportRunLease  = 2 * time.Minute
	// MaxReportRunTries is the durable retry budget held in report_runs.attempt_count.
	// It is NOT WorkClassOptions.MaxAttempts: a custom worker maintainer receives
	// only (now, batch) and gets no per-item lease or durable attempt count, so
	// WorkClassOptions cannot be relied on here. The run row is the authority.
	MaxReportRunTries  = 5
	reportManifestSchema = "clearsight.report-run.v1"
)
```

The manifest must also record the source boundary. `as_of` alone is an upper time bound, not a reproducible snapshot — the register has no version to compare it against, so a reader could not tell whether a rerun would produce the same rows:

```go
// SourceBoundary is captured before generation and persisted even if generation
// later fails, so a reader can tell which material versions the report was read
// from. Without it, "as of" is a timestamp rather than a reconstruction point.
type SourceBoundary struct {
	CapturedAt          time.Time          `json:"captured_at"`
	ProjectionVersion   string             `json:"projection_version"`
	SourceHighWater     map[string]time.Time `json:"source_high_water"`
	Population          int                `json:"population"`
	PopulationComplete  bool               `json:"population_complete"`
}
```

- [ ] **Step 4: Write `internal/reporting/repository.go`**

```go
// ReportScope is mandatory on every exact read and write. There is no
// tenant-only path: a report belongs to one legal entity.
type ReportScope struct {
	TenantID      string
	LegalEntityID string
}

type DefinitionRepository interface {
	CreateDefinition(ctx context.Context, scope ReportScope, definition ReportDefinition, revision ReportDefinitionRevision) (ReportDefinition, error)
	GetDefinition(ctx context.Context, scope ReportScope, id string) (ReportDefinition, error)
	GetDefinitionByCode(ctx context.Context, scope ReportScope, code string) (ReportDefinition, error)
	ListDefinitions(ctx context.Context, scope ReportScope, includeRetired bool) ([]ReportDefinition, error)
	ListDefinitionHistory(ctx context.Context, scope ReportScope, id string) ([]ReportDefinitionRevision, error)
	TransitionDefinition(ctx context.Context, scope ReportScope, id string, expectedVersion int64, next DefinitionStatus, decision DecisionRecord) (ReportDefinition, error)
}

type RunRepository interface {
	CreateRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error)
	GetRun(ctx context.Context, scope ReportScope, id string) (ReportRun, error)
	ListRuns(ctx context.Context, scope ReportScope, definitionID string, limit int) ([]ReportRun, error)
	ClaimQueuedRuns(ctx context.Context, workerID string, limit int) ([]ReportRun, error)
	CompleteRun(ctx context.Context, run ReportRun) (ReportRun, error)
	FailRun(ctx context.Context, scope ReportScope, id, failureCode string) (ReportRun, error)
	RecordRunDownload(ctx context.Context, scope ReportScope, id, downloadedBy string) error
}
```

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/reporting/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/reporting/model.go internal/reporting/repository.go internal/reporting/model_test.go
git commit -m "feat(reporting): add the report definition and run domain model

The definition checksum covers every governed field, so an approval cannot
silently extend to content it did not see. Every exact read and write carries
a tenant and legal-entity scope; there is no tenant-only path."
```

---

## Task 5: The service — lifecycle, one transition gate, bounded runs

**Files:**
- Create: `internal/reporting/service.go`
- Test: `internal/reporting/service_test.go`

- [ ] **Step 1: Write the failing lifecycle tests**

Cover, at minimum, each of these as a named test:

- `TestProposeDefinitionIgnoresActorFieldsFromTheRequestBody` — a `maker_id` in the body is overwritten from verified context, never trusted.
- `TestSubmitRequiresADraftDefinition` and `TestSubmitRejectsAMissingAuthorityRoute`.
- `TestReviewRefusesWhenTheReviewerIsTheMaker` — maker-checker separation is enforced in the service, not only by the database constraint.
- `TestReviewRefusesWhenTheDefinitionChangedSinceTheReviewerSawIt` — the decision carries the checksum the reviewer reviewed; a mismatch is refused.
- `TestReviewRecordsAReviewerDistinctFromBothMakerAndAuthorizer` — the approved design keeps responsibility, review and authorization distinct, so the three principals must all differ.
- `TestActivateRefusesUnlessAReviewWasRecorded` — activation without a recorded review is refused. There is no path that skips review.
- `TestActivateRefusesWhenTheAuthorizerIsTheMakerOrTheReviewer` — three-way separation.
- `TestActivateMarksAFutureEffectiveDateAsNotYetCurrent` — an `effective_from` in the future yields `ACTIVE` with a not-yet-effective marker, not a current report.
- `TestRetireIsReversibleByANewRevision` — retiring version N leaves version N reconstructable and allows a fresh proposal.
- `TestValidateTransitionForWriteIsTheOnlyTransitionGate` — a structural test asserting that `service.go` contains no second switch over `DefinitionStatus` outside `ValidateTransitionForWrite`.
- `TestCreateRunRejectsAnUnapprovedDefinition`
- `TestCreateRunRejectsARetiredDefinition`
- `TestCreateRunStopsAtTheRowBound` — 10,001 matching rows produces a `FAILED` run with `failure_code = "row_limit_exceeded"`, not a truncated artefact silently presented as complete.
- `TestCreateRunStopsAtTheByteBound`
- `TestCreateRunRejectsACrossEntityDefinition` — a definition from another legal entity cannot be run.
- `TestCreateRunPersistsTheSourceBoundaryBeforeGeneration` — the source boundary is written with the request, so a run that fails to generate still records what it was reading.
- `TestOpenRunAuthorisesOnEveryDownload` and `TestOpenRunRejectsAnExpiredRun`.
- `TestOpenRunRejectsARunFromAnotherLegalEntity` — tenant-only matching is not enough; a same-tenant second entity's run must not open.
- `TestRunManifestRecordsTheSourceBoundaryAndPopulation` — the manifest states the definition version, the source high-water, the row count and whether the population was complete.
- `TestRunArtefactKeyIsScopedAndOpaque` — the key contains no activity name, owner or other row content.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/reporting/ -run "TestProposeDefinition|TestApprove|TestCreateRun|TestOpenRun|TestRunManifest|TestRunArtefactKey|TestValidateTransition" -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `internal/reporting/service.go`**

Structure:

```go
// Service owns every report command. Production command actors come from the
// verified request identity supplied through WithActor; any actor field in a
// request body is ignored.
type Service struct {
	repo        Repository
	runs        RunRepository
	objects     evidence.ObjectStore
	authority   AuthorityChecker
	Now         func() time.Time
	WorkerID    string
}

type AuthorityChecker interface {
	// ResolveReportProposer / ResolveReportReviewer re-evaluate the current
	// versioned authority route. A missing route or a service failure must fail
	// closed in production rather than falling back to a default approver.
	ResolveReportProposer(ctx context.Context, scope ReportScope) (string, error)
	ResolveReportReviewer(ctx context.Context, scope ReportScope) (string, error)
}
```

Implement, in this order:

1. `ValidateTransitionForWrite(current ReportDefinition, next DefinitionStatus) error` — the single state machine, with a `switch` that is the only place `DefinitionStatus` transitions are decided. Call it from both the service and the repository.
2. `Propose`, `Submit`, `Review`, `Activate`, `Reject`, `Retire`. Each takes the verified actor from context, re-resolves authority, calls `ValidateTransitionForWrite`, and delegates to `TransitionDefinition` in one transaction.

   The lifecycle has **four distinct responsibilities**, per the approved design's governance table — a reviewer is not the authorizer:

   ```
   DRAFT --submit(PROPOSER)--> PENDING_REVIEW
   PENDING_REVIEW --review(REVIEWER)--> REVIEWED
   REVIEWED --activate(AUTHORIZER)--> ACTIVE
   any --retire(AUTHORIZER)--> RETIRED
   ```

   Add `DefinitionStatusReviewed`. Activation requires a recorded review whose reviewer differs from both the maker and the authorizer. There is no transition that skips review.

   **Decisions are report-owned.** `governance_decisions.object_type` is a closed check that permits only `ROUTING_POLICY`, `DELEGATION`, `SEGREGATION_RULE`, `SCIM_SOURCE` and `DIRECTORY_GROUP_ROLE_BINDING`. Do not widen it and do not insert report rows into it. Record the decision in `report_definition_revisions` (`decision`, `decision_note`, `approved_by`, `approved_at`) plus the revision checksum, which is why migration 000093 gives that table those columns.

3. `CreateRun` — resolves the definition by exact id and scope, requires `ACTIVE`, effective and reviewed, captures the `SourceBoundary`, and writes a `QUEUED` run carrying the filter, the dataset, the definition version and the definition checksum. It does not render inline; a worker does that. Generation is asynchronous by design, and unlike `internal/activity/export.go` this service must not generate inside the HTTP request.
4. `ExecuteRun(ctx, run ReportRun) (ReportRun, error)` — claims the run through `ClaimQueuedRuns`, pages the dataset with `ReportPageSQL()` in `ReportRunPageSize` chunks, renders CSV or NDJSON, checks `MaxReportRunRows` and `MaxReportRunBytes` before writing, stores the artefact and manifest through `objects.Put`, computes SHA-256 over the exact bytes written, then `CompleteRun`. On any bound breach it calls `FailRun` with a specific code and never writes a partial artefact.

   The durable retry budget is `report_runs.attempt_count`, not `WorkClassOptions.MaxAttempts`. `internal/runtime/work_class.go:251-262` calls a custom maintainer with only `(now, batch)`, so a maintainer gets no per-item lease and no durable attempt count. The run row is the authority; the worker class is only the polling loop.

5. `Manifest` — a struct carrying `schema`, `generated_at`, `as_of`, the `SourceBoundary`, `definition_code`, `definition_version`, `definition_checksum`, `dataset`, `scope`, `row_count`, `population_complete`, `filter`, `coverage`, `retention_until` and `data_sha256`.
6. `Open(ctx, scope, runID, downloadedBy) (ReportRun, io.ReadCloser, error)` — re-reads the run under the full `ReportScope`, rejects `QUEUED`/`RUNNING`/`FAILED`/expired, records the download, then opens the object.

   The download path must be stricter than `internal/httpapi/audit_export_handlers.go`, which matches on tenant only and resolves no authority. This one verifies the exact legal entity, re-resolves the current download authority through `AuthorityChecker`, and verifies the object's SHA-256 against the persisted digest before returning bytes — the pattern at `internal/evidence/artifact_open.go:21-41`.

Key detail for `ExecuteRun` — bounds are checked before commit, not after:

```go
// The row and byte bounds are checked while streaming. A run that reaches a
// bound is failed with a specific code, so an operator sees why the report
// stopped rather than receiving a truncated file that looks complete.
if rows > MaxReportRunRows {
	return s.failRun(ctx, run, "row_limit_exceeded")
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/reporting/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/reporting/service.go internal/reporting/service_test.go
git commit -m "feat(reporting): govern definitions and bound every report run

Definitions move through one transition gate called from both the service and
the repository, with maker-checker separation, effective dating and a
checksum-bound decision. Runs stop at an explicit row or byte bound and fail
with a named code rather than producing a truncated artefact that looks
complete."
```

---

## Task 6: PostgreSQL repository and the bounded report query

**Files:**
- Create: `internal/reporting/postgres.go`
- Create: `internal/reporting/report_postgres.go`
- Test: `internal/reporting/postgres_test.go` (build tag `postgres`)

- [ ] **Step 1: Write the failing PostgreSQL tests**

- `TestCreateDefinitionWritesCurrentAndRevisionInOneTransaction` — force the revision insert to fail and assert no current row survives.
- `TestTransitionDefinitionRejectsAStaleExpectedVersion` — concurrent transition attempt loses.
- `TestTransitionDefinitionWritesRevisionDecisionAndOutboxTogether`
- `TestClaimQueuedRunsLeasesExactlyOnce` — two workers, one claim each row.
- `TestClaimQueuedRunsDoesNotClaimATerminalRun`
- `TestCompleteRunRefusesWithoutArtefacts` — the database guard rejects a `READY` transition with no object keys.
- `TestReportPageSQLReturnsEachRowOnceAcrossPages` — 1,200 activities paged at 500 yields 1,200 distinct ids and no duplicates at a page boundary.
- `TestReportPageSQLAppliesScopeInsideThePage` — a second legal entity's rows never appear.
- `TestReportPageSQLAppliesTheFilterBeforeTheLimit`
- `TestExceptionDatasetReturnsOnlyExceptedActivities`
- `TestTransitionDefinitionRejectsACrossTenantDefinitionID` — a same-shaped id from another tenant returns not-found, not another tenant's row.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/reporting/ -tags postgres -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write `internal/reporting/report_postgres.go`**

`ReportPageSQL()` must put every scope and filter predicate **inside** the materialised page CTE, before its `LIMIT`, exactly as `internal/ropa/summaries_postgres.go:13` does. It returns a query with a `%s` placeholder for the filter fragment and takes the dataset's `WHERE` clause as its first return value, so the SQL predicate from `exceptions.go` is composed rather than restated:

```go
// ReportPageSQL returns one bounded page of report rows. The caller supplies,
// in order: tenant ID, legal-entity ID, dataset marker, the scope reference
// (empty for a legal-entity-wide report), the validated filter's bound
// arguments, has-cursor, the cursor id and limit+1.
//
// Scope and filter predicates sit inside the materialised page CTE before its
// limit, so a page can never be filled with rows the caller may not see. The
// keyset predicate reuses the same status-rank, review-date and id expressions
// as migration 000092's ropa_register_keyset_idx, so this query rides the Tranche 1 index.
func ReportPageSQL(datasetClause, filterFragment string) string {
	return fmt.Sprintf(`
WITH page AS MATERIALIZED (
  SELECT a.id::text,
         a.tenant_id::text,
         a.legal_entity_id::text,
         a.code,
         a.name,
         a.status,
         a.purpose,
         a.lawful_basis,
         a.controller,
         a.processor,
         a.automated_decision_making,
         a.data_subject_categories,
         a.personal_data_categories,
         a.retention_period,
         a.security_measures,
         a.next_review_date,
         COALESCE(a.owner_principal_id::text, ''),
         COALESCE(a.program_id::text, ''),
         COALESCE(a.matter_id::text, ''),
         a.version,
         a.updated_at,
         %s AS exceptions
  FROM ropa_processing_activities a
  WHERE a.tenant_id = $1::uuid
    AND a.legal_entity_id = $2::uuid
    AND ($3 = '' OR a.status::text = $3)
    AND ($4 = '' OR a.program_id = $4::uuid OR a.matter_id = $4::uuid)
    AND (%s)
  ORDER BY
    (CASE a.status WHEN 'NEW' THEN 0 WHEN 'OPEN' THEN 1 ELSE 2 END),
    COALESCE(a.next_review_date, 'infinity'::timestamptz),
    a.id
  LIMIT $%d
)
SELECT * FROM page
WHERE (
  $%d = false OR
  (CASE page.status WHEN 'NEW' THEN 0 WHEN 'OPEN' THEN 1 ELSE 2 END),
   COALESCE(page.next_review_date, 'infinity'::timestamptz),
   page.id
) > ($%d, $%d, $%d::text)
ORDER BY
  (CASE page.status WHEN 'NEW' THEN 0 WHEN 'OPEN' THEN 1 ELSE 2 END),
  COALESCE(page.next_review_date, 'infinity'::timestamptz),
  page.id
LIMIT $%d`, datasetClause, filterFragment)
}
```

> The bound-argument positions must line up with the filter's own `$n` numbering. Build the filter fragment with `ReportFilterSQL(expression, 6)` so the first filter argument is `$6`, then append the cursor and limit positions after the filter's argument count. Write a test that asserts this alignment rather than trusting it.

`datasetClause` is `ExceptionColumnSQL` for the exception dataset and `ARRAY[]::text[]` for the full-activities dataset. The `$3 = ''` and `$4 = ''` forms let a legal-entity-wide report and an unknown scope share one prepared statement, which keeps the plan index-only on the keyed columns.

- [ ] **Step 4: Write `internal/reporting/postgres.go`**

Implement both interfaces. `TransitionDefinition` is one transaction:

```sql
BEGIN;
SELECT status, current_version, checksum, version, maker_id, dataset, scope_kind,
       scope_ref, format, filter, legal_entity_id
  FROM report_definitions
 WHERE id=$1 AND tenant_id=$2 AND legal_entity_id=$3
 FOR UPDATE;
-- reject a stale expected version, then call ValidateTransitionForWrite in Go
INSERT INTO report_definition_revisions (...) VALUES (...);
UPDATE report_definitions SET status=..., checker_id=..., ... , version=version+1 WHERE ...;
INSERT INTO outbox_events (...) VALUES (...);
COMMIT;
```

Assert the row count on every `UPDATE`. A zero-row update under `FOR UPDATE` means the scope or version moved; return not-found or a conflict rather than reporting success.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/reporting/ -tags postgres -count=1` and `go test ./internal/reporting/ -count=1`
Expected: both PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/reporting/postgres.go internal/reporting/report_postgres.go internal/reporting/postgres_test.go
git commit -m "feat(reporting): add the PostgreSQL repository and bounded report query

Scope and filter predicates sit inside the materialised page CTE before its
limit, and the keyset predicate reuses the Tranche 1 index expressions. A
definition transition writes its revision, decision and outbox message in one
transaction, and asserts the row count so a moved scope cannot look like a
success."
```

---

## Task 7: Program and Matter datasets

The approved design states that reports "read Program, Matter and ROPA", and the original ask was reports for exceptions and compliance **for a program/matter**. The ROPA datasets alone do not deliver that. This task adds the two non-ROPA datasets.

**Files:**
- Create: `internal/reporting/program_matter_postgres.go`
- Create: `internal/reporting/program_matter_fields.go`
- Test: `internal/reporting/program_matter_postgres_test.go` (build tag `postgres`)

- [ ] **Step 1: Write the failing tests**

- `TestProgramReportPageAppliesScopeBeforeTheLimit`
- `TestMatterReportPageAppliesVisibilityBeforeTheLimit` — the decisive one. A `RESTRICTED` Matter the verified principal cannot see must not consume a page slot. Mirror `internal/continuity/matter_summary_visibility_postgres_integration_test.go:15-124`, and include the malformed-policy cases: an `access` value that is not `PUBLIC`/`INTERNAL`/`RESTRICTED`, a non-array `allowed_principal_ids`, and an array with no non-blank principal. All three must hide the row, not admit it.
- `TestMatterReportPageNeverReturnsAnotherLegalEntitysMatter`
- `TestProgramReportCarriesItsCalculatedStateAndVersion` — the row states the calculated `overall_state`, the `assessed_program_version`, the `projection_version` and whether the projection is stale. A report that omits the version gives a reader no way to know what it read.
- `TestMatterExceptionDatasetSelectsOpenExceptionsAndOverdueObligations`
- `TestProgramAndMatterPagesAreKeysetStableAcrossPageBoundaries` — no duplicate or skipped row at a boundary.
- `TestProgramReportReasonsOmitIsNotSilentlyZero` — `internal/continuity/summaries_postgres.go` truncates reasons to six and returns `ReasonsOmitted`. The report must carry that count, not imply six was all of them.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/reporting/ -tags postgres -run "TestProgramReport|TestMatterReport|TestMatterException|TestProgramAndMatterPages" -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Add the allow-listed Program and Matter filter fields**

Extend `ReportFilterFieldVocabulary` in `internal/reporting/filter.go` with a second group, and give each field its own SQL fragment. Use the vocabulary the source already records:

- Program: `status`, `owner_principal_id`, `overall_state` (`CURRENT`, `AT_RISK`, `GAP_IDENTIFIED`, `EVIDENCE_INSUFFICIENT`, `IMPLEMENTATION_PENDING`, `OVERDUE`, `UNDER_REVIEW`, `NOT_APPLICABLE`, `UNKNOWN`), `jurisdiction`, `has_open_matters`.
- Matter: `status`, `owner_principal_id`, `matter_type`, `priority`, `due_condition` (`NO_DUE_DATE`, `OVERDUE`, `DUE_7_DAYS`, `DUE_30_DAYS`), `program`, `latest_verification_result` (`PASS`, `FAIL`, `INCONCLUSIVE`).

Validate every value against that closed list in `normalizeReportFilterValue`, and map every field to an indexed or already-computed column. Reject a field that belongs to the other dataset rather than silently ignoring it.

- [ ] **Step 4: Write the two bounded queries**

`ProgramReportPageSQL()` and `MatterReportPageSQL()` follow `internal/continuity/summaries_postgres.go` exactly:

- tenant and legal-entity scope in the main `WHERE`;
- for Matters, the visibility predicate **inside the `WHERE` clause before `ORDER BY` and `LIMIT`**, using the same fail-closed rules as `internal/continuity/access.go:74-91`;
- allow-listed filter predicates inside the page CTE before its `LIMIT`;
- `LIMIT limit+1` with a keyset cursor;
- the Program keyset is `(status rank, updated_at, id)`; the Matter keyset is `(priority, updated_at, id)`.

The Matter visibility predicate is the security-critical part. Reproduce the fail-closed rules exactly; do not approximate them, and do not fetch a broad Matter population and filter in Go — `AGENTS.md` forbids a broad data load followed by application-memory authorization.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/reporting/ -tags postgres -count=1` and `go test ./internal/reporting/ -count=1`
Expected: both PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/reporting/program_matter_postgres.go internal/reporting/program_matter_fields.go internal/reporting/program_matter_postgres_test.go internal/reporting/filter.go
git commit -m "feat(reporting): add Program and Matter report datasets

The approved design reads Program, Matter and ROPA. Matter visibility is
applied inside the query before its limit, with the same fail-closed rules as
the register, so a restricted row the principal cannot see never consumes a
page slot. Program rows carry their calculated state, version and projection
staleness so a reader knows what was read."
```

---

## Task 8: Memory repository and demo data

**Files:**
- Create: `internal/reporting/memory.go`
- Test: `internal/reporting/memory_test.go`

- [ ] **Step 1: Write the failing parity test**

```go
func TestMemoryRepositoryEnforcesTheSameRulesAsPostgres(t *testing.T) {
	// The memory repository backs the demo. If it is laxer than PostgreSQL the
	// demo shows behaviour production cannot produce.
	// - a stale expected version is refused
	// - maker and checker cannot be the same principal
	// - a READY run without artefacts is refused
	// - a cross-entity definition id is not found
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/reporting/ -run TestMemoryRepositoryEnforcesTheSameRulesAsPostgres -count=1`
Expected: FAIL.

- [ ] **Step 3: Write `internal/reporting/memory.go`**

In-memory implementations of both interfaces calling `ValidateTransitionForWrite` and enforcing the same constraints, plus `NewMemoryObjectStore` reuse from `internal/evidence`. Add a `Demo` installer mirroring `internal/ropa/demo.go`:

- **"Processing activities with open exceptions"** — `PROCESSING_ACTIVITY_EXCEPTIONS`, legal-entity scope, `ACTIVE` with an effective date in the past.
- **"Cross-border transfers in one program"** — `PROCESSING_ACTIVITIES`, program scope, filter `cross_border_transfer is true`, `ACTIVE`.
- **"Overdue obligations in one issue or change"** — `MATTER_EXCEPTIONS`, matter scope, filter `due_condition is overdue`, `PENDING_REVIEW`.
- **"Program health across the entity"** — `PROGRAMS`, legal-entity scope, `REVIEWED` so the workspace shows a definition that has been reviewed but not yet activated.

The four states are the point of the sample set: the workspace has to be able to show a definition that is ready to run, one that is waiting for a review, and one that has been reviewed but not yet authorised. One definition alone would hide the governance.

Human working language. Realistic bank owners. Clearly labelled sample data. Never imply the connected bank is compliant. `Code` values like `ROPA-OPEN-EXCEPTIONS`, `ROPA-CROSS-BORDER-TRANSFERS`, `ISSUES-OVERDUE-OBLIGATIONS`, `PROGRAM-HEALTH`.

Also add a demo run that **failed on a bound** (`failure_code: "row_limit_exceeded"`, `row_count: 0`, no artefact keys), so the bounded-stop state is reachable in the demo and not only in a test. A demo that only shows successful runs teaches an operator that a stopped run is not a thing that happens.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/reporting/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/reporting/memory.go internal/reporting/memory_test.go
git commit -m "feat(reporting): add the memory repository and sample definitions

The memory repository enforces the same rules as PostgreSQL so the demo
cannot show behaviour production cannot produce. The sample set covers one
active exception report and one definition awaiting approval."
```

---

## Task 9: HTTP routes, handlers, composition and OpenAPI

**Files:**
- Create: `internal/httpapi/reporting_routes.go`
- Create: `internal/httpapi/reporting_handlers.go`
- Test: `internal/httpapi/reporting_routes_test.go`
- Test: `internal/httpapi/reporting_handlers_test.go`
- Modify: `internal/httpapi/route_registry.go` (call `a.reportingRoutes()`)
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `api/runtime.openapi.json`

- [ ] **Step 1: Write the failing route and handler tests**

- `TestReportingRoutesAreRegistered` — every path in `reportingRoutes()` appears in the registry.
- `TestForgedScopeInTheRequestBodyIsOverwritten` — a `tenant_id`/`legal_entity_id` in the body is replaced by verified context for propose, submit, approve, reject, retire and run.
- `TestRunDownloadReAuthorises` — a caller who could read the list cannot download unless they also hold the download permission.
- `TestRunDownloadSendsNoStoreAndContentDisposition`
- `TestRunDownloadRefusesAnExpiredRunWithAnExplanation`
- `TestFilterVocabularyEndpointIsPublished` — the UI can discover the closed vocabulary from the server rather than hard-coding it.
- `TestUnknownFilterFieldIsRejectedWithA400NamingTheField` — the error must tell the operator what was wrong and which fields are available, not "invalid request".

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/httpapi/ -run "TestReporting|TestForgedScope|TestRunDownload|TestFilterVocabulary|TestUnknownFilterField" -count=1`
Expected: FAIL.

- [ ] **Step 3: Write `internal/httpapi/reporting_routes.go`**

```go
func (a *API) reportingRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/ropa/reports/filter-fields", a.listReportFilterFields),
		read("/api/v1/ropa/reports/definitions", a.listReportDefinitions),
		material("/api/v1/ropa/reports/definitions", "report.definition.propose", a.proposeReportDefinition, commandPolicy{
			ObjectType: "REPORT_DEFINITION", Responsibility: authority.ResponsibilityProposer,
			Materiality: 4, BindLegalEntity: true, ActorField: "maker_id",
		}),
		read("/api/v1/ropa/reports/definitions/{id}", a.getReportDefinition),
		read("/api/v1/ropa/reports/definitions/{id}/history", a.getReportDefinitionHistory),
		material("/api/v1/ropa/reports/definitions/{id}/submit", "report.definition.submit", a.reportDefinitionAction("submit"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityProposer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/review", "report.definition.review", a.reportDefinitionAction("review"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/activate", "report.definition.activate", a.reportDefinitionAction("activate"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/reject", "report.definition.reject", a.reportDefinitionAction("reject"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/retire", "report.definition.retire", a.reportDefinitionAction("retire"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}),
		read("/api/v1/ropa/reports/runs", a.listReportRuns),
		material("/api/v1/ropa/reports/runs", "report.run.create", a.createReportRun, commandPolicy{
			ObjectType: "REPORT_RUN", Responsibility: authority.ResponsibilityPerformer, Materiality: 3, BindLegalEntity: true,
		}),
		read("/api/v1/ropa/reports/runs/{id}", a.getReportRun),
		withPermission(read("/api/v1/ropa/reports/runs/{id}/download", a.downloadReportRun), identity.PermissionReportDownload),
	}
}
```

`read()` takes no permission, so the download route is registered with `withPermission` directly. The approved design calls for a "separate export authority" for downloading a protected artefact, and `internal/identity/model.go:17-25` has no report-specific capability. Add one:

- `identity.PermissionReportDownload = "REPORT_DOWNLOAD"` in `internal/identity/model.go`, deliberately **not** `PermissionAuditExport`. `AUDIT_EXPORT` is the system-activity export capability and a data-privacy role does not hold it today; sharing it would mean a privacy officer gains system-activity export by gaining reports, and would let a report download be authorised by a route permission meant for something else.
- Grant it in `developmentPermissions` in `internal/identity/authenticator.go` to the roles that should hold it. Decide explicitly and say which in the PR body — the candidates are `GRC_ADMIN` and a data-privacy role if one exists.
- Adding a capability adds no durable table, so it needs no schema-ownership row.

Responsibilities stay distinct and none is hard-coded: submit is `PROPOSER`, review is `REVIEWER`, activate and retire are `AUTHORIZER`, run is `PERFORMER`. All five re-resolve through `commandauth.Guard`, which fails closed on a missing, ambiguous or unavailable route.

- [ ] **Step 4: Write `internal/httpapi/reporting_handlers.go`**

For each body-decoding handler, overwrite the scope and actor from verified context before calling the service, reusing the existing `bindJSONIdentity`/`WithActor` helper. The download handler must send `Cache-Control: no-store`, a `Content-Disposition` built from the definition code, and the artefact's `text/csv` or `application/x-ndjson` media type. An unknown filter field returns 400 with the rejected field name and the available vocabulary.

- [ ] **Step 5: Wire composition**

- `cmd/api/services_postgres.go`: construct the repository, object store, authority checker and service; assign to the API dependency struct.
- `cmd/api/services_memory.go`: same, plus the demo installer under the existing `cfg.DemoMode` guard only, matching how `ropa.InstallDemo` is called. Return the install error; do not swallow it.
- `cmd/worker/services_postgres.go`: add a `reportRunClass` constant, `service.ConfigureClass(reportRunClass, workflowruntime.WorkClassOptions{Poll: 15 * time.Second, Batch: 5, Timeout: ReportRunLease, Lease: ReportRunLease})`, and `service.AddMaintainerClass(reportRunClass, reporting.NewRunMaintainer(...))`.
- `internal/httpapi/route_registry.go`: append `a.reportingRoutes()`.
- `api/runtime.openapi.json`: add every new operation. A parity test asserts route count matches the OpenAPI operation count.

- [ ] **Step 6: Run to verify pass**

Run: `go test ./internal/httpapi/ -count=1` and `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/httpapi/reporting_routes.go internal/httpapi/reporting_handlers.go internal/httpapi/reporting_routes_test.go internal/httpapi/reporting_handlers_test.go internal/httpapi/route_registry.go cmd/api/services_postgres.go cmd/api/services_memory.go cmd/worker/services_postgres.go api/runtime.openapi.json
git commit -m "feat(reporting): expose report definitions and runs over HTTP

Proposer, reviewer and authorizer stay distinct responsibilities and both
resolve through the authority service, so no approver is hard-coded. Scope
and actor fields in a request body are overwritten from verified context, and
the download re-authorises on every request."
```

---

## Task 10: Documentation — schema ownership, requirement coverage, performance

**Files:**
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: `docs/product/archer-requirements-gap.md`
- Modify: `docs/engineering/ui-use-case-acceptance-matrix.md`
- Modify: `docs/quality/performance-test-plan.md`
- Modify: `docs/README.md`
- Create: `internal/reporting/load_test.go` (build tag `load`)

- [ ] **Step 1: Add the load test**

Build 100,000 processing activities with a realistic exception mix, then report the p50 and p95 for a first page and a cursor page, for both datasets, at 1440-wide equivalent filter selectivity. Budget: **750 ms**, matching the Tranche 1 register budget.

```go
//go:build load
```

- [ ] **Step 2: Run it and record the real numbers**

Run: `go test ./internal/reporting/ -tags load -run TestReportPageBudget -count=1 -v`
Expected: PASS with measured timings printed.

- [ ] **Step 3: Add ownership rows**

One row per new table, matching the existing seven-column format exactly, between the `<!-- schema-ownership:begin -->` and `<!-- schema-ownership:end -->` markers, plus a T7 section describing the boundary and what the tranche does not implement. Run the executable schema-ownership test.

- [ ] **Step 4: Correct the requirement coverage**

`docs/product/archer-requirements-gap.md` must record #26 as moving from Partial to its honest new state, naming exactly what is delivered (governed definition, closed filter vocabulary, bounded run, protected download, UI) and what is not (scheduling, recurring reports, cross-entity, streaming). Do not claim the requirement is complete if the Archer workbook asks for something not built. `docs/engineering/ui-use-case-acceptance-matrix.md` gains the report workspace rows. Update `docs/README.md` if a new document is added.

- [ ] **Step 5: Commit**

```bash
git add docs/architecture/durable-schema-ownership.md docs/product/archer-requirements-gap.md docs/engineering/ui-use-case-acceptance-matrix.md docs/quality/performance-test-plan.md docs/README.md internal/reporting/load_test.go
git commit -m "docs(reporting): record ownership, coverage and measured report budgets"
```

---

## Task 11: Web workspace — types, client, UI

**Files:**
- Create: `web/src/reportingTypes.ts`
- Create: `web/src/reportingApi.ts`
- Create: `web/src/components/ReportFilterEditor.tsx`
- Create: `web/src/components/ReportingPage.tsx`
- Modify: `web/src/components/ropa.css`
- Modify: `web/src/routing.ts` (or wherever `#ropa/reports` is dispatched)
- Modify: `web/src/components/RopaRegisterPage.tsx` (link to the workspace)
- Test: `web/src/reportingApi.test.ts`
- Test: `web/src/components/ReportFilterEditor.test.tsx`
- Test: `web/src/components/ReportingPage.test.tsx`

- [ ] **Step 1: Write the failing tests**

- `TestReportingPageListsDefinitionsWithTheirGovernanceState` — each row shows status, scope, dataset, maker, reviewer, authorizer, effective date and current version. No row shows a bare API status code as its primary label.
- `TestReportingPageShowsWhyARunCannotStartYet` — a `DRAFT`, `PENDING_REVIEW` or `REVIEWED` definition's run control is disabled and the reason names the missing step: a review, or an authorisation. A definition that is merely "not active" is not an explanation.
- `TestReportingPageShowsRunFreshnessAndSourceBoundary` — a run row states its `as_of`, generation time, row count, the source projection version and high-water, and whether the population was complete. A truncated run says why it stopped rather than showing a count as if it were the whole population.
- `TestReportingPageShowsTheSourceHighWater` — the reader can see which material versions the report was read from, not just a timestamp.
- `TestReportingPageNamesTheReviewerAndAuthorizerSeparately` — the three principals are shown as three roles, because that separation is the control.
- `TestReportingPageDoesNotOfferAFilterTheServerWillReject` — the editor's options come from the server's published vocabulary, so an unknown field is unreachable in the UI.
- `TestFilterEditorRejectsAnEmptyGroup` — a group with no conditions cannot be saved, and says what to add.
- `TestFilterEditorShowsWhenAFieldIsNotIndexed` — a non-indexed filter is slower, and the operator should know before they build a report on it.
- `TestFilterEditorRejectsAFieldFromTheOtherDataset` — a Program field offered on a ROPA report is refused, with a message naming the datasets it belongs to.
- `TestReportingPageShowsTheDecisionHistory` — a definition's history shows who proposed, who reviewed, who authorised, when, and any note.
- `TestEmptyStateNamesThePopulationAndTheNextAction` — per the copy gate, the empty state states what was checked, the result and the valid next action.
- `TestCopyQualityPassesForTheNewWorkspace`

- [ ] **Step 2: Run to verify failure**

Run: `cd web; npm test -- ReportingPage ReportFilterEditor reportingApi`
Expected: FAIL.

- [ ] **Step 3: Write `web/src/reportingTypes.ts` and `web/src/reportingApi.ts`**

Mirror the Go JSON exactly. Note the existing known inconsistency: `ActivityPage` in Go has no JSON tags, so the client handles both `Rows` and `rows`. Do **not** copy that mistake — give every new struct snake_case tags, and fix `ActivityPage` while touching this area.

- [ ] **Step 4: Write `ReportFilterEditor.tsx`**

Build conditions and and/or groups from the server's published vocabulary. Show each field's human label, not the field name. Where the server marks a field as not indexed, say so in the help text, because a non-indexed filter is slower and the operator should know. Enforce the same node and depth limits client-side for immediate feedback, while the server remains authoritative.

- [ ] **Step 5: Write `ReportingPage.tsx`**

Follow the register's structure: a status strip, a definitions table, a selected definition's governance history, a runs table and the download control. Use the existing `StatusBadge`, `Table` and `EmptyState` components rather than new ones. Every visible string must satisfy the copy gate. Disabled controls explain why.

- [ ] **Step 6: Verify**

Run: `cd web; npm run typecheck` and `npm test -- ReportingPage ReportFilterEditor reportingApi copyQuality` and `npm run check:ui-contracts`
Expected: all clean.

- [ ] **Step 7: Commit**

```bash
git add web/src/reportingTypes.ts web/src/reportingApi.ts web/src/components/ReportFilterEditor.tsx web/src/components/ReportingPage.tsx web/src/components/ropa.css web/src/routing.ts web/src/components/RopaRegisterPage.tsx web/src/reportingApi.test.ts web/src/components/ReportFilterEditor.test.tsx web/src/components/ReportingPage.test.tsx
git commit -m "feat(reporting): add the report workspace

The filter editor offers only the fields the server publishes, so an unknown
field is unreachable in the UI. Definitions show who proposed, who checked and
when, and a run states its population and whether it completed."
```

---

## Task 12: Rendered evidence and the review transport

**Files:**
- Create: `web/src/reportingEvidence.ts`
- Test: `web/src/reportingEvidence.test.ts`
- Modify: `web/src/evidenceMain.tsx`
- Modify: `web/scripts/capture-ui-evidence.mjs`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`

- [ ] **Step 1: Write the failing installer test**

Model on `web/src/ropaEvidence.ts`: capture the previous `fetch`, normalise string/URL/`Request` inputs, match method and path, return JSON `Response` objects, record handled URLs on `window.reportingEvidenceReads`, and delegate everything else.

- [ ] **Step 2: Serve the real routes in `web/src/reportingEvidence.ts`**

Handle `GET /api/v1/ropa/reports/filter-fields`, `/definitions`, `/definitions/{id}`, `/definitions/{id}/history`, `/runs`, `/runs/{id}`. The sample content must include one `ACTIVE` definition, one `PENDING_REVIEW` definition, one `REVIEWED` definition, one `READY` run with a complete population, and one `FAILED` run whose `failure_code` is `row_limit_exceeded` and whose `row_count` is `0` — so the bounded-stop state is visible in review evidence rather than only in a test, and so a failed run is visibly not a successful short report.

Give the sample activity the name `Customer account opening` if any capture waits for that text; check `capture-ui-evidence.mjs` for the exact expected string before choosing names.

- [ ] **Step 3: Add three captures and keep the manifest in sync**

- `140-report-definitions-light-1440x900.png` — the definitions list with both governance states.
- `141-report-definitions-dark-mobile-390x844.png` — responsive proof.
- `142-report-run-failed-light-1440x900.png` — the bounded-stop state, which is the one most likely to ship broken.

Add the names and states to `review-ui-flow-manifest.mjs`. Keep the gate exactly as strict as it is now.

- [ ] **Step 4: Run the review and read the renders**

Run: `cd web; npm run review:ui`
Expected: PASS with coverage increased by three records, and no existing record lost.

Then **read each PNG** and confirm: labels are complete and not clipped; the failed run's reason is visible; the disabled run control explains itself; the mobile view replaces the table rather than shrinking it; and the primary action is not blocked by a notice.

Fix the highest-impact failure you find and re-run. Do not proceed on a report that only the test suite says is fine.

- [ ] **Step 5: Commit**

```bash
git add web/src/reportingEvidence.ts web/src/reportingEvidence.test.ts web/src/evidenceMain.tsx web/scripts/capture-ui-evidence.mjs web/scripts/review-ui-flow-manifest.mjs
git commit -m "test(reporting): give the report workspace rendered review evidence

The bounded-stop state is captured deliberately: a report that hit a row limit
is the case most likely to look complete when it is not."
```

---

## Task 13: Full verification

- [ ] **Step 1: Backend**

Run: `go test ./... -count=1`
Expected: PASS.

Run: `go build -tags postgres ./...` and `go build -tags load ./...`
Expected: both succeed.

Run: `go test ./internal/reporting/ -tags postgres -count=1`
Expected: PASS against real PostgreSQL, including the migration round trip and the report-page keyset test.

- [ ] **Step 2: Frontend**

Run: `cd web; npm run typecheck`, `npm test -- copyQuality`, `npm run check:ui-contracts`, `npm run review:ui`
Expected: all clean, review PASS with the new records.

If the full web suite fails at default parallelism but passes at `--maxWorkers=1`, record that as the known pre-existing host contention in this repository rather than claiming a clean run. The pre-existing baseline is unrelated to reporting and its files must not be modified to make it pass.

- [ ] **Step 3: Load**

Run: `go test ./internal/reporting/ -tags load -run TestReportPageBudget -count=1 -v`
Expected: PASS within the 750 ms budget, with the measured numbers pasted into `docs/quality/performance-test-plan.md`.

- [ ] **Step 4: Migrations**

Run the migration up/down/up cycle against real PostgreSQL and confirm the down migration refuses to erase populated report history.

- [ ] **Step 5: Open a PR and report honestly**

The PR body must state what is delivered, what is deliberately not, the measured report budgets, the review coverage numbers, and any check that could not be verified locally. Do not describe #26 as complete if the Archer workbook asks for scheduling or recurring reports — list them as remaining.

---

## Acceptance criteria

The tranche is complete when all of the following are true and each has a named test or measured artefact behind it:

1. A privacy officer can define a report with a filter built only from the server's published vocabulary, and an unlisted field is rejected server-side with a message naming the field.
2. No submitted filter can produce a query that scans an unindexed combination; the field-to-SQL mapping is closed and every value is a bound parameter.
3. A definition cannot become active without a review **and** an activation, performed by three different principals — proposer, reviewer, authorizer — and the decision is bound to the checksum each role saw. There is no path that skips review.
4. Revision content is immutable from insert: an approval cannot cover a definition that was edited afterwards.
5. A report run stops at an explicit row or byte bound, fails with a named code, and never leaves a partial artefact that looks complete.
6. A download re-authorises, verifies the artefact digest, refuses an expired run with an explanation, and records who downloaded it. It requires a capability distinct from the system-activity export.
7. The report's exception dataset selects exactly the activities the register would refuse to close, proven against real PostgreSQL.
8. A restricted Matter the verified principal cannot see never consumes a report page slot, proven at the repository with a same-tenant second legal entity in the fixture.
9. Every run records the source projection version and high-water, not only a timestamp, so a reader knows what was read.
10. A report cannot read another legal entity's rows, proven at the repository with a second entity in the same tenant.
11. 100,000 activities produce a first and cursor page inside 750 ms, measured and recorded.
12. `review:ui` passes with the new records, and the renders were inspected, not just generated.
13. The gap analysis states the delivered and remaining parts of #26 without overstating either.
