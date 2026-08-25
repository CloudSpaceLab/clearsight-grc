# Vendor Management Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one canonical vendor/service-relationship domain and a first-class Vendors sidebar workspace without duplicating Program, Matter, Evidence, Source, Workflow or authority state.

**Architecture:** A focused `internal/thirdparty` service owns vendor organizations and bank-to-vendor service relationships. Commands use verified actor and legal-entity context; PostgreSQL writes the authoritative rows, append-only domain event and outbox event in one transaction. The React Vendors workspace uses bounded list/detail APIs, preserves browser routing, and links later work to canonical Matter/Evidence surfaces.

**Tech Stack:** Go 1.25 standard HTTP, PostgreSQL 18/pgx, existing command guard and route registry, React 19, TypeScript 7, Vite 8, Vitest/Testing Library, existing ClearSight CSS tokens.

**Source contract:** GitHub issue #80 and `docs/implementation-plan.md`.

**Execution status (2026-08-25):** Tasks 1–8 are implemented for the vendor-organization and service-relationship foundation. The normal Go, PostgreSQL-tagged, frontend, copy and rendered UI gates pass. The database-backed integration test exists but is skipped locally when `TEST_DATABASE_URL` is not configured. Issue #80 remains open for contracts, assessments, document validity, governed activation and continuation, reassessment and verified exit.

---

## File map

- Create `internal/thirdparty/model.go`: domain types, statuses, page/cursor and command inputs.
- Create `internal/thirdparty/repository.go`: narrow authoritative repository interface and errors.
- Create `internal/thirdparty/service.go`: validation, actor binding, command orchestration and bounded reads.
- Create `internal/thirdparty/memory.go`: deterministic memory repository for local/demo operation and unit tests.
- Create `internal/thirdparty/service_test.go`: service red/green tests.
- Create `internal/thirdparty/postgres.go`: actor/legal-entity-scoped reads and transactional writes.
- Create `internal/thirdparty/postgres_integration_test.go`: PostgreSQL transaction, isolation and pagination proof.
- Create `migrations/000035_third_party_foundation.up.sql` and `.down.sql`: vendor, relationship and event tables plus indexes and tenant-safe foreign keys.
- Modify `docs/architecture/durable-schema-ownership.md`: classify all new tables.
- Create `internal/httpapi/third_party_handlers.go`: actor-bound list/detail/create/update handlers.
- Create `internal/httpapi/third_party_handlers_test.go`: identity, tenant/entity, error and payload tests.
- Modify `internal/httpapi/server.go`, `route_registry.go`: dependency and route inventory.
- Modify `cmd/api/services.go`, `services_memory.go`, `services_postgres.go`, `main.go`: compose the one service.
- Create `web/src/vendorTypes.ts` and `vendorApi.ts`: typed contracts and HTTP clients.
- Create `web/src/components/VendorsWorkspace.tsx`: list/detail/create/edit orchestration.
- Create `web/src/components/VendorsWorkspace.test.tsx`: primary workflow and state tests.
- Create `web/src/vendors.css`: workspace styling using existing tokens.
- Modify `web/src/appRouting.ts`, `App.tsx`, `AppViews.tsx`, `components/NavigationIcon.tsx`, `main.tsx`: first-class Vendors navigation and routing.
- Modify `web/src/staticDemo.ts` only if the deterministic visual harness requires vendor responses.
- Modify `api/runtime.openapi.json` through the existing generator/check path.
- Modify `README.md`, `docs/implementation-plan.md`, issue #80: synchronize delivered scope.

---

### Task 1: Define the canonical vendor and relationship contract

**Files:**
- Create: `internal/thirdparty/model.go`
- Create: `internal/thirdparty/repository.go`
- Test: `internal/thirdparty/service_test.go`

- [ ] **Step 1: Write failing validation tests**

```go
func TestCreateRelationshipRequiresVendorServiceAndLegalEntity(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.CreateRelationship(context.Background(), Actor{TenantID:"bank", LegalEntityID:"entity", PrincipalID:"owner"}, CreateRelationshipInput{})
	if !errors.Is(err, ErrInvalid) { t.Fatalf("expected invalid input, got %v", err) }
}

func TestCreateRelationshipBindsOwnerToVerifiedActor(t *testing.T) {
	service := NewService(NewMemoryRepository())
	got, err := service.CreateRelationship(context.Background(), Actor{TenantID:"bank", LegalEntityID:"entity", PrincipalID:"owner"}, validCreateInput())
	if err != nil { t.Fatal(err) }
	if got.Relationship.BusinessOwnerPrincipalID != "owner" { t.Fatalf("unexpected owner %q", got.Relationship.BusinessOwnerPrincipalID) }
}
```

- [ ] **Step 2: Run the tests and confirm RED**

Run: `go test ./internal/thirdparty -run 'TestCreateRelationship' -count=1`

Expected: compilation fails because the package/types do not exist.

- [ ] **Step 3: Implement the minimum domain types**

Define:

```go
type Actor struct { TenantID, LegalEntityID, PrincipalID string }
type VendorStatus string
const (VendorActive VendorStatus = "ACTIVE"; VendorInactive VendorStatus = "INACTIVE")
type RelationshipStatus string
const (
	RelationshipProposed RelationshipStatus = "PROPOSED"
	RelationshipUnderReview RelationshipStatus = "UNDER_REVIEW"
	RelationshipActive RelationshipStatus = "ACTIVE"
	RelationshipRestricted RelationshipStatus = "RESTRICTED"
	RelationshipSuspended RelationshipStatus = "SUSPENDED"
	RelationshipExiting RelationshipStatus = "EXITING"
	RelationshipTerminated RelationshipStatus = "TERMINATED"
)
type Criticality string
const (CriticalityStandard Criticality = "STANDARD"; CriticalityImportant Criticality = "IMPORTANT"; CriticalityCritical Criticality = "CRITICAL")
type PrivacyRole string
const (PrivacyNone PrivacyRole = "NONE"; PrivacyProcessor PrivacyRole = "PROCESSOR"; PrivacyJointController PrivacyRole = "JOINT_CONTROLLER")
```

`Vendor` owns identity only. `Relationship` owns bank legal entity, service, accountable owner, criticality, privacy role, dates and lifecycle. `RelationshipAggregate` embeds the vendor and relationship. Command inputs do not accept tenant, legal entity or actor as trusted domain fields; those come from `Actor`.

- [ ] **Step 4: Implement the narrow repository contract**

```go
type Repository interface {
	CreateRelationship(context.Context, CreateRecord) (RelationshipAggregate, error)
	UpdateRelationship(context.Context, UpdateRecord) (RelationshipAggregate, error)
	GetRelationship(context.Context, Scope, string) (RelationshipAggregate, error)
	ListRelationships(context.Context, ListFilter) (RelationshipPage, error)
}
```

Use `ErrNotFound`, `ErrConflict`, `ErrInvalid`. List filters require tenant and legal entity, cap limit at 100 and use a stable `(updated_at,id)` cursor.

- [ ] **Step 5: Run tests and confirm GREEN**

Run: `go test ./internal/thirdparty -count=1`

Expected: PASS.

---

### Task 2: Implement memory commands and bounded reads

**Files:**
- Create: `internal/thirdparty/service.go`
- Create: `internal/thirdparty/memory.go`
- Modify: `internal/thirdparty/service_test.go`

- [ ] **Step 1: Write failing behavior tests**

Cover:

```go
func TestListRelationshipsNeverCrossesLegalEntity(t *testing.T) { /* create entity A and B; list A returns only A */ }
func TestCreateRelationshipReusesExactExternalVendorIdentity(t *testing.T) { /* same source/external ref, two services -> one vendor */ }
func TestUpdateRelationshipRequiresCurrentVersion(t *testing.T) { /* stale expected version -> ErrConflict */ }
func TestListRelationshipsSearchesVendorAndService(t *testing.T) { /* bounded case-insensitive search */ }
```

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/thirdparty -run 'Test(List|CreateRelationshipReuses|Update)' -count=1`

Expected: failures for missing repository/service behavior.

- [ ] **Step 3: Implement minimal memory behavior**

Create and update under one mutex. Normalize display fields only for matching; preserve entered working language. Reuse a vendor only on exact tenant + source ID + external reference. Never fuzzy-merge. Sort by `updated_at DESC,id DESC`, apply legal entity and search before limit, and emit a cursor only when another page exists.

- [ ] **Step 4: Run all third-party tests**

Run: `go test ./internal/thirdparty -count=1`

Expected: PASS.

---

### Task 3: Add durable schema and transactional PostgreSQL persistence

**Files:**
- Create: `migrations/000035_third_party_foundation.up.sql`
- Create: `migrations/000035_third_party_foundation.down.sql`
- Create: `internal/thirdparty/postgres.go`
- Create: `internal/thirdparty/postgres_integration_test.go`
- Modify: `docs/architecture/durable-schema-ownership.md`

- [ ] **Step 1: Write failing PostgreSQL integration tests**

Tests must prove:

1. create writes one vendor, one relationship, one `third_party_events` row and one `outbox_events` row;
2. a repeated exact source identity creates a new relationship against the same vendor;
3. duplicate source identity cannot race into duplicate vendors;
4. actor legal entity filtering happens before limit/cursor;
5. cross-tenant IDs return `ErrNotFound`;
6. stale update leaves row/event/outbox counts unchanged;
7. migration rollback/reapply succeeds.

- [ ] **Step 2: Run and confirm RED**

Run: `go test -tags "postgres postgresintegration" ./internal/thirdparty -count=1`

Expected with `TEST_DATABASE_URL`: FAIL because migration/repository are absent. Without it: tests report SKIP and must be run in CI before completion.

- [ ] **Step 3: Add the migration**

Create:

- `third_parties`: tenant identity, exact source/external reference, legal/trading name, registration/jurisdiction, status, timestamps, version; unique exact source identity when present;
- `third_party_relationships`: tenant, vendor, bank legal entity, service, owner, criticality, privacy role, source reference, effective/renewal dates, status, timestamps, version;
- `third_party_events`: append-only tenant, aggregate type/id/version, actor, event type, safe payload, occurrence.

Use tenant-safe composite foreign keys, checks for enumerations, partial indexes for active lists, and `(tenant_id,legal_entity_id,updated_at DESC,id DESC)` for keyset reads. The down migration drops only these three tables in dependency order.

- [ ] **Step 4: Implement transactional repository methods**

Use `pgx.BeginTx`. Lock/update by tenant + relationship + expected version. Insert the authoritative row change, domain event and safe outbox event before commit. Do not return a command failure after commit because response hydration failed; return the committed aggregate or a committed receipt through the existing command-outcome boundary.

- [ ] **Step 5: Update schema ownership and run checks**

Classify `third_parties`, `third_party_relationships`, and `third_party_events` as authoritative domain tables owned by `internal/thirdparty`, with temporal reconstruction and retention notes.

Run:

```powershell
go test -tags postgres ./internal/thirdparty ./internal/platform/database -count=1
go test ./internal/thirdparty -count=1
```

Expected: PASS; integration tests require the configured database.

---

### Task 4: Expose verified, actor-scoped APIs

**Files:**
- Create: `internal/httpapi/third_party_handlers.go`
- Create: `internal/httpapi/third_party_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/route_registry.go`

- [ ] **Step 1: Write failing HTTP tests**

Cover:

```go
func TestListVendorsUsesVerifiedTenantAndLegalEntity(t *testing.T) { /* conflicting query rejected */ }
func TestCreateVendorRelationshipOverwritesActorAndScope(t *testing.T) { /* forged tenant/entity/actor cannot reach service */ }
func TestGetVendorRelationshipReturnsNotFoundOutsideEntity(t *testing.T) { /* no existence leak */ }
func TestUpdateVendorRelationshipUsesRouteIDAndExpectedVersion(t *testing.T) { /* body cannot redirect ID */ }
```

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./internal/httpapi -run 'Test(List|Create|Get|Update)Vendor' -count=1`

Expected: FAIL because routes/handlers/dependency are absent.

- [ ] **Step 3: Implement handlers and routes**

Routes:

```go
read("/api/v1/vendors", a.listVendorRelationships)
read("/api/v1/vendors/{id}", a.getVendorRelationship)
material("/api/v1/vendors", "thirdparty.relationship.create", a.createVendorRelationship,
  commandPolicy{ObjectType:"VENDOR_RELATIONSHIP", Responsibility:authority.ResponsibilityOwner, Materiality:3, BindLegalEntity:true})
material("/api/v1/vendors/{id}", "thirdparty.relationship.update", a.updateVendorRelationship,
  commandPolicy{ObjectType:"VENDOR_RELATIONSHIP", Responsibility:authority.ResponsibilityOwner, Materiality:3})
```

Handlers derive `thirdparty.Actor` only from `identity.Require`. Creation overwrites business owner with the verified actor for the initial tranche; reassignment remains governed follow-up work. Return `{items,next_cursor}` for lists and normal domain errors as 404/409/422/503 with bank-working-language recovery text.

- [ ] **Step 4: Run route/API tests**

Run: `go test ./internal/httpapi ./internal/thirdparty -count=1`

Expected: PASS.

---

### Task 5: Compose the service in memory and PostgreSQL

**Files:**
- Modify: `cmd/api/services.go`
- Modify: `cmd/api/services_memory.go`
- Modify: `cmd/api/services_postgres.go`
- Modify: `cmd/api/main.go`
- Test: `cmd/api/services_composition_test.go` or existing composition tests

- [ ] **Step 1: Write failing composition assertions**

Assert that both memory and PostgreSQL `serviceSet` provide one `ThirdParty` service and that `httpapi.Dependencies.ThirdParty` receives it.

- [ ] **Step 2: Run and confirm RED**

Run: `go test ./cmd/api -count=1`

Expected: FAIL because `ThirdParty` is not composed.

- [ ] **Step 3: Add composition**

Memory uses `thirdparty.NewMemoryRepository()`. PostgreSQL uses `thirdparty.NewPostgresRepository(pool)`. Do not create a second pool, worker or background service.

- [ ] **Step 4: Run backend tests**

Run: `go test ./... -count=1`

Expected: PASS.

---

### Task 6: Add Vendors routing and first-class sidebar navigation

**Files:**
- Modify: `web/src/appRouting.ts`
- Modify: `web/src/components/NavigationIcon.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/AppViews.tsx`
- Test: `web/src/appRouting.test.ts`
- Test: existing App/navigation tests

- [ ] **Step 1: Write failing routing/navigation tests**

```ts
expect(parseRoute("#vendors/rel-1")).toEqual({view:"vendors", target:{vendorRelationshipID:"rel-1"}});
expect(routeHash("vendors", {vendorRelationshipID:"rel-1"}, "matters")).toBe("#vendors/rel-1");
expect(screen.getByRole("button", {name:"Vendors"})).toHaveAttribute("aria-current", "page");
```

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- appRouting.test.ts App.test.tsx`

Expected: FAIL because `vendors` is not a supported view.

- [ ] **Step 3: Add the route and navigation item**

Add `vendors` to `View`, `vendorRelationshipID` to `WorkspaceTarget`, route parsing/hash behavior, and navigation between Programs and Work. Add one consistent outline SVG icon showing a building/service relationship; no emoji or vendor logo. Desktop sidebar and mobile navigation both derive from the same navigation array.

- [ ] **Step 4: Run routing/navigation tests**

Run: `npm test -- appRouting.test.ts App.test.tsx`

Expected: PASS.

---

### Task 7: Build typed vendor API clients

**Files:**
- Create: `web/src/vendorTypes.ts`
- Create: `web/src/vendorApi.ts`
- Create: `web/src/vendorApi.test.ts`

- [ ] **Step 1: Write failing client tests**

Assert exact GET/POST paths, expected-version payload, and no client-supplied tenant/legal-entity/actor fields.

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- vendorApi.test.ts`

Expected: FAIL because the module is absent.

- [ ] **Step 3: Implement types and clients**

Expose `loadVendorRelationships`, `loadVendorRelationship`, `createVendorRelationship`, and `updateVendorRelationship`. Use the existing `requestJSON`/HTTP error model and stable response types.

- [ ] **Step 4: Run client tests**

Run: `npm test -- vendorApi.test.ts`

Expected: PASS.

---

### Task 8: Build the Vendors management workspace

**Files:**
- Create: `web/src/components/VendorsWorkspace.tsx`
- Create: `web/src/components/VendorsWorkspace.test.tsx`
- Create: `web/src/vendors.css`
- Modify: `web/src/AppViews.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: Write failing component tests**

Prove:

1. loading, unavailable, empty and populated states identify the exact population checked;
2. search filters vendor and service working language;
3. `Add vendor` creates vendor + relationship and opens the authoritative returned record;
4. field errors appear beside the field and API conflicts preserve entered values;
5. detail shows vendor, service, owner, criticality, privacy role, source and freshness/version;
6. edit submits current expected version and refreshes from server response;
7. the dominant action is singular for the current state;
8. browser back returns to the prior list/search state;
9. no enabled control is a placeholder.

- [ ] **Step 2: Run and confirm RED**

Run: `npm test -- VendorsWorkspace.test.tsx`

Expected: FAIL because the workspace is absent.

- [ ] **Step 3: Implement the workspace**

Use the existing topbar, panels, status marks, buttons, semantic tokens and error components. Desktop uses a list/detail composition; under the existing mobile breakpoint it becomes a stacked card list and focused record page, not a squeezed table. The sidebar label is `Vendors`; the page heading is `Vendors`; supporting text says which legal-entity relationships are shown and what the user can do next.

Create fields:

- legal name;
- trading name (optional);
- registration/reference (optional);
- jurisdiction;
- service supplied;
- criticality;
- privacy role;
- source system/reference (optional);
- effective/renewal dates (optional).

The verified actor becomes the initial accountable owner. Copy must not claim assessment, approval, activation or compliance until those records exist.

- [ ] **Step 4: Run component and accessibility tests**

Run:

```powershell
npm test -- VendorsWorkspace.test.tsx
npm run typecheck
```

Expected: PASS.

---

### Task 9: Synchronize executable API/docs and verify the slice

**Files:**
- Modify: `api/runtime.openapi.json`
- Modify: `README.md`
- Modify: `docs/implementation-plan.md`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Modify: issue #80

- [ ] **Step 1: Generate/check the runtime route inventory**

Use the repository's existing runtime OpenAPI generator/check command discovered in CI. Do not hand-authorize routes in a descriptive schema.

- [ ] **Step 2: Update truthful capability documentation**

State that the foundation supports vendor/service relationship creation, scoped list/detail and maintenance. Keep assessment, documents, approval, activation, reassessment and exit explicitly incomplete until later plans pass.

- [ ] **Step 3: Run full verification**

```powershell
gofmt -w internal/thirdparty internal/httpapi/third_party_handlers.go internal/httpapi/third_party_handlers_test.go
go test ./... -count=1
go test -tags postgres ./... -count=1
go vet ./...
Set-Location web
npm test -- --run
npm run typecheck
npm run build
```

Run PostgreSQL integration with `TEST_DATABASE_URL` in the configured CI/test environment. Render Vendors at 1440×900, 768×1024, 390×844 and 200% zoom/reflow; inspect loading, empty, create, populated, unavailable and conflict states in both themes.

- [ ] **Step 4: Commit the dependency-complete slice**

Commit only intended source, tests, migrations and documentation. Do not include `.codex-tmp/`.
