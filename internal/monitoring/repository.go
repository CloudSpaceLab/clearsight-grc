package monitoring

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("monitoring record not found")
	ErrConflict = errors.New("monitoring record version conflict")
	ErrInvalid  = errors.New("monitoring record is invalid")
)

type LifecycleTransition struct {
	TenantID        string
	LegalEntityID   string
	ProgramID       string
	ID              string
	ExpectedVersion int64
	To              LifecycleStatus
	ActorID         string
	At              time.Time
}

type Repository interface {
	CreateFormRevision(context.Context, FormTemplate) (FormTemplate, error)
	FormRevision(context.Context, string, string, string, string, int64) (FormTemplate, error)
	ListFormRevisions(context.Context, string, string, string, int) ([]FormTemplate, error)
	TransitionForm(context.Context, LifecycleTransition) (FormTemplate, error)
	CreateCheckRevision(context.Context, MonitoringCheck) (MonitoringCheck, error)
	CheckRevision(context.Context, string, string, int64) (MonitoringCheck, error)
	LatestCheckRevision(context.Context, string, string) (MonitoringCheck, error)
	ListCheckRevisions(context.Context, string, string, int) ([]MonitoringCheck, error)
	TransitionCheck(context.Context, LifecycleTransition) (MonitoringCheck, error)
	AppendResult(context.Context, MonitoringResult) (MonitoringResult, error)
	Result(context.Context, string, string) (MonitoringResult, error)
	ListResults(context.Context, string, string, int) ([]MonitoringResult, error)
}

// ReusableFormRepository exposes exact, legal-entity-scoped form revisions to
// workflows that are not owned by a single Program, such as vendor due
// diligence and record-linked vendor requests.
type ReusableFormRepository interface {
	ReusableFormRevision(context.Context, string, string, string, int64) (FormTemplate, error)
	ListReusableFormRevisions(context.Context, string, string, int) ([]FormTemplate, error)
}
