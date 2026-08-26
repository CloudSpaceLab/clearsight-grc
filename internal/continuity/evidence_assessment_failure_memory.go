package continuity

import "context"

func (r *MemoryRepository) RecordEvidenceAssessmentWithFailure(ctx context.Context, bundle EvidenceAssessmentFailureBundle) (EvidenceAssessmentFailureResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	program, ok := r.programs[bundle.TenantID][bundle.ProgramID]
	if !ok || !r.visibleLegalEntity(ctx, program.Program.TenantID, program.Program.LegalEntityID) {
		return EvidenceAssessmentFailureResult{}, ErrNotFound
	}
	if program.Program.Version != bundle.ExpectedVersion || bundle.ProgramEvent.AggregateVersion != bundle.ExpectedVersion+1 {
		return EvidenceAssessmentFailureResult{}, ErrVersionConflict
	}
	programEvent, err := normalizeMemoryProgramEvent(program, bundle.ProgramEvent)
	if err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	if err := applyProgramEventToAggregate(&program, programEvent); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	program.Program.Version = programEvent.AggregateVersion
	program.Program.UpdatedAt = programEvent.OccurredAt

	result := EvidenceAssessmentFailureResult{}
	for _, existing := range r.matters[bundle.TenantID] {
		if existing.Matter.TriggerKey == bundle.Matter.TriggerKey && existing.Matter.Status != MatterClosed && existing.Matter.Status != MatterCancelled &&
			existing.Matter.LegalEntityID == program.Program.LegalEntityID && matterLinkedToProgram(existing, bundle.ProgramID) {
			result.Matter = existing.Matter
			break
		}
	}
	if result.Matter.ID == "" {
		if bundle.Matter.LegalEntityID == "" || bundle.Matter.LegalEntityID != program.Program.LegalEntityID || bundle.Link.ProgramID != bundle.ProgramID {
			return EvidenceAssessmentFailureResult{}, ErrNotFound
		}
		matterAggregate := MatterAggregate{Matter: bundle.Matter, Closure: ClosureAssessment{Ready: false}}
		if err := applyMatterEventToAggregate(&matterAggregate, bundle.LinkEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		matterAggregate.Matter.Version = bundle.LinkEvent.AggregateVersion
		matterAggregate.Matter.UpdatedAt = bundle.LinkEvent.OccurredAt
		result.Matter = matterAggregate.Matter
		result.MatterCreated = true
		if r.matters[bundle.TenantID] == nil {
			r.matters[bundle.TenantID] = map[string]MatterAggregate{}
			r.matterEvents[bundle.TenantID] = map[string][]Event{}
		}
		r.matters[bundle.TenantID][bundle.Matter.ID] = matterAggregate
		r.matterEvents[bundle.TenantID][bundle.Matter.ID] = []Event{bundle.MatterEvent, bundle.LinkEvent}
	}

	r.programs[bundle.TenantID][bundle.ProgramID] = program
	r.programEvents[bundle.TenantID][bundle.ProgramID] = append(r.programEvents[bundle.TenantID][bundle.ProgramID], programEvent)
	return result, nil
}

var _ EvidenceAssessmentFailureRepository = (*MemoryRepository)(nil)
