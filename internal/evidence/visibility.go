package evidence

import (
	"context"
	"strings"
)

type visibleRequestRepository interface {
	ListVisibleRequests(context.Context, string, string, int) ([]Request, error)
}

// ListVisibleRequests applies subject visibility before the requested limit.
// Repository implementations should perform the access predicate in the
// database query; lightweight repositories use the supplied fail-closed check
// before truncating the result.
func (s *Service) ListVisibleRequests(ctx context.Context, tenant, principal string, limit int, allowed func(Request) bool) ([]Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" {
		return nil, ErrNotFound
	}
	limit = bounded(limit)
	if repo, ok := s.repo.(visibleRequestRepository); ok {
		return repo.ListVisibleRequests(ctx, tenant, principal, limit)
	}
	values, err := s.repo.ListRequests(ctx, tenant, 200)
	if err != nil {
		return nil, err
	}
	visible := make([]Request, 0, limit)
	for _, value := range values {
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
