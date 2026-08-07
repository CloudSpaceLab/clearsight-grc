package continuity

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WorkRequirement is a derived, non-authoritative description of one human
// action that is safe to project into Workflow. Canonical Matter state remains
// the source of truth; this type only describes an executable next step.
type WorkRequirement struct {
	Key                 string
	CommandName         string
	TargetStatus        string
	Responsibility      string
	Materiality         int
	Title               string
	PrimaryAction       string
	WhyNow              string
	InterventionClass   string
	SubresourceType     string
	SubresourceID       string
	DueAt               *time.Time
	RequiredPrincipalID string
	Verification        *WorkVerificationContext
}

type WorkVerificationContext struct {
	ContractID        string
	ExpectedOutcome   string
	EvidenceState     string
	IndependentReview bool
}

type WorkAmbiguity struct {
	Key             string
	Title           string
	Reason          string
	SubresourceType string
	SubresourceID   string
	AllowedTargets  []string
}

type LifecyclePolicy struct {
	Responsibility string
	Materiality    int
}

// DecisionLifecyclePolicy is shared by command authorization and work
// compilation so the UI projection cannot drift from the executable command
// boundary.
func DecisionLifecyclePolicy(target DecisionStatus) (LifecyclePolicy, error) {
	switch target {
	case DecisionProposed:
		return LifecyclePolicy{Responsibility: "PROPOSER", Materiality: 2}, nil
	case DecisionInReview, DecisionReturned:
		return LifecyclePolicy{Responsibility: "REVIEWER", Materiality: 3}, nil
	case DecisionChallenged:
		return LifecyclePolicy{Responsibility: "INDEPENDENT_CHALLENGER", Materiality: 3}, nil
	case DecisionApproved, DecisionConditionallyApproved, DecisionRejected, DecisionExpired, DecisionSuperseded:
		return LifecyclePolicy{Responsibility: "AUTHORIZER", Materiality: 4}, nil
	default:
		return LifecyclePolicy{}, fmt.Errorf("%w: unsupported decision lifecycle target %s", ErrInvalidState, target)
	}
}

// ResponseLifecyclePolicy is shared by command authorization and work
// compilation. It validates the transition before returning its responsibility.
func ResponseLifecyclePolicy(from, target ResponseStatus) (LifecyclePolicy, error) {
	if !allowedResponseTransition(from, target) {
		return LifecyclePolicy{}, fmt.Errorf("%w: response cannot move from %s to %s", ErrInvalidState, from, target)
	}
	switch target {
	case ResponseInReview, ResponseRejected:
		return LifecyclePolicy{Responsibility: "REVIEWER", Materiality: 3}, nil
	case ResponseApproved:
		return LifecyclePolicy{Responsibility: "SIGNATORY", Materiality: 4}, nil
	case ResponseTransmitted:
		return LifecyclePolicy{Responsibility: "TRANSMITTER", Materiality: 4}, nil
	case ResponseAcknowledged:
		return LifecyclePolicy{Responsibility: "ACKNOWLEDGEMENT_RECORDER", Materiality: 3}, nil
	case ResponseDraft:
		if from == ResponseRejected {
			return LifecyclePolicy{Responsibility: "PROPOSER", Materiality: 2}, nil
		}
		return LifecyclePolicy{Responsibility: "REVIEWER", Materiality: 3}, nil
	case ResponseWithdrawn:
		if from == ResponseApproved {
			return LifecyclePolicy{Responsibility: "SIGNATORY", Materiality: 4}, nil
		}
		return LifecyclePolicy{Responsibility: "PROPOSER", Materiality: 2}, nil
	default:
		return LifecyclePolicy{}, fmt.Errorf("%w: unsupported response lifecycle target %s", ErrInvalidState, target)
	}
}

// CompileMatterWork derives only executable next steps that are unambiguous in
// current canonical state. Ambiguous lifecycle branches are returned separately
// so a projection can represent them as blocked without choosing an actor.
func CompileMatterWork(aggregate MatterAggregate, now time.Time) ([]WorkRequirement, []WorkAmbiguity) {
	now = now.UTC()
	if aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled {
		return nil, nil
	}

	requirements := make([]WorkRequirement, 0, 4)
	ambiguities := make([]WorkAmbiguity, 0, 2)
	priority := aggregate.Matter.Priority

	for _, response := range currentResponses(aggregate.ResponsePackages) {
		targets := allowedResponseTargets(response.Status)
		if len(targets) == 1 {
			target := targets[0]
			policy, err := ResponseLifecyclePolicy(response.Status, target)
			if err != nil {
				continue
			}
			primary, why := responseWorkCopy(response.Status, target)
			requirements = append(requirements, WorkRequirement{
				Key:               "response:" + response.ID + ":" + string(target),
				CommandName:       "matter.response.transition",
				TargetStatus:      string(target),
				Responsibility:    policy.Responsibility,
				Materiality:       maxInt(policy.Materiality, priority),
				Title:             firstNonBlank(response.Purpose, "External response"),
				PrimaryAction:     primary,
				WhyNow:            why,
				InterventionClass: "EXTERNAL_REPRESENTATION",
				SubresourceType:   "RESPONSE",
				SubresourceID:     response.ID,
				DueAt:             aggregate.Matter.DueAt,
			})
		} else if len(targets) > 1 {
			values := make([]string, len(targets))
			for index := range targets {
				values[index] = string(targets[index])
			}
			ambiguities = append(ambiguities, WorkAmbiguity{
				Key: "response:" + response.ID + ":branch", Title: firstNonBlank(response.Purpose, "External response"),
				Reason: "The response has more than one valid next transition; policy must select the next action before assignment.",
				SubresourceType: "RESPONSE", SubresourceID: response.ID, AllowedTargets: values,
			})
		}
	}

	latestResults := latestVerificationResults(aggregate.VerificationResults)
	for _, contract := range aggregate.VerificationContracts {
		if contract.Status != VerificationActive {
			continue
		}
		if _, alreadyRecorded := latestResults[contract.ID]; alreadyRecorded {
			continue
		}
		readyAt, ok := verificationReadyAt(aggregate, contract)
		if !ok || now.Before(readyAt) {
			continue
		}
		due := readyAt
		requirements = append(requirements, WorkRequirement{
			Key:                 "verification:" + contract.ID + ":record",
			CommandName:         "matter.outcome.record",
			Responsibility:      "REVIEWER",
			Materiality:         maxInt(4, priority),
			Title:               firstNonBlank(contract.ExpectedOutcome, "Record outcome check"),
			PrimaryAction:       "Record outcome check",
			WhyNow:              "The observation period has completed and this outcome check has no recorded result.",
			InterventionClass:   "VERIFICATION",
			SubresourceType:     "VERIFICATION_CONTRACT",
			SubresourceID:       contract.ID,
			DueAt:               &due,
			RequiredPrincipalID: strings.TrimSpace(contract.AuthorityPrincipalID),
			Verification: &WorkVerificationContext{
				ContractID: contract.ID, ExpectedOutcome: contract.ExpectedOutcome,
				EvidenceState: "Outcome check ready", IndependentReview: contract.ActionID != "",
			},
		})
	}

	for _, decision := range currentDecisions(aggregate.Decisions) {
		targets := allowedDecisionTargets(decision.Status)
		if len(targets) <= 1 {
			continue
		}
		values := make([]string, len(targets))
		for index := range targets {
			values[index] = string(targets[index])
		}
		ambiguities = append(ambiguities, WorkAmbiguity{
			Key: "decision:" + decision.ID + ":branch", Title: firstNonBlank(decision.Type, "Decision"),
			Reason: "The decision has more than one valid next transition; state alone does not identify the next authorized action.",
			SubresourceType: "DECISION", SubresourceID: decision.ID, AllowedTargets: values,
		})
	}

	sort.Slice(requirements, func(i, j int) bool { return requirements[i].Key < requirements[j].Key })
	sort.Slice(ambiguities, func(i, j int) bool { return ambiguities[i].Key < ambiguities[j].Key })
	return requirements, ambiguities
}

func allowedResponseTargets(from ResponseStatus) []ResponseStatus {
	ordered := []ResponseStatus{ResponseDraft, ResponseInReview, ResponseApproved, ResponseTransmitted, ResponseAcknowledged, ResponseRejected, ResponseWithdrawn}
	result := make([]ResponseStatus, 0, 4)
	for _, target := range ordered {
		if allowedResponseTransition(from, target) {
			result = append(result, target)
		}
	}
	return result
}

func allowedDecisionTargets(from DecisionStatus) []DecisionStatus {
	ordered := []DecisionStatus{DecisionProposed, DecisionInReview, DecisionChallenged, DecisionApproved, DecisionConditionallyApproved, DecisionRejected, DecisionReturned, DecisionExpired, DecisionSuperseded}
	result := make([]DecisionStatus, 0, 6)
	for _, target := range ordered {
		if ValidateDecisionLifecycle([]Decision{{Type: "WORK", Status: from}}, "WORK", target) == nil {
			result = append(result, target)
		}
	}
	return result
}

func verificationReadyAt(aggregate MatterAggregate, contract VerificationContract) (time.Time, bool) {
	anchor := contract.CreatedAt.UTC()
	if anchor.IsZero() {
		return time.Time{}, false
	}
	if contract.ActionID != "" {
		action, ok := actionByID(aggregate.Actions, contract.ActionID)
		if !ok || action.Status != ActionImplemented || action.ImplementedAt == nil {
			return time.Time{}, false
		}
		if action.ImplementedAt.After(anchor) {
			anchor = action.ImplementedAt.UTC()
		}
	}
	return anchor.Add(time.Duration(contract.ObservationPeriodMinutes) * time.Minute), true
}

func responseWorkCopy(from, target ResponseStatus) (string, string) {
	switch {
	case from == ResponseTransmitted && target == ResponseAcknowledged:
		return "Record acknowledgement", "The response was transmitted and is waiting for acknowledgement to be recorded."
	case from == ResponseRejected && target == ResponseDraft:
		return "Revise response", "The response was rejected and must return to draft before it can be reviewed again."
	default:
		return "Continue response", "This response has one valid next transition."
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
