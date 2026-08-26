package continuity

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound                  = errors.New("program or matter not found")
	ErrVersionConflict           = errors.New("program or matter version conflict")
	ErrMakerChecker              = errors.New("a different authorized reviewer must approve this change")
	ErrInvalidState              = errors.New("invalid lifecycle transition")
	ErrClosureBlocked            = errors.New("matter closure requirements are not met")
	ErrDuplicate                 = errors.New("duplicate program or triggered matter")
	ErrTriggerDuplicate          = errors.New("program trigger already processed")
	ErrLegalEntityAmbiguous      = errors.New("legal entity identifier is ambiguous")
	ErrEvidenceSourceInvalid     = errors.New("evidence source is not active in this legal entity")
	ErrEvidenceSourceUnavailable = errors.New("evidence source validation is unavailable")
)

type EvidenceSourceValidator interface {
	ValidateActiveSourcesForEntity(context.Context, string, string, []string) error
}

// LegalEntityResolver turns an active tenant-bound legal-entity ID or code
// into the canonical immutable ID used by aggregates and creation events.
type LegalEntityResolver interface {
	ResolveLegalEntity(context.Context, string, string) (string, error)
}

type Repository interface {
	CreateProgram(context.Context, Program, Event) (Program, error)
	ListPrograms(context.Context, string, int) ([]ProgramAggregate, error)
	GetProgram(context.Context, string, string) (ProgramAggregate, error)
	ApplyProgramEvent(context.Context, string, string, int64, Event) (int64, error)
	RecordProgramTrigger(context.Context, Trigger) (bool, error)
	ProgramEvents(context.Context, string, string, *time.Time) ([]Event, error)

	CreateMatter(context.Context, Matter, Event) (Matter, error)
	ListMatters(context.Context, string, string, int) ([]MatterAggregate, error)
	GetMatter(context.Context, string, string) (MatterAggregate, error)
	ApplyMatterEvent(context.Context, string, string, int64, Event) (int64, error)
	MatterByTriggerKey(context.Context, string, string) (Matter, error)
	MatterEvents(context.Context, string, string, *time.Time) ([]Event, error)
	ResponsePackageHistory(context.Context, string, string, string, int) ([]ResponseHistoryItem, bool, error)
	OpenMatterCount(context.Context, string, string) (int, error)
	LinkedProgramIDs(context.Context, string, string) ([]string, error)
}
