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

type BatchRepository interface {
	LatestMany(context.Context, string, []string) ([]Snapshot, error)
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

func (s *Service) GetMany(ctx context.Context, tenantID string, legalEntityIDs []string) ([]Snapshot, error) {
	tenantID = strings.TrimSpace(tenantID)
	if s == nil || s.repository == nil || tenantID == "" || len(legalEntityIDs) == 0 || len(legalEntityIDs) > 256 {
		return nil, ErrInvalid
	}
	normalized := make([]string, 0, len(legalEntityIDs))
	seen := make(map[string]struct{}, len(legalEntityIDs))
	for _, id := range legalEntityIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, ErrInvalid
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	var values []Snapshot
	if repository, ok := s.repository.(BatchRepository); ok {
		batch, err := repository.LatestMany(ctx, tenantID, normalized)
		if err != nil {
			return nil, err
		}
		values = batch
	} else {
		values = make([]Snapshot, 0, len(normalized))
		for _, id := range normalized {
			value, err := s.repository.Latest(ctx, Scope{TenantID: tenantID, LegalEntityID: id})
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
	}
	for index := range values {
		values[index] = s.decorate(values[index])
	}
	return values, nil
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
