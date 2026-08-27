package formcontract

import "strings"

// NormalizeDraft applies the complete form contract normalization boundary while
// deferring only COMPLIANCE allocation completeness. Drafts must still contain
// structurally valid sections, fields, conditions, constraints and per-answer
// scoring; exact 100 percent allocation is enforced by Normalize before review.
func NormalizeDraft(input Contract) (Contract, error) {
	if !strings.EqualFold(strings.TrimSpace(string(input.ScoringMode)), string(ScoringCompliance)) {
		return Normalize(input)
	}

	weights := make([]int, len(input.Sections))
	sections := append([]Section(nil), input.Sections...)
	for index := range sections {
		if sections[index].Weight < 0 || sections[index].Weight > 100 {
			return Contract{}, invalid("every section requires a bounded id and title")
		}
		weights[index] = sections[index].Weight
		sections[index].Weight = 0
	}

	validation := input
	validation.ScoringMode = ScoringRisk
	validation.Sections = sections
	normalized, err := Normalize(validation)
	if err != nil {
		return Contract{}, err
	}

	normalized.ScoringMode = ScoringCompliance
	for index := range normalized.Sections {
		if index < len(weights) {
			normalized.Sections[index].Weight = weights[index]
		}
	}
	return normalized, nil
}
