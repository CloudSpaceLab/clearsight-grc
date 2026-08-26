package governance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestDelegationCandidateSearchUsesExactEntityResponsibilityAndSafeLabels(t *testing.T) {
	repo := NewMemoryRepositoryWithDelegationCandidates([]DelegationCandidateDirectoryEntry{
		{PrincipalID: "ada", DisplayName: "Ada Okafor", ContextLabel: "Risk assurance lead", TenantID: "bank", LegalEntityID: "entity-1", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: true},
		{PrincipalID: "tunde", DisplayName: "Tunde Bello", ContextLabel: "Payment reviewer", TenantID: "bank", LegalEntityID: "entity-1", Responsibilities: []string{"REVIEWER"}, CanReceive: false, Active: true},
		{PrincipalID: "nneka", DisplayName: "Nneka Eze", ContextLabel: "Risk assurance lead", TenantID: "bank", LegalEntityID: "entity-2", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: true},
		{PrincipalID: "foreign", DisplayName: "Foreign Person", TenantID: "other", LegalEntityID: "entity-1", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: true},
		{PrincipalID: "inactive", DisplayName: "Inactive Person", TenantID: "bank", LegalEntityID: "entity-1", Responsibilities: []string{"REVIEWER"}, CanReceive: true, Active: false},
	})
	service := NewService(repo)

	page, err := service.SearchDelegationCandidates(context.Background(), "bank", "entity-1", "REVIEWER", "risk", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].PrincipalID != "ada" || !page.Items[0].CanGive || !page.Items[0].CanReceive {
		t.Fatalf("unexpected candidates: %#v", page)
	}
	if strings.Contains(string(mustJSON(t, page)), "tenant_id") || strings.Contains(string(mustJSON(t, page)), "responsibilities") || strings.Contains(string(mustJSON(t, page)), "active") {
		t.Fatalf("candidate response exposed directory internals: %s", mustJSON(t, page))
	}

	page, err = service.SearchDelegationCandidates(context.Background(), "bank", "entity-1", "REVIEWER", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || !page.HasMore {
		t.Fatalf("bounded page = %#v", page)
	}
}

func TestDelegationCandidateSearchFailsClosedWhenDirectoryUnavailableOrInputInvalid(t *testing.T) {
	service := NewService(NewMemoryRepository())
	if _, err := service.SearchDelegationCandidates(context.Background(), "bank", "entity-1", "REVIEWER", "", 50); !errors.Is(err, ErrDelegationCandidatesUnavailable) {
		t.Fatalf("unconfigured directory error = %v", err)
	}

	configured := NewService(NewMemoryRepositoryWithDelegationCandidates(nil))
	for _, test := range []struct {
		name, tenant, entity, responsibility, query string
	}{
		{name: "missing tenant", entity: "entity-1", responsibility: "REVIEWER"},
		{name: "wildcard entity", tenant: "bank", entity: "*", responsibility: "REVIEWER"},
		{name: "unsupported responsibility", tenant: "bank", entity: "entity-1", responsibility: "ADMIN"},
		{name: "oversized search", tenant: "bank", entity: "entity-1", responsibility: "REVIEWER", query: strings.Repeat("a", 101)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := configured.SearchDelegationCandidates(context.Background(), test.tenant, test.entity, test.responsibility, test.query, 50); !errors.Is(err, ErrDelegationCandidateSearchInvalid) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
