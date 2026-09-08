package httpapi

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func documentProtection(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
}
func documentQueryFromRequest(r *http.Request) (evidence.DocumentQuery, error) {
	v := r.URL.Query()
	q := evidence.DocumentQuery{FileKind: evidence.DocumentFileKind(v.Get("file_kind")), Query: v.Get("query"), FormTemplateID: v.Get("form_template_id"), RelationshipID: v.Get("relationship_id"), ResponseRevisionID: v.Get("response_revision_id"), CurrentOnly: true, Cursor: v.Get("cursor"), Limit: 25}
	var err error
	if v.Has("limit") {
		q.Limit, err = strconv.Atoi(v.Get("limit"))
		if err != nil {
			return q, evidence.ErrDistributionInvalid
		}
	}
	if v.Has("current_only") {
		q.CurrentOnly, err = strconv.ParseBool(v.Get("current_only"))
		if err != nil {
			return q, evidence.ErrDistributionInvalid
		}
	}
	return q, nil
}
func documentScope(w http.ResponseWriter, r *http.Request, q *evidence.DocumentQuery) bool {
	actor, ok := distributionActor(w, r)
	if !ok {
		return false
	}
	requested := ""
	if actor.LegalEntityID == "*" {
		requested = r.URL.Query().Get("legal_entity_id")
	}
	entity, ok := distributionLegalEntity(w, r, actor, requested)
	if !ok {
		return false
	}
	q.TenantID = actor.TenantID
	q.LegalEntityID = entity
	q.PrincipalID = actor.PrincipalID
	return true
}
func (a *API) listFormDocuments(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	q, err := documentQueryFromRequest(r)
	if err != nil {
		writeDocumentError(w, err)
		return
	}
	if !documentScope(w, r, &q) {
		return
	}
	page, err := service.ListDocuments(r.Context(), q)
	if err != nil {
		writeDocumentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
func (a *API) openFormDocument(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	q := evidence.DocumentQuery{ResponseRevisionID: r.URL.Query().Get("response_revision_id"), SubmissionID: r.PathValue("submission_id"), FieldID: r.PathValue("field_id"), ArtifactID: r.PathValue("artifact_id"), Limit: 1}
	if !documentScope(w, r, &q) {
		return
	}
	if q.SubmissionID == "" || q.FieldID == "" || q.ArtifactID == "" {
		writeDocumentError(w, evidence.ErrNotFound)
		return
	}
	page, err := service.ListDocuments(r.Context(), q)
	if err != nil {
		writeDocumentError(w, err)
		return
	}
	if len(page.Items) != 1 {
		writeDocumentError(w, evidence.ErrNotFound)
		return
	}
	v := page.Items[0]
	if v.ArtifactStatus != evidence.ArtifactAvailable && !v.DemoPreviewAvailable {
		writeDocumentError(w, evidence.ErrNotFound)
		return
	}
	capture, ok := a.evidenceService(w)
	if !ok {
		return
	}
	open := capture.OpenArtifact
	if v.DemoPreviewAvailable {
		open = capture.OpenDemoSampleArtifact
	}
	artifact, reader, err := open(r.Context(), q.TenantID, v.ArtifactRequestID, v.ArtifactID)
	if err != nil {
		writeDocumentError(w, err)
		return
	}
	defer reader.Close()
	if artifact.SHA256 != v.SHA256 || artifact.SizeBytes != v.SizeBytes {
		writeDocumentError(w, evidence.ErrNotFound)
		return
	}
	media, _, err := mime.ParseMediaType(strings.ToLower(strings.TrimSpace(artifact.MediaType)))
	if err != nil {
		media = "application/octet-stream"
	}
	disposition := "attachment"
	if r.URL.Query().Get("download") != "true" && (media == "application/pdf" || media == "image/png" || media == "image/jpeg" || media == "image/gif" || media == "image/webp") {
		disposition = "inline"
	}
	filename := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '/' || r == '\\' {
			return '_'
		}
		return r
	}, artifact.FileName)
	w.Header().Set("Content-Type", media)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprint(artifact.SizeBytes))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
func writeDocumentError(w http.ResponseWriter, err error) {
	if err == evidence.ErrDistributionInvalid {
		httpx.WriteError(w, 422, "document_filters_invalid", "Check the file filters and try again.")
		return
	}
	if err == evidence.ErrNotFound {
		httpx.WriteError(w, 404, "document_unavailable", "This file is unavailable. Refresh the file list or contact the request owner.")
		return
	}
	httpx.WriteError(w, 500, "documents_unavailable", "Files could not be loaded. Try again.")
}
