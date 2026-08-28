package evidence

import (
	"context"
	"sort"
	"strings"
)

type distributionRecipientCandidateDirectory interface {
	SearchDistributionRecipientCandidates(context.Context, string, string, string, int) ([]RecipientCandidate, error)
}

// SearchDistributionRecipientCandidates returns a bounded, display-safe set of
// active internal people for a legal entity. Subject-specific visibility is
// intentionally not applied here: distribution creation can target several
// governed subject types, while the authenticated HTTP boundary already binds
// the caller to the current tenant and legal entity.
func (s *Service) SearchDistributionRecipientCandidates(ctx context.Context, tenantID, legalEntityID string, search RecipientCandidateSearch) (RecipientCandidatePage, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	search.Query = strings.TrimSpace(search.Query)
	if s == nil || tenantID == "" || legalEntityID == "" {
		return RecipientCandidatePage{}, ErrNotFound
	}
	if len([]rune(search.Query)) > 100 {
		return RecipientCandidatePage{}, ErrRecipientCandidateSearchInvalid
	}
	directory, ok := s.repo.(distributionRecipientCandidateDirectory)
	if !ok {
		return RecipientCandidatePage{}, ErrRecipientCandidatesUnavailable
	}
	limit := boundedRecipientCandidateLimit(search.Limit)
	values, err := directory.SearchDistributionRecipientCandidates(ctx, tenantID, legalEntityID, search.Query, limit+1)
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

func (r *MemoryRepository) SearchDistributionRecipientCandidates(_ context.Context, tenantID, legalEntityID, query string, limit int) ([]RecipientCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	values := make([]RecipientCandidate, 0, limit)
	for _, candidate := range r.candidates {
		if candidate.TenantID != tenantID || candidate.Kind != "PERSON" || !candidate.Active || !containsExact(candidate.LegalEntityIDs, legalEntityID) {
			continue
		}
		if normalizedQuery != "" && !strings.Contains(strings.ToLower(candidate.DisplayName), normalizedQuery) && !strings.Contains(strings.ToLower(candidate.ContextLabel), normalizedQuery) {
			continue
		}
		values = append(values, RecipientCandidate{PrincipalID: candidate.PrincipalID, DisplayName: candidate.DisplayName, ContextLabel: candidate.ContextLabel})
	}
	sortRecipientCandidates(values)
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func sortRecipientCandidates(values []RecipientCandidate) {
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
}

var _ distributionRecipientCandidateDirectory = (*MemoryRepository)(nil)
