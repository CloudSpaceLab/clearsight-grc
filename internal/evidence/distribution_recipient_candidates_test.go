package evidence

import (
	"context"
	"testing"
)

func TestDistributionRecipientCandidatesAreEntityScopedFilteredAndBounded(t *testing.T) {
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, nil, []RecipientCandidate{
		{PrincipalID: "person-a", DisplayName: "Ada Okafor", ContextLabel: "Risk manager", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true},
		{PrincipalID: "person-b", DisplayName: "Bola James", ContextLabel: "Operations", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true},
		{PrincipalID: "inactive", DisplayName: "Inactive Person", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: false},
		{PrincipalID: "team", DisplayName: "Risk Team", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "TEAM", Active: true},
		{PrincipalID: "other-entity", DisplayName: "Other Entity", TenantID: "bank", LegalEntityIDs: []string{"entity-b"}, Kind: "PERSON", Active: true},
		{PrincipalID: "other-tenant", DisplayName: "Other Tenant", TenantID: "other", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true},
	})
	service := NewService(repo, nil)

	page, err := service.SearchDistributionRecipientCandidates(context.Background(), "bank", "entity-a", RecipientCandidateSearch{Query: "risk", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].PrincipalID != "person-a" || page.Items[0].ContextLabel != "Risk manager" || page.HasMore {
		t.Fatalf("unexpected filtered page: %#v", page)
	}

	page, err = service.SearchDistributionRecipientCandidates(context.Background(), "bank", "entity-a", RecipientCandidateSearch{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].PrincipalID != "person-a" || !page.HasMore {
		t.Fatalf("bounded entity page = %#v", page)
	}
}
