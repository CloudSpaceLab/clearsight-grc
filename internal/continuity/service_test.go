package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestProgramStateMovesFromUnknownToCurrentAndTriggerCreatesOneMatter(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "NDPA", Name: "Data protection", Type: "PRIVACY", OwningFunction: "Privacy", OwnerPrincipalID: "owner", AuthorityPrincipalID: "dpo", Scope: json.RawMessage(`{"legal_entity":"Bank NG"}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if program.StateLabel != "Setup in progress" || program.CurrentState == nil || program.CurrentState.Overall != StateUnknown {
		t.Fatalf("expected setup program, got %#v", program)
	}

	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "ROPA", Title: "Keep processing records current", Statement: "The bank must maintain current processing records.", Modality: "MUST", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	requirement := program.Requirements[0]
	program, err = service.DetermineApplicability(ctx, DetermineApplicabilityInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: requirement.ID, Status: ApplicabilityApplicable, Scope: json.RawMessage(`{"entity":"Bank NG"}`), Rationale: "The bank processes personal data.", ApprovedBy: "dpo", EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "ROPA-CURRENT", Name: "Current processing records", Outcome: "Processing records reflect active systems, vendors and purposes.", Status: ObjectiveActive})
	if err != nil {
		t.Fatal(err)
	}
	objective := program.ControlObjectives[0]
	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: objective.ID, Name: "Quarterly owner review", Description: "Processing owners review changed records each quarter.", ImplementationType: "REVIEW", Status: ImplementationImplemented, EffectiveFrom: now, Scope: json.RawMessage(`{"entity":"Bank NG"}`)})
	if err != nil {
		t.Fatal(err)
	}
	implementation := program.ControlImplementations[0]
	program, err = service.LinkRequirementControl(ctx, LinkRequirementControlInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: requirement.ID, ImplementationID: implementation.ID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ControlImplementationID: implementation.ID, Code: "ROPA-COVERAGE", Name: "Processing record coverage", Claim: "Every active processing activity has a current owner-approved record.", PopulationScope: json.RawMessage(`{"population":"active_processing_activities"}`), FreshnessMinutes: 43200, MinimumCoverage: 0.95, ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive})
	if err != nil {
		t.Fatal(err)
	}
	contract := program.EvidenceContracts[0]
	validUntil := now.Add(30 * 24 * time.Hour)
	program, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ContractID: contract.ID, Conclusion: EvidenceSupported, Coverage: 1, Basis: json.RawMessage(`{"records":124}`), ValidUntil: &validUntil, AssessedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionProgram(ctx, ProgramTransitionInput{TenantID: "bank", ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: ProgramActive, ActorID: "dpo", Rationale: "Initial setup approved."})
	if err != nil {
		t.Fatal(err)
	}
	if program.CurrentState == nil || program.CurrentState.Overall != StateCurrent || program.StateLabel != "Up to date" {
		t.Fatalf("expected current program, got %#v", program.CurrentState)
	}

	trigger := Trigger{TenantID: "bank", ProgramID: program.Program.ID, Type: "EVIDENCE_CONTRADICTION", SubjectType: "EVIDENCE_CONTRACT", SubjectID: contract.ID, DedupeKey: "contract-contradiction-1", Payload: json.RawMessage(`{"summary":"Owner confirmation conflicts with the system record."}`), ObservedAt: now.Add(time.Hour), Source: "evidence-assessor"}
	program, matter, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted || matter == nil || matter.Type != MatterEvidenceContradiction {
		t.Fatalf("expected a new contradiction matter, got inserted=%v matter=%#v", inserted, matter)
	}
	if program.CurrentState == nil || program.CurrentState.Overall != StateAtRisk || program.CurrentState.OpenMatterCount != 1 {
		t.Fatalf("expected program to need attention, got %#v", program.CurrentState)
	}
	_, duplicateMatter, inserted, err := service.ApplyTrigger(ctx, trigger)
	if err != nil {
		t.Fatal(err)
	}
	if inserted || duplicateMatter == nil || duplicateMatter.ID != matter.ID {
		t.Fatalf("expected duplicate trigger to return the existing matter, got inserted=%v matter=%#v", inserted, duplicateMatter)
	}
	matters, err := service.ListMatters(ctx, "bank", "", 20)
	if err != nil || len(matters) != 1 {
		t.Fatalf("expected one matter, got %d err=%v", len(matters), err)
	}
}

func TestMatterCannotCloseUntilActionOutcomeIsVerified(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", Type: MatterControlGap, Priority: 4, Title: "Restore privileged-access review evidence", Summary: "Four accounts do not have current business-need evidence.", Scope: json.RawMessage(`{"application":"Treasury platform"}`), KnownFacts: json.RawMessage(`{"affected_accounts":4}`)})
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []MatterStatus{MatterInitialReview, MatterAssessment, MatterActionsInProgress} {
		matter, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: state, Rationale: "Progressing the review."})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddAction(ctx, AddActionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Obtain current account-owner confirmation", Description: "Confirm current business need for the four unresolved accounts."})
	if err != nil {
		t.Fatal(err)
	}
	action := matter.Actions[0]
	matter, err = service.TransitionAction(ctx, TransitionActionInput{TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: ActionInProgress, Rationale: "Owner confirmations requested."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionAction(ctx, TransitionActionInput{TenantID: "bank", MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: ActionImplemented, Rationale: "All confirmations received."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: action.ID, ExpectedOutcome: "All privileged accounts have current approved business-need evidence.", Baseline: json.RawMessage(`{"unresolved":4}`), Scope: json.RawMessage(`{"accounts":4}`), Threshold: json.RawMessage(`{"unresolved":0}`), FailureResponse: "REOPEN"})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: MatterVerification, Rationale: "Implementation is complete; check the result."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: MatterClosed, ActorID: "reviewer", Rationale: "Close."}); !errors.Is(err, ErrClosureBlocked) {
		t.Fatalf("expected closure to be blocked, got %v", err)
	}
	contract := matter.VerificationContracts[0]
	matter, err = service.RecordVerificationResult(ctx, RecordVerificationResultInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: contract.ID, Result: VerificationPassed, Observations: json.RawMessage(`{"unresolved":0}`), EvidenceReferences: json.RawMessage(`[]`), ReviewerPrincipalID: "reviewer", Rationale: "IAM report shows no unresolved accounts.", ObservedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: MatterClosed, ActorID: "reviewer", Rationale: "The defined outcome passed."})
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.Status != MatterClosed || matter.StatusLabel != "Closed" || matter.Matter.ClosedAt == nil {
		t.Fatalf("expected closed matter, got %#v", matter.Matter)
	}
	matter, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "bank", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: MatterAssessment, ActorID: "reviewer", Rationale: "A new related fact requires review."})
	if err != nil || matter.Matter.ReopenCount != 1 || matter.Matter.ClosedAt != nil {
		t.Fatalf("expected reopened matter, got %#v err=%v", matter.Matter, err)
	}
}

func TestPointInTimeReplayDoesNotIncludeLaterChanges(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "CYBER", Name: "Cybersecurity", Type: "CYBER", OwningFunction: "Information Security", OwnerPrincipalID: "owner", AuthorityPrincipalID: "authority", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := now.Add(time.Minute)
	service.now = func() time.Time { return now.Add(2 * time.Minute) }
	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "ACCESS", Title: "Review privileged access", Statement: "Privileged access must be reviewed.", Modality: "MUST", Status: RequirementApproved, EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	historical, err := service.ProgramAt(ctx, "bank", program.Program.ID, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if len(historical.Requirements) != 0 || historical.Program.Version >= program.Program.Version {
		t.Fatalf("historical state includes later change: %#v", historical)
	}
}

func TestMatterCanLinkToMoreThanOneProgramWithoutDuplicateLinks(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	first, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", Code: "VENDOR", Name: "Vendor assurance", Type: "THIRD_PARTY", OwningFunction: "Procurement", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", Type: MatterVendorDeficiency, Priority: 3, Title: "Replace an expired vendor certificate", Summary: "The current certificate is no longer valid.", Scope: json.RawMessage(`{}`), ProgramID: first.Program.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(matter.Links) != 1 {
		t.Fatalf("expected first link, got %#v", matter.Links)
	}
	matter, err = service.AddMatterLink(ctx, AddMatterLinkInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ProgramID: second.Program.ID, Relationship: "AFFECTS"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matter.Links) != 2 {
		t.Fatalf("expected two program links, got %#v", matter.Links)
	}
	version := matter.Matter.Version
	matter, err = service.AddMatterLink(ctx, AddMatterLinkInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: version, ProgramID: second.Program.ID, Relationship: "AFFECTS"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matter.Links) != 2 || matter.Matter.Version != version {
		t.Fatalf("identical link should be idempotent: %#v", matter)
	}
}
