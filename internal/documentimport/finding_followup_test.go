package documentimport

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestFindingFollowUpCompleteXLSXWithBlankFormattedRows(t *testing.T) {
	rows := `<row r="1"><c r="A1" t="inlineStr"><is><t>S/N</t></is></c><c r="B1" t="inlineStr"><is><t>SERVICE PROVIDER</t></is></c><c r="C1" t="inlineStr"><is><t>SERVICES OFFERED</t></is></c><c r="D1" t="inlineStr"><is><t>DATE OF ASSESSMENT</t></is></c><c r="E1" t="inlineStr"><is><t>FINDINGS</t></is></c><c r="F1" t="inlineStr"><is><t>RECOMMENDATIONS</t></is></c></row>
	<row r="2"><c r="A2"><v>1</v></c><c r="B2" t="inlineStr"><is><t>Sample vendor</t></is></c><c r="C2" t="inlineStr"><is><t>Payments</t></is></c><c r="D2" t="inlineStr"><is><t>2026-02-06</t></is></c><c r="E2" t="inlineStr"><is><t>First finding</t></is></c><c r="F2" t="inlineStr"><is><t>Provide report</t></is></c></row>
	<row r="3"><c r="A3" s="1"/><c r="E3" t="inlineStr"><is><t></t></is></c></row>
	<row r="4"><c r="A4"><v>1</v></c><c r="E4" t="inlineStr"><is><t>Second finding</t></is></c><c r="F4" t="inlineStr"><is><t>Confirm scope</t></is></c></row>
	<row r="5"><c r="A5" s="1"/></row>`
	data := mergedWorkbook(t, rows, "")
	extraction := Extract("blank-rows.xlsx", "", data)
	metadata, err := InspectTabularArtifact(context.Background(), "blank-rows.xlsx", "", data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if extraction.Status != ExtractionExtracted || extraction.SectionsTotal != 3 || metadata.RowsTotal != 4 {
		t.Fatalf("fixture does not reproduce different row populations: %#v %#v", extraction, metadata)
	}
	d := Document{ID: "blank-rows", Version: 1, SHA256: strings64("d"), ExtractionStatus: extraction.Status, ParserVersion: extraction.ParserVersion, Elements: extraction.Elements, Sections: extraction.Sections, SectionsTotal: extraction.SectionsTotal, SectionsOmitted: extraction.SectionsOmitted, ContentTruncated: extraction.ContentTruncated, Tabular: &metadata}
	groups, err := FindingFollowUpAssessments(d)
	if err != nil || len(groups) != 1 || groups[0].FindingCount != 2 {
		t.Fatalf("complete register rejected due to blank rows: %#v %v", groups, err)
	}
	p, err := ProposeFindingFollowUp(d, groups[0].ID, DefaultProposalPolicy())
	if err != nil || len(p.FieldChanges) != 10 || p.FieldChanges[5].Anchor.RowStart != 4 {
		t.Fatalf("blank source row became a finding or displaced its anchor: %#v %v", p, err)
	}
	for _, testCase := range []struct {
		name   string
		mutate func(*Document)
	}{
		{"missing finding cells", func(d *Document) { d.Elements = d.Elements[:2] }},
		{"missing entire finding", func(d *Document) { d.Elements = d.Elements[:2]; d.Sections = d.Sections[:2] }},
		{"omitted source content", func(d *Document) { d.SectionsOmitted = 1 }},
		{"no completeness receipt", func(d *Document) { d.SectionsTotal = 0 }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			incomplete := d
			testCase.mutate(&incomplete)
			if _, err := FindingFollowUpAssessments(incomplete); err == nil {
				t.Fatal("incomplete finding extraction became eligible")
			}
		})
	}
}

func findingDocument(t *testing.T, rows ...[]string) Document {
	t.Helper()
	return spreadsheetProposalDocument(t, append([][]string{{"S/N", "SERVICE PROVIDER", "SERVICES OFFERED", "DATE OF ASSESSMENT", "FINDINGS", "RECOMMENDATIONS", "RESPONSIBILITY", "STATUS", "TIMELINE"}}, rows...), DefaultExtractionPolicy())
}

func followUpForFirst(t *testing.T, d Document) FormTemplateProposal {
	t.Helper()
	groups, err := FindingFollowUpAssessments(d)
	if err != nil || len(groups) == 0 {
		t.Fatalf("assessments: %#v %v", groups, err)
	}
	p, err := ProposeFindingFollowUp(d, groups[0].ID, DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindingFollowUpStructuredTextAndRowResponsibility(t *testing.T) {
	d := findingDocument(t,
		[]string{"1", "Sample vendor", "Payments", "6 February 2026", "Finding 1\nColumn 10: quoted example inside the finding", "Provide report", "First owner", "Open", "31 March 2026"},
		[]string{"1", "", "", "", "Finding 2", "Amend agreement", "Second owner", "Open", "31 March 2026"},
		[]string{"1", "", "", "", "Finding 3", "Confirm scope", "", "Open", ""},
	)
	p := followUpForFirst(t, d)
	if p.Provenance.ProposalVersion != "FINDING_FOLLOW_UP_V2" || len(p.FieldChanges) != 15 || p.Contract.ScoringMode != formcontract.ScoringNone {
		t.Fatalf("unexpected proposal: %#v", p)
	}
	if !strings.Contains(p.Contract.Sections[0].Help, "Column 10: quoted example inside the finding") {
		t.Fatal("finding text was parsed as columns")
	}
	if !strings.Contains(p.Contract.Fields[7].Description, "Second owner") || strings.Contains(p.Contract.Fields[7].Description, "First owner") {
		t.Fatal("second finding lost its own responsibility")
	}
	if !strings.Contains(p.Contract.Fields[12].Description, "not recorded") || strings.Contains(p.Contract.Fields[12].Description, "First owner") {
		t.Fatal("blank responsibility was inherited")
	}
	for i, change := range p.FieldChanges {
		wantType := []formcontract.Type{formcontract.TypeLongText, formcontract.TypeLongText, formcontract.TypeShortText, formcontract.TypeDate, formcontract.TypeFile}[i%5]
		if change.Anchor.RowStart != 2+i/5 || change.Anchor.RowEnd != 2+i/5 || change.Field.Required != (i%5 < 3) || change.Field.Type != wantType || change.Field.Scoring != nil {
			t.Fatalf("incorrect field %d: %#v", i, change)
		}
	}
	if !reflect.DeepEqual(p.Contract.Fields[4].AcceptedFormats, []string{"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}) {
		t.Fatalf("formats: %#v", p.Contract.Fields[4].AcceptedFormats)
	}
	if !reflect.DeepEqual(p, followUpForFirst(t, d)) {
		t.Fatal("proposal IDs or context changed on retry")
	}
}

func TestFindingFollowUpNumberedHeaderAndBlankBoundary(t *testing.T) {
	rows := [][]string{
		{"1", "First vendor", "Payments", "2026-02-06", "First finding", "Provide report"},
		{"2", "Second vendor", "Hosting", "2026-03-01", "", ""},
		{"", "", "", "", "Second finding", "Provide certificate"},
	}
	d := findingDocument(t, rows...)
	if _, err := FindingFollowUpAssessments(d); err == nil || !strings.Contains(err.Error(), "S/N") {
		t.Fatalf("blank finding silently grouped: %v", err)
	}
	ordinary, err := ProposeFormTemplate(d, DefaultProposalPolicy())
	if err != nil || ordinary.Provenance.ProposalVersion != formProposalVersion || len(ordinary.FieldChanges) != 2 || len(ordinary.Provenance.FindingAssessments) != 0 {
		t.Fatalf("ordinary proposal changed: %#v %v", ordinary, err)
	}
	if !containsUnresolved(ordinary.UnresolvedItems, "FINDING_FOLLOW_UP_UNAVAILABLE") {
		t.Fatal("ambiguous follow-up has no recovery warning")
	}
	rows[2][0] = "2"
	d = findingDocument(t, rows...)
	groups, err := FindingFollowUpAssessments(d)
	if err != nil || len(groups) != 2 || groups[1].FindingCount != 1 || groups[1].RowStart != 3 || groups[1].RowEnd != 4 || !strings.Contains(groups[1].Label, "Second vendor") {
		t.Fatalf("numbered header lost: %#v %v", groups, err)
	}
	p, err := ProposeFindingFollowUp(d, groups[1].ID, DefaultProposalPolicy())
	if err != nil || len(p.FieldChanges) != 5 || p.FieldChanges[0].Anchor.RowStart != 4 || !strings.Contains(p.Contract.Sections[0].Help, "Hosting") || strings.Contains(p.Contract.Sections[0].Help, "First vendor") {
		t.Fatalf("wrong vendor context: %#v %v", p, err)
	}
}

func TestFindingFollowUpBoundsAssessmentChoicesAndRequiresStructuredHeaders(t *testing.T) {
	d := findingDocument(t, []string{"1", "Sample vendor", "Payments", "2026-02-06", "Finding", "Provide report"})
	d.Elements = nil
	if _, err := FindingFollowUpAssessments(d); err == nil {
		t.Fatal("metadata-only register did not request extraction")
	}
	rows := [][]string{}
	for i := 0; i < 201; i++ {
		rows = append(rows, []string{fmt.Sprint(i + 1), "Sample vendor", "Payments", "2026-02-06", "Finding", "Provide report"})
	}
	d = findingDocument(t, rows...)
	if _, err := FindingFollowUpAssessments(d); err == nil {
		t.Fatal("unbounded assessment choices")
	}
}

func TestFindingFollowUpRejectsAmbiguousAndIncompleteSources(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Document)
	}{
		{"blank serial", func(d *Document) { d.Elements[2].Values[0][0] = "" }},
		{"contradictory vendor", func(d *Document) { d.Elements[2].Values[0][1] = "Different vendor" }},
		{"contradictory service", func(d *Document) { d.Elements[2].Values[0][2] = "Different service" }},
		{"contradictory date", func(d *Document) { d.Elements[2].Values[0][3] = "2026-09-01" }},
		{"partial", func(d *Document) { d.ExtractionStatus = ExtractionPartial }},
		{"truncated", func(d *Document) { d.ContentTruncated = true }},
		{"omitted", func(d *Document) { d.SectionsOmitted = 1 }},
		{"missing values", func(d *Document) { d.Elements[2].Values = nil }},
		{"missing retained row", func(d *Document) { d.Elements = d.Elements[:2] }},
		{"missing source quote", func(d *Document) { d.Sections = d.Sections[:2] }},
		{"malformed anchor", func(d *Document) { d.Elements[2].Anchor.RowEnd++ }},
		{"duplicate row", func(d *Document) { d.Elements[2].Anchor = d.Elements[1].Anchor }},
		{"duplicate header", func(d *Document) { d.Elements[0].Values[0] = append(d.Elements[0].Values[0], "S/N") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := findingDocument(t, []string{"1", "Sample vendor", "Payments", "2026-02-06", "Finding 1", "Provide report"}, []string{"1", "", "", "", "Finding 2", "Provide certificate"})
			tc.mutate(&d)
			if _, err := FindingFollowUpAssessments(d); err == nil {
				t.Fatal("unsafe source became eligible")
			}
		})
	}
}

func TestFindingFollowUpSelectedAssessmentLimitsAndLongContext(t *testing.T) {
	d := findingDocument(t,
		[]string{"1", "First vendor", "Payments", "2026-02-06", strings.Repeat("Long finding. ", 200), "Provide report", "Owner", "Open", "31 March 2026"},
		[]string{"2", "Second vendor", "Hosting", "2026-03-01", "Finding 2", "Provide certificate"},
		[]string{"2", "", "", "", "Finding 3", "Provide certificate"},
	)
	groups, err := FindingFollowUpAssessments(d)
	if err != nil {
		t.Fatal(err)
	}
	policy := ProposalPolicy{MaxFields: 5, MaxSections: 1}
	p, err := ProposeFindingFollowUp(d, groups[0].ID, policy)
	if err != nil || len(p.FieldChanges) != 5 || !containsUnresolved(p.UnresolvedItems, "ROW_CONTEXT_TRUNCATED") {
		t.Fatalf("selected group failed for unrelated group or long context: %#v %v", p, err)
	}
	if !strings.Contains(p.Contract.Sections[0].Help, "31 March 2026") {
		t.Fatal("long finding displaced historical deadline")
	}
	policy.MaxUnresolved = 1
	p, err = ProposeFindingFollowUp(d, groups[0].ID, policy)
	if err != nil || len(p.FieldChanges) != 5 || len(p.UnresolvedItems) > 1 || !containsUnresolved(p.UnresolvedItems, "ROW_CONTEXT_TRUNCATED") {
		t.Fatalf("long context aborted a complete assessment at warning bound: %#v %v", p, err)
	}
	if _, err := ProposeFindingFollowUp(d, groups[1].ID, policy); err == nil {
		t.Fatal("partial assessment produced at bounds")
	}
	policy.MaxFields = 10
	if _, err := ProposeFindingFollowUp(d, groups[1].ID, policy); err == nil {
		t.Fatal("section bound produced partial assessment")
	}
	if _, err := ProposeFindingFollowUp(d, "unknown", DefaultProposalPolicy()); err == nil {
		t.Fatal("unknown group accepted")
	}
	ordinary, err := ProposeFormTemplate(d, DefaultProposalPolicy())
	if err != nil || len(ordinary.FieldChanges) != 3 || len(ordinary.Provenance.FindingAssessments) != 2 || ordinary.Provenance.ProposalVersion != formProposalVersion {
		t.Fatalf("default changed: %#v %v", ordinary, err)
	}
}
