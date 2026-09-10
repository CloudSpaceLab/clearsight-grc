package bankverticals

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type archivedVendorRepository struct{ *thirdparty.MemoryRepository }

func (r archivedVendorRepository) ListRelationships(ctx context.Context, filter thirdparty.ListFilter) (thirdparty.RelationshipPage, error) {
	if !filter.IncludeArchived {
		return thirdparty.RelationshipPage{Items: []thirdparty.Aggregate{}}, nil
	}
	return r.MemoryRepository.ListRelationships(ctx, filter)
}

func TestReferenceInstallerReusesArchivedVendor(t *testing.T) {
	repo := archivedVendorRepository{thirdparty.NewMemoryRepository()}
	vendors := thirdparty.NewService(repo)
	installer := &Service{}
	config := DemoSeedConfig()
	first, err := installer.EnsureReferenceVendor(context.Background(), config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	second, err := installer.EnsureReferenceVendor(context.Background(), config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	if first.Relationship.ID != second.Relationship.ID {
		t.Fatal("archived reference vendor was duplicated")
	}
}
