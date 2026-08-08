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
	var input evidence.SourceObservation
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.SourceID = r.PathValue("id")
	value, err := service.RecordSourceObservation(r.Context(), input)
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Source not found.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "observation_invalid", err.Error())
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}

func (a *API) listEvidenceRequests(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	actor, authenticated := identity.FromContext(r.Context())
	if !authenticated || actor.TenantID != tenant {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence requests not found.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListManageableRequests(r.Context(), tenant, actor.PrincipalID, limit, func(value evidence.Request) bool {
		return a.canReadEvidenceRequest(r.Context(), value)
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "requests_failed", "Evidence requests could not be loaded.")
		return
	}
	values = a.filterEvidenceRequests(r.Context(), values)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
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
	if errors.Is(err, evidence.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "request_failed", "Evidence request could not be loaded.")
		return
	}
	if !a.canReadEvidenceRequest(r.Context(), value) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) submitEvidenceRequest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.Submission
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.RequestID = r.PathValue("id")
	if input.Channel == "" {
		input.Channel = "INTERNAL"
	}
	receipt, err := service.Submit(r.Context(), input)
	writeEvidenceSubmissionResult(w, receipt, err)
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
	issued, err := service.IssueInvitation(r.Context(), input)
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invitation_invalid", err.Error())
	default:
		httpx.WriteJSON(w, http.StatusCreated, issued)
	}
}

func (a *API) redeemEvidenceInvitation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input struct {
		Token    string `json:"token"`
		Audience string `json:"audience"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	session, err := service.RedeemInvitation(r.Context(), input.Token, input.Audience)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invitation_unavailable", "This invitation is unavailable. Request a new invitation from the sender.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (a *API) getEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token, ok := bearerToken(w, r)
	if !ok {
		return
	}
	session, request, err := service.SessionRequest(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"session": session, "request": request})
}

func (a *API) submitEvidenceSession(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token, ok := bearerToken(w, r)
	if !ok {
		return
	}
	var input struct {
		Answers         map[string]string `json:"answers"`
		ExpectedVersion int64             `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	receipt, err := service.SubmitSession(r.Context(), token, input.Answers, input.ExpectedVersion)
	writeEvidenceSubmissionResult(w, receipt, err)
}

func (a *API) uploadEvidenceArtifact(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	maximum := a.deps.MaxArtifactBytes
	if maximum <= 0 {
		maximum = 20 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximum+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "artifact_invalid", "The upload could not be read or exceeds the allowed size.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "file_required", "A file is required.")
		return
	}
	defer file.Close()

	tenant := strings.TrimSpace(r.FormValue("tenant_id"))
	requestID := strings.TrimSpace(r.FormValue("request_id"))
	createdBy := strings.TrimSpace(r.FormValue("created_by"))
	sessionToken := ""
	if actor, authenticated := identity.FromContext(r.Context()); authenticated {
		if tenant != "" && tenant != actor.TenantID {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
			return
		}
		tenant = actor.TenantID
		createdBy = actor.PrincipalID
	} else {
		token := optionalBearerToken(r)
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "session_required", "A capture session is required.")
			return
		}
		session, request, sessionErr := service.SessionRequest(r.Context(), token)
		if sessionErr != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
			return
		}
		tenant, requestID, createdBy, sessionToken = session.TenantID, request.ID, "", token
	}
	if tenant == "" || requestID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "artifact_scope_required", "tenant_id and request_id are required.")
		return
	}
	mediaType := multipartMediaType(header)
	artifact, err := service.StoreArtifact(r.Context(), evidence.ArtifactInput{TenantID: tenant, RequestID: requestID, SubmissionID: strings.TrimSpace(r.FormValue("submission_id")), FileName: header.Filename, MediaType: mediaType, CreatedBy: createdBy, SessionToken: sessionToken}, file)
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrRecipientMismatch):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open for uploads.")
	case errors.Is(err, evidence.ErrArtifactTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "artifact_too_large", fmt.Sprintf("The file exceeds the %d-byte limit.", maximum))
	case errors.Is(err, evidence.ErrMediaType):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "media_type_not_allowed", "This file type is not allowed.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "artifact_failed", "The artifact could not be stored.")
	default:
		httpx.WriteJSON(w, http.StatusCreated, artifact)
	}
}

func writeEvidenceSubmissionResult(w http.ResponseWriter, receipt evidence.SubmissionReceipt, err error) {
	switch {
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "The request changed. Reload before submitting.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "submission_invalid", err.Error())
	default:
		httpx.WriteJSON(w, http.StatusOK, receipt)
	}
}

func bearerToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	token := optionalBearerToken(r)
	if token == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "session_required", "A capture session is required.")
		return "", false
	}
	return token, true
}

func optionalBearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) < 8 || !strings.EqualFold(value[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(value[7:])
}

func multipartMediaType(header *multipart.FileHeader) string {
	value := strings.TrimSpace(header.Header.Get("Content-Type"))
	if value == "" {
		return "application/octet-stream"
	}
	if semicolon := strings.Index(value, ";"); semicolon >= 0 {
		value = value[:semicolon]
	}
	return strings.ToLower(strings.TrimSpace(value))
}
