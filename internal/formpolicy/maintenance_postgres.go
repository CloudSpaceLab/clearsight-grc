//go:build postgres

package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (repo *PostgresRepository) SeedCompensations(ctx context.Context, now time.Time, limit int) (int, error) {
	if repo == nil || repo.pool == nil || now.IsZero() || limit < 1 || limit > maintenanceBatchLimit {
		return 0, ErrInvalid
	}
	tag, err := repo.pool.Exec(ctx, `
		INSERT INTO form_response_policy_maintenance_jobs(
			tenant_id,legal_entity_id,job_type,response_revision_id,policy_execution_id,matter_id,
			rollback_policy_id,rollback_policy_version,due_at,state,created_at,updated_at
		)
		SELECT candidate.tenant_id,candidate.legal_entity_id,'COMPENSATION',candidate.response_revision_id,
		       candidate.execution_id,candidate.matter_id,candidate.rollback_policy_id,candidate.rollback_policy_version,
		       $1,'READY',$1,$1
		FROM (
			SELECT p.tenant_id,p.legal_entity_id,p.id rollback_policy_id,p.version rollback_policy_version,
			       e.id execution_id,e.response_revision_id,e.matter_id,e.created_at
			FROM form_response_policy_definitions p
			JOIN form_response_policy_executions e
			  ON e.tenant_id=p.tenant_id AND e.legal_entity_id=p.legal_entity_id
			 AND e.policy_id=p.rollback_of_policy_id AND e.created_matter AND e.matter_id IS NOT NULL
			WHERE p.status='ACTIVE' AND p.rollback_of_policy_id IS NOT NULL
			  AND p.activated_at IS NOT NULL AND p.activated_at<=$1 AND e.created_at<=p.activated_at
			  AND NOT EXISTS (
				SELECT 1 FROM form_response_policy_compensations c
				WHERE c.tenant_id=p.tenant_id AND c.legal_entity_id=p.legal_entity_id
				  AND c.rollback_policy_id=p.id AND c.rollback_policy_version=p.version
				  AND c.original_execution_id=e.id
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM form_response_policy_maintenance_jobs job
				WHERE job.tenant_id=p.tenant_id AND job.legal_entity_id=p.legal_entity_id
				  AND job.job_type='COMPENSATION' AND job.rollback_policy_id=p.id
				  AND job.rollback_policy_version=p.version AND job.policy_execution_id=e.id
			  )
			ORDER BY e.created_at,e.id,p.id LIMIT $2
		) candidate
		ON CONFLICT (tenant_id,legal_entity_id,rollback_policy_id,rollback_policy_version,policy_execution_id) WHERE job_type='COMPENSATION'
		DO NOTHING`, now.UTC(), limit)
	if err != nil {
		return 0, normalizePostgresError(err)
	}
	return int(tag.RowsAffected()), nil
}

func (repo *PostgresRepository) ClaimCompensations(ctx context.Context, workerID string, now time.Time, lease time.Duration, limit int) ([]CompensationCandidate, error) {
	if repo == nil || repo.pool == nil || strings.TrimSpace(workerID) == "" || now.IsZero() || lease <= 0 || limit < 1 || limit > maintenanceBatchLimit {
		return nil, ErrInvalid
	}
	rows, err := repo.pool.Query(ctx, `WITH due AS (
		SELECT id FROM form_response_policy_maintenance_jobs
		WHERE job_type='COMPENSATION' AND due_at<=$1
		  AND (state='READY' OR (state='CLAIMED' AND lease_until<$1))
		ORDER BY due_at,id FOR UPDATE SKIP LOCKED LIMIT $2
	), claimed AS (
		UPDATE form_response_policy_maintenance_jobs job
		SET state='CLAIMED',locked_by=$3,lease_until=$1+$4::interval,attempts=attempts+1,updated_at=$1
		FROM due WHERE job.id=due.id RETURNING job.*
	)
	SELECT claimed.id::text,`+postgresPolicyColumns+`,
		e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.policy_id::text,e.policy_version,
		e.automation_policy_id::text,e.automation_policy_version,e.response_revision_id::text,e.state,
		COALESCE(e.matter_id::text,''),e.reason_code,e.created_matter,e.created_at
	FROM claimed
	JOIN form_response_policy_definitions p ON p.id=claimed.rollback_policy_id AND p.tenant_id=claimed.tenant_id AND p.legal_entity_id=claimed.legal_entity_id AND p.version=claimed.rollback_policy_version
	JOIN form_response_policy_executions e ON e.id=claimed.policy_execution_id AND e.tenant_id=claimed.tenant_id AND e.legal_entity_id=claimed.legal_entity_id
	ORDER BY claimed.due_at,claimed.id`, now.UTC(), limit, strings.TrimSpace(workerID), lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]CompensationCandidate, 0, limit)
	for rows.Next() {
		value, scanErr := scanPostgresCompensationCandidate(rows, true)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (repo *PostgresRepository) CompleteCompensation(ctx context.Context, jobID, workerID string, now time.Time) error {
	if repo == nil || repo.pool == nil || strings.TrimSpace(jobID) == "" || strings.TrimSpace(workerID) == "" || now.IsZero() {
		return ErrInvalid
	}
	tag, err := repo.pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET state='COMPLETED',locked_by=NULL,lease_until=NULL,last_error='',updated_at=$3 WHERE id=$1::uuid AND job_type='COMPENSATION' AND state='CLAIMED' AND locked_by=$2 AND lease_until>=clock_timestamp()`, jobID, strings.TrimSpace(workerID), now.UTC())
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (repo *PostgresRepository) RetryCompensation(ctx context.Context, jobID, workerID string, now time.Time, reason string) error {
	if repo == nil || repo.pool == nil || strings.TrimSpace(jobID) == "" || strings.TrimSpace(workerID) == "" || now.IsZero() {
		return ErrInvalid
	}
	if len(reason) > 1000 {
		reason = reason[:1000]
	}
	tag, err := repo.pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs
		SET state='READY',due_at=$3+LEAST(interval '1 hour',interval '30 seconds'*power(2,LEAST(attempts,7))),
		    locked_by=NULL,lease_until=NULL,last_error=$4,updated_at=$3
		WHERE id=$1::uuid AND job_type='COMPENSATION' AND state='CLAIMED' AND locked_by=$2 AND lease_until>=clock_timestamp()`, jobID, strings.TrimSpace(workerID), now.UTC(), reason)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (repo *PostgresRepository) ListPendingCompensations(ctx context.Context, now time.Time, limit int) ([]CompensationCandidate, error) {
	if repo == nil || repo.pool == nil || now.IsZero() || limit < 1 || limit > maintenanceBatchLimit {
		return nil, ErrInvalid
	}
	rows, err := repo.pool.Query(ctx, `SELECT `+postgresPolicyColumns+`,
		e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.policy_id::text,e.policy_version,
		e.automation_policy_id::text,e.automation_policy_version,e.response_revision_id::text,e.state,
		COALESCE(e.matter_id::text,''),e.reason_code,e.created_matter,e.created_at
		FROM form_response_policy_definitions p
		JOIN form_response_policy_executions e
		  ON e.tenant_id=p.tenant_id AND e.legal_entity_id=p.legal_entity_id
		 AND e.policy_id=p.rollback_of_policy_id AND e.created_matter AND e.matter_id IS NOT NULL
		JOIN matters m ON m.id=e.matter_id AND m.tenant_id=e.tenant_id AND m.legal_entity_id=e.legal_entity_id
		WHERE p.status='ACTIVE' AND p.rollback_of_policy_id IS NOT NULL
		  AND p.activated_at IS NOT NULL AND p.activated_at<=$1 AND e.created_at<=p.activated_at
		  AND NOT EXISTS (
			SELECT 1 FROM form_response_policy_compensations c
			WHERE c.tenant_id=p.tenant_id AND c.legal_entity_id=p.legal_entity_id
			  AND c.rollback_policy_id=p.id AND c.rollback_policy_version=p.version
			  AND c.original_execution_id=e.id
		  )
		ORDER BY e.created_at,e.id,p.id LIMIT $2`, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]CompensationCandidate, 0, limit)
	for rows.Next() {
		value, scanErr := scanPostgresCompensationCandidate(rows, false)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func scanPostgresCompensationCandidate(row postgresRow, includesJobID bool) (CompensationCandidate, error) {
	var value CompensationCandidate
	var policy Policy
	var execution ExecutionReceipt
	var eligibility, action, blast, outcome []byte
	var rollout, status string
	targets := make([]any, 0, 46)
	if includesJobID {
		targets = append(targets, &value.JobID)
	}
	targets = append(targets, postgresPolicyScanTargets(&policy, &eligibility, &action, &blast, &outcome, &rollout, &status)...)
	targets = append(targets, &execution.ID, &execution.TenantID, &execution.LegalEntityID, &execution.PolicyID, &execution.PolicyVersion,
		&execution.AutomationPolicyID, &execution.AutomationPolicyVersion, &execution.ResponseRevisionID, &execution.State,
		&execution.MatterID, &execution.ReasonCode, &execution.CreatedMatter, &execution.CreatedAt)
	if err := row.Scan(targets...); err != nil {
		return CompensationCandidate{}, err
	}
	decoded, err := decodePostgresPolicy(policy, eligibility, action, blast, outcome, rollout, status)
	if err != nil {
		return CompensationCandidate{}, err
	}
	value.RollbackPolicy, value.OriginalExecution = decoded, execution
	return value, nil
}

func (repo *PostgresRepository) SeedReconciliation(ctx context.Context, now time.Time, limit int) (int, error) {
	if repo == nil || repo.pool == nil || now.IsZero() || limit < 1 || limit > maintenanceBatchLimit {
		return 0, ErrInvalid
	}
	tag, err := repo.pool.Exec(ctx, `
		INSERT INTO form_response_policy_maintenance_jobs(
			tenant_id,legal_entity_id,job_type,response_revision_id,due_at,state,created_at,updated_at
		)
		SELECT candidate.tenant_id,candidate.legal_entity_id,'RECONCILE',candidate.response_revision_id,$1,'READY',$1,$1
		FROM (
			SELECT r.tenant_id,r.legal_entity_id,r.id AS response_revision_id,min(r.created_at) AS completed_at
			FROM capture_response_revisions r
			JOIN capture_form_distributions d
			  ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
			JOIN form_response_policy_definitions p
			  ON p.tenant_id=r.tenant_id AND p.legal_entity_id=r.legal_entity_id
			 AND p.form_template_id=d.form_template_id AND p.form_template_version=d.form_template_version
			 AND p.status='ACTIVE'
			 AND p.activated_at IS NOT NULL AND p.activated_at<=r.created_at
			 AND (p.effective_from IS NULL OR p.effective_from<=r.created_at)
			 AND (p.effective_until IS NULL OR p.effective_until>r.created_at)
			WHERE r.state IN ('FINAL','PROVISIONAL') AND r.score_state IN ('FINAL','PROVISIONAL')
			  AND NOT EXISTS (
				SELECT 1 FROM form_response_policy_executions execution
				WHERE execution.tenant_id=r.tenant_id AND execution.legal_entity_id=r.legal_entity_id
				  AND execution.policy_id=p.id AND execution.policy_version=p.version
				  AND execution.response_revision_id=r.id
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM form_response_policy_maintenance_jobs job
				WHERE job.tenant_id=r.tenant_id AND job.legal_entity_id=r.legal_entity_id
				  AND job.job_type='RECONCILE' AND job.response_revision_id=r.id
			  )
			GROUP BY r.tenant_id,r.legal_entity_id,r.id
			ORDER BY min(r.created_at),r.id
			LIMIT $2
		) candidate
		ON CONFLICT (tenant_id,legal_entity_id,response_revision_id) WHERE job_type='RECONCILE'
		DO UPDATE SET state='READY',due_at=EXCLUDED.due_at,locked_by=NULL,lease_until=NULL,last_error='',updated_at=EXCLUDED.updated_at
		WHERE form_response_policy_maintenance_jobs.state='COMPLETED'`, now.UTC(), limit)
	if err != nil {
		return 0, normalizePostgresError(err)
	}
	return int(tag.RowsAffected()), nil
}

func (repo *PostgresRepository) ClaimReconciliation(ctx context.Context, workerID string, now time.Time, lease time.Duration, limit int) ([]ScoredResponseEvent, error) {
	if repo == nil || repo.pool == nil || strings.TrimSpace(workerID) == "" || now.IsZero() || lease <= 0 || limit < 1 || limit > maintenanceBatchLimit {
		return nil, ErrInvalid
	}
	rows, err := repo.pool.Query(ctx, `
		WITH due AS (
			SELECT id FROM form_response_policy_maintenance_jobs
			WHERE job_type='RECONCILE' AND due_at<=$1
			  AND (state='READY' OR (state='CLAIMED' AND lease_until<$1))
			ORDER BY due_at,id FOR UPDATE SKIP LOCKED LIMIT $2
		), claimed AS (
			UPDATE form_response_policy_maintenance_jobs job
			SET state='CLAIMED',locked_by=$3,lease_until=$1+$4::interval,attempts=attempts+1,updated_at=$1
			FROM due WHERE job.id=due.id
			RETURNING job.id,job.tenant_id,job.response_revision_id,job.created_at
		)
		SELECT claimed.id::text,claimed.tenant_id::text,claimed.response_revision_id::text,claimed.created_at
		FROM claimed ORDER BY claimed.created_at,claimed.id`, now.UTC(), limit, strings.TrimSpace(workerID), lease.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]ScoredResponseEvent, 0, limit)
	for rows.Next() {
		var event ScoredResponseEvent
		if err := rows.Scan(&event.ID, &event.TenantID, &event.ResponseRevisionID, &event.OccurredAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (repo *PostgresRepository) CompleteReconciliation(ctx context.Context, eventID, workerID string, now time.Time) error {
	if repo == nil || repo.pool == nil || strings.TrimSpace(eventID) == "" || strings.TrimSpace(workerID) == "" || now.IsZero() {
		return ErrInvalid
	}
	tag, err := repo.pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET state='COMPLETED',locked_by=NULL,lease_until=NULL,last_error='',updated_at=$3 WHERE id=$1::uuid AND job_type='RECONCILE' AND state='CLAIMED' AND locked_by=$2 AND lease_until>=clock_timestamp()`, eventID, strings.TrimSpace(workerID), now.UTC())
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (repo *PostgresRepository) RetryReconciliation(ctx context.Context, eventID, workerID string, now time.Time, reason string, keepRetrying bool) error {
	if len(reason) > 1000 {
		reason = reason[:1000]
	}
	tag, err := repo.pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs
		SET state=CASE WHEN $5 THEN 'READY' WHEN attempts>=5 THEN 'FAILED' ELSE 'READY' END,
		    due_at=$3+CASE WHEN $5 THEN LEAST(interval '1 hour',interval '30 seconds'*power(2,LEAST(attempts,7))) ELSE interval '30 seconds' END,
		    locked_by=NULL,lease_until=NULL,last_error=$4,updated_at=$3
		WHERE id=$1::uuid AND job_type='RECONCILE' AND state='CLAIMED' AND locked_by=$2 AND lease_until>=clock_timestamp()`, eventID, strings.TrimSpace(workerID), now.UTC(), reason, keepRetrying)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (repo *PostgresRepository) retryMaintenanceJob(ctx context.Context, jobID, workerID string, now time.Time, reason string) error {
	if len(reason) > 1000 {
		reason = reason[:1000]
	}
	tag, err := repo.pool.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET state=CASE WHEN attempts>=5 THEN 'FAILED' ELSE 'READY' END,due_at=$3+interval '30 seconds',locked_by=NULL,lease_until=NULL,last_error=$4,updated_at=$3 WHERE id=$1::uuid AND state='CLAIMED' AND locked_by=$2 AND lease_until>=clock_timestamp()`, jobID, strings.TrimSpace(workerID), now.UTC(), reason)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

type claimedOutcomeCheck struct {
	ID string
}

func (repo *PostgresRepository) MaintainOutcomeChecks(ctx context.Context, workerID string, now time.Time, lease time.Duration, limit int) (int, error) {
	if repo == nil || repo.pool == nil || strings.TrimSpace(workerID) == "" || now.IsZero() || lease <= 0 || limit < 1 || limit > maintenanceBatchLimit {
		return 0, ErrInvalid
	}
	rows, err := repo.pool.Query(ctx, `
		WITH due AS (
			SELECT id FROM form_response_policy_maintenance_jobs
			WHERE job_type='OUTCOME_CHECK' AND due_at<=$1
			  AND (state='READY' OR (state='CLAIMED' AND lease_until<$1))
			ORDER BY due_at,id FOR UPDATE SKIP LOCKED LIMIT $2
		), claimed AS (
			UPDATE form_response_policy_maintenance_jobs job
			SET state='CLAIMED',locked_by=$3,lease_until=$1+$4::interval,attempts=attempts+1,updated_at=$1
			FROM due WHERE job.id=due.id RETURNING job.id
		)
		SELECT id::text FROM claimed ORDER BY id`, now.UTC(), limit, strings.TrimSpace(workerID), lease.String())
	if err != nil {
		return 0, err
	}
	checks := make([]claimedOutcomeCheck, 0, limit)
	for rows.Next() {
		var check claimedOutcomeCheck
		if err := rows.Scan(&check.ID); err != nil {
			rows.Close()
			return 0, err
		}
		checks = append(checks, check)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	processed := 0
	var failures []error
	for _, check := range checks {
		if ctx.Err() != nil {
			return processed, errors.Join(append(failures, ctx.Err())...)
		}
		if err := repo.processOutcomeCheck(ctx, check.ID, workerID, now.UTC()); err != nil {
			failures = append(failures, errors.Join(err, repo.retryMaintenanceJob(ctx, check.ID, workerID, now, err.Error())))
		}
		processed++
	}
	return processed, errors.Join(failures...)
}

func (repo *PostgresRepository) processOutcomeCheck(ctx context.Context, jobID, workerID string, now time.Time) error {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var episodeID, matterID, responseID string
	var matterStatus continuity.MatterStatus
	var closedAt *time.Time
	var outcomeRaw []byte
	err = tx.QueryRow(ctx, `
		SELECT job.adverse_episode_id::text,job.matter_id::text,job.response_revision_id::text,m.status,m.closed_at,p.outcome_contract
		FROM form_response_policy_maintenance_jobs job
		JOIN form_response_policy_executions execution
		  ON execution.id=job.policy_execution_id AND execution.tenant_id=job.tenant_id AND execution.legal_entity_id=job.legal_entity_id
		JOIN form_response_policy_definitions p
		  ON p.id=execution.policy_id AND p.tenant_id=execution.tenant_id AND p.legal_entity_id=execution.legal_entity_id
		JOIN form_response_policy_adverse_episodes episode
		  ON episode.id=job.adverse_episode_id AND episode.tenant_id=job.tenant_id AND episode.legal_entity_id=job.legal_entity_id
		JOIN matters m ON m.id=job.matter_id AND m.tenant_id=job.tenant_id AND m.legal_entity_id=job.legal_entity_id
		WHERE job.id=$1::uuid AND job.job_type='OUTCOME_CHECK' AND job.state='CLAIMED'
		  AND job.locked_by=$2 AND job.lease_until>=clock_timestamp()
		FOR UPDATE OF job,episode`, jobID, strings.TrimSpace(workerID)).Scan(&episodeID, &matterID, &responseID, &matterStatus, &closedAt, &outcomeRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	var outcome OutcomeContract
	if json.Unmarshal(outcomeRaw, &outcome) != nil || outcome.CheckAfterMinutes < 1 {
		return ErrInvalid
	}
	if matterStatus != continuity.MatterClosed || closedAt == nil {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET state='READY',due_at=$2,locked_by=NULL,lease_until=NULL,last_error='MATTER_NOT_VERIFIED_CLOSED',updated_at=$3 WHERE id=$1::uuid AND state='CLAIMED' AND locked_by=$4 AND lease_until>=clock_timestamp()`, jobID, now.Add(time.Duration(outcome.CheckAfterMinutes)*time.Minute), now, strings.TrimSpace(workerID))
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrConflict
		}
		return tx.Commit(ctx)
	}
	var episodeVersion int64
	err = tx.QueryRow(ctx, `UPDATE form_response_policy_adverse_episodes SET state='CLOSED',closed_at=$2,updated_at=$3,record_version=record_version+1 WHERE id=$1::uuid AND state='OPEN' RETURNING record_version`, episodeID, closedAt, now).Scan(&episodeVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, `SELECT record_version FROM form_response_policy_adverse_episodes WHERE id=$1::uuid AND state='CLOSED'`, episodeID).Scan(&episodeVersion); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"episode_id": episodeID, "matter_id": matterID, "response_revision_id": responseID, "state": EpisodeClosed, "version": episodeVersion})
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) SELECT tenant_id,'FORM_RESPONSE_POLICY_EPISODE',$2::uuid,'FORM_RESPONSE_POLICY_EPISODE_CLOSED',$3::jsonb,$4,$4,$4 FROM form_response_policy_maintenance_jobs WHERE id=$1::uuid ON CONFLICT DO NOTHING`, jobID, episodeID, payload, now); err != nil {
		return err
	}
	if tag, updateErr := tx.Exec(ctx, `UPDATE form_response_policy_maintenance_jobs SET state='COMPLETED',locked_by=NULL,lease_until=NULL,last_error='',updated_at=$2 WHERE id=$1::uuid AND state='CLAIMED' AND locked_by=$3 AND lease_until>=clock_timestamp()`, jobID, now, strings.TrimSpace(workerID)); updateErr != nil {
		return updateErr
	} else if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return tx.Commit(ctx)
}

var _ MaintenanceRepository = (*PostgresRepository)(nil)
