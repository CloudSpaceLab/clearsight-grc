package continuity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateAndAssignProgram(t *testing.T) {
	service := NewService(NewMemoryRepository())
	created, err := service.CreateProgram(WithTrustedSystemScope(t.Context()), CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity", Code: "NDPA", Name: "Data protection",
		Type: "PRIVACY", OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1",
		AuthorityPrincipalID: "approver-1", Scope: json.RawMessage(`{"business_lines":["Retail"]}`),
		EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateProgramDetails(WithTrustedSystemScope(t.Context()), UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		Name: "Nigeria data protection", OwningFunction: "Data Protection Office", Jurisdiction: "Nigeria",
		Scope: json.RawMessage(`{"business_lines":["Retail","Corporate"]}`), EffectiveFrom: created.Program.EffectiveFrom,
		ActorID: "owner-1", Rationale: "Confirm the approved operating scope.",
	})
	if err != nil {
		t.Fatal(err)
	}
	assigned, err := service.AssignProgram(WithTrustedSystemScope(t.Context()), AssignProgramInput{
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
	created, err := service.CreateProgram(WithTrustedSystemScope(t.Context()), CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "AML", Name: "Financial crime", Type: "AML", OwningFunction: "Compliance",
		OwnerPrincipalID: "owner-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateProgramDetails(WithTrustedSystemScope(t.Context()), UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		Name: created.Program.Name, OwningFunction: created.Program.OwningFunction, EffectiveFrom: created.Program.EffectiveFrom,
	})
	if err == nil {
		t.Fatal("detail update without actor and rationale succeeded")
	}
	_, err = service.AssignProgram(WithTrustedSystemScope(t.Context()), AssignProgramInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version - 1,
		OwnerPrincipalID: "owner-2", ActorID: "owner-1", Rationale: "Current DPO position.",
	})
	if err == nil {
		t.Fatal("stale assignment succeeded")
	}
}

func TestSupersedeRequirementPreservesHistory(t *testing.T) {
	service := NewService(NewMemoryRepository())
	effectiveFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	created, err := service.CreateProgram(WithTrustedSystemScope(t.Context()), CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: "owner-1",
		EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	withRequirement, err := service.AddRequirement(WithTrustedSystemScope(t.Context()), AddRequirementInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		SourceID: "source-1", Code: "CAR-01", Title: "File the annual return",
		Statement: "The bank must file its annual compliance return.", SourceAnchor: "GAID 2025, section 7",
		Modality: "MUST", Actor: "The bank", Action: "file", Object: "the annual compliance return",
		Status: RequirementApproved, EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	priorID := withRequirement.Requirements[0].ID
	replacementFrom := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	updated, err := service.SupersedeRequirement(WithTrustedSystemScope(t.Context()), SupersedeRequirementInput{
		TenantID: "bank", ProgramID: created.Program.ID, RequirementID: priorID,
		ExpectedVersion: withRequirement.Program.Version, SourceID: "source-2", Code: "CAR-01",
		Title: "File the annual return", Statement: "The bank must file its annual compliance return through a licensed DPCO.",
		SourceAnchor: "GAID 2025, section 7.2", Modality: "MUST", Actor: "The bank", Action: "file",
		Object: "the annual compliance return", EffectiveFrom: replacementFrom,
		ActorID: "owner-1", Rationale: "The regulator changed the filing channel.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Requirements) != 2 {
		t.Fatalf("expected preserved and replacement requirements, got %#v", updated.Requirements)
	}
	prior := updated.Requirements[0]
	replacement := updated.Requirements[1]
	if prior.ID != priorID || prior.Status != RequirementSuperseded || prior.EffectiveUntil == nil || !prior.EffectiveUntil.Equal(replacementFrom) {
		t.Fatalf("prior requirement history was not closed correctly: %#v", prior)
	}
	if replacement.ID == priorID || replacement.Status != RequirementApproved || replacement.Statement == prior.Statement || !replacement.EffectiveFrom.Equal(replacementFrom) {
		t.Fatalf("replacement requirement is incorrect: %#v", replacement)
	}
}

func TestSupersedeRequirementRejectsInvalidHistoryChanges(t *testing.T) {
	service := NewService(NewMemoryRepository())
	effectiveFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	created, err := service.CreateProgram(WithTrustedSystemScope(t.Context()), CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "AML", Name: "Financial crime", Type: "AML", OwningFunction: "Compliance",
		OwnerPrincipalID: "owner-1", EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	withRequirement, err := service.AddRequirement(WithTrustedSystemScope(t.Context()), AddRequirementInput{
		TenantID: "bank", ProgramID: created.Program.ID, ExpectedVersion: created.Program.Version,
		Code: "AML-01", Title: "Review transactions", Statement: "The bank must review transactions.",
		SourceAnchor: "AML rule 1", EffectiveFrom: effectiveFrom, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SupersedeRequirement(WithTrustedSystemScope(t.Context()), SupersedeRequirementInput{
		TenantID: "bank", ProgramID: created.Program.ID, RequirementID: withRequirement.Requirements[0].ID,
		ExpectedVersion: withRequirement.Program.Version, Code: "AML-01", Title: "Review transactions",
		Statement: "The bank must review all transactions.", SourceAnchor: "AML rule 1", EffectiveFrom: effectiveFrom,
		ActorID: "owner-1", Rationale: "Clarify the population.",
	})
	if err == nil {
		t.Fatal("supersession with an overlapping effective date succeeded")
	}
}
