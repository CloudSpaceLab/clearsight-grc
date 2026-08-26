package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type prepareVendorWorkRequest struct {
	thirdparty.PrepareVendorWorkInput
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
	ReviewerID    string `json:"reviewer_id,omitempty"`
}

type sendVendorWorkRequest struct {
	thirdparty.SendVendorWorkInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}
type startVendorWorkReviewRequest struct {
	thirdparty.StartVendorWorkReviewInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}
type requestVendorWorkChangesRequest struct {
	thirdparty.RequestVendorWorkChangesInput
	TenantID   string `json:"tenant_id,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	ReviewerID string `json:"reviewer_id,omitempty"`
}
type acceptVendorWorkRequest struct {
	thirdparty.AcceptVendorWorkInput
	TenantID   string `json:"tenant_id,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	ReviewerID string `json:"reviewer_id,omitempty"`
}
type cancelVendorWorkRequest struct {
	thirdparty.CancelVendorWorkInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}
type retryVendorWorkRequest struct {
	thirdparty.RetryVendorWorkInput
	TenantID string `json:"tenant_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
}

func (a *API) vendorWorkService(w http.ResponseWriter) (*thirdparty.VendorWorkService, bool) {
	if a.deps.ThirdPartyWork == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_work_unavailable", "Vendor requests are temporarily unavailable. No change was made.")
		return nil, false
	}
	return a.deps.ThirdPartyWork, true
}

func (a *API) prepareVendorWork(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorWorkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to prepare this vendor request.")
		return
	}
	var input prepareVendorWorkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the vendor request details and try again.")
		return
	}
	input.RelationshipID = r.PathValue("id")
	value, err := service.Prepare(r.Context(), actor, input.PrepareVendorWorkInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, value)
}

func (a *API) sendVendorWork(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input sendVendorWorkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the delivery details and try again.")
		return
	}
	value, err := service.Send(r.Context(), actor, work.ID, input.SendVendorWorkInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) startVendorWorkReview(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input startVendorWorkReviewRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the request version and try again.")
		return
	}
	value, err := service.StartReview(r.Context(), actor, work.ID, input.StartVendorWorkReviewInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) requestVendorWorkChanges(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input requestVendorWorkChangesRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the requested changes and try again.")
		return
	}
	value, err := service.RequestChanges(r.Context(), actor, work.ID, input.RequestVendorWorkChangesInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) acceptVendorWork(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input acceptVendorWorkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Add the review rationale and try again.")
		return
	}
	value, err := service.Accept(r.Context(), actor, work.ID, input.AcceptVendorWorkInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) cancelVendorWork(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input cancelVendorWorkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Add a cancellation reason and try again.")
		return
	}
	value, err := service.Cancel(r.Context(), actor, work.ID, input.CancelVendorWorkInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) retryVendorWork(w http.ResponseWriter, r *http.Request) {
	service, actor, work, ok := a.vendorWorkCommandContext(w, r)
	if !ok {
		return
	}
	var input retryVendorWorkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the delivery details and try again.")
		return
	}
	value, err := service.Retry(r.Context(), actor, work.ID, input.RetryVendorWorkInput)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) getVendorWork(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorWorkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view this vendor request.")
		return
	}
	value, err := service.Get(r.Context(), actor, r.PathValue("request_id"))
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) getVendorWorkResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorWorkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to review this vendor response.")
		return
	}
	view, err := service.Response(r.Context(), actor, r.PathValue("request_id"))
	if err != nil || view.Work.RelationshipID != r.PathValue("id") {
		writeVendorWorkError(w, thirdparty.ErrNotFound)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (a *API) listVendorWork(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorWorkService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view vendor requests.")
		return
	}
	limit, parseErr := strconv.Atoi(r.URL.Query().Get("limit"))
	if r.URL.Query().Get("limit") != "" && parseErr != nil {
		writeVendorWorkError(w, thirdparty.ErrInvalid)
		return
	}
	input := thirdparty.VendorWorkListInput{RelationshipID: strings.TrimSpace(r.URL.Query().Get("relationship_id")), TargetType: thirdparty.LinkTargetType(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("target_type")))), TargetID: strings.TrimSpace(r.URL.Query().Get("target_id")), Cursor: strings.TrimSpace(r.URL.Query().Get("cursor")), Limit: limit}
	page, err := service.List(r.Context(), actor, input)
	if err != nil {
		writeVendorWorkError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) vendorWorkCommandContext(w http.ResponseWriter, r *http.Request) (*thirdparty.VendorWorkService, thirdparty.Actor, thirdparty.VendorWorkRequest, bool) {
	service, ok := a.vendorWorkService(w)
	if !ok {
		return nil, thirdparty.Actor{}, thirdparty.VendorWorkRequest{}, false
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to change this vendor request.")
		return nil, thirdparty.Actor{}, thirdparty.VendorWorkRequest{}, false
	}
	work, err := service.Get(r.Context(), actor, r.PathValue("request_id"))
	if err != nil || work.RelationshipID != r.PathValue("id") {
		writeVendorWorkError(w, thirdparty.ErrNotFound)
		return nil, thirdparty.Actor{}, thirdparty.VendorWorkRequest{}, false
	}
	return service, actor, work, true
}

func writeVendorWorkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, thirdparty.ErrVendorWorkIdentityMismatch):
		httpx.WriteError(w, http.StatusForbidden, "vendor_work_not_allowed", "Your current authority does not allow this vendor request action.")
	case errors.Is(err, thirdparty.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_work_not_found", "The vendor request was not found in your current scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_work_conflict", "This vendor request changed. Reload it before trying again.")
	case errors.Is(err, thirdparty.ErrInvalidAssessmentTransition):
		httpx.WriteError(w, http.StatusConflict, "vendor_work_state_changed", "This action is not available in the current request state. Reload the request to continue.")
	case errors.Is(err, thirdparty.ErrVendorWorkAcceptanceBlocked):
		httpx.WriteError(w, http.StatusConflict, "vendor_work_acceptance_blocked", "A submitted document is pending inspection, quarantined or unavailable. Wait for inspection or request a replacement before accepting this response.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_work_invalid", "Check the request details, current version and due date, then try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_work_failed", "The vendor request could not be changed. Check its current state before retrying.")
	}
}
