package reporting

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRenderXLSXReportWritesAReadableBoundedWorkbook(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	data, err := renderXLSXReport("Third-party risk register", asOf, []string{"record_reference", "finding", "current_status"}, []ReportRow{{
		ID: "matter-1",
		Values: map[string]any{
			"record_reference": "MAT-001",
			"finding":          "=HYPERLINK(\"https://untrusted.example\",\"formula\")",
			"current_status":   "OPEN",
		},
	}})
	if err != nil {
		t.Fatalf("render xlsx: %v", err)
	}
	if len(data) == 0 || !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatalf("workbook is not an xlsx zip archive")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("read xlsx zip: %v", err)
	}
	content := readZipEntry(t, reader, "xl/sharedStrings.xml")
	for _, want := range []string{"Third-party risk register", "MAT-001", "&#39;=HYPERLINK"} {
		if !strings.Contains(content, want) {
			t.Fatalf("workbook text does not contain %q: %s", want, content)
		}
	}
}

func TestXLSXReportUsesTheWorkbookExtension(t *testing.T) {
	if got := reportExtension(FormatXLSX); got != ".xlsx" {
		t.Fatalf("xlsx extension = %q, want .xlsx", got)
	}
}

func readZipEntry(t *testing.T, reader *zip.Reader, name string) string {
	t.Helper()
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		defer stream.Close()
		data, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(data)
	}
	t.Fatalf("xlsx entry %q was not present", name)
	return ""
}
