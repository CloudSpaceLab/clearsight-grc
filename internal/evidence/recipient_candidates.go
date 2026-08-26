package evidence

import (
	"context"
	"errors"
	"sort"
	"strings"
)

var ErrRecipientCandidatesUnavailable = errors.New("evidence recipient candidates are unavailable")
var ErrRecipientCandidateSearchInvalid = errors.New("evidence recipient candidate search is invalid")

type recipientCandidateRepository interface {
	SearchRecipientCandidates(context.Context, string, string, string, string, string, int) ([]RecipientCandidate, error)
}

func (s *Service) ListRecipientCandidates(ctx context.Context, scope ActorRequestScope, requestID string, limit int) ([]RecipientCandidate, error) {
	page, err := s.SearchRecipientCandidates(ctx, scope, requestID, RecipientCandidateSearch{Limit: limit})
	return page.Items, err
}

func (s *Service) SearchRecipientCandidates(ctx context.Context, scope ActorRequestScope, requestID string, search RecipientCandidateSearch) (RecipientCandidatePage, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.ActorPrincipalID = strings.TrimSpace(scope.ActorPrincipalID)
	requestID = strings.TrimSpace(requestID)
	search.Query = strings.TrimSpace(search.Query)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.ActorPrincipalID == "" || requestID == "" {
		return RecipientCandidatePage{}, ErrNotFound
	}
	if len([]rune(search.Query)) > 100 {
		return RecipientCandidatePage{}, ErrRecipientCandidateSearchInvalid
	}
	request, err := s.GetRequestForEntity(ctx, scope.TenantID, scope.LegalEntityID, requestID)
	if err != nil {
		return RecipientCandidatePage{}, err
	}
	if request.CreatedBy == "" || request.CreatedBy != scope.ActorPrincipalID || !requestOpenAt(request, s.now().UTC()) || request.AudienceType != "INTERNAL" {
		return RecipientCandidatePage{}, ErrNotFound
	}
	subjectType := strings.ToUpper(strings.TrimSpace(request.SubjectType))
	if subjectType != "PROGRAM" && subjectType != "MATTER" {
		return RecipientCandidatePage{}, ErrNotFound
	}
	checker, ok := s.repo.(SubjectAccessChecker)
	if !ok {
		return RecipientCandidatePage{}, ErrRecipientCandidatesUnavailable
	}
	allowed, err := checker.CanReadSubject(ctx, request.TenantID, scope.ActorPrincipalID, request.SubjectType, request.SubjectID)
	if err != nil {
		return RecipientCandidatePage{}, err
	}
	if !allowed {
		return RecipientCandidatePage{}, ErrNotFound
	}
	repository, ok := s.repo.(recipientCandidateRepository)
	if !ok {
		return RecipientCandidatePage{}, ErrRecipientCandidatesUnavailable
	}
	limit := boundedRecipientCandidateLimit(search.Limit)
	values, err := repository.SearchRecipientCandidates(ctx, scope.TenantID, scope.LegalEntityID, request.ID, scope.ActorPrincipalID, search.Query, limit+1)
	if err != nil {
		return RecipientCandidatePage{}, err
	}
	page := RecipientCandidatePage{Items: values}
	if len(values) > limit {
		page.HasMore = true
		page.Items = values[:limit]
	}
	return page, nil
}

func (r *MemoryRepository) ListRecipientCandidates(ctx context.Context, tenant, legalEntityID, requestID, actorPrincipalID string, limit int) ([]RecipientCandidate, error) {
	return r.SearchRecipientCandidates(ctx, tenant, legalEntityID, requestID, actorPrincipalID, "", boundedRecipientCandidateLimit(limit))
}

func (r *MemoryRepository) SearchRecipientCandidates(_ context.Context, tenant, legalEntityID, requestID, actorPrincipalID, query string, limit int) ([]RecipientCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	request, ok := r.requests[requestID]
	if !ok || request.TenantID != tenant || request.LegalEntityID != legalEntityID || request.CreatedBy != actorPrincipalID {
		return nil, ErrNotFound
	}
	subjectKey := strings.ToUpper(strings.TrimSpace(request.SubjectType)) + ":" + strings.TrimSpace(request.SubjectID)
	requester, ok := r.candidates[actorPrincipalID]
	if !ok || !memoryRecipientEligibleForScope(requester, tenant, legalEntityID, subjectKey) {
		return nil, ErrNotFound
	}
	values := make([]RecipientCandidate, 0, limit)
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	for _, candidate := range r.candidates {
		if !memoryRecipientEligibleForScope(candidate, tenant, legalEntityID, subjectKey) {
			continue
		}
		if normalizedQuery != "" && !strings.Contains(strings.ToLower(candidate.DisplayName), normalizedQuery) && !strings.Contains(strings.ToLower(candidate.ContextLabel), normalizedQuery) {
			continue
		}
		values = append(values, RecipientCandidate{PrincipalID: candidate.PrincipalID, DisplayName: candidate.DisplayName, ContextLabel: candidate.ContextLabel})
	}
	sort.Slice(values, func(i, j int) bool {
		left, right := strings.ToLower(values[i].DisplayName), strings.ToLower(values[j].DisplayName)
		if left == right {
			leftContext, rightContext := strings.ToLower(values[i].ContextLabel), strings.ToLower(values[j].ContextLabel)
			if leftContext == rightContext {
				return values[i].PrincipalID < values[j].PrincipalID
			}
			return leftContext < rightContext
		}
		return left < right
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func memoryRecipientEligibleForScope(candidate RecipientCandidate, tenant, legalEntityID, subjectKey string) bool {
	return candidate.TenantID == tenant && candidate.Kind == "PERSON" && candidate.Active &&
		containsExact(candidate.LegalEntityIDs, legalEntityID) && candidate.ReadableSubjects[subjectKey]
}

func boundedRecipientCandidateLimit(limit int) int {
	if limit <= 0 || limit > 50 {
		return 50
	}
	return limit
}

func containsExact(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

var _ recipientCandidateRepository = (*MemoryRepository)(nil)
