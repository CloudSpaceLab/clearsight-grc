package runtimecontext

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestIdentityResolverUsesVerifiedActorAndExplicitLabels(t *testing.T) {
	actor := identity.Actor{
		TenantID: "tenant-live", LegalEntityID: "entity-live", PrincipalID: "person-live", Kind: "PERSON",
		RoleCodes: []string{"CRO"}, AuthenticationMethod: "OIDC", AssuranceLevel: "AAL2", SessionID: "session-live",
		DepartmentGrants: []identity.DepartmentGrant{{Path: []string{"RISK"}, RoleCodes: []string{"CRO"}}},
	}
	resolver := IdentityResolver{
		TenantNames:      map[string]string{"tenant-live": "Live Bank"},
		LegalEntityNames: map[string]string{"entity-live": "Live Bank Nigeria"},
		PrincipalNames:   map[string]string{"person-live": "Ada Example"},
	}
	got, err := resolver.Resolve(t.Context(), actor)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tenant.ID != actor.TenantID || got.Tenant.Name != "Live Bank" || got.LegalEntity.ID != actor.LegalEntityID || got.Actor.ID != actor.PrincipalID || got.Actor.Name != "Ada Example" {
		t.Fatalf("resolved context = %#v", got)
	}
	if got.Actor.Authentication != "OIDC" || got.Actor.AssuranceLevel != "AAL2" || len(got.Actor.RoleCodes) != 1 || got.Actor.RoleCodes[0] != "CRO" {
		t.Fatalf("resolved actor metadata = %#v", got.Actor)
	}

	actor.RoleCodes[0] = "MUTATED"
	actor.DepartmentGrants[0].Path[0] = "MUTATED"
	if got.Actor.RoleCodes[0] != "CRO" || got.Actor.DepartmentGrants[0].Path[0] != "RISK" {
		t.Fatalf("resolver leaked mutable actor slices: %#v", got.Actor)
	}
}

func TestIdentityResolverFallsBackToVerifiedIdentifiers(t *testing.T) {
	actor := identity.Actor{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-a"}
	got, err := (IdentityResolver{}).Resolve(t.Context(), actor)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tenant.Name != actor.TenantID || got.LegalEntity.Name != actor.LegalEntityID || got.Actor.Name != actor.PrincipalID {
		t.Fatalf("fallback context invented display data: %#v", got)
	}
}
