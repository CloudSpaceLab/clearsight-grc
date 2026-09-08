package formpolicy

import (
	"context"
	"sort"
	"time"
)

// History contains policy-operation facts only. It does not grant access to a
// restricted response, answer, issue or subject through configuration access.
type ExecutionHistoryItem struct {
	ID                string         `json:"id"`
	State             ExecutionState `json:"state"`
	ResultBasis       ResultBasis    `json:"result_basis"`
	AssessmentVersion int64          `json:"assessment_version,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}
type executionHistoryReader interface {
	ListExecutionHistory(context.Context, string, string, string, int) ([]ExecutionHistoryItem, error)
}

// ExecutionResultSource keeps unchecked target identifiers server-only.
type ExecutionResultSource struct {
	ExecutionHistoryItem
	ResponseRevisionID string `json:"-"`
	MatterID           string `json:"-"`
}

type executionResultReader interface {
	GetExecutionResult(context.Context, string, string, string, string) (ExecutionResultSource, error)
}

func (service *Service) ExecutionResult(ctx context.Context, actor Actor, policyID, executionID string) (ExecutionResultSource, error) {
	policy, err := service.Get(ctx, actor, policyID)
	if err != nil {
		return ExecutionResultSource{}, err
	}
	reader, ok := service.repo.(executionResultReader)
	if !ok {
		return ExecutionResultSource{}, ErrAuthorityUnavailable
	}
	return reader.GetExecutionResult(ctx, policy.TenantID, policy.LegalEntityID, policy.ID, executionID)
}

func (repo *MemoryRepository) GetExecutionResult(_ context.Context, tenant, entity, policy, execution string) (ExecutionResultSource, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	for _, receipts := range []map[string]ExecutionReceipt{repo.executions, repo.executionFailures} {
		for _, value := range receipts {
			if value.ID == execution && value.TenantID == tenant && value.LegalEntityID == entity && value.PolicyID == policy {
				return ExecutionResultSource{ExecutionHistoryItem: ExecutionHistoryItem{ID: value.ID, State: value.State, ResultBasis: receiptBasis(value), AssessmentVersion: value.AssessmentVersion, CreatedAt: value.CreatedAt}, ResponseRevisionID: value.ResponseRevisionID, MatterID: value.MatterID}, nil
			}
		}
	}
	return ExecutionResultSource{}, ErrNotFound
}

func (service *Service) ExecutionHistory(ctx context.Context, actor Actor, policyID string) ([]ExecutionHistoryItem, error) {
	policy, err := service.Get(ctx, actor, policyID)
	if err != nil {
		return nil, err
	}
	reader, ok := service.repo.(executionHistoryReader)
	if !ok {
		return nil, ErrAuthorityUnavailable
	}
	return reader.ListExecutionHistory(ctx, policy.TenantID, policy.LegalEntityID, policy.ID, 50)
}
func (repo *MemoryRepository) ListExecutionHistory(_ context.Context, tenant, entity, policy string, limit int) ([]ExecutionHistoryItem, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := []ExecutionHistoryItem{}
	appendReceipt := func(value ExecutionReceipt) {
		if value.TenantID == tenant && value.LegalEntityID == entity && value.PolicyID == policy {
			result = append(result, ExecutionHistoryItem{ID: value.ID, State: value.State, ResultBasis: receiptBasis(value), AssessmentVersion: value.AssessmentVersion, CreatedAt: value.CreatedAt})
		}
	}
	for _, value := range repo.executions {
		appendReceipt(value)
	}
	for _, value := range repo.executionFailures {
		appendReceipt(value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
