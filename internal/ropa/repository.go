package ropa

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("processing activity not found")
	ErrVersionConflict = errors.New("processing activity version conflict")
	ErrDuplicate       = errors.New("processing activity code already exists in this legal entity")
	ErrInvalid         = errors.New("processing activity is not valid")
	ErrClosureBlocked  = errors.New("processing activity closure requirements are not met")
	ErrScopeMismatch   = errors.New("processing activity is outside the requested scope")
)

type Repository interface {
	CreateActivity(context.Context, ProcessingActivity, Event) (ProcessingActivity, error)
	GetActivity(context.Context, string, string) (ProcessingActivity, error)
	ApplyActivityEvent(context.Context, string, string, int64, Event) (int64, error)
	ActivityEvents(context.Context, string, string) ([]Event, error)
	ActivityByCode(context.Context, string, string, string) (ProcessingActivity, error)
}

type ListActivitiesFilter struct {
	TenantID         string
	LegalEntityID    string
	Status           Status
	LawfulBasis      string
	OwnerPrincipalID string
	Search           string
	IncludeRetired   bool
	Cursor           string
	Limit            int
}

type ActivityPage struct {
	Rows       []ProcessingActivity
	NextCursor string
	HasMore    bool
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
	ListActivities(context.Context, ListActivitiesFilter) (ActivityPage, error)
}

type SummaryRepository interface {
	LatestSummary(context.Context, string, string) (RegisterSummary, error)
	ReplaceSummary(context.Context, RegisterSummary) error
}
