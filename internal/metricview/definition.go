package metricview

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

var legacyHomeDefinitions = buildHomeDefinitions(LegacyHomeDefinitionRevision, DrillCurrentState)
var homeDefinitions = buildHomeDefinitions(HomeDefinitionRevision, DrillSnapshotExact)

func buildHomeDefinitions(revision string, consistency DrillConsistency) [4]Definition {
	return [4]Definition{
		{
			ID: "critical_high_open", Revision: revision, Label: "Critical and high", Unit: "COUNT",
			Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
			Drill: DrillTarget{Workspace: "oversight", Filter: "critical-high", Consistency: consistency},
		},
		{
			ID: "overdue_open", Revision: revision, Label: "Overdue", Unit: "COUNT",
			Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
			Drill: DrillTarget{Workspace: "oversight", Filter: "overdue", Consistency: consistency},
		},
		{
			ID: "routing_gaps", Revision: revision, Label: "Routing gaps", Unit: "COUNT",
			Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
			Drill: DrillTarget{Workspace: "oversight", Filter: "routing-gaps", Consistency: consistency},
		},
		{
			ID: "outcome_failures", Revision: revision, Label: "Outcome failures", Unit: "COUNT",
			Basis: MetricBasisCurrentPosture, ConditionRule: ConditionRuleZeroClear, AggregationRule: AggregationSumDisjointCounts,
			Drill: DrillTarget{Workspace: "oversight", Filter: "outcome-failures", Consistency: consistency},
		},
	}
}

func HomeDefinitionList() []Definition {
	values := make([]Definition, len(homeDefinitions))
	copy(values, homeDefinitions[:])
	return values
}

func AllHomeDefinitionList() []Definition {
	values := make([]Definition, 0, len(homeDefinitions)+len(legacyHomeDefinitions))
	values = append(values, legacyHomeDefinitions[:]...)
	values = append(values, homeDefinitions[:]...)
	return values
}

func HomeDefinition(id string) (Definition, bool) {
	return HomeDefinitionForRevision(id, HomeDefinitionRevision)
}

func HomeDefinitionForRevision(id, revision string) (Definition, bool) {
	var values [4]Definition
	switch revision {
	case LegacyHomeDefinitionRevision:
		values = legacyHomeDefinitions
	case HomeDefinitionRevision:
		values = homeDefinitions
	default:
		return Definition{}, false
	}
	for _, definition := range values {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}
