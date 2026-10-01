package runtimecontext

import (
	"context"
	"errors"
	"testing"
)

func TestIdentifierResolverReturnsOnlyVerifiedIdentifiers(t *testing.T) {
	resolver := IdentifierResolver{}
	scope := Scope{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a"}
	value, err := resolver.Resolve(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if value != (DisplayContext{TenantName: "tenant-a", LegalEntityName: "entity-a", PrincipalName: "principal-a"}) {
		t.Fatalf("display context = %#v", value)
	}
	if _, err := resolver.Resolve(context.Background(), Scope{TenantID: "tenant-a"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("incomplete scope error = %v", err)
	}

	hierarchy, err := resolver.ResolveHierarchy(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if hierarchy.State != HierarchyCurrentOnly {
		t.Fatalf("hierarchy state = %q", hierarchy.State)
	}
	if hierarchy.Root.Kind != ScopeKindOrganization || hierarchy.Root.ID != scope.TenantID {
		t.Fatalf("root = %#v", hierarchy.Root)
	}
	if hierarchy.Current.Kind != ScopeKindLegalEntity || hierarchy.Current.ID != scope.LegalEntityID || !hierarchy.Current.Current {
		t.Fatalf("current = %#v", hierarchy.Current)
	}
	if len(hierarchy.LegalEntities) != 1 || hierarchy.LegalEntities[0] != hierarchy.Current {
		t.Fatalf("legal entities = %#v", hierarchy.LegalEntities)
	}
}
