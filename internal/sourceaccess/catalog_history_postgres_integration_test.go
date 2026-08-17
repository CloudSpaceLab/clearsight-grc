//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	historyTenantID     = "6b111111-1111-7111-8111-111111111111"
	historyActorID      = "6b222222-2222-7222-8222-222222222222"
	historySourceID     = "6b333333-3333-7333-8333-333333333333"
	historyConnectionID = "6b444444-4444-7444-8444-444444444444"
	historyViewID       = "6b555555-5555-7555-8555-555555555555"
	historyBindingID    = "6b666666-6666-7666-8666-666666666666"
)

func TestPostgresCatalogListsBoundedRevisionHistoryLatestFirst(t *testing.T) {
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
	cleanupHistoryFixture(ctx, pool)
	t.Cleanup(func() { cleanupHistoryFixture(context.Background(), pool) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-catalog-history','Source catalog history')`, historyTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-catalog-history-actor','Source catalog history actor')`, historyActorID, historyTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'HISTORY-SOURCE','History source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, historySourceID, historyTenantID, historyActorID); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresCatalogRepository(pool)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	connection := catalogConnectionRevision(now)
	connection.RevisionID = "6b777777-7777-7777-8777-777777777777"
	connection.ConnectionID = historyConnectionID
	connection.TenantID = historyTenantID
	connection.SourceID = historySourceID
	connection.OwnerPrincipalID = historyActorID
	connection.CreatedBy = historyActorID
	view := catalogViewRevision(now)
	view.RevisionID = "6b888888-8888-7888-8888-888888888888"
	view.ViewID = historyViewID
	view.TenantID = historyTenantID
	view.SourceID = historySourceID
	view.ConnectionID = historyConnectionID
	view.CreatedBy = historyActorID
	binding := catalogBindingRevision(now)
	binding.RevisionID = "6b999999-9999-7999-8999-999999999999"
	binding.BindingID = historyBindingID
	binding.TenantID = historyTenantID
	binding.SourceID = historySourceID
	binding.ViewID = historyViewID
	binding.CreatedBy = historyActorID
	if _, err := repository.CreateConnectionRevision(ctx, connection); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateViewRevision(ctx, view); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateBindingRevision(ctx, binding); err != nil {
		t.Fatal(err)
	}

	retiredAt := now.Add(time.Hour)
	connectionHistory := connection
	connectionHistory.RevisionID = "6c111111-1111-7111-8111-111111111111"
	connectionHistory.Version = 2
	connectionHistory.Status = RevisionRetired
	connectionHistory.IsCurrent = false
	connectionHistory.EffectiveUntil = timePointer(retiredAt)
	connectionHistory.UpdatedAt = retiredAt
	if _, err := repository.CreateConnectionRevision(ctx, connectionHistory); err != nil {
		t.Fatal(err)
	}
	viewHistory := view
	viewHistory.RevisionID = "6c222222-2222-7222-8222-222222222222"
	viewHistory.Version = 2
	viewHistory.Status = RevisionRetired
	viewHistory.IsCurrent = false
	viewHistory.EffectiveUntil = timePointer(retiredAt)
	viewHistory.UpdatedAt = retiredAt
	if _, err := repository.CreateViewRevision(ctx, viewHistory); err != nil {
		t.Fatal(err)
	}
	bindingHistory := binding
	bindingHistory.RevisionID = "6c333333-3333-7333-8333-333333333333"
	bindingHistory.Version = 2
	bindingHistory.Status = RevisionRetired
	bindingHistory.IsCurrent = false
	bindingHistory.EffectiveUntil = timePointer(retiredAt)
	bindingHistory.UpdatedAt = retiredAt
	if _, err := repository.CreateBindingRevision(ctx, bindingHistory); err != nil {
		t.Fatal(err)
	}

	connections, err := repository.ListConnectionRevisions(ctx, historyTenantID, historySourceID, 20)
	if err != nil || len(connections) != 2 || connections[0].Version != 2 || connections[1].Version != 1 {
		t.Fatalf("connection history=%#v err=%v", connections, err)
	}
	views, err := repository.ListViewRevisions(ctx, historyTenantID, historyConnectionID, 20)
	if err != nil || len(views) != 2 || views[0].Version != 2 || views[1].Version != 1 {
		t.Fatalf("view history=%#v err=%v", views, err)
	}
	bindings, err := repository.ListBindingRevisions(ctx, historyTenantID, historyViewID, 20)
	if err != nil || len(bindings) != 2 || bindings[0].Version != 2 || bindings[1].Version != 1 {
		t.Fatalf("binding history=%#v err=%v", bindings, err)
	}

	currentConnections, err := repository.ListCurrentConnections(ctx, historyTenantID, historySourceID, 20)
	if err != nil || len(currentConnections) != 1 || currentConnections[0].Version != 1 {
		t.Fatalf("current connections=%#v err=%v", currentConnections, err)
	}
	currentViews, err := repository.ListCurrentViews(ctx, historyTenantID, historyConnectionID, 20)
	if err != nil || len(currentViews) != 1 || currentViews[0].Version != 1 {
		t.Fatalf("current views=%#v err=%v", currentViews, err)
	}
	currentBindings, err := repository.ListCurrentBindings(ctx, historyTenantID, historyViewID, 20)
	if err != nil || len(currentBindings) != 1 || currentBindings[0].Version != 1 {
		t.Fatalf("current bindings=%#v err=%v", currentBindings, err)
	}
}

func cleanupHistoryFixture(ctx context.Context, pool *pgxpool.Pool) {
	for _, statement := range []string{
		`DELETE FROM source_bindings WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_views WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_connections WHERE tenant_id=$1::uuid`,
		`DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = pool.Exec(ctx, statement, historyTenantID)
	}
}
