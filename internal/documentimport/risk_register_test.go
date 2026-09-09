package documentimport

import "testing"

func TestRiskRegisterSuggestsContinuationWithoutLosingOriginalCells(t *testing.T) {
	d := spreadsheetProposalDocument(t, [][]string{
		{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS", "RESPONSIBILITY", "TIMELINE", "STATUS"},
		{"1", "Example Ltd", "Payments", "Missing certificate", "Provide certificate", "Alex ", "March 31st 2026", "Open"},
		{"", "", "", "Missing test", "Provide test", "", "31 March 2026", "Open"},
		{"2", "Example Ltd", "Hosting", "Missing plan", "Provide plan", "Vendor", "2026-04-01", "Closed"},
	}, DefaultExtractionPolicy())
	d.SectionsTotal = len(d.Sections)
	p, err := ParseRiskRegister(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Groups) != 2 || len(p.Rows) != 3 {
		t.Fatalf("unexpected population: %#v", p)
	}
	if p.Rows[1].GroupID != p.Rows[0].GroupID || p.Rows[1].Responsibility != "Alex" || len(p.Rows[1].Inherited) == 0 {
		t.Fatalf("continuation: %#v", p.Rows[1])
	}
	if p.Rows[1].Original["RESPONSIBILITY"] != "" || p.Rows[1].Anchor.RowStart != 3 {
		t.Fatal("original source changed")
	}
	if p.Rows[2].Responsibility != "Vendor" || p.Rows[2].RecordedStatus != "Closed" {
		t.Fatal("new assessment inherited prior context")
	}
	if p.Rows[0].SuggestedDueDate != "2026-03-31" {
		t.Fatalf("date: %s", p.Rows[0].SuggestedDueDate)
	}
}

func TestRiskRegisterRejectsIncompleteAndOrphanedSources(t *testing.T) {
	d := spreadsheetProposalDocument(t, [][]string{{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS"}, {"", "", "", "Orphan", "Fix"}}, DefaultExtractionPolicy())
	d.SectionsTotal = len(d.Sections)
	if _, err := ParseRiskRegister(d); err == nil {
		t.Fatal("orphan accepted")
	}
	d.ContentTruncated = true
	if _, err := ParseRiskRegister(d); err == nil {
		t.Fatal("truncated accepted")
	}
}

func TestRiskRegisterRejectsMissingStructuredRow(t *testing.T) {
	d := spreadsheetProposalDocument(t, [][]string{{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RECOMMENDATIONS"}, {"1", "Example Ltd", "Payments", "Finding one", "Fix"}, {"", "", "", "Finding two", "Fix"}}, DefaultExtractionPolicy())
	d.SectionsTotal = len(d.Sections)
	d.Elements = d.Elements[:len(d.Elements)-1]
	if _, err := ParseRiskRegister(d); err == nil {
		t.Fatal("missing structured row accepted")
	}
}
