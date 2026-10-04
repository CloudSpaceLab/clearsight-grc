//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
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
		tenantID      = "8f500000-0000-4000-8000-000000000001"
		entityID      = "8f500000-0000-4000-8000-000000000002"
		otherEntityID = "8f500000-0000-4000-8000-000000000004"
		snapshotID    = "8f500000-0000-4000-8000-000000000003"
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
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($2::uuid,$1::uuid,'METRIC-NG','Metric Nigeria','NG',$4),
			($10::uuid,$1::uuid,'METRIC-GH','Metric Ghana','GH',$4);
		INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,metric_membership_revision,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
		) VALUES(
			$3::uuid,$1::uuid,$2::uuid,$5,$6,$6,$6,
			$7,'home-oversight-v3',$8::jsonb,100,2,3,$9::jsonb
		)`,
		pgx.QueryExecModeSimpleProtocol,
		tenantID,
		entityID,
		snapshotID,
		now.Add(-time.Hour),
		now.Add(-90*24*time.Hour),
		now,
		oversight.ProjectionVersion,
		string(highWater),
		string(payload),
		otherEntityID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oversight_snapshot_metric_membership_sets(
			oversight_snapshot_id,tenant_id,legal_entity_id,definition_revision
		) VALUES($1::uuid,$2::uuid,$3::uuid,'home-oversight-v3')`,
		snapshotID, tenantID, entityID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oversight_snapshot_metric_memberships(
			oversight_snapshot_id,metric_id,definition_revision,member_id,target_type,target_id,target_title,state
		)
		SELECT $1::uuid,'critical_high_open','home-oversight-v3',
		       md5('metric-critical-' || gs::text)::uuid,'MATTER',md5('metric-critical-target-' || gs::text)::uuid,
		       'Critical issue ' || gs,'TRIAGE'
		FROM generate_series(1,7) gs
		UNION ALL
		SELECT $1::uuid,'overdue_open','home-oversight-v3',
		       md5('metric-overdue-' || gs::text)::uuid,'MATTER',md5('metric-overdue-target-' || gs::text)::uuid,
		       'Overdue issue ' || gs,'TRIAGE'
		FROM generate_series(1,4) gs
		UNION ALL
		SELECT $1::uuid,'routing_gaps','home-oversight-v3',
		       md5('metric-routing-' || gs::text)::uuid,
		       CASE WHEN gs=2 THEN 'PROGRAM' ELSE 'MATTER' END,
		       md5('metric-routing-target-' || gs::text)::uuid,
		       'Unassigned work ' || gs,'READY'
		FROM generate_series(1,2) gs
		UNION ALL
		SELECT $1::uuid,'outcome_failures','home-oversight-v3',
		       md5('metric-outcome-' || gs::text)::uuid,'MATTER',md5('metric-outcome-target-' || gs::text)::uuid,
		       'Failed outcome ' || gs,'VERIFICATION'
		FROM generate_series(1,1) gs
	`, snapshotID); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO metric_observations(
			tenant_id,legal_entity_id,metric_id,definition_revision,
			source_kind,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,
			value,condition,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,'critical_high_open',$3,
			'OVERSIGHT_SNAPSHOT',$4::uuid,$5,'{}'::jsonb,
			$6,$7,$6,$6,
			1,'ATTENTION','CURRENT','COMPLETE',1,0,0
		)`,
		tenantID, otherEntityID, HomeDefinitionRevision, snapshotID, oversight.ProjectionVersion,
		now, now.Add(-time.Hour),
	); err == nil {
		t.Fatal("metric observation accepted a source snapshot from another legal entity")
	}

	const legacySnapshotID = "8f500000-0000-4000-8000-000000000005"
	if _, err := pool.Exec(ctx, `
		INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4,$5,$5,$5,$6,'{}'::jsonb,0,0,0,'{"counts":{}}'::jsonb
		)`,
		legacySnapshotID, tenantID, entityID, now.Add(-90*24*time.Hour), now.Add(-time.Hour), oversight.ProjectionVersion,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO metric_observations(
			tenant_id,legal_entity_id,metric_id,definition_revision,source_kind,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,value,condition,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,'critical_high_open',$3,'OVERSIGHT_SNAPSHOT',$4::uuid,$5,'{}'::jsonb,
			$6,$7,$6,$6,0,'CLEAR','CURRENT','COMPLETE',0,0,0
		)`,
		tenantID, entityID, HomeDefinitionRevision, legacySnapshotID, oversight.ProjectionVersion, now, now.Add(-90*24*time.Hour),
	); err == nil {
		t.Fatal("legacy snapshot without retained membership accepted a v3 observation")
	}

	repository := NewObservationRepository(pool)
	if err := repository.validateDefinitions(ctx); err != nil {
		t.Fatalf("definition parity: %v", err)
	}

	excludedValue, unknownValue := 2, 3
	sourceSnapshot := oversight.Snapshot{
		SnapshotID: snapshotID, LegalEntityID: entityID, GeneratedAt: now,
		PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now, PostureAsOf: now,
		ProjectionVersion: oversight.ProjectionVersion, Freshness: oversight.FreshnessCurrent,
		SourceHighWater: map[string]time.Time{"matters": now.Add(-time.Minute)},
		Coverage:        oversight.Coverage{Population: 100, Excluded: &excludedValue, Unknown: &unknownValue},
		Counts:          oversight.Counts{CriticalHigh: 7, Overdue: 4, RoutingFailures: 2, OutcomeFailures: 1},
	}
	partial, err := ObservationsFromBundle(tenantID, entityID, snapshotID, sourceSnapshot.SourceHighWater, FromOversight(sourceSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	partialHighWater, err := json.Marshal(partial[0].SourceHighWater)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO metric_observations(
			tenant_id,legal_entity_id,metric_id,definition_revision,
			source_kind,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,
			value,condition,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,$3,$4,$5,$6::uuid,$7,$8::jsonb,
			$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
		)`,
		partial[0].TenantID, partial[0].LegalEntityID, partial[0].MetricID, partial[0].DefinitionRevision,
		partial[0].SourceKind, partial[0].SourceID, partial[0].SourceRevision, partialHighWater,
		partial[0].GeneratedAt, partial[0].PeriodStart, partial[0].PeriodEnd, partial[0].PostureAsOf,
		partial[0].Value, partial[0].Condition, partial[0].Freshness, partial[0].Completeness,
		partial[0].Population, partial[0].Excluded, partial[0].Unknown,
	); err != nil {
		t.Fatal(err)
	}

	maintainer := &ObservationMaintainer{Repository: repository}
	if completed, err := maintainer.Maintain(ctx, now, 10); err != nil || completed < 1 || completed > 10 {
		t.Fatalf("partial repair completed=%d err=%v", completed, err)
	}

	count, err := repository.countObservationsForSource(ctx, snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(homeDefinitions) {
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

	members := NewMembershipRepository(pool)
	firstPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", snapshotID, "critical_high_open", HomeDefinitionRevision, "metric-viewer", "", 3,
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstPage.Count != 7 || len(firstPage.Items) != 3 || firstPage.NextCursor == "" {
		t.Fatalf("critical member page=%#v", firstPage)
	}
	secondPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", snapshotID, "critical_high_open", HomeDefinitionRevision, "metric-viewer", firstPage.NextCursor, 3,
	)
	if err != nil {
		t.Fatal(err)
	}
	if secondPage.Count != 7 || len(secondPage.Items) != 3 || secondPage.Items[0].MemberID == firstPage.Items[0].MemberID {
		t.Fatalf("critical second page=%#v", secondPage)
	}
	routingPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", snapshotID, "routing_gaps", HomeDefinitionRevision, "metric-viewer", "", 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	var sawProgram bool
	for _, item := range routingPage.Items {
		if item.TargetType == "PROGRAM" {
			sawProgram = true
		}
	}
	if routingPage.Count != 2 || !sawProgram {
		t.Fatalf("typed routing members=%#v", routingPage)
	}
	if _, err := members.ListSnapshotMembers(
		ctx, tenantID, otherEntityID, "", snapshotID, "critical_high_open", HomeDefinitionRevision, "metric-viewer", "", 10,
	); !errors.Is(err, ErrMetricMembershipNotFound) {
		t.Fatalf("cross-entity membership error=%v", err)
	}

	if completed, err := maintainer.Maintain(ctx, now.Add(time.Minute), 10); err != nil || completed != 0 {
		t.Fatalf("idempotent maintain completed=%d err=%v", completed, err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE metric_observations SET value=value+1
		WHERE source_id=$1::uuid AND metric_id='critical_high_open'`, snapshotID); err == nil {
		t.Fatal("metric observation update was accepted")
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM metric_observations
		WHERE source_id=$1::uuid AND metric_id='routing_gaps'`, snapshotID); err == nil {
		t.Fatal("metric observation delete was accepted")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE metric_definitions SET label='Changed'
		WHERE metric_id='critical_high_open' AND revision=$1`, HomeDefinitionRevision); err == nil {
		t.Fatal("metric definition mutation was accepted")
	}

	if _, err := pool.Exec(ctx, `DELETE FROM oversight_snapshots WHERE id=$1::uuid`, snapshotID); err != nil {
		t.Fatalf("retained metric observation blocked source snapshot retention cleanup: %v", err)
	}
	if _, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", snapshotID, "critical_high_open", HomeDefinitionRevision, "metric-viewer", "", 10,
	); !errors.Is(err, ErrMetricMembershipNotFound) {
		t.Fatalf("expired source membership remained readable after source cleanup: %v", err)
	}
	count, err = repository.countObservationsForSource(ctx, snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(homeDefinitions) {
		t.Fatalf("retained observation count after source cleanup=%d", count)
	}
}
