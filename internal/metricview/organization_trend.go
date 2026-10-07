package metricview

import (
	"context"
	"time"
)

type OrganizationTrendPoint struct {
	Date           string    `json:"date"`
	At             time.Time `json:"at"`
	Value          int       `json:"value"`
	SourceRevision string    `json:"source_revision"`
	SourceComplete bool      `json:"source_complete"`
}

type OrganizationTrendSeries struct {
	MetricID            string                   `json:"metric_id"`
	DefinitionRevision  string                   `json:"definition_revision"`
	OrganizationScopeID string                   `json:"organization_scope_id"`
	Start               time.Time                `json:"start"`
	End                 time.Time                `json:"end"`
	Resolution          TrendResolution          `json:"resolution"`
	Points              []OrganizationTrendPoint `json:"points"`
}

type OrganizationTrendReader interface {
	OrganizationTrend(
		context.Context,
		string,
		string,
		string,
		string,
		time.Time,
		time.Time,
	) (OrganizationTrendSeries, error)
}
