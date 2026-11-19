//go:build postgres && postgresintegration

package testfixture

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCleanupPreservesOtherEntityAndProductionForeignKeys(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var tenant, a, b string
	if err := pool.QueryRow(ctx, `INSERT INTO tenants(slug,name) VALUES('fixture-cleanup-'||uuidv7()::text,'Fixture cleanup regression') RETURNING id::text`).Scan(&tenant); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := Exec(ctx, pool, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenant); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenant); err != nil {
			t.Error(err)
		}
	}()
	for i, target := range []*string{&a, &b} {
		if err := pool.QueryRow(ctx, `INSERT INTO legal_entities(tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2,'Fixture entity','NG') RETURNING id::text`, tenant, []string{"A", "B"}[i]).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenant, b).Scan(&before); err != nil || before == 0 {
		t.Fatalf("trigger did not install forms: count=%d err=%v", before, err)
	}
	// Real direct deletion must still fail: no production FK or trigger is weakened.
	if _, err := pool.Exec(ctx, `DELETE FROM legal_entities WHERE id=$1::uuid`, b); err == nil {
		t.Fatal("direct entity deletion bypassed retained starter dependencies")
	}
	tag, err := Exec(ctx, pool, `DELETE FROM legal_entities WHERE id=$1::uuid`, a)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("scoped cleanup failed: %v %v", tag, err)
	}
	var after int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenant, b).Scan(&after); err != nil || after != before {
		t.Fatalf("other entity changed: before=%d after=%d err=%v", before, after, err)
	}
	// A failed parent deletion rolls dependency cleanup back in the same transaction.
	if _, err := pool.Exec(ctx, `INSERT INTO monitoring_form_templates(tenant_id,legal_entity_id,code,name,purpose,fields,status,version) VALUES($1::uuid,$2::uuid,'CLEANUP-BLOCKER','Retained custom form','Test cleanup rollback.','[{"id":"q","label":"Question","type":"short_text","required":false}]','DRAFT',1)`, tenant, b); err != nil {
		t.Fatal(err)
	}
	if _, err := Exec(ctx, pool, `DELETE FROM legal_entities WHERE id=$1::uuid`, b); err == nil {
		t.Fatal("custom form should prevent parent deletion")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND starter_catalog_code LIKE 'FRAMEWORK_%'`, tenant, b).Scan(&after); err != nil || after != before {
		t.Fatalf("failed cleanup did not roll back starter deletion: before=%d after=%d err=%v", before, after, err)
	}
	if _, err := Exec(ctx, pool, `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND code='CLEANUP-BLOCKER'`, tenant); err != nil {
		t.Fatal(err)
	}
}
