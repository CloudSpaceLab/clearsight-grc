package workflow

import (
	"context"
	"fmt"
	"strings"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	if strings.TrimSpace(filter.TenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	values, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return s.appendDocumentProposalActorWork(ctx, filter, values)
}
