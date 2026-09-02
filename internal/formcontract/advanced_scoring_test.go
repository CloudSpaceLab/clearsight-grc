package formcontract

import (
	"errors"
	"testing"
)

func TestEvaluateScoreProfileUsesDirectionAndCrossFieldRule(t *testing.T) {
	profile := ScoreProfile{
		Version: "score-profile-v2",
		Mode:    ScoringCompliance,
		Bands:   DefaultConcernBands(),
		Contributions: []ScoreContribution{
			{
				ID: "cert", Weight: 70,
				Predicate:   Predicate{FieldID: "certified", Operator: PredicateEquals, Values: []string{"Yes"}},
				MatchPoints: 100, NonMatchPoints: 0, Missing: MissingIndeterminate,
			},
			{
				ID: "expiry", Weight: 30,
				Predicate:   Predicate{FieldID: "expires_on", Operator: PredicateDateOnOrAfter, Values: []string{"2026-09-01"}},
				MatchPoints: 100, NonMatchPoints: 0, Missing: MissingIndeterminate,
			},
		},
		Rules: []ScoreRule{{
			ID: "expired-cert",
			Predicate: Predicate{Operator: PredicateAnd, Children: []Predicate{
				{FieldID: "certified", Operator: PredicateEquals, Values: []string{"Yes"}},
				{FieldID: "expires_on", Operator: PredicateDateBefore, Values: []string{"2026-09-01"}},
			}},
			Effect: RuleEffect{Kind: EffectDisqualify},
		}},
	}
	contract := Contract{
		ScoringMode: ScoringCompliance,
		Sections:    []Section{{ID: "certifications", Title: "Certifications"}},
		Fields: []Field{
			{ID: "certified", SectionID: "certifications", Label: "Certified", Type: TypeYesNo, Required: true},
			{ID: "expires_on", SectionID: "certifications", Label: "Expires on", Type: TypeDate, Required: true},
		},
	}
	result, err := EvaluateScoreProfile(profile, contract, TextAnswers(map[string]string{
		"certified":  "Yes",
		"expires_on": "2026-08-31",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.RawScore == nil || *result.RawScore != 70 {
		t.Fatalf("raw score = %#v, want 70", result.RawScore)
	}
	if result.AdverseScore == nil || *result.AdverseScore != 30 {
		t.Fatalf("adverse score = %#v, want 30", result.AdverseScore)
	}
	if result.Band != ConcernCritical || !result.Disqualified {
		t.Fatalf("result = %#v, want critical disqualification", result)
	}
}

func TestNormalizeScoreProfileDerivesDirectionAndRejectsInvalidBounds(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringRisk,
		Sections:    []Section{{ID: "risk", Title: "Risk"}},
		Fields:      []Field{{ID: "exposure", SectionID: "risk", Label: "Exposure", Type: TypePercentage}},
		ScoreProfile: &ScoreProfile{
			Version: "risk-v2", Mode: ScoringRisk,
			Contributions: []ScoreContribution{{ID: "exposure", Weight: 100, Predicate: Predicate{FieldID: "exposure", Operator: PredicateGreaterEqual, Values: []string{"75"}}, MatchPoints: 100, Missing: MissingIndeterminate}},
			Bands:         DefaultConcernBands(),
		},
	}
	normalized, err := Normalize(contract)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.ScoreProfile.Direction != DirectionHighIsPoor {
		t.Fatalf("direction = %q", normalized.ScoreProfile.Direction)
	}

	invalidFloorCap := contract
	profile := *contract.ScoreProfile
	profile.Rules = []ScoreRule{
		{ID: "floor", Predicate: Predicate{FieldID: "exposure", Operator: PredicateAnswered}, Effect: RuleEffect{Kind: EffectFloor, Value: 80}},
		{ID: "cap", Predicate: Predicate{FieldID: "exposure", Operator: PredicateAnswered}, Effect: RuleEffect{Kind: EffectCap, Value: 70}},
	}
	invalidFloorCap.ScoreProfile = &profile
	if _, err := Normalize(invalidFloorCap); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid floor/cap, got %v", err)
	}

	tooMany := contract
	tooManyProfile := *contract.ScoreProfile
	tooManyProfile.Contributions = make([]ScoreContribution, 101)
	tooMany.ScoreProfile = &tooManyProfile
	if _, err := Normalize(tooMany); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected contribution limit error, got %v", err)
	}
}

func TestEvaluateScoreProfileSupportsNumericAndMultiSelectPredicates(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringRisk,
		Sections:    []Section{{ID: "risk", Title: "Risk"}},
		Fields: []Field{
			{ID: "exposure", SectionID: "risk", Label: "Exposure", Type: TypePercentage},
			{ID: "regions", SectionID: "risk", Label: "Regions", Type: TypeMultiSelect, Options: []string{"Nigeria", "United Kingdom", "United States"}},
		},
	}
	profile := ScoreProfile{Version: "risk-v2", Mode: ScoringRisk, Bands: DefaultConcernBands(), Contributions: []ScoreContribution{
		{ID: "exposure", Weight: 60, Predicate: Predicate{FieldID: "exposure", Operator: PredicateNumberBetween, Values: []string{"70", "90"}}, MatchPoints: 100, Missing: MissingIndeterminate},
		{ID: "regions", Weight: 40, Predicate: Predicate{FieldID: "regions", Operator: PredicateContainsAll, Values: []string{"Nigeria", "United Kingdom"}}, MatchPoints: 50, Missing: MissingIndeterminate},
	}}
	percentage := "80"
	result, err := EvaluateScoreProfile(profile, contract, map[string]AnswerValue{
		"exposure": {Text: &percentage},
		"regions":  {Values: []string{"United Kingdom", "Nigeria"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RawScore == nil || *result.RawScore != 80 || result.Band != ConcernCritical {
		t.Fatalf("result = %#v, want risk score 80 critical", result)
	}
}

func TestEvaluateScoreProfileExcludesHiddenFieldsAndAppliesFloor(t *testing.T) {
	contract := Contract{
		ScoringMode: ScoringRisk,
		Sections:    []Section{{ID: "risk", Title: "Risk"}},
		Fields: []Field{
			{ID: "uses_cert", SectionID: "risk", Label: "Uses certification", Type: TypeYesNo},
			{ID: "cert_scope", SectionID: "risk", Label: "Certification scope", Type: TypeShortText, Condition: &VisibilityCondition{FieldID: "uses_cert", Operator: ConditionEquals, Values: []string{"Yes"}}},
		},
	}
	profile := ScoreProfile{Version: "risk-v2", Mode: ScoringRisk, Bands: DefaultConcernBands(), Contributions: []ScoreContribution{
		{ID: "uses-cert", Weight: 50, Predicate: Predicate{FieldID: "uses_cert", Operator: PredicateEquals, Values: []string{"No"}}, MatchPoints: 20, Missing: MissingIndeterminate},
		{ID: "scope", Weight: 50, Predicate: Predicate{FieldID: "cert_scope", Operator: PredicateAnswered}, MatchPoints: 0, NonMatchPoints: 100, Missing: MissingIndeterminate},
	}, Rules: []ScoreRule{{ID: "minimum", Predicate: Predicate{FieldID: "uses_cert", Operator: PredicateEquals, Values: []string{"No"}}, Effect: RuleEffect{Kind: EffectFloor, Value: 55}}}}
	result, err := EvaluateScoreProfile(profile, contract, TextAnswers(map[string]string{"uses_cert": "No"}))
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage != 1 || result.RawScore == nil || *result.RawScore != 20 || result.AdverseScore == nil || *result.AdverseScore != 55 || result.Band != ConcernHigh {
		t.Fatalf("result = %#v", result)
	}
}
