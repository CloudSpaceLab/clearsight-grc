package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type InternalDraftRecord struct {
	Draft                           ResponseDraft
	Correction                      *AssessmentCorrection
	PrincipalID                     string
	ExpectedVersion, RequestVersion int64
}
type internalResponseStore interface {
	GetInternalDraft(context.Context, string, string, string, string) (ResponseDraft, error)
	SaveInternalDraft(context.Context, InternalDraftRecord) (ResponseDraft, error)
	ListSubmissionReviews(context.Context, string, string, string) ([]SubmissionReview, error)
	RecordSubmissionReview(context.Context, Request, Submission, SubmissionReview, int64) (SubmissionReview, error)
}

func (s *Service) internalDraftRequest(ctx context.Context, requestID string, correction *AssessmentCorrection) (Request, identity.Actor, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return Request{}, actor, ErrRecipientMismatch
	}
	req, err := s.GetRequestForEntity(ctx, actor.TenantID, actor.LegalEntityID, requestID)
	if err != nil {
		return req, actor, err
	}
	if !RequestAssignedTo(req, actor.PrincipalID) {
		return req, actor, ErrRecipientMismatch
	}
	if correction != nil {
		return req, actor, validateCurrentRequestScope(ctx, s.repo, req, actor.LegalEntityID)
	}
	if !requestOpenAt(req, s.now().UTC()) {
		return req, actor, ErrRequestClosed
	}
	return req, actor, nil
}
func (s *Service) GetInternalResponseDraft(ctx context.Context, requestID string) (ResponseDraft, error) {
	return s.GetInternalAssessmentDraft(ctx, requestID, nil)
}

func (s *Service) GetInternalAssessmentDraft(ctx context.Context, requestID string, correction *AssessmentCorrection) (ResponseDraft, error) {
	req, actor, err := s.internalDraftRequest(ctx, requestID, correction)
	if err != nil {
		return ResponseDraft{}, err
	}
	store, ok := s.repo.(internalResponseStore)
	if !ok {
		return ResponseDraft{}, ErrNotFound
	}
	answers := map[string]formcontract.AnswerValue{}
	reviewID := ""
	if correction != nil {
		base, err := s.correctionBase(ctx, req, actor.PrincipalID, *correction)
		if err != nil {
			return ResponseDraft{}, err
		}
		answers, reviewID = base.Answers, correction.ReviewID
	}
	value, err := store.GetInternalDraft(ctx, req.TenantID, req.ID, actor.PrincipalID, reviewID)
	if err == ErrNotFound {
		return ResponseDraft{TenantID: req.TenantID, RequestID: req.ID, Answers: answers, PresentationMode: defaultDraftPresentation(req)}, nil
	}
	if err != nil {
		return ResponseDraft{}, err
	}
	value.Attachments = s.internalDraftAttachments(ctx, req, actor.PrincipalID, value.Answers)
	return value, nil
}

func (s *Service) internalDraftAttachments(ctx context.Context, req Request, principal string, answers map[string]formcontract.AnswerValue) []InternalDraftAttachment {
	ids, err := submissionArtifactIDs(req, answers)
	if err != nil {
		return nil
	}
	var attachments []InternalDraftAttachment
	for _, id := range ids {
		file, err := s.repo.GetArtifact(ctx, req.TenantID, req.ID, id)
		if err != nil || file.SubmissionID != "" || file.CreatedBy != principal {
			continue
		}
		attachments = append(attachments, InternalDraftAttachment{ID: file.ID, FileName: file.FileName, Status: file.Status, SizeBytes: file.SizeBytes})
	}
	return attachments
}

func (s *Service) SaveInternalResponseDraft(ctx context.Context, requestID string, input SaveDraftInput) (ResponseDraft, error) {
	req, actor, err := s.internalDraftRequest(ctx, requestID, input.Correction)
	if err != nil {
		return ResponseDraft{}, err
	}
	if input.Correction != nil {
		if _, err := s.correctionBase(ctx, req, actor.PrincipalID, *input.Correction); err != nil {
			return ResponseDraft{}, err
		}
	}
	if input.ExpectedVersion < 0 {
		return ResponseDraft{}, ErrDraftInvalid
	}
	if input.PresentationMode == "" {
		input.PresentationMode = defaultDraftPresentation(req)
	}
	if input.PresentationMode != formcontract.PresentationClassic && input.PresentationMode != formcontract.PresentationWizard && input.PresentationMode != formcontract.PresentationAutomatic {
		return ResponseDraft{}, ErrDraftInvalid
	}
	if input.Answers == nil {
		input.Answers = map[string]formcontract.AnswerValue{}
	}
	encoded, err := json.Marshal(input.Answers)
	if err != nil || len(encoded) > maxDraftAnswersBytes {
		return ResponseDraft{}, ErrDraftInvalid
	}
	if err = s.validateDraftAnswers(ctx, req, input.Answers); err != nil {
		return ResponseDraft{}, fmt.Errorf("%w: %v", ErrDraftInvalid, err)
	}
	store, ok := s.repo.(internalResponseStore)
	if !ok {
		return ResponseDraft{}, ErrNotFound
	}
	id, err := platformid.NewUUIDv7()
	if err != nil {
		return ResponseDraft{}, err
	}
	return store.SaveInternalDraft(ctx, InternalDraftRecord{Correction: input.Correction, Draft: ResponseDraft{ID: id, TenantID: req.TenantID, RequestID: req.ID, Answers: cloneAnswerValues(input.Answers), PresentationMode: input.PresentationMode, UpdatedAt: s.now().UTC()}, PrincipalID: actor.PrincipalID, ExpectedVersion: input.ExpectedVersion, RequestVersion: req.Version})
}

type QuestionReviewDecision struct {
	FieldID     string   `json:"field_id"`
	Outcome     string   `json:"outcome"`
	Rationale   string   `json:"rationale,omitempty"`
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
}

type GroupReviewDecision struct {
	Questions   []QuestionReviewDecision `json:"questions,omitempty"`
	GroupID     string                   `json:"group_id"`
	Outcome     string                   `json:"outcome"`
	Rationale   string                   `json:"rationale"`
	ArtifactIDs []string                 `json:"artifact_ids,omitempty"`
}
type RecordSubmissionReviewInput struct {
	SubmissionID    string                `json:"submission_id,omitempty"`
	CorrectionDueAt *time.Time            `json:"correction_due_at,omitempty"`
	ExpectedVersion int64                 `json:"expected_version"`
	Rationale       string                `json:"rationale"`
	Groups          []GroupReviewDecision `json:"groups"`
}
type SubmissionReview struct {
	CorrectionDueAt *time.Time            `json:"correction_due_at,omitempty"`
	ID              string                `json:"id"`
	RequestID       string                `json:"request_id"`
	SubmissionID    string                `json:"submission_id"`
	Status          string                `json:"status"`
	Rationale       string                `json:"rationale"`
	Groups          []GroupReviewDecision `json:"groups"`
	ReviewedBy      string                `json:"reviewed_by"`
	Version         int64                 `json:"version"`
	CreatedAt       time.Time             `json:"created_at"`
}

func (s *Service) SubmissionReviews(ctx context.Context, tenant, entity, program string) ([]SubmissionReview, error) {
	store, ok := s.repo.(internalResponseStore)
	if !ok {
		return nil, ErrNotFound
	}
	return store.ListSubmissionReviews(ctx, tenant, entity, program)
}
func (s *Service) ReviewInternalSubmission(ctx context.Context, programID, requestID string, input RecordSubmissionReviewInput) (SubmissionReview, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return SubmissionReview{}, ErrRecipientMismatch
	}
	req, err := s.GetRequestForEntity(ctx, actor.TenantID, actor.LegalEntityID, requestID)
	if err != nil {
		return SubmissionReview{}, err
	}
	if req.SubjectType != "PROGRAM" || req.SubjectID != programID || !RequestReviewableBy(req, actor.PrincipalID) || req.CreatedBy == actor.PrincipalID {
		return SubmissionReview{}, ErrRecipientMismatch
	}
	sub, err := s.GetSubmissionForRequest(ctx, req.TenantID, req.ID)
	if err != nil {
		return SubmissionReview{}, err
	}
	if sub.SubmittedBy == actor.PrincipalID {
		return SubmissionReview{}, ErrRecipientMismatch
	}
	if (input.SubmissionID != "" || sub.Revision > 1) && sub.ID != input.SubmissionID {
		return SubmissionReview{}, ErrVersionConflict
	}
	decisions, err := validateSubmissionGroupReview(req, sub, input)
	if err != nil {
		return SubmissionReview{}, err
	}
	id, err := platformid.NewUUIDv7()
	if err != nil {
		return SubmissionReview{}, err
	}
	value := SubmissionReview{ID: id, RequestID: req.ID, SubmissionID: sub.ID, Status: "REVIEWED", Rationale: strings.TrimSpace(input.Rationale), Groups: decisions, ReviewedBy: actor.PrincipalID, Version: input.ExpectedVersion + 1, CreatedAt: s.now().UTC()}
	for _, d := range decisions {
		if d.Outcome == "FOLLOW_UP_REQUIRED" {
			value.Status = "FOLLOW_UP_REQUIRED"
		}
	}
	if due := input.CorrectionDueAt; due != nil {
		if value.Status != reviewFollowUp || !due.After(value.CreatedAt) {
			return SubmissionReview{}, ErrDraftInvalid
		}
		utc := due.UTC()
		value.CorrectionDueAt = &utc
	}
	store, ok := s.repo.(internalResponseStore)
	if !ok {
		return SubmissionReview{}, ErrNotFound
	}
	return store.RecordSubmissionReview(ctx, req, sub, value, input.ExpectedVersion)
}
func validateSubmissionGroupReview(req Request, sub Submission, input RecordSubmissionReviewInput) ([]GroupReviewDecision, error) {
	if input.ExpectedVersion < 0 || strings.TrimSpace(input.Rationale) == "" || len(input.Rationale) > 4000 || len(input.Groups) == 0 || len(input.Groups) > 20 {
		return nil, ErrDraftInvalid
	}
	groups := map[string][]Field{}
	for _, field := range req.Fields {
		group := field.SectionID
		if group == "" {
			group = "all"
		}
		groups[group] = append(groups[group], field)
	}
	if len(input.Groups) != len(groups) {
		return nil, fmt.Errorf("%w: review every question group", ErrDraftInvalid)
	}
	out := make([]GroupReviewDecision, 0, len(input.Groups))
	seen := map[string]bool{}
	for _, decision := range input.Groups {
		fields := groups[decision.GroupID]
		decision.Rationale = strings.TrimSpace(decision.Rationale)
		if len(fields) == 0 || seen[decision.GroupID] || !reviewOutcomeValid(decision.Outcome) ||
			decision.Rationale == "" || len(decision.Rationale) > 4000 || len(decision.Questions) > len(fields) {
			return nil, ErrDraftInvalid
		}
		seen[decision.GroupID] = true
		decision.ArtifactIDs = nil
		fieldArtifacts := make(map[string][]string, len(fields))
		for _, field := range fields {
			var ids []string
			if isFileFieldType(field.Type) && sub.Answers[field.ID].Answered() {
				f := formcontract.Field{ID: field.ID, Label: field.Label, Type: formcontract.Type(strings.ToLower(field.Type))}
				ids, _ = answerArtifactIDs(f, sub.Answers[field.ID])
			}
			fieldArtifacts[field.ID] = ids
			decision.ArtifactIDs = append(decision.ArtifactIDs, ids...)
		}
		for i, question := range decision.Questions {
			ids, exists := fieldArtifacts[question.FieldID]
			question.Rationale = strings.TrimSpace(question.Rationale)
			if !exists || !reviewOutcomeValid(question.Outcome) || len(question.Rationale) > 4000 ||
				(question.Outcome == reviewFollowUp && question.Rationale == "") {
				return nil, ErrDraftInvalid
			}
			delete(fieldArtifacts, question.FieldID)
			question.ArtifactIDs = ids
			decision.Questions[i] = question
			if question.Outcome == reviewFollowUp {
				decision.Outcome = reviewFollowUp
			}
		}
		out = append(out, decision)
	}
	return out, nil
}

func reviewOutcomeValid(outcome string) bool {
	return outcome == "REVIEWED" || outcome == reviewFollowUp
}
