package apiquestion

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestQuestionFingerprintPinsExactDefinitionAndResultType(t *testing.T) {
	field := formcontract.Field{ID: "q1", SectionID: "general", Label: "Is access reviewed?", Type: formcontract.TypeYesNo, Required: true}
	baseline, err := questionFingerprint(field, "BOOLEAN")
	if err != nil || len(baseline) != 64 {
		t.Fatalf("fingerprint %q: %v", baseline, err)
	}
	repeated, err := questionFingerprint(field, "BOOLEAN")
	if err != nil || repeated != baseline {
		t.Fatalf("fingerprint not deterministic: %q %v", repeated, err)
	}
	changed := field
	changed.Required = false
	next, err := questionFingerprint(changed, "BOOLEAN")
	if err != nil || next == baseline {
		t.Fatalf("changed definition not captured: %q %v", next, err)
	}
	changed = field
	changed.Label = "Is access reviewed every month?"
	next, err = questionFingerprint(changed, "BOOLEAN")
	if err != nil || next == baseline {
		t.Fatalf("changed wording not captured: %q %v", next, err)
	}
	if _, err = questionFingerprint(field, "NUMBER"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("incompatible result type: %v", err)
	}
}

func TestQuestionFingerprintRejectsInvalidAndUnsupportedFields(t *testing.T) {
	for _, field := range []formcontract.Field{
		{ID: "", SectionID: "general", Label: "Question", Type: formcontract.TypeYesNo},
		{ID: "q", SectionID: "general", Label: "Question", Type: formcontract.TypeShortText},
		{ID: "q", SectionID: "general", Label: "", Type: formcontract.TypeDecimal},
		{ID: strings.Repeat("q", 81), SectionID: "general", Label: "Question", Type: formcontract.TypeDecimal},
	} {
		if _, err := questionFingerprint(field, "NUMBER"); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid field accepted: %#v, %v", field, err)
		}
	}
	for _, kind := range []formcontract.Type{formcontract.TypeDecimal, formcontract.TypePercentage} {
		field := formcontract.Field{ID: "q", SectionID: "general", Label: "Score?", Type: kind}
		if _, err := questionFingerprint(field, "NUMBER"); err != nil {
			t.Fatalf("supported numeric type %q: %v", kind, err)
		}
	}
	invalidNumber := math.NaN()
	field := formcontract.Field{ID: "q", SectionID: "general", Label: "Score?", Type: formcontract.TypeDecimal, Constraints: formcontract.Constraints{Minimum: &invalidNumber}}
	if _, err := questionFingerprint(field, "NUMBER"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-finite constraint accepted: %v", err)
	}
}

func TestQuestionFingerprintIncludesNestedScoringAndVisibility(t *testing.T) {
	field := formcontract.Field{ID: "q", SectionID: "general", Label: "Access?", Type: formcontract.TypeYesNo,
		Condition: &formcontract.VisibilityCondition{FieldID: "prior", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}},
		Scoring:   &formcontract.Scoring{Weight: 10, AnswerScores: map[string]int{"Yes": 0, "No": 5}},
	}
	first, err := questionFingerprint(field, "BOOLEAN")
	if err != nil {
		t.Fatal(err)
	}
	field.Scoring = &formcontract.Scoring{Weight: 10, AnswerScores: map[string]int{"No": 5, "Yes": 0}}
	reordered, err := questionFingerprint(field, "BOOLEAN")
	if err != nil || reordered != first {
		t.Fatalf("equivalent map order changed fingerprint: %q %v", reordered, err)
	}
	field.Condition = &formcontract.VisibilityCondition{FieldID: "prior", Operator: formcontract.ConditionEquals, Values: []string{"No"}}
	changed, err := questionFingerprint(field, "BOOLEAN")
	if err != nil || changed == first {
		t.Fatalf("visibility change not captured: %q %v", changed, err)
	}
}
