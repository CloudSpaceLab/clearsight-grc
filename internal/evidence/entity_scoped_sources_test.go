package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

type sourceEntityResolverStub struct{ canonical string }

func (s sourceEntityResolverStub) ResolveLegalEntity(context.Context, string, string) (string, error) {
	return s.canonical, nil
}

func TestListSourcesForEntityFiltersBeforeLimitAndPaginates(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository([]Source{
		{ID: "foreign-first", TenantID: "bank", LegalEntityID: "entity-b", Name: "A foreign source", Status: SourceActive, CreatedAt: now},
		{ID: "source-a", TenantID: "bank", LegalEntityID: "entity-a", Name: "B exact source", Status: SourceActive, CreatedAt: now},
		{ID: "source-b", TenantID: "bank", LegalEntityID: "entity-a", Name: "C exact source", Status: SourcePaused, CreatedAt: now},
		{ID: "other-tenant", TenantID: "other", LegalEntityID: "entity-a", Name: "D other tenant", Status: SourceActive, CreatedAt: now},
	}, nil)
	service := NewService(repo, NewMemoryObjectStore())

	first, err := service.ListSourcesForEntity(context.Background(), SourceListQuery{TenantID: "bank", LegalEntityID: "entity-a", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != "source-a" || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := service.ListSourcesForEntity(context.Background(), SourceListQuery{TenantID: "bank", LegalEntityID: "entity-a", Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "source-b" || second.HasMore {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestValidateActiveSourcesForEntityRejectsForgedAndInactiveIDs(t *testing.T) {
	repo := NewMemoryRepository([]Source{
		{ID: "source-a", TenantID: "bank", LegalEntityID: "entity-a", Name: "Exact", Status: SourceActive},
		{ID: "source-b", TenantID: "bank", LegalEntityID: "entity-b", Name: "Foreign", Status: SourceActive},
		{ID: "source-paused", TenantID: "bank", LegalEntityID: "entity-a", Name: "Paused", Status: SourcePaused},
	}, nil)
	service := NewService(repo, NewMemoryObjectStore())

	if err := service.ValidateActiveSourcesForEntity(context.Background(), "bank", "entity-a", nil); err != nil {
		t.Fatalf("manual/no-source path failed: %v", err)
	}
	if err := service.ValidateActiveSourcesForEntity(context.Background(), "bank", "entity-a", []string{"source-a"}); err != nil {
		t.Fatalf("exact active source failed: %v", err)
	}
	for _, ids := range [][]string{{"source-b"}, {"source-paused"}, {"missing"}} {
		if err := service.ValidateActiveSourcesForEntity(context.Background(), "bank", "entity-a", ids); !errors.Is(err, ErrSourceScopeMismatch) {
			t.Fatalf("ids %v returned %v, want scope mismatch", ids, err)
		}
	}
}

func TestListSourcesForEntityRequiresExactScope(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	for _, entity := range []string{"", "*"} {
		if _, err := service.ListSourcesForEntity(context.Background(), SourceListQuery{TenantID: "bank", LegalEntityID: entity}); !errors.Is(err, ErrSourceScopeRequired) {
			t.Fatalf("entity %q returned %v", entity, err)
		}
	}
}

func TestListSourcesForEntityCanonicalizesEntityCode(t *testing.T) {
	repo := NewMemoryRepository([]Source{{ID: "source-a", TenantID: "bank", LegalEntityID: "entity-uuid", Name: "Exact", Status: SourceActive}}, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.ConfigureLegalEntityResolver(sourceEntityResolverStub{canonical: "entity-uuid"})
	page, err := service.ListSourcesForEntity(context.Background(), SourceListQuery{TenantID: "bank", LegalEntityID: "BANK-NG", Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("canonical source list failed: %#v err=%v", page, err)
	}
}
