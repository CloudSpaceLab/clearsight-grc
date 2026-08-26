package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSafeguardLifecycleRequiresVersionedCorrectionAssignmentAndTransitions(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	service := NewServiceWithClock(repo, func() time.Time { return now })

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-1", Code: "SAFEGUARDS", Name: "Safeguard lifecycle", Type: "ASSURANCE",
		OwningFunction: "Control Assurance", OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "authorizer", EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "ACCESS", Name: "Access remains controlled", Outcome: "Every privileged account remains approved.", Status: ObjectiveActive, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ObjectiveID: program.ControlObjectives[0].ID, Name: "Access review", Description: "Review privileged access monthly.", ImplementationType: "REVIEW",
		OwnerPrincipalID: "control-owner", Scope: json.RawMessage(`{"population":"privileged accounts"}`), EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	created := program.ControlImplementations[0]
	if created.Status != ImplementationPlanned || created.Version != 1 {
		t.Fatalf("created safeguard = %#v", created)
	}

	now = now.Add(time.Minute)
	program, err = service.ReviseControlImplementation(ctx, ReviseControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: created.ID,
		ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: created.Version,
		Name: "Privileged access review", Description: "Review privileged access and record exceptions monthly.", ImplementationType: "REVIEW",
		Scope: json.RawMessage(`{"population":"privileged accounts","frequency":"monthly"}`), EffectiveFrom: now, Rationale: "Clarify the current control procedure.", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	revised := program.ControlImplementations[0]
	if revised.Version != 2 || revised.Name != "Privileged access review" || revised.Status != ImplementationPlanned || revised.OwnerPrincipalID != "control-owner" {
		t.Fatalf("revised safeguard = %#v", revised)
	}

	now = now.Add(time.Minute)
	program, err = service.AssignControlImplementation(ctx, AssignControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: revised.ID,
		ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: revised.Version,
		OwnerPrincipalID: "control-owner-2", Rationale: "The operating team changed.", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	assigned := program.ControlImplementations[0]
	if assigned.Version != 3 || assigned.OwnerPrincipalID != "control-owner-2" {
		t.Fatalf("assigned safeguard = %#v", assigned)
	}

	for _, target := range []ControlImplementationStatus{ImplementationInProgress, ImplementationImplemented, ImplementationInactive, ImplementationInProgress, ImplementationRetired} {
		now = now.Add(time.Minute)
		program, err = service.TransitionControlImplementation(ctx, TransitionControlImplementationInput{
			TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: assigned.ID,
			ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: program.ControlImplementations[0].Version,
			To: target, Rationale: "Record the current safeguard operating state.", ActorID: "control-owner-2",
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", target, err)
		}
	}
	retired := program.ControlImplementations[0]
	if retired.Status != ImplementationRetired || retired.EffectiveUntil == nil || retired.Version != 8 {
		t.Fatalf("retired safeguard = %#v", retired)
	}
	if _, err = service.TransitionControlImplementation(ctx, TransitionControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: retired.ID,
		ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: retired.Version,
		To: ImplementationInProgress, Rationale: "Invalid reactivation.", ActorID: "control-owner-2",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("retired transition error = %v", err)
	}

	events, err := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{EventControlImplementationRevised, EventControlImplementationOwnerChanged, EventControlImplementationStatusChanged} {
		found := false
		for _, event := range events {
			if event.Type == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("history missing %s: %#v", want, events)
		}
	}
}

func TestSafeguardLifecycleRejectsStaleSubresourceVersionWithoutPartialMutation(t *testing.T) {
	service, ctx, program := programWithPlannedSafeguard(t)
	beforeEvents, err := service.repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	implementation := program.ControlImplementations[0]
	_, err = service.AssignControlImplementation(ctx, AssignControlImplementationInput{
		TenantID: "bank", ProgramID: program.Program.ID, ImplementationID: implementation.ID,
		ExpectedVersion: program.Program.Version, ExpectedImplementationVersion: implementation.Version + 1,
		OwnerPrincipalID: "other-owner", Rationale: "Stale update.", ActorID: "program-owner",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale assignment error = %v", err)
	}
	after, err := service.GetProgram(ctx, "bank", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterEvents, err := service.repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if after.Program.Version != program.Program.Version || !reflect.DeepEqual(after.ControlImplementations[0], implementation) || len(afterEvents) != len(beforeEvents) {
		t.Fatalf("stale assignment partially mutated record: before=%#v after=%#v events=%d/%d", implementation, after.ControlImplementations[0], len(beforeEvents), len(afterEvents))
	}
}

func TestEvidenceContractLifecycleIsVersionedAndReconstructable(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service := NewServiceWithClock(repo, func() time.Time { return now })
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-1", Code: "EVIDENCE", Name: "Evidence lifecycle", Type: "ASSURANCE", OwningFunction: "Control Assurance",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "authorizer", EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "E-1", Title: "Keep evidence", Statement: "The bank keeps current evidence.", Status: RequirementApproved, EffectiveFrom: now, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID,
		Code: "CHECK", Name: "Current evidence", Claim: "Evidence remains current.", FreshnessMinutes: 60, MinimumCoverage: 1,
		ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractDraft, ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	createdAt := now
	contract := program.EvidenceContracts[0]
	if contract.Status != EvidenceContractDraft || contract.Version != 1 {
		t.Fatalf("created evidence check = %#v", contract)
	}

	now = now.Add(time.Minute)
	program, err = service.ReviseEvidenceContract(ctx, ReviseEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ContractID: contract.ID,
		ExpectedVersion: program.Program.Version, ExpectedContractVersion: contract.Version,
		Name: "Current filing evidence", Claim: "The filing evidence remains current and complete.", AcceptableSourceIDs: []string{},
		PopulationScope: json.RawMessage(`{"population":"filings"}`), FreshnessMinutes: 1440, MinimumCoverage: .95, IndependenceRequired: true,
		ContradictionPolicy: "FAIL", FailureAction: "MATTER", Rationale: "Align the check with the current filing process.", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	revised := program.EvidenceContracts[0]
	if revised.Version != 2 || revised.Name != "Current filing evidence" || revised.FreshnessMinutes != 1440 || revised.MinimumCoverage != .95 || revised.ContradictionPolicy != "FAIL" || revised.Status != EvidenceContractDraft {
		t.Fatalf("revised evidence check = %#v", revised)
	}

	now = now.Add(time.Minute)
	program, err = service.TransitionEvidenceContract(ctx, TransitionEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ContractID: revised.ID,
		ExpectedVersion: program.Program.Version, ExpectedContractVersion: revised.Version,
		To: EvidenceContractActive, Rationale: "The current evidence rules are ready for use.", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	active := program.EvidenceContracts[0]
	if active.Status != EvidenceContractActive || active.Version != 3 {
		t.Fatalf("active evidence check = %#v", active)
	}

	historical, err := service.ProgramAt(ctx, "bank", program.Program.ID, createdAt.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(historical.EvidenceContracts) != 1 || historical.EvidenceContracts[0].Version != 1 || historical.EvidenceContracts[0].Status != EvidenceContractDraft || historical.EvidenceContracts[0].Name != "Current evidence" {
		t.Fatalf("historical evidence check = %#v", historical.EvidenceContracts)
	}

	now = now.Add(time.Minute)
	program, err = service.TransitionEvidenceContract(ctx, TransitionEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ContractID: active.ID,
		ExpectedVersion: program.Program.Version, ExpectedContractVersion: active.Version,
		To: EvidenceContractRetired, Rationale: "The filing process now uses a replacement check.", ActorID: "program-owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	retired := program.EvidenceContracts[0]
	if retired.Status != EvidenceContractRetired || retired.Version != 4 {
		t.Fatalf("retired evidence check = %#v", retired)
	}
	if _, err = service.ReviseEvidenceContract(ctx, ReviseEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ContractID: retired.ID,
		ExpectedVersion: program.Program.Version, ExpectedContractVersion: retired.Version,
		Name: retired.Name, Claim: retired.Claim, PopulationScope: retired.PopulationScope, FreshnessMinutes: retired.FreshnessMinutes,
		MinimumCoverage: retired.MinimumCoverage, ContradictionPolicy: retired.ContradictionPolicy, FailureAction: retired.FailureAction,
		Rationale: "Invalid retired correction.", ActorID: "program-owner",
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("retired revision error = %v", err)
	}
}

func programWithPlannedSafeguard(t *testing.T) (*Service, context.Context, ProgramAggregate) {
	t.Helper()
	service := NewService(NewMemoryRepository())
	ctx := WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-1", Code: "SAFE", Name: "Safeguard", Type: "ASSURANCE", OwningFunction: "Risk", OwnerPrincipalID: "program-owner", EffectiveFrom: now, ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "OBJ", Name: "Objective", Outcome: "The objective is achieved.", Status: ObjectiveActive, ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: program.ControlObjectives[0].ID, Name: "Safeguard", Description: "Operate the safeguard.", ImplementationType: "REVIEW", OwnerPrincipalID: "control-owner", Scope: json.RawMessage(`{}`), EffectiveFrom: now, ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx, program
}
