package authority

import (
	"context"
	"errors"
	"testing"
)

func TestResolveMaterialAuthorizer(t *testing.T) {
	version, rules := DemoPolicySet()
	resolver := NewResolver(version, rules)
	resolution, err := resolver.Resolve(context.Background(), ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: ResponsibilityAuthorizer, Materiality: 5})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolution.Principal.Role != "CRO" {
		t.Fatalf("expected CRO, got %s", resolution.Principal.Role)
	}
}

func TestSimulationExplainsRejectedRules(t *testing.T) {
	version, rules := DemoPolicySet()
	resolver := NewResolver(version, rules)
	simulation, err := resolver.Simulate(context.Background(), ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: ResponsibilityAuthorizer, Materiality: 2})
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if simulation.Selected != nil {
		t.Fatal("expected no selected route")
	}
	if len(simulation.Candidates) == 0 {
		t.Fatal("expected candidates")
	}
}

func TestDecisionTypeMustMatch(t *testing.T) {
	resolver := NewResolver("v1", []Rule{{ID: "reportability", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityAuthorizer, DecisionType: "REPORTABILITY", Principal: Principal{ID: "p1", DisplayName: "MLRO"}, Priority: 1}})
	_, err := resolver.Resolve(context.Background(), ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "m1", Responsibility: ResponsibilityAuthorizer, Materiality: 5})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("expected no route when decision type is omitted, got %v", err)
	}
}

func TestUnresolvedSelectorIsNotEligible(t *testing.T) {
	resolver := NewResolver("v1", []Rule{{ID: "missing", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityAuthorizer, Principal: Principal{}, Priority: 1}})
	_, err := resolver.Resolve(context.Background(), ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "m1", Responsibility: ResponsibilityAuthorizer, Materiality: 5})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("expected no route, got %v", err)
	}
	findings, err := resolver.Integrity(context.Background(), "bank-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) < 2 {
		t.Fatalf("expected unresolved selector and missing authorizer findings, got %#v", findings)
	}
}

func TestInvalidInputRejected(t *testing.T) {
	resolver := NewResolver("v1", nil)
	_, err := resolver.Simulate(context.Background(), ResolveInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
