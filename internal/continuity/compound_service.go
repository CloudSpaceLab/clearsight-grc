package continuity

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (s *Service) createMatterWithInitialLink(ctx context.Context, input CreateMatterInput, matter Matter, event Event, repo CompoundRepository) (MatterAggregate, error) {
	linkID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	link := MatterLink{ID: linkID, TenantID: input.TenantID, MatterID: matter.ID, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlID: input.ControlID, Relationship: "AFFECTS", CreatedAt: matter.CreatedAt}
	linkEvent, err := newEvent(input.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, actorFor(input.ActorID), input.ActorID, matter.CreatedAt)
	if err != nil {
		return MatterAggregate{}, err
	}
	if _, err = repo.CreateMatterWithLink(ctx, MatterLinkBundle{Matter: matter, MatterEvent: event, Link: link, LinkEvent: linkEvent}); err != nil {
		return MatterAggregate{}, err
	}
	if err = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventMatterLinked, matter.ID, input.ActorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, matter.ID)
}

func (s *Service) applyTriggerBundle(ctx context.Context, trigger Trigger, aggregate ProgramAggregate, repo TriggerBundleRepository) (ProgramAggregate, *Matter, bool, error) {
	programEvent, err := newEvent(trigger.TenantID, "PROGRAM", trigger.ProgramID, aggregate.Program.Version+1, EventProgramTriggerRecorded, trigger, ActorSystem, "", trigger.ObservedAt)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	bundle := TriggerBundle{Trigger: trigger, ProgramEvent: programEvent}
	matterType, title, summary, create := matterForTrigger(trigger)
	if create {
		matterID, err := id.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		now := s.now().UTC()
		matter := Matter{ID: matterID, TenantID: trigger.TenantID, Reference: matterReference(matterID), Type: matterType, Status: MatterInitialReview, Priority: triggerPriority(trigger.Type), Title: title, Summary: summary, Scope: append(json.RawMessage(nil), trigger.Payload...), TriggerType: trigger.Type, TriggerID: trigger.ID, TriggerKey: trigger.DedupeKey, KnownFacts: append(json.RawMessage(nil), trigger.Payload...), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1}
		matterEvent, err := newEvent(trigger.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, ActorSystem, "", now)
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		linkID, err := id.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		link := MatterLink{ID: linkID, TenantID: trigger.TenantID, MatterID: matter.ID, ProgramID: trigger.ProgramID, Relationship: "AFFECTS", CreatedAt: now}
		linkEvent, err := newEvent(trigger.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, ActorSystem, "", now)
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		bundle.Matter, bundle.MatterEvent, bundle.Link, bundle.LinkEvent = &matter, &matterEvent, &link, &linkEvent
	}
	result, err := repo.ApplyTriggerBundle(ctx, bundle)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	if err = s.requestProgramRefresh(ctx, trigger.TenantID, trigger.ProgramID, trigger.Type, trigger.ID, "system"); err != nil {
		return ProgramAggregate{}, result.Matter, result.Inserted, err
	}
	program, err := s.repo.GetProgram(ctx, trigger.TenantID, trigger.ProgramID)
	if err != nil {
		return ProgramAggregate{}, result.Matter, result.Inserted, err
	}
	return program, result.Matter, result.Inserted, nil
}

func (s *Service) requestProgramRefresh(ctx context.Context, tenant, programID, reason, triggerID, requestedBy string) error {
	if transactional, ok := s.repo.(TransactionalProjectionRepository); ok && transactional.ProjectionQueuedWithCommands() {
		return nil
	}
	return s.refreshProgram(ctx, tenant, programID, reason, triggerID)
}

func (s *Service) programStateAt(ctx context.Context, tenant, programID string, at *time.Time) (*ProgramStateSnapshot, error) {
	if repo, ok := s.repo.(ProgramStateRepository); ok {
		return repo.ProgramStateAt(ctx, tenant, programID, at)
	}
	return nil, nil
}
