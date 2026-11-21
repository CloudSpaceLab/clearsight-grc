//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRelationshipCreateReplayAndVersionPreconditions(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	newID := func() string {
		value, err := id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	tenant, entity, actorID, otherActor := newID(), newID(), newID(), newID()
	slug := "vendor-replay-" + tenant
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$5,'Vendor idempotency'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'A','A','Nigeria'); INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($3::uuid,$1::uuid,'PERSON','Owner','ACTIVE'),($4::uuid,$1::uuid,'PERSON','Other','ACTIVE')`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actorID, otherActor, slug)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresRepository(pool))
	actor := Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: actorID}
	input := validCreateInput()
	input.IdempotencyKey = "vendor-create-concurrency-key"
	input.SourceID = ""
	input.ExternalRef = ""
	const n = 8
	results := make([]Aggregate, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) { defer wg.Done(); results[i], errs[i] = service.CreateRelationship(ctx, actor, input) }(i)
	}
	wg.Wait()
	aliasReplay, err := service.CreateRelationship(ctx, Actor{TenantID: slug, LegalEntityID: entity, PrincipalID: actorID}, input)
	if err != nil || aliasReplay.Relationship.ID != results[0].Relationship.ID || aliasReplay.Vendor.TenantID != slug || aliasReplay.Relationship.TenantID != slug {
		t.Fatalf("slug replay: %#v %v", aliasReplay, err)
	}
	var storedVendorTenant, storedRelationshipTenant string
	if err := pool.QueryRow(ctx, `SELECT result->'vendor'->>'tenant_id',result->'relationship'->>'tenant_id' FROM third_party_relationship_create_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, tenant, entity, input.IdempotencyKey).Scan(&storedVendorTenant, &storedRelationshipTenant); err != nil || storedVendorTenant != tenant || storedRelationshipTenant != tenant {
		t.Fatalf("stored receipt changed on replay: %q %q %v", storedVendorTenant, storedRelationshipTenant, err)
	}
	for i := range results {
		if errs[i] != nil || results[i].Relationship.ID != results[0].Relationship.ID {
			t.Fatalf("request %d: %#v %v", i, results[i], errs[i])
		}
	}
	var relationships, events, outbox, receipts int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM third_party_relationships WHERE tenant_id=$1::uuid), (SELECT count(*) FROM third_party_events WHERE tenant_id=$1::uuid AND event_type='VendorRelationshipCreated'), (SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='VendorRelationshipCreated'), (SELECT count(*) FROM third_party_relationship_create_receipts WHERE tenant_id=$1::uuid)`, tenant).Scan(&relationships, &events, &outbox, &receipts); err != nil {
		t.Fatal(err)
	}
	if relationships != 1 || events != 1 || outbox != 1 || receipts != 1 {
		t.Fatalf("relationships/events/outbox/receipts: %d/%d/%d/%d", relationships, events, outbox, receipts)
	}
	changed := input
	changed.ServiceName = "Changed"
	if _, err = service.CreateRelationship(ctx, actor, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed payload: %v", err)
	}
	if _, err = service.CreateRelationship(ctx, Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: otherActor}, input); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different actor: %v", err)
	}
	wrongID := newID()
	if _, err = pool.Exec(ctx, `UPDATE third_party_relationship_create_receipts SET result=jsonb_set(result,'{relationship,id}',to_jsonb($4::text)) WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, tenant, entity, input.IdempotencyKey, wrongID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateRelationship(ctx, actor, input); !errors.Is(err, ErrNotFound) {
		t.Fatalf("receipt with mismatched relationship id: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE third_party_relationship_create_receipts SET result=jsonb_set(result,'{relationship,id}',to_jsonb($4::text)) WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3`, tenant, entity, input.IdempotencyKey, results[0].Relationship.ID); err != nil {
		t.Fatal(err)
	}
	first := results[0]
	reuse := validCreateInput()
	reuse.SourceID = ""
	reuse.ExternalRef = ""
	reuse.IdempotencyKey = "vendor-reuse-version-key"
	reuse.ExistingRelationshipID = first.Relationship.ID
	reuse.ExpectedExistingRelationshipVersion = first.Relationship.Version
	reuse.ExpectedVendorVersion = first.Vendor.Version
	_, err = pool.Exec(ctx, `UPDATE third_parties SET version=version+1 WHERE id=$1::uuid`, first.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateRelationship(ctx, actor, reuse); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale vendor: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM third_party_relationships WHERE tenant_id=$1::uuid`, tenant).Scan(&relationships); err != nil || relationships != 1 {
		t.Fatalf("stale command created relationship: %d %v", relationships, err)
	}
	reuse.ExpectedVendorVersion++
	reused, err := service.CreateRelationship(ctx, actor, reuse)
	if err != nil || reused.Vendor.ID != first.Vendor.ID || reused.Relationship.ID == first.Relationship.ID {
		t.Fatalf("reuse: %#v %v", reused, err)
	}
	_, err = pool.Exec(ctx, `UPDATE third_party_relationships SET version=version+1 WHERE id=$1::uuid`, first.Relationship.ID)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.CreateRelationship(ctx, actor, reuse)
	if err != nil || replayed.Relationship.ID != reused.Relationship.ID || replayed.Vendor.Version != reused.Vendor.Version {
		t.Fatalf("replay after source changed: %#v %v", replayed, err)
	}
	stale := reuse
	stale.IdempotencyKey = "vendor-reuse-stale-relationship"
	if _, err = service.CreateRelationship(ctx, actor, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale relationship: %v", err)
	}
}
