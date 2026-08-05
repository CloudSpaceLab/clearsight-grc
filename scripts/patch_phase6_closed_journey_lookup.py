from pathlib import Path

p = Path('internal/continuity/lookup.go')
s = p.read_text()
s = s.replace(
'''type programCodeRepository interface {
	ProgramByCode(context.Context, string, string) (ProgramAggregate, error)
}
''',
'''type programCodeRepository interface {
	ProgramByCode(context.Context, string, string) (ProgramAggregate, error)
}

type matterTriggerLookupRepository interface {
	MatterAggregateByTriggerKey(context.Context, string, string) (MatterAggregate, error)
}
''', 1)
s = s.replace(
'''func (s *Service) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(triggerKey) == "" {
		return MatterAggregate{}, ErrNotFound
	}
	matter, err := s.repo.MatterByTriggerKey(ctx, tenant, triggerKey)
	if err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, tenant, matter.ID)
}
''',
'''func (s *Service) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
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
''', 1)
s += '''
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
'''
p.write_text(s)

p = Path('internal/continuity/lookup_postgres.go')
s = p.read_text()
s = s.replace(
'''func (r *PostgresRepository) ProgramByCode(ctx context.Context, tenant, code string) (ProgramAggregate, error) {''',
'''func (r *PostgresRepository) MatterAggregateByTriggerKey(ctx context.Context, tenant, triggerKey string) (MatterAggregate, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 ORDER BY m.updated_at DESC,m.id DESC LIMIT 1`, tenant, triggerKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterAggregate{}, ErrNotFound
	}
	if err != nil {
		return MatterAggregate{}, err
	}
	return r.GetMatter(ctx, tenant, id)
}

func (r *PostgresRepository) ProgramByCode(ctx context.Context, tenant, code string) (ProgramAggregate, error) {''', 1)
s = s.replace(
'''var _ programCodeRepository = (*PostgresRepository)(nil)
''',
'''var _ programCodeRepository = (*PostgresRepository)(nil)
var _ matterTriggerLookupRepository = (*PostgresRepository)(nil)
''', 1)
p.write_text(s)
