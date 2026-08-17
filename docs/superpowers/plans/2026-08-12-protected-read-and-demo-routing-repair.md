# Protected Read and Demo Routing Repair Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Make actor-authorized PostgreSQL records use canonical tenant identity, provide governed demo authority routes, and remove the intentional signed-out demo 401 probe.

**Architecture:** PostgreSQL repositories continue accepting a tenant UUID or slug as lookup input but emit tenants.id::text in actor-facing records. The existing demo foundation script installs a stable effective-dated routing policy through the current authority projection. A public identity-safe status endpoint lets the React gate decide whether to request protected context.

**Tech Stack:** Go 1.26, pgx, PostgreSQL 18, Bash/psql, React 19, TypeScript, Vite, Vitest, GitHub Actions, Docker Compose.

---

## File structure

- internal/continuity current and summary PostgreSQL readers: canonical Program and Matter identity.
- internal/evidence PostgreSQL readers: canonical request, source, recipient and artifact identity.
- internal/workflow/postgres.go: canonical actor-work identity.
- deploy/scripts/seed-demo-foundation.sh: idempotent governed demo routing.
- internal/httpapi/session_status.go and route contract: safe session discovery.
- web/src/api.ts and DemoAuthGate: status-first authentication flow.
- PostgreSQL, API, React and deployment tests: red/green and negative access proof.

### Task 1: Canonical tenant identity in protected PostgreSQL reads

**Files:**
- Modify: internal/continuity/current_postgres.go
- Modify: internal/continuity/summaries_postgres.go
- Modify: internal/evidence/postgres.go
- Modify: internal/evidence/lookup_postgres.go
- Modify: internal/evidence/visibility_postgres.go
- Modify: internal/evidence/artifact_lookup_postgres.go
- Modify: internal/evidence/recipient_postgres.go
- Modify: internal/workflow/postgres.go
- Test: internal/continuity/matter_summary_visibility_postgres_integration_test.go
- Test: internal/evidence/recipient_postgres_integration_test.go
- Test: internal/workflow/evidence_request_projector_postgres_integration_test.go

- [ ] **Step 1: Write failing canonical-identity assertions**

For fixtures whose lookup slug differs from their UUID, authenticate and query with the UUID. Assert returned Matter, evidence request and workflow Task TenantID values equal the UUID. Retain negative assertions for another tenant, wrong recipient and restricted non-member.

    actor := identity.WithActor(ctx, identity.Actor{TenantID: tenantID, PrincipalID: principalA})
    value, err := current.GetMatter(actor, tenantID, visibleID)
    if err != nil { t.Fatal(err) }
    if value.Matter.TenantID != tenantID {
        t.Fatalf("Matter tenant identity = %q, want %q", value.Matter.TenantID, tenantID)
    }

- [ ] **Step 2: Run focused PostgreSQL tests and verify failure**

Run: go test -p 1 -tags "postgres postgresintegration" ./internal/continuity ./internal/evidence ./internal/workflow

Expected: the new assertions report a slug where the canonical UUID is required.

- [ ] **Step 3: Emit canonical UUIDs without changing lookup ergonomics**

Change actor-facing selected fields from t.slug to t.id::text. Keep predicates accepting either UUID or slug.

    SELECT er.id::text,t.id::text,er.subject_type,er.subject_id
    FROM capture_requests er
    JOIN tenants t ON t.id=er.tenant_id
    WHERE (t.id::text=$1 OR t.slug=$1)

For aggregate JSON, build tenant_id with t.id::text. Audit every selected t.slug in the listed files. Leave slug-based maintenance cursors and lookup predicates intact when they are not serialized into domain records.

- [ ] **Step 4: Run focused and composition tests**

Run:
- go test -p 1 -tags "postgres postgresintegration" ./internal/continuity ./internal/evidence ./internal/workflow
- go test -tags postgres ./...

Expected: both exit zero and all negative visibility tests still pass.

- [ ] **Step 5: Commit**

    git add internal/continuity internal/evidence internal/workflow
    git commit -m "fix(postgres): canonicalize protected read tenant identity"

### Task 2: Governed idempotent demo authority routes

**Files:**
- Modify: deploy/scripts/seed-demo-foundation.sh
- Modify: deploy/tests/deployment_config_test.py
- Test: internal/authority/postgres_integration_test.go

- [ ] **Step 1: Write failing fixture and resolution tests**

Require stable policy code CLEARSIGHT-DEMO-AUTHORITY, routing_policy_versions, refresh_effective_authority_routes, maker principal 00000000-0000-4000-8000-000000000104 and checker principal 00000000-0000-4000-8000-000000000106 in the foundation script.

Add an authority integration case with direct rules for ACCOUNTABLE_OWNER, REVIEWER, AUTHORIZER, SIGNATORY, TRANSMITTER and ACKNOWLEDGEMENT_RECORDER. Assert exact PROGRAM, program.transition, materiality 3 input resolves only the intended authorizer.

- [ ] **Step 2: Run tests and verify failure**

Run:
- python -m unittest deploy.tests.deployment_config_test
- go test -p 1 -tags "postgres postgresintegration" ./internal/authority

Expected: the deployment contract fails because the foundation has no policy fixture.

- [ ] **Step 3: Add the stable policy**

Inside the foundation transaction, insert one demo-scoped active routing policy and version with stable UUIDs. GRC Administrator is maker and Internal Auditor is independent checker. Use direct PRINCIPAL selectors and exact legal entity UUID. Include this Program rule plus wildcard-object journey responsibilities:

    {
      "id": "demo-program-authorizer",
      "legal_entity_id": "00000000-0000-4000-8000-000000000002",
      "object_type": "PROGRAM",
      "responsibility": "AUTHORIZER",
      "decision_type": "program.transition",
      "min_materiality": 0,
      "priority": 100,
      "selector": {"kind": "PRINCIPAL", "ref": "00000000-0000-4000-8000-000000000101"}
    }

Validate incompatible code/ID collisions explicitly. Finish with refresh_effective_authority_routes for the demo tenant UUID.

- [ ] **Step 4: Prove repeatability and resolution**

Run the foundation twice against a migrated test database. Assert one policy, one current version and the expected route count. Resolve Program transition and all configured responsibilities. Run authority integrity and require no unresolved selector.

- [ ] **Step 5: Commit**

    git add deploy/scripts/seed-demo-foundation.sh deploy/tests/deployment_config_test.py internal/authority/postgres_integration_test.go
    git commit -m "fix(demo): install governed authority routes"

### Task 3: Identity-safe session discovery

**Files:**
- Create: internal/httpapi/session_status.go
- Modify: internal/httpapi/route_registry.go
- Modify: internal/httpapi/demo_routes_test.go
- Modify: api/runtime.openapi.json

- [ ] **Step 1: Write failing API tests**

GET /api/v1/session/status while signed out must return status 200 with authenticated false and demo_login_available true, with no principal, tenant, role or legal entity fields. Protected context must remain 401. With a login cookie, status must report authenticated true without identity fields.

- [ ] **Step 2: Verify red**

Run: go test ./internal/httpapi -run "TestDemo|TestRuntimeOpenAPI"

Expected: the new status request receives 404.

- [ ] **Step 3: Implement endpoint and contract**

The handler reads identity.FromContext and whether dependencies provide a DemoSessionAuthenticator. It writes only two booleans. Register GET /api/v1/session/status as PUBLIC and document it with empty security and PUBLIC route class in api/runtime.openapi.json.

- [ ] **Step 4: Verify green**

Run:
- go test ./internal/httpapi -run "TestDemo|TestRuntimeOpenAPI"
- go test ./internal/httpapi

Expected: all pass; context remains protected.

- [ ] **Step 5: Commit**

    git add internal/httpapi api/runtime.openapi.json
    git commit -m "fix(auth): add quiet session discovery"

### Task 4: Status-first React authentication gate

**Files:**
- Modify: web/src/api.ts
- Modify: web/src/components/DemoAuthGate.tsx
- Modify: web/src/components/DemoAuthGate.test.tsx

- [ ] **Step 1: Write failing behavior tests**

Mock loadSessionStatus. For authenticated false and demo_login_available true, assert the role login appears and loadContext was never called. Add cases for authenticated sessions, failed status fallback, live-preview role-switch hiding, and login followed by context load.

- [ ] **Step 2: Verify red**

Run: npm test -- DemoAuthGate.test.tsx

Expected: the API has no loadSessionStatus export or loadContext is called before login.

- [ ] **Step 3: Implement status-first behavior**

Add SessionStatus and loadSessionStatus in api.ts. In DemoAuthGate, request status first. Only request protected context when authenticated. Load demo accounts immediately when signed out and demo login is available. If status discovery itself fails, retain the existing context-first degraded fallback. Never infer authentication from the query string or browser state.

- [ ] **Step 4: Verify web**

Run:
- npm test
- npm run typecheck
- npm run build

Expected: no failed tests, TypeScript exits zero and the production bundle builds.

- [ ] **Step 5: Commit**

    git add web/src/api.ts web/src/components/DemoAuthGate.tsx web/src/components/DemoAuthGate.test.tsx
    git commit -m "fix(web): avoid protected pre-login probe"

### Task 5: Full verification, deployment and live acceptance

**Files:**
- Modify: docs/engineering/demo-role-login.md
- Modify: docs/engineering/demo-deployment.md

- [ ] **Step 1: Update documentation**

Document boolean-only status discovery, protected context behavior, canonical PostgreSQL tenant UUIDs and idempotent governed demo authority repair.

- [ ] **Step 2: Run full local release gates**

Run:
- gofmt on all cmd and internal Go files, then ensure gofmt -l is empty
- go test ./...
- go test -tags postgres ./...
- go test -p 1 -tags "postgres postgresintegration" ./internal/...
- go vet ./...
- python -m unittest deploy.tests.deployment_config_test
- from web: npm test, npm run typecheck, npm run build

Expected: every command exits zero.

- [ ] **Step 3: Inspect acceptance and diff**

Run git diff --check, git status --short and git log -5 --oneline. Confirm exact tenant equality remains in HTTP checks, restricted access is unchanged, no PostgreSQL service was added and demo=0 handling is unchanged.

- [ ] **Step 4: Commit documentation and push main**

    git add docs/engineering/demo-role-login.md docs/engineering/demo-deployment.md docs/superpowers/plans/2026-08-12-protected-read-and-demo-routing-repair.md
    git commit -m "docs: record demo routing and session repair"
    git push origin main

- [ ] **Step 5: Wait for CI and auto-deployment**

Require the pushed SHA's CI and dependent deployment runs to conclude successfully. Then require /health/ready to be ready.

- [ ] **Step 6: Verify original symptoms live**

With fresh sessions:
1. Session status is 200 signed out and leaks no identity.
2. Signed-out landing makes no context request.
3. CRO can GET the reported regulatory Matter.
4. Program Owner can GET the reported evidence request.
5. CRO gets 200 for exact Program transition authority resolution.
6. A non-recipient still gets 404 for the evidence request.
7. Browser network/console on default and demo=0 has none of the reported intentional 401, authorized-record 404 or missing-route 422 errors.

- [ ] **Step 7: Record evidence**

Report deployed SHA, CI/deployment run links, health, the two original UUID response codes, policy version, negative access response and browser-console result.
