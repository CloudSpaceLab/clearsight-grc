package httpapi

import (
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestExactRecordActorRejectsFixedEntityMismatch(t *testing.T) {
	_, err := (&API{}).exactRecordActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "owner-1", LegalEntityID: "entity-a"}, "bank", "bank", "entity-b")
	if !errors.Is(err, commandauth.ErrLegalEntityMismatch) {
		t.Fatalf("fixed entity mismatch was not rejected: %v", err)
	}
}

func TestExactRecordActorAcceptsCanonicalEquivalentEntityCode(t *testing.T) {
	repository := continuity.NewMemoryRepository()
	repository.RegisterLegalEntity("bank", "entity-a-id", "ENTITY-A")
	api := &API{deps: Dependencies{Continuity: continuity.NewService(repository)}}

	actor, err := api.exactRecordActor(t.Context(), identity.Actor{TenantID: "bank", PrincipalID: "owner-1", LegalEntityID: "ENTITY-A"}, "bank", "bank", "entity-a-id")
	if err != nil {
		t.Fatalf("canonical entity code was rejected: %v", err)
	}
	if actor.LegalEntityID != "entity-a-id" {
		t.Fatalf("actor was not rebound to the canonical record entity: %#v", actor)
	}
}
