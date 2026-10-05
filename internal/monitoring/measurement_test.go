package monitoring

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestNormalizeMeasurementSpecRequiresUnitMetadata(t *testing.T) {
	for _, test := range []struct {
		name string
		spec MeasurementSpec
	}{
		{name: "money without currency", spec: MeasurementSpec{Field: "loss", Unit: MeasurementMoney}},
		{name: "duration without unit", spec: MeasurementSpec{Field: "downtime", Unit: MeasurementDuration}},
		{name: "percent with currency", spec: MeasurementSpec{Field: "success", Unit: MeasurementPercent, Currency: "NGN"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := normalizeMeasurementSpec(&test.spec); err == nil {
				t.Fatal("expected invalid measurement configuration")
			}
		})
	}
	valid, err := normalizeMeasurementSpec(&MeasurementSpec{
		Field: " loss ", Label: " Net loss ", Unit: MeasurementMoney, Currency: "ngn", Precision: 2,
		Limits: []MeasurementLimit{{Operator: OperatorLessOrEqual, Expected: " 1000000.00 "}},
	})
	if err != nil {
		t.Fatalf("normalize measurement: %v", err)
	}
	if valid.Field != "loss" || valid.Label != "Net loss" || valid.Currency != "NGN" || len(valid.Limits) != 1 || valid.Limits[0].Expected != "1000000.00" {
		t.Fatalf("unexpected normalized measurement: %#v", valid)
	}
}

func TestCaptureSourceMeasurementKeepsNativeValueAndLimits(t *testing.T) {
	spec := &MeasurementSpec{Field: "success_rate", Label: "Success rate", Unit: MeasurementPercent, Precision: 2}
	rules := []SourceRule{{ID: "minimum", Field: "success_rate", Operator: OperatorGreaterOrEqual, Expected: "99.50", RiskPoints: 100}}
	record := sourceaccess.Record{
		"success_rate": {Kind: sourceaccess.ScalarNumber, Text: "98.70"},
	}
	measurement, err := captureSourceMeasurement(spec, rules, evidence.SourceResolution{
		State:   evidence.SourceResolutionCurrent,
		Records: []sourceaccess.Record{record},
		Receipt: &sourceaccess.OperationReceipt{Completeness: sourceaccess.CompletenessComplete},
	})
	if err != nil {
		t.Fatalf("capture measurement: %v", err)
	}
	if measurement == nil || measurement.Value != "98.70" || len(measurement.Limits) != 1 ||
		measurement.Limits[0].Operator != OperatorGreaterOrEqual || measurement.Limits[0].Expected != "99.50" ||
		measurement.Condition != MeasurementConditionBreached {
		t.Fatalf("unexpected measurement: %#v", measurement)
	}
}

func TestFormMeasurementRequiresCompatibleTypedField(t *testing.T) {
	money := MeasurementSpec{
		Field: "loss", Unit: MeasurementMoney, Currency: "NGN", Precision: 2,
		Limits: []MeasurementLimit{{Operator: OperatorLessOrEqual, Expected: "1000000"}},
	}
	if err := validateFormMeasurementField(money, []TemplateField{{ID: "loss", Type: formcontract.TypeLongText}}); err == nil {
		t.Fatal("long-text historical field must not become a money measurement")
	}
	if err := validateFormMeasurementField(money, []TemplateField{{ID: "loss", Type: formcontract.TypeCurrency, Constraints: formcontract.Constraints{Currency: "NGN"}}}); err != nil {
		t.Fatalf("typed currency field rejected: %v", err)
	}
	text := "1250000.00"
	measurement, err := captureFormMeasurement(&money, map[string]formcontract.AnswerValue{"loss": {Text: &text}})
	if err != nil || measurement == nil || measurement.Value != text || measurement.Condition != MeasurementConditionBreached {
		t.Fatalf("form measurement = %#v, err=%v", measurement, err)
	}
}

func TestNativeMeasurementConditionUsesExactLimits(t *testing.T) {
	within, err := EvaluateNativeMeasurementCondition(&NativeMeasurement{
		Unit: MeasurementPercent, Value: "99.50",
		Limits: []MeasurementLimit{{Operator: OperatorGreaterOrEqual, Expected: "99.500"}},
	})
	if err != nil || within != MeasurementConditionWithin {
		t.Fatalf("within condition = %s, err=%v", within, err)
	}

	breached, err := EvaluateNativeMeasurementCondition(&NativeMeasurement{
		Unit: MeasurementPercent, Value: "99.499999999999999999",
		Limits: []MeasurementLimit{{Operator: OperatorGreaterOrEqual, Expected: "99.5"}},
	})
	if err != nil || breached != MeasurementConditionBreached {
		t.Fatalf("breached condition = %s, err=%v", breached, err)
	}

	unknown, err := EvaluateNativeMeasurementCondition(&NativeMeasurement{Unit: MeasurementPercent, Value: "99.5"})
	if err != nil || unknown != MeasurementConditionUnknown {
		t.Fatalf("unknown condition = %s, err=%v", unknown, err)
	}
}

func TestNormalizeMeasurementSpecRejectsInvalidNativeLimits(t *testing.T) {
	for _, spec := range []MeasurementSpec{
		{Field: "value", Unit: MeasurementCount, Limits: []MeasurementLimit{{Operator: OperatorPresent, Expected: "1"}}},
		{Field: "value", Unit: MeasurementCount, Limits: []MeasurementLimit{{Operator: OperatorGreaterOrEqual, Expected: "not-a-number"}}},
		{Field: "value", Unit: MeasurementCount, Limits: []MeasurementLimit{
			{Operator: OperatorGreaterOrEqual, Expected: "1"},
			{Operator: OperatorLessOrEqual, Expected: "2"},
			{Operator: OperatorNotEquals, Expected: "1.5"},
			{Operator: OperatorEquals, Expected: "2"},
			{Operator: OperatorGreaterThan, Expected: "0"},
		}},
	} {
		if _, err := normalizeMeasurementSpec(&spec); err == nil {
			t.Fatalf("invalid native limit was accepted: %#v", spec)
		}
	}
}

func TestParseExactDecimalRejectsNonDecimalSyntaxAndKeepsExponentExact(t *testing.T) {
	if _, ok := parseExactDecimal("1/2"); ok {
		t.Fatal("fraction syntax must not be accepted as a decimal measurement")
	}
	if _, ok := parseExactDecimal("NaN"); ok {
		t.Fatal("non-finite syntax must not be accepted")
	}
	left, ok := parseExactDecimal("9.007199254740993e15")
	if !ok {
		t.Fatal("scientific decimal was rejected")
	}
	right, ok := parseExactDecimal("9007199254740992")
	if !ok || left.Cmp(right) <= 0 {
		t.Fatalf("scientific exact comparison collapsed: %v vs %v", left, right)
	}
	if _, ok := parseExactDecimal("1e101"); ok {
		t.Fatal("unbounded exponent must fail closed")
	}
}
