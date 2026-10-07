//go:build postgres && postgresintegration

package organization

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationScopeLineagePreservesAncestryAcrossMove(t *testing.T) {
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
	defer func() { _ = tx.Rollback(ctx) }()

	const (
		tenantID = "8f760000-0000-4000-8000-000000000001"
		entityID = "8f760000-0000-4000-8000-000000000002"
		rootAID  = "8f760000-0000-4000-8000-000000000010"
		rootBID  = "8f760000-0000-4000-8000-000000000011"
		childID  = "8f760000-0000-4000-8000-000000000012"
		leafID   = "8f760000-0000-4000-8000-000000000013"
	)
	now := time.Now().UTC().Truncate(time.Second)
	mustLineageExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}

	mustLineageExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'lineage-test','Lineage Test')`, tenantID)
	mustLineageExec(`
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'LINEAGE-NG','Lineage Nigeria','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour))
	mustLineageExec(`
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
		  ($1::uuid,$5::uuid,$6::uuid,NULL,'A','Division A','BUSINESS_UNIT',ARRAY['A'],'MANAGED','ACTIVE',$7),
		  ($2::uuid,$5::uuid,$6::uuid,NULL,'B','Division B','BUSINESS_UNIT',ARRAY['B'],'MANAGED','ACTIVE',$7),
		  ($3::uuid,$5::uuid,$6::uuid,$1::uuid,'CHILD','Child','DEPARTMENT',ARRAY['A','CHILD'],'MANAGED','ACTIVE',$7),
		  ($4::uuid,$5::uuid,$6::uuid,$3::uuid,'LEAF','Leaf','BRANCH',ARRAY['A','CHILD','LEAF'],'MANAGED','ACTIVE',$7)`,
		rootAID, rootBID, childID, leafID, tenantID, entityID, now.Add(-time.Hour))

	var createPath []string
	var createKind string
	if err := tx.QueryRow(ctx, `
		SELECT department_path,event_kind
		FROM organization_scope_lineage_events
		WHERE scope_id=$1::uuid
		ORDER BY effective_at,id
		LIMIT 1`, leafID).Scan(&createPath, &createKind); err != nil {
		t.Fatal(err)
	}
	if createKind != "CREATE" || strings.Join(createPath, "/") != "A/CHILD/LEAF" {
		t.Fatalf("create lineage kind=%q path=%v", createKind, createPath)
	}

	mustLineageExec(`
		UPDATE organization_scopes
		SET department_path=ARRAY['B','CHILD']::text[] || department_path[3:cardinality(department_path)],
		    version=version+1,updated_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		  AND cardinality(department_path)>=2
		  AND department_path[1:2]=ARRAY['A','CHILD']::text[]`,
		tenantID, entityID)
	mustLineageExec(`
		UPDATE organization_scopes
		SET parent_scope_id=$2::uuid,updated_at=clock_timestamp()
		WHERE id=$1::uuid`, childID, rootBID)

	var leafEvents int
	var oldPathPreserved, newPathCaptured bool
	if err := tx.QueryRow(ctx, `
		SELECT count(*),
		       bool_or(department_path=ARRAY['A','CHILD','LEAF']::text[]),
		       bool_or(department_path=ARRAY['B','CHILD','LEAF']::text[])
		FROM organization_scope_lineage_events
		WHERE scope_id=$1::uuid`, leafID).
		Scan(&leafEvents, &oldPathPreserved, &newPathCaptured); err != nil {
		t.Fatal(err)
	}
	if leafEvents != 2 || !oldPathPreserved || !newPathCaptured {
		t.Fatalf("leaf lineage events=%d old=%v new=%v", leafEvents, oldPathPreserved, newPathCaptured)
	}

	var latestParent string
	var latestPath []string
	var latestKind string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(parent_scope_id::text,''),department_path,event_kind
		FROM organization_scope_lineage_events
		WHERE scope_id=$1::uuid
		ORDER BY effective_at DESC,id DESC
		LIMIT 1`, childID).Scan(&latestParent, &latestPath, &latestKind); err != nil {
		t.Fatal(err)
	}
	if latestParent != rootBID || strings.Join(latestPath, "/") != "B/CHILD" || latestKind != "MOVE" {
		t.Fatalf("latest child lineage parent=%s path=%v kind=%s", latestParent, latestPath, latestKind)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE organization_scope_lineage_events
		SET department_path=ARRAY['TAMPERED']::text[]
		WHERE scope_id=$1::uuid`, leafID); err == nil || !strings.Contains(err.Error(), "lineage history is immutable") {
		t.Fatalf("lineage mutation err=%v", err)
	}
}
