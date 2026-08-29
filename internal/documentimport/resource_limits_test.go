package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestCSVExtractionStreamsAndEnforcesRowBudget(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxRows = 2
	result := ExtractWithPolicy(context.Background(), "rows.csv", "text/csv", []byte("name,value\na,1\nb,2\n"), policy)
	assertResourceFailure(t, result, "row count exceeds 2")
}

func TestCSVExtractionEnforcesCellSizeBudget(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxCellBytes = 4
	result := ExtractWithPolicy(context.Background(), "cells.csv", "text/csv", []byte("name\nexcess\n"), policy)
	assertResourceFailure(t, result, "exceeds 4 bytes")
}

func TestExtractionReportsExactOmittedSectionCount(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxSections = 1
	policy.MaxRows = 10
	result := ExtractWithPolicy(context.Background(), "rows.csv", "text/csv", []byte("name,value\na,1\nb,2\nc,3\n"), policy)
	if result.Status != ExtractionTruncated {
		t.Fatalf("expected bounded truncation, got %#v", result)
	}
	if len(result.Sections) != 1 || result.SectionsTotal != 3 || result.SectionsOmitted != 2 || !result.ContentTruncated {
		t.Fatalf("unexpected completeness metadata: %#v", result)
	}
}

func TestExtractionReportsRetainedTextTruncation(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxExtractedTextBytes = 5
	result := ExtractWithPolicy(context.Background(), "notice.txt", "text/plain", []byte("alpha beta gamma."), policy)
	if result.Status != ExtractionTruncated || len(result.Sections) != 1 {
		t.Fatalf("expected one bounded truncated text section, got %#v", result)
	}
	if result.Sections[0].Text != "alpha" || !result.ContentTruncated || result.SectionsTotal != 1 || result.SectionsOmitted != 0 {
		t.Fatalf("retained-text budget was not represented truthfully: %#v", result)
	}
}

func TestOOXMLArchiveRejectsEntryCountBeforeParsing(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"word/document.xml": []byte(`<document/>`),
		"word/styles.xml":   []byte(`<styles/>`),
	})
	policy := DefaultExtractionPolicy()
	policy.MaxArchiveEntries = 1
	result := ExtractWithPolicy(context.Background(), "entries.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, policy)
	assertResourceFailure(t, result, "archive entry count exceeds 1")
}

func TestOOXMLArchiveRejectsExpandedSizeBeforeParsing(t *testing.T) {
	data := zipFixture(t, map[string][]byte{"word/document.xml": bytes.Repeat([]byte("A"), 2<<20)})
	policy := DefaultExtractionPolicy()
	policy.MaxExpandedBytes = 1 << 20
	result := ExtractWithPolicy(context.Background(), "bomb.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, policy)
	assertResourceFailure(t, result, "expanded size")
}

func TestOOXMLArchiveRejectsExtremeCompressionRatio(t *testing.T) {
	data := zipFixture(t, map[string][]byte{"word/document.xml": bytes.Repeat([]byte("Z"), 2<<20)})
	policy := DefaultExtractionPolicy()
	policy.MaxExpandedBytes = 8 << 20
	policy.MaxCompressionRatio = 5
	policy.CompressionRatioFloor = 1
	result := ExtractWithPolicy(context.Background(), "ratio.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data, policy)
	assertResourceFailure(t, result, "compression ratio")
}

func TestXLSXTokenStreamEnforcesSheetBudget(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"xl/worksheets/sheet1.xml": []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`),
		"xl/worksheets/sheet2.xml": []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`),
	})
	policy := DefaultExtractionPolicy()
	policy.MaxSheets = 1
	result := ExtractWithPolicy(context.Background(), "sheets.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, policy)
	assertResourceFailure(t, result, "worksheet count exceeds 1")
}

func TestXLSXTokenStreamEnforcesColumnBudget(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"xl/worksheets/sheet1.xml": []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="C1"><v>1</v></c></row></sheetData></worksheet>`),
	})
	policy := DefaultExtractionPolicy()
	policy.MaxColumns = 2
	result := ExtractWithPolicy(context.Background(), "wide.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, policy)
	assertResourceFailure(t, result, "column index")
}

func TestXLSXTokenStreamEnforcesCellBudget(t *testing.T) {
	data := zipFixture(t, map[string][]byte{
		"xl/worksheets/sheet1.xml": []byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1"><v>1</v></c><c r="B1"><v>2</v></c></row></sheetData></worksheet>`),
	})
	policy := DefaultExtractionPolicy()
	policy.MaxCells = 1
	result := ExtractWithPolicy(context.Background(), "cells.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, policy)
	assertResourceFailure(t, result, "cell count exceeds 1")
}

func TestImportHonorsCancellationBeforeExtractionCompletes(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Import(ctx, ImportInput{TenantID: "bank-demo", FileName: "cancel.txt", CreatedBy: "reviewer-1"}, strings.NewReader("The bank must retain records for five years."))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func assertResourceFailure(t *testing.T, result ExtractionResult, contains string) {
	t.Helper()
	if result.Status != ExtractionFailed {
		t.Fatalf("expected resource-limit failure, got %#v", result)
	}
	if !strings.Contains(strings.Join(result.Limitations, " "), contains) {
		t.Fatalf("expected limitation containing %q, got %#v", contains, result.Limitations)
	}
}

func zipFixture(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
