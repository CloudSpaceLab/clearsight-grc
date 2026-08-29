package monitoring

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

type MemoryFormProposalStore struct {
	mu      sync.RWMutex
	values  map[string]FormTemplateProposal
	sources map[string]string
}

func NewMemoryFormProposalStore() *MemoryFormProposalStore {
	return &MemoryFormProposalStore{
		values:  make(map[string]FormTemplateProposal),
		sources: make(map[string]string),
	}
}

func (s *MemoryFormProposalStore) QueuesGeneration() bool { return false }

func (s *MemoryFormProposalStore) Create(_ context.Context, value FormTemplateProposal) (FormTemplateProposal, error) {
	if err := validateNewFormProposal(value); err != nil {
		return FormTemplateProposal{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sourceKey := formProposalSourceKey(value)
	if existingID, ok := s.sources[sourceKey]; ok {
		return cloneFormTemplateProposal(s.values[formProposalKey(value.TenantID, value.LegalEntityID, existingID)]), nil
	}
	key := formProposalKey(value.TenantID, value.LegalEntityID, value.ID)
	if _, exists := s.values[key]; exists {
		return FormTemplateProposal{}, ErrConflict
	}
	stored := cloneFormTemplateProposal(value)
	s.values[key] = stored
	s.sources[sourceKey] = value.ID
	return cloneFormTemplateProposal(stored), nil
}

func (s *MemoryFormProposalStore) Get(_ context.Context, tenantID, legalEntityID, proposalID string) (FormTemplateProposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.values[formProposalKey(tenantID, legalEntityID, proposalID)]
	if !ok {
		return FormTemplateProposal{}, ErrNotFound
	}
	return cloneFormTemplateProposal(value), nil
}

func (s *MemoryFormProposalStore) CompleteGeneration(_ context.Context, value FormTemplateProposal, expectedVersion int64) (FormTemplateProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := formProposalKey(value.TenantID, value.LegalEntityID, value.ID)
	current, ok := s.values[key]
	if !ok {
		return FormTemplateProposal{}, ErrNotFound
	}
	if current.Version != expectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if current.Status != FormProposalGenerating {
		return FormTemplateProposal{}, ErrFormProposalState
	}
	if !sameProposalSource(current, value) {
		return FormTemplateProposal{}, ErrFormProposalSourceChanged
	}
	if value.Status != FormProposalReviewRequired {
		return FormTemplateProposal{}, ErrInvalid
	}
	value.CreatedAt = current.CreatedAt
	value.CreatedBy = current.CreatedBy
	value.Version = current.Version + 1
	stored := cloneFormTemplateProposal(value)
	s.values[key] = stored
	return cloneFormTemplateProposal(stored), nil
}

func (s *MemoryFormProposalStore) FailGeneration(_ context.Context, tenantID, legalEntityID, proposalID string, expectedVersion int64, code, message string, at time.Time) (FormTemplateProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := formProposalKey(tenantID, legalEntityID, proposalID)
	current, ok := s.values[key]
	if !ok {
		return FormTemplateProposal{}, ErrNotFound
	}
	if current.Version != expectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if current.Status != FormProposalGenerating {
		return FormTemplateProposal{}, ErrFormProposalState
	}
	current.Status = FormProposalFailed
	current.FailureCode = boundedProposalText(code, 128)
	current.FailureMessage = boundedProposalText(message, 2000)
	current.UpdatedAt = at.UTC()
	current.Version++
	s.values[key] = current
	return cloneFormTemplateProposal(current), nil
}

func (s *MemoryFormProposalStore) Review(_ context.Context, mutation FormProposalReviewMutation) (FormTemplateProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := formProposalKey(mutation.TenantID, mutation.LegalEntityID, mutation.ProposalID)
	current, ok := s.values[key]
	if !ok {
		return FormTemplateProposal{}, ErrNotFound
	}
	changeIDs := normalizeProposalChangeIDs(mutation.ChangeIDs)
	if current.Status == mutation.Status && current.ReviewedBy == mutation.ReviewerID && current.ResultTemplateID == mutation.ResultTemplateID && current.ResultTemplateVersion == mutation.ResultTemplateVersion && slices.Equal(current.AcceptedChangeIDs, changeIDs) {
		return cloneFormTemplateProposal(current), nil
	}
	if current.Version != mutation.ExpectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if current.Status != FormProposalReviewRequired || (mutation.Status != FormProposalAccepted && mutation.Status != FormProposalRejected) {
		return FormTemplateProposal{}, ErrFormProposalState
	}
	if mutation.Status == FormProposalAccepted {
		if strings.TrimSpace(mutation.ResultTemplateID) == "" || mutation.ResultTemplateVersion < 1 || len(changeIDs) == 0 {
			return FormTemplateProposal{}, ErrInvalid
		}
	} else if mutation.ResultTemplateID != "" || mutation.ResultTemplateVersion != 0 || len(changeIDs) != 0 {
		return FormTemplateProposal{}, ErrInvalid
	}
	at := mutation.At.UTC()
	current.Status = mutation.Status
	current.ReviewedBy = strings.TrimSpace(mutation.ReviewerID)
	current.ReviewedAt = &at
	current.AcceptedChangeIDs = append([]string(nil), changeIDs...)
	current.ResultTemplateID = strings.TrimSpace(mutation.ResultTemplateID)
	current.ResultTemplateVersion = mutation.ResultTemplateVersion
	current.UpdatedAt = at
	current.Version++
	s.values[key] = current
	return cloneFormTemplateProposal(current), nil
}

func normalizeProposalChangeIDs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func formProposalKey(tenantID, legalEntityID, proposalID string) string {
	return strings.TrimSpace(tenantID) + "\x00" + strings.TrimSpace(legalEntityID) + "\x00" + strings.TrimSpace(proposalID)
}

func formProposalSourceKey(value FormTemplateProposal) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d\x00%s\x00%s\x00%d",
		strings.TrimSpace(value.TenantID), strings.TrimSpace(value.LegalEntityID), value.SourceKind,
		strings.TrimSpace(value.SourceDocumentID), value.SourceDocumentVersion, strings.TrimSpace(value.SourceSHA256),
		strings.TrimSpace(value.BaseTemplateID), value.BaseTemplateVersion)
}
