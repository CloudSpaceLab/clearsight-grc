# Seeded Runtime Truth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every normal and demonstration API/web read derive from verified identity plus stored repositories or projections, while retaining deterministic UI evidence through a separate non-shipping entry.

**Architecture:** Add a tenant-scoped runtime-context resolver backed by the existing `tenants`, `legal_entities`, and `principals` tables. Inject it into the actor context handler and fail explicitly when the verified scope cannot be resolved. Remove code-bound Today and actor-name fixtures. Move the static fetch interceptor and evidence-only pages behind a separate Vite entry/config whose import graph is unreachable from `web/src/main.tsx`; enforce both backend and frontend boundaries in CI.

**Tech Stack:** Go 1.26, pgx/PostgreSQL 18, React 19, TypeScript 7, Vite 8, Vitest 4, Node 24.

---

## File map

- Create `internal/runtimecontext/model.go` for the resolved display context and resolver contract.
- Create `internal/runtimecontext/postgres.go` and `internal/runtimecontext/postgres_integration_test.go` for exact actor-scope lookup.
- Create `internal/runtimecontext/architecture_test.go` for backend fixture-boundary enforcement.
- Modify `internal/httpapi/server.go`, `internal/httpapi/actor_read_handlers.go`, and their focused tests to consume the resolver.
- Modify `cmd/api/services.go`, `cmd/api/services_postgres.go`, `cmd/api/services_memory.go`, and `cmd/api/main.go` to compose the resolver without demo label maps.
- Delete `internal/today.DemoItems` and its fixture-specific test; update affected HTTP/service tests with explicit local test data.
- Delete the unused hardcoded `context` and `today` handlers from `internal/httpapi/handlers.go`.
- Create `web/src/evidenceMain.tsx`, `web/evidence.html`, and `web/vite.evidence.config.ts` for deterministic visual evidence.
- Modify `web/src/main.tsx`, `web/vite.config.ts`, `web/package.json`, `web/scripts/run-ui-ux-review.mjs`, and `.github/workflows/ui-evidence.yml`.
- Create `web/scripts/runtime-fixture-boundary.nodecheck.mjs` and add it to CI.

### Task 1: Resolve display context from the verified PostgreSQL scope

- [ ] **Step 1: Write the failing resolver integration tests**

Create `internal/runtimecontext/postgres_integration_test.go` with fixtures for two tenants, two legal entities, and two principals. Assert that the exact verified triple resolves and every cross-tenant/cross-entity combination returns `ErrNotFound`.

```go
func TestPostgresResolverUsesExactVerifiedScope(t *testing.T) {
	ctx, pool := runtimeContextDatabase(t)
	seedRuntimeContext(t, ctx, pool)
	resolver := NewPostgresResolver(pool)

	value, err := resolver.Resolve(ctx, Scope{
		TenantID: "8f100000-0000-4000-8000-000000000001",
		LegalEntityID: "8f100000-0000-4000-8000-000000000002",
		PrincipalID: "8f100000-0000-4000-8000-000000000003",
	})
	if err != nil { t.Fatal(err) }
	if value.TenantName != "Reference Bank" || value.LegalEntityName != "Reference Bank Nigeria" || value.PrincipalName != "Compliance Officer" {
		t.Fatalf("resolved context = %#v", value)
	}

	_, err = resolver.Resolve(ctx, Scope{
		TenantID: "8f200000-0000-4000-8000-000000000001",
		LegalEntityID: "8f100000-0000-4000-8000-000000000002",
		PrincipalID: "8f100000-0000-4000-8000-000000000003",
	})
	if !errors.Is(err, ErrNotFound) { t.Fatalf("cross-tenant resolve error = %v", err) }
}
```

- [ ] **Step 2: Run the focused test and confirm it fails**

Run: `go test -count=1 -tags "postgres postgresintegration" ./internal/runtimecontext`

Expected: compilation fails because the package and resolver do not exist.

- [ ] **Step 3: Implement the resolver contract and exact indexed query**

Create `internal/runtimecontext/model.go`:

```go
package runtimecontext

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("runtime context is unavailable")

type Scope struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type DisplayContext struct {
	TenantName      string
	LegalEntityName string
	PrincipalName   string
}

type Resolver interface {
	Resolve(context.Context, Scope) (DisplayContext, error)
}
```

Create `internal/runtimecontext/postgres.go` with one bounded lookup:

```go
err := r.pool.QueryRow(ctx, `
	SELECT t.name,le.name,p.display_name
	FROM tenants t
	JOIN legal_entities le ON le.tenant_id=t.id
	JOIN principals p ON p.tenant_id=t.id
	WHERE (t.id::text=$1 OR t.slug=$1)
	  AND (le.id::text=$2 OR le.code=$2)
	  AND p.id::text=$3
	  AND p.status='ACTIVE'
	  AND le.valid_from<=clock_timestamp()
	  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
	  AND p.valid_from<=clock_timestamp()
	  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
	LIMIT 1`, scope.TenantID, scope.LegalEntityID, scope.PrincipalID).
	Scan(&value.TenantName, &value.LegalEntityName, &value.PrincipalName)
```

Normalize `pgx.ErrNoRows` to `ErrNotFound`; reject blank scope before querying.

- [ ] **Step 4: Run the resolver tests**

Run: `go test -count=1 -tags "postgres postgresintegration" ./internal/runtimecontext`

Expected: PASS.

- [ ] **Step 5: Commit the resolver slice**

```bash
git add internal/runtimecontext
git commit -m "feat: resolve runtime labels from verified scope"
```

### Task 2: Make `/api/v1/context` use stored labels with explicit failure

- [ ] **Step 1: Write failing handler tests**

Extend `internal/httpapi/demo_routes_test.go` with a recording resolver and these assertions:

```go
func TestActorContextUsesResolvedStoredLabels(t *testing.T) {
	resolver := &runtimeContextStub{value: runtimecontext.DisplayContext{
		TenantName: "Stored Bank", LegalEntityName: "Stored Bank Nigeria", PrincipalName: "Stored Compliance Officer",
	}}
	handler := actorContextHandler(t, resolver)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, verifiedContextRequest())
	if response.Code != http.StatusOK { t.Fatalf("status = %d body=%s", response.Code, response.Body.String()) }
	if resolver.scope.PrincipalID != identity.DurableDemoPrincipalCCO { t.Fatalf("scope = %#v", resolver.scope) }
	for _, value := range []string{"Stored Bank", "Stored Bank Nigeria", "Stored Compliance Officer"} {
		if !strings.Contains(response.Body.String(), value) { t.Fatalf("missing %q", value) }
	}
}

func TestActorContextDoesNotInventLabelsWhenDirectoryScopeIsMissing(t *testing.T) {
	handler := actorContextHandler(t, &runtimeContextStub{err: runtimecontext.ErrNotFound})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, verifiedContextRequest())
	if response.Code != http.StatusServiceUnavailable { t.Fatalf("status = %d", response.Code) }
	if strings.Contains(response.Body.String(), "Clear Bank") || strings.Contains(response.Body.String(), "Chief Compliance Officer") {
		t.Fatalf("response invented display labels: %s", response.Body.String())
	}
}
```

- [ ] **Step 2: Run the tests and confirm failure**

Run: `go test ./internal/httpapi -run 'TestActorContext' -count=1`

Expected: FAIL because `Dependencies` has no runtime-context resolver and the handler still uses `demoActorName`.

- [ ] **Step 3: Inject the resolver and remove demo fallback logic**

Add `RuntimeContext runtimecontext.Resolver` to `internal/httpapi.Dependencies`. In `actorContext`, call it using only values from `identity.Require(r.Context())`. Return:

```go
httpx.WriteError(w, http.StatusServiceUnavailable, "directory_context_unavailable", "Your organization and role details could not be loaded. Refresh the workspace; no task data was changed.")
```

when the resolver is missing or returns `runtimecontext.ErrNotFound`. Delete `demoActorName` completely. Keep roles, permissions, grants, assurance level, session ID, and capabilities derived from the verified actor.

- [ ] **Step 4: Wire PostgreSQL and memory composition**

Add `RuntimeContext runtimecontext.Resolver` to `cmd/api/serviceSet`. Set it to `runtimecontext.NewPostgresResolver(pool)` in `cmd/api/services_postgres.go`. In memory composition use a resolver that returns the verified identifiers as identifiers and marks no friendly display data; do not install bank, role, person, task, count, vendor, Matter, or form constants.

Pass `services.RuntimeContext` from `cmd/api/main.go` into `httpapi.Dependencies`.

- [ ] **Step 5: Run handler and composition tests**

Run:

```bash
go test ./internal/httpapi ./cmd/api -run 'Context|Services' -count=1
go test -tags postgres ./cmd/api ./internal/httpapi -run 'Context|Services' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the API slice**

```bash
git add internal/httpapi cmd/api
git commit -m "fix: serve actor context from stored identity"
```

### Task 3: Remove code-bound Today and dead handler fixtures

- [ ] **Step 1: Replace fixture-dependent tests with explicit local inputs**

In `internal/today/service_test.go`, replace `TestDemoItemsExposeStructuredInterventions` with tests that construct `[]AttentionItem` inside the test and verify sorting/dynamic loading. In `internal/httpapi/server_test.go`, replace:

```go
today.NewService(today.DemoItems())
```

with:

```go
today.NewService([]today.AttentionItem{{
	ID: "matter-test", Type: "MATTER", Title: "Review the test issue", DueAt: time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC),
}})
```

- [ ] **Step 2: Delete production fixture functions**

Delete `DemoItems` from `internal/today/service.go`. Delete the unused `context` and `today` methods from `internal/httpapi/handlers.go`; registered production routes already use `actorContext` and `actorToday`.

- [ ] **Step 3: Prove no normal backend package can add the fixtures back**

Create `internal/runtimecontext/architecture_test.go` that walks non-test `.go` files under `cmd/api`, `internal/httpapi`, and `internal/today` and fails on these structural markers:

```go
var forbiddenRuntimeTruth = []string{
	"today.DemoItems(",
	"func DemoItems(",
	"func demoActorName(",
	"DurableDemoPrincipalCRO:",
	"DurableDemoPrincipalCCO:",
	`"tenant":       map[string]string{"id": "bank-demo"`,
}
```

The test must ignore `_test.go` files and report the exact offending path and marker.

- [ ] **Step 4: Run the boundary and current Today tests**

Run: `go test ./internal/runtimecontext ./internal/today ./internal/httpapi ./cmd/api -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the removal**

```bash
git add internal/today internal/httpapi internal/runtimecontext cmd/api
git commit -m "refactor: remove code-bound runtime fixtures"
```

### Task 4: Isolate deterministic UI evidence from the shipping entry

- [ ] **Step 1: Add a failing import-graph boundary check**

Create `web/scripts/runtime-fixture-boundary.nodecheck.mjs`. Starting from `src/main.tsx`, recursively resolve relative static and dynamic imports and fail if the graph reaches any of:

```js
const evidenceOnly = new Set([
  "src/staticDemo.ts",
  "src/staticDemoBootstrap.ts",
  "src/staticExternalCapture.ts",
  "src/components/LifecycleTodayEvidencePage.tsx",
  "src/components/OperatingMutationsEvidencePage.tsx",
  "src/components/oversight/OversightEvidencePage.tsx",
]);
```

Also assert `src/main.tsx` contains neither `VITE_STATIC_DEMO` nor `VITE_UI_EVIDENCE`.

- [ ] **Step 2: Run the boundary check and confirm failure**

Run: `cd web && node --test scripts/runtime-fixture-boundary.nodecheck.mjs`

Expected: FAIL because `main.tsx` dynamically imports `staticDemoBootstrap` and imports evidence pages.

- [ ] **Step 3: Create the dedicated evidence entry**

Create `web/evidence.html` with `/src/evidenceMain.tsx`. Move fixture bootstrap, evidence-page selection, demo presentation, and gallery routing from `main.tsx` into `evidenceMain.tsx`. The evidence entry must set up the interceptor unconditionally and must render a visible `Synthetic UI evidence` label in its shell.

Reduce `web/src/main.tsx` to the product routes:

```tsx
const presentation = runtimePresentation(window.location.search);
const application = invitationToken !== null
  ? <ExternalCaptureApp invitationToken={invitationToken}/>
  : <SessionGate presentation={presentation}>
      <Suspense fallback={<p role="status">Loading the ClearSight workspace…</p>}>
        <App presentation={presentation}/>
      </Suspense>
    </SessionGate>;
```

Create `web/vite.evidence.config.ts` with `root: "."`, `build.outDir: "dist-evidence"`, and `rollupOptions.input: "evidence.html"`. Do not add `evidence.html` to normal `vite.config.ts` inputs.

- [ ] **Step 4: Add explicit build scripts and update review runners**

Add to `web/package.json`:

```json
"build:evidence": "tsc -b && vite build --config vite.evidence.config.ts",
"preview:evidence": "vite preview --config vite.evidence.config.ts",
"check:runtime-truth": "node --test scripts/runtime-fixture-boundary.nodecheck.mjs"
```

Update `web/scripts/run-ui-ux-review.mjs` to call `npm run build:evidence` and start the preview with `vite.evidence.config.ts`. Remove both `VITE_STATIC_DEMO` and `VITE_UI_EVIDENCE` from all build environments. Update `.github/workflows/ui-evidence.yml` identically.

- [ ] **Step 5: Run normal and evidence builds separately**

Run:

```bash
cd web
npm run check:runtime-truth
npm run typecheck
npm test
npm run build
npm run build:evidence
```

Expected: both builds pass; `dist` contains no static interceptor chunk or unique `StaticDemoHTTPError` string, while `dist-evidence` does.

- [ ] **Step 6: Commit the entry-point isolation**

```bash
git add web .github/workflows/ui-evidence.yml
git commit -m "test: isolate synthetic UI evidence runtime"
```

### Task 5: Put architectural truth gates in required CI

- [ ] **Step 1: Add explicit CI steps**

In `.github/workflows/ci.yml`, after Go formatting add:

```yaml
- name: Verify stored runtime truth boundary
  run: go test ./internal/runtimecontext -run 'Architecture' -count=1
```

In the web job, after `npm ci`, add:

```yaml
- name: Verify shipping entry excludes fixture transport
  run: npm run check:runtime-truth
```

- [ ] **Step 2: Add a normal-bundle assertion to the Node check**

When `dist` exists, recursively inspect `.js` and `.map` files and fail if they contain `StaticDemoHTTPError`, `static_demo_failed`, or `vendor-static-new`. This supplements the import-graph check and catches Vite config regressions.

- [ ] **Step 3: Run the full current-contract gates**

Run:

```bash
gofmt -w internal/runtimecontext internal/httpapi cmd/api internal/today
go test -race ./...
go test -tags postgres ./...
go vet ./...
cd web
npm run check:runtime-truth
npm run typecheck
npm test
npm run build
npm run check:runtime-truth
npm run build:evidence
```

Expected: PASS, with no legacy/backward-compatibility test additions.

- [ ] **Step 4: Update documentation and commit**

Update `README.md`, `docs/architecture/application-architecture.md`, and `docs/quality/acceptance-tests.md` to state that stakeholder demonstration uses PostgreSQL reference records and the evidence entry is synthetic, non-shipping test infrastructure.

```bash
git add .github/workflows/ci.yml README.md docs/architecture/application-architecture.md docs/quality/acceptance-tests.md web internal cmd
git commit -m "ci: enforce persisted runtime truth"
```

### Task 6: Render and inspect the affected states

- [ ] **Step 1: Capture the evidence entry and normal runtime separately**

Run `cd web && npm run review:ui` for deterministic evidence, then start the real API/web composition against a seeded PostgreSQL database and capture `/api/v1/context`, Today empty/populated states, Forms, Vendors, Matters, and Oversight.

- [ ] **Step 2: Inspect required display variants**

Inspect desktop and 390px mobile, light and dark mode, and 200% browser zoom. Confirm the evidence label never appears in the normal runtime and missing stored directory context produces the recovery notice without invented labels.

- [ ] **Step 3: Fix the highest-impact defect and re-render**

Any material visual fix must update its focused component test and, if it adds a token/variant, `DESIGN.md` in the same commit.

- [ ] **Step 4: Record the redacted evidence receipt**

Update `docs/quality/acceptance-tests.md` with exact command names and artifact directories only; do not include recipient data, secure URLs, OTPs, or secrets.
