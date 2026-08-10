package assurance

import (
	"encoding/json"
	"testing"
)

func TestNumberEvaluationRejectsIntegersOutsideExactFloatRange(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "amount", Type: TypeNumber}}}
	compiled, err := CompileCondition(schema, Condition{Op: OpGT, Field: "amount", Value: NumberLiteral(1)}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}

	for name, value := range map[string]any{
		"signed":      int64(maxExactFloatInteger + 1),
		"unsigned":    uint64(maxExactFloatInteger + 1),
		"json-number": json.Number("9007199254740993"),
	} {
		t.Run(name, func(t *testing.T) {
			if got := compiled.Evaluate(map[string]any{"amount": value}); got != ResultUnknown {
				t.Fatalf("unsafe integer must fail closed as UNKNOWN, got %s", got)
			}
		})
	}

	if got := compiled.Evaluate(map[string]any{"amount": int64(maxExactFloatInteger)}); got != ResultMatch {
		t.Fatalf("largest exactly representable integer should remain evaluable, got %s", got)
	}
}

func TestConditionRejectsNumericLiteralOutsideBoundedDomain(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "amount", Type: TypeNumber}}}
	if _, err := CompileCondition(schema, Condition{Op: OpGT, Field: "amount", Value: NumberLiteral(float64(maxExactFloatInteger) + 2)}, ConditionLimits{}); err == nil {
		t.Fatal("numeric literal outside the T0 bounded domain must be rejected")
	}
	if _, err := CompileCondition(schema, Condition{Op: OpGT, Field: "amount", Value: NumberLiteral(float64(maxExactFloatInteger))}, ConditionLimits{}); err != nil {
		t.Fatalf("bounded numeric literal should remain valid: %v", err)
	}
}
