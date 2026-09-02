package oversight

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("oversight scope is required")
	ErrNotFound = errors.New("oversight snapshot is not available")
)

type Repository interface {
	Latest(context.Context, Scope) (Snapshot, error)
}

type Service struct {
	repository Repository
	Now        func() time.Time
	StaleAfter time.Duration
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, Now: time.Now, StaleAfter: 15 * time.Minute}
}

func (s *Service) Get(ctx context.Context, scope Scope) (Snapshot, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(scope.TenantID) == "" || strings.TrimSpace(scope.LegalEntityID) == "" {
		return Snapshot{}, ErrInvalid
	}
	value, err := s.repository.Latest(ctx, scope)
	if err != nil {
		return Snapshot{}, err
	}
	value.Freshness = FreshnessCurrent
	if value.GeneratedAt.IsZero() || s.Now().UTC().Sub(value.GeneratedAt) > s.StaleAfter || value.ProjectionVersion != ProjectionVersion {
		value.Freshness = FreshnessStale
	}
	return value, nil
}

func interventionCopy(item Intervention, now time.Time) (string, string) {
	switch {
	case item.DueAt != nil && item.DueAt.Before(now):
		return "The issue is overdue and remains open.", "Review the issue and confirm the current recovery plan"
	case item.OwnerID == "":
		return "The issue has no accountable owner.", "Assign an eligible owner"
	case item.Priority >= 4:
		return "The issue is high priority and remains open.", "Review the current facts and next action"
	default:
		return "The issue requires oversight attention.", "Open the issue"
	}
}
