package evidence

import (
	"context"
	"encoding/json"
)

func (s *MemoryDistributionStore) ReadResponseReviews(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer, before int64, limit int) (assessmentMaterial, []ResponseReview, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, err := s.assessmentMaterialLocked(ctx, tenant, entity, principal, response, authorize)
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	versions := s.responseReviews[response]
	var latest int64
	if len(versions) > 0 {
		latest = versions[len(versions)-1].Version
	}
	items := make([]ResponseReview, 0, limit)
	for i := len(versions) - 1; i >= 0 && len(items) < limit; i-- {
		if before == 0 || versions[i].Version < before {
			items = append(items, versions[i])
		}
	}
	return m, items, latest, nil
}

func (s *MemoryDistributionStore) WriteResponseReview(ctx context.Context, tenant, entity, principal, response string, input RecordResponseReviewInput, authorize assessmentReadAuthorizer, apply func(context.Context, assessmentMaterial, *ResponseReview) (ResponseReview, error)) (ResponseReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.assessmentMaterialLocked(ctx, tenant, entity, principal, response, authorize)
	if err != nil {
		return ResponseReview{}, err
	}
	versions := s.responseReviews[response]
	var key string
	if input.RequestKey != "" {
		encoded, _ := json.Marshal([]string{tenant, entity, principal, input.RequestKey})
		key = string(encoded)
		if stored, ok := s.responseReviewKeys[key]; ok {
			// Re-run the service's verified session, independence and current route checks.
			var expectedPrevious *ResponseReview
			if input.ExpectedVersion > 0 {
				expectedPrevious = &ResponseReview{Version: input.ExpectedVersion}
			}
			if _, err := apply(ctx, m, expectedPrevious); err != nil {
				return ResponseReview{}, err
			}
			if stored.ResponseID != response || stored.Version != input.ExpectedVersion+1 || stored.Decision != input.Decision || stored.Comment != input.Comment {
				return ResponseReview{}, ErrAssessmentConflict
			}
			return stored, nil
		}
	}
	var previous *ResponseReview
	if len(versions) > 0 {
		v := versions[len(versions)-1]
		previous = &v
	}
	review, err := apply(ctx, m, previous)
	if err != nil {
		return ResponseReview{}, err
	}
	if s.responseReviews == nil {
		s.responseReviews = map[string][]ResponseReview{}
	}
	s.responseReviews[response] = append(versions, review)
	if key != "" {
		if s.responseReviewKeys == nil {
			s.responseReviewKeys = map[string]ResponseReview{}
		}
		s.responseReviewKeys[key] = review
	}
	event := responseReviewEvent(m, review)
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return review, nil
}
