package rcsa

import (
	"context"
	"sort"
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

func (r *MemoryRepository) UpdateCycle(ctx context.Context, scope Scope, next Cycle, expected int64, event Event) (Cycle, error) {
	if err := ctx.Err(); err != nil {
		return Cycle{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cycleKey(scope.TenantID, scope.LegalEntityID, next.ID)
	current, ok := r.cycles[key]
	if !ok {
		return Cycle{}, ErrNotFound
	}
	if current.Cycle.Version != expected {
		return Cycle{}, ErrVersionConflict
	}
	if next.TenantID != current.Cycle.TenantID || next.LegalEntityID != current.Cycle.LegalEntityID ||
		next.ID != current.Cycle.ID || next.Code != current.Cycle.Code || next.Name != current.Cycle.Name ||
		next.TriggerKind != current.Cycle.TriggerKind || next.FirstLineOwnerID != current.Cycle.FirstLineOwnerID ||
		next.PopulationChecksum != current.Cycle.PopulationChecksum || !next.CreatedAt.Equal(current.Cycle.CreatedAt) ||
		next.Version != expected+1 || event.CycleID != next.ID || event.CycleVersion != next.Version {
		return Cycle{}, ErrInvalid
	}
	current.Cycle = next
	r.cycles[key] = cloneAggregate(current)
	return next, nil
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


func (r *MemoryRepository) ListCycles(ctx context.Context, scope Scope, filter CycleFilter) (CyclePage, error) {
	if err := ctx.Err(); err != nil {
		return CyclePage{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return CyclePage{}, err
	}
	cursor, err := decodeCycleCursor(filter.Cursor)
	if err != nil {
		return CyclePage{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]CycleSummary, 0, len(r.cycles))
	for _, aggregate := range r.cycles {
		if aggregate.Cycle.TenantID != scope.TenantID || aggregate.Cycle.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if filter.Status != "" && aggregate.Cycle.Status != filter.Status {
			continue
		}
		item := CycleSummary{
			Cycle:        aggregate.Cycle,
			RiskCount:    len(aggregate.Risks),
			ControlCount: len(aggregate.Controls),
		}
		if cycleAfterCursor(item, cursor) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].Cycle.UpdatedAt.Equal(items[j].Cycle.UpdatedAt) {
			return items[i].Cycle.UpdatedAt.After(items[j].Cycle.UpdatedAt)
		}
		return items[i].Cycle.ID > items[j].Cycle.ID
	})

	page := CyclePage{Items: items}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		page.NextCursor, err = encodeCycleCursor(page.Items[len(page.Items)-1])
		if err != nil {
			return CyclePage{}, err
		}
	}
	return page, nil
}

var _ CycleListRepository = (*MemoryRepository)(nil)
