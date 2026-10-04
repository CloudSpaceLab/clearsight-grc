//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestExactMetricDrillIsScopedPagedAndRetained(t *testing.T) {
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

	newID := func() string {
		t.Helper()
		var value string
		if err := pool.QueryRow(ctx, `SELECT uuidv7()::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	tenantID, entityA, entityB := newID(), newID(), newID()
	snapshotID := newID()
	memberA, memberB, memberC := newID(), newID(), newID()
	now := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Exact Drill Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($3::uuid,$1::uuid,'EXACT-A','Exact A','NG',$5),
			($4::uuid,$1::uuid,'EXACT-B','Exact B','GH',$5);
		INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,metric_membership_version,source_high_water,
			coverage_population,coverage_excluded,coverage_unknown,payload
		) VALUES(
			$6::uuid,$1::uuid,$3::uuid,$7,$8,$8,$8,
			$9,$10,'{}'::jsonb,3,0,0,'{"counts":{"critical_high":3}}'::jsonb
		);
		INSERT INTO metric_observations(
			tenant_id,legal_entity_id,metric_id,definition_revision,
			source_kind,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,
			value,condition,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$3::uuid,'critical_high_open',$11,
			'OVERSIGHT_SNAPSHOT',$6::uuid,$9,'{}'::jsonb,
			$8,$7,$8,$8,
			3,'ATTENTION','CURRENT','COMPLETE',3,0,0
		);
		INSERT INTO oversight_metric_members(
			snapshot_id,tenant_id,legal_entity_id,source_generated_at,
			metric_id,member_type,member_id,subject_type,subject_id,
			reference,title,state,priority
		) VALUES
			($6::uuid,$1::uuid,$3::uuid,$8,'critical_high_open','MATTER',$12::uuid,'MATTER',$12::uuid,'MAT-A','Alpha issue','TRIAGE',5),
			($6::uuid,$1::uuid,$3::uuid,$8,'critical_high_open','MATTER',$13::uuid,'MATTER',$13::uuid,'MAT-B','Beta issue','ASSESSMENT',4),
			($6::uuid,$1::uuid,$3::uuid,$8,'critical_high_open','MATTER',$14::uuid,'MATTER',$14::uuid,'MAT-C','Gamma issue','RESPONSE',5)
	`,
		tenantID, "exact-drill-"+tenantID, entityA, entityB, now.Add(-time.Hour),
		snapshotID, now.Add(-90*24*time.Hour), now,
		oversight.ProjectionVersion, oversight.MetricMembershipVersion, HomeDefinitionRevision,
		memberA, memberB, memberC,
	); err != nil {
		t.Fatal(err)
	}

	repository := NewObservationRepository(pool)
	first, err := repository.ListExactDrill(ctx, DrillQuery{
		TenantID: "exact-drill-" + tenantID, LegalEntityID: "EXACT-A",
		SourceID: snapshotID, MetricID: "critical_high_open", Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 3 || len(first.Items) != 2 || first.NextCursor == "" ||
		first.DefinitionRevision != HomeDefinitionRevision {
		t.Fatalf("first page=%#v", first)
	}
	second, err := repository.ListExactDrill(ctx, DrillQuery{
		TenantID: tenantID, LegalEntityID: entityA,
		SourceID: snapshotID, MetricID: "critical_high_open", Cursor: first.NextCursor, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Total != 3 || len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("second page=%#v", second)
	}

	if _, err := repository.ListExactDrill(ctx, DrillQuery{
		TenantID: tenantID, LegalEntityID: entityB,
		SourceID: snapshotID, MetricID: "critical_high_open",
	}); !errors.Is(err, ErrDrillNotFound) {
		t.Fatalf("cross-entity drill error=%v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM oversight_snapshots WHERE id=$1::uuid`, snapshotID); err != nil {
		t.Fatalf("source retention cleanup failed: %v", err)
	}
	retained, err := repository.ListExactDrill(ctx, DrillQuery{
		TenantID: tenantID, LegalEntityID: entityA,
		SourceID: snapshotID, MetricID: "critical_high_open",
	})
	if err != nil || retained.Total != 3 || len(retained.Items) != 3 {
		t.Fatalf("retained drill=%#v err=%v", retained, err)
	}

	if _, err := pool.Exec(ctx, `
		DELETE FROM oversight_metric_members
		WHERE snapshot_id=$1::uuid AND metric_id='critical_high_open' AND member_id=$2::uuid
	`, snapshotID, memberC); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ListExactDrill(ctx, DrillQuery{
		TenantID: tenantID, LegalEntityID: entityA,
		SourceID: snapshotID, MetricID: "critical_high_open",
	}); !errors.Is(err, ErrDrillMismatch) {
		t.Fatalf("corrupt membership error=%v", err)
	}
}
