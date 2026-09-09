package documentimport

import (
	"context"
	"strings"
	"testing"
)

func mergedWorkbook(t *testing.T, rows, merges string) []byte {
	t.Helper()
	return zipFixture(t, map[string][]byte{"xl/worksheets/sheet1.xml": []byte(`<worksheet><sheetData>` + rows + `</sheetData><mergeCells>` + merges + `</mergeCells></worksheet>`)})
}

func TestXLSXVerticalMergesPreserveSectionsAndStructuredValues(t *testing.T) {
	rows := `<row r="1"><c r="A1" t="inlineStr"><is><t>Privacy</t></is></c><c r="B1" t="inlineStr"><is><t>First requirement</t></is></c></row><row r="2"><c r="B2" t="inlineStr"><is><t>Second requirement</t></is></c></row><row r="3"><c r="B3" t="inlineStr"><is><t>Unmerged requirement</t></is></c></row>`
	result := Extract("merged.xlsx", "", mergedWorkbook(t, rows, `<mergeCell ref="A1:A2"/>`))
	if result.Status != ExtractionExtracted || len(result.Sections) != 3 || len(result.Elements) != 3 {
		t.Fatalf("extraction = %#v", result)
	}
	if !strings.Contains(result.Sections[1].Text, "Column 1: Privacy") || result.Elements[1].Values[0][0] != "Privacy" {
		t.Fatalf("merged row lost source context: %#v / %#v", result.Sections[1], result.Elements[1])
	}
	if strings.Contains(result.Sections[2].Text, "Privacy") || result.Elements[2].Values[0][0] != "" {
		t.Fatalf("unmerged blank inherited content: %#v", result.Elements[2])
	}
	if result.ParserVersion != "XLSX_XML_STREAM_V4" {
		t.Fatalf("parser provenance = %q", result.ParserVersion)
	}
}

func TestXLSXMergeRangesRejectInvalidAndOverlappingCoordinates(t *testing.T) {
	for _, refs := range []string{`<mergeCell ref="A2:A1"/>`, `<mergeCell ref="A0:A2"/>`, `<mergeCell ref="A1:A999999999999999999999"/>`, `<mergeCell ref="A1garbage:A2"/>`, `<mergeCell ref="A1:A2:B3"/>`, `<mergeCell ref="A1:A100001"/>`, `<mergeCell ref="IW1:IW2"/>`, `<mergeCell/>`, `<mergeCell ref="A1:A3"/><mergeCell ref="A2:A4"/>`, `<mergeCell ref="A1:A2"/><mergeCell ref="A1:B1"/>`} {
		t.Run(refs, func(t *testing.T) {
			result := Extract("invalid.xlsx", "", mergedWorkbook(t, `<row r="1"><c r="A1"><v>1</v></c></row>`, refs))
			if result.Status != ExtractionFailed || len(result.Sections) != 0 || len(result.Elements) != 0 {
				t.Fatalf("unsafe merge accepted: %#v", result)
			}
		})
	}
}

func TestXLSXMergeRejectsConflictingPopulatedCellsEvenWhenNotRetained(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxSections = 1
	result := ExtractWithPolicy(context.Background(), "conflict.xlsx", "", mergedWorkbook(t, `<row r="1"><c r="A1"><v>1</v></c></row><row r="2"><c r="A2"><v>2</v></c></row>`, `<mergeCell ref="A1:A2"/>`), policy)
	if result.Status != ExtractionFailed {
		t.Fatalf("conflicting merged content accepted: %#v", result)
	}
}

func TestXLSXMergeDoesNotExpandAbsentRowsOrHorizontalCells(t *testing.T) {
	result := Extract("sparse.xlsx", "", mergedWorkbook(t, `<row r="1"><c r="A1"><v>1</v></c><c r="B1"><v>2</v></c></row><row r="100000"><c r="D100000"><v>4</v></c></row>`, `<mergeCell ref="A1:A100000"/><mergeCell ref="B1:C1"/>`))
	if result.Status != ExtractionExtracted || result.SectionsTotal != 2 || len(result.Elements) != 2 {
		t.Fatalf("sparse ranges expanded: %#v", result)
	}
	if result.Elements[1].Values[0][0] != "1" || len(result.Elements[0].Values[0]) != 2 {
		t.Fatalf("range inheritance = %#v", result.Elements)
	}
}

func TestXLSXMergeRangeCountIsBounded(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxCells = 1
	result := ExtractWithPolicy(context.Background(), "many.xlsx", "", mergedWorkbook(t, ``, `<mergeCell ref="A1:A2"/><mergeCell ref="B1:B2"/>`), policy)
	if result.Status != ExtractionFailed {
		t.Fatalf("merge count bypassed budget: %#v", result)
	}
}
