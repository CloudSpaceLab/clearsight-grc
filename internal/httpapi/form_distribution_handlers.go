package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type createFormDistributionRequest struct {
	TenantID            string                                `json:"tenant_id,omitempty"`
	LegalEntityID       string                                `json:"legal_entity_id,omitempty"`
	FormTemplateID      string                                `json:"form_template_id"`
	FormTemplateVersion int64                                 `json:"form_template_version"`
	SubjectType         string                                `json:"subject_type"`
	SubjectID           string                                `json:"subject_id"`
	Title               string                                `json:"title"`
	Purpose             string                                `json:"purpose"`
	AccessPolicy        evidence.AccessPolicy                 `json:"access_policy"`
	EstimatedMinutes    int                                   `json:"estimated_minutes"`
	Deadline            time.Time                             `json:"deadline"`
	RouteExpiresAt      time.Time                             `json:"route_expires_at"`
	ReminderPolicy      map[string]any                        `json:"reminder_policy,omitempty"`
	Recipients          []evidence.DistributionRecipientInput `json:"recipients"`
}

type amendFormDistributionRequest struct {
	TenantID        string          `json:"tenant_id,omitempty"`
	LegalEntityID   string          `json:"legal_entity_id,omitempty"`
	ExpectedVersion int64           `json:"expected_version"`
	Deadline        *time.Time      `json:"deadline,omitempty"`
	RouteExpiresAt  *time.Time      `json:"route_expires_at,omitempty"`
	ReminderPolicy  *map[string]any `json:"reminder_policy,omitempty"`
}

type distributionVersionRequest struct {
	TenantID        string `json:"tenant_id,omitempty"`
	LegalEntityID   string `json:"legal_entity_id,omitempty"`
	ExpectedVersion int64  `json:"expected_version"`
}

type accessStartRequest struct {
	RouteSelector string `json:"route_selector"`
}
type accessOTPSendRequest struct {
	RouteSelector     string `json:"route_selector"`
	RecipientSelector string `json:"recipient_selector"`
}
type accessOTPVerifyRequest struct {
	RouteSelector string `json:"route_selector"`
	ChallengeID   string `json:"challenge_id"`
	Code          string `json:"code"`
}
type accessRedeemRequest struct {
	RouteSelector string `json:"route_selector"`
}

func (a *API) formDistributionService(w http.ResponseWriter) (*evidence.DistributionService, bool) {
	if a.deps.FormDistributions == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_distribution_unavailable", "Form distribution services are unavailable.")
		return nil, false
	}
	return a.deps.FormDistributions, true
}
func (a *API) formDistributionAccessService(w http.ResponseWriter) (*evidence.DistributionAccessService, bool) {
	if a.deps.FormDistributionAccess == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_access_unavailable", "Secure form access is unavailable.")
		return nil, false
	}
	return a.deps.FormDistributionAccess, true
}
func (a *API) listFormDistributions(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_distribution_filter", "Choose a page size from 1 to 100 distributions.")
			return
		}
		limit = value
	}
	page, err := service.List(r.Context(), evidence.DistributionListQuery{TenantID: actor.TenantID, LegalEntityID: legalEntityID, Status: evidence.DistributionStatus(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))), Limit: limit, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
func (a *API) createFormDistribution(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var request createFormDistributionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The distribution request must be valid JSON.")
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, request.LegalEntityID)
	if !ok {
		return
	}
	bundle, err := service.Create(r.Context(), evidence.CreateDistributionInput{TenantID: actor.TenantID, LegalEntityID: legalEntityID, FormTemplateID: request.FormTemplateID, FormTemplateVersion: request.FormTemplateVersion, SubjectType: request.SubjectType, SubjectID: request.SubjectID, Title: request.Title, Purpose: request.Purpose, AccessPolicy: request.AccessPolicy, EstimatedMinutes: request.EstimatedMinutes, Deadline: request.Deadline, RouteExpiresAt: request.RouteExpiresAt, ReminderPolicy: request.ReminderPolicy, CreatedBy: actor.PrincipalID, Recipients: request.Recipients})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, distributionBundleJSON(bundle))
}
func (a *API) getFormDistribution(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	bundle, err := service.Get(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"))
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, distributionBundleJSON(bundle))
}
func (a *API) amendFormDistribution(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	var request amendFormDistributionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The amendment request must be valid JSON.")
		return
	}
	value, err := service.Amend(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), evidence.AmendDistributionInput{ExpectedVersion: request.ExpectedVersion, Deadline: request.Deadline, RouteExpiresAt: request.RouteExpiresAt, ReminderPolicy: request.ReminderPolicy})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"distribution": distributionBundleJSON(value.Bundle), "impact": value.Impact})
}
func (a *API) rotateFormDistributionAccessRoute(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	var request distributionVersionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil || request.ExpectedVersion < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "A current distribution version is required.")
		return
	}
	bundle, err := service.Get(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"))
	if err != nil || bundle.Distribution.Version != request.ExpectedVersion {
		writeFormDistributionError(w, evidence.ErrDistributionConflict)
		return
	}
	issued, err := access.RotateDistributionAccessRoute(r.Context(), actor.TenantID, legalEntityID, bundle.Distribution.ID, r.PathValue("route_id"), actor.PrincipalID)
	if err != nil {
		writeFormAccessAdminError(w, err)
		return
	}
	secureNoStore(w)
	httpx.WriteJSON(w, http.StatusOK, issued)
}
func (a *API) lockFormDistribution(w http.ResponseWriter, r *http.Request) {
	a.transitionFormDistribution(w, r, "lock")
}
func (a *API) reopenFormDistribution(w http.ResponseWriter, r *http.Request) {
	a.transitionFormDistribution(w, r, "reopen")
}
func (a *API) revokeFormDistribution(w http.ResponseWriter, r *http.Request) {
	a.transitionFormDistribution(w, r, "revoke")
}
func (a *API) transitionFormDistribution(w http.ResponseWriter, r *http.Request, action string) {
	service, actor, legalEntityID, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	var request distributionVersionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil || request.ExpectedVersion < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "A current distribution version is required.")
		return
	}
	var bundle evidence.DistributionBundle
	var err error
	switch action {
	case "lock":
		bundle, err = service.Lock(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), request.ExpectedVersion, actor.PrincipalID)
	case "reopen":
		bundle, err = service.Reopen(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), request.ExpectedVersion, actor.PrincipalID)
	case "revoke":
		bundle, err = service.Revoke(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), request.ExpectedVersion, actor.PrincipalID)
	default:
		err = evidence.ErrDistributionInvalid
	}
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, distributionBundleJSON(bundle))
}
func (a *API) supersedeFormDistribution(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	httpx.WriteError(w, http.StatusConflict, "supersede_preview_required", "Preview compatible answers against the replacement form revision before confirming supersession.")
}
func (a *API) startFormAccess(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var request accessStartRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	value, err := access.StartDistributionAccess(r.Context(), request.RouteSelector)
	if err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) sendFormAccessOTP(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var request accessOTPSendRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	value, err := access.SendOTP(r.Context(), request.RouteSelector, request.RecipientSelector)
	if err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) verifyFormAccessOTP(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var request accessOTPVerifyRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	value, err := access.VerifyOTP(r.Context(), request.RouteSelector, request.ChallengeID, request.Code)
	if err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) redeemFormAccess(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var request accessRedeemRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	value, err := access.RedeemDirectRoute(r.Context(), request.RouteSelector)
	if err != nil {
		writeGenericFormAccessFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) getFormResponseWorkspace(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	token := distributionBearerToken(r)
	session, request, err := access.SessionRequest(r.Context(), token)
	if err != nil {
		writeGenericFormSessionFailure(w)
		return
	}
	workspace, err := access.GetResponseWorkspace(r.Context(), token)
	if err != nil {
		writeGenericFormSessionFailure(w)
		return
	}
	recoveryContext, err := access.ResponseRecoveryContext(r.Context(), session)
	if err != nil {
		writeGenericFormSessionFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"session": session, "request": request, "workspace": workspace, "recovery_context": recoveryContext})
}
func (a *API) saveFormResponseWorkspace(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var input evidence.SaveWorkspaceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "workspace_invalid", "The response update is invalid.")
		return
	}
	value, err := access.SaveResponseWorkspace(r.Context(), distributionBearerToken(r), input)
	if err != nil {
		var conflict evidence.WorkspaceConflict
		if errors.As(err, &conflict) {
			httpx.WriteJSON(w, http.StatusConflict, conflict)
			return
		}
		writeGenericFormSessionFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
func (a *API) submitFormResponseWorkspace(w http.ResponseWriter, r *http.Request) {
	access, ok := a.formDistributionAccessService(w)
	if !ok {
		return
	}
	secureNoStore(w)
	var input evidence.SubmitWorkspaceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "submission_invalid", "The response submission is invalid.")
		return
	}
	value, err := access.SubmitResponseWorkspace(r.Context(), distributionBearerToken(r), input)
	if err != nil {
		var conflict evidence.WorkspaceConflict
		if errors.As(err, &conflict) {
			httpx.WriteJSON(w, http.StatusConflict, conflict)
			return
		}
		writeGenericFormSessionFailure(w)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}
func (a *API) distributionMutationContext(w http.ResponseWriter, r *http.Request) (*evidence.DistributionService, identity.Actor, string, bool) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return nil, identity.Actor{}, "", false
	}
	return service, actor, legalEntityID, true
}
func distributionActor(w http.ResponseWriter, r *http.Request) (identity.Actor, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to work with form distributions.")
		return identity.Actor{}, false
	}
	return actor, true
}
func distributionLegalEntity(w http.ResponseWriter, _ *http.Request, actor identity.Actor, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if actor.LegalEntityID != "*" {
		if requested != "" && requested != actor.LegalEntityID {
			httpx.WriteError(w, http.StatusForbidden, "legal_entity_not_allowed", "This distribution is outside your signed-in legal-entity scope.")
			return "", false
		}
		return actor.LegalEntityID, true
	}
	if requested == "" || requested == "*" {
		httpx.WriteError(w, http.StatusBadRequest, "legal_entity_required", "Select one legal entity before working with form distributions.")
		return "", false
	}
	return requested, true
}
func distributionBearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}
func secureNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
}
func writeGenericFormAccessFailure(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, "form_access_failed", "The secure form link could not be verified.")
}
func writeGenericFormSessionFailure(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, "form_session_invalid", "This secure form session is no longer available.")
}
func writeFormAccessAdminError(w http.ResponseWriter, _ error) {
	httpx.WriteError(w, http.StatusConflict, "form_access_conflict", "The secure access route could not be changed. Refresh the distribution and try again.")
}
func writeFormDistributionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "distribution_not_found", "The form distribution was not found in this legal entity.")
	case errors.Is(err, evidence.ErrDistributionConflict):
		httpx.WriteError(w, http.StatusConflict, "distribution_conflict", "The form distribution changed. Refresh before trying again.")
	case errors.Is(err, evidence.ErrDistributionInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "distribution_invalid", "The form distribution is invalid. Check its form revision, recipients and dates.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "distribution_failed", "The form distribution could not be completed.")
	}
}
func distributionBundleJSON(bundle evidence.DistributionBundle) map[string]any {
	return map[string]any{"distribution": map[string]any{"id": bundle.Distribution.ID, "tenant_id": bundle.Distribution.TenantID, "legal_entity_id": bundle.Distribution.LegalEntityID, "form_template_id": bundle.Distribution.FormTemplateID, "form_template_version": bundle.Distribution.FormTemplateVersion, "subject_type": bundle.Distribution.SubjectType, "subject_id": bundle.Distribution.SubjectID, "title": bundle.Distribution.Title, "purpose": bundle.Distribution.Purpose, "access_policy": bundle.Distribution.AccessPolicy, "status": bundle.Distribution.Status, "deadline": bundle.Distribution.Deadline, "route_expires_at": bundle.Distribution.RouteExpiresAt, "reminder_policy": bundle.Distribution.ReminderPolicy, "created_by": bundle.Distribution.CreatedBy, "version": bundle.Distribution.Version, "created_at": bundle.Distribution.CreatedAt, "updated_at": bundle.Distribution.UpdatedAt}, "recipients": bundle.Recipients, "workspace": bundle.Workspace}
}
