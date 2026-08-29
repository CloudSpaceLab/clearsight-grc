package thirdparty

import (
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type AssessmentScopeKind string

const (
	AssessmentScopeFull    AssessmentScopeKind = "FULL"
	AssessmentScopeFocused AssessmentScopeKind = "FOCUSED"
)

var ErrInvalidAssessmentScope = errors.New("assessment field scope is invalid or stale")

func NormalizeAssessmentScope(kind AssessmentScopeKind, selected []string) (AssessmentScopeKind, []string, error) {
	kind = AssessmentScopeKind(strings.ToUpper(strings.TrimSpace(string(kind))))
	if kind == "" {
		kind = AssessmentScopeFull
	}
	if kind != AssessmentScopeFull && kind != AssessmentScopeFocused {
		return "", nil, ErrInvalidAssessmentScope
	}
	if kind == AssessmentScopeFull {
		if len(selected) != 0 {
			return "", nil, ErrInvalidAssessmentScope
		}
		return kind, []string{}, nil
	}
	if len(selected) == 0 || len(selected) > formcontract.MaxFields {
		return "", nil, ErrInvalidAssessmentScope
	}
	seen := make(map[string]struct{}, len(selected))
	result := make([]string, 0, len(selected))
	for _, id := range selected {
		id = strings.TrimSpace(id)
		if id == "" {
			return "", nil, ErrInvalidAssessmentScope
		}
		if _, exists := seen[id]; exists {
			return "", nil, ErrInvalidAssessmentScope
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return kind, result, nil
}

func ComposeAssessmentScope(contract formcontract.Contract, kind AssessmentScopeKind, selected []string) ([]formcontract.Section, []formcontract.Field, error) {
	kind, selected, err := NormalizeAssessmentScope(kind, selected)
	if err != nil {
		return nil, nil, err
	}
	if kind == AssessmentScopeFull {
		return append([]formcontract.Section(nil), contract.Sections...), append([]formcontract.Field(nil), contract.Fields...), nil
	}
	byID := make(map[string]formcontract.Field, len(contract.Fields))
	include := make(map[string]bool, len(selected))
	for _, field := range contract.Fields {
		byID[field.ID] = field
	}
	for _, id := range selected {
		if _, exists := byID[id]; !exists {
			return nil, nil, ErrInvalidAssessmentScope
		}
		include[id] = true
	}
	for changed := true; changed; {
		changed = false
		for id := range include {
			field := byID[id]
			if field.Condition != nil && !include[field.Condition.FieldID] {
				if _, exists := byID[field.Condition.FieldID]; !exists {
					return nil, nil, ErrInvalidAssessmentScope
				}
				include[field.Condition.FieldID] = true
				changed = true
			}
		}
	}
	sectionIDs := map[string]bool{}
	fields := make([]formcontract.Field, 0, len(include))
	for _, field := range contract.Fields {
		if include[field.ID] {
			fields = append(fields, field)
			sectionIDs[field.SectionID] = true
		}
	}
	sections := make([]formcontract.Section, 0, len(sectionIDs))
	for _, section := range contract.Sections {
		if sectionIDs[section.ID] {
			sections = append(sections, section)
		}
	}
	if len(fields) == 0 || len(sections) == 0 {
		return nil, nil, ErrInvalidAssessmentScope
	}
	return sections, fields, nil
}
