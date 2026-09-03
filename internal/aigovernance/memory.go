package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                sync.RWMutex
	policies          map[string]Policy
	workloads         map[string]Workload
	gatewayTransports map[string]GatewayTransportRevision
	receipts          map[string]DecisionReceipt
	grants            map[string]ExecutionGrant
	grantDigests      map[string][sha256.Size]byte
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		policies: map[string]Policy{}, workloads: map[string]Workload{}, gatewayTransports: map[string]GatewayTransportRevision{},
		receipts: map[string]DecisionReceipt{}, grants: map[string]ExecutionGrant{}, grantDigests: map[string][sha256.Size]byte{},
	}
}

func memKey(tenant, id string) string { return tenant + "|" + id }
func (r *MemoryRepository) CreatePolicy(_ context.Context, v Policy) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ID)
	if _, ok := r.policies[k]; ok {
		return Policy{}, ErrConflict
	}
	r.policies[k] = v
	return v, nil
}

func (r *MemoryRepository) NextPolicyVersion(_ context.Context, tenantID, code string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var maxVersion int64
	for _, value := range r.policies {
		if value.TenantID == tenantID && value.Code == code && value.Version > maxVersion {
			maxVersion = value.Version
		}
	}
	return maxVersion + 1, nil
}

func (r *MemoryRepository) HasShadowHistory(_ context.Context, tenantID, code string, beforeVersion int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.policies {
		if value.TenantID == tenantID && value.Code == code && value.Version < beforeVersion && value.RolloutMode == "SHADOW" && (value.Status == "ACTIVE" || value.Status == "SUSPENDED" || value.Status == "RETIRED") {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryRepository) Policy(_ context.Context, t, id string) (Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.policies[memKey(t, id)]
	if !ok {
		return Policy{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) ListPolicies(_ context.Context, t string, limit int) ([]Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := []Policy{}
	for _, v := range r.policies {
		if v.TenantID == t {
			o = append(o, v)
		}
	}
	sort.Slice(o, func(i, j int) bool { return o[i].Code < o[j].Code })
	if len(o) > limit {
		o = o[:limit]
	}
	return o, nil
}
func (r *MemoryRepository) UpdatePolicy(_ context.Context, v Policy, expected int64) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ID)
	old, ok := r.policies[k]
	if !ok {
		return Policy{}, ErrNotFound
	}
	if old.RecordVersion != expected {
		return Policy{}, ErrConflict
	}
	r.policies[k] = v
	return v, nil
}
func (r *MemoryRepository) CreateWorkload(_ context.Context, v Workload) (Workload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ID)
	if _, ok := r.workloads[k]; ok {
		return Workload{}, ErrConflict
	}
	r.workloads[k] = v
	return v, nil
}

func (r *MemoryRepository) NextWorkloadVersion(_ context.Context, tenantID, workloadID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var maxVersion int64
	for _, value := range r.workloads {
		if value.TenantID == tenantID && value.WorkloadID == workloadID && value.Version > maxVersion {
			maxVersion = value.Version
		}
	}
	return maxVersion + 1, nil
}

func (r *MemoryRepository) Workload(_ context.Context, t, id string) (Workload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.workloads[memKey(t, id)]
	if !ok {
		return Workload{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) WorkloadByCredential(_ context.Context, d [32]byte) (Workload, Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	hexv := hex.EncodeToString(d[:])
	for _, w := range r.workloads {
		if w.KeySHA256 == hexv && w.State == "ACTIVE" {
			p, ok := r.policies[memKey(w.TenantID, w.PolicyID)]
			if !ok || p.Status != "ACTIVE" {
				return Workload{}, Policy{}, ErrNotFound
			}
			return w, p, nil
		}
	}
	return Workload{}, Policy{}, ErrNotFound
}
func (r *MemoryRepository) ListWorkloads(_ context.Context, t string, limit int) ([]Workload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := []Workload{}
	for _, v := range r.workloads {
		if v.TenantID == t {
			o = append(o, v)
		}
	}
	sort.Slice(o, func(i, j int) bool { return o[i].Code < o[j].Code })
	if len(o) > limit {
		o = o[:limit]
	}
	return o, nil
}
func (r *MemoryRepository) UpdateWorkload(_ context.Context, v Workload, expected int64) (Workload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ID)
	old, ok := r.workloads[k]
	if !ok {
		return Workload{}, ErrNotFound
	}
	if old.RecordVersion != expected {
		return Workload{}, ErrConflict
	}
	if v.State == "ACTIVE" {
		for otherKey, other := range r.workloads {
			if otherKey != k && other.State == "ACTIVE" && other.KeySHA256 == v.KeySHA256 {
				return Workload{}, ErrConflict
			}
		}
	}
	r.workloads[k] = v
	return v, nil
}
func (r *MemoryRepository) IngestReceipt(_ context.Context, v DecisionReceipt) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ReceiptID)
	if old, ok := r.receipts[k]; ok {
		if old.Fingerprint == v.Fingerprint {
			return false, nil
		}
		return false, ErrConflict
	}
	r.receipts[k] = v
	return true, nil
}
func (r *MemoryRepository) CreateGrant(_ context.Context, v ExecutionGrant, d [32]byte) (ExecutionGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memKey(v.TenantID, v.ID)
	r.grants[k] = v
	r.grantDigests[k] = d
	v.Token = ""
	return v, nil
}
func (r *MemoryRepository) ConsumeGrant(_ context.Context, t, w, action string, d [32]byte, now time.Time) (ExecutionGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, g := range r.grants {
		if g.TenantID != t || g.WorkloadID != w || g.ActionHash != action || g.State != "ACTIVE" || !g.ExpiresAt.After(now) {
			continue
		}
		stored := r.grantDigests[k]
		if stored != d {
			continue
		}
		used := now
		g.State = "USED"
		g.UsedAt = &used
		g.RecordVersion++
		r.grants[k] = g
		return g, nil
	}
	return ExecutionGrant{}, ErrGrantInvalid
}

func (r *MemoryRepository) MaintainRetention(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	processed := 0
	for key, receipt := range r.receipts {
		if processed >= limit {
			break
		}
		if !receipt.ExpiresAt.After(now) {
			delete(r.receipts, key)
			processed++
		}
	}
	for key, grant := range r.grants {
		if processed >= limit {
			break
		}
		if grant.State == "ACTIVE" && !grant.ExpiresAt.After(now) {
			grant.State = "EXPIRED"
			grant.RecordVersion++
			r.grants[key] = grant
			processed++
		}
	}
	return processed, nil
}
