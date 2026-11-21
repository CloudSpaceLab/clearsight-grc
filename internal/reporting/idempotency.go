package reporting

import (
	"context"
	"strings"
)

// runReceiptRepository atomically records the command receipt with its run.
type runReceiptRepository interface {
	FindRunReceipt(context.Context, ReportScope, string, string, string) (ReportRun, error)
	CreateRunOnce(context.Context, ReportScope, ReportRun, string, string) (ReportRun, error)
}

func validRunReceipt(key, fingerprint, principal string) bool {
	if !isUUID(key) || strings.TrimSpace(principal) == "" || len(fingerprint) != 64 {
		return false
	}
	for _, c := range fingerprint {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

type memoryRunReceipt struct{ fingerprint, runID string }

func runReceiptKey(scope ReportScope, principal, key string) string {
	return scope.TenantID + "\x00" + scope.LegalEntityID + "\x00" + principal + "\x00" + key
}

func (r *MemoryRepository) FindRunReceipt(ctx context.Context, scope ReportScope, principal, key, fingerprint string) (ReportRun, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if r == nil || ctx == nil || validateReportScope(scope) != nil || !validRunReceipt(key, fingerprint, principal) {
		return ReportRun{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.findRunReceiptLocked(scope, principal, key, fingerprint)
}
func (r *MemoryRepository) findRunReceiptLocked(scope ReportScope, principal, key, fingerprint string) (ReportRun, error) {
	receipt, ok := r.receipts[runReceiptKey(scope, principal, key)]
	if !ok {
		return ReportRun{}, ErrNotFound
	}
	if receipt.fingerprint != fingerprint {
		return ReportRun{}, ErrConflict
	}
	run, ok := r.runs[receipt.runID]
	if !ok {
		return ReportRun{}, ErrInvalid
	}
	return cloneReportRun(run), nil
}
func (r *MemoryRepository) CreateRunOnce(ctx context.Context, scope ReportScope, run ReportRun, key, fingerprint string) (ReportRun, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if r == nil || ctx == nil || validateReportScope(scope) != nil || !validRunReceipt(key, fingerprint, run.RequestedByRef) {
		return ReportRun{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if previous, err := r.findRunReceiptLocked(scope, run.RequestedByRef, key, fingerprint); err == nil {
		return previous, nil
	} else if err != ErrNotFound {
		return ReportRun{}, err
	}
	created, err := r.createRunLocked(ctx, scope, run)
	if err != nil {
		return ReportRun{}, err
	}
	r.receipts[runReceiptKey(scope, run.RequestedByRef, key)] = memoryRunReceipt{fingerprint, created.ID}
	return created, nil
}
