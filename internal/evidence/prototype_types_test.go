package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
)

func TestPrototypeAnswerValidation(t *testing.T) {
	svc := &Service{}
	tests := []struct {
		field  formcontract.Field
		answer string
		valid  bool
	}{
		{formcontract.Field{Label: "Grid", Type: formcontract.TypeGridRadio, Required: true, Rows: []string{"MFA", "Backup"}, Columns: []string{"Yes", "No"}}, `{"MFA":["Yes"],"Backup":["No"]}`, true},
		{formcontract.Field{Label: "Grid", Type: formcontract.TypeGridRadio, Required: true, Rows: []string{"MFA", "Backup"}, Columns: []string{"Yes", "No"}}, `{"MFA":["Yes"]}`, false},
		{formcontract.Field{Label: "Grid", Type: formcontract.TypeGridCheckbox, Rows: []string{"MFA"}, Columns: []string{"Yes", "No"}}, `{"MFA":["Yes","No"]}`, true},
		{formcontract.Field{Label: "Rating", Type: formcontract.TypeRating, ScaleMax: 5}, "6", false},
		{formcontract.Field{Label: "Scale", Type: formcontract.TypeLinearScale, ScaleMax: 10}, "7", true},
		{formcontract.Field{Label: "Time", Type: formcontract.TypeTime}, "25:30", false},
		{formcontract.Field{Label: "Time", Type: formcontract.TypeTime}, "14:30", true},
		{formcontract.Field{Label: "Choice", Type: formcontract.TypeDropdown, Options: []string{"Risk", "Security"}}, "Risk", true},
	}
	for _, tc := range tests {
		err := svc.validateTypedAnswer(context.Background(), Request{}, Field{}, tc.field, formcontract.AnswerValue{Text: &tc.answer})
		if (err == nil) != tc.valid {
			t.Errorf("%s %s: %v", tc.field.Type, tc.answer, err)
		}
	}
}
