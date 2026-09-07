package evidence

import (
	"context"
	"sort"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func (r *MemoryRepository) ClaimArtifactScanJobs(_ context.Context, worker string, now time.Time, limit int) ([]ArtifactScanJob, error) {
	if worker == "" || limit <= 0 {
		return nil, ErrArtifactScannerUnavailable
	}
	if limit > 10 {
		limit = 10
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0)
	for id, job := range r.scanJobs {
		if (job.State == "PENDING" && !job.NextAttemptAt.After(now)) || (job.State == "RUNNING" && !job.LeaseUntil.After(now)) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := r.scanJobs[ids[i]], r.scanJobs[ids[j]]
		if a.NextAttemptAt.Equal(b.NextAttemptAt) {
			return ids[i] < ids[j]
		}
		return a.NextAttemptAt.Before(b.NextAttemptAt)
	})
	if len(ids) > limit {
		ids = ids[:limit]
	}
	result := make([]ArtifactScanJob, 0, len(ids))
	for _, id := range ids {
		job := r.scanJobs[id]
		if job.Attempt >= artifactScanMaxAttempts {
			job.State = "FAILED"
			job.FailureCode = "ATTEMPTS_EXHAUSTED"
			job.WorkerID = ""
			job.LeaseUntil = time.Time{}
			r.scanJobs[id] = job
			continue
		}
		job.Attempt++
		job.State = "RUNNING"
		job.WorkerID = worker
		job.LeaseUntil = now.Add(artifactScanLease)
		r.scanJobs[id] = job
		result = append(result, job)
	}
	return result, nil
}

func (r *MemoryRepository) ArtifactScanQueueHealth(_ context.Context) (workflowruntime.QueueHealth, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var health workflowruntime.QueueHealth
	for _, job := range r.scanJobs {
		if job.State == "COMPLETED" {
			continue
		}
		if job.Attempt > health.HighestAttempts {
			health.HighestAttempts = job.Attempt
		}
		if job.State == "FAILED" {
			health.Terminal++
			continue
		}
		health.Pending++
		due := job.NextAttemptAt
		if health.OldestPending == nil || due.Before(*health.OldestPending) {
			health.OldestPending = &due
		}
	}
	return health, nil
}

func (r *MemoryRepository) CompleteArtifactScan(_ context.Context, claim ArtifactScanJob, receipt ArtifactScanReceipt, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.scanJobs[claim.Artifact.ID]
	artifact := r.artifacts[claim.Artifact.ID]
	if !ok || current.State != "RUNNING" || current.WorkerID != claim.WorkerID || current.Attempt != claim.Attempt || !current.LeaseUntil.After(now) || current.Artifact.SHA256 != artifact.SHA256 || current.Artifact.SizeBytes != artifact.SizeBytes || artifact.Status != ArtifactStoredUnscanned || !validScanReceipt(current, receipt, now) {
		return ErrArtifactScanLease
	}
	updated, status := scanOutcome(current, receipt, now)
	artifact.Status = status
	r.scanJobs[artifact.ID] = updated
	r.artifacts[artifact.ID] = artifact
	r.scanReceipts = append(r.scanReceipts, receipt)
	return nil
}
