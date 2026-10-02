package httpapi

import (
	"context"
	"errors"
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

type riskIndicatorRead struct {
	Link                risk.IndicatorLink         `json:"link"`
	ProgramID           string                     `json:"program_id"`
	ProgramName         string                     `json:"program_name"`
	CheckID             string                     `json:"check_id"`
	CheckCode           string                     `json:"check_code"`
	CheckName           string                     `json:"check_name"`
	Claim               string                     `json:"claim"`
	CheckStatus         monitoring.LifecycleStatus `json:"check_status"`
	CheckVersion        int64                      `json:"check_version"`
	InputKind           monitoring.InputKind       `json:"input_kind"`
	OwnerDisplayName    string                     `json:"owner_display_name,omitempty"`
	ReviewerDisplayName string                     `json:"reviewer_display_name,omitempty"`
	Measurement         risk.IndicatorMeasurement  `json:"measurement"`
	Unit                string                     `json:"unit"`
	Denominator         int                        `json:"denominator"`
	State               riskIndicatorState         `json:"state"`
	Reason              string                     `json:"reason"`
	Score               *float64                   `json:"score,omitempty"`
	Band                monitoring.RiskBand        `json:"band,omitempty"`
	Coverage            *float64                   `json:"coverage,omitempty"`
	MinimumCoverage     float64                    `json:"minimum_coverage"`
	FreshnessMinutes    int                        `json:"freshness_minutes"`
	ResultID            string                     `json:"result_id,omitempty"`
	EvaluatedAt         *time.Time                 `json:"evaluated_at,omitempty"`
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

	type pendingLabel struct {
		index int
		owner bool
		id    string
	}
	currentIndicators := currentRiskIndicatorLinks(value.Indicators)
	pending := make([]pendingLabel, 0, len(currentIndicators)*2)
	programs := map[string]continuity.ProgramAggregate{}
	now := time.Now().UTC()
	monitorActor := monitoring.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}

	for _, link := range currentIndicators {
		check, err := a.deps.Monitoring.Check(ctx, monitorActor, link.MonitoringCheckID, link.MonitoringCheckVersion)
		if err != nil || check.ProgramID != link.ProgramID {
			result.IndicatorDetailsComplete = false
			continue
		}
		program, ok := programs[check.ProgramID]
		if !ok {
			program, err = a.deps.Continuity.GetProgram(ctx, actor.TenantID, check.ProgramID)
			if err == nil {
				program, err = a.programForActor(ctx, program, nil)
			}
			if err != nil || program.Program.LegalEntityID != actor.LegalEntityID {
				result.IndicatorDetailsComplete = false
				continue
			}
			programs[check.ProgramID] = program
		}

		detail := riskIndicatorRead{
			Link:             link,
			ProgramID:        program.Program.ID,
			ProgramName:      program.Program.Name,
			CheckID:          check.ID,
			CheckCode:        check.Code,
			CheckName:        check.Name,
			Claim:            check.Claim,
			CheckStatus:      check.Status,
			CheckVersion:     check.Version,
			InputKind:        check.InputKind,
			Measurement:      risk.IndicatorMonitoringRiskScore,
			Unit:             risk.IndicatorRiskScoreUnit,
			Denominator:      risk.IndicatorRiskScoreDenominator,
			State:            riskIndicatorUnknown,
			Reason:           "No current monitoring result.",
			MinimumCoverage:  check.MinimumCoverage,
			FreshnessMinutes: check.FreshnessMinutes,
		}
		resultValue, resultErr := a.deps.Monitoring.LatestResultRevision(ctx, monitorActor, check.ID, check.Version)
		switch {
		case resultErr == nil:
			detail.ResultID = resultValue.ID
			detail.Score = resultValue.Evaluation.Score
			detail.Band = resultValue.Evaluation.Band
			coverage := resultValue.Evaluation.Coverage
			detail.Coverage = &coverage
			evaluatedAt := resultValue.EvaluatedAt
			detail.EvaluatedAt = &evaluatedAt
			detail.State, detail.Reason = currentRiskIndicatorState(check, resultValue, now)
		case errors.Is(resultErr, monitoring.ErrNotFound):
			// Absence of a result is known Indicator truth: UNKNOWN, not an incomplete API projection.
		default:
			result.IndicatorDetailsComplete = false
			continue
		}

		result.IndicatorDetails = append(result.IndicatorDetails, detail)
		index := len(result.IndicatorDetails) - 1
		if check.OwnerPrincipalID != "" {
			pending = append(pending, pendingLabel{index: index, owner: true, id: check.OwnerPrincipalID})
		}
		if check.ReviewerPrincipalID != "" {
			pending = append(pending, pendingLabel{index: index, id: check.ReviewerPrincipalID})
		}
	}

	ids := make([]string, 0, len(pending))
	for _, value := range pending {
		ids = append(ids, value.id)
	}
	labels := a.exactAssessmentLabels(ctx, actor, actor.LegalEntityID, ids)
	for _, value := range pending {
		if value.owner {
			result.IndicatorDetails[value.index].OwnerDisplayName = labels[value.id]
		} else {
			result.IndicatorDetails[value.index].ReviewerDisplayName = labels[value.id]
		}
	}
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
