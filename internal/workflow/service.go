package workflow

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
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
	filter.Search = strings.ToLower(strings.TrimSpace(filter.Search))
	if utf8.RuneCountInString(filter.Search) > 120 {
		return nil, fmt.Errorf("work search cannot exceed 120 characters")
	}
	if (filter.VisibleActorWorkOnly || filter.VisibleMatterWorkOnly) && strings.TrimSpace(filter.PrincipalID) == "" {
		return nil, fmt.Errorf("actor work requires principal_id")
	}
	if filter.DueFrom != nil && filter.DueBefore != nil && !filter.DueFrom.Before(*filter.DueBefore) {
		return nil, fmt.Errorf("due window must end after it begins")
	}
	if filter.PageAfter != nil && (!filter.ActiveOnly || filter.PageAfter.ID == "" || filter.PageAfter.UpdatedAt.IsZero()) {
		return nil, fmt.Errorf("invalid active work position")
	}
	if filter.UpdatedThrough != nil && filter.UpdatedThrough.IsZero() {
		return nil, fmt.Errorf("invalid work update fence")
	}
	values, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return s.appendDocumentProposalActorWork(ctx, filter, values)
}
