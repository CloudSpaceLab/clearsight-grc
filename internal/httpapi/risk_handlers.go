package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (a *API) riskService(w http.ResponseWriter) (*risk.Service, bool) {
	if a == nil || a.deps.Risk == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "risk_unavailable", "Risk information is unavailable. Try again.")
		return nil, false
	}
	return a.deps.Risk, true
}

func (a *API) riskActorScope(w http.ResponseWriter, r *http.Request) (identity.Actor, risk.Scope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view or change risks.")
		return identity.Actor{}, risk.Scope{}, false
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" || strings.TrimSpace(actor.PrincipalID) == "" {
		httpx.WriteError(w, http.StatusForbidden, "risk_scope_unavailable", "Choose an eligible legal entity before using the risk register.")
		return identity.Actor{}, risk.Scope{}, false
	}
	return actor, risk.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, true
}

func (a *API) listRisks(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, r.URL.Query().Get("organization_scope_id"), true)
	if err != nil {
		writeRiskOrganizationScopeError(w, err, false)
		return
	}
	limit, ok := riskLimit(w, r)
	if !ok {
		return
	}
	page, err := service.List(r.Context(), scope, risk.ListFilter{
		Status:           risk.Status(strings.TrimSpace(r.URL.Query().Get("status"))),
		Category:         strings.TrimSpace(r.URL.Query().Get("category")),
		OwnerPrincipalID: strings.TrimSpace(r.URL.Query().Get("owner_principal_id")),
		Search:           strings.TrimSpace(r.URL.Query().Get("search")),
		AppetitePosition:     risk.AppetitePosition(strings.TrimSpace(r.URL.Query().Get("appetite_position"))),
		OrganizationScopeID:  selection.ID,
		OrganizationScopeIDs: selection.IDs,
		Cursor:               strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit:            limit,
	})
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) getRisk(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.riskAggregateWithDetails(r.Context(), actor, value))
}

func (a *API) createRisk(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	var input risk.CreateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, input.OrganizationScopeID, false)
	if err != nil {
		writeRiskOrganizationScopeError(w, err, true)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.OrganizationScopeID = selection.ID
	input.ActorID = actor.PrincipalID
	input.OwnerPrincipalID = actor.PrincipalID
	value, err := service.Create(r.Context(), input)
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) updateRisk(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	var input risk.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.RiskID = r.PathValue("id")
	input.ActorID = actor.PrincipalID
	value, err := service.Update(r.Context(), input)
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) assessRisk(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	var input risk.AssessmentInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.RiskID = r.PathValue("id")
	input.ActorID = actor.PrincipalID
	current, assessment, err := service.AddAssessment(r.Context(), input)
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"risk": current, "assessment": assessment})
}

func (a *API) activateRiskAppetite(w http.ResponseWriter, r *http.Request) {
	service, ok := a.riskService(w)
	if !ok {
		return
	}
	actor, _, ok := a.riskActorScope(w, r)
	if !ok {
		return
	}
	var input risk.AppetiteInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRiskError(w, risk.ErrInvalid)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.RiskID = r.PathValue("id")
	input.ActorID = actor.PrincipalID
	current, statement, err := service.ActivateAppetite(r.Context(), input)
	if err != nil {
		writeRiskError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"risk": current, "appetite": statement})
}

func writeRiskOrganizationScopeError(w http.ResponseWriter, err error, attribution bool) {
	if errors.Is(err, errOrganizationScopeUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "organization_scope_unavailable", "Organization scope could not be verified. Try again.")
		return
	}
	message := "This organization scope is not available for Risks."
	if attribution {
		message = "This organization scope is not available for Risk attribution."
	}
	httpx.WriteError(w, http.StatusForbidden, "organization_scope_forbidden", message)
}

func riskLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 50, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "risk_filter_invalid", "The risk page size must be between 1 and 100.")
		return 0, false
	}
	return value, true
}

func writeRiskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, risk.ErrNotFound), errors.Is(err, risk.ErrScopeMismatch):
		httpx.WriteError(w, http.StatusNotFound, "risk_not_found", "This risk is not available in your legal entity.")
	case errors.Is(err, risk.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "risk_version_conflict", "This risk changed since you opened it. Review the current record and try again.")
	case errors.Is(err, risk.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "risk_duplicate", "A risk with this code already exists in this legal entity.")
	case errors.Is(err, risk.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "risk_invalid", "The risk information is not valid. Review the required fields and current version, then try again.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "risk_unavailable", "Risk information is unavailable. No change was made; try again.")
	}
}
