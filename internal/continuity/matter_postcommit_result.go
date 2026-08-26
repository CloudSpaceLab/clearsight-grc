package continuity

import "context"

// applyMatterValueAndResult validates the exact event against the pre-command
// aggregate before committing it, then returns either the current aggregate or
// a deterministic description of the committed event when reconstruction is
// temporarily unavailable.
func (s *Service) applyMatterValueAndResult(ctx context.Context, fallback MatterAggregate, tenant, matterID string, expected int64, eventType string, value any, actorID string) (MatterAggregate, error) {
	event, err := newEvent(tenant, "MATTER", matterID, expected+1, eventType, value, actorFor(actorID), actorID, s.now().UTC())
	if err != nil {
		return MatterAggregate{}, err
	}
	return s.applyMatterEventAndResult(ctx, fallback, event)
}

func (s *Service) applyMatterEventAndResult(ctx context.Context, fallback MatterAggregate, event Event) (MatterAggregate, error) {
	committed, err := matterResultFromEvents(fallback, event)
	if err != nil {
		return MatterAggregate{}, err
	}
	if _, err = s.repo.ApplyMatterEvent(ctx, event.TenantID, event.AggregateID, event.AggregateVersion-1, event); err != nil {
		return MatterAggregate{}, err
	}
	return s.currentMatterOrFallback(ctx, event.TenantID, event.AggregateID, committed), nil
}

func matterResultFromEvents(fallback MatterAggregate, events ...Event) (MatterAggregate, error) {
	committed := fallback
	for _, event := range events {
		if err := applyMatterEventToAggregate(&committed, event); err != nil {
			return MatterAggregate{}, err
		}
		committed.Matter.Version = event.AggregateVersion
		committed.Matter.UpdatedAt = event.OccurredAt
	}
	committed.Closure = assessClosure(committed)
	return decorateMatter(committed), nil
}

func (s *Service) currentMatterOrFallback(ctx context.Context, tenant, matterID string, fallback MatterAggregate) MatterAggregate {
	if current, err := s.GetMatter(ctx, tenant, matterID); err == nil {
		return current
	}
	fallback.Closure = assessClosure(fallback)
	return decorateMatter(fallback)
}
