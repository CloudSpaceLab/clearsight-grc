package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type evidenceSourceValidatorStub struct {
	wantEntity string
	rejectID   string
}

func (v evidenceSourceValidatorStub) ValidateActiveSourcesForEntity(_ context.Context, _, entity string, ids []string) error {
	if entity != v.wantEntity {
		return errors.New("wrong legal entity")
	}
	for _, id := range ids {
		if id == v.rejectID {
			return ErrEvidenceSourceInvalid
		}
	}
	return nil
}

func TestEvidenceContractsRejectSourcesOutsideAggregateEntity(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	service := NewService(NewMemoryRepository())
	service.ConfigureEvidenceSourceValidator(evidenceSourceValidatorStub{wantEntity: "entity-a", rejectID: "foreign-source"})
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "P", Name: "Program", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "REQ", Title: "Retain evidence", Statement: "The bank retains evidence.", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	version := program.Program.Version
	_, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: version, RequirementID: program.Requirements[0].ID, Code: "CHECK", Name: "Check", Claim: "The outcome remains supported.", AcceptableSourceIDs: []string{"foreign-source"}, FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive})
	if !errors.Is(err, ErrEvidenceSourceInvalid) {
		t.Fatalf("program forged source returned %v", err)
	}
	current, _ := service.GetProgram(ctx, "bank", program.Program.ID)
	if current.Program.Version != version || len(current.EvidenceContracts) != 0 {
		t.Fatalf("rejected source changed Program: %#v", current)
	}

	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 2, Title: "Issue", Summary: "A control outcome needs review.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matterVersion := matter.Matter.Version
	_, err = service.AddVerificationContract(ctx, AddVerificationContractInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matterVersion, ExpectedOutcome: "The control operates.", MeasurementSourceID: "foreign-source", FailureResponse: "BLOCK_CLOSE"})
	if !errors.Is(err, ErrEvidenceSourceInvalid) {
		t.Fatalf("matter forged source returned %v", err)
	}
	currentMatter, _ := service.GetMatter(ctx, "bank", matter.Matter.ID)
	if currentMatter.Matter.Version != matterVersion || len(currentMatter.VerificationContracts) != 0 {
		t.Fatalf("rejected source changed Matter: %#v", currentMatter)
	}
}

func TestEvidenceContractsAllowManualNoSourceWithoutSourceService(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "P", Name: "Program", Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "REQ", Title: "Retain evidence", Statement: "The bank retains evidence.", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, Code: "MANUAL", Name: "Manual check", Claim: "A reviewer confirms the retained evidence.", FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive})
	if err != nil || len(program.EvidenceContracts) != 1 {
		t.Fatalf("manual Program contract failed: %v %#v", err, program)
	}

	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 2, Title: "Issue", Summary: "A control outcome needs review.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ExpectedOutcome: "A reviewer confirms the outcome manually.", FailureResponse: "BLOCK_CLOSE"})
	if err != nil || len(matter.VerificationContracts) != 1 {
		t.Fatalf("manual Matter contract failed: %v %#v", err, matter)
	}
}
