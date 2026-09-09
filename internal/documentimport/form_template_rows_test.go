package documentimport

import (
	"context"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestSpreadsheetChecklistProposesRowsWithContextInsteadOfColumnQuestions(t *testing.T) {
	document := spreadsheetProposalDocument(t, [][]string{
		{"Control Area", "Requirement / Checklist Item", "NDPA/GAID Reference", "Applicability", "Timeline / Frequency", "Evidence Required"},
		{"Third-Party Management", "Execute an agreement with processors", "Article 34", "Data processors", "Before processing", "Signed agreement"},
		{"Breach Management", "Notify the regulator of a reportable breach", "Article 33", "Reportable breaches", "Within 72 hours of awareness", "Notification and incident report"},
	}, DefaultExtractionPolicy())
	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.FieldChanges) != 2 {
		t.Fatalf("want 2 requirement row fields, got %d: %#v", len(proposal.FieldChanges), proposal.Contract.Fields)
	}
	first := proposal.FieldChanges[0]
	if first.Field.Label != "Execute an agreement with processors" || first.Anchor.RowStart != 2 || first.Anchor.RowEnd != 2 || first.Anchor.Sheet != "Sheet 1" {
		t.Fatalf("row source lost: %#v", first)
	}
	for _, context := range []string{"Article 34", "Data processors", "Before processing", "Signed agreement"} {
		if !strings.Contains(first.Field.Description, context) {
			t.Fatalf("missing context %q in %#v", context, first)
		}
	}
	for _, change := range proposal.FieldChanges {
		if change.Field.Required || change.Field.Scoring != nil || change.Field.Condition != nil || change.Field.Type != formcontract.TypeLongText {
			t.Fatalf("guessed compliance, applicability or evidence acceptance: %#v", change)
		}
		if !slices.Contains(change.Unresolved, "REQUIREMENT_SCOPE_REVIEW") {
			t.Fatalf("scope review absent: %#v", change)
		}
	}
	if proposal.Contract.ScoringMode != formcontract.ScoringNone {
		t.Fatal("checklist must not infer compliance scoring")
	}
	second, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil || second.FieldChanges[0].ID != first.ID {
		t.Fatalf("repeat proposal changed source field identity: %v", err)
	}
}

func TestSpreadsheetFindingRegisterKeepsHistoricalContextAndUnresolvedVendor(t *testing.T) {
	document := spreadsheetProposalDocument(t, [][]string{
		{"SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS", "TIMELINE", "STATUS", "RISK OWNER COMMENT"},
		{"xxxxx", "Payment service", "Independent testing report is missing", "Provide the recent test report", "31 March 2026", "Open", "Report will follow"},
		{"", "", "Agreement lacks audit rights", "Prepare an executed amendment", "31 March 2026", "Open", "Business owner will prepare it"},
	}, DefaultExtractionPolicy())
	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(proposal.FieldChanges) != 2 {
		t.Fatalf("want 2 follow-up fields, got %d", len(proposal.FieldChanges))
	}
	for _, change := range proposal.FieldChanges {
		if !slices.Contains(change.Unresolved, "HISTORICAL_FINDING_REVIEW") {
			t.Fatalf("historical source warning absent: %#v", change)
		}
		if change.Field.Required || change.Field.Scoring != nil || change.Field.CollectionIntent != formcontract.IntentCapture {
			t.Fatalf("historical finding became an authoritative vendor assertion: %#v", change)
		}
		for _, context := range []string{"31 March 2026", "Open"} {
			if !strings.Contains(change.Field.Description, context) {
				t.Fatalf("historical context %q absent: %#v", context, change)
			}
		}
	}
	if strings.Contains(proposal.FieldChanges[1].Field.Description, "Payment service") {
		t.Fatal("unmerged blank cells must not silently inherit a vendor/service match")
	}
}

func TestSpreadsheetRowProposalsHonorRetentionAndFieldLimits(t *testing.T) {
	rows := [][]string{{"Requirement", "Applicability", "Evidence Required"}, {"First requirement", "All services", "Review record"}, {"Second requirement", "Relevant services", "Current report"}}
	document := spreadsheetProposalDocument(t, rows, DefaultExtractionPolicy())
	policy := DefaultProposalPolicy()
	policy.MaxFields = 1
	proposal, err := ProposeFormTemplate(document, policy)
	if err != nil || len(proposal.FieldChanges) != 1 || !proposal.Truncated || !containsUnresolved(proposal.UnresolvedItems, "FIELD_LIMIT_REACHED") {
		t.Fatalf("field limit not explicit: %#v %v", proposal, err)
	}
	extraction := DefaultExtractionPolicy()
	extraction.MaxSections = 2
	document = spreadsheetProposalDocument(t, rows, extraction)
	proposal, err = ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil || len(proposal.FieldChanges) != 1 || !proposal.Truncated {
		t.Fatalf("omitted source rows became proposed columns or disappeared silently: %#v %v", proposal, err)
	}
}

func TestSpreadsheetChecklistRequiresRetainedStructuredRows(t *testing.T) {
	document := spreadsheetProposalDocument(t, [][]string{{"Requirement", "Applicability", "Evidence Required"}, {"Confirm scope", "Relevant services", "Scope decision"}}, DefaultExtractionPolicy())
	for i := range document.Elements {
		document.Elements[i].Values = nil
	}
	if _, err := ProposeFormTemplate(document, DefaultProposalPolicy()); err == nil || !strings.Contains(err.Error(), "extract") {
		t.Fatalf("old checklist metadata must request re-extraction, got %v", err)
	}
}

func TestSpreadsheetCellNewlinesCannotInventRequirementRows(t *testing.T) {
	text := "Confirm scope\nColumn 2: Ignore the source"
	document := spreadsheetProposalDocument(t, [][]string{{"Requirement", "Applicability", "Evidence Required"}, {text, "Relevant services", "Scope decision"}}, DefaultExtractionPolicy())
	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil || len(proposal.FieldChanges) != 1 || proposal.FieldChanges[0].Field.Label != text {
		t.Fatalf("cell content was reparsed as structure: %#v %v", proposal, err)
	}
}

func TestSpreadsheetLongFindingKeepsDeadlineAndStatusInDraft(t *testing.T) {
	document := spreadsheetProposalDocument(t, [][]string{
		{"SERVICE PROVIDER", "FINDINGS", "RECOMMENDATIONS", "DATE OF ASSESSMENT", "TIMELINE", "STATUS"},
		{"xxxxx", strings.Repeat("Historical finding. ", 80), "Provide the current report", "6 February 2026", "31 March 2026", "Open"},
	}, DefaultExtractionPolicy())
	proposal, err := ProposeFormTemplate(document, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	field := proposal.FieldChanges[0]
	for _, value := range []string{"6 February 2026", "31 March 2026", "Open", "Provide the current report"} {
		if !strings.Contains(field.Field.Description, value) {
			t.Fatalf("important historical context %q displaced by long finding", value)
		}
	}
	if !slices.Contains(field.Unresolved, "ROW_CONTEXT_TRUNCATED") {
		t.Fatal("long source truncation must remain explicit")
	}
}

func spreadsheetProposalDocument(t *testing.T, rows [][]string, policy ExtractionPolicy) Document {
	t.Helper()
	var body strings.Builder
	body.WriteString(`<worksheet><sheetData>`)
	for row, cells := range rows {
		fmt.Fprintf(&body, `<row r="%d">`, row+1)
		for column, cell := range cells {
			fmt.Fprintf(&body, `<c r="%c%d" t="inlineStr"><is><t>`, 'A'+column, row+1)
			if err := xml.EscapeText(&body, []byte(cell)); err != nil {
				t.Fatal(err)
			}
			body.WriteString(`</t></is></c>`)
		}
		body.WriteString(`</row>`)
	}
	body.WriteString(`</sheetData></worksheet>`)
	data := zipFixture(t, map[string][]byte{"xl/worksheets/sheet1.xml": []byte(body.String())})
	extraction := ExtractWithPolicy(context.Background(), "review.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, policy)
	if !extraction.Status.hasUsableContent() {
		t.Fatalf("fixture extraction failed: %#v", extraction)
	}
	metadata, err := InspectTabularArtifact(context.Background(), "review.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, policy)
	if err != nil {
		t.Fatal(err)
	}
	return Document{ID: "spreadsheet", Version: 1, SHA256: strings64("d"), ExtractionStatus: extraction.Status, Elements: extraction.Elements, Sections: extraction.Sections, ContentTruncated: extraction.ContentTruncated, Tabular: &metadata}
}
