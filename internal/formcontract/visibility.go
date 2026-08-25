package formcontract

import (
	"fmt"
	"slices"
	"strings"
)

const (
	MaxConditionValues = 20
	MaxConditionDepth  = 5
)

func validateVisibility(contract *Contract) error {
	fieldIndex := make(map[string]int, len(contract.Fields))
	fieldDepth := make(map[string]int, len(contract.Fields))
	for index, field := range contract.Fields {
		fieldIndex[field.ID] = index
	}

	sectionCondition := make(map[string]*VisibilityCondition, len(contract.Sections))
	for index := range contract.Sections {
		condition, err := normalizeCondition(contract.Sections[index].Condition)
		if err != nil {
			return err
		}
		contract.Sections[index].Condition = condition
		sectionCondition[contract.Sections[index].ID] = condition
	}

	for index := range contract.Fields {
		field := &contract.Fields[index]
		condition, err := normalizeCondition(field.Condition)
		if err != nil {
			return err
		}
		field.Condition = condition

		depth := 0
		for _, current := range []*VisibilityCondition{sectionCondition[field.SectionID], field.Condition} {
			if current == nil {
				continue
			}
			dependencyIndex, exists := fieldIndex[current.FieldID]
			if !exists {
				return invalid("%s visibility references an unknown field", field.Label)
			}
			if dependencyIndex >= index {
				return invalid("%s visibility must reference an earlier field", field.Label)
			}
			candidateDepth := fieldDepth[current.FieldID] + 1
			if candidateDepth > depth {
				depth = candidateDepth
			}
		}
		if depth > MaxConditionDepth {
			return invalid("%s visibility exceeds %d dependent levels", field.Label, MaxConditionDepth)
		}
		fieldDepth[field.ID] = depth
	}
	return nil
}

func normalizeCondition(condition *VisibilityCondition) (*VisibilityCondition, error) {
	if condition == nil {
		return nil, nil
	}
	copy := *condition
	copy.FieldID = strings.TrimSpace(copy.FieldID)
	copy.Operator = ConditionOperator(strings.ToUpper(strings.TrimSpace(string(copy.Operator))))
	copy.Values = append([]string(nil), copy.Values...)
	for index := range copy.Values {
		copy.Values[index] = strings.TrimSpace(copy.Values[index])
		if copy.Values[index] == "" {
			return nil, invalid("visibility comparison values cannot be empty")
		}
	}
	if copy.FieldID == "" || !slices.Contains([]ConditionOperator{ConditionEquals, ConditionNotEquals, ConditionIn, ConditionNotIn, ConditionAnswered}, copy.Operator) {
		return nil, invalid("visibility requires a field and supported operator")
	}
	switch copy.Operator {
	case ConditionAnswered:
		if len(copy.Values) != 0 {
			return nil, invalid("ANSWERED visibility cannot define comparison values")
		}
	case ConditionEquals, ConditionNotEquals:
		if len(copy.Values) != 1 {
			return nil, invalid("%s visibility requires one comparison value", copy.Operator)
		}
	case ConditionIn, ConditionNotIn:
		if len(copy.Values) < 1 || len(copy.Values) > MaxConditionValues {
			return nil, invalid("%s visibility requires 1-%d comparison values", copy.Operator, MaxConditionValues)
		}
	}
	return &copy, nil
}

func VisibleFields(contract Contract, answers map[string]AnswerValue) ([]Field, error) {
	normalized, err := Normalize(contract)
	if err != nil {
		return nil, err
	}
	sections := make(map[string]bool, len(normalized.Sections))
	for _, section := range normalized.Sections {
		sections[section.ID] = conditionMatches(section.Condition, answers)
	}
	visible := make([]Field, 0, len(normalized.Fields))
	for _, field := range normalized.Fields {
		if sections[field.SectionID] && conditionMatches(field.Condition, answers) {
			visible = append(visible, field)
		}
	}
	return visible, nil
}

func conditionMatches(condition *VisibilityCondition, answers map[string]AnswerValue) bool {
	if condition == nil {
		return true
	}
	answer, exists := answers[condition.FieldID]
	if condition.Operator == ConditionAnswered {
		return exists && answer.Answered()
	}
	actual := answerComparableValues(answer)
	match := false
	for _, value := range actual {
		if slices.Contains(condition.Values, value) {
			match = true
			break
		}
	}
	switch condition.Operator {
	case ConditionEquals, ConditionIn:
		return exists && match
	case ConditionNotEquals, ConditionNotIn:
		return !exists || !match
	default:
		panic(fmt.Sprintf("validated visibility operator %q was not handled", condition.Operator))
	}
}

func answerComparableValues(answer AnswerValue) []string {
	if text, exists := answer.ScalarText(); exists && text != "" {
		return []string{text}
	}
	values := make([]string, 0, len(answer.Values))
	for _, value := range answer.Values {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}
