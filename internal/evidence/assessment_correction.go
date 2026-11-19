package evidence

import (
	"context"
	"slices"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"time"
)

const reviewFollowUp = "FOLLOW_UP_REQUIRED"

type AssessmentCorrection struct {
	SubmissionID string `json:"submission_id"`
	ReviewID     string `json:"review_id"`
}

func CorrectionReference(submissionID, reviewID string) *AssessmentCorrection {
	if submissionID == "" && reviewID == "" {
		return nil
	}
	return &AssessmentCorrection{SubmissionID: submissionID, ReviewID: reviewID}
}

type AssessmentCorrectionWindow struct {
	AssessmentCorrection
	DueAt time.Time `json:"due_at"`
}

type assessmentCorrectionReader interface {
	AssessmentCorrectionBase(context.Context, Request, string, AssessmentCorrection) (Submission, error)
}

func (s *Service) correctionBase(ctx context.Context, req Request, principal string, proof AssessmentCorrection) (Submission, error) {
	if !RequestAssignedTo(req, principal) {
		return Submission{}, ErrRecipientMismatch
	}
	if req.Status != RequestSubmitted || req.SubjectType != "PROGRAM" || req.AudienceType != "INTERNAL" ||
		req.FormTemplateID == "" || req.FormTemplateVersion < 1 || req.KnownFacts["reviewer"] == "" ||
		proof.SubmissionID == "" || proof.ReviewID == "" {
		return Submission{}, ErrRequestClosed
	}
	store, ok := s.repo.(assessmentCorrectionReader)
	if !ok {
		return Submission{}, ErrRequestClosed
	}
	return store.AssessmentCorrectionBase(ctx, req, principal, proof)
}

// Files from the previous response stay bound to that original submission.
// Only attachments retained on the same question can be carried forward.
func correctionUploads(req Request, base, updated Submission) []string {
	var uploads []string
	for _, field := range req.Fields {
		if !isFileFieldType(field.Type) || !updated.Answers[field.ID].Answered() {
			continue
		}
		f := formcontract.Field{ID: field.ID, Label: field.Label, Type: formcontract.Type(strings.ToLower(strings.TrimSpace(field.Type)))}
		original, _ := answerArtifactIDs(f, base.Answers[field.ID])
		current, _ := answerArtifactIDs(f, updated.Answers[field.ID]) // Already validated by Submit.
		for _, id := range current {
			id = strings.TrimSpace(id)
			if !slices.Contains(original, id) {
				uploads = append(uploads, id)
			}
		}
	}
	return uploads
}
