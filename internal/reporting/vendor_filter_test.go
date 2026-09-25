package reporting

import (
	"strings"
	"testing"
)

func TestVendorReportFilterVocabularyIsClosedToVendorFields(t *testing.T) {
	expression, err := NormalizeReportFilterForDataset(DatasetVendors, &ReportFilterExpression{
		Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "restricted"},
			{Kind: "condition", Field: ReportFieldCriticality, Operator: "is", Value: "critical"},
			{Kind: "condition", Field: ReportFieldVendorName, Operator: "contains", Value: "Cloud"},
		},
	})
	if err != nil {
		t.Fatalf("normalize vendor filter: %v", err)
	}
	fragment, args, err := ReportFilterSQLForDataset(DatasetVendors, expression, 6)
	if err != nil {
		t.Fatalf("build vendor filter SQL: %v", err)
	}
	if !strings.Contains(fragment, "a.status = $6") ||
		!strings.Contains(fragment, "a.criticality = $7") ||
		!strings.Contains(fragment, "a.vendor_name ILIKE '%' || $8 || '%'") {
		t.Fatalf("unexpected vendor filter SQL: %s", fragment)
	}
	if len(args) != 3 || args[0] != "RESTRICTED" || args[1] != "CRITICAL" || args[2] != "Cloud" {
		t.Fatalf("unexpected vendor filter args: %#v", args)
	}
}

func TestVendorReportRejectsNonVendorFilterFieldsAndNarrowScopes(t *testing.T) {
	if _, err := NormalizeReportFilterForDataset(DatasetVendors, &ReportFilterExpression{
		Kind: "condition", Field: ReportFieldMatterType, Operator: "is", Value: "EXCEPTION",
	}); err == nil {
		t.Fatal("vendor reports accepted a Matter-only field")
	}
	if !validReportDataset(DatasetVendors) {
		t.Fatal("vendor report dataset is not recognized")
	}
	if !validReportDatasetScope(DatasetVendors, ScopeLegalEntity) {
		t.Fatal("vendor reports must support legal-entity scope")
	}
	if validReportDatasetScope(DatasetVendors, ScopeProgram) || validReportDatasetScope(DatasetVendors, ScopeMatter) {
		t.Fatal("vendor reports must not accept Program or Matter scope")
	}
}
