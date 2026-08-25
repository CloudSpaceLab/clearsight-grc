package formcontract

import (
	"errors"
	"testing"
)

func TestVisibleFieldsUsesEarlierAnswersAndSectionConditions(t *testing.T) {
	contract := Contract{
		Sections: []Section{
			{ID: "profile", Title: "Profile"},
			{ID: "data", Title: "Customer data", Condition: &VisibilityCondition{FieldID: "handles_data", Operator: ConditionEquals, Values: []string{"Yes"}}},
		},
		Fields: []Field{
			{ID: "handles_data", SectionID: "profile", Label: "Handles customer data", Type: TypeYesNo, Required: true},
			{ID: "data_location", SectionID: "data", Label: "Data location", Type: TypeShortText, Required: true},
			{ID: "outside_nigeria", SectionID: "data", Label: "Outside Nigeria", Type: TypeYesNo, Condition: &VisibilityCondition{FieldID: "data_location", Operator: ConditionAnswered}},
		},
	}
	normalized, err := Normalize(contract)
	if err != nil {
		t.Fatal(err)
	}
	visible, err := VisibleFields(normalized, map[string]AnswerValue{"handles_data": TextAnswer("No")})
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != "handles_data" {
		t.Fatalf("unexpected fields %#v", visible)
	}
	visible, err = VisibleFields(normalized, map[string]AnswerValue{"handles_data": TextAnswer("Yes"), "data_location": TextAnswer("Lagos")})
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 3 || visible[2].ID != "outside_nigeria" {
		t.Fatalf("unexpected fields %#v", visible)
	}
}

func TestNormalizeRejectsInvalidVisibilityGraph(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
	}{
		{name: "later dependency", fields: []Field{
			{ID: "detail", Label: "Detail", Type: TypeShortText, Condition: &VisibilityCondition{FieldID: "decision", Operator: ConditionEquals, Values: []string{"Yes"}}},
			{ID: "decision", Label: "Decision", Type: TypeYesNo},
		}},
		{name: "unknown dependency", fields: []Field{{ID: "detail", Label: "Detail", Type: TypeShortText, Condition: &VisibilityCondition{FieldID: "missing", Operator: ConditionAnswered}}}},
		{name: "unsupported operator", fields: []Field{
			{ID: "decision", Label: "Decision", Type: TypeYesNo},
			{ID: "detail", Label: "Detail", Type: TypeShortText, Condition: &VisibilityCondition{FieldID: "decision", Operator: "CONTAINS"}},
		}},
		{name: "too many values", fields: []Field{
			{ID: "decision", Label: "Decision", Type: TypeYesNo},
			{ID: "detail", Label: "Detail", Type: TypeShortText, Condition: &VisibilityCondition{FieldID: "decision", Operator: ConditionIn, Values: make([]string, 21)}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize(Contract{Fields: test.fields})
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid visibility contract, got %v", err)
			}
		})
	}
}

func TestNormalizeRejectsVisibilityDepthOverFive(t *testing.T) {
	fields := []Field{{ID: "field_0", Label: "Field 0", Type: TypeYesNo}}
	for index := 1; index <= 6; index++ {
		fields = append(fields, Field{
			ID: "field_" + string(rune('0'+index)), Label: "Field", Type: TypeYesNo,
			Condition: &VisibilityCondition{FieldID: "field_" + string(rune('0'+index-1)), Operator: ConditionEquals, Values: []string{"Yes"}},
		})
	}
	if _, err := Normalize(Contract{Fields: fields}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected excessive condition depth, got %v", err)
	}
}
