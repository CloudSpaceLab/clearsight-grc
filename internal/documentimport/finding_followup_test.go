package documentimport

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestXLSXMergedContextStopsAtRangeBoundary(t *testing.T) {
	xml := `<worksheet><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Vendor</t></is></c></row><row r="2"><c r="A2" t="inlineStr"><is><t>First vendor</t></is></c><c r="B2" t="inlineStr"><is><t>Finding one</t></is></c></row><row r="3"><c r="B3" t="inlineStr"><is><t>Finding two</t></is></c></row><row r="4"><c r="B4" t="inlineStr"><is><t>Unassigned finding</t></is></c></row></sheetData><mergeCells><mergeCell ref="A2:A3"/></mergeCells></worksheet>`
	data := zipFixture(t, map[string][]byte{"xl/worksheets/sheet1.xml": []byte(xml)})
	r := ExtractWithPolicy(context.Background(), "register.xlsx", "", data, DefaultExtractionPolicy())
	if r.Status != ExtractionExtracted || len(r.Sections) != 4 {
		t.Fatalf("%+v", r)
	}
	if !strings.Contains(r.Sections[2].Text, "Column 1: First vendor") {
		t.Fatal("merged value not propagated")
	}
	if strings.Contains(r.Sections[3].Text, "First vendor") {
		t.Fatal("merged value leaked past range")
	}
}

func followUpRegisterFixture() Document {
	headers := []string{"S/N", "ASSESSOR", "BUSINESS OWNER", "SERVICE PROVIDER", "SERVICES OFFERED", "FINDINGS", "RISK/ IMPLICATIONS", "Severity", "OVERALL RATING", "RECOMMENDATIONS", "DATE OF ASSESSMENT", "RESPONSIBILITY", "TIMELINE", "STATUS", "RISK OWNER COMMENT"}
	resource := TabularResource{Name: "Sheet 1"}
	for _, name := range headers {
		resource.Fields = append(resource.Fields, TabularField{Name: name})
	}
	doc := Document{ID: "register", SHA256: strings.Repeat("a", 64), Version: 2, ExtractionStatus: ExtractionExtracted, Tabular: &TabularMetadata{Format: TabularXLSX, Resources: []TabularResource{resource}}}
	for i := 0; i < 5; i++ {
		text := fmt.Sprintf("Column 6: Finding %d\nColumn 7: Risk %d\nColumn 8: Medium\nColumn 10: Recommendation %d\nColumn 13: March 31st 2026\nColumn 14: Open\nColumn 15: Historical comment %d", i+1, i+1, i+1, i+1)
		if i == 0 {
			text = "Column 1: 1\nColumn 2: Blessing\nColumn 3: POS Business\nColumn 4: xxxxx\nColumn 5: Moneytor GetPaid application\nColumn 9: Medium\nColumn 11: 13th February 2026\nColumn 12: Hakeem\n" + text
		}
		if i == 3 {
			text = "Column 1: 2\nColumn 2: Joel\nColumn 3: POS Business\nColumn 4: xxxxx\nColumn 5: PTSP\nColumn 9: Medium\nColumn 11: 6th February 2026\nColumn 12: Hakeem\n" + text
		}
		doc.Sections = append(doc.Sections, Section{Sheet: "Sheet 1", RowStart: i + 2, RowEnd: i + 2, Text: text})
	}
	return doc
}

func TestFindingFollowUpPreservesFiveFindingsAndSeparateAssessments(t *testing.T) {
	p, err := ProposeFormTemplate(followUpRegisterFixture(), DefaultProposalPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Contract.Sections) != 5 || len(p.Contract.Fields) != 25 {
		t.Fatalf("unexpected counts: %d sections %d fields", len(p.Contract.Sections), len(p.Contract.Fields))
	}
	groups := map[string]int{}
	for _, c := range p.FieldChanges {
		groups[c.GroupID]++
		if c.Anchor.RowStart < 2 || c.Anchor.RowStart > 6 {
			t.Fatal("missing exact row")
		}
	}
	if len(groups) != 2 {
		t.Fatalf("groups=%v", groups)
	}
	for _, n := range groups {
		if n != 15 && n != 10 {
			t.Fatal(groups)
		}
	}
	for i, s := range p.Contract.Sections {
		date := "13th February 2026"
		assessor := "Blessing"
		if i >= 3 {
			date = "6th February 2026"
			assessor = "Joel"
		}
		if !strings.Contains(s.Help, date) || !strings.Contains(s.Help, assessor) || !strings.Contains(s.Help, fmt.Sprintf("Finding %d", i+1)) {
			t.Fatal(s.Help)
		}
		fields := p.Contract.Fields[i*5 : i*5+5]
		if fields[0].Type != formcontract.TypeLongText || fields[3].Type != formcontract.TypeDate || fields[4].Type != formcontract.TypeFile {
			t.Fatal("wrong controls")
		}
		if got := fields[4].AcceptedFormats; len(got) != 3 || got[0] != "application/pdf" {
			t.Fatalf("supporting evidence must have approved formats: %v", got)
		}
		if !strings.Contains(fields[1].Description, fmt.Sprintf("Recommendation %d", i+1)) {
			t.Fatal("lost recommendation")
		}
		for _, f := range fields {
			if f.Label == "Severity" || f.Label == "ASSESSOR" {
				t.Fatal("bank field made editable")
			}
		}
	}
}

func TestFindingFollowUpFailsClosedForIncompleteOrAmbiguousSource(t *testing.T) {
	for _, kind := range []string{"partial", "orphan", "conflict", "limit", "long"} {
		t.Run(kind, func(t *testing.T) {
			d := followUpRegisterFixture()
			policy := DefaultProposalPolicy()
			switch kind {
			case "partial":
				d.ExtractionStatus = ExtractionPartial
			case "orphan":
				d.Sections = d.Sections[1:]
			case "conflict":
				d.Sections[1].Text += "\nColumn 5: Another service"
			case "limit":
				policy.MaxFields = 10
			case "long":
				d.Sections[0].Text += "\nColumn 10: " + strings.Repeat("x", 1100)
			}
			if _, err := ProposeFormTemplate(d, policy); err == nil {
				t.Fatal("unsafe source produced proposal")
			}
		})
	}
}
