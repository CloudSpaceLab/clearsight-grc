package reporting

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeReportFilterRejectsUnknownField(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: "secret_column", Operator: "is", Value: "x",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unknown field to be rejected, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsUnknownOperator(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: ReportFieldStatus, Operator: "matches_regex", Value: "OPEN",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unknown operator to be rejected, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsNodeAndDepthBudget(t *testing.T) {
	deep := &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
		{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
				{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
					{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN"},
				}},
			}},
		}},
	}}
	if _, err := NormalizeReportFilter(deep); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected the depth budget to be enforced, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsOverWideTree(t *testing.T) {
	children := make([]ReportFilterExpression, 12)
	for index := range children {
		children[index] = ReportFilterExpression{
			Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN",
		}
	}
	if _, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "group", Operator: "and", Children: children,
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected the node budget to be enforced, got %v", err)
	}
}

func TestNormalizeReportFilterRejectsValueOutsideTheFieldVocabulary(t *testing.T) {
	_, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "MADE_UP",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected an unlisted status value to be rejected, got %v", err)
	}
}

func TestReportFilterSQLBindsEveryValueAsAParameter(t *testing.T) {
	expression, err := NormalizeReportFilter(&ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "open"},
			{Kind: "condition", Field: ReportFieldName, Operator: "contains", Value: "'; DROP TABLE x; --"},
		},
	})
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}
	fragment, args, err := ReportFilterSQL(expression, 3)
	if err != nil {
		t.Fatalf("build SQL: %v", err)
	}
	if strings.Contains(fragment, "DROP TABLE") || strings.Contains(fragment, "open") {
		t.Fatalf("filter SQL must carry no literal user value: %q", fragment)
	}
	if len(args) != 2 {
		t.Fatalf("expected both values to be bound parameters, got %d", len(args))
	}
	if args[0] != "OPEN" || args[1] != "'; DROP TABLE x; --" {
		t.Fatalf("unexpected bound arguments: %#v", args)
	}
}

func TestReportFilterSQLRejectsAnUnboundFieldEvenIfNormalisationWasSkipped(t *testing.T) {
	// Defence in depth: SQL generation validates the field itself rather than
	// trusting that NormalizeReportFilter ran first.
	if _, _, err := ReportFilterSQL(&ReportFilterExpression{
		Kind: "condition", Field: "owner_principal_id || password", Operator: "is", Value: "x",
	}, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected SQL generation to reject an unbound field, got %v", err)
	}
}
