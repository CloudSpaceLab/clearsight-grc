# ROPA register, dashboard and reporting — design

Date: 2026-09-23
Status: proposed
Requirements: Fidelity Bank Archer workbook #26, #28, #29 (Data Privacy)
Related: `docs/product/archer-requirements-gap.md`, `docs/engineering/ui-use-case-acceptance-matrix.md`

## What we are building

A central register of processing activities for the bank's personal data, with a
dashboard that shows what is complete, what is overdue for review, and how fresh the
numbers are.

Two tranches:

| | Tranche 1 | Tranche 2 |
|---|---|---|
| Delivers | Processing activity register + dashboard (#29, base of #28) | Configurable exception/compliance reports (#26) |
| New domain | Yes — `internal/ropa` | No — reads Program, Matter and ROPA |
| Why split | A new aggregate has to land whole; #26 is read-scoped and can reuse Tranche 1's filter machinery | Keeps the risky part isolated |

## Not in scope

- Consent management (#30), breach register (#24), DPIA register (#22), PIA screening (#21) — separate aggregates, separate tranches.
- Row-level export of restricted activities to a shared surface. Exports are purpose-bound artefacts, not ad-hoc downloads.
- Tenant-wide cross-legal-entity views. A processing activity belongs to exactly one legal entity.

## Correction to the current gap analysis

`docs/product/archer-requirements-gap.md` records #28 and #29 as *Partial*, with the
covered part described as "a central ROPA population with audit and role-scoped access".
That overstates what exists. The only ROPA artefacts in the repository are:

- `internal/continuity/demo.go:18` — `ROPA-CURRENT`, a Program **requirement** whose text
  is "The bank must keep an up-to-date record of personal-data processing activities".
  A statement of obligation, not a register.
- `internal/continuity/demo.go:43` — `ROPA-COVERAGE`, an **evidence contract** claiming
  ≥95% coverage of `{"population":"active_processing_activities"}`. That population is a
  JSON string label, not a backed set of records.

There is no `internal/ropa` package, no ROPA migration, no ROPA route and no
processing-activity records. This tranche updates the gap document to say so.

## Data model

New aggregate `ProcessingActivity`, in `internal/ropa`, following the Program/Matter
patterns in `internal/continuity`.

**Current row** — `ropa_processing_activities`:

- `id` uuidv7 primary key, `tenant_id` → `tenants(id)`
- `legal_entity_id` **not null and immutable after insert** (as Programs and Matters, `migrations/000035_matter_legal_entity_scope.up.sql`)
- `code` — stable within legal entity
- `name`, `description`
- `status` — `NEW` | `OPEN` | `CLOSED`
- `purpose`, `lawful_basis`
- `controller`, `processor`
- `automated_decision_making` boolean
- `data_subject_categories`, `personal_data_categories`, `security_measures`, `retention_period`
- `start_date`, `end_date`, `next_review_date`
- `owner_principal_id`, `required_authority_principal_id` — distinct, never collapsed
- `program_id` nullable — a link for continuity, not containment
- `version`, `created_at`, `updated_at`

Frequently filtered values stay typed columns. JSONB is used only for bounded,
versioned attributes.

**Normalised children** — independently governed, so not one unbounded blob:

- `ropa_processing_activity_data_categories`
- `ropa_processing_activity_recipients`
- `ropa_processing_activity_systems`
- `ropa_processing_activity_reviews`

**Reconstruction** — the rule that material records are versioned and rebuildable:

- `ropa_processing_activity_revisions` — immutable, protected by trigger, as the risk-register
  pattern in `migrations/000088_risk_register_migrations.up.sql`
- `ropa_events` — append-only, unique per `(tenant, aggregate_type, aggregate_id, aggregate_version)`,
  mirroring `continuity_events` (`migrations/000008_programs_matters.up.sql:404-420`)
- transactional outbox insert in the same transaction

No new generic audit table. Migration `000019_schema_ownership_cleanup.up.sql` removed
`audit_events` deliberately; we use domain events, revisions, outbox and report receipts.

**`end_date` means processing ceased.** A retired activity drops off the live register but
stays fully reconstructable in history and in period reports.

## Validation rules

An activity cannot reach `CLOSED` without: a lawful basis, a named owner, at least one
data-subject category, and a completed review. Enforced in the domain service
(`service.go` pattern in `internal/continuity/service.go:348-397`). The UI disables the
action and states which rule is unmet — it does not offer a control that then fails.

## Reads and scale

Target customer is a very large bank, so the dashboard is designed for 10k–100k
processing activities and 1M+ child rows.

**The dashboard reads a projection, never live aggregates.** A `ropa_register_summary`
projection is maintained per exact `(tenant, legal_entity)` in bounded leased batches and
serves every tile. This is the oversight pattern that already works
(`internal/oversight/postgres.go:69-127`).

**Honest denominators.** Every tile reports `included` / `excluded` / `unknown`
separately, and shows `generated_at`, `source_high_water` and `projection_version`. A
stale projection is labelled stale, never presented as current. Copied from
`internal/oversight/model.go:99-116`.

**Keyset pagination, no offsets.** Stable tuple `(status_rank, next_review_date, id)`,
opaque base64 cursor, `limit+1` page detection. Visibility is applied **in SQL before
`LIMIT`**, so a restricted activity never consumes a page slot — the same rule as
`internal/continuity/summaries_postgres.go:259-288` and its regression test
`summary_visibility_test.go:12-31`.

**Partitioning — deliberately not on the current table.** Current rows are mutable, so
range-partitioning buys little and costs operational pain. Thresholds recorded instead of
pre-built infrastructure: partition `ropa_events` and revisions by month once either
exceeds 50M rows or 12 months of retention.

**Indexes from actual query shapes:**

- `(tenant_id, legal_entity_id, status, next_review_date, id)` — register list and keyset
- `(tenant_id, legal_entity_id, lawful_basis)` — dashboard filter dimension
- `(tenant_id, legal_entity_id, owner_principal_id)` — "my activities"
- `(activity_id, …)` on each child table

**Report generation is asynchronous**, reusing the bounded envelope proven in
`internal/activity/export.go:22-32`: 100-row pages, 10,000-row ceiling, 32 MiB, 7-day
artefact retention. Artefact plus manifest go to versioned object storage with a SHA-256
and recorded `as_of` / source high-water. A run that cannot finish reports failure — it
never returns a partial file that looks whole.

**Performance targets, load-tested before merge:** dashboard p95 < 500 ms and register
list p95 < 750 ms at 100k activities.

## Governance and authority

Report definitions are configuration, not ad-hoc filters. They follow the maker-checker
model in `internal/governance/model.go:48-85`: current row, immutable revision, checksum,
maker and checker, effective dating, rollback, audit.

Responsibilities stay distinct (`internal/authority/model.go:9-22`):

| Operation | Responsibility |
|---|---|
| Create or edit an activity | `ACCOUNTABLE_OWNER` |
| Submit a report definition | `PROPOSER` |
| Review a report definition | `REVIEWER` |
| Activate a definition | `AUTHORIZER` |
| Run a report | `PERFORMER` |
| Download a protected artefact | separate export authority |

Material commands re-resolve the current authority route at execution and fail closed on
missing, ambiguous or unavailable authority (`internal/commandauth/guard.go:72-121`).
Client-supplied actor, tenant and legal-entity fields are overwritten from verified
identity. No hard-coded approvers.

**Review cycles (#28).** `next_review_date` is stored in Tranche 1. Reminders and
escalation reuse the existing `Workflow Task` timer and worker class if they can carry a
ROPA review without a new worker class; otherwise reminders land in Tranche 1.5 with the
scope/freshness machinery already proven. We decide this at implementation time by
reading the timer class, not by guessing now.

## Reports (#26, Tranche 2)

Configurable reports over exceptions and compliance, scoped to a Program, a Matter or the
whole legal entity.

**Filter vocabulary is an allow-list** of indexed dimensions, enforced server-side. A user
cannot construct a combination that forces a full-table scan. This is the same rule the
existing summary queries already follow.

Output is a versioned object-storage artefact with a manifest, downloadable through the
protected path used by audit exports (`internal/httpapi/audit_export_handlers.go:107-146`).

## User interface

Per `DESIGN.md:124` — "Do not default every concept to a dashboard card." So the ROPA home
is a **status and coverage strip over a searchable register**, not a KPI wall. This is also
what both reference products do: OvalEdge shows a colour-coded status bar above the
processing-activity list; Legit.eu leads with a real-time inventory view.

Routes, following the existing pattern in `web/src/appRouting.ts`:

- `#ropa` — register, with status/coverage strip and filters
- `#ropa/activity/{id}` — one activity
- `#ropa/reports` — report definitions and runs

`View` gains `"ropa"`; a `RopaPage` sub-page union follows `VendorPage`
(`"overview" | "register"`). Sidebar entry added at `web/src/App.tsx:393`.

Reference UI patterns worth borrowing from OvalEdge:

- colour-coded status bar summarising all activities, with hover summary
- activity summary page combining description, data terms, engagement and dates
- history of changes, sortable oldest/newest
- owner / steward / custodian governance roles
- a dated report that pulls active activities and flags any falling outside its window

Copy follows `AGENTS.md`: business language, no internal codes in visible text, every
count backed by stored data, unknown denominators shown as unknown.

## Testing

Layers, matching `internal/continuity`:

- **Migration** — filename/checksum conventions, up/down transaction structure, ownership
  register exact match, cross-entity foreign-key rejection, revision immutability, index shape
- **Domain** — lifecycle transitions, validation rules, owner/authority distinctness,
  effective dating, replay, command version separate from projection version
- **Command** — tenant and legal-entity mismatch, body actor overwrite, route ID wins,
  current authority required, delegation only for the correct stored authority,
  maker-checker separation, stale version conflict, rollback if event/outbox insert fails
- **Read** — exact entity scope, restricted rows excluded before limit, keyset stability,
  invalid cursor rejection, freshness exposed, stale never labelled current
- **Route** — ROPA routes present in `productionRoutes()`, correct class and permission,
  OpenAPI parity in `api/runtime.openapi.json`
- **Performance** — the 100k-activity load test above

`docs/architecture/durable-schema-ownership.md` is executable: every new table needs its
ownership row in the same change or CI fails.

## Delivery sequence

1. Migration + ownership register rows
2. `internal/ropa` model, service, memory repository
3. PostgreSQL command repository, events, outbox, revisions
4. Summary projection + dashboard read
5. HTTP routes, handlers, OpenAPI
6. Register list, activity record, review surfaces
7. Web: routes, nav, register, status strip
8. Composition in API memory/Postgres and worker
9. Tests, load test, rendered evidence
10. Correct `archer-requirements-gap.md`

## Decisions taken

- New `internal/ropa` aggregate, not a Program child
- `end_date` = processing ceased; retired leaves the live register, stays in history
- `next_review_date` stored in Tranche 1; reminders reuse existing timers or defer to 1.5
- No partitioning on mutable current rows; thresholds recorded for append-only tables
- Report filter vocabulary allow-listed server-side
- Dashboard is a register with a coverage strip, not a KPI wall
