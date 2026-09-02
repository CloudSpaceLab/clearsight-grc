package thirdparty

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryActivationRepository struct {
	mu            sync.RWMutex
	policies      map[string]ActivationPolicy
	relationships map[string]Relationship
	facts         map[string]ActivationFacts
	receipts      map[string]ActivationReceipt
	source        *MemoryAssessmentRepository
}

func NewMemoryActivationRepository(sources ...*MemoryAssessmentRepository) *MemoryActivationRepository {
	var source *MemoryAssessmentRepository
	if len(sources) > 0 {
		source = sources[0]
	}
	return &MemoryActivationRepository{policies: map[string]ActivationPolicy{}, relationships: map[string]Relationship{}, facts: map[string]ActivationFacts{}, receipts: map[string]ActivationReceipt{}, source: source}
}

func (r *MemoryActivationRepository) ProposeActivationPolicy(_ context.Context, policy ActivationPolicy) (ActivationPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	max := 0
	for _, value := range r.policies {
		if value.TenantID == policy.TenantID && value.LegalEntityID == policy.LegalEntityID && value.PolicyNumber > max {
			max = value.PolicyNumber
		}
	}
	policy.PolicyNumber = max + 1
	r.policies[policy.ID] = cloneActivationPolicy(policy)
	return cloneActivationPolicy(policy), nil
}

func (r *MemoryActivationRepository) TransitionActivationPolicy(_ context.Context, scope Scope, policyID string, expectedVersion int64, to ActivationPolicyStatus, actorID, rationale string, at time.Time) (ActivationPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	policy, ok := r.policies[policyID]
	if !ok || policy.TenantID != scope.TenantID || policy.LegalEntityID != scope.LegalEntityID {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	if policy.Version != expectedVersion {
		return ActivationPolicy{}, ErrVersionConflict
	}
	if err := validatePolicyTransition(policy.Status, to); err != nil {
		return ActivationPolicy{}, err
	}
	if to == ActivationPolicyPendingApproval {
		policy.Status = to
	} else {
		for id, prior := range r.policies {
			if id == policy.ID || prior.TenantID != scope.TenantID || prior.LegalEntityID != scope.LegalEntityID || prior.Status != ActivationPolicyActive {
				continue
			}
			if prior.EffectiveUntil == nil || prior.EffectiveUntil.After(policy.EffectiveFrom) {
				end := policy.EffectiveFrom
				prior.EffectiveUntil, prior.Status, prior.UpdatedAt, prior.Version = &end, ActivationPolicyRetired, at, prior.Version+1
				r.policies[id] = prior
			}
		}
		policy.Status, policy.ApprovedBy, policy.ApprovalRationale = to, actorID, rationale
	}
	policy.UpdatedAt, policy.Version = at.UTC(), policy.Version+1
	r.policies[policyID] = policy
	return cloneActivationPolicy(policy), nil
}

func (r *MemoryActivationRepository) GetActivationPolicy(_ context.Context, scope Scope, policyID string) (ActivationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.policies[policyID]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	return cloneActivationPolicy(value), nil
}

func (r *MemoryActivationRepository) CurrentActivationPolicy(_ context.Context, scope Scope, at time.Time) (ActivationPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var current []ActivationPolicy
	for _, value := range r.policies {
		if value.TenantID == scope.TenantID && value.LegalEntityID == scope.LegalEntityID && value.Status == ActivationPolicyActive && !value.EffectiveFrom.After(at) && (value.EffectiveUntil == nil || value.EffectiveUntil.After(at)) {
			current = append(current, value)
		}
	}
	if len(current) != 1 {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	return cloneActivationPolicy(current[0]), nil
}

func (r *MemoryActivationRepository) ListActivationCandidates(_ context.Context, scope Scope, limit int) ([]Relationship, error) {
	if r.source != nil {
		page, err := r.source.ListRelationships(context.Background(), ListFilter{Scope: scope, Limit: limit})
		if err != nil {
			return nil, err
		}
		result := make([]Relationship, 0, len(page.Items))
		for _, item := range page.Items {
			if item.Relationship.Status == RelationshipProposed || item.Relationship.Status == RelationshipUnderReview {
				result = append(result, item.Relationship)
			}
		}
		return result, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []Relationship{}
	for _, value := range r.relationships {
		if value.TenantID == scope.TenantID && value.LegalEntityID == scope.LegalEntityID && (value.Status == RelationshipProposed || value.Status == RelationshipUnderReview) {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *MemoryActivationRepository) ReadActivationFacts(ctx context.Context, scope Scope, relationshipID string, _ ActivationPolicy) (Relationship, ActivationFacts, error) {
	if r.source != nil {
		aggregate, err := r.source.GetRelationship(ctx, scope, relationshipID)
		if err != nil {
			return Relationship{}, ActivationFacts{}, err
		}
		r.mu.RLock()
		facts := cloneActivationFacts(r.facts[relationshipID])
		r.mu.RUnlock()
		if assessment, assessmentErr := r.source.GetCurrentAssessment(ctx, scope, relationshipID, AssessmentReviewOnboarding); assessmentErr == nil {
			facts.AssessmentID = assessment.ID
			facts.AssessmentVersion = assessment.Version
			facts.AssessmentStatus = assessment.Status
			facts.AssessmentConclusion = assessment.Conclusion
			if assessment.CompletedAt != nil {
				facts.AssessmentCompletedAt = assessment.CompletedAt.UTC()
			}
		}
		return aggregate.Relationship, facts, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	relationship, ok := r.relationships[relationshipID]
	if !ok || relationship.TenantID != scope.TenantID || relationship.LegalEntityID != scope.LegalEntityID {
		return Relationship{}, ActivationFacts{}, ErrNotFound
	}
	return relationship, cloneActivationFacts(r.facts[relationshipID]), nil
}

func (r *MemoryActivationRepository) CommitRelationshipActivation(_ context.Context, commit ActivationCommit) (Relationship, ActivationReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.source != nil {
		r.source.MemoryRepository.mu.Lock()
		defer r.source.MemoryRepository.mu.Unlock()
		relationship, ok := r.source.MemoryRepository.relationships[commit.RelationshipID]
		if !ok || relationship.TenantID != commit.TenantID || relationship.LegalEntityID != commit.LegalEntityID {
			return Relationship{}, ActivationReceipt{}, ErrNotFound
		}
		if relationship.Version != commit.ExpectedVersion {
			return Relationship{}, ActivationReceipt{}, ErrVersionConflict
		}
		if err := r.validateActivationCommit(commit); err != nil {
			return Relationship{}, ActivationReceipt{}, err
		}
		relationship.Status, relationship.EffectiveFrom, relationship.UpdatedAt, relationship.Version = RelationshipActive, &commit.EffectiveAt, commit.EffectiveAt, relationship.Version+1
		r.source.MemoryRepository.relationships[relationship.ID] = relationship
		receipt := activationReceiptFromCommit(commit, relationship)
		r.receipts[receipt.ID] = receipt
		return relationship, receipt, nil
	}
	relationship, ok := r.relationships[commit.RelationshipID]
	if !ok || relationship.TenantID != commit.TenantID || relationship.LegalEntityID != commit.LegalEntityID {
		return Relationship{}, ActivationReceipt{}, ErrNotFound
	}
	if relationship.Version != commit.ExpectedVersion {
		return Relationship{}, ActivationReceipt{}, ErrVersionConflict
	}
	if err := r.validateActivationCommit(commit); err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	relationship.Status, relationship.EffectiveFrom, relationship.UpdatedAt, relationship.Version = RelationshipActive, &commit.EffectiveAt, commit.EffectiveAt, relationship.Version+1
	r.relationships[relationship.ID] = relationship
	receipt := activationReceiptFromCommit(commit, relationship)
	r.receipts[receipt.ID] = receipt
	return relationship, receipt, nil
}

func (r *MemoryActivationRepository) validateActivationCommit(commit ActivationCommit) error {
	currentFacts := r.facts[commit.RelationshipID]
	if r.source != nil {
		assessment, err := r.source.GetAssessment(context.Background(), Scope{TenantID: commit.TenantID, LegalEntityID: commit.LegalEntityID}, commit.Facts.AssessmentID)
		if err != nil || assessment.Version != commit.Facts.AssessmentVersion {
			return ErrVersionConflict
		}
		currentFacts.AssessmentID, currentFacts.AssessmentVersion = assessment.ID, assessment.Version
	}
	if currentFacts.AssessmentID != commit.Facts.AssessmentID || currentFacts.AssessmentVersion != commit.Facts.AssessmentVersion || currentFacts.VerificationResultID != commit.Facts.VerificationResultID {
		return ErrVersionConflict
	}
	currentPolicy, ok := r.policies[commit.Policy.ID]
	if !ok || currentPolicy.Version != commit.Policy.Version || currentPolicy.Status != ActivationPolicyActive || currentPolicy.EffectiveFrom.After(commit.EffectiveAt) || (currentPolicy.EffectiveUntil != nil && !currentPolicy.EffectiveUntil.After(commit.EffectiveAt)) {
		return ErrActivationPolicyUnavailable
	}
	return nil
}

func activationReceiptFromCommit(commit ActivationCommit, relationship Relationship) ActivationReceipt {
	return ActivationReceipt{ID: commit.ReceiptID, TenantID: commit.TenantID, LegalEntityID: commit.LegalEntityID, RelationshipID: relationship.ID, RelationshipVersion: relationship.Version, PolicyID: commit.Policy.ID, PolicyVersion: commit.Policy.Version, AssessmentID: commit.Facts.AssessmentID, AssessmentVersion: commit.Facts.AssessmentVersion, DecisionIDs: append([]string(nil), commit.Facts.DecisionIDs...), AddressMatterID: commit.Facts.AddressMatterID, VerificationResultID: commit.Facts.VerificationResultID, ActivatedBy: commit.ActorID, ActivatedAt: commit.EffectiveAt, Rationale: commit.Rationale}
}

func (r *MemoryActivationRepository) PutPolicyForTest(value ActivationPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policies[value.ID] = cloneActivationPolicy(value)
}
func (r *MemoryActivationRepository) PutRelationshipForTest(value Relationship) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.relationships[value.ID] = value
}
func (r *MemoryActivationRepository) PutActivationFactsForTest(id string, value ActivationFacts) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.facts[id] = cloneActivationFacts(value)
}
func (r *MemoryActivationRepository) RelationshipForTest(id string) Relationship {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relationships[id]
}

func cloneActivationPolicy(value ActivationPolicy) ActivationPolicy {
	value.AllowedConclusions = append([]AssessmentConclusion(nil), value.AllowedConclusions...)
	value.RequiredDecisionTypes = append([]string(nil), value.RequiredDecisionTypes...)
	value.BlockingMatterTypes = append([]string(nil), value.BlockingMatterTypes...)
	if value.EffectiveUntil != nil {
		copied := *value.EffectiveUntil
		value.EffectiveUntil = &copied
	}
	return value
}
func cloneActivationFacts(value ActivationFacts) ActivationFacts {
	value.SatisfiedDecisionTypes = append([]string(nil), value.SatisfiedDecisionTypes...)
	value.DecisionIDs = append([]string(nil), value.DecisionIDs...)
	return value
}

var _ ActivationRepository = (*MemoryActivationRepository)(nil)
