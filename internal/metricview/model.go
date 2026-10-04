package metricview

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

const (
	HomeDefinitionRevision             = "home-oversight-v3"
	HomeCurrentStateDefinitionRevision = "home-oversight-v2"
)

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
	DrillCurrentState   DrillConsistency = "CURRENT_STATE"
	DrillSourceSnapshot DrillConsistency = "SOURCE_SNAPSHOT"
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
	SourceID           string                    `json:"source_id,omitempty"`
	SourceRevision     string                    `json:"source_revision"`
	DefinitionRevision string                    `json:"definition_revision"`
	Items              []Metric                  `json:"items"`
}

func FromOversight(snapshot oversight.Snapshot) Bundle {
	completeness := coverageCompleteness(snapshot.Coverage)
	if snapshot.OrganizationScopeID != "" {
		// A subordinate view never counts unattributed legal-entity records because
		// that would leak sibling-sensitive totals. Until attribution coverage can
		// be proven independently, report scoped completeness as unknown.
		completeness = CompletenessUnknown
	}
	values := map[string]int{
		"critical_high_open": snapshot.Counts.CriticalHigh,
		"overdue_open":       snapshot.Counts.Overdue,
		"routing_gaps":       snapshot.Counts.RoutingFailures,
		"outcome_failures":   snapshot.Counts.OutcomeFailures,
	}
	definitions := homeDefinitions[:]
	definitionRevision := HomeDefinitionRevision
	if snapshot.SnapshotID == "" {
		definitions = homeCurrentStateDefinitions[:]
		definitionRevision = HomeCurrentStateDefinitionRevision
	}
	items := make([]Metric, 0, len(definitions))
	for _, definition := range definitions {
		value := values[definition.ID]
		condition := ConditionClear
		if value > 0 {
			condition = ConditionAttention
		}
		items = append(items, Metric{
			ID:                 definition.ID,
			Label:              definition.Label,
			Value:              value,
			Unit:               definition.Unit,
			Condition:          condition,
			Freshness:          snapshot.Freshness,
			Completeness:       completeness,
			Population:         snapshot.Coverage.Population,
			Excluded:           snapshot.Coverage.Excluded,
			Unknown:            snapshot.Coverage.Unknown,
			GeneratedAt:        snapshot.GeneratedAt,
			SourceRevision:     snapshot.ProjectionVersion,
			DefinitionRevision: definition.Revision,
			Basis:              definition.Basis,
			Drill:              definition.Drill,
		})
	}

	scopeID, scopeKind := snapshot.LegalEntityID, "LEGAL_ENTITY"
	if snapshot.OrganizationScopeID != "" {
		scopeID, scopeKind = snapshot.OrganizationScopeID, "ORGANIZATION_SCOPE"
	}
	return Bundle{
		GeneratedAt:        snapshot.GeneratedAt,
		PeriodStart:        snapshot.PeriodStart,
		PeriodEnd:          snapshot.PeriodEnd,
		ReportingPeriod:    snapshot.ReportingPeriod,
		PostureAsOf:        snapshot.PostureAsOf,
		ScopeID:            scopeID,
		ScopeKind:          scopeKind,
		Freshness:          snapshot.Freshness,
		Completeness:       completeness,
		Population:         snapshot.Coverage.Population,
		Excluded:           snapshot.Coverage.Excluded,
		Unknown:            snapshot.Coverage.Unknown,
		SourceID:           snapshot.SnapshotID,
		SourceRevision:     snapshot.ProjectionVersion,
		DefinitionRevision: definitionRevision,
		Items:              items,
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
