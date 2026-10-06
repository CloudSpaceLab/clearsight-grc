package httpapi

import (
	"context"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type riskIndicatorState string

const (
	riskIndicatorNormal  riskIndicatorState = "NORMAL"
	riskIndicatorWatch   riskIndicatorState = "WATCH"
	riskIndicatorBreach  riskIndicatorState = "BREACH"
	riskIndicatorUnknown riskIndicatorState = "UNKNOWN"
)

type riskIndicatorInterventionRead struct {
	MatterID  string                  `json:"matter_id"`
	Reference string                  `json:"reference"`
	Status    continuity.MatterStatus `json:"status"`
	Priority  int                     `json:"priority"`
	CreatedAt time.Time               `json:"created_at"`
}

type riskIndicatorRead struct {
	Link                risk.IndicatorLink             `json:"link"`
	ProgramID           string                         `json:"program_id"`
	ProgramName         string                         `json:"program_name"`
	CheckID             string                         `json:"check_id"`
	CheckCode           string                         `json:"check_code"`
	CheckName           string                         `json:"check_name"`
	Claim               string                         `json:"claim"`
	CheckStatus         monitoring.LifecycleStatus     `json:"check_status"`
	CheckVersion        int64                          `json:"check_version"`
	InputKind           monitoring.InputKind           `json:"input_kind"`
	OwnerDisplayName    string                         `json:"owner_display_name,omitempty"`
	ReviewerDisplayName string                         `json:"reviewer_display_name,omitempty"`
	Measurement         risk.IndicatorMeasurement      `json:"measurement"`
	Unit                string                         `json:"unit"`
	Denominator         int                            `json:"denominator"`
	NativeMeasurement   *monitoring.NativeMeasurement  `json:"native_measurement,omitempty"`
	Movement            *riskIndicatorMovementRead     `json:"movement,omitempty"`
	State               riskIndicatorState             `json:"state"`
	Reason              string                         `json:"reason"`
	Score               *float64                       `json:"score,omitempty"`
	Band                monitoring.RiskBand            `json:"band,omitempty"`
	Coverage            *float64                       `json:"coverage,omitempty"`
	MinimumCoverage     float64                        `json:"minimum_coverage"`
	FreshnessMinutes    int                            `json:"freshness_minutes"`
	ResultID            string                         `json:"result_id,omitempty"`
	EvaluatedAt         *time.Time                     `json:"evaluated_at,omitempty"`
	Intervention        *riskIndicatorInterventionRead `json:"intervention,omitempty"`
}

func (a *API) riskAggregateWithDetails(ctx context.Context, actor identity.Actor, value risk.Aggregate) riskAggregateRead {
	result := a.riskAggregateWithControls(ctx, actor, value)
	result.IndicatorDetails = []riskIndicatorRead{}
	result.IndicatorDetailsComplete = true
	if len(value.Indicators) == 0 {
		return result
	}
	if a == nil || a.deps.Monitoring == nil || a.deps.Continuity == nil {
		result.IndicatorDetailsComplete = false
		return result
	}

	builder := newRiskIndicatorReadBuilder(a, ctx, actor)
	labels := make([]riskIndicatorLabelPair, 0, len(value.Indicators))
	for _, link := range currentRiskIndicatorLinks(value.Indicators) {
		detail, labelPair, visible, complete := builder.build(link)
		if !visible {
			result.IndicatorDetailsComplete = false
			continue
		}
		if !complete {
			result.IndicatorDetailsComplete = false
		}
		result.IndicatorDetails = append(result.IndicatorDetails, detail)
		labels = append(labels, labelPair)
	}
	a.applyRiskIndicatorLabels(ctx, actor, result.IndicatorDetails, labels)
	return result
}

func currentRiskIndicatorLinks(values []risk.IndicatorLink) []risk.IndicatorLink {
	latest := make(map[string]risk.IndicatorLink, len(values))
	for _, value := range values {
		current, ok := latest[value.MonitoringCheckID]
		if !ok || value.RiskVersion > current.RiskVersion ||
			(value.RiskVersion == current.RiskVersion && value.ID > current.ID) {
			latest[value.MonitoringCheckID] = value
		}
	}
	result := make([]risk.IndicatorLink, 0, len(latest))
	for _, value := range latest {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RiskVersion != result[j].RiskVersion {
			return result[i].RiskVersion > result[j].RiskVersion
		}
		return result[i].ID > result[j].ID
	})
	return result
}

func currentRiskIndicatorState(check monitoring.MonitoringCheck, result monitoring.MonitoringResult, now time.Time) (riskIndicatorState, string) {
	if check.Status != monitoring.LifecycleActive || !check.IsCurrent {
		return riskIndicatorUnknown, "Monitoring check is no longer current."
	}
	if result.MonitoringCheckID != check.ID || result.MonitoringCheckVersion != check.Version {
		return riskIndicatorUnknown, "Monitoring result does not match the linked check revision."
	}
	if result.EvaluatedAt.IsZero() || (check.FreshnessMinutes > 0 && now.Sub(result.EvaluatedAt) > time.Duration(check.FreshnessMinutes)*time.Minute) {
		return riskIndicatorUnknown, "Latest monitoring result is stale."
	}
	if result.Evaluation.Coverage < check.MinimumCoverage {
		return riskIndicatorUnknown, "Monitoring coverage is below the approved minimum."
	}
	if result.Evaluation.Measurement != nil {
		condition, err := monitoring.EvaluateNativeMeasurementCondition(result.Evaluation.Measurement)
		if err != nil || condition == monitoring.MeasurementConditionUnknown {
			return riskIndicatorUnknown, "Native measurement or approved limit is unavailable."
		}
		if condition == monitoring.MeasurementConditionBreached {
			return riskIndicatorBreach, "Latest native measurement is outside its approved limit."
		}
		return riskIndicatorNormal, "Latest native measurement is within its approved limit."
	}
	switch result.Evaluation.Band {
	case monitoring.RiskLow:
		return riskIndicatorNormal, "Latest complete result is in the low band."
	case monitoring.RiskModerate:
		return riskIndicatorWatch, "Latest complete result is in the moderate band."
	case monitoring.RiskHigh, monitoring.RiskCritical:
		return riskIndicatorBreach, "Latest complete result is in a high or critical band."
	default:
		return riskIndicatorUnknown, "Latest monitoring result is not assessed."
	}
}

func riskIndicatorMatterLinkedToProgram(value continuity.MatterAggregate, programID string) bool {
	for _, link := range value.Links {
		if link.ProgramID == programID && link.RetiredAt == nil {
			return true
		}
	}
	return false
}

func currentRiskIndicatorNativeMeasurement(check monitoring.MonitoringCheck, result *monitoring.MonitoringResult) *monitoring.NativeMeasurement {
	if result != nil && result.Evaluation.Measurement != nil {
		value := *result.Evaluation.Measurement
		value.Limits = append([]monitoring.MeasurementLimit(nil), result.Evaluation.Measurement.Limits...)
		return &value
	}
	return monitoring.MeasurementDefinition(check.Measurement, check.SourceRules)
}
