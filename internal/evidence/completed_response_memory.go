package evidence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func (s *MemoryDistributionStore) ListCompletedResponses(ctx context.Context, query CompletedResponseQuery) (CompletedResponsePage, error) {
	cursor, err := normalizeCompletedResponseQuery(&query)
	if err != nil {
		return CompletedResponsePage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]CompletedResponseSummary, 0)
	for distributionID, revisions := range s.responseRevisions {
		distribution, ok := s.distributions[distributionID]
		if !ok || distribution.TenantID != query.TenantID || distribution.LegalEntityID != query.LegalEntityID {
			continue
		}
		if allowed, accessErr := s.completedResponseSubjectVisible(ctx, query.TenantID, query.PrincipalID, distribution.SubjectType, distribution.SubjectID); accessErr != nil {
			return CompletedResponsePage{}, accessErr
		} else if !allowed {
			continue
		}
		for _, revision := range revisions {
			value := completedResponseSummary(distribution, revision)
			if completedResponseMatches(value, query) {
				values = append(values, value)
			}
		}
	}
	sort.Slice(values, func(i, j int) bool { return completedResponseLess(values[i], values[j], query.Sort) })
	if cursor.ID != "" {
		filtered := values[:0]
		for _, value := range values {
			if completedResponseAfterCursor(value, cursor) {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	page := CompletedResponsePage{Items: make([]CompletedResponseSummary, 0, min(query.Limit, len(values)))}
	if len(values) > query.Limit {
		page.Items = append(page.Items, values[:query.Limit]...)
		page.NextCursor = encodeCompletedResponseCursor(page.Items[len(page.Items)-1], query.Sort)
	} else {
		page.Items = append(page.Items, values...)
	}
	return page, nil
}

func (s *MemoryDistributionStore) GetCompletedResponse(ctx context.Context, tenantID, legalEntityID, principalID, revisionID string) (CompletedResponseSummary, ResponseRevision, error) {
	if s == nil || tenantID == "" || legalEntityID == "" || principalID == "" || revisionID == "" {
		return CompletedResponseSummary{}, ResponseRevision{}, fmt.Errorf("completed response scope is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for distributionID, revisions := range s.responseRevisions {
		distribution := s.distributions[distributionID]
		if distribution.TenantID != tenantID || distribution.LegalEntityID != legalEntityID {
			continue
		}
		if allowed, accessErr := s.completedResponseSubjectVisible(ctx, tenantID, principalID, distribution.SubjectType, distribution.SubjectID); accessErr != nil {
			return CompletedResponseSummary{}, ResponseRevision{}, accessErr
		} else if !allowed {
			continue
		}
		for _, revision := range revisions {
			if revision.ID == revisionID {
				return completedResponseSummary(distribution, revision), cloneResponseRevision(revision), nil
			}
		}
	}
	return CompletedResponseSummary{}, ResponseRevision{}, ErrNotFound
}

func (s *MemoryDistributionStore) completedResponseSubjectVisible(ctx context.Context, tenantID, principalID, subjectType, subjectID string) (bool, error) {
	switch strings.ToUpper(strings.TrimSpace(subjectType)) {
	case "PROGRAM", "MATTER", "VENDOR_RELATIONSHIP":
		if s.repo == nil {
			return false, nil
		}
		return s.repo.CanReadSubject(ctx, tenantID, principalID, subjectType, subjectID)
	default:
		return true, nil
	}
}

func completedResponseMatches(value CompletedResponseSummary, query CompletedResponseQuery) bool {
	if query.CurrentOnly && !value.Current || query.FormTemplateID != "" && value.FormTemplateID != query.FormTemplateID || query.FormTemplateVersion > 0 && value.FormTemplateVersion != query.FormTemplateVersion || query.SubjectType != "" && value.SubjectType != query.SubjectType || query.SubjectID != "" && value.SubjectID != query.SubjectID {
		return false
	}
	if query.CompletedFrom != nil && value.CompletedAt.Before(query.CompletedFrom.UTC()) || query.CompletedUntil != nil && value.CompletedAt.After(query.CompletedUntil.UTC()) {
		return false
	}
	if value.Score == nil {
		return len(query.Modes) == 0 && len(query.Bands) == 0 && len(query.States) == 0 && query.RawMinimum == nil && query.RawMaximum == nil && query.AdverseMinimum == nil && query.AdverseMaximum == nil
	}
	if !containsScoreMode(query.Modes, value.Score.Mode) || !containsConcernBand(query.Bands, value.Score.Band) || !containsScoreState(query.States, value.Score.State) {
		return false
	}
	return scoreWithin(value.Score.RawScore, query.RawMinimum, query.RawMaximum) && scoreWithin(value.Score.AdverseScore, query.AdverseMinimum, query.AdverseMaximum)
}

func containsScoreMode(values []formcontract.ScoringMode, value formcontract.ScoringMode) bool {
	if len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsConcernBand(values []formcontract.ConcernBand, value formcontract.ConcernBand) bool {
	if len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsScoreState(values []ResponseScoreState, value ResponseScoreState) bool {
	if len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func scoreWithin(value, minimum, maximum *float64) bool {
	if minimum == nil && maximum == nil {
		return true
	}
	if value == nil {
		return false
	}
	return (minimum == nil || *value >= *minimum) && (maximum == nil || *value <= *maximum)
}

func completedResponseLess(left, right CompletedResponseSummary, sortOrder ResponseSort) bool {
	leftScore, rightScore := completedResponseSortScore(left, sortOrder), completedResponseSortScore(right, sortOrder)
	if sortOrder != ResponseSortNewest {
		if leftScore == nil || rightScore == nil {
			if leftScore == nil && rightScore != nil {
				return false
			}
			if leftScore != nil && rightScore == nil {
				return true
			}
		} else if *leftScore != *rightScore {
			if sortOrder == ResponseSortRawAsc {
				return *leftScore < *rightScore
			}
			return *leftScore > *rightScore
		}
	}
	if !left.CompletedAt.Equal(right.CompletedAt) {
		return left.CompletedAt.After(right.CompletedAt)
	}
	return left.ID > right.ID
}

func completedResponseSortScore(value CompletedResponseSummary, sortOrder ResponseSort) *float64 {
	if value.Score == nil {
		return nil
	}
	if sortOrder == ResponseSortConcern {
		return value.Score.AdverseScore
	}
	return value.Score.RawScore
}

func completedResponseAfterCursor(value CompletedResponseSummary, cursor completedResponseCursor) bool {
	probe := CompletedResponseSummary{ID: cursor.ID, CompletedAt: cursor.CompletedAt, Score: &ResponseScoreResult{}}
	if cursor.Sort == ResponseSortConcern {
		probe.Score.AdverseScore = cursor.Score
	} else {
		probe.Score.RawScore = cursor.Score
	}
	return completedResponseLess(probe, value, cursor.Sort)
}
