package reporting

import (
	"fmt"
	"strings"
	"time"
)

const (
	reportDateLayout             = "2006-01-02"
	maxReportRunParameterArgs    = 3
	maxReportOwnerPrincipalRunes = 200
)

// ReportRunParameters are execution-time constraints. They are captured on the
// immutable run receipt rather than changing the governed saved setup.
type ReportRunParameters struct {
	StartDate        string `json:"start_date,omitempty"`
	EndDate          string `json:"end_date,omitempty"`
	OwnerPrincipalID string `json:"owner_principal_id,omitempty"`
}

func NormalizeReportRunParameters(value ReportRunParameters) (ReportRunParameters, error) {
	value.StartDate = strings.TrimSpace(value.StartDate)
	value.EndDate = strings.TrimSpace(value.EndDate)
	value.OwnerPrincipalID = strings.TrimSpace(value.OwnerPrincipalID)

	var start, end time.Time
	var err error
	if value.StartDate != "" {
		start, err = time.Parse(reportDateLayout, value.StartDate)
		if err != nil {
			return ReportRunParameters{}, invalidReportFilter("report start date must use YYYY-MM-DD")
		}
	}
	if value.EndDate != "" {
		end, err = time.Parse(reportDateLayout, value.EndDate)
		if err != nil {
			return ReportRunParameters{}, invalidReportFilter("report end date must use YYYY-MM-DD")
		}
	}
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return ReportRunParameters{}, invalidReportFilter("report start date must not be after end date")
	}
	if len([]rune(value.OwnerPrincipalID)) > maxReportOwnerPrincipalRunes {
		return ReportRunParameters{}, invalidReportFilter("report owner reference is too long")
	}
	return value, nil
}

// ReportRunParametersSQL applies the run-time period to record creation time and
// the owner constraint to the exact assigned principal. Callers append the
// returned arguments before the governed setup-filter arguments.
func ReportRunParametersSQL(value ReportRunParameters, startIndex int) (string, []any, error) {
	if startIndex < 1 {
		return "", nil, ErrInvalid
	}
	normalized, err := NormalizeReportRunParameters(value)
	if err != nil {
		return "", nil, err
	}
	parts := []string{"TRUE"}
	args := make([]any, 0, maxReportRunParameterArgs)
	next := startIndex
	if normalized.StartDate != "" {
		parts = append(parts, fmt.Sprintf("a.created_at >= $%d::date", next))
		args = append(args, normalized.StartDate)
		next++
	}
	if normalized.EndDate != "" {
		parts = append(parts, fmt.Sprintf("a.created_at < ($%d::date + INTERVAL '1 day')", next))
		args = append(args, normalized.EndDate)
		next++
	}
	if normalized.OwnerPrincipalID != "" {
		parts = append(parts, fmt.Sprintf("COALESCE(a.owner_principal_id::text,'') = $%d", next))
		args = append(args, normalized.OwnerPrincipalID)
	}
	return strings.Join(parts, " AND "), args, nil
}

func combineReportRunFilter(parameters ReportRunParameters, dataset ReportDataset, filter *ReportFilterExpression, startIndex int) (string, []any, error) {
	parameterSQL, parameterArgs, err := ReportRunParametersSQL(parameters, startIndex)
	if err != nil {
		return "", nil, err
	}
	filterSQL, filterArgs, err := ReportFilterSQLForDataset(dataset, filter, startIndex+len(parameterArgs))
	if err != nil {
		return "", nil, err
	}
	args := append(parameterArgs, filterArgs...)
	return "(" + parameterSQL + ") AND (" + filterSQL + ")", args, nil
}
