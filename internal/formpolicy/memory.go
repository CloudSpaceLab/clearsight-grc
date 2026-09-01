package formpolicy

import (
	"context"
	"sort"
	"sync"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	policies    map[string]Policy
	simulations map[string]SimulationReceipt
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{policies: map[string]Policy{}, simulations: map[string]SimulationReceipt{}}
}

func policyKey(tenantID, legalEntityID, id string) string {
	return tenantID + "|" + legalEntityID + "|" + id
}

func (repo *MemoryRepository) CreatePolicy(_ context.Context, value Policy) (Policy, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	key := policyKey(value.TenantID, value.LegalEntityID, value.ID)
	if _, exists := repo.policies[key]; exists {
		return Policy{}, ErrConflict
	}
	repo.policies[key] = clonePolicy(value)
	return clonePolicy(value), nil
}

func (repo *MemoryRepository) GetPolicy(_ context.Context, tenantID, legalEntityID, id string) (Policy, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	value, exists := repo.policies[policyKey(tenantID, legalEntityID, id)]
	if !exists {
		return Policy{}, ErrNotFound
	}
	return clonePolicy(value), nil
}

func (repo *MemoryRepository) ListPolicies(_ context.Context, tenantID, legalEntityID string, limit int) ([]Policy, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	values := make([]Policy, 0)
	for _, value := range repo.policies {
		if value.TenantID == tenantID && value.LegalEntityID == legalEntityID {
			values = append(values, clonePolicy(value))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		return values[i].Version > values[j].Version
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (repo *MemoryRepository) NextPolicyVersion(_ context.Context, tenantID, legalEntityID, code string) (int64, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	var maximum int64
	for _, value := range repo.policies {
		if value.TenantID == tenantID && value.LegalEntityID == legalEntityID && value.Code == code && value.Version > maximum {
			maximum = value.Version
		}
	}
	return maximum + 1, nil
}

func (repo *MemoryRepository) UpdatePolicy(_ context.Context, value Policy, expected int64) (Policy, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	key := policyKey(value.TenantID, value.LegalEntityID, value.ID)
	current, exists := repo.policies[key]
	if !exists {
		return Policy{}, ErrNotFound
	}
	if current.RecordVersion != expected {
		return Policy{}, ErrConflict
	}
	if value.Status == PolicyActive {
		for otherKey, other := range repo.policies {
			if otherKey != key && other.TenantID == value.TenantID && other.LegalEntityID == value.LegalEntityID && other.Code == value.Code && other.Status == PolicyActive {
				return Policy{}, ErrConflict
			}
		}
	}
	repo.policies[key] = clonePolicy(value)
	return clonePolicy(value), nil
}

func (repo *MemoryRepository) HasShadowHistory(_ context.Context, tenantID, legalEntityID, code string, beforeVersion int64) (bool, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	for _, value := range repo.policies {
		if value.TenantID == tenantID && value.LegalEntityID == legalEntityID && value.Code == code && value.Version < beforeVersion && value.Rollout == RolloutShadow && (value.Status == PolicyActive || value.Status == PolicySuspended || value.Status == PolicyRetired) {
			return true, nil
		}
	}
	return false, nil
}

func (repo *MemoryRepository) SaveSimulation(_ context.Context, value SimulationReceipt) (SimulationReceipt, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	key := policyKey(value.TenantID, value.LegalEntityID, value.ID)
	if _, exists := repo.simulations[key]; exists {
		return SimulationReceipt{}, ErrConflict
	}
	repo.simulations[key] = value
	return value, nil
}

func (repo *MemoryRepository) GetSimulation(_ context.Context, tenantID, legalEntityID, id string) (SimulationReceipt, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	value, exists := repo.simulations[policyKey(tenantID, legalEntityID, id)]
	if !exists {
		return SimulationReceipt{}, ErrNotFound
	}
	return value, nil
}

func clonePolicy(value Policy) Policy {
	value.Eligibility.SubjectTypes = append([]string(nil), value.Eligibility.SubjectTypes...)
	value.Eligibility.Bands = append([]formcontract.ConcernBand(nil), value.Eligibility.Bands...)
	return value
}
