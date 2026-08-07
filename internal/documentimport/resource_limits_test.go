package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCSVExtractionStreamsAndEnforcesRowBudget(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxRows = 2
	result := ExtractWithPolicy(context.Background(), "rows.csv", "text/csv", []byte("name,value\na,1\nb,2\n"), policy)
	if result.Status != ExtractionFailed {
		t.Fatalf("expected row budget failure, got %#v", result)
	}
	if !strings.Contains(strings.Join(result.Limitations, " "), "row count exceeds 2") {
		t.Fatalf("missing row budget limitation: %#v", result.Limitations)
	}
}

func TestExtractionReportsExactOmittedSectionCount(t *testing.T) {
	policy := DefaultExtractionPolicy()
	policy.MaxSections = 1
	policy.MaxRows = 10
	result := ExtractWithPolicy(context.Background(), "rows.csv", "text/csv", []byte("name,value\na,1\nb,2\nc,3\n"), policy)
	if result.Status != ExtractionExtracted {
		t.Fatalf("expected bounded extraction, got %#v", result)
	}
	if len(result.Sections) != 1 || result.SectionsTotal != 3 || result.SectionsOmitted != 2 || !result.ContentTruncated {
		t.Fatalf("unexpected completeness metadata: %#v", result)
	}
}

func TestOOXMLArchiveRejectsExpandedSizeBeforeParsing(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(bytes.Repeat([]byte("A"), 2<<20)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	policy := DefaultExtractionPolicy()
	policy.MaxExpandedBytes = 1 << 20
	result := ExtractWithPolicy(context.Background(), "bomb.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", buffer.Bytes(), policy)
	if result.Status != ExtractionFailed || !strings.Contains(strings.Join(result.Limitations, " "), "expanded size") {
		t.Fatalf("expanded archive was not rejected: %#v", result)
	}
}

func TestOOXMLArchiveRejectsExtremeCompressionRatio(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(bytes.Repeat([]byte("Z"), 2<<20)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	policy := DefaultExtractionPolicy()
	policy.MaxExpandedBytes = 8 << 20
	policy.MaxCompressionRatio = 5
	policy.CompressionRatioFloor = 1
	result := ExtractWithPolicy(context.Background(), "ratio.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", buffer.Bytes(), policy)
	if result.Status != ExtractionFailed || !strings.Contains(strings.Join(result.Limitations, " "), "compression ratio") {
		t.Fatalf("high-ratio archive was not rejected: %#v", result)
	}
}

func TestXLSXTokenStreamEnforcesColumnBudget(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write([]byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="C1"><v>1</v></c></row></sheetData></worksheet>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	policy := DefaultExtractionPolicy()
	policy.MaxColumns = 2
	result := ExtractWithPolicy(context.Background(), "wide.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes(), policy)
	if result.Status != ExtractionFailed || !strings.Contains(strings.Join(result.Limitations, " "), "column index") {
		t.Fatalf("wide worksheet was not rejected: %#v", result)
	}
}

func TestImportHonorsCancellationBeforeExtractionCompletes(t *testing.T) {
	service := NewService(NewMemoryRepository(), newMemoryStoreForTest())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Import(ctx, ImportInput{TenantID: "bank-demo", FileName: "cancel.txt", CreatedBy: "reviewer-1"}, strings.NewReader("The bank must retain records for five years."))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func newMemoryStoreForTest() *testObjectStore {
	return &testObjectStore{objects: map[string][]byte{}}
}

type testObjectStore struct{ objects map[string][]byte }

func (s *testObjectStore) Put(_ context.Context, key string, reader interface{ Read([]byte) (int, error) }, max int64) (objectInfoStub, error) {
	panic("unused")
}
