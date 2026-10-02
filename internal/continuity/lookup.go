package continuity

import (
	"context"
	"encoding/json"
	"strings"
)

type programCodeRepository interface {
	ProgramByCode(context.Context, string, string) (ProgramAggregate, error)
}

type matterTriggerLookupRepository interface {
	MatterAggregateByTriggerKey(context.Context, string, string) (MatterAggregate, error)
}

type openMonitoringMatterRepository interface {
	OpenMonitoringMatter(context.Context, string, string, string) (MatterAggregate, error)
}

func (s *Service) ProgramByCode(ctx context.Context, tenant, code string) (ProgramAggregate, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(code) == "" {
		return ProgramAggregate{}, ErrNotFound
	}
	if repo, ok := s.repo.(programCodeRepository); ok {
		return repo.ProgramByCode(ctx, tenant, strings.ToUpper(strings.TrimSpace(code)))
	}
	values, err := s.repo.ListPrograms(ctx, tenant, 200)
	if err != nil {
		return ProgramAggregate{}, err
	}
	for _, value := range values {
		if strings.EqualFold(value.Program.Code, code) {
			return value, nil
		}
	}
	return ProgramAggregate{}, ErrNotFound
}

func (s *Service) OpenMonitoringMatter(ctx context.Context, tenant, programID, checkID string) (MatterAggregate, error) {
	tenant = strings.TrimSpace(tenant)
	programID = strings.TrimSpace(programID)
	checkID = strings.TrimSpace(checkID)
	if tenant == "" || programID == "" || checkID == "" {
		return MatterAggregate{}, ErrNotFound
	}
	repo, ok := s.repo.(openMonitoringMatterRepository)
	if !ok {
		return MatterAggregate{}, ErrNotFound
	}
	return repo.OpenMonitoringMatter(ctx, tenant, programID, checkID)
}

func (s *Service) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(triggerKey) == "" {
		return MatterAggregate{}, ErrNotFound
	}
	if repo, ok := s.repo.(matterTriggerLookupRepository); ok {
		return repo.MatterAggregateByTriggerKey(ctx, tenant, triggerKey)
	}
	matter, err := s.repo.MatterByTriggerKey(ctx, tenant, triggerKey)
	if err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, tenant, matter.ID)
}

func (r *MemoryRepository) ProgramByCode(ctx context.Context, tenant, code string) (ProgramAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, aggregate := range r.programs[tenant] {
		if strings.EqualFold(aggregate.Program.Code, code) && r.visibleLegalEntity(ctx, aggregate.Program.TenantID, aggregate.Program.LegalEntityID) {
			return decorateProgram(cloneProgramAggregate(aggregate)), nil
		}
	}
	return ProgramAggregate{}, ErrNotFound
}

func (r *MemoryRepository) OpenMonitoringMatter(ctx context.Context, tenant, programID, checkID string) (MatterAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selected MatterAggregate
	found := false
	for _, aggregate := range r.matters[tenant] {
		matter := aggregate.Matter
		if matter.Status == MatterClosed || matter.Status == MatterCancelled ||
			!strings.EqualFold(matter.TriggerType, "MONITORING_RESULT_ADVERSE") ||
			!r.visibleLegalEntity(ctx, matter.TenantID, matter.LegalEntityID) ||
			monitoringCheckIDFromMatter(matter) != checkID ||
			!matterLinkedToProgram(aggregate, programID) {
			continue
		}
		if !found || matter.UpdatedAt.After(selected.Matter.UpdatedAt) ||
			(matter.UpdatedAt.Equal(selected.Matter.UpdatedAt) && matter.ID > selected.Matter.ID) {
			selected = cloneMatterAggregate(aggregate)
			found = true
		}
	}
	if !found {
		return MatterAggregate{}, ErrNotFound
	}
	return decorateMatter(selected), nil
}

func monitoringCheckIDFromMatter(matter Matter) string {
	var facts struct {
		MonitoringCheckID string `json:"monitoring_check_id"`
	}
	if len(matter.KnownFacts) == 0 || json.Unmarshal(matter.KnownFacts, &facts) != nil {
		return ""
	}
	return strings.TrimSpace(facts.MonitoringCheckID)
}

func (r *MemoryRepository) MatterAggregateByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selected MatterAggregate
	found := false
	for _, aggregate := range r.matters[tenant] {
		if aggregate.Matter.TriggerKey != triggerKey || !r.visibleLegalEntity(ctx, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID) {
			continue
		}
		if !found || aggregate.Matter.UpdatedAt.After(selected.Matter.UpdatedAt) {
			selected = cloneMatterAggregate(aggregate)
			found = true
		}
	}
	if !found {
		return MatterAggregate{}, ErrNotFound
	}
	return decorateMatter(selected), nil
}
