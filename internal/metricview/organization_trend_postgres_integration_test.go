//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationTrendUsesHistoricalLineageAcrossScopeMove(t *testing.T) {
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

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	tenantID := mustOrganizationTrendID(t)
	entityID := mustOrganizationTrendID(t)
	parentAID := mustOrganizationTrendID(t)
	parentBID := mustOrganizationTrendID(t)
	childID := mustOrganizationTrendID(t)
	sourceDay1 := mustOrganizationTrendID(t)
	sourceDay2 := mustOrganizationTrendID(t)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}

	now := time.Now().UTC()
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Organization Trend')`,
		tenantID, "organization-trend-"+tenantID[len(tenantID)-8:])
	mustExec(`
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'Organization Trend Nigeria','NG',$4)`,
		entityID, tenantID, "OT-"+entityID[len(entityID)-8:], now.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
		  ($1::uuid,$4::uuid,$5::uuid,NULL,'A','Area A','BUSINESS_UNIT',ARRAY['A'],'MANAGED','ACTIVE',$6),
		  ($2::uuid,$4::uuid,$5::uuid,NULL,'B','Area B','BUSINESS_UNIT',ARRAY['B'],'MANAGED','ACTIVE',$6),
		  ($3::uuid,$4::uuid,$5::uuid,$1::uuid,'CHILD','Child','DEPARTMENT',ARRAY['A','CHILD'],'MANAGED','ACTIVE',$6)`,
		parentAID, parentBID, childID, tenantID, entityID, now.Add(-30*24*time.Hour))

	sourceAt1 := time.Now().UTC()
	var preMoveLineageID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM organization_scope_lineage_events
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND scope_id=$3::uuid
		ORDER BY effective_at DESC,id DESC
		LIMIT 1`, tenantID, entityID, childID).Scan(&preMoveLineageID); err != nil {
		t.Fatal(err)
	}

	day1 := time.Date(sourceAt1.Year(), sourceAt1.Month(), sourceAt1.Day(), 0, 0, 0, 0, time.UTC)
	mustExec(`
		INSERT INTO organization_metric_daily_sources(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			source_id,source_revision,source_generated_at,source_high_water
		) VALUES($1::uuid,$2::uuid,$3,$4::date,$5::uuid,$6,$7,'{}'::jsonb)`,
		tenantID, entityID, DomainDefinitionRevision, day1, sourceDay1, DomainSourceRevision, sourceAt1)
	mustExec(`
		INSERT INTO organization_metric_daily_buckets(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			metric_id,organization_scope_id,lineage_event_id,value
		) VALUES($1::uuid,$2::uuid,$3,$4::date,'risks_outside_appetite',$5::uuid,$6::uuid,3)`,
		tenantID, entityID, DomainDefinitionRevision, day1, childID, preMoveLineageID)

	time.Sleep(5 * time.Millisecond)
	mustExec(`
		UPDATE organization_scopes
		SET parent_scope_id=$1::uuid,
		    department_path=ARRAY['B','CHILD'],
		    version=version+1,
		    updated_at=clock_timestamp()
		WHERE tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND id=$4::uuid`,
		parentBID, tenantID, entityID, childID)

	var postMoveLineageID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM organization_scope_lineage_events
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND scope_id=$3::uuid
		ORDER BY effective_at DESC,id DESC
		LIMIT 1`, tenantID, entityID, childID).Scan(&postMoveLineageID); err != nil {
		t.Fatal(err)
	}
	if postMoveLineageID == preMoveLineageID {
		t.Fatal("scope move did not create a new lineage event")
	}

	sourceAt2 := sourceAt1.Add(24 * time.Hour)
	day2 := day1.Add(24 * time.Hour)
	mustExec(`
		INSERT INTO organization_metric_daily_sources(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			source_id,source_revision,source_generated_at,source_high_water
		) VALUES($1::uuid,$2::uuid,$3,$4::date,$5::uuid,$6,$7,'{}'::jsonb)`,
		tenantID, entityID, DomainDefinitionRevision, day2, sourceDay2, DomainSourceRevision, sourceAt2)
	mustExec(`
		INSERT INTO organization_metric_daily_buckets(
			tenant_id,legal_entity_id,definition_revision,bucket_date,
			metric_id,organization_scope_id,lineage_event_id,value
		) VALUES($1::uuid,$2::uuid,$3,$4::date,'risks_outside_appetite',$5::uuid,$6::uuid,5)`,
		tenantID, entityID, DomainDefinitionRevision, day2, childID, postMoveLineageID)

	start := day1
	end := day2.Add(24*time.Hour - time.Nanosecond)
	areaA, err := organizationTrendPoints(
		ctx, tx, tenantID, entityID, parentAID, "risks_outside_appetite", start, end,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(areaA) != 2 || areaA[0].Value != 3 || areaA[1].Value != 0 {
		t.Fatalf("Area A trend=%#v", areaA)
	}

	areaB, err := organizationTrendPoints(
		ctx, tx, tenantID, entityID, parentBID, "risks_outside_appetite", start, end,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(areaB) != 2 || areaB[0].Value != 0 || areaB[1].Value != 5 {
		t.Fatalf("Area B trend=%#v", areaB)
	}
}

func mustOrganizationTrendID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
