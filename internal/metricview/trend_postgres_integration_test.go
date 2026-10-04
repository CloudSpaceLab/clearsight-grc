//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMetricTrendRetentionPreservesComparisonAndBoundsRawHistory(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const tenantID = "8f630000-0000-4000-8000-000000000001"
	const entityID = "8f630000-0000-4000-8000-000000000002"
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM metric_observation_daily_rollups WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM metric_observations WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshot_metric_memberships WHERE oversight_snapshot_id IN (SELECT id FROM oversight_snapshots WHERE tenant_id=$1::uuid)`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshot_metric_membership_sets WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshots WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE id=$1::uuid`, entityID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'metric-trend-retention','Metric Trend Retention')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'TREND-NG','Trend Nigeria','NG',$3)`, entityID, tenantID, now.Add(-3*365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	insertObservation := func(at time.Time, value int) string {
		t.Helper()
		var snapshotID string
		payload := fmt.Sprintf(`{"counts":{"overdue":%d}}`, value)
		if err := pool.QueryRow(ctx, `
			INSERT INTO oversight_snapshots(
				tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,projection_version,
				metric_membership_revision,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
			) VALUES($1::uuid,$2::uuid,$3,$4,$4,$4,$5,'home-oversight-v3','{}'::jsonb,10,0,0,$6::jsonb)
			RETURNING id::text`,
			tenantID, entityID, at.Add(-90*24*time.Hour), at, oversight.ProjectionVersion, payload,
		).Scan(&snapshotID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO oversight_snapshot_metric_membership_sets(oversight_snapshot_id,tenant_id,legal_entity_id,definition_revision)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4)`, snapshotID, tenantID, entityID, HomeDefinitionRevision); err != nil {
			t.Fatal(err)
		}
		if value > 0 {
			if _, err := pool.Exec(ctx, `
				INSERT INTO oversight_snapshot_metric_memberships(
					oversight_snapshot_id,metric_id,definition_revision,member_id,target_type,target_id,target_title,state
				)
				SELECT $1::uuid,'overdue_open',$2,uuidv7(),'MATTER',uuidv7(),'Retained overdue issue','ASSESSMENT'
				FROM generate_series(1,$3)`, snapshotID, HomeDefinitionRevision, value); err != nil {
				t.Fatal(err)
			}
		}
		condition := "ATTENTION"
		if value == 0 {
			condition = "CLEAR"
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO metric_observations(
				tenant_id,legal_entity_id,metric_id,definition_revision,source_kind,source_id,source_revision,source_high_water,
				generated_at,period_start,period_end,posture_as_of,value,condition,freshness,completeness,population,excluded,unknown
			) VALUES(
				$1::uuid,$2::uuid,'overdue_open',$3,'OVERSIGHT_SNAPSHOT',$4::uuid,$5,'{}'::jsonb,
				$6,$7,$6,$6,$8,$9,'CURRENT','COMPLETE',10,0,0
			)`, tenantID, entityID, HomeDefinitionRevision, snapshotID, oversight.ProjectionVersion, at, at.Add(-90*24*time.Hour), value, condition); err != nil {
			t.Fatal(err)
		}
		return snapshotID
	}

	oldAt := now.Add(-200 * 24 * time.Hour)
	insertObservation(oldAt.Add(-time.Hour), 9)
	insertObservation(oldAt, 8)
	start := now.Add(-30 * 24 * time.Hour)
	insertObservation(start, 5)
	insertObservation(now.Add(-3*time.Hour), 4)
	currentSnapshotID := insertObservation(now.Add(-time.Hour), 3)

	repository := NewObservationRepository(pool)
	if err := repository.maintainTrendRetention(ctx, now, 100); err != nil {
		t.Fatal(err)
	}

	var oldRaw int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM metric_observations WHERE tenant_id=$1::uuid AND generated_at<$2`, tenantID, now.Add(-RawObservationRetention)).Scan(&oldRaw); err != nil {
		t.Fatal(err)
	}
	if oldRaw != 0 {
		t.Fatalf("old raw observations=%d, want 0", oldRaw)
	}
	var rolledValue int
	if err := pool.QueryRow(ctx, `SELECT value FROM metric_observation_daily_rollups WHERE tenant_id=$1::uuid AND metric_id='overdue_open' AND bucket_date=($2::timestamptz AT TIME ZONE 'UTC')::date`, tenantID, oldAt).Scan(&rolledValue); err != nil {
		t.Fatal(err)
	}
	if rolledValue != 8 {
		t.Fatalf("daily rollup value=%d, want 8", rolledValue)
	}

	series, err := repository.Trend(ctx, tenantID, entityID, "overdue_open", start, now)
	if err != nil {
		t.Fatal(err)
	}
	if series.Resolution != TrendResolutionDay || series.Baseline == nil || series.Baseline.Value != 8 || series.Current == nil || series.Current.Value != 3 {
		t.Fatalf("trend series=%#v", series)
	}
	if series.Delta == nil || *series.Delta != -5 || series.Direction != TrendImproved || series.ComparisonQuality != ComparisonComplete {
		t.Fatalf("comparison=%#v", series)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM metric_observations WHERE source_id=$1::uuid`, currentSnapshotID); err == nil {
		t.Fatal("recent raw metric observation deletion was accepted")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO metric_observation_daily_rollups(
			tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date,observation_id,source_id,source_revision,
			source_high_water,generated_at,value,condition,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,'overdue_open',$3,($4::timestamptz AT TIME ZONE 'UTC')::date,uuidv7(),uuidv7(),$5,
			'{}'::jsonb,$4,1,'ATTENTION','CURRENT','COMPLETE',10,0,0
		)`, tenantID, entityID, HomeDefinitionRevision, now.Add(-731*24*time.Hour), oversight.ProjectionVersion); err != nil {
		t.Fatal(err)
	}
	if err := repository.maintainTrendRetention(ctx, now, 100); err != nil {
		t.Fatal(err)
	}
	var staleRollups int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM metric_observation_daily_rollups WHERE tenant_id=$1::uuid AND bucket_date<($2::timestamptz AT TIME ZONE 'UTC')::date`, tenantID, now.Add(-DailyRollupRetention)).Scan(&staleRollups); err != nil {
		t.Fatal(err)
	}
	if staleRollups != 0 {
		t.Fatalf("stale rollups=%d, want 0", staleRollups)
	}
}
