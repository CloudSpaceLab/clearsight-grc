//go:build postgres

package continuity

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReliablePostgresRepository keeps only a short-lived copy of a Matter create result
// that PostgreSQL has already committed. It is never authoritative state: a
// successful normalized current read removes the fallback. Its sole purpose is
// to keep an immediate post-commit read failure from turning a successful
// create into a false command failure.
type ReliablePostgresRepository struct {
	*CurrentPostgresRepository
	mu             sync.Mutex
	createdMatters map[string]MatterAggregate
}

func NewReliablePostgresRepository(pool *pgxpool.Pool) *ReliablePostgresRepository {
	return &ReliablePostgresRepository{
		CurrentPostgresRepository: NewCurrentPostgresRepository(pool),
		createdMatters:            map[string]MatterAggregate{},
	}
}

func reliableKey(tenant, id string) string { return tenant + "\x00" + id }

func (r *ReliablePostgresRepository) CreateMatter(ctx context.Context, matter Matter, event Event) (Matter, error) {
	created, err := r.CurrentPostgresRepository.CreateMatter(ctx, matter, event)
	if err != nil {
		return Matter{}, err
	}
	r.mu.Lock()
	r.createdMatters[reliableKey(matter.TenantID, matter.ID)] = MatterAggregate{Matter: created}
	r.mu.Unlock()
	return created, nil
}

func (r *ReliablePostgresRepository) CreateMatterWithLink(ctx context.Context, bundle MatterLinkBundle) (Matter, error) {
	created, err := r.CurrentPostgresRepository.CreateMatterWithLink(ctx, bundle)
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
	value, err := r.CurrentPostgresRepository.GetMatter(ctx, tenant, id)
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
		if !visibleToActorLegalEntity(ctx, created.Matter.TenantID, created.Matter.LegalEntityID) {
			return MatterAggregate{}, ErrNotFound
		}
		created.Closure = assessClosure(created)
		return decorateMatter(created), nil
	}
	return MatterAggregate{}, err
}
