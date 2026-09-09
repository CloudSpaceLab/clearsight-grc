package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"net/http"
)

func (a *API) getVendorAssessmentCollection(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	if a.deps.ThirdPartyAssessmentReviews == nil {
		httpx.WriteError(w, 503, "vendor_collection_unavailable", "The vendor request checklist is temporarily unavailable. Try again.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, 401, "sign_in_required", "Sign in to view the vendor request.")
		return
	}
	value, err := a.deps.ThirdPartyAssessmentReviews.GetCollection(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func (a *API) reconcileVendorAssessmentCollection(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	if a.deps.ThirdPartyAssessmentReviews == nil {
		httpx.WriteError(w, 503, "vendor_collection_unavailable", "The document cannot be linked right now. Try again.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, 401, "sign_in_required", "Sign in to link the submitted document.")
		return
	}
	var input struct {
		thirdparty.ReconcileAssessmentCollectionInput
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
		ActorID       string `json:"actor_id,omitempty"`
	}
	if err = httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "Select the submitted document and explain why it covers this request.")
		return
	}
	value, err := a.deps.ThirdPartyAssessmentReviews.ReconcileCollection(r.Context(), actor, r.PathValue("id"), r.PathValue("field_id"), input.ReconcileAssessmentCollectionInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func (a *API) prepareVendorAssessmentRequest(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	if a.deps.ThirdPartyAssessmentRequests == nil {
		httpx.WriteError(w, 503, "vendor_request_unavailable", "The vendor request cannot be prepared right now. Try again.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, 401, "sign_in_required", "Sign in to prepare the vendor request.")
		return
	}
	var input struct {
		thirdparty.PrepareAssessmentRequestInput
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
		ActorID       string `json:"actor_id,omitempty"`
	}
	if err = httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "Check the vendor contact and response deadline.")
		return
	}
	value, err := a.deps.ThirdPartyAssessmentRequests.PrepareRequest(r.Context(), actor, r.PathValue("id"), input.PrepareAssessmentRequestInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, 200, value)
}
