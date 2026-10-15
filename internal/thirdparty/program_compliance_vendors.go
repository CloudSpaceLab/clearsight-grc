package thirdparty

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// A Program's expected assessment population must be approved independently
// of captured requests. Absence of an assessment remains part of the denominator.
const vendorComplianceBasisPoints int64 = 10000

type ProgramVendorAssessmentExpectation struct {
	Key                 string
	RelationshipID      string
	FormTemplateID      string
	FormTemplateVersion int64
	ReviewKind          AssessmentReviewKind
	Weight              int64
}

type programVendorAssessmentKey struct {
	relationshipID string
	formID         string
	formVersion    int64
	reviewKind     AssessmentReviewKind
}

// ProgramVendorAssessmentInputs is a pure, scoped adapter over existing
// third-party assessments and current vendor form reads. The caller must obtain
// all expected obligations from the approved Program policy, with exact
// tenant/entity/Program access checks performed by the source repositories.
func ProgramVendorAssessmentInputs(
	tenantID, legalEntityID string,
	expected []ProgramVendorAssessmentExpectation,
	assessments []Assessment,
	forms []evidence.VendorFormRow,
	periodStart, periodEnd, asOf time.Time,
	category string,
) ([]continuity.ProgramComplianceInput, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(legalEntityID) == "" ||
		strings.TrimSpace(category) == "" || category != strings.TrimSpace(category) ||
		periodStart.IsZero() || !periodStart.Before(periodEnd) || asOf.IsZero() ||
		len(expected) > 10000 {
		return nil, continuity.ErrProgramComplianceCalculationInvalid
	}
	// Index supplied authoritative, scope-filtered rows once; large Programs
	// must not scan the full assessment or response population per obligation.
	latestByKey := make(map[programVendorAssessmentKey]*Assessment, len(assessments))
	for index := range assessments {
		candidate := &assessments[index]
		if candidate.TenantID != tenantID || candidate.LegalEntityID != legalEntityID ||
			candidate.ReviewDueAt.Before(periodStart) || !candidate.ReviewDueAt.Before(periodEnd) ||
			candidate.StartedAt.After(asOf) || candidate.UpdatedAt.After(asOf) {
			continue
		}
		key := programVendorAssessmentKey{
			relationshipID: candidate.RelationshipID,
			formID:         candidate.FormTemplateID,
			formVersion:    candidate.FormTemplateVersion,
			reviewKind:     candidate.ReviewKind,
		}
		selected := latestByKey[key]
		if selected == nil || candidate.StartedAt.After(selected.StartedAt) ||
			(candidate.StartedAt.Equal(selected.StartedAt) && candidate.ID > selected.ID) {
			latestByKey[key] = candidate
		}
	}
	currentFormsByRequest := make(map[string]*evidence.VendorFormRow, len(forms))
	ambiguousRequests := make(map[string]bool)
	for index := range forms {
		row := &forms[index]
		if row.Current {
			if _, duplicate := currentFormsByRequest[row.RequestID]; duplicate {
				ambiguousRequests[row.RequestID] = true
			}
			currentFormsByRequest[row.RequestID] = row
		}
	}
	seen := make(map[string]bool, len(expected))
	output := make([]continuity.ProgramComplianceInput, 0, len(expected))
	for _, item := range expected {
		if item.Key == "" || len(item.Key) > 192 || seen[item.Key] ||
			item.RelationshipID == "" || item.FormTemplateID == "" || item.FormTemplateVersion <= 0 ||
			item.Weight < 1 || item.Weight > vendorComplianceBasisPoints ||
			(item.ReviewKind != AssessmentReviewOnboarding &&
				item.ReviewKind != AssessmentReviewPeriodic &&
				item.ReviewKind != AssessmentReviewTriggered) {
			return nil, continuity.ErrProgramComplianceCalculationInvalid
		}
		seen[item.Key] = true
		input := continuity.ProgramComplianceInput{
			Key:      "VENDOR_ASSESSMENT:" + item.Key,
			Category: category,
			Weight:   item.Weight,
			State:    continuity.ProgramComplianceUnknown,
			Reason:   "ASSESSMENT_NOT_RECEIVED",
		}
		selected := latestByKey[programVendorAssessmentKey{
			relationshipID: item.RelationshipID,
			formID:         item.FormTemplateID,
			formVersion:    item.FormTemplateVersion,
			reviewKind:     item.ReviewKind,
		}]
		if selected == nil {
			output = append(output, input)
			continue
		}
		input.Reason = "ASSESSMENT_PENDING"
		if selected.Status != AssessmentCompleted ||
			selected.Conclusion == AssessmentInconclusive ||
			selected.CompletedAt == nil || selected.CompletedAt.After(asOf) ||
			selected.CurrentRequestID == "" {
			output = append(output, input)
			continue
		}
		if ambiguousRequests[selected.CurrentRequestID] {
			return nil, continuity.ErrProgramComplianceCalculationInvalid
		}
		matched := currentFormsByRequest[selected.CurrentRequestID]
		if matched != nil &&
			(matched.RelationshipID != item.RelationshipID ||
				matched.FormTemplateID != item.FormTemplateID ||
				matched.FormTemplateVersion != item.FormTemplateVersion) {
			matched = nil
		}
		if matched == nil || matched.ResponseState != "SUBMITTED" ||
			matched.SubmittedAt == nil || matched.SubmittedAt.After(asOf) ||
			matched.ResponseCurrency != "CURRENT" ||
			matched.Outdated == nil || *matched.Outdated {
			input.Reason = "CURRENT_EVIDENCE_UNAVAILABLE"
			output = append(output, input)
			continue
		}
		if selected.Conclusion == AssessmentUnsatisfactory {
			score := int64(0) // An independently confirmed adverse conclusion.
			input.State = continuity.ProgramComplianceAssessed
			input.ScoreBps = &score
			input.Reason = "ASSESSMENT_UNSATISFACTORY"
			output = append(output, input)
			continue
		}
		if selected.Conclusion != AssessmentSatisfactory {
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
			score.CalculatedAt.IsZero() || score.CalculatedAt.After(asOf) ||
			math.IsNaN(*score.RawScore) || math.IsInf(*score.RawScore, 0) ||
			*score.RawScore < 0 || *score.RawScore > 100 {
			input.Reason = "BANK_SCORE_NOT_FINAL"
			output = append(output, input)
			continue
		}
		points := int64(math.Round(*score.RawScore * 100))
		input.State = continuity.ProgramComplianceAssessed
		input.ScoreBps = &points
		input.Reason = "BANK_ASSESSED_RESPONSE"
		output = append(output, input)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Key < output[j].Key })
	return output, nil
}
