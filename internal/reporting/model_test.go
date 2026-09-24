package reporting

import (
	"encoding/json"
	"testing"
)

func TestDefinitionChecksumIsStableAndCoversEveryGovernedField(t *testing.T) {
	base := ReportDefinition{
		TenantID: "t", LegalEntityID: "e", Code: "ROPA-EXCEPTIONS",
		Dataset: DatasetProcessingActivityExceptions, ScopeKind: ScopeLegalEntity,
		Format: FormatCSV, CurrentVersion: 1, MakerID: "maker-1",
		Filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN"},
		}},
	}
	first := base.Checksum()
	if len(first) != 64 {
		t.Fatalf("expected a SHA-256 hex checksum, got %q", first)
	}
	if first != base.Checksum() {
		t.Fatal("checksum must be stable for identical content")
	}
	for _, mutate := range []func(*ReportDefinition){
		func(d *ReportDefinition) { d.TenantID = "other-tenant" },
		func(d *ReportDefinition) { d.LegalEntityID = "other-entity" },
		func(d *ReportDefinition) { d.Code = "ROPA-OTHER" },
		func(d *ReportDefinition) { d.Name = "A different report" },
		func(d *ReportDefinition) { d.Description = "A different description" },
		func(d *ReportDefinition) { d.Dataset = DatasetProcessingActivities },
		func(d *ReportDefinition) { d.ScopeKind = ScopeProgram },
		func(d *ReportDefinition) { d.ScopeRef = "program-1" },
		func(d *ReportDefinition) { d.Format = FormatNDJSON },
		func(d *ReportDefinition) { d.Filter.Children[0].Value = "CLOSED" },
		func(d *ReportDefinition) { d.MakerID = "maker-2" },
		func(d *ReportDefinition) { d.CurrentVersion = 2 },
	} {
		changed := base
		changed.Filter = cloneFilter(base.Filter)
		mutate(&changed)
		if changed.Checksum() == first {
			t.Fatalf("checksum did not change after mutating a governed field")
		}
	}
}

func TestDefinitionChecksumDoesNotBindLifecycleOrReceiptState(t *testing.T) {
	base := ReportDefinition{
		TenantID: "t", LegalEntityID: "e", Code: "ROPA-EXCEPTIONS",
		Dataset: DatasetProcessingActivityExceptions, ScopeKind: ScopeLegalEntity,
		Format: FormatCSV, CurrentVersion: 1, MakerID: "maker-1",
		Filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{
			{Kind: "condition", Field: ReportFieldStatus, Operator: "is", Value: "OPEN"},
		}},
	}
	first := base.Checksum()
	for _, mutate := range []func(*ReportDefinition){
		func(d *ReportDefinition) { d.ID = "definition-2" },
		func(d *ReportDefinition) { d.Status = DefinitionActive },
		func(d *ReportDefinition) { d.StoredChecksum = "different stored checksum" },
		func(d *ReportDefinition) { d.CheckerID = "checker-1" },
		func(d *ReportDefinition) { d.Version = 9 },
	} {
		changed := base
		changed.Filter = cloneFilter(base.Filter)
		mutate(&changed)
		if changed.Checksum() != first {
			t.Fatal("checksum must not change for lifecycle or receipt-only state")
		}
	}
}

func TestDefinitionStatusVocabularyIsClosed(t *testing.T) {
	for _, value := range []string{"DRAFT", "PENDING_REVIEW", "ACTIVE", "RETIRED"} {
		if !validDefinitionStatus(DefinitionStatus(value)) {
			t.Fatalf("status %q should be valid", value)
		}
	}
	if validDefinitionStatus("APPROVED") || validDefinitionStatus("") {
		t.Fatal("unlisted statuses must be rejected")
	}
}

// cloneFilter deep-copies so a mutation in the checksum table cannot leak into
// the next case and make the test pass for the wrong reason.
func cloneFilter(expression *ReportFilterExpression) *ReportFilterExpression {
	if expression == nil {
		return nil
	}
	raw, err := json.Marshal(expression)
	if err != nil {
		panic(err)
	}
	var clone ReportFilterExpression
	if err := json.Unmarshal(raw, &clone); err != nil {
		panic(err)
	}
	return &clone
}
