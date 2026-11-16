package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"strings"
	"time"
)

type InternalDraftRecord struct {
	Draft                           ResponseDraft
	PrincipalID                     string
	ExpectedVersion, RequestVersion int64
}
type internalResponseStore interface {
	GetInternalDraft(context.Context, string, string, string) (ResponseDraft, error)
	SaveInternalDraft(context.Context, InternalDraftRecord) (ResponseDraft, error)
	ListSubmissionReviews(context.Context, string, string, string) ([]SubmissionReview, error)
	RecordSubmissionReview(context.Context, Request, Submission, SubmissionReview, int64) (SubmissionReview, error)
}

func (s *Service) internalDraftRequest(ctx context.Context, requestID string) (Request, identity.Actor, error) {
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
	if !requestOpenAt(req, s.now().UTC()) {
		return req, actor, ErrRequestClosed
	}
	return req, actor, nil
}
func (s *Service) GetInternalResponseDraft(ctx context.Context, requestID string) (ResponseDraft, error) {
	req, actor, err := s.internalDraftRequest(ctx, requestID)
	if err != nil {
		return ResponseDraft{}, err
	}
	store, ok := s.repo.(internalResponseStore)
	if !ok {
		return ResponseDraft{}, ErrNotFound
	}
	value, err := store.GetInternalDraft(ctx, req.TenantID, req.ID, actor.PrincipalID)
	if err == ErrNotFound {
		return ResponseDraft{TenantID: req.TenantID, RequestID: req.ID, Answers: map[string]formcontract.AnswerValue{}, PresentationMode: defaultDraftPresentation(req)}, nil
	}
	return value, err
}
func (s *Service) SaveInternalResponseDraft(ctx context.Context, requestID string, input SaveDraftInput) (ResponseDraft, error) {
	req, actor, err := s.internalDraftRequest(ctx, requestID)
	if err != nil {
		return ResponseDraft{}, err
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
	return store.SaveInternalDraft(ctx, InternalDraftRecord{Draft: ResponseDraft{ID: id, TenantID: req.TenantID, RequestID: req.ID, Answers: cloneAnswerValues(input.Answers), PresentationMode: input.PresentationMode, UpdatedAt: s.now().UTC()}, PrincipalID: actor.PrincipalID, ExpectedVersion: input.ExpectedVersion, RequestVersion: req.Version})
}

type GroupReviewDecision struct {
	GroupID     string   `json:"group_id"`
	Outcome     string   `json:"outcome"`
	Rationale   string   `json:"rationale"`
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
}
type RecordSubmissionReviewInput struct {
	ExpectedVersion int64                 `json:"expected_version"`
	Rationale       string                `json:"rationale"`
	Groups          []GroupReviewDecision `json:"groups"`
}
type SubmissionReview struct {
	ID           string                `json:"id"`
	RequestID    string                `json:"request_id"`
	SubmissionID string                `json:"submission_id"`
	Status       string                `json:"status"`
	Rationale    string                `json:"rationale"`
	Groups       []GroupReviewDecision `json:"groups"`
	ReviewedBy   string                `json:"reviewed_by"`
	Version      int64                 `json:"version"`
	CreatedAt    time.Time             `json:"created_at"`
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
	groups := map[string]bool{}
	for _, f := range req.Fields {
		key := f.SectionID
		if key == "" {
			key = "all"
		}
		groups[key] = true
	}
	if len(input.Groups) != len(groups) {
		return nil, fmt.Errorf("%w: review every question group", ErrDraftInvalid)
	}
	out := make([]GroupReviewDecision, 0, len(input.Groups))
	seen := map[string]bool{}
	for _, decision := range input.Groups {
		if !groups[decision.GroupID] || seen[decision.GroupID] || (decision.Outcome != "REVIEWED" && decision.Outcome != "FOLLOW_UP_REQUIRED") || strings.TrimSpace(decision.Rationale) == "" || len(decision.Rationale) > 4000 {
			return nil, ErrDraftInvalid
		}
		seen[decision.GroupID] = true
		decision.Rationale = strings.TrimSpace(decision.Rationale)
		decision.ArtifactIDs = nil
		for _, f := range req.Fields {
			key := f.SectionID
			if key == "" {
				key = "all"
			}
			if key == decision.GroupID {
				decision.ArtifactIDs = append(decision.ArtifactIDs, sub.Answers[f.ID].ArtifactIDs...)
			}
		}
		out = append(out, decision)
	}
	return out, nil
}
