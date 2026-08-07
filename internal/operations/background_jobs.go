package operations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxJobs = 200

type Job struct {
	ID          string     `json:"id"`
	Queue       string     `json:"queue"`
	Kind        string     `json:"kind"`
	State       string     `json:"state"`
	Attempts    int        `json:"attempts"`
	AvailableAt *time.Time `json:"available_at,omitempty"`
	LeaseUntil  *time.Time `json:"lease_until,omitempty"`
	LockedBy    string     `json:"locked_by,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	TerminalAt  *time.Time `json:"terminal_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type QueueSummary struct {
	Queue           string     `json:"queue"`
	Pending         int        `json:"pending"`
	Running         int        `json:"running"`
	Terminal        int        `json:"terminal"`
	HighestAttempts int        `json:"highest_attempts"`
	OldestPending   *time.Time `json:"oldest_pending,omitempty"`
}

type Snapshot struct {
	Queues []QueueSummary `json:"queues"`
	Jobs   []Job          `json:"jobs"`
}

type Source interface {
	ListBackgroundJobs(context.Context, string, int) ([]Job, error)
}

type Service struct {
	sources []Source
}

func NewService(sources ...Source) *Service {
	filtered := make([]Source, 0, len(sources))
	for _, source := range sources {
		if source != nil {
			filtered = append(filtered, source)
		}
	}
	return &Service{sources: filtered}
}

func (s *Service) Snapshot(ctx context.Context, tenant string, limit int) (Snapshot, error) {
	if strings.TrimSpace(tenant) == "" {
		return Snapshot{}, fmt.Errorf("tenant_id is required")
	}
	if limit <= 0 || limit > maxJobs {
		limit = 100
	}
	jobs := make([]Job, 0, limit)
	for _, source := range s.sources {
		values, err := source.ListBackgroundJobs(ctx, tenant, limit)
		if err != nil {
			return Snapshot{}, err
		}
		jobs = append(jobs, values...)
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobTime(jobs[i]).After(jobTime(jobs[j]))
	})
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return Snapshot{Queues: summarize(jobs), Jobs: jobs}, nil
}

func summarize(jobs []Job) []QueueSummary {
	byQueue := map[string]QueueSummary{}
	for _, job := range jobs {
		summary := byQueue[job.Queue]
		summary.Queue = job.Queue
		if job.Attempts > summary.HighestAttempts {
			summary.HighestAttempts = job.Attempts
		}
		switch strings.ToUpper(job.State) {
		case "FAILED", "DEAD_LETTERED":
			summary.Terminal++
		case "CLAIMED", "RUNNING":
			summary.Running++
			summary.Pending++
		default:
			if job.TerminalAt == nil && strings.ToUpper(job.State) != "COMPLETED" && strings.ToUpper(job.State) != "FIRED" && strings.ToUpper(job.State) != "PUBLISHED" {
				summary.Pending++
			}
		}
		if summary.Pending > 0 && job.TerminalAt == nil {
			candidate := job.AvailableAt
			if candidate == nil {
				candidate = job.CreatedAt
			}
			if candidate != nil && (summary.OldestPending == nil || candidate.Before(*summary.OldestPending)) {
				value := candidate.UTC()
				summary.OldestPending = &value
			}
		}
		byQueue[job.Queue] = summary
	}
	result := make([]QueueSummary, 0, len(byQueue))
	for _, summary := range byQueue {
		result = append(result, summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Queue < result[j].Queue })
	return result
}

func jobTime(job Job) time.Time {
	if job.UpdatedAt != nil {
		return *job.UpdatedAt
	}
	if job.CreatedAt != nil {
		return *job.CreatedAt
	}
	if job.AvailableAt != nil {
		return *job.AvailableAt
	}
	return time.Time{}
}
