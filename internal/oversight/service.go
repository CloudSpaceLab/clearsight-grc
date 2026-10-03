package oversight

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid                    = errors.New("oversight scope is required")
	ErrNotFound                   = errors.New("oversight snapshot is not available")
	ErrInvalidReportingPeriod     = errors.New("reporting period is invalid")
	ErrHistoricalEndUnsupported   = errors.New("historical reporting period end is unsupported")
	ErrReportingPeriodUnavailable = errors.New("custom reporting period is unavailable")
)

type Repository interface {
	Latest(context.Context, Scope) (Snapshot, error)
}

type PeriodRepository interface {
	BuildPeriod(context.Context, Scope, time.Time, time.Time) (Snapshot, error)
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
	return s.GetForPeriod(ctx, scope, PeriodRequest{})
}

func (s *Service) GetForPeriod(ctx context.Context, scope Scope, request PeriodRequest) (Snapshot, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(scope.TenantID) == "" || strings.TrimSpace(scope.LegalEntityID) == "" {
		return Snapshot{}, ErrInvalid
	}
	request.StartDate = strings.TrimSpace(request.StartDate)
	request.EndDate = strings.TrimSpace(request.EndDate)
	if request.StartDate == "" && request.EndDate == "" {
		if strings.TrimSpace(scope.OrganizationScopeID) == "" {
			value, err := s.repository.Latest(ctx, scope)
			if err != nil {
				return Snapshot{}, err
			}
			return s.decorate(value), nil
		}
		repository, ok := s.repository.(PeriodRepository)
		if !ok {
			return Snapshot{}, ErrReportingPeriodUnavailable
		}
		now := s.Now().UTC()
		value, err := repository.BuildPeriod(ctx, scope, now.Add(-90*24*time.Hour), now)
		if err != nil {
			return Snapshot{}, err
		}
		return s.decorate(value), nil
	}
	if request.StartDate == "" {
		return Snapshot{}, ErrInvalidReportingPeriod
	}
	now := s.Now().UTC()
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endDate := request.EndDate
	if endDate == "" {
		endDate = currentDate.Format(ReportingDateLayout)
	}
	parsedEnd, err := time.Parse(ReportingDateLayout, endDate)
	if err != nil {
		return Snapshot{}, ErrInvalidReportingPeriod
	}
	parsedEnd = parsedEnd.UTC()
	if !parsedEnd.Equal(currentDate) {
		return Snapshot{}, ErrHistoricalEndUnsupported
	}
	start, err := time.Parse(ReportingDateLayout, request.StartDate)
	if err != nil {
		return Snapshot{}, ErrInvalidReportingPeriod
	}
	start = start.UTC()
	if start.After(currentDate) || currentDate.Sub(start) >= ReportingPeriodMaxDays*24*time.Hour {
		return Snapshot{}, ErrInvalidReportingPeriod
	}
	repository, ok := s.repository.(PeriodRepository)
	if !ok {
		return Snapshot{}, ErrReportingPeriodUnavailable
	}
	value, err := repository.BuildPeriod(ctx, scope, start, now)
	if err != nil {
		return Snapshot{}, err
	}
	return s.decorate(value), nil
}

func (s *Service) decorate(value Snapshot) Snapshot {
	value.Freshness = FreshnessCurrent
	now := s.Now().UTC()
	if value.GeneratedAt.IsZero() || now.Sub(value.GeneratedAt) > s.StaleAfter || value.ProjectionVersion != ProjectionVersion {
		value.Freshness = FreshnessStale
	}
	if value.PostureAsOf.IsZero() {
		value.PostureAsOf = value.GeneratedAt
	}
	value.ReportingPeriod = ReportingPeriod{
		StartDate:              value.PeriodStart.UTC().Format(ReportingDateLayout),
		EndDate:                value.PeriodEnd.UTC().Format(ReportingDateLayout),
		Mode:                   ReportingPeriodCurrentWindow,
		MaxDays:                ReportingPeriodMaxDays,
		HistoricalEndSupported: false,
	}
	return value
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
