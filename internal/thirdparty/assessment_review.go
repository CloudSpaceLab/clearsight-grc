package thirdparty

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const (
	assessmentReviewMaxRequests  = 20
	assessmentReviewMaxArtifacts = 200
	assessmentReviewMaxMatters   = 50
)

type AssessmentAnswerVisibility string

const (
	AssessmentAnswerVisible              AssessmentAnswerVisibility = "VISIBLE"
	AssessmentAnswerConditionallyOmitted AssessmentAnswerVisibility = "CONDITIONALLY_OMITTED"
)

type AssessmentEvidenceClass string

const (
	AssessmentEvidenceVendorSupplied AssessmentEvidenceClass = "VENDOR_SUPPLIED"
	AssessmentEvidenceBankValidated  AssessmentEvidenceClass = "BANK_VALIDATED"
	AssessmentEvidenceOfficialSource AssessmentEvidenceClass = "OFFICIAL_SOURCE"
)

type AssessmentReviewRequest struct {
	RequestID           string                   `json:"request_id"`
	Purpose             AssessmentRequestPurpose `json:"purpose"`
	Sequence            int                      `json:"sequence"`
	OriginSequence      int                      `json:"origin_sequence"`
	Status              evidence.RequestStatus   `json:"status"`
	Deadline            time.Time                `json:"deadline"`
	FormTemplateID      string                   `json:"form_template_id"`
	FormTemplateVersion int64                    `json:"form_template_version"`
}

type AssessmentReviewResponse struct {
	SubmissionID     string    `json:"submission_id"`
	RequestID        string    `json:"request_id"`
	SubmittedAt      time.Time `json:"submitted_at"`
	AnswerCount      int       `json:"answer_count"`
	ArtifactCount    int       `json:"artifact_count"`
	ProvisionalScore *float64  `json:"provisional_score,omitempty"`
}

type AssessmentReviewCoverage struct {
	VisibleFields    int     `json:"visible_fields"`
	AnsweredFields   int     `json:"answered_fields"`
	RequiredFields   int     `json:"required_fields"`
	AnsweredRequired int     `json:"answered_required"`
	Ratio            float64 `json:"ratio"`
}

type AssessmentReviewAnswer struct {
	FieldID    string                     `json:"field_id"`
	Label      string                     `json:"label"`
	Type       formcontract.Type          `json:"type"`
	Required   bool                       `json:"required"`
	Visibility AssessmentAnswerVisibility `json:"visibility"`
	Value      *formcontract.AnswerValue  `json:"value,omitempty"`
	Provenance *evidence.AnswerProvenance `json:"provenance,omitempty"`
	Baseline   *evidence.RecordBaseline   `json:"baseline,omitempty"`
}

type AssessmentReviewDocument struct {
	FieldID        string                  `json:"field_id"`
	ArtifactID     string                  `json:"artifact_id"`
	FileName       string                  `json:"file_name"`
	MediaType      string                  `json:"media_type"`
	SizeBytes      int64                   `json:"size_bytes"`
	ArtifactStatus evidence.ArtifactStatus `json:"artifact_status"`
	Status         string                  `json:"status"`
	EvidenceClass  AssessmentEvidenceClass `json:"evidence_class"`
	DocumentType   string                  `json:"document_type"`
	Reference      string                  `json:"reference,omitempty"`
	IssuedBy       string                  `json:"issued_by,omitempty"`
	IssuedOn       string                  `json:"issued_on,omitempty"`
	ExpiresOn      string                  `json:"expires_on,omitempty"`
}

type AssessmentReviewMatter struct {
	TenantID      string `json:"-"`
	LegalEntityID string `json:"-"`
	AssessmentID  string `json:"-"`
	MatterID      string `json:"matter_id"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Title         string `json:"title"`
}

type AssessmentReviewView struct {
	Assessment         Assessment                  `json:"assessment"`
	Requests           []AssessmentReviewRequest   `json:"requests"`
	Response           *AssessmentReviewResponse   `json:"response,omitempty"`
	Answers            []AssessmentReviewAnswer    `json:"answers"`
	Coverage           AssessmentReviewCoverage    `json:"coverage"`
	Documents          []AssessmentReviewDocument  `json:"documents"`
	ProvisionalScore   *formcontract.ScoreResult   `json:"provisional_score,omitempty"`
	Matters            []AssessmentReviewMatter    `json:"matters"`
	ApplicationReceipt *ResponseApplicationReceipt `json:"application_receipt,omitempty"`
	artifactStatuses   []evidence.ArtifactStatus
}

type AssessmentReviewLinkReader interface {
	ListAssessmentRequestLinks(context.Context, Scope, string) ([]AssessmentRequestLink, error)
}

type AssessmentReviewEvidenceReader interface {
	GetRequest(context.Context, string, string) (evidence.Request, error)
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
	GetArtifact(context.Context, string, string, string) (evidence.Artifact, error)
}

type AssessmentReviewMatterReader interface {
	ListAssessmentReviewMatters(context.Context, Actor, Scope, string, int) ([]AssessmentReviewMatter, error)
}

type AssessmentReviewDocumentReader interface {
	ListAssessmentDocuments(context.Context, Scope, string, int) ([]AssessmentDocument, error)
}

type AssessmentReviewApplicationReader interface {
	GetResponseApplicationReceipt(context.Context, Scope, string, string) (ResponseApplicationReceipt, error)
}

type AssessmentReviewAuthority interface {
	Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error)
}

type AssessmentReviewRelationshipReader interface {
	GetRelationship(context.Context, Scope, string) (Aggregate, error)
}

type AssessmentReviewService struct {
	assessments *AssessmentService
	links       AssessmentReviewLinkReader
	evidence    AssessmentReviewEvidenceReader
	matters     AssessmentReviewMatterReader
	authority   AssessmentReviewAuthority
}

func NewAssessmentReviewService(assessments *AssessmentService, links AssessmentReviewLinkReader, evidenceReader AssessmentReviewEvidenceReader, matters AssessmentReviewMatterReader) *AssessmentReviewService {
	return &AssessmentReviewService{assessments: assessments, links: links, evidence: evidenceReader, matters: matters}
}

func (s *AssessmentReviewService) ConfigureAuthority(resolver AssessmentReviewAuthority) {
	if s != nil {
		s.authority = resolver
	}
}

func (s *AssessmentReviewService) GetReview(ctx context.Context, actor Actor, assessmentID string) (AssessmentReviewView, error) {
	if s == nil || s.assessments == nil || s.links == nil || s.evidence == nil {
		return AssessmentReviewView{}, ErrInvalid
	}
	assessment, err := s.assessments.GetAssessment(ctx, actor, assessmentID)
	if err != nil {
		return AssessmentReviewView{}, err
	}
	scope := Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID}
	if err := s.authorizeRead(ctx, actor, scope, assessment); err != nil {
		return AssessmentReviewView{}, err
	}
	links, err := s.links.ListAssessmentRequestLinks(ctx, scope, assessment.ID)
	if err != nil {
		return AssessmentReviewView{}, err
	}
	if len(links) > assessmentReviewMaxRequests {
		return AssessmentReviewView{}, ErrInvalid
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Sequence < links[j].Sequence })
	view := AssessmentReviewView{Assessment: assessment, Requests: make([]AssessmentReviewRequest, 0, len(links)), Answers: []AssessmentReviewAnswer{}, Documents: []AssessmentReviewDocument{}, Matters: []AssessmentReviewMatter{}}
	requests := make(map[string]evidence.Request, len(links))
	lastSequence := 0
	for _, link := range links {
		if link.TenantID != scope.TenantID || link.LegalEntityID != scope.LegalEntityID || link.AssessmentID != assessment.ID ||
			link.OriginType != AssessmentRequestOrigin || link.OriginID != assessment.ID || link.Sequence < 1 || link.Sequence <= lastSequence || link.OriginSequence != link.Sequence {
			return AssessmentReviewView{}, ErrNotFound
		}
		request, readErr := s.evidence.GetRequest(ctx, scope.TenantID, link.RequestID)
		if readErr != nil {
			return AssessmentReviewView{}, readErr
		}
		if request.ID != link.RequestID || request.TenantID != scope.TenantID || request.SubjectType != "VENDOR_RELATIONSHIP" || request.SubjectID != assessment.RelationshipID ||
			request.Origin.Type != AssessmentRequestOrigin || request.Origin.ID != assessment.ID || request.Origin.Version != int64(link.OriginSequence) ||
			request.FormTemplateID != assessment.FormTemplateID || request.FormTemplateVersion != assessment.FormTemplateVersion {
			return AssessmentReviewView{}, ErrNotFound
		}
		if _, duplicate := requests[request.ID]; duplicate {
			return AssessmentReviewView{}, ErrNotFound
		}
		requests[request.ID] = request
		view.Requests = append(view.Requests, AssessmentReviewRequest{RequestID: request.ID, Purpose: link.Purpose, Sequence: link.Sequence, OriginSequence: link.OriginSequence, Status: request.Status, Deadline: request.Deadline, FormTemplateID: request.FormTemplateID, FormTemplateVersion: request.FormTemplateVersion})
		lastSequence = link.Sequence
	}
	if assessment.CurrentRequestID == "" && assessment.SubmissionID != "" {
		return AssessmentReviewView{}, ErrNotFound
	}
	if (assessment.Status == AssessmentSubmitted || assessment.Status == AssessmentUnderReview || assessment.Status == AssessmentCompleted) && (assessment.CurrentRequestID == "" || assessment.SubmissionID == "") {
		return AssessmentReviewView{}, ErrNotFound
	}
	if (assessment.Status == AssessmentSetupPending || assessment.Status == AssessmentReadyToSend || assessment.Status == AssessmentCollecting) && assessment.SubmissionID != "" {
		return AssessmentReviewView{}, ErrNotFound
	}
	currentRequest, hasCurrent := requests[assessment.CurrentRequestID]
	if assessment.CurrentRequestID != "" && !hasCurrent {
		return AssessmentReviewView{}, ErrNotFound
	}
	if assessment.SubmissionID != "" {
		submission, readErr := s.evidence.GetSubmission(ctx, scope.TenantID, assessment.SubmissionID)
		if readErr != nil {
			return AssessmentReviewView{}, readErr
		}
		if submission.ID != assessment.SubmissionID || submission.TenantID != scope.TenantID || submission.RequestID != currentRequest.ID {
			return AssessmentReviewView{}, ErrNotFound
		}
		if err = s.addSubmission(ctx, &view, currentRequest, submission); err != nil {
			return AssessmentReviewView{}, err
		}
	}
	if documents, ok := s.links.(AssessmentReviewDocumentReader); ok {
		values, readErr := documents.ListAssessmentDocuments(ctx, scope, assessment.ID, assessmentReviewMaxArtifacts+1)
		if readErr != nil {
			return AssessmentReviewView{}, readErr
		}
		if len(values) > assessmentReviewMaxArtifacts {
			return AssessmentReviewView{}, ErrInvalid
		}
		byArtifact := make(map[string]AssessmentDocument, len(values))
		for _, document := range values {
			if document.TenantID != scope.TenantID || document.LegalEntityID != scope.LegalEntityID || document.AssessmentID != assessment.ID || document.RelationshipID != assessment.RelationshipID || document.RequestID != assessment.CurrentRequestID || !validAssessmentIdentifier(document.ArtifactID) || !validAssessmentDocumentEvidenceClass(document.EvidenceClass) || (document.Status != AssessmentDocumentValidated && document.Status != AssessmentDocumentRejected) {
				return AssessmentReviewView{}, ErrNotFound
			}
			if _, duplicate := byArtifact[document.ArtifactID]; duplicate {
				return AssessmentReviewView{}, ErrNotFound
			}
			byArtifact[document.ArtifactID] = document
		}
		for index := range view.Documents {
			stored, exists := byArtifact[view.Documents[index].ArtifactID]
			if !exists {
				continue
			}
			view.Documents[index].Status = string(stored.Status)
			view.Documents[index].EvidenceClass = AssessmentEvidenceClass(stored.EvidenceClass)
			view.Documents[index].DocumentType = stored.DocumentType
			view.Documents[index].Reference = stored.Reference
			view.Documents[index].IssuedBy = stored.IssuedBy
			view.Documents[index].IssuedOn = assessmentDocumentDateString(stored.IssuedOn)
			view.Documents[index].ExpiresOn = assessmentDocumentDateString(stored.ExpiresOn)
			delete(byArtifact, stored.ArtifactID)
		}
		if len(byArtifact) != 0 {
			return AssessmentReviewView{}, ErrNotFound
		}
	}
	if s.matters != nil {
		values, readErr := s.matters.ListAssessmentReviewMatters(ctx, actor, scope, assessment.ID, assessmentReviewMaxMatters+1)
		if readErr != nil {
			return AssessmentReviewView{}, readErr
		}
		if len(values) > assessmentReviewMaxMatters {
			return AssessmentReviewView{}, ErrInvalid
		}
		for index := range values {
			matter := &values[index]
			matter.MatterID, matter.Type, matter.Status, matter.Title = strings.TrimSpace(matter.MatterID), strings.TrimSpace(matter.Type), strings.TrimSpace(matter.Status), strings.TrimSpace(matter.Title)
			if matter.TenantID != scope.TenantID || matter.LegalEntityID != scope.LegalEntityID || matter.AssessmentID != assessment.ID {
				return AssessmentReviewView{}, ErrNotFound
			}
			if !validAssessmentIdentifier(matter.MatterID) || !validAssessmentReviewCode(matter.Type, 80) || !validAssessmentReviewCode(matter.Status, 80) || !validAssessmentReviewText(matter.Title, 200) {
				return AssessmentReviewView{}, ErrInvalid
			}
		}
		view.Matters = append(view.Matters, values...)
	}
	if assessment.SubmissionID != "" {
		if applications, ok := s.links.(AssessmentReviewApplicationReader); ok {
			receipt, readErr := applications.GetResponseApplicationReceipt(ctx, scope, assessment.ID, assessment.SubmissionID)
			if readErr == nil {
				view.ApplicationReceipt = &receipt
			} else if !errors.Is(readErr, ErrNotFound) {
				return AssessmentReviewView{}, readErr
			}
		}
	}
	return view, nil
}

func (s *AssessmentReviewService) authorizeRead(ctx context.Context, actor Actor, scope Scope, assessment Assessment) error {
	relationships, ok := s.links.(AssessmentReviewRelationshipReader)
	if !ok {
		return ErrNotFound
	}
	aggregate, err := relationships.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return ErrNotFound
	}
	principalID := strings.TrimSpace(actor.PrincipalID)
	if principalID != "" && (principalID == strings.TrimSpace(assessment.StartedByPrincipalID) || principalID == strings.TrimSpace(aggregate.Relationship.BusinessOwnerPrincipalID)) {
		return nil
	}
	if s.authority == nil {
		return ErrNotFound
	}
	resolution, err := s.authority.Resolve(ctx, authority.ResolveInput{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, ObjectType: assessmentObjectType, ObjectID: assessment.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: AssessmentReviewCommand, Materiality: 3,
	})
	if err != nil || !resolution.AllowsPrincipal(principalID) {
		return ErrNotFound
	}
	return nil
}

func (s *AssessmentReviewService) addSubmission(ctx context.Context, view *AssessmentReviewView, request evidence.Request, submission evidence.Submission) error {
	contract := reviewContract(request)
	normalized, err := formcontract.Normalize(contract)
	if err != nil {
		return err
	}
	visible, err := formcontract.VisibleFields(normalized, submission.Answers)
	if err != nil {
		return err
	}
	visibleByID := make(map[string]formcontract.Field, len(visible))
	requestFieldByID := make(map[string]evidence.Field, len(request.Fields))
	for _, field := range request.Fields {
		requestFieldByID[field.ID] = field
	}
	for _, field := range visible {
		visibleByID[field.ID] = field
	}
	if len(submission.Answers) > formcontract.MaxFields || len(submission.AnswerProvenance) > formcontract.MaxFields {
		return ErrInvalid
	}
	for fieldID := range submission.Answers {
		if _, ok := visibleByID[fieldID]; !ok {
			return ErrInvalid
		}
	}
	for fieldID := range submission.AnswerProvenance {
		if _, ok := submission.Answers[fieldID]; !ok {
			return ErrInvalid
		}
	}
	answered, answeredRequired, required := 0, 0, 0
	artifactIDs := map[string]struct{}{}
	for _, field := range visible {
		value, answered := submission.Answers[field.ID]
		if !answered || !value.Answered() || !reviewArtifactField(field.Type) {
			continue
		}
		for _, artifactID := range reviewArtifactIDs(value) {
			if !validAssessmentIdentifier(artifactID) {
				return ErrInvalid
			}
			artifactIDs[artifactID] = struct{}{}
			if len(artifactIDs) > assessmentReviewMaxArtifacts {
				return ErrInvalid
			}
		}
	}
	orderedArtifactIDs := make([]string, 0, len(artifactIDs))
	for artifactID := range artifactIDs {
		orderedArtifactIDs = append(orderedArtifactIDs, artifactID)
	}
	sort.Strings(orderedArtifactIDs)
	artifacts := make(map[string]evidence.Artifact, len(orderedArtifactIDs))
	for _, artifactID := range orderedArtifactIDs {
		artifact, readErr := s.evidence.GetArtifact(ctx, request.TenantID, request.ID, artifactID)
		if readErr != nil {
			return readErr
		}
		if artifact.ID != artifactID || artifact.TenantID != request.TenantID || artifact.RequestID != request.ID || artifact.SubmissionID != submission.ID {
			return ErrNotFound
		}
		if !validAssessmentReviewText(artifact.FileName, 255) || !validAssessmentReviewText(artifact.MediaType, 200) || artifact.SizeBytes < 1 {
			return ErrInvalid
		}
		artifacts[artifactID] = artifact
		view.artifactStatuses = append(view.artifactStatuses, artifact.Status)
	}
	scoreFields := make([]formcontract.Scoring, 0)
	for _, field := range normalized.Fields {
		visibleField, isVisible := visibleByID[field.ID]
		answer := AssessmentReviewAnswer{FieldID: field.ID, Label: field.Label, Type: field.Type, Required: field.Required, Visibility: AssessmentAnswerConditionallyOmitted}
		if requestField := requestFieldByID[field.ID]; requestField.RecordBaseline != nil {
			baseline := *requestField.RecordBaseline
			baseline.ExpiresAt = cloneAssessmentTime(requestField.RecordBaseline.ExpiresAt)
			answer.Baseline = &baseline
		}
		if isVisible {
			answer.Visibility = AssessmentAnswerVisible
			if field.Required {
				required++
			}
			if value, ok := submission.Answers[field.ID]; ok && value.Answered() {
				copy := value
				answer.Value = &copy
				answered++
				if field.Required {
					answeredRequired++
				}
			}
			if provenance, ok := submission.AnswerProvenance[field.ID]; ok {
				copy := provenance
				answer.Provenance = &copy
			}
			if visibleField.Scoring != nil {
				scoreFields = append(scoreFields, *visibleField.Scoring)
			}
			if answer.Value != nil && reviewArtifactField(field.Type) {
				if field.Type == formcontract.TypeVendorDocument && answer.Value.Document != nil {
					document := answer.Value.Document
					artifact, exists := artifacts[strings.TrimSpace(document.ArtifactID)]
					if !exists {
						return ErrInvalid
					}
					if !validAssessmentReviewText(document.DocumentType, 100) || !validOptionalAssessmentReviewText(document.Reference, 200) || !validOptionalAssessmentReviewText(document.IssuedBy, 200) || len(document.IssuedOn) > 10 || len(document.ExpiresOn) > 10 {
						return ErrInvalid
					}
					view.Documents = append(view.Documents, AssessmentReviewDocument{FieldID: field.ID, ArtifactID: artifact.ID, FileName: artifact.FileName, MediaType: artifact.MediaType, SizeBytes: artifact.SizeBytes, ArtifactStatus: artifact.Status, Status: "SUBMITTED", EvidenceClass: AssessmentEvidenceVendorSupplied, DocumentType: document.DocumentType, Reference: document.Reference, IssuedBy: document.IssuedBy, IssuedOn: document.IssuedOn, ExpiresOn: document.ExpiresOn})
				}
			}
		}
		view.Answers = append(view.Answers, answer)
	}
	ratio := float64(1)
	if required > 0 {
		ratio = float64(answeredRequired) / float64(required)
	}
	view.Coverage = AssessmentReviewCoverage{VisibleFields: len(visible), AnsweredFields: answered, RequiredFields: required, AnsweredRequired: answeredRequired, Ratio: ratio}
	view.Response = &AssessmentReviewResponse{SubmissionID: submission.ID, RequestID: submission.RequestID, SubmittedAt: submission.SubmittedAt, AnswerCount: answered, ArtifactCount: len(artifacts)}
	if len(scoreFields) > 0 {
		score, scoreErr := formcontract.ScoreAnswers(scoreFields, submission.Answers)
		if scoreErr != nil {
			return scoreErr
		}
		view.ProvisionalScore = &score
		view.Response.ProvisionalScore = score.Score
	}
	return nil
}

func reviewArtifactField(fieldType formcontract.Type) bool {
	switch fieldType {
	case formcontract.TypeFile, formcontract.TypePhoto, formcontract.TypeSignature, formcontract.TypeVendorDocument:
		return true
	default:
		return false
	}
}

func reviewArtifactIDs(answer formcontract.AnswerValue) []string {
	values := make([]string, 0, len(answer.ArtifactIDs)+1)
	for _, value := range answer.ArtifactIDs {
		values = append(values, strings.TrimSpace(value))
	}
	if answer.Document != nil {
		values = append(values, strings.TrimSpace(answer.Document.ArtifactID))
	}
	return values
}

func validAssessmentReviewCode(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func validAssessmentReviewText(value string, maximum int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maximum {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validOptionalAssessmentReviewText(value string, maximum int) bool {
	return strings.TrimSpace(value) == "" || validAssessmentReviewText(value, maximum)
}

func reviewContract(request evidence.Request) formcontract.Contract {
	fields := make([]formcontract.Field, len(request.Fields))
	for i, field := range request.Fields {
		var scoring *formcontract.Scoring
		if field.Scoring != nil {
			copy := *field.Scoring
			copy.AnswerScores = make(map[string]int, len(field.Scoring.AnswerScores))
			for key, value := range field.Scoring.AnswerScores {
				copy.AnswerScores[key] = value
			}
			copy.CriticalAnswers = append([]string(nil), field.Scoring.CriticalAnswers...)
			scoring = &copy
		}
		var condition *formcontract.VisibilityCondition
		if field.Condition != nil {
			copy := *field.Condition
			copy.Values = append([]string(nil), field.Condition.Values...)
			condition = &copy
		}
		fields[i] = formcontract.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: formcontract.Type(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: condition, Scoring: scoring}
	}
	sections := append([]formcontract.Section(nil), request.Sections...)
	return formcontract.Contract{Presentation: request.Presentation, Sections: sections, Fields: fields}
}
