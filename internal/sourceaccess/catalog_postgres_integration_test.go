//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCatalogPersistsVersionedSourceHierarchy(t *testing.T) {
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
	cleanupCatalogFixture(ctx, pool)
	t.Cleanup(func() { cleanupCatalogFixture(context.Background(), pool) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-catalog-test','Source catalog test')`, catalogTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-catalog-actor','Source catalog actor')`, catalogActorID, catalogTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'CORE-RISK','Core risk source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, catalogSourceID, catalogTenantID, catalogActorID); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresCatalogRepository(pool)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection, err := repository.CreateConnectionRevision(ctx, catalogConnectionRevision(now))
	if err != nil {
		t.Fatal(err)
	}
	view, err := repository.CreateViewRevision(ctx, catalogViewRevision(now))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := repository.CreateBindingRevision(ctx, catalogBindingRevision(now))
	if err != nil {
		t.Fatal(err)
	}

	loadedConnection, err := repository.CurrentConnection(ctx, catalogTenantID, catalogConnectionID)
	if err != nil {
		t.Fatal(err)
	}
	loadedView, err := repository.CurrentView(ctx, catalogTenantID, catalogViewID)
	if err != nil {
		t.Fatal(err)
	}
	loadedBinding, err := repository.CurrentBinding(ctx, catalogTenantID, catalogBindingID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedConnection.RevisionID != connection.RevisionID || loadedView.ConnectionVersion != connection.Version || loadedBinding.ViewVersion != view.Version || loadedBinding.Limits != binding.Limits {
		t.Fatalf("catalog hierarchy changed during persistence: connection=%#v view=%#v binding=%#v", loadedConnection, loadedView, loadedBinding)
	}
	if _, err := loadedConnection.Contract(); err != nil {
		t.Fatal(err)
	}
	if _, err := loadedView.Contract(loadedConnection); err != nil {
		t.Fatal(err)
	}
	if _, err := loadedBinding.Contract(loadedView); err != nil {
		t.Fatal(err)
	}

	connections, err := repository.ListCurrentConnections(ctx, catalogTenantID, catalogSourceID, 1)
	if err != nil || len(connections) != 1 {
		t.Fatalf("connections=%#v err=%v", connections, err)
	}
	views, err := repository.ListCurrentViews(ctx, catalogTenantID, catalogConnectionID, 1)
	if err != nil || len(views) != 1 {
		t.Fatalf("views=%#v err=%v", views, err)
	}
	bindings, err := repository.ListCurrentBindings(ctx, catalogTenantID, catalogViewID, 1)
	if err != nil || len(bindings) != 1 {
		t.Fatalf("bindings=%#v err=%v", bindings, err)
	}

	duplicateCode := catalogConnectionRevision(now)
	duplicateCode.RevisionID = "41111111-1111-7111-8111-111111111111"
	duplicateCode.ConnectionID = "42222222-2222-7222-8222-222222222222"
	if _, err := repository.CreateConnectionRevision(ctx, duplicateCode); !errors.Is(err, ErrCatalogConflict) {
		t.Fatalf("duplicate current connection code should fail, got %v", err)
	}

	historical := catalogConnectionRevision(now)
	historical.RevisionID = "43333333-3333-7333-8333-333333333333"
	historical.Version = 2
	historical.Status = RevisionRetired
	historical.IsCurrent = false
	historical.EffectiveUntil = timePointer(now.Add(time.Hour))
	if _, err := repository.CreateConnectionRevision(ctx, historical); err != nil {
		t.Fatal(err)
	}
	currentOverHistorical := catalogViewRevision(now)
	currentOverHistorical.RevisionID = "44444444-4444-7444-8444-444444444444"
	currentOverHistorical.ViewID = "45555555-5555-7555-8555-555555555555"
	currentOverHistorical.ConnectionVersion = 2
	if _, err := repository.CreateViewRevision(ctx, currentOverHistorical); !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("current view over historical connection should fail, got %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE source_connections SET status='RETIRED',is_current=false,effective_until=$2 WHERE revision_id=$1::uuid`, connection.RevisionID, now.Add(time.Hour)); err == nil {
		t.Fatal("current connection retired while a current view still referenced it")
	}
	if _, err := pool.Exec(ctx, `UPDATE source_views SET status='RETIRED',is_current=false,effective_until=$2 WHERE revision_id=$1::uuid`, view.RevisionID, now.Add(time.Hour)); err == nil {
		t.Fatal("current view retired while a current binding still referenced it")
	}

	otherSourceID := "46666666-6666-7666-8666-666666666666"
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'OTHER','Other source','SYSTEM','SYSTEM_OF_RECORD',15,'UNKNOWN','ACTIVE',1)`, otherSourceID, catalogTenantID); err != nil {
		t.Fatal(err)
	}
	crossSource := historical
	crossSource.RevisionID = "47777777-7777-7777-8777-777777777777"
	crossSource.SourceID = otherSourceID
	crossSource.Version = 3
	if _, err := repository.CreateConnectionRevision(ctx, crossSource); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("stable connection moved across sources, got %v", err)
	}
}

func cleanupCatalogFixture(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, catalogTenantID)
}
