package monitoring

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview/measure"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestSourceNumericComparisonPreservesExactBoundaries(t *testing.T) {
	cases := []struct {
		name, actual, expected string
		operator               SourceOperator
		passed                 bool
	}{
		{"large count above", "9007199254740993", "9007199254740992", OperatorGreaterThan, true},
		{"large amount not below", "9007199254740992.01", "9007199254740992.00", OperatorLessOrEqual, false},
		{"precise lower boundary", "0.100000000000000001", "0.1", OperatorGreaterThan, true},
		{"inclusive lower equality", "98.700", "98.7", OperatorGreaterOrEqual, true},
		{"exclusive lower equality", "98.700", "98.7", OperatorGreaterThan, false},
		{"inclusive upper equality", "120.00", "120", OperatorLessOrEqual, true},
		{"exclusive upper equality", "120.00", "120", OperatorLessThan, false},
		{"negative boundary", "-1.000000000000000001", "-1", OperatorLessThan, true},
		{"scientific rate", "9.87e1", "99.5", OperatorLessThan, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := numericSourceEvaluation(tc.actual, tc.expected, tc.operator)
			if err != nil {
				t.Fatal(err)
			}
			want, points := RuleFailed, 100.0
			if tc.passed {
				want, points = RulePassed, 0
			}
			if len(result.RuleResults) != 1 || result.RuleResults[0].Outcome != want || result.Score == nil || *result.Score != points || result.Coverage != 1 {
				t.Fatalf("unexpected numeric evaluation: %+v", result)
			}
			if result.SourceComparisonVersion != measure.DecimalComparisonVersion {
				t.Fatal("source comparison provenance was not retained")
			}
		})
	}
}

func TestSourceNumericComparisonRejectsNonFiniteAndInvalidInput(t *testing.T) {
	invalid := []string{"NaN", "+Inf", "-Inf", "Infinity", "1/2", "1e9999999", "1_000", "1,000", strings.Repeat("9", measure.MaxDecimalDigits+1)}
	for _, value := range invalid {
		for _, operator := range []SourceOperator{OperatorGreaterThan, OperatorGreaterOrEqual, OperatorLessThan, OperatorLessOrEqual} {
			if result, err := numericSourceEvaluation(value, "10", operator); err == nil || result.Score != nil {
				t.Errorf("invalid source produced a score: %+v", result)
			}
			if result, err := numericSourceEvaluation("10", value, operator); err == nil || result.Score != nil {
				t.Errorf("invalid limit produced a score: %+v", result)
			}
		}
	}
}

func TestSourceNumericComparisonDoesNotChangeTextOrBooleanEquality(t *testing.T) {
	for _, tc := range []struct {
		value sourceaccess.Scalar
		limit string
		want  bool
	}{
		{sourceaccess.Scalar{Kind: sourceaccess.ScalarBool, Text: "TRUE"}, "true", true},
		{sourceaccess.Scalar{Text: "01"}, "1", false},
		{sourceaccess.Scalar{Text: "Open"}, "open", false},
	} {
		got, err := compareScalar(tc.value, SourceRule{ID: "status", Operator: OperatorEquals, Expected: tc.limit}, time.Time{})
		if err != nil || got != tc.want {
			t.Fatalf("existing equality semantics changed: %v, %v", got, err)
		}
	}
}

func TestSourceNumericComparisonJSONPreservesProvenanceWithoutSourceValue(t *testing.T) {
	result, err := numericSourceEvaluation("9007199254740992.01", "9007199254740992.00", OperatorLessOrEqual)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"source_comparison_version":"decimal-v1"`) || strings.Contains(string(data), "9007199254740992") {
		t.Fatal("comparison version missing or raw source value disclosed")
	}
	var decoded Evaluation
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.SourceComparisonVersion != measure.DecimalComparisonVersion {
		t.Fatal("evaluation provenance did not round trip")
	}
	var legacy Evaluation
	if err := json.Unmarshal([]byte(`{"band":"LOW","coverage":1,"score":0}`), &legacy); err != nil || legacy.SourceComparisonVersion != "" {
		t.Fatal("historical evaluation is not readable or was relabelled")
	}
}

func numericSourceEvaluation(actual, expected string, operator SourceOperator) (Evaluation, error) {
	return EvaluateSource(
		[]SourceRule{{ID: "native-boundary", Field: "value", Operator: operator, Expected: expected, RiskPoints: 100}},
		evidence.SourceResolution{
			State: evidence.SourceResolutionCurrent,
			Records: []sourceaccess.Record{{"value": {Text: actual}}},
			Receipt: &sourceaccess.OperationReceipt{Completeness: sourceaccess.CompletenessComplete},
		},
		DefaultThresholds(), time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
	)
}
