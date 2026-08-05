package evidence

import (
	"context"
	"sort"
	"strings"
)

type subjectLookupRepository interface {
	LatestRequestForSubject(context.Context, string, string, string) (Request, error)
	SourcesByCodes(context.Context, string, []string) ([]Source, error)
}

func (s *Service) LatestRequestForSubject(ctx context.Context, tenant, subjectType, subjectID string) (Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(subjectType) == "" || strings.TrimSpace(subjectID) == "" {
		return Request{}, ErrNotFound
	}
	if repo, ok := s.repo.(subjectLookupRepository); ok {
		return repo.LatestRequestForSubject(ctx, tenant, strings.ToUpper(subjectType), subjectID)
	}
	values, err := s.repo.ListRequests(ctx, tenant, 200)
	if err != nil {
		return Request{}, err
	}
	matches := make([]Request, 0, 1)
	for _, value := range values {
		if strings.EqualFold(value.SubjectType, subjectType) && value.SubjectID == subjectID {
			matches = append(matches, value)
		}
	}
	if len(matches) == 0 {
		return Request{}, ErrNotFound
	}
	sort.Slice(matches, func(i, j int) bool { return requestComesFirst(matches[i], matches[j]) })
	return matches[0], nil
}

func (s *Service) SourcesByCodes(ctx context.Context, tenant string, codes []string) ([]Source, error) {
	if strings.TrimSpace(tenant) == "" || len(codes) == 0 {
		return []Source{}, nil
	}
	if repo, ok := s.repo.(subjectLookupRepository); ok {
		return repo.SourcesByCodes(ctx, tenant, codes)
	}
	values, err := s.repo.ListSources(ctx, tenant, 200)
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, code := range codes {
		wanted[strings.ToUpper(strings.TrimSpace(code))] = true
	}
	result := make([]Source, 0, len(codes))
	for _, value := range values {
		if wanted[strings.ToUpper(value.Code)] {
			result = append(result, value)
		}
	}
	return result, nil
}

func (r *MemoryRepository) LatestRequestForSubject(_ context.Context, tenant, subjectType, subjectID string) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	matches := make([]Request, 0, 2)
	for _, value := range r.requests {
		if value.TenantID == tenant && strings.EqualFold(value.SubjectType, subjectType) && value.SubjectID == subjectID {
			matches = append(matches, cloneRequest(value))
		}
	}
	if len(matches) == 0 {
		return Request{}, ErrNotFound
	}
	sort.Slice(matches, func(i, j int) bool { return requestComesFirst(matches[i], matches[j]) })
	return matches[0], nil
}

func requestComesFirst(left, right Request) bool {
	leftActionable, rightActionable := requestStatusActionable(left.Status), requestStatusActionable(right.Status)
	if leftActionable != rightActionable {
		return leftActionable
	}
	if leftActionable && !left.Deadline.Equal(right.Deadline) {
		return left.Deadline.Before(right.Deadline)
	}
	if !left.UpdatedAt.Equal(right.UpdatedAt) {
		return left.UpdatedAt.After(right.UpdatedAt)
	}
	return left.ID > right.ID
}

func requestStatusActionable(status RequestStatus) bool {
	return status == RequestDraft || status == RequestReady || status == RequestInProgress
}

func (r *MemoryRepository) SourcesByCodes(_ context.Context, tenant string, codes []string) ([]Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wanted := map[string]bool{}
	for _, code := range codes {
		wanted[strings.ToUpper(strings.TrimSpace(code))] = true
	}
	result := make([]Source, 0, len(codes))
	for _, source := range r.sources {
		if source.TenantID == tenant && wanted[strings.ToUpper(source.Code)] {
			result = append(result, source)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}
