//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5"
)

const assessmentSetupJobProjection = `
	j.id::text,j.tenant_id::text,j.legal_entity_id::text,j.assessment_id::text,j.job_type,j.dedupe_key,j.state,
	j.attempts,j.available_at,COALESCE(j.lease_token::text,''),j.lease_expires_at,j.last_failure_code,j.created_at,j.updated_at`

func (r *PostgresRepository) ClaimAssessmentSetupJobs(ctx context.Context, workerID string, now time.Time, lease time.Duration, maxAttempts, limit int) ([]AssessmentSetupJob, error) {
	if workerID == "" || lease <= 0 || maxAttempts < 1 || maxAttempts > 20 {
		return nil, ErrInvalid
	}
	limit = boundedAssessmentJobLimit(limit)
	now = now.UTC()
	expires := now.Add(lease)
	if _, err := r.pool.Exec(ctx, `
		WITH exhausted AS (
			SELECT id FROM third_party_assessment_jobs
			WHERE job_type='SETUP_REVIEW' AND attempts >= $2
			  AND (state='READY' OR (state='LEASED' AND lease_expires_at<=$1))
			ORDER BY available_at,id
			FOR UPDATE SKIP LOCKED
			LIMIT $4
		)
		UPDATE third_party_assessment_jobs j
		SET state='FAILED',lease_token=NULL,lease_expires_at=NULL,last_failure_code=$3,updated_at=$1
		FROM exhausted e WHERE j.id=e.id`, now, maxAttempts, AssessmentSetupFailureAttemptsExhausted, limit); err != nil {
		return nil, fmt.Errorf("terminalize exhausted assessment setup jobs: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM third_party_assessment_jobs
			WHERE job_type='SETUP_REVIEW' AND available_at<=$1
			  AND attempts < $4 AND (state='READY' OR (state='LEASED' AND lease_expires_at<=$1))
			ORDER BY available_at,id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		), claimed AS (
			UPDATE third_party_assessment_jobs j
			SET state='LEASED',attempts=j.attempts+1,lease_token=uuidv7(),lease_expires_at=$3,updated_at=$1
			FROM candidates c WHERE j.id=c.id
			RETURNING j.*
		)
		SELECT `+assessmentSetupJobProjection+` FROM claimed j ORDER BY j.available_at,j.id`, now, limit, expires, maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("claim assessment setup jobs: %w", err)
	}
	defer rows.Close()
	values := make([]AssessmentSetupJob, 0, limit)
	for rows.Next() {
		value, err := scanAssessmentSetupJob(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) CompleteAssessmentSetupJob(ctx context.Context, job AssessmentSetupJob, expectedVersion int64, matterID string, at time.Time) (Assessment, error) {
	if expectedVersion < 1 || !validAssessmentIdentifiers(job.ID, job.AssessmentID, job.LeaseToken, matterID) {
		return Assessment{}, ErrInvalid
	}
	at = at.UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assessment{}, fmt.Errorf("begin assessment setup completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, job.TenantID)
	if err != nil {
		return Assessment{}, err
	}
	var claimedJobID string
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM third_party_assessment_jobs
		WHERE id::text=$1 AND tenant_id=$2::uuid AND legal_entity_id::text=$3 AND assessment_id::text=$4
		  AND job_type='SETUP_REVIEW' AND state='LEASED' AND lease_token::text=$5 AND lease_expires_at>$6
		FOR UPDATE`, job.ID, tenantID, job.LegalEntityID, job.AssessmentID, job.LeaseToken, at).Scan(&claimedJobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrAssessmentJobLeaseLost
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("lock assessment setup job: %w", err)
	}
	current, err := lockAssessment(ctx, tx, tenantID, job.LegalEntityID, job.AssessmentID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Version != expectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentSetupPending {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	triggerKey := "thirdparty-assessment:" + current.ID
	var matterExists bool
	err = tx.QueryRow(ctx, `
		SELECT true FROM matters
		WHERE tenant_id=$1::uuid AND id::text=$2 AND matter_type=$3 AND trigger_key=$4
		FOR SHARE`, tenantID, matterID, continuity.MatterVendorReview, triggerKey).Scan(&matterExists)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrNotFound
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("verify assessment review matter: %w", err)
	}
	canonical, err := ensureAssessmentMatterRelationshipLink(ctx, tx, tenantID, current, matterID, AssessmentMatterReview, current.StartedByPrincipalID, at)
	if err != nil {
		return Assessment{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessment_matter_links(tenant_id,legal_entity_id,assessment_id,matter_id,relationship_link_id,link_kind,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'REVIEW',$6)
		ON CONFLICT (tenant_id,assessment_id,matter_id) DO NOTHING`, tenantID, job.LegalEntityID, current.ID, matterID, canonical.ID, at)
	if err != nil {
		return Assessment{}, fmt.Errorf("link assessment review matter: %w", err)
	}
	current.Status = AssessmentReadyToSend
	current.ReviewMatterID = matterID
	current.Version++
	current.UpdatedAt = at
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return Assessment{}, err
	}
	eventID, err := appendAssessmentEvent(ctx, tx, tenantID, current, "", "AssessmentSetupCompleted")
	if err != nil {
		return Assessment{}, err
	}
	snapshot, err := json.Marshal(current)
	if err != nil {
		return Assessment{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessment_reactions(
			tenant_id,legal_entity_id,assessment_id,reaction_kind,causation_id,job_id,event_id,matter_id,request_id,submission_id,
			resulting_version,result_snapshot,applied_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'SETUP_COMPLETED',$4,$4::uuid,NULL,$5::uuid,NULL,NULL,$6,$7::jsonb,$8)`,
		tenantID, job.LegalEntityID, current.ID, job.ID, matterID, current.Version, string(snapshot), at)
	if err != nil {
		return Assessment{}, fmt.Errorf("store assessment setup receipt: %w", err)
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE third_party_assessment_jobs
		SET state='COMPLETED',lease_token=NULL,lease_expires_at=NULL,last_failure_code='',updated_at=$6
		WHERE id::text=$1 AND tenant_id=$2::uuid AND legal_entity_id::text=$3 AND assessment_id::text=$4
		  AND state='LEASED' AND lease_token::text=$5`, job.ID, tenantID, job.LegalEntityID, job.AssessmentID, job.LeaseToken, at)
	if err != nil {
		return Assessment{}, fmt.Errorf("complete assessment setup job: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return Assessment{}, ErrAssessmentJobLeaseLost
	}
	if err := r.commitThirdPartyEvents(ctx, tx, assessmentCommitProof(eventID, current, "AssessmentSetupCompleted")); err != nil {
		return Assessment{}, fmt.Errorf("commit assessment setup completion: %w", err)
	}
	return current, nil
}

func (r *PostgresRepository) FailAssessmentSetupJob(ctx context.Context, job AssessmentSetupJob, maxAttempts int, failureCode string, at, availableAt time.Time) (AssessmentSetupJob, error) {
	if maxAttempts < 1 || maxAttempts > 20 || !validAssessmentFailureCode(failureCode) || availableAt.Before(at) || !validAssessmentIdentifiers(job.ID, job.AssessmentID, job.LeaseToken) {
		return AssessmentSetupJob{}, ErrInvalid
	}
	value, err := scanAssessmentSetupJob(r.pool.QueryRow(ctx, `
		UPDATE third_party_assessment_jobs j
		SET state=CASE WHEN attempts >= $6 THEN 'FAILED' ELSE 'READY' END,
			available_at=CASE WHEN attempts >= $6 THEN available_at ELSE $7 END,
			lease_token=NULL,lease_expires_at=NULL,last_failure_code=$8,updated_at=$9
		WHERE id::text=$1 AND tenant_id::text=$2 AND legal_entity_id::text=$3 AND assessment_id::text=$4
		  AND state='LEASED' AND lease_token::text=$5 AND lease_expires_at>$9
		RETURNING `+assessmentSetupJobProjection,
		job.ID, job.TenantID, job.LegalEntityID, job.AssessmentID, job.LeaseToken, maxAttempts, availableAt.UTC(), failureCode, at.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentSetupJob{}, ErrAssessmentJobLeaseLost
	}
	if err != nil {
		return AssessmentSetupJob{}, fmt.Errorf("release assessment setup job: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) RequeueAssessmentSetup(ctx context.Context, record RequeueAssessmentSetupRecord) (AssessmentSetupJob, Assessment, error) {
	if !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID) || record.ExpectedVersion < 1 || record.QueuedAt.IsZero() {
		return AssessmentSetupJob{}, Assessment{}, ErrInvalid
	}
	record.QueuedAt = record.QueuedAt.UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, fmt.Errorf("begin assessment setup retry: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, err
	}
	job, err := scanAssessmentSetupJob(tx.QueryRow(ctx, `
		SELECT `+assessmentSetupJobProjection+`
		FROM third_party_assessment_jobs j
		WHERE j.tenant_id=$1::uuid AND j.legal_entity_id::text=$2 AND j.assessment_id::text=$3 AND j.job_type='SETUP_REVIEW'
		FOR UPDATE`, tenantID, record.LegalEntityID, record.AssessmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentSetupJob{}, Assessment{}, ErrNotFound
	}
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, fmt.Errorf("lock assessment setup retry job: %w", err)
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, err
	}
	if current.Version != record.ExpectedVersion {
		if current.Version == record.ExpectedVersion+1 {
			var replay bool
			err = tx.QueryRow(ctx, `
				SELECT true FROM third_party_events
				WHERE tenant_id=$1::uuid AND aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$2::uuid
				  AND aggregate_version=$3 AND event_type='AssessmentSetupRetryQueued'`, tenantID, current.ID, current.Version).Scan(&replay)
			if err == nil && replay {
				return job, current, nil
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return AssessmentSetupJob{}, Assessment{}, fmt.Errorf("verify assessment setup retry replay: %w", err)
			}
		}
		return AssessmentSetupJob{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentSetupPending || job.State != AssessmentJobFailed || job.LeaseToken != "" || job.LeaseExpiresAt != nil || !validAssessmentFailureCode(job.LastFailureCode) {
		return AssessmentSetupJob{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	previousFailureCode := job.LastFailureCode
	job.State = AssessmentJobReady
	job.Attempts = 0
	job.AvailableAt = record.QueuedAt
	job.LeaseToken = ""
	job.LeaseExpiresAt = nil
	job.LastFailureCode = ""
	job.UpdatedAt = record.QueuedAt
	commandTag, err := tx.Exec(ctx, `
		UPDATE third_party_assessment_jobs
		SET state='READY',attempts=0,available_at=$5,lease_token=NULL,lease_expires_at=NULL,last_failure_code='',updated_at=$5
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id::text=$3 AND assessment_id::text=$4 AND state='FAILED'`,
		job.ID, tenantID, record.LegalEntityID, record.AssessmentID, record.QueuedAt)
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, fmt.Errorf("requeue assessment setup job: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return AssessmentSetupJob{}, Assessment{}, ErrVersionConflict
	}
	current.Version++
	current.UpdatedAt = record.QueuedAt
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentSetupJob{}, Assessment{}, err
	}
	eventID, err := appendAssessmentSetupRetryEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, job.ID, previousFailureCode)
	if err != nil {
		return AssessmentSetupJob{}, Assessment{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, assessmentCommitProof(eventID, current, "AssessmentSetupRetryQueued")); err != nil {
		return AssessmentSetupJob{}, Assessment{}, fmt.Errorf("commit assessment setup retry: %w", err)
	}
	return job, current, nil
}

func appendAssessmentSetupRetryEvent(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment, actorID, jobID, failureCode string) (string, error) {
	var eventID string
	err := tx.QueryRow(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,$4::uuid,'AssessmentSetupRetryQueued',
			jsonb_build_object('status',$5::text,'relationship_id',$6::text,'matter_id',$7::text,'setup_job_id',$8::text,'previous_failure_code',$9::text),$10)
		RETURNING id::text`,
		tenantID, assessment.ID, assessment.Version, actorID, assessment.Status, assessment.RelationshipID, assessment.ReviewMatterID, jobID, failureCode, assessment.UpdatedAt).Scan(&eventID)
	if err != nil {
		return "", fmt.Errorf("append assessment setup retry event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,'AssessmentSetupRetryQueued',
			jsonb_build_object('version',$3::bigint,'status',$4::text,'relationship_id',$5::text,'matter_id',$6::text,'setup_job_id',$7::text,'previous_failure_code',$8::text),$9,$9)`,
		tenantID, assessment.ID, assessment.Version, assessment.Status, assessment.RelationshipID, assessment.ReviewMatterID, jobID, failureCode, assessment.UpdatedAt)
	if err != nil {
		return "", fmt.Errorf("append assessment setup retry outbox event: %w", err)
	}
	return eventID, nil
}

func (r *PostgresRepository) ListAssessmentSetupJobs(ctx context.Context, scope Scope, assessmentID string) ([]AssessmentSetupJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+assessmentSetupJobProjection+`
		FROM third_party_assessment_jobs j JOIN tenants t ON t.id=j.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND j.legal_entity_id::text=$2 AND ($3='' OR j.assessment_id::text=$3)
		ORDER BY j.created_at,j.id`, scope.TenantID, scope.LegalEntityID, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("list assessment setup jobs: %w", err)
	}
	defer rows.Close()
	values := []AssessmentSetupJob{}
	for rows.Next() {
		value, err := scanAssessmentSetupJob(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) GetAssessmentSetupJob(ctx context.Context, scope Scope, assessmentID string) (AssessmentSetupJob, error) {
	value, err := scanAssessmentSetupJob(r.pool.QueryRow(ctx, `
		SELECT `+assessmentSetupJobProjection+`
		FROM third_party_assessment_jobs j JOIN tenants t ON t.id=j.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND j.legal_entity_id::text=$2 AND j.assessment_id::text=$3
		ORDER BY j.created_at DESC,j.id DESC LIMIT 1`, scope.TenantID, scope.LegalEntityID, assessmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentSetupJob{}, ErrNotFound
	}
	if err != nil {
		return AssessmentSetupJob{}, fmt.Errorf("get assessment setup job: %w", err)
	}
	return value, nil
}

func scanAssessmentSetupJob(row rowScanner) (AssessmentSetupJob, error) {
	var value AssessmentSetupJob
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.AssessmentID, &value.JobType, &value.DedupeKey, &value.State,
		&value.Attempts, &value.AvailableAt, &value.LeaseToken, &value.LeaseExpiresAt, &value.LastFailureCode, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

var _ AssessmentSetupRepository = (*PostgresRepository)(nil)
