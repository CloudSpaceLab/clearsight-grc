package reporting

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	for _, value := range []string{"DRAFT", "PENDING_REVIEW", "REVIEWED", "ACTIVE", "RETIRED"} {
		if !validDefinitionStatus(DefinitionStatus(value)) {
			t.Fatalf("status %q should be valid", value)
		}
	}
	if validDefinitionStatus("APPROVED") || validDefinitionStatus("") {
		t.Fatal("unlisted statuses must be rejected")
	}
}

func TestReportRunEnvelopeUsesTheApprovedBounds(t *testing.T) {
	if ReportRunPageSize != 100 {
		t.Fatalf("report page size = %d, want 100", ReportRunPageSize)
	}
	if MaxReportRunRows != 10_000 {
		t.Fatalf("report row ceiling = %d, want 10000", MaxReportRunRows)
	}
	if MaxReportRunBytes != 32<<20 {
		t.Fatalf("report byte ceiling = %d, want 32 MiB", MaxReportRunBytes)
	}
}

func TestReportModelIncludesAllDatasetsAndSeparateReviewProvenance(t *testing.T) {
	datasets := []ReportDataset{
		DatasetProcessingActivities,
		DatasetProcessingActivityExceptions,
		DatasetPrograms,
		DatasetMatterExceptions,
	}
	seen := make(map[ReportDataset]struct{}, len(datasets))
	for _, dataset := range datasets {
		if dataset == "" {
			t.Fatal("report dataset must be named")
		}
		if _, exists := seen[dataset]; exists {
			t.Fatalf("report dataset %q is duplicated", dataset)
		}
		seen[dataset] = struct{}{}
	}
	definition := ReportDefinition{Status: DefinitionReviewed, ReviewerID: "reviewer-1", ReviewerNote: "Checked scope and filter."}
	if definition.ReviewerID == "" || definition.ReviewerNote == "" {
		t.Fatal("reviewed definition must retain the reviewer and note")
	}
	revision := ReportDefinitionRevision{ReviewedBy: "reviewer-1"}
	if revision.ReviewedBy == "" {
		t.Fatal("revision must retain separate review provenance")
	}
}

func TestRunAndManifestCarryTheSourceBoundary(t *testing.T) {
	captured := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	boundary := SourceBoundary{
		CapturedAt:         captured,
		ProjectionVersion:  "ropa-register.v1",
		SourceHighWater:    map[string]time.Time{"processing_activities": captured.Add(-time.Minute)},
		Population:         42,
		PopulationComplete: true,
	}
	run := ReportRun{SourceBoundary: boundary, DefinitionChecksum: strings.Repeat("a", 64)}
	if !run.SourceBoundary.CapturedAt.Equal(captured) || run.SourceBoundary.ProjectionVersion == "" || len(run.SourceBoundary.SourceHighWater) != 1 {
		t.Fatal("queued run did not retain the complete source boundary")
	}
	manifest := Manifest{Source: boundary, RetentionUntil: captured.Add(ReportRunRetention)}
	if !manifest.Source.PopulationComplete || manifest.Source.Population != 42 || manifest.RetentionUntil.IsZero() {
		t.Fatal("manifest did not retain source completeness and retention")
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
