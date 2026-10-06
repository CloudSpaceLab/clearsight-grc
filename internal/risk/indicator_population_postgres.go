//go:build postgres

package risk

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) ListIndicatorPopulation(ctx context.Context, scope Scope, filter IndicatorPopulationFilter) (IndicatorPopulationPage, error) {
	if r == nil || r.pool == nil {
		return IndicatorPopulationPage{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return IndicatorPopulationPage{}, err
	}
	const currentLinks = `
		WITH current_links AS (
			SELECT i.id,i.risk_id,i.risk_version,i.program_id,i.monitoring_check_id,i.monitoring_check_version,
			       i.kind,i.measurement,i.linked_by,i.created_at,
			       r.code AS risk_code,r.name AS risk_name,r.organization_scope_id,r.status AS risk_status,
			       row_number() OVER (
			          PARTITION BY i.risk_id,i.monitoring_check_id
			          ORDER BY i.risk_version DESC,i.id DESC
			       ) AS current_rank
			FROM risk_indicator_links i
			JOIN risks r ON r.tenant_id=i.tenant_id AND r.legal_entity_id=i.legal_entity_id AND r.id=i.risk_id
			JOIN tenants t ON t.id=i.tenant_id
			JOIN legal_entities le ON le.tenant_id=i.tenant_id AND le.id=i.legal_entity_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND (NOT $3::boolean OR r.organization_scope_id=ANY($4::uuid[]))
		),
		canonical AS (
			SELECT DISTINCT ON (monitoring_check_id)
			       id,risk_id,risk_version,program_id,monitoring_check_id,monitoring_check_version,
			       kind,measurement,linked_by,created_at
			FROM current_links
			WHERE current_rank=1
			ORDER BY monitoring_check_id,monitoring_check_version DESC,risk_version DESC,created_at DESC,id DESC
		)
	`
	rows, err := r.pool.Query(ctx, currentLinks+`
		SELECT c.id::text,c.risk_id::text,c.risk_version,c.program_id::text,c.monitoring_check_id::text,
		       c.monitoring_check_version,c.kind,c.measurement,COALESCE(c.linked_by::text,''),c.created_at,
		       (SELECT count(*) FROM current_links x WHERE x.current_rank=1 AND x.monitoring_check_id=c.monitoring_check_id),
		       EXISTS(SELECT 1 FROM current_links x WHERE x.current_rank=1 AND x.monitoring_check_id=c.monitoring_check_id AND x.kind<>c.kind)
		FROM canonical c
		WHERE ($5='' OR c.kind=$5)
		ORDER BY c.created_at DESC,c.monitoring_check_id DESC
		LIMIT $6
	`, scope.TenantID, scope.LegalEntityID, filter.OrganizationScopeID != "", filter.OrganizationScopeIDs, string(filter.Kind), filter.Limit+1)
	if err != nil {
		return IndicatorPopulationPage{}, fmt.Errorf("list indicator population: %w", err)
	}
	defer rows.Close()

	items := make([]IndicatorPopulationItem, 0, filter.Limit+1)
	checkIDs := make([]string, 0, filter.Limit)
	for rows.Next() {
		var item IndicatorPopulationItem
		if err := rows.Scan(
			&item.Link.ID, &item.Link.RiskID, &item.Link.RiskVersion, &item.Link.ProgramID,
			&item.Link.MonitoringCheckID, &item.Link.MonitoringCheckVersion, &item.Link.Kind,
			&item.Link.Measurement, &item.Link.LinkedBy, &item.Link.CreatedAt,
			&item.RiskCount, &item.KindConflict,
		); err != nil {
			return IndicatorPopulationPage{}, fmt.Errorf("scan indicator population: %w", err)
		}
		items = append(items, item)
		if len(items) <= filter.Limit {
			checkIDs = append(checkIDs, item.Link.MonitoringCheckID)
		}
	}
	if err := rows.Err(); err != nil {
		return IndicatorPopulationPage{}, err
	}

	page := IndicatorPopulationPage{OrganizationScopeID: filter.OrganizationScopeID}
	if len(items) > filter.Limit {
		page.Truncated = true
		items = items[:filter.Limit]
	}
	page.Items = items
	if len(items) == 0 {
		return page, nil
	}

	refs, err := r.pool.Query(ctx, currentLinks+`
		SELECT q.monitoring_check_id::text,q.risk_id::text,q.risk_code,q.risk_name,
		       COALESCE(q.organization_scope_id::text,''),q.risk_status
		FROM (
			SELECT x.*,
			       row_number() OVER (PARTITION BY x.monitoring_check_id ORDER BY x.risk_code,x.risk_id) AS reference_rank
			FROM current_links x
			WHERE x.current_rank=1 AND x.monitoring_check_id=ANY($5::uuid[])
		) q
		WHERE q.reference_rank<=$6
		ORDER BY q.monitoring_check_id,q.reference_rank
	`, scope.TenantID, scope.LegalEntityID, filter.OrganizationScopeID != "", filter.OrganizationScopeIDs, checkIDs, indicatorPopulationRiskPreviewLimit)
	if err != nil {
		return IndicatorPopulationPage{}, fmt.Errorf("list indicator risk references: %w", err)
	}
	defer refs.Close()
	byCheck := make(map[string]int, len(items))
	for index := range items {
		byCheck[items[index].Link.MonitoringCheckID] = index
		items[index].RisksTruncated = items[index].RiskCount > indicatorPopulationRiskPreviewLimit
	}
	for refs.Next() {
		var checkID string
		var reference IndicatorRiskReference
		if err := refs.Scan(&checkID, &reference.ID, &reference.Code, &reference.Name, &reference.OrganizationScopeID, &reference.Status); err != nil {
			return IndicatorPopulationPage{}, fmt.Errorf("scan indicator risk reference: %w", err)
		}
		if index, ok := byCheck[checkID]; ok {
			items[index].Risks = append(items[index].Risks, reference)
		}
	}
	if err := refs.Err(); err != nil {
		return IndicatorPopulationPage{}, err
	}
	page.Items = items
	return page, nil
}

var _ IndicatorPopulationRepository = (*PostgresRepository)(nil)
