package continuity

import (
	"fmt"
	"strings"
)

func allowedMatterTransition(from, to MatterStatus) bool {
	allowed := map[MatterStatus]map[MatterStatus]bool{
		MatterDraft: {
			MatterInitialReview: true,
			MatterCancelled:     true,
		},
		MatterInitialReview: {
			MatterAssessment: true,
			MatterCancelled:  true,
		},
		MatterAssessment: {
			MatterDecisionRequired:    true,
			MatterActionsInProgress:   true,
			MatterResponsePreparation: true,
			MatterVerification:        true,
			MatterCancelled:           true,
		},
		MatterDecisionRequired: {
			MatterActionsInProgress:   true,
			MatterResponsePreparation: true,
			MatterVerification:        true,
			MatterCancelled:           true,
		},
		MatterActionsInProgress: {
			MatterVerification:        true,
			MatterResponsePreparation: true,
			MatterCancelled:           true,
		},
		MatterResponsePreparation: {
			MatterVerification: true,
			MatterCancelled:    true,
		},
		MatterVerification: {
			MatterClosed:              true,
			MatterDecisionRequired:    true,
			MatterActionsInProgress:   true,
			MatterResponsePreparation: true,
			MatterCancelled:           true,
		},
		MatterClosed: {
			MatterAssessment: true,
		},
	}
	return allowed[from][to]
}

func assessClosure(aggregate MatterAggregate) ClosureAssessment {
	reasons := make([]string, 0, 8)

	openActions := 0
	implementedActions := 0
	for _, action := range aggregate.Actions {
		switch action.Status {
		case ActionImplemented, ActionCancelled:
			if action.Status == ActionImplemented {
				implementedActions++
			}
		default:
			openActions++
		}
	}
	if openActions > 0 {
		reasons = append(reasons, fmt.Sprintf("%d action(s) are still open.", openActions))
	}

	latestResults := make(map[string]VerificationResult)
	for _, result := range aggregate.VerificationResults {
		current, ok := latestResults[result.ContractID]
		if !ok || result.ObservedAt.After(current.ObservedAt) {
			latestResults[result.ContractID] = result
		}
	}
	activeContracts := 0
	failedContracts := 0
	missingResults := 0
	for _, contract := range aggregate.VerificationContracts {
		if contract.Status != VerificationActive {
			continue
		}
		activeContracts++
		result, ok := latestResults[contract.ID]
		if !ok {
			missingResults++
			continue
		}
		if result.Result != VerificationPassed {
			failedContracts++
		}
	}
	if missingResults > 0 {
		reasons = append(reasons, fmt.Sprintf("%d outcome check(s) have no result.", missingResults))
	}
	if failedContracts > 0 {
		reasons = append(reasons, fmt.Sprintf("%d outcome check(s) did not pass.", failedContracts))
	}

	approvedDecision := false
	expiringDecision := false
	for _, decision := range aggregate.Decisions {
		if decision.Status == DecisionApproved || decision.Status == DecisionConditionallyApproved {
			approvedDecision = true
			if decision.ExpiresAt != nil || len(strings.TrimSpace(string(decision.Conditions))) > 2 {
				expiringDecision = true
			}
		}
	}
	acknowledgedResponse := false
	for _, response := range aggregate.ResponsePackages {
		if response.Status == ResponseAcknowledged {
			acknowledgedResponse = true
		}
	}

	requiresVerification := map[MatterType]bool{
		MatterSupervisoryFinding:    true,
		MatterRiskSituation:         true,
		MatterControlGap:            true,
		MatterAuditFinding:          true,
		MatterIncident:              true,
		MatterOperationalLoss:       true,
		MatterDataBreach:            true,
		MatterVendorDeficiency:      true,
		MatterCustomerConcern:       true,
		MatterOverdueObligation:     true,
		MatterFailedVerification:    true,
		MatterEvidenceContradiction: true,
		MatterKRIBreach:             true,
	}

	switch aggregate.Matter.Type {
	case MatterAuthorityRequest:
		if !acknowledgedResponse {
			reasons = append(reasons, "The external response has not been acknowledged.")
		}
	case MatterRegulatoryChange:
		if !approvedDecision {
			reasons = append(reasons, "The regulatory position has not been approved.")
		}
		noChangeRequired := false
		for _, decision := range aggregate.Decisions {
			if (decision.Status == DecisionApproved || decision.Status == DecisionConditionallyApproved) && strings.EqualFold(strings.TrimSpace(decision.SelectedOption), "NO_CHANGE_REQUIRED") {
				noChangeRequired = true
			}
		}
		if !noChangeRequired && activeContracts == 0 {
			reasons = append(reasons, "No outcome check has been defined for the required change.")
		}
		if implementedActions == 0 && !noChangeRequired {
			reasons = append(reasons, "No implemented change has been recorded.")
		}
	case MatterException:
		if !approvedDecision {
			reasons = append(reasons, "The exception has not been approved.")
		} else if !expiringDecision {
			reasons = append(reasons, "The exception approval has no expiry or conditions.")
		}
	default:
		if requiresVerification[aggregate.Matter.Type] && activeContracts == 0 {
			reasons = append(reasons, "No outcome check has been defined.")
		}
	}

	return ClosureAssessment{Ready: len(reasons) == 0, Reasons: reasons}
}

func allowedActionTransition(from, to ActionStatus) bool {
	allowed := map[ActionStatus]map[ActionStatus]bool{
		ActionPlanned: {
			ActionInProgress: true,
			ActionBlocked:    true,
			ActionCancelled:  true,
		},
		ActionInProgress: {
			ActionImplemented: true,
			ActionBlocked:     true,
			ActionCancelled:   true,
		},
		ActionBlocked: {
			ActionInProgress: true,
			ActionCancelled:  true,
		},
	}
	return allowed[from][to]
}

func allowedResponseTransition(from, to ResponseStatus) bool {
	allowed := map[ResponseStatus]map[ResponseStatus]bool{
		ResponseDraft: {
			ResponseInReview:  true,
			ResponseWithdrawn: true,
		},
		ResponseInReview: {
			ResponseApproved:  true,
			ResponseRejected:  true,
			ResponseDraft:     true,
			ResponseWithdrawn: true,
		},
		ResponseApproved: {
			ResponseTransmitted: true,
			ResponseWithdrawn:   true,
		},
		ResponseTransmitted: {
			ResponseAcknowledged: true,
		},
		ResponseRejected: {
			ResponseDraft: true,
		},
	}
	return allowed[from][to]
}
