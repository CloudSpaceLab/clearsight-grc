package monitoring

import (
	"strings"
)

type MeasurementMovementDirection string

const (
	MeasurementMovementUp   MeasurementMovementDirection = "UP"
	MeasurementMovementDown MeasurementMovementDirection = "DOWN"
	MeasurementMovementFlat MeasurementMovementDirection = "FLAT"
)

type MeasurementMovement struct {
	Direction MeasurementMovementDirection `json:"direction"`
	Delta     string                       `json:"delta"`
}

func CompareNativeMeasurements(current, previous *NativeMeasurement) (MeasurementMovement, bool) {
	if !compatibleNativeMeasurements(current, previous) {
		return MeasurementMovement{}, false
	}
	currentValue, currentOK := parseExactDecimal(current.Value)
	previousValue, previousOK := parseExactDecimal(previous.Value)
	if !currentOK || !previousOK {
		return MeasurementMovement{}, false
	}
	delta := currentValue.Sub(currentValue, previousValue)
	precision := nativeMeasurementDeltaPrecision(current, previous)
	text := delta.FloatString(precision)
	direction := MeasurementMovementFlat
	if delta.Sign() > 0 {
		direction = MeasurementMovementUp
		if !strings.HasPrefix(text, "+") {
			text = "+" + text
		}
	} else if delta.Sign() < 0 {
		direction = MeasurementMovementDown
	}
	return MeasurementMovement{Direction: direction, Delta: text}, true
}

func compatibleNativeMeasurements(current, previous *NativeMeasurement) bool {
	if current == nil || previous == nil || strings.TrimSpace(current.Value) == "" || strings.TrimSpace(previous.Value) == "" {
		return false
	}
	return current.Field == previous.Field &&
		current.Unit == previous.Unit &&
		current.Currency == previous.Currency &&
		current.DurationUnit == previous.DurationUnit
}

func nativeMeasurementDeltaPrecision(values ...*NativeMeasurement) int {
	precision := 0
	for _, value := range values {
		if value == nil {
			continue
		}
		if value.Precision > precision {
			precision = value.Precision
		}
		if scale := exactDecimalScale(value.Value); scale > precision {
			precision = scale
		}
	}
	return precision
}

func exactDecimalScale(value string) int {
	match := exactDecimalPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return 0
	}
	exponent := 0
	if match[4] != "" {
		for _, char := range strings.TrimPrefix(strings.TrimPrefix(match[4], "+"), "-") {
			if char < '0' || char > '9' {
				return 0
			}
			exponent = exponent*10 + int(char-'0')
		}
		if strings.HasPrefix(match[4], "-") {
			exponent = -exponent
		}
	}
	scale := len(match[3]) - exponent
	if scale < 0 {
		return 0
	}
	return scale
}
