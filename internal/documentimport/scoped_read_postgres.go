//go:build postgres

package documentimport

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ListScoped(ctx context.Context, tenant, legalEntityID string, limit int) ([]DocumentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT di.id::text,t.slug,COALESCE(di.legal_entity_id::text,''),di.file_name,di.media_type,di.purpose,di.source_type,
		       di.size_bytes,di.sha256,di.artifact_status,di.extraction_status,di.analysis_status,
		       di.sections_total,di.sections_omitted,di.proposals_total,di.proposals_omitted,
		       COALESCE((SELECT count(*)::int FROM jsonb_array_elements(di.proposals) p WHERE p->>'status'='PENDING_REVIEW'),0),
		       GREATEST(jsonb_array_length(di.proposals)-COALESCE((SELECT count(*)::int FROM jsonb_array_elements(di.proposals) p WHERE p->>'status'='PENDING_REVIEW'),0),0),
		       di.content_truncated,di.processed_at,di.created_at,di.updated_at,di.version
		FROM document_imports di JOIN tenants t ON t.id=di.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND COALESCE(di.legal_entity_id::text,'')=$2
		ORDER BY di.created_at DESC,di.id DESC LIMIT $3`, tenant, legalEntityID, limit)
	if err != nil {
		return nil, fmt.Errorf("list scoped document imports: %w", err)
	}
	defer rows.Close()
	values := make([]DocumentSummary, 0, limit)
	for rows.Next() {
		value, scanErr := scanSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan scoped document import summary: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list scoped document imports: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) GetScoped(ctx context.Context, tenant, legalEntityID, id string) (Document, error) {
	row := r.pool.QueryRow(ctx, documentSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND di.id=$2::uuid AND COALESCE(di.legal_entity_id::text,'')=$3`, tenant, id, legalEntityID)
	value, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("load scoped document import: %w", err)
	}
	return value, nil
}

var _ scopedReadRepository = (*PostgresRepository)(nil)
