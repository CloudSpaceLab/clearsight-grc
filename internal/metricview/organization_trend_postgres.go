//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type organizationTrendQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *ObservationRepository) OrganizationTrend(
	ctx context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	metricID string,
	start time.Time,
	end time.Time,
) (OrganizationTrendSeries, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	organizationScopeID = strings.TrimSpace(organizationScopeID)
	metricID = strings.TrimSpace(metricID)
	if r == nil || r.pool == nil || ctx == nil || tenantID == "" || legalEntityID == "" ||
		organizationScopeID == "" || metricID != "risks_outside_appetite" {
		return OrganizationTrendSeries{}, ErrTrendInvalid
	}
	if _, err := trendResolution(start, end); err != nil {
		return OrganizationTrendSeries{}, err
	}

	points, err := organizationTrendPoints(
		ctx,
		r.pool,
		tenantID,
		legalEntityID,
		organizationScopeID,
		metricID,
		start.UTC(),
		end.UTC(),
	)
	if err != nil {
		return OrganizationTrendSeries{}, err
	}
	if len(points) == 0 {
		return OrganizationTrendSeries{}, ErrTrendNotFound
	}
	return OrganizationTrendSeries{
		MetricID:            metricID,
		DefinitionRevision:  DomainDefinitionRevision,
		OrganizationScopeID: organizationScopeID,
		Start:               start.UTC(),
		End:                 end.UTC(),
		Resolution:          TrendResolutionDay,
		Points:              points,
	}, nil
}

func organizationTrendPoints(
	ctx context.Context,
	q organizationTrendQuerier,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	metricID string,
	start time.Time,
	end time.Time,
) ([]OrganizationTrendPoint, error) {
	rows, err := q.Query(ctx, `
		WITH source_days AS (
			SELECT daily.tenant_id,
			       daily.legal_entity_id,
			       daily.definition_revision,
			       daily.bucket_date,
			       daily.source_id,
			       daily.source_revision,
			       daily.source_generated_at,
			       selected.department_path AS selected_path,
			       COALESCE(quality.source_complete,false) AS source_complete
			FROM organization_metric_daily_sources daily
			JOIN tenants tenant ON tenant.id=daily.tenant_id
			JOIN legal_entities entity
			  ON entity.tenant_id=daily.tenant_id
			 AND entity.id=daily.legal_entity_id
			JOIN LATERAL (
				SELECT event.department_path
				FROM organization_scope_lineage_events event
				WHERE event.tenant_id=daily.tenant_id
				  AND event.legal_entity_id=daily.legal_entity_id
				  AND event.scope_id=$3::uuid
				  AND event.effective_at<=daily.source_generated_at
				ORDER BY event.effective_at DESC,event.id DESC
				LIMIT 1
			) selected ON true
			LEFT JOIN LATERAL (
				SELECT (
					candidate.freshness='CURRENT'
					AND candidate.completeness='COMPLETE'
					AND COALESCE(candidate.excluded,0)=0
					AND COALESCE(candidate.unknown,0)=0
				) AS source_complete
				FROM (
					SELECT observation.freshness,
					       observation.completeness,
					       observation.excluded,
					       observation.unknown,
					       0 AS source_priority
					FROM metric_observations observation
					WHERE observation.tenant_id=daily.tenant_id
					  AND observation.legal_entity_id=daily.legal_entity_id
					  AND observation.source_kind='DOMAIN_SNAPSHOT'
					  AND observation.source_id=daily.source_id
					  AND observation.metric_id=$4
					  AND observation.definition_revision=daily.definition_revision
					UNION ALL
					SELECT rollup.freshness,
					       rollup.completeness,
					       rollup.excluded,
					       rollup.unknown,
					       1 AS source_priority
					FROM metric_observation_daily_rollups rollup
					WHERE rollup.tenant_id=daily.tenant_id
					  AND rollup.legal_entity_id=daily.legal_entity_id
					  AND rollup.source_id=daily.source_id
					  AND rollup.metric_id=$4
					  AND rollup.definition_revision=daily.definition_revision
				) candidate
				ORDER BY candidate.source_priority
				LIMIT 1
			) quality ON true
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND daily.definition_revision=$5
			  AND daily.bucket_date>=($6::timestamptz AT TIME ZONE 'UTC')::date
			  AND daily.bucket_date<=($7::timestamptz AT TIME ZONE 'UTC')::date
		)
		SELECT source.bucket_date,
		       source.source_generated_at,
		       COALESCE(sum(
		         CASE
		           WHEN bucket.organization_scope_id IS NOT NULL
		            AND direct.department_path IS NOT NULL
		            AND cardinality(direct.department_path)>=cardinality(source.selected_path)
		            AND direct.department_path[1:cardinality(source.selected_path)]=source.selected_path
		           THEN bucket.value
		           ELSE 0
		         END
		       ),0)::bigint AS value,
		       source.source_revision,
		       source.source_complete
		FROM source_days source
		LEFT JOIN organization_metric_daily_buckets bucket
		  ON bucket.tenant_id=source.tenant_id
		 AND bucket.legal_entity_id=source.legal_entity_id
		 AND bucket.definition_revision=source.definition_revision
		 AND bucket.bucket_date=source.bucket_date
		 AND bucket.metric_id=$4
		LEFT JOIN organization_scope_lineage_events direct
		  ON direct.id=bucket.lineage_event_id
		GROUP BY source.bucket_date,source.source_generated_at,source.source_revision,source.source_complete
		ORDER BY source.bucket_date`,
		tenantID,
		legalEntityID,
		organizationScopeID,
		metricID,
		DomainDefinitionRevision,
		start.UTC(),
		end.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("load organization risk trend: %w", err)
	}
	defer rows.Close()

	points := make([]OrganizationTrendPoint, 0, 90)
	for rows.Next() {
		var date time.Time
		var point OrganizationTrendPoint
		var value int64
		if err := rows.Scan(
			&date,
			&point.At,
			&value,
			&point.SourceRevision,
			&point.SourceComplete,
		); err != nil {
			return nil, fmt.Errorf("scan organization risk trend: %w", err)
		}
		point.Date = date.UTC().Format("2006-01-02")
		point.At = point.At.UTC()
		point.Value = int(value)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization risk trend: %w", err)
	}
	return points, nil
}

var _ OrganizationTrendReader = (*ObservationRepository)(nil)
