//go:build postgres

package risk

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) IndicatorPortfolio(ctx context.Context, scope Scope, filter IndicatorPortfolioFilter) (IndicatorPortfolioPage, error) {
	if r == nil || r.pool == nil {
		return IndicatorPortfolioPage{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	filter, err = normalizeIndicatorPortfolioFilter(filter)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	cursor, err := decodeIndicatorPortfolioCursor(filter.Cursor)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	hasCursor := cursor.MonitoringCheckID != ""

	rows, err := r.pool.Query(ctx, `
		WITH ranked AS (
			SELECT
				i.risk_id,
				i.program_id,
				i.monitoring_check_id,
				i.monitoring_check_version,
				i.kind,
				ROW_NUMBER() OVER (
					PARTITION BY i.risk_id,i.monitoring_check_id
					ORDER BY i.risk_version DESC,i.id DESC
				) AS position
			FROM risk_indicator_links i
			JOIN risks r
			  ON r.tenant_id=i.tenant_id
			 AND r.legal_entity_id=i.legal_entity_id
			 AND r.id=i.risk_id
			JOIN tenants t ON t.id=i.tenant_id
			JOIN legal_entities le ON le.tenant_id=i.tenant_id AND le.id=i.legal_entity_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND r.status='ACTIVE'
		),
		grouped AS (
			SELECT
				program_id,
				monitoring_check_id,
				monitoring_check_version,
				kind,
				COUNT(*)::integer AS risk_count
			FROM ranked
			WHERE position=1
			  AND ($3='' OR kind=$3)
			GROUP BY program_id,monitoring_check_id,monitoring_check_version,kind
		)
		SELECT
			program_id::text,
			monitoring_check_id::text,
			monitoring_check_version,
			kind,
			risk_count
		FROM grouped
		WHERE (
			NOT $4::boolean OR
			(kind,monitoring_check_id::text,monitoring_check_version,program_id::text)
			  > ($5::text,$6::text,$7::bigint,$8::text)
		)
		ORDER BY kind,monitoring_check_id::text,monitoring_check_version,program_id::text
		LIMIT $9`,
		scope.TenantID, scope.LegalEntityID, string(filter.Kind), hasCursor,
		string(cursor.Kind), cursor.MonitoringCheckID, cursor.MonitoringCheckVersion, cursor.ProgramID,
		filter.Limit+1,
	)
	if err != nil {
		return IndicatorPortfolioPage{}, fmt.Errorf("list indicator portfolio: %w", err)
	}
	defer rows.Close()

	items := make([]IndicatorPortfolioItem, 0, filter.Limit+1)
	for rows.Next() {
		var item IndicatorPortfolioItem
		if err := rows.Scan(
			&item.ProgramID,
			&item.MonitoringCheckID,
			&item.MonitoringCheckVersion,
			&item.Kind,
			&item.RiskCount,
		); err != nil {
			return IndicatorPortfolioPage{}, fmt.Errorf("scan indicator portfolio: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return IndicatorPortfolioPage{}, err
	}

	page := IndicatorPortfolioPage{Items: items}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		page.NextCursor, err = encodeIndicatorPortfolioCursor(page.Items[len(page.Items)-1])
		if err != nil {
			return IndicatorPortfolioPage{}, err
		}
	}
	return page, nil
}
