package reporting

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeReportRunParameters(t *testing.T) {
	value, err := NormalizeReportRunParameters(ReportRunParameters{
		StartDate:        " 2026-09-01 ",
		EndDate:          "2026-09-30",
		OwnerPrincipalID: " owner-1 ",
	})
	if err != nil {
		t.Fatalf("normalize report run parameters: %v", err)
	}
	if value.StartDate != "2026-09-01" || value.EndDate != "2026-09-30" || value.OwnerPrincipalID != "owner-1" {
		t.Fatalf("unexpected normalized parameters: %#v", value)
	}

	for _, testCase := range []ReportRunParameters{
		{StartDate: "09/01/2026"},
		{EndDate: "2026-13-01"},
		{StartDate: "2026-10-01", EndDate: "2026-09-30"},
	} {
		if _, err := NormalizeReportRunParameters(testCase); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected invalid parameters for %#v, got %v", testCase, err)
		}
	}
}

func TestReportRunParametersSQLUsesBoundArguments(t *testing.T) {
	fragment, args, err := ReportRunParametersSQL(ReportRunParameters{
		StartDate:        "2026-09-01",
		EndDate:          "2026-09-30",
		OwnerPrincipalID: "owner-1",
	}, 6)
	if err != nil {
		t.Fatalf("build run parameter SQL: %v", err)
	}
	for _, expected := range []string{
		"a.created_at >= $6::date",
		"a.created_at < ($7::date + INTERVAL '1 day')",
		"COALESCE(a.owner_principal_id::text,'') = $8",
	} {
		if !strings.Contains(fragment, expected) {
			t.Fatalf("run parameter SQL %q does not contain %q", fragment, expected)
		}
	}
	if len(args) != 3 || args[0] != "2026-09-01" || args[1] != "2026-09-30" || args[2] != "owner-1" {
		t.Fatalf("unexpected run parameter arguments: %#v", args)
	}
	if strings.Contains(fragment, "owner-1") {
		t.Fatalf("owner value leaked into SQL: %q", fragment)
	}
}

func TestCombineReportRunFilterKeepsSetupFilterAfterRunParameters(t *testing.T) {
	filter := &ReportFilterExpression{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "ACTIVE"}
	fragment, args, err := combineReportRunFilter(ReportRunParameters{StartDate: "2026-09-01", OwnerPrincipalID: "owner-1"}, DatasetPrograms, filter, 6)
	if err != nil {
		t.Fatalf("combine report filters: %v", err)
	}
	if !strings.Contains(fragment, "$6::date") || !strings.Contains(fragment, "$7") || !strings.Contains(fragment, "a.status = $8") {
		t.Fatalf("unexpected combined SQL: %q", fragment)
	}
	if len(args) != 3 || args[2] != "ACTIVE" {
		t.Fatalf("unexpected combined arguments: %#v", args)
	}
}
