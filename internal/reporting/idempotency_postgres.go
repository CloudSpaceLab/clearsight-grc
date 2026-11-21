//go:build postgres || load

package reporting

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type receiptQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *PostgresRepository) FindRunReceipt(ctx context.Context, scope ReportScope, principal, key, fingerprint string) (ReportRun, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if r == nil || r.pool == nil || validateReportScope(scope) != nil || !validRunReceipt(key, fingerprint, principal) {
		return ReportRun{}, ErrInvalid
	}
	return findPostgresRunReceipt(ctx, r.pool, scope, principal, key, fingerprint)
}

func findPostgresRunReceipt(ctx context.Context, query receiptQueryer, scope ReportScope, principal, key, fingerprint string) (ReportRun, error) {
	var stored, runID string
	err := query.QueryRow(ctx, `SELECT request_fingerprint,run_id::text FROM report_run_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND principal_id=$3 AND idempotency_key=$4::uuid`, scope.TenantID, scope.LegalEntityID, principal, key).Scan(&stored, &runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrNotFound
	}
	if err != nil {
		return ReportRun{}, err
	}
	if stored != fingerprint {
		return ReportRun{}, ErrConflict
	}
	run, err := scanRun(query.QueryRow(ctx, `SELECT `+runProjection+` FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid AND r.requested_by_ref=$4`, scope.TenantID, scope.LegalEntityID, runID, principal))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrInvalid
	}
	return run, err
}
