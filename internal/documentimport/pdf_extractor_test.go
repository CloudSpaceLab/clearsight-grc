package documentimport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

type scriptedPDFRunner struct {
	responses map[string]pdfCommandResponse
	calls     []pdfCommandCall
}

type pdfCommandCall struct {
	path        string
	arguments   []string
	stdoutLimit int64
	stderrLimit int64
}

func (r *scriptedPDFRunner) Run(_ context.Context, path string, arguments []string, stdoutLimit, stderrLimit int64) pdfCommandResponse {
	r.calls = append(r.calls, pdfCommandCall{path: path, arguments: append([]string(nil), arguments...), stdoutLimit: stdoutLimit, stderrLimit: stderrLimit})
	return r.responses[path]
}

func TestPDFExtractionPreservesPageBoundariesAndProposalAnchors(t *testing.T) {
	runner := &scriptedPDFRunner{responses: map[string]pdfCommandResponse{
		"pdfinfo":   {Stdout: []byte("Title: Example\nPages:          2\n")},
		"pdftotext": {Stdout: []byte("The institution must retain records for five years.\fManagement shall review resilience controls quarterly.\f")},
	}}
	policy := DefaultExtractionPolicy()
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)

	if result.Status != ExtractionExtracted || result.Method != pdfExtractionMethod {
		t.Fatalf("expected extracted PDF, got %#v", result)
	}
	if len(result.Sections) != 2 || result.Sections[0].Page != 1 || result.Sections[1].Page != 2 {
		t.Fatalf("expected two page-anchored sections, got %#v", result.Sections)
	}
	if result.Sections[0].Title != "Page 1" || result.Sections[1].Title != "Page 2" {
		t.Fatalf("unexpected page titles: %#v", result.Sections)
	}
	proposals := AnalyzeBounded(result.Sections, 10).Proposals
	if len(proposals) != 2 || proposals[0].Anchor.Page != 1 || proposals[1].Anchor.Page != 2 {
		t.Fatalf("expected proposal page anchors, got %#v", proposals)
	}
	if len(runner.calls) != 2 || runner.calls[1].path != "pdftotext" || !containsArgument(runner.calls[1].arguments, "-layout") {
		t.Fatalf("expected pdfinfo then layout-preserving pdftotext, got %#v", runner.calls)
	}
}

func TestPDFExtractionReportsImageOnlyDocumentAsOCRRequired(t *testing.T) {
	runner := &scriptedPDFRunner{responses: map[string]pdfCommandResponse{
		"pdfinfo":   {Stdout: []byte("Pages: 3\n")},
		"pdftotext": {Stdout: []byte("\f\f\f")},
	}}
	policy := DefaultExtractionPolicy()
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)

	if result.Status != ExtractionUnsupported || result.Method != pdfExtractionMethod || len(result.Sections) != 0 {
		t.Fatalf("expected explicit unsupported image-only PDF, got %#v", result)
	}
	if !strings.Contains(strings.ToLower(strings.Join(result.Limitations, " ")), "ocr") {
		t.Fatalf("expected OCR limitation, got %#v", result.Limitations)
	}
}

func TestPDFExtractionReportsUnavailableToolsWithoutRunning(t *testing.T) {
	runner := &scriptedPDFRunner{}
	policy := DefaultExtractionPolicy()
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{}, runner)

	if result.Status != ExtractionUnsupported || result.Method != "NONE" || len(runner.calls) != 0 {
		t.Fatalf("expected unavailable-tool result, got result=%#v calls=%#v", result, runner.calls)
	}
}

func TestPDFExtractionEnforcesPageBudgetBeforeTextConversion(t *testing.T) {
	runner := &scriptedPDFRunner{responses: map[string]pdfCommandResponse{
		"pdfinfo": {Stdout: []byte("Pages: 501\n")},
	}}
	policy := DefaultExtractionPolicy()
	policy.MaxPDFPages = 500
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)

	assertResourceFailure(t, result, "page count exceeds 500")
	if len(runner.calls) != 1 || runner.calls[0].path != "pdfinfo" {
		t.Fatalf("text conversion should not run after page limit failure: %#v", runner.calls)
	}
}

func TestPDFExtractionReportsBoundedOutputFailure(t *testing.T) {
	runner := &scriptedPDFRunner{responses: map[string]pdfCommandResponse{
		"pdfinfo":   {Stdout: []byte("Pages: 1\n")},
		"pdftotext": {Err: ErrResourceLimit},
	}}
	policy := DefaultExtractionPolicy()
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)

	if result.Status != ExtractionFailed || !strings.Contains(strings.Join(result.Limitations, " "), "resource limit") {
		t.Fatalf("expected bounded output failure, got %#v", result)
	}
}

func TestPDFExtractionReportsConverterFailureWithoutLeakingUnboundedDiagnostics(t *testing.T) {
	runner := &scriptedPDFRunner{responses: map[string]pdfCommandResponse{
		"pdfinfo":   {Stdout: []byte("Pages: 1\n")},
		"pdftotext": {Stderr: []byte("Incorrect password"), Err: errors.New("exit status 3")},
	}}
	policy := DefaultExtractionPolicy()
	result := extractPDFWithTools(context.Background(), []byte("%PDF fixture"), newSectionCollector(policy), policy, pdfToolPaths{Info: "pdfinfo", Text: "pdftotext"}, runner)

	if result.Status != ExtractionFailed || !strings.Contains(strings.Join(result.Limitations, " "), "Incorrect password") {
		t.Fatalf("expected converter diagnostic, got %#v", result)
	}
}

func TestPDFExtractionWithInstalledPoppler(t *testing.T) {
	tools := discoverPDFTools()
	if tools.Info == "" || tools.Text == "" {
		t.Skip("Poppler tools are not installed as directly executable binaries")
	}
	data := testPDF(t,
		"The institution must retain records for five years.",
		"Management shall review resilience controls quarterly.",
	)
	result := ExtractWithPolicy(context.Background(), "regulation.pdf", "application/pdf", data, DefaultExtractionPolicy())
	if result.Status != ExtractionExtracted || len(result.Sections) != 2 {
		t.Fatalf("real Poppler extraction failed: %#v", result)
	}
	if result.Sections[0].Page != 1 || result.Sections[1].Page != 2 {
		t.Fatalf("real Poppler page anchors were not preserved: %#v", result.Sections)
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Fatal(err)
	}
}

func testPDF(t *testing.T, pageText ...string) []byte {
	t.Helper()
	pageCount := len(pageText)
	if pageCount == 0 {
		pageText = []string{""}
		pageCount = 1
	}
	fontObject := 3 + pageCount
	firstContentObject := fontObject + 1
	objects := make([]string, 0, firstContentObject+pageCount-1)
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	pageReferences := make([]string, 0, pageCount)
	for index := 0; index < pageCount; index++ {
		pageReferences = append(pageReferences, fmt.Sprintf("%d 0 R", 3+index))
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(pageReferences, " "), pageCount))
	for index := 0; index < pageCount; index++ {
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", fontObject, firstContentObject+index))
	}
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for _, text := range pageText {
		stream := ""
		if text != "" {
			stream = fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", escapePDFText(text))
		}
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	var document bytes.Buffer
	document.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = document.Len()
		fmt.Fprintf(&document, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := document.Len()
	fmt.Fprintf(&document, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index < len(offsets); index++ {
		fmt.Fprintf(&document, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&document, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return document.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	return strings.ReplaceAll(value, ")", "\\)")
}

func containsArgument(arguments []string, expected string) bool {
	for _, argument := range arguments {
		if argument == expected {
			return true
		}
	}
	return false
}
