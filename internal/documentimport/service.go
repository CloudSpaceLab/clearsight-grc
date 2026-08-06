package documentimport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type Service struct {
	repo                   Repository
	store                  evidence.ObjectStore
	maxBytes               int64
	allowUnscannedAnalysis bool
	now                    func() time.Time
}

func NewService(repo Repository, store evidence.ObjectStore) *Service {
	return &Service{repo: repo, store: store, maxBytes: 20 << 20, allowUnscannedAnalysis: true, now: time.Now}
}

func (s *Service) Configure(maxBytes int64, allowUnscannedAnalysis ...bool) {
	if maxBytes > 0 {
		s.maxBytes = maxBytes
	}
	if len(allowUnscannedAnalysis) > 0 {
		s.allowUnscannedAnalysis = allowUnscannedAnalysis[0]
	}
}

func (s *Service) Import(ctx context.Context, input ImportInput, reader io.Reader) (Document, error) {
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
	maximum := s.maxBytes
	if maximum <= 0 {
		maximum = 20 << 20
	}
	limited := &io.LimitedReader{R: reader, N: maximum + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return Document{}, fmt.Errorf("read document: %w", err)
	}
	if int64(len(data)) > maximum {
		return Document{}, ErrTooLarge
	}
	if len(data) == 0 {
		return Document{}, fmt.Errorf("document is empty")
	}
	id := newID()
	key := filepath.ToSlash(filepath.Join("document-imports", input.TenantID, id, input.FileName))
	object, err := s.store.Put(ctx, key, bytes.NewReader(data), maximum)
	if err != nil {
		return Document{}, fmt.Errorf("store original document: %w", err)
	}
	extraction := Extract(input.FileName, input.MediaType, data)
	proposals := []Proposal{}
	analysisStatus := AnalysisUnavailable
	analysisMethod := "DETERMINISTIC_RULES_V1"
	limitations := append([]string{
		"The artifact has not passed a production malware-scanning service.",
		"Analysis output is a review proposal, not an approved obligation, control, legal interpretation, or compliance conclusion.",
	}, extraction.Limitations...)
	if extraction.Status == ExtractionExtracted && s.allowUnscannedAnalysis {
		proposals = Analyze(extraction.Sections)
		if len(proposals) == 0 {
			analysisStatus = AnalysisNoProposals
		} else {
			analysisStatus = AnalysisReviewRequired
		}
	} else if extraction.Status == ExtractionExtracted && !s.allowUnscannedAnalysis {
		limitations = append(limitations, "Analysis is blocked until the artifact is marked available by an approved scanning pipeline.")
	}
	now := s.now().UTC()
	value := Document{
		ID: id, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		FileName: input.FileName, MediaType: input.MediaType, Purpose: input.Purpose, SourceType: input.SourceType,
		SizeBytes: object.SizeBytes, SHA256: object.SHA256, StorageKey: object.Key, ArtifactStatus: "STORED_UNSCANNED",
		ExtractionStatus: extraction.Status, ExtractionMethod: extraction.Method,
		AnalysisStatus: analysisStatus, AnalysisMethod: analysisMethod,
		Limitations: limitations, Sections: extraction.Sections, Proposals: proposals,
		CreatedBy: input.CreatedBy, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	created, err := s.repo.Create(ctx, value)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return Document{}, err
	}
	return created, nil
}

func (s *Service) List(ctx context.Context, tenant string, limit int) ([]Document, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	return s.repo.List(ctx, strings.TrimSpace(tenant), limit)
}

func (s *Service) Get(ctx context.Context, tenant, id string) (Document, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return Document{}, ErrNotFound
	}
	return s.repo.Get(ctx, strings.TrimSpace(tenant), strings.TrimSpace(id))
}

func (s *Service) ReviewProposal(ctx context.Context, input ReviewInput) (Document, error) {
	if input.Status != ProposalAccepted && input.Status != ProposalRejected {
		return Document{}, ErrInvalidReview
	}
	if strings.TrimSpace(input.ReviewerID) == "" || input.ExpectedVersion < 1 {
		return Document{}, ErrInvalidReview
	}
	value, err := s.repo.Get(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.DocumentID))
	if err != nil {
		return Document{}, err
	}
	if value.Version != input.ExpectedVersion {
		return Document{}, ErrVersionConflict
	}
	found := false
	now := s.now().UTC()
	for index := range value.Proposals {
		if value.Proposals[index].ID != input.ProposalID {
			continue
		}
		if value.Proposals[index].Status != ProposalPending {
			return Document{}, ErrInvalidReview
		}
		value.Proposals[index].Status = input.Status
		value.Proposals[index].ReviewedBy = input.ReviewerID
		value.Proposals[index].ReviewedAt = &now
		value.Proposals[index].ReviewNote = strings.TrimSpace(input.Note)
		found = true
		break
	}
	if !found {
		return Document{}, ErrNotFound
	}
	value.UpdatedAt = now
	return s.repo.SaveReview(ctx, value, input.ExpectedVersion)
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
