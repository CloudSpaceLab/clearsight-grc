//go:build !postgres

package continuity

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
)

func (r *MemoryRepository) remediationMaps() {
	if r.matterFormBindings == nil {
		r.matterFormBindings = map[string]MatterFormRemediationBinding{}
	}
	if r.matterFormApplications == nil {
		r.matterFormApplications = map[string]MatterFormApplication{}
	}
}

func (r *MemoryRepository) CreateMatterFormBinding(ctx context.Context, binding MatterFormRemediationBinding) (MatterFormRemediationBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remediationMaps()
	if binding.SubjectType != "MATTER" || binding.SubjectID != binding.MatterID || binding.Status != MatterFormBindingActive || binding.EffectiveFrom.IsZero() || strings.TrimSpace(binding.Purpose) == "" || binding.AudienceClass != "EXTERNAL" || strings.TrimSpace(binding.ResponderClass) == "" {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	aggregate, ok := r.matters[binding.TenantID][binding.MatterID]
	if !ok || !r.visibleLegalEntity(ctx, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID) {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	if aggregate.Matter.Version != binding.MatterVersionAtBinding {
		return MatterFormRemediationBinding{}, ErrVersionConflict
	}
	for _, existing := range r.matterFormBindings {
		if existing.TenantID != binding.TenantID || existing.MatterID != binding.MatterID {
			continue
		}
		for _, left := range existing.Mappings {
			for _, right := range binding.Mappings {
				if strings.EqualFold(left.MissingItem, right.MissingItem) {
					return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
				}
			}
		}
	}
	r.matterFormBindings[binding.ID] = cloneMatterFormBinding(binding)
	return cloneMatterFormBinding(binding), nil
}

func (r *MemoryRepository) GetMatterFormBinding(ctx context.Context, tenant, matterID, bindingID string) (MatterFormRemediationBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	binding, ok := r.matterFormBindings[bindingID]
	if !ok || binding.TenantID != tenant || binding.MatterID != matterID {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	aggregate, ok := r.matters[tenant][matterID]
	if !ok || !r.visibleLegalEntity(ctx, tenant, aggregate.Matter.LegalEntityID) {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	return cloneMatterFormBinding(binding), nil
}

func (r *MemoryRepository) ListMatterFormBindings(ctx context.Context, tenant, matterID string, limit int) ([]MatterFormRemediationBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	aggregate, ok := r.matters[tenant][matterID]
	if !ok || !r.visibleLegalEntity(ctx, tenant, aggregate.Matter.LegalEntityID) {
		return nil, ErrNotFound
	}
	values := []MatterFormRemediationBinding{}
	for _, binding := range r.matterFormBindings {
		if binding.TenantID == tenant && binding.MatterID == matterID {
			values = append(values, cloneMatterFormBinding(binding))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].CreatedAt.After(values[j].CreatedAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) GetMatterFormApplication(_ context.Context, tenant, bindingID, responseRevisionID string) (MatterFormApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.matterFormApplications {
		if value.TenantID == tenant && value.BindingID == bindingID && (responseRevisionID == "" || value.ResponseRevisionID == responseRevisionID) {
			return cloneMatterFormApplication(value), nil
		}
	}
	return MatterFormApplication{}, ErrNotFound
}

func (r *MemoryRepository) ApplyMatterFormApplication(ctx context.Context, command MatterFormApplicationCommand) (MatterAggregate, MatterFormApplication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remediationMaps()
	for _, value := range r.matterFormApplications {
		if value.TenantID == command.Binding.TenantID && value.BindingID == command.Binding.ID && value.ResponseRevisionID == command.ResponseRevisionID {
			aggregate := cloneMatterAggregate(r.matters[value.TenantID][value.MatterID])
			aggregate.Closure = assessClosure(aggregate)
			return decorateMatter(aggregate), cloneMatterFormApplication(value), nil
		}
	}
	aggregate, ok := r.matters[command.Binding.TenantID][command.Binding.MatterID]
	if !ok || !r.visibleLegalEntity(ctx, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID) {
		return MatterAggregate{}, MatterFormApplication{}, ErrNotFound
	}
	if aggregate.Matter.Version != command.ExpectedMatterVersion || command.Binding.Version < 1 {
		return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
	}
	updated, applied, err := applyMatterFormAnswers(aggregate.Matter, command.Binding, command.Answers, command.ResponseRevisionID)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	updated.UpdatedAt = command.AppliedAt
	updated.Version = aggregate.Matter.Version + 1
	application := MatterFormApplication{ID: command.ApplicationID, TenantID: command.Binding.TenantID, LegalEntityID: command.Binding.LegalEntityID, BindingID: command.Binding.ID, BindingVersion: command.Binding.Version, MatterID: command.Binding.MatterID, MatterVersion: updated.Version, DistributionID: command.DistributionID, ResponseRevisionID: command.ResponseRevisionID, ResponseRevision: command.ResponseRevision, SubmissionID: command.SubmissionID, VerificationContractID: command.Binding.VerificationContractID, AppliedFieldIDs: applied, AppliedBy: command.ActorID, AppliedAt: command.AppliedAt}
	payload, err := json.Marshal(matterFormResponseAppliedEvent{Matter: updated, Application: application, Rationale: command.Rationale})
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	event := Event{ID: command.EventID, TenantID: command.Binding.TenantID, AggregateType: "MATTER", AggregateID: command.Binding.MatterID, AggregateVersion: updated.Version, Type: EventMatterFormApplied, Payload: payload, ActorType: actorFor(command.ActorID), ActorID: command.ActorID, OccurredAt: command.AppliedAt}
	if err := applyMatterEventToAggregate(&aggregate, event); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	aggregate.Matter.Version, aggregate.Matter.UpdatedAt = event.AggregateVersion, event.OccurredAt
	aggregate.Closure = assessClosure(aggregate)
	r.matters[command.Binding.TenantID][command.Binding.MatterID] = aggregate
	r.matterEvents[command.Binding.TenantID][command.Binding.MatterID] = append(r.matterEvents[command.Binding.TenantID][command.Binding.MatterID], event)
	r.matterFormApplications[application.ID] = application
	return decorateMatter(cloneMatterAggregate(aggregate)), cloneMatterFormApplication(application), nil
}

func cloneMatterFormBinding(value MatterFormRemediationBinding) MatterFormRemediationBinding {
	value.Mappings = append([]MatterFormFieldMapping(nil), value.Mappings...)
	value.MinimumScore = cloneFloat(value.MinimumScore)
	value.MaximumAdverseScore = cloneFloat(value.MaximumAdverseScore)
	return value
}
func cloneMatterFormApplication(value MatterFormApplication) MatterFormApplication {
	value.AppliedFieldIDs = append([]string(nil), value.AppliedFieldIDs...)
	return value
}

var _ MatterFormRemediationRepository = (*MemoryRepository)(nil)
