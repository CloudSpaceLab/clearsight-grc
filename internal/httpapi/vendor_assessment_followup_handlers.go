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

type createVendorAssessmentDeficiencyRequest struct {
	thirdparty.CreateAssessmentDeficiencyInput
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

func (a *API) createVendorAssessmentDeficiency(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentDeficiencies == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_deficiency_unavailable", "The vendor finding could not be created. The assessment was not changed.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to record this vendor finding.")
		return
	}
	var request createVendorAssessmentDeficiencyRequest
	if err = httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the finding title, details, stable key and due date, then try again.")
		return
	}
	outcome, err := a.deps.ThirdPartyAssessmentDeficiencies.CreateDeficiency(r.Context(), actor, r.PathValue("id"), request.CreateAssessmentDeficiencyInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, outcome)
}
