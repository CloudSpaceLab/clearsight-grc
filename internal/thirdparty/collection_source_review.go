package thirdparty

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

// CollectionSourceReviewReader checks only the exact review already named by a
// collection receipt. It must not call GetReview: capture holds the request lock.
type CollectionSourceReviewReader struct {
	Documents AssessmentReviewDocumentReader
}

func (r CollectionSourceReviewReader) ReadCollectionSourceReview(ctx context.Context, tenant, entity, assessmentID, requestID, artifactID string) (evidence.DocumentReview, string, error) {
	if r.Documents == nil {
		return evidence.DocumentReview{}, "", ErrAssessmentReadinessUnavailable
	}
	documents, err := r.Documents.ListAssessmentDocuments(ctx, Scope{TenantID: tenant, LegalEntityID: entity}, assessmentID, assessmentReviewMaxArtifacts+1)
	if err != nil {
		return evidence.DocumentReview{}, "", err
	}
	if len(documents) > assessmentReviewMaxArtifacts {
		return evidence.DocumentReview{}, "", ErrInvalid
	}
	for _, document := range documents {
		if document.RequestID == requestID && document.ArtifactID == artifactID {
			at := document.ValidatedAt
			return evidence.DocumentReview{ID: document.ID, Status: string(document.Status), ReviewedBy: document.ValidatedByPrincipalID, ReviewedAt: &at, Source: "VENDOR_ASSESSMENT"}, assessmentDocumentDateString(document.ExpiresOn), nil
		}
	}
	return evidence.DocumentReview{}, "", nil
}
