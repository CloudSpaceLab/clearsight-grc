package monitoring

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestEvaluateFormUsesConfiguredWeightsAndThresholds(t *testing.T) {
	fields := []FormField{
		{ID: "identity_check", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
		{ID: "session_revocation", Required: true, Weight: 3, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
	}

	result, err := EvaluateForm(fields, formcontract.TextAnswers(map[string]string{"identity_check": "Yes", "session_revocation": "No"}), DefaultThresholds())
	if err != nil {
		t.Fatalf("evaluate form: %v", err)
	}
	if result.Score == nil || *result.Score != 75 {
		t.Fatalf("score = %#v, want 75", result.Score)
	}
	if result.Band != RiskCritical || result.Coverage != 1 {
		t.Fatalf("result = %#v, want critical with full coverage", result)
	}
}

func TestEvaluateFormCriticalAnswerOverridesWeightedBand(t *testing.T) {
	fields := []FormField{
		{ID: "reset_channel", Required: true, Weight: 9, AnswerScores: map[string]int{"Yes": 0, "No": 20}, CriticalAnswers: []string{"No"}},
		{ID: "logging", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
	}

	result, err := EvaluateForm(fields, formcontract.TextAnswers(map[string]string{"reset_channel": "No", "logging": "Yes"}), DefaultThresholds())
	if err != nil {
		t.Fatalf("evaluate form: %v", err)
	}
	if result.Score == nil || *result.Score != 18 {
		t.Fatalf("score = %#v, want 18", result.Score)
	}
	if result.Band != RiskCritical || len(result.CriticalFailures) != 1 || result.CriticalFailures[0].FieldID != "reset_channel" {
		t.Fatalf("critical result = %#v", result)
	}
}

func TestEvaluateFormMissingRequiredAnswerIsNotAssessed(t *testing.T) {
	fields := []FormField{
		{ID: "identity_check", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
		{ID: "logging", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
	}

	result, err := EvaluateForm(fields, formcontract.TextAnswers(map[string]string{"identity_check": "Yes"}), DefaultThresholds())
	if err != nil {
		t.Fatalf("evaluate form: %v", err)
	}
	if result.Score != nil || result.Band != RiskNotAssessed || result.Coverage != .5 {
		t.Fatalf("result = %#v, want not assessed at 50%% coverage", result)
	}
}

func TestEvaluateFormRejectsAnswerOutsideConfiguredChoices(t *testing.T) {
	_, err := EvaluateForm(
		[]FormField{{ID: "identity_check", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}},
		formcontract.TextAnswers(map[string]string{"identity_check": "Probably"}),
		DefaultThresholds(),
	)
	if err == nil {
		t.Fatal("expected invalid answer to fail")
	}
}

func TestEvaluateSourcePassesCurrentCompleteValues(t *testing.T) {
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	resolution := evidence.SourceResolution{
		State: evidence.SourceResolutionCurrent,
		Records: []sourceaccess.Record{{
			"sdk_enabled": {Kind: sourceaccess.ScalarBool, Text: "true"},
			"observed_at": {Kind: sourceaccess.ScalarTime, Text: now.Add(-15 * time.Minute).Format(time.RFC3339)},
		}},
		Receipt: &sourceaccess.OperationReceipt{Completeness: sourceaccess.CompletenessComplete},
	}
	rules := []SourceRule{
		{ID: "enabled", Field: "sdk_enabled", Operator: OperatorEquals, Expected: "true", RiskPoints: 100, Critical: true},
		{ID: "fresh", Field: "observed_at", Operator: OperatorMaxAgeMinutes, Expected: "60", RiskPoints: 100, Critical: true},
	}

	result, err := EvaluateSource(rules, resolution, DefaultThresholds(), now)
	if err != nil {
		t.Fatalf("evaluate source: %v", err)
	}
	if result.Score == nil || *result.Score != 0 || result.Band != RiskLow || result.Coverage != 1 {
		t.Fatalf("result = %#v, want low risk", result)
	}
}

func TestEvaluateSourceStaleOrPartialInputIsNotAssessed(t *testing.T) {
	for _, test := range []struct {
		name       string
		resolution evidence.SourceResolution
	}{
		{name: "stale", resolution: evidence.SourceResolution{State: evidence.SourceResolutionStale}},
		{name: "partial", resolution: evidence.SourceResolution{State: evidence.SourceResolutionCurrent, Receipt: &sourceaccess.OperationReceipt{Completeness: sourceaccess.CompletenessPartial}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := EvaluateSource([]SourceRule{{ID: "enabled", Field: "sdk_enabled", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}}, test.resolution, DefaultThresholds(), time.Now().UTC())
			if err != nil {
				t.Fatalf("evaluate source: %v", err)
			}
			if result.Score != nil || result.Band != RiskNotAssessed {
				t.Fatalf("result = %#v, want not assessed", result)
			}
		})
	}
}

func TestCompareScalarKeepsExactDecimalOrdering(t *testing.T) {
	passed, err := compareScalar(
		sourceaccess.Scalar{Kind: sourceaccess.ScalarNumber, Text: "9007199254740993"},
		SourceRule{ID: "precise", Field: "value", Operator: OperatorGreaterThan, Expected: "9007199254740992"},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("compare exact decimal: %v", err)
	}
	if !passed {
		t.Fatal("exact decimal comparison collapsed distinct integer values")
	}
}

func TestCompareScalarNumericEqualityIsExact(t *testing.T) {
	for _, test := range []struct {
		operator SourceOperator
		actual   string
		expected string
		want     bool
	}{
		{operator: OperatorEquals, actual: "1.0", expected: "1", want: true},
		{operator: OperatorNotEquals, actual: "1.00", expected: "1", want: false},
		{operator: OperatorNotEquals, actual: "1.0001", expected: "1", want: true},
	} {
		passed, err := compareScalar(
			sourceaccess.Scalar{Kind: sourceaccess.ScalarNumber, Text: test.actual},
			SourceRule{ID: "numeric-equality", Field: "value", Operator: test.operator, Expected: test.expected},
			time.Now().UTC(),
		)
		if err != nil || passed != test.want {
			t.Fatalf("%s %s %s = %v, err=%v, want %v", test.actual, test.operator, test.expected, passed, err, test.want)
		}
	}
}
