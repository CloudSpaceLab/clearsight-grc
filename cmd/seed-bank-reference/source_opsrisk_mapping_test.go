//go:build postgres

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func syntheticBIARecord(row int) sourceRecord {
	labels := []string{
		"S/N", "Group/Division", "Business Function", "Business Process Name",
		"Impact on bank customers", "Column1", "Impact on finance", "Column2",
		"Impact on staff", "Column3", "Impact on regulatoror legal exposure", "Column4",
		"Impact on other Dependent processes", "Column5", "Impact on Reputation image", "Column6",
		"Average Rating", "Outcome of activity being delivered", "Applications used by process",
		"Equipment/ other Resources used by process", "Which Business Units are you dependent on",
		"Which Business Units are dependent on you", "Who are your Key vendors or External dependencies",
		"Electronic Vital Records", "MBCO", "MAO", "RTO", "RPO",
	}
	record := sourceRecord{
		Key: fmt.Sprintf("bia-row-%d", row), Title: fmt.Sprintf("Row %d", row),
		SourceRange: fmt.Sprintf("BIA Master  !A%d:AB%d", row, row),
	}
	for col, label := range labels {
		value := "0"
		if col == 3 {
			value = fmt.Sprintf("Synthetic continuity process %d", row)
		}
		record.Fields = append(record.Fields, sourceRecordField{Label: label, Value: value, SourceCell: fmt.Sprintf("%s%d", sourceColumnName(col), row)})
	}
	return record
}

func TestBIARatingPresentationUsesValidatedAdjacentImpactDimension(t *testing.T) {
	group := sourceRecordGroup{SourceFile: "Business_Impact_Analysis.xlsx", SourceSheet: "BIA Master  ", PresentationVersion: 2}
	record := syntheticBIARecord(2)
	for pair := 0; pair < 6; pair++ {
		index := 5 + 2*pair
		got, ok := sourceBIARatingLabel(group, record, index)
		if !ok || got != biaImpactHeadings[pair]+" — rating" {
			t.Fatalf("BIA impact %d title = %q, %v", pair+1, got, ok)
		}
		description := sourceV2FieldDescription(group, record, index)
		if !strings.Contains(description, fmt.Sprintf("Column%d", pair+1)) ||
			!strings.Contains(description, record.Fields[index].SourceCell) {
			t.Fatalf("lost original label/cell provenance: %q", description)
		}
	}
	if title := sourceRecordDisplayTitle(group, record); title != "Synthetic continuity process 2" {
		t.Fatalf("BIA title used division instead of process: %q", title)
	}
	text := sourceRecordTextForGroupV2(group, record)
	if !strings.Contains(text, "Impact on bank customers — rating (Column1, F2): 0") {
		t.Fatalf("BIA typed dimension/coordinate lost: %q", text)
	}
	record.Fields[5].Value = ""
	if text := sourceRecordTextForGroupV2(group, record); !strings.Contains(text, "Impact on bank customers — rating (Column1, F2): Not recorded in source") {
		t.Fatalf("blank rating was not preserved as unanswered: %q", text)
	}
	if record.Fields[5].Label != "Column1" {
		t.Fatal("source label was overwritten")
	}
	record = syntheticBIARecord(2)
	record.Fields[5].SourceCell = "H2"
	if _, ok := sourceBIARatingLabel(group, record, 5); ok {
		t.Fatal("non-adjacent source cells were semantically paired")
	}
	record = syntheticBIARecord(2)
	record.Fields[4].Label = "Unrelated field"
	if _, ok := sourceBIARatingLabel(group, record, 5); ok {
		t.Fatal("different source impact heading was guessed")
	}
	group.SourceFile = "Branch KRI .xlsx"
	if _, ok := sourceBIARatingLabel(group, syntheticBIARecord(2), 5); ok {
		t.Fatal("BIA-specific generic column mapping escaped its source workbook")
	}
}

func syntheticBranchKRI() sourceRecordGroup {
	group := sourceRecordGroup{
		Key: "ops-branch-kri", ProgramCode: "OPS-RISK", SourceFile: "Branch KRI .xlsx",
		SourceSheet: "Branch KRI November '25", PresentationVersion: 2, ResponsePerRecord: true,
	}
	for index := 0; index < 31; index++ {
		row := index + 2
		record := sourceRecord{
			Key: fmt.Sprintf("ops-branch-kri-r%d", row), Title: fmt.Sprintf("Row %d", row),
			SourceRange: fmt.Sprintf("Branch KRI November '25!A%d:BG%d", row, row),
		}
		for col := 0; col < 59; col++ {
			label, value := fmt.Sprintf("Source metric %d", col), "0"
			switch col {
			case 0:
				label, value = "Directorate", "Synthetic directorate"
			case 1:
				label, value = "Region", "Synthetic region"
			case 2:
				label, value = "Branch", fmt.Sprintf("Synthetic branch %d", row)
			}
			if col >= 55 || (col == 3 && index < 4) {
				value = ""
			}
			record.Fields = append(record.Fields, sourceRecordField{Label: label, Value: value, SourceCell: fmt.Sprintf("%s%d", sourceColumnName(col), row)})
		}
		group.Records = append(group.Records, record)
	}
	return group
}

func TestBranchV2Requires31UniqueComplete59ColumnResponsesAnd128Blanks(t *testing.T) {
	group := syntheticBranchKRI()
	if err := sourceValidateV2Group(group); err != nil {
		t.Fatalf("valid 31x59 V2 source rejected: %v", err)
	}
	answers := registerAnswers(formcontract.TextAnswer("SHA-256: synthetic"), group.Records[8])
	if _, has := answers["r0_f55"]; has {
		t.Fatal("unanswered source cell became an answer")
	}
	if answer := answers["r0_f3"]; answer.Text == nil || *answer.Text != "0" {
		t.Fatalf("recorded zero was not preserved: %+v", answer)
	}
	first := sourceV2RegisterContext(formcontract.TextAnswer("Source digest"), group.Records[0])
	last := sourceV2RegisterContext(formcontract.TextAnswer("Source digest"), group.Records[30])
	if first.Text == nil || last.Text == nil || !strings.Contains(*first.Text, "!A2:BG2") ||
		!strings.Contains(*last.Text, "!A32:BG32") || *first.Text == *last.Text {
		t.Fatalf("per-response coordinates collapsed to first source row: %v %v", first.Text, last.Text)
	}
	record := group.Records[6]
	group.Records[6].Fields[3].Value = ""
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("129 unanswered cells passed as the original 128")
	}
	group.Records[6] = record
	group.Records[10].Fields[20].SourceCell = "Q12"
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("wrong source cell coordinate passed")
	}
	group = syntheticBranchKRI()
	group.Records[10].Fields[20].Label = "Changed metric"
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("schema drift between branch records passed")
	}
	group = syntheticBranchKRI()
	group.Records = group.Records[:30]
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("partial branch group passed")
	}
}

func TestBIAMasterV2RequiresEveryOriginalRowAndAllSixPairedRatings(t *testing.T) {
	group := sourceRecordGroup{
		Key: "ops-bia-master", ProgramCode: "OPS-RISK", SourceFile: "Business_Impact_Analysis.xlsx",
		SourceSheet: "BIA Master  ", PresentationVersion: 2, ResponsePerRecord: true,
	}
	for row := 2; row <= 603; row++ {
		group.Records = append(group.Records, syntheticBIARecord(row))
	}
	if err := sourceValidateV2Group(group); err != nil {
		t.Fatalf("valid 602-row BIA source rejected: %v", err)
	}
	group.Records[100].Fields[7].Label = "Unknown score"
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("misidentified second rating column passed")
	}
	group = group
	group.Records = group.Records[:601]
	if err := sourceValidateV2Group(group); err == nil {
		t.Fatal("partial BIA master was silently imported")
	}
}

func TestHeadOfficeV2NeverImportsReportAndCalculationAsFreshIndicators(t *testing.T) {
	group := sourceRecordGroup{
		Key: "ops-head-kri", ProgramCode: "OPS-RISK", SourceFile: "HEAD OFFICE KRI MONTHLY.xlsx",
		SourceSheet: "Sheet1", PresentationVersion: 2,
		Records: []sourceRecord{{Key: "metric-row-3", Title: "Metric", SourceRange: "Sheet1!A3:I3",
			Fields: []sourceRecordField{{Label: "RISK METRICS", Value: "Synthetic service metric", SourceCell: "C3"}}}},
	}
	if err := sourceValidateV2Group(group); err != nil {
		t.Fatalf("primary worksheet must remain eligible for reviewed V2 presentation: %v", err)
	}
	for _, view := range []string{"Board report", "calculation"} {
		group.SourceSheet = view
		if err := sourceValidateV2Group(group); err == nil {
			t.Fatalf("duplicate current KRI import allowed from %s", view)
		}
		group.PresentationVersion = 0
		if err := sourceValidateV2Group(group); err != nil {
			t.Fatalf("historical V1 %s was rewritten/rejected: %v", view, err)
		}
		group.PresentationVersion = 2
	}
}
