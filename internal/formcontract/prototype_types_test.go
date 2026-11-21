package formcontract

import "testing"

func TestPrototypeQuestionDefinitions(t *testing.T) {
	for _, field := range []Field{
		{ID: "radio", Label: "Controls", Type: TypeGridRadio, Rows: []string{"MFA", "Backups"}, Columns: []string{"Yes", "No"}},
		{ID: "check", Label: "Evidence", Type: TypeGridCheckbox, Rows: []string{"Policy"}, Columns: []string{"Current", "Approved"}},
		{ID: "rating", Label: "Rating", Type: TypeRating, ScaleMax: 5},
		{ID: "scale", Label: "Scale", Type: TypeLinearScale, ScaleMax: 10},
		{ID: "time", Label: "Time", Type: TypeTime},
		{ID: "dropdown", Label: "Owner", Type: TypeDropdown, Options: []string{"Risk", "Security"}},
	} {
		t.Run(string(field.Type), func(t *testing.T) {
			if _, err := Normalize(Contract{Fields: []Field{field}}); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := Normalize(Contract{Fields: []Field{{ID: "bad", Label: "Bad", Type: TypeGridRadio, Rows: []string{"Same", "Same"}, Columns: []string{"Yes"}}}}); err == nil {
		t.Fatal("duplicate rows accepted")
	}
	if _, err := Normalize(Contract{Fields: []Field{{ID: "bad", Label: "Bad", Type: TypeRating, ScaleMax: 11}}}); err == nil {
		t.Fatal("invalid scale accepted")
	}
	if _, err := Normalize(Contract{Fields: []Field{{ID: "bad", Label: "Bad", Type: TypeShortText, Image: &QuestionImage{Data: "data:image/png;base64,YWJj", Alt: "test"}}}}); err == nil {
		t.Fatal("invalid image accepted")
	}
}
