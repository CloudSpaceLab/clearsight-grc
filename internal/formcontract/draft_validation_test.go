package formcontract

import (
	"errors"
	"testing"
)

func incompleteComplianceContract() Contract {
	return Contract{
		Presentation: Presentation{DefaultMode: PresentationClassic},
		ScoringMode:  ScoringCompliance,
		Sections: []Section{
			{ID: "identity", Title: "Vendor identity", Weight: 60},
			{ID: "security", Title: "Security", Weight: 20},
		},
		Fields: []Field{
			{ID: "registered", SectionID: "identity", Label: "Registration verified", Type: TypeYesNo, Required: true, Options: []string{"Yes", "No"}, Scoring: &Scoring{Weight: 80, AnswerScores: map[string]int{"Yes": 100, "No": 0}}},
			{ID: "mfa", SectionID: "security", Label: "MFA required", Type: TypeYesNo, Required: true, Options: []string{"Yes", "No"}, Scoring: &Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}}},
		},
	}
}

func TestNormalizeDraftDefersOnlyComplianceAllocationTotals(t *testing.T) {
	input := incompleteComplianceContract()
	if _, err := Normalize(input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("strict normalization error = %v, want ErrInvalid", err)
	}

	normalized, err := NormalizeDraft(input)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.ScoringMode != ScoringCompliance || normalized.Sections[0].Weight != 60 || normalized.Sections[1].Weight != 20 {
		t.Fatalf("draft allocation was not preserved: %#v", normalized)
	}
	if normalized.Fields[0].Scoring == nil || normalized.Fields[0].Scoring.ID != "registered" || !normalized.Fields[0].Scoring.Required {
		t.Fatalf("field scoring was not canonically normalized: %#v", normalized.Fields[0].Scoring)
	}
}

func TestNormalizeDraftStillRejectsBrokenVisibility(t *testing.T) {
	input := incompleteComplianceContract()
	input.Fields[0].Condition = &VisibilityCondition{FieldID: "mfa", Operator: ConditionEquals, Values: []string{"Yes"}}
	if _, err := NormalizeDraft(input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forward visibility reference error = %v, want ErrInvalid", err)
	}
}
