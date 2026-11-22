package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestVisibleInterventionMemoryRequiresExactScopeAndMatterAccess(t *testing.T) {
	repo := NewMemoryRepository()
	repo.matters["bank"] = map[string]MatterAggregate{"m1": {Matter: Matter{ID: "m1", TenantID: "bank", LegalEntityID: "entity", Reference: "MAT-1", Status: MatterAssessment, TriggerKey: "trigger", OwnerPrincipalID: "owner", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["reader"]}`)}, Links: []MatterLink{{ProgramID: "program"}}, Actions: []Action{{OwnerPrincipalID: "assigned"}}}}
	service := NewService(repo)
	actor := func(principal, entity string, expires time.Time) context.Context {
		return identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: entity, PrincipalID: principal, ExpiresAt: expires})
	}
	now := time.Now()
	for _, principal := range []string{"owner", "reader", "assigned"} {
		value, err := service.VisibleIntervention(actor(principal, "entity", now.Add(time.Hour)), "bank", "entity", "trigger", "program")
		if err != nil || value.ID != "m1" {
			t.Fatalf("%s: %#v %v", principal, value, err)
		}
	}
	for _, ctx := range []context.Context{context.Background(), actor("other", "entity", now.Add(time.Hour)), actor("owner", "entity", now.Add(-time.Second)), actor("owner", "different", now.Add(time.Hour))} {
		if _, err := service.VisibleIntervention(ctx, "bank", "entity", "trigger", "program"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("unauthorized lookup: %v", err)
		}
	}
	if _, err := service.VisibleIntervention(actor("owner", "entity", now.Add(time.Hour)), "bank", "entity", "trigger", "different"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong program: %v", err)
	}
}

type interventionUnsupportedRepository struct{ Repository }

func TestVisibleInterventionUnsupportedRepositoryIsUnavailable(t *testing.T) {
	svc := NewService(interventionUnsupportedRepository{Repository: NewMemoryRepository()})
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner", ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := svc.VisibleIntervention(ctx, "bank", "entity", "trigger", "program"); !errors.Is(err, ErrInterventionUnavailable) {
		t.Fatalf("unsupported repository returned %v", err)
	}
}
