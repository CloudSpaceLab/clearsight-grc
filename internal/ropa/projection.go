package ropa

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultSummaryBatchSize = 500

// MaxProjectionPages bounds one keyset traversal. At the default 500-row batch
// size this covers 500,000 activities, well above the 100,000-activity target
// while preventing a faulty lister from advancing forever.
const MaxProjectionPages = 1000

// MaxProjectionBatchSize bounds the page size requested by a summary
// maintainer. Larger caller-supplied values are clamped before the lister is
// called.
const MaxProjectionBatchSize = 1000

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
	pageCount := 0

	// ActivityLister owns snapshot stability across this traversal. The
	// PostgreSQL lister lands in a later task; it must hold one transaction or
	// repeatable-read snapshot for the full scope traversal. The maintainer
	// detects wrong-scope rows and unknown statuses, but cannot detect or
	// repair every mid-scan keyset drift; a stable lister snapshot is therefore
	// part of the trust boundary.
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if pageCount >= MaxProjectionPages {
			return fmt.Errorf("%w: projection exceeded %d pages", ErrInvalid, MaxProjectionPages)
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
		pageCount++
		if err := ctx.Err(); err != nil {
			return err
		}
		if page.HasMore && len(page.Rows) == 0 {
			return fmt.Errorf("%w: projection lister returned HasMore with an empty page", ErrInvalid)
		}

		for _, activity := range page.Rows {
			if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
				return fmt.Errorf("%w: activity %q belongs to tenant/entity %q/%q, requested %q/%q", ErrScopeMismatch, activity.ID, activity.TenantID, activity.LegalEntityID, scope.TenantID, scope.LegalEntityID)
			}

			counts.Total++
			switch activity.Status {
			case StatusNew:
				counts.New++
			case StatusOpen:
				counts.Open++
			case StatusClosed:
				counts.Closed++
			default:
				return fmt.Errorf("%w: unknown processing activity status %q", ErrInvalid, activity.Status)
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
			return fmt.Errorf("%w: projection lister returned an empty or unchanged cursor", ErrInvalid)
		}
		if _, alreadySeen := seenCursors[page.NextCursor]; alreadySeen {
			return fmt.Errorf("%w: projection lister returned a previously seen cursor", ErrInvalid)
		}
		if pageCount >= MaxProjectionPages {
			return fmt.Errorf("%w: projection exceeded %d pages", ErrInvalid, MaxProjectionPages)
		}
		seenCursors[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}

	if err := ctx.Err(); err != nil {
		return err
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
	if m == nil || m.BatchSize <= 0 {
		return defaultSummaryBatchSize
	}
	if m.BatchSize > MaxProjectionBatchSize {
		return MaxProjectionBatchSize
	}
	return m.BatchSize
}
