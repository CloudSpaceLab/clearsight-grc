package thirdparty

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

// DocumentContextReader is the memory-store adapter. Production performs the
// same stored-link and authority checks in the document inventory SQL before
// pagination. No filename, recipient address or artifact first-use inference.
type DocumentContextReader struct {
	Assessments *AssessmentReviewService
	Work        *VendorWorkService
}

func (reader DocumentContextReader) ResolveDocumentContext(ctx context.Context, q evidence.DocumentQuery, request evidence.Request, submission evidence.Submission) (evidence.DocumentContext, error) {
	if q.TenantID != request.TenantID || q.LegalEntityID != request.LegalEntityID || q.PrincipalID == "" || submission.TenantID != q.TenantID || submission.RequestID != request.ID {
		return evidence.DocumentContext{}, evidence.ErrNotFound
	}
	actor := Actor{TenantID: q.TenantID, LegalEntityID: q.LegalEntityID, PrincipalID: q.PrincipalID}
	scope := Scope{TenantID: q.TenantID, LegalEntityID: q.LegalEntityID}
	switch request.Origin.Type {
	case AssessmentRequestOrigin:
		service := reader.Assessments
		if service == nil || service.assessments == nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		assessment, err := service.assessments.GetAssessment(ctx, actor, request.Origin.ID)
		if err != nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		if err := service.authorizeRead(ctx, actor, scope, assessment); err != nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		if request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != assessment.RelationshipID || request.FormTemplateID != assessment.FormTemplateID || request.FormTemplateVersion != assessment.FormTemplateVersion {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		links, err := service.links.ListAssessmentRequestLinks(ctx, scope, assessment.ID)
		if err != nil {
			return evidence.DocumentContext{}, err
		}
		found := false
		for _, link := range links {
			if link.RequestID == request.ID && link.TenantID == q.TenantID && link.LegalEntityID == q.LegalEntityID && link.OriginID == assessment.ID && link.OriginType == request.Origin.Type && int64(link.OriginSequence) == request.Origin.Version {
				found = true
			}
		}
		if !found {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		result := evidence.DocumentContext{RelationshipID: assessment.RelationshipID, AssessmentID: assessment.ID, Current: assessment.CurrentRequestID == request.ID, Reviews: map[string]evidence.DocumentReview{}, Expiries: map[string]string{}}
		if documents, ok := service.links.(AssessmentReviewDocumentReader); ok {
			values, err := documents.ListAssessmentDocuments(ctx, scope, assessment.ID, assessmentReviewMaxArtifacts+1)
			if err != nil {
				return evidence.DocumentContext{}, err
			}
			if len(values) > assessmentReviewMaxArtifacts {
				return evidence.DocumentContext{}, evidence.ErrNotFound
			}
			for _, doc := range values {
				if doc.RequestID == request.ID && doc.RelationshipID == assessment.RelationshipID {
					result.Reviews[doc.ArtifactID] = evidence.DocumentReview{ID: doc.ID, Status: string(doc.Status), ReviewedBy: doc.ValidatedByPrincipalID, ReviewedAt: doc.ValidatedAt, Source: "VENDOR_ASSESSMENT"}
					result.Expiries[doc.ArtifactID] = assessmentDocumentDateString(doc.ExpiresOn)
				}
			}
		}
		return result, nil
	case VendorWorkOrigin:
		service := reader.Work
		if service == nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		work, err := service.repo.GetVendorWork(ctx, scope, request.Origin.ID)
		if err != nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		if err := service.authorizeRead(ctx, actor, work); err != nil {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		if request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != work.RelationshipID {
			return evidence.DocumentContext{}, evidence.ErrNotFound
		}
		links, err := service.repo.ListVendorWorkCaptures(ctx, scope, work.ID)
		if err != nil {
			return evidence.DocumentContext{}, err
		}
		for _, link := range links {
			if link.TenantID == q.TenantID && link.LegalEntityID == q.LegalEntityID && link.RequestID == request.ID && link.OriginVersion == request.Origin.Version {
				return evidence.DocumentContext{RelationshipID: work.RelationshipID, WorkRequestID: work.ID, Current: work.CurrentRequestID == request.ID}, nil
			}
		}
	}
	return evidence.DocumentContext{}, evidence.ErrNotFound
}
