package continuity

import (
	"context"
	"testing"
	"time"
)

func TestApplicabilityReasonRequiredForExceptions(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	svc := NewService(NewMemoryRepository())
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	program, err := svc.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity", Code: "TEST", Name: "Test", Type: "PRIVACY", OwningFunction: "Risk", OwnerPrincipalID: "owner", AuthorityPrincipalID: "approver", EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = svc.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "R1", Title: "Review access", Statement: "Review access.", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	input := DetermineApplicabilityInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, Status: ApplicabilityApplicable, ApprovedBy: "approver", EffectiveFrom: now}
	updated, err := svc.DetermineApplicability(ctx, input)
	if err != nil {
		t.Fatalf("applicable decision without explanation: %v", err)
	}
	if got := updated.Applicability[len(updated.Applicability)-1]; got.Rationale != "" || got.ApprovedBy != "approver" || got.Status != ApplicabilityApplicable {
		t.Fatalf("decision not retained: %#v", got)
	}
	input.ExpectedVersion = updated.Program.Version
	for _, status := range []ApplicabilityStatus{ApplicabilityNotApplicable, ApplicabilityPartial, ApplicabilityPotential, ApplicabilityLater} {
		input.Status = status
		if _, err := svc.DetermineApplicability(ctx, input); err == nil {
			t.Errorf("%s accepted without a reason", status)
		}
	}
	input.Status, input.Rationale = ApplicabilityNotApplicable, "The service has no privileged accounts."
	updated, err = svc.DetermineApplicability(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Applicability[len(updated.Applicability)-1]; got.Rationale != input.Rationale || got.Status != ApplicabilityNotApplicable {
		t.Fatalf("exception not retained: %#v", got)
	}
}
