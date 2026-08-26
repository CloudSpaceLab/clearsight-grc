package governance

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                             sync.Mutex
	policies                       map[string]RoutingPolicy
	revisions                      map[string][]RoutingPolicyRevision
	delegations                    map[string]Delegation
	conflicts                      map[string][]ConflictFinding
	delegationCandidates           []DelegationCandidateDirectoryEntry
	delegationCandidatesConfigured bool
}

func NewMemoryRepositoryWithDelegationCandidates(entries []DelegationCandidateDirectoryEntry) *MemoryRepository {
	repository := NewMemoryRepository()
	repository.delegationCandidatesConfigured = true
	repository.delegationCandidates = append([]DelegationCandidateDirectoryEntry(nil), entries...)
	return repository
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		policies: map[string]RoutingPolicy{}, revisions: map[string][]RoutingPolicyRevision{},
		delegations: map[string]Delegation{}, conflicts: map[string][]ConflictFinding{},
	}
}
func key(tenantID, id string) string { return tenantID + ":" + id }

func (r *MemoryRepository) ListPolicies(_ context.Context, tenantID string) ([]RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []RoutingPolicy{}
	for _, v := range r.policies {
		if v.TenantID == tenantID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}
func (r *MemoryRepository) ListPoliciesForEntity(_ context.Context, tenantID, legalEntityID string, limit int) ([]RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		return []RoutingPolicy{}, nil
	}
	out := []RoutingPolicy{}
	for _, v := range r.policies {
		if v.TenantID == tenantID && v.LegalEntityID == legalEntityID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r *MemoryRepository) GetPolicy(_ context.Context, tenantID, id string) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.policies[key(tenantID, id)]
	if !ok {
		return RoutingPolicy{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) GetPolicyForEntity(_ context.Context, tenantID, legalEntityID, id string) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.policies[key(tenantID, id)]
	if !ok || v.LegalEntityID != legalEntityID {
		return RoutingPolicy{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) CreatePolicy(_ context.Context, v RoutingPolicy) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policies[key(v.TenantID, v.ID)] = v
	return v, nil
}
func (r *MemoryRepository) TransitionPolicy(_ context.Context, tenantID, legalEntityID, id string, expected int64, from, to PolicyState, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	v, ok := r.policies[k]
	if !ok || v.LegalEntityID != legalEntityID {
		return RoutingPolicy{}, ErrNotFound
	}
	if v.Version != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}
	if v.Status != from {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	v.Status = to
	v.Version++
	v.UpdatedAt = at
	v.LatestDecision = &GovernanceDecisionSummary{FromState: string(from), ToState: string(to), ActorID: actor, Rationale: rationale, DecidedAt: at, RecordVersion: v.Version}
	if to == PolicyPendingApproval {
		v.SubmittedAt = &at
	}
	if to == PolicyActive {
		v.CheckerID = actor
		v.ApprovedAt = &at
		if v.EffectiveFrom == nil {
			v.EffectiveFrom = &at
		}
	}
	if to == PolicyRetired {
		v.RetiredAt = &at
	}
	r.policies[k] = v
	return v, nil
}
func (r *MemoryRepository) ActivatePolicy(_ context.Context, tenantID, legalEntityID, id string, expected int64, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	policy, ok := r.policies[k]
	if !ok || policy.LegalEntityID != legalEntityID {
		return RoutingPolicy{}, ErrNotFound
	}
	if policy.Version != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}
	if policy.Status != PolicyPendingApproval {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if actor == "" || actor == policy.MakerID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	if len(r.conflicts[tenantID+":policy:"+id]) > 0 {
		return RoutingPolicy{}, ErrConflict
	}
	policy.Status = PolicyActive
	policy.CheckerID = actor
	policy.ApprovedAt = &at
	policy.UpdatedAt = at
	policy.Version++
	policy.LatestDecision = &GovernanceDecisionSummary{FromState: string(PolicyPendingApproval), ToState: string(PolicyActive), ActorID: actor, Rationale: rationale, DecidedAt: at, RecordVersion: policy.Version}
	if policy.EffectiveFrom == nil {
		policy.EffectiveFrom = &at
	}
	r.policies[k] = policy
	return policy, nil
}
func (r *MemoryRepository) CreatePolicyRevision(_ context.Context, tenantID, id string, expected int64, actor string, definition []byte, checksum string, at time.Time) (RoutingPolicyRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	policy, ok := r.policies[k]
	if !ok {
		return RoutingPolicyRevision{}, ErrNotFound
	}
	if policy.Status != PolicyActive {
		return RoutingPolicyRevision{}, ErrInvalidTransition
	}
	if policy.Version != expected {
		return RoutingPolicyRevision{}, ErrVersionConflict
	}
	version := policy.CurrentVersion + 1
	for _, revision := range r.revisions[k] {
		if revision.Version >= version {
			version = revision.Version + 1
		}
	}
	revision := RoutingPolicyRevision{
		PolicyID: id, TenantID: tenantID, LegalEntityID: policy.LegalEntityID, Version: version, BaseVersion: policy.CurrentVersion,
		Definition: append([]byte(nil), definition...), Checksum: checksum, MakerID: actor, CreatedAt: at,
	}
	r.revisions[k] = append(r.revisions[k], revision)
	policy.Version++
	policy.UpdatedAt = at
	r.policies[k] = policy
	return revision, nil
}
func (r *MemoryRepository) PendingPolicyRevision(_ context.Context, tenantID, id string) (RoutingPolicyRevision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	policy, ok := r.policies[k]
	if !ok {
		return RoutingPolicyRevision{}, ErrNotFound
	}
	var selected RoutingPolicyRevision
	found := false
	for _, revision := range r.revisions[k] {
		if revision.Version <= policy.CurrentVersion || revision.ApprovedAt != nil {
			continue
		}
		if !found || revision.Version > selected.Version {
			selected, found = revision, true
		}
	}
	if !found {
		return RoutingPolicyRevision{}, ErrNotFound
	}
	return selected, nil
}
func (r *MemoryRepository) ActivatePolicyRevision(_ context.Context, tenantID, id string, expected int64, revisionVersion int, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	policy, ok := r.policies[k]
	if !ok {
		return RoutingPolicy{}, ErrNotFound
	}
	if policy.Status != PolicyActive {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if policy.Version != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}
	latest := -1
	index := -1
	for i, revision := range r.revisions[k] {
		if revision.Version > policy.CurrentVersion && revision.ApprovedAt == nil && revision.Version > latest {
			latest, index = revision.Version, i
		}
	}
	if index < 0 || latest != revisionVersion {
		return RoutingPolicy{}, ErrRevisionStale
	}
	revision := r.revisions[k][index]
	if revision.MakerID == actor {
		return RoutingPolicy{}, ErrMakerChecker
	}
	if revision.LegalEntityID != policy.LegalEntityID || len(r.conflicts[tenantID+":policy:"+id]) > 0 {
		return RoutingPolicy{}, ErrConflict
	}
	if err := validatePolicyDefinition(revision.Definition); err != nil {
		return RoutingPolicy{}, err
	}
	if err := validatePolicyLegalEntity(revision.Definition, policy.LegalEntityID); err != nil {
		return RoutingPolicy{}, err
	}
	revision.ApprovedBy = actor
	revision.ApprovedAt = &at
	revision.EffectiveFrom = &at
	r.revisions[k][index] = revision
	policy.CurrentVersion = revision.Version
	policy.Definition = append([]byte(nil), revision.Definition...)
	policy.Checksum = revision.Checksum
	policy.CheckerID = actor
	policy.ApprovedAt = &at
	policy.EffectiveFrom = &at
	policy.Version++
	policy.UpdatedAt = at
	r.policies[k] = policy
	return policy, nil
}
func (r *MemoryRepository) PolicyConflicts(_ context.Context, policy RoutingPolicy) ([]ConflictFinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ConflictFinding(nil), r.conflicts[policy.TenantID+":policy:"+policy.ID]...), nil
}
func (r *MemoryRepository) EscalationReferenceConflicts(_ context.Context, _ string, _ []byte) ([]ConflictFinding, error) {
	return nil, nil
}
func (r *MemoryRepository) ListDelegations(_ context.Context, tenantID string) ([]Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Delegation{}
	for _, v := range r.delegations {
		if v.TenantID == tenantID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *MemoryRepository) ListDelegationsForEntity(_ context.Context, tenantID, legalEntityID string, limit int) ([]Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		return []Delegation{}, nil
	}
	out := []Delegation{}
	for _, v := range r.delegations {
		if v.TenantID == tenantID && v.LegalEntityID == legalEntityID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r *MemoryRepository) GetDelegation(_ context.Context, tenantID, id string) (Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.delegations[key(tenantID, id)]
	if !ok {
		return Delegation{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) GetDelegationForEntity(_ context.Context, tenantID, legalEntityID, id string) (Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.delegations[key(tenantID, id)]
	if !ok || v.LegalEntityID != legalEntityID {
		return Delegation{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryRepository) CreateDelegation(_ context.Context, v Delegation) (Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delegations[key(v.TenantID, v.ID)] = v
	return v, nil
}
func (r *MemoryRepository) TransitionDelegation(_ context.Context, tenantID, legalEntityID, id string, expected int64, from, to DelegationState, actor, rationale string, at time.Time) (Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	v, ok := r.delegations[k]
	if !ok || v.LegalEntityID != legalEntityID {
		return Delegation{}, ErrNotFound
	}
	if v.Version != expected {
		return Delegation{}, ErrVersionConflict
	}
	if v.Status != from {
		return Delegation{}, ErrInvalidTransition
	}
	v.Status = to
	v.Version++
	v.UpdatedAt = at
	if to == DelegationPendingApproval {
		v.SubmittedAt = &at
	}
	if to == DelegationApproved || to == DelegationActive {
		v.ApproverID = actor
		v.ApprovedAt = &at
	}
	if to == DelegationRevoked {
		v.RevokedAt = &at
	}
	r.delegations[k] = v
	return v, nil
}
func (r *MemoryRepository) HasDelegationCycle(_ context.Context, candidate Delegation) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	graph := map[string][]string{}
	for _, value := range r.delegations {
		if value.TenantID != candidate.TenantID || value.LegalEntityID != candidate.LegalEntityID || value.Responsibility != candidate.Responsibility || value.ID == candidate.ID {
			continue
		}
		if value.Status != DelegationApproved && value.Status != DelegationActive {
			continue
		}
		if !value.StartsAt.Before(candidate.EndsAt) || !candidate.StartsAt.Before(value.EndsAt) {
			continue
		}
		graph[value.FromPrincipalID] = append(graph[value.FromPrincipalID], value.ToPrincipalID)
	}
	seen := map[string]bool{}
	var reaches func(string) bool
	reaches = func(node string) bool {
		if node == candidate.FromPrincipalID {
			return true
		}
		if seen[node] {
			return false
		}
		seen[node] = true
		for _, next := range graph[node] {
			if reaches(next) {
				return true
			}
		}
		return false
	}
	return reaches(candidate.ToPrincipalID), nil
}
func (r *MemoryRepository) DelegationConflicts(_ context.Context, tenantID, principalID, responsibility string) ([]ConflictFinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ConflictFinding(nil), r.conflicts[tenantID+":"+principalID+":"+responsibility]...), nil
}
func (r *MemoryRepository) ActivateDelegation(_ context.Context, tenantID, legalEntityID, id string, expected int64, actor, rationale string, at time.Time) (Delegation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, id)
	candidate, ok := r.delegations[k]
	if !ok || candidate.LegalEntityID != legalEntityID {
		return Delegation{}, ErrNotFound
	}
	if candidate.Version != expected {
		return Delegation{}, ErrVersionConflict
	}
	if candidate.Status != DelegationPendingApproval {
		return Delegation{}, ErrInvalidTransition
	}
	if actor == "" || actor == candidate.MakerID || actor == candidate.FromPrincipalID || actor == candidate.ToPrincipalID {
		return Delegation{}, ErrMakerChecker
	}
	if len(r.conflicts[tenantID+":"+candidate.ToPrincipalID+":"+candidate.Responsibility]) > 0 {
		return Delegation{}, ErrConflict
	}
	graph := map[string][]string{}
	for _, value := range r.delegations {
		if value.ID == candidate.ID || value.TenantID != tenantID || value.LegalEntityID != legalEntityID || value.Responsibility != candidate.Responsibility {
			continue
		}
		if value.Status != DelegationApproved && value.Status != DelegationActive {
			continue
		}
		if value.StartsAt.Before(candidate.EndsAt) && candidate.StartsAt.Before(value.EndsAt) {
			graph[value.FromPrincipalID] = append(graph[value.FromPrincipalID], value.ToPrincipalID)
		}
	}
	seen := map[string]bool{}
	var reaches func(string) bool
	reaches = func(node string) bool {
		if node == candidate.FromPrincipalID {
			return true
		}
		if seen[node] {
			return false
		}
		seen[node] = true
		for _, next := range graph[node] {
			if reaches(next) {
				return true
			}
		}
		return false
	}
	if reaches(candidate.ToPrincipalID) {
		return Delegation{}, ErrConflict
	}
	if !r.memoryDelegationParticipantsEligible(candidate, at) {
		return Delegation{}, ErrDelegationEligibility
	}
	target := DelegationApproved
	if !at.Before(candidate.StartsAt) && at.Before(candidate.EndsAt) {
		target = DelegationActive
	}
	candidate.Status = target
	candidate.ApproverID = actor
	candidate.ApprovedAt = &at
	candidate.UpdatedAt = at
	candidate.Version++
	r.delegations[k] = candidate
	return candidate, nil
}
func (r *MemoryRepository) ActivateDueDelegations(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, limit)
	for k, v := range r.delegations {
		if v.Status == DelegationApproved && !now.Before(v.StartsAt) && now.Before(v.EndsAt) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}
	for _, k := range keys {
		candidate := r.delegations[k]
		if len(r.conflicts[candidate.TenantID+":"+candidate.ToPrincipalID+":"+candidate.Responsibility]) > 0 || memoryDelegationCycle(r.delegations, candidate) {
			return 0, ErrConflict
		}
		if !r.memoryDelegationParticipantsEligible(candidate, now) {
			return 0, ErrDelegationEligibility
		}
	}
	for _, k := range keys {
		v := r.delegations[k]
		v.Status = DelegationActive
		v.Version++
		v.UpdatedAt = now
		r.delegations[k] = v
	}
	return len(keys), nil
}

func (r *MemoryRepository) memoryDelegationParticipantsEligible(candidate Delegation, at time.Time) bool {
	if !r.delegationCandidatesConfigured {
		return false
	}
	giverCurrent := false
	recipientCurrent := false
	for _, entry := range r.delegationCandidates {
		if !entry.Active || entry.TenantID != candidate.TenantID || entry.LegalEntityID != candidate.LegalEntityID {
			continue
		}
		if entry.PrincipalID == candidate.FromPrincipalID && containsResponsibility(entry.Responsibilities, candidate.Responsibility) {
			giverCurrent = true
		}
		if entry.PrincipalID == candidate.ToPrincipalID && entry.CanReceive {
			recipientCurrent = true
		}
	}
	if !giverCurrent {
		for _, source := range r.delegations {
			if source.ID == candidate.ID || source.TenantID != candidate.TenantID || source.LegalEntityID != candidate.LegalEntityID || source.ToPrincipalID != candidate.FromPrincipalID || source.Responsibility != candidate.Responsibility || source.Status != DelegationActive {
				continue
			}
			if !source.StartsAt.After(candidate.StartsAt) && !source.EndsAt.Before(candidate.EndsAt) && !at.Before(source.StartsAt) && at.Before(source.EndsAt) {
				giverCurrent = true
				break
			}
		}
	}
	return giverCurrent && recipientCurrent
}

func memoryDelegationCycle(values map[string]Delegation, candidate Delegation) bool {
	graph := map[string][]string{}
	for _, value := range values {
		if value.ID == candidate.ID || value.TenantID != candidate.TenantID || value.LegalEntityID != candidate.LegalEntityID || value.Responsibility != candidate.Responsibility {
			continue
		}
		if value.Status != DelegationApproved && value.Status != DelegationActive {
			continue
		}
		if value.StartsAt.Before(candidate.EndsAt) && candidate.StartsAt.Before(value.EndsAt) {
			graph[value.FromPrincipalID] = append(graph[value.FromPrincipalID], value.ToPrincipalID)
		}
	}
	seen := map[string]bool{}
	var reaches func(string) bool
	reaches = func(node string) bool {
		if node == candidate.FromPrincipalID {
			return true
		}
		if seen[node] {
			return false
		}
		seen[node] = true
		for _, next := range graph[node] {
			if reaches(next) {
				return true
			}
		}
		return false
	}
	return reaches(candidate.ToPrincipalID)
}
func (r *MemoryRepository) ExpireDueDelegations(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for k, v := range r.delegations {
		if n >= limit {
			break
		}
		if (v.Status == DelegationApproved || v.Status == DelegationActive) && !now.Before(v.EndsAt) {
			v.Status = DelegationExpired
			v.Version++
			v.UpdatedAt = now
			r.delegations[k] = v
			n++
		}
	}
	return n, nil
}
