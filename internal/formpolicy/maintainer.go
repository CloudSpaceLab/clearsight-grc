package formpolicy

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	maintenanceBatchLimit = 100
	maintenanceLease      = time.Minute
)

type MaintenanceRepository interface {
	SeedCompensations(context.Context, time.Time, int) (int, error)
	ClaimCompensations(context.Context, string, time.Time, time.Duration, int) ([]CompensationCandidate, error)
	CompleteCompensation(context.Context, string, string, time.Time) error
	RetryCompensation(context.Context, string, string, time.Time, string) error
	SeedReconciliation(context.Context, time.Time, int) (int, error)
	ClaimReconciliation(context.Context, string, time.Time, time.Duration, int) ([]ScoredResponseEvent, error)
	CompleteReconciliation(context.Context, string, string, time.Time) error
	RetryReconciliation(context.Context, string, string, time.Time, string, bool) error
	MaintainOutcomeChecks(context.Context, string, time.Time, time.Duration, int) (int, error)
}

type PolicyExecutionHandler interface {
	Handle(context.Context, ScoredResponseEvent) ([]ExecutionReceipt, error)
	HandleCompensation(context.Context, CompensationCandidate) (CompensationReceipt, error)
}

type Maintainer struct {
	repository MaintenanceRepository
	executor   PolicyExecutionHandler
	workerID   string
}

func NewMaintainer(repository MaintenanceRepository, executor PolicyExecutionHandler, workerID string) *Maintainer {
	return &Maintainer{repository: repository, executor: executor, workerID: strings.TrimSpace(workerID)}
}

func (maintainer *Maintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if maintainer == nil || maintainer.repository == nil || maintainer.executor == nil || maintainer.workerID == "" || now.IsZero() {
		return 0, ErrInvalid
	}
	if limit < 1 || limit > maintenanceBatchLimit {
		limit = maintenanceBatchLimit
	}
	now = now.UTC()
	processed, outcomeErr := maintainer.repository.MaintainOutcomeChecks(ctx, maintainer.workerID, now, maintenanceLease, limit)
	_, compensationSeedErr := maintainer.repository.SeedCompensations(ctx, now, limit)
	candidates, compensationClaimErr := maintainer.repository.ClaimCompensations(ctx, maintainer.workerID, now, maintenanceLease, limit)
	var compensationErrs []error
	if compensationSeedErr == nil && compensationClaimErr == nil {
		for _, candidate := range candidates {
			if ctx.Err() != nil {
				return processed, errors.Join(outcomeErr, ctx.Err())
			}
			_, compensationErr := maintainer.executor.HandleCompensation(ctx, candidate)
			if compensationErr == nil {
				compensationErr = maintainer.repository.CompleteCompensation(ctx, candidate.JobID, maintainer.workerID, now)
			} else {
				compensationErr = errors.Join(compensationErr, maintainer.repository.RetryCompensation(ctx, candidate.JobID, maintainer.workerID, now, compensationErr.Error()))
			}
			if compensationErr != nil {
				compensationErrs = append(compensationErrs, compensationErr)
			}
			processed++
		}
	}
	_, seedErr := maintainer.repository.SeedReconciliation(ctx, now, limit)
	if seedErr != nil {
		return processed, errors.Join(append(compensationErrs, outcomeErr, compensationSeedErr, compensationClaimErr, seedErr)...)
	}
	events, claimErr := maintainer.repository.ClaimReconciliation(ctx, maintainer.workerID, now, maintenanceLease, limit)
	if claimErr != nil {
		return processed, errors.Join(append(compensationErrs, outcomeErr, compensationSeedErr, compensationClaimErr, claimErr)...)
	}
	failures := compensationErrs
	if outcomeErr != nil {
		failures = append(failures, outcomeErr)
	}
	if compensationSeedErr != nil {
		failures = append(failures, compensationSeedErr)
	}
	if compensationClaimErr != nil {
		failures = append(failures, compensationClaimErr)
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return processed, errors.Join(append(failures, ctx.Err())...)
		}
		_, executionErr := maintainer.executor.Handle(ctx, event)
		if executionErr == nil {
			executionErr = maintainer.repository.CompleteReconciliation(ctx, event.ID, maintainer.workerID, now)
		} else {
			keepRetrying := errors.Is(executionErr, ErrActivationAuthority) || errors.Is(executionErr, ErrAuthorityUnavailable)
			retryErr := maintainer.repository.RetryReconciliation(ctx, event.ID, maintainer.workerID, now, executionErr.Error(), keepRetrying)
			executionErr = errors.Join(executionErr, retryErr)
		}
		if executionErr != nil {
			failures = append(failures, executionErr)
		}
		processed++
	}
	return processed, errors.Join(failures...)
}
