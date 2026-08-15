package evidence

import (
	"context"
	"strings"
)

type visibleRequestRepository interface {
	ListVisibleRequests(context.Context, string, string, int) ([]Request, error)
}

type manageableRequestRepository interface {
	ListManageableRequests(context.Context, string, string, int) ([]Request, error)
}

// ListVisibleRequests is the actor-work boundary. It requires canonical direct
// recipient assignment and subject visibility before the requested limit.
func (s *Service) ListVisibleRequests(ctx context.Context, tenant, principal string, limit int, allowed func(Request) bool) ([]Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" {
		return nil, ErrNotFound
	}
	limit = bounded(limit)
	if repo, ok := s.repo.(visibleRequestRepository); ok {
		values, err := repo.ListVisibleRequests(ctx, tenant, principal, limit)
		if err != nil {
			return nil, err
		}
		return respondentRequests(values), nil
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
	return respondentRequests(visible), nil
}

// ListManageableRequests is the authenticated management boundary. A request
// is manageable only by its direct internal recipient or trusted creator, and
// subject visibility is applied before LIMIT in production repositories.
func (s *Service) ListManageableRequests(ctx context.Context, tenant, principal string, limit int, allowed func(Request) bool) ([]Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(principal) == "" {
		return nil, ErrNotFound
	}
	limit = bounded(limit)
	if repo, ok := s.repo.(manageableRequestRepository); ok {
		values, err := repo.ListManageableRequests(ctx, tenant, principal, limit)
		if err != nil {
			return nil, err
		}
		return manageableRequestViews(values, principal), nil
	}
	values, err := s.repo.ListRequests(ctx, tenant, 200)
	if err != nil {
		return nil, err
	}
	manageable := make([]Request, 0, limit)
	for _, value := range values {
		withRecipient, hydrateErr := hydrateRequestRecipient(ctx, s.repo, value)
		if hydrateErr != nil {
			return nil, hydrateErr
		}
		if !RequestManageableBy(withRecipient, principal) {
			continue
		}
		if allowed != nil && !allowed(withRecipient) {
			continue
		}
		manageable = append(manageable, withRecipient)
		if len(manageable) == limit {
			break
		}
	}
	return manageableRequestViews(manageable, principal), nil
}
