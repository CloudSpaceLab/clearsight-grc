//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMetricObservationProjectionIsDurableIdempotentAndRepairable(t *testing.T) {
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

	const (
		tenantID   = "8f500000-0000-4000-8000-000000000001"
		entityID   = "8f500000-0000-4000-8000-000000000002"
		snapshotID = "8f500000-0000-4000-8000-000000000003"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM metric_observations WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshots WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	highWater, err := json.Marshal(map[string]time.Time{"matters": now.Add(-time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"counts": oversight.Counts{CriticalHigh: 7, Overdue: 4, RoutingFailures: 2, OutcomeFailures: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'metric-observation-test','Metric Observation Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($2::uuid,$1::uuid,'METRIC-NG','Metric Nigeria','NG',$4);
		INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
		) VALUES(
			$3::uuid,$1::uuid,$2::uuid,$5,$6,$6,$6,
			$7,$8::jsonb,100,2,3,$9::jsonb
		)`,
		tenantID,
		entityID,
		snapshotID,
		now.Add(-time.Hour),
		now.Add(-90*24*time.Hour),
		now,
		oversight.ProjectionVersion,
		highWater,
		payload,
	); err != nil {
		t.Fatal(err)
	}

	repository := NewObservationRepository(pool)
	if err := repository.validateDefinitions(ctx); err != nil {
		t.Fatalf("definition parity: %v", err)
	}
	maintainer := &ObservationMaintainer{Repository: repository}
	if _, err := maintainer.Maintain(ctx, now, 10); err != nil {
		t.Fatal(err)
	}

	count, err := repository.countObservationsForSource(ctx, snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(HomeDefinitions) {
		t.Fatalf("observation count=%d", count)
	}

	var value int
	var condition, sourceRevision string
	var population int
	var excluded, unknown *int
	var storedHighWater []byte
	if err := pool.QueryRow(ctx, `
		SELECT value,condition,source_revision,population,excluded,unknown,source_high_water
		FROM metric_observations
		WHERE source_id=$1::uuid AND metric_id='critical_high_open' AND definition_revision=$2`,
		snapshotID, HomeDefinitionRevision,
	).Scan(&value, &condition, &sourceRevision, &population, &excluded, &unknown, &storedHighWater); err != nil {
		t.Fatal(err)
	}
	if value != 7 || condition != string(ConditionAttention) || sourceRevision != oversight.ProjectionVersion ||
		population != 100 || excluded == nil || *excluded != 2 || unknown == nil || *unknown != 3 {
		t.Fatalf("stored observation value=%d condition=%q source=%q population=%d excluded=%v unknown=%v",
			value, condition, sourceRevision, population, excluded, unknown)
	}
	var decodedHighWater map[string]time.Time
	if err := json.Unmarshal(storedHighWater, &decodedHighWater); err != nil {
		t.Fatal(err)
	}
	if !decodedHighWater["matters"].Equal(now.Add(-time.Minute)) {
		t.Fatalf("source high-water=%#v", decodedHighWater)
	}

	if completed, err := maintainer.Maintain(ctx, now.Add(time.Minute), 10); err != nil || completed != 0 {
		t.Fatalf("idempotent maintain completed=%d err=%v", completed, err)
	}

	if _, err := pool.Exec(ctx, `
		DELETE FROM metric_observations
		WHERE source_id=$1::uuid AND metric_id='routing_gaps' AND definition_revision=$2`,
		snapshotID, HomeDefinitionRevision,
	); err != nil {
		t.Fatal(err)
	}
	if completed, err := maintainer.Maintain(ctx, now.Add(2*time.Minute), 10); err != nil || completed != 1 {
		t.Fatalf("repair maintain completed=%d err=%v", completed, err)
	}
	count, err = repository.countObservationsForSource(ctx, snapshotID)
	if err != nil || count != len(HomeDefinitions) {
		t.Fatalf("repaired observation count=%d err=%v", count, err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE metric_observations SET value=value+1
		WHERE source_id=$1::uuid AND metric_id='critical_high_open'`, snapshotID); err == nil {
		t.Fatal("metric observation mutation was accepted")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE metric_definitions SET label='Changed'
		WHERE metric_id='critical_high_open' AND revision=$1`, HomeDefinitionRevision); err == nil {
		t.Fatal("metric definition mutation was accepted")
	}
}
