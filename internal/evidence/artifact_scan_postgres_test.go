//go:build postgres

package evidence

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresArtifactScanTransactionAndLeases(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured; artifact scan transactions were not exercised")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tenant, _ := id.NewUUIDv7()
	entity, _ := id.NewUUIDv7()
	request, _ := id.NewUUIDv7()
	artifactID, _ := id.NewUUIDv7()
	now := time.Now().UTC().Truncate(time.Microsecond)
	defer func() {
		for _, statement := range []string{
			`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
			`DELETE FROM capture_artifact_scan_receipts WHERE tenant_id=$1::uuid`,
			`DELETE FROM capture_artifact_scan_jobs WHERE tenant_id=$1::uuid`,
			`DELETE FROM capture_artifacts WHERE tenant_id=$1::uuid`,
			`DELETE FROM capture_requests WHERE tenant_id=$1::uuid`,
			`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`,
			`DELETE FROM tenants WHERE id=$1::uuid`,
		} {
			_, _ = pool.Exec(ctx, statement, tenant)
		}
	}()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$1,'Artifact inspection test');
 INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'SCAN','Inspection test','NG',$4);
 INSERT INTO capture_requests(id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,fields,status)
 VALUES($3::uuid,$1::uuid,$2::uuid,'PROGRAM','test','Review evidence','Evidence inspection test','Assigned evidence','INTERNAL','INTERNAL',1,$5,'[{"id":"file","type":"file","label":"Evidence"}]','READY')`, pgx.QueryExecModeSimpleProtocol, tenant, entity, request, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	a := Artifact{ID: artifactID, TenantID: tenant, RequestID: request, FileName: "evidence.txt", MediaType: "text/plain", SHA256: strings.Repeat("a", 64), SizeBytes: 12, StorageKey: "scan-test/" + artifactID, Status: ArtifactStoredUnscanned, CreatedAt: now}
	if _, err := repo.CreateArtifact(ctx, a); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_artifact_scan_jobs WHERE artifact_id=$1::uuid`, artifactID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("transactional job count=%d err=%v", count, err)
	}
	// Lock unrelated jobs so this global worker test never claims another fixture.
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback(ctx)
	rows, err := lock.Query(ctx, `SELECT artifact_id FROM capture_artifact_scan_jobs WHERE artifact_id<>$1::uuid FOR UPDATE`, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
	}
	rows.Close()
	jobs, err := repo.ClaimArtifactScanJobs(ctx, "scanner-test", now, 1)
	if err != nil || len(jobs) != 1 || jobs[0].Artifact.ID != artifactID {
		t.Fatalf("claim=%+v err=%v", jobs, err)
	}
	job := jobs[0]
	if next, err := repo.ClaimArtifactScanJobs(ctx, "other-worker", now, 1); err != nil || len(next) != 0 {
		t.Fatalf("concurrent claim=%v %v", next, err)
	}
	receipt := ArtifactScanReceipt{ArtifactID: artifactID, SHA256: a.SHA256, SizeBytes: a.SizeBytes, Attempt: job.Attempt, Verdict: "CLEAN", Scanner: "test-fixture", Version: "1", InspectedAt: now}
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now.Add(time.Minute)); !errors.Is(err, ErrArtifactScanLease) {
		t.Fatalf("expired lease=%v", err)
	}
	// An outbox constraint failure must roll back both the receipt and availability.
	if _, err := pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT artifact_scan_test_failure CHECK (aggregate_id<>'`+artifactID+`'::uuid) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT IF EXISTS artifact_scan_test_failure`)
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now); err == nil {
		t.Fatal("expected outbox failure")
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT status FROM capture_artifacts WHERE id=$1::uuid`, artifactID).Scan(&state); err != nil || state != "STORED_UNSCANNED" {
		t.Fatalf("rollback state=%s err=%v", state, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_artifact_scan_receipts WHERE artifact_id=$1::uuid`, artifactID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rollback receipts=%d err=%v", count, err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT artifact_scan_test_failure`); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now); !errors.Is(err, ErrArtifactScanLease) {
		t.Fatalf("replay=%v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM capture_artifacts WHERE id=$1::uuid`, artifactID).Scan(&state); err != nil || state != "AVAILABLE" {
		t.Fatalf("state=%s err=%v", state, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='ArtifactInspectionRecorded'`, artifactID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbox count=%d err=%v", count, err)
	}
}
