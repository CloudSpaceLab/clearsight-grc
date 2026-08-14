#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path

# HTTP dependency boundary.
path = Path("internal/httpapi/server.go")
text = path.read_text()
old = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
new = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
if old not in text:
    raise SystemExit("server import insertion point changed")
text = text.replace(old, new, 1)
old = '\tEvidence         *evidence.Service\n\tDocumentImports  *documentimport.Service\n'
new = '\tEvidence         *evidence.Service\n\tSourceCatalog    *sourceaccess.CatalogService\n\tDocumentImports  *documentimport.Service\n'
if old not in text:
    raise SystemExit("server dependency insertion point changed")
path.write_text(text.replace(old, new, 1))

# Permissioned source catalog routes.
path = Path("internal/httpapi/route_registry.go")
text = path.read_text()
needle = '\t\tread("/api/v1/evidence/sources", a.listEvidenceSources),\n'
block = '''\t\twithPermission(read("/api/v1/config/sources/{source_id}/connections", a.listSourceConnections), identity.PermissionConfigRead),
\t\twithPermission(write(http.MethodPost, "/api/v1/config/sources/{source_id}/connections", a.createSourceConnectionDraft, nil), identity.PermissionConfigWrite),
\t\twithPermission(read("/api/v1/config/source-connections/{connection_id}", a.getSourceConnection), identity.PermissionConfigRead),
\t\twithPermission(read("/api/v1/config/source-connections/{connection_id}/views", a.listSourceViews), identity.PermissionConfigRead),
\t\twithPermission(write(http.MethodPost, "/api/v1/config/source-connections/{connection_id}/views", a.createSourceViewDraft, nil), identity.PermissionConfigWrite),
\t\twithPermission(read("/api/v1/config/source-connections/{connection_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageConnection, "connection_id")), identity.PermissionConfigRead),
\t\twithPermission(read("/api/v1/config/source-views/{view_id}", a.getSourceView), identity.PermissionConfigRead),
\t\twithPermission(operation("/api/v1/config/source-views/{view_id}/inspect", a.inspectSourceView, nil), identity.PermissionConfigRead),
\t\twithPermission(read("/api/v1/config/source-views/{view_id}/bindings", a.listSourceBindings), identity.PermissionConfigRead),
\t\twithPermission(write(http.MethodPost, "/api/v1/config/source-views/{view_id}/bindings", a.createSourceBindingDraft, nil), identity.PermissionConfigWrite),
\t\twithPermission(read("/api/v1/config/source-views/{view_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageView, "view_id")), identity.PermissionConfigRead),
\t\twithPermission(read("/api/v1/config/source-bindings/{binding_id}", a.getSourceBinding), identity.PermissionConfigRead),
\t\twithPermission(operation("/api/v1/config/source-bindings/{binding_id}/preview", a.previewSourceBinding, nil), identity.PermissionConfigRead),
\t\twithPermission(read("/api/v1/config/source-bindings/{binding_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageBinding, "binding_id")), identity.PermissionConfigRead),

\t\tread("/api/v1/evidence/sources", a.listEvidenceSources),
'''
if needle not in text:
    raise SystemExit("route insertion point changed")
text = text.replace(needle, block, 1)
old = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"\n)'
new = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n)'
if old not in text:
    raise SystemExit("route registry import insertion point changed")
path.write_text(text.replace(old, new, 1))

# Service set.
path = Path("cmd/api/services.go")
text = path.read_text()
old = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"'
new = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"'
if old not in text:
    raise SystemExit("service set import insertion point changed")
text = text.replace(old, new, 1)
old = '\tEvidence        *evidence.Service\n\tDocumentImports *documentimport.Service\n'
new = '\tEvidence        *evidence.Service\n\tSourceCatalog   *sourceaccess.CatalogService\n\tDocumentImports *documentimport.Service\n'
if old not in text:
    raise SystemExit("service set field insertion point changed")
path.write_text(text.replace(old, new, 1))

# Memory wiring.
path = Path("cmd/api/services_memory.go")
text = path.read_text()
old = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
new = '"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
if old not in text:
    raise SystemExit("memory import insertion point changed")
text = text.replace(old, new, 1)
needle = '\tevidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)\n'
block = '''\tevidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
\tsourceScopes := []sourceaccess.SourceScope{}
\tif cfg.DemoMode {
\t\tfor _, source := range evidence.DemoSources() {
\t\t\tsourceScopes = append(sourceScopes, sourceaccess.SourceScope{TenantID: source.TenantID, SourceID: source.ID})
\t\t}
\t}
\tsourceCatalog := sourceaccess.NewCatalogService(sourceaccess.NewMemoryCatalogRepository(sourceScopes), sourceaccess.EnvironmentSecretResolver{}, sourceaccess.DefaultCatalogAdapters())
'''
if needle not in text:
    raise SystemExit("memory source catalog insertion point changed")
text = text.replace(needle, block, 1)
old = '\t\tEvidence: evidenceService, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService,\n'
new = '\t\tEvidence: evidenceService, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService,\n'
if old not in text:
    raise SystemExit("memory return insertion point changed")
path.write_text(text.replace(old, new, 1))

# PostgreSQL wiring.
path = Path("cmd/api/services_postgres.go")
text = path.read_text()
old = '"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
new = '"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"'
if old not in text:
    raise SystemExit("postgres import insertion point changed")
text = text.replace(old, new, 1)
needle = '\tevidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)\n'
block = '''\tevidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
\tsourceCatalog := sourceaccess.NewCatalogService(sourceaccess.NewPostgresCatalogRepository(pool), sourceaccess.EnvironmentSecretResolver{}, sourceaccess.DefaultCatalogAdapters())
'''
if needle not in text:
    raise SystemExit("postgres source catalog insertion point changed")
text = text.replace(needle, block, 1)
old = '\t\tEvidence: evidenceService, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService,\n'
new = '\t\tEvidence: evidenceService, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService,\n'
if old not in text:
    raise SystemExit("postgres return insertion point changed")
path.write_text(text.replace(old, new, 1))

# API dependency injection.
path = Path("cmd/api/main.go")
text = path.read_text()
old = '\t\tEvidence: services.Evidence, DocumentImports: services.DocumentImports, Coverage: services.Coverage,\n'
new = '\t\tEvidence: services.Evidence, SourceCatalog: services.SourceCatalog, DocumentImports: services.DocumentImports, Coverage: services.Coverage,\n'
if old not in text:
    raise SystemExit("main dependency insertion point changed")
path.write_text(text.replace(old, new, 1))
PY

gofmt -w \
  internal/sourceaccess/catalog_service.go \
  internal/sourceaccess/catalog_service_test.go \
  internal/httpapi/source_catalog_handlers.go \
  internal/httpapi/source_catalog_handlers_test.go \
  internal/httpapi/server.go \
  internal/httpapi/route_registry.go \
  cmd/api/services.go \
  cmd/api/services_memory.go \
  cmd/api/services_postgres.go \
  cmd/api/main.go

go test ./internal/sourceaccess ./internal/httpapi ./cmd/api
go test -tags postgres ./internal/sourceaccess ./internal/httpapi ./cmd/api

rm -f .github/workflows/sourceaccess-t1b-bootstrap.yml scripts/apply-sourceaccess-t1b-bootstrap.sh

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add \
  internal/sourceaccess/catalog_service.go \
  internal/sourceaccess/catalog_service_test.go \
  internal/httpapi/source_catalog_handlers.go \
  internal/httpapi/source_catalog_handlers_test.go \
  internal/httpapi/server.go \
  internal/httpapi/route_registry.go \
  cmd/api/services.go \
  cmd/api/services_memory.go \
  cmd/api/services_postgres.go \
  cmd/api/main.go \
  .github/workflows/sourceaccess-t1b-bootstrap.yml \
  scripts/apply-sourceaccess-t1b-bootstrap.sh
git commit -m "feat(sourceaccess): expose governed catalog operations"
git push origin HEAD:codex/issue-61-sourceaccess-t1b
