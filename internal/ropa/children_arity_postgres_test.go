//go:build postgres

package ropa

import (
	"regexp"
	"testing"
)

func TestChildInsertStatementsHaveOneCorrectlyBoundTuplePerRow(t *testing.T) {
	activity := ProcessingActivity{
		TenantID:      "tenant-1",
		LegalEntityID: "entity-1",
		ID:            "activity-1",
		DataCategories: []DataCategory{
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Contact", Sensitivity: "INDIRECT_PERSONAL"},
		},
		Recipients: []Recipient{
			{Recipient: "Processor A", RecipientKind: "EXTERNAL", CountryCode: "US", IsCrossBorder: true, TransferBasis: TransferBasisStandardContractClauses},
			{Recipient: "Processor B", RecipientKind: "EXTERNAL", CountryCode: "GB", IsCrossBorder: true, TransferBasis: TransferBasisAdequacy},
		},
		Systems: []System{
			{SystemName: "Core", SystemKind: "APPLICATION"},
			{SystemName: "Warehouse", SystemKind: "DATABASE"},
		},
		Reviews: []Review{
			{ID: "review-a"},
			{ID: "review-b"},
		},
	}
	statements := buildActivityChildInsertStatements(activity)
	if len(statements) != 8 {
		t.Fatalf("statement count = %d, want one statement for each of eight child rows", len(statements))
	}
	placeholder := regexp.MustCompile(`\$[0-9]+`)
	for index, statement := range statements {
		placeholders := placeholder.FindAllString(statement.query, -1)
		if len(placeholders) != len(statement.values) {
			t.Errorf("statement %d (%s) has %d placeholders for %d bound values", index, statement.name, len(placeholders), len(statement.values))
		}
		seen := make(map[string]struct{}, len(placeholders))
		for _, value := range placeholders {
			seen[value] = struct{}{}
		}
		if len(seen) != len(placeholders) {
			t.Errorf("statement %d repeats a placeholder: %v", index, placeholders)
		}
	}
}
