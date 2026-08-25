package continuity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateAndAssignProgram(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.CreateProgram(t.Context(), CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity", Code: "NDPA", Name: "Data protection",
		Type: "PRIVACY", OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1",
		AuthorityPrincipalID: "approver-1", Scope: json.RawMessage(`{"business_lines":["Retail"]}`),
		EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateProgramDetails(t.Context(), UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		Name: "Nigeria data protection", OwningFunction: "Data Protection Office", Jurisdiction: "Nigeria",
		Scope: json.RawMessage(`{"business_lines":["Retail","Corporate"]}`), EffectiveFrom: created.Program.EffectiveFrom,
		ActorID: "owner-1", Rationale: "Confirm the approved operating scope.",
	})
	if err != nil {
		t.Fatal(err)
	}
	assigned, err := service.AssignProgram(t.Context(), AssignProgramInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: updated.Program.Version,
		OwnerPrincipalID: "owner-2", ActorID: "owner-1", Rationale: "Move accountability to the current DPO position.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Program.OwnerPrincipalID != "owner-2" {
		t.Fatalf("owner not changed: %#v", assigned.Program)
	}
	if assigned.Program.Code != "NDPA" || assigned.Program.Type != "PRIVACY" || assigned.Program.LegalEntityID != "entity" || assigned.Program.AuthorityPrincipalID != "approver-1" {
		t.Fatalf("immutable identity or authority changed: %#v", assigned.Program)
	}
	if assigned.Program.Name != "Nigeria data protection" || assigned.Program.Jurisdiction != "Nigeria" {
		t.Fatalf("details not changed: %#v", assigned.Program)
	}
}

func TestProgramEditsRequireActorRationaleAndCurrentVersion(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.CreateProgram(t.Context(), CreateProgramInput{
		TenantID: "bank", Code: "AML", Name: "Financial crime", Type: "AML", OwningFunction: "Compliance",
		OwnerPrincipalID: "owner-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateProgramDetails(t.Context(), UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		Name: created.Program.Name, OwningFunction: created.Program.OwningFunction, EffectiveFrom: created.Program.EffectiveFrom,
	})
	if err == nil {
		t.Fatal("detail update without actor and rationale succeeded")
	}
	_, err = service.AssignProgram(t.Context(), AssignProgramInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version - 1,
		OwnerPrincipalID: "owner-2", ActorID: "owner-1", Rationale: "Current DPO position.",
	})
	if err == nil {
		t.Fatal("stale assignment succeeded")
	}
}
