package metricview

import (
	"context"
	"errors"
	"time"
)

const (
	DomainDefinitionRevision = "enterprise-domain-v1"
	DomainSourceRevision     = "enterprise-domain-v2"
)

var (
	ErrDomainMetricsInvalid  = errors.New("domain metric request is invalid")
	ErrDomainMetricsNotFound = errors.New("domain metrics are not available")
)

var domainDefinitions = [...]Definition{
	{
		ID: "risks_outside_appetite", Revision: DomainDefinitionRevision, Label: "Outside appetite", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "risks", Filter: "outside-appetite", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "indicator_breaches", Revision: DomainDefinitionRevision, Label: "Indicator breaches", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "risks", Filter: "indicator-breach", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "assurance_failures", Revision: DomainDefinitionRevision, Label: "Assurance failures", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "risks", Filter: "assurance-failed", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "losses_without_issue", Revision: DomainDefinitionRevision, Label: "Losses without issue", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "losses", Filter: "without-issue", Consistency: DrillSourceSnapshot},
	},
}

type DomainBundle struct {
	GeneratedAt        time.Time `json:"generated_at"`
	PostureAsOf        time.Time `json:"posture_as_of"`
	ScopeID            string    `json:"scope_id"`
	ScopeKind          string    `json:"scope_kind"`
	SourceID           string    `json:"source_id"`
	SourceRevision     string    `json:"source_revision"`
	DefinitionRevision string    `json:"definition_revision"`
	Items              []Metric  `json:"items"`
}

type DomainReader interface {
	LatestDomainMetrics(context.Context, string, string) (DomainBundle, error)
}

type ScopedDomainReader interface {
	CurrentDomainMetrics(context.Context, string, string, string, []string, time.Time) (DomainBundle, error)
}

func DomainDefinitionList() []Definition {
	values := make([]Definition, len(domainDefinitions))
	copy(values, domainDefinitions[:])
	return values
}

func DomainDefinition(id string) (Definition, bool) {
	for _, definition := range domainDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}

func metricDefinition(id string) (Definition, bool) {
	if definition, ok := HomeDefinition(id); ok {
		return definition, true
	}
	return DomainDefinition(id)
}
