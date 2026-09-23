package ropa

import (
	"context"
	"strings"
	"time"
)

const defaultSummaryBatchSize = 500

// Scope identifies one exact tenant and legal-entity register projection.
type Scope struct {
	TenantID      string
	LegalEntityID string
}

// SummaryMaintainer rebuilds one register summary from bounded, scoped pages.
// The repository and service lister remain separate so a production composition
// can use a command repository and a separate read lister.
type SummaryMaintainer struct {
	repository Repository
	summaries  SummaryRepository
	service    *Service
	BatchSize  int
}

func NewSummaryMaintainer(repository Repository, summaries SummaryRepository, service *Service) *SummaryMaintainer {
	return &SummaryMaintainer{
		repository: repository,
		summaries:  summaries,
		service:    service,
		BatchSize:  defaultSummaryBatchSize,
	}
}

func (m *SummaryMaintainer) Maintain(ctx context.Context, scope Scope) error {
	if m == nil || m.repository == nil || m.summaries == nil || m.service == nil || ctx == nil {
		return ErrInvalid
	}

	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" {
		return ErrInvalid
	}

	lister, err := m.activityLister()
	if err != nil {
		return err
	}

	now := m.service.now()
	counts := RegisterCounts{}
	highWater := time.Time{}
	cursor := ""
	batchSize := m.batchSize()
	seenCursors := make(map[string]struct{})

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := lister.ListActivities(ctx, ListActivitiesFilter{
			TenantID:       scope.TenantID,
			LegalEntityID:  scope.LegalEntityID,
			IncludeRetired: true,
			Cursor:         cursor,
			Limit:          batchSize,
		})
		if err != nil {
			return err
		}

		for _, activity := range page.Rows {
			counts.Total++
			switch activity.Status {
			case StatusNew:
				counts.New++
			case StatusOpen:
				counts.Open++
			case StatusClosed:
				counts.Closed++
			}

			aggregate := Aggregate{ProcessingActivity: activity}
			if aggregate.IsRetired() {
				counts.Retired++
			}
			if strings.TrimSpace(activity.LawfulBasis) == "" {
				counts.MissingLawfulBasis++
			}
			if strings.TrimSpace(activity.OwnerPrincipalID) == "" {
				counts.MissingOwner++
			}
			if strings.TrimSpace(activity.DataSubjectCategories) == "" {
				counts.NoDataSubjects++
			}
			if aggregate.ReviewOverdue(now) {
				counts.ReviewOverdue++
			}
			if activity.UpdatedAt.After(highWater) {
				highWater = activity.UpdatedAt.UTC()
			}
		}

		if !page.HasMore {
			break
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return ErrInvalid
		}
		if _, alreadySeen := seenCursors[page.NextCursor]; alreadySeen {
			return ErrInvalid
		}
		seenCursors[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}

	excluded := 0
	unknown := 0
	return m.summaries.ReplaceSummary(ctx, RegisterSummary{
		TenantID:          scope.TenantID,
		LegalEntityID:     scope.LegalEntityID,
		GeneratedAt:       now,
		ProjectionVersion: ProjectionVersion,
		SourceHighWater:   highWater,
		Coverage: Coverage{
			Population: counts.Total,
			Excluded:   &excluded,
			Unknown:    &unknown,
		},
		Counts: counts,
	})
}

// MaintainAll refreshes scopes in the supplied order and stops at the first
// failure so a worker can retry the remaining scopes later.
func (m *SummaryMaintainer) MaintainAll(ctx context.Context, scopes []Scope) error {
	for _, scope := range scopes {
		if err := m.Maintain(ctx, scope); err != nil {
			return err
		}
	}
	return nil
}

func (m *SummaryMaintainer) activityLister() (ActivityLister, error) {
	if m.service != nil && m.service.lister != nil {
		return m.service.lister, nil
	}
	if lister, ok := m.repository.(ActivityLister); ok {
		return lister, nil
	}
	return nil, ErrInvalid
}

func (m *SummaryMaintainer) batchSize() int {
	if m != nil && m.BatchSize > 0 {
		return m.BatchSize
	}
	return defaultSummaryBatchSize
}
