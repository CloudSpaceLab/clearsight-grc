package continuity

import (
	"errors"
	"strings"
)

// ProgramComplianceCalculation aggregates already evaluated, immutable source
// results. It does not score form answers or determine legal compliance.
var ErrProgramComplianceCalculationInvalid = errors.New("Program compliance calculation is invalid")

const programComplianceBasisPoints int64 = 10000

type ProgramComplianceInputState string

const (
	ProgramComplianceAssessed      ProgramComplianceInputState = "ASSESSED"
	ProgramComplianceUnknown       ProgramComplianceInputState = "UNKNOWN"
	ProgramComplianceNotApplicable ProgramComplianceInputState = "NOT_APPLICABLE"
)

type ProgramComplianceCalculationState string

const (
	ProgramComplianceNotAssessed ProgramComplianceCalculationState = "NOT_ASSESSED"
	ProgramComplianceProvisional ProgramComplianceCalculationState = "PROVISIONAL"
	ProgramComplianceFinal       ProgramComplianceCalculationState = "FINAL"
)

type ProgramComplianceCategoryWeight struct {
	Code      string `json:"code"`
	WeightBps int64  `json:"weight_bps"`
}

// One expected obligation remains in the denominator until an authorized
// applicability decision removes it; request creation is not the population.
type ProgramComplianceExpectedVendorAssessment struct {
	Key                 string `json:"key"`
	RelationshipID      string `json:"relationship_id"`
	FormTemplateID      string `json:"form_template_id"`
	FormTemplateVersion int64  `json:"form_template_version"`
	ReviewKind          string `json:"review_kind"`
	Weight              int64  `json:"weight"`
}

type ProgramComplianceCalculationPolicy struct {
	Revision                  string                                      `json:"revision"`
	MinimumCoverageBps        int64                                       `json:"minimum_coverage_bps"`
	Categories                []ProgramComplianceCategoryWeight           `json:"categories"`
	ExpectedVendorAssessments []ProgramComplianceExpectedVendorAssessment `json:"expected_vendor_assessments,omitempty"`
}

type ProgramComplianceInput struct {
	Key      string                      `json:"key"`
	Category string                      `json:"category"`
	Weight   int64                       `json:"weight"`
	State    ProgramComplianceInputState `json:"state"`
	ScoreBps *int64                      `json:"score_bps,omitempty"`
	Reason   string                      `json:"reason,omitempty"`
}

type ProgramComplianceCategoryResult struct {
	Code          string `json:"code"`
	WeightBps     int64  `json:"weight_bps"`
	EligibleCount int    `json:"eligible_count"`
	AssessedCount int    `json:"assessed_count"`
	UnknownCount  int    `json:"unknown_count"`
	ExcludedCount int    `json:"excluded_count"`
	CoverageBps   int64  `json:"coverage_bps"`
	ScoreBps      *int64 `json:"score_bps,omitempty"`
}

type ProgramComplianceCalculation struct {
	PolicyRevision string                            `json:"policy_revision"`
	State          ProgramComplianceCalculationState `json:"state"`
	ScoreBps       *int64                            `json:"score_bps,omitempty"`
	// AssessedScoreBps is diagnostic only. Never present it as a compliance
	// percentage while coverage is below the approved minimum.
	AssessedScoreBps      *int64                            `json:"assessed_score_bps,omitempty"`
	CoverageBps           int64                             `json:"coverage_bps"`
	EligibleCount         int                               `json:"eligible_count"`
	AssessedCount         int                               `json:"assessed_count"`
	UnknownCount          int                               `json:"unknown_count"`
	ExcludedCount         int                               `json:"excluded_count"`
	UnpopulatedCategories []string                          `json:"unpopulated_categories,omitempty"`
	Categories            []ProgramComplianceCategoryResult `json:"categories"`
}

type programComplianceAccumulator struct {
	totalWeight    int64
	assessedWeight int64
	earnedPoints   int64
}

// ValidateProgramComplianceCalculationPolicy is the approval-time check for
// the same deterministic rules consumed by the calculation. It performs no
// source lookup or material Program state change.
func ValidateProgramComplianceCalculationPolicy(policy ProgramComplianceCalculationPolicy) error {
	_, err := validatedProgramComplianceCategories(policy)
	return err
}

func validatedProgramComplianceCategories(policy ProgramComplianceCalculationPolicy) (map[string]int, error) {
	if strings.TrimSpace(policy.Revision) == "" || len(policy.Revision) > 128 ||
		policy.MinimumCoverageBps < 1 || policy.MinimumCoverageBps > programComplianceBasisPoints ||
		len(policy.Categories) == 0 || len(policy.Categories) > 8 {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	categories := make(map[string]int, len(policy.Categories))
	var total int64
	for index, category := range policy.Categories {
		code := strings.TrimSpace(category.Code)
		if code == "" || len(code) > 64 || code != category.Code ||
			category.WeightBps < 1 || category.WeightBps > programComplianceBasisPoints {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		if _, duplicate := categories[code]; duplicate {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		categories[code] = index
		total += category.WeightBps
	}
	if total != programComplianceBasisPoints {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	if len(policy.ExpectedVendorAssessments) > 10000 {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	if len(policy.ExpectedVendorAssessments) > 0 {
		if _, exists := categories["VENDORS"]; !exists {
			return nil, ErrProgramComplianceCalculationInvalid
		}
	}
	type vendorIdentity struct {
		relationship, form string
		version            int64
		kind               string
	}
	seenKeys := make(map[string]bool, len(policy.ExpectedVendorAssessments))
	seenIdentities := make(map[vendorIdentity]bool, len(policy.ExpectedVendorAssessments))
	for _, expectation := range policy.ExpectedVendorAssessments {
		identity := vendorIdentity{
			relationship: expectation.RelationshipID,
			form:         expectation.FormTemplateID,
			version:      expectation.FormTemplateVersion,
			kind:         expectation.ReviewKind,
		}
		if strings.TrimSpace(expectation.Key) != expectation.Key || expectation.Key == "" ||
			len(expectation.Key) > 192 ||
			strings.TrimSpace(expectation.RelationshipID) != expectation.RelationshipID || expectation.RelationshipID == "" ||
			strings.TrimSpace(expectation.FormTemplateID) != expectation.FormTemplateID || expectation.FormTemplateID == "" ||
			expectation.FormTemplateVersion < 1 ||
			(expectation.ReviewKind != "ONBOARDING" && expectation.ReviewKind != "PERIODIC" && expectation.ReviewKind != "TRIGGERED") ||
			expectation.Weight < 1 || expectation.Weight > programComplianceBasisPoints ||
			seenKeys[expectation.Key] || seenIdentities[identity] {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		seenKeys[expectation.Key], seenIdentities[identity] = true, true
	}
	return categories, nil
}

// CalculateProgramCompliance aggregates fixed-point input scores supplied by
// authoritative Program, evidence and assessment readers. Caller validation
// must establish applicability, due dates, source currency and permissions.
// Bounded input counts keep all basis-point arithmetic in int64 without floats.
func CalculateProgramCompliance(
	policy ProgramComplianceCalculationPolicy,
	inputs []ProgramComplianceInput,
) (ProgramComplianceCalculation, error) {
	if len(inputs) > 10000 {
		return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
	}
	categories, err := validatedProgramComplianceCategories(policy)
	if err != nil {
		return ProgramComplianceCalculation{}, err
	}
	result := ProgramComplianceCalculation{
		PolicyRevision: policy.Revision,
		State:          ProgramComplianceNotAssessed,
		Categories:     make([]ProgramComplianceCategoryResult, len(policy.Categories)),
	}
	accumulators := make([]programComplianceAccumulator, len(policy.Categories))
	for index, category := range policy.Categories {
		result.Categories[index] = ProgramComplianceCategoryResult{Code: category.Code, WeightBps: category.WeightBps}
	}

	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		key := strings.TrimSpace(input.Key)
		index, validCategory := categories[input.Category]
		if key == "" || len(key) > 256 || input.Key != key || !validCategory ||
			input.Weight < 1 || input.Weight > programComplianceBasisPoints {
			return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
		}
		if _, duplicate := seen[key]; duplicate {
			return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
		}
		seen[key] = struct{}{}

		category := &result.Categories[index]
		switch input.State {
		case ProgramComplianceNotApplicable:
			// Only an authoritative approved applicability decision can
			// supply this state. Never infer exclusion from a blank answer.
			if input.ScoreBps != nil || strings.TrimSpace(input.Reason) == "" {
				return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
			}
			category.ExcludedCount++
			result.ExcludedCount++
			continue
		case ProgramComplianceUnknown:
			if input.ScoreBps != nil {
				return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
			}
			category.UnknownCount++
			result.UnknownCount++
		case ProgramComplianceAssessed:
			if input.ScoreBps == nil || *input.ScoreBps < 0 || *input.ScoreBps > programComplianceBasisPoints {
				return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
			}
			accumulators[index].assessedWeight += input.Weight
			accumulators[index].earnedPoints += input.Weight * *input.ScoreBps
			category.AssessedCount++
			result.AssessedCount++
		default:
			return ProgramComplianceCalculation{}, ErrProgramComplianceCalculationInvalid
		}
		accumulators[index].totalWeight += input.Weight
		category.EligibleCount++
		result.EligibleCount++
	}

	// Scaled mass is 10000 (policy weight) × 10000 (score/coverage);
	// per-category normalization preserves the declared weights even when
	// only a subset of the category's expected population is assessed.
	var assessedMass, earnedMass int64
	activePolicyMass := programComplianceBasisPoints
	for index, weight := range policy.Categories {
		category := &result.Categories[index]
		aggregate := accumulators[index]
		if aggregate.totalWeight == 0 {
			if category.ExcludedCount > 0 {
				// The source explicitly declared every item not applicable.
				// Only these categories may leave the policy denominator.
				activePolicyMass -= weight.WeightBps
			} else {
				// No input is not proof that the category does not apply.
				result.UnpopulatedCategories = append(result.UnpopulatedCategories, category.Code)
			}
			continue
		}
		category.CoverageBps = roundComplianceRatio(
			aggregate.assessedWeight*programComplianceBasisPoints, aggregate.totalWeight)
		assessedMass += roundComplianceRatio(
			weight.WeightBps*aggregate.assessedWeight*programComplianceBasisPoints, aggregate.totalWeight)
		earnedMass += roundComplianceRatio(
			weight.WeightBps*aggregate.earnedPoints, aggregate.totalWeight)
		if aggregate.assessedWeight > 0 {
			value := roundComplianceRatio(aggregate.earnedPoints, aggregate.assessedWeight)
			category.ScoreBps = &value
		}
	}
	if activePolicyMass == 0 {
		return result, nil // Every category is explicitly not applicable.
	}
	result.CoverageBps = roundComplianceRatio(assessedMass, activePolicyMass)
	if assessedMass == 0 {
		return result, nil
	}
	score := roundComplianceRatio(earnedMass*programComplianceBasisPoints, assessedMass)
	result.AssessedScoreBps = &score
	if result.CoverageBps < policy.MinimumCoverageBps || result.EligibleCount == 0 || len(result.UnpopulatedCategories) > 0 {
		result.State = ProgramComplianceProvisional
		return result, nil
	}
	result.State = ProgramComplianceFinal
	result.ScoreBps = &score
	return result, nil
}

func roundComplianceRatio(numerator, denominator int64) int64 {
	if denominator == 0 {
		return 0
	}
	return (numerator + denominator/2) / denominator
}
