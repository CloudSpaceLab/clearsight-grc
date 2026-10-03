package rcsa

import (
	"context"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	cycles   map[string]Aggregate
	codeKeys map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{cycles: map[string]Aggregate{}, codeKeys: map[string]string{}}
}

func (r *MemoryRepository) Create(ctx context.Context, cycle Cycle, risks []RiskSnapshot, controls []ControlSnapshot, _ Event) (Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return Aggregate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cycleKey(cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	codeKey := cycleCodeKey(cycle.TenantID, cycle.LegalEntityID, cycle.Code)
	if _, ok := r.cycles[key]; ok {
		return Aggregate{}, ErrDuplicate
	}
	if _, ok := r.codeKeys[codeKey]; ok {
		return Aggregate{}, ErrDuplicate
	}
	value := cloneAggregate(Aggregate{Cycle: cycle, Risks: risks, Controls: controls})
	r.cycles[key] = value
	r.codeKeys[codeKey] = cycle.ID
	return cloneAggregate(value), nil
}

func (r *MemoryRepository) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return Aggregate{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.cycles[cycleKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))]
	if !ok {
		return Aggregate{}, ErrNotFound
	}
	return cloneAggregate(value), nil
}

func (r *MemoryRepository) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	tenant = strings.TrimSpace(tenant)
	id = strings.TrimSpace(id)
	if tenant == "" || id == "" {
		return "", ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.cycles {
		if value.Cycle.TenantID == tenant && value.Cycle.ID == id {
			return value.Cycle.LegalEntityID, nil
		}
	}
	return "", ErrNotFound
}

func cycleKey(tenant, entity, id string) string {
	return tenant + "\x00" + entity + "\x00" + id
}

func cycleCodeKey(tenant, entity, code string) string {
	return tenant + "\x00" + entity + "\x00" + strings.ToUpper(strings.TrimSpace(code))
}

func cloneAggregate(value Aggregate) Aggregate {
	cloned := value
	cloned.Risks = append([]RiskSnapshot(nil), value.Risks...)
	cloned.Controls = append([]ControlSnapshot(nil), value.Controls...)
	return cloned
}
