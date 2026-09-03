package activity

import (
	"context"
	"sync"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type MemoryExportRepository struct {
	mu       sync.RWMutex
	receipts map[string]ExportReceipt
	now      func() time.Time
}

func NewMemoryExportRepository() *MemoryExportRepository {
	return &MemoryExportRepository{receipts: map[string]ExportReceipt{}, now: time.Now}
}

func (r *MemoryExportRepository) CreateExport(_ context.Context, receipt ExportReceipt) (ExportReceipt, error) {
	if r == nil || receipt.TenantID == "" || receipt.RequestedBy == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	id, err := platformid.New("auditexp", 16)
	if err != nil {
		return ExportReceipt{}, err
	}
	receipt.ID = id
	receipt.Status = "GENERATING"
	receipt.CreatedAt = r.now().UTC()
	r.mu.Lock()
	r.receipts[id] = receipt
	r.mu.Unlock()
	return receipt, nil
}

func (r *MemoryExportRepository) CompleteExport(_ context.Context, receipt ExportReceipt) (ExportReceipt, error) {
	if r == nil || receipt.TenantID == "" || receipt.ID == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.receipts[receipt.ID]
	if !ok || current.TenantID != receipt.TenantID {
		return ExportReceipt{}, ErrNotFound
	}
	receipt.CreatedAt = current.CreatedAt
	receipt.ExpiresAt = current.ExpiresAt
	r.receipts[receipt.ID] = receipt
	return receipt, nil
}

func (r *MemoryExportRepository) FailExport(_ context.Context, tenantID, exportID, failureCode string) error {
	if r == nil || tenantID == "" || exportID == "" || failureCode == "" {
		return ErrExportInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	receipt, ok := r.receipts[exportID]
	if !ok || receipt.TenantID != tenantID {
		return ErrNotFound
	}
	now := r.now().UTC()
	receipt.Status = ExportStatusFailed
	receipt.FailureCode = failureCode
	receipt.CompletedAt = &now
	r.receipts[exportID] = receipt
	return nil
}

func (r *MemoryExportRepository) GetExport(_ context.Context, tenantID, exportID string) (ExportReceipt, error) {
	if r == nil || tenantID == "" || exportID == "" {
		return ExportReceipt{}, ErrExportInvalid
	}
	r.mu.RLock()
	receipt, ok := r.receipts[exportID]
	r.mu.RUnlock()
	if !ok || receipt.TenantID != tenantID {
		return ExportReceipt{}, ErrNotFound
	}
	return receipt, nil
}
