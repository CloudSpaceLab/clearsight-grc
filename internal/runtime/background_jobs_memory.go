package runtime

import (
	"context"
	"sort"

	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

func (r *MemoryRepository) BackgroundJobs(_ context.Context, tenant string, limit int) (operations.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	timerSummary := operations.QueueSummary{Queue: "workflow-timers"}
	outboxSummary := operations.QueueSummary{Queue: "outbox-delivery"}
	jobs := make([]operations.Job, 0, len(r.timers)+len(r.outbox))
	for _, timer := range r.timers {
		if timer.TenantID != tenant {
			continue
		}
		job := operations.Job{ID: timer.ID, Queue: "workflow-timers", Kind: timer.Type, State: string(timer.State), Attempts: timer.Attempts, AvailableAt: timePtr(timer.DueAt), LeaseUntil: timer.LeaseUntil, LockedBy: timer.LockedBy, LastError: timer.LastError, TerminalAt: timer.FailedAt}
		jobs = append(jobs, job)
		if timer.Attempts > timerSummary.HighestAttempts {
			timerSummary.HighestAttempts = timer.Attempts
		}
		switch timer.State {
		case TimerReady:
			timerSummary.Pending++
			oldest(&timerSummary, timer.DueAt)
		case TimerClaimed:
			timerSummary.Pending++
			timerSummary.Running++
			oldest(&timerSummary, timer.DueAt)
		case TimerFailed:
			timerSummary.Terminal++
		}
	}
	for _, event := range r.outbox {
		if event.TenantID != tenant {
			continue
		}
		state := "READY"
		terminal := event.DeadLetteredAt
		if terminal != nil {
			state = "DEAD_LETTERED"
		} else if event.LockedBy != "" {
			state = "CLAIMED"
		}
		available := event.NextAttemptAt
		if available == nil {
			available = timePtr(event.OccurredAt)
		}
		jobs = append(jobs, operations.Job{ID: event.ID, Queue: "outbox-delivery", Kind: event.EventType, State: state, Attempts: event.Attempts, AvailableAt: available, LeaseUntil: event.LeaseUntil, LockedBy: event.LockedBy, LastError: event.LastError, TerminalAt: terminal, CreatedAt: timePtr(event.OccurredAt)})
		if event.Attempts > outboxSummary.HighestAttempts {
			outboxSummary.HighestAttempts = event.Attempts
		}
		if terminal != nil {
			outboxSummary.Terminal++
		} else {
			outboxSummary.Pending++
			if state == "CLAIMED" {
				outboxSummary.Running++
			}
			oldest(&outboxSummary, *available)
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		left, right := jobs[i].AvailableAt, jobs[j].AvailableAt
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.After(*right)
	})
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return operations.Snapshot{Queues: []operations.QueueSummary{timerSummary, outboxSummary}, Jobs: jobs}, nil
}

func oldest(summary *operations.QueueSummary, candidate time.Time) {
	if summary.OldestPending == nil || candidate.Before(*summary.OldestPending) {
		value := candidate.UTC()
		summary.OldestPending = &value
	}
}
