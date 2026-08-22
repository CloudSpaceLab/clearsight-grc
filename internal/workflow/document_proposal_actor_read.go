package workflow

import "context"

type documentProposalActorWorkRepository interface {
	ListDocumentProposalActorWork(context.Context, ListFilter) ([]Task, error)
}

func (s *Service) appendDocumentProposalActorWork(ctx context.Context, filter ListFilter, current []Task) ([]Task, error) {
	if s == nil || !filter.VisibleActorWorkOnly || len(current) >= filter.Limit {
		return current, nil
	}
	repo, ok := s.repo.(documentProposalActorWorkRepository)
	if !ok {
		return current, nil
	}
	remaining := filter.Limit - len(current)
	documentFilter := filter
	documentFilter.WorkflowKind = DocumentProposalWorkflowKind
	documentFilter.VisibleMatterWorkOnly = false
	documentFilter.VisibleActorWorkOnly = false
	documentFilter.Limit = remaining
	values, err := repo.ListDocumentProposalActorWork(ctx, documentFilter)
	if err != nil {
		return nil, err
	}
	return append(current, values...), nil
}
