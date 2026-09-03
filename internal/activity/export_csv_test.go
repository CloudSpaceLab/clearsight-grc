package activity

import "testing"

func TestSpreadsheetSafeCSVCellNeutralizesFormulaPrefixes(t *testing.T) {
	cases := map[string]string{
		"=HYPERLINK(\"https://example.invalid\")": "'=HYPERLINK(\"https://example.invalid\")",
		"+cmd|' /C calc'!A0":                      "'+cmd|' /C calc'!A0",
		"-1+1":                                    "'-1+1",
		"@SUM(1,1)":                               "'@SUM(1,1)",
		"  =SUM(1,1)":                             "'  =SUM(1,1)",
		"\tformula-like":                          "'\tformula-like",
		"ordinary vendor":                         "ordinary vendor",
	}
	for input, want := range cases {
		if got := spreadsheetSafeCSVCell(input); got != want {
			t.Fatalf("spreadsheetSafeCSVCell(%q) = %q, want %q", input, got, want)
		}
	}
}
