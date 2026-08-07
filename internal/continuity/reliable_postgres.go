//go:build postgres

package continuity

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReliablePostgresRepository keeps only a short-lived copy of a create result
// that PostgreSQL has already committed. It is never authoritative state: a
// successful reconstruction removes the fallback. Its sole purpose is to keep
// an immediate post-commit read failure from turning a successful create into
// a false command failure.
type ReliablePostgresRepository struct {
	*PostgresRepository
	mu              sync.Mutex
	createdPrograms map[string]Program
	createdMatters  map[string]MatterAggregate
}

func NewReliablePostgresRepository(pool *pgxpool.Pool) *ReliablePostgresRepository {
	return &ReliablePostgresRepository{
		PostgresRepository: NewPostgresRepository(pool),
		createdPrograms:    map[string]Program{},
		createdMatters:     map[string]MatterAggregate{},
	}
}

func reliableKey(tenant, id string) string { return tenant + "\x00" + id }

func (r *ReliablePostgresRepository) CreateProgram(ctx context.Context, program Program, event Event) (Program, error) {
	created, err := r.PostgresRepository.CreateProgram(ctx, program, event)
	if err != nil {
		return Program{}, err
	}
	r.mu.Lock()
	r.createdPrograms[reliableKey(program.TenantID, program.ID)] = created
	r.mu.Unlock()
	return created, nil
}

func (r *ReliablePostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	value, err := r.PostgresRepository.GetProgram(ctx, tenant, id)
	key := reliableKey(tenant, id)
	if err == nil {
		r.mu.Lock()
		delete(r.createdPrograms, key)
		r.mu.Unlock()
		return value, nil
	}
	r.mu.Lock()
	created, ok := r.createdPrograms[key]
	if ok {
		delete(r.createdPrograms, key)
	}
	r.mu.Unlock()
	if ok {
		return decorateProgram(ProgramAggregate{Program: created}), nil
	}
	return ProgramAggregate{}, err
}

func (r *ReliablePostgresRepository) CreateMatter(ctx context.Context, matter Matter, event Event) (Matter, error) {
	created, err := r.PostgresRepository.CreateMatter(ctx, matter, event)
	if err != nil {
		return Matter{}, err
	}
	r.mu.Lock()
	r.createdMatters[reliableKey(matter.TenantID, matter.ID)] = MatterAggregate{Matter: created}
	r.mu.Unlock()
	return created, nil
}

func (r *ReliablePostgresRepository) CreateMatterWithLink(ctx context.Context, bundle MatterLinkBundle) (Matter, error) {
	created, err := r.PostgresRepository.CreateMatterWithLink(ctx, bundle)
	if err != nil {
		return Matter{}, err
	}
	aggregate := MatterAggregate{Matter: created, Links: []MatterLink{bundle.Link}}
	r.mu.Lock()
	r.createdMatters[reliableKey(bundle.Matter.TenantID, bundle.Matter.ID)] = aggregate
	r.mu.Unlock()
	return created, nil
}

func (r *ReliablePostgresRepository) GetMatter(ctx context.Context, tenant, id string) (MatterAggregate, error) {
	value, err := r.PostgresRepository.GetMatter(ctx, tenant, id)
	key := reliableKey(tenant, id)
	if err == nil {
		r.mu.Lock()
		delete(r.createdMatters, key)
		r.mu.Unlock()
		return value, nil
	}
	r.mu.Lock()
	created, ok := r.createdMatters[key]
	if ok {
		delete(r.createdMatters, key)
	}
	r.mu.Unlock()
	if ok {
		created.Closure = assessClosure(created)
		return decorateMatter(created), nil
	}
	return MatterAggregate{}, err
}
