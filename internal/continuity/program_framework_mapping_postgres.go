//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetFrameworkMapping resolves only one scoped mapping. Bulk mapping history
// never inflates the Program aggregate or leaks another entity's obligations.
func (r *PostgresRepository) GetFrameworkMapping(ctx context.Context, tenant, programID, mappingID string) (FrameworkClauseMapping, error) {
	var m FrameworkClauseMapping
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err := r.pool.QueryRow(ctx, `
		SELECT l.id::text,t.id::text,l.legal_entity_id::text,l.program_id::text,
			l.adoption_id::text,l.edition_id::text,l.clause_id::text,
			l.requirement_id::text,l.mapping_kind,l.rationale,l.mapped_by::text,
			COALESCE(l.reviewed_by::text,''),l.proposed_at,l.reviewed_at,
			l.effective_from,l.status,l.version
		FROM framework_clause_requirement_links l
		JOIN tenants t ON t.id=l.tenant_id
		JOIN programs p ON p.id=l.program_id AND p.tenant_id=l.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND l.program_id=$2::uuid
		  AND l.id=$3::uuid
		  AND (NOT $4 OR (
		    (t.id::text=$5 OR t.slug=$5)
		    AND p.legal_entity_id IS NOT NULL
		    AND ($6='*' OR p.legal_entity_id=(
		      SELECT le.id FROM legal_entities le
		      WHERE le.tenant_id=p.tenant_id
		        AND (le.id::text=$6 OR le.code=$6)
		        AND le.valid_from<=clock_timestamp()
		        AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		      ORDER BY le.valid_from DESC,le.id LIMIT 1))))`,
		tenant, programID, mappingID, enforce, actorTenant, actorEntity,
	).Scan(&m.ID, &m.TenantID, &m.LegalEntityID, &m.ProgramID,
		&m.AdoptionID, &m.EditionID, &m.ClauseID, &m.RequirementID,
		&m.MappingKind, &m.Rationale, &m.MappedBy, &m.ReviewedBy,
		&m.ProposedAt, &m.ReviewedAt, &m.EffectiveFrom, &m.Status, &m.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return FrameworkClauseMapping{}, ErrNotFound
	}
	return m, err
}
