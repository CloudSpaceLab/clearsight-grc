package continuity

import (
	"context"
	"encoding/json"
	"strings"
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
	// The compound command is already authoritative at this point. PostgreSQL
	// queues projection maintenance in the command transaction; repositories
	// without that contract may refresh best-effort but must not turn a
	// committed command into a false API failure.
	_ = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventMatterLinked, matter.ID, input.ActorID)
	return s.GetMatter(ctx, input.TenantID, matter.ID)
}

func (s *Service) applyTriggerBundle(ctx context.Context, trigger Trigger, aggregate ProgramAggregate, repo TriggerBundleRepository) (ProgramAggregate, *Matter, bool, error) {
	programEvent, err := newEvent(trigger.TenantID, "PROGRAM", trigger.ProgramID, aggregate.Program.Version+1, EventProgramTriggerRecorded, trigger, actorFor(trigger.ActorID), trigger.ActorID, trigger.ObservedAt)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	committedProgram := aggregate
	if err := applyProgramEventToAggregate(&committedProgram, programEvent); err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	committedProgram.Program.Version = programEvent.AggregateVersion
	committedProgram.Program.UpdatedAt = programEvent.OccurredAt
	bundle := TriggerBundle{Trigger: trigger, ProgramEvent: programEvent}
	matterType, title, summary, create := matterForTrigger(trigger)
	if create {
		matterID, err := id.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		now := s.now().UTC()
		matter := Matter{ID: matterID, TenantID: trigger.TenantID, LegalEntityID: aggregate.Program.LegalEntityID, Reference: matterReference(matterID), Type: matterType, Status: MatterInitialReview, Priority: triggerPriority(trigger.Type), Title: title, Summary: summary, Scope: append(json.RawMessage(nil), trigger.Payload...), TriggerType: trigger.Type, TriggerID: trigger.ID, TriggerKey: trigger.DedupeKey, KnownFacts: append(json.RawMessage(nil), trigger.Payload...), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), CreatedAt: now, UpdatedAt: now, Version: 1}
		if strings.EqualFold(trigger.Type, "MONITORING_RESULT_ADVERSE") {
			matter.SourceType = "MONITORING_RESULT"
			matter.SourceID = trigger.SubjectID
			matter.OwnerPrincipalID = aggregate.Program.OwnerPrincipalID
			matter.RequiredAuthority = "CONTROL_ASSURANCE"
		}
		matterEvent, err := newEvent(trigger.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, actorFor(trigger.ActorID), trigger.ActorID, now)
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		linkID, err := id.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		link := MatterLink{ID: linkID, TenantID: trigger.TenantID, MatterID: matter.ID, ProgramID: trigger.ProgramID, Relationship: "AFFECTS", CreatedAt: now}
		linkEvent, err := newEvent(trigger.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, actorFor(trigger.ActorID), trigger.ActorID, now)
		if err != nil {
			return ProgramAggregate{}, nil, false, err
		}
		bundle.Matter, bundle.MatterEvent, bundle.Link, bundle.LinkEvent = &matter, &matterEvent, &link, &linkEvent
	}
	result, err := repo.ApplyTriggerBundle(ctx, bundle)
	if err != nil {
		return ProgramAggregate{}, nil, false, err
	}
	_ = s.requestProgramRefresh(ctx, trigger.TenantID, trigger.ProgramID, trigger.Type, trigger.ID, "system")
	responseProgram := aggregate
	if result.Inserted {
		responseProgram = committedProgram
	}
	// A synchronous in-memory refresh can make the latest derived state
	// available immediately. Enrich the response when that read succeeds, but
	// never turn a committed bundle into a false command failure when the
	// current-state read is temporarily unavailable.
	if refreshed, readErr := s.repo.GetProgram(ctx, trigger.TenantID, trigger.ProgramID); readErr == nil {
		responseProgram = refreshed
	}
	return responseProgram, result.Matter, result.Inserted, nil
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
