package thirdparty

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestProgramVendorAssessmentInputsKeepUnrequestedVendorInDenominator(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now.Add(24 * time.Hour)
	completed := now.Add(-time.Hour)
	fresh := false
	raw := 84.0
	expected := []ProgramVendorAssessmentExpectation{
		{Key: "vendor-a:security:oct", RelationshipID: "vendor-a", FormTemplateID: "security", FormTemplateVersion: 2, ReviewKind: AssessmentReviewPeriodic, Weight: 1},
		{Key: "vendor-b:security:oct", RelationshipID: "vendor-b", FormTemplateID: "security", FormTemplateVersion: 2, ReviewKind: AssessmentReviewPeriodic, Weight: 1},
	}
	assessments := []Assessment{{
		ID: "assessment-a", TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor-a", FormTemplateID: "security",
		FormTemplateVersion: 2, ReviewKind: AssessmentReviewPeriodic,
		Status: AssessmentCompleted, Conclusion: AssessmentSatisfactory,
		ReviewDueAt: now, StartedAt: now.Add(-2 * time.Hour), UpdatedAt: completed, CompletedAt: &completed,
		CurrentRequestID: "request-a",
	}}
	forms := []evidence.VendorFormRow{{
		RequestID: "request-a", RelationshipID: "vendor-a", FormTemplateID: "security", FormTemplateVersion: 2,
		Current: true, ResponseState: "SUBMITTED", ResponseCurrency: "CURRENT", Outdated: &fresh, SubmittedAt: &completed,
		AssessmentState: "ASSESSED", AssessedScore: &evidence.ResponseScoreResult{
			Mode: formcontract.ScoringCompliance, RawScore: &raw, Coverage: 1,
			Final: true, State: evidence.ResponseScoreFinal, CalculatedAt: completed,
		},
	}}
	inputs, err := ProgramVendorAssessmentInputs("bank", "entity", expected, assessments, forms, periodStart, periodEnd, now, "VENDORS")
	if err != nil || len(inputs) != 2 {
		t.Fatalf("expected both vendor obligations: %v, %+v", err, inputs)
	}
	if inputs[0].State != continuity.ProgramComplianceAssessed || inputs[0].ScoreBps == nil ||
		*inputs[0].ScoreBps != 8400 || inputs[1].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("missing vendor or reviewed score incorrect: %+v", inputs)
	}
	policy := continuity.ProgramComplianceCalculationPolicy{
		Revision: "approved", MinimumCoverageBps: 10000,
		Categories: []continuity.ProgramComplianceCategoryWeight{{Code: "VENDORS", WeightBps: 10000}},
	}
	result, err := continuity.CalculateProgramCompliance(policy, inputs)
	if err != nil || result.State != continuity.ProgramComplianceProvisional ||
		result.ScoreBps != nil || result.CoverageBps != 5000 {
		t.Fatalf("unrequested vendor inflated compliance: %+v, %v", result, err)
	}

	// A response for the wrong form revision must not satisfy the expected obligation.
	forms[0].FormTemplateVersion = 1
	inputs, err = ProgramVendorAssessmentInputs("bank", "entity", expected, assessments, forms, periodStart, periodEnd, now, "VENDORS")
	if err != nil || inputs[0].State != continuity.ProgramComplianceUnknown {
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
		ReviewKind: AssessmentReviewPeriodic, Weight: 1,
	}}
	assessment := Assessment{
		ID: "one", TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor", FormTemplateID: "form",
		FormTemplateVersion: 1, ReviewKind: AssessmentReviewPeriodic,
		Status: AssessmentCompleted, Conclusion: AssessmentSatisfactory,
		ReviewDueAt: now, StartedAt: past, UpdatedAt: past, CompletedAt: &past,
		CurrentRequestID: "request",
	}
	row := evidence.VendorFormRow{
		RequestID: "request", RelationshipID: "vendor", FormTemplateID: "form", FormTemplateVersion: 1,
		Current: true, ResponseState: "SUBMITTED", ResponseCurrency: "CURRENT", Outdated: &fresh, SubmittedAt: &past,
		AssessmentState: "AWAITING_REVIEW", Score: &evidence.ResponseScoreResult{
			Mode: formcontract.ScoringCompliance, RawScore: &score, Coverage: 1,
			Final: true, State: evidence.ResponseScoreFinal,
		},
	}
	calculate := func(a Assessment, rows []evidence.VendorFormRow) ([]continuity.ProgramComplianceInput, error) {
		return ProgramVendorAssessmentInputs("bank", "entity", expected, []Assessment{a}, rows,
			now.Add(-24*time.Hour), now.Add(24*time.Hour), now, "VENDORS")
	}
	inputs, err := calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || len(inputs) != 1 || inputs[0].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("unreviewed automatic score promoted: %+v, %v", inputs, err)
	}
	row.AssessmentState = "ASSESSED"
	row.AssessedScore = &evidence.ResponseScoreResult{
		Mode: formcontract.ScoringRisk, RawScore: &score, Coverage: 1,
		Final: true, State: evidence.ResponseScoreFinal,
	}
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("risk score silently became compliance: %+v, %v", inputs, err)
	}
	row.AssessedScore.Mode = formcontract.ScoringCompliance
	row.Outdated = nil
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("unknown evidence freshness promoted: %+v, %v", inputs, err)
	}
	row.Outdated = &fresh
	assessment.Conclusion = AssessmentUnsatisfactory
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != continuity.ProgramComplianceAssessed ||
		inputs[0].ScoreBps == nil || *inputs[0].ScoreBps != 0 {
		t.Fatalf("independent adverse conclusion failed to count: %+v, %v", inputs, err)
	}
	assessment.Conclusion = AssessmentSatisfactoryWithConditions
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("conditional conclusion given invented score: %+v, %v", inputs, err)
	}
	assessment.Conclusion = AssessmentSatisfactory
	assessment.LegalEntityID = "sibling-entity"
	inputs, err = calculate(assessment, []evidence.VendorFormRow{row})
	if err != nil || inputs[0].State != continuity.ProgramComplianceUnknown {
		t.Fatalf("cross-entity assessment supplied compliance: %+v, %v", inputs, err)
	}
	assessment.LegalEntityID = "entity"
	duplicate := append([]ProgramVendorAssessmentExpectation(nil), expected...)
	another := expected[0]
	another.Key = "vendor:oct:duplicate"
	duplicate = append(duplicate, another)
	if _, err := ProgramVendorAssessmentInputs("bank", "entity", duplicate, []Assessment{assessment},
		[]evidence.VendorFormRow{row}, now.Add(-24*time.Hour), now.Add(24*time.Hour), now, "VENDORS"); err == nil {
		t.Fatal("one expected assessment could be counted twice under different keys")
	}
	_, err = calculate(assessment, []evidence.VendorFormRow{row, row})
	if err == nil {
		t.Fatal("ambiguous current form result silently deduplicated")
	}
}

func TestProgramVendorPolicyInputsIncludeUnrequestedExpectations(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	policy := continuity.ProgramComplianceCalculationPolicy{
		Revision: "program-v1", MinimumCoverageBps: 10000,
		Categories: []continuity.ProgramComplianceCategoryWeight{{Code: "VENDORS", WeightBps: 10000}},
		ExpectedVendorAssessments: []continuity.ProgramComplianceExpectedVendorAssessment{
			{Key: "missing:security", RelationshipID: "vendor-1",
				FormTemplateID: "security", FormTemplateVersion: 2, ReviewKind: "PERIODIC", Weight: 1},
		},
	}
	links := []RelationshipLink{{
		TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor-1",
		TargetType: LinkTargetProgram, TargetID: "program", State: RelationshipLinkActive,
		CreatedAt: now.Add(-time.Hour),
	}}
	inputs, err := ProgramVendorAssessmentInputsForPolicy(
		"bank", "entity", "program", policy, links, true, nil, nil,
		now.Add(-24*time.Hour), now.Add(24*time.Hour), now,
	)
	if err != nil || len(inputs) != 1 || inputs[0].State != continuity.ProgramComplianceUnknown ||
		inputs[0].Key != "VENDOR_ASSESSMENT:missing:security" {
		t.Fatalf("unsent required assessment disappeared: %+v / %v", inputs, err)
	}
	if _, err := ProgramVendorAssessmentInputsForPolicy(
		"bank", "entity", "program", policy, links, false, nil, nil,
		now.Add(-24*time.Hour), now.Add(24*time.Hour), now,
	); err == nil {
		t.Fatal("incomplete Program relationship population accepted")
	}
	links[0].TargetID = "another-program"
	if _, err := ProgramVendorAssessmentInputsForPolicy(
		"bank", "entity", "program", policy, links, true, nil, nil,
		now.Add(-24*time.Hour), now.Add(24*time.Hour), now,
	); err == nil {
		t.Fatal("another Program's vendor link authorized the assessment")
	}
	links[0].TargetID = "program"
	result, err := continuity.CalculateProgramCompliance(policy, inputs)
	if err != nil || result.CoverageBps != 0 || result.ScoreBps != nil || result.UnknownCount != 1 {
		t.Fatalf("unrequested obligation inflated compliance: %+v / %v", result, err)
	}
}
