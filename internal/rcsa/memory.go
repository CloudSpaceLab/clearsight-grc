package rcsa

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	cycles        map[string]Cycle
	codes         map[string]string
	items         map[string][]Item
	controls      map[string][]ItemControl
	distributions map[string]DistributionLink
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		cycles: map[string]Cycle{}, codes: map[string]string{}, items: map[string][]Item{},
		controls: map[string][]ItemControl{}, distributions: map[string]DistributionLink{},
	}
}

func (r *MemoryRepository) CreateCycle(ctx context.Context, cycle Cycle, items []Item, controls []ItemControl) (Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return Aggregate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cycleKey(cycle.TenantID, cycle.LegalEntityID, cycle.ID)
	codeKey := cycleCodeKey(cycle.TenantID, cycle.LegalEntityID, cycle.Code)
	if _, exists := r.cycles[key]; exists {
		return Aggregate{}, ErrDuplicate
	}
	if _, exists := r.codes[codeKey]; exists {
		return Aggregate{}, ErrDuplicate
	}
	itemIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.CycleID != cycle.ID || item.TenantID != cycle.TenantID || item.LegalEntityID != cycle.LegalEntityID ||
			item.RiskID == "" || item.RiskVersion < 1 || item.RespondentPrincipalID == "" {
			return Aggregate{}, ErrInvalid
		}
		if _, exists := itemIDs[item.ID]; exists {
			return Aggregate{}, ErrDuplicate
		}
		itemIDs[item.ID] = struct{}{}
	}
	for _, control := range controls {
		if _, exists := itemIDs[control.ItemID]; !exists || control.RiskID == "" || control.ControlLinkID == "" {
			return Aggregate{}, ErrInvalid
		}
	}
	r.cycles[key] = cycle
	r.codes[codeKey] = cycle.ID
	r.items[key] = append([]Item(nil), items...)
	r.controls[key] = append([]ItemControl(nil), controls...)
	return aggregateMemory(cycle, items, controls, r.distributions), nil
}

func (r *MemoryRepository) GetCycle(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return Aggregate{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Aggregate{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := cycleKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	cycle, ok := r.cycles[key]
	if !ok {
		return Aggregate{}, ErrNotFound
	}
	return aggregateMemory(cycle, r.items[key], r.controls[key], r.distributions), nil
}

func (r *MemoryRepository) AttachDistribution(ctx context.Context, scope Scope, link DistributionLink) (DistributionLink, error) {
	if err := ctx.Err(); err != nil {
		return DistributionLink{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return DistributionLink{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var found bool
	for key, items := range r.items {
		cycle := r.cycles[key]
		if cycle.TenantID != scope.TenantID || cycle.LegalEntityID != scope.LegalEntityID {
			continue
		}
		for _, item := range items {
			if item.ID == link.ItemID {
				found = true
				break
			}
		}
	}
	if !found {
		return DistributionLink{}, ErrNotFound
	}
	if existing, exists := r.distributions[link.ItemID]; exists {
		if existing.DistributionID == link.DistributionID {
			return existing, nil
		}
		return DistributionLink{}, ErrConflict
	}
	r.distributions[link.ItemID] = link
	return link, nil
}

func aggregateMemory(cycle Cycle, items []Item, controls []ItemControl, distributions map[string]DistributionLink) Aggregate {
	value := Aggregate{
		Cycle: cycle, Items: append([]Item(nil), items...),
		Controls: map[string][]ItemControl{}, Distributions: map[string]DistributionLink{},
	}
	sort.Slice(value.Items, func(i, j int) bool {
		if value.Items[i].RiskCode != value.Items[j].RiskCode {
			return value.Items[i].RiskCode < value.Items[j].RiskCode
		}
		return value.Items[i].ID < value.Items[j].ID
	})
	for _, control := range controls {
		value.Controls[control.ItemID] = append(value.Controls[control.ItemID], control)
	}
	for _, item := range items {
		if link, ok := distributions[item.ID]; ok {
			value.Distributions[item.ID] = link
		}
	}
	return value
}

func cycleKey(tenant, entity, id string) string { return tenant + "\x00" + entity + "\x00" + id }
func cycleCodeKey(tenant, entity, code string) string {
	return tenant + "\x00" + entity + "\x00" + strings.ToUpper(strings.TrimSpace(code))
}

var _ Repository = (*MemoryRepository)(nil)
