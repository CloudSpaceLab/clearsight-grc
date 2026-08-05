package authority

import (
	"errors"
	"testing"
)

func TestResolverUsesMostSpecificHighPriorityRule(t *testing.T) {
	version, rules := DemoPolicySet()
	resolver := NewResolver(version, rules)
	resolution, err := resolver.Resolve(ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "PROGRAM", ObjectID: "ndpa", Responsibility: ResponsibilityAuthorizer, Materiality: 3})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolution.Principal.Role != "DPO" {
		t.Fatalf("expected DPO, got %s", resolution.Principal.Role)
	}
	if resolution.PolicyVersion != version {
		t.Fatalf("expected policy version %s, got %s", version, resolution.PolicyVersion)
	}
}

func TestResolverRejectsUnroutableRequest(t *testing.T) {
	version, rules := DemoPolicySet()
	resolver := NewResolver(version, rules)
	_, err := resolver.Resolve(ResolveInput{TenantID: "other-bank", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: ResponsibilityAuthorizer, Materiality: 5})
	if !errors.Is(err, ErrNoRoute) {
		t.Fatalf("expected ErrNoRoute, got %v", err)
	}
}

func BenchmarkResolver(b *testing.B) {
	version, rules := DemoPolicySet()
	resolver := NewResolver(version, rules)
	input := ResolveInput{TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: ResponsibilityAuthorizer, Materiality: 5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := resolver.Resolve(input); err != nil {
			b.Fatal(err)
		}
	}
}
