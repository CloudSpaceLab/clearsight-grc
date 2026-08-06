package documentimport

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("document import not found")
	ErrVersionConflict = errors.New("document import version conflict")
	ErrInvalidReview   = errors.New("invalid proposal review")
	ErrTooLarge        = errors.New("document exceeds the configured size limit")
)

type ExtractionStatus string
type AnalysisStatus string
type ProposalStatus string

const (
	ExtractionExtracted   ExtractionStatus = "EXTRACTED"
	ExtractionUnsupported ExtractionStatus = "UNSUPPORTED"
	ExtractionFailed      ExtractionStatus = "FAILED"

	AnalysisReviewRequired AnalysisStatus = "REVIEW_REQUIRED"
	AnalysisNoProposals    AnalysisStatus = "NO_PROPOSALS"
	AnalysisUnavailable    AnalysisStatus = "UNAVAILABLE"

	ProposalPending  ProposalStatus = "PENDING_REVIEW"
	ProposalAccepted ProposalStatus = "ACCEPTED"
	ProposalRejected ProposalStatus = "REJECTED"
)

type Anchor struct {
	SectionID string `json:"section_id"`
	Quote     string `json:"quote"`
	Page      int    `json:"page,omitempty"`
	Sheet     string `json:"sheet,omitempty"`
	RowStart  int    `json:"row_start,omitempty"`
	RowEnd    int    `json:"row_end,omitempty"`
}

type Section struct {
	ID       string `json:"id"`
	Sequence int    `json:"sequence"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Page     int    `json:"page,omitempty"`
	Sheet    string `json:"sheet,omitempty"`
	RowStart int    `json:"row_start,omitempty"`
	RowEnd   int    `json:"row_end,omitempty"`
}

type Proposal struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Title      string         `json:"title"`
	Statement  string         `json:"statement"`
	Confidence float64        `json:"confidence"`
	Anchor     Anchor         `json:"anchor"`
	Status     ProposalStatus `json:"status"`
	ReviewedBy string         `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time     `json:"reviewed_at,omitempty"`
	ReviewNote string         `json:"review_note,omitempty"`
}

type Document struct {
	ID                string           `json:"id"`
	TenantID          string           `json:"tenant_id"`
	LegalEntityID     string           `json:"legal_entity_id,omitempty"`
	FileName          string           `json:"file_name"`
	MediaType         string           `json:"media_type"`
	Purpose           string           `json:"purpose"`
	SourceType        string           `json:"source_type"`
	SizeBytes         int64            `json:"size_bytes"`
	SHA256            string           `json:"sha256"`
	StorageKey        string           `json:"storage_key"`
	ArtifactStatus    string           `json:"artifact_status"`
	ExtractionStatus  ExtractionStatus `json:"extraction_status"`
	ExtractionMethod  string           `json:"extraction_method"`
	AnalysisStatus    AnalysisStatus   `json:"analysis_status"`
	AnalysisMethod    string           `json:"analysis_method"`
	Limitations       []string         `json:"limitations"`
	Sections          []Section        `json:"sections"`
	Proposals         []Proposal       `json:"proposals"`
	CreatedBy         string           `json:"created_by"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	Version           int64            `json:"version"`
}

type ImportInput struct {
	TenantID      string
	LegalEntityID string
	FileName      string
	MediaType     string
	Purpose       string
	SourceType    string
	CreatedBy     string
}

type ReviewInput struct {
	TenantID        string         `json:"tenant_id,omitempty"`
	DocumentID      string         `json:"document_id,omitempty"`
	ProposalID      string         `json:"proposal_id,omitempty"`
	ReviewerID      string         `json:"reviewer_id,omitempty"`
	Status          ProposalStatus `json:"status"`
	Note            string         `json:"note,omitempty"`
	ExpectedVersion int64          `json:"expected_version"`
}
