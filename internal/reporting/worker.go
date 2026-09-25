package reporting

import (
	"context"
	"errors"
	"strings"
	"time"
)

// QueuedRunRepository is the bounded discovery surface used by the report
// worker. It lists only scopes that currently have queued durable work; the
// worker never performs a tenant-wide report-data read.
type QueuedRunRepository interface {
	RunRepository
	ListQueuedRunScopes(context.Context, int) ([]ReportScope, error)
}

// RunMaintainer adapts asynchronous report execution to the existing polling
// worker runtime. The run row owns attempts; this class deliberately has no
// runtime attempt budget of its own.
type RunMaintainer struct {
	repository QueuedRunRepository
	service    *Service
}

func NewRunMaintainer(repository QueuedRunRepository, service *Service) *RunMaintainer {
	return &RunMaintainer{repository: repository, service: service}
}

func (m *RunMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.repository == nil || m.service == nil || ctx == nil || strings.TrimSpace(m.service.WorkerID) == "" {
		return 0, ErrInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 5
	}
	now = now.UTC()
	previousNow := m.service.Now
	m.service.Now = func() time.Time { return now }
	defer func() { m.service.Now = previousNow }()
	scopes, err := m.repository.ListQueuedRunScopes(ctx, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, scope := range scopes {
		if processed >= limit {
			break
		}
		runs, err := m.repository.ListRuns(ctx, scope, "", limit-processed)
		if err != nil {
			return processed, err
		}
		for _, run := range runs {
			if processed >= limit {
				break
			}
			if run.Status != RunQueued {
				continue
			}
			executedRun, executeErr := m.service.ExecuteRun(ctx, run)
			processed++
			if executeErr == nil {
				continue
			}
			// A bounded stop or exhausted retry budget is a committed terminal
			// receipt, not a worker-loop failure. Other errors remain observable.
			terminalRunID := run.ID
			if executedRun.ID != "" {
				terminalRunID = executedRun.ID
			}
			current, readErr := m.repository.GetRun(ctx, scope, terminalRunID)
			if readErr == nil && (current.Status == RunReady || current.Status == RunFailed) {
				if current.Status == RunFailed && current.FailureCode == "" {
					return processed, executeErr
				}
				continue
			}
			if errors.Is(executeErr, ErrConflict) && readErr != nil {
				return processed, executeErr
			}
			return processed, executeErr
		}
	}
	return processed, nil
}

var _ interface {
	Maintain(context.Context, time.Time, int) (int, error)
} = (*RunMaintainer)(nil)
