package documentimport

import (
	"context"
	"fmt"
	"strings"
)

type scopedReadRepository interface {
	ListScoped(context.Context, string, string, int) ([]DocumentSummary, error)
	GetScoped(context.Context, string, string, string) (Document, error)
}

func (s *Service) ListVisible(ctx context.Context, tenant, legalEntityID string, limit int) ([]DocumentSummary, error) {
	tenant = strings.TrimSpace(tenant)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenant == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	if legalEntityID == "*" {
		return s.List(ctx, tenant, limit)
	}
	repo, ok := s.repo.(scopedReadRepository)
	if !ok {
		return nil, fmt.Errorf("legal-entity-scoped document reads are unavailable")
	}
	return repo.ListScoped(ctx, tenant, legalEntityID, limit)
}

func (s *Service) GetVisible(ctx context.Context, tenant, legalEntityID, id string) (Document, error) {
	tenant = strings.TrimSpace(tenant)
	legalEntityID = strings.TrimSpace(legalEntityID)
	id = strings.TrimSpace(id)
	if tenant == "" || id == "" {
		return Document{}, ErrNotFound
	}
	if legalEntityID == "*" {
		return s.Get(ctx, tenant, id)
	}
	repo, ok := s.repo.(scopedReadRepository)
	if !ok {
		return Document{}, ErrNotFound
	}
	return repo.GetScoped(ctx, tenant, legalEntityID, id)
}
