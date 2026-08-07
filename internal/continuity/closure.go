package continuity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func allowedMatterTransition(from, to MatterStatus) bool {
	allowed := map[MatterStatus]map[MatterStatus]bool{
		MatterDraft: {MatterInitialReview: true, MatterCancelled: true},
		MatterInitialReview: {MatterAssessment: true, MatterCancelled: true},
		MatterAssessment: {MatterDecisionRequired: true, MatterActionsInProgress: true, MatterResponsePreparation: true, MatterVerification: true, MatterCancelled: true},
		MatterDecisionRequired: {MatterActionsInProgress: true, MatterResponsePreparation: true, MatterVerification: true, MatterCancelled: true},
		MatterActionsInProgress: {MatterVerification: true, MatterResponsePreparation: true, MatterCancelled: true},
		MatterResponsePreparation: {MatterVerification: true, MatterCancelled: true},
		MatterVerification: {MatterClosed: true, MatterDecisionRequired: true, MatterActionsInProgress: true, MatterResponsePreparation: true, MatterCancelled: true},
		MatterClosed: {MatterAssessment: true},
	}
	return allowed[from][to]
}

func assessClosure(aggregate MatterAggregate) ClosureAssessment {
	return assessClosureAt(aggregate, time.Now().UTC())
}

func assessClosureAt(aggregate MatterAggregate, now time.Time) ClosureAssessment {
	now = now.UTC()
	reasons := make([]string, 0, 10)

	openActions := 0
	implementedActions := 0
	for _, action := range aggregate.Actions {
		switch action.Status {
		case ActionImplemented:
			implementedActions++
		case ActionCancelled:
		default:
			openActions++
		}
	}
	if openActions > 0 {
		reasons = append(reasons, fmt.Sprintf("%d action(s) are still open.", openActions))
	}

	latestResults := latestVerificationResults(aggregate.VerificationResults)
	activeContracts := 0
	missingResults := 0
	failedResults := 0
	invalidResults := 0
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
			failedResults++
			continue
		}
		if err := validateVerificationResult(aggregate, contract, result.ReviewerPrincipalID, result.ObservedAt, now); err != nil {
			invalidResults++
		}
	}
	if missingResults > 0 {
		reasons = append(reasons, fmt.Sprintf("%d outcome check(s) have no result.", missingResults))
	}
	if failedResults > 0 {
		reasons = append(reasons, fmt.Sprintf("%d outcome check(s) did not pass.", failedResults))
	}
	if invalidResults > 0 {
		reasons = append(reasons, fmt.Sprintf("%d passing outcome check(s) are not yet valid for closure.", invalidResults))
	}

	decisions := currentDecisions(aggregate.Decisions)
	approvedDecision := false
	adverseCurrentDecision := false
	noChangeRequired := false
	validExceptionDecision := false
	for _, decision := range decisions {
		if !decisionQualifies(decision, now) {
			adverseCurrentDecision = true
			continue
		}
		approvedDecision = true
		if strings.EqualFold(strings.TrimSpace(decision.SelectedOption), "NO_CHANGE_REQUIRED") {
			noChangeRequired = true
		}
		if exceptionDecisionQualifies(decision, now) {
			validExceptionDecision = true
		}
	}

	responses := currentResponses(aggregate.ResponsePackages)
	acknowledgedResponse := len(responses) > 0
	for _, response := range responses {
		if response.Status != ResponseAcknowledged || response.TransmittedAt == nil || response.AcknowledgedAt == nil || response.AcknowledgedAt.Before(*response.TransmittedAt) {
			acknowledgedResponse = false
			break
		}
	}

	requiresVerification := map[MatterType]bool{
		MatterSupervisoryFinding: true, MatterRiskSituation: true, MatterControlGap: true,
		MatterAuditFinding: true, MatterIncident: true, MatterOperationalLoss: true,
		MatterDataBreach: true, MatterVendorDeficiency: true, MatterCustomerConcern: true,
		MatterOverdueObligation: true, MatterFailedVerification: true,
		MatterEvidenceContradiction: true, MatterKRIBreach: true,
	}

	switch aggregate.Matter.Type {
	case MatterAuthorityRequest:
		if !acknowledgedResponse {
			reasons = append(reasons, "Every current external response must be transmitted and acknowledged before closure.")
		}
	case MatterRegulatoryChange:
		if !approvedDecision || adverseCurrentDecision {
			reasons = append(reasons, "Every current regulatory decision must be resolved and at least one current position must be approved.")
		}
		if !noChangeRequired && activeContracts == 0 {
			reasons = append(reasons, "No outcome check has been defined for the required change.")
		}
		if implementedActions == 0 && !noChangeRequired {
			reasons = append(reasons, "No implemented change has been recorded.")
		}
	case MatterException:
		if !validExceptionDecision || adverseCurrentDecision {
			reasons = append(reasons, "The current exception approval is expired, unresolved or missing an enforceable expiry/condition.")
		}
	default:
		if requiresVerification[aggregate.Matter.Type] && activeContracts == 0 {
			reasons = append(reasons, "No outcome check has been defined.")
		}
	}

	return ClosureAssessment{Ready: len(reasons) == 0, Reasons: reasons}
}

func currentDecisions(values []Decision) []Decision {
	current := make(map[string]Decision)
	for _, decision := range values {
		key := strings.ToUpper(strings.TrimSpace(decision.Type))
		if key == "" {
			key = "__UNSPECIFIED__"
		}
		previous, ok := current[key]
		if !ok || decisionAfter(decision, previous) {
			current[key] = decision
		}
	}
	result := make([]Decision, 0, len(current))
	for _, decision := range current {
		result = append(result, decision)
	}
	return result
}

func decisionAfter(left, right Decision) bool {
	leftAt := decisionCurrentAt(left)
	rightAt := decisionCurrentAt(right)
	if leftAt.Equal(rightAt) {
		return left.ID > right.ID
	}
	return leftAt.After(rightAt)
}

func newerDecision(left, right Decision) bool { return decisionAfter(left, right) }

func decisionCurrentAt(value Decision) time.Time {
	latest := value.CreatedAt
	if value.DecidedAt != nil && value.DecidedAt.After(latest) {
		latest = *value.DecidedAt
	}
	if value.UpdatedAt.After(latest) {
		latest = value.UpdatedAt
	}
	return latest
}

func decisionQualifies(decision Decision, now time.Time) bool {
	if decision.Status != DecisionApproved && decision.Status != DecisionConditionallyApproved {
		return false
	}
	if decision.ExpiresAt != nil && !now.Before(*decision.ExpiresAt) {
		return false
	}
	if decision.Status == DecisionConditionallyApproved && !decisionConditionsSatisfied(decision.Conditions) {
		return false
	}
	return true
}

func exceptionDecisionQualifies(decision Decision, now time.Time) bool {
	if !decisionQualifies(decision, now) {
		return false
	}
	if decision.ExpiresAt != nil && now.Before(*decision.ExpiresAt) {
		return true
	}
	return decisionConditionsSatisfied(decision.Conditions)
}

func decisionConditionsSatisfied(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "[]" || trimmed == "{}" {
		return false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return conditionValueSatisfied(value)
}

func conditionValueSatisfied(value any) bool {
	switch typed := value.(type) {
	case []any:
		if len(typed) == 0 {
			return false
		}
		for _, item := range typed {
			if !conditionValueSatisfied(item) {
				return false
			}
		}
		return true
	case map[string]any:
		if resolved, ok := typed["resolved"].(bool); ok {
			return resolved
		}
		if satisfied, ok := typed["satisfied"].(bool); ok {
			return satisfied
		}
		if status, ok := typed["status"].(string); ok {
			switch strings.ToUpper(strings.TrimSpace(status)) {
			case "SATISFIED", "WAIVED", "COMPLETE", "COMPLETED", "RESOLVED":
				return true
			default:
				return false
			}
		}
		if nested, ok := typed["conditions"]; ok {
			return conditionValueSatisfied(nested)
		}
		return false
	default:
		return false
	}
}

func currentResponses(values []ResponsePackage) []ResponsePackage {
	current := make(map[string]ResponsePackage)
	for _, response := range values {
		key := strings.ToUpper(strings.TrimSpace(response.Purpose)) + "\x00" + strings.ToUpper(strings.TrimSpace(response.Audience))
		previous, ok := current[key]
		if !ok || responseAfter(response, previous) {
			current[key] = response
		}
	}
	result := make([]ResponsePackage, 0, len(current))
	for _, response := range current {
		result = append(result, response)
	}
	return result
}

func responseAfter(left, right ResponsePackage) bool {
	leftAt := left.UpdatedAt
	if leftAt.IsZero() {
		leftAt = left.CreatedAt
	}
	rightAt := right.UpdatedAt
	if rightAt.IsZero() {
		rightAt = right.CreatedAt
	}
	if leftAt.Equal(rightAt) {
		return left.ID > right.ID
	}
	return leftAt.After(rightAt)
}

func latestVerificationResults(values []VerificationResult) map[string]VerificationResult {
	latest := make(map[string]VerificationResult)
	for _, result := range values {
		current, ok := latest[result.ContractID]
		if !ok || verificationResultAfter(result, current) {
			latest[result.ContractID] = result
		}
	}
	return latest
}

func verificationResultAfter(left, right VerificationResult) bool {
	if !left.ObservedAt.Equal(right.ObservedAt) {
		return left.ObservedAt.After(right.ObservedAt)
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.After(right.CreatedAt)
	}
	return left.ID > right.ID
}

func validateVerificationResult(aggregate MatterAggregate, contract VerificationContract, reviewer string, observedAt, now time.Time) error {
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		return fmt.Errorf("reviewer_principal_id is required")
	}
	if observedAt.IsZero() {
		return fmt.Errorf("observed_at is required")
	}
	observedAt = observedAt.UTC()
	now = now.UTC()
	if observedAt.After(now) {
		return fmt.Errorf("observed_at cannot be in the future")
	}
	if contract.AuthorityPrincipalID != "" && reviewer != contract.AuthorityPrincipalID {
		return fmt.Errorf("reviewer is not the authority assigned to this outcome check")
	}

	anchor := contract.CreatedAt.UTC()
	if contract.ActionID != "" {
		action, ok := actionByID(aggregate.Actions, contract.ActionID)
		if !ok {
			return fmt.Errorf("the outcome check action no longer exists")
		}
		if action.Status != ActionImplemented || action.ImplementedAt == nil {
			return fmt.Errorf("the linked action must be implemented before outcome verification")
		}
		if action.OwnerPrincipalID != "" && reviewer == action.OwnerPrincipalID {
			return fmt.Errorf("the outcome reviewer must be independent of the action owner")
		}
		if action.ImplementedAt.After(anchor) {
			anchor = action.ImplementedAt.UTC()
		}
	}
	readyAt := anchor.Add(time.Duration(contract.ObservationPeriodMinutes) * time.Minute)
	if observedAt.Before(readyAt) {
		return fmt.Errorf("the outcome observation period has not completed")
	}
	return nil
}

func actionByID(values []Action, id string) (Action, bool) {
	for _, action := range values {
		if action.ID == id {
			return action, true
		}
	}
	return Action{}, false
}

func allowedActionTransition(from, to ActionStatus) bool {
	allowed := map[ActionStatus]map[ActionStatus]bool{
		ActionPlanned: {ActionInProgress: true, ActionBlocked: true, ActionCancelled: true},
		ActionInProgress: {ActionImplemented: true, ActionBlocked: true, ActionCancelled: true},
		ActionBlocked: {ActionInProgress: true, ActionCancelled: true},
	}
	return allowed[from][to]
}

func allowedResponseTransition(from, to ResponseStatus) bool {
	allowed := map[ResponseStatus]map[ResponseStatus]bool{
		ResponseDraft: {ResponseInReview: true, ResponseWithdrawn: true},
		ResponseInReview: {ResponseApproved: true, ResponseRejected: true, ResponseDraft: true, ResponseWithdrawn: true},
		ResponseApproved: {ResponseTransmitted: true, ResponseWithdrawn: true},
		ResponseTransmitted: {ResponseAcknowledged: true},
		ResponseRejected: {ResponseDraft: true},
	}
	return allowed[from][to]
}
