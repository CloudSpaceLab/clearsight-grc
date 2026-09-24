package reporting

import "context"

// ReportScope is mandatory on every exact read and write. There is no
// tenant-only path: a report belongs to one legal entity.
type ReportScope struct {
	TenantID      string
	LegalEntityID string
}

type DefinitionRepository interface {
	CreateDefinition(ctx context.Context, scope ReportScope, definition ReportDefinition, revision ReportDefinitionRevision) (ReportDefinition, error)
	GetDefinition(ctx context.Context, scope ReportScope, id string) (ReportDefinition, error)
	GetDefinitionByCode(ctx context.Context, scope ReportScope, code string) (ReportDefinition, error)
	ListDefinitions(ctx context.Context, scope ReportScope, includeRetired bool) ([]ReportDefinition, error)
	ListDefinitionHistory(ctx context.Context, scope ReportScope, id string) ([]ReportDefinitionRevision, error)
	// TransitionDefinition must lock the scoped current row, check the expected
	// command version, call ValidateTransitionForWrite again, and persist the
	// current decision, report-owned revision decision, event, and outbox in one
	// transaction.
	TransitionDefinition(ctx context.Context, scope ReportScope, id string, expectedVersion int64, next DefinitionStatus, decision DecisionRecord) (ReportDefinition, error)
}

type RunRepository interface {
	CreateRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error)
	GetRun(ctx context.Context, scope ReportScope, id string) (ReportRun, error)
	ListRuns(ctx context.Context, scope ReportScope, definitionID string, limit int) ([]ReportRun, error)
	ClaimQueuedRuns(ctx context.Context, scope ReportScope, workerID string, limit int) ([]ReportRun, error)
	CompleteRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error)
	FailRun(ctx context.Context, scope ReportScope, id, failureCode string) (ReportRun, error)
	RecordRunDownload(ctx context.Context, scope ReportScope, id, downloadedBy string) error
	// CaptureSourceBoundary freezes the reconstruction facts before the queued
	// row is written. ListReportRows reads only that exact run snapshot and
	// returns at most one bounded keyset page.
	CaptureSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error)
	ListReportRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error)
}

// Repository is the complete state boundary used by the report service.
type Repository interface {
	DefinitionRepository
	RunRepository
}
