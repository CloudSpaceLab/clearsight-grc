package monitoring

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

var (
	currencyCodePattern = regexp.MustCompile("^[A-Z]{3}$")
	exactDecimalPattern = regexp.MustCompile("^([+-]?)(\\d+)(?:\\.(\\d+))?(?:[eE]([+-]?\\d+))?$")
)

func normalizeMeasurementSpec(input *MeasurementSpec) (*MeasurementSpec, error) {
	if input == nil {
		return nil, nil
	}
	value := *input
	value.Field = strings.TrimSpace(value.Field)
	value.Label = strings.TrimSpace(value.Label)
	value.Currency = strings.ToUpper(strings.TrimSpace(value.Currency))
	value.Limits = append([]MeasurementLimit(nil), value.Limits...)
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
	if len(value.Limits) > 4 {
		return nil, fmt.Errorf("measurement may define at most four limits")
	}
	for index := range value.Limits {
		value.Limits[index].Expected = strings.TrimSpace(value.Limits[index].Expected)
		if !measurementLimitOperator(value.Limits[index].Operator) || value.Limits[index].Expected == "" {
			return nil, fmt.Errorf("measurement limit is invalid")
		}
		if _, ok := parseExactDecimal(value.Limits[index].Expected); !ok {
			return nil, fmt.Errorf("measurement limit requires a numeric value")
		}
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
		if rule.Field != spec.Field || !measurementLimitOperator(rule.Operator) {
			continue
		}
		if _, ok := parseExactDecimal(rule.Expected); ok {
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
		Limits: append([]MeasurementLimit(nil), spec.Limits...),
	}
	if len(value.Limits) > 0 {
		return value
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
	if scalar.Kind != sourceaccess.ScalarNumber {
		return nil, fmt.Errorf("measurement field %s is no longer numeric", spec.Field)
	}
	actual := strings.TrimSpace(scalar.Text)
	parsed, ok := parseExactDecimal(actual)
	if !ok {
		return nil, fmt.Errorf("measurement field %s requires a numeric value", spec.Field)
	}
	if err := validateNativeMeasurementValue(*spec, parsed); err != nil {
		return nil, err
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
	parsed, ok := parseExactDecimal(actual)
	if !ok {
		return nil, fmt.Errorf("measurement field %s requires a numeric answer", spec.Field)
	}
	if err := validateNativeMeasurementValue(*spec, parsed); err != nil {
		return nil, err
	}
	value := MeasurementDefinition(spec, nil)
	value.Value = actual
	condition, err := EvaluateNativeMeasurementCondition(value)
	if err != nil {
		return nil, err
	}
	value.Condition = condition
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

func validateNativeMeasurementValue(spec MeasurementSpec, value *big.Rat) error {
	if spec.Unit == MeasurementCount && !value.IsInt() {
		return fmt.Errorf("count measurement field %s requires a whole number", spec.Field)
	}
	if spec.Unit == MeasurementDuration && value.Sign() < 0 {
		return fmt.Errorf("duration measurement field %s cannot be negative", spec.Field)
	}
	return nil
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
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return nil, false
	}
	match := exactDecimalPattern.FindStringSubmatch(value)
	if match == nil {
		return nil, false
	}
	exponent := 0
	if match[4] != "" {
		parsedExponent, err := strconv.Atoi(match[4])
		if err != nil || parsedExponent < -100 || parsedExponent > 100 {
			return nil, false
		}
		exponent = parsedExponent
	}
	digits := match[2] + match[3]
	numerator := new(big.Int)
	if _, ok := numerator.SetString(digits, 10); !ok {
		return nil, false
	}
	if match[1] == "-" {
		numerator.Neg(numerator)
	}
	scale := len(match[3]) - exponent
	if scale <= 0 {
		numerator.Mul(numerator, pow10(-scale))
		return new(big.Rat).SetInt(numerator), true
	}
	return new(big.Rat).SetFrac(numerator, pow10(scale)), true
}

func pow10(exponent int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
}
