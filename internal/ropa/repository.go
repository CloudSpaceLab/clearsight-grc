package ropa

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrNotFound        = errors.New("processing activity not found")
	ErrVersionConflict = errors.New("processing activity version conflict")
	ErrDuplicate       = errors.New("processing activity code already exists in this legal entity")
	ErrInvalid         = errors.New("processing activity is not valid")
	ErrClosureBlocked  = errors.New("processing activity closure requirements are not met")
	ErrScopeMismatch   = errors.New("processing activity is outside the requested scope")
)

// ActivityScope is the exact tenant and legal-entity boundary for an
// activity read, history read, or material write. There is deliberately no
// tenant-only exact-read variant.
type ActivityScope struct {
	TenantID      string
	LegalEntityID string
}

const (
	// DefaultActivityEventPageSize is used when a caller does not request a
	// bounded history page explicitly.
	DefaultActivityEventPageSize = 100
	// MaxActivityEventPageSize keeps one history read bounded. Callers can
	// continue from the last returned aggregate version to reconstruct all
	// history without issuing an unbounded query.
	MaxActivityEventPageSize = 200
)

type Repository interface {
	CreateActivity(context.Context, ProcessingActivity, Event) (ProcessingActivity, error)
	GetActivity(context.Context, ActivityScope, string) (ProcessingActivity, error)
	ApplyActivityEvent(context.Context, ActivityScope, string, int64, Event) (int64, error)
	// ActivityEvents reads at most limit events whose aggregate version is
	// greater than afterVersion. The boolean reports that another page exists.
	ActivityEvents(context.Context, ActivityScope, string, int64, int) ([]Event, bool, error)
	ActivityByCode(context.Context, ActivityScope, string) (ProcessingActivity, error)
}

type ListActivitiesFilter struct {
	Status           Status
	LawfulBasis      string
	OwnerPrincipalID string
	Search           string
	IncludeRetired   bool
	Cursor           string
	Limit            int
}

type ActivityPage struct {
	Rows       []ProcessingActivity `json:"rows"`
	NextCursor string               `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

// ActivityLister returns bounded, keyset-paginated pages for one exact scope.
// For a given scope and cursor traversal, an implementation must return a
// stable, non-moving snapshot for the duration of that traversal. In
// particular, changes to status or next_review_date must not move a row across
// the cursor between calls, or keyset pagination would be ill-defined. A
// production implementation should provide that stability with one transaction
// or repeatable-read snapshot; callers may detect scope/status contract
// violations but cannot repair mid-scan drift.
type ActivityLister interface {
	ListActivities(context.Context, ActivityScope, ListActivitiesFilter) (ActivityPage, error)
}

type SummaryRepository interface {
	LatestSummary(context.Context, string, string) (RegisterSummary, error)
	ReplaceSummary(context.Context, RegisterSummary) error
}

func normalizeActivityScope(scope ActivityScope) (ActivityScope, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" {
		return ActivityScope{}, ErrInvalid
	}
	return scope, nil
}
