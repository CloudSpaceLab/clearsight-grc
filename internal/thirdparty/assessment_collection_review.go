package thirdparty

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (s *AssessmentReviewService) addCollectionEvidence(view *AssessmentReviewView, request evidence.Request) {
	for _, field := range request.Fields {
		receipt := field.CollectionResolution
		if receipt == nil {
			continue
		}
		source := receipt.Source
		status := receipt.BankReviewState
		if !evidence.CollectionFieldFulfilled(field, s.assessments.now()) {
			switch {
			case !source.Current:
				status = "SUPERSEDED"
			case source.Review != nil && source.Review.Status == "REJECTED":
				status = "REJECTED"
			case !s.artifactUseAllowed(source.ArtifactStatus):
				status = "SUBMITTED"
			case receipt.BankReviewState != "REJECTED":
				status = "EXPIRED"
			}
		}
		if status == "PENDING" {
			status = "SUBMITTED"
		}
		class := AssessmentEvidenceVendorSupplied
		if status == "VALIDATED" {
			class = AssessmentEvidenceBankValidated
		}
		view.Documents = append(view.Documents, AssessmentReviewDocument{FieldID: field.ID, RequestID: receipt.SourceArtifactRequestID, ArtifactID: source.ArtifactID, FileName: source.FileName, MediaType: source.MediaType, SizeBytes: source.SizeBytes, ArtifactStatus: source.ArtifactStatus, DemoUnscannedAllowed: s.demoUnscannedAllowed(source.ArtifactStatus), Status: status, EvidenceClass: class, DocumentType: receipt.Document.DocumentType, Reference: receipt.Document.Reference, IssuedBy: receipt.Document.IssuedBy, IssuedOn: receipt.Document.IssuedOn, ExpiresOn: receipt.Document.ExpiresOn})
		found := false
		for i := range view.Answers {
			if view.Answers[i].FieldID == field.ID {
				view.Answers[i].CollectionResolution = receipt
				found = true
			}
		}
		if !found {
			view.Answers = append(view.Answers, AssessmentReviewAnswer{FieldID: field.ID, Label: field.Label, Type: formcontract.Type(field.Type), Required: field.Required, Visibility: AssessmentAnswerVisible, CollectionResolution: receipt})
		}
	}
}

func (s *AssessmentReviewService) reviewCollectionDocument(ctx context.Context, actor Actor, view AssessmentReviewView, request evidence.Request, field evidence.Field, input ReviewAssessmentDocumentInput) (AssessmentReviewView, error) {
	prior := field.CollectionResolution
	if prior == nil {
		return AssessmentReviewView{}, ErrNotFound
	}
	if input.Decision == AssessmentDocumentValidate && !evidence.CollectionFieldFulfilled(field, s.assessments.now()) {
		return AssessmentReviewView{}, ErrAssessmentCompletionBlocked
	}
	expiry, err := assessmentDocumentDate(input.ValidUntil)
	if err != nil {
		return AssessmentReviewView{}, ErrInvalid
	}
	receipt := *prior
	receipt.ID, err = id.NewUUIDv7()
	if err != nil {
		return AssessmentReviewView{}, err
	}
	receipt.Document.DocumentType = input.DocumentType
	if expiry != nil {
		receipt.Document.ExpiresOn = assessmentDocumentDateString(expiry)
	}
	receipt.BankReviewState = "VALIDATED"
	if input.Decision == AssessmentDocumentReject {
		receipt.BankReviewState = "REJECTED"
	}
	now := s.assessments.now().UTC()
	receipt.ReviewedAt = &now
	receipt.ReviewedBy = actor.PrincipalID
	store, ok := s.links.(assessmentCollectionStore)
	if !ok {
		return AssessmentReviewView{}, ErrAssessmentReadinessUnavailable
	}
	writer, _ := s.evidence.(collectionRequestWriter)
	saved, updated, _, err := store.WriteAssessmentCollection(ctx, assessmentCollectionRecord{Scope: scopeFrom(actor), AssessmentID: view.Assessment.ID, RequestID: request.ID, FieldID: field.ID, ActorPrincipalID: actor.PrincipalID, ExpectedVersion: input.ExpectedVersion, ExpectedRequestVersion: request.Version, Resolution: receipt, At: now, Review: true, Authorize: func(txctx context.Context) error {
		_, err := s.assessments.authorize(txctx, view.Assessment.ID, assessmentObjectType, AssessmentDocumentReviewCommand, authority.ResponsibilityReviewer)
		return err
	}}, writer)
	if err != nil {
		return AssessmentReviewView{}, err
	}
	if refreshed, readErr := s.GetReview(ctx, actor, saved.ID); readErr == nil {
		return refreshed, nil
	}
	view.Assessment = saved
	heldFields := map[string]bool{}
	for _, field := range updated.Fields {
		if field.CollectionResolution != nil {
			heldFields[field.ID] = true
		}
	}
	kept := view.Documents[:0]
	for _, document := range view.Documents {
		if !heldFields[document.FieldID] {
			kept = append(kept, document)
		}
	}
	view.Documents = kept
	s.addCollectionEvidence(&view, updated)
	return view, nil
}
