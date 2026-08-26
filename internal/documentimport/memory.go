package documentimport

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]Document
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: map[string]Document{}}
}

func (r *MemoryRepository) Create(_ context.Context, value Document) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value.Version == 0 {
		value.Version = 1
	}
	r.items[value.ID] = cloneDocument(value)
	return cloneDocument(value), nil
}

func (r *MemoryRepository) List(_ context.Context, tenant string, limit int) ([]DocumentSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]DocumentSummary, 0, limit)
	for _, value := range r.items {
		if value.TenantID != tenant {
			continue
		}
		values = append(values, summarizeDocument(value))
		if len(values) == limit {
			break
		}
	}
	return values, nil
}

func (r *MemoryRepository) Get(_ context.Context, tenant, id string) (Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.items[id]
	if !ok || value.TenantID != tenant {
		return Document{}, ErrNotFound
	}
	return cloneDocument(value), nil
}

func (r *MemoryRepository) ReviewProposal(_ context.Context, input ReviewInput, now time.Time) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.items[input.DocumentID]
	if !ok || value.TenantID != strings.TrimSpace(input.TenantID) {
		return Document{}, ErrNotFound
	}
	if value.Version != input.ExpectedVersion {
		for _, proposal := range value.Proposals {
			if proposal.ID == input.ProposalID && proposal.Status == input.Status && proposal.ReviewedBy == input.ReviewerID {
				return cloneDocument(value), nil
			}
		}
		return Document{}, ErrVersionConflict
	}
	for index := range value.Proposals {
		if value.Proposals[index].ID != input.ProposalID {
			continue
		}
		if value.Proposals[index].Status == input.Status && value.Proposals[index].ReviewedBy == input.ReviewerID {
			return cloneDocument(value), nil
		}
		if value.Proposals[index].Status != ProposalPending {
			return Document{}, ErrInvalidReview
		}
		value.Proposals[index].Status = input.Status
		value.Proposals[index].ReviewedBy = input.ReviewerID
		value.Proposals[index].ReviewedAt = &now
		value.Proposals[index].ReviewNote = strings.TrimSpace(input.Note)
		if input.Status == ProposalAccepted {
			value.Proposals[index].Handoff = newAcceptedProposalHandoff(input, value.Proposals[index].Title, value.Proposals[index].Statement, now)
		}
		value.UpdatedAt = now
		value.Version++
		r.items[value.ID] = cloneDocument(value)
		return cloneDocument(value), nil
	}
	return Document{}, ErrNotFound
}

func (r *MemoryRepository) SaveProcessing(_ context.Context, value Document, expected int64) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[value.ID]
	if !ok || current.TenantID != value.TenantID {
		return Document{}, ErrNotFound
	}
	if current.Version != expected {
		return Document{}, ErrVersionConflict
	}
	value.Version = current.Version + 1
	r.items[value.ID] = cloneDocument(value)
	return cloneDocument(value), nil
}

func cloneDocument(value Document) Document {
	value.Limitations = append([]string(nil), value.Limitations...)
	value.Sections = append([]Section(nil), value.Sections...)
	value.Proposals = append([]Proposal(nil), value.Proposals...)
	for index := range value.Proposals {
		if value.Proposals[index].Obligation != nil {
			obligation := *value.Proposals[index].Obligation
			obligation.Citations = append([]string(nil), obligation.Citations...)
			obligation.Dates = append([]string(nil), obligation.Dates...)
			obligation.Topics = append([]string(nil), obligation.Topics...)
			obligation.Uncertainty = append([]string(nil), obligation.Uncertainty...)
			value.Proposals[index].Obligation = &obligation
		}
		if value.Proposals[index].Handoff != nil {
			handoff := *value.Proposals[index].Handoff
			if handoff.Route != nil {
				route := *handoff.Route
				handoff.Route = &route
			}
			value.Proposals[index].Handoff = &handoff
		}
	}
	if value.Tabular != nil {
		metadata := *value.Tabular
		metadata.RowErrors = append([]TabularRowError(nil), value.Tabular.RowErrors...)
		metadata.Resources = append([]TabularResource(nil), value.Tabular.Resources...)
		for index := range metadata.Resources {
			metadata.Resources[index].Fields = append([]TabularField(nil), value.Tabular.Resources[index].Fields...)
		}
		value.Tabular = &metadata
	}
	return value
}
