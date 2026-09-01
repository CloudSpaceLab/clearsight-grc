package formpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type MemoryRepository struct {
	mu                 sync.RWMutex
	policies           map[string]Policy
	simulations        map[string]SimulationReceipt
	executions         map[string]ExecutionReceipt
	executionFailures  map[string]ExecutionReceipt
	episodes           map[string]AdverseEpisode
	episodeHistory     map[string]AdverseEpisode
	matters            map[string]continuity.Matter
	contracts          map[string]continuity.VerificationContract
	links              map[string]continuity.MatterLink
	outcomes           map[string]memoryOutcomeCheck
	compensations      map[string]CompensationReceipt
	operationalActions map[string]continuity.Action
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{policies: map[string]Policy{}, simulations: map[string]SimulationReceipt{}, executions: map[string]ExecutionReceipt{}, executionFailures: map[string]ExecutionReceipt{}, episodes: map[string]AdverseEpisode{}, episodeHistory: map[string]AdverseEpisode{}, matters: map[string]continuity.Matter{}, contracts: map[string]continuity.VerificationContract{}, links: map[string]continuity.MatterLink{}, outcomes: map[string]memoryOutcomeCheck{}, compensations: map[string]CompensationReceipt{}, operationalActions: map[string]continuity.Action{}}
}

type memoryOutcomeCheck struct {
	EpisodeKey string
	MatterID   string
	DueAt      time.Time
	Completed  bool
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

func (repo *MemoryRepository) ListEffectivePolicies(_ context.Context, tenantID, legalEntityID, formTemplateID string, formTemplateVersion int64, completedAt time.Time, limit int) ([]Policy, error) {
	if limit < 1 || limit > executionPolicyLimit || strings.TrimSpace(formTemplateID) == "" || formTemplateVersion < 1 || completedAt.IsZero() {
		return nil, ErrInvalid
	}
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	values := make([]Policy, 0)
	for _, value := range repo.policies {
		if value.TenantID != tenantID || value.LegalEntityID != legalEntityID || value.Status != PolicyActive || value.ActivatedAt == nil || value.ActivatedAt.After(completedAt) || value.Eligibility.FormTemplateID != formTemplateID || value.Eligibility.FormTemplateVersion != formTemplateVersion || value.EffectiveFrom != nil && value.EffectiveFrom.After(completedAt) || value.EffectiveUntil != nil && !value.EffectiveUntil.After(completedAt) {
			continue
		}
		values = append(values, clonePolicy(value))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		return values[i].Version > values[j].Version
	})
	if len(values) > limit {
		return nil, ErrExecutionPolicyLimit
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

func (repo *MemoryRepository) CreateExecution(_ context.Context, value ExecutionReceipt) (ExecutionReceipt, bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	key := value.TenantID + "|" + value.LegalEntityID + "|" + value.PolicyID + "|" + fmt.Sprint(value.PolicyVersion) + "|" + value.ResponseRevisionID
	if stored, exists := repo.executions[key]; exists {
		if executionFingerprint(stored) != executionFingerprint(value) {
			return ExecutionReceipt{}, false, ErrConflict
		}
		return stored, false, nil
	}
	repo.executions[key] = value
	return value, true, nil
}

func (repo *MemoryRepository) OpenEpisode(_ context.Context, value AdverseEpisode) (AdverseEpisode, bool, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	key := value.TenantID + "|" + value.LegalEntityID + "|" + value.PolicyCode + "|" + value.SubjectType + "|" + value.SubjectID
	if stored, exists := repo.episodes[key]; exists && stored.State == EpisodeOpen {
		return stored, false, nil
	}
	repo.episodes[key] = value
	repo.episodeHistory[value.ID] = value
	return value, true, nil
}

func (repo *MemoryRepository) ApplyExecution(_ context.Context, command ExecutionCommand) (ExecutionReceipt, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	receipt := command.Receipt
	key := receipt.TenantID + "|" + receipt.LegalEntityID + "|" + receipt.PolicyID + "|" + fmt.Sprint(receipt.PolicyVersion) + "|" + receipt.ResponseRevisionID
	if stored, exists := repo.executions[key]; exists {
		return stored, nil
	}
	if receipt.State != ExecutionApplied {
		if receipt.State == ExecutionFailed {
			failureKey := key + "|" + command.EventID
			if stored, exists := repo.executionFailures[failureKey]; exists {
				return stored, nil
			}
			repo.executionFailures[failureKey] = receipt
			if command.FailureMatter != nil && command.FailureAction != nil {
				triggerKey := command.FailureMatter.TriggerKey
				for _, matter := range repo.matters {
					if matter.TriggerKey == triggerKey && matter.Status != continuity.MatterClosed && matter.Status != continuity.MatterCancelled {
						return receipt, nil
					}
				}
				failureMatter := *command.FailureMatter
				failureMatter.Version = 2
				repo.matters[command.FailureMatter.ID] = failureMatter
				repo.operationalActions[command.FailureAction.ID] = *command.FailureAction
			}
			return receipt, nil
		}
		repo.recordExecutionRouteRecovery(command, receipt)
		repo.executions[key] = receipt
		return receipt, nil
	}
	episodeKey := command.Episode.TenantID + "|" + command.Episode.LegalEntityID + "|" + command.Episode.PolicyCode + "|" + command.Episode.SubjectType + "|" + command.Episode.SubjectID
	if episode, exists := repo.episodes[episodeKey]; exists && episode.State == EpisodeOpen {
		matter := repo.matters[episode.MatterID]
		if matter.Status == continuity.MatterClosed && matter.ClosedAt != nil {
			closedAt := matter.ClosedAt.UTC()
			episode.State, episode.ClosedAt, episode.UpdatedAt = EpisodeClosed, &closedAt, receipt.CreatedAt
			episode.RecordVersion++
			repo.episodes[episodeKey] = episode
			repo.episodeHistory[episode.ID] = episode
		} else if matter.Status == continuity.MatterCancelled {
			receipt.State, receipt.MatterID, receipt.ReasonCode = ExecutionFailed, episode.MatterID, "OPEN_EPISODE_MATTER_CANCELLED"
			repo.executions[key] = receipt
			return receipt, nil
		} else {
			receipt.State, receipt.MatterID, receipt.CreatedMatter = ExecutionReused, episode.MatterID, false
			receipt.ReasonCode = "OPEN_EPISODE_REUSED"
			episode.PolicyID, episode.PolicyVersion, episode.LastResponseRevisionID, episode.UpdatedAt = command.Policy.ID, command.Policy.Version, command.Response.ID, receipt.CreatedAt
			episode.RecordVersion++
			repo.episodes[episodeKey] = episode
			repo.episodeHistory[episode.ID] = episode
			matter.SourceID, matter.UpdatedAt = command.Response.ID, receipt.CreatedAt
			matter.Version++
			repo.matters[matter.ID] = matter
			repo.recordExecutionRouteRecovery(command, receipt)
			repo.executions[key] = receipt
			repo.outcomes[receipt.ID] = memoryOutcomeCheck{EpisodeKey: episodeKey, MatterID: episode.MatterID, DueAt: receipt.CreatedAt.Add(time.Duration(command.Policy.Outcome.CheckAfterMinutes) * time.Minute)}
			return receipt, nil
		}
	}
	runStart := receipt.CreatedAt.UTC().Truncate(executionRunWindow)
	createdThisRun := 0
	for _, stored := range repo.executions {
		if stored.TenantID == receipt.TenantID && stored.LegalEntityID == receipt.LegalEntityID && stored.PolicyID == receipt.PolicyID && stored.PolicyVersion == receipt.PolicyVersion && stored.CreatedMatter && !stored.CreatedAt.Before(runStart) && stored.CreatedAt.Before(runStart.Add(executionRunWindow)) {
			createdThisRun++
		}
	}
	if createdThisRun >= command.Policy.BlastRadius.PerRun {
		receipt.State, receipt.ReasonCode = ExecutionBlastSuppressed, "PER_RUN_LIMIT"
		repo.recordExecutionRouteRecovery(command, receipt)
		repo.executions[key] = receipt
		return receipt, nil
	}
	dayStart := receipt.CreatedAt.UTC().Truncate(24 * time.Hour)
	createdToday := 0
	for _, stored := range repo.executions {
		if stored.TenantID == receipt.TenantID && stored.LegalEntityID == receipt.LegalEntityID && stored.PolicyID == receipt.PolicyID && stored.PolicyVersion == receipt.PolicyVersion && stored.CreatedMatter && !stored.CreatedAt.Before(dayStart) && stored.CreatedAt.Before(dayStart.Add(24*time.Hour)) {
			createdToday++
		}
	}
	if createdToday >= command.Policy.BlastRadius.PerDay {
		receipt.State, receipt.ReasonCode = ExecutionBlastSuppressed, "PER_DAY_LIMIT"
		repo.recordExecutionRouteRecovery(command, receipt)
		repo.executions[key] = receipt
		return receipt, nil
	}
	receipt.MatterID, receipt.CreatedMatter = command.Matter.ID, true
	command.Episode.MatterID = command.Matter.ID
	storedMatter := command.Matter
	storedMatter.Version = 2
	if command.Link != nil {
		storedMatter.Version = 3
		repo.links[command.Link.ID] = *command.Link
	}
	repo.matters[command.Matter.ID] = storedMatter
	repo.contracts[command.Outcome.ID] = command.Outcome
	repo.episodes[episodeKey] = command.Episode
	repo.episodeHistory[command.Episode.ID] = command.Episode
	repo.recordExecutionRouteRecovery(command, receipt)
	repo.executions[key] = receipt
	repo.outcomes[receipt.ID] = memoryOutcomeCheck{EpisodeKey: episodeKey, MatterID: command.Matter.ID, DueAt: receipt.CreatedAt.Add(time.Duration(command.Policy.Outcome.CheckAfterMinutes) * time.Minute)}
	return receipt, nil
}

func (repo *MemoryRepository) recordExecutionRouteRecovery(command ExecutionCommand, receipt ExecutionReceipt) {
	if strings.TrimSpace(command.Route.ServicePrincipalID) == "" || receipt.State == ExecutionFailed {
		return
	}
	triggerKey := "form-response-policy-execution-failure:" + command.Policy.ID + ":" + command.Response.ID
	for matterID, matter := range repo.matters {
		if matter.TriggerKey != triggerKey || matter.Status == continuity.MatterClosed || matter.Status == continuity.MatterCancelled {
			continue
		}
		facts := map[string]any{}
		if len(matter.KnownFacts) > 0 && json.Unmarshal(matter.KnownFacts, &facts) != nil {
			return
		}
		facts["route_recovery_execution_id"] = receipt.ID
		facts["route_recovery_recorded_at"] = receipt.CreatedAt.UTC().Format(time.RFC3339Nano)
		matter.KnownFacts = mustExecutionJSON(facts)
		matter.UpdatedAt = receipt.CreatedAt
		matter.Version++
		for actionID, action := range repo.operationalActions {
			if action.MatterID != matter.ID || action.OriginKey != "form-response-policy-execution-recovery" || action.Status == continuity.ActionImplemented || action.Status == continuity.ActionCancelled {
				continue
			}
			if action.Status == continuity.ActionPlanned || action.Status == continuity.ActionBlocked {
				action.Status = continuity.ActionInProgress
				action.Version++
				matter.Version++
			}
			if action.Status == continuity.ActionInProgress {
				action.Status = continuity.ActionImplemented
				implementedAt := receipt.CreatedAt.UTC()
				action.ImplementedAt = &implementedAt
				action.UpdatedAt = receipt.CreatedAt
				action.Version++
				action.Description = strings.TrimSpace(action.Description) + " Route restored by policy execution " + receipt.ID + "."
				matter.Version++
				repo.operationalActions[actionID] = action
			}
		}
		repo.matters[matterID] = matter
		return
	}
}

func (repo *MemoryRepository) ListPendingCompensations(_ context.Context, now time.Time, limit int) ([]CompensationCandidate, error) {
	if now.IsZero() || limit < 1 || limit > maintenanceBatchLimit {
		return nil, ErrInvalid
	}
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	values := make([]CompensationCandidate, 0)
	for _, policy := range repo.policies {
		if policy.Status != PolicyActive || policy.RollbackOfPolicyID == "" || policy.ActivatedAt == nil || policy.ActivatedAt.After(now) {
			continue
		}
		for _, execution := range repo.executions {
			key := compensationKey(policy, execution)
			if execution.TenantID != policy.TenantID || execution.LegalEntityID != policy.LegalEntityID || execution.PolicyID != policy.RollbackOfPolicyID || !execution.CreatedMatter || execution.MatterID == "" || execution.CreatedAt.After(*policy.ActivatedAt) {
				continue
			}
			if _, exists := repo.compensations[key]; exists {
				continue
			}
			values = append(values, CompensationCandidate{RollbackPolicy: clonePolicy(policy), OriginalExecution: execution})
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if !values[i].OriginalExecution.CreatedAt.Equal(values[j].OriginalExecution.CreatedAt) {
			return values[i].OriginalExecution.CreatedAt.Before(values[j].OriginalExecution.CreatedAt)
		}
		return values[i].OriginalExecution.ID < values[j].OriginalExecution.ID
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (repo *MemoryRepository) ApplyCompensation(_ context.Context, command CompensationCommand) (CompensationReceipt, error) {
	if !validCompensationCommand(command) {
		return CompensationReceipt{}, ErrInvalid
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	policy, exists := repo.policies[policyKey(command.Receipt.TenantID, command.Receipt.LegalEntityID, command.Receipt.RollbackPolicyID)]
	if !exists || policy.Status != PolicyActive || policy.Version != command.Receipt.RollbackPolicyVersion || policy.RollbackOfPolicyID != command.Candidate.OriginalExecution.PolicyID || policy.ActivatedAt == nil || command.Candidate.OriginalExecution.CreatedAt.After(*policy.ActivatedAt) {
		return CompensationReceipt{}, ErrConflict
	}
	key := compensationKey(policy, command.Candidate.OriginalExecution)
	if stored, exists := repo.compensations[key]; exists {
		return stored, nil
	}
	matter, exists := repo.matters[command.Receipt.MatterID]
	if !exists || matter.TenantID != command.Receipt.TenantID || matter.LegalEntityID != command.Receipt.LegalEntityID {
		return CompensationReceipt{}, ErrConflict
	}
	if matter.Status != continuity.MatterClosed && matter.Status != continuity.MatterCancelled {
		facts := map[string]any{}
		if len(matter.KnownFacts) > 0 && json.Unmarshal(matter.KnownFacts, &facts) != nil {
			return CompensationReceipt{}, ErrInvalid
		}
		facts["form_response_policy_compensation"] = map[string]any{"state": command.Receipt.State, "rollback_policy_id": command.Receipt.RollbackPolicyID, "original_execution_id": command.Receipt.OriginalExecutionID}
		matter.KnownFacts = mustExecutionJSON(facts)
		matter.UpdatedAt = command.Receipt.CreatedAt
		matter.Version++
		repo.matters[matter.ID] = matter
	}
	reviewMatter := command.ReviewMatter
	reviewMatter.Version = 2
	repo.matters[reviewMatter.ID] = reviewMatter
	repo.operationalActions[command.ReviewAction.ID] = command.ReviewAction
	repo.compensations[key] = command.Receipt
	return command.Receipt, nil
}

func compensationKey(policy Policy, execution ExecutionReceipt) string {
	return strings.Join([]string{policy.TenantID, policy.LegalEntityID, policy.ID, fmt.Sprint(policy.Version), execution.ID}, "|")
}

func (repo *MemoryRepository) MaintainOutcomeChecks(_ context.Context, workerID string, now time.Time, lease time.Duration, limit int) (int, error) {
	if strings.TrimSpace(workerID) == "" || now.IsZero() || lease <= 0 || limit < 1 || limit > maintenanceBatchLimit {
		return 0, ErrInvalid
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	processed := 0
	for id, check := range repo.outcomes {
		if processed >= limit || check.Completed || check.DueAt.After(now) {
			continue
		}
		matter, exists := repo.matters[check.MatterID]
		if !exists || matter.Status != continuity.MatterClosed || matter.ClosedAt == nil {
			continue
		}
		episode, exists := repo.episodes[check.EpisodeKey]
		if !exists || episode.State != EpisodeOpen || episode.MatterID != matter.ID {
			continue
		}
		closedAt := matter.ClosedAt.UTC()
		episode.State, episode.ClosedAt, episode.UpdatedAt = EpisodeClosed, &closedAt, closedAt
		episode.RecordVersion++
		repo.episodes[check.EpisodeKey] = episode
		repo.episodeHistory[episode.ID] = episode
		check.Completed = true
		repo.outcomes[id] = check
		processed++
	}
	return processed, nil
}

func executionFingerprint(value ExecutionReceipt) string {
	return strings.Join([]string{value.TenantID, value.LegalEntityID, value.PolicyID, fmt.Sprint(value.PolicyVersion), value.AutomationPolicyID, fmt.Sprint(value.AutomationPolicyVersion), value.ResponseRevisionID, string(value.State), value.MatterID, value.ReasonCode, fmt.Sprint(value.CreatedMatter)}, "|")
}

func clonePolicy(value Policy) Policy {
	value.Eligibility.SubjectTypes = append([]string(nil), value.Eligibility.SubjectTypes...)
	value.Eligibility.Bands = append([]formcontract.ConcernBand(nil), value.Eligibility.Bands...)
	return value
}
