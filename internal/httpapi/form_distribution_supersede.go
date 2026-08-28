package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type supersedeFormDistributionRequest struct {
	ExpectedVersion          int64    `json:"expected_version"`
	TargetFormVersion        int64    `json:"target_form_version"`
	Confirm                  bool     `json:"confirm"`
	ExpectedWorkspaceVersion int64    `json:"expected_workspace_version,omitempty"`
	CarryForward             bool     `json:"carry_forward,omitempty"`
	ConfirmedFieldIDs        []string `json:"confirmed_field_ids,omitempty"`
}

func (a *API) supersedeGovernedFormDistribution(w http.ResponseWriter, r *http.Request) {
	_, actor, legalEntityID, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	var request supersedeFormDistributionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The supersession request must be valid JSON.")
		return
	}
	distributionID := r.PathValue("id")
	if !request.Confirm {
		preview, err := access.PreviewDistributionSupersession(r.Context(), actor.TenantID, legalEntityID, distributionID, evidence.SupersessionPreviewInput{
			ExpectedVersion: request.ExpectedVersion, TargetFormVersion: request.TargetFormVersion,
		})
		if err != nil {
			writeFormSupersessionError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"preview": preview, "confirmation_required": true})
		return
	}

	result, err := access.SupersedeDistribution(r.Context(), actor.TenantID, legalEntityID, distributionID, evidence.SupersedeDistributionInput{
		ExpectedVersion: request.ExpectedVersion, ExpectedWorkspaceVersion: request.ExpectedWorkspaceVersion,
		TargetFormVersion: request.TargetFormVersion, CarryForward: request.CarryForward,
		ConfirmedFieldIDs: request.ConfirmedFieldIDs, ActorID: actor.PrincipalID,
	})
	if err != nil {
		writeFormSupersessionError(w, err)
		return
	}
	response := map[string]any{
		"previous":          distributionBundleJSON(result.Previous),
		"replacement":       distributionBundleJSON(result.Replacement),
		"carried_field_ids": result.CarriedFieldIDs,
	}
	if len(result.IssuedRoutes) > 0 {
		secureNoStore(w)
		response["issued_access_routes"] = result.IssuedRoutes
	}
	httpx.WriteJSON(w, http.StatusCreated, response)
}

func writeFormSupersessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrSupersessionPreviewMismatch):
		httpx.WriteError(w, http.StatusConflict, "supersession_preview_stale", "The distribution or response changed after preview. Preview the replacement again before confirming.")
	case errors.Is(err, evidence.ErrDistributionAccessUnavailable), errors.Is(err, evidence.ErrProtectedRecipientInvalid):
		httpx.WriteError(w, http.StatusConflict, "supersession_access_unavailable", "Secure recipient access could not be prepared for the replacement distribution.")
	default:
		writeFormDistributionError(w, err)
	}
}
