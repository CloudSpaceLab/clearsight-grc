//go:build postgres

package oversight

import (
	"context"
	"fmt"
	"strings"
)

func (r *PostgresRepository) LatestMany(ctx context.Context, tenantID string, legalEntityIDs []string) ([]Snapshot, error) {
	tenantID = strings.TrimSpace(tenantID)
	if r == nil || r.pool == nil || tenantID == "" || len(legalEntityIDs) == 0 || len(legalEntityIDs) > 256 {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (os.legal_entity_id)
		       os.id::text,os.generated_at,os.period_start,os.period_end,os.projection_version,os.source_high_water,
		       os.coverage_population,os.coverage_excluded,os.coverage_unknown,os.payload,
		       t.id::text,le.id::text
		FROM oversight_snapshots os
		JOIN tenants t ON t.id=os.tenant_id
		JOIN legal_entities le ON le.tenant_id=os.tenant_id AND le.id=os.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND os.legal_entity_id=ANY($2::uuid[])
		ORDER BY os.legal_entity_id,os.generated_at DESC,os.id DESC`, tenantID, legalEntityIDs)
	if err != nil {
		return nil, fmt.Errorf("load group oversight snapshots: %w", err)
	}
	defer rows.Close()

	values := make([]Snapshot, 0, len(legalEntityIDs))
	for rows.Next() {
		var value Snapshot
		var highWater, payload []byte
		if err := rows.Scan(
			&value.SnapshotID, &value.GeneratedAt, &value.PeriodStart, &value.PeriodEnd, &value.ProjectionVersion, &highWater,
			&value.Coverage.Population, &value.Coverage.Excluded, &value.Coverage.Unknown, &payload,
			&value.TenantID, &value.LegalEntityID,
		); err != nil {
			return nil, fmt.Errorf("scan group oversight snapshot: %w", err)
		}
		if err := decodeStoredSnapshot(&value, highWater, payload); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group oversight snapshots: %w", err)
	}
	return values, nil
}

var _ BatchRepository = (*PostgresRepository)(nil)
