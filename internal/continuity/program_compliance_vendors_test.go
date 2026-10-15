package continuity

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestProgramVendorAssessmentInputsKeepUnrequestedVendorInDenominator(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now.Add(24 * time.Hour)
	completed := now.Add(-time.Hour)
	fresh := false
	raw := 84.0
	expected := []ProgramVendorAssessmentExpectation{
		{Key: "vendor-a:security:oct", RelationshipID: "vendor-a", FormTemplateID: "security", FormTemplateVersion: 2, ReviewKind: thirdparty.AssessmentReviewPeriodic, Weight: 1},
		{Key: "vendor-b:security:oct", RelationshipID: "vendor-b", FormTemplateID: "security", FormTemplateVersion: 2, ReviewKind: thirdparty.AssessmentReviewPeriodic, Weight: 1},
	}
	assessments := []thirdparty.Assessment{{
		ID: "assessment-a", TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor-a", FormTemplateID: "security",
		FormTemplateVersion: 2, ReviewKind: thirdparty.AssessmentReviewPeriodic,
		Status: thirdparty.AssessmentCompleted, Conclusion: thirdparty.AssessmentSatisfactory,
		ReviewDueAt: now, StartedAt: now.Add(-2 * time.Hour), UpdatedAt: completed, CompletedAt: &completed,
		CurrentRequestID: "request-a",
	}}
	forms := []evidence.VendorFormRow{{
		RequestID: "request-a", RelationshipID: "vendor-a", FormTemplateID: "security", FormTemplateVersion: 2,
		Current: true, ResponseState: "SUBMITTED", ResponseCurrency: "CURRENT", Outdated: &fresh,
		AssessmentState: "ASSESSED", AssessedScore: &evidence.ResponseScoreResult{
			Mode: formcontract.ScoringCompliance, RawScore: &raw, Coverage: 1,
			Final: true, State: evidence.ResponseScoreFinal,
		},
	}}
	inputs, err := ProgramVendorAssessmentInputs("bank", "entity", expected, assessments, forms, periodStart, periodEnd, now, "VENDORS")
	if err != nil || len(inputs) != 2 {
		t.Fatalf("expected both vendor obligations: %v, %+v", err, inputs)
	}
	if inputs[0].State != ProgramComplianceAssessed || inputs[0].ScoreBps == nil ||
		*inputs[0].ScoreBps != 8400 || inputs[1].State != ProgramComplianceUnknown {
		t.Fatalf("missing vendor or reviewed score incorrect: %+v", inputs)
	}
	policy := ProgramComplianceCalculationPolicy{
		Revision: "approved", MinimumCoverageBps: 10000,
		Categories: []ProgramComplianceCategoryWeight{{Code: "VENDORS", WeightBps: 10000}},
	}
	result, err := CalculateProgramCompliance(policy, inputs)
	if err != nil || result.State != ProgramComplianceProvisional ||
		result.ScoreBps != nil || result.CoverageBps != 5000 {
		t.Fatalf("unrequested vendor inflated compliance: %+v, %v", result, err)
	}

	// A response for the wrong form revision must not satisfy the expected obligation.
	forms[0].FormTemplateVersion = 1
	inputs, err = ProgramVendorAssessmentInputs("bank", "entity", expected, assessments, forms, periodStart, periodEnd, now, "VENDORS")
	if err != nil || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("old form revision reused: %+v, %v", inputs, err)
	}
}

func TestProgramVendorAssessmentInputsFailClosedOnPendingAmbiguousOrUnreviewed(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	fresh := false
	score := 95.0
	expected := []ProgramVendorAssessmentExpectation{{
		Key: "vendor:oct", RelationshipID: "vendor", FormTemplateID: "form", FormTemplateVersion: 1,
		ReviewKind: thirdparty.AssessmentReviewPeriodic, Weight: 1,
	}}
	assessment := thirdparty.Assessment{
		ID: "one", TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor", FormTemplateID: "form",
		FormTemplateVersion: 1, ReviewKind: thirdparty.AssessmentReviewPeriodic,
		Status: thirdparty.AssessmentCompleted, Conclusion: thirdparty.AssessmentSatisfactory,
		ReviewDueAt: now, StartedAt: past, UpdatedAt: past, CompletedAt: &past,
		CurrentRequestID: "request",
	}
	row := evidence.VendorFormRow{
		RequestID: "request", RelationshipID: "vendor", FormTemplateID: "form", FormTemplateVersion: 1,
		Current: true, ResponseState: "SUBMITTED", ResponseCurrency: "CURRENT", Outdated: &fresh,
		AssessmentState: "AWAITING_REVIEW", Score: &evidence.ResponseScoreResult{
			Mode: formcontract.ScoringCompliance, RawScore: &score, Coverage: 1,
			Final: true, State: evidence.ResponseScoreFinal,
		},
	}
	calculate := func(a thirdparty.Assessment, rows []evidence.VendorFormRow) ([]ProgramComplianceInput, error) {
		return ProgramVendorAssessmentInputs("bank", "entity", expected, []thirdparty.Assessment{a}, rows,
			now.Add(-24*time.Hour), now.Add(24*time.Hour), now, "VENDORS")
	}
	inputs, err := calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || len(inputs) != 1 || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("unreviewed automatic score promoted: %+v, %v", inputs, err)
	}
	row.AssessmentState = "ASSESSED"
	row.AssessedScore = &evidence.ResponseScoreResult{
		Mode: formcontract.ScoringRisk, RawScore: &score, Coverage: 1,
		Final: true, State: evidence.ResponseScoreFinal,
	}
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("risk score silently became compliance: %+v, %v", inputs, err)
	}
	row.AssessedScore.Mode = formcontract.ScoringCompliance
	row.Outdated = nil
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("unknown evidence freshness promoted: %+v, %v", inputs, err)
	}
	row.Outdated = &fresh
	assessment.Conclusion = thirdparty.AssessmentUnsatisfactory
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != ProgramComplianceAssessed ||
		inputs[0].ScoreBps == nil || *inputs[0].ScoreBps != 0 {
		t.Fatalf("independent adverse conclusion failed to count: %+v, %v", inputs, err)
	}
	assessment.Conclusion = thirdparty.AssessmentSatisfactoryWithConditions
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("conditional conclusion given invented score: %+v, %v", inputs, err)
	}
	assessment.Conclusion = thirdparty.AssessmentSatisfactory
	assessment.LegalEntityID = "sibling-entity"
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != ProgramComplianceUnknown {
		t.Fatalf("cross-entity assessment supplied compliance: %+v, %v", inputs, err)
	}
	assessment.LegalEntityID = "entity"
	_, err = calculate(assessment, []evidence.VendorFormRow{row, row})
	if err == nil {
		t.Fatal("ambiguous current form result silently deduplicated")
	}
}
