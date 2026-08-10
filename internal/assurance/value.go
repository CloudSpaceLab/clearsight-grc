package assurance

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

const hardMaxEvaluatedStringBytes = 64 << 10

func coerceValue(kind LogicalType, raw any) (typedValue, bool) {
	switch kind {
	case TypeString:
		value, ok := raw.(string)
		if !ok || len(value) > hardMaxEvaluatedStringBytes {
			return typedValue{}, false
		}
		return typedValue{kind: kind, stringValue: value}, true
	case TypeNumber:
		value, ok := numericValue(raw)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return typedValue{}, false
		}
		return typedValue{kind: kind, numberValue: value}, true
	case TypeBool:
		value, ok := raw.(bool)
		if !ok {
			return typedValue{}, false
		}
		return typedValue{kind: kind, boolValue: value}, true
	case TypeTime:
		value, ok := timeValue(raw)
		if !ok {
			return typedValue{}, false
		}
		return typedValue{kind: kind, timeValue: value.UTC()}, true
	default:
		return typedValue{}, false
	}
}

func numericValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint8:
		return float64(value), true
	case uint16:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		if value > 1<<53 {
			return 0, false
		}
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func timeValue(raw any) (time.Time, bool) {
	if value, ok := raw.(time.Time); ok {
		return value.UTC(), !value.IsZero()
	}
	value, ok := raw.(string)
	if !ok || len(value) > hardMaxEvaluatedStringBytes {
		return time.Time{}, false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), true
		}
	}
	// Unix seconds are intentionally not inferred from arbitrary strings. A
	// source mapping must make that semantic explicit before profiling/rules.
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Time{}, false
	}
	return time.Time{}, false
}
