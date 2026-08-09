package continuity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	programReviewItemLimit      = 8
	programReviewEventScanLimit = 64
)

type ProgramReviewCheckpoint struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	ProgramID         string    `json:"program_id"`
	PrincipalID       string    `json:"principal_id"`
	ProgramVersion    int64     `json:"program_version"`
	ProjectionVersion int64     `json:"projection_version"`
	AcceptedAt        time.Time `json:"accepted_at"`
}

type ProgramReviewChange struct {
	Kind       string `json:"kind"`
	Summary    string `json:"summary"`
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
}

type ProgramReviewDigest struct {
	ProgramID                string                   `json:"program_id"`
	State                    string                   `json:"state"`
	ReviewRequired           bool                     `json:"review_required"`
	Checkpoint               *ProgramReviewCheckpoint `json:"checkpoint,omitempty"`
	CurrentProgramVersion    int64                    `json:"current_program_version"`
	CurrentProjectionVersion int64                    `json:"current_projection_version"`
	CurrentOverall           ProgramState             `json:"current_overall"`
	BaselineOverall          ProgramState             `json:"baseline_overall,omitempty"`
	OpenMatterCount          int                      `json:"open_matter_count"`
	OpenMatterDelta          int                      `json:"open_matter_delta,omitempty"`
	Changes                  []ProgramReviewChange    `json:"changes"`
	ChangesTotal             int                      `json:"changes_total"`
	ChangesOmitted           int                      `json:"changes_omitted"`
	HistoryTruncated         bool                     `json:"history_truncated"`
	CurrentExceptions        []StateReason            `json:"current_exceptions"`
	CurrentExceptionsTotal   int                      `json:"current_exceptions_total"`
	NewExceptions            []StateReason            `json:"new_exceptions"`
	NewExceptionsTotal       int                      `json:"new_exceptions_total"`
	ResolvedExceptions       []StateReason            `json:"resolved_exceptions"`
	ResolvedExceptionsTotal  int                      `json:"resolved_exceptions_total"`
}

type AcceptProgramReviewInput struct {
	TenantID                  string `json:"-"`
	ProgramID                 string `json:"-"`
	PrincipalID               string `json:"-"`
	ExpectedProgramVersion    int64  `json:"expected_program_version"`
	ExpectedProjectionVersion int64  `json:"expected_projection_version"`
}

type ProgramReviewRepository interface {
	LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error)
	RecordProgramReview(ctx context.Context, checkpoint ProgramReviewCheckpoint) (ProgramReviewCheckpoint, error)
	ProgramStateVersion(ctx context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error)
	ProgramEventsAfterVersion(ctx context.Context, tenant, programID string, afterVersion int64, limit int) ([]Event, bool, error)
}

func (s *Service) ProgramReviewDigest(ctx context.Context, tenant, programID, principalID string) (ProgramReviewDigest, error) {
	tenant = strings.TrimSpace(tenant)
	programID = strings.TrimSpace(programID)
	principalID = strings.TrimSpace(principalID)
	if tenant == "" || programID == "" || principalID == "" {
		return ProgramReviewDigest{}, fmt.Errorf("tenant_id, program_id and principal_id are required")
	}
	repo, ok := s.repo.(ProgramReviewRepository)
	if !ok {
		return ProgramReviewDigest{}, fmt.Errorf("Program review checkpoints are unavailable")
	}
	aggregate, err := s.GetProgram(ctx, tenant, programID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	current, err := currentReviewState(aggregate)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	checkpoint, err := repo.LatestProgramReview(ctx, tenant, programID, principalID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	digest := ProgramReviewDigest{
		ProgramID:                programID,
		State:                    "NO_BASELINE",
		ReviewRequired:           true,
		Checkpoint:               checkpoint,
		CurrentProgramVersion:    aggregate.Program.Version,
		CurrentProjectionVersion: current.ProjectionVersion,
		CurrentOverall:           current.Overall,
		OpenMatterCount:          current.OpenMatterCount,
		CurrentExceptions:        limitReasons(current.Reasons, programReviewItemLimit),
		CurrentExceptionsTotal:   len(current.Reasons),
		Changes:                  []ProgramReviewChange{},
		NewExceptions:            []StateReason{},
		ResolvedExceptions:       []StateReason{},
	}
	if checkpoint == nil {
		return digest, nil
	}

	baseline, err := repo.ProgramStateVersion(ctx, tenant, programID, checkpoint.ProjectionVersion)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	if baseline == nil || baseline.ProgramVersion != checkpoint.ProgramVersion {
		return ProgramReviewDigest{}, fmt.Errorf("%w: accepted Program review baseline is unavailable", ErrInvalidState)
	}
	digest.BaselineOverall = baseline.Overall
	digest.OpenMatterDelta = current.OpenMatterCount - baseline.OpenMatterCount
	newReasons, resolvedReasons := diffStateReasons(baseline.Reasons, current.Reasons)
	digest.NewExceptionsTotal = len(newReasons)
	digest.NewExceptions = limitReasons(newReasons, programReviewItemLimit)
	digest.ResolvedExceptionsTotal = len(resolvedReasons)
	digest.ResolvedExceptions = limitReasons(resolvedReasons, programReviewItemLimit)

	events, truncated, err := repo.ProgramEventsAfterVersion(ctx, tenant, programID, checkpoint.ProgramVersion, programReviewEventScanLimit)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	digest.HistoryTruncated = truncated
	changes := deriveProgramReviewChanges(aggregate, *baseline, *current, events, newReasons)
	digest.ChangesTotal = len(changes)
	if len(changes) > programReviewItemLimit {
		digest.ChangesOmitted = len(changes) - programReviewItemLimit
		changes = changes[:programReviewItemLimit]
	}
	digest.Changes = changes
	digest.ReviewRequired = checkpoint.ProgramVersion != aggregate.Program.Version || checkpoint.ProjectionVersion != current.ProjectionVersion || len(changes) > 0
	if digest.ReviewRequired {
		digest.State = "CHANGED"
	} else {
		digest.State = "CURRENT"
	}
	return digest, nil
}

func (s *Service) AcceptProgramReview(ctx context.Context, input AcceptProgramReviewInput) (ProgramReviewDigest, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ProgramID = strings.TrimSpace(input.ProgramID)
	input.PrincipalID = strings.TrimSpace(input.PrincipalID)
	if input.TenantID == "" || input.ProgramID == "" || input.PrincipalID == "" || input.ExpectedProgramVersion < 1 || input.ExpectedProjectionVersion < 1 {
		return ProgramReviewDigest{}, fmt.Errorf("tenant_id, program_id, principal_id and positive expected versions are required")
	}
	repo, ok := s.repo.(ProgramReviewRepository)
	if !ok {
		return ProgramReviewDigest{}, fmt.Errorf("Program review checkpoints are unavailable")
	}
	aggregate, err := s.GetProgram(ctx, input.TenantID, input.ProgramID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	current, err := currentReviewState(aggregate)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	if aggregate.Program.Version != input.ExpectedProgramVersion || current.ProjectionVersion != input.ExpectedProjectionVersion {
		return ProgramReviewDigest{}, ErrVersionConflict
	}
	checkpointID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	_, err = repo.RecordProgramReview(ctx, ProgramReviewCheckpoint{
		ID:                checkpointID,
		TenantID:          input.TenantID,
		ProgramID:         input.ProgramID,
		PrincipalID:       input.PrincipalID,
		ProgramVersion:    input.ExpectedProgramVersion,
		ProjectionVersion: input.ExpectedProjectionVersion,
		AcceptedAt:        s.now().UTC(),
	})
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	return s.ProgramReviewDigest(ctx, input.TenantID, input.ProgramID, input.PrincipalID)
}

func currentReviewState(aggregate ProgramAggregate) (*ProgramStateSnapshot, error) {
	if aggregate.CurrentState == nil || aggregate.CurrentState.ProjectionVersion < 1 {
		return nil, fmt.Errorf("%w: current Program status has not been projected yet", ErrInvalidState)
	}
	if aggregate.CurrentState.ProgramVersion != aggregate.Program.Version {
		return nil, ErrVersionConflict
	}
	return aggregate.CurrentState, nil
}
