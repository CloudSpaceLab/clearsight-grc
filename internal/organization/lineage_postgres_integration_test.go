//go:build postgres && postgresintegration

package organization

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationScopeLineageRetainsPreMoveAndPostMoveAncestry(t *testing.T) {
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

	const (
		tenantID  = "8f760000-0000-4000-8000-000000000001"
		entityID  = "8f760000-0000-4000-8000-000000000002"
		parentAID = "8f760000-0000-4000-8000-000000000010"
		parentBID = "8f760000-0000-4000-8000-000000000011"
		childID   = "8f760000-0000-4000-8000-000000000012"
	)
	now := time.Now().UTC().Truncate(time.Second)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}

	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'lineage-history-test','Lineage History Test')`, tenantID)
	mustExec(
		`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		 VALUES($1::uuid,$2::uuid,'LINEAGE-NG','Lineage Nigeria','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour),
	)
	mustExec(
		`INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		 ) VALUES
		 ($1::uuid,$4::uuid,$5::uuid,NULL,'A','Area A','BUSINESS_UNIT',ARRAY['A'],'MANAGED','ACTIVE',$6),
		 ($2::uuid,$4::uuid,$5::uuid,NULL,'B','Area B','BUSINESS_UNIT',ARRAY['B'],'MANAGED','ACTIVE',$6),
		 ($3::uuid,$4::uuid,$5::uuid,$1::uuid,'CHILD','Child','DEPARTMENT',ARRAY['A','CHILD'],'MANAGED','ACTIVE',$6)`,
		parentAID, parentBID, childID, tenantID, entityID, now.Add(-30*time.Minute),
	)

	var initialKind string
	var initialVersion int64
	var initialParent string
	var initialPath []string
	if err := tx.QueryRow(ctx, `
		SELECT event_kind,scope_version,COALESCE(parent_scope_id::text,''),department_path
		FROM organization_scope_lineage_events
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND scope_id=$3::uuid
		ORDER BY effective_at,id
		LIMIT 1`, tenantID, entityID, childID).
		Scan(&initialKind, &initialVersion, &initialParent, &initialPath); err != nil {
		t.Fatal(err)
	}
	if initialKind != "CREATE" || initialVersion != 1 || initialParent != parentAID ||
		!reflect.DeepEqual(initialPath, []string{"A", "CHILD"}) {
		t.Fatalf("initial lineage kind=%s version=%d parent=%s path=%v", initialKind, initialVersion, initialParent, initialPath)
	}

	mustExec(`
		UPDATE organization_scopes
		SET parent_scope_id=$1::uuid,
		    department_path=ARRAY['B','CHILD'],
		    version=version+1,
		    updated_at=clock_timestamp()
		WHERE tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND id=$4::uuid`,
		parentBID, tenantID, entityID, childID,
	)

	rows, err := tx.Query(ctx, `
		SELECT event_kind,scope_version,COALESCE(parent_scope_id::text,''),department_path
		FROM organization_scope_lineage_events
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND scope_id=$3::uuid
		ORDER BY effective_at,id`, tenantID, entityID, childID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type event struct {
		kind    string
		version int64
		parent  string
		path    []string
	}
	events := make([]event, 0, 2)
	for rows.Next() {
		var value event
		if err := rows.Scan(&value.kind, &value.version, &value.parent, &value.path); err != nil {
			t.Fatal(err)
		}
		events = append(events, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("lineage events=%#v", events)
	}
	if events[0].kind != "CREATE" || events[0].parent != parentAID ||
		!reflect.DeepEqual(events[0].path, []string{"A", "CHILD"}) {
		t.Fatalf("pre-move lineage=%#v", events[0])
	}
	if events[1].kind != "MOVE" || events[1].version != 2 || events[1].parent != parentBID ||
		!reflect.DeepEqual(events[1].path, []string{"B", "CHILD"}) {
		t.Fatalf("post-move lineage=%#v", events[1])
	}
}
