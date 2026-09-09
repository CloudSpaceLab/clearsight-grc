package evidence

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type workspaceArtifactLoader func(context.Context, string, string, string) (Artifact, error)

type workspaceValidationRepository struct {
	Repository
	loadArtifact workspaceArtifactLoader
}

func (repo workspaceValidationRepository) GetArtifact(ctx context.Context, tenantID, requestID, artifactID string) (Artifact, error) {
	if repo.loadArtifact == nil {
		return Artifact{}, ErrNotFound
	}
	return repo.loadArtifact(ctx, tenantID, requestID, artifactID)
}

func validateWorkspaceAnswerSet(
	ctx context.Context,
	repo Repository,
	request Request,
	answers map[string]formcontract.AnswerValue,
	requireComplete bool,
	loadArtifact workspaceArtifactLoader,
	now time.Time,
) error {
	if repo == nil {
		return ErrWorkspaceUnavailable
	}
	validator := NewService(workspaceValidationRepository{Repository: repo, loadArtifact: loadArtifact}, nil)
	validator.now = func() time.Time { return now }
	return validator.validateAnswerSet(ctx, request, answers, requireComplete)
}
