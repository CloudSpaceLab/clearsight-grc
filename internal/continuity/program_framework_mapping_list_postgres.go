//go:build postgres

package continuity

import (
	"context"

)

// ListFrameworkMappings reads only current pending/approved links and never
// includes another Program, tenant or legal entity in a reviewer's queue.
// Rejected proposals remain available through immutable Program event history.
func (r *PostgresRepository) ListFrameworkMappings(
	ctx context.Context, tenant, programID, adoptionID, afterID string, limit int,
) ([]FrameworkClauseMapping, error) {
	if limit <= 0 || limit > 100 {
		return nil, ErrInvalidState
	}
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text,t.id::text,l.legal_entity_id::text,l.program_id::text,
			l.adoption_id::text,l.edition_id::text,l.clause_id::text,
			c.code,l.requirement_id::text,l.mapping_kind,l.rationale,
			l.mapped_by::text,COALESCE(l.reviewed_by::text,''),l.proposed_at,
			l.reviewed_at,l.effective_from,l.status,l.version
		FROM framework_clause_requirement_links l
		JOIN framework_catalog_clauses c ON c.id=l.clause_id
			AND c.edition_id=l.edition_id
		JOIN tenants t ON t.id=l.tenant_id
		JOIN programs p ON p.id=l.program_id AND p.tenant_id=l.tenant_id
			AND p.legal_entity_id=l.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND l.program_id=$2::uuid AND l.adoption_id=$3::uuid
		  AND l.retired_at IS NULL AND l.status IN ('PENDING','APPROVED')
		  AND ($4::text='' OR l.id>NULLIF($4::text,'')::uuid)
		  AND (NOT $5 OR (
		    (t.id::text=$6 OR t.slug=$6) AND p.legal_entity_id IS NOT NULL
		    AND ($7='*' OR p.legal_entity_id=(
		      SELECT le.id FROM legal_entities le
		      WHERE le.tenant_id=p.tenant_id
		        AND (le.id::text=$7 OR le.code=$7)
		        AND le.valid_from<=clock_timestamp()
		        AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		      ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		ORDER BY l.id LIMIT $8`,
		tenant, programID, adoptionID, afterID, enforce, actorTenant, actorEntity, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]FrameworkClauseMapping, 0, limit)
	for rows.Next() {
		var m FrameworkClauseMapping
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.LegalEntityID, &m.ProgramID,
			&m.AdoptionID, &m.EditionID, &m.ClauseID, &m.ClauseCode,
			&m.RequirementID, &m.MappingKind, &m.Rationale,
			&m.MappedBy, &m.ReviewedBy, &m.ProposedAt, &m.ReviewedAt,
			&m.EffectiveFrom, &m.Status, &m.Version,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
