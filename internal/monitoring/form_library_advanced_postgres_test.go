//go:build postgres

package monitoring

import (
	"strings"
	"testing"
)

func TestAdvancedFormFilterSQLUsesOnlyTypedParameterizedFields(t *testing.T) {
	expression, err := NormalizeFormFilterExpression(&FormFilterExpression{Kind: "group", Operator: "or", Children: []FormFilterExpression{
		{Kind: "condition", Field: FormFilterStatus, Operator: "is", Value: "active"},
		{Kind: "group", Operator: "and", Children: []FormFilterExpression{
			{Kind: "condition", Field: FormFilterProgram, Operator: "is", Value: "00000000-0000-0000-0000-000000000001"},
			{Kind: "condition", Field: FormFilterTag, Operator: "is", Value: "Third-Party"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	query, args, next := formFilterSQL(expression, "f", 4)
	if len(args) != 3 || next != 7 {
		t.Fatalf("args = %#v, next = %d", args, next)
	}
	for _, value := range []string{"ACTIVE", "00000000-0000-0000-0000-000000000001", "third-party"} {
		if strings.Contains(query, value) {
			t.Fatalf("query interpolated value %q: %s", value, query)
		}
	}
	for _, fragment := range []string{"f.status=$4", "f.program_id=NULLIF($5,'')::uuid", "f.tags @> ARRAY[$6]::text[]", " OR ", " AND "} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query %q missing %q", query, fragment)
		}
	}
}
