from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

# Register operational projection routes.
p = Path("internal/httpapi/server.go")
s = p.read_text()
anchor = '\tmux.HandleFunc("GET /api/v1/evidence/sources", api.listEvidenceSources)'
if 'GET /api/v1/operations/projections' not in s:
    s = replace_once(
        s,
        anchor,
        '\tmux.HandleFunc("GET /api/v1/operations/projections", api.projectionHealth)\n\tmux.HandleFunc("POST /api/v1/operations/projections/reconcile", api.command("projection.reconcile", api.reconcileProgramState))\n\tmux.HandleFunc("POST /api/v1/operations/projections/rebuild", api.command("projection.rebuild", api.rebuildProgramState))\n\n' + anchor,
        "projection routes",
    )
p.write_text(s)

# Ensure a queue operation originating from a linked Matter records the actual
# current Program command version rather than zero.
p = Path("internal/continuity/projection_postgres.go")
s = p.read_text()
anchor = '''func queueProgramStateTx(ctx context.Context, tx pgx.Tx, tenant, programID string, sourceVersion int64, reason, triggerID, requestedBy string, now time.Time) (ProjectionJob, error) {
\tjobID, err := id.NewUUIDv7()'''
replacement = '''func queueProgramStateTx(ctx context.Context, tx pgx.Tx, tenant, programID string, sourceVersion int64, reason, triggerID, requestedBy string, now time.Time) (ProjectionJob, error) {
\tif sourceVersion <= 0 {
\t\tif err := tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid`, tenant, programID).Scan(&sourceVersion); err != nil {
\t\t\tif errors.Is(err, pgx.ErrNoRows) {
\t\t\t\treturn ProjectionJob{}, ErrNotFound
\t\t\t}
\t\t\treturn ProjectionJob{}, err
\t\t}
\t}
\tjobID, err := id.NewUUIDv7()'''
s = replace_once(s, anchor, replacement, "projection source version")
p.write_text(s)

# Extend the formal HTTP contract.
p = Path("api/openapi.yaml")
s = p.read_text().replace("  version: 0.6.0", "  version: 0.7.0")
anchor = "  /api/v1/program-summaries:\n"
if "/api/v1/operations/projections:" not in s:
    block = '''  /api/v1/operations/projections:
    get:
      summary: Read Program status update health
      parameters:
        - { $ref: '#/components/parameters/Tenant' }
      responses:
        '200': { description: Current pending, failed and completed update status }
  /api/v1/operations/projections/reconcile:
    post:
      summary: Check Program status records and queue missing updates
      responses:
        '202': { description: Reconciliation accepted }
        '401': { description: Verified identity required }
        '403': { description: Current reviewer authority required }
  /api/v1/operations/projections/rebuild:
    post:
      summary: Queue a governed Program status recalculation
      responses:
        '202': { description: Recalculation queued }
        '401': { description: Verified identity required }
        '403': { description: Current authorizer authority required }

'''
    s = replace_once(s, anchor, block + anchor, "OpenAPI projection paths")
p.write_text(s)

# Mark repository implementation status without claiming external IdP readiness.
p = Path("docs/implementation-plan.md")
s = p.read_text()
s = s.replace("- [ ] authenticated actor binding and automatic authority checks on every material command;", "- [x] signed request-actor binding and automatic authority checks on material Program/Matter commands;\n- [ ] direct OIDC/SAML identity-provider integration and organization synchronization;")
s = s.replace("- [ ] projection-first high-cardinality list/read model and performance baselines;", "- [x] projection-first high-cardinality Program/Matter list models and bounded search;\n- [x] separately versioned Program-status maintenance queue, health, reconcile and rebuild operations;\n- [ ] representative production-scale p95/p99 evidence and retained query plans;")
if "separately versioned Program-status maintenance" not in s:
    s += "\n- [x] Signed request actor binding and authority checks for material Program/Matter commands.\n- [x] Separately versioned Program-status maintenance with health, reconcile and rebuild operations.\n"
p.write_text(s)
