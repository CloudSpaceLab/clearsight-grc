#!/usr/bin/env bash
set -euo pipefail

tranche="${1:-auto}"
if [[ "$tranche" == "auto" ]]; then
  if [[ -f internal/sourceevent/adapter.go && -f internal/sourceevent/projector.go ]]; then
    tranche="t2c"
  elif [[ -f internal/documentimport/sourceaccess_adapter.go && -f migrations/000032_document_import_tabular_metadata.up.sql ]]; then
    tranche="t2b"
  elif [[ -f internal/sourceaccess/rest_json_adapter.go && -f internal/sourceaccess/rest_json_read.go ]]; then
    tranche="t2a"
  elif [[ -f migrations/000031_source_access_runtime_state.up.sql && -f internal/sourceaccess/checkpoint.go && -f internal/evidence/source_health_scoped.go ]]; then
    tranche="t1c"
  elif [[ -f internal/sourceaccess/catalog_service.go && -f internal/httpapi/source_catalog_handlers.go ]]; then
    tranche="t1b"
  elif [[ -f migrations/000030_source_access_catalog.up.sql ]]; then
    tranche="t1a"
  else
    tranche="t0"
  fi
fi
if [[ "$tranche" != "t0" && "$tranche" != "t1a" && "$tranche" != "t1b" && "$tranche" != "t1c" && "$tranche" != "t2a" && "$tranche" != "t2b" && "$tranche" != "t2c" ]]; then
  echo "usage: $0 [auto|t0|t1a|t1b|t1c|t2a|t2b|t2c]" >&2
  exit 2
fi

required_t0=(
  internal/sourceaccess/contracts_types.go
  internal/sourceaccess/contracts_validation.go
  internal/sourceaccess/postgres_adapter.go
  internal/sourceaccess/postgres_definition.go
  internal/sourceaccess/postgres_read.go
  internal/sourceaccess/postgres_aggregate.go
  internal/sourceaccess/postgres_integration_test.go
  internal/assurance/source_execution.go
  internal/assurance/postgres_source_executor.go
  internal/assurance/postgres_source_executor_integration_test.go
)
for file in "${required_t0[@]}"; do
  test -s "$file" || { echo "missing T0 contract: $file" >&2; exit 1; }
done

if grep -R -n --include='*.go' 'internal/assurance' internal/sourceaccess; then
  echo "sourceaccess must not depend on assurance" >&2
  exit 1
fi
grep -R -q --include='*.go' 'internal/sourceaccess' internal/assurance || {
  echo "assurance is not consuming the shared sourceaccess boundary" >&2
  exit 1
}

go test ./internal/sourceaccess ./internal/assurance
go test -tags postgres ./internal/sourceaccess ./internal/assurance

if [[ "$tranche" == "t0" ]]; then
  if compgen -G 'internal/sourceaccess/catalog_*.go' >/dev/null || [[ -e migrations/000030_source_access_catalog.up.sql ]]; then
    echo "T0 must remain contract/adapter-only and must not introduce catalog persistence" >&2
    exit 1
  fi
  if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
    psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT CASE WHEN to_regclass('public.source_connections') IS NULL AND to_regclass('public.source_views') IS NULL AND to_regclass('public.source_bindings') IS NULL THEN 'ok' ELSE 'unexpected source catalog persistence in T0' END" | grep -qx ok
  fi
  summary="T0 shared contracts, PostgreSQL adapter extraction and assurance compatibility passed."
else
  required_t1a=(
    internal/sourceaccess/catalog_types.go
    internal/sourceaccess/catalog_validation.go
    internal/sourceaccess/catalog_memory.go
    internal/sourceaccess/catalog_postgres_connection.go
    internal/sourceaccess/catalog_postgres_view.go
    internal/sourceaccess/catalog_postgres_binding.go
    internal/sourceaccess/catalog_postgres_integration_test.go
    internal/evidence/source_catalog_postgres_integration_test.go
    migrations/000030_source_access_catalog.up.sql
    migrations/000030_source_access_catalog.down.sql
  )
  for file in "${required_t1a[@]}"; do
    test -s "$file" || { echo "missing T1a contract: $file" >&2; exit 1; }
  done
  grep -q 'Endpoint.*json:"-"' internal/evidence/model.go || {
    echo "legacy endpoint compatibility input must remain hidden from JSON" >&2
    exit 1
  }
  if grep -R -nE --include='*.go' '\bes\.endpoint\b' internal; then
    echo "runtime code still reads the retired evidence_sources.endpoint column" >&2
    exit 1
  fi
  go test ./internal/evidence
  go test -tags postgres ./internal/evidence
  if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
    psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE
  catalog_tables integer;
  endpoint_columns integer;
  catalog_triggers integer;
BEGIN
  SELECT count(*) INTO catalog_tables
    FROM information_schema.tables
   WHERE table_schema='public'
     AND table_name IN ('source_connections','source_views','source_bindings');
  IF catalog_tables <> 3 THEN
    RAISE EXCEPTION 'expected three durable source catalog tables, found %', catalog_tables;
  END IF;
  SELECT count(*) INTO endpoint_columns
    FROM information_schema.columns
   WHERE table_schema='public' AND table_name='evidence_sources' AND column_name='endpoint';
  IF endpoint_columns <> 0 THEN
    RAISE EXCEPTION 'legacy evidence_sources.endpoint is still present';
  END IF;
  SELECT count(*) INTO catalog_triggers
    FROM pg_trigger
   WHERE NOT tgisinternal
     AND tgname IN ('source_connection_revision_guard_trigger','source_view_revision_guard_trigger','source_binding_revision_guard_trigger');
  IF catalog_triggers <> 3 THEN
    RAISE EXCEPTION 'expected three source catalog revision guards, found %', catalog_triggers;
  END IF;
END
$$;
SQL
  fi

  if [[ "$tranche" == "t1a" ]]; then
    summary="T0 contracts plus T1a durable Connection/View/Binding catalog, migration and legacy endpoint retirement passed."
  else
    required_t1b=(
      internal/sourceaccess/catalog_service.go
      internal/sourceaccess/catalog_service_test.go
      internal/sourceaccess/catalog_history_postgres_integration_test.go
      internal/httpapi/source_catalog_handlers.go
      internal/httpapi/source_catalog_handlers_test.go
      api/runtime.openapi.json
      cmd/api/services.go
      cmd/api/services_memory.go
      cmd/api/services_postgres.go
    )
    for file in "${required_t1b[@]}"; do
      test -s "$file" || { echo "missing T1b contract: $file" >&2; exit 1; }
    done

    grep -q 'ListConnectionRevisions' internal/sourceaccess/catalog_service.go || {
      echo "T1b configuration reads do not retain Connection draft history" >&2
      exit 1
    }
    grep -q 'ListViewRevisions' internal/sourceaccess/catalog_service.go || {
      echo "T1b configuration reads do not retain View draft history" >&2
      exit 1
    }
    grep -q 'ListBindingRevisions' internal/sourceaccess/catalog_service.go || {
      echo "T1b configuration reads do not retain Binding draft history" >&2
      exit 1
    }
    grep -q '!revisionExecutable(bindingRevision.Status)' internal/sourceaccess/catalog_service.go || {
      echo "T1b preview does not reject retired or rejected Binding revisions" >&2
      exit 1
    }
    grep -q 'InspectViewDraft' internal/sourceaccess/catalog_service.go || {
      echo "T1b inspection does not create an immutable schema-bearing View revision" >&2
      exit 1
    }

    python3 - <<'PY'
import json
from pathlib import Path

expected = {
    ("get", "/api/v1/config/sources/{source_id}/connections"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("post", "/api/v1/config/sources/{source_id}/connections"): ("AUTHENTICATED_WRITE", "CONFIG_WRITE"),
    ("get", "/api/v1/config/source-connections/{connection_id}"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("get", "/api/v1/config/source-connections/{connection_id}/views"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("post", "/api/v1/config/source-connections/{connection_id}/views"): ("AUTHENTICATED_WRITE", "CONFIG_WRITE"),
    ("get", "/api/v1/config/source-connections/{connection_id}/where-used"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("get", "/api/v1/config/source-views/{view_id}"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("post", "/api/v1/config/source-views/{view_id}/inspect"): ("AUTHENTICATED_WRITE", "CONFIG_WRITE"),
    ("get", "/api/v1/config/source-views/{view_id}/bindings"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("post", "/api/v1/config/source-views/{view_id}/bindings"): ("AUTHENTICATED_WRITE", "CONFIG_WRITE"),
    ("get", "/api/v1/config/source-views/{view_id}/where-used"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("get", "/api/v1/config/source-bindings/{binding_id}"): ("AUTHENTICATED_READ", "CONFIG_READ"),
    ("post", "/api/v1/config/source-bindings/{binding_id}/preview"): ("AUTHENTICATED_OPERATION", "CONFIG_READ"),
    ("get", "/api/v1/config/source-bindings/{binding_id}/where-used"): ("AUTHENTICATED_READ", "CONFIG_READ"),
}
document = json.loads(Path("api/runtime.openapi.json").read_text())
for (method, path), (route_class, permission) in expected.items():
    operation = document.get("paths", {}).get(path, {}).get(method)
    if not operation:
        raise SystemExit(f"runtime OpenAPI is missing {method.upper()} {path}")
    if operation.get("x-clearsight-route-class") != route_class:
        raise SystemExit(f"wrong route class for {method.upper()} {path}")
    if operation.get("x-clearsight-permission") != permission:
        raise SystemExit(f"wrong permission for {method.upper()} {path}")
PY

    go test ./internal/sourceaccess ./internal/httpapi ./cmd/api
    go test -tags postgres ./internal/sourceaccess ./internal/httpapi ./cmd/api
    if [[ "$tranche" == "t1b" ]]; then
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
	  if [[ "$tranche" == "t2a" || "$tranche" == "t2b" || "$tranche" == "t2c" ]]; then
	    required_t2a=(
	      internal/sourceaccess/rest_json_adapter.go
	      internal/sourceaccess/rest_json_definition.go
	      internal/sourceaccess/rest_json_read.go
	      internal/sourceaccess/rest_json_test.go
	      internal/sourceaccess/schema_guard.go
	    )
	    for file in "${required_t2a[@]}"; do
	      test -s "$file" || { echo "missing T2a contract: $file" >&2; exit 1; }
	    done
	    grep -q 'AdapterRESTJSON' internal/sourceaccess/contracts_types.go || { echo "REST/JSON adapter kind is not in the shared contract" >&2; exit 1; }
	    grep -q 'AdapterRESTJSON: NewRESTJSONAdapter' internal/sourceaccess/catalog_service.go || { echo "REST/JSON adapter is not registered in the catalog" >&2; exit 1; }
	    grep -q 'parsed.Scheme != "https"' internal/sourceaccess/rest_json_definition.go || { echo "REST/JSON adapter does not enforce HTTPS origins" >&2; exit 1; }
	    grep -q 'http.ErrUseLastResponse' internal/sourceaccess/rest_json_adapter.go || { echo "REST/JSON redirects are not explicitly disabled" >&2; exit 1; }
	    if grep -R -nE --include='rest_json*.go' '(POST|PUT|PATCH|DELETE).*http.Method' internal/sourceaccess; then
	      echo "T2a REST/JSON adapter introduced a write method" >&2
	      exit 1
	    fi
	    if [[ "$tranche" == "t2a" ]] && compgen -G 'migrations/000032*' >/dev/null; then
	      echo "T2a REST/JSON adapter must not introduce durable tables or migration 32" >&2
	      exit 1
	    fi
	    go test ./internal/sourceaccess
	    go test -tags postgres ./internal/sourceaccess ./internal/assurance
	    summary="T2a REST/JSON source access passed: fixed HTTPS origin, strict templates, bounded inspect/page/lookup, cursor/ETag positions, secret isolation and schema-drift blocking."
	    if [[ "$tranche" == "t2b" || "$tranche" == "t2c" ]]; then
	      required_t2b=(
	        internal/documentimport/tabular.go
	        internal/documentimport/sourceaccess_adapter.go
	        internal/documentimport/sourceaccess_catalog_test.go
	        internal/documentimport/tabular_postgres_integration_test.go
	        internal/sourceaccess/tabular_artifact_definition.go
	        migrations/000032_document_import_tabular_metadata.up.sql
	        migrations/000032_document_import_tabular_metadata.down.sql
	      )
	      for file in "${required_t2b[@]}"; do
	        test -s "$file" || { echo "missing T2b contract: $file" >&2; exit 1; }
	      done
	      grep -q 'AdapterTabularArtifact' internal/sourceaccess/contracts_types.go || { echo "tabular artifact adapter kind is missing" >&2; exit 1; }
	      grep -q 'tabular_metadata' migrations/000032_document_import_tabular_metadata.up.sql || { echo "tabular parser metadata is not durable" >&2; exit 1; }
	      if grep -qiE 'CREATE[[:space:]]+TABLE' migrations/000032_document_import_tabular_metadata.up.sql; then
	        echo "T2b introduced a source-row or adapter-specific table" >&2
	        exit 1
	      fi
	      grep -q 'SourceAccessAdapter' cmd/api/services_memory.go || { echo "memory runtime does not register the tabular adapter" >&2; exit 1; }
	      grep -q 'SourceAccessAdapter' cmd/api/services_postgres.go || { echo "PostgreSQL runtime does not register the tabular adapter" >&2; exit 1; }
	      if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
	        psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='document_imports' AND column_name='tabular_metadata'" | grep -qx 1
	      fi
	      go test ./internal/documentimport ./internal/sourceaccess ./cmd/api
	      go test -tags postgres ./internal/documentimport ./internal/sourceaccess ./internal/assurance ./cmd/api
	      summary="T2b governed tabular capture passed: CSV/JSON/NDJSON/XLSX originals, bounded parser receipts, row diagnostics/schema fingerprints and reusable artifact-backed inspect/page/lookup without a source-row table."
	    fi
	    if [[ "$tranche" == "t2c" ]]; then
	      required_t2c=(
	        internal/sourceevent/adapter.go
	        internal/sourceevent/adapter_test.go
	        internal/sourceevent/projector.go
	        internal/sourceevent/projector_test.go
	        internal/sourceaccess/webhook_event_definition.go
	        internal/sourceaccess/catalog_changes.go
	        internal/httpapi/source_event_handlers.go
	        internal/httpapi/source_event_handlers_test.go
	        internal/runtime/inbox_outbox_postgres_integration_test.go
	      )
	      for file in "${required_t2c[@]}"; do
	        test -s "$file" || { echo "missing T2c contract: $file" >&2; exit 1; }
	      done
	      grep -q 'AdapterWebhookEvent' internal/sourceaccess/contracts_types.go || { echo "webhook event adapter kind is missing" >&2; exit 1; }
	      grep -q 'RecordInboxWithOutbox' internal/runtime/repository.go || { echo "atomic inbound receipt/outbox primitive is missing" >&2; exit 1; }
	      grep -q 'actor.Kind != "SERVICE"' internal/httpapi/source_event_handlers.go || { echo "source event ingress is not service-identity-only" >&2; exit 1; }
	      grep -q '/api/v1/source-bindings/{id}/events' api/runtime.openapi.json || { echo "source event route is missing from runtime contract" >&2; exit 1; }
	      grep -q 'NewCheckpointProjector' cmd/worker/services_postgres.go || { echo "source event checkpoint recovery is not wired into outbox delivery" >&2; exit 1; }
	      if grep -R -nE --include='*.go' --exclude='*_test.go' 'internal/(runtime|sourceevent)' internal/sourceaccess; then
	        echo "sourceaccess production code depends on runtime/sourceevent" >&2
	        exit 1
	      fi
	      if compgen -G 'migrations/000033*' >/dev/null; then
	        echo "T2c webhook/event capture must not introduce an adapter/event table" >&2
	        exit 1
	      fi
	      go test ./internal/sourceevent ./internal/sourceaccess ./internal/runtime ./internal/httpapi ./cmd/api ./cmd/worker
	      go test -tags postgres ./internal/sourceevent ./internal/sourceaccess ./internal/runtime ./internal/httpapi ./cmd/api ./cmd/worker
	      summary="T2c webhook/event capture passed: service-only bounded CHANGES ingress, atomic provider-event dedupe and outbox durability, strict event/watermark positions, and retrying inbox-backed Binding checkpoint convergence without a webhook table or second event bus."
	    fi
	  fi
    fi
  fi
fi

printf '%s\n' "$summary"
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "## Source-access ${tranche^^} acceptance"
    echo
    echo "- ✅ $summary"
    echo "- ✅ Package dependency direction is sourceaccess → adapters and assurance → sourceaccess."
    echo "- ✅ Focused default and PostgreSQL-tagged tests passed."
  } >> "$GITHUB_STEP_SUMMARY"
fi
