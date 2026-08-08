package evidence

import (
	"context"
	"strings"
)

type visibleRequestRepository interface {
	ListVisibleRequests(context.Context, string, string, int) ([]Request, error)
}

// ListVisibleRequests requires both canonical recipient assignment and subject
// visibility before the requested limit. PostgreSQL performs both predicates
// in the query; lightweight adapters first obtain recipient-bound candidates
// and then apply the supplied subject-access check.
func (s *Service) ListVisibleRequests(ctx context.Context, tenant, principal string, limit int, allowed func(Request) bool) ([]Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" {
		return nil, ErrNotFound
	}
	limit = bounded(limit)
	if repo, ok := s.repo.(visibleRequestRepository); ok {
		return repo.ListVisibleRequests(ctx, tenant, principal, limit)
	}
	store, err := recipientPersistence(s.repo)
	if err != nil {
		return nil, err
	}
	values, err := store.ListRecipientRequests(ctx, tenant, principal, 200)
	if err != nil {
		return nil, err
	}
	visible := make([]Request, 0, limit)
	for _, value := range values {
		if !RequestAssignedTo(value, principal) {
			continue
		}
		if allowed != nil && !allowed(value) {
			continue
		}
		visible = append(visible, value)
		if len(visible) == limit {
			break
		}
	}
	return visible, nil
}
