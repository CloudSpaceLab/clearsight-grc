package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestRestrictedMatterVisibility(t *testing.T) {
	matter := continuity.Matter{Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-1"]}`)}
	if canReadMatter(context.Background(), matter) {
		t.Fatal("restricted matter was visible without a verified actor")
	}
	unauthorized := identity.WithActor(context.Background(), identity.Actor{PrincipalID: "person-2", LegalEntityID: "bank-ng"})
	if canReadMatter(unauthorized, matter) {
		t.Fatal("restricted matter was visible to an unlisted principal")
	}
	authorized := identity.WithActor(context.Background(), identity.Actor{PrincipalID: "person-1", LegalEntityID: "bank-ng"})
	if !canReadMatter(authorized, matter) {
		t.Fatal("restricted matter was hidden from an allowed principal")
	}
	bankWide := identity.WithActor(context.Background(), identity.Actor{PrincipalID: "oversight", LegalEntityID: "*"})
	if !canReadMatter(bankWide, matter) {
		t.Fatal("restricted matter was hidden from a bank-wide actor")
	}
}
