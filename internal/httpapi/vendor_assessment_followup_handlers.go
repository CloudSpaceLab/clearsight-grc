package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type requestVendorAssessmentClarificationRequest struct {
	thirdparty.RequestAssessmentClarificationInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

func (a *API) requestVendorAssessmentClarification(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentRequests == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_clarification_unavailable", "The clarification request is temporarily unavailable. No invitation was issued.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to request vendor clarification.")
		return
	}
	var request requestVendorAssessmentClarificationRequest
	if err = httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Select the response fields, provide the recipient and deadline, then try again.")
		return
	}
	outcome, err := a.deps.ThirdPartyAssessmentRequests.RequestClarification(r.Context(), actor, r.PathValue("id"), request.RequestAssessmentClarificationInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, outcome)
}

// The finding command is withheld until Matter creation, association and
// current authority checks share one material transaction.
func (a *API) createVendorAssessmentDeficiency(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_deficiency_unavailable", "Finding creation is temporarily unavailable.")
}
