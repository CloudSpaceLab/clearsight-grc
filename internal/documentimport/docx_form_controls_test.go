package documentimport

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestDOCXFormControlsKeepQuestionAndOptions(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Vendor questionnaire</w:t></w:r></w:p><w:p><w:r><w:t>Select a country: </w:t></w:r><w:sdt><w:sdtPr><w:alias w:val="Country"/><w:dropDownList><w:listItem w:displayText="Nigeria" w:value="NG"/><w:listItem w:displayText="Ghana" w:value="GH"/></w:dropDownList></w:sdtPr><w:sdtContent><w:r><w:t>Nigeria</w:t></w:r></w:sdtContent></w:sdt></w:p></w:body></w:document>`),
	})

	result, err := extractDOCX(context.Background(), data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ExtractionExtracted || result.AdapterVersion != docxElementAdapterVersion {
		t.Fatalf("unexpected extraction outcome: %#v", result)
	}
	for _, element := range result.Elements {
		if element.Control != nil && element.Control.Kind == "DROPDOWN" && element.Control.Label == "Country" && slices.Equal(element.Control.Options, []string{"Nigeria", "Ghana"}) && element.Text == "Nigeria" {
			return
		}
	}
	t.Fatalf("dropdown control was not preserved: %#v", result.Elements)
}

func TestDOCXFormControlsKeepCheckboxAndTextInput(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:sdt><w:sdtPr><w:alias w:val="Approved"/><w:checkBox><w:checked w:val="1"/></w:checkBox></w:sdtPr><w:sdtContent><w:r><w:t>Yes</w:t></w:r></w:sdtContent></w:sdt></w:p><w:p><w:fldSimple w:instr=" FORMTEXT "><w:ffData><w:name w:val="Legal name"/><w:helpText w:val="Registered entity name"/><w:textInput/></w:ffData><w:r><w:t>Acme Bank</w:t></w:r></w:fldSimple></w:p></w:body></w:document>`),
	})

	result, err := extractDOCX(context.Background(), data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	var checkbox, textInput *FormControl
	for _, element := range result.Elements {
		if element.Control == nil {
			continue
		}
		switch element.Control.Label {
		case "Approved":
			checkbox = element.Control
		case "Legal name":
			textInput = element.Control
		}
	}
	if checkbox == nil || checkbox.Kind != "CHECKBOX" || checkbox.Checked == nil || !*checkbox.Checked {
		t.Fatalf("checkbox was not preserved: %#v", result.Elements)
	}
	if textInput == nil || textInput.Kind != "TEXT" || textInput.Help != "Registered entity name" {
		t.Fatalf("text input metadata was not preserved: %#v", result.Elements)
	}
}

func TestDOCXNumberingAndNestedTableAnchorsArePreserved(t *testing.T) {
	entries := map[string][]byte{}
	entries["word/numbering.xml"] = []byte(`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:abstractNum w:abstractNumId="7"><w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl></w:abstractNum><w:num w:numId="3"><w:abstractNumId w:val="7"/></w:num></w:numbering>`)
	entries["word/document.xml"] = []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="3"/></w:numPr></w:pPr><w:r><w:t>Provide registration details</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>Country</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Nigeria</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`)
	data := zipFixture(t, entries)

	result, err := extractDOCX(context.Background(), data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	foundNumbered := false
	foundTable := false
	for _, element := range result.Elements {
		if element.Kind == ElementHeading || element.Kind == ElementParagraph {
			foundNumbered = foundNumbered || strings.HasPrefix(element.Text, "1. Provide registration details")
		}
		if element.Kind == ElementTable && len(element.Values) == 1 && slices.Equal(element.Values[0], []string{"Country", "Nigeria"}) && element.Anchor.Table == "table-1" && element.Anchor.RowStart == 1 && element.Anchor.RowEnd == 1 {
			foundTable = true
		}
	}
	if !foundNumbered || !foundTable {
		t.Fatalf("numbering/table structure was not preserved: %#v", result.Elements)
	}
}

func TestDOCXHyperlinksResolveOnlyAllowlistedRelationships(t *testing.T) {
	entries := map[string][]byte{}
	entries["word/_rels/document.xml.rels"] = []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com/policy" TargetMode="External"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="javascript:alert(1)" TargetMode="External"/></Relationships>`)
	entries["word/document.xml"] = []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body><w:p><w:hyperlink r:id="rId1"><w:r><w:t>Policy source</w:t></w:r></w:hyperlink></w:p><w:p><w:hyperlink r:id="rId2"><w:r><w:t>Unsafe source</w:t></w:r></w:hyperlink></w:p></w:body></w:document>`)
	data := zipFixture(t, entries)

	result, err := extractDOCX(context.Background(), data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	foundSafe := false
	for _, element := range result.Elements {
		if element.Kind == ElementLink && element.Text == "Policy source" && element.Target == "https://example.com/policy" {
			foundSafe = true
		}
		if element.Kind == ElementLink && strings.Contains(element.Target, "javascript:") {
			t.Fatalf("unsafe hyperlink target leaked into extraction: %#v", element)
		}
	}
	if !foundSafe {
		t.Fatalf("safe hyperlink was not preserved: %#v", result.Elements)
	}
	if result.Status != ExtractionPartial || !hasDegradation(result.Degradations, "DOCX_LINK_TARGET_REJECTED") {
		t.Fatalf("unsafe relationship must be explicit partial degradation: %#v", result)
	}
}

func TestDOCXElementBudgetReportsTruncation(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>First paragraph.</w:t></w:r></w:p><w:p><w:r><w:t>Second paragraph.</w:t></w:r></w:p></w:body></w:document>`),
	})
	policy := DefaultExtractionPolicy()
	policy.MaxElements = 1

	result, err := extractDOCX(context.Background(), data, policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ExtractionTruncated || !result.ContentTruncated || len(result.Elements) != 1 {
		t.Fatalf("element cap was not represented as truncation: %#v", result)
	}
}

func hasDegradation(values []Degradation, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}
