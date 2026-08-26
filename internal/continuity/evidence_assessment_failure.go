package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func evidenceAssessmentNeedsFailureAction(value EvidenceAssessment, contract EvidenceContract) bool {
	return value.Conclusion != EvidenceSupported || value.Coverage < contract.MinimumCoverage
}

func (s *Service) recordEvidenceAssessmentWithFailure(ctx context.Context, aggregate ProgramAggregate, contract EvidenceContract, assessment EvidenceAssessment, repo EvidenceAssessmentFailureRepository) (ProgramAggregate, error) {
	now := assessment.CreatedAt
	programEvent, err := newEvent(assessment.TenantID, "PROGRAM", assessment.ProgramID, aggregate.Program.Version+1, EventEvidenceAssessmentRecorded, assessment, actorFor(assessment.AssessedBy), assessment.AssessedBy, now)
	if err != nil {
		return ProgramAggregate{}, err
	}
	matterID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	triggerKey := "program-evidence-failure:" + assessment.ProgramID + ":" + assessment.ContractID
	matterType := MatterControlGap
	triggerType := "EVIDENCE_CHECK_FAILED"
	title := "Resolve failed evidence check: " + contract.Name
	summary := fmt.Sprintf("The %s result did not support the claim: %s", strings.ToLower(strings.ReplaceAll(string(assessment.Conclusion), "_", " ")), contract.Claim)
	requiredAuthority := "CONTROL_ASSURANCE"
	missingFacts := json.RawMessage(`[]`)
	if contract.FailureAction == "REQUEST" {
		triggerKey = "program-evidence-request:" + assessment.ProgramID + ":" + assessment.ContractID
		matterType = MatterAuthorityRequest
		triggerType = "EVIDENCE_REQUEST_REQUIRED"
		title = "Provide evidence for check: " + contract.Name
		summary = "The latest result did not support the evidence claim. The Program owner must provide or obtain current evidence for review."
		requiredAuthority = "EVIDENCE_OWNER"
		missingFacts, _ = json.Marshal([]string{"evidence that supports: " + contract.Claim})
	}
	priority := 3
	if assessment.Conclusion == EvidenceUnsupported || assessment.Conclusion == EvidenceContradicted {
		priority = 4
	}
	scope, _ := json.Marshal(map[string]any{
		"program_id": assessment.ProgramID, "evidence_contract_id": assessment.ContractID,
		"evidence_assessment_id": assessment.ID, "population_scope": json.RawMessage(contract.PopulationScope),
	})
	known, _ := json.Marshal(map[string]any{
		"evidence_check": contract.Name, "claim": contract.Claim, "conclusion": assessment.Conclusion,
		"coverage": assessment.Coverage, "required_coverage": contract.MinimumCoverage, "assessed_at": assessment.AssessedAt,
	})
	matter := Matter{
		ID: matterID, TenantID: assessment.TenantID, LegalEntityID: aggregate.Program.LegalEntityID,
		Reference: matterReference(matterID), Type: matterType, Status: MatterInitialReview, Priority: priority,
		Title: title, Summary: summary,
		Scope: scope, SourceType: "EVIDENCE_CONTRACT", SourceID: contract.ID,
		TriggerType: triggerType, TriggerID: assessment.ID, TriggerKey: triggerKey,
		KnownFacts: known, MissingFacts: missingFacts, Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: aggregate.Program.OwnerPrincipalID, RequiredAuthority: requiredAuthority,
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	matterEvent, err := newEvent(assessment.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, actorFor(assessment.AssessedBy), assessment.AssessedBy, now)
	if err != nil {
		return ProgramAggregate{}, err
	}
	linkID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramAggregate{}, err
	}
	link := MatterLink{
		ID: linkID, TenantID: assessment.TenantID, MatterID: matter.ID, ProgramID: assessment.ProgramID,
		RequirementID: contract.RequirementID, ControlID: contract.ControlImplementationID,
		Relationship: "AFFECTS", CreatedAt: now,
	}
	linkEvent, err := newEvent(assessment.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, actorFor(assessment.AssessedBy), assessment.AssessedBy, now)
	if err != nil {
		return ProgramAggregate{}, err
	}
	_, err = repo.RecordEvidenceAssessmentWithFailure(ctx, EvidenceAssessmentFailureBundle{
		TenantID: assessment.TenantID, ProgramID: assessment.ProgramID, ExpectedVersion: aggregate.Program.Version,
		ProgramEvent: programEvent, Matter: matter, MatterEvent: matterEvent, Link: link, LinkEvent: linkEvent,
	})
	if err != nil {
		return ProgramAggregate{}, err
	}
	_ = s.requestProgramRefresh(ctx, assessment.TenantID, assessment.ProgramID, EventEvidenceAssessmentRecorded, assessment.ID, assessment.AssessedBy)
	if current, readErr := s.repo.GetProgram(ctx, assessment.TenantID, assessment.ProgramID); readErr == nil {
		return current, nil
	}
	// The material command is committed. A later current-read failure must not
	// turn that successful command into a false API failure.
	_ = applyProgramEventToAggregate(&aggregate, programEvent)
	aggregate.Program.Version = programEvent.AggregateVersion
	aggregate.Program.UpdatedAt = programEvent.OccurredAt
	return decorateProgram(aggregate), nil
}
