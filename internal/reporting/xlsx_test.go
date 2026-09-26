package reporting

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRenderXLSXReportWritesAReadableSummaryChartAndDataWorkbook(t *testing.T) {
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
	for _, want := range []string{"Third-party risk register", "MAT-001", "&#39;=HYPERLINK", "Rows", "Status", "Open", "Category", "Count"} {
		if !strings.Contains(content, want) {
			t.Fatalf("workbook text does not contain %q: %s", want, content)
		}
	}
	if !zipContainsPrefix(reader, "xl/charts/chart") {
		t.Fatal("workbook does not contain the derived summary chart")
	}
	workbook := readZipEntry(t, reader, "xl/workbook.xml")
	for _, sheet := range []string{reportSummarySheet, reportDataSheet, reportInformationSheet} {
		if !strings.Contains(workbook, `name="`+sheet+`"`) {
			t.Fatalf("workbook does not contain %q sheet: %s", sheet, workbook)
		}
	}
}

func TestReportBreakdownPrefersMeaningfulStatusAndBoundsChartBuckets(t *testing.T) {
	rows := make([]ReportRow, 0, 14)
	for index := 0; index < 14; index++ {
		rows = append(rows, ReportRow{Values: map[string]any{
			"status": fmt.Sprintf("STATE_%02d", index),
			"priority": index % 5,
		}})
	}
	breakdown, ok := buildReportBreakdown([]string{"priority", "status"}, rows)
	if !ok {
		t.Fatal("expected a report breakdown")
	}
	if breakdown.Label != "Status" {
		t.Fatalf("breakdown label = %q, want Status", breakdown.Label)
	}
	if len(breakdown.Buckets) != maxReportChartBuckets {
		t.Fatalf("breakdown buckets = %d, want %d", len(breakdown.Buckets), maxReportChartBuckets)
	}
	if breakdown.Buckets[len(breakdown.Buckets)-1].Label != "Other" {
		t.Fatalf("last bucket = %#v, want Other", breakdown.Buckets[len(breakdown.Buckets)-1])
	}
}

func TestReportDisplayTitleRemovesGeneratedCodeSuffix(t *testing.T) {
	if got := reportDisplayTitle("BOARD_VENDOR_SUMMARY_1234ABCD"); got != "Board Vendor Summary" {
		t.Fatalf("display title = %q, want Board Vendor Summary", got)
	}
	if got := reportDisplayTitle("Board vendor summary"); got != "Board vendor summary" {
		t.Fatalf("human title changed to %q", got)
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

func zipContainsPrefix(reader *zip.Reader, prefix string) bool {
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, prefix) {
			return true
		}
	}
	return false
}
