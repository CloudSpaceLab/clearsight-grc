//go:build postgres

package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var sourceCellRef = regexp.MustCompile(`(?i)^\$?([A-Z]{1,3})\$?([1-9][0-9]*)$`)

var biaImpactHeadings = [...]string{
	"Impact on bank customers",
	"Impact on finance",
	"Impact on staff",
	"Impact on regulatoror legal exposure",
	"Impact on other Dependent processes",
	"Impact on Reputation image",
}

func sourceWorkbookName(group sourceRecordGroup) string {
	return strings.ToLower(path.Base(strings.ReplaceAll(group.SourceFile, "\\", "/")))
}

func sourceCellParts(cell string) (column, row string) {
	match := sourceCellRef.FindStringSubmatch(strings.TrimSpace(cell))
	if len(match) != 3 {
		return "", ""
	}
	return strings.ToUpper(match[1]), match[2]
}

func sourceColumnName(index int) string {
	var name string
	for index++; index > 0; index = (index - 1) / 26 {
		name = string(rune('A'+(index-1)%26)) + name
	}
	return name
}

// The six generic BIA headings are adjacent numeric ratings for six named
// impact dimensions. Require the exact workbook, sheet, heading and cells;
// never apply this inference to other "ColumnN" fields.
func sourceBIARatingLabel(group sourceRecordGroup, record sourceRecord, index int) (string, bool) {
	if sourceWorkbookName(group) != "business_impact_analysis.xlsx" ||
		!strings.EqualFold(strings.TrimSpace(group.SourceSheet), "BIA Master") ||
		index < 5 || index > 15 || index%2 == 0 || index >= len(record.Fields) {
		return "", false
	}
	pair := (index - 5) / 2
	rating, dimension := record.Fields[index], record.Fields[index-1]
	if sourceHeaderKey(rating.Label) != fmt.Sprintf("column%d", pair+1) ||
		sourceHeaderKey(dimension.Label) != sourceHeaderKey(biaImpactHeadings[pair]) {
		return "", false
	}
	dimensionColumn, dimensionRow := sourceCellParts(dimension.SourceCell)
	ratingColumn, ratingRow := sourceCellParts(rating.SourceCell)
	if dimensionColumn != sourceColumnName(index-1) || ratingColumn != sourceColumnName(index) ||
		dimensionRow == "" || dimensionRow != ratingRow {
		return "", false
	}
	return strings.TrimSpace(dimension.Label) + " — rating", true
}

func sourceV2FieldLabel(group sourceRecordGroup, record sourceRecord, index int) string {
	if title, ok := sourceBIARatingLabel(group, record, index); ok {
		return title
	}
	if index >= 0 && index < len(record.Fields) {
		if label := strings.TrimSpace(record.Fields[index].Label); label != "" {
			return label
		}
	}
	return "Source value"
}

func sourceV2FieldDescription(group sourceRecordGroup, record sourceRecord, index int) string {
	field := record.Fields[index]
	if _, mapped := sourceBIARatingLabel(group, record, index); mapped {
		column, _ := sourceCellParts(field.SourceCell)
		if group.ResponsePerRecord {
			return fmt.Sprintf("Original heading: %s · source column %s", field.Label, column)
		}
		return fmt.Sprintf("Original heading: %s · source cell %s", field.Label, field.SourceCell)
	}
	if group.ResponsePerRecord {
		if column, _ := sourceCellParts(field.SourceCell); column != "" {
			return "Source column " + column
		}
		return ""
	}
	return field.SourceCell
}

// A readable representation of an immutable source row. Original header,
// source-cell identity and values remain unchanged in the manifest.
func sourceRecordTextForGroupV2(group sourceRecordGroup, record sourceRecord) string {
	lines := make([]string, 0, len(record.Fields))
	seen := make(map[string]int, len(record.Fields))
	for _, field := range record.Fields {
		seen[sourceHeaderKey(field.Label)]++
	}
	for index, field := range record.Fields {
		label := sourceV2FieldLabel(group, record, index)
		if _, mapped := sourceBIARatingLabel(group, record, index); mapped {
			label += fmt.Sprintf(" (%s, %s)", field.Label, field.SourceCell)
		} else if seen[sourceHeaderKey(field.Label)] > 1 || sourceHeaderKey(field.Label) == "" {
			if cell := strings.TrimSpace(field.SourceCell); cell != "" {
				label += " (" + cell + ")"
			}
		}
		value := strings.TrimSpace(field.Value)
		if value == "" {
			value = "Not recorded in source"
		} else {
			value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
			value = strings.ReplaceAll(value, "\n", "\n  ")
		}
		lines = append(lines, label+": "+value)
	}
	return strings.Join(lines, "\n")
}

// Fail closed for source-specific V2 manifests. Historical V1 forms and
// responses are never constrained or revised by these additional checks.
func validateOpsRiskV2Group(group sourceRecordGroup) error {
	if group.PresentationVersion != 2 {
		return nil
	}
	switch sourceWorkbookName(group) {
	case "branch kri .xlsx":
		if group.Key != "ops-branch-kri" || !strings.EqualFold(strings.TrimSpace(group.SourceSheet), "Branch KRI November '25") ||
			!group.ResponsePerRecord || len(group.Records) != 31 {
			return fmt.Errorf("Branch KRI V2 requires one 31-record response-per-record register with verified source sheet")
		}
		blanks := 0
		for index, record := range group.Records {
			row := index + 2
			if len(record.Fields) != 59 || strings.TrimSpace(sourceFieldValue(record, "Branch")) == "" ||
				!strings.HasSuffix(strings.ToUpper(record.SourceRange), strings.ToUpper(fmt.Sprintf("!A%d:BG%d", row, row))) {
				return fmt.Errorf("Branch KRI V2 row %d lacks the 59-column source mapping", row)
			}
			for column, field := range record.Fields {
				name, sourceRow := sourceCellParts(field.SourceCell)
				if name != sourceColumnName(column) || sourceRow != fmt.Sprint(row) ||
					sourceHeaderKey(field.Label) != sourceHeaderKey(group.Records[0].Fields[column].Label) {
					return fmt.Errorf("Branch KRI V2 row %d column %s lacks source cell/header parity", row, sourceColumnName(column))
				}
				if strings.TrimSpace(field.Value) == "" {
					blanks++
				}
			}
		}
		if blanks != 128 {
			return fmt.Errorf("Branch KRI V2 has %d unanswered cells, want 128 (never infer No or 0)", blanks)
		}
	case "business_impact_analysis.xlsx":
		if !strings.EqualFold(strings.TrimSpace(group.SourceSheet), "BIA Master") {
			return nil // RTO, 2019 and technical sheets retain independent identities.
		}
		if len(group.Records) != 602 {
			return fmt.Errorf("BIA Master V2 requires all 602 original rows in a single source register")
		}
		for index, record := range group.Records {
			row := index + 2
			if len(record.Fields) != 28 ||
				!strings.HasSuffix(strings.ToUpper(record.SourceRange), strings.ToUpper(fmt.Sprintf("!A%d:AB%d", row, row))) {
				return fmt.Errorf("BIA Master V2 row %d lacks its 28-column source mapping", row)
			}
			for column, field := range record.Fields {
				name, sourceRow := sourceCellParts(field.SourceCell)
				if name != sourceColumnName(column) || sourceRow != fmt.Sprint(row) {
					return fmt.Errorf("BIA Master V2 row %d column %s lacks source provenance", row, sourceColumnName(column))
				}
			}
			for column := 5; column <= 15; column += 2 {
				if _, ok := sourceBIARatingLabel(group, record, column); !ok {
					return fmt.Errorf("BIA Master V2 row %d impact column %s cannot be safely paired", row, sourceColumnName(column))
				}
			}
		}
	case "head office kri monthly.xlsx":
		if strings.EqualFold(strings.TrimSpace(group.SourceSheet), "Board report") ||
			strings.EqualFold(strings.TrimSpace(group.SourceSheet), "calculation") {
			return fmt.Errorf("Head Office KRI %q is a report/calculation view, not an independent V2 observation source", group.SourceSheet)
		}
	}
	return nil
}
