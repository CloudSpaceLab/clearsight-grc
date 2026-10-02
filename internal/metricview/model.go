package metricview

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

const HomeDefinitionRevision = "home-oversight-v2"

type MetricBasis string

const MetricBasisCurrentPosture MetricBasis = "CURRENT_POSTURE"

type Completeness string

const (
	CompletenessComplete Completeness = "COMPLETE"
	CompletenessPartial  Completeness = "PARTIAL"
	CompletenessUnknown  Completeness = "UNKNOWN"
)

type Condition string

const (
	ConditionClear     Condition = "CLEAR"
	ConditionAttention Condition = "ATTENTION"
)

type DrillConsistency string

const (
	// DrillCurrentState means the target resolves against the current
	// authorized population. The UI must not imply that a later drill result is
	// the immutable population that produced an older card.
	DrillCurrentState DrillConsistency = "CURRENT_STATE"
)

type DrillTarget struct {
	Workspace   string           `json:"workspace"`
	Filter      string           `json:"filter"`
	Consistency DrillConsistency `json:"consistency"`
}

type Metric struct {
	ID                 string              `json:"id"`
	Label              string              `json:"label"`
	Value              int                 `json:"value"`
	Unit               string              `json:"unit"`
	Condition          Condition           `json:"condition"`
	Freshness          oversight.Freshness `json:"freshness"`
	Completeness       Completeness        `json:"completeness"`
	Population         int                 `json:"population"`
	Excluded           *int                `json:"excluded,omitempty"`
	Unknown            *int                `json:"unknown,omitempty"`
	GeneratedAt        time.Time           `json:"generated_at"`
	SourceRevision     string              `json:"source_revision"`
	DefinitionRevision string              `json:"definition_revision"`
	Basis              MetricBasis         `json:"basis"`
	Drill              DrillTarget         `json:"drill"`
}

type Bundle struct {
	GeneratedAt        time.Time                 `json:"generated_at"`
	PeriodStart        time.Time                 `json:"period_start"`
	PeriodEnd          time.Time                 `json:"period_end"`
	ReportingPeriod    oversight.ReportingPeriod `json:"reporting_period"`
	PostureAsOf        time.Time                 `json:"posture_as_of"`
	ScopeID            string                    `json:"scope_id"`
	ScopeKind          string                    `json:"scope_kind"`
	Freshness          oversight.Freshness       `json:"freshness"`
	Completeness       Completeness              `json:"completeness"`
	Population         int                       `json:"population"`
	Excluded           *int                      `json:"excluded,omitempty"`
	Unknown            *int                      `json:"unknown,omitempty"`
	SourceRevision     string                    `json:"source_revision"`
	DefinitionRevision string                    `json:"definition_revision"`
	Items              []Metric                  `json:"items"`
}

func FromOversight(snapshot oversight.Snapshot) Bundle {
	completeness := coverageCompleteness(snapshot.Coverage)
	common := func(id, label, filter string, value int) Metric {
		condition := ConditionClear
		if value > 0 {
			condition = ConditionAttention
		}
		return Metric{
			ID:                 id,
			Label:              label,
			Value:              value,
			Unit:               "COUNT",
			Condition:          condition,
			Freshness:          snapshot.Freshness,
			Completeness:       completeness,
			Population:         snapshot.Coverage.Population,
			Excluded:           snapshot.Coverage.Excluded,
			Unknown:            snapshot.Coverage.Unknown,
			GeneratedAt:        snapshot.GeneratedAt,
			SourceRevision:     snapshot.ProjectionVersion,
			DefinitionRevision: HomeDefinitionRevision,
			Basis:              MetricBasisCurrentPosture,
			Drill: DrillTarget{
				Workspace:   "oversight",
				Filter:      filter,
				Consistency: DrillCurrentState,
			},
		}
	}

	return Bundle{
		GeneratedAt:        snapshot.GeneratedAt,
		PeriodStart:        snapshot.PeriodStart,
		PeriodEnd:          snapshot.PeriodEnd,
		ReportingPeriod:    snapshot.ReportingPeriod,
		PostureAsOf:        snapshot.PostureAsOf,
		ScopeID:            snapshot.LegalEntityID,
		ScopeKind:          "LEGAL_ENTITY",
		Freshness:          snapshot.Freshness,
		Completeness:       completeness,
		Population:         snapshot.Coverage.Population,
		Excluded:           snapshot.Coverage.Excluded,
		Unknown:            snapshot.Coverage.Unknown,
		SourceRevision:     snapshot.ProjectionVersion,
		DefinitionRevision: HomeDefinitionRevision,
		Items: []Metric{
			common("critical_high_open", "Critical and high", "critical-high", snapshot.Counts.CriticalHigh),
			common("overdue_open", "Overdue", "overdue", snapshot.Counts.Overdue),
			common("routing_gaps", "Routing gaps", "routing-gaps", snapshot.Counts.RoutingFailures),
			common("outcome_failures", "Outcome failures", "outcome-failures", snapshot.Counts.OutcomeFailures),
		},
	}
}

func coverageCompleteness(coverage oversight.Coverage) Completeness {
	if coverage.Unknown == nil {
		return CompletenessUnknown
	}
	if *coverage.Unknown > 0 {
		return CompletenessPartial
	}
	return CompletenessComplete
}
