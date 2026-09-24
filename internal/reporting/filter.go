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
	ReportFieldStatus                   ReportFilterField = "status"
	ReportFieldLawfulBasis              ReportFilterField = "lawful_basis"
	ReportFieldOwner                    ReportFilterField = "owner_principal_id"
	ReportFieldProgram                  ReportFilterField = "program_id"
	ReportFieldMatter                   ReportFilterField = "matter_id"
	ReportFieldAutomated                ReportFilterField = "automated_decision_making"
	ReportFieldCrossBorder              ReportFilterField = "cross_border_transfer"
	ReportFieldReviewOverdue            ReportFilterField = "review_overdue"
	ReportFieldMissingBasis             ReportFilterField = "missing_lawful_basis"
	ReportFieldMissingOwner             ReportFilterField = "missing_owner"
	ReportFieldMissingSubjects          ReportFilterField = "missing_data_subjects"
	ReportFieldName                     ReportFilterField = "name"
	ReportFieldOverallState             ReportFilterField = "overall_state"
	ReportFieldHasOpenMatters           ReportFilterField = "has_open_matters"
	ReportFieldJurisdiction             ReportFilterField = "jurisdiction"
	ReportFieldMatterType               ReportFilterField = "matter_type"
	ReportFieldPriority                 ReportFilterField = "priority"
	ReportFieldDueCondition             ReportFilterField = "due_condition"
	ReportFieldMatterProgram            ReportFilterField = "program"
	ReportFieldLatestVerificationResult ReportFilterField = "latest_verification_result"
)

// ReportFilterFieldVocabulary is published to the web workspace so the builder
// can only offer filters the server will accept.
var ReportFilterFieldVocabulary = []ReportFilterFieldDefinition{
	{Field: ReportFieldStatus, Label: "Processing activity status", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldLawfulBasis, Label: "Lawful basis", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldOwner, Label: "Named owner", Dataset: DatasetProcessingActivities, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldProgram, Label: "Related program", Dataset: DatasetProcessingActivities, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldMatter, Label: "Related issue or change", Dataset: DatasetProcessingActivities, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldAutomated, Label: "Automated decision making", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldCrossBorder, Label: "Cross-border transfer", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldReviewOverdue, Label: "Review overdue", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldMissingBasis, Label: "Lawful basis not recorded", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingOwner, Label: "Owner not recorded", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldMissingSubjects, Label: "Data subject categories not recorded", Dataset: DatasetProcessingActivities, Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldName, Label: "Activity name contains", Dataset: DatasetProcessingActivities, Operators: []string{"contains"}, Indexed: false},

	{Field: ReportFieldStatus, Label: "Program status", Dataset: DatasetPrograms, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldOwner, Label: "Program owner", Dataset: DatasetPrograms, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldOverallState, Label: "Calculated Program state", Dataset: DatasetPrograms, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldJurisdiction, Label: "Program jurisdiction", Dataset: DatasetPrograms, Operators: []string{"is"}, Indexed: false},
	{Field: ReportFieldHasOpenMatters, Label: "Open issues and changes", Dataset: DatasetPrograms, Operators: []string{"is"}, Indexed: true},

	{Field: ReportFieldStatus, Label: "Issue or change status", Dataset: DatasetMatterExceptions, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldOwner, Label: "Issue or change owner", Dataset: DatasetMatterExceptions, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldMatterType, Label: "Issue or change type", Dataset: DatasetMatterExceptions, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldPriority, Label: "Priority", Dataset: DatasetMatterExceptions, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldDueCondition, Label: "Due condition", Dataset: DatasetMatterExceptions, Operators: []string{"is"}, Indexed: true},
	{Field: ReportFieldMatterProgram, Label: "Related Program", Dataset: DatasetMatterExceptions, Operators: []string{"is", "is_not"}, Indexed: true},
	{Field: ReportFieldLatestVerificationResult, Label: "Latest outcome result", Dataset: DatasetMatterExceptions, Operators: []string{"is"}, Indexed: true},
}

type ReportFilterFieldDefinition struct {
	Field     ReportFilterField `json:"field"`
	Label     string            `json:"label"`
	Dataset   ReportDataset     `json:"dataset"`
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
	case ReportFieldOverallState:
		if operator != "is" {
			return "", fmt.Errorf("overall state supports only the is operator")
		}
		return "a.overall_state = $%d", nil
	case ReportFieldHasOpenMatters:
		if operator != "is" {
			return "", fmt.Errorf("open issue count supports only the is operator")
		}
		return "a.has_open_matters = $%d::boolean", nil
	case ReportFieldJurisdiction:
		if operator != "is" {
			return "", fmt.Errorf("jurisdiction supports only the is operator")
		}
		return "lower(btrim(a.jurisdiction)) = lower(btrim($%d))", nil
	case ReportFieldMatterType:
		if operator != "is" {
			return "", fmt.Errorf("matter type supports only the is operator")
		}
		return "a.matter_type = $%d", nil
	case ReportFieldPriority:
		if operator != "is" {
			return "", fmt.Errorf("priority supports only the is operator")
		}
		return "a.priority = $%d::integer", nil
	case ReportFieldDueCondition:
		if operator != "is" {
			return "", fmt.Errorf("due condition supports only the is operator")
		}
		return "CASE $%d WHEN 'NO_DUE_DATE' THEN a.due_at IS NULL WHEN 'OVERDUE' THEN a.due_at<$5::timestamptz AND a.status NOT IN ('CLOSED','CANCELLED') WHEN 'DUE_7_DAYS' THEN a.due_at>=$5::timestamptz AND a.due_at<=$5::timestamptz+interval '7 days' AND a.status NOT IN ('CLOSED','CANCELLED') WHEN 'DUE_30_DAYS' THEN a.due_at>=$5::timestamptz AND a.due_at<=$5::timestamptz+interval '30 days' AND a.status NOT IN ('CLOSED','CANCELLED') ELSE false END", nil
	case ReportFieldMatterProgram:
		switch operator {
		case "is":
			return "EXISTS (SELECT 1 FROM matter_links ml JOIN programs linked_program ON linked_program.id=ml.program_id AND linked_program.tenant_id=ml.tenant_id AND linked_program.legal_entity_id=$2::uuid WHERE ml.tenant_id=a.tenant_id::uuid AND ml.matter_id=a.id::uuid AND ml.program_id=$%d::uuid AND ml.retired_at IS NULL)", nil
		case "is_not":
			return "(NOT EXISTS (SELECT 1 FROM matter_links ml WHERE ml.tenant_id=a.tenant_id AND ml.matter_id=a.id AND ml.program_id=$%d::uuid AND ml.retired_at IS NULL))", nil
		}
	case ReportFieldLatestVerificationResult:
		if operator != "is" {
			return "", fmt.Errorf("latest verification result supports only the is operator")
		}
		return "a.latest_verification_result = $%d", nil
	}
	return "", fmt.Errorf("field %q is not available for report filtering", field)
}

// NormalizeReportFilter validates a filter against the union allow-list kept
// for callers that only need to inspect a filter. Report definitions and
// persisted runs use NormalizeReportFilterForDataset so a field belonging to
// another dataset cannot be silently accepted.
func NormalizeReportFilter(expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	return normalizeReportFilterForDataset("", expression)
}

// NormalizeReportFilterForDataset validates both the closed field set and the
// closed values for one report dataset. An empty filter remains empty.
func NormalizeReportFilterForDataset(dataset ReportDataset, expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	if dataset != "" && !validReportDataset(dataset) {
		return nil, invalidReportFilter("unknown report dataset %q", dataset)
	}
	return normalizeReportFilterForDataset(dataset, expression)
}

func normalizeReportFilterForDataset(dataset ReportDataset, expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	if expression == nil {
		return nil, nil
	}
	if isEmptyReportFilter(expression) {
		return nil, nil
	}
	nodes := 0
	normalized, err := normalizeReportFilterNode(dataset, *expression, 1, &nodes)
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

func normalizeReportFilterNode(dataset ReportDataset, expression ReportFilterExpression, depth int, nodes *int) (ReportFilterExpression, error) {
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
		if !reportFieldAllowedForDataset(dataset, expression.Field) {
			return ReportFilterExpression{}, invalidReportFilter("field %q is not available for dataset %q", expression.Field, dataset)
		}
		if _, err := filterSQLFragment(expression.Field, expression.Operator); err != nil {
			return ReportFilterExpression{}, errors.Join(ErrInvalid, err)
		}
		value, err := normalizeReportFilterValue(dataset, expression.Field, expression.Value)
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
			normalized, err := normalizeReportFilterNode(dataset, child, depth+1, nodes)
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

func normalizeReportFilterValue(dataset ReportDataset, field ReportFilterField, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalidReportFilter("report filter values cannot be empty")
	}
	if len([]rune(value)) > maxReportFilterValue {
		return "", invalidReportFilter("report filter values are limited to %d characters", maxReportFilterValue)
	}
	if strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return "", invalidReportFilter("report filter values cannot contain control characters")
	}
	switch field {
	case ReportFieldStatus:
		value = strings.ToUpper(value)
		if dataset == DatasetPrograms {
			if !oneOfReportFilterValue(value, "DRAFT", "ACTIVE", "PAUSED", "RETIRED") {
				return "", invalidReportFilter("%q is not a recorded Program status", value)
			}
			return value, nil
		}
		if dataset == DatasetMatterExceptions {
			if !oneOfReportFilterValue(value, "DRAFT", "TRIAGE", "ASSESSMENT", "DECISION_REQUIRED", "ACTION_IN_PROGRESS", "RESPONSE_PREPARATION", "VERIFICATION", "CLOSED", "CANCELLED") {
				return "", invalidReportFilter("%q is not a recorded Matter status", value)
			}
			return value, nil
		}
		if !ropa.ValidStatus(ropa.Status(value)) {
			return "", invalidReportFilter("%q is not a recorded processing activity status", value)
		}
		return value, nil
	case ReportFieldOverallState:
		value = strings.ToUpper(value)
		if !oneOfReportFilterValue(value, "CURRENT", "AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE", "UNDER_REVIEW", "NOT_APPLICABLE", "UNKNOWN") {
			return "", invalidReportFilter("%q is not a recorded calculated Program state", value)
		}
		return value, nil
	case ReportFieldHasOpenMatters, ReportFieldAutomated, ReportFieldCrossBorder, ReportFieldReviewOverdue,
		ReportFieldMissingBasis, ReportFieldMissingOwner, ReportFieldMissingSubjects:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", invalidReportFilter("%s is recorded as true or false", field)
		}
		return strconv.FormatBool(parsed), nil
	case ReportFieldOwner, ReportFieldProgram, ReportFieldMatter, ReportFieldMatterProgram:
		if !isUUID(value) {
			return "", invalidReportFilter("%s must be a recorded identifier", field)
		}
		return strings.ToLower(value), nil
	case ReportFieldPriority:
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 5 {
			return "", invalidReportFilter("priority must be a recorded value from 1 through 5")
		}
		return strconv.Itoa(parsed), nil
	case ReportFieldMatterType:
		value = strings.ToUpper(value)
		if !oneOfReportFilterValue(value,
			"REGULATORY_CHANGE", "SUPERVISORY_FINDING", "AUTHORITY_REQUEST", "RISK_SITUATION", "CONTROL_GAP",
			"AUDIT_FINDING", "EXCEPTION", "INCIDENT", "OPERATIONAL_LOSS", "DATA_BREACH", "VENDOR_DEFICIENCY",
			"CUSTOMER_CONCERN", "VENDOR_REVIEW", "OVERDUE_OBLIGATION", "FAILED_VERIFICATION", "EVIDENCE_CONTRADICTION", "KRI_BREACH") {
			return "", invalidReportFilter("%q is not a recorded Matter type", value)
		}
		return value, nil
	case ReportFieldDueCondition:
		value = strings.ToUpper(value)
		if !oneOfReportFilterValue(value, "NO_DUE_DATE", "OVERDUE", "DUE_7_DAYS", "DUE_30_DAYS") {
			return "", invalidReportFilter("%q is not a recorded due condition", value)
		}
		return value, nil
	case ReportFieldLatestVerificationResult:
		value = strings.ToUpper(value)
		if !oneOfReportFilterValue(value, "PASS", "FAIL", "INCONCLUSIVE") {
			return "", invalidReportFilter("%q is not a recorded verification result", value)
		}
		return value, nil
	case ReportFieldLawfulBasis, ReportFieldJurisdiction, ReportFieldName:
		return value, nil
	default:
		return "", invalidReportFilter("field %q is not available for report filtering", field)
	}
}

func reportFieldAllowedForDataset(dataset ReportDataset, field ReportFilterField) bool {
	if dataset == "" {
		return true
	}
	switch dataset {
	case DatasetProcessingActivities, DatasetProcessingActivityExceptions:
		switch field {
		case ReportFieldStatus, ReportFieldLawfulBasis, ReportFieldOwner, ReportFieldProgram, ReportFieldMatter,
			ReportFieldAutomated, ReportFieldCrossBorder, ReportFieldReviewOverdue, ReportFieldMissingBasis,
			ReportFieldMissingOwner, ReportFieldMissingSubjects, ReportFieldName:
			return true
		}
	case DatasetPrograms:
		switch field {
		case ReportFieldStatus, ReportFieldOwner, ReportFieldOverallState, ReportFieldJurisdiction, ReportFieldHasOpenMatters:
			return true
		}
	case DatasetMatterExceptions:
		switch field {
		case ReportFieldStatus, ReportFieldOwner, ReportFieldMatterType, ReportFieldPriority, ReportFieldDueCondition,
			ReportFieldMatterProgram, ReportFieldLatestVerificationResult:
			return true
		}
	}
	return false
}

func oneOfReportFilterValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

// ReportFilterSQL renders the validated expression into a parameterised
// fragment starting at the given position. The returned arguments are in
// fragment order, so the caller appends them directly to its query arguments.
func ReportFilterSQL(expression *ReportFilterExpression, nextPosition int) (string, []any, error) {
	return reportFilterSQLForDataset("", expression, nextPosition)
}

// ReportFilterSQLForDataset is the report query boundary. It repeats the
// dataset check at SQL-generation time so a persisted filter cannot be routed
// to the wrong query even if normalisation was skipped by a caller.
func ReportFilterSQLForDataset(dataset ReportDataset, expression *ReportFilterExpression, nextPosition int) (string, []any, error) {
	return reportFilterSQLForDataset(dataset, expression, nextPosition)
}

func reportFilterSQLForDataset(dataset ReportDataset, expression *ReportFilterExpression, nextPosition int) (string, []any, error) {
	if nextPosition < 1 {
		return "", nil, invalidReportFilter("report filter SQL parameter positions start at one")
	}
	normalized, err := NormalizeReportFilterForDataset(dataset, expression)
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
