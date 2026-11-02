package metricview

type AggregationRule string

const (
	AggregationSumDisjointCounts AggregationRule = "SUM_DISJOINT_COUNTS"
	AggregationSumSameCurrency   AggregationRule = "SUM_SAME_CURRENCY"
)

type ConditionRule string

const (
	ConditionRuleZeroClear ConditionRule = "ZERO_CLEAR_POSITIVE_ATTENTION"
	ConditionRuleNone      ConditionRule = "NO_CONDITION"
)

type Definition struct {
	ID              string
	Revision        string
	Label           string
	Unit            MetricUnit
	Basis           MetricBasis
	ConditionRule   ConditionRule
	AggregationRule AggregationRule
	Drill           DrillTarget
}

var homeDefinitions = [...]Definition{
	{
		ID: "critical_high_open", Revision: HomeDefinitionRevision, Label: "Critical and high", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "critical-high", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "overdue_open", Revision: HomeDefinitionRevision, Label: "Overdue", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "overdue", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "routing_gaps", Revision: HomeDefinitionRevision, Label: "Routing gaps", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "routing-gaps", Consistency: DrillSourceSnapshot},
	},
	{
		ID: "outcome_failures", Revision: HomeDefinitionRevision, Label: "Outcome failures", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "outcome-failures", Consistency: DrillSourceSnapshot},
	},
}

var homeCurrentStateDefinitions = [...]Definition{
	{
		ID: "critical_high_open", Revision: HomeCurrentStateDefinitionRevision, Label: "Critical and high", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "critical-high", Consistency: DrillCurrentState},
	},
	{
		ID: "overdue_open", Revision: HomeCurrentStateDefinitionRevision, Label: "Overdue", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "overdue", Consistency: DrillCurrentState},
	},
	{
		ID: "routing_gaps", Revision: HomeCurrentStateDefinitionRevision, Label: "Routing gaps", Unit: MetricUnitCount,
		Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "oversight", Filter: "routing-gaps", Consistency: DrillCurrentState},
	},
	{
		ID: "outcome_failures", Revision: HomeCurrentStateDefinitionRevision, Label: "Outcome failures", Unit: MetricUnitCount,
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
