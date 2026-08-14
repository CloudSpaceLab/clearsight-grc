#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

migration = Path("migrations/000030_source_access_catalog.up.sql")
text = migration.read_text()
replacements = [
    (
        "CREATE FUNCTION source_connection_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nBEGIN\n",
        "CREATE FUNCTION source_connection_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nBEGIN\n    PERFORM pg_advisory_xact_lock(hashtextextended('source_connection:' || NEW.connection_id::text,0));\n",
    ),
    (
        "CREATE FUNCTION source_view_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nDECLARE\n    parent_current boolean;\n    parent_kind text;\nBEGIN\n",
        "CREATE FUNCTION source_view_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nDECLARE\n    parent_current boolean;\n    parent_kind text;\nBEGIN\n    PERFORM pg_advisory_xact_lock(hashtextextended('source_view:' || NEW.view_id::text,0));\n",
    ),
    (
        "     WHERE connection_id=NEW.connection_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.connection_version;\n",
        "     WHERE connection_id=NEW.connection_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.connection_version\n     FOR SHARE;\n",
    ),
    (
        "CREATE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nDECLARE\n    parent_current boolean;\n    parent_stable_keys jsonb;\n    parent_native_schema jsonb;\nBEGIN\n",
        "CREATE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$\nDECLARE\n    parent_current boolean;\n    parent_stable_keys jsonb;\n    parent_native_schema jsonb;\nBEGIN\n    PERFORM pg_advisory_xact_lock(hashtextextended('source_binding:' || NEW.binding_id::text,0));\n",
    ),
    (
        "     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version;\n",
        "     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version\n     FOR SHARE;\n",
    ),
]
for old, new in replacements:
    if old not in text:
        raise SystemExit(f"migration concurrency insertion point changed: {old[:80]!r}")
    text = text.replace(old, new, 1)
migration.write_text(text)

for file_name, table_alias, id_label in [
    ("internal/sourceaccess/catalog_postgres_connection.go", "connection", "ConnectionRevision"),
    ("internal/sourceaccess/catalog_postgres_view.go", "view", "ViewRevision"),
    ("internal/sourceaccess/catalog_postgres_binding.go", "binding", "BindingRevision"),
]:
    path = Path(file_name)
    text = path.read_text()
    old = f'''func (r *PostgresCatalogRepository) {id_label}(ctx context.Context, tenantID, {table_alias}ID string, version int64) ({id_label}, error) {{\n\tif r == nil || r.pool == nil || version < 1 {{\n\t\treturn {id_label}{{}}, ErrCatalogInvalid\n\t}}\n'''
    new = f'''func (r *PostgresCatalogRepository) {id_label}(ctx context.Context, tenantID, {table_alias}ID string, version int64) ({id_label}, error) {{\n\tif r == nil || r.pool == nil {{\n\t\treturn {id_label}{{}}, ErrCatalogStorage\n\t}}\n\tif version < 1 {{\n\t\treturn {id_label}{{}}, ErrCatalogInvalid\n\t}}\n'''
    if old not in text:
        raise SystemExit(f"exact read guard changed in {file_name}")
    path.write_text(text.replace(old, new, 1))

path = Path("internal/sourceaccess/catalog_postgres_view.go")
text = path.read_text()
old = "\t\t   AND sc.version=$4`, tenantID, sourceID, connectionID, version))\n"
new = "\t\t   AND sc.version=$4\n\t\t FOR SHARE`, tenantID, sourceID, connectionID, version))\n"
if old not in text:
    raise SystemExit("connection parent read shape changed")
path.write_text(text.replace(old, new, 1))

path = Path("internal/sourceaccess/catalog_postgres_binding.go")
text = path.read_text()
old = "\t\t   AND sv.version=$4`, tenantID, sourceID, viewID, version))\n"
new = "\t\t   AND sv.version=$4\n\t\t FOR SHARE`, tenantID, sourceID, viewID, version))\n"
if old not in text:
    raise SystemExit("view parent read shape changed")
path.write_text(text.replace(old, new, 1))
PY

gofmt -w internal/sourceaccess/catalog_postgres_connection.go internal/sourceaccess/catalog_postgres_view.go internal/sourceaccess/catalog_postgres_binding.go internal/sourceaccess/catalog_concurrency_postgres_integration_test.go internal/sourceaccess/catalog_postgres_read_guard_test.go

go test -tags postgres ./internal/sourceaccess
if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
  for migration in migrations/*.up.sql; do
    psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration" >/dev/null
  done
  go test -p 1 -tags "postgres postgresintegration" ./internal/sourceaccess
fi

rm -f .github/workflows/sourceaccess-concurrency-closeout.yml scripts/apply-sourceaccess-concurrency-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add migrations/000030_source_access_catalog.up.sql \
  internal/sourceaccess/catalog_postgres_connection.go \
  internal/sourceaccess/catalog_postgres_view.go \
  internal/sourceaccess/catalog_postgres_binding.go \
  internal/sourceaccess/catalog_concurrency_postgres_integration_test.go \
  .github/workflows/sourceaccess-concurrency-closeout.yml \
  scripts/apply-sourceaccess-concurrency-closeout.sh
git commit -m "fix(sourceaccess): serialize catalog invariants"
git push origin HEAD:codex/issue-61-sourceaccess-t1a
