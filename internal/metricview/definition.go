package metricview

import "github.com/CloudSpaceLab/clearsight-grc/internal/metricid"

type AggregationRule string

const (
	AggregationSumDisjointCounts AggregationRule = "SUM_DISJOINT_COUNTS"
)

type ConditionRule string

const (
	ConditionRuleZeroClear ConditionRule = "ZERO_CLEAR_POSITIVE_ATTENTION"
)

type Definition struct {
	ID              string
	Revision        string
	Label           string
	Unit            string
	Basis           MetricBasis
	ConditionRule   ConditionRule
	AggregationRule AggregationRule
	Drill           DrillTarget
}

var homeDefinitions = [...]Definition{
	{
		ID: metricid.CriticalHighOpen, Revision: HomeDefinitionRevision, Label: "Critical and high", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "critical-high", Consistency: DrillCurrentState},
	},
	{
		ID: metricid.OverdueOpen, Revision: HomeDefinitionRevision, Label: "Overdue", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "overdue", Consistency: DrillCurrentState},
	},
	{
		ID: metricid.RoutingGaps, Revision: HomeDefinitionRevision, Label: "Routing gaps", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "routing-gaps", Consistency: DrillCurrentState},
	},
	{
		ID: metricid.OutcomeFailures, Revision: HomeDefinitionRevision, Label: "Outcome failures", Unit: "COUNT",
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "outcome-failures", Consistency: DrillCurrentState},
	},
}

func HomeDefinitionList() []Definition {
	values := make([]Definition, len(homeDefinitions))
	copy(values, homeDefinitions[:])
	return values
}

func HomeDefinition(id string) (Definition, bool) {
	for _, definition := range homeDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}
