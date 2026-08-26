package httpapi

import (
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func (a *API) openVendorAssessmentDocument(w http.ResponseWriter, r *http.Request) {
	if a.deps.ThirdPartyAssessmentReviews == nil || a.deps.Evidence == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_document_unavailable", "This document is temporarily unavailable.")
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to open this document.")
		return
	}
	assessmentID, requestID, artifactID := r.PathValue("id"), r.PathValue("request_id"), r.PathValue("artifact_id")
	view, err := a.deps.ThirdPartyAssessmentReviews.GetReview(r.Context(), actor, assessmentID)
	if err != nil {
		writeThirdPartyAssessmentError(w, err)
		return
	}
	if !assessmentDocumentAvailable(view, requestID, artifactID) {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	artifact, reader, err := a.deps.Evidence.OpenArtifact(r.Context(), actor.TenantID, requestID, artifactID)
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

func assessmentDocumentAvailable(view thirdparty.AssessmentReviewView, requestID, artifactID string) bool {
	requestID, artifactID = strings.TrimSpace(requestID), strings.TrimSpace(artifactID)
	if requestID == "" || artifactID == "" || view.Response == nil || view.Response.RequestID != requestID || view.Assessment.CurrentRequestID != requestID {
		return false
	}
	for _, request := range view.Requests {
		if request.RequestID == requestID {
			for _, document := range view.Documents {
				if document.ArtifactID == artifactID && document.ArtifactStatus == evidence.ArtifactAvailable {
					return true
				}
			}
		}
	}
	return false
}
