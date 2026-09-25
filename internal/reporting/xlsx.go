package reporting

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
)

const reportInformationSheet = "Report information"

func renderXLSXReport(title string, asOf time.Time, columns []string, rows []ReportRow) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()

	dataSheet := "Report data"
	if err := book.SetSheetName(book.GetSheetName(0), dataSheet); err != nil {
		return nil, err
	}
	if err := book.SetCellValue(dataSheet, "A1", strings.TrimSpace(title)); err != nil {
		return nil, fmt.Errorf("set report title: %w", err)
	}
	if err := book.SetCellValue(dataSheet, "A2", "As of "+asOf.UTC().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("set report as-of: %w", err)
	}
	if err := book.SetSheetRow(dataSheet, "A4", toSheetValues(columns)); err != nil {
		return nil, fmt.Errorf("set report headings: %w", err)
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
			return nil, fmt.Errorf("report row address: %w", err)
		}
		if err := book.SetSheetRow(dataSheet, cell, toSheetValues(values)); err != nil {
			return nil, fmt.Errorf("set report row: %w", err)
		}
	}
	lastColumn, err := excelize.ColumnNumberToName(max(1, len(columns)))
	if err != nil {
		return nil, fmt.Errorf("report column range: %w", err)
	}
	if err := book.AutoFilter(dataSheet, fmt.Sprintf("A4:%s%d", lastColumn, max(4, len(rows)+4)), nil); err != nil {
		return nil, fmt.Errorf("set report filter: %w", err)
	}
	if err := book.SetPanes(dataSheet, &excelize.Panes{Freeze: true, Split: true, XSplit: 0, YSplit: 4, TopLeftCell: "A5", ActivePane: "bottomLeft"}); err != nil {
		return nil, fmt.Errorf("freeze report headings: %w", err)
	}
	if err := book.SetColWidth(dataSheet, "A", lastColumn, 24); err != nil {
		return nil, fmt.Errorf("set report column widths: %w", err)
	}
	headerStyle, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"0F766E"}, Pattern: 1}, Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"}})
	if err != nil {
		return nil, fmt.Errorf("create report heading style: %w", err)
	}
	if err := book.SetCellStyle(dataSheet, "A4", lastColumn+"4", headerStyle); err != nil {
		return nil, fmt.Errorf("style report headings: %w", err)
	}
	if err := book.SetRowHeight(dataSheet, 4, 32); err != nil {
		return nil, fmt.Errorf("set report heading height: %w", err)
	}

	if _, err := book.NewSheet(reportInformationSheet); err != nil {
		return nil, fmt.Errorf("create report information sheet: %w", err)
	}
	metadata := [][]string{{"Report title", strings.TrimSpace(title)}, {"As of", asOf.UTC().Format(time.RFC3339)}, {"Rows", fmt.Sprintf("%d", len(rows))}, {"Data sheet", dataSheet}}
	for index, row := range metadata {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			return nil, fmt.Errorf("report information address: %w", err)
		}
		if err := book.SetSheetRow(reportInformationSheet, cell, toSheetValues(row)); err != nil {
			return nil, fmt.Errorf("set report information row: %w", err)
		}
	}
	if err := book.SetColWidth(reportInformationSheet, "A", "B", 30); err != nil {
		return nil, fmt.Errorf("set report information width: %w", err)
	}
	book.SetActiveSheet(0)
	output, err := book.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write report workbook: %w", err)
	}
	return output.Bytes(), nil
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
