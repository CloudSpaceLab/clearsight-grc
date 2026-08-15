#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

# 1. Extend the existing evidence observation contract with exact source-access scope.
path = Path("internal/evidence/model.go")
text = path.read_text()
old = '''type SourceObservation struct {
\tID          string    `json:"id"`
\tTenantID    string    `json:"tenant_id"`
\tSourceID    string    `json:"source_id"`
\tObservedAt  time.Time `json:"observed_at"`
\tSuccess     bool      `json:"success"`
\tUnavailable bool      `json:"unavailable"`
\tLatencyMS   int       `json:"latency_ms,omitempty"`
\tDetail      string    `json:"detail,omitempty"`
\tRecordedBy  string    `json:"recorded_by,omitempty"`
}
'''
new = '''type SourceObservation struct {
\tID                string                 `json:"id"`
\tTenantID          string                 `json:"tenant_id"`
\tSourceID          string                 `json:"source_id"`
\tScope             SourceObservationScope `json:"scope,omitempty"`
\tConnectionID      string                 `json:"connection_id,omitempty"`
\tConnectionVersion int64                  `json:"connection_version,omitempty"`
\tViewID            string                 `json:"view_id,omitempty"`
\tViewVersion       int64                  `json:"view_version,omitempty"`
\tBindingID         string                 `json:"binding_id,omitempty"`
\tBindingVersion    int64                  `json:"binding_version,omitempty"`
\tObservedAt        time.Time              `json:"observed_at"`
\tSuccess           bool                   `json:"success"`
\tUnavailable       bool                   `json:"unavailable"`
\tLatencyMS         int                    `json:"latency_ms,omitempty"`
\tDetail            string                 `json:"detail,omitempty"`
\tRecordedBy        string                 `json:"recorded_by,omitempty"`
}
'''
if old not in text:
    raise SystemExit("SourceObservation shape changed")
path.write_text(text.replace(old, new, 1))

# 2. Route evidence source health through the scoped repository when available.
path = Path("internal/evidence/service.go")
text = path.read_text()
old = '''func (s *Service) RecordSourceObservation(ctx context.Context, observation SourceObservation) (Source, error) {
\tif strings.TrimSpace(observation.TenantID) == "" || strings.TrimSpace(observation.SourceID) == "" {
\t\treturn Source{}, fmt.Errorf("tenant and source are required")
\t}
\tif observation.ObservedAt.IsZero() {
\t\tobservation.ObservedAt = s.now().UTC()
\t}
\tvalueID, err := id.NewUUIDv7()
\tif err != nil {
\t\treturn Source{}, err
\t}
\tobservation.ID = valueID
\thealth := HealthDegraded
\tif observation.Unavailable {
\t\thealth = HealthUnavailable
\t} else if observation.Success {
\t\thealth = HealthCurrent
\t}
\treturn s.repo.RecordSourceObservation(ctx, observation, health)
}

func (s *Service) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
\tif now.IsZero() {
\t\tnow = s.now().UTC()
\t}
\tlimit = bounded(limit)
\texpired, err := s.repo.ExpireRequests(ctx, now, limit)
\tif err != nil {
\t\treturn expired, err
\t}
\tstale, err := s.repo.EvaluateSourceHealth(ctx, now, limit)
\treturn expired + stale, err
}
'''
new = '''func (s *Service) RecordSourceObservation(ctx context.Context, observation SourceObservation) (Source, error) {
\tif strings.TrimSpace(observation.TenantID) == "" || strings.TrimSpace(observation.SourceID) == "" {
\t\treturn Source{}, fmt.Errorf("tenant and source are required")
\t}
\tvar err error
\tobservation, err = normalizeSourceObservationScope(observation)
\tif err != nil {
\t\treturn Source{}, err
\t}
\tevaluatedAt := s.now().UTC()
\tif observation.ObservedAt.IsZero() {
\t\tobservation.ObservedAt = evaluatedAt
\t} else {
\t\tobservation.ObservedAt = observation.ObservedAt.UTC()
\t}
\tvalueID, err := id.NewUUIDv7()
\tif err != nil {
\t\treturn Source{}, err
\t}
\tobservation.ID = valueID
\tif scoped, ok := s.repo.(ScopedSourceHealthRepository); ok {
\t\treturn scoped.RecordScopedSourceObservation(ctx, observation, evaluatedAt)
\t}
\thealth := HealthDegraded
\tif observation.Unavailable {
\t\thealth = HealthUnavailable
\t} else if observation.Success {
\t\thealth = HealthCurrent
\t}
\treturn s.repo.RecordSourceObservation(ctx, observation, health)
}

func (s *Service) ListSourceScopeHealth(ctx context.Context, tenant, sourceID string, limit int) ([]SourceScopeHealth, error) {
\tif strings.TrimSpace(tenant) == "" || strings.TrimSpace(sourceID) == "" {
\t\treturn nil, fmt.Errorf("tenant and source are required")
\t}
\tscoped, ok := s.repo.(ScopedSourceHealthRepository)
\tif !ok {
\t\treturn nil, fmt.Errorf("scoped source health is unavailable")
\t}
\treturn scoped.ListSourceScopeHealth(ctx, tenant, sourceID, s.now().UTC(), healthLimit(limit))
}

func (s *Service) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
\tif now.IsZero() {
\t\tnow = s.now().UTC()
\t}
\tlimit = bounded(limit)
\texpired, err := s.repo.ExpireRequests(ctx, now, limit)
\tif err != nil {
\t\treturn expired, err
\t}
\tif scoped, ok := s.repo.(ScopedSourceHealthRepository); ok {
\t\tstale, scopedErr := scoped.EvaluateScopedSourceHealth(ctx, now, limit)
\t\treturn expired + stale, scopedErr
\t}
\tstale, err := s.repo.EvaluateSourceHealth(ctx, now, limit)
\treturn expired + stale, err
}
'''
if old not in text:
    raise SystemExit("Evidence observation service block changed")
path.write_text(text.replace(old, new, 1))

# 3. Bind observation tenant/recorder to verified identity even when handler is called directly.
path = Path("internal/httpapi/evidence_handlers.go")
text = path.read_text()
old = '''\tvar input evidence.SourceObservation
\tif err := httpx.DecodeJSON(w, r, &input); err != nil {
\t\thttpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
\t\treturn
\t}
\tinput.SourceID = r.PathValue("id")
\tvalue, err := service.RecordSourceObservation(r.Context(), input)
'''
new = '''\tactor, identityErr := identity.Require(r.Context())
\tif identityErr != nil {
\t\thttpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
\t\treturn
\t}
\tvar input evidence.SourceObservation
\tif err := httpx.DecodeJSON(w, r, &input); err != nil {
\t\thttpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
\t\treturn
\t}
\tinput.TenantID = actor.TenantID
\tinput.SourceID = r.PathValue("id")
\tinput.RecordedBy = actor.PrincipalID
\tvalue, err := service.RecordSourceObservation(r.Context(), input)
'''
if old not in text:
    raise SystemExit("Evidence observation handler block changed")
path.write_text(text.replace(old, new, 1))

# 4. Register the scoped health read under CONFIG_READ.
path = Path("internal/httpapi/route_registry.go")
text = path.read_text()
needle = '\t\twithPermission(read("/api/v1/config/sources/{source_id}/connections", a.listSourceConnections), identity.PermissionConfigRead),\n'
replacement = needle + '\t\twithPermission(read("/api/v1/config/sources/{source_id}/health", a.sourceScopeHealth), identity.PermissionConfigRead),\n'
if needle not in text:
    raise SystemExit("source config route insertion point changed")
path.write_text(text.replace(needle, replacement, 1))

# 5. Keep the executable runtime route contract exact.
path = Path("api/runtime.openapi.json")
text = path.read_text()
needle = '''    "/api/v1/config/sources/{source_id}/connections": {
'''
replacement = '''    "/api/v1/config/sources/{source_id}/health": { "get": { "operationId": "sourceScopeHealth", "x-clearsight-route-class": "AUTHENTICATED_READ", "x-clearsight-permission": "CONFIG_READ" } },
    "/api/v1/config/sources/{source_id}/connections": {
'''
if needle not in text:
    raise SystemExit("runtime OpenAPI source-config insertion point changed")
path.write_text(text.replace(needle, replacement, 1))

# 6. Correct the ranked scope-health SQL aliases (PostgreSQL does not invent coalesce_1 names).
path = Path("internal/evidence/source_health_postgres.go")
text = path.read_text()
text = text.replace(
'''\t\t\tSELECT so.scope_kind,COALESCE(so.connection_id::text,''),COALESCE(so.connection_version,0),
\t\t\t       COALESCE(so.view_id::text,''),COALESCE(so.view_version,0),
\t\t\t       COALESCE(so.binding_id::text,''),COALESCE(so.binding_version,0),
''',
'''\t\t\tSELECT so.scope_kind,
\t\t\t       COALESCE(so.connection_id::text,'') AS connection_id,COALESCE(so.connection_version,0) AS connection_version,
\t\t\t       COALESCE(so.view_id::text,'') AS view_id,COALESCE(so.view_version,0) AS view_version,
\t\t\t       COALESCE(so.binding_id::text,'') AS binding_id,COALESCE(so.binding_version,0) AS binding_version,
''', 1)
text = text.replace(
'''\t\tSELECT scope_kind,coalesce,coalesce_1,coalesce_2,coalesce_3,coalesce_4,coalesce_5,
\t\t       observed_at,success,unavailable,latency_ms,last_success_at
\t\t  FROM ranked
\t\t WHERE row_number=1
\t\t ORDER BY scope_kind,coalesce,coalesce_2,coalesce_4,observed_at DESC
''',
'''\t\tSELECT scope_kind,connection_id,connection_version,view_id,view_version,binding_id,binding_version,
\t\t       observed_at,success,unavailable,latency_ms,last_success_at
\t\t  FROM ranked
\t\t WHERE row_number=1
\t\t ORDER BY scope_kind,connection_id,view_id,binding_id,observed_at DESC
''', 1)
path.write_text(text)

# 7. Update the source-access architecture from T1a limitations to completed T1 runtime state.
path = Path("docs/architecture/connected-source-access.md")
text = path.read_text()
old = "The current repository supports revision creation, exact-version reads, current-version reads and bounded current-child lists. Lifecycle transitions, maker-checker administration and user-facing configuration are not part of this change."
new = "The repository supports revision creation, exact-version reads, current-version reads and bounded revision-history lists. Permissioned `CONFIG_READ` / `CONFIG_WRITE` routes expose server-owned draft creation, immutable schema inspection, bounded preview and where-used discovery. Lifecycle activation and maker-checker transitions remain a later governance tranche rather than implicit catalog mutation."
if old not in text:
    raise SystemExit("connected-source repository paragraph changed")
text = text.replace(old, new, 1)
anchor = "## Current limitations\n"
addition = '''## Stateful Binding checkpoints

Bindings that support `PAGE` or `CHANGES` may own one infrastructure checkpoint for an exact Binding revision. The checkpoint records only a bounded cursor, ETag, watermark or event ID plus runtime lease/retry state; it is not business truth and it does not copy source rows.

Checkpoint workers use the same claim/lease/backoff semantics as ClearSight runtime work. Advancement requires an existing durable runtime inbox receipt for the corresponding consumer/event. If processing commits and the process dies before checkpoint advancement, the lease expires, the same source position is replayed, the inbox receipt suppresses duplicate domain processing, and the checkpoint can then advance without skipping records.

Raw source errors are not persisted in checkpoint state. Only bounded error codes are retained.

## Scoped source health

The existing `source_observations` history now accepts exact `SOURCE`, `CONNECTION`, `VIEW` and `BINDING` scopes. Child-scoped observations retain exact parent revisions. The current health of each exact scope is derived from its latest observation, and Evidence Source health is the worst current scoped state (`UNAVAILABLE` → `STALE` → `DEGRADED` → `UNKNOWN` → `CURRENT`). An unrelated healthy path therefore cannot hide a failed Binding.

Freshness maintenance re-evaluates successful observations using the Source freshness window. Observations that arrive out of order remain historical evidence but cannot replace a newer observation for the same scope. Health remains part of the existing Evidence Source / Source Observation model; no connector-health authority is introduced.

'''
if anchor not in text:
    raise SystemExit("connected-source limitations heading changed")
text = text.replace(anchor, addition + anchor, 1)
text = text.replace("- Catalog administration has no API or user interface.\n", "- Catalog configuration has APIs but no dedicated user interface yet.\n", 1)
text = text.replace("- Connection-, View- and Binding-level health is not reconciled into Source health.\n", "", 1)
text = text.replace("- Cursor, ETag and watermark checkpoints are not stored.\n", "", 1)
path.write_text(text)

# 8. Register the infrastructure checkpoint table and scoped observation semantics in durable ownership.
path = Path("docs/architecture/durable-schema-ownership.md")
text = path.read_text()
old = "| `source_bindings` | active authoritative state | source access | source catalog repository | assurance and configured consumers | immutable revision lineage tied to an exact View revision | retain revision history while consumers or reconstruction require it | `internal/sourceaccess/catalog_postgres_binding.go`; migration `000030_source_access_catalog`; source catalog integration |\n| `source_observations` | active authoritative state | evidence | evidence observation service | source health and reconciliation | append-only observed/valid timestamps | retain governed source-health history | evidence/reconciliation integration |"
new = "| `source_bindings` | active authoritative state | source access | source catalog repository | assurance and configured consumers | immutable revision lineage tied to an exact View revision | retain revision history while consumers or reconstruction require it | `internal/sourceaccess/catalog_postgres_binding.go`; migration `000030_source_access_catalog`; source catalog integration |\n| `source_binding_checkpoints` | active infrastructure ledger | source access / runtime execution | leased source-access workers after durable consumer processing | source-access workers and operational diagnostics | one retry/lease/checkpoint position per exact stateful Binding revision | retain while the Binding is statefully consumed; never business truth | `internal/sourceaccess/checkpoint_postgres.go`; runtime inbox receipt integration; migration `000031_source_access_runtime_state` |\n| `source_observations` | active authoritative state | evidence | evidence observation service | scoped source health and Source-level reconciliation | append-only observations scoped to exact Source/Connection/View/Binding revisions | retain governed source-health history | `internal/evidence/source_health_postgres.go`; migration `000031_source_access_runtime_state`; scoped-health integration |"
if old not in text:
    raise SystemExit("schema ownership source-access rows changed")
path.write_text(text.replace(old, new, 1))

# 9. Make the T1c migration the explicit rollback/reapply proof while retaining migration 30 endpoint coverage.
path = Path(".github/workflows/ci.yml")
text = path.read_text()
start = text.index("      - name: Verify latest migration rollback and reapply\n")
end = text.index("      - name: Verify deployment migration ledger\n", start)
replacement = '''      - name: Verify source-access runtime rollback and reapply
        env:
          PGPASSWORD: clearsight
        run: |
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -f migrations/000031_source_access_runtime_state.down.sql
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT CASE WHEN to_regclass('public.source_binding_checkpoints') IS NULL THEN 'ok' ELSE 'checkpoint table survived rollback' END" | grep -qx ok
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='source_observations' AND column_name='scope_kind'" | grep -qx 0
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -f migrations/000030_source_access_catalog.down.sql
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 <<'SQL'
          INSERT INTO tenants(id,slug,name)
          VALUES('6f300000-0000-7000-8000-000000000001','source-catalog-migration','Source catalog migration');
          INSERT INTO evidence_sources(
              id,tenant_id,code,name,source_type,authority_class,endpoint,
              expected_freshness_minutes,health,status,version
          ) VALUES (
              '6f300000-0000-7000-8000-000000000002',
              '6f300000-0000-7000-8000-000000000001',
              'LEGACY-ENDPOINT','Legacy endpoint','SYSTEM','SYSTEM_OF_RECORD',
              ' https://legacy.example.invalid/source ',15,'UNKNOWN','ACTIVE',1
          );
          SQL
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -f migrations/000030_source_access_catalog.up.sql
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='evidence_sources' AND column_name='endpoint'" | grep -qx 0
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM source_connections WHERE source_id='6f300000-0000-7000-8000-000000000002' AND code='PRIMARY_REFERENCE' AND adapter_kind='REFERENCE' AND adapter_version='reference-v1' AND status='ACTIVE' AND is_current AND version=1 AND definition->>'endpoint'='https://legacy.example.invalid/source'" | grep -qx 1
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -f migrations/000031_source_access_runtime_state.up.sql
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT CASE WHEN to_regclass('public.source_binding_checkpoints') IS NOT NULL THEN 'ok' ELSE 'missing checkpoint table after reapply' END" | grep -qx ok
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='source_observations' AND column_name IN ('scope_kind','connection_id','connection_version','view_id','view_version','binding_id','binding_version')" | grep -qx 7
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -c "DELETE FROM source_connections WHERE source_id='6f300000-0000-7000-8000-000000000002'; DELETE FROM evidence_sources WHERE id='6f300000-0000-7000-8000-000000000002'; DELETE FROM tenants WHERE id='6f300000-0000-7000-8000-000000000001';"
          psql -h localhost -U clearsight -d clearsight -v ON_ERROR_STOP=1 -Atc "SELECT CASE WHEN to_regclass('public.audit_events') IS NULL AND to_regclass('public.readiness_snapshots') IS NULL THEN 'ok' ELSE 'unexpected compatibility table' END" | grep -qx ok
'''
text = text[:start] + replacement + text[end:]
path.write_text(text)

# 10. Promote the tranche verifier from T1b to complete T1.
path = Path("scripts/verify-sourceaccess-tranche.sh")
text = path.read_text()
old = '''if [[ "$tranche" == "auto" ]]; then
  if [[ -f internal/sourceaccess/catalog_service.go && -f internal/httpapi/source_catalog_handlers.go ]]; then
    tranche="t1b"
  elif [[ -f migrations/000030_source_access_catalog.up.sql ]]; then
'''
new = '''if [[ "$tranche" == "auto" ]]; then
  if [[ -f migrations/000031_source_access_runtime_state.up.sql && -f internal/sourceaccess/checkpoint.go && -f internal/evidence/source_health_scoped.go ]]; then
    tranche="t1c"
  elif [[ -f internal/sourceaccess/catalog_service.go && -f internal/httpapi/source_catalog_handlers.go ]]; then
    tranche="t1b"
  elif [[ -f migrations/000030_source_access_catalog.up.sql ]]; then
'''
if old not in text:
    raise SystemExit("sourceaccess auto tranche block changed")
text = text.replace(old, new, 1)
text = text.replace('if [[ "$tranche" != "t0" && "$tranche" != "t1a" && "$tranche" != "t1b" ]]; then\n  echo "usage: $0 [auto|t0|t1a|t1b]" >&2', 'if [[ "$tranche" != "t0" && "$tranche" != "t1a" && "$tranche" != "t1b" && "$tranche" != "t1c" ]]; then\n  echo "usage: $0 [auto|t0|t1a|t1b|t1c]" >&2', 1)
old = '''    summary="T0 and T1a plus T1b governed draft operations, bounded inspect/preview, revision-history reads, route permissions and runtime contract passed."
  fi
fi
'''
new = '''    if [[ "$tranche" == "t1b" ]]; then
      summary="T0 and T1a plus T1b governed draft operations, bounded inspect/preview, revision-history reads, route permissions and runtime contract passed."
    else
      required_t1c=(
        internal/sourceaccess/checkpoint.go
        internal/sourceaccess/checkpoint_memory.go
        internal/sourceaccess/checkpoint_postgres.go
        internal/sourceaccess/checkpoint_runtime_postgres_integration_test.go
        internal/evidence/source_health_scoped.go
        internal/evidence/source_health_memory.go
        internal/evidence/source_health_postgres.go
        internal/httpapi/source_health_handlers.go
        migrations/000031_source_access_runtime_state.up.sql
        migrations/000031_source_access_runtime_state.down.sql
      )
      for file in "${required_t1c[@]}"; do
        test -s "$file" || { echo "missing T1c contract: $file" >&2; exit 1; }
      done
      grep -q 'InboxProcessed' internal/sourceaccess/checkpoint.go || {
        echo "T1c checkpoint advancement is not tied to the runtime inbox receipt" >&2
        exit 1
      }
      grep -q 'source_binding_checkpoints' docs/architecture/durable-schema-ownership.md || {
        echo "source_binding_checkpoints is missing from durable schema ownership" >&2
        exit 1
      }
      grep -q '/api/v1/config/sources/{source_id}/health' api/runtime.openapi.json || {
        echo "scoped source health is missing from the runtime contract" >&2
        exit 1
      }
      if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
        psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT CASE WHEN to_regclass('public.source_binding_checkpoints') IS NOT NULL THEN 'ok' ELSE 'missing source_binding_checkpoints' END" | grep -qx ok
        psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='source_observations' AND column_name IN ('scope_kind','connection_id','connection_version','view_id','view_version','binding_id','binding_version')" | grep -qx 7
        psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal AND tgname IN ('source_observation_scope_guard_trigger','source_binding_checkpoint_guard_trigger')" | grep -qx 2
      fi
      go test ./internal/sourceaccess ./internal/evidence ./internal/httpapi ./internal/runtime
      go test -tags postgres ./internal/sourceaccess ./internal/evidence ./internal/httpapi ./internal/runtime
      summary="Complete T1 durable reusable sources passed: catalog, governed operations, replay-safe checkpoints, scoped health aggregation, schema ownership and rollback/reapply controls."
    fi
  fi
fi
'''
if old not in text:
    raise SystemExit("sourceaccess T1b summary block changed")
path.write_text(text.replace(old, new, 1))
PY

gofmt -w \
  internal/sourceaccess/checkpoint.go \
  internal/sourceaccess/checkpoint_memory.go \
  internal/sourceaccess/checkpoint_postgres.go \
  internal/sourceaccess/checkpoint_test.go \
  internal/sourceaccess/checkpoint_runtime_postgres_integration_test.go \
  internal/evidence/model.go \
  internal/evidence/service.go \
  internal/evidence/source_health_scoped.go \
  internal/evidence/source_health_memory.go \
  internal/evidence/source_health_postgres.go \
  internal/evidence/source_health_scoped_test.go \
  internal/httpapi/evidence_handlers.go \
  internal/httpapi/source_health_handlers.go \
  internal/httpapi/source_health_handlers_test.go \
  internal/httpapi/route_registry.go

go test ./internal/sourceaccess ./internal/evidence ./internal/httpapi ./internal/runtime
go test -tags postgres ./internal/sourceaccess ./internal/evidence ./internal/httpapi ./internal/runtime

if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
  for migration in migrations/*.up.sql; do
    psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration" >/dev/null
  done
  bash scripts/verify-sourceaccess-tranche.sh t1c
  go test -p 1 -tags "postgres postgresintegration" ./internal/sourceaccess ./internal/evidence ./internal/httpapi ./internal/runtime
fi

rm -f .github/workflows/sourceaccess-t1c-closeout.yml scripts/apply-sourceaccess-t1c-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add -A
git commit -m "feat(sourceaccess): complete durable T1 runtime state"
git push origin HEAD:codex/issue-61-sourceaccess-t1c
