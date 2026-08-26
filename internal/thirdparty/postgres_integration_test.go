//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	thirdPartyTenantID  = "33333333-3333-7333-8333-333333333331"
	thirdPartyEntityA   = "33333333-3333-7333-8333-333333333332"
	thirdPartyEntityB   = "33333333-3333-7333-8333-333333333333"
	thirdPartyPrincipal = "33333333-3333-7333-8333-333333333334"
)

func TestPostgresRelationshipTransactionReuseAndScope(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'third-party-bank','Third Party Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
			($2::uuid,$1::uuid,'ENTITY-A','Entity A','Nigeria'),($3::uuid,$1::uuid,'ENTITY-B','Entity B','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($4::uuid,$1::uuid,'PERSON','Vendor Owner','ACTIVE')`,
		pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, thirdPartyEntityA, thirdPartyEntityB, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewPostgresRepository(pool))
	actorA := Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal}
	first, err := service.CreateRelationship(ctx, actorA, validPostgresCreateInput("Card transaction processing"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateRelationship(ctx, actorA, validPostgresCreateInput("Settlement reporting"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Vendor.ID != second.Vendor.ID {
		t.Fatalf("exact source identity created duplicate vendors: %s %s", first.Vendor.ID, second.Vendor.ID)
	}
	assertPostgresCount(t, pool, "third_parties", 1)
	assertPostgresCount(t, pool, "third_party_relationships", 2)
	var relationshipEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_events WHERE aggregate_type='VENDOR_RELATIONSHIP'`).Scan(&relationshipEvents); err != nil || relationshipEvents != 2 {
		t.Fatalf("relationship event count=%d err=%v", relationshipEvents, err)
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR_RELATIONSHIP'`, thirdPartyTenantID).Scan(&outboxCount); err != nil || outboxCount != 2 {
		t.Fatalf("outbox count=%d err=%v", outboxCount, err)
	}

	actorB := Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityB, PrincipalID: thirdPartyPrincipal}
	inputB := validPostgresCreateInput("Cloud hosting")
	inputB.ExternalRef = "vendor-20002"
	if _, err := service.CreateRelationship(ctx, actorB, inputB); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListRelationships(ctx, actorA, ListInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Relationship.LegalEntityID != thirdPartyEntityA {
		t.Fatalf("legal entity scope was not applied before limit: %#v", page)
	}
	if _, err := service.GetRelationship(ctx, actorB, first.Relationship.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity read returned %v", err)
	}

	beforeEvents := postgresCount(t, pool, "third_party_events")
	beforeOutbox := outboxCountForTenant(t, pool)
	_, err = service.UpdateRelationship(ctx, actorA, first.Relationship.ID, UpdateRelationshipInput{ExpectedVersion: 99, ServiceName: "Stale edit", Criticality: CriticalityStandard, PrivacyRole: PrivacyNone})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	if postgresCount(t, pool, "third_party_events") != beforeEvents || outboxCountForTenant(t, pool) != beforeOutbox {
		t.Fatal("stale update wrote an event or outbox row")
	}
}

func validPostgresCreateInput(service string) CreateRelationshipInput {
	return CreateRelationshipInput{LegalName: "Acme Processing Limited", TradingName: "Acme Processing", RegistrationRef: "RC-10001", Jurisdiction: "Nigeria", SourceID: "procurement", ExternalRef: "vendor-10001", ServiceName: service, Criticality: CriticalityImportant, PrivacyRole: PrivacyProcessor}
}

func assertPostgresCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	if got := postgresCount(t, pool, table); got != want {
		t.Fatalf("%s count=%d want=%d", table, got, want)
	}
}

func postgresCount(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	allowed := map[string]bool{"third_parties": true, "third_party_relationships": true, "third_party_events": true}
	if !allowed[table] {
		t.Fatalf("unsupported test table %q", table)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func outboxCountForTenant(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR_RELATIONSHIP'`, thirdPartyTenantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
