package httpapi

import (
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func (a *API) openVendorWorkDocument(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorWorkService(w)
	if !ok || a.deps.Evidence == nil {
		if ok {
			httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_document_unavailable", "This document is temporarily unavailable.")
		}
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to open this document.")
		return
	}
	view, err := service.Response(r.Context(), actor, r.PathValue("request_id"))
	captureRequestID := r.PathValue("capture_request_id")
	if err != nil || view.Work.RelationshipID != r.PathValue("id") || !vendorWorkDocumentAvailable(view, captureRequestID, r.PathValue("artifact_id")) {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	artifact, reader, err := a.deps.Evidence.OpenArtifact(r.Context(), actor.TenantID, captureRequestID, r.PathValue("artifact_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", artifact.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": artifact.FileName}))
	w.Header().Set("Content-Length", strconv.FormatInt(artifact.SizeBytes, 10))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.CopyN(w, reader, artifact.SizeBytes)
}

func vendorWorkDocumentAvailable(view thirdparty.VendorWorkReviewView, requestID, artifactID string) bool {
	for _, document := range view.Documents {
		if document.RequestID == requestID && document.ArtifactID == artifactID && document.ArtifactStatus == evidence.ArtifactAvailable {
			return true
		}
	}
	return false
}
