package documentimport

import (
	"context"
	"errors"
	"testing"
)

func TestScopedDocumentReadsDoNotLeakOtherLegalEntities(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil)
	_, _ = repo.Create(ctx, Document{ID: "doc-a", TenantID: "tenant-a", LegalEntityID: "entity-a", Version: 1})
	_, _ = repo.Create(ctx, Document{ID: "doc-b", TenantID: "tenant-a", LegalEntityID: "entity-b", Version: 1})

	values, err := service.ListVisible(ctx, "tenant-a", "entity-a", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "doc-a" {
		t.Fatalf("scoped list leaked or omitted documents: %#v", values)
	}
	if _, err := service.GetVisible(ctx, "tenant-a", "entity-a", "doc-b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity direct read should be hidden, got %v", err)
	}
	if value, err := service.GetVisible(ctx, "tenant-a", "*", "doc-b"); err != nil || value.ID != "doc-b" {
		t.Fatalf("bank-wide actor should retain tenant-wide read: value=%#v err=%v", value, err)
	}
}
