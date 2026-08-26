package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
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

type reissueVendorAssessmentRequest struct {
	thirdparty.ReissueAssessmentRequestInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type retryVendorAssessmentSetupRequest struct {
	thirdparty.RetryAssessmentSetupInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type startVendorAssessmentReviewRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	TenantID        string `json:"tenant_id,omitempty"`
	LegalEntityID   string `json:"legal_entity_id,omitempty"`
	ActorID         string `json:"actor_id,omitempty"`
}

type completeVendorAssessmentRequest struct {
	thirdparty.CompleteAssessmentInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type reviewVendorAssessmentDocumentRequest struct {
	thirdparty.ReviewAssessmentDocumentInput
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
	if a.deps.ThirdPartyAssessmentSetup != nil && assessment.Status == thirdparty.AssessmentSetupPending {
		_, _ = a.deps.ThirdPartyAssessmentSetup.Maintain(r.Context(), time.Now().UTC(), 1)
		if current, loadErr := service.GetAssessment(r.Context(), actor, assessment.ID); loadErr == nil {
			assessment = current
		}
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

func (a *API) reissueVendorAssessmentRequest(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentRequests == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_request_unavailable", "The replacement vendor invitation is temporarily unavailable. No invitation was issued.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to replace this vendor invitation.")
		return
	}
	var request reissueVendorAssessmentRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the recipient and invitation period, then try again.")
		return
	}
	outcome, err := a.deps.ThirdPartyAssessmentRequests.ReissueRequest(r.Context(), actor, r.PathValue("id"), request.ReissueAssessmentRequestInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, outcome)
}

func (a *API) retryVendorAssessmentSetup(w http.ResponseWriter, r *http.Request) {
	service, ok := a.assessmentService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to retry due-diligence setup.")
		return
	}
	var request retryVendorAssessmentSetupRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Reload the assessment and retry setup with its current version.")
		return
	}
	outcome, err := service.RetryAssessmentSetup(r.Context(), actor, r.PathValue("id"), request.RetryAssessmentSetupInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, outcome)
}

func (a *API) getVendorAssessmentReview(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentReviews == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_assessment_review_unavailable", "The vendor response cannot be loaded right now; try loading the assessment again.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to review this vendor response.")
		return
	}
	view, err := a.deps.ThirdPartyAssessmentReviews.GetReview(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (a *API) startVendorAssessmentReview(w http.ResponseWriter, r *http.Request) {
	service, ok := a.assessmentService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to start this vendor review.")
		return
	}
	var request startVendorAssessmentReviewRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The review request is incomplete. Reload the assessment, then start the vendor review again.")
		return
	}
	assessment, err := service.StartAssessmentReview(r.Context(), actor, r.PathValue("id"), request.ExpectedVersion)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, assessment)
}

func (a *API) completeVendorAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.assessmentService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to complete this vendor assessment.")
		return
	}
	var request completeVendorAssessmentRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the conclusion, rationale and next review date, then try again.")
		return
	}
	assessment, err := service.CompleteAssessment(r.Context(), actor, r.PathValue("id"), request.CompleteAssessmentInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, assessment)
}

func (a *API) reviewVendorAssessmentDocument(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentReviews == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_document_review_unavailable", "The document decision cannot be recorded right now. No review state was changed.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to review this vendor document.")
		return
	}
	var request reviewVendorAssessmentDocumentRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the document decision, evidence class and current assessment version, then try again.")
		return
	}
	view, err := a.deps.ThirdPartyAssessmentReviews.ReviewDocument(r.Context(), actor, r.PathValue("id"), r.PathValue("artifact_id"), request.ReviewAssessmentDocumentInput)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func writeThirdPartyAssessmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrMissingIdentity):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, thirdparty.ErrAssessmentIdentityMismatch), errors.Is(err, thirdparty.ErrAssessmentAuthorityUnavailable):
		httpx.WriteError(w, http.StatusForbidden, "vendor_assessment_not_authorized", "Your current role cannot complete this due-diligence action.")
	case errors.Is(err, thirdparty.ErrNotFound), errors.Is(err, monitoring.ErrNotFound), errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_assessment_not_found", "This due-diligence record was not found in your current legal-entity scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_changed", "This due-diligence record changed. Reload it before continuing.")
	case errors.Is(err, thirdparty.ErrInvalidAssessmentTransition):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_action_unavailable", "This due-diligence action is not available in the current state. Reload the record.")
	case errors.Is(err, thirdparty.ErrAssessmentCompletionBlocked):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_review_incomplete", "Required responses, available files and document decisions must be resolved before this assessment can be completed.")
	case errors.Is(err, thirdparty.ErrAssessmentReadinessUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_assessment_readiness_unavailable", "Completion checks are temporarily unavailable. No assessment conclusion was recorded.")
	case errors.Is(err, monitoring.ErrInactive):
		httpx.WriteError(w, http.StatusConflict, "vendor_assessment_form_inactive", "The selected due-diligence form is no longer active. Select the current approved form.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_assessment_invalid", "Check the due-diligence details and current version, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_assessment_failed", "The due-diligence action could not be completed. Review the record before retrying.")
	}
}
