package authority

import (
	"context"
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
