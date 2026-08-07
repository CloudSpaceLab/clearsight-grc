package continuity

import (
	"context"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func (r *MemoryRepository) BackgroundJobs(_ context.Context, tenant string, limit int) (operations.Snapshot, error) {
	data := projectionData(r)
	data.mu.Lock()
	defer data.mu.Unlock()

	summary := operations.QueueSummary{Queue: "program-projection"}
	jobs := []operations.Job{}
	for _, job := range data.jobs {
		if job.TenantID != tenant || job.Status == ProjectionJobCompleted {
			continue
		}
		item := operations.Job{ID: job.ID, Queue: "program-projection", Kind: job.Reason, State: string(job.Status), Attempts: job.Attempts, AvailableAt: &job.AvailableAt, LockedBy: job.ClaimedBy, LastError: job.LastError, CreatedAt: &job.CreatedAt, UpdatedAt: &job.UpdatedAt}
		if job.Status == ProjectionJobFailed {
			item.TerminalAt = &job.UpdatedAt
		}
		jobs = append(jobs, item)
		if job.Attempts > summary.HighestAttempts {
			summary.HighestAttempts = job.Attempts
		}
		switch job.Status {
		case ProjectionJobReady:
			summary.Pending++
			projectionOldest(&summary, job.AvailableAt)
		case ProjectionJobClaimed:
			summary.Pending++
			summary.Running++
			projectionOldest(&summary, job.AvailableAt)
		case ProjectionJobFailed:
			summary.Terminal++
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool { return jobs[i].UpdatedAt.After(*jobs[j].UpdatedAt) })
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return operations.Snapshot{Queues: []operations.QueueSummary{summary}, Jobs: jobs}, nil
}

func projectionOldest(summary *operations.QueueSummary, value time.Time) {
	if summary.OldestPending == nil || value.Before(*summary.OldestPending) {
		copy := value.UTC()
		summary.OldestPending = &copy
	}
}
