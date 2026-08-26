package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type failReadAfterProgramCreateRepository struct {
	*MemoryRepository
	created bool
}

func (r *failReadAfterProgramCreateRepository) CreateProgram(ctx context.Context, program Program, event Event) (Program, error) {
	created, err := r.MemoryRepository.CreateProgram(ctx, program, event)
	if err == nil {
		r.created = true
	}
	return created, err
}

func (r *failReadAfterProgramCreateRepository) GetProgram(ctx context.Context, tenant, programID string) (ProgramAggregate, error) {
	if r.created {
		return ProgramAggregate{}, errors.New("post-commit Program read unavailable")
	}
	return r.MemoryRepository.GetProgram(ctx, tenant, programID)
}

func TestCreateProgramReturnsCommittedAggregateWhenCurrentReadIsUnavailable(t *testing.T) {
	repo := &failReadAfterProgramCreateRepository{MemoryRepository: NewMemoryRepository()}
	service := NewService(repo)
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	result, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-1", Code: "CREATE-READ",
		Name: "Create read recovery", Type: "COMPLIANCE", OwningFunction: "Risk",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer",
		EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatalf("committed Program creation returned a read failure: %v", err)
	}
	if result.Program.ID == "" || result.Program.Version != 1 || result.Program.Code != "CREATE-READ" || result.Program.Status != ProgramDraft {
		t.Fatalf("committed Program creation result = %#v", result.Program)
	}
}

func TestDefineEvidenceCheckReturnsCommittedAggregateWhenCurrentReadIsUnavailable(t *testing.T) {
	base := NewMemoryRepository()
	service := NewService(base)
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-1", Code: "EVIDENCE-DEFINE-READ",
		Name: "Evidence definition read recovery", Type: "ASSURANCE", OwningFunction: "Risk",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer",
		EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ", Title: "Retain evidence", Statement: "Evidence must be retained.",
		EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionProgram(ctx, ProgramTransitionInput{
		TenantID: "bank", ID: program.Program.ID, ExpectedVersion: program.Program.Version,
		To: ProgramActive, ActorID: "program-authorizer", Rationale: "The approved requirement is ready to operate.",
	})
	if err != nil {
		t.Fatal(err)
	}

	wrapper := &failReadAfterProgramApplyRepository{Repository: base}
	service.repo = wrapper
	result, err := service.AddEvidenceContract(ctx, AddEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		RequirementID: program.Requirements[0].ID, Code: "CHECK", Name: "Evidence check",
		Claim: "Evidence remains current.", PopulationScope: json.RawMessage(`{"population":"accounts"}`),
		FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW",
		FailureAction: "MATTER", Status: EvidenceContractDraft, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatalf("committed evidence definition returned a read failure: %v", err)
	}
	if result.Program.Version != program.Program.Version+1 || len(result.EvidenceContracts) != 1 {
		t.Fatalf("committed evidence definition result = %#v", result)
	}
	if got := result.EvidenceContracts[0]; got.Code != "CHECK" || got.ConfiguredBy != "program-owner" || got.Status != EvidenceContractDraft {
		t.Fatalf("committed evidence check = %#v", got)
	}
	if result.CurrentState == nil || result.CurrentState.ProgramVersion == result.Program.Version {
		t.Fatalf("fallback must retain an explicitly stale projection, got Program version %d and state %#v", result.Program.Version, result.CurrentState)
	}
	if result.StateLabel != programStateLabel(StateUnknown) {
		t.Fatalf("stale projection was presented as current: label=%q state=%#v", result.StateLabel, result.CurrentState)
	}
}

func TestUpdateProgramDetailsReturnsCommittedAggregateWhenCurrentReadIsUnavailable(t *testing.T) {
	base := NewMemoryRepository()
	service := NewService(base)
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-1", Code: "DETAIL-READ",
		Name: "Original Program name", Type: "COMPLIANCE", OwningFunction: "Risk",
		OwnerPrincipalID: "program-owner", EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	wrapper := &failReadAfterProgramApplyRepository{Repository: base}
	service.repo = wrapper
	result, err := service.UpdateProgramDetails(ctx, UpdateProgramDetailsInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Name: "Updated Program name", OwningFunction: "Compliance", Jurisdiction: "Nigeria",
		Scope: json.RawMessage(`{"services":["payments"]}`), EffectiveFrom: program.Program.EffectiveFrom,
		ActorID: "program-owner", Rationale: "The accountable scope and operating function changed.",
	})
	if err != nil {
		t.Fatalf("committed Program edit returned a read failure: %v", err)
	}
	if result.Program.Version != program.Program.Version+1 || result.Program.Name != "Updated Program name" || result.Program.OwningFunction != "Compliance" || result.Program.Jurisdiction != "Nigeria" {
		t.Fatalf("committed Program edit result = %#v", result.Program)
	}
}
