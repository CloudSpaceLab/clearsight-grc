package formcontract

import (
	"encoding/json"
	"testing"
)

func TestAssessmentContractRoundTripAndValidation(t *testing.T) {
	var c Contract
	if err := json.Unmarshal([]byte(`{"scoring_mode":"RISK","fields":[{"id":"doc","label":"Test report","type":"file","assessment":{"mode":"MANUAL","required":true,"weight":50,"reviewer_role":"RISK_REVIEWER","rubric":[{"id":"poor","label":"Evidence incomplete","points":80}]}}]}`), &c); err != nil {
		t.Fatal(err)
	}
	n, err := Normalize(c)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(n)
	var raw map[string]any
	json.Unmarshal(encoded, &raw)
	if raw["fields"].([]any)[0].(map[string]any)["assessment"] == nil {
		t.Fatal("assessment configuration lost")
	}
}
func TestAssessmentInvalidModeRejected(t *testing.T) {
	var c Contract
	json.Unmarshal([]byte(`{"fields":[{"id":"doc","label":"Report","type":"file","assessment":{"mode":"SCRIPT","weight":1}}]}`), &c)
	if _, err := Normalize(c); err == nil {
		t.Fatal("invalid assessment mode accepted")
	}
}

func TestAssessmentRequiredPendingAndHidden(t *testing.T) {
	c := Contract{ScoringMode: ScoringRisk, Fields: []Field{{ID: "doc", Label: "Report", Type: TypeFile, Assessment: &FieldAssessment{Mode: AssessmentManual, Required: true, Weight: 50, ReviewerRole: "RISK", Rubric: []AssessmentOutcome{{ID: "poor", Label: "Incomplete", Points: 80}}}}}}
	pending, err := EvaluateAssessment(c, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Final || pending.RawScore != nil {
		t.Fatalf("pending review appears final: %+v", pending)
	}
	final, err := EvaluateAssessment(c, nil, map[string]string{"doc": "poor"})
	if err != nil {
		t.Fatal(err)
	}
	if !final.Final || final.RawScore == nil || *final.RawScore != 80 {
		t.Fatalf("manual contribution missing: %+v", final)
	}
	c.Fields = append([]Field{{ID: "applies", Label: "Applies", Type: TypeYesNo}}, c.Fields...)
	c.Fields[1].Condition = &VisibilityCondition{FieldID: "applies", Operator: ConditionEquals, Values: []string{"Yes"}}
	hidden, err := EvaluateAssessment(c, TextAnswers(map[string]string{"applies": "No"}), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hidden.Final {
		t.Fatal("hidden field counted as pending")
	}
}

func TestAssessmentComplianceManualWeightAndCriticalFloor(t *testing.T) {
	c := Contract{ScoringMode: ScoringCompliance, Sections: []Section{{ID: "s", Title: "Review", Weight: 100}}, Fields: []Field{{ID: "doc", SectionID: "s", Label: "Report", Type: TypeFile, Assessment: &FieldAssessment{Mode: AssessmentManual, Required: true, Weight: 100, ReviewerRole: "RISK", Rubric: []AssessmentOutcome{{ID: "good", Label: "Sufficient", Points: 100}}}}}}
	r, err := EvaluateAssessment(c, nil, map[string]string{"doc": "good"})
	if err != nil {
		t.Fatal(err)
	}
	if r.AdverseScore == nil || *r.AdverseScore != 0 {
		t.Fatalf("compliance direction: %+v", r)
	}
	c.ScoringMode = ScoringRisk
	c.Sections[0].Weight = 0
	c.Fields = append(c.Fields, Field{ID: "control", SectionID: "s", Label: "Control", Type: TypeYesNo})
	c.ScoreProfile = &ScoreProfile{Version: "v1", Mode: ScoringRisk, Direction: DirectionHighIsPoor, Contributions: []ScoreContribution{{ID: "control", Weight: 100, Predicate: Predicate{FieldID: "control", Operator: PredicateEquals, Values: []string{"Yes"}}, Missing: MissingIndeterminate}}, Rules: []ScoreRule{{ID: "floor", Predicate: Predicate{FieldID: "control", Operator: PredicateAnswered}, Effect: RuleEffect{Kind: EffectFloor, Value: 85}}}}
	r, err = EvaluateAssessment(c, TextAnswers(map[string]string{"control": "Yes"}), map[string]string{"doc": "good"})
	if err != nil {
		t.Fatal(err)
	}
	if r.AdverseScore == nil || *r.AdverseScore < 85 {
		t.Fatalf("manual erased floor: %+v", r)
	}
}
func TestAssessmentNoneCannotRetainAutomaticProfileContribution(t *testing.T) {
	c := Contract{ScoringMode: ScoringRisk, Fields: []Field{{ID: "a", Label: "Answer", Type: TypeYesNo, Assessment: &FieldAssessment{Mode: AssessmentNone}}}, ScoreProfile: &ScoreProfile{Version: "v1", Mode: ScoringRisk, Contributions: []ScoreContribution{{ID: "c", Weight: 100, Predicate: Predicate{FieldID: "a", Operator: PredicateAnswered}, Missing: MissingIndeterminate}}}}
	if _, err := Normalize(c); err == nil {
		t.Fatal("unscored field retained an automatic contribution")
	}
}

func TestAssessmentAutomaticWeightsAndCombinedCannotReduceConcern(t *testing.T) {
	c := Contract{ScoringMode: ScoringRisk, Fields: []Field{{ID: "a", Label: "Control", Type: TypeYesNo, Assessment: &FieldAssessment{Mode: AssessmentAutomaticReview, Required: true, Weight: 80, ReviewerRole: "RISK", Rubric: []AssessmentOutcome{{ID: "good", Label: "Sufficient", Points: 0}}}}, {ID: "b", Label: "Other control", Type: TypeYesNo, Assessment: &FieldAssessment{Mode: AssessmentAutomatic, Weight: 20}}}, ScoreProfile: &ScoreProfile{Version: "v1", Mode: ScoringRisk, Contributions: []ScoreContribution{{ID: "a", Weight: 1, Predicate: Predicate{FieldID: "a", Operator: PredicateEquals, Values: []string{"Yes"}}, MatchPoints: 100, Missing: MissingIndeterminate}, {ID: "b", Weight: 1, Predicate: Predicate{FieldID: "b", Operator: PredicateAnswered}, Missing: MissingIndeterminate}}}}
	n, err := Normalize(c)
	if err != nil {
		t.Fatal(err)
	}
	automatic, err := EvaluateScoreProfile(*n.ScoreProfile, n, TextAnswers(map[string]string{"a": "Yes", "b": "Yes"}))
	if err != nil {
		t.Fatal(err)
	}
	if automatic.RawScore == nil || *automatic.RawScore != 80 {
		t.Fatalf("automatic assessment weights ignored %+v", automatic)
	}
	assessed, err := EvaluateAssessment(n, TextAnswers(map[string]string{"a": "Yes", "b": "Yes"}), map[string]string{"a": "good"})
	if err != nil {
		t.Fatal(err)
	}
	if assessed.RawScore == nil || *assessed.RawScore != 80 {
		t.Fatalf("combined review reduced automatic concern %+v", assessed)
	}
}

func TestAssessmentCombinedRequiresOneDirectAutomaticContribution(t *testing.T) {
	c := Contract{ScoringMode: ScoringRisk, Fields: []Field{{ID: "a", Label: "Control", Type: TypeYesNo, Assessment: &FieldAssessment{Mode: AssessmentAutomaticReview, Required: true, Weight: 100, ReviewerRole: "RISK", Rubric: []AssessmentOutcome{{ID: "good", Label: "Sufficient", Points: 0}}}}, {ID: "b", Label: "Other", Type: TypeYesNo}}, ScoreProfile: &ScoreProfile{Version: "v1", Mode: ScoringRisk, Contributions: []ScoreContribution{{ID: "cross", Weight: 100, Predicate: Predicate{Operator: PredicateAnd, Children: []Predicate{{FieldID: "a", Operator: PredicateAnswered}, {FieldID: "b", Operator: PredicateAnswered}}}, Missing: MissingIndeterminate}}}}
	if _, err := Normalize(c); err == nil {
		t.Fatal("combined rubric had no unique automatic contribution to review")
	}
}

func TestAssessmentCompliancePreservesSectionShareWhenFieldHidden(t *testing.T) {
	var c Contract
	if err := json.Unmarshal([]byte(`{"scoring_mode":"COMPLIANCE","sections":[{"id":"a","title":"Evidence","weight":50},{"id":"b","title":"Controls","weight":50}],"fields":[{"id":"applies","section_id":"a","label":"Applies","type":"yes_no"},{"id":"doc","section_id":"a","label":"Report","type":"file","assessment":{"mode":"MANUAL","required":true,"weight":50,"reviewer_role":"RISK","rubric":[{"id":"poor","label":"Incomplete","points":0}]}},{"id":"hidden","section_id":"a","label":"Other report","type":"file","condition":{"field_id":"applies","operator":"EQUALS","values":["Yes"]},"assessment":{"mode":"MANUAL","required":true,"weight":50,"reviewer_role":"RISK","rubric":[{"id":"good","label":"Sufficient","points":100}]}},{"id":"control","section_id":"b","label":"Control","type":"file","assessment":{"mode":"MANUAL","required":true,"weight":100,"reviewer_role":"RISK","rubric":[{"id":"good","label":"Sufficient","points":100}]}}]}`), &c); err != nil {
		t.Fatal(err)
	}
	r, err := EvaluateAssessment(c, TextAnswers(map[string]string{"applies": "No"}), map[string]string{"doc": "poor", "control": "good"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Final || r.RawScore == nil || *r.RawScore != 50 {
		t.Fatalf("hidden field changed section share: %+v", r)
	}
}
