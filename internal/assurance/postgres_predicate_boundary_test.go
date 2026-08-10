package assurance

import (
	"strings"
	"testing"
	"time"
)

func TestPostgresPredicateCarriesBoundedValueGuards(t *testing.T) {
	schema := Schema{Fields: []Field{
		{Name: "status", Type: TypeString},
		{Name: "score", Type: TypeNumber},
	}}
	compiled, err := CompileCondition(schema, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGTE, Field: "score", Value: NumberLiteral(80)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := compiled.PostgresPredicate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(predicate.MatchSQL, "octet_length") || !strings.Contains(predicate.UnknownSQL, "octet_length") {
		t.Fatalf("string validity guard missing: %+v", predicate)
	}
	if !strings.Contains(predicate.MatchSQL, "double precision") || !strings.Contains(predicate.UnknownSQL, "9007199254740992") {
		t.Fatalf("number validity guard missing: %+v", predicate)
	}
}

func TestPostgresPredicateRejectsSubMicrosecondTimeLiteral(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "due_at", Type: TypeTime}}}
	compiled, err := CompileCondition(schema, Condition{
		Op:    OpGTE,
		Field: "due_at",
		Value: TimeLiteral(time.Date(2026, 8, 10, 12, 0, 0, 123, time.UTC)),
	}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compiled.PostgresPredicate(); err == nil {
		t.Fatal("PostgreSQL pushdown must reject time literals finer than PostgreSQL microsecond precision")
	}
}
