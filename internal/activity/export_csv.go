package activity

import "unicode"

// spreadsheetSafeCSVRow preserves the recorded text while ensuring common
// spreadsheet applications do not interpret actor-, vendor-, or object-controlled
// values as formulas when a human opens a CSV export. NDJSON remains the exact
// machine-readable representation.
func spreadsheetSafeCSVRow(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = spreadsheetSafeCSVCell(value)
	}
	return result
}

func spreadsheetSafeCSVCell(value string) string {
	for index, r := range value {
		if index == 0 && (r == '\t' || r == '\r' || r == '\n') {
			return "'" + value
		}
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '=', '+', '-', '@':
			return "'" + value
		default:
			return value
		}
	}
	return value
}
