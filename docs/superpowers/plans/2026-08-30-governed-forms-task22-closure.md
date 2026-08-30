# Governed Forms Task 22 Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close issue #94 with exact-release verification, bank-scale PostgreSQL proof, complete deterministic Forms state evidence, and synchronized documentation that identifies the external acceptance remainder.

**Architecture:** Keep Forms domain behavior in the existing monitoring, evidence and third-party modules. Add only one small runtime contract—the immutable release revision on readiness—and otherwise strengthen integration/evidence tooling around existing production components, repositories, events and histories. A capability-tagged Forms scenario registry drives a compact rendered cover and a validator proves that every Task 22 state and presentation dimension is represented.

**Tech Stack:** Go 1.26, pgx/PostgreSQL 18 integration tests, React 19/TypeScript/Vitest, Node.js 24, Playwright 1.55, Bash deployment tooling, GitHub Actions.

---

## File structure

- `internal/platform/config/config.go` owns validated release-revision configuration.
- `internal/platform/config/config_test.go` proves revision normalization and production fail-closed behavior.
- `internal/httpapi/server.go` carries the immutable revision into the HTTP boundary.
- `internal/httpapi/handlers.go` exposes readiness as `{status, mode, revision}`.
- `internal/httpapi/server_test.go` proves the public contract without changing the public route inventory.
- `cmd/api/main.go` composes the configured revision into the API.
- `deploy/compose.demo.yaml` binds the immutable image tag to `CLEARSIGHT_RELEASE_SHA`.
- `deploy/scripts/verify-hosted-release.sh` performs the read-only exact-SHA hosted verification.
- `deploy/tests/deployment_config_test.py` locks the deployment/revision/verifier contract.
- `.github/workflows/deploy-demo.yml` runs the hosted verifier after the constrained deployment succeeds.
- `internal/integration/form_system_scale_test.go` owns the deterministic bank-scale, bounded-query, claim and reconstruction proof.
- `web/scripts/forms-evidence-scenarios.mjs` is the single scenario/capability registry.
- `web/scripts/forms-evidence-scenarios.nodecheck.mjs` proves registry uniqueness and Task 22 coverage before a browser is started.
- `web/scripts/capture-forms-evidence.mjs` executes the registry using production Forms components and deterministic static fixtures.
- `web/src/staticDemo.ts` supplies only the missing deterministic Forms loading, unavailable and lifecycle variants.
- `web/src/staticDemo.test.ts` proves those variants through the same transport used by the application.
- `web/scripts/review-ui-flow-manifest.mjs` validates capability, viewport, theme and artifact coverage instead of duplicating a fixed Forms name list.
- `docs/implementation-plan.md`, `docs/product/governed-forms.md`, `docs/quality/acceptance-tests.md`, `docs/quality/rendered-ui-evidence.md`, and `docs/architecture/system-data-and-performance.md` synchronize completion truth and external dependencies.

### Task 1: Exact release revision on readiness

**Files:**
- Modify: `internal/platform/config/config.go`
- Modify: `internal/platform/config/config_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/handlers.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `cmd/api/main.go`
- Modify: `deploy/compose.demo.yaml`

- [ ] **Step 1: Write failing configuration tests**

Add tests that require a normalized 40-character lowercase Git SHA and reject malformed production values:

```go
func TestLoadNormalizesReleaseSHA(t *testing.T) {
	sha := strings.Repeat("A", 40)
	t.Setenv("CLEARSIGHT_RELEASE_SHA", sha)
	cfg, err := Load()
	if err != nil { t.Fatal(err) }
	if cfg.ReleaseSHA != strings.ToLower(sha) { t.Fatalf("release sha=%q", cfg.ReleaseSHA) }
}

func TestProductionRequiresValidReleaseSHAWhenProvided(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("CLEARSIGHT_RELEASE_SHA", "main")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CLEARSIGHT_RELEASE_SHA") {
		t.Fatalf("expected invalid release SHA rejection, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests and verify RED**

Run: `go test ./internal/platform/config -run 'TestLoadNormalizesReleaseSHA|TestProductionRequiresValidReleaseSHAWhenProvided' -count=1`

Expected: FAIL because `Config.ReleaseSHA` and validation do not exist.

- [ ] **Step 3: Implement the minimal configuration contract**

Add `ReleaseSHA string` to `Config`, load `CLEARSIGHT_RELEASE_SHA`, lowercase it, and validate non-empty values with `^[0-9a-f]{40}$`. Development may remain `unknown`; deployment always supplies an exact SHA.

```go
var releaseSHAPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

releaseSHA := strings.ToLower(env("CLEARSIGHT_RELEASE_SHA", ""))
if releaseSHA != "" && !releaseSHAPattern.MatchString(releaseSHA) {
	return Config{}, fmt.Errorf("CLEARSIGHT_RELEASE_SHA must be a 40-character Git commit SHA")
}
cfg.ReleaseSHA = releaseSHA
```

- [ ] **Step 4: Write the failing readiness contract test**

```go
func TestReadyReportsExactReleaseRevision(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(Dependencies{Logger: logger, Mode: "postgres", ReleaseSHA: strings.Repeat("a", 40)})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusOK { t.Fatalf("status=%d", response.Code) }
	if response.Body.String() != `{"mode":"postgres","revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","status":"ready"}`+"\n" {
		t.Fatalf("body=%s", response.Body.String())
	}
}
```

- [ ] **Step 5: Run the readiness test and verify RED**

Run: `go test ./internal/httpapi -run TestReadyReportsExactReleaseRevision -count=1`

Expected: FAIL because readiness omits `revision`.

- [ ] **Step 6: Pass the immutable revision through composition**

Add `ReleaseSHA` to `httpapi.Dependencies`, set it from `cfg.ReleaseSHA` in `cmd/api/main.go`, and return `unknown` only for local/test composition when the value is empty:

```go
func (a *API) ready(w http.ResponseWriter, _ *http.Request) {
	revision := strings.TrimSpace(a.deps.ReleaseSHA)
	if revision == "" { revision = "unknown" }
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ready", "mode": a.deps.Mode, "revision": revision,
	})
}
```

Bind the deployed image tag in `deploy/compose.demo.yaml`:

```yaml
environment:
  CLEARSIGHT_HTTP_ADDR: 127.0.0.1:13281
  CLEARSIGHT_RELEASE_SHA: ${CLEARSIGHT_IMAGE_TAG}
```

- [ ] **Step 7: Run focused and regression tests**

Run: `go test ./internal/platform/config ./internal/httpapi ./cmd/api -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

Run:

```powershell
git add internal/platform/config internal/httpapi cmd/api/main.go deploy/compose.demo.yaml
git commit -m "feat: report exact deployed revision"
```

### Task 2: Read-only hosted release verifier

**Files:**
- Create: `deploy/scripts/verify-hosted-release.sh`
- Modify: `deploy/tests/deployment_config_test.py`
- Modify: `.github/workflows/deploy-demo.yml`
- Modify: `deploy/scripts/release.sh`

- [ ] **Step 1: Write failing deployment-contract tests**

Add a test that requires the verifier to check readiness revision, authenticated session, bounded Today/Forms/Vendors reads, opaque denial and redaction, and requires the workflow/release bundle to carry the script:

```python
def test_hosted_verifier_is_exact_sha_read_only_and_redacted(self) -> None:
    verifier = self.read("deploy/scripts/verify-hosted-release.sh")
    for value in (
        'expected_sha="${1:?expected sha is required}"',
        "/health/ready", "/api/v1/demo/login", "/api/v1/session/status",
        "/api/v1/today", "/api/v1/forms/templates?limit=1",
        "/api/v1/vendors?limit=1", "/api/v1/evidence/access/start",
        '"revision"', '"authenticated":true', "invalid_access_selector",
    ):
        self.assertIn(value, verifier)
    self.assertNotIn("set -x", verifier)
    workflow = self.read(".github/workflows/deploy-demo.yml")
    self.assertIn("verify-hosted-release.sh", workflow)
    self.assertIn('"$RELEASE_SHA"', workflow)
```

- [ ] **Step 2: Run the test and verify RED**

Run: `python -m unittest deploy.tests.deployment_config_test.DeploymentConfigTest.test_hosted_verifier_is_exact_sha_read_only_and_redacted`

Expected: FAIL because the verifier does not exist.

- [ ] **Step 3: Implement the verifier**

Create a strict Bash script that uses a temporary cookie jar, validates JSON with `python3 -c`, never echoes bodies containing credentials, and removes the jar on exit. Its only POSTs are demo login and an invalid capability start that cannot mutate a record:

```bash
#!/usr/bin/env bash
set -Eeuo pipefail
expected_sha="${1:?expected sha is required}"
base_url="${2:-https://clearsight.cloudspacetechs.com}"
[[ "$expected_sha" =~ ^[0-9a-f]{40}$ ]]
cookie_jar="$(mktemp)"
trap 'rm -f "$cookie_jar"' EXIT

ready="$(curl -fsS "$base_url/health/ready")"
python3 -c 'import json,sys; d=json.load(sys.stdin); assert d=={"mode":"postgres","revision":sys.argv[1],"status":"ready"}' "$expected_sha" <<<"$ready"

curl -fsS -c "$cookie_jar" -H 'Content-Type: application/json' \
  --data '{"username":"system-admin@demo.clearsight.local","password":"demo"}' \
  "$base_url/api/v1/demo/login" >/dev/null

for path in /api/v1/session/status /api/v1/today '/api/v1/forms/templates?limit=1' '/api/v1/vendors?limit=1'; do
  body="$(curl -fsS -b "$cookie_jar" "$base_url$path")"
  python3 -c 'import json,sys; d=json.load(sys.stdin); assert isinstance(d,dict)' <<<"$body"
done

session="$(curl -fsS -b "$cookie_jar" "$base_url/api/v1/session/status")"
python3 -c 'import json,sys; assert json.load(sys.stdin).get("authenticated") is True' <<<"$session"

denial="$(curl -sS -w '\n%{http_code}' -H 'Content-Type: application/json' \
  --data '{"selector":"invalid_access_selector"}' "$base_url/api/v1/evidence/access/start")"
status="${denial##*$'\n'}"; body="${denial%$'\n'*}"
[[ "$status" == 404 || "$status" == 401 || "$status" == 422 ]]
[[ "$body" != *"distribution"* && "$body" != *"recipient"* && "$body" != *"audience_hint"* ]]
printf 'verified hosted release %s\n' "$expected_sha"
```

- [ ] **Step 4: Wire the verifier into deployment**

Copy the verifier into the release bundle, install it without exposing configuration, and invoke it in the GitHub runner after SSH deployment returns:

```yaml
- name: Verify hosted exact release
  if: steps.freshness.outputs.current == 'true'
  run: bash deploy/scripts/verify-hosted-release.sh "$RELEASE_SHA"
```

- [ ] **Step 5: Run deployment tests and shell syntax checks**

Run:

```powershell
python -m unittest deploy.tests.deployment_config_test
bash -n deploy/scripts/verify-hosted-release.sh
bash -n deploy/scripts/release.sh
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```powershell
git add deploy .github/workflows/deploy-demo.yml
git commit -m "test: verify the hosted exact release"
```

### Task 3: Bank-scale Forms, bounded work and reconstruction proof

**Files:**
- Modify: `internal/integration/form_system_scale_test.go`

- [ ] **Step 1: Split the scale test into named subtests and add failing cardinality assertions**

Introduce `formsScaleFixture` with stable IDs/timestamps and helpers `seedFormsScaleFixture`, `assertFormPagination`, `assertDistributionPagination`, `assertExactResponseLookup`, `assertBoundedReminderClaims`, `assertBoundedRefreshMaintenance`, and `assertFormsReconstruction`.

The first RED assertion must require the missing population:

```go
t.Run("population", func(t *testing.T) {
	for table, want := range map[string]int{
		"monitoring_form_templates": 1000,
		"capture_form_distributions": 400,
		"capture_distribution_recipients": 1200,
		"capture_response_revisions": 800,
	} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE tenant_id=$1::uuid", tenant).Scan(&got); err != nil { t.Fatal(err) }
		if got != want { t.Fatalf("%s=%d, want %d", table, got, want) }
	}
})
```

- [ ] **Step 2: Run the PostgreSQL test and verify RED**

Run: `go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/integration -run TestGovernedFormsStayBoundedAtBankScale`

Expected: FAIL with recipient/response counts of zero. If `TEST_DATABASE_URL` is absent locally, record the skip and use the CI PostgreSQL job for the mandatory RED/GREEN evidence before merge.

- [ ] **Step 3: Seed recipients, requests, submissions, workspaces and response revisions**

Use set-based SQL with deterministic UUIDs. Create three recipients and two immutable revisions per distribution, keep only revision two current, and keep all foreign keys valid. Use fixed `baseTime` values rather than `clock_timestamp()` for cursor/reconstruction determinism.

- [ ] **Step 4: Prove exact bounded queries and index shape**

Add behavior assertions for ten stable pages, cross-tenant/entity emptiness before `LIMIT`, exact response ID lookup scoped by tenant/entity/distribution, and normalized `EXPLAIN (COSTS OFF)` assertions for:

```sql
SELECT id FROM capture_response_revisions
WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
  AND distribution_id=$3::uuid AND id=$4::uuid
```

Also assert `capture_response_revisions_history_idx`, recipient/distribution indexes, and no `Seq Scan` in the normalized plan when `enable_seqscan=off`.

- [ ] **Step 5: Prove reminder and refresh work are bounded and deduplicated**

Run the real `evidence.CommunicationReminderScheduler` with a limit of seven twice. Assert the first call creates no more than the eligible bounded population, the second creates no duplicates, and outbox payloads contain no address/token values. Run the real `thirdparty.RefreshMaintainer` against seeded eligible/ineligible relationships with `BatchSize: 5`; assert no more than five attentions are created and a repeated run is idempotent.

- [ ] **Step 6: Prove point-in-time reconstruction**

Choose a timestamp between version one and version two and assert the historical value for:

```text
monitoring_form_templates
form_communication_template_revisions
capture_distribution_events
capture_response_revisions
third_party_response_application_receipts
third_party_document_events / superseded document lineage
```

Each assertion must compare the historical selection with the current selection and prove that the current row did not overwrite the material prior record.

- [ ] **Step 7: Run focused PostgreSQL verification**

Run: `go test -count=1 -p 1 -tags "postgres postgresintegration" ./internal/integration -run TestGovernedFormsStayBoundedAtBankScale -v`

Expected: PASS with all named subtests.

- [ ] **Step 8: Commit**

Run:

```powershell
git add internal/integration/form_system_scale_test.go
git commit -m "test: prove governed Forms scale and history"
```

### Task 4: Capability-tagged Forms evidence registry

**Files:**
- Create: `web/scripts/forms-evidence-scenarios.mjs`
- Create: `web/scripts/forms-evidence-scenarios.nodecheck.mjs`
- Modify: `web/scripts/capture-forms-evidence.mjs`
- Modify: `web/scripts/review-ui-flow-manifest.mjs`

- [ ] **Step 1: Write the failing registry test**

Define the required Task 22 capabilities exactly and require unique names, fixture keys and capture metadata:

```js
import assert from "node:assert/strict";
import test from "node:test";
import { formsEvidenceScenarios, requiredFormsCapabilities } from "./forms-evidence-scenarios.mjs";

test("Forms scenarios cover every Task 22 capability", () => {
  const covered = new Set(formsEvidenceScenarios.flatMap((scenario) => scenario.capabilities));
  assert.deepEqual(requiredFormsCapabilities.filter((capability) => !covered.has(capability)), []);
  assert.equal(new Set(formsEvidenceScenarios.map(({ name }) => name)).size, formsEvidenceScenarios.length);
  for (const scenario of formsEvidenceScenarios) {
    assert.match(scenario.name, /^\d{2,3}-forms-/);
    assert.ok(["light", "dark"].includes(scenario.theme));
    assert.ok(scenario.viewport.width >= 320);
  }
});
```

Capabilities must enumerate: library empty/list/search/saved-filter/recent/bulk-action; draft/pending/active/retired; invalid/valid weights; import pending/partial/truncated/failed/proposal; compose/delivered/fallback/amended/rotated/superseded/revoked; direct-link/shared-OTP/direct-OTP; OTP expired/exhausted; server-saved/device-only/conflict/recovered/file-reselection; first/amended response; vendor confirm/correct/replace/review/conflict/applied; desktop/mobile/reflow-320/zoom-200/light/dark.

- [ ] **Step 2: Run the registry test and verify RED**

Run: `cd web && node --test scripts/forms-evidence-scenarios.nodecheck.mjs`

Expected: FAIL because the registry does not exist.

- [ ] **Step 3: Implement the registry**

Export immutable scenario records with `{name, fixture, route, state, viewport, theme, touch, zoom, capabilities, run}`. Reuse the existing five captures and add the minimum representative scenarios needed to cover every capability without assigning a capability to a scenario whose rendered state does not show it.

- [ ] **Step 4: Refactor capture execution around the registry**

Replace the five top-level capture calls with:

```js
for (const scenario of formsEvidenceScenarios) {
  await captureScenario(scenario);
}
```

`captureScenario` must open `/?tour=off&fixture=<fixture>#forms`, apply theme/viewport/zoom, execute the scenario's concrete Playwright interaction, assert its dominant state, check browser errors/horizontal overflow, and write `capabilities`, `fixture`, `zoom` and `focus` into the manifest record.

- [ ] **Step 5: Replace fixed Forms names with capability validation**

Keep fixed names for the existing non-Forms suite. Derive expected Forms names and required Forms capabilities from the registry. Fail with exact missing capabilities, missing view classes or missing themes:

```js
const formCaptures = manifest.captures.filter(({ route }) => route === "#forms");
const covered = new Set(formCaptures.flatMap(({ capabilities = [] }) => capabilities));
const missing = requiredFormsCapabilities.filter((capability) => !covered.has(capability));
if (missing.length) failures.push(`Forms capability coverage is missing: ${missing.join(", ")}`);
```

- [ ] **Step 6: Run the registry and static manifest tests**

Run:

```powershell
cd web
node --test scripts/forms-evidence-scenarios.nodecheck.mjs
npm run typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```powershell
git add web/scripts
git commit -m "test: define governed Forms evidence coverage"
```

### Task 5: Deterministic Forms lifecycle and degraded fixtures

**Files:**
- Modify: `web/src/staticDemo.ts`
- Modify: `web/src/staticDemo.test.ts`
- Modify: `web/src/components/FormsWorkspace.test.tsx`
- Modify: `web/src/components/forms/Task11FormsViews.test.tsx`
- Modify: `web/src/components/capture/CaptureForm.test.tsx`
- Modify: `web/scripts/capture-forms-evidence.mjs`

- [ ] **Step 1: Write failing static transport tests**

Add table-driven tests for `forms-loading`, `forms-unavailable`, template lifecycle, distribution lifecycle and response-history fixtures. Assert the transport returns only truthful production-shaped fields and never exposes external addresses or capability values:

```ts
for (const [fixture, status] of [["forms-template-draft", "DRAFT"], ["forms-template-pending", "PENDING_APPROVAL"], ["forms-template-retired", "RETIRED"]] as const) {
  window.history.replaceState(null, "", `/?fixture=${fixture}`);
  const page = await staticDemoRequest<any>("/api/v1/forms/templates?limit=20");
  expect(page.items[0].template.status).toBe(status);
}
```

Add separate assertions for unavailable HTTP errors, saved filters, import status variants, distribution fallback/amended/rotated/superseded/revoked, and two response revisions.

- [ ] **Step 2: Run focused web tests and verify RED**

Run: `cd web && npm test -- staticDemo.test.ts FormsWorkspace.test.tsx Task11FormsViews.test.tsx CaptureForm.test.tsx`

Expected: FAIL because the fixture variants are not implemented.

- [ ] **Step 3: Implement minimal deterministic fixture branches**

Use `activeFixture()` to derive immutable variants from the existing `demoForms`, `staticDistributionDetail`, import proposals and response revisions. Delay only the specific Forms library call for loading; throw a production-shaped 503 only for the unavailable fixture. Do not add stateful behavior that production does not support.

- [ ] **Step 4: Add interaction assertions for non-visual behavior**

Extend existing component tests to prove saved-filter application, bulk transition eligibility, weight approval blocking, supersession confirmation, access-policy choices, OTP expired/exhausted recovery copy, server/device recovery conflict and file reselection. Tests must use production components and assert real disabled/enabled actions rather than mock call counts alone.

- [ ] **Step 5: Complete each Playwright scenario interaction**

Map every scenario to a concrete visible assertion. Examples:

```js
await page.getByRole("button", { name: "Sent forms" }).click();
await page.getByText("Superseded", { exact: true }).waitFor({ state: "visible" });
await page.getByRole("button", { name: "Responses" }).click();
await page.getByText("Revision 1", { exact: false }).waitFor({ state: "visible" });
```

For access and recovery states that live in external capture, navigate through the supported `capture_invite` bootstrap only with deterministic static values, confirm the URL is scrubbed, and capture the production external-capture component.

- [ ] **Step 6: Run focused behavior checks**

Run:

```powershell
cd web
npm test -- staticDemo.test.ts FormsWorkspace.test.tsx Task11FormsViews.test.tsx CaptureForm.test.tsx copyQuality.test.ts Accessibility.test.tsx
npm run typecheck
```

Expected: PASS.

- [ ] **Step 7: Run the complete rendered review**

Run: `cd web && npm run review:ui`

Expected: PASS with every Forms capability, desktop/mobile/reflow/zoom and light/dark dimension covered.

- [ ] **Step 8: Inspect and repair rendered evidence**

Inspect each new PNG at original resolution. Record one highest-impact issue, add a failing component or manifest assertion for it, implement the smallest correction, re-run the affected test and `npm run review:ui`, then inspect the corrected image again.

- [ ] **Step 9: Commit**

Run:

```powershell
git add web/src web/scripts
git commit -m "test: render the governed Forms state matrix"
```

### Task 6: Synchronize retention, acceptance and completion truth

**Files:**
- Modify: `docs/implementation-plan.md`
- Modify: `docs/product/governed-forms.md`
- Modify: `docs/quality/acceptance-tests.md`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/architecture/system-data-and-performance.md`

- [ ] **Step 1: Update the Forms workload/retention table**

Add a table covering templates, distributions/recipients/events, response workspaces/drafts/revisions, communications/delivery receipts, vendor application receipts, document history, access routes/sessions and OTP challenges. Each row must name cardinality/growth, retention status, index/partition trigger, maintenance owner, alert and recovery. Mark unapproved retention durations as policy dependencies and state that no deletion job is introduced by this tranche.

- [ ] **Step 2: Update acceptance truth**

Change the Forms acceptance section from a five-render summary to the capability-tagged matrix, exact PostgreSQL population, bounded work/reconstruction proof and exact-release hosted verifier. Keep the real 200%/assistive-technology, provider, object-storage/scanning, human timing, sustained load and disaster-recovery exercises explicitly open.

- [ ] **Step 3: Update product and implementation status**

Mark Task 22 repository evidence complete only after exact-head gates pass. State the automated population and evidence date, link issue #94 and the design/plan, and avoid “bug-free,” “fully compliant,” or enterprise-wide claims.

- [ ] **Step 4: Record rendered inspection evidence**

Document capture count, viewport/theme/zoom coverage, inspected artifact location, the highest-impact defect, the corrective test/change and the rechecked result. State that the 200% capture remains an approximation.

- [ ] **Step 5: Run documentation/copy checks**

Run:

```powershell
git diff --check
rg -n "TO.DO|T.BD|FIX.ME|bug[- ]free|fully compliant|enterprise-wide complete" docs/implementation-plan.md docs/product/governed-forms.md docs/quality/acceptance-tests.md docs/quality/rendered-ui-evidence.md docs/architecture/system-data-and-performance.md
cd web
npm test -- copyQuality.test.ts
```

Expected: `git diff --check` and copy tests PASS; phrase scan contains no new unsupported completion claim.

- [ ] **Step 6: Commit**

Run:

```powershell
git add docs
git commit -m "docs: close governed Forms release evidence"
```

### Task 7: Exact-head verification, PR, merge and deployment evidence

**Files:**
- Modify only when a failing verification exposes a defect; every defect requires a failing regression test before the fix.

- [ ] **Step 1: Run backend exact-head gates**

Run:

```powershell
$goFiles = rg --files cmd internal -g '*.go'
gofmt -w $goFiles
go test -race ./...
go test -tags postgres ./...
go vet ./...
python -m unittest deploy.tests.deployment_config_test
```

Expected: PASS with pristine output apart from documented PostgreSQL-integration skips when no local `TEST_DATABASE_URL` exists.

- [ ] **Step 2: Run frontend exact-head gates**

Run:

```powershell
cd web
npm ci --no-audit --no-fund
npm test
npm run typecheck
npm run build
npm run review:ui
```

Expected: PASS with no blocking axe, copy, runtime, overflow, state-coverage or bundle-budget findings.

- [ ] **Step 3: Verify the final diff and requirements**

Run:

```powershell
git diff --check origin/main...HEAD
git status --short
git log --oneline origin/main..HEAD
```

Review every section of `docs/superpowers/specs/2026-08-30-governed-forms-task22-closure-design.md` against the changed files and fresh evidence. Expected: clean worktree and no uncovered spec requirement.

- [ ] **Step 4: Push and open the PR**

Push `codex/forms-task22-closure-20260830`, open a PR referencing #94, and include the exact local verification commands, external remainder and rendered-evidence caveat.

- [ ] **Step 5: Wait for every required GitHub gate**

Require backend CI including serialized PostgreSQL integration, web CI, Compose runtime and UI/UX flow review on the exact PR head. Investigate failures with the systematic-debugging skill and add a failing regression test before any fix.

- [ ] **Step 6: Merge and verify main**

Merge only after the exact-head gates pass. Wait for main CI, Compose runtime, UI review and the deployment workflow. Confirm the hosted verifier reports the merged commit.

- [ ] **Step 7: Close issue #94 with exact evidence**

Comment with the merged PR, commit, CI/deploy run links, scale population, rendered artifact count and external remainder. Close #94 only if every automated exit criterion is satisfied. Leave #13, #57, #74 and #80 open.

## Plan self-review

- **Spec coverage:** Tasks 1–2 implement exact-release and hosted read-only proof; Task 3 covers scale, isolation, index, bounded work and six-family reconstruction; Tasks 4–5 cover the full capability-tagged state and presentation matrix with behavioral and rendered evidence; Task 6 covers retention/ownership and synchronized truth; Task 7 covers exact-head CI, merge, deployment and issue traceability.
- **No duplicate systems:** The plan adds no Forms store, workflow, authority route, task model or mutating hosted acceptance path. It reuses the current monitoring, evidence, third-party, static-fixture and deployment foundations.
- **Type consistency:** `ReleaseSHA`, `revision`, `formsEvidenceScenarios`, `requiredFormsCapabilities`, `capabilities`, `fixture`, `viewport`, `theme` and `zoom` retain the same names across tests, runtime and evidence tooling.
- **Safety:** Hosted verification is read-only except for demo-session establishment and an invalid capability request that cannot resolve a record. Tokens, cookies, recipient addresses and response bodies are not printed. PostgreSQL fixtures use isolated test tenants and existing serialized integration execution.
- **External truth:** Provider delivery, object storage/scanning, real browser 200%/assistive technology, human timing, sustained load/DR and a safe hosted mutating tenant remain explicit external acceptance items.
- **Placeholder scan:** The plan contains no forbidden placeholder marker, deferred implementation instruction, unspecified test or unnamed error-handling step.
