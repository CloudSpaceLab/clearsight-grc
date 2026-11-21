package thirdparty

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestRelationshipCreateReplayAndConflict(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}
	input := validCreateInput()
	input.IdempotencyKey = "create-relationship-0001"
	const count = 12
	results := make([]Aggregate, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = service.CreateRelationship(context.Background(), actor, input)
		}(i)
	}
	wg.Wait()
	for i := range results {
		if errs[i] != nil || results[i].Relationship.ID != results[0].Relationship.ID {
			t.Fatalf("replay %d: %#v %v", i, results[i], errs[i])
		}
	}
	if len(repo.relationships) != 1 || len(repo.vendors) != 1 {
		t.Fatal("duplicate creation")
	}
	if len(repo.relationshipEvents) != 1 || len(repo.relationshipOutbox) != 1 {
		t.Fatal("duplicate or missing audit and outbox event")
	}
	input.ServiceName = "Different service"
	if _, err := service.CreateRelationship(context.Background(), actor, input); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed payload: %v", err)
	}
}

func TestRelationshipReuseChecksBothVersions(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	ctx := context.Background()
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}
	first, err := service.CreateRelationship(ctx, actor, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	input.ExistingRelationshipID = first.Relationship.ID
	input.ExpectedExistingRelationshipVersion = first.Relationship.Version
	input.ExpectedVendorVersion = first.Vendor.Version
	input.ServiceName = "Settlement"
	input.IdempotencyKey = "reuse-relationship-0001"
	repo.mu.Lock()
	vendor := repo.vendors[first.Vendor.ID]
	vendor.Version++
	repo.vendors[vendor.ID] = vendor
	repo.mu.Unlock()
	if _, err := service.CreateRelationship(ctx, actor, input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale vendor: %v", err)
	}
	if len(repo.relationships) != 1 {
		t.Fatal("stale command created a relationship")
	}
}
