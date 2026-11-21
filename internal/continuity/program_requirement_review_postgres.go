//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetRequirementReview returns a single scoped review; it must not be used
// as a substitute for current command authority or version checks.
func (r *PostgresRepository) GetRequirementReview(ctx context.Context, tenant, programID, reviewID string) (RequirementReview, error) {
	var value RequirementReview
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err := r.pool.QueryRow(ctx, `
 SELECT rr.id::text,t.id::text,rr.program_id::text,rr.requirement_id::text,
        rr.proposed_by::text,rr.proposal_rationale,rr.proposed_at,
        COALESCE(rr.reviewed_by::text,''),COALESCE(rr.review_rationale,''),
        rr.reviewed_at,rr.status,rr.version
 FROM program_requirement_reviews rr
 JOIN tenants t ON t.id=rr.tenant_id
 JOIN programs p ON p.id=rr.program_id AND p.tenant_id=rr.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND rr.program_id=$2::uuid AND rr.id=$3::uuid
   AND (NOT $4 OR ((t.id::text=$5 OR t.slug=$5) AND p.legal_entity_id IS NOT NULL
     AND ($6='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le
       WHERE le.tenant_id=p.tenant_id AND (le.id::text=$6 OR le.code=$6)
       AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
       ORDER BY le.valid_from DESC,le.id LIMIT 1))))`, tenant, programID, reviewID, enforce, actorTenant, actorEntity,
	).Scan(&value.ID, &value.TenantID, &value.ProgramID, &value.RequirementID, &value.ProposedBy,
		&value.ProposalRationale, &value.ProposedAt, &value.ReviewedBy, &value.ReviewRationale,
		&value.ReviewedAt, &value.Status, &value.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return RequirementReview{}, ErrNotFound
	}
	return value, err
}

// ListRequirementReviews returns at most limit+1 scoped rows for pagination.
func (r *PostgresRepository) ListRequirementReviews(ctx context.Context, tenant, programID, after string, limit int) ([]RequirementReview, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidState
	}
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `
 SELECT rr.id::text,t.id::text,rr.program_id::text,rr.requirement_id::text,
 rr.proposed_by::text,rr.proposal_rationale,rr.proposed_at,COALESCE(rr.reviewed_by::text,''),
 COALESCE(rr.review_rationale,''),rr.reviewed_at,rr.status,rr.version
 FROM program_requirement_reviews rr JOIN tenants t ON t.id=rr.tenant_id
 JOIN programs p ON p.id=rr.program_id AND p.tenant_id=rr.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND rr.program_id=$2::uuid
 AND ($3='' OR rr.id>NULLIF($3,'')::uuid)
 AND (NOT $4 OR ((t.id::text=$5 OR t.slug=$5) AND p.legal_entity_id IS NOT NULL
 AND ($6='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id
 AND (le.id::text=$6 OR le.code=$6) AND le.valid_from<=clock_timestamp()
 AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
 ORDER BY le.valid_from DESC,le.id LIMIT 1))))
 ORDER BY rr.id LIMIT $7`, tenant, programID, after, enforce, actorTenant, actorEntity, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]RequirementReview, 0, limit+1)
	for rows.Next() {
		var v RequirementReview
		if err = rows.Scan(&v.ID, &v.TenantID, &v.ProgramID, &v.RequirementID, &v.ProposedBy, &v.ProposalRationale, &v.ProposedAt, &v.ReviewedBy, &v.ReviewRationale, &v.ReviewedAt, &v.Status, &v.Version); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
