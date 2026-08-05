package bankverticals

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func authorityEvidenceRequest(config SeedConfig, matterID string) evidence.CreateRequestInput {
	return evidence.CreateRequestInput{
		TenantID:         config.TenantID,
		SubjectType:      "MATTER",
		SubjectID:        matterID,
		Title:            "Provide incident containment and customer communication records",
		Purpose:          "Complete the restricted NDPC response package.",
		WhyYou:           "You own the incident response records requested by Regulatory Affairs.",
		Sensitivity:      "RESTRICTED",
		AudienceType:     "INTERNAL",
		EstimatedMinutes: 10,
		Deadline:         config.Now.Add(48 * time.Hour),
		KnownFacts:       map[string]string{"case_reference": "NDPC/ENF/2026/0142", "incident_reference": "PRI-2026-008"},
		Fields: []evidence.Field{
			{ID: "containment_record", Label: "Containment record", Type: "FILE", Required: true, AcceptedFormats: []string{"application/pdf", "text/csv"}},
			{ID: "communication_decision", Label: "Customer communication decision", Type: "TEXT", Required: true, Description: "State the approved decision and approving authority."},
		},
		CreatedBy: config.ActorID,
	}
}

func referenceMatter(matter continuity.Matter, code Code) bool {
	return scopeString(matter.Scope, "sample") == "true" && scopeString(matter.Scope, "journey_code") == string(code)
}

func advanceMatter(ctx context.Context, service *continuity.Service, config SeedConfig, matter continuity.MatterAggregate, target continuity.MatterStatus, rationale string) (continuity.MatterAggregate, error) {
	if statusAtLeast(matter.Matter.Status, target) {
		return matter, nil
	}
	return service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: config.ActorID, Rationale: rationale})
}

func hasActiveVerificationContract(matter continuity.MatterAggregate, actionID string) bool {
	return activeVerificationContract(matter, actionID) != nil
}

func activeVerificationContract(matter continuity.MatterAggregate, actionID string) *continuity.VerificationContract {
	for index := range matter.VerificationContracts {
		value := matter.VerificationContracts[index]
		if value.Status == continuity.VerificationActive && (actionID == "" || value.ActionID == actionID) {
			copy := value
			return &copy
		}
	}
	return nil
}

func contractHasIndependentPass(matter continuity.MatterAggregate, contract continuity.VerificationContract) bool {
	for _, result := range matter.VerificationResults {
		if result.ContractID == contract.ID && result.Result == continuity.VerificationPassed && result.ReviewerPrincipalID == contract.AuthorityPrincipalID {
			return true
		}
	}
	return false
}

func responseRank(status continuity.ResponseStatus) int {
	switch status {
	case continuity.ResponseDraft:
		return 0
	case continuity.ResponseInReview:
		return 1
	case continuity.ResponseApproved:
		return 2
	case continuity.ResponseTransmitted:
		return 3
	case continuity.ResponseAcknowledged:
		return 4
	default:
		return -1
	}
}

func actionRank(status continuity.ActionStatus) int {
	switch status {
	case continuity.ActionPlanned:
		return 0
	case continuity.ActionInProgress:
		return 1
	case continuity.ActionImplemented:
		return 2
	default:
		return -1
	}
}
