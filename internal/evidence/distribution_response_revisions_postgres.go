//go:build postgres

package evidence

import (
	"context"
	"fmt"
)

func (s *PostgresDistributionStore) ListDistributionResponseRevisions(ctx context.Context, tenantID, legalEntityID, distributionID string, limit int) ([]ResponseRevision, error) {
	if s == nil || s.repo == nil || s.repo.pool == nil {
		return nil, fmt.Errorf("postgres distribution repository is required")
	}
	rows, err := s.repo.pool.Query(ctx, `
		SELECT `+responseRevisionProjection+`
		FROM capture_response_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid
		ORDER BY revision DESC,id DESC
		LIMIT $4`, tenantID, legalEntityID, distributionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list distribution response revisions: %w", err)
	}
	defer rows.Close()
	values := make([]ResponseRevision, 0)
	for rows.Next() {
		value, scanErr := scanPostgresResponseRevision(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan distribution response revision: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate distribution response revisions: %w", err)
	}
	return values, nil
}
