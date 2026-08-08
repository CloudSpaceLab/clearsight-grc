package continuity

import (
	"sort"
	"strings"
)

type WorkSequenceChoice struct {
	AmbiguityKey string
	Responsibility string
	RuleID string
	PolicyVersion string
}

// ApplyWorkSequenceChoices converts only policy-resolved ambiguities into
// actor work. The policy selects a responsibility/gate; the packet preserves
// every currently legal lifecycle action for that responsibility so no
// substantive outcome is pre-decided.
func ApplyWorkSequenceChoices(aggregate MatterAggregate, requirements []WorkRequirement, ambiguities []WorkAmbiguity, choices []WorkSequenceChoice) ([]WorkRequirement, []WorkAmbiguity) {
	selected := make(map[string]WorkSequenceChoice, len(choices))
	for _, choice := range choices {
		key := strings.TrimSpace(choice.AmbiguityKey)
		if key != "" && strings.TrimSpace(choice.Responsibility) != "" {
			selected[key] = choice
		}
	}
	unresolved := make([]WorkAmbiguity, 0, len(ambiguities))
	for _, ambiguity := range ambiguities {
		choice, ok := selected[ambiguity.Key]
		if !ok {
			unresolved = append(unresolved, ambiguity)
			continue
		}
		packet, ok := workPacketForResponsibility(aggregate, ambiguity, choice)
		if !ok {
			unresolved = append(unresolved, ambiguity)
			continue
		}
		requirements = append(requirements, packet)
	}
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].Key < requirements[j].Key })
	sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].Key < unresolved[j].Key })
	return requirements, unresolved
}

func workPacketForResponsibility(aggregate MatterAggregate, ambiguity WorkAmbiguity, choice WorkSequenceChoice) (WorkRequirement, bool) {
	responsibility := strings.ToUpper(strings.TrimSpace(choice.Responsibility))
	allowed := make([]string, 0, len(ambiguity.AllowedTargets))
	materiality := aggregate.Matter.Priority
	for _, target := range ambiguity.AllowedTargets {
		policy, err := ambiguityTargetPolicy(ambiguity, target)
		if err != nil || policy.Responsibility != responsibility {
			continue
		}
		allowed = append(allowed, target)
		materiality = maxInt(materiality, policy.Materiality)
	}
	if len(allowed) == 0 {
		return WorkRequirement{}, false
	}
	sort.Strings(allowed)
	target := ""
	if len(allowed) == 1 {
		target = allowed[0]
	}
	primary, intervention := sequenceWorkCopy(ambiguity.SubresourceType, responsibility)
	return WorkRequirement{
		Key:                   sequenceRequirementKey(ambiguity, responsibility),
		CommandName:           ambiguity.CommandName,
		TargetStatus:          target,
		AllowedTargets:        allowed,
		Responsibility:        responsibility,
		Materiality:           materiality,
		Title:                 ambiguity.Title,
		PrimaryAction:         primary,
		WhyNow:                "Current routing policy selects " + humanResponsibility(responsibility) + " as the next governed responsibility; the responsible actor must still choose the permitted outcome.",
		InterventionClass:     intervention,
		SubresourceType:       ambiguity.SubresourceType,
		SubresourceID:         ambiguity.SubresourceID,
		DueAt:                 aggregate.Matter.DueAt,
		SequenceRuleID:        choice.RuleID,
		SequencePolicyVersion: choice.PolicyVersion,
	}, true
}

func ambiguityTargetPolicy(ambiguity WorkAmbiguity, target string) (LifecyclePolicy, error) {
	switch ambiguity.SubresourceType {
	case "DECISION":
		return DecisionLifecyclePolicy(DecisionStatus(target))
	case "RESPONSE":
		return ResponseLifecyclePolicy(ResponseStatus(ambiguity.LifecycleState), ResponseStatus(target))
	default:
		return LifecyclePolicy{}, ErrInvalidState
	}
}

func sequenceRequirementKey(ambiguity WorkAmbiguity, responsibility string) string {
	prefix := strings.TrimSuffix(ambiguity.Key, ":branch")
	return prefix + ":gate:" + strings.ToLower(responsibility)
}

func sequenceWorkCopy(subresourceType, responsibility string) (string, string) {
	switch responsibility {
	case "PROPOSER":
		if subresourceType == "RESPONSE" { return "Prepare response", "EXTERNAL_REPRESENTATION" }
		return "Prepare decision", "DECISION"
	case "REVIEWER":
		if subresourceType == "RESPONSE" { return "Review response", "REVIEW" }
		return "Review decision", "REVIEW"
	case "INDEPENDENT_CHALLENGER":
		return "Challenge decision", "REVIEW"
	case "AUTHORIZER":
		return "Decide", "AUTHORIZATION"
	case "SIGNATORY":
		return "Review and sign response", "EXTERNAL_REPRESENTATION"
	case "TRANSMITTER":
		return "Transmit response", "EXTERNAL_REPRESENTATION"
	case "ACKNOWLEDGEMENT_RECORDER":
		return "Record acknowledgement", "EXTERNAL_REPRESENTATION"
	default:
		return "Review next action", "REVIEW"
	}
}

func humanResponsibility(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "_", " "))
}
