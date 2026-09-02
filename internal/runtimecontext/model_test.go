package runtimecontext

import (
	"context"
	"errors"
	"testing"
)

func TestIdentifierResolverReturnsOnlyVerifiedIdentifiers(t *testing.T) {
	resolver := IdentifierResolver{}
	value, err := resolver.Resolve(context.Background(), Scope{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a"})
	if err != nil {
		t.Fatal(err)
	}
	if value != (DisplayContext{TenantName: "tenant-a", LegalEntityName: "entity-a", PrincipalName: "principal-a"}) {
		t.Fatalf("display context = %#v", value)
	}
	if _, err := resolver.Resolve(context.Background(), Scope{TenantID: "tenant-a"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("incomplete scope error = %v", err)
	}
}
