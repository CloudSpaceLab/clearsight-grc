package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	maxFormFilterNodes = 12
	maxFormFilterDepth = 3
)

type FormFilterField string

type FormFilterExpression struct {
	Kind     string                 `json:"kind"`
	Field    FormFilterField        `json:"field,omitempty"`
	Operator string                 `json:"operator"`
	Value    string                 `json:"value,omitempty"`
	Children []FormFilterExpression `json:"children,omitempty"`
}

type FormLibraryFacets struct {
	Status map[LifecycleStatus]int `json:"status,omitempty"`
}

const (
	FormFilterStatus  FormFilterField = "status"
	FormFilterOwner   FormFilterField = "owner"
	FormFilterProgram FormFilterField = "program"
	FormFilterUse     FormFilterField = "use"
	FormFilterTag     FormFilterField = "tag"
)

type advancedFormLibraryRepository interface {
	ListAdvancedFormLibrary(context.Context, FormLibraryFilter, bool) (FormTemplatePage, error)
}

func (s *Service) ListAdvancedFormLibrary(ctx context.Context, filter FormLibraryFilter, includeStatusFacets bool) (FormTemplatePage, error) {
	actor, err := s.requireFormActor(ctx)
	if err != nil {
		return FormTemplatePage{}, err
	}
	normalized, err := NormalizeFormFilterExpression(filter.Expression)
	if err != nil {
		return FormTemplatePage{}, err
	}
	filter.Expression = normalized
	filter.TenantID = actor.TenantID
	filter.LegalEntityID = actor.LegalEntityID
	repo, ok := s.repo.(advancedFormLibraryRepository)
	if !ok {
		return FormTemplatePage{}, errors.Join(ErrInvalid, fmt.Errorf("advanced form filtering is unavailable"))
	}
	return repo.ListAdvancedFormLibrary(ctx, filter, includeStatusFacets)
}

func NormalizeFormFilterExpression(expression *FormFilterExpression) (*FormFilterExpression, error) {
	if expression == nil {
		return nil, nil
	}
	nodes := 0
	normalized, err := normalizeFormFilterNode(*expression, 1, &nodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func normalizeFormFilterNode(expression FormFilterExpression, depth int, nodes *int) (FormFilterExpression, error) {
	(*nodes)++
	if *nodes > maxFormFilterNodes || depth > maxFormFilterDepth {
		return FormFilterExpression{}, errors.Join(ErrInvalid, fmt.Errorf("form filter expressions are limited to %d nodes and %d levels", maxFormFilterNodes, maxFormFilterDepth))
	}
	expression.Kind = strings.ToLower(strings.TrimSpace(expression.Kind))
	expression.Operator = strings.ToLower(strings.TrimSpace(expression.Operator))
	switch expression.Kind {
	case "condition":
		if len(expression.Children) != 0 || expression.Operator != "is" {
			return FormFilterExpression{}, errors.Join(ErrInvalid, fmt.Errorf("form filter conditions must use the is operator"))
		}
		expression.Value = strings.TrimSpace(expression.Value)
		if len(expression.Value) > 200 {
			return FormFilterExpression{}, errors.Join(ErrInvalid, fmt.Errorf("form filter values are limited to 200 characters"))
		}
		normalizedValue, err := normalizeFormFilterValue(expression.Field, expression.Value)
		if err != nil {
			return FormFilterExpression{}, err
		}
		expression.Value = normalizedValue
		return expression, nil
	case "group":
		if expression.Field != "" || expression.Value != "" || (expression.Operator != "and" && expression.Operator != "or") || len(expression.Children) == 0 {
			return FormFilterExpression{}, errors.Join(ErrInvalid, fmt.Errorf("form filter groups must contain conditions joined by and/or"))
		}
		children := make([]FormFilterExpression, 0, len(expression.Children))
		for _, child := range expression.Children {
			normalized, err := normalizeFormFilterNode(child, depth+1, nodes)
			if err != nil {
				return FormFilterExpression{}, err
			}
			children = append(children, normalized)
		}
		expression.Children = children
		return expression, nil
	default:
		return FormFilterExpression{}, errors.Join(ErrInvalid, fmt.Errorf("form filter nodes must be conditions or groups"))
	}
}

func normalizeFormFilterValue(field FormFilterField, value string) (string, error) {
	if value == "" {
		return "", errors.Join(ErrInvalid, fmt.Errorf("form filter values cannot be empty"))
	}
	switch field {
	case FormFilterStatus:
		value = strings.ToUpper(value)
		switch LifecycleStatus(value) {
		case LifecycleDraft, LifecyclePendingApproval, LifecycleActive, LifecyclePaused, LifecycleRejected, LifecycleRetired:
			return value, nil
		default:
			return "", errors.Join(ErrInvalid, fmt.Errorf("form status filter is invalid"))
		}
	case FormFilterUse:
		return strings.ToUpper(value), nil
	case FormFilterTag:
		return strings.ToLower(value), nil
	case FormFilterOwner, FormFilterProgram:
		return value, nil
	default:
		return "", errors.Join(ErrInvalid, fmt.Errorf("form filter field is not supported"))
	}
}

func combinedFormFilterExpression(filter FormLibraryFilter) (*FormFilterExpression, error) {
	children := make([]FormFilterExpression, 0, 6)
	appendCondition := func(field FormFilterField, value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		normalized, err := normalizeFormFilterValue(field, value)
		if err != nil {
			return err
		}
		children = append(children, FormFilterExpression{Kind: "condition", Field: field, Operator: "is", Value: normalized})
		return nil
	}
	if err := appendCondition(FormFilterStatus, string(filter.Status)); err != nil {
		return nil, err
	}
	if err := appendCondition(FormFilterOwner, filter.OwnerPrincipalID); err != nil {
		return nil, err
	}
	if err := appendCondition(FormFilterProgram, filter.ProgramID); err != nil {
		return nil, err
	}
	if err := appendCondition(FormFilterUse, filter.Use); err != nil {
		return nil, err
	}
	if err := appendCondition(FormFilterTag, filter.Tag); err != nil {
		return nil, err
	}
	if filter.Expression != nil {
		normalized, err := NormalizeFormFilterExpression(filter.Expression)
		if err != nil {
			return nil, err
		}
		if normalized != nil {
			children = append(children, *normalized)
		}
	}
	if len(children) == 0 {
		return nil, nil
	}
	if len(children) == 1 {
		value := children[0]
		return NormalizeFormFilterExpression(&value)
	}
	combined := FormFilterExpression{Kind: "group", Operator: "and", Children: children}
	return NormalizeFormFilterExpression(&combined)
}

func formFilterExpressionMatches(expression *FormFilterExpression, value FormTemplate) bool {
	if expression == nil {
		return true
	}
	if expression.Kind == "condition" {
		switch expression.Field {
		case FormFilterStatus:
			return value.Status == LifecycleStatus(expression.Value)
		case FormFilterOwner:
			return value.OwnerPrincipalID == expression.Value
		case FormFilterProgram:
			return value.ProgramID == expression.Value
		case FormFilterUse:
			return containsFold(value.ApprovedUses, expression.Value)
		case FormFilterTag:
			return containsFold(value.Tags, expression.Value)
		default:
			return false
		}
	}
	if expression.Operator == "and" {
		for index := range expression.Children {
			if !formFilterExpressionMatches(&expression.Children[index], value) {
				return false
			}
		}
		return true
	}
	for index := range expression.Children {
		if formFilterExpressionMatches(&expression.Children[index], value) {
			return true
		}
	}
	return false
}

func formFilterExpressionWithoutField(expression *FormFilterExpression, field FormFilterField) *FormFilterExpression {
	if expression == nil {
		return nil
	}
	if expression.Kind == "condition" {
		if expression.Field == field {
			return nil
		}
		cloned := *expression
		return &cloned
	}
	children := make([]FormFilterExpression, 0, len(expression.Children))
	for index := range expression.Children {
		child := formFilterExpressionWithoutField(&expression.Children[index], field)
		if child == nil {
			if expression.Operator == "or" {
				return nil
			}
			continue
		}
		children = append(children, *child)
	}
	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		child := children[0]
		return &child
	}
	return &FormFilterExpression{Kind: "group", Operator: expression.Operator, Children: children}
}

func cloneFormFilterExpression(expression *FormFilterExpression) *FormFilterExpression {
	if expression == nil {
		return nil
	}
	cloned := *expression
	if expression.Children != nil {
		cloned.Children = make([]FormFilterExpression, len(expression.Children))
		for index := range expression.Children {
			child := cloneFormFilterExpression(&expression.Children[index])
			cloned.Children[index] = *child
		}
	}
	return &cloned
}
