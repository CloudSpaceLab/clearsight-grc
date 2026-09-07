//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ClaimArtifactScanJobs(ctx context.Context, worker string, now time.Time, limit int) ([]ArtifactScanJob, error) {
	if worker == "" || limit <= 0 {
		return nil, ErrArtifactScannerUnavailable
	}
	if limit > 10 {
		limit = 10
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT j.artifact_id::text,j.tenant_id::text,j.sha256,j.size_bytes,a.request_id::text,a.storage_key,j.attempt
 FROM capture_artifact_scan_jobs j JOIN capture_artifacts a ON a.id=j.artifact_id AND a.tenant_id=j.tenant_id
 WHERE (j.state='PENDING' AND j.next_attempt_at<=$1) OR (j.state='RUNNING' AND j.lease_until<=$1)
 ORDER BY j.next_attempt_at,j.artifact_id LIMIT $2 FOR UPDATE OF j SKIP LOCKED`, now, limit)
	if err != nil {
		return nil, err
	}
	jobs := []ArtifactScanJob{}
	for rows.Next() {
		var job ArtifactScanJob
		if err := rows.Scan(&job.Artifact.ID, &job.Artifact.TenantID, &job.Artifact.SHA256, &job.Artifact.SizeBytes, &job.Artifact.RequestID, &job.Artifact.StorageKey, &job.Attempt); err != nil {
			rows.Close()
			return nil, err
		}
		jobs = append(jobs, job)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	claimed := make([]ArtifactScanJob, 0, len(jobs))
	for _, job := range jobs {
		if job.Attempt >= artifactScanMaxAttempts {
			if _, err := tx.Exec(ctx, `UPDATE capture_artifact_scan_jobs SET state='FAILED',failure_code='ATTEMPTS_EXHAUSTED',worker_id='',lease_until=NULL WHERE artifact_id=$1::uuid`, job.Artifact.ID); err != nil {
				return nil, err
			}
			continue
		}
		job.Attempt++
		job.State = "RUNNING"
		job.WorkerID = worker
		job.LeaseUntil = now.Add(artifactScanLease)
		if _, err := tx.Exec(ctx, `UPDATE capture_artifact_scan_jobs SET state='RUNNING',attempt=$2,worker_id=$3,lease_until=$4 WHERE artifact_id=$1::uuid`, job.Artifact.ID, job.Attempt, worker, job.LeaseUntil); err != nil {
			return nil, err
		}
		claimed = append(claimed, job)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r *PostgresRepository) CompleteArtifactScan(ctx context.Context, claim ArtifactScanJob, receipt ArtifactScanReceipt, now time.Time) error {
	if !validScanReceipt(claim, receipt, now) {
		return ErrArtifactScanLease
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current ArtifactScanJob
	var artifactStatus ArtifactStatus
	var digest string
	var size int64
	err = tx.QueryRow(ctx, `SELECT j.artifact_id::text,j.tenant_id::text,j.sha256,j.size_bytes,j.state,j.attempt,j.worker_id,j.lease_until,a.status,a.sha256,a.size_bytes
 FROM capture_artifact_scan_jobs j JOIN capture_artifacts a ON a.id=j.artifact_id AND a.tenant_id=j.tenant_id
 WHERE j.artifact_id=$1::uuid AND j.tenant_id=$2::uuid AND j.state='RUNNING' FOR UPDATE OF j,a`, claim.Artifact.ID, claim.Artifact.TenantID).Scan(&current.Artifact.ID, &current.Artifact.TenantID, &current.Artifact.SHA256, &current.Artifact.SizeBytes, &current.State, &current.Attempt, &current.WorkerID, &current.LeaseUntil, &artifactStatus, &digest, &size)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrArtifactScanLease
	}
	if err != nil {
		return err
	}
	// Re-read database time after acquiring locks; a queued completion cannot
	// use a timestamp captured before a blocking transaction to revive its lease.
	var databaseNow time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&databaseNow); err != nil {
		return err
	}
	if databaseNow.After(now) {
		now = databaseNow
	}
	if current.WorkerID != claim.WorkerID || current.Attempt != claim.Attempt || !current.LeaseUntil.After(now) || artifactStatus != ArtifactStoredUnscanned || digest != current.Artifact.SHA256 || size != current.Artifact.SizeBytes || !validScanReceipt(current, receipt, now) {
		return ErrArtifactScanLease
	}
	updated, status := scanOutcome(current, receipt, now)
	var receiptID string
	if err := tx.QueryRow(ctx, `INSERT INTO capture_artifact_scan_receipts(artifact_id,tenant_id,sha256,size_bytes,attempt,verdict,scanner,scanner_version,inspected_at,failure_code) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id::text`, receipt.ArtifactID, current.Artifact.TenantID, receipt.SHA256, receipt.SizeBytes, receipt.Attempt, receipt.Verdict, receipt.Scanner, receipt.Version, receipt.InspectedAt, receipt.FailureCode).Scan(&receiptID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_artifacts SET status=$2,inspected_at=$3,inspection_reference=$4 WHERE id=$1::uuid`, current.Artifact.ID, status, now, receiptID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_artifact_scan_jobs SET state=$2,worker_id='',lease_until=NULL,next_attempt_at=$3,failure_code=$4 WHERE artifact_id=$1::uuid`, current.Artifact.ID, updated.State, updated.NextAttemptAt, updated.FailureCode); err != nil {
		return err
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'CAPTURE_ARTIFACT',$2::uuid,'ArtifactInspectionRecorded',$3::jsonb,$4,$4,$4)`, current.Artifact.TenantID, current.Artifact.ID, payload, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ ArtifactScanRepository = (*PostgresRepository)(nil)

func (r *PostgresRepository) ArtifactScanQueueHealth(ctx context.Context) (workflowruntime.QueueHealth, error) {
	var health workflowruntime.QueueHealth
	err := r.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE state IN ('PENDING','RUNNING')),count(*) FILTER (WHERE state='FAILED'),COALESCE(max(attempt),0),min(next_attempt_at) FILTER (WHERE state IN ('PENDING','RUNNING')) FROM capture_artifact_scan_jobs WHERE state<>'COMPLETED'`).Scan(&health.Pending, &health.Terminal, &health.HighestAttempts, &health.OldestPending)
	return health, err
}
