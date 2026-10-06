package oploss

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu         sync.RWMutex
	losses     map[string]Loss
	codeKeys   map[string]string
	recoveries map[string][]Recovery
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		losses: map[string]Loss{}, codeKeys: map[string]string{}, recoveries: map[string][]Recovery{},
	}
}

func (r *MemoryRepository) Create(ctx context.Context, loss Loss, _ Event) (Loss, error) {
	if err := ctx.Err(); err != nil {
		return Loss{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := lossKey(loss.TenantID, loss.LegalEntityID, loss.ID)
	codeKey := lossCodeKey(loss.TenantID, loss.LegalEntityID, loss.Code)
	if _, ok := r.losses[key]; ok {
		return Loss{}, ErrDuplicate
	}
	if _, ok := r.codeKeys[codeKey]; ok {
		return Loss{}, ErrDuplicate
	}
	r.losses[key] = loss
	r.codeKeys[codeKey] = loss.ID
	return loss, nil
}

func (r *MemoryRepository) Update(ctx context.Context, scope Scope, next Loss, expected int64, _ Event) (Loss, error) {
	if err := ctx.Err(); err != nil {
		return Loss{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := lossKey(scope.TenantID, scope.LegalEntityID, next.ID)
	current, ok := r.losses[key]
	if !ok {
		return Loss{}, ErrNotFound
	}
	if current.Version != expected {
		return Loss{}, ErrVersionConflict
	}
	if next.ID != current.ID || next.Code != current.Code || next.TenantID != current.TenantID ||
		next.LegalEntityID != current.LegalEntityID || next.OwnerPrincipalID != current.OwnerPrincipalID ||
		!next.CreatedAt.Equal(current.CreatedAt) || next.Version != expected+1 {
		return Loss{}, ErrInvalid
	}
	r.losses[key] = next
	return next, nil
}

func (r *MemoryRepository) AddRecovery(ctx context.Context, scope Scope, id string, expected int64, recovery Recovery, event Event) (Loss, Recovery, error) {
	if err := ctx.Err(); err != nil {
		return Loss{}, Recovery{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := lossKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	current, ok := r.losses[key]
	if !ok {
		return Loss{}, Recovery{}, ErrNotFound
	}
	if current.Version != expected {
		return Loss{}, Recovery{}, ErrVersionConflict
	}
	if recovery.LossID != current.ID || recovery.LossVersion != expected+1 || recovery.Currency != current.Currency ||
		event.LossID != current.ID || event.LossVersion != expected+1 || event.Type != EventLossRecoveryRecorded {
		return Loss{}, Recovery{}, ErrInvalid
	}
	existing := r.recoveries[key]
	currentTotals, err := totals(current, existing)
	if err != nil {
		return Loss{}, Recovery{}, err
	}
	nextRecovered := currentTotals.RecoveredAmountMinor
	if recovery.Kind == RecoveryCash {
		nextRecovered += recovery.AmountMinor
	} else {
		nextRecovered -= recovery.AmountMinor
	}
	if nextRecovered < 0 || nextRecovered > current.GrossAmountMinor {
		return Loss{}, Recovery{}, ErrRecoveryLimit
	}
	current.Version++
	current.UpdatedAt = event.OccurredAt.UTC()
	r.losses[key] = current
	r.recoveries[key] = append(existing, recovery)
	return current, recovery, nil
}

func (r *MemoryRepository) Get(ctx context.Context, scope Scope, id string) (Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return Aggregate{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := lossKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	loss, ok := r.losses[key]
	if !ok {
		return Aggregate{}, ErrNotFound
	}
	recoveries := append([]Recovery(nil), r.recoveries[key]...)
	sort.Slice(recoveries, func(i, j int) bool {
		if recoveries[i].RecoveredAt.Equal(recoveries[j].RecoveredAt) {
			return recoveries[i].ID > recoveries[j].ID
		}
		return recoveries[i].RecoveredAt.After(recoveries[j].RecoveredAt)
	})
	calculated, err := totals(loss, recoveries)
	if err != nil {
		return Aggregate{}, err
	}
	return Aggregate{Loss: loss, Recoveries: recoveries, Totals: calculated}, nil
}

func (r *MemoryRepository) List(ctx context.Context, scope Scope, filter ListFilter) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	cursor, err := decodeListCursor(filter.Cursor)
	if err != nil {
		return Page{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]Summary, 0)
	organizationScopes := make(map[string]struct{}, len(filter.OrganizationScopeIDs))
	for _, id := range filter.OrganizationScopeIDs {
		organizationScopes[id] = struct{}{}
	}
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for key, loss := range r.losses {
		if loss.TenantID != scope.TenantID || loss.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if filter.Status != "" && loss.Status != filter.Status {
			continue
		}
		if filter.EventType != "" && loss.EventType != filter.EventType {
			continue
		}
		if filter.Currency != "" && loss.Currency != filter.Currency {
			continue
		}
		if filter.OrganizationScopeID != "" {
			if _, ok := organizationScopes[loss.OrganizationScopeID]; !ok {
				continue
			}
		}
		if filter.RiskID != "" && loss.RiskID != filter.RiskID {
			continue
		}
		if filter.MatterID != "" && loss.MatterID != filter.MatterID {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(strings.Join([]string{
			loss.Code, loss.Title, loss.Cause, loss.Description,
		}, " ")), search) {
			continue
		}
		if !cursor.UpdatedAt.IsZero() &&
			!(loss.UpdatedAt.Before(cursor.UpdatedAt) || (loss.UpdatedAt.Equal(cursor.UpdatedAt) && loss.ID < cursor.ID)) {
			continue
		}
		calculated, err := totals(loss, r.recoveries[key])
		if err != nil {
			return Page{}, err
		}
		if filter.RecoveryStatus != "" && calculated.RecoveryStatus != filter.RecoveryStatus {
			continue
		}
		values = append(values, Summary{Loss: loss, Totals: calculated})
	}
	sort.Slice(values, func(i, j int) bool {
		if !values[i].Loss.UpdatedAt.Equal(values[j].Loss.UpdatedAt) {
			return values[i].Loss.UpdatedAt.After(values[j].Loss.UpdatedAt)
		}
		return values[i].Loss.ID > values[j].Loss.ID
	})
	page := Page{Items: values, OrganizationScopeID: filter.OrganizationScopeID}
	if len(values) > filter.Limit {
		page.Items = values[:filter.Limit]
		page.NextCursor, err = encodeListCursor(page.Items[len(page.Items)-1].Loss)
		if err != nil {
			return Page{}, err
		}
	}
	return page, nil
}

func (r *MemoryRepository) ResolveLegalEntity(ctx context.Context, tenant, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, loss := range r.losses {
		if loss.TenantID == strings.TrimSpace(tenant) && loss.ID == strings.TrimSpace(id) {
			return loss.LegalEntityID, nil
		}
	}
	return "", ErrNotFound
}

func lossKey(tenant, entity, id string) string {
	return tenant + "\x00" + entity + "\x00" + id
}

func lossCodeKey(tenant, entity, code string) string {
	return tenant + "\x00" + entity + "\x00" + strings.ToUpper(strings.TrimSpace(code))
}
