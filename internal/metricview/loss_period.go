package metricview

import (
	"context"
	"errors"
	"time"
)

const (
	LossPeriodDefinitionRevision = "operational-loss-period-v1"
	LossPeriodSourceRevision     = "operational-loss-ledger-v1"

	LossPeriodMetricEvents = "operational_loss_events"
	LossPeriodMetricNet    = "operational_loss_net"
)

var (
	ErrLossPeriodInvalid  = errors.New("operational loss period request is invalid")
	ErrLossPeriodNotFound = errors.New("operational loss period is not available")
)

var lossPeriodDefinitions = [...]Definition{
	{
		ID: LossPeriodMetricEvents, Revision: LossPeriodDefinitionRevision, Label: "Loss events", Unit: MetricUnitCount,
		Basis: MetricBasisPeriodFlow, ConditionRule: ConditionRuleNone, AggregationRule: AggregationSumDisjointCounts,
		Drill: DrillTarget{Workspace: "losses", Filter: "period-events", Consistency: DrillSourceSnapshot},
	},
	{
		ID: LossPeriodMetricNet, Revision: LossPeriodDefinitionRevision, Label: "Net operational loss", Unit: MetricUnitMoney,
		Basis: MetricBasisPeriodFlow, ConditionRule: ConditionRuleNone, AggregationRule: AggregationSumSameCurrency,
		Drill: DrillTarget{Workspace: "losses", Filter: "period-net", Consistency: DrillSourceSnapshot},
	},
}

type LossCurrencyFlow struct {
	Currency           string     `json:"currency"`
	Gross              MoneyValue `json:"gross"`
	Recovery           MoneyValue `json:"recovery"`
	Reversal           MoneyValue `json:"reversal"`
	Net                MoneyValue `json:"net"`
	LossEventCount     int        `json:"loss_event_count"`
	RecoveryEventCount int        `json:"recovery_event_count"`
	ReversalEventCount int        `json:"reversal_event_count"`
}

type LossOrganizationFlow struct {
	Key                   string             `json:"key"`
	ScopeID               string             `json:"scope_id,omitempty"`
	Label                 string             `json:"label"`
	Kind                  string             `json:"kind"`
	LossEventCount        int                `json:"loss_event_count"`
	ContributingLossCount int                `json:"contributing_loss_count"`
	MixedCurrencies       bool               `json:"mixed_currencies"`
	NetLoss               *MoneyValue        `json:"net_loss,omitempty"`
	Currencies            []LossCurrencyFlow `json:"currencies"`
}

type LossFlowResolution string

const (
	LossFlowResolutionDay  LossFlowResolution = "DAY"
	LossFlowResolutionWeek LossFlowResolution = "WEEK"
)

type LossFlowPoint struct {
	Start                 time.Time          `json:"start"`
	End                   time.Time          `json:"end"`
	LossEventCount        int                `json:"loss_event_count"`
	ContributingLossCount int                `json:"contributing_loss_count"`
	MixedCurrencies       bool               `json:"mixed_currencies"`
	NetLoss               *MoneyValue        `json:"net_loss,omitempty"`
	Currencies            []LossCurrencyFlow `json:"currencies"`
}

type LossPeriodComparison struct {
	PeriodStart            time.Time          `json:"period_start"`
	PeriodEnd              time.Time          `json:"period_end"`
	EventCount             int                `json:"event_count"`
	ContributingLossCount  int                `json:"contributing_loss_count"`
	MixedCurrencies        bool               `json:"mixed_currencies"`
	NetLoss                *MoneyValue        `json:"net_loss,omitempty"`
	Currencies             []LossCurrencyFlow `json:"currencies"`
	EventDelta             int                `json:"event_delta"`
	NetDelta               *MoneyValue        `json:"net_delta,omitempty"`
	Direction              TrendDirection     `json:"direction"`
	ComparisonQuality      ComparisonQuality  `json:"comparison_quality"`
}

type LossPeriodBundle struct {
	GeneratedAt            time.Time          `json:"generated_at"`
	PeriodStart            time.Time          `json:"period_start"`
	PeriodEnd              time.Time          `json:"period_end"`
	ScopeID                string             `json:"scope_id"`
	ScopeKind              string             `json:"scope_kind"`
	SourceID               string             `json:"source_id"`
	SourceRevision         string             `json:"source_revision"`
	DefinitionRevision     string             `json:"definition_revision"`
	EventCount             int                `json:"event_count"`
	ContributingLossCount  int                `json:"contributing_loss_count"`
	UnattributedEventCount int                `json:"unattributed_event_count"`
	MixedCurrencies        bool               `json:"mixed_currencies"`
	NetLoss                *MoneyValue        `json:"net_loss,omitempty"`
	Currencies            []LossCurrencyFlow    `json:"currencies"`
	OrganizationBreakdown []LossOrganizationFlow `json:"organization_breakdown"`
	FlowResolution        LossFlowResolution      `json:"flow_resolution"`
	FlowPoints            []LossFlowPoint          `json:"flow_points"`
	Comparison            LossPeriodComparison     `json:"comparison"`
}

type LossPeriodReader interface {
	CurrentLossPeriod(
		context.Context,
		string,
		string,
		string,
		[]string,
		time.Time,
		time.Time,
		time.Time,
	) (LossPeriodBundle, error)
}

func LossPeriodDefinitionList() []Definition {
	values := make([]Definition, len(lossPeriodDefinitions))
	copy(values, lossPeriodDefinitions[:])
	return values
}

func LossPeriodDefinition(id string) (Definition, bool) {
	for _, definition := range lossPeriodDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}
