package governance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testEntityA = "11111111-1111-7111-8111-111111111111"
	testEntityB = "22222222-2222-7222-8222-222222222222"
)

func TestCreatePolicyRequiresCanonicalLegalEntity(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	definition := json.RawMessage(`{"rules":[{"id":"r1","legal_entity_id":"` + testEntityA + `","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}`)

	_, err := svc.CreatePolicy(context.Background(), CreatePolicyInput{TenantID: "bank", Code: "risk", Name: "Risk", MakerID: "maker", Definition: definition})
	if err == nil || !strings.Contains(err.Error(), "legal_entity_id is required") {
		t.Fatalf("expected missing legal entity rejection, got %v", err)
	}
}

func TestCreatePolicyRejectsMixedLegalEntityDefinition(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	definition := json.RawMessage(`{"rules":[
		{"id":"r1","legal_entity_id":"` + testEntityA + `","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}},
		{"id":"r2","legal_entity_id":"` + testEntityB + `","responsibility":"REVIEWER","selector":{"kind":"ROLE","ref":"COMPLIANCE"}}
	]}`)

	_, err := svc.CreatePolicy(context.Background(), CreatePolicyInput{TenantID: "bank", LegalEntityID: testEntityA, Code: "risk", Name: "Risk", MakerID: "maker", Definition: definition})
	if err == nil || !strings.Contains(err.Error(), "must match policy legal_entity_id") {
		t.Fatalf("expected mixed entity rejection, got %v", err)
	}
}

func TestCreateDelegationStrictScope(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	base := CreateDelegationInput{
		TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "from", ToPrincipalID: "to",
		Responsibility: "REVIEWER", StartsAt: now, EndsAt: now.Add(time.Hour), MakerID: "maker",
	}
	tests := []struct {
		name  string
		scope string
		want  string
	}{
		{name: "unknown field", scope: `{"legal_entity_id":"` + testEntityA + `","department":"risk"}`, want: "unknown field"},
		{name: "mixed entity", scope: `{"legal_entity_id":"` + testEntityB + `"}`, want: "must match delegation legal_entity_id"},
		{name: "object without type", scope: `{"legal_entity_id":"` + testEntityA + `","object_id":"matter-1"}`, want: "object_type is required"},
		{name: "unsupported object", scope: `{"legal_entity_id":"` + testEntityA + `","object_type":"EVIDENCE"}`, want: "unsupported object_type"},
		{name: "wildcard exact object", scope: `{"legal_entity_id":"` + testEntityA + `","object_type":"MATTER","object_id":"*"}`, want: "object_id must be exact"},
		{name: "inverted materiality", scope: `{"legal_entity_id":"` + testEntityA + `","min_materiality":4,"max_materiality":2}`, want: "min_materiality must not exceed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			input.Scope = json.RawMessage(tt.scope)
			_, err := NewService(NewMemoryRepository()).CreateDelegation(context.Background(), input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q rejection, got %v", tt.want, err)
			}
		})
	}
}

func TestCreateDelegationCanonicalizesTypedScope(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	delegation, err := NewService(NewMemoryRepository()).CreateDelegation(context.Background(), CreateDelegationInput{
		TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "from", ToPrincipalID: "to",
		Responsibility: "reviewer", StartsAt: now, EndsAt: now.Add(time.Hour), MakerID: "maker",
		Scope: json.RawMessage(`{"legal_entity_id":"` + testEntityA + `","object_type":"matter","object_id":"matter-1","decision_type":"matter.review","min_materiality":1,"max_materiality":4}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if delegation.LegalEntityID != testEntityA || delegation.Responsibility != "REVIEWER" {
		t.Fatalf("delegation was not canonicalized: %#v", delegation)
	}
	var scope DelegationScope
	if err := json.Unmarshal(delegation.Scope, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.LegalEntityID != testEntityA || scope.ObjectType != "MATTER" || scope.ObjectID != "matter-1" {
		t.Fatalf("unexpected canonical scope: %#v", scope)
	}
}

func TestMemoryRepositoryScopedListsFilterBeforeLimit(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	for _, policy := range []RoutingPolicy{
		{ID: "a-1", TenantID: "bank", LegalEntityID: testEntityA, Code: "A", CreatedAt: now},
		{ID: "a-2", TenantID: "bank", LegalEntityID: testEntityA, Code: "B", CreatedAt: now},
		{ID: "b-1", TenantID: "bank", LegalEntityID: testEntityB, Code: "A", CreatedAt: now},
	} {
		if _, err := repo.CreatePolicy(ctx, policy); err != nil {
			t.Fatal(err)
		}
	}
	policies, err := repo.ListPoliciesForEntity(ctx, "bank", testEntityB, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(policies) != 1 || policies[0].ID != "b-1" {
		t.Fatalf("policy scope was applied after limiting: %#v", policies)
	}
	if _, err := repo.GetPolicyForEntity(ctx, "bank", testEntityA, "b-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other-entity exact policy read was not hidden: %v", err)
	}

	for _, delegation := range []Delegation{
		{ID: "a-d", TenantID: "bank", LegalEntityID: testEntityA, CreatedAt: now},
		{ID: "b-d", TenantID: "bank", LegalEntityID: testEntityB, CreatedAt: now.Add(-time.Minute)},
	} {
		if _, err := repo.CreateDelegation(ctx, delegation); err != nil {
			t.Fatal(err)
		}
	}
	delegations, err := repo.ListDelegationsForEntity(ctx, "bank", testEntityB, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(delegations) != 1 || delegations[0].ID != "b-d" {
		t.Fatalf("delegation scope was applied after limiting: %#v", delegations)
	}
	if _, err := repo.GetDelegationForEntity(ctx, "bank", testEntityA, "b-d"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other-entity exact delegation read was not hidden: %v", err)
	}
}

func TestMemoryActivationRevalidatesDelegationCycle(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	candidate := Delegation{ID: "candidate", TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "a", ToPrincipalID: "b", Responsibility: "REVIEWER", StartsAt: now, EndsAt: now.Add(time.Hour), Status: DelegationPendingApproval, MakerID: "maker", Version: 2}
	if _, err := repo.CreateDelegation(ctx, candidate); err != nil {
		t.Fatal(err)
	}
	// This edge appears after the service's earlier read/check but before the
	// repository activation transaction.
	if _, err := repo.CreateDelegation(ctx, Delegation{ID: "new-edge", TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "b", ToPrincipalID: "a", Responsibility: "REVIEWER", StartsAt: now, EndsAt: now.Add(time.Hour), Status: DelegationActive}); err != nil {
		t.Fatal(err)
	}

	_, err := repo.ActivateDelegation(ctx, "bank", testEntityA, "candidate", 2, "checker", "reviewed", now)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected transaction-local cycle rejection, got %v", err)
	}
	stored, _ := repo.GetDelegation(ctx, "bank", "candidate")
	if stored.Status != DelegationPendingApproval || stored.Version != 2 {
		t.Fatalf("failed activation changed candidate: %#v", stored)
	}
}

func TestDueDelegationActivationRevalidatesCycle(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	_, _ = repo.CreateDelegation(ctx, Delegation{ID: "due", TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "a", ToPrincipalID: "b", Responsibility: "REVIEWER", StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour), Status: DelegationApproved, Version: 3})
	_, _ = repo.CreateDelegation(ctx, Delegation{ID: "edge", TenantID: "bank", LegalEntityID: testEntityA, FromPrincipalID: "b", ToPrincipalID: "a", Responsibility: "REVIEWER", StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour), Status: DelegationActive, Version: 2})

	count, err := repo.ActivateDueDelegations(ctx, now, 10)
	if !errors.Is(err, ErrConflict) || count != 0 {
		t.Fatalf("expected due activation conflict with no transition, got count=%d err=%v", count, err)
	}
	stored, _ := repo.GetDelegation(ctx, "bank", "due")
	if stored.Status != DelegationApproved || stored.Version != 3 {
		t.Fatalf("failed due activation changed delegation: %#v", stored)
	}
}

func TestMemoryActivationRevalidatesPolicyConflict(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	policy := RoutingPolicy{
		ID: "policy", TenantID: "bank", LegalEntityID: testEntityA, Status: PolicyPendingApproval,
		MakerID: "maker", Version: 2, CurrentVersion: 1, Definition: validDefinition(),
	}
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	// The conflict appears after the service's earlier check.
	repo.conflicts["bank:policy:policy"] = []ConflictFinding{{Code: "SELECTOR_CARDINALITY", Summary: "role is no longer uniquely occupied"}}

	_, err := repo.ActivatePolicy(ctx, "bank", testEntityA, "policy", 2, "checker", "reviewed", now)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected transaction-local policy conflict rejection, got %v", err)
	}
	stored, _ := repo.GetPolicy(ctx, "bank", "policy")
	if stored.Status != PolicyPendingApproval || stored.Version != 2 {
		t.Fatalf("failed activation changed policy: %#v", stored)
	}
}

func TestMemoryRevisionActivationRevalidatesPolicyConflict(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	policy := RoutingPolicy{ID: "policy", TenantID: "bank", LegalEntityID: testEntityA, Status: PolicyActive, MakerID: "maker", Version: 3, CurrentVersion: 1, Definition: validDefinition()}
	_, _ = repo.CreatePolicy(ctx, policy)
	repo.revisions[key("bank", "policy")] = []RoutingPolicyRevision{{PolicyID: "policy", TenantID: "bank", LegalEntityID: testEntityA, Version: 2, BaseVersion: 1, MakerID: "revision-maker", Definition: validDefinition()}}
	repo.conflicts["bank:policy:policy"] = []ConflictFinding{{Code: "SELECTOR_CARDINALITY", Summary: "role changed"}}

	_, err := repo.ActivatePolicyRevision(ctx, "bank", "policy", 3, 2, "checker", "reviewed", now)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected transaction-local revision conflict rejection, got %v", err)
	}
	stored, _ := repo.GetPolicy(ctx, "bank", "policy")
	if stored.CurrentVersion != 1 || stored.Version != 3 {
		t.Fatalf("failed revision activation changed policy: %#v", stored)
	}
}

func TestMaterialTransitionsHideOtherEntityRecords(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		run  func(*Service, string) error
	}{
		{name: "submit policy", run: func(s *Service, id string) error {
			_, err := s.SubmitPolicy(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "maker", ExpectedVersion: 2})
			return err
		}},
		{name: "approve policy", run: func(s *Service, id string) error {
			_, err := s.ApprovePolicy(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "checker", ExpectedVersion: 2})
			return err
		}},
		{name: "reject policy", run: func(s *Service, id string) error {
			_, err := s.RejectPolicy(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "checker", ExpectedVersion: 2, Rationale: "reviewed"})
			return err
		}},
		{name: "retire policy", run: func(s *Service, id string) error {
			_, err := s.RetirePolicy(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "checker", ExpectedVersion: 2, Rationale: "retire"})
			return err
		}},
		{name: "submit delegation", run: func(s *Service, id string) error {
			_, err := s.SubmitDelegation(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "maker", ExpectedVersion: 2})
			return err
		}},
		{name: "approve delegation", run: func(s *Service, id string) error {
			_, err := s.ApproveDelegation(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "checker", ExpectedVersion: 2})
			return err
		}},
		{name: "revoke delegation", run: func(s *Service, id string) error {
			_, err := s.RevokeDelegation(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityB, ID: id, ActorID: "from", ExpectedVersion: 2, Rationale: "revoke"})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			policyState := PolicyPendingApproval
			if strings.Contains(tt.name, "retire") {
				policyState = PolicyActive
			}
			_, _ = repo.CreatePolicy(ctx, RoutingPolicy{ID: "record", TenantID: "bank", LegalEntityID: testEntityA, Status: policyState, MakerID: "maker", Version: 2, Definition: validDefinition()})
			delegationState := DelegationPendingApproval
			if strings.Contains(tt.name, "revoke") {
				delegationState = DelegationActive
			} else if strings.Contains(tt.name, "submit") {
				delegationState = DelegationDraft
			}
			_, _ = repo.CreateDelegation(ctx, Delegation{ID: "record", TenantID: "bank", LegalEntityID: testEntityA, Status: delegationState, MakerID: "maker", FromPrincipalID: "from", ToPrincipalID: "to", Responsibility: "REVIEWER", StartsAt: now, EndsAt: now.Add(time.Hour), Version: 2})
			if err := tt.run(NewService(repo), "record"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected other-entity record to be hidden, got %v", err)
			}
		})
	}
}
