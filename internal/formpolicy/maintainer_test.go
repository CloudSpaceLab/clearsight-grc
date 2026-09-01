package formpolicy

import (
	"context"
	"errors"
	"testing"
	"time"
)

type maintenanceRepositoryStub struct {
	events                []ScoredResponseEvent
	outcomes              int
	seeded                int
	completed             []string
	retried               []string
	claimWorker           string
	claimLease            time.Duration
	claimLimit            int
	processingErr         error
	compensations         []CompensationCandidate
	compensationSeeded    int
	compensationCompleted []string
	compensationRetried   []string
	retryContinuously     bool
}

func (stub *maintenanceRepositoryStub) SeedReconciliation(_ context.Context, _ time.Time, limit int) (int, error) {
	stub.seeded = limit
	return 0, nil
}

func (stub *maintenanceRepositoryStub) ClaimReconciliation(_ context.Context, worker string, _ time.Time, lease time.Duration, limit int) ([]ScoredResponseEvent, error) {
	stub.claimWorker, stub.claimLease, stub.claimLimit = worker, lease, limit
	return stub.events, nil
}

func (stub *maintenanceRepositoryStub) CompleteReconciliation(_ context.Context, eventID, _ string, _ time.Time) error {
	stub.completed = append(stub.completed, eventID)
	return nil
}

func (stub *maintenanceRepositoryStub) RetryReconciliation(_ context.Context, eventID, _ string, _ time.Time, _ string, continuously bool) error {
	stub.retried = append(stub.retried, eventID)
	stub.retryContinuously = continuously
	return nil
}

func (stub *maintenanceRepositoryStub) MaintainOutcomeChecks(_ context.Context, _ string, _ time.Time, _ time.Duration, limit int) (int, error) {
	if stub.processingErr != nil {
		return 0, stub.processingErr
	}
	return stub.outcomes, nil
}

func (stub *maintenanceRepositoryStub) SeedCompensations(_ context.Context, _ time.Time, limit int) (int, error) {
	stub.compensationSeeded = limit
	return 0, nil
}

func (stub *maintenanceRepositoryStub) ClaimCompensations(_ context.Context, _ string, _ time.Time, _ time.Duration, _ int) ([]CompensationCandidate, error) {
	return stub.compensations, nil
}

func (stub *maintenanceRepositoryStub) CompleteCompensation(_ context.Context, jobID, _ string, _ time.Time) error {
	stub.compensationCompleted = append(stub.compensationCompleted, jobID)
	return nil
}

func (stub *maintenanceRepositoryStub) RetryCompensation(_ context.Context, jobID, _ string, _ time.Time, _ string) error {
	stub.compensationRetried = append(stub.compensationRetried, jobID)
	return nil
}

type maintenanceExecutorStub struct {
	events      []ScoredResponseEvent
	err         error
	compensated []CompensationCandidate
}

func (stub *maintenanceExecutorStub) Handle(_ context.Context, event ScoredResponseEvent) ([]ExecutionReceipt, error) {
	stub.events = append(stub.events, event)
	return nil, stub.err
}

func (stub *maintenanceExecutorStub) HandleCompensation(_ context.Context, candidate CompensationCandidate) (CompensationReceipt, error) {
	stub.compensated = append(stub.compensated, candidate)
	return CompensationReceipt{}, stub.err
}

func TestMaintainerReconcilesBoundedLeasedWorkAndRunsOutcomeChecks(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo := &maintenanceRepositoryStub{outcomes: 2, compensations: []CompensationCandidate{{JobID: "comp-job-a", RollbackPolicy: Policy{ID: "rollback-a"}}}, events: []ScoredResponseEvent{{ID: "job-a", TenantID: "bank", ResponseRevisionID: "response-a", OccurredAt: now}}}
	executor := &maintenanceExecutorStub{}
	maintainer := NewMaintainer(repo, executor, "worker-a")
	processed, err := maintainer.Maintain(t.Context(), now, 100)
	if err != nil || processed != 4 || repo.seeded != 100 || repo.compensationSeeded != 100 || repo.claimWorker != "worker-a" || repo.claimLease != time.Minute || repo.claimLimit != 100 || len(executor.events) != 1 || len(executor.compensated) != 1 || len(repo.compensationCompleted) != 1 || repo.compensationCompleted[0] != "comp-job-a" || len(repo.completed) != 1 || repo.completed[0] != "job-a" {
		t.Fatalf("processed=%d err=%v repo=%#v events=%#v", processed, err, repo, executor.events)
	}
}

func TestMaintainerReleasesFailedCompensationForRetry(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo := &maintenanceRepositoryStub{compensations: []CompensationCandidate{{JobID: "comp-job-a", RollbackPolicy: Policy{ID: "rollback-a"}}}}
	executor := &maintenanceExecutorStub{err: ErrAuthorityUnavailable}
	processed, err := NewMaintainer(repo, executor, "worker-a").Maintain(t.Context(), now, 100)
	if processed != 1 || !errors.Is(err, ErrAuthorityUnavailable) || len(repo.compensationCompleted) != 0 || len(repo.compensationRetried) != 1 || repo.compensationRetried[0] != "comp-job-a" {
		t.Fatalf("processed=%d err=%v completed=%v retried=%v", processed, err, repo.compensationCompleted, repo.compensationRetried)
	}
}

func TestMaintainerReleasesFailedReconciliationForRetry(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo := &maintenanceRepositoryStub{events: []ScoredResponseEvent{{ID: "job-a", TenantID: "bank", ResponseRevisionID: "response-a", OccurredAt: now}}}
	executor := &maintenanceExecutorStub{err: ErrAuthorityUnavailable}
	maintainer := NewMaintainer(repo, executor, "worker-a")
	processed, err := maintainer.Maintain(t.Context(), now, 100)
	if processed != 1 || !errors.Is(err, ErrAuthorityUnavailable) || len(repo.completed) != 0 || len(repo.retried) != 1 || repo.retried[0] != "job-a" || !repo.retryContinuously {
		t.Fatalf("processed=%d err=%v completed=%v retried=%v", processed, err, repo.completed, repo.retried)
	}
}
