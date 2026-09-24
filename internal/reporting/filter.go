package reporting

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

const (
	// A report filter is a bounded allow-list expression, never free-form SQL.
	maxReportFilterNodes = 12
	maxReportFilterDepth = 3
	maxReportFilterValue = 200
)

// ReportFilterField is a field a report may be filtered on. The set is closed:
// every entry maps to a bounded, server-owned column or child relation.
type ReportFilterField string

const (
	ReportFieldStatus          ReportFilterField = "status"
	ReportFieldLawfulBasis     ReportFilterField = "lawful_basis"
	ReportFieldOwner           ReportFilterField = "owner_principal_id"
	ReportFieldProgram         ReportFilterField = "program_id"
	ReportFieldMatter          ReportFilterField = "matter_id"
	ReportFieldAutomated       ReportFilterField = "automated_decision_making"
	ReportFieldCrossBorder     ReportFilterField = "cross_border_transfer"
	ReportFieldReviewOverdue   ReportFilterField = "review_overdue"
	ReportFieldMissingBasis    ReportFilterField = "missing_lawful_basis"
	ReportFieldMissingOwner    ReportFilterField = "missing_owner"
	ReportFieldMissingSubjects ReportFilterField = "missing_data_subjects"
	ReportFieldName            ReportFilterField = "name"
)

// ReportFilterFieldVocabulary is published to the web workspace so the builder
// can only offer filters the server will accept.
var ReportFilterFieldVocabulary = []ReportFilterFieldDefinition{
	{Field: ReportFieldStatus, Label: "Processing activity status", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldLawfulBasis, Label: "Lawful basis", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldOwner, Label: "Named owner", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldProgram, Label: "Related program", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldMatter, Label: "Related issue or change", Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldAutomated, Label: "Automated decision making", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldCrossBorder, Label: "Cross-border transfer", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldReviewOverdue, Label: "Review overdue", Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldMissingBasis, Label: "Lawful basis not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingOwner, Label: "Owner not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingSubjects, Label: "Data subject categories not recorded", Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldName, Label: "Activity name contains", Operators: []string{"contains"}, Indexed: false},
}

type ReportFilterFieldDefinition struct {
	Field     ReportFilterField `json:"field"`
	Label     string            `json:"label"`
	Operators []string          `json:"operators"`
	Indexed   bool              `json:"indexed"`
}

// ReportFilterExpression is a bounded condition tree. Kind is "condition" or
// "group"; a group joins children with and/or and holds no field or value.
type ReportFilterExpression struct {
	Kind     string                   `json:"kind"`
	Field    ReportFilterField        `json:"field,omitempty"`
	Operator string                   `json:"operator"`
	Value    string                   `json:"value,omitempty"`
	Children []ReportFilterExpression `json:"children,omitempty"`
}

func invalidReportFilter(format string, arguments ...any) error {
	return errors.Join(ErrInvalid, fmt.Errorf(format, arguments...))
}

// filterSQLFragment is the only place a field becomes a SQL fragment. A field
// absent from this switch can never reach a query, whatever the caller sends.
func filterSQLFragment(field ReportFilterField, operator string) (string, error) {
	switch field {
	case ReportFieldStatus:
		if operator != "is" {
			return "", fmt.Errorf("status supports only the is operator")
		}
		return "a.status = $%d", nil
	case ReportFieldLawfulBasis:
		if operator != "is" {
			return "", fmt.Errorf("lawful basis supports only the is operator")
		}
		return "a.lawful_basis = $%d", nil
	case ReportFieldOwner:
		switch operator {
		case "is":
			return "a.owner_principal_id = $%d::uuid", nil
		case "is_not":
			return "(a.owner_principal_id IS NULL OR a.owner_principal_id <> $%d::uuid)", nil
		}
	case ReportFieldProgram:
		switch operator {
		case "is":
			return "a.program_id = $%d::uuid", nil
		case "is_not":
			return "(a.program_id IS NULL OR a.program_id <> $%d::uuid)", nil
		}
	case ReportFieldMatter:
		switch operator {
		case "is":
			return "a.matter_id = $%d::uuid", nil
		case "is_not":
			return "(a.matter_id IS NULL OR a.matter_id <> $%d::uuid)", nil
		}
	case ReportFieldAutomated:
		if operator != "is" {
			return "", fmt.Errorf("automated decision making supports only the is operator")
		}
		return "a.automated_decision_making = $%d::boolean", nil
	case ReportFieldCrossBorder:
		if operator != "is" {
			return "", fmt.Errorf("cross-border transfer supports only the is operator")
		}
		return "EXISTS (SELECT 1 FROM ropa_processing_activity_recipients rc WHERE rc.tenant_id=a.tenant_id AND rc.legal_entity_id=a.legal_entity_id AND rc.activity_id=a.id AND rc.is_cross_border = $%d::boolean)", nil
	case ReportFieldReviewOverdue:
		if operator != "is" {
			return "", fmt.Errorf("review overdue supports only the is operator")
		}
		// The value is a boolean. Compare the stored review date with the
		// database's current date rather than binding a boolean as a timestamp.
		return "CASE WHEN $%d::boolean THEN (a.next_review_date IS NOT NULL AND a.next_review_date < CURRENT_DATE) ELSE (a.next_review_date IS NULL OR a.next_review_date >= CURRENT_DATE) END", nil
	case ReportFieldMissingBasis:
		if operator != "is" {
			return "", fmt.Errorf("missing lawful basis supports only the is operator")
		}
		return "(btrim(a.lawful_basis) = '' AND $%d::boolean)", nil
	case ReportFieldMissingOwner:
		if operator != "is" {
			return "", fmt.Errorf("missing owner supports only the is operator")
		}
		return "(a.owner_principal_id IS NULL AND $%d::boolean)", nil
	case ReportFieldMissingSubjects:
		if operator != "is" {
			return "", fmt.Errorf("missing data subject categories supports only the is operator")
		}
		return "(btrim(a.data_subject_categories) = '' AND $%d::boolean)", nil
	case ReportFieldName:
		if operator != "contains" {
			return "", fmt.Errorf("activity name supports only the contains operator")
		}
		return "a.name ILIKE '%' || $%d || '%'", nil
	}
	return "", fmt.Errorf("field %q is not available for report filtering", field)
}

// NormalizeReportFilter validates a filter against the allow-list before it can
// reach SQL. It returns a nil expression for an empty filter so an unfiltered
// report renders the whole scoped population rather than nothing.
func NormalizeReportFilter(expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	if expression == nil {
		return nil, nil
	}
	if isEmptyReportFilter(expression) {
		return nil, nil
	}
	nodes := 0
	normalized, err := normalizeReportFilterNode(*expression, 1, &nodes)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func isEmptyReportFilter(expression *ReportFilterExpression) bool {
	if expression == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(expression.Kind))
	operator := strings.ToLower(strings.TrimSpace(expression.Operator))
	return kind == "group" && len(expression.Children) == 0 && expression.Field == "" && expression.Value == "" &&
		(operator == "" || operator == "and" || operator == "or")
}

func normalizeReportFilterNode(expression ReportFilterExpression, depth int, nodes *int) (ReportFilterExpression, error) {
	(*nodes)++
	if *nodes > maxReportFilterNodes || depth > maxReportFilterDepth {
		return ReportFilterExpression{}, invalidReportFilter("report filters are limited to %d nodes and %d levels", maxReportFilterNodes, maxReportFilterDepth)
	}
	expression.Kind = strings.ToLower(strings.TrimSpace(expression.Kind))
	expression.Operator = strings.ToLower(strings.TrimSpace(expression.Operator))
	switch expression.Kind {
	case "condition":
		if len(expression.Children) != 0 {
			return ReportFilterExpression{}, invalidReportFilter("report filter conditions cannot contain children")
		}
		if _, err := filterSQLFragment(expression.Field, expression.Operator); err != nil {
			return ReportFilterExpression{}, errors.Join(ErrInvalid, err)
		}
		value, err := normalizeReportFilterValue(expression.Field, expression.Value)
		if err != nil {
			return ReportFilterExpression{}, err
		}
		expression.Value = value
		return expression, nil
	case "group":
		if expression.Field != "" || expression.Value != "" {
			return ReportFilterExpression{}, invalidReportFilter("report filter groups cannot carry a field or value")
		}
		if expression.Operator != "and" && expression.Operator != "or" {
			return ReportFilterExpression{}, invalidReportFilter("report filter groups join conditions with and or or")
		}
		if len(expression.Children) == 0 {
			return ReportFilterExpression{}, invalidReportFilter("report filter groups need at least one condition")
		}
		children := make([]ReportFilterExpression, 0, len(expression.Children))
		for _, child := range expression.Children {
			normalized, err := normalizeReportFilterNode(child, depth+1, nodes)
			if err != nil {
				return ReportFilterExpression{}, err
			}
			children = append(children, normalized)
		}
		expression.Children = children
		return expression, nil
	default:
		return ReportFilterExpression{}, invalidReportFilter("report filter nodes are conditions or groups")
	}
}

func normalizeReportFilterValue(field ReportFilterField, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalidReportFilter("report filter values cannot be empty")
	}
	if len([]rune(value)) > maxReportFilterValue {
		return "", invalidReportFilter("report filter values are limited to %d characters", maxReportFilterValue)
	}
	switch field {
	case ReportFieldStatus:
		value = strings.ToUpper(value)
		if !ropa.ValidStatus(ropa.Status(value)) {
			return "", invalidReportFilter("%q is not a recorded processing activity status", value)
		}
		return value, nil
	case ReportFieldAutomated, ReportFieldCrossBorder, ReportFieldReviewOverdue,
		ReportFieldMissingBasis, ReportFieldMissingOwner, ReportFieldMissingSubjects:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", invalidReportFilter("%s is recorded as true or false", field)
		}
		return strconv.FormatBool(parsed), nil
	case ReportFieldOwner, ReportFieldProgram, ReportFieldMatter:
		if !isUUID(value) {
			return "", invalidReportFilter("%s must be a recorded identifier", field)
		}
		return strings.ToLower(value), nil
	case ReportFieldLawfulBasis, ReportFieldName:
		return value, nil
	default:
		return "", invalidReportFilter("field %q is not available for report filtering", field)
	}
}

// ReportFilterSQL renders the validated expression into a parameterised
// fragment starting at the given position. The returned arguments are in
// fragment order, so the caller appends them directly to its query arguments.
func ReportFilterSQL(expression *ReportFilterExpression, nextPosition int) (string, []any, error) {
	if nextPosition < 1 {
		return "", nil, invalidReportFilter("report filter SQL parameter positions start at one")
	}
	normalized, err := NormalizeReportFilter(expression)
	if err != nil {
		return "", nil, err
	}
	if normalized == nil {
		return "TRUE", nil, nil
	}
	args := make([]any, 0, maxReportFilterNodes)
	position := nextPosition
	fragment, err := renderReportFilterNode(normalized, &position, &args)
	if err != nil {
		return "", nil, err
	}
	return fragment, args, nil
}

// renderReportFilterNode advances position for every bound value, so the first
// value is $nextPosition and each subsequent one is the next number after it.
func renderReportFilterNode(expression *ReportFilterExpression, position *int, args *[]any) (string, error) {
	if expression == nil {
		return "TRUE", nil
	}
	switch strings.ToLower(strings.TrimSpace(expression.Kind)) {
	case "condition":
		fragment, err := filterSQLFragment(expression.Field, strings.ToLower(strings.TrimSpace(expression.Operator)))
		if err != nil {
			return "", errors.Join(ErrInvalid, err)
		}
		*args = append(*args, expression.Value)
		bound := *position
		(*position)++
		// Replace only the closed placeholder token. Calling fmt.Sprintf on the
		// fragment would interpret the SQL wildcard '%' in an ILIKE predicate.
		return strings.ReplaceAll(fragment, "%d", strconv.Itoa(bound)), nil
	case "group":
		parts := make([]string, 0, len(expression.Children))
		for index := range expression.Children {
			part, err := renderReportFilterNode(&expression.Children[index], position, args)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		joiner := " AND "
		if strings.EqualFold(strings.TrimSpace(expression.Operator), "or") {
			joiner = " OR "
		}
		return "(" + strings.Join(parts, joiner) + ")", nil
	default:
		return "", invalidReportFilter("report filter nodes are conditions or groups")
	}
}

func isUUID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 36 {
		return false
	}
	for index, character := range trimmed {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
				return false
			}
		}
	}
	return true
}
