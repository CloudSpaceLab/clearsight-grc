package continuity

import (
	"fmt"
	"strings"
)

func CurrentDecisionForType(values []Decision, decisionType string) *Decision {
	var current *Decision
	for index := range values {
		candidate := values[index]
		if !strings.EqualFold(strings.TrimSpace(candidate.Type), strings.TrimSpace(decisionType)) {
			continue
		}
		copy := candidate
		current = &copy
	}
	return current
}

func ValidateDecisionLifecycle(values []Decision, decisionType string, target DecisionStatus) error {
	current := CurrentDecisionForType(values, decisionType)
	if current == nil {
		if target == DecisionProposed || target == DecisionApproved || target == DecisionConditionallyApproved || target == DecisionRejected {
			return nil
		}
		return fmt.Errorf("%w: a decision must be proposed before %s", ErrInvalidState, target)
	}
	allowed := map[DecisionStatus]map[DecisionStatus]bool{
		DecisionProposed:              {DecisionInReview: true, DecisionChallenged: true, DecisionApproved: true, DecisionConditionallyApproved: true, DecisionRejected: true, DecisionReturned: true, DecisionSuperseded: true},
		DecisionInReview:              {DecisionChallenged: true, DecisionApproved: true, DecisionConditionallyApproved: true, DecisionRejected: true, DecisionReturned: true, DecisionSuperseded: true},
		DecisionChallenged:            {DecisionApproved: true, DecisionConditionallyApproved: true, DecisionRejected: true, DecisionReturned: true, DecisionSuperseded: true},
		DecisionReturned:              {DecisionProposed: true, DecisionSuperseded: true},
		DecisionRejected:              {DecisionProposed: true, DecisionSuperseded: true},
		DecisionApproved:              {DecisionProposed: true, DecisionSuperseded: true, DecisionExpired: true},
		DecisionConditionallyApproved: {DecisionProposed: true, DecisionSuperseded: true, DecisionExpired: true},
		DecisionExpired:               {DecisionProposed: true, DecisionSuperseded: true},
		DecisionSuperseded:            {DecisionProposed: true},
	}
	if !allowed[current.Status][target] {
		return fmt.Errorf("%w: decision %s cannot move from %s to %s", ErrInvalidState, strings.TrimSpace(decisionType), current.Status, target)
	}
	return nil
}

func setDecisionActor(value *Decision, status DecisionStatus, actorID string) {
	actorID = strings.TrimSpace(actorID)
	value.ProposedBy = ""
	value.ReviewedBy = ""
	value.ChallengedBy = ""
	if status != DecisionApproved && status != DecisionConditionallyApproved && status != DecisionRejected && status != DecisionExpired && status != DecisionSuperseded {
		value.AuthorityPrincipalID = ""
	}
	switch status {
	case DecisionProposed:
		value.ProposedBy = actorID
	case DecisionInReview, DecisionReturned:
		value.ReviewedBy = actorID
	case DecisionChallenged:
		value.ChallengedBy = actorID
	default:
		value.AuthorityPrincipalID = actorID
	}
}

func setResponseActor(response *ResponsePackage, target ResponseStatus, actorID string) {
	actorID = strings.TrimSpace(actorID)
	switch target {
	case ResponseDraft:
		response.PreparedBy = actorID
	case ResponseInReview:
		response.ReviewedBy = actorID
	case ResponseRejected:
		response.RejectedBy = actorID
	case ResponseWithdrawn:
		response.WithdrawnBy = actorID
	case ResponseApproved:
		response.ApprovedBy = actorID
	case ResponseTransmitted:
		response.TransmittedBy = actorID
	case ResponseAcknowledged:
		response.AcknowledgedBy = actorID
	}
}
