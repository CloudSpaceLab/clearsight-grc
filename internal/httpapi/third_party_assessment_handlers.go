package httpapi

import (
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type startVendorAssessmentRequest struct {
	thirdparty.StartAssessmentInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type sendVendorAssessmentRequest struct {
	thirdparty.SendAssessmentRequestInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type vendorAssessmentCurrentResponse struct {
	Assessment thirdparty.Assessment             `json:"assessment"`
	Setup      *thirdparty.AssessmentSetupStatus `json:"setup,omitempty"`
}

func (a *API) assessmentService(w http.ResponseWriter) (*thirdparty.AssessmentService, bool) {
	if a.deps.ThirdPartyAssessments == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_assessments_unavailable", "Vendor due diligence is temporarily unavailable. No change was made.")
		return nil, false
	}
	return a.deps.ThirdPartyAssessments, true
}

func (a *API) startVendorAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.assessmentService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to start vendor due diligence.")
		return
	}
	var request startVendorAssessmentRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the due-diligence dates and form version, then try again.")
		return
	}
	assessment, err := service.StartAssessment(r.Context(), actor, r.PathValue("id"), request.StartAssessmentInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, assessment)
}

func (a *API) getCurrentVendorAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.assessmentService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view vendor due diligence.")
		return
	}
	assessment, err := service.GetCurrentAssessment(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	response := vendorAssessmentCurrentResponse{Assessment: assessment}
	setup, setupErr := service.GetAssessmentSetupStatus(r.Context(), actor, assessment.ID)
	if setupErr == nil {
		response.Setup = &setup
	} else if assessment.Status == thirdparty.AssessmentSetupPending {
		writeThirdPartyAssessmentError(w, setupErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (a *API) sendVendorAssessmentRequest(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentRequests == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_request_unavailable", "The vendor request is temporarily unavailable. No invitation was issued.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to send this vendor request.")
		return
	}
	var request sendVendorAssessmentRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the recipient, deadline and invitation period, then try again.")
		return
	}
	outcome, err := a.deps.ThirdPartyAssessmentRequests.SendRequest(r.Context(), actor, r.PathValue("id"), request.SendAssessmentRequestInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, outcome)
}

func writeThirdPartyAssessmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrMissingIdentity):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, thirdparty.ErrAssessmentIdentityMismatch), errors.Is(err, thirdparty.ErrAssessmentAuthorityUnavailable):
		httpx.WriteError(w, http.StatusForbidden, "vendor_assessment_not_authorized", "Your current role cannot complete this due-diligence action.")
	case errors.Is(err, thirdparty.ErrNotFound), errors.Is(err, monitoring.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_assessment_not_found", "This due-diligence record was not found in your current legal-entity scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_changed", "This due-diligence record changed. Reload it before continuing.")
	case errors.Is(err, thirdparty.ErrInvalidAssessmentTransition):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_action_unavailable", "This due-diligence action is not available in the current state. Reload the record.")
	case errors.Is(err, monitoring.ErrInactive):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_form_inactive", "The selected due-diligence form is no longer active. Select the current approved form.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_assessment_invalid", "Check the due-diligence details and current version, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_assessment_failed", "The due-diligence action could not be completed. Review the record before retrying.")
	}
}
