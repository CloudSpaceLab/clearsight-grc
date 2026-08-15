package httpapi

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) evidenceService(w http.ResponseWriter) (*evidence.Service, bool) {
	if a.deps.Evidence == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "evidence_unavailable", "Evidence services are unavailable.")
		return nil, false
	}
	return a.deps.Evidence, true
}

func (a *API) listEvidenceSources(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListSources(r.Context(), tenant, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "sources_failed", "Sources could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createEvidenceSource(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.CreateSourceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateSource(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "source_invalid", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) recordEvidenceSourceObservation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, identityErr := identity.Require(r.Context())
	if identityErr != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input evidence.SourceObservation
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.TenantID = actor.TenantID
	input.SourceID = r.PathValue("id")
	input.RecordedBy = actor.PrincipalID
	value, err := service.RecordSourceObservation(r.Context(), input)
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Source not found.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "observation_invalid", "Source observation is invalid.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}

func (a *API) listEvidenceRequests(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	filter := evidence.RequestFilter{TenantID: r.URL.Query().Get("tenant_id"), Status: evidence.RequestStatus(r.URL.Query().Get("status"))}
	filter.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListRequests(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "requests_failed", "Requests could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) getEvidenceRequest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := service.GetRequest(r.Context(), tenant, r.PathValue("id"))
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Request not found.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "request_failed", "Request could not be loaded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}

func (a *API) createEvidenceRequest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.CreateRequestInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateRequest(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "request_invalid", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) declareEvidenceWrongRecipient(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID        string `json:"tenant_id"`
		ActorID         string `json:"actor_id"`
		Reason          string `json:"reason"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.DeclareWrongRecipient(r.Context(), input.TenantID, r.PathValue("id"), input.ActorID, input.Reason, input.ExpectedVersion)
	writeEvidenceRequestMutation(w, value, err)
}

func (a *API) reassignEvidenceRecipient(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID        string `json:"tenant_id"`
		ActorID         string `json:"actor_id"`
		Recipient       string `json:"recipient_principal_id"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.ReassignRecipient(r.Context(), input.TenantID, r.PathValue("id"), input.ActorID, input.Recipient, input.ExpectedVersion)
	writeEvidenceRequestMutation(w, value, err)
}

func (a *API) submitEvidenceRequest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.SubmissionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.Submit(r.Context(), input)
	writeEvidenceRequestMutation(w, value, err)
}

func (a *API) issueEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.IssueInvitationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.RequestID = r.PathValue("id")
	value, err := service.IssueInvitation(r.Context(), input)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) redeemEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct{ Token string `json:"token"` }
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.RedeemInvitation(r.Context(), input.Token)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) revokeEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID string `json:"tenant_id"`
		ActorID  string `json:"actor_id"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := service.RevokeInvitation(r.Context(), input.TenantID, r.PathValue("id"), input.ActorID); err != nil {
		writeEvidenceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) getEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token := bearerToken(r)
	value, err := service.GetCaptureSession(r.Context(), token)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) submitEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token := bearerToken(r)
	var input evidence.CaptureSubmissionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.SubmitCaptureSession(r.Context(), token, input)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) revokeEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct {
		TenantID string `json:"tenant_id"`
		ActorID  string `json:"actor_id"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := service.RevokeCaptureSession(r.Context(), input.TenantID, r.PathValue("id"), input.ActorID); err != nil {
		writeEvidenceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) uploadEvidenceArtifact(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(a.deps.MaxArtifactBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "multipart_invalid", "Upload could not be read.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "file_missing", "file is required")
		return
	}
	defer file.Close()
	input := evidence.UploadArtifactInput{
		TenantID: strings.TrimSpace(r.FormValue("tenant_id")),
		ActorID:  strings.TrimSpace(r.FormValue("actor_id")),
		FileName: header.Filename,
		MediaType: header.Header.Get("Content-Type"),
		Reader:   file,
	}
	value, err := service.UploadArtifact(r.Context(), input)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) uploadEvidenceArtifactCapabilities(r *http.Request) (string, error) {
	if a.deps.Evidence == nil {
		return "", fmt.Errorf("evidence services are unavailable")
	}
	return a.deps.Evidence.SessionTenant(r.Context(), bearerToken(r))
}

func (a *API) issueCapture(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.IssueCaptureInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.IssueCapture(r.Context(), input)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func writeEvidenceRequestMutation(w http.ResponseWriter, value evidence.Request, err error) {
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Request not found.")
	case errors.Is(err, evidence.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This request changed since you opened it. Reload and retry.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "request_invalid", err.Error())
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}

func writeEvidenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The requested evidence resource was not found.")
	case errors.Is(err, evidence.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This evidence resource changed since it was loaded. Reload and retry.")
	case errors.Is(err, evidence.ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "This action is not permitted for the current actor.")
	case errors.Is(err, evidence.ErrGone):
		httpx.WriteError(w, http.StatusGone, "gone", "This capture link is no longer active.")
	case errors.Is(err, evidence.ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "artifact_too_large", "This file is larger than the allowed upload limit.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "evidence_invalid", err.Error())
	}
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func multipartFile(r *http.Request, name string) (multipart.File, *multipart.FileHeader, error) {
	return r.FormFile(name)
}
