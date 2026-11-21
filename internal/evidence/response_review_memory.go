package evidence

import "context"

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

func (s *MemoryDistributionStore) WriteResponseReview(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer, apply func(context.Context, assessmentMaterial, *ResponseReview) (ResponseReview, error)) (ResponseReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.assessmentMaterialLocked(ctx, tenant, entity, principal, response, authorize)
	if err != nil {
		return ResponseReview{}, err
	}
	versions := s.responseReviews[response]
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
	event := responseReviewEvent(m, review)
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return review, nil
}
