package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type indicatorInsightsRead struct {
	Kind                risk.IndicatorKind             `json:"kind"`
	RiskCount           int                            `json:"risk_count"`
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

type indicatorInsightsPageRead struct {
	Items      []indicatorInsightsRead `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
	Complete   bool                    `json:"complete"`
	GeneratedAt time.Time              `json:"generated_at"`
}

func (a *API) indicatorInsights(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	if a.deps.Monitoring == nil || a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "indicator_insights_unavailable", "Indicator insights are unavailable. Try again.")
		return
	}
	limit, ok := riskLimit(w, r)
	if !ok {
		return
	}
	if limit > 50 {
		limit = 50
	}
	page, err := service.IndicatorPortfolio(r.Context(), scope, risk.IndicatorPortfolioFilter{
		Kind: risk.IndicatorKind(strings.TrimSpace(r.URL.Query().Get("kind"))),
		Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit: limit,
	})
	if err != nil {
		writeRiskError(w, err)
		return
	}

	result := indicatorInsightsPageRead{
		Items: make([]indicatorInsightsRead, 0, len(page.Items)),
		NextCursor: page.NextCursor,
		Complete: true,
		GeneratedAt: time.Now().UTC(),
	}
	type pendingLabel struct {
		index int
		owner bool
		id    string
	}
	pending := make([]pendingLabel, 0, len(page.Items)*2)
	programs := make(map[string]continuity.ProgramAggregate)
	monitorActor := monitoring.Actor{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	}

	for _, item := range page.Items {
		check, err := a.deps.Monitoring.Check(r.Context(), monitorActor, item.MonitoringCheckID, item.MonitoringCheckVersion)
		if err != nil || check.ProgramID != item.ProgramID {
			result.Complete = false
			continue
		}
		program, cached := programs[item.ProgramID]
		if !cached {
			program, err = a.deps.Continuity.GetProgram(r.Context(), actor.TenantID, item.ProgramID)
			if err == nil {
				program, err = a.programForActor(r.Context(), program, nil)
			}
			if err != nil || program.Program.LegalEntityID != actor.LegalEntityID {
				result.Complete = false
				continue
			}
			programs[item.ProgramID] = program
		}

		detail := indicatorInsightsRead{
			Kind: item.Kind, RiskCount: item.RiskCount,
			ProgramID: program.Program.ID, ProgramName: program.Program.Name,
			CheckID: check.ID, CheckCode: check.Code, CheckName: check.Name, Claim: check.Claim,
			CheckStatus: check.Status, CheckVersion: check.Version, InputKind: check.InputKind,
			Measurement: risk.IndicatorMonitoringRiskScore,
			Unit: risk.IndicatorRiskScoreUnit, Denominator: risk.IndicatorRiskScoreDenominator,
			NativeMeasurement: currentRiskIndicatorNativeMeasurement(check, nil),
			State: riskIndicatorUnknown, Reason: "No current monitoring result.",
			MinimumCoverage: check.MinimumCoverage, FreshnessMinutes: check.FreshnessMinutes,
		}

		latest, latestErr := a.deps.Monitoring.LatestResultRevision(r.Context(), monitorActor, check.ID, check.Version)
		switch {
		case latestErr == nil:
			detail.ResultID = latest.ID
			detail.NativeMeasurement = currentRiskIndicatorNativeMeasurement(check, &latest)
			detail.Score = latest.Evaluation.Score
			detail.Band = latest.Evaluation.Band
			coverage := latest.Evaluation.Coverage
			detail.Coverage = &coverage
			evaluatedAt := latest.EvaluatedAt
			detail.EvaluatedAt = &evaluatedAt
			detail.State, detail.Reason = currentRiskIndicatorState(check, latest, result.GeneratedAt)
			if program.Program.Status != continuity.ProgramActive {
				detail.State = riskIndicatorUnknown
				detail.Reason = "Source Program is not active."
			}
		case errors.Is(latestErr, monitoring.ErrNotFound):
			// No result is known UNKNOWN state.
		default:
			result.Complete = false
			continue
		}

		intervention, interventionErr := a.deps.Continuity.OpenMatterByTriggerKey(r.Context(), actor.TenantID, "monitoring-check-adverse:"+check.ID)
		switch {
		case interventionErr == nil:
			if riskIndicatorMatterLinkedToProgram(intervention, program.Program.ID) {
				detail.Intervention = &riskIndicatorInterventionRead{
					MatterID: intervention.Matter.ID, Reference: intervention.Matter.Reference,
					Status: intervention.Matter.Status, Priority: intervention.Matter.Priority, CreatedAt: intervention.Matter.CreatedAt,
				}
			} else {
				result.Complete = false
			}
		case errors.Is(interventionErr, continuity.ErrNotFound):
			// No open intervention is valid state.
		default:
			result.Complete = false
		}

		result.Items = append(result.Items, detail)
		index := len(result.Items) - 1
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
	labels := a.exactAssessmentLabels(r.Context(), actor, actor.LegalEntityID, ids)
	for _, value := range pending {
		if value.owner {
			result.Items[value.index].OwnerDisplayName = labels[value.id]
		} else {
			result.Items[value.index].ReviewerDisplayName = labels[value.id]
		}
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
