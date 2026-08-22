package workflow

import (
	"context"
	"sort"
)

type documentProposalActorWorkRepository interface {
	ListDocumentProposalActorWork(context.Context, ListFilter) ([]Task, error)
}

func (s *Service) appendDocumentProposalActorWork(ctx context.Context, filter ListFilter, current []Task) ([]Task, error) {
	if s == nil || !filter.VisibleActorWorkOnly || (filter.WorkflowKind != "" && filter.WorkflowKind != DocumentProposalWorkflowKind) {
		return current, nil
	}
	repo, ok := s.repo.(documentProposalActorWorkRepository)
	if !ok {
		return current, nil
	}
	documentFilter := filter
	documentFilter.WorkflowKind = DocumentProposalWorkflowKind
	documentFilter.VisibleMatterWorkOnly = false
	documentFilter.VisibleActorWorkOnly = false
	documentFilter.Limit = filter.Limit
	values, err := repo.ListDocumentProposalActorWork(ctx, documentFilter)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return current, nil
	}

	merged := make([]Task, 0, len(current)+len(values))
	seen := make(map[string]struct{}, len(current)+len(values))
	appendUnique := func(tasks []Task) {
		for _, task := range tasks {
			if _, ok := seen[task.ID]; ok {
				continue
			}
			seen[task.ID] = struct{}{}
			merged = append(merged, task)
		}
	}
	appendUnique(current)
	appendUnique(values)
	sort.SliceStable(merged, func(i, j int) bool {
		left, right := merged[i], merged[j]
		if (left.DueAt == nil) != (right.DueAt == nil) {
			return left.DueAt != nil
		}
		if left.DueAt != nil && right.DueAt != nil && !left.DueAt.Equal(*right.DueAt) {
			return left.DueAt.Before(*right.DueAt)
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.ID < right.ID
	})
	if len(merged) > filter.Limit {
		merged = merged[:filter.Limit]
	}
	return merged, nil
}
