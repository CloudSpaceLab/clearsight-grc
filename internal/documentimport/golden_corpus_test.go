package documentimport

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type goldenManifest struct {
	Version int          `json:"version"`
	Cases   []goldenCase `json:"cases"`
}

type goldenCase struct {
	Name                 string           `json:"name"`
	SourceKind           string           `json:"source_kind"`
	ExecutionMode        string           `json:"execution_mode"`
	ExpectedStatus       ExtractionStatus `json:"expected_status"`
	ElementKinds         []ElementKind    `json:"element_kinds"`
	RequiredDegradations []string         `json:"required_degradations"`
	MaxDurationClass     string           `json:"max_duration_class"`
}

func TestGoldenCorpus(t *testing.T) {
	manifest := readGoldenManifest(t)
	if manifest.Version != 1 || len(manifest.Cases) == 0 {
		t.Fatalf("unexpected golden manifest: %#v", manifest)
	}

	seen := map[string]struct{}{}
	for _, testCase := range manifest.Cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.Name == "" || testCase.SourceKind == "" || testCase.ExecutionMode == "" || testCase.ExpectedStatus == "" || testCase.MaxDurationClass == "" {
				t.Fatalf("incomplete golden case: %#v", testCase)
			}
			if _, duplicate := seen[testCase.Name]; duplicate {
				t.Fatalf("duplicate golden case %q", testCase.Name)
			}
			seen[testCase.Name] = struct{}{}

			if strings.HasPrefix(testCase.ExecutionMode, "TASK16_") {
				if testCase.MaxDurationClass != "BOUNDED_EXTERNAL" {
					t.Fatalf("future adapter case must declare bounded external execution: %#v", testCase)
				}
				return
			}

			started := time.Now()
			result := runGoldenCase(t, testCase.Name)
			if elapsed := time.Since(started); elapsed > goldenDurationLimit(testCase.MaxDurationClass) {
				t.Fatalf("golden case %q exceeded %s duration class: %s", testCase.Name, testCase.MaxDurationClass, elapsed)
			}
			if result.Status != testCase.ExpectedStatus {
				t.Fatalf("golden case %q status: got %s want %s; result=%#v", testCase.Name, result.Status, testCase.ExpectedStatus, result)
			}
			actualKinds := make([]ElementKind, 0, len(result.Elements))
			for _, element := range result.Elements {
				actualKinds = append(actualKinds, element.Kind)
			}
			if !reflect.DeepEqual(actualKinds, testCase.ElementKinds) {
				t.Fatalf("golden case %q element kinds: got %#v want %#v", testCase.Name, actualKinds, testCase.ElementKinds)
			}
			for _, code := range testCase.RequiredDegradations {
				if !hasDegradation(result.Degradations, code) {
					t.Fatalf("golden case %q missing degradation %q: %#v", testCase.Name, code, result.Degradations)
				}
			}
		})
	}
}

func TestDOCXArchiveRejectsPathTraversal(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"../word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`),
	})
	result := ExtractWithPolicy(context.Background(), "unsafe.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, DefaultExtractionPolicy())
	assertResourceFailure(t, result, "unsafe entry path")
}

func readGoldenManifest(t *testing.T) goldenManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/golden/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest goldenManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func runGoldenCase(t *testing.T, name string) ExtractionResult {
	t.Helper()
	switch name {
	case "docx_dropdown":
		data := zipFixture(t, map[string][]byte{
			"word/document.xml": []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Choose a country: </w:t></w:r><w:sdt><w:sdtPr><w:alias w:val="Country"/><w:dropDownList><w:listItem w:displayText="Nigeria" w:value="NG"/><w:listItem w:displayText="Ghana" w:value="GH"/></w:dropDownList></w:sdtPr><w:sdtContent><w:r><w:t>Nigeria</w:t></w:r></w:sdtContent></w:sdt></w:p></w:body></w:document>`),
		})
		return ExtractWithPolicy(context.Background(), "questionnaire.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, DefaultExtractionPolicy())
	case "docx_unsafe_link":
		entries := map[string][]byte{}
		entries["word/_rels/document.xml.rels"] = []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="javascript:alert(1)" TargetMode="External"/></Relationships>`)
		entries["word/document.xml"] = []byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body><w:p><w:hyperlink r:id="rId1"><w:r><w:t>Unsafe source</w:t></w:r></w:hyperlink></w:p></w:body></w:document>`)
		return ExtractWithPolicy(context.Background(), "unsafe-link.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", zipFixture(t, entries), DefaultExtractionPolicy())
	case "xlsx_single_row":
		data := zipFixture(t, map[string][]byte{
			"xl/worksheets/sheet1.xml": []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1"><v>42</v></c></row></sheetData></worksheet>`),
		})
		return ExtractWithPolicy(context.Background(), "single.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, DefaultExtractionPolicy())
	case "pdf_native_text":
		policy := DefaultExtractionPolicy()
		responses := map[string]pdfCommandResponse{}
		responses["pdfinfo"] = pdfCommandResponse{Stdout: []byte("Pages: 1\n")}
		responses["pdftotext"] = pdfCommandResponse{Stdout: []byte("The bank must retain records for five years.\f")}
		runner := &scriptedPDFRunner{responses: responses}
		return extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)
	case "pdf_scanned":
		policy := DefaultExtractionPolicy()
		responses := map[string]pdfCommandResponse{}
		responses["pdfinfo"] = pdfCommandResponse{Stdout: []byte("Pages: 1\n")}
		responses["pdftotext"] = pdfCommandResponse{Stdout: []byte("\f")}
		runner := &scriptedPDFRunner{responses: responses}
		return extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)
	case "docx_malformed":
		return ExtractWithPolicy(context.Background(), "malformed.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("not a zip archive"), DefaultExtractionPolicy())
	case "docx_expansion_bomb":
		policy := DefaultExtractionPolicy()
		policy.MaxExpandedBytes = 1024
		data := zipFixture(t, map[string][]byte{"word/document.xml": bytes.Repeat([]byte("A"), 2048)})
		return ExtractWithPolicy(context.Background(), "bomb.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, policy)
	default:
		t.Fatalf("no executable golden builder for %q", name)
		return ExtractionResult{}
	}
}

func goldenDurationLimit(class string) time.Duration {
	switch class {
	case "FAST":
		return 2 * time.Second
	case "BOUNDED_EXTERNAL":
		return 30 * time.Second
	default:
		return 0
	}
}
