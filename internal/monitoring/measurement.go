package monitoring

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func normalizeMeasurementSpec(input *MeasurementSpec) (*MeasurementSpec, error) {
	if input == nil {
		return nil, nil
	}
	value := *input
	value.Field = strings.TrimSpace(value.Field)
	value.Label = strings.TrimSpace(value.Label)
	value.Currency = strings.ToUpper(strings.TrimSpace(value.Currency))
	if value.Field == "" {
		return nil, fmt.Errorf("measurement field is required")
	}
	if len(value.Label) > 200 {
		return nil, fmt.Errorf("measurement label is too long")
	}
	if value.Precision < 0 || value.Precision > 6 {
		return nil, fmt.Errorf("measurement precision must be between 0 and 6")
	}
	switch value.Unit {
	case MeasurementCount, MeasurementPercent:
		if value.Currency != "" || value.DurationUnit != "" {
			return nil, fmt.Errorf("count and percent measurements cannot declare currency or duration units")
		}
	case MeasurementMoney:
		if !currencyCodePattern.MatchString(value.Currency) || value.DurationUnit != "" {
			return nil, fmt.Errorf("money measurements require a three-letter currency")
		}
	case MeasurementDuration:
		if value.Currency != "" || !validDurationUnit(value.DurationUnit) {
			return nil, fmt.Errorf("duration measurements require seconds, minutes, hours or days")
		}
	default:
		return nil, fmt.Errorf("measurement unit is invalid")
	}
	return &value, nil
}

func validDurationUnit(value MeasurementDurationUnit) bool {
	switch value {
	case DurationSeconds, DurationMinutes, DurationHours, DurationDays:
		return true
	default:
		return false
	}
}

func validateFormMeasurementField(spec MeasurementSpec, fields []TemplateField) error {
	for _, field := range fields {
		if field.ID != spec.Field {
			continue
		}
		switch spec.Unit {
		case MeasurementCount:
			if field.Type != formcontract.TypeInteger {
				return fmt.Errorf("count measurements require an integer form field")
			}
		case MeasurementPercent:
			if field.Type != formcontract.TypePercentage && field.Type != formcontract.TypeDecimal {
				return fmt.Errorf("percent measurements require a percentage or decimal form field")
			}
		case MeasurementMoney:
			if field.Type != formcontract.TypeCurrency {
				return fmt.Errorf("money measurements require a currency form field")
			}
			if field.Constraints.Currency != "" && !strings.EqualFold(field.Constraints.Currency, spec.Currency) {
				return fmt.Errorf("measurement currency does not match the form field")
			}
		case MeasurementDuration:
			if field.Type != formcontract.TypeInteger && field.Type != formcontract.TypeDecimal {
				return fmt.Errorf("duration measurements require an integer or decimal form field")
			}
		}
		return nil
	}
	return fmt.Errorf("measurement field is not present in the form")
}

func sourceMeasurementHasRule(spec MeasurementSpec, rules []SourceRule) bool {
	for _, rule := range rules {
		if rule.Field == spec.Field && measurementLimitOperator(rule.Operator) && strings.TrimSpace(rule.Expected) != "" {
			return true
		}
	}
	return false
}

func MeasurementDefinition(spec *MeasurementSpec, rules []SourceRule) *NativeMeasurement {
	if spec == nil {
		return nil
	}
	value := &NativeMeasurement{
		Field: spec.Field, Label: spec.Label, Unit: spec.Unit, Currency: spec.Currency,
		DurationUnit: spec.DurationUnit, Precision: spec.Precision,
	}
	for _, rule := range rules {
		if rule.Field != spec.Field || !measurementLimitOperator(rule.Operator) {
			continue
		}
		expected := strings.TrimSpace(rule.Expected)
		if expected == "" {
			continue
		}
		value.Limits = append(value.Limits, MeasurementLimit{Operator: rule.Operator, Expected: expected})
	}
	return value
}

func captureSourceMeasurement(spec *MeasurementSpec, rules []SourceRule, resolution evidence.SourceResolution) (*NativeMeasurement, error) {
	if spec == nil || resolution.State != evidence.SourceResolutionCurrent || resolution.Receipt == nil ||
		resolution.Receipt.Completeness != sourceaccess.CompletenessComplete || len(resolution.Records) != 1 {
		return nil, nil
	}
	scalar, ok := resolution.Records[0][spec.Field]
	if !ok || scalar.Kind == sourceaccess.ScalarNull {
		return nil, nil
	}
	actual := strings.TrimSpace(scalar.Text)
	if _, ok := parseExactDecimal(actual); !ok {
		return nil, fmt.Errorf("measurement field %s requires a numeric value", spec.Field)
	}
	value := MeasurementDefinition(spec, rules)
	value.Value = actual
	condition, err := EvaluateNativeMeasurementCondition(value)
	if err != nil {
		return nil, err
	}
	value.Condition = condition
	return value, nil
}

func captureFormMeasurement(spec *MeasurementSpec, answers map[string]formcontract.AnswerValue) (*NativeMeasurement, error) {
	if spec == nil {
		return nil, nil
	}
	answer, ok := answers[spec.Field]
	if !ok || answer.Text == nil || strings.TrimSpace(*answer.Text) == "" {
		return nil, nil
	}
	actual := strings.TrimSpace(*answer.Text)
	if _, ok := parseExactDecimal(actual); !ok {
		return nil, fmt.Errorf("measurement field %s requires a numeric answer", spec.Field)
	}
	value := MeasurementDefinition(spec, nil)
	value.Value = actual
	return value, nil
}

func EvaluateNativeMeasurementCondition(measurement *NativeMeasurement) (MeasurementCondition, error) {
	if measurement == nil || strings.TrimSpace(measurement.Value) == "" || len(measurement.Limits) == 0 {
		return MeasurementConditionUnknown, nil
	}
	actual, ok := parseExactDecimal(measurement.Value)
	if !ok {
		return MeasurementConditionUnknown, fmt.Errorf("measurement value is not numeric")
	}
	for _, limit := range measurement.Limits {
		expected, expectedOK := parseExactDecimal(limit.Expected)
		if !expectedOK {
			return MeasurementConditionUnknown, fmt.Errorf("measurement limit is not numeric")
		}
		comparison := actual.Cmp(expected)
		passed := false
		switch limit.Operator {
		case OperatorEquals:
			passed = comparison == 0
		case OperatorNotEquals:
			passed = comparison != 0
		case OperatorGreaterThan:
			passed = comparison > 0
		case OperatorGreaterOrEqual:
			passed = comparison >= 0
		case OperatorLessThan:
			passed = comparison < 0
		case OperatorLessOrEqual:
			passed = comparison <= 0
		default:
			return MeasurementConditionUnknown, fmt.Errorf("measurement limit operator is unsupported")
		}
		if !passed {
			return MeasurementConditionBreached, nil
		}
	}
	return MeasurementConditionWithin, nil
}

func measurementLimitOperator(value SourceOperator) bool {
	switch value {
	case OperatorEquals, OperatorNotEquals, OperatorGreaterThan, OperatorGreaterOrEqual, OperatorLessThan, OperatorLessOrEqual:
		return true
	default:
		return false
	}
}

func parseExactDecimal(value string) (*big.Rat, bool) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	return parsed, ok
}
