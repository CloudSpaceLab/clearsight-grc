package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// These handlers are intentionally not registered. PostgreSQL transaction,
// authorization and revocation integration tests are required before release.
func (a *API) internalWorkspaceAccess(w http.ResponseWriter, r *http.Request) (*evidence.DistributionAccessService, bool) {
	secureNoStore(w)
	if _, err := identity.Require(r.Context()); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to respond to this form.")
		return nil, false
	}
	if len(r.URL.Query()) != 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the form request and try again.")
		return nil, false
	}
	if a.deps.FormDistributionAccess == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_unavailable", "Form unavailable. Try again.")
		return nil, false
	}
	return a.deps.FormDistributionAccess, true
}

func (a *API) getInternalFormWorkspace(w http.ResponseWriter, r *http.Request) {
	service, ok := a.internalWorkspaceAccess(w, r)
	if !ok {
		return
	}
	request, workspace, err := service.GetInternalWorkspace(r.Context(), r.PathValue("request_id"))
	if err != nil {
		writeInternalWorkspaceFailure(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"request": request, "workspace": workspace})
}

func (a *API) saveInternalFormWorkspace(w http.ResponseWriter, r *http.Request) {
	service, ok := a.internalWorkspaceAccess(w, r)
	if !ok {
		return
	}
	var input evidence.SaveWorkspaceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "workspace_invalid", "Check the response and try again.")
		return
	}
	workspace, err := service.SaveInternalWorkspace(r.Context(), r.PathValue("request_id"), input)
	if err != nil {
		writeInternalWorkspaceFailure(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspace)
}

func (a *API) submitInternalFormWorkspace(w http.ResponseWriter, r *http.Request) {
	service, ok := a.internalWorkspaceAccess(w, r)
	if !ok {
		return
	}
	var input evidence.SubmitWorkspaceInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "submission_invalid", "Check the response and try again.")
		return
	}
	result, err := service.SubmitInternalWorkspace(r.Context(), r.PathValue("request_id"), input)
	if err != nil {
		writeInternalWorkspaceFailure(w, err)
		return
	}
	revision := struct {
		responseRevisionResponse
		SubmissionID string `json:"submission_id"`
	}{responseRevisionJSON(result.Revision), result.Revision.SubmissionID}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"workspace": result.Workspace, "revision": revision, "submission": result.Submission})
}

// This upload handler is deliberately unregistered until workspace authority
// and upload recovery have passed the release checks.
func (a *API) uploadInternalFormWorkspaceDocument(w http.ResponseWriter, r *http.Request) {
	access, ok := a.internalWorkspaceAccess(w, r)
	if !ok {
		return
	}
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
		httpx.WriteError(w, http.StatusBadRequest, "document_invalid", "Check the document and try again.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if len(r.MultipartForm.Value) != 1 || len(r.MultipartForm.Value["field_id"]) != 1 || strings.TrimSpace(r.FormValue("field_id")) == "" || len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		httpx.WriteError(w, http.StatusBadRequest, "document_invalid", "Check the document and try again.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "document_invalid", "Check the document and try again.")
		return
	}
	defer file.Close()
	value, err := service.StoreArtifactForInternalWorkspace(r.Context(), access, r.PathValue("request_id"), evidence.ArtifactInput{FieldID: r.FormValue("field_id"), FileName: header.Filename, MediaType: multipartMediaType(header)}, file)
	if err != nil {
		writeInternalWorkspaceFailure(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": value.ID, "request_id": value.RequestID, "file_name": value.FileName, "media_type": value.MediaType, "size_bytes": value.SizeBytes, "sha256": value.SHA256, "status": value.Status, "demo_unscanned_allowed": value.DemoUnscannedAllowed})
}

func writeInternalWorkspaceFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, evidence.ErrArtifactOutcomeUnknown) {
		httpx.WriteError(w, http.StatusServiceUnavailable, "document_outcome_unknown", "Document status unknown. Contact the requester.")
		return
	}
	var conflict evidence.WorkspaceConflict
	if errors.As(err, &conflict) {
		httpx.WriteJSON(w, http.StatusConflict, conflict)
		return
	}
	httpx.WriteError(w, http.StatusNotFound, "form_unavailable", "Form unavailable. Contact the requester.")
}
