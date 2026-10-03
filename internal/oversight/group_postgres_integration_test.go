//go:build postgres && postgresintegration

package oversight

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresGroupProjectionPreservesChildRevisionsAndAuthorization(t *testing.T) {
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
		tenantID     = "8f400000-0000-4000-8000-000000000001"
		principalID  = "8f400000-0000-4000-8000-000000000002"
		roleID       = "8f400000-0000-4000-8000-000000000003"
		entityA      = "8f400000-0000-4000-8000-000000000010"
		entityB      = "8f400000-0000-4000-8000-000000000011"
		entityC      = "8f400000-0000-4000-8000-000000000012"
		entityD      = "8f400000-0000-4000-8000-000000000013"
		positionA    = "8f400000-0000-4000-8000-000000000020"
		positionB    = "8f400000-0000-4000-8000-000000000021"
		positionC    = "8f400000-0000-4000-8000-000000000022"
		positionD    = "8f400000-0000-4000-8000-000000000023"
		snapshotA    = "8f400000-0000-4000-8000-000000000030"
		snapshotB    = "8f400000-0000-4000-8000-000000000031"
		snapshotC    = "8f400000-0000-4000-8000-000000000032"
		snapshotANew = "8f400000-0000-4000-8000-000000000033"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM group_oversight_runs WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshots WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM position_role_bindings WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM org_positions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM role_templates WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	if _, err = pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'group-oversight-test','Group Oversight Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($4::uuid,$1::uuid,'ENTITY-A','Entity A','NG',$14),
			($5::uuid,$1::uuid,'ENTITY-B','Entity B','GH',$14),
			($6::uuid,$1::uuid,'ENTITY-C','Entity C','ZA',$14),
			($7::uuid,$1::uuid,'ENTITY-D','Entity D','KE',$14);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($2::uuid,$1::uuid,'PERSON','Group reader','ACTIVE',$14);
		INSERT INTO role_templates(id,tenant_id,code,name,capabilities,valid_from)
		VALUES($3::uuid,$1::uuid,'GROUP_OVERSIGHT','Group oversight',ARRAY['OVERSIGHT_READ'],$14);
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from) VALUES
			($8::uuid,$1::uuid,$4::uuid,'A-GLOBAL','A global',$2::uuid,ARRAY[]::text[],$14),
			($9::uuid,$1::uuid,$5::uuid,'B-GLOBAL','B global',$2::uuid,ARRAY[]::text[],$14),
			($10::uuid,$1::uuid,$6::uuid,'C-DEPT','C department',$2::uuid,ARRAY['BANK','RISK'],$14),
			($11::uuid,$1::uuid,$7::uuid,'D-GLOBAL','D global',$2::uuid,ARRAY[]::text[],$14);
		INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,valid_from) VALUES
			($1::uuid,$8::uuid,$3::uuid,$14),
			($1::uuid,$9::uuid,$3::uuid,$14),
			($1::uuid,$10::uuid,$3::uuid,$14),
			($1::uuid,$11::uuid,$3::uuid,$14);
	`, pgx.QueryExecModeSimpleProtocol,
		tenantID, principalID, roleID, entityA, entityB, entityC, entityD,
		positionA, positionB, positionC, positionD, snapshotA, snapshotB, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	insertSnapshot := func(id, entity string, generated time.Time, counts Counts) {
		t.Helper()
		payload, marshalErr := json.Marshal(map[string]any{"counts": counts})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, execErr := pool.Exec(ctx, `
			INSERT INTO oversight_snapshots(
				id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
				projection_version,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$5,$7,'{}'::jsonb,10,0,0,$8::jsonb)
		`, id, tenantID, entity, generated.Add(-90*24*time.Hour), generated, generated.Truncate(5*time.Minute), ProjectionVersion, payload); execErr != nil {
			t.Fatal(execErr)
		}
	}
	insertSnapshot(snapshotA, entityA, now.Add(-2*time.Minute), Counts{CriticalHigh: 2, Overdue: 1})
	insertSnapshot(snapshotB, entityB, now.Add(-30*time.Minute), Counts{CriticalHigh: 3, DueSoon: 4})
	insertSnapshot(snapshotC, entityC, now.Add(-time.Minute), Counts{CriticalHigh: 999})

	repository := NewPostgresRepository(pool)
	projection, err := repository.buildGroupProjection(ctx, tenantID, now)
	if err != nil {
		t.Fatal(err)
	}
	if projection.ActiveChildCount != 4 || projection.CapturedChildCount != 3 || projection.MissingChildCount != 1 || projection.StaleChildCount != 1 {
		t.Fatalf("projection coverage = %#v", projection)
	}
	inserted, err := repository.storeGroupProjection(ctx, projection)
	if err != nil || !inserted {
		t.Fatalf("store group projection inserted=%v err=%v", inserted, err)
	}

	service := NewGroupService(repository, access.NewPostgresResolver(pool))
	service.Now = func() time.Time { return now }
	value, err := service.Get(ctx, identity.Actor{
		TenantID: "group-oversight-test", LegalEntityID: "ENTITY-A", PrincipalID: principalID,
		PermissionCodes: []string{identity.PermissionOversightRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Coverage.AuthorizedChildren != 3 || value.Coverage.IncludedChildren != 2 ||
		value.Coverage.MissingChildren != 1 || value.Coverage.StaleChildren != 1 || value.Coverage.Complete {
		t.Fatalf("authorized group coverage = %#v", value.Coverage)
	}
	if value.Counts.CriticalHigh != 5 || value.Counts.Overdue != 1 || value.Counts.DueSoon != 4 {
		t.Fatalf("authorized group counts = %#v", value.Counts)
	}
	if len(value.Children) != 3 {
		t.Fatalf("authorized group children = %#v", value.Children)
	}
	var sawA bool
	for _, child := range value.Children {
		if child.LegalEntityID == entityC || child.Counts.CriticalHigh == 999 {
			t.Fatalf("department-only sibling leaked: %#v", child)
		}
		if child.LegalEntityID == entityA {
			sawA = true
			if child.ChildSnapshotID != snapshotA {
				t.Fatalf("A contributing revision = %q", child.ChildSnapshotID)
			}
		}
	}
	if !sawA {
		t.Fatal("authorized Entity A missing")
	}

	insertSnapshot(snapshotANew, entityA, now.Add(time.Minute), Counts{CriticalHigh: 9})
	stored, err := repository.LatestGroup(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range stored.Children {
		if child.LegalEntityID == entityA && child.ChildSnapshotID != snapshotA {
			t.Fatalf("stored group revision changed after child refresh: %#v", child)
		}
	}

	next, err := repository.buildGroupProjection(ctx, tenantID, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	inserted, err = repository.storeGroupProjection(ctx, next)
	if err != nil || !inserted {
		t.Fatalf("store next group projection inserted=%v err=%v", inserted, err)
	}
	latest, err := repository.LatestGroup(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	var latestA string
	for _, child := range latest.Children {
		if child.LegalEntityID == entityA {
			latestA = child.ChildSnapshotID
		}
	}
	if latestA != snapshotANew {
		t.Fatalf("latest A contributing revision = %q", latestA)
	}

	if _, err = pool.Exec(ctx, `
		UPDATE group_oversight_child_facts
		SET counts='{}'::jsonb
		WHERE run_id=$1::uuid AND legal_entity_id=$2::uuid
	`, projection.ID, entityA); err == nil {
		t.Fatal("group child projection mutation was accepted")
	}
}
