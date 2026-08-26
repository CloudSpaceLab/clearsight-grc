package continuity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

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
		ContradictionPolicy: "REVIEW", FailureAction: "MATTER", Status: EvidenceContractActive,
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
		ContradictionPolicy: "REVIEW", FailureAction: "FLAG", Status: EvidenceContractActive,
	})
	if err == nil || !strings.Contains(err.Error(), "linked issue") {
		t.Fatalf("unsupported failure action error = %v", err)
	}
}
