package continuity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func legacyEvidenceProgram(t *testing.T, failureAction string) (*MemoryRepository, *Service, ProgramAggregate, time.Time) {
	t.Helper()
	repo := NewMemoryRepository()
	service := NewService(repo)
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LEGACY-" + failureAction, Name: "Legacy evidence handling",
		Type: "COMPLIANCE", OwningFunction: "Compliance", OwnerPrincipalID: "program-owner", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ", Title: "Retain evidence", Statement: "Evidence must be retained.", Status: RequirementApproved, EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	contractID, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	contract := EvidenceContract{
		ID: contractID, TenantID: "bank", ProgramID: program.Program.ID, RequirementID: program.Requirements[0].ID,
		Code: "CHECK", Name: "Retention evidence", Claim: "Required evidence is retained.", PopulationScope: json.RawMessage(`{}`),
		FreshnessMinutes: 60, MinimumCoverage: 1, ContradictionPolicy: "REVIEW", FailureAction: failureAction,
		Status: EvidenceContractActive, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if err := service.applyProgramValue(ctx, "bank", program.Program.ID, program.Program.Version, EventEvidenceContractAdded, contract, "program-owner"); err != nil {
		t.Fatal(err)
	}
	program, err = service.refreshAndGetProgram(ctx, "bank", program.Program.ID, EventEvidenceContractAdded, contract.ID)
	if err != nil {
		t.Fatal(err)
	}
	return repo, service, program, now
}

func TestNonSupportingEvidenceAssessmentCreatesOneLinkedEntityScopedMatter(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy compliance",
		Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-approver",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ-1", Title: "File the return", Statement: "The annual return must be filed.", Status: RequirementApproved,
		EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		RequirementID: program.Requirements[0].ID, Code: "RETURN", Name: "Annual return filing",
		Claim: "The annual return was filed before the deadline.", FreshnessMinutes: 1440, MinimumCoverage: 1,
		ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractDraft,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.TransitionEvidenceContract(ctx, TransitionEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ContractID: program.EvidenceContracts[0].ID,
		ExpectedVersion: program.Program.Version, ExpectedContractVersion: program.EvidenceContracts[0].Version,
		To: EvidenceContractActive, Rationale: "Independent review approved the evidence rules.", ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	failed, err := service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceUnsupported, Coverage: .7,
		Basis: json.RawMessage(`{"missing_sections":3}`), AssessedBy: "reviewer-1", AssessedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failed.EvidenceAssessments) != 1 || failed.EvidenceAssessments[0].Conclusion != EvidenceUnsupported {
		t.Fatalf("assessment was not retained: %#v", failed.EvidenceAssessments)
	}
	if failed.CurrentState == nil || failed.CurrentState.OpenMatterCount != 1 || failed.CurrentState.ProgramVersion != failed.Program.Version {
		t.Fatalf("current projection did not include the failure issue: %#v", failed.CurrentState)
	}
	matters, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matters) != 1 {
		t.Fatalf("failure matters = %d, want 1", len(matters))
	}
	matter := matters[0]
	if matter.Matter.LegalEntityID != "entity-a" || matter.Matter.Type != MatterControlGap || matter.Matter.SourceID != program.EvidenceContracts[0].ID {
		t.Fatalf("failure matter scope/source = %#v", matter.Matter)
	}
	if len(matter.Links) != 1 || matter.Links[0].ProgramID != program.Program.ID || matter.Links[0].RequirementID != program.Requirements[0].ID {
		t.Fatalf("failure matter link = %#v", matter.Links)
	}
	programEvents, err := repo.ProgramEvents(ctx, "bank", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	matterEvents, err := repo.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if programEvents[len(programEvents)-1].Type != EventEvidenceAssessmentRecorded || len(matterEvents) != 2 || matterEvents[0].Type != EventMatterCreated || matterEvents[1].Type != EventMatterLinked {
		t.Fatalf("material history program=%#v matter=%#v", programEvents, matterEvents)
	}

	// A later failed result remains material history but reuses the open issue.
	now = now.Add(time.Hour)
	again, err := service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "bank", ProgramID: failed.Program.ID, ExpectedVersion: failed.Program.Version,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceContradicted, Coverage: .6,
		Basis: json.RawMessage(`{"contradiction":"register mismatch"}`), AssessedBy: "reviewer-1", AssessedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.EvidenceAssessments) != 2 {
		t.Fatalf("assessment history = %d, want 2", len(again.EvidenceAssessments))
	}
	matters, err = service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matters) != 1 {
		t.Fatalf("deduplicated failure matters = %d, want 1", len(matters))
	}

	// Retrying an already committed expected version cannot append either record.
	_, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "bank", ProgramID: failed.Program.ID, ExpectedVersion: failed.Program.Version,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceContradicted, Coverage: .6,
		Basis: json.RawMessage(`{}`), AssessedBy: "reviewer-1", AssessedAt: now,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("retry error = %v, want version conflict", err)
	}
}

func TestEvidenceFailureConfigurationIsNarrowedToExecutableMatterAction(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := WithTrustedSystemScope(t.Context())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "OPS", Name: "Operational compliance",
		Type: "COMPLIANCE", OwningFunction: "Compliance", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "REQ", Title: "Retain evidence", Statement: "Evidence must be retained.", Status: RequirementApproved, EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddEvidenceContract(ctx, AddEvidenceContractInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		RequirementID: program.Requirements[0].ID, Code: "CHECK", Name: "Retention evidence",
		Claim: "Required evidence is retained.", FreshnessMinutes: 60, MinimumCoverage: 1,
		ContradictionPolicy: "REVIEW", FailureAction: "FLAG", Status: EvidenceContractDraft,
	})
	if err == nil || !strings.Contains(err.Error(), "linked issue") {
		t.Fatalf("unsupported failure action error = %v", err)
	}
}

func TestLegacyFlagAndBlockEvidenceFailuresRetainAssessmentAndProgramAttention(t *testing.T) {
	for _, action := range []string{"FLAG", "BLOCK"} {
		t.Run(action, func(t *testing.T) {
			repo, service, program, now := legacyEvidenceProgram(t, action)
			result, err := service.RecordEvidenceAssessment(WithTrustedSystemScope(t.Context()), RecordEvidenceAssessmentInput{
				TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
				ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceUnsupported, Coverage: .4,
				Basis: json.RawMessage(`{"missing":2}`), AssessedBy: "reviewer-1", AssessedAt: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.EvidenceAssessments) != 1 || result.Program.Version != program.Program.Version+1 {
				t.Fatalf("legacy %s assessment = %#v", action, result)
			}
			if result.CurrentState == nil || result.CurrentState.Dimensions.EvidenceSufficiency == StateCurrent {
				t.Fatalf("legacy %s did not change Program attention state: %#v", action, result.CurrentState)
			}
			matters, err := service.ListMatters(WithTrustedSystemScope(t.Context()), "bank", "OPEN", 20)
			if err != nil {
				t.Fatal(err)
			}
			if len(matters) != 0 {
				t.Fatalf("legacy %s created an unconfigured issue: %#v", action, matters)
			}
			events, err := repo.ProgramEvents(WithTrustedSystemScope(t.Context()), "bank", program.Program.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if events[len(events)-1].Type != EventEvidenceAssessmentRecorded {
				t.Fatalf("legacy %s history = %#v", action, events)
			}
		})
	}
}

func TestLegacyRequestEvidenceFailureCreatesOneLinkedEvidenceRequest(t *testing.T) {
	_, service, program, now := legacyEvidenceProgram(t, "REQUEST")
	ctx := WithTrustedSystemScope(t.Context())
	result, err := service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceIndeterminate, Coverage: .2,
		Basis: json.RawMessage(`{"reason":"source unavailable"}`), AssessedBy: "reviewer-1", AssessedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	matters, err := service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matters) != 1 {
		t.Fatalf("request matters = %d, want 1", len(matters))
	}
	request := matters[0]
	if request.Matter.Type != MatterAuthorityRequest || request.Matter.LegalEntityID != "entity-a" || request.Matter.TriggerType != "EVIDENCE_REQUEST_REQUIRED" || request.Matter.RequiredAuthority != "EVIDENCE_OWNER" {
		t.Fatalf("request matter = %#v", request.Matter)
	}
	if len(request.Links) != 1 || request.Links[0].ProgramID != program.Program.ID || request.Links[0].RequirementID != program.Requirements[0].ID {
		t.Fatalf("request link = %#v", request.Links)
	}
	if !strings.Contains(string(request.Matter.MissingFacts), "Required evidence is retained") {
		t.Fatalf("request does not state the missing evidence: %s", request.Matter.MissingFacts)
	}
	_, err = service.RecordEvidenceAssessment(ctx, RecordEvidenceAssessmentInput{
		TenantID: "bank", ProgramID: result.Program.ID, ExpectedVersion: result.Program.Version,
		ContractID: program.EvidenceContracts[0].ID, Conclusion: EvidenceUnsupported, Coverage: .1,
		Basis: json.RawMessage(`{}`), AssessedBy: "reviewer-1", AssessedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	matters, err = service.ListMatters(ctx, "bank", "OPEN", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matters) != 1 {
		t.Fatalf("deduplicated request matters = %d, want 1", len(matters))
	}
}
