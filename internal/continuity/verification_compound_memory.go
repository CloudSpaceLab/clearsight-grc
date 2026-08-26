package continuity

import "context"

func (r *MemoryRepository) ApplyVerificationResultBundle(_ context.Context, bundle VerificationResultBundle) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	aggregate, ok := r.matters[bundle.TenantID][bundle.MatterID]
	if !ok {
		return ErrNotFound
	}
	if aggregate.Matter.Version != bundle.ExpectedVersion || bundle.ResultEvent.AggregateVersion != bundle.ExpectedVersion+1 {
		return ErrVersionConflict
	}

	next := cloneMatterAggregate(aggregate)
	if err := applyMatterEventToAggregate(&next, bundle.ResultEvent); err != nil {
		return err
	}
	next.Matter.Version = bundle.ResultEvent.AggregateVersion
	next.Matter.UpdatedAt = bundle.ResultEvent.OccurredAt

	finalEvents := []Event{bundle.ResultEvent}
	if bundle.TransitionEvent != nil {
		if bundle.TransitionEvent.AggregateVersion != next.Matter.Version+1 {
			return ErrVersionConflict
		}
		if err := applyMatterEventToAggregate(&next, *bundle.TransitionEvent); err != nil {
			return err
		}
		next.Matter.Version = bundle.TransitionEvent.AggregateVersion
		next.Matter.UpdatedAt = bundle.TransitionEvent.OccurredAt
		finalEvents = append(finalEvents, *bundle.TransitionEvent)
	}
	if bundle.EscalationEvent != nil {
		if bundle.EscalationAction == nil || bundle.EscalationEvent.AggregateVersion != next.Matter.Version+1 {
			return ErrVersionConflict
		}
		if err := applyMatterEventToAggregate(&next, *bundle.EscalationEvent); err != nil {
			return err
		}
		next.Matter.Version = bundle.EscalationEvent.AggregateVersion
		next.Matter.UpdatedAt = bundle.EscalationEvent.OccurredAt
		finalEvents = append(finalEvents, *bundle.EscalationEvent)
	}
	next.Closure = assessClosure(next)

	var followAggregate MatterAggregate
	if bundle.FollowUpMatter != nil {
		for _, existing := range r.matters[bundle.TenantID] {
			if existing.Matter.TriggerKey == bundle.FollowUpMatter.TriggerKey && existing.Matter.Status != MatterClosed && existing.Matter.Status != MatterCancelled {
				return ErrDuplicate
			}
		}
		follow := *bundle.FollowUpMatter
		follow.Version = 1
		followAggregate = MatterAggregate{Matter: follow, Closure: ClosureAssessment{Ready: false}}
		if bundle.FollowUpLink != nil {
			if bundle.FollowUpLinkEvent == nil || bundle.FollowUpLinkEvent.AggregateVersion != 2 {
				return ErrVersionConflict
			}
			if r.programs[bundle.TenantID][bundle.FollowUpLink.ProgramID].Program.ID == "" {
				return ErrNotFound
			}
			if err := applyMatterEventToAggregate(&followAggregate, *bundle.FollowUpLinkEvent); err != nil {
				return err
			}
			followAggregate.Matter.Version = 2
			followAggregate.Matter.UpdatedAt = bundle.FollowUpLinkEvent.OccurredAt
		}
		followAggregate.Closure = assessClosure(followAggregate)
	}

	r.matters[bundle.TenantID][bundle.MatterID] = next
	r.matterEvents[bundle.TenantID][bundle.MatterID] = append(r.matterEvents[bundle.TenantID][bundle.MatterID], finalEvents...)
	if bundle.FollowUpMatter != nil {
		if r.matters[bundle.TenantID] == nil {
			r.matters[bundle.TenantID] = map[string]MatterAggregate{}
		}
		if r.matterEvents[bundle.TenantID] == nil {
			r.matterEvents[bundle.TenantID] = map[string][]Event{}
		}
		r.matters[bundle.TenantID][bundle.FollowUpMatter.ID] = followAggregate
		events := []Event{*bundle.FollowUpEvent}
		if bundle.FollowUpLinkEvent != nil {
			events = append(events, *bundle.FollowUpLinkEvent)
		}
		r.matterEvents[bundle.TenantID][bundle.FollowUpMatter.ID] = events
	}
	return nil
}
