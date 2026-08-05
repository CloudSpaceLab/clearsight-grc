package continuity

import (
	"context"
	"strings"
)

type programCodeRepository interface {
	ProgramByCode(context.Context, string, string) (ProgramAggregate, error)
}

type matterTriggerLookupRepository interface {
	MatterAggregateByTriggerKey(context.Context, string, string) (MatterAggregate, error)
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

func (r *MemoryRepository) ProgramByCode(_ context.Context, tenant, code string) (ProgramAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, aggregate := range r.programs[tenant] {
		if strings.EqualFold(aggregate.Program.Code, code) {
			return decorateProgram(cloneProgramAggregate(aggregate)), nil
		}
	}
	return ProgramAggregate{}, ErrNotFound
}

func (r *MemoryRepository) MatterAggregateByTriggerKey(_ context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selected MatterAggregate
	found := false
	for _, aggregate := range r.matters[tenant] {
		if aggregate.Matter.TriggerKey != triggerKey {
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
