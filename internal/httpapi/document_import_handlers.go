package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) documentImportService(w http.ResponseWriter) (*documentimport.Service, bool) {
	if a.deps.DocumentImports == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "document_import_unavailable", "Document import is unavailable.")
		return nil, false
	}
	return a.deps.DocumentImports, true
}

func (a *API) listDocumentImports(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.List(r.Context(), actor.TenantID, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "document_imports_failed", "Imported documents could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createDocumentImport(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	maximum := a.deps.MaxArtifactBytes
	if maximum <= 0 {
		maximum = 20 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximum+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "document_invalid", "The document could not be read or exceeds the allowed size.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "file_required", "A document file is required.")
		return
	}
	defer file.Close()
	mediaType := strings.TrimSpace(header.Header.Get("Content-Type"))
	value, err := service.Import(r.Context(), documentimport.ImportInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, FileName: header.Filename, MediaType: mediaType,
		Purpose: r.FormValue("purpose"), SourceType: r.FormValue("source_type"), CreatedBy: actor.PrincipalID,
	}, file)
	switch {
	case errors.Is(err, documentimport.ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "document_too_large", "The document exceeds the configured size limit.")
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "document_import_failed", err.Error())
	default:
		// The in-memory runtime extracts synchronously, so make its comparison
		// immediately available. PostgreSQL imports remain pending here and are
		// processed durably by the outbox worker.
		if a.deps.Coverage != nil && value.ExtractionStatus == documentimport.ExtractionExtracted {
			if _, coverageErr := a.deps.Coverage.Process(r.Context(), actor.TenantID, value.ID); coverageErr != nil && a.deps.Logger != nil {
				a.deps.Logger.WarnContext(r.Context(), "document coverage comparison failed after import", "document_id", value.ID, "error", coverageErr)
			}
		}
		httpx.WriteJSON(w, http.StatusCreated, value)
	}
}

func (a *API) getDocumentImport(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	value, err := service.Get(r.Context(), actor.TenantID, r.PathValue("id"))
	if errors.Is(err, documentimport.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Imported document not found.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "document_import_failed", "The imported document could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) reviewDocumentProposal(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input struct {
		Status          documentimport.ProposalStatus `json:"status"`
		Note            string                        `json:"note,omitempty"`
		ExpectedVersion int64                         `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.ReviewProposal(r.Context(), documentimport.ReviewInput{
		TenantID: actor.TenantID, DocumentID: r.PathValue("id"), ProposalID: r.PathValue("proposal_id"), ReviewerID: actor.PrincipalID,
		Status: input.Status, Note: input.Note, ExpectedVersion: input.ExpectedVersion,
	})
	switch {
	case errors.Is(err, documentimport.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Document proposal not found.")
	case errors.Is(err, documentimport.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This document changed. Reload it before recording the review.")
	case errors.Is(err, documentimport.ErrInvalidReview):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "review_invalid", "Choose accept or reject and use the current document version.")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "review_failed", "The review could not be recorded.")
	default:
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}
