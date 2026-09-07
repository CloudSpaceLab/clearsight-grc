package evidence

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type CompletedResponseAnswer struct {
	FieldID string                   `json:"field_id"`
	Label   string                   `json:"label"`
	Type    string                   `json:"type"`
	Value   formcontract.AnswerValue `json:"value"`
}
type CompletedResponseAnswers struct {
	SubmissionID                   string                    `json:"submission_id"`
	Answers                        []CompletedResponseAnswer `json:"answers"`
	subjectType, subjectID, formID string
	formVersion                    int64
}
type completedResponseAnswerStore interface {
	ReadCompletedResponseAnswers(context.Context, string, string, string) (CompletedResponseAnswers, error)
}

func (s *DistributionService) GetCompletedResponseAnswers(ctx context.Context, tenant, entity, principal, revisionID string) (CompletedResponseAnswers, error) {
	summary, revision, err := s.GetCompletedResponse(ctx, tenant, entity, principal, revisionID)
	if err != nil {
		return CompletedResponseAnswers{}, err
	}
	switch summary.SubjectType {
	case "PROGRAM", "MATTER", "VENDOR_RELATIONSHIP":
	default:
		return CompletedResponseAnswers{}, ErrNotFound
	}
	reader, ok := s.store.(completedResponseAnswerStore)
	if !ok {
		return CompletedResponseAnswers{}, ErrNotFound
	}
	result, err := reader.ReadCompletedResponseAnswers(ctx, tenant, entity, revision.SubmissionID)
	if err != nil {
		return CompletedResponseAnswers{}, err
	}
	if result.subjectType != summary.SubjectType || result.subjectID != summary.SubjectID || result.formID != summary.FormTemplateID || result.formVersion != summary.FormTemplateVersion {
		return CompletedResponseAnswers{}, ErrNotFound
	}
	return result, nil
}
func readCompletedResponseAnswers(ctx context.Context, repo Repository, tenant, entity, submissionID string) (CompletedResponseAnswers, error) {
	reader, ok := repo.(SubmissionReader)
	if !ok {
		return CompletedResponseAnswers{}, ErrNotFound
	}
	submission, err := reader.GetSubmission(ctx, tenant, submissionID)
	if err != nil {
		return CompletedResponseAnswers{}, err
	}
	request, err := repo.GetRequest(ctx, tenant, submission.RequestID)
	if err != nil || request.LegalEntityID != entity {
		return CompletedResponseAnswers{}, ErrNotFound
	}
	result := CompletedResponseAnswers{SubmissionID: submission.ID, Answers: []CompletedResponseAnswer{}, subjectType: request.SubjectType, subjectID: request.SubjectID, formID: request.FormTemplateID, formVersion: request.FormTemplateVersion}
	for _, field := range request.Fields {
		result.Answers = append(result.Answers, CompletedResponseAnswer{FieldID: field.ID, Label: field.Label, Type: field.Type, Value: submission.Answers[field.ID]})
	}
	return result, nil
}
func (s *MemoryDistributionStore) ReadCompletedResponseAnswers(ctx context.Context, tenant, entity, submission string) (CompletedResponseAnswers, error) {
	return readCompletedResponseAnswers(ctx, s.repo, tenant, entity, submission)
}
