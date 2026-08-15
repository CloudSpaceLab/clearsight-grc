#!/usr/bin/env bash
set -Eeuo pipefail

database_url="${DATABASE_URL:-${TEST_DATABASE_URL:-}}"
: "${database_url:?DATABASE_URL or TEST_DATABASE_URL is required}"
readonly database_url
readonly migration="000033_t2_binding_reuse.up.sql"

query() {
  psql -XAt "$database_url" -v ON_ERROR_STOP=1 -c "$1"
}

ledger_available() {
  [[ "$(query "SELECT to_regclass('public.clearsight_schema_migrations') IS NOT NULL;")" == "t" ]]
}

ledger_count() {
  if ledger_available; then
    query "SELECT count(*) FROM public.clearsight_schema_migrations WHERE filename='${migration}';"
  else
    printf '0\n'
  fi
}

assert_applied() {
  local columns constraints
  columns="$(query "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND (table_name,column_name,data_type) IN (('capture_requests','source_bindings','jsonb'),('capture_submissions','answer_provenance','jsonb'),('workflow_tasks','source_bindings','jsonb')); ")"
  constraints="$(query "SELECT count(*) FROM pg_constraint WHERE conname IN ('capture_requests_source_bindings_array','capture_submissions_answer_provenance_object','workflow_tasks_source_bindings_array');")"
  [[ "$columns" == "3" ]] || { echo "T2 binding columns are incomplete: $columns/3" >&2; exit 1; }
  [[ "$constraints" == "3" ]] || { echo "T2 binding constraints are incomplete: $constraints/3" >&2; exit 1; }
  if ledger_available; then
    local ledger
    ledger="$(ledger_count)"
    [[ "$ledger" == "1" ]] || { echo "T2 migration ledger entry is incomplete: $ledger/1" >&2; exit 1; }
  fi
}

assert_rolled_back() {
  local columns constraints
  columns="$(query "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND ((table_name='capture_requests' AND column_name='source_bindings') OR (table_name='capture_submissions' AND column_name='answer_provenance') OR (table_name='workflow_tasks' AND column_name='source_bindings')); ")"
  constraints="$(query "SELECT count(*) FROM pg_constraint WHERE conname IN ('capture_requests_source_bindings_array','capture_submissions_answer_provenance_object','workflow_tasks_source_bindings_array');")"
  [[ "$columns" == "0" ]] || { echo "T2 binding columns survived rollback: $columns" >&2; exit 1; }
  [[ "$constraints" == "0" ]] || { echo "T2 binding constraints survived rollback: $constraints" >&2; exit 1; }
  if ledger_available; then
    local ledger
    ledger="$(ledger_count)"
    [[ "$ledger" == "0" ]] || { echo "T2 migration ledger entry survived rollback: $ledger" >&2; exit 1; }
  fi
}

assert_applied
had_ledger=false
if ledger_available; then
  [[ "$(ledger_count)" == "1" ]] || { echo "T2 migration ledger is inconsistent before rollback" >&2; exit 1; }
  had_ledger=true
fi

psql -X "$database_url" -v ON_ERROR_STOP=1 -f migrations/000033_t2_binding_reuse.down.sql >/dev/null
if [[ "$had_ledger" == "true" ]]; then
  query "DELETE FROM public.clearsight_schema_migrations WHERE filename='${migration}';" >/dev/null
fi
assert_rolled_back

if [[ "$had_ledger" == "true" ]]; then
  DATABASE_URL="$database_url" bash deploy/scripts/migrate.sh migrations
else
  psql -X "$database_url" -v ON_ERROR_STOP=1 -f migrations/000033_t2_binding_reuse.up.sql >/dev/null
fi
assert_applied

echo "T2 binding reuse migration rollback/reapply verification passed."
