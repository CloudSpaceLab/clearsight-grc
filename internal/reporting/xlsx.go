package reporting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
)

const (
	reportSummarySheet     = "Summary"
	reportDataSheet        = "Report data"
	reportInformationSheet = "Report information"
	maxReportChartBuckets  = 10
)

type reportBreakdownBucket struct {
	Label string
	Count int
}

type reportBreakdown struct {
	Label   string
	Buckets []reportBreakdownBucket
}

func renderXLSXReport(title string, asOf time.Time, columns []string, rows []ReportRow) ([]byte, error) {
	title = reportDisplayTitle(title)
	book := excelize.NewFile()
	defer book.Close()

	if err := book.SetSheetName(book.GetSheetName(0), reportSummarySheet); err != nil {
		return nil, err
	}
	if _, err := book.NewSheet(reportDataSheet); err != nil {
		return nil, fmt.Errorf("create report data sheet: %w", err)
	}
	if _, err := book.NewSheet(reportInformationSheet); err != nil {
		return nil, fmt.Errorf("create report information sheet: %w", err)
	}

	if err := renderReportSummarySheet(book, title, asOf, columns, rows); err != nil {
		return nil, err
	}
	if err := renderReportDataSheet(book, title, asOf, columns, rows); err != nil {
		return nil, err
	}
	if err := renderReportInformationSheet(book, title, asOf, len(rows)); err != nil {
		return nil, err
	}

	book.SetActiveSheet(0)
	output, err := book.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write report workbook: %w", err)
	}
	return output.Bytes(), nil
}

func renderReportSummarySheet(book *excelize.File, title string, asOf time.Time, columns []string, rows []ReportRow) error {
	if err := book.SetCellValue(reportSummarySheet, "A1", strings.TrimSpace(title)); err != nil {
		return fmt.Errorf("set report summary title: %w", err)
	}
	if err := book.SetCellValue(reportSummarySheet, "A2", "As of "+asOf.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set report summary as-of: %w", err)
	}
	if err := book.SetCellValue(reportSummarySheet, "A4", "Rows"); err != nil {
		return fmt.Errorf("set report summary metric label: %w", err)
	}
	if err := book.SetCellValue(reportSummarySheet, "B4", len(rows)); err != nil {
		return fmt.Errorf("set report summary row count: %w", err)
	}

	titleStyle, err := book.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18},
	})
	if err != nil {
		return fmt.Errorf("create report summary title style: %w", err)
	}
	if err := book.SetCellStyle(reportSummarySheet, "A1", "B1", titleStyle); err != nil {
		return fmt.Errorf("style report summary title: %w", err)
	}

	metricStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("create report summary metric style: %w", err)
	}
	if err := book.SetCellStyle(reportSummarySheet, "A4", "B4", metricStyle); err != nil {
		return fmt.Errorf("style report summary metric: %w", err)
	}

	breakdown, ok := buildReportBreakdown(columns, rows)
	if ok {
		if err := book.SetCellValue(reportSummarySheet, "A6", breakdown.Label); err != nil {
			return fmt.Errorf("set report breakdown title: %w", err)
		}
		if err := book.SetSheetRow(reportSummarySheet, "A7", toSheetValues([]string{"Category", "Count"})); err != nil {
			return fmt.Errorf("set report breakdown headings: %w", err)
		}
		for index, bucket := range breakdown.Buckets {
			rowNumber := index + 8
			if err := book.SetCellValue(reportSummarySheet, fmt.Sprintf("A%d", rowNumber), bucket.Label); err != nil {
				return fmt.Errorf("set report breakdown category: %w", err)
			}
			if err := book.SetCellValue(reportSummarySheet, fmt.Sprintf("B%d", rowNumber), bucket.Count); err != nil {
				return fmt.Errorf("set report breakdown count: %w", err)
			}
		}
		headerStyle, err := reportHeaderStyle(book)
		if err != nil {
			return err
		}
		if err := book.SetCellStyle(reportSummarySheet, "A7", "B7", headerStyle); err != nil {
			return fmt.Errorf("style report breakdown headings: %w", err)
		}
		lastRow := 7 + len(breakdown.Buckets)
		if err := book.AddChart(reportSummarySheet, "D4", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{{
				Name:       reportSummarySheet + "!$B$7",
				Categories: fmt.Sprintf("%s!$A$8:$A$%d", reportSummarySheet, lastRow),
				Values:     fmt.Sprintf("%s!$B$8:$B$%d", reportSummarySheet, lastRow),
			}},
			Title: []excelize.RichTextRun{{Text: breakdown.Label}},
			PlotArea: excelize.ChartPlotArea{
				ShowCatName: false,
				ShowSerName: false,
				ShowVal:     true,
			},
			ShowBlanksAs: "zero",
		}); err != nil {
			return fmt.Errorf("add report summary chart: %w", err)
		}
	}

	if err := book.SetColWidth(reportSummarySheet, "A", "A", 28); err != nil {
		return fmt.Errorf("set report summary label width: %w", err)
	}
	if err := book.SetColWidth(reportSummarySheet, "B", "B", 16); err != nil {
		return fmt.Errorf("set report summary value width: %w", err)
	}
	return nil
}

func renderReportDataSheet(book *excelize.File, title string, asOf time.Time, columns []string, rows []ReportRow) error {
	if err := book.SetCellValue(reportDataSheet, "A1", strings.TrimSpace(title)); err != nil {
		return fmt.Errorf("set report title: %w", err)
	}
	if err := book.SetCellValue(reportDataSheet, "A2", "As of "+asOf.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("set report as-of: %w", err)
	}
	if err := book.SetSheetRow(reportDataSheet, "A4", toSheetValues(columns)); err != nil {
		return fmt.Errorf("set report headings: %w", err)
	}
	for rowIndex, row := range rows {
		values := make([]string, len(columns))
		for columnIndex, column := range columns {
			if column == "id" {
				values[columnIndex] = spreadsheetSafeReportValue(row.ID)
				continue
			}
			values[columnIndex] = spreadsheetSafeReportValue(reportValueString(row.Values[column]))
		}
		cell, err := excelize.CoordinatesToCellName(1, rowIndex+5)
		if err != nil {
			return fmt.Errorf("report row address: %w", err)
		}
		if err := book.SetSheetRow(reportDataSheet, cell, toSheetValues(values)); err != nil {
			return fmt.Errorf("set report row: %w", err)
		}
	}
	lastColumn, err := excelize.ColumnNumberToName(max(1, len(columns)))
	if err != nil {
		return fmt.Errorf("report column range: %w", err)
	}
	if err := book.AutoFilter(reportDataSheet, fmt.Sprintf("A4:%s%d", lastColumn, max(4, len(rows)+4)), nil); err != nil {
		return fmt.Errorf("set report filter: %w", err)
	}
	if err := book.SetPanes(reportDataSheet, &excelize.Panes{Freeze: true, Split: true, XSplit: 0, YSplit: 4, TopLeftCell: "A5", ActivePane: "bottomLeft"}); err != nil {
		return fmt.Errorf("freeze report headings: %w", err)
	}
	if err := book.SetColWidth(reportDataSheet, "A", lastColumn, 24); err != nil {
		return fmt.Errorf("set report column widths: %w", err)
	}
	headerStyle, err := reportHeaderStyle(book)
	if err != nil {
		return err
	}
	if err := book.SetCellStyle(reportDataSheet, "A4", lastColumn+"4", headerStyle); err != nil {
		return fmt.Errorf("style report headings: %w", err)
	}
	if err := book.SetRowHeight(reportDataSheet, 4, 32); err != nil {
		return fmt.Errorf("set report heading height: %w", err)
	}
	return nil
}

func renderReportInformationSheet(book *excelize.File, title string, asOf time.Time, rowCount int) error {
	metadata := [][]string{
		{"Report title", strings.TrimSpace(title)},
		{"As of", asOf.UTC().Format(time.RFC3339)},
		{"Rows", strconv.Itoa(rowCount)},
		{"Data sheet", reportDataSheet},
		{"Summary sheet", reportSummarySheet},
	}
	for index, row := range metadata {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			return fmt.Errorf("report information address: %w", err)
		}
		if err := book.SetSheetRow(reportInformationSheet, cell, toSheetValues(row)); err != nil {
			return fmt.Errorf("set report information row: %w", err)
		}
	}
	if err := book.SetColWidth(reportInformationSheet, "A", "B", 30); err != nil {
		return fmt.Errorf("set report information width: %w", err)
	}
	return nil
}

func reportHeaderStyle(book *excelize.File) (int, error) {
	style, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"0F766E"}, Pattern: 1},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"},
	})
	if err != nil {
		return 0, fmt.Errorf("create report heading style: %w", err)
	}
	return style, nil
}

func buildReportBreakdown(columns []string, rows []ReportRow) (reportBreakdown, bool) {
	if len(rows) == 0 {
		return reportBreakdown{}, false
	}
	column := preferredBreakdownColumn(columns)
	if column == "" {
		return reportBreakdown{}, false
	}
	counts := make(map[string]int)
	for _, row := range rows {
		value := strings.TrimSpace(reportValueString(row.Values[column]))
		if value == "" {
			value = "Not recorded"
		} else {
			value = humanizeReportValue(value)
		}
		counts[value]++
	}
	if len(counts) == 0 {
		return reportBreakdown{}, false
	}
	buckets := make([]reportBreakdownBucket, 0, len(counts))
	for label, count := range counts {
		buckets = append(buckets, reportBreakdownBucket{Label: label, Count: count})
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Count == buckets[j].Count {
			return buckets[i].Label < buckets[j].Label
		}
		return buckets[i].Count > buckets[j].Count
	})
	if len(buckets) > maxReportChartBuckets {
		other := 0
		for _, bucket := range buckets[maxReportChartBuckets-1:] {
			other += bucket.Count
		}
		buckets = append(buckets[:maxReportChartBuckets-1], reportBreakdownBucket{Label: "Other", Count: other})
	}
	return reportBreakdown{Label: reportBreakdownLabel(column), Buckets: buckets}, true
}

func preferredBreakdownColumn(columns []string) string {
	available := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		available[column] = struct{}{}
	}
	for _, candidate := range []string{
		"overall_state",
		"status",
		"current_status",
		"criticality",
		"matter_type",
		"privacy_role",
		"priority",
		"jurisdiction",
		"vendor_status",
	} {
		if _, ok := available[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func reportBreakdownLabel(column string) string {
	switch column {
	case "overall_state":
		return "Overall state"
	case "current_status", "status":
		return "Status"
	case "criticality":
		return "Criticality"
	case "matter_type":
		return "Work type"
	case "privacy_role":
		return "Privacy role"
	case "priority":
		return "Priority"
	case "vendor_status":
		return "Vendor status"
	case "jurisdiction":
		return "Jurisdiction"
	default:
		return humanizeReportValue(column)
	}
}

func reportDisplayTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "ClearSight report"
	}
	if value != strings.ToUpper(value) || strings.ContainsAny(value, " \t") {
		return value
	}
	parts := strings.Split(value, "_")
	if len(parts) > 1 && isShortHexSuffix(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
		value = strings.Join(parts, "_")
	}
	return humanizeReportValue(value)
}

func isShortHexSuffix(value string) bool {
	if len(value) != 8 {
		return false
	}
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

func humanizeReportValue(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "_", " "), "-", " "))
	if value == "" {
		return ""
	}
	words := strings.Fields(strings.ToLower(value))
	for index, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

func toSheetValues(values []string) *[]interface{} {
	result := make([]interface{}, len(values))
	for index, value := range values {
		result[index] = value
	}
	return &result
}

func spreadsheetSafeReportValue(value string) string {
	for _, character := range value {
		if character == '\t' || character == '\r' || character == '\n' {
			return "'" + value
		}
		if unicode.IsSpace(character) {
			continue
		}
		if character == '=' || character == '+' || character == '-' || character == '@' {
			return "'" + value
		}
		break
	}
	return value
}
