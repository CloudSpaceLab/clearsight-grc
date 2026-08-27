package formcontract

import (
	"errors"
	"testing"
)

func TestNormalizeComplianceRequiresExactWeights(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringCompliance,
		Sections:    []Section{{ID: "identity", Title: "Identity", Weight: 100}},
		Fields: []Field{{
			ID: "registered", SectionID: "identity", Label: "Registration verified", Type: TypeYesNo, Required: true,
			Scoring: &Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}},
		}},
	}
	if _, err := Normalize(contract); err != nil {
		t.Fatalf("normalize compliance form: %v", err)
	}
	contract.Fields[0].Scoring.Weight = 90
	if _, err := Normalize(contract); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid weight total, got %v", err)
	}
}

func TestNormalizeKeepsExistingRiskWeightsBackwardCompatible(t *testing.T) {
	contract := Contract{
		Sections: []Section{{ID: "general", Title: "General"}},
		Fields: []Field{{
			ID: "risk", SectionID: "general", Label: "Risk", Type: TypeYesNo,
			Scoring: &Scoring{Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
		}},
	}
	normalized, err := Normalize(contract)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.ScoringMode != ScoringRisk {
		t.Fatalf("risk compatibility mode = %q", normalized.ScoringMode)
	}
}

func TestScoreComplianceUsesApplicableWeightedPopulation(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringCompliance,
		Sections: []Section{
			{ID: "identity", Title: "Identity", Weight: 60},
			{ID: "hosting", Title: "Hosting", Weight: 40, Condition: &VisibilityCondition{FieldID: "handles_data", Operator: ConditionEquals, Values: []string{"Yes"}}},
		},
		Fields: []Field{
			{ID: "handles_data", SectionID: "identity", Label: "Handles customer data", Type: TypeYesNo, Required: true, Scoring: &Scoring{Weight: 50, AnswerScores: map[string]int{"Yes": 100, "No": 100}}},
			{ID: "registered", SectionID: "identity", Label: "Registration verified", Type: TypeYesNo, Required: true, Scoring: &Scoring{Weight: 50, AnswerScores: map[string]int{"Yes": 100, "No": 0}, CriticalAnswers: []string{"No"}}},
			{ID: "encrypted", SectionID: "hosting", Label: "Encryption enabled", Type: TypeYesNo, Required: true, Scoring: &Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}, CriticalAnswers: []string{"No"}}},
		},
	}

	notApplicable, err := ScoreCompliance(contract, map[string]AnswerValue{
		"handles_data": TextAnswer("No"),
		"registered":   TextAnswer("Yes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if notApplicable.Score == nil || *notApplicable.Score != 100 || notApplicable.Coverage != 1 || !notApplicable.Final || notApplicable.Denominator != 2 {
		t.Fatalf("unexpected not-applicable result: %#v", notApplicable)
	}

	incomplete, err := ScoreCompliance(contract, map[string]AnswerValue{
		"handles_data": TextAnswer("Yes"),
		"registered":   TextAnswer("Yes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if incomplete.Score == nil || *incomplete.Score != 60 || incomplete.Coverage != .6 || incomplete.Final || incomplete.Denominator != 3 {
		t.Fatalf("unexpected incomplete result: %#v", incomplete)
	}
}

func TestScoreComplianceRenormalizesVisibleFieldsWithinSection(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringCompliance,
		Sections:    []Section{{ID: "review", Title: "Review", Weight: 100}},
		Fields: []Field{
			{ID: "applies", SectionID: "review", Label: "Does the detail apply", Type: TypeYesNo, Required: true, Scoring: &Scoring{Weight: 20, AnswerScores: map[string]int{"Yes": 100, "No": 100}}},
			{ID: "control", SectionID: "review", Label: "Control operating", Type: TypeYesNo, Required: true, Scoring: &Scoring{Weight: 40, AnswerScores: map[string]int{"Yes": 100, "No": 0}}},
			{ID: "detail", SectionID: "review", Label: "Detail complete", Type: TypeYesNo, Required: true, Condition: &VisibilityCondition{FieldID: "applies", Operator: ConditionEquals, Values: []string{"Yes"}}, Scoring: &Scoring{Weight: 40, AnswerScores: map[string]int{"Yes": 100, "No": 0}}},
		},
	}

	result, err := ScoreCompliance(contract, map[string]AnswerValue{
		"applies": TextAnswer("No"),
		"control": TextAnswer("No"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score == nil || *result.Score != 33.33 || result.Coverage != 1 || result.Denominator != 2 {
		t.Fatalf("unexpected normalized visible-field result: %#v", result)
	}
}
