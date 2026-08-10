package assurance

import (
	"strings"
	"testing"
)

func TestProfileRowsEnforcesTotalFieldRowCellBudget(t *testing.T) {
	fields := make([]Field, 256)
	for index := range fields {
		fields[index] = Field{Name: "field_" + strings.Repeat("x", index%3) + string(rune('A'+index%26)) + string(rune('a'+index/26)), Type: TypeString}
	}
	rows := make([]map[string]any, 300)
	for index := range rows {
		rows[index] = map[string]any{}
	}
	profile, err := ProfileRows(Schema{Fields: fields}, rows, ProfileLimits{MaxFields: len(fields), MaxRows: len(rows)})
	if err != nil {
		t.Fatal(err)
	}
	wantRows := hardMaxProfileCells / len(fields)
	if profile.RowsObserved != wantRows || profile.RowsOmitted != len(rows)-wantRows {
		t.Fatalf("profile cell budget not enforced: observed=%d omitted=%d want=%d/%d", profile.RowsObserved, profile.RowsOmitted, wantRows, len(rows)-wantRows)
	}
}

func TestConditionRejectsAggregateLiteralCountBudget(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	children := make([]Condition, 5)
	for childIndex := range children {
		values := make([]Literal, hardMaxConditionInValues)
		for valueIndex := range values {
			values[valueIndex] = StringLiteral("ACTIVE")
		}
		children[childIndex] = Condition{Op: OpIn, Field: "status", Values: values}
	}
	_, err := CompileCondition(schema, Condition{Op: OpOr, Children: children}, ConditionLimits{MaxInValues: hardMaxConditionInValues})
	if err == nil || !strings.Contains(err.Error(), "total literal values") {
		t.Fatalf("aggregate literal-count budget should reject condition, got %v", err)
	}
}

func TestConditionRejectsAggregateLiteralByteBudget(t *testing.T) {
	schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
	values := make([]Literal, 17)
	for index := range values {
		values[index] = StringLiteral(strings.Repeat("A", hardMaxConditionStringBytes))
	}
	_, err := CompileCondition(schema, Condition{Op: OpIn, Field: "status", Values: values}, ConditionLimits{MaxInValues: len(values), MaxStringBytes: hardMaxConditionStringBytes})
	if err == nil || !strings.Contains(err.Error(), "literal payload") {
		t.Fatalf("aggregate literal-byte budget should reject condition, got %v", err)
	}
}
