//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	catalogConcurrencyTenantID      = "8a111111-1111-7111-8111-111111111111"
	catalogConcurrencyActorID       = "8a222222-2222-7222-8222-222222222222"
	catalogConcurrencySourceAID     = "8a333333-3333-7333-8333-333333333333"
	catalogConcurrencySourceBID     = "8a444444-4444-7444-8444-444444444444"
	catalogConcurrencyConnectionID  = "8a555555-5555-7555-8555-555555555555"
	catalogConcurrencyConnectionRev = "8a666666-6666-7666-8666-666666666666"
	catalogConcurrencyViewID        = "8a777777-7777-7777-8777-777777777777"
	catalogConcurrencyViewRev       = "8a888888-8888-7888-8888-888888888888"
)

func TestPostgresCatalogSerializesStableIdentityAcrossSources(t *testing.T) {
	pool := catalogConcurrencyPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	setupCatalogConcurrencyFixture(t, ctx, pool)

	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback(context.Background())
	if err := insertConcurrencyConnection(ctx, tx1, catalogConcurrencyConnectionRev, catalogConcurrencyConnectionID, catalogConcurrencySourceAID, "PRIMARY", RevisionActive, true, 1, now, nil); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		tx2, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			result <- beginErr
			return
		}
		defer tx2.Rollback(context.Background())
		until := now.Add(time.Hour)
		insertErr := insertConcurrencyConnection(ctx, tx2, "8a999999-9999-7999-8999-999999999999", catalogConcurrencyConnectionID, catalogConcurrencySourceBID, "HISTORICAL", RevisionRetired, false, 2, now, &until)
		if insertErr == nil {
			insertErr = tx2.Commit(ctx)
		}
		result <- insertErr
	}()

	select {
	case err := <-result:
		t.Fatalf("cross-source identity insert completed before the first revision committed: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("database accepted one stable connection identity under two sources")
		}
	case <-ctx.Done():
		t.Fatal("cross-source identity insert did not finish after the first transaction committed")
	}
}

func TestPostgresCatalogSerializesChildCreationAgainstParentRetirement(t *testing.T) {
	pool := catalogConcurrencyPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	setupCatalogConcurrencyFixture(t, ctx, pool)

	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := insertConcurrencyConnection(ctx, pool, catalogConcurrencyConnectionRev, catalogConcurrencyConnectionID, catalogConcurrencySourceAID, "PRIMARY", RevisionActive, true, 1, now, nil); err != nil {
		t.Fatal(err)
	}

	childTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer childTx.Rollback(context.Background())
	if _, err := childTx.Exec(ctx, `
		INSERT INTO source_views(
			revision_id,view_id,tenant_id,source_id,connection_id,connection_version,
			code,name,definition,output_kind,stable_keys,native_schema,schema_fingerprint,
			status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,1,
			'ACTIVE_ACCOUNTS','Active accounts','{"query":"SELECT account_id FROM active_accounts"}'::jsonb,
			'RECORDS','["account_id"]'::jsonb,
			'[{"name":"account_id","native_type":"uuid","nullable":false}]'::jsonb,$6,
			'ACTIVE',true,$7,1,$8::uuid,$7,$7
		)`, catalogConcurrencyViewRev, catalogConcurrencyViewID, catalogConcurrencyTenantID, catalogConcurrencySourceAID, catalogConcurrencyConnectionID, strings.Repeat("a", 64), now, catalogConcurrencyActorID); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		_, updateErr := pool.Exec(ctx, `
			UPDATE source_connections
			   SET status='RETIRED',is_current=false,effective_until=$2,updated_at=$2
			 WHERE revision_id=$1::uuid`, catalogConcurrencyConnectionRev, now.Add(time.Hour))
		result <- updateErr
	}()

	select {
	case err := <-result:
		t.Fatalf("parent retirement completed while a current child insert was uncommitted: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := childTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("database retired a current connection after its current view committed")
		}
	case <-ctx.Done():
		t.Fatal("parent retirement did not finish after the child transaction committed")
	}

	var status RevisionStatus
	var current bool
	if err := pool.QueryRow(ctx, `SELECT status,is_current FROM source_connections WHERE revision_id=$1::uuid`, catalogConcurrencyConnectionRev).Scan(&status, &current); err != nil {
		t.Fatal(err)
	}
	if status != RevisionActive || !current {
		t.Fatalf("failed retirement changed the parent: status=%s current=%v", status, current)
	}
}

func catalogConcurrencyPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func setupCatalogConcurrencyFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, catalogConcurrencyTenantID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, catalogConcurrencyTenantID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-catalog-concurrency','Source catalog concurrency')`, catalogConcurrencyTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-catalog-concurrency-actor','Source catalog concurrency actor')`, catalogConcurrencyActorID, catalogConcurrencyTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version)
		VALUES
			($1::uuid,$3::uuid,'SOURCE-A','Source A','SYSTEM','SYSTEM_OF_RECORD',$4::uuid,15,'UNKNOWN','ACTIVE',1),
			($2::uuid,$3::uuid,'SOURCE-B','Source B','SYSTEM','SYSTEM_OF_RECORD',$4::uuid,15,'UNKNOWN','ACTIVE',1)`, catalogConcurrencySourceAID, catalogConcurrencySourceBID, catalogConcurrencyTenantID, catalogConcurrencyActorID); err != nil {
		t.Fatal(err)
	}
}

func insertConcurrencyConnection(ctx context.Context, executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, revisionID, connectionID, sourceID, code string, status RevisionStatus, current bool, version int64, effectiveFrom time.Time, effectiveUntil *time.Time) error {
	_, err := executor.Exec(ctx, `
		INSERT INTO source_connections(
			revision_id,connection_id,tenant_id,source_id,code,name,adapter_kind,adapter_version,
			secret_ref,definition,declared_capabilities,verified_capabilities,owner_principal_id,
			status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$5,$6,$7,
			'secret://catalog/concurrency','{}'::jsonb,'["INSPECT"]'::jsonb,'["INSPECT"]'::jsonb,$8::uuid,
			$9,$10,$11,$12,$13,$8::uuid,$11,$11
		)`, revisionID, connectionID, catalogConcurrencyTenantID, sourceID, code, AdapterPostgres, PostgresAdapterVersion, catalogConcurrencyActorID, status, current, effectiveFrom, effectiveUntil, version)
	return err
}
