#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

path = Path("internal/sourceaccess/catalog_types.go")
text = path.read_text()
text = text.replace(
    "\tListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error)\n",
    "\tListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error)\n\tListConnectionRevisions(context.Context, string, string, int) ([]ConnectionRevision, error)\n",
    1,
)
text = text.replace(
    "\tListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error)\n",
    "\tListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error)\n\tListViewRevisions(context.Context, string, string, int) ([]ViewRevision, error)\n",
    1,
)
text = text.replace(
    "\tListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error)\n",
    "\tListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error)\n\tListBindingRevisions(context.Context, string, string, int) ([]BindingRevision, error)\n",
    1,
)
path.write_text(text)

path = Path("internal/sourceaccess/catalog_memory.go")
text = path.read_text()
needle = "func (r *MemoryCatalogRepository) CreateViewRevision(_ context.Context, value ViewRevision) (ViewRevision, error) {\n"
method = '''func (r *MemoryCatalogRepository) ListConnectionRevisions(_ context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
\tr.mu.RLock()
\tdefer r.mu.RUnlock()
\tvalues := make([]ConnectionRevision, 0)
\tfor _, value := range r.connections {
\t\tif value.TenantID == tenantID && value.SourceID == sourceID {
\t\t\tvalues = append(values, cloneConnectionRevision(value))
\t\t}
\t}
\tsort.Slice(values, func(i, j int) bool {
\t\tif values[i].Code != values[j].Code { return values[i].Code < values[j].Code }
\t\tif values[i].ConnectionID != values[j].ConnectionID { return values[i].ConnectionID < values[j].ConnectionID }
\t\treturn values[i].Version > values[j].Version
\t})
\treturn limitConnections(values, catalogListLimit(limit)), nil
}

'''
if needle not in text: raise SystemExit("memory connection insertion point changed")
text = text.replace(needle, method + needle, 1)
needle = "func (r *MemoryCatalogRepository) CreateBindingRevision(_ context.Context, value BindingRevision) (BindingRevision, error) {\n"
method = '''func (r *MemoryCatalogRepository) ListViewRevisions(_ context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
\tr.mu.RLock()
\tdefer r.mu.RUnlock()
\tvalues := make([]ViewRevision, 0)
\tfor _, value := range r.views {
\t\tif value.TenantID == tenantID && value.ConnectionID == connectionID {
\t\t\tvalues = append(values, cloneViewRevision(value))
\t\t}
\t}
\tsort.Slice(values, func(i, j int) bool {
\t\tif values[i].Code != values[j].Code { return values[i].Code < values[j].Code }
\t\tif values[i].ViewID != values[j].ViewID { return values[i].ViewID < values[j].ViewID }
\t\treturn values[i].Version > values[j].Version
\t})
\treturn limitViews(values, catalogListLimit(limit)), nil
}

'''
if needle not in text: raise SystemExit("memory view insertion point changed")
text = text.replace(needle, method + needle, 1)
needle = "func (r *MemoryCatalogRepository) connectionScopeConflict(candidate ConnectionRevision) bool {\n"
method = '''func (r *MemoryCatalogRepository) ListBindingRevisions(_ context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
\tr.mu.RLock()
\tdefer r.mu.RUnlock()
\tvalues := make([]BindingRevision, 0)
\tfor _, value := range r.bindings {
\t\tif value.TenantID == tenantID && value.ViewID == viewID {
\t\t\tvalues = append(values, cloneBindingRevision(value))
\t\t}
\t}
\tsort.Slice(values, func(i, j int) bool {
\t\tif values[i].Code != values[j].Code { return values[i].Code < values[j].Code }
\t\tif values[i].BindingID != values[j].BindingID { return values[i].BindingID < values[j].BindingID }
\t\treturn values[i].Version > values[j].Version
\t})
\treturn limitBindings(values, catalogListLimit(limit)), nil
}

'''
if needle not in text: raise SystemExit("memory binding insertion point changed")
path.write_text(text.replace(needle, method + needle, 1))

path = Path("internal/sourceaccess/catalog_postgres_connection.go")
text = path.read_text()
needle = "func requireCatalogSource(ctx context.Context, tx pgx.Tx, tenantID, sourceID string) error {\n"
method = '''func (r *PostgresCatalogRepository) ListConnectionRevisions(ctx context.Context, tenantID, sourceID string, limit int) ([]ConnectionRevision, error) {
\tif r == nil || r.pool == nil { return nil, ErrCatalogStorage }
\trows, err := r.pool.Query(ctx, `
\t\tSELECT `+connectionRevisionColumns+`
\t\t  FROM source_connections sc
\t\t  JOIN tenants t ON t.id=sc.tenant_id
\t\t WHERE (t.id::text=$1 OR t.slug=$1)
\t\t   AND sc.source_id=$2::uuid
\t\t ORDER BY sc.code,sc.connection_id,sc.version DESC
\t\t LIMIT $3`, tenantID, sourceID, catalogListLimit(limit))
\tif err != nil { return nil, catalogReadError(err) }
\tdefer rows.Close()
\tvalues := make([]ConnectionRevision, 0)
\tfor rows.Next() {
\t\tvalue, scanErr := scanConnectionRevision(rows)
\t\tif scanErr != nil { return nil, scanErr }
\t\tvalues = append(values, value)
\t}
\tif err := rows.Err(); err != nil { return nil, catalogReadError(err) }
\treturn values, nil
}

'''
if needle not in text: raise SystemExit("postgres connection insertion point changed")
path.write_text(text.replace(needle, method + needle, 1))

path = Path("internal/sourceaccess/catalog_postgres_view.go")
text = path.read_text()
needle = "func connectionRevisionForChild(ctx context.Context, tx pgx.Tx, tenantID, sourceID, connectionID string, version int64) (ConnectionRevision, error) {\n"
method = '''func (r *PostgresCatalogRepository) ListViewRevisions(ctx context.Context, tenantID, connectionID string, limit int) ([]ViewRevision, error) {
\tif r == nil || r.pool == nil { return nil, ErrCatalogStorage }
\trows, err := r.pool.Query(ctx, `
\t\tSELECT `+viewRevisionColumns+`
\t\t  FROM source_views sv
\t\t  JOIN tenants t ON t.id=sv.tenant_id
\t\t WHERE (t.id::text=$1 OR t.slug=$1)
\t\t   AND sv.connection_id=$2::uuid
\t\t ORDER BY sv.code,sv.view_id,sv.version DESC
\t\t LIMIT $3`, tenantID, connectionID, catalogListLimit(limit))
\tif err != nil { return nil, catalogReadError(err) }
\tdefer rows.Close()
\tvalues := make([]ViewRevision, 0)
\tfor rows.Next() {
\t\tvalue, scanErr := scanViewRevision(rows)
\t\tif scanErr != nil { return nil, scanErr }
\t\tvalues = append(values, value)
\t}
\tif err := rows.Err(); err != nil { return nil, catalogReadError(err) }
\treturn values, nil
}

'''
if needle not in text: raise SystemExit("postgres view insertion point changed")
path.write_text(text.replace(needle, method + needle, 1))

path = Path("internal/sourceaccess/catalog_postgres_binding.go")
text = path.read_text()
needle = "func viewRevisionForChild(ctx context.Context, tx pgx.Tx, tenantID, sourceID, viewID string, version int64) (ViewRevision, error) {\n"
method = '''func (r *PostgresCatalogRepository) ListBindingRevisions(ctx context.Context, tenantID, viewID string, limit int) ([]BindingRevision, error) {
\tif r == nil || r.pool == nil { return nil, ErrCatalogStorage }
\trows, err := r.pool.Query(ctx, `
\t\tSELECT `+bindingRevisionColumns+`
\t\t  FROM source_bindings sb
\t\t  JOIN tenants t ON t.id=sb.tenant_id
\t\t WHERE (t.id::text=$1 OR t.slug=$1)
\t\t   AND sb.view_id=$2::uuid
\t\t ORDER BY sb.code,sb.binding_id,sb.version DESC
\t\t LIMIT $3`, tenantID, viewID, catalogListLimit(limit))
\tif err != nil { return nil, catalogReadError(err) }
\tdefer rows.Close()
\tvalues := make([]BindingRevision, 0)
\tfor rows.Next() {
\t\tvalue, scanErr := scanBindingRevision(rows)
\t\tif scanErr != nil { return nil, scanErr }
\t\tvalues = append(values, value)
\t}
\tif err := rows.Err(); err != nil { return nil, catalogReadError(err) }
\treturn values, nil
}

'''
if needle not in text: raise SystemExit("postgres binding insertion point changed")
path.write_text(text.replace(needle, method + needle, 1))

path = Path("internal/sourceaccess/catalog_service.go")
text = path.read_text()
old = '''\towner := strings.TrimSpace(input.OwnerPrincipalID)
\tif owner == "" {
\t\towner = actor.PrincipalID
\t}
'''
new = '''\towner := strings.TrimSpace(input.OwnerPrincipalID)
\tif owner != "" && owner != actor.PrincipalID {
\t\treturn ConnectionRevision{}, fmt.Errorf("%w: owner assignment requires a governed principal-selection command", ErrCatalogInvalid)
\t}
\towner = actor.PrincipalID
'''
if old not in text: raise SystemExit("owner block changed")
text = text.replace(old, new, 1)
text = text.replace("return s.repoOrError().ListCurrentConnections(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), limit)", "return s.repoOrError().ListConnectionRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), limit)", 1)
text = text.replace("return s.repoOrError().ListCurrentViews(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(connectionID), limit)", "return s.repoOrError().ListViewRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(connectionID), limit)", 1)
text = text.replace("return s.repoOrError().ListCurrentBindings(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(viewID), limit)", "return s.repoOrError().ListBindingRevisions(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(viewID), limit)", 1)
old = '''\tbindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
\tif err != nil {
\t\treturn RecordPage{}, err
\t}
'''
new = '''\tbindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
\tif err != nil {
\t\treturn RecordPage{}, err
\t}
\tif !revisionExecutable(bindingRevision.Status) {
\t\treturn RecordPage{}, ErrCatalogInvalid
\t}
'''
if old not in text: raise SystemExit("preview lifecycle block changed")
text = text.replace(old, new, 1)
text = text.replace("values, err := s.Views(ctx, tenantID, resourceID, limit)", "values, err := s.repoOrError().ListCurrentViews(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(resourceID), limit)", 1)
text = text.replace("values, err := s.Bindings(ctx, tenantID, resourceID, limit)", "values, err := s.repoOrError().ListCurrentBindings(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(resourceID), limit)", 1)
old = '''\tif input.AdapterKind == AdapterPostgres {
\t\tif _, ok := environmentSecretName(input.SecretRef); !ok {
\t\t\treturn fmt.Errorf("%w: PostgreSQL credentials require an env:// secret reference", ErrCatalogInvalid)
\t\t}
'''
new = '''\tif input.AdapterKind == AdapterPostgres {
\t\tsecretRef := strings.TrimSpace(input.SecretRef)
\t\tif secretRef == "" || secretRef != input.SecretRef || len(secretRef) > HardMaxIdentifierBytes || containsControl(secretRef) {
\t\t\treturn fmt.Errorf("%w: PostgreSQL connections require a bounded opaque secret reference", ErrCatalogInvalid)
\t\t}
'''
if old not in text: raise SystemExit("secret reference block changed")
text = text.replace(old, new, 1)
text = text.replace(
    "func (unavailableCatalogRepository) ListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    "func (unavailableCatalogRepository) ListCurrentConnections(context.Context, string, string, int) ([]ConnectionRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\nfunc (unavailableCatalogRepository) ListConnectionRevisions(context.Context, string, string, int) ([]ConnectionRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    1,
)
text = text.replace(
    "func (unavailableCatalogRepository) ListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    "func (unavailableCatalogRepository) ListCurrentViews(context.Context, string, string, int) ([]ViewRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\nfunc (unavailableCatalogRepository) ListViewRevisions(context.Context, string, string, int) ([]ViewRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    1,
)
text = text.replace(
    "func (unavailableCatalogRepository) ListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    "func (unavailableCatalogRepository) ListCurrentBindings(context.Context, string, string, int) ([]BindingRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\nfunc (unavailableCatalogRepository) ListBindingRevisions(context.Context, string, string, int) ([]BindingRevision, error) {\n\treturn nil, ErrCatalogStorage\n}\n",
    1,
)
path.write_text(text)
PY

gofmt -w \
  internal/sourceaccess/catalog_types.go \
  internal/sourceaccess/catalog_memory.go \
  internal/sourceaccess/catalog_postgres_connection.go \
  internal/sourceaccess/catalog_postgres_view.go \
  internal/sourceaccess/catalog_postgres_binding.go \
  internal/sourceaccess/catalog_service.go \
  internal/sourceaccess/catalog_service_test.go \
  internal/sourceaccess/catalog_history_postgres_integration_test.go \
  internal/httpapi/source_catalog_handlers_test.go

go test ./internal/sourceaccess ./internal/httpapi ./cmd/api
go test -tags postgres ./internal/sourceaccess ./internal/httpapi ./cmd/api

if [[ -n "${TEST_DATABASE_URL:-}" ]]; then
  for migration in migrations/*.up.sql; do
    psql -X "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration" >/dev/null
  done
  go test -p 1 -tags "postgres postgresintegration" ./internal/sourceaccess ./internal/httpapi
fi

rm -f .github/workflows/sourceaccess-t1b-history-closeout.yml scripts/apply-sourceaccess-t1b-history-closeout.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add \
  internal/sourceaccess/catalog_types.go \
  internal/sourceaccess/catalog_memory.go \
  internal/sourceaccess/catalog_postgres_connection.go \
  internal/sourceaccess/catalog_postgres_view.go \
  internal/sourceaccess/catalog_postgres_binding.go \
  internal/sourceaccess/catalog_service.go \
  internal/sourceaccess/catalog_service_test.go \
  internal/sourceaccess/catalog_history_postgres_integration_test.go \
  internal/httpapi/source_catalog_handlers_test.go \
  .github/workflows/sourceaccess-t1b-history-closeout.yml \
  scripts/apply-sourceaccess-t1b-history-closeout.sh
git commit -m "fix(sourceaccess): keep draft history visible and bounded"
git push origin HEAD:codex/issue-61-sourceaccess-t1b
