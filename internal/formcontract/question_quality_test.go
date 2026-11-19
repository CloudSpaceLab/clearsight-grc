package formcontract

import "testing"

func TestQuestionQualityContract(t *testing.T) {
	for _, days := range []int{-1, 3651} {
		_, err := Normalize(Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeShortText, AnswerValidityDays: days}}})
		if err == nil {
			t.Fatalf("accepted validity %d", days)
		}
	}
	for _, fieldType := range []Type{TypeFile, TypeMultiSelect, TypeGridRadio} {
		_, err := Normalize(Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: fieldType, ValidationBinding: &ValidationBinding{BindingID: "binding", BindingVersion: 1, ValueField: "id"}}}})
		if err == nil {
			t.Fatalf("accepted source lookup for %s", fieldType)
		}
	}
	got, err := Normalize(Contract{Fields: []Field{{ID: "answer", Label: "Answer", Type: TypeShortText, AnswerValidityDays: 7, ValidationBinding: &ValidationBinding{BindingID: " binding ", BindingVersion: 2, ValueField: " employee_id "}}}})
	if err != nil || got.Fields[0].ValidationBinding.ValueField != "employee_id" || got.Fields[0].AnswerValidityDays != 7 {
		t.Fatal(got, err)
	}
}
