package thirdparty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var (
	ErrAssessmentCompletionBlocked    = errors.New("vendor assessment review conditions are unresolved")
	ErrAssessmentReadinessUnavailable = errors.New("vendor assessment completion readiness is unavailable")
)

type AssessmentDocumentDecision string

const (
	AssessmentDocumentValidate AssessmentDocumentDecision = "VALIDATE"
	AssessmentDocumentReject   AssessmentDocumentDecision = "REJECT"
)

type AssessmentDocumentStatus string

const (
	AssessmentDocumentSubmitted  AssessmentDocumentStatus = "SUBMITTED"
	AssessmentDocumentValidated  AssessmentDocumentStatus = "VALIDATED"
	AssessmentDocumentRejected   AssessmentDocumentStatus = "REJECTED"
	AssessmentDocumentExpired    AssessmentDocumentStatus = "EXPIRED"
	AssessmentDocumentSuperseded AssessmentDocumentStatus = "SUPERSEDED"
)

type AssessmentDocumentEvidenceClass string

const (
	AssessmentDocumentVendorSupplied AssessmentDocumentEvidenceClass = "VENDOR_SUPPLIED"
	AssessmentDocumentBankValidated  AssessmentDocumentEvidenceClass = "BANK_VALIDATED"
	AssessmentDocumentOfficialSource AssessmentDocumentEvidenceClass = "OFFICIAL_SOURCE"
)

type AssessmentDocument struct {
	ID string `json:"id"`
	Scope
	RelationshipID         string                          `json:"relationship_id"`
	AssessmentID           string                          `json:"assessment_id"`
	RequestID              string                          `json:"request_id"`
	ArtifactID             string                          `json:"artifact_id"`
	SupersedesDocumentID   string                          `json:"supersedes_document_id,omitempty"`
	DocumentType           string                          `json:"document_type"`
	Reference              string                          `json:"reference,omitempty"`
	IssuedBy               string                          `json:"issued_by,omitempty"`
	IssuedOn               *time.Time                      `json:"issued_on,omitempty"`
	ExpiresOn              *time.Time                      `json:"expires_on,omitempty"`
	EvidenceClass          AssessmentDocumentEvidenceClass `json:"evidence_class"`
	Status                 AssessmentDocumentStatus        `json:"status"`
	ValidatedByPrincipalID string                          `json:"validated_by_principal_id"`
	ValidatedAt            time.Time                       `json:"validated_at"`
	Version                int64                           `json:"version"`
	CreatedAt              time.Time                       `json:"created_at"`
	UpdatedAt              time.Time                       `json:"updated_at"`
}

type ReviewAssessmentDocumentInput struct {
	ExpectedVersion int64                           `json:"expected_version"`
	Decision        AssessmentDocumentDecision      `json:"decision"`
	DocumentType    string                          `json:"document_type"`
	EvidenceClass   AssessmentDocumentEvidenceClass `json:"evidence_class"`
	ValidUntil      string                          `json:"valid_until,omitempty"`
}

type AssessmentDocumentReviewRecord struct {
	Scope
	AssessmentID     string
	ExpectedVersion  int64
	ActorPrincipalID string
	Artifact         evidence.Artifact
	Document         formcontract.DocumentAnswer
	Decision         AssessmentDocumentDecision
	DocumentType     string
	EvidenceClass    AssessmentDocumentEvidenceClass
	ExpiresOn        *time.Time
	At               time.Time
}

type AssessmentCompletionReadiness interface {
	CheckAssessmentCompletion(context.Context, Actor, string) error
}

func (s *AssessmentReviewService) ReviewDocument(ctx context.Context, _ Actor, assessmentID, artifactID string, input ReviewAssessmentDocumentInput) (AssessmentReviewView, error) {
	assessmentID, artifactID = strings.TrimSpace(assessmentID), strings.TrimSpace(artifactID)
	input.DocumentType = strings.TrimSpace(input.DocumentType)
	if s == nil || s.assessments == nil || !validAssessmentIdentifiers(assessmentID, artifactID) || input.ExpectedVersion < 1 || !validAssessmentDocumentDecision(input.Decision) || !validAssessmentDocumentEvidenceClass(input.EvidenceClass) || !validAssessmentReviewText(input.DocumentType, 128) {
		return AssessmentReviewView{}, ErrInvalid
	}
	actor, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentDocumentReviewCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return AssessmentReviewView{}, err
	}
	view, err := s.GetReview(ctx, actor, assessmentID)
	if err != nil {
		return AssessmentReviewView{}, err
	}
	if view.Assessment.Version != input.ExpectedVersion {
		return AssessmentReviewView{}, ErrVersionConflict
	}
	if view.Assessment.Status != AssessmentUnderReview || view.Response == nil || view.Response.RequestID != view.Assessment.CurrentRequestID {
		return AssessmentReviewView{}, ErrInvalidAssessmentTransition
	}
	var submitted AssessmentReviewDocument
	found := false
	for _, document := range view.Documents {
		if document.ArtifactID == artifactID {
			submitted, found = document, true
			break
		}
	}
	if !found {
		return AssessmentReviewView{}, ErrNotFound
	}
	var submittedMetadata *formcontract.DocumentAnswer
	for _, answer := range view.Answers {
		if answer.FieldID == submitted.FieldID && answer.Value != nil && answer.Value.Document != nil && strings.TrimSpace(answer.Value.Document.ArtifactID) == artifactID {
			copy := *answer.Value.Document
			submittedMetadata = &copy
			break
		}
	}
	if submittedMetadata == nil {
		return AssessmentReviewView{}, ErrNotFound
	}
	if input.Decision == AssessmentDocumentValidate && submitted.ArtifactStatus != evidence.ArtifactAvailable {
		return AssessmentReviewView{}, ErrAssessmentCompletionBlocked
	}
	expiresOn, err := assessmentDocumentDate(input.ValidUntil)
	if err != nil {
		return AssessmentReviewView{}, ErrInvalid
	}
	if expiresOn == nil {
		expiresOn, err = assessmentDocumentDate(submittedMetadata.ExpiresOn)
		if err != nil {
			return AssessmentReviewView{}, ErrInvalid
		}
	}
	issuedOn, err := assessmentDocumentDate(submittedMetadata.IssuedOn)
	if err != nil || (issuedOn != nil && expiresOn != nil && expiresOn.Before(*issuedOn)) {
		return AssessmentReviewView{}, ErrInvalid
	}
	repository, ok := s.links.(interface {
		ReviewAssessmentDocument(context.Context, AssessmentDocumentReviewRecord) (AssessmentDocument, Assessment, error)
	})
	if !ok {
		return AssessmentReviewView{}, ErrAssessmentReadinessUnavailable
	}
	document, assessment, err := repository.ReviewAssessmentDocument(ctx, AssessmentDocumentReviewRecord{
		Scope: scopeFrom(actor), AssessmentID: assessmentID, ExpectedVersion: input.ExpectedVersion, ActorPrincipalID: actor.PrincipalID,
		Artifact: evidence.Artifact{ID: submitted.ArtifactID, TenantID: actor.TenantID, RequestID: view.Assessment.CurrentRequestID, SubmissionID: view.Assessment.SubmissionID, Status: submitted.ArtifactStatus},
		Document: *submittedMetadata,
		Decision: input.Decision, DocumentType: input.DocumentType, EvidenceClass: input.EvidenceClass, ExpiresOn: expiresOn, At: s.assessments.now().UTC(),
	})
	if err != nil {
		return AssessmentReviewView{}, err
	}
	refreshed, err := s.GetReview(ctx, actor, assessmentID)
	if err != nil {
		view.Assessment = assessment
		for index := range view.Documents {
			if view.Documents[index].ArtifactID != document.ArtifactID {
				continue
			}
			view.Documents[index].Status = string(document.Status)
			view.Documents[index].EvidenceClass = AssessmentEvidenceClass(document.EvidenceClass)
			view.Documents[index].DocumentType = document.DocumentType
			view.Documents[index].Reference = document.Reference
			view.Documents[index].IssuedBy = document.IssuedBy
			view.Documents[index].IssuedOn = assessmentDocumentDateString(document.IssuedOn)
			view.Documents[index].ExpiresOn = assessmentDocumentDateString(document.ExpiresOn)
		}
		return view, nil
	}
	if refreshed.Assessment.ID != assessment.ID || refreshed.Assessment.Version != assessment.Version || refreshed.Assessment.Status != assessment.Status {
		return AssessmentReviewView{}, ErrNotFound
	}
	return refreshed, nil
}

func (s *AssessmentReviewService) CheckAssessmentCompletion(ctx context.Context, actor Actor, assessmentID string) error {
	view, err := s.GetReview(ctx, actor, assessmentID)
	if err != nil {
		return err
	}
	if view.Assessment.Status != AssessmentUnderReview || view.Response == nil || view.Response.RequestID != view.Assessment.CurrentRequestID || view.Coverage.AnsweredRequired != view.Coverage.RequiredFields {
		return ErrAssessmentCompletionBlocked
	}
	for _, status := range view.artifactStatuses {
		if status != evidence.ArtifactAvailable {
			return ErrAssessmentCompletionBlocked
		}
	}
	documents := make(map[string]AssessmentReviewDocument, len(view.Documents))
	for _, document := range view.Documents {
		documents[document.ArtifactID] = document
	}
	for _, answer := range view.Answers {
		if answer.Visibility != AssessmentAnswerVisible || !answer.Required || answer.Type != formcontract.TypeVendorDocument || answer.Value == nil || answer.Value.Document == nil {
			continue
		}
		document, ok := documents[strings.TrimSpace(answer.Value.Document.ArtifactID)]
		if !ok || (document.Status != string(AssessmentDocumentValidated) && document.Status != string(AssessmentDocumentRejected)) {
			return ErrAssessmentCompletionBlocked
		}
	}
	if len(view.Requests) == 0 || view.Requests[len(view.Requests)-1].RequestID != view.Assessment.CurrentRequestID || view.Requests[len(view.Requests)-1].Status != evidence.RequestSubmitted {
		return ErrAssessmentCompletionBlocked
	}
	return nil
}

func validAssessmentDocumentDecision(value AssessmentDocumentDecision) bool {
	return value == AssessmentDocumentValidate || value == AssessmentDocumentReject
}

func validAssessmentDocumentEvidenceClass(value AssessmentDocumentEvidenceClass) bool {
	return value == AssessmentDocumentVendorSupplied || value == AssessmentDocumentBankValidated || value == AssessmentDocumentOfficialSource
}

func assessmentDocumentDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func assessmentDocumentDateString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}
