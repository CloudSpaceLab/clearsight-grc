//go:build postgres

package reporting

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgresReportRunIdempotencyConcurrent(t *testing.T) {
	f := newReportingPostgresFixture(t)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM report_run_receipts WHERE tenant_id=$1::uuid`, f.tenantID)
	})
	definition, _ := f.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	seed := f.createRun(t, definition, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	key, hash := f.newID(), strings.Repeat("a", 64)
	results := make(chan struct {
		run ReportRun
		err error
	}, 2)
	for range 2 {
		run := seed
		run.ID = f.newID()
		go func() {
			value, err := f.repository.CreateRunOnce(f.ctx, f.scope, run, key, hash)
			results <- struct {
				run ReportRun
				err error
			}{value, err}
		}()
	}
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.run.ID != b.run.ID {
		t.Fatalf("concurrent receipts: %+v %+v", a, b)
	}
	var receipts, events int
	if err := f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM report_run_receipts WHERE tenant_id=$1::uuid AND idempotency_key=$2::uuid), (SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$3::uuid AND event_type='ReportRunQueued')`, f.tenantID, key, a.run.ID).Scan(&receipts, &events); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || events != 1 {
		t.Fatalf("receipts=%d outbox=%d", receipts, events)
	}
	if _, err := f.repository.FindRunReceipt(f.ctx, f.scope, f.performerID, key, strings.Repeat("b", 64)); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed request: %v", err)
	}
	if _, err := f.repository.FindRunReceipt(f.ctx, f.scope, f.makerID, key, hash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other actor: %v", err)
	}
}

func TestReportRunReceiptMigrationRefusesDataLoss(t *testing.T) {
	f := newReportingPostgresFixture(t)
	tx, err := f.pool.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(f.ctx)
	if _, err := tx.Exec(f.ctx, `CREATE TEMP TABLE report_run_receipts(id integer)`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(f.ctx, `INSERT INTO report_run_receipts VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000145_report_run_receipts.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.TrimSpace(string(down))
	sql = strings.TrimPrefix(sql, "BEGIN;")
	sql = strings.TrimSuffix(strings.TrimSpace(sql), "COMMIT;")
	if _, err := tx.Exec(f.ctx, sql, pgx.QueryExecModeSimpleProtocol); err == nil || !strings.Contains(err.Error(), "cannot remove report run receipts") {
		t.Fatalf("rollback guard: %v", err)
	}
}

func TestPostgresReportRunReceiptFailureRollsBackRunAndOutbox(t *testing.T) {
	f := newReportingPostgresFixture(t)
	definition, _ := f.proposal(t, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	seed := f.createRun(t, definition, DatasetProcessingActivityExceptions, ScopeLegalEntity, "", emptyReportFilter())
	run := seed
	run.ID = f.newID()
	key := f.newID()
	installFailingTrigger(t, f, "report_run_receipts", "INSERT", "receipt", f.tenantID)
	if _, err := f.repository.CreateRunOnce(f.ctx, f.scope, run, key, strings.Repeat("a", 64)); err == nil {
		t.Fatal("receipt insertion unexpectedly succeeded")
	}
	var runs, events, receipts int
	if err := f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM report_runs WHERE id=$1::uuid), (SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid), (SELECT count(*) FROM report_run_receipts WHERE tenant_id=$2::uuid AND idempotency_key=$3::uuid)`, run.ID, f.tenantID, key).Scan(&runs, &events, &receipts); err != nil {
		t.Fatal(err)
	}
	if runs != 0 || events != 0 || receipts != 0 {
		t.Fatalf("rollback leaked state: runs=%d events=%d receipts=%d", runs, events, receipts)
	}
}
