package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestRestrictedMatterVisibility(t *testing.T) {
	matter := continuity.Matter{TenantID: "bank-a", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1"]}`)}
	if canReadMatter(context.Background(), matter) {
		t.Fatal("restricted matter was visible without a verified actor")
	}
	unauthorized := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-2", LegalEntityID: "bank-ng"})
	if canReadMatter(unauthorized, matter) {
		t.Fatal("restricted matter was visible to an unlisted principal")
	}
	authorized := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	if !canReadMatter(authorized, matter) {
		t.Fatal("restricted matter was hidden from an allowed principal")
	}
	wildcard := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "oversight", LegalEntityID: "*"})
	if canReadMatter(wildcard, matter) {
		t.Fatal("legal-entity wildcard bypassed the explicit allow-list")
	}
	wrongTenant := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-b", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	if canReadMatter(wrongTenant, matter) {
		t.Fatal("matter was visible across tenants")
	}
}

func TestMalformedRestrictionFailsClosed(t *testing.T) {
	actor := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	for _, scope := range []json.RawMessage{
		json.RawMessage(`{"access":`),
		json.RawMessage(`{"access":"RESTRICTED"}`),
		json.RawMessage(`{"access":"SECRET"}`),
	} {
		if canReadMatter(actor, continuity.Matter{TenantID: "bank-a", Scope: scope}) {
			t.Fatalf("malformed or unsupported access policy was visible: %s", scope)
		}
	}
}
