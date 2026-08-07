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
	ErrResourceLimit   = errors.New("document exceeds an extraction resource limit")
)

type ExtractionStatus string
type AnalysisStatus string
type ProposalStatus string

const (
	ExtractionPending     ExtractionStatus = "PENDING"
	ExtractionExtracted   ExtractionStatus = "EXTRACTED"
	ExtractionUnsupported ExtractionStatus = "UNSUPPORTED"
	ExtractionFailed      ExtractionStatus = "FAILED"

	AnalysisPending        AnalysisStatus = "PENDING"
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
	ID               string           `json:"id"`
	TenantID         string           `json:"tenant_id"`
	LegalEntityID    string           `json:"legal_entity_id,omitempty"`
	FileName         string           `json:"file_name"`
	MediaType        string           `json:"media_type"`
	Purpose          string           `json:"purpose"`
	SourceType       string           `json:"source_type"`
	SizeBytes        int64            `json:"size_bytes"`
	SHA256           string           `json:"sha256"`
	StorageKey       string           `json:"storage_key"`
	ArtifactStatus   string           `json:"artifact_status"`
	ExtractionStatus ExtractionStatus `json:"extraction_status"`
	ExtractionMethod string           `json:"extraction_method"`
	AnalysisStatus   AnalysisStatus   `json:"analysis_status"`
	AnalysisMethod   string           `json:"analysis_method"`
	Limitations      []string         `json:"limitations"`
	Sections         []Section        `json:"sections"`
	Proposals        []Proposal       `json:"proposals"`
	SectionsTotal    int              `json:"sections_total"`
	SectionsOmitted  int              `json:"sections_omitted"`
	ProposalsTotal   int              `json:"proposals_total"`
	ProposalsOmitted int              `json:"proposals_omitted"`
	ContentTruncated bool             `json:"content_truncated"`
	ProcessedAt      *time.Time       `json:"processed_at,omitempty"`
	CreatedBy        string           `json:"created_by"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Version          int64            `json:"version"`
}

type DocumentSummary struct {
	ID                    string           `json:"id"`
	TenantID              string           `json:"tenant_id"`
	LegalEntityID         string           `json:"legal_entity_id,omitempty"`
	FileName              string           `json:"file_name"`
	MediaType             string           `json:"media_type"`
	Purpose               string           `json:"purpose"`
	SourceType            string           `json:"source_type"`
	SizeBytes             int64            `json:"size_bytes"`
	SHA256                string           `json:"sha256"`
	ArtifactStatus        string           `json:"artifact_status"`
	ExtractionStatus      ExtractionStatus `json:"extraction_status"`
	AnalysisStatus        AnalysisStatus   `json:"analysis_status"`
	SectionsTotal         int              `json:"sections_total"`
	SectionsOmitted       int              `json:"sections_omitted"`
	ProposalsTotal        int              `json:"proposals_total"`
	ProposalsOmitted      int              `json:"proposals_omitted"`
	PendingProposalCount  int              `json:"pending_proposal_count"`
	ReviewedProposalCount int              `json:"reviewed_proposal_count"`
	ContentTruncated      bool             `json:"content_truncated"`
	ProcessedAt           *time.Time       `json:"processed_at,omitempty"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
	Version               int64            `json:"version"`
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

func summarizeDocument(value Document) DocumentSummary {
	pending := 0
	for _, proposal := range value.Proposals {
		if proposal.Status == ProposalPending {
			pending++
		}
	}
	return DocumentSummary{
		ID: value.ID, TenantID: value.TenantID, LegalEntityID: value.LegalEntityID,
		FileName: value.FileName, MediaType: value.MediaType, Purpose: value.Purpose, SourceType: value.SourceType,
		SizeBytes: value.SizeBytes, SHA256: value.SHA256, ArtifactStatus: value.ArtifactStatus,
		ExtractionStatus: value.ExtractionStatus, AnalysisStatus: value.AnalysisStatus,
		SectionsTotal: value.SectionsTotal, SectionsOmitted: value.SectionsOmitted,
		ProposalsTotal: value.ProposalsTotal, ProposalsOmitted: value.ProposalsOmitted,
		PendingProposalCount: pending, ReviewedProposalCount: len(value.Proposals) - pending,
		ContentTruncated: value.ContentTruncated, ProcessedAt: value.ProcessedAt,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, Version: value.Version,
	}
}
