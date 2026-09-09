package thirdparty

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"strings"
	"time"
)

const AssessmentCollectionReconcileCommand = "thirdparty.assessment.reconcile_document"

type AssessmentCollectionField struct {
	FieldID              string                         `json:"field_id"`
	Label                string                         `json:"label"`
	Type                 string                         `json:"type"`
	Required             bool                           `json:"required"`
	CollectionState      string                         `json:"collection_state"`
	VendorActionRequired bool                           `json:"vendor_action_required"`
	BankReviewState      string                         `json:"bank_review_state"`
	Resolution           *evidence.CollectionResolution `json:"resolution,omitempty"`
}
type AssessmentCollection struct {
	AssessmentID          string                      `json:"assessment_id"`
	AssessmentVersion     int64                       `json:"assessment_version"`
	RequestID             string                      `json:"request_id,omitempty"`
	RequestVersion        int64                       `json:"request_version,omitempty"`
	Prepared              bool                        `json:"prepared"`
	Deadline              *time.Time                  `json:"deadline,omitempty"`
	AudienceHint          string                      `json:"audience_hint,omitempty"`
	CanStartReview        bool                        `json:"can_start_review"`
	CanReconcile          bool                        `json:"can_reconcile"`
	ObservedAt            time.Time                   `json:"observed_at"`
	Fields                []AssessmentCollectionField `json:"fields"`
	VendorPendingCount    int                         `json:"vendor_pending_count"`
	BankPendingCount      int                         `json:"bank_pending_count"`
	AcceptedDocumentCount int                         `json:"accepted_document_count"`
}
type ReconcileAssessmentCollectionInput struct {
	ExpectedVersion          int64  `json:"expected_version"`
	RequestID                string `json:"request_id"`
	ExpectedRequestVersion   int64  `json:"expected_request_version"`
	SourceSubmissionID       string `json:"source_submission_id"`
	SourceFieldID            string `json:"source_field_id"`
	SourceArtifactID         string `json:"source_artifact_id"`
	SourceResponseRevisionID string `json:"source_response_revision_id,omitempty"`
	Rationale                string `json:"rationale"`
}
type ReconcileAssessmentCollectionOutcome struct {
	Collection AssessmentCollection          `json:"collection"`
	Receipt    evidence.CollectionResolution `json:"receipt"`
}
type collectionDocumentReader interface {
	ListDocuments(context.Context, evidence.DocumentQuery) (evidence.DocumentPage, error)
}
type collectionRequestWriter interface {
	MutateCollectionRequest(context.Context, string, string, int64, *evidence.CollectionResolution, func(*evidence.Request) error) (evidence.Request, error)
}
type assessmentCollectionRecord struct {
	Scope
	AssessmentID, RequestID, FieldID, ActorPrincipalID string
	ExpectedVersion, ExpectedRequestVersion            int64
	Resolution                                         evidence.CollectionResolution
	At                                                 time.Time
	Review                                             bool
	Authorize                                          func(context.Context) error
}
type assessmentCollectionStore interface {
	WriteAssessmentCollection(context.Context, assessmentCollectionRecord, collectionRequestWriter) (Assessment, evidence.Request, evidence.CollectionResolution, error)
}

func (s *AssessmentReviewService) ConfigureCollectionSources(reader collectionDocumentReader, forms assessmentFormReader) {
	s.collectionSources = reader
	s.collectionForms = forms
}

func collectionEditable(status AssessmentStatus) bool {
	return status == AssessmentReadyToSend || status == AssessmentCollecting || status == AssessmentSubmitted || status == AssessmentUnderReview
}

func (s *AssessmentReviewService) GetCollection(ctx context.Context, actor Actor, assessmentID string) (AssessmentCollection, error) {
	a, err := s.assessments.GetAssessment(ctx, actor, assessmentID)
	if err != nil {
		return AssessmentCollection{}, err
	}
	if err = s.authorizeRead(ctx, actor, scopeFrom(actor), a); err != nil {
		return AssessmentCollection{}, err
	}
	var request evidence.Request
	answers := map[string]formcontract.AnswerValue{}
	if a.CurrentRequestID != "" {
		request, err = s.evidence.GetRequest(ctx, a.TenantID, a.CurrentRequestID)
		if err != nil {
			return AssessmentCollection{}, err
		}
		if err = validateCollectionRequest(a, request); err != nil {
			return AssessmentCollection{}, err
		}
		request = evidence.RefreshCollectionResolutions(ctx, request, s.evidence.GetArtifact, s.assessments.now())
		if a.SubmissionID != "" {
			submission, readErr := s.evidence.GetSubmission(ctx, a.TenantID, a.SubmissionID)
			if readErr != nil {
				return AssessmentCollection{}, readErr
			}
			if submission.RequestID != request.ID {
				return AssessmentCollection{}, ErrNotFound
			}
			answers = submission.Answers
		}
	} else if s.collectionForms != nil {
		form, readErr := s.collectionForms.ReusableFormRevision(ctx, a.TenantID, a.LegalEntityID, a.FormTemplateID, a.FormTemplateVersion)
		if readErr != nil {
			return AssessmentCollection{}, readErr
		}
		sections, fields, readErr := ComposeAssessmentScope(formcontract.Contract{Presentation: form.Presentation, Sections: form.Sections, Fields: form.Fields}, a.ScopeKind, a.SelectedFieldIDs)
		if readErr != nil {
			return AssessmentCollection{}, readErr
		}
		request.Presentation = form.Presentation
		request.Sections = sections
		for _, field := range fields {
			request.Fields = append(request.Fields, evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Condition: field.Condition, Options: field.Options})
		}
	}
	result, err := buildAssessmentCollection(a, request, answers, s.assessments.now().UTC())
	if err != nil {
		return result, err
	}
	for i := range result.Fields {
		row := &result.Fields[i]
		if row.Type != "vendor_document" || row.CollectionState != "RECEIVED" || answers[row.FieldID].Document == nil {
			continue
		}
		artifact, readErr := s.evidence.GetArtifact(ctx, a.TenantID, request.ID, answers[row.FieldID].Document.ArtifactID)
		if readErr != nil || artifact.Status != evidence.ArtifactAvailable {
			if row.BankReviewState == "PENDING" {
				result.BankPendingCount--
			}
			row.BankReviewState = "NOT_REQUIRED"
			row.CollectionState = "MISSING"
			if row.Required && !row.VendorActionRequired {
				row.VendorActionRequired = true
				result.VendorPendingCount++
			}
		}
	}
	if documents, ok := s.links.(AssessmentReviewDocumentReader); ok && a.CurrentRequestID != "" {
		stored, readErr := documents.ListAssessmentDocuments(ctx, scopeFrom(actor), a.ID, assessmentReviewMaxArtifacts+1)
		if readErr != nil {
			return result, readErr
		}
		if len(stored) > assessmentReviewMaxArtifacts {
			return result, ErrInvalid
		}
		applyCollectionDocumentReviews(&result, request.ID, answers, stored, s.assessments.now())
	}
	if a.Status == AssessmentSubmitted {
		_, authErr := s.assessments.authorize(ctx, a.ID, assessmentObjectType, AssessmentReviewCommand, authority.ResponsibilityReviewer)
		result.CanStartReview = authErr == nil
	}
	if result.Prepared && collectionEditable(a.Status) {
		_, authErr := s.assessments.authorize(ctx, a.ID, assessmentObjectType, AssessmentCollectionReconcileCommand, authority.ResponsibilityReviewer)
		result.CanReconcile = authErr == nil
	}
	return result, nil
}

func validateCollectionRequest(a Assessment, r evidence.Request) error {
	if r.ID != a.CurrentRequestID || r.LegalEntityID != a.LegalEntityID || r.SubjectType != "VENDOR_RELATIONSHIP" || r.SubjectID != a.RelationshipID || r.Origin.Type != AssessmentRequestOrigin || r.Origin.ID != a.ID || r.FormTemplateID != a.FormTemplateID || r.FormTemplateVersion != a.FormTemplateVersion {
		return ErrNotFound
	}
	return nil
}

func buildAssessmentCollection(a Assessment, request evidence.Request, answers map[string]formcontract.AnswerValue, now time.Time) (AssessmentCollection, error) {
	result := AssessmentCollection{AssessmentID: a.ID, AssessmentVersion: a.Version, RequestID: request.ID, RequestVersion: request.Version, Prepared: request.ID != "", ObservedAt: now, Fields: []AssessmentCollectionField{}}
	if request.ID != "" {
		deadline := request.Deadline
		result.Deadline = &deadline
		result.AudienceHint = request.Recipient.AudienceHint
	}
	if len(request.Fields) == 0 {
		return result, nil
	}
	visible, err := formcontract.VisibleFields(reviewContract(request), answers)
	if err != nil {
		return result, err
	}
	shown := map[string]bool{}
	for _, field := range visible {
		shown[field.ID] = true
	}
	sections := map[string]*formcontract.VisibilityCondition{}
	for _, section := range request.Sections {
		sections[section.ID] = section.Condition
	}
	for _, field := range request.Fields {
		row := AssessmentCollectionField{FieldID: field.ID, Label: field.Label, Type: field.Type, Required: field.Required, CollectionState: "NOT_REQUIRED", BankReviewState: "NOT_REQUIRED", Resolution: field.CollectionResolution}
		unknown := false
		for _, condition := range []*formcontract.VisibilityCondition{field.Condition, sections[field.SectionID]} {
			if condition != nil {
				value, ok := answers[condition.FieldID]
				if !ok || !value.Answered() {
					unknown = true
				}
			}
		}
		if unknown {
			row.CollectionState = "CONDITION_UNKNOWN"
			row.VendorActionRequired = field.Required
		} else if shown[field.ID] {
			value, answered := answers[field.ID]
			answered = answered && value.Answered()
			if evidence.CollectionFieldFulfilled(field, now) {
				row.CollectionState = "REUSED"
				row.BankReviewState = field.CollectionResolution.BankReviewState
			} else if answered {
				row.CollectionState = "RECEIVED"
				if field.Type == "vendor_document" {
					row.BankReviewState = "PENDING"
				}
			} else if field.Required {
				row.CollectionState = "MISSING"
				row.VendorActionRequired = true
			}
		}
		if row.VendorActionRequired {
			result.VendorPendingCount++
		}
		if row.BankReviewState == "PENDING" {
			result.BankPendingCount++
		}
		if row.BankReviewState == "VALIDATED" {
			result.AcceptedDocumentCount++
		}
		result.Fields = append(result.Fields, row)
	}
	return result, nil
}

func (s *AssessmentReviewService) ReconcileCollection(ctx context.Context, _ Actor, assessmentID, fieldID string, input ReconcileAssessmentCollectionInput) (ReconcileAssessmentCollectionOutcome, error) {
	input.Rationale = strings.TrimSpace(input.Rationale)
	if !validAssessmentIdentifiers(assessmentID, input.RequestID, input.SourceSubmissionID, input.SourceArtifactID) || strings.TrimSpace(fieldID) == "" || strings.TrimSpace(input.SourceFieldID) == "" || input.ExpectedVersion < 1 || input.ExpectedRequestVersion < 1 || input.Rationale == "" || len(input.Rationale) > 2000 || s.collectionSources == nil {
		return ReconcileAssessmentCollectionOutcome{}, ErrInvalid
	}
	actor, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentCollectionReconcileCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return ReconcileAssessmentCollectionOutcome{}, err
	}
	a, err := s.assessments.GetAssessment(ctx, actor, assessmentID)
	if err != nil {
		return ReconcileAssessmentCollectionOutcome{}, err
	}
	if a.Version != input.ExpectedVersion {
		return ReconcileAssessmentCollectionOutcome{}, ErrVersionConflict
	}
	if !collectionEditable(a.Status) || a.CurrentRequestID != input.RequestID {
		return ReconcileAssessmentCollectionOutcome{}, ErrInvalidAssessmentTransition
	}
	source, err := s.collectionSources.ListDocuments(ctx, evidence.DocumentQuery{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, RelationshipID: a.RelationshipID, SubmissionID: input.SourceSubmissionID, FieldID: input.SourceFieldID, ArtifactID: input.SourceArtifactID, ResponseRevisionID: input.SourceResponseRevisionID, CurrentOnly: true, Limit: 1})
	if err != nil {
		return ReconcileAssessmentCollectionOutcome{}, err
	}
	if len(source.Items) != 1 {
		return ReconcileAssessmentCollectionOutcome{}, ErrNotFound
	}
	occurrence := source.Items[0]
	if occurrence.SubmissionChannel != "MAGIC_LINK" {
		return ReconcileAssessmentCollectionOutcome{}, ErrAssessmentCompletionBlocked
	}
	if occurrence.RelationshipID != a.RelationshipID || occurrence.SubmissionID != input.SourceSubmissionID || occurrence.FieldID != input.SourceFieldID || occurrence.ArtifactID != input.SourceArtifactID || (input.SourceResponseRevisionID != "" && occurrence.ResponseRevisionID != input.SourceResponseRevisionID) {
		return ReconcileAssessmentCollectionOutcome{}, ErrNotFound
	}
	receiptID, err := id.NewUUIDv7()
	if err != nil {
		return ReconcileAssessmentCollectionOutcome{}, err
	}
	receipt := evidence.CollectionResolution{ID: receiptID, Version: 1, Source: occurrence, SourceArtifactRequestID: occurrence.ArtifactRequestID, Document: formcontract.DocumentAnswer{ArtifactID: occurrence.ArtifactID, ExpiresOn: occurrence.ExpiresOn}, ReconciledBy: actor.PrincipalID, ReconciledAt: s.assessments.now().UTC(), Rationale: input.Rationale, BankReviewState: "PENDING"}
	if receipt.SourceArtifactRequestID == "" {
		receipt.SourceArtifactRequestID = occurrence.RequestID
	}
	if !evidence.CollectionFieldFulfilled(evidence.Field{Type: "vendor_document", CollectionResolution: &receipt}, s.assessments.now()) {
		return ReconcileAssessmentCollectionOutcome{}, ErrAssessmentCompletionBlocked
	}
	store, ok := s.links.(assessmentCollectionStore)
	if !ok {
		return ReconcileAssessmentCollectionOutcome{}, ErrAssessmentReadinessUnavailable
	}
	writer, _ := s.evidence.(collectionRequestWriter)
	saved, request, receipt, err := store.WriteAssessmentCollection(ctx, assessmentCollectionRecord{Scope: scopeFrom(actor), AssessmentID: a.ID, RequestID: input.RequestID, FieldID: fieldID, ActorPrincipalID: actor.PrincipalID, ExpectedVersion: input.ExpectedVersion, ExpectedRequestVersion: input.ExpectedRequestVersion, Resolution: receipt, At: s.assessments.now().UTC(), Authorize: func(txctx context.Context) error {
		_, err := s.assessments.authorize(txctx, assessmentID, assessmentObjectType, AssessmentCollectionReconcileCommand, authority.ResponsibilityReviewer)
		return err
	}}, writer)
	if err != nil {
		return ReconcileAssessmentCollectionOutcome{}, err
	}
	// The committed receipt is returned even if a derived read is unavailable.
	collection, _ := buildAssessmentCollection(saved, request, nil, s.assessments.now().UTC())
	collection.CanReconcile = collectionEditable(saved.Status)
	if refreshed, readErr := s.GetCollection(ctx, actor, assessmentID); readErr == nil {
		collection = refreshed
	}
	return ReconcileAssessmentCollectionOutcome{Collection: collection, Receipt: receipt}, nil
}

func applyCollectionRecord(current *Assessment, request *evidence.Request, record assessmentCollectionRecord) (evidence.CollectionResolution, error) {
	if current.Version != record.ExpectedVersion || request.Version != record.ExpectedRequestVersion {
		return evidence.CollectionResolution{}, ErrVersionConflict
	}
	if !collectionEditable(current.Status) {
		return evidence.CollectionResolution{}, ErrInvalidAssessmentTransition
	}
	if err := validateCollectionRequest(*current, *request); err != nil {
		return evidence.CollectionResolution{}, err
	}
	if request.Status == evidence.RequestCancelled || request.Status == evidence.RequestExpired {
		return evidence.CollectionResolution{}, ErrInvalidAssessmentTransition
	}
	index := -1
	for i, field := range request.Fields {
		if field.ID == record.FieldID && field.Type == "vendor_document" {
			index = i
			break
		}
	}
	if index < 0 {
		return evidence.CollectionResolution{}, ErrNotFound
	}
	receipt := record.Resolution
	if !record.Review {
		if err := evidence.ValidateCollectionSourceForField(request.Fields[index], receipt.Source); err != nil {
			return evidence.CollectionResolution{}, ErrInvalid
		}
	}
	prior := request.Fields[index].CollectionResolution
	if prior != nil {
		receipt.Version = prior.Version + 1
		receipt.SupersedesID = prior.ID
	}
	if (!record.Review || receipt.BankReviewState == "VALIDATED") && !evidence.CollectionFieldFulfilled(evidence.Field{Type: "vendor_document", CollectionResolution: &receipt}, record.At) {
		return evidence.CollectionResolution{}, ErrAssessmentCompletionBlocked
	}
	request.Fields[index].CollectionResolution = &receipt
	request.Version++
	request.UpdatedAt = record.At
	current.Version++
	current.UpdatedAt = record.At
	// No submission is created when every requested item is already held.
	if current.SubmissionID == "" && current.Status != AssessmentUnderReview {
		all := len(request.Fields) > 0
		for _, field := range request.Fields {
			if field.Condition != nil || field.Required && !evidence.CollectionFieldFulfilled(field, record.At) {
				all = false
			}
		}
		for _, section := range request.Sections {
			if section.Condition != nil {
				all = false
			}
		}
		if all {
			current.Status = AssessmentUnderReview
			at := record.At
			current.CollectionCompletedAt = &at
			current.ReviewStartedAt = &at
			current.ReviewerPrincipalID = record.ActorPrincipalID
		}
	}
	return receipt, nil
}

func applyCollectionDocumentReviews(result *AssessmentCollection, requestID string, answers map[string]formcontract.AnswerValue, documents []AssessmentDocument, now time.Time) {
	for i := range result.Fields {
		row := &result.Fields[i]
		if row.Resolution != nil || row.Type != "vendor_document" || row.CollectionState != "RECEIVED" {
			continue
		}
		answer := answers[row.FieldID]
		if answer.Document == nil {
			continue
		}
		var selected *AssessmentDocument
		for j := range documents {
			doc := &documents[j]
			if doc.RequestID == requestID && doc.ArtifactID == answer.Document.ArtifactID && (selected == nil || doc.Version > selected.Version) {
				selected = doc
			}
		}
		if selected == nil {
			continue
		}
		status := selected.Status
		expired := status == AssessmentDocumentExpired || (selected.ExpiresOn != nil && now.UTC().Format("2006-01-02") > selected.ExpiresOn.UTC().Format("2006-01-02"))
		if expired {
			if row.BankReviewState == "PENDING" {
				result.BankPendingCount--
			}
			row.BankReviewState = "NOT_REQUIRED"
			row.CollectionState = "MISSING"
			if row.Required && !row.VendorActionRequired {
				row.VendorActionRequired = true
				result.VendorPendingCount++
			}
			continue
		}
		if status != AssessmentDocumentValidated && status != AssessmentDocumentRejected {
			continue
		}
		if row.BankReviewState == "PENDING" {
			result.BankPendingCount--
		}
		row.BankReviewState = string(status)
		if status == AssessmentDocumentValidated {
			result.AcceptedDocumentCount++
		}
	}
}
