package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type DefinitionStatus string

const (
	DefinitionDraft         DefinitionStatus = "DRAFT"
	DefinitionPendingReview DefinitionStatus = "PENDING_REVIEW"
	DefinitionReviewed      DefinitionStatus = "REVIEWED"
	DefinitionActive        DefinitionStatus = "ACTIVE"
	DefinitionRetired       DefinitionStatus = "RETIRED"
)

var definitionStatuses = map[DefinitionStatus]struct{}{
	DefinitionDraft:         {},
	DefinitionPendingReview: {},
	DefinitionReviewed:      {},
	DefinitionActive:        {},
	DefinitionRetired:       {},
}

func validDefinitionStatus(status DefinitionStatus) bool {
	_, ok := definitionStatuses[status]
	return ok
}

type RunStatus string

const (
	RunQueued  RunStatus = "QUEUED"
	RunRunning RunStatus = "RUNNING"
	RunReady   RunStatus = "READY"
	RunFailed  RunStatus = "FAILED"
)

type ReportDataset string

const (
	DatasetProcessingActivities         ReportDataset = "PROCESSING_ACTIVITIES"
	DatasetProcessingActivityExceptions ReportDataset = "PROCESSING_ACTIVITY_EXCEPTIONS"
	DatasetPrograms                     ReportDataset = "PROGRAMS"
	DatasetMatterExceptions             ReportDataset = "MATTER_EXCEPTIONS"
)

type ReportScopeKind string

const (
	ScopeLegalEntity ReportScopeKind = "LEGAL_ENTITY"
	ScopeProgram     ReportScopeKind = "PROGRAM"
	ScopeMatter      ReportScopeKind = "MATTER"
)

type ReportFormat string

const (
	FormatCSV    ReportFormat = "CSV"
	FormatNDJSON ReportFormat = "NDJSON"
)

// ReportDefinition is the current governed row for a report definition. The
// immutable revision row carries the same governed content for reconstruction.
type ReportDefinition struct {
	ID             string                  `json:"id"`
	TenantID       string                  `json:"tenant_id"`
	LegalEntityID  string                  `json:"legal_entity_id"`
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Dataset        ReportDataset           `json:"dataset"`
	ScopeKind      ReportScopeKind         `json:"scope_kind"`
	ScopeRef       string                  `json:"scope_ref,omitempty"`
	Format         ReportFormat            `json:"format"`
	Filter         *ReportFilterExpression `json:"filter,omitempty"`
	Status         DefinitionStatus        `json:"status"`
	CurrentVersion int                     `json:"current_version"`
	// StoredChecksum is the persisted digest; Checksum() computes the
	// governed content digest from the current definition.
	StoredChecksum string     `json:"checksum"`
	MakerID        string     `json:"maker_id"`
	CheckerID      string     `json:"checker_id,omitempty"`
	ReviewerID     string     `json:"reviewer_id,omitempty"`
	ReviewerNote   string     `json:"reviewer_note,omitempty"`
	EffectiveFrom  *time.Time `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	RetiredAt      *time.Time `json:"retired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Version        int64      `json:"version"`
}

// ReportDefinitionRevision is an immutable proposal or decision snapshot.
type ReportDefinitionRevision struct {
	DefinitionID  string                  `json:"definition_id"`
	TenantID      string                  `json:"tenant_id"`
	LegalEntityID string                  `json:"legal_entity_id"`
	Version       int                     `json:"version"`
	BaseVersion   int                     `json:"base_version"`
	Dataset       ReportDataset           `json:"dataset"`
	ScopeKind     ReportScopeKind         `json:"scope_kind"`
	ScopeRef      string                  `json:"scope_ref,omitempty"`
	Format        ReportFormat            `json:"format"`
	Filter        *ReportFilterExpression `json:"filter"`
	Checksum      string                  `json:"checksum"`
	MakerID       string                  `json:"maker_id"`
	CreatedAt     time.Time               `json:"created_at"`
	ReviewedBy    string                  `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time              `json:"reviewed_at,omitempty"`
	ApprovedBy    string                  `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time              `json:"approved_at,omitempty"`
	Decision      string                  `json:"decision"`
	DecisionNote  string                  `json:"decision_note"`
}

// DecisionRecord records the actor's decision at the checksum the actor saw.
// Lifecycle timestamps and actors belong to the revision ledger; this record is
// the command's audit input.
type DecisionRecord struct {
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	Note         string    `json:"note,omitempty"`
	ChecksumSeen string    `json:"checksum_seen"`
	Timestamp    time.Time `json:"timestamp"`
}

// ReportRun is the immutable receipt for one bounded report execution.
type ReportRun struct {
	ID                 string                  `json:"id"`
	TenantID           string                  `json:"tenant_id"`
	LegalEntityID      string                  `json:"legal_entity_id"`
	DefinitionID       string                  `json:"definition_id"`
	DefinitionVersion  int                     `json:"definition_version"`
	DefinitionChecksum string                  `json:"definition_checksum"`
	RequestedByRef     string                  `json:"requested_by_ref"`
	AsOf               time.Time               `json:"as_of"`
	Filter             *ReportFilterExpression `json:"filter"`
	Dataset            ReportDataset           `json:"dataset"`
	Format             ReportFormat            `json:"format"`
	Status             RunStatus               `json:"status"`
	AttemptCount       int                     `json:"attempt_count"`
	RowCount           int                     `json:"row_count"`
	DataObjectKey      string                  `json:"data_object_key,omitempty"`
	DataSHA256         string                  `json:"data_sha256,omitempty"`
	ManifestObjectKey  string                  `json:"manifest_object_key,omitempty"`
	ManifestSHA256     string                  `json:"manifest_sha256,omitempty"`
	FailureCode        string                  `json:"failure_code,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	CompletedAt        *time.Time              `json:"completed_at,omitempty"`
	ExpiresAt          time.Time               `json:"expires_at"`
	SourceBoundary     SourceBoundary          `json:"source_boundary"`
}

// SourceBoundary is captured before generation and persisted even if generation
// later fails, so a reader can tell which material versions the report was read
// from. Without it, as_of is a timestamp rather than a reconstruction point.
type SourceBoundary struct {
	CapturedAt         time.Time            `json:"captured_at"`
	ProjectionVersion  string               `json:"projection_version"`
	SourceHighWater    map[string]time.Time `json:"source_high_water"`
	Population         int                  `json:"population"`
	PopulationComplete bool                 `json:"population_complete"`
}

// Checksum binds every governed field of a definition. Approval is granted
// against this value, so a definition that changes after approval must produce
// a different checksum or the approval would silently cover new content.
func (d ReportDefinition) Checksum() string {
	filter, _ := json.Marshal(d.Filter)
	payload := strings.Join([]string{
		d.TenantID, d.LegalEntityID, d.Code, d.Name, d.Description,
		string(d.Dataset), string(d.ScopeKind), d.ScopeRef, string(d.Format),
		string(filter), strconv.Itoa(d.CurrentVersion), d.MakerID,
	}, "\x1f")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// Manifest is written beside every artefact. It states what was asked for,
// what was actually read, and whether the population was complete, so a
// reader never has to infer completeness from a row count.
type Manifest struct {
	Schema             string                  `json:"schema"`
	GeneratedAt        time.Time               `json:"generated_at"`
	AsOf               time.Time               `json:"as_of"`
	Source             SourceBoundary          `json:"source"`
	DefinitionCode     string                  `json:"definition_code"`
	DefinitionVersion  int                     `json:"definition_version"`
	DefinitionChecksum string                  `json:"definition_checksum"`
	Dataset            ReportDataset           `json:"dataset"`
	ScopeKind          ReportScopeKind         `json:"scope_kind"`
	ScopeRef           string                  `json:"scope_ref,omitempty"`
	RowCount           int                     `json:"row_count"`
	PopulationComplete bool                    `json:"population_complete"`
	Filter             *ReportFilterExpression `json:"filter,omitempty"`
	Coverage           ManifestCoverage        `json:"coverage"`
	DataSHA256         string                  `json:"data_sha256"`
	RetentionUntil     time.Time               `json:"retention_until"`
}

// ManifestCoverage is never a persuasive number. A count the run could not
// establish is omitted, so a reader sees its absence rather than a zero.
type ManifestCoverage struct {
	Population int  `json:"population"`
	Excluded   *int `json:"excluded,omitempty"`
	Unknown    *int `json:"unknown,omitempty"`
}

const (
	ReportRunPageSize  = 100
	MaxReportRunRows   = 10_000
	MaxReportRunBytes  = int64(32 << 20)
	ReportRunRetention = 7 * 24 * time.Hour
	MaxReportRunLease  = 2 * time.Minute
	MaxReportRunTries  = 5

	reportManifestSchema = "clearsight.report-run.v1"
)
