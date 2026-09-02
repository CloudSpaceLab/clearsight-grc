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
	LastError   string     `json:"-"`
	FailureCode string     `json:"failure_code,omitempty"`
	TerminalAt  *time.Time `json:"terminal_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func SafeFailureCode(message string) string {
	value := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(value, "invalid input syntax for type uuid"):
		return "INVALID_TENANT_IDENTIFIER"
	case strings.Contains(value, "communication is unavailable"):
		return "COMMUNICATION_CONFIGURATION_UNAVAILABLE"
	case strings.Contains(value, "timeout") || strings.Contains(value, "deadline exceeded"):
		return "DEPENDENCY_TIMEOUT"
	case value == "":
		return "FAILURE_DETAIL_UNAVAILABLE"
	default:
		return "PROCESSING_FAILED"
	}
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
	BackgroundJobs(context.Context, string, int) (Snapshot, error)
}

type Service struct{ sources []Source }

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
	result := Snapshot{}
	for _, source := range s.sources {
		part, err := source.BackgroundJobs(ctx, tenant, limit)
		if err != nil {
			return Snapshot{}, err
		}
		result.Queues = append(result.Queues, part.Queues...)
		result.Jobs = append(result.Jobs, part.Jobs...)
	}
	sort.Slice(result.Queues, func(i, j int) bool { return result.Queues[i].Queue < result.Queues[j].Queue })
	sort.SliceStable(result.Jobs, func(i, j int) bool { return jobTime(result.Jobs[i]).After(jobTime(result.Jobs[j])) })
	if len(result.Jobs) > limit {
		result.Jobs = result.Jobs[:limit]
	}
	return result, nil
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
