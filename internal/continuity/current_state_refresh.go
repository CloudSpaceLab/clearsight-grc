package continuity

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// refreshProgramCurrent is the projection-worker path. It augments the
// event-backed aggregate with current effective evidence targets and the
// authoritative evidence-source denominator before deriving status, while
// retaining the existing projection transaction model.
func (s *Service) refreshProgramCurrent(ctx context.Context, tenant, programID, triggerType, triggerID string) error {
	aggregate, err := s.repo.GetProgram(ctx, tenant, programID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	aggregate.EvidenceContracts = effectiveEvidenceContracts(aggregate, now)
	openMatters, err := s.repo.OpenMatterCount(ctx, tenant, programID)
	if err != nil {
		return err
	}
	sourceState := inferProgramSourceState(aggregate, now)
	if sourceRepo, ok := s.repo.(ProgramSourceStateRepository); ok {
		sourceState, err = sourceRepo.CurrentProgramSourceState(ctx, tenant, programID)
		if err != nil {
			return err
		}
	}
	state := deriveProgramStateWithSourceState(aggregate, openMatters, now, sourceState)
	state.ID, err = id.NewUUIDv7()
	if err != nil {
		return err
	}
	state.TriggerType = triggerType
	state.TriggerID = triggerID
	state.ProgramVersion = aggregate.Program.Version
	if aggregate.CurrentState != nil && stateEquivalent(*aggregate.CurrentState, state) && aggregate.CurrentState.ProgramVersion == aggregate.Program.Version {
		return nil
	}
	if projectionRepo, ok := s.repo.(ProgramStateRepository); ok {
		_, err = projectionRepo.SaveProgramState(ctx, tenant, programID, aggregate.Program.Version, state)
		return err
	}
	event, err := newEvent(tenant, "PROGRAM", programID, aggregate.Program.Version+1, EventProgramStateUpdated, state, ActorSystem, "", s.now().UTC())
	if err != nil {
		return err
	}
	_, err = s.repo.ApplyProgramEvent(ctx, tenant, programID, aggregate.Program.Version, event)
	return err
}
