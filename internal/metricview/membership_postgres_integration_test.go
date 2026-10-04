//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMetricMembershipRetainsHistoricalCountButRedactsChangedMatterAccess(t *testing.T) {
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

	const tenantID = "8f610000-0000-4000-8000-000000000001"
	const entityID = "8f610000-0000-4000-8000-000000000002"
	const snapshotID = "8f610000-0000-4000-8000-000000000003"
	const matterID = "8f610000-0000-4000-8000-000000000004"
	const memberID = "8f610000-0000-4000-8000-000000000005"
	const principalA = "8f610000-0000-4000-8000-000000000006"
	const principalB = "8f610000-0000-4000-8000-000000000007"
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshot_metric_memberships WHERE oversight_snapshot_id=$1::uuid`, snapshotID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshot_metric_membership_sets WHERE oversight_snapshot_id=$1::uuid`, snapshotID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshots WHERE id=$1::uuid`, snapshotID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM matters WHERE id=$1::uuid`, matterID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE id IN ($1::uuid,$2::uuid)`, principalA, principalB)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE id=$1::uuid`, entityID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := pool.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'metric-membership-access','Metric Membership Access')`, tenantID)
	mustExec(
		`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		 VALUES($1::uuid,$2::uuid,'MEM-NG','Membership Nigeria','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour),
	)
	mustExec(
		`INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		 ($1::uuid,$3::uuid,'PERSON','Viewer A','ACTIVE',$4),
		 ($2::uuid,$3::uuid,'PERSON','Viewer B','ACTIVE',$4)`,
		principalA, principalB, tenantID, now.Add(-time.Hour),
	)
	mustExec(
		`INSERT INTO matters(
			id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,
			created_at,updated_at,version
		 ) VALUES(
			$1::uuid,$2::uuid,$3::uuid,'MEM-001','CONTROL_GAP','ASSESSMENT',4,
			'Retained restricted issue','Access can change after the snapshot',
			jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($4::text)),
			$5,$5,1
		 )`,
		matterID, tenantID, entityID, principalA, now.Add(-time.Hour),
	)
	mustExec(
		`INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,metric_membership_revision,source_high_water,
			coverage_population,coverage_excluded,coverage_unknown,payload
		 ) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4,$5,$5,$5,
			'oversight-v5','home-oversight-v3','{}'::jsonb,1,0,0,
			'{"counts":{"critical_high":1}}'::jsonb
		 )`,
		snapshotID, tenantID, entityID, now.Add(-time.Hour), now,
	)
	mustExec(
		`INSERT INTO oversight_snapshot_metric_membership_sets(
			oversight_snapshot_id,tenant_id,legal_entity_id,definition_revision
		 ) VALUES($1::uuid,$2::uuid,$3::uuid,'home-oversight-v3')`,
		snapshotID, tenantID, entityID,
	)
	mustExec(
		`INSERT INTO oversight_snapshot_metric_memberships(
			oversight_snapshot_id,metric_id,definition_revision,member_id,target_type,target_id,target_title,state
		 ) VALUES(
			$1::uuid,'critical_high_open','home-oversight-v3',$2::uuid,'MATTER',$3::uuid,
			'Retained restricted issue','ASSESSMENT'
		 )`,
		snapshotID, memberID, matterID,
	)

	repository := NewMembershipRepository(pool)
	page, err := repository.ListSnapshotMembers(
		ctx, tenantID, entityID, snapshotID, "critical_high_open", HomeDefinitionRevision, principalA, "", 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if page.Count != 1 || len(page.Items) != 1 || !page.Items[0].Accessible ||
		page.Items[0].TargetID != matterID || page.Items[0].TargetTitle != "Retained restricted issue" {
		t.Fatalf("initial retained member=%#v", page)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE matters
		SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)),
		    updated_at=$3,
		    version=version+1
		WHERE id=$1::uuid
	`, matterID, principalB, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	redacted, err := repository.ListSnapshotMembers(
		ctx, tenantID, entityID, snapshotID, "critical_high_open", HomeDefinitionRevision, principalA, "", 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if redacted.Count != 1 || len(redacted.Items) != 1 || redacted.Items[0].Accessible ||
		redacted.Items[0].TargetID != "" || redacted.Items[0].TargetTitle != "Record access changed" ||
		redacted.Items[0].State != "ACCESS_CHANGED" {
		t.Fatalf("redacted retained member=%#v", redacted)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE matters
		SET owner_principal_id=$2::uuid,
		    updated_at=$3,
		    version=version+1
		WHERE id=$1::uuid
	`, matterID, principalA, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}

	responsible, err := repository.ListSnapshotMembers(
		ctx, tenantID, entityID, snapshotID, "critical_high_open", HomeDefinitionRevision, principalA, "", 10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if responsible.Count != 1 || len(responsible.Items) != 1 || !responsible.Items[0].Accessible ||
		responsible.Items[0].TargetID != matterID {
		t.Fatalf("recorded responsibility did not restore drill access: %#v", responsible)
	}
}
