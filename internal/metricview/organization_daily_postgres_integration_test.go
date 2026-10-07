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

func TestFinalizeOrganizationMetricDayRetainsDirectAttributionAndLineage(t *testing.T) {
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

	tenantID := mustOrganizationDailyID(t)
	entityID := mustOrganizationDailyID(t)
	parentID := mustOrganizationDailyID(t)
	childID := mustOrganizationDailyID(t)
	sourceID := mustOrganizationDailyID(t)
	outsideMemberID := mustOrganizationDailyID(t)
	outsideTargetID := mustOrganizationDailyID(t)
	unattributedMemberID := mustOrganizationDailyID(t)
	unattributedTargetID := mustOrganizationDailyID(t)
	indicatorMemberID := mustOrganizationDailyID(t)
	indicatorTargetID := mustOrganizationDailyID(t)

	sourceAt := time.Now().UTC().Truncate(DomainSnapshotInterval).Add(11 * time.Minute)
	bucketDate := time.Date(sourceAt.Year(), sourceAt.Month(), sourceAt.Day(), 0, 0, 0, 0, time.UTC)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}

	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Organization Daily History')`,
		tenantID, "organization-daily-"+tenantID[len(tenantID)-8:])
	mustExec(`
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'Organization Daily Nigeria','NG',$4)`,
		entityID, tenantID, "OD-"+entityID[len(entityID)-8:], sourceAt.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
		  ($1::uuid,$3::uuid,$4::uuid,NULL,'TECH','Technology','BUSINESS_UNIT',ARRAY['TECH'],'MANAGED','ACTIVE',$5),
		  ($2::uuid,$3::uuid,$4::uuid,$1::uuid,'INFRA','Infrastructure','DEPARTMENT',ARRAY['TECH','INFRA'],'MANAGED','ACTIVE',$5)`,
		parentID, childID, tenantID, entityID, sourceAt.Add(-30*24*time.Hour))

	mustExec(`
		INSERT INTO domain_metric_snapshots(
			id,tenant_id,legal_entity_id,definition_revision,source_revision,source_high_water,bucket_start,generated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'{}'::jsonb,$6,$7)`,
		sourceID, tenantID, entityID, DomainDefinitionRevision, DomainSourceRevision,
		sourceAt.Truncate(DomainSnapshotInterval), sourceAt)

	mustExec(`
		INSERT INTO domain_metric_snapshot_memberships(
			source_id,metric_id,definition_revision,member_id,target_type,target_id,organization_scope_id,target_title,state
		) VALUES
		  ($1::uuid,'risks_outside_appetite',$2,$3::uuid,'RISK',$4::uuid,$5::uuid,'Infrastructure exposure','BREACHED'),
		  ($1::uuid,'risks_outside_appetite',$2,$6::uuid,'RISK',$7::uuid,NULL,'Unattributed exposure','BREACHED'),
		  ($1::uuid,'indicator_breaches',$2,$8::uuid,'RISK',$9::uuid,$5::uuid,'Infrastructure KRI','HIGH')`,
		sourceID, DomainDefinitionRevision,
		outsideMemberID, outsideTargetID, childID,
		unattributedMemberID, unattributedTargetID,
		indicatorMemberID, indicatorTargetID)

	candidate := organizationDailySourceCandidate{
		TenantID: tenantID, LegalEntityID: entityID,
		DefinitionRevision: DomainDefinitionRevision,
		BucketDate: bucketDate,
		SourceID: sourceID, SourceRevision: DomainSourceRevision,
		SourceGeneratedAt: sourceAt,
		SourceHighWater: []byte(`{}`),
	}
	inserted, err := finalizeOrganizationMetricDay(ctx, tx, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("expected first daily organization source to be finalized")
	}

	type bucket struct {
		metricID  string
		scopeID   string
		lineageID string
		value     int64
	}
	rows, err := tx.Query(ctx, `
		SELECT metric_id,
		       COALESCE(organization_scope_id::text,''),
		       COALESCE(lineage_event_id::text,''),
		       value
		FROM organization_metric_daily_buckets
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND bucket_date=$3::date
		ORDER BY metric_id,organization_scope_id NULLS FIRST`,
		tenantID, entityID, bucketDate)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	values := make([]bucket, 0, 3)
	for rows.Next() {
		var value bucket
		if err := rows.Scan(&value.metricID, &value.scopeID, &value.lineageID, &value.value); err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 {
		t.Fatalf("daily buckets=%#v", values)
	}

	var sawAttributedOutside, sawUnattributedOutside, sawIndicator bool
	for _, value := range values {
		switch {
		case value.metricID == "risks_outside_appetite" && value.scopeID == childID:
			sawAttributedOutside = value.value == 1 && value.lineageID != ""
			if value.lineageID != "" {
				var path []string
				if err := tx.QueryRow(ctx, `
					SELECT department_path
					FROM organization_scope_lineage_events
					WHERE id=$1::uuid`, value.lineageID).Scan(&path); err != nil {
					t.Fatal(err)
				}
				if len(path) != 2 || path[0] != "TECH" || path[1] != "INFRA" {
					t.Fatalf("captured lineage path=%v", path)
				}
			}
		case value.metricID == "risks_outside_appetite" && value.scopeID == "":
			sawUnattributedOutside = value.value == 1 && value.lineageID == ""
		case value.metricID == "indicator_breaches" && value.scopeID == childID:
			sawIndicator = value.value == 1 && value.lineageID != ""
		}
	}
	if !sawAttributedOutside || !sawUnattributedOutside || !sawIndicator {
		t.Fatalf("unexpected daily buckets=%#v", values)
	}

	inserted, err = finalizeOrganizationMetricDay(ctx, tx, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("daily organization source must be idempotent")
	}

	var sourceCount, bucketCount int
	if err := tx.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM organization_metric_daily_sources
		   WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND bucket_date=$3::date),
		  (SELECT count(*) FROM organization_metric_daily_buckets
		   WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND bucket_date=$3::date)`,
		tenantID, entityID, bucketDate).Scan(&sourceCount, &bucketCount); err != nil {
		t.Fatal(err)
	}
	if sourceCount != 1 || bucketCount != 3 {
		t.Fatalf("daily source count=%d bucket count=%d", sourceCount, bucketCount)
	}
}

func mustOrganizationDailyID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
