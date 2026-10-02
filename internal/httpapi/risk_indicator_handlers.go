package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (a *API) linkRiskIndicator(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	if a.deps.Monitoring == nil || a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "risk_indicator_unavailable", "Risk indicators are unavailable. Try again.")
		return
	}
	var input risk.LinkIndicatorInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	check, err := a.deps.Monitoring.Check(r.Context(), monitoring.Actor{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	}, input.MonitoringCheckID, input.MonitoringCheckVersion)
	if err != nil || check.Status != monitoring.LifecycleActive || !check.IsCurrent {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	program, err := a.deps.Continuity.GetProgram(r.Context(), actor.TenantID, check.ProgramID)
	if err == nil {
		program, err = a.programForActor(r.Context(), program, nil)
	}
	if err != nil || program.Program.LegalEntityID != actor.LegalEntityID {
		if errors.Is(err, continuity.ErrNotFound) {
			writeRiskError(w, risk.ErrInvalid)
			return
		}
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.RiskID = r.PathValue("id")
	input.ActorID = actor.PrincipalID
	current, indicator, err := service.LinkIndicator(r.Context(), input)
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"risk": current, "indicator": indicator})
}
