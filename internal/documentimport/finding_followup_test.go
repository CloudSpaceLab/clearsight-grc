package documentimport

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

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
