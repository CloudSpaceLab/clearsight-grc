#!/usr/bin/env bash
set -Eeuo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
readonly migration="000033_t2_binding_reuse.up.sql"

query() {
  psql -XAt "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$1"
}

assert_applied() {
  local columns constraints ledger
  columns="$(query "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND (table_name,column_name,data_type) IN (('capture_requests','source_bindings','jsonb'),('capture_submissions','answer_provenance','jsonb'),('workflow_tasks','source_bindings','jsonb')); ")"
  constraints="$(query "SELECT count(*) FROM pg_constraint WHERE conname IN ('capture_requests_source_bindings_array','capture_submissions_answer_provenance_object','workflow_tasks_source_bindings_array');")"
  ledger="$(query "SELECT count(*) FROM public.clearsight_schema_migrations WHERE filename='${migration}';")"
  [[ "$columns" == "3" ]] || { echo "T2 binding columns are incomplete: $columns/3" >&2; exit 1; }
  [[ "$constraints" == "3" ]] || { echo "T2 binding constraints are incomplete: $constraints/3" >&2; exit 1; }
  [[ "$ledger" == "1" ]] || { echo "T2 migration ledger entry is incomplete: $ledger/1" >&2; exit 1; }
}

assert_rolled_back() {
  local columns constraints ledger
  columns="$(query "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND ((table_name='capture_requests' AND column_name='source_bindings') OR (table_name='capture_submissions' AND column_name='answer_provenance') OR (table_name='workflow_tasks' AND column_name='source_bindings')); ")"
  constraints="$(query "SELECT count(*) FROM pg_constraint WHERE conname IN ('capture_requests_source_bindings_array','capture_submissions_answer_provenance_object','workflow_tasks_source_bindings_array');")"
  ledger="$(query "SELECT count(*) FROM public.clearsight_schema_migrations WHERE filename='${migration}';")"
  [[ "$columns" == "0" ]] || { echo "T2 binding columns survived rollback: $columns" >&2; exit 1; }
  [[ "$constraints" == "0" ]] || { echo "T2 binding constraints survived rollback: $constraints" >&2; exit 1; }
  [[ "$ledger" == "0" ]] || { echo "T2 migration ledger entry survived rollback: $ledger" >&2; exit 1; }
}

assert_applied
psql -X "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000033_t2_binding_reuse.down.sql >/dev/null
query "DELETE FROM public.clearsight_schema_migrations WHERE filename='${migration}';" >/dev/null
assert_rolled_back
bash deploy/scripts/migrate.sh migrations
assert_applied

echo "T2 binding reuse migration rollback/reapply verification passed."
