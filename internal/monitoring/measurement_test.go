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
	valid, err := normalizeMeasurementSpec(&MeasurementSpec{Field: " loss ", Label: " Net loss ", Unit: MeasurementMoney, Currency: "ngn", Precision: 2})
	if err != nil {
		t.Fatalf("normalize measurement: %v", err)
	}
	if valid.Field != "loss" || valid.Label != "Net loss" || valid.Currency != "NGN" {
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
		measurement.Limits[0].Operator != OperatorGreaterOrEqual || measurement.Limits[0].Expected != "99.50" {
		t.Fatalf("unexpected measurement: %#v", measurement)
	}
}

func TestFormMeasurementRequiresCompatibleTypedField(t *testing.T) {
	money := MeasurementSpec{Field: "loss", Unit: MeasurementMoney, Currency: "NGN", Precision: 2}
	if err := validateFormMeasurementField(money, []TemplateField{{ID: "loss", Type: formcontract.TypeLongText}}); err == nil {
		t.Fatal("long-text historical field must not become a money measurement")
	}
	if err := validateFormMeasurementField(money, []TemplateField{{ID: "loss", Type: formcontract.TypeCurrency, Constraints: formcontract.Constraints{Currency: "NGN"}}}); err != nil {
		t.Fatalf("typed currency field rejected: %v", err)
	}
	text := "1250000.00"
	measurement, err := captureFormMeasurement(&money, map[string]formcontract.AnswerValue{"loss": {Text: &text}})
	if err != nil || measurement == nil || measurement.Value != text {
		t.Fatalf("form measurement = %#v, err=%v", measurement, err)
	}
}
