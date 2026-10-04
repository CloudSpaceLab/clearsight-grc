//go:build postgres

package metricview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *ObservationRepository) Trend(ctx context.Context, tenantID, legalEntityID, metricID string, start, end time.Time) (TrendSeries, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	metricID = strings.TrimSpace(metricID)
	if r == nil || r.pool == nil || ctx == nil || tenantID == "" || legalEntityID == "" || !validHomeTrendMetric(metricID) {
		return TrendSeries{}, ErrTrendInvalid
	}
	resolution, err := trendResolution(start, end)
	if err != nil {
		return TrendSeries{}, err
	}
	start, end = start.UTC(), end.UTC()
	series := TrendSeries{
		MetricID: metricID, DefinitionRevision: HomeDefinitionRevision, Start: start, End: end,
		Resolution: resolution, Points: []TrendPoint{}, Direction: TrendUnknown, ComparisonQuality: ComparisonMissing,
	}

	if resolution == TrendResolutionHour {
		series.Points, err = r.hourlyTrendPoints(ctx, tenantID, legalEntityID, metricID, start, end)
	} else {
		series.Points, err = r.dailyTrendPoints(ctx, tenantID, legalEntityID, metricID, start, end)
	}
	if err != nil {
		return TrendSeries{}, err
	}
	series.Baseline, err = r.comparisonPoint(ctx, tenantID, legalEntityID, metricID, start, true)
	if err != nil && !errors.Is(err, ErrTrendNotFound) {
		return TrendSeries{}, err
	}
	series.Current, err = r.comparisonPoint(ctx, tenantID, legalEntityID, metricID, end, false)
	if err != nil && !errors.Is(err, ErrTrendNotFound) {
		return TrendSeries{}, err
	}
	if len(series.Points) == 0 && series.Current == nil {
		return TrendSeries{}, ErrTrendNotFound
	}
	decorateTrendComparison(&series)
	return series, nil
}

func (r *ObservationRepository) hourlyTrendPoints(ctx context.Context, tenantID, legalEntityID, metricID string, start, end time.Time) ([]TrendPoint, error) {
	rows, err := r.pool.Query(ctx, `
		WITH ranked AS (
			SELECT observation.generated_at,observation.value,observation.freshness,observation.completeness,
			       observation.population,observation.excluded,observation.unknown,observation.source_revision,
			       row_number() OVER (
			         PARTITION BY date_trunc('hour',observation.generated_at)
			         ORDER BY observation.generated_at DESC,observation.id DESC
			       ) sequence
			FROM metric_observations observation
			JOIN tenants tenant ON tenant.id=observation.tenant_id
			JOIN legal_entities entity ON entity.tenant_id=observation.tenant_id AND entity.id=observation.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND observation.metric_id=$3
			  AND observation.definition_revision=$4
			  AND observation.generated_at>=$5
			  AND observation.generated_at<=$6
		)
		SELECT generated_at,value,freshness,completeness,population,excluded,unknown,source_revision
		FROM ranked WHERE sequence=1
		ORDER BY generated_at`,
		tenantID, legalEntityID, metricID, HomeDefinitionRevision, start, end)
	if err != nil {
		return nil, fmt.Errorf("load hourly metric trend: %w", err)
	}
	defer rows.Close()
	return scanTrendPoints(rows)
}

func (r *ObservationRepository) dailyTrendPoints(ctx context.Context, tenantID, legalEntityID, metricID string, start, end time.Time) ([]TrendPoint, error) {
	rows, err := r.pool.Query(ctx, `
		WITH raw_ranked AS (
			SELECT (observation.generated_at AT TIME ZONE 'UTC')::date bucket_date,
			       observation.generated_at,observation.value,observation.freshness,observation.completeness,
			       observation.population,observation.excluded,observation.unknown,observation.source_revision,
			       row_number() OVER (
			         PARTITION BY (observation.generated_at AT TIME ZONE 'UTC')::date
			         ORDER BY observation.generated_at DESC,observation.id DESC
			       ) sequence
			FROM metric_observations observation
			JOIN tenants tenant ON tenant.id=observation.tenant_id
			JOIN legal_entities entity ON entity.tenant_id=observation.tenant_id AND entity.id=observation.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND observation.metric_id=$3
			  AND observation.definition_revision=$4
			  AND observation.generated_at>=$5
			  AND observation.generated_at<=$6
		), combined AS (
			SELECT bucket_date,generated_at,value,freshness,completeness,population,excluded,unknown,source_revision,0 source_priority
			FROM raw_ranked WHERE sequence=1
			UNION ALL
			SELECT rollup.bucket_date,rollup.generated_at,rollup.value,rollup.freshness,rollup.completeness,
			       rollup.population,rollup.excluded,rollup.unknown,rollup.source_revision,1
			FROM metric_observation_daily_rollups rollup
			JOIN tenants tenant ON tenant.id=rollup.tenant_id
			JOIN legal_entities entity ON entity.tenant_id=rollup.tenant_id AND entity.id=rollup.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND rollup.metric_id=$3
			  AND rollup.definition_revision=$4
			  AND rollup.bucket_date>=($5 AT TIME ZONE 'UTC')::date
			  AND rollup.bucket_date<=($6 AT TIME ZONE 'UTC')::date
		), chosen AS (
			SELECT DISTINCT ON (bucket_date)
			       bucket_date,generated_at,value,freshness,completeness,population,excluded,unknown,source_revision
			FROM combined
			ORDER BY bucket_date,source_priority,generated_at DESC
		)
		SELECT generated_at,value,freshness,completeness,population,excluded,unknown,source_revision
		FROM chosen ORDER BY bucket_date`,
		tenantID, legalEntityID, metricID, HomeDefinitionRevision, start, end)
	if err != nil {
		return nil, fmt.Errorf("load daily metric trend: %w", err)
	}
	defer rows.Close()
	return scanTrendPoints(rows)
}

func (r *ObservationRepository) comparisonPoint(ctx context.Context, tenantID, legalEntityID, metricID string, boundary time.Time, before bool) (*TrendPoint, error) {
	operator := "<="
	if before {
		operator = "<"
	}
	query := `
		WITH candidates AS (
			SELECT observation.generated_at,observation.value,observation.freshness,observation.completeness,
			       observation.population,observation.excluded,observation.unknown,observation.source_revision,0 source_priority
			FROM metric_observations observation
			JOIN tenants tenant ON tenant.id=observation.tenant_id
			JOIN legal_entities entity ON entity.tenant_id=observation.tenant_id AND entity.id=observation.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND observation.metric_id=$3
			  AND observation.definition_revision=$4
			  AND observation.generated_at ` + operator + ` $5
			UNION ALL
			SELECT rollup.generated_at,rollup.value,rollup.freshness,rollup.completeness,
			       rollup.population,rollup.excluded,rollup.unknown,rollup.source_revision,1
			FROM metric_observation_daily_rollups rollup
			JOIN tenants tenant ON tenant.id=rollup.tenant_id
			JOIN legal_entities entity ON entity.tenant_id=rollup.tenant_id AND entity.id=rollup.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND rollup.metric_id=$3
			  AND rollup.definition_revision=$4
			  AND rollup.generated_at ` + operator + ` $5
		)
		SELECT generated_at,value,freshness,completeness,population,excluded,unknown,source_revision
		FROM candidates
		ORDER BY generated_at DESC,source_priority
		LIMIT 1`
	row := r.pool.QueryRow(ctx, query, tenantID, legalEntityID, metricID, HomeDefinitionRevision, boundary.UTC())
	point := TrendPoint{}
	if err := row.Scan(&point.At,&point.Value,&point.Freshness,&point.Completeness,&point.Population,&point.Excluded,&point.Unknown,&point.SourceRevision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTrendNotFound
		}
		return nil, fmt.Errorf("load metric comparison point: %w", err)
	}
	point.At = point.At.UTC()
	return &point, nil
}

func scanTrendPoints(rows pgx.Rows) ([]TrendPoint, error) {
	points := make([]TrendPoint, 0, 180)
	for rows.Next() {
		var point TrendPoint
		if err := rows.Scan(&point.At,&point.Value,&point.Freshness,&point.Completeness,&point.Population,&point.Excluded,&point.Unknown,&point.SourceRevision); err != nil {
			return nil, fmt.Errorf("scan metric trend point: %w", err)
		}
		point.At = point.At.UTC()
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric trend points: %w", err)
	}
	return points, nil
}

func (r *ObservationRepository) maintainTrendRetention(ctx context.Context, now time.Time, limit int) error {
	if r == nil || r.pool == nil {
		return ErrInvalidObservation
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	now = now.UTC()
	if _, err := r.pool.Exec(ctx, `
		WITH latest AS (
			SELECT * FROM (
				SELECT DISTINCT ON (
					observation.tenant_id,observation.legal_entity_id,observation.metric_id,
					observation.definition_revision,(observation.generated_at AT TIME ZONE 'UTC')::date
				)
				observation.tenant_id,observation.legal_entity_id,observation.metric_id,observation.definition_revision,
				(observation.generated_at AT TIME ZONE 'UTC')::date bucket_date,
				observation.id observation_id,observation.source_id,observation.source_revision,observation.source_high_water,
				observation.generated_at,observation.value,observation.condition,observation.freshness,observation.completeness,
				observation.population,observation.excluded,observation.unknown
				FROM metric_observations observation
				LEFT JOIN metric_observation_daily_rollups rollup
				  ON rollup.tenant_id=observation.tenant_id
				 AND rollup.legal_entity_id=observation.legal_entity_id
				 AND rollup.metric_id=observation.metric_id
				 AND rollup.definition_revision=observation.definition_revision
				 AND rollup.bucket_date=(observation.generated_at AT TIME ZONE 'UTC')::date
				WHERE observation.generated_at<date_trunc('day',$1::timestamptz)
				  AND (rollup.generated_at IS NULL OR observation.generated_at>rollup.generated_at)
				ORDER BY observation.tenant_id,observation.legal_entity_id,observation.metric_id,
				         observation.definition_revision,(observation.generated_at AT TIME ZONE 'UTC')::date,
				         observation.generated_at DESC,observation.id DESC
			) candidate
			ORDER BY generated_at
			LIMIT $2
		)
		INSERT INTO metric_observation_daily_rollups(
			tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date,observation_id,source_id,source_revision,
			source_high_water,generated_at,value,condition,freshness,completeness,population,excluded,unknown,rolled_at
		)
		SELECT tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date,observation_id,source_id,source_revision,
		       source_high_water,generated_at,value,condition,freshness,completeness,population,excluded,unknown,clock_timestamp()
		FROM latest
		ON CONFLICT(tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date) DO UPDATE SET
			observation_id=EXCLUDED.observation_id,source_id=EXCLUDED.source_id,source_revision=EXCLUDED.source_revision,
			source_high_water=EXCLUDED.source_high_water,generated_at=EXCLUDED.generated_at,value=EXCLUDED.value,
			condition=EXCLUDED.condition,freshness=EXCLUDED.freshness,completeness=EXCLUDED.completeness,
			population=EXCLUDED.population,excluded=EXCLUDED.excluded,unknown=EXCLUDED.unknown,rolled_at=clock_timestamp()
		WHERE EXCLUDED.generated_at>metric_observation_daily_rollups.generated_at`, now, limit); err != nil {
		return fmt.Errorf("roll up metric observations: %w", err)
	}

	if _, err := r.pool.Exec(ctx, `
		DELETE FROM metric_observations
		WHERE id IN (
			SELECT observation.id
			FROM metric_observations observation
			WHERE observation.generated_at<$1::timestamptz
			ORDER BY observation.generated_at,observation.id
			LIMIT $2
		)`, now.Add(-RawObservationRetention), limit); err != nil {
		return fmt.Errorf("prune raw metric observations: %w", err)
	}

	if _, err := r.pool.Exec(ctx, `
		DELETE FROM metric_observation_daily_rollups rollup
		WHERE (rollup.tenant_id,rollup.legal_entity_id,rollup.metric_id,rollup.definition_revision,rollup.bucket_date) IN (
			SELECT candidate.tenant_id,candidate.legal_entity_id,candidate.metric_id,candidate.definition_revision,candidate.bucket_date
			FROM metric_observation_daily_rollups candidate
			WHERE candidate.bucket_date<($1::timestamptz AT TIME ZONE 'UTC')::date
			  AND NOT EXISTS (
				SELECT 1 FROM metric_observations observation
				WHERE observation.tenant_id=candidate.tenant_id
				  AND observation.legal_entity_id=candidate.legal_entity_id
				  AND observation.metric_id=candidate.metric_id
				  AND observation.definition_revision=candidate.definition_revision
				  AND (observation.generated_at AT TIME ZONE 'UTC')::date=candidate.bucket_date
			  )
			ORDER BY candidate.bucket_date
			LIMIT $2
		)`, now.Add(-DailyRollupRetention), limit); err != nil {
		return fmt.Errorf("prune metric daily rollups: %w", err)
	}
	return nil
}

var _ TrendReader = (*ObservationRepository)(nil)
