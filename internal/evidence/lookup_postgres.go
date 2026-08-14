//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) LatestRequestForSubject(ctx context.Context, tenant, subjectType, subjectID string) (Request, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT cr.id::text
		FROM capture_requests cr
		JOIN tenants t ON t.id=cr.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND cr.subject_type=$2 AND cr.subject_id=$3
		ORDER BY
			CASE WHEN cr.status IN ('DRAFT','READY','IN_PROGRESS') THEN 0 ELSE 1 END,
			CASE WHEN cr.status IN ('DRAFT','READY','IN_PROGRESS') THEN cr.deadline END ASC NULLS LAST,
			cr.updated_at DESC,cr.id DESC
		LIMIT 1`, tenant, subjectType, subjectID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, err
	}
	return r.GetRequest(ctx, tenant, id)
}

func (r *PostgresRepository) SourcesByCodes(ctx context.Context, tenant string, codes []string) ([]Source, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT es.id::text,
		       t.id::text,
		       COALESCE(es.legal_entity_id::text,''),
		       es.code,
		       es.name,
		       es.source_type,
		       es.authority_class,
		       COALESCE(es.owner_principal_id::text,''),
		       es.expected_freshness_minutes,
		       es.last_observed_at,
		       es.last_success_at,
		       es.health,
		       es.status,
		       es.version,
		       es.created_at,
		       es.updated_at
		FROM evidence_sources es
		JOIN tenants t ON t.id=es.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND es.code=ANY($2::text[])
		ORDER BY es.code`, tenant, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Source{}
	for rows.Next() {
		value, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ subjectLookupRepository = (*PostgresRepository)(nil)
