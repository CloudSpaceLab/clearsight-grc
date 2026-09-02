package httpapi

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
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
	actor, identityErr := identity.Require(r.Context())
	if identityErr != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	tenant := actor.TenantID
	legalEntityID := actor.LegalEntityID
	if legalEntityID == "*" {
		legalEntityID = strings.TrimSpace(r.URL.Query().Get("legal_entity_id"))
	}
	if legalEntityID == "" || legalEntityID == "*" {
		httpx.WriteError(w, http.StatusBadRequest, "source_scope_required", "Select one legal entity before loading evidence sources.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := service.ListSourcesForEntity(r.Context(), evidence.SourceListQuery{TenantID: tenant, LegalEntityID: legalEntityID, Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if errors.Is(err, evidence.ErrSourceScopeRequired) || errors.Is(err, evidence.ErrSourceScopeAmbiguous) {
		httpx.WriteError(w, http.StatusNotFound, "source_scope_not_found", "Evidence sources are unavailable for the selected legal entity.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "sources_failed", "Sources could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
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
	if a.deps.SourceCatalog != nil {
		if err := a.deps.SourceCatalog.RegisterSourceScope(r.Context(), sourceaccess.SourceScope{TenantID: value.TenantID, SourceID: value.ID}); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "source_registration_failed", "The source was created but could not be prepared for connection setup.")
			return
		}
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
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListManageableRequestsForEntity(r.Context(), evidence.ActorRequestScope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ActorPrincipalID: actor.PrincipalID,
	}, limit)
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
	if errors.Is(err, evidence.ErrSubjectUnsupported) || errors.Is(err, evidence.ErrSubjectScopeMismatch) || errors.Is(err, evidence.ErrSubjectAccessDenied) {
		httpx.WriteError(w, http.StatusNotFound, "subject_not_found", "The selected record is not available for an evidence request.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "request_invalid", "The evidence request is not valid. Check the recipient, deadline and required information.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getEvidenceRequest(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, identityErr := identity.Require(r.Context())
	if identityErr != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	value, err := service.GetRequestForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, r.PathValue("id"))
	if evidenceRequestUnavailable(err) {
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
	if value.CreatedBy != actor.PrincipalID && evidence.RequestAssignedTo(value, actor.PrincipalID) {
		value = evidence.RespondentRequest(value)
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
	if strings.EqualFold(strings.TrimSpace(input.Channel), "INTERNAL") {
		actor, identityErr := identity.Require(r.Context())
		if identityErr != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required to submit assigned evidence.")
			return
		}
		request, requestErr := service.GetRequestForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, input.RequestID)
		if requestErr != nil || !a.canReadEvidenceRequest(r.Context(), request) || !evidence.RequestAssignedTo(request, actor.PrincipalID) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
			return
		}
		input.TenantID = actor.TenantID
		input.SubmittedBy = actor.PrincipalID
	}
	receipt, err := service.Submit(r.Context(), input)
	if err == nil && a.deps.Monitoring != nil {
		results, evaluateErr := a.deps.Monitoring.EvaluateSubmission(r.Context(), input.TenantID, receipt.SubmissionID)
		if evaluateErr != nil {
			if a.deps.Logger != nil {
				a.deps.Logger.Warn("monitoring evaluation after evidence submission failed", "tenant_id", input.TenantID, "submission_id", receipt.SubmissionID, "error", evaluateErr)
			}
		} else if len(results) > 0 {
			w.Header().Set("X-ClearSight-Monitoring-Results", strconv.Itoa(len(results)))
		}
	}
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
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrSubjectScopeMismatch), errors.Is(err, evidence.ErrSubjectAccessDenied), errors.Is(err, evidence.ErrRecipientMismatch), errors.Is(err, evidence.ErrRecipientManagerRequired):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invitation_invalid", "The invitation could not be issued. Check the current recipient and expiry, then try again.")
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

func (a *API) getEvidenceSessionDraft(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token, ok := bearerToken(w, r)
	if !ok {
		return
	}
	draft, err := service.GetDraft(r.Context(), token)
	switch {
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This access has ended. Ask the sender for a new link.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "draft_unavailable", "The saved response could not be loaded. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, evidenceDraftPayload(draft))
	}
}

func (a *API) saveEvidenceSessionDraft(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	token, ok := bearerToken(w, r)
	if !ok {
		return
	}
	var input evidence.SaveDraftInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "draft_invalid", "The saved response could not be read. Check the response and try again.")
		return
	}
	draft, err := service.SaveDraft(r.Context(), token, input)
	switch {
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This access has ended. Ask the sender for a new link.")
	case errors.Is(err, evidence.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "draft_conflict", "The saved response changed. Check the latest saved response before trying again.")
	case errors.Is(err, evidence.ErrDraftInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "draft_invalid", "The saved response contains an invalid answer. Review the response and try again.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "draft_unavailable", "The response could not be saved. Your entries remain on this screen; try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, evidenceDraftPayload(draft))
	}
}

func evidenceDraftPayload(draft evidence.ResponseDraft) map[string]any {
	payload := map[string]any{
		"answers":           draft.Answers,
		"presentation_mode": draft.PresentationMode,
		"version":           draft.Version,
	}
	if !draft.UpdatedAt.IsZero() {
		payload["updated_at"] = draft.UpdatedAt
	}
	return payload
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
		Answers         map[string]formcontract.AnswerValue `json:"answers"`
		ExpectedVersion int64                               `json:"expected_version"`
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
	canonicalSession := false
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
		if access := a.deps.FormDistributionAccess; access != nil {
			session, request, sessionErr := access.SessionRequest(r.Context(), token)
			if sessionErr == nil {
				tenant, requestID, createdBy, sessionToken = session.TenantID, request.ID, "", token
				canonicalSession = true
			}
		}
		if !canonicalSession {
			session, request, sessionErr := service.SessionRequest(r.Context(), token)
			if sessionErr != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
				return
			}
			tenant, requestID, createdBy, sessionToken = session.TenantID, request.ID, "", token
		}
	}
	if tenant == "" || requestID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "artifact_scope_required", "tenant_id and request_id are required.")
		return
	}
	mediaType := multipartMediaType(header)
	artifactInput := evidence.ArtifactInput{TenantID: tenant, RequestID: requestID, FieldID: strings.TrimSpace(r.FormValue("field_id")), SubmissionID: strings.TrimSpace(r.FormValue("submission_id")), FileName: header.Filename, MediaType: mediaType, CreatedBy: createdBy}
	var artifact evidence.Artifact
	if canonicalSession {
		artifact, err = service.StoreArtifactForDistributionSession(r.Context(), a.deps.FormDistributionAccess, sessionToken, artifactInput, file)
	} else {
		artifactInput.SessionToken = sessionToken
		artifact, err = service.StoreArtifact(r.Context(), artifactInput, file)
	}
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrRecipientMismatch):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open for uploads.")
	case errors.Is(err, evidence.ErrArtifactTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "artifact_too_large", "The file exceeds the limit for this request or question.")
	case errors.Is(err, evidence.ErrFileName):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "filename_invalid", "Rename the file with a supported extension and without path or control characters.")
	case errors.Is(err, evidence.ErrContentInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "file_content_invalid", "The file contents do not match a supported PDF, DOCX, XLSX, image or text file, or include active embedded content.")
	case errors.Is(err, evidence.ErrFieldInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "field_file_invalid", "The file does not meet this question's format or size rules.")
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
	case evidenceRequestUnavailable(err):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Evidence request not found.")
	case errors.Is(err, evidence.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "The request changed. Reload before submitting.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case errors.Is(err, evidence.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, "session_unavailable", "This capture session is unavailable.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "submission_invalid", "The evidence response is incomplete or no longer matches the current request. Reload the request and check each required answer.")
	default:
		httpx.WriteJSON(w, http.StatusOK, receipt)
	}
}

func evidenceRequestUnavailable(err error) bool {
	return errors.Is(err, evidence.ErrNotFound) ||
		errors.Is(err, evidence.ErrSubjectUnsupported) ||
		errors.Is(err, evidence.ErrSubjectScopeMismatch) ||
		errors.Is(err, evidence.ErrSubjectAccessDenied) ||
		errors.Is(err, evidence.ErrRecipientMismatch) ||
		errors.Is(err, evidence.ErrRecipientManagerRequired)
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
