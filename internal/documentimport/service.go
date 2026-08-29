package documentimport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const EventDocumentProcessingRequested = "DocumentImportProcessingRequested"

type Service struct {
	repo                   Repository
	store                  evidence.ObjectStore
	maxBytes               int64
	allowUnscannedAnalysis bool
	extractionPolicy       ExtractionPolicy
	parserAdapter          ParserAdapter
	parserAdapterPolicy    ParserAdapterPolicy
	legacyOfficeConverter  *LegacyOfficeConverter
	now                    func() time.Time
}

func NewService(repo Repository, store evidence.ObjectStore) *Service {
	return &Service{
		repo: repo, store: store, maxBytes: 20 << 20, allowUnscannedAnalysis: true,
		extractionPolicy: DefaultExtractionPolicy(), now: time.Now,
	}
}

func (s *Service) Configure(maxBytes int64, allowUnscannedAnalysis ...bool) {
	if maxBytes > 0 {
		s.maxBytes = maxBytes
	}
	if len(allowUnscannedAnalysis) > 0 {
		s.allowUnscannedAnalysis = allowUnscannedAnalysis[0]
	}
}

func (s *Service) ConfigureExtractionPolicy(policy ExtractionPolicy) {
	s.extractionPolicy = policy.normalized()
}

func (s *Service) ConfigureAdvancedExtraction(adapter ParserAdapter, policy ParserAdapterPolicy, converter *LegacyOfficeConverter) {
	if s == nil {
		return
	}
	policy = policy.normalized(s.extractionPolicy, s.maximumBytes())
	if adapter == nil {
		policy.Enabled = false
	}
	s.parserAdapter = adapter
	s.parserAdapterPolicy = policy
	s.legacyOfficeConverter = converter
}

func (s *Service) Import(ctx context.Context, input ImportInput, reader io.Reader) (Document, error) {
	if s == nil || s.repo == nil || s.store == nil {
		return Document{}, fmt.Errorf("document import is unavailable")
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.FileName = safeFileName(input.FileName)
	input.MediaType = strings.TrimSpace(input.MediaType)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.CreatedBy = strings.TrimSpace(input.CreatedBy)
	if input.TenantID == "" || input.FileName == "" || input.CreatedBy == "" {
		return Document{}, fmt.Errorf("tenant, file name and importing principal are required")
	}
	if input.Purpose == "" {
		input.Purpose = "Review imported source material"
	}
	if input.SourceType == "" {
		input.SourceType = "DOCUMENT"
	}
	maximum := s.maximumBytes()
	id := newID()
	key := filepath.ToSlash(filepath.Join("document-imports", input.TenantID, id, input.FileName))
	object, err := s.store.Put(ctx, key, reader, maximum)
	if errors.Is(err, evidence.ErrArtifactTooLarge) {
		return Document{}, ErrTooLarge
	}
	if err != nil {
		return Document{}, fmt.Errorf("store original document: %w", err)
	}
	if object.SizeBytes <= 0 {
		_ = s.store.Delete(ctx, key)
		return Document{}, fmt.Errorf("document is empty")
	}

	now := s.now().UTC()
	value := Document{
		ID: id, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		FileName: input.FileName, MediaType: input.MediaType, Purpose: input.Purpose, SourceType: input.SourceType,
		SizeBytes: object.SizeBytes, SHA256: object.SHA256, StorageKey: object.Key, ArtifactStatus: "STORED_UNSCANNED",
		ExtractionStatus: ExtractionPending, ExtractionMethod: "PENDING",
		AnalysisStatus: AnalysisPending, AnalysisMethod: "DETERMINISTIC_RULES_V2",
		Limitations: baseLimitations(), Sections: []Section{}, Elements: []ExtractedElement{}, Degradations: []Degradation{}, Proposals: []Proposal{},
		CreatedBy: input.CreatedBy, CreatedAt: now, UpdatedAt: now, Version: 1,
	}

	if queued, ok := s.repo.(QueuedRepository); ok {
		created, createErr := queued.CreatePending(ctx, value)
		if createErr != nil {
			_ = s.store.Delete(ctx, key)
			return Document{}, createErr
		}
		return withDerivedExtractionDetails(created), nil
	}

	processed, err := s.processStored(ctx, value)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return Document{}, err
	}
	created, err := s.repo.Create(ctx, processed)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return Document{}, err
	}
	return withDerivedExtractionDetails(created), nil
}

func (s *Service) List(ctx context.Context, tenant string, limit int) ([]DocumentSummary, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	return s.repo.List(ctx, strings.TrimSpace(tenant), limit)
}

func (s *Service) Get(ctx context.Context, tenant, id string) (Document, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return Document{}, ErrNotFound
	}
	value, err := s.repo.Get(ctx, strings.TrimSpace(tenant), strings.TrimSpace(id))
	if err != nil {
		return Document{}, err
	}
	return withDerivedExtractionDetails(value), nil
}

func (s *Service) ReviewProposal(ctx context.Context, input ReviewInput) (Document, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.ProposalID = strings.TrimSpace(input.ProposalID)
	input.ReviewerID = strings.TrimSpace(input.ReviewerID)
	input.Note = strings.TrimSpace(input.Note)
	if input.Status != ProposalAccepted && input.Status != ProposalRejected {
		return Document{}, ErrInvalidReview
	}
	if input.TenantID == "" || input.DocumentID == "" || input.ProposalID == "" || input.ReviewerID == "" || input.ExpectedVersion < 1 {
		return Document{}, ErrInvalidReview
	}
	value, err := s.repo.ReviewProposal(ctx, input, s.now().UTC())
	if err != nil {
		return Document{}, err
	}
	return withDerivedExtractionDetails(value), nil
}

// Publish consumes the existing durable outbox request produced with a pending
// PostgreSQL import. Durable retry/dead-letter behavior remains owned by the
// runtime outbox rather than a document-specific job framework.
func (s *Service) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if s == nil || event.AggregateType != "DOCUMENT_IMPORT" || event.EventType != EventDocumentProcessingRequested {
		return nil
	}
	return s.Process(ctx, event.TenantID, event.AggregateID)
}

func (s *Service) Process(ctx context.Context, tenant, id string) error {
	value, err := s.repo.Get(ctx, strings.TrimSpace(tenant), strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if value.ExtractionStatus != ExtractionPending && value.AnalysisStatus != AnalysisPending {
		return nil
	}
	processed, err := s.processStored(ctx, value)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, err = s.repo.SaveProcessing(ctx, processed, value.Version)
	if !errors.Is(err, ErrVersionConflict) {
		return err
	}
	current, getErr := s.repo.Get(ctx, value.TenantID, value.ID)
	if getErr == nil && current.ExtractionStatus != ExtractionPending && current.AnalysisStatus != AnalysisPending {
		return nil
	}
	if getErr != nil {
		return getErr
	}
	return err
}

func (s *Service) processStored(ctx context.Context, value Document) (Document, error) {
	stream, err := s.store.Open(ctx, value.StorageKey)
	if err != nil {
		return Document{}, fmt.Errorf("open stored document: %w", err)
	}
	defer stream.Close()
	data, err := readBounded(ctx, stream, s.maximumBytes())
	if err != nil {
		return Document{}, err
	}
	extraction := ExtractAdvanced(ctx, value, data, s.extractionPolicy, s.parserAdapter, s.parserAdapterPolicy, s.legacyOfficeConverter)
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	value.ExtractionStatus = extraction.Status
	value.ExtractionMethod = extraction.Method
	value.ParserVersion = extraction.ParserVersion
	value.AdapterVersion = extraction.AdapterVersion
	value.Sections = extraction.Sections
	value.Elements = cloneElements(extraction.Elements)
	value.Degradations = cloneDegradations(extraction.Degradations)
	value.SectionsTotal = extraction.SectionsTotal
	value.SectionsOmitted = extraction.SectionsOmitted
	value.ContentTruncated = extraction.ContentTruncated
	value.Proposals = []Proposal{}
	value.ProposalsTotal = 0
	value.ProposalsOmitted = 0
	value.Limitations = append(baseLimitations(), extraction.Limitations...)
	value.Tabular = nil
	if _, supported := DetectTabularFormat(value.FileName, value.MediaType); supported {
		metadata, metadataErr := InspectTabularArtifact(ctx, value.FileName, value.MediaType, data, s.extractionPolicy)
		value.Tabular = &metadata
		if metadataErr != nil {
			value.Limitations = append(value.Limitations, "Structured tabular parsing failed; the original artifact remains available for governed review.")
		} else if metadata.RowsRejected > 0 {
			value.Limitations = append(value.Limitations, fmt.Sprintf("Structured tabular parsing rejected %d of %d rows; bounded row diagnostics are retained with the import receipt.", metadata.RowsRejected, metadata.RowsTotal))
		}
	}
	value.AnalysisStatus = AnalysisUnavailable
	value.AnalysisMethod = "DETERMINISTIC_RULES_V2"

	if extraction.Status.hasUsableContent() && s.allowUnscannedAnalysis {
		analysis := AnalyzeBounded(extraction.Sections, s.extractionPolicy.normalized().MaxProposals)
		value.Proposals = analysis.Proposals
		value.ProposalsTotal = analysis.Total
		value.ProposalsOmitted = analysis.Omitted
		if analysis.Omitted > 0 {
			value.Limitations = append(value.Limitations, fmt.Sprintf("%d of %d extracted proposals are available for review. %d additional proposals were omitted because this document exceeded the review limit.", len(analysis.Proposals), analysis.Total, analysis.Omitted))
		}
		if len(analysis.Proposals) == 0 {
			value.AnalysisStatus = AnalysisNoProposals
		} else {
			value.AnalysisStatus = AnalysisReviewRequired
		}
	} else if extraction.Status.hasUsableContent() {
		value.Limitations = append(value.Limitations, "Analysis is blocked until the artifact is marked available by an approved scanning pipeline.")
	}
	if extraction.ContentTruncated || extraction.SectionsOmitted > 0 {
		value.Limitations = append(value.Limitations, fmt.Sprintf("%d source sections were extracted and %d were omitted because this document exceeded the extraction limit.", len(extraction.Sections), extraction.SectionsOmitted))
	}
	now := s.now().UTC()
	value.ProcessedAt = &now
	value.UpdatedAt = now
	return value, nil
}

func readBounded(ctx context.Context, reader io.Reader, maximum int64) ([]byte, error) {
	if maximum <= 0 {
		return nil, ErrTooLarge
	}
	limited := &io.LimitedReader{R: reader, N: maximum + 1}
	data, err := io.ReadAll(&contextReader{ctx: ctx, reader: limited})
	if err != nil {
		return nil, fmt.Errorf("read stored document: %w", err)
	}
	if int64(len(data)) > maximum {
		return nil, ErrTooLarge
	}
	return data, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func (s *Service) maximumBytes() int64 {
	if s.maxBytes > 0 {
		return s.maxBytes
	}
	return 20 << 20
}

func baseLimitations() []string {
	return []string{
		"The artifact has not passed a production malware-scanning service.",
		"Analysis output is a review proposal, not an approved obligation, control, legal interpretation, or compliance conclusion.",
	}
}

func safeFileName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "..", "")
	return strings.TrimSpace(value)
}

func newID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		raw = digest[:16]
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

var _ workflowruntime.Publisher = (*Service)(nil)
