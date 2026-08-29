package documentimport

import (
	"strings"
	"testing"
)

func TestExtractionNeverReturnsEmptySuccessAfterParserFailure(t *testing.T) {
	result := Extract("broken.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("not a zip"))
	if result.Status == ExtractionExtracted && len(result.Elements) == 0 {
		t.Fatalf("empty success: %#v", result)
	}
}

func TestCollectorMarksRetainedOutputTruncated(t *testing.T) {
	result := ExtractWithPolicy(t.Context(), "large.txt", "text/plain", []byte(strings.Repeat("section\n\n", 100)), ExtractionPolicy{MaxSections: 2})
	if result.Status != ExtractionTruncated || !result.ContentTruncated || result.SectionsOmitted == 0 {
		t.Fatalf("missing truncation truth: %#v", result)
	}
}

func TestElementsPreserveSectionStructureAndAnchors(t *testing.T) {
	result := Extract("policy.md", "text/markdown", []byte("# Reporting duty\nThe bank must file monthly.\n\nSupporting paragraph."))
	if result.Status != ExtractionExtracted {
		t.Fatalf("unexpected extraction status: %#v", result)
	}
	if result.ParserVersion != "PLAIN_TEXT_V2" || result.AdapterVersion != extractionElementAdapterVersion {
		t.Fatalf("missing version lineage: %#v", result)
	}
	if len(result.Elements) < 2 {
		t.Fatalf("expected structured elements, got %#v", result.Elements)
	}
	if result.Elements[0].Kind != ElementHeading || result.Elements[0].Text != "Reporting duty" {
		t.Fatalf("heading was not preserved: %#v", result.Elements)
	}
}

func TestElementTableAnchorPreservesSheetAndRow(t *testing.T) {
	sections := []Section{{ID: "section-1", Title: "Sheet 1 row 7", Text: "Column 1: value", Sheet: "Sheet 1", RowStart: 7, RowEnd: 7}}
	elements := elementsFromSections(sections)
	if len(elements) != 1 || elements[0].Kind != ElementTable {
		t.Fatalf("expected table element, got %#v", elements)
	}
	if elements[0].Anchor.Sheet != "Sheet 1" || elements[0].Anchor.RowStart != 7 || elements[0].Anchor.RowEnd != 7 {
		t.Fatalf("row anchor was not preserved: %#v", elements[0].Anchor)
	}
}

func TestDOCXRecoveredFormControlDropsLegacyPartialDegradation(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:sdt><w:sdtPr><w:alias w:val="Country"/></w:sdtPr><w:sdtContent><w:p><w:r><w:t>Nigeria</w:t></w:r></w:p></w:sdtContent></w:sdt></w:body></w:document>`),
	})

	result := Extract("questionnaire.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data)
	if result.Status != ExtractionExtracted {
		t.Fatalf("expected recovered control extraction, got %#v", result)
	}
	for _, degradation := range result.Degradations {
		if degradation.Code == "DOCX_FORM_CONTROLS_NOT_EXTRACTED" {
			t.Fatalf("legacy form-control degradation survived Task 14: %#v", result.Degradations)
		}
	}
	for _, element := range result.Elements {
		if element.Kind == ElementFormControl && element.Control != nil && element.Control.Label == "Country" && element.Text == "Nigeria" {
			return
		}
	}
	t.Fatalf("recovered form control missing: %#v", result.Elements)
}
