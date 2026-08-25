package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	programs      map[string]map[string]ProgramAggregate
	matters       map[string]map[string]MatterAggregate
	programEvents map[string]map[string][]Event
	matterEvents  map[string]map[string][]Event
	triggers      map[string]map[string]Trigger
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		programs:      map[string]map[string]ProgramAggregate{},
		matters:       map[string]map[string]MatterAggregate{},
		programEvents: map[string]map[string][]Event{},
		matterEvents:  map[string]map[string][]Event{},
		triggers:      map[string]map[string]Trigger{},
	}
}

func (r *MemoryRepository) CreateProgram(_ context.Context, program Program, event Event) (Program, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.programs[program.TenantID] == nil {
		r.programs[program.TenantID] = map[string]ProgramAggregate{}
		r.programEvents[program.TenantID] = map[string][]Event{}
	}
	for _, existing := range r.programs[program.TenantID] {
		if existing.Program.Code == program.Code && existing.Program.Status != ProgramRetired {
			return Program{}, ErrDuplicate
		}
	}
	r.programs[program.TenantID][program.ID] = ProgramAggregate{Program: program}
	r.programEvents[program.TenantID][program.ID] = []Event{event}
	return program, nil
}

func (r *MemoryRepository) ListPrograms(_ context.Context, tenant string, limit int) ([]ProgramAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]ProgramAggregate, 0, len(r.programs[tenant]))
	for _, aggregate := range r.programs[tenant] {
		values = append(values, decorateProgram(cloneProgramAggregate(aggregate)))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Program.UpdatedAt.Equal(values[j].Program.UpdatedAt) {
			return values[i].Program.ID < values[j].Program.ID
		}
		return values[i].Program.UpdatedAt.After(values[j].Program.UpdatedAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) GetProgram(_ context.Context, tenant, id string) (ProgramAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.programs[tenant][id]
	if !ok {
		return ProgramAggregate{}, ErrNotFound
	}
	return decorateProgram(cloneProgramAggregate(aggregate)), nil
}

func (r *MemoryRepository) ApplyProgramEvent(_ context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	aggregate, ok := r.programs[tenant][id]
	if !ok {
		return 0, ErrNotFound
	}
	if aggregate.Program.Version != expected {
		return 0, ErrVersionConflict
	}
	if event.AggregateVersion != expected+1 {
		return 0, ErrVersionConflict
	}
	var err error
	event, err = normalizeMemoryProgramEvent(aggregate, event)
	if err != nil {
		return 0, err
	}
	if err := applyProgramEventToAggregate(&aggregate, event); err != nil {
		return 0, err
	}
	aggregate.Program.Version = event.AggregateVersion
	aggregate.Program.UpdatedAt = event.OccurredAt
	r.programs[tenant][id] = aggregate
	r.programEvents[tenant][id] = append(r.programEvents[tenant][id], event)
	return event.AggregateVersion, nil
}

func normalizeMemoryProgramEvent(aggregate ProgramAggregate, event Event) (Event, error) {
	if event.Type != EventEvidenceAssessmentRecorded {
		return event, nil
	}
	var assessment EvidenceAssessment
	if err := json.Unmarshal(event.Payload, &assessment); err != nil {
		return Event{}, err
	}
	var contract *EvidenceContract
	for index := range aggregate.EvidenceContracts {
		if aggregate.EvidenceContracts[index].ID == assessment.ContractID {
			contract = &aggregate.EvidenceContracts[index]
			break
		}
	}
	if contract == nil {
		return Event{}, ErrNotFound
	}
	validUntil := boundedAssessmentValidity(assessment, *contract)
	if validUntil.IsZero() || !assessment.AssessedAt.Before(validUntil) {
		return Event{}, fmt.Errorf("valid_until must be after assessed_at and within the evidence contract freshness boundary")
	}
	assessment.ValidUntil = &validUntil
	payload, err := json.Marshal(assessment)
	if err != nil {
		return Event{}, err
	}
	event.Payload = payload
	return event, nil
}

func (r *MemoryRepository) RecordProgramTrigger(_ context.Context, trigger Trigger) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.programs[trigger.TenantID][trigger.ProgramID].Program.ID == "" {
		return false, ErrNotFound
	}
	if r.triggers[trigger.TenantID] == nil {
		r.triggers[trigger.TenantID] = map[string]Trigger{}
	}
	if _, ok := r.triggers[trigger.TenantID][trigger.DedupeKey]; ok {
		return false, nil
	}
	r.triggers[trigger.TenantID][trigger.DedupeKey] = trigger
	return true, nil
}

func (r *MemoryRepository) ProgramEvents(_ context.Context, tenant, id string, until *time.Time) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values, ok := r.programEvents[tenant][id]
	if !ok {
		return nil, ErrNotFound
	}
	return filterEvents(values, until), nil
}

func (r *MemoryRepository) CreateMatter(_ context.Context, matter Matter, event Event) (Matter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.matters[matter.TenantID] == nil {
		r.matters[matter.TenantID] = map[string]MatterAggregate{}
		r.matterEvents[matter.TenantID] = map[string][]Event{}
	}
	if matter.TriggerKey != "" {
		for _, existing := range r.matters[matter.TenantID] {
			if existing.Matter.TriggerKey == matter.TriggerKey && existing.Matter.Status != MatterClosed && existing.Matter.Status != MatterCancelled {
				return Matter{}, ErrDuplicate
			}
		}
	}
	r.matters[matter.TenantID][matter.ID] = MatterAggregate{Matter: matter, Closure: ClosureAssessment{Ready: false}}
	r.matterEvents[matter.TenantID][matter.ID] = []Event{event}
	return matter, nil
}

func (r *MemoryRepository) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	actor, enforceVisibility := identity.FromContext(ctx)
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]MatterAggregate, 0, len(r.matters[tenant]))
	for _, aggregate := range r.matters[tenant] {
		if enforceVisibility && (aggregate.Matter.TenantID != actor.TenantID || !MatterVisibleTo(aggregate.Matter, actor.PrincipalID)) {
			continue
		}
		if status == "OPEN" && (aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled) {
			continue
		}
		if status != "" && status != "OPEN" && string(aggregate.Matter.Status) != status {
			continue
		}
		value := cloneMatterAggregate(aggregate)
		value.Closure = assessClosure(value)
		values = append(values, decorateMatter(value))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Matter.Priority == values[j].Matter.Priority {
			return values[i].Matter.UpdatedAt.After(values[j].Matter.UpdatedAt)
		}
		return values[i].Matter.Priority > values[j].Matter.Priority
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) GetMatter(_ context.Context, tenant, id string) (MatterAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.matters[tenant][id]
	if !ok {
		return MatterAggregate{}, ErrNotFound
	}
	value := cloneMatterAggregate(aggregate)
	value.Closure = assessClosure(value)
	return decorateMatter(value), nil
}

func (r *MemoryRepository) ApplyMatterEvent(_ context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	aggregate, ok := r.matters[tenant][id]
	if !ok {
		return 0, ErrNotFound
	}
	if aggregate.Matter.Version != expected || event.AggregateVersion != expected+1 {
		return 0, ErrVersionConflict
	}
	if err := applyMatterEventToAggregate(&aggregate, event); err != nil {
		return 0, err
	}
	aggregate.Matter.Version = event.AggregateVersion
	aggregate.Matter.UpdatedAt = event.OccurredAt
	aggregate.Closure = assessClosure(aggregate)
	r.matters[tenant][id] = aggregate
	r.matterEvents[tenant][id] = append(r.matterEvents[tenant][id], event)
	return event.AggregateVersion, nil
}

func (r *MemoryRepository) MatterByTriggerKey(_ context.Context, tenant, triggerKey string) (Matter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, aggregate := range r.matters[tenant] {
		if aggregate.Matter.TriggerKey == triggerKey && aggregate.Matter.Status != MatterClosed && aggregate.Matter.Status != MatterCancelled {
			return aggregate.Matter, nil
		}
	}
	return Matter{}, ErrNotFound
}

func (r *MemoryRepository) MatterEvents(_ context.Context, tenant, id string, until *time.Time) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values, ok := r.matterEvents[tenant][id]
	if !ok {
		return nil, ErrNotFound
	}
	return filterEvents(values, until), nil
}

func (r *MemoryRepository) OpenMatterCount(_ context.Context, tenant, programID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, aggregate := range r.matters[tenant] {
		if aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled {
			continue
		}
		for _, link := range aggregate.Links {
			if link.ProgramID == programID {
				count++
				break
			}
		}
	}
	return count, nil
}

func (r *MemoryRepository) LinkedProgramIDs(_ context.Context, tenant, matterID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.matters[tenant][matterID]
	if !ok {
		return nil, ErrNotFound
	}
	seen := map[string]bool{}
	values := []string{}
	for _, link := range aggregate.Links {
		if link.ProgramID != "" && !seen[link.ProgramID] {
			seen[link.ProgramID] = true
			values = append(values, link.ProgramID)
		}
	}
	return values, nil
}

func applyProgramEventToAggregate(aggregate *ProgramAggregate, event Event) error {
	switch event.Type {
	case EventProgramStatusChanged:
		var value Program
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Program = value
	case EventRequirementAdded:
		var value Requirement
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Requirements = append(aggregate.Requirements, value)
	case EventApplicabilityDetermined:
		var value Applicability
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Applicability = append(aggregate.Applicability, value)
	case EventControlObjectiveAdded:
		var value ControlObjective
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.ControlObjectives = append(aggregate.ControlObjectives, value)
	case EventControlImplementationAdded:
		var value ControlImplementation
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.ControlImplementations = append(aggregate.ControlImplementations, value)
	case EventRequirementControlLinked:
		var value RequirementControlLink
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.RequirementControlLinks = append(aggregate.RequirementControlLinks, value)
	case EventEvidenceContractAdded:
		var value EvidenceContract
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.EvidenceContracts = append(aggregate.EvidenceContracts, value)
	case EventEvidenceAssessmentRecorded:
		var value EvidenceAssessment
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.EvidenceAssessments = append(aggregate.EvidenceAssessments, value)
	case EventProgramStateUpdated:
		var value ProgramStateSnapshot
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.CurrentState = &value
	case EventProgramTriggerRecorded:
		var value Trigger
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Triggers = append(aggregate.Triggers, value)
	default:
		return ErrInvalidState
	}
	return nil
}

func applyMatterEventToAggregate(aggregate *MatterAggregate, event Event) error {
	switch event.Type {
	case EventMatterLinked:
		var value MatterLink
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Links = append(aggregate.Links, value)
	case EventMatterStateChanged:
		var value Matter
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Matter = value
	case EventMatterDetailsUpdated:
		var value matterDetailsUpdatedEvent
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Matter = value.Matter
	case EventMatterContextChanged:
		var value matterContextChangedEvent
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Matter = value.Matter
	case EventMatterOwnerChanged:
		var value matterOwnerChangedEvent
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Matter = value.Matter
	case EventDecisionAdded:
		var value Decision
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		setDecisionActor(&value, value.Status, event.ActorID)
		aggregate.Decisions = append(aggregate.Decisions, value)
	case EventActionAdded:
		var value Action
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Actions = append(aggregate.Actions, value)
	case EventActionStateChanged:
		var value Action
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Actions = upsertAction(aggregate.Actions, value)
	case EventActionUpdated:
		var value actionUpdatedEvent
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Actions = upsertAction(aggregate.Actions, value.Action)
	case EventActionAssigned:
		var value actionAssignedEvent
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.Actions = upsertAction(aggregate.Actions, value.Action)
	case EventVerificationContractAdded:
		var value VerificationContract
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.VerificationContracts = append(aggregate.VerificationContracts, value)
	case EventVerificationResultRecorded:
		var value VerificationResult
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		aggregate.VerificationResults = append(aggregate.VerificationResults, value)
	case EventResponsePackageAdded:
		var value ResponsePackage
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		setResponseActor(&value, value.Status, event.ActorID)
		aggregate.ResponsePackages = append(aggregate.ResponsePackages, value)
	case EventResponsePackageStateChanged:
		var value ResponsePackage
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return err
		}
		setResponseActor(&value, value.Status, event.ActorID)
		aggregate.ResponsePackages = upsertResponsePackage(aggregate.ResponsePackages, value)
	default:
		return ErrInvalidState
	}
	return nil
}

func filterEvents(values []Event, until *time.Time) []Event {
	result := make([]Event, 0, len(values))
	for _, event := range values {
		if until == nil || !event.OccurredAt.After(*until) {
			result = append(result, event)
		}
	}
	return result
}

func cloneProgramAggregate(value ProgramAggregate) ProgramAggregate {
	encoded, _ := json.Marshal(value)
	var cloned ProgramAggregate
	_ = json.Unmarshal(encoded, &cloned)
	return cloned
}

func cloneMatterAggregate(value MatterAggregate) MatterAggregate {
	encoded, _ := json.Marshal(value)
	var cloned MatterAggregate
	_ = json.Unmarshal(encoded, &cloned)
	return cloned
}

func (r *MemoryRepository) triggerTypeForProgram(tenant, programID string) string {
	for _, trigger := range r.triggers[tenant] {
		if trigger.ProgramID == programID {
			return strings.ToUpper(trigger.Type)
		}
	}
	return ""
}
