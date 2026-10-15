package continuity

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

// A Program's expected assessment population must be approved independently
// of captured requests. Absence of an assessment remains part of the denominator.
type ProgramVendorAssessmentExpectation struct {
	Key            string
	RelationshipID string
	FormTemplateID string
	ReviewKind     thirdparty.AssessmentReviewKind
	Weight         int64
}

// ProgramVendorAssessmentInputs is a pure, scoped adapter over existing
// third-party assessments and current vendor form reads. The caller must obtain
// all expected obligations from the approved Program policy, with exact
// tenant/entity/Program access checks performed by the source repositories.
func ProgramVendorAssessmentInputs(
	expected []ProgramVendorAssessmentExpectation,
	assessments []thirdparty.Assessment,
	forms []evidence.VendorFormRow,
	periodStart, periodEnd, asOf time.Time,
	category string,
) ([]ProgramComplianceInput, error) {
	if strings.TrimSpace(category) == "" || category != strings.TrimSpace(category) ||
		periodStart.IsZero() || !periodStart.Before(periodEnd) || asOf.IsZero() ||
		len(expected) > 10000 {
		return nil, ErrProgramComplianceCalculationInvalid
	}
	seen := make(map[string]bool, len(expected))
	output := make([]ProgramComplianceInput, 0, len(expected))
	for _, item := range expected {
		if item.Key == "" || len(item.Key) > 192 || seen[item.Key] ||
			item.RelationshipID == "" || item.FormTemplateID == "" ||
			item.Weight < 1 || item.Weight > programComplianceBasisPoints ||
			(item.ReviewKind != thirdparty.AssessmentReviewOnboarding &&
				item.ReviewKind != thirdparty.AssessmentReviewPeriodic &&
				item.ReviewKind != thirdparty.AssessmentReviewTriggered) {
			return nil, ErrProgramComplianceCalculationInvalid
		}
		seen[item.Key] = true
		input := ProgramComplianceInput{
			Key:      "VENDOR_ASSESSMENT:" + item.Key,
			Category: category,
			Weight:   item.Weight,
			State:    ProgramComplianceUnknown,
			Reason:   "ASSESSMENT_NOT_RECEIVED",
		}
		var selected *thirdparty.Assessment
		for index := range assessments {
			candidate := &assessments[index]
			if candidate.RelationshipID != item.RelationshipID ||
				candidate.FormTemplateID != item.FormTemplateID ||
				candidate.ReviewKind != item.ReviewKind ||
				candidate.ReviewDueAt.Before(periodStart) || !candidate.ReviewDueAt.Before(periodEnd) ||
				candidate.StartedAt.After(asOf) || candidate.UpdatedAt.After(asOf) {
				continue
			}
			if selected == nil || candidate.StartedAt.After(selected.StartedAt) ||
				(candidate.StartedAt.Equal(selected.StartedAt) && candidate.ID > selected.ID) {
				selected = candidate
			}
		}
		if selected == nil {
			output = append(output, input)
			continue
		}
		input.Reason = "ASSESSMENT_PENDING"
		if selected.Status != thirdparty.AssessmentCompleted ||
			selected.Conclusion == thirdparty.AssessmentInconclusive ||
			selected.CompletedAt == nil || selected.CompletedAt.After(asOf) ||
			selected.CurrentRequestID == "" {
			output = append(output, input)
			continue
		}
		var matched *evidence.VendorFormRow
		for index := range forms {
			row := &forms[index]
			if row.RelationshipID != item.RelationshipID ||
				row.FormTemplateID != item.FormTemplateID ||
				row.RequestID != selected.CurrentRequestID || !row.Current {
				continue
			}
			if matched != nil {
				return nil, ErrProgramComplianceCalculationInvalid // Ambiguous authoritative current result.
			}
			matched = row
		}
		if matched == nil || matched.ResponseState != "SUBMITTED" ||
			matched.ResponseCurrency != "CURRENT" ||
			matched.Outdated == nil || *matched.Outdated {
			input.Reason = "CURRENT_EVIDENCE_UNAVAILABLE"
			output = append(output, input)
			continue
		}
		if selected.Conclusion == thirdparty.AssessmentUnsatisfactory {
			score := int64(0) // An independently confirmed adverse conclusion.
			input.State = ProgramComplianceAssessed
			input.ScoreBps = &score
			input.Reason = "ASSESSMENT_UNSATISFACTORY"
			output = append(output, input)
			continue
		}
		if selected.Conclusion != thirdparty.AssessmentSatisfactory {
			input.Reason = "CONDITIONAL_ASSESSMENT_REQUIRES_POLICY_MAPPING"
			output = append(output, input)
			continue
		}
		// Only an independently reviewed, final compliance-mode score can be
		// used directly. Risk scores require a separate approved conversion.
		score := matched.AssessedScore
		if matched.AssessmentState != "ASSESSED" || score == nil ||
			score.State != evidence.ResponseScoreFinal || !score.Final ||
			score.Mode != formcontract.ScoringCompliance || score.RawScore == nil ||
			score.Coverage < 1 || score.Coverage > 1 ||
			math.IsNaN(*score.RawScore) || math.IsInf(*score.RawScore, 0) ||
			*score.RawScore < 0 || *score.RawScore > 100 {
			input.Reason = "BANK_SCORE_NOT_FINAL"
			output = append(output, input)
			continue
		}
		points := int64(math.Round(*score.RawScore * 100))
		input.State = ProgramComplianceAssessed
		input.ScoreBps = &points
		input.Reason = "BANK_ASSESSED_RESPONSE"
		output = append(output, input)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Key < output[j].Key })
	return output, nil
}
