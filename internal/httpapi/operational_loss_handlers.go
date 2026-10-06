package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) operationalLossService(w http.ResponseWriter) (*oploss.Service, bool) {
	if a == nil || a.deps.OperationalLoss == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_unavailable", "Loss information is unavailable. Try again.")
		return nil, false
	}
	return a.deps.OperationalLoss, true
}

func (a *API) operationalLossActorScope(w http.ResponseWriter, r *http.Request) (identity.Actor, oploss.Scope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view or change losses.")
		return identity.Actor{}, oploss.Scope{}, false
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" ||
		actor.LegalEntityID == "*" || strings.TrimSpace(actor.PrincipalID) == "" {
		httpx.WriteError(w, http.StatusForbidden, "loss_scope_unavailable", "Choose an eligible legal entity before using the loss register.")
		return identity.Actor{}, oploss.Scope{}, false
	}
	return actor, oploss.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, true
}

func (a *API) listOperationalLosses(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, r.URL.Query().Get("organization_scope_id"), true)
	if err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for Losses.")
		return
	}
	limit, ok := operationalLossLimit(w, r)
	if !ok {
		return
	}
	page, err := service.List(r.Context(), scope, oploss.ListFilter{
		Status:               oploss.Status(strings.TrimSpace(r.URL.Query().Get("status"))),
		EventType:            oploss.EventType(strings.TrimSpace(r.URL.Query().Get("event_type"))),
		Currency:             strings.TrimSpace(r.URL.Query().Get("currency")),
		OrganizationScopeID:  selection.ID,
		OrganizationScopeIDs: selection.IDs,
		RiskID:               strings.TrimSpace(r.URL.Query().Get("risk_id")),
		MatterID:             strings.TrimSpace(r.URL.Query().Get("matter_id")),
		RecoveryStatus:       strings.TrimSpace(r.URL.Query().Get("recovery_status")),
		Search:               strings.TrimSpace(r.URL.Query().Get("search")),
		Cursor:               strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit:                limit,
	})
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) getOperationalLoss(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	labels := a.exactAssessmentLabels(r.Context(), actor, value.Loss.LegalEntityID, []string{value.Loss.OwnerPrincipalID})
	notificationHistory, notificationHistoryComplete := a.notificationDeliveryHistory(r.Context(), actor, "LOSS", value.Loss.ID)
	httpx.WriteJSON(w, http.StatusOK, operationalLossRead{
		Aggregate:                   value,
		OwnerDisplayName:            labels[value.Loss.OwnerPrincipalID],
		NotificationHistory:         notificationHistory,
		NotificationHistoryComplete: notificationHistoryComplete,
	})
}

func (a *API) createOperationalLoss(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, _, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	var input oploss.CreateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeOperationalLossError(w, oploss.ErrInvalid)
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, input.OrganizationScopeID, false)
	if err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for Loss attribution.")
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.OrganizationScopeID = selection.ID
	input.OwnerPrincipalID = actor.PrincipalID
	input.MatterID = ""
	input.ActorID = actor.PrincipalID
	value, err := service.Create(r.Context(), input)
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) updateOperationalLoss(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, _, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	var input oploss.UpdateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeOperationalLossError(w, oploss.ErrInvalid)
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, input.OrganizationScopeID, false)
	if err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for Loss attribution.")
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.OrganizationScopeID = selection.ID
	input.LossID = r.PathValue("id")
	current, err := service.Get(r.Context(), oploss.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, input.LossID)
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	input.OwnerPrincipalID = ""
	input.MatterID = current.Loss.MatterID
	input.ActorID = actor.PrincipalID
	value, err := service.Update(r.Context(), input)
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) recordOperationalLossRecovery(w http.ResponseWriter, r *http.Request) {
	service, ok := a.operationalLossService(w)
	if !ok {
		return
	}
	actor, _, ok := a.operationalLossActorScope(w, r)
	if !ok {
		return
	}
	var input oploss.RecoveryInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeOperationalLossError(w, oploss.ErrInvalid)
		return
	}
	input.TenantID = actor.TenantID
	input.LegalEntityID = actor.LegalEntityID
	input.LossID = r.PathValue("id")
	input.ActorID = actor.PrincipalID
	loss, recovery, err := service.AddRecovery(r.Context(), input)
	if err != nil {
		writeOperationalLossError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"loss": loss, "recovery": recovery})
}

func operationalLossLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 50, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "loss_filter_invalid", "The loss page size must be between 1 and 100.")
		return 0, false
	}
	return value, true
}

func writeOperationalLossError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, oploss.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "loss_not_found", "This loss is not available in your legal entity.")
	case errors.Is(err, oploss.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "loss_version_conflict", "This loss changed since you opened it. Review the current record and try again.")
	case errors.Is(err, oploss.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "loss_duplicate", "A loss with this code already exists in this legal entity.")
	case errors.Is(err, oploss.ErrRecoveryLimit):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "loss_recovery_invalid", "The recovery would make the recovered amount invalid.")
	case errors.Is(err, oploss.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "loss_invalid", "Review the required loss fields and current version.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "loss_unavailable", "Loss information is unavailable. No change was made; try again.")
	}
}
