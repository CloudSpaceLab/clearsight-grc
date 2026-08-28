package evidence

import (
	"context"
	"errors"
	"testing"
)

type responseRevisionTestStore struct {
	bundle       DistributionBundle
	revisions    []ResponseRevision
	responseRead bool
}

func (store *responseRevisionTestStore) CreateDistribution(context.Context, CreateDistributionInput) (DistributionBundle, error) {
	return DistributionBundle{}, ErrDistributionInvalid
}
func (store *responseRevisionTestStore) GetDistribution(_ context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	value := store.bundle.Distribution
	if value.TenantID != tenantID || value.LegalEntityID != legalEntityID || value.ID != distributionID {
		return DistributionBundle{}, ErrNotFound
	}
	return store.bundle, nil
}
func (store *responseRevisionTestStore) ListDistributions(context.Context, DistributionListQuery) ([]FormDistribution, error) {
	return nil, nil
}
func (store *responseRevisionTestStore) ListDistributionResponseRevisions(_ context.Context, tenantID, legalEntityID, distributionID string, limit int) ([]ResponseRevision, error) {
	store.responseRead = true
	if store.bundle.Distribution.TenantID != tenantID || store.bundle.Distribution.LegalEntityID != legalEntityID || store.bundle.Distribution.ID != distributionID || limit != 100 {
		return nil, ErrNotFound
	}
	return store.revisions, nil
}

func TestDistributionResponseRevisionsResolveScopeBeforeHistoryRead(t *testing.T) {
	store := &responseRevisionTestStore{
		bundle: DistributionBundle{Distribution: FormDistribution{ID: "distribution-a", TenantID: "bank", LegalEntityID: "entity-a"}},
		revisions: []ResponseRevision{{ID: "revision-a", Revision: 2, Current: true, State: ResponseRevisionFinal}},
	}
	service := NewDistributionService(store)

	if _, err := service.ListResponseRevisions(context.Background(), "bank", "entity-b", "distribution-a", 100); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected wrong-entity response history to be hidden, got %v", err)
	}
	if store.responseRead {
		t.Fatal("response history store must not be called before distribution scope resolves")
	}

	values, err := service.ListResponseRevisions(context.Background(), "bank", "entity-a", "distribution-a", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !store.responseRead || len(values) != 1 || values[0].ID != "revision-a" || !values[0].Current {
		t.Fatalf("unexpected response revision projection: %#v", values)
	}
}

func TestDistributionResponseRevisionQueryIsBounded(t *testing.T) {
	service := NewDistributionService(&responseRevisionTestStore{bundle: DistributionBundle{Distribution: FormDistribution{ID: "distribution-a", TenantID: "bank", LegalEntityID: "entity-a"}}})
	for _, limit := range []int{0, 101} {
		if _, err := service.ListResponseRevisions(context.Background(), "bank", "entity-a", "distribution-a", limit); !errors.Is(err, ErrDistributionInvalid) {
			t.Fatalf("expected invalid bounded limit %d to fail, got %v", limit, err)
		}
	}
}
