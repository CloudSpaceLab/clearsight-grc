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
	receipt := assessmentCollectionDocument(view, requestID, artifactID)
	if receipt != nil {
		service, ok := a.formDistributionService(w)
		if !ok {
			return
		}
		source := receipt.Source
		page, readErr := service.ListDocuments(r.Context(), evidence.DocumentQuery{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, RelationshipID: view.Assessment.RelationshipID, SubmissionID: source.SubmissionID, FieldID: source.FieldID, ArtifactID: artifactID, ResponseRevisionID: source.ResponseRevisionID, Limit: 1})
		if readErr != nil || len(page.Items) != 1 || page.Items[0].ArtifactRequestID != requestID || !evidence.ArtifactUseAllowed(page.Items[0].ArtifactStatus, page.Items[0].DemoUnscannedAllowed) || page.Items[0].SHA256 != source.SHA256 || page.Items[0].SizeBytes != source.SizeBytes {
			httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
			return
		}
	}
	if receipt == nil && !assessmentDocumentAvailable(view, requestID, artifactID) {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	artifact, reader, err := a.deps.Evidence.OpenArtifact(r.Context(), actor.TenantID, requestID, artifactID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	defer reader.Close()
	if receipt != nil && (artifact.SHA256 != receipt.Source.SHA256 || artifact.SizeBytes != receipt.Source.SizeBytes) {
		httpx.WriteError(w, http.StatusNotFound, "vendor_document_not_found", "This document is not available for review.")
		return
	}
	documentProtection(w)
	w.Header().Set("Content-Type", artifact.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": artifact.FileName}))
	w.Header().Set("Content-Length", strconv.FormatInt(artifact.SizeBytes, 10))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.CopyN(w, reader, artifact.SizeBytes)
}

func assessmentCollectionDocument(view thirdparty.AssessmentReviewView, requestID, artifactID string) *evidence.CollectionResolution {
	if requestID == "" || artifactID == "" {
		return nil
	}
	for _, answer := range view.Answers {
		r := answer.CollectionResolution
		if r != nil && r.SourceArtifactRequestID == requestID && r.Source.ArtifactID == artifactID && evidence.ArtifactUseAllowed(r.Source.ArtifactStatus, r.Source.DemoUnscannedAllowed) {
			return r
		}
	}
	return nil
}

func assessmentDocumentAvailable(view thirdparty.AssessmentReviewView, requestID, artifactID string) bool {
	requestID, artifactID = strings.TrimSpace(requestID), strings.TrimSpace(artifactID)
	if requestID == "" || artifactID == "" || view.Response == nil || view.Response.RequestID != requestID || view.Assessment.CurrentRequestID != requestID {
		return false
	}
	for _, request := range view.Requests {
		if request.RequestID == requestID {
			for _, document := range view.Documents {
				if document.ArtifactID == artifactID && evidence.ArtifactUseAllowed(document.ArtifactStatus, document.DemoUnscannedAllowed) {
					return true
				}
			}
		}
	}
	return false
}
