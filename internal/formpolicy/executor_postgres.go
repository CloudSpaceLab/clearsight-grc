//go:build postgres

package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5"
)

const formPolicyExecutorConsumerPrefix = "form-response-policy-executor:"

func (repo *PostgresRepository) ApplyExecution(ctx context.Context, command ExecutionCommand) (ExecutionReceipt, error) {
	if repo == nil || repo.pool == nil || !validExecutionCommand(command) {
		return ExecutionReceipt{}, ErrInvalid
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return ExecutionReceipt{}, err
	}
	defer tx.Rollback(ctx)
	lockKey := strings.Join([]string{command.Receipt.TenantID, command.Receipt.LegalEntityID, command.Receipt.PolicyID, fmt.Sprint(command.Receipt.PolicyVersion), command.Receipt.ResponseRevisionID}, "|")
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return ExecutionReceipt{}, err
	}
	stored, err := getExecutionTx(ctx, tx, command.Receipt)
	if err == nil {
		return stored, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return ExecutionReceipt{}, err
	}
	consumer := fmt.Sprintf("%s%s:%d", formPolicyExecutorConsumerPrefix, command.Policy.ID, command.Policy.Version)
	if _, err = tx.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4) ON CONFLICT DO NOTHING`, command.Receipt.TenantID, consumer, command.EventID, command.Receipt.CreatedAt); err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	receipt := command.Receipt
	if receipt.State == ExecutionFailed {
		receipt, err = applyFailedExecutionTx(ctx, tx, command)
		if err != nil {
			return ExecutionReceipt{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return ExecutionReceipt{}, err
		}
		return receipt, nil
	}
	if receipt.State == ExecutionApplied {
		receipt, err = applyMatchedExecutionTx(ctx, tx, command)
		if err != nil {
			return ExecutionReceipt{}, err
		}
	}
	if err = insertExecutionTx(ctx, tx, receipt); err != nil {
		return ExecutionReceipt{}, err
	}
	if err = advanceExecutionRecoveryMatterTx(ctx, tx, command, receipt); err != nil {
		return ExecutionReceipt{}, err
	}
	if receipt.State == ExecutionApplied || receipt.State == ExecutionReused {
		if err = insertOutcomeCheckTx(ctx, tx, command, receipt); err != nil {
			return ExecutionReceipt{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return ExecutionReceipt{}, err
	}
	return receipt, nil
}

func advanceExecutionRecoveryMatterTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand, receipt ExecutionReceipt) error {
	if strings.TrimSpace(command.Route.ServicePrincipalID) == "" || receipt.State == ExecutionFailed {
		return nil
	}
	triggerKey := "form-response-policy-execution-failure:" + command.Policy.ID + ":" + command.Response.ID
	var matterID string
	var matterVersion int64
	var action continuity.Action
	err := tx.QueryRow(ctx, `SELECT m.id::text,m.version,a.id::text,a.tenant_id::text,a.matter_id::text,COALESCE(a.origin_key,''),a.title,a.description,COALESCE(a.owner_principal_id::text,''),a.required_responsibility,a.status,a.due_at,a.implemented_at,a.created_at,a.updated_at,a.version
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		JOIN legal_entities le ON le.id=m.legal_entity_id AND le.tenant_id=m.tenant_id
		JOIN matter_actions a ON a.tenant_id=m.tenant_id AND a.matter_id=m.id AND a.origin_key='form-response-policy-execution-recovery'
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		  AND m.trigger_key=$3 AND m.status NOT IN ('CLOSED','CANCELLED')
		ORDER BY m.created_at DESC,m.id DESC LIMIT 1 FOR UPDATE OF m,a`, receipt.TenantID, receipt.LegalEntityID, triggerKey).
		Scan(&matterID, &matterVersion, &action.ID, &action.TenantID, &action.MatterID, &action.OriginKey, &action.Title, &action.Description, &action.OwnerPrincipalID, &action.RequiredResponsibility, &action.Status, &action.DueAt, &action.ImplementedAt, &action.CreatedAt, &action.UpdatedAt, &action.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `WITH prior AS (
		SELECT m.* FROM matters m WHERE m.id=$1::uuid AND m.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND m.version=$3 FOR UPDATE
	), updated AS (
		UPDATE matters m SET known_facts=m.known_facts || jsonb_build_object('route_recovery_execution_id',to_jsonb($4::text),'route_recovery_recorded_at',to_jsonb($5::text)),updated_at=$6,version=m.version+1
		FROM prior WHERE m.id=prior.id RETURNING m.*
	), payload AS (
		SELECT jsonb_build_object(
			'matter',(to_jsonb(updated)-'matter_type') || jsonb_build_object('type',updated.matter_type),
			'kind','ADD_FACT','key','route_recovery_execution_id','value',to_jsonb($4::text),
			'rationale','The repaired authority route accepted this policy execution. The escalation owner must review the recorded recovery before closing the exception.',
			'evidence_references',jsonb_build_array()
		) value,updated.* FROM updated
	), event AS (
		INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at)
		SELECT tenant_id,'MATTER',id,version,'MATTER_CONTEXT_CHANGED',value,'SERVICE',$7::uuid,$6 FROM payload
		RETURNING tenant_id,aggregate_id,payload
	), outbox AS (
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		SELECT tenant_id,'MATTER',aggregate_id,'MATTER_CONTEXT_CHANGED',payload,$6,$6,$6 FROM event
		RETURNING aggregate_id
	)
	SELECT updated.version FROM updated JOIN outbox ON outbox.aggregate_id=updated.id`, matterID, receipt.TenantID, matterVersion, receipt.ID, receipt.CreatedAt.UTC().Format(time.RFC3339Nano), receipt.CreatedAt, command.Route.ServicePrincipalID).Scan(&matterVersion)
	if err != nil {
		return normalizePostgresError(err)
	}
	if action.Status == continuity.ActionImplemented || action.Status == continuity.ActionCancelled {
		return nil
	}
	if action.Status == continuity.ActionPlanned || action.Status == continuity.ActionBlocked {
		if err = appendExecutionRecoveryActionStateTx(ctx, tx, &action, matterVersion, continuity.ActionInProgress, receipt, command.Route.ServicePrincipalID); err != nil {
			return err
		}
		matterVersion++
	}
	if action.Status == continuity.ActionInProgress {
		action.Description = strings.TrimSpace(action.Description) + " Route restored by policy execution " + receipt.ID + "."
		return appendExecutionRecoveryActionStateTx(ctx, tx, &action, matterVersion, continuity.ActionImplemented, receipt, command.Route.ServicePrincipalID)
	}
	return nil
}

func appendExecutionRecoveryActionStateTx(ctx context.Context, tx pgx.Tx, action *continuity.Action, matterVersion int64, target continuity.ActionStatus, receipt ExecutionReceipt, actorID string) error {
	priorActionVersion := action.Version
	action.Status = target
	action.UpdatedAt = receipt.CreatedAt
	action.Version++
	if target == continuity.ActionImplemented {
		implementedAt := receipt.CreatedAt.UTC()
		action.ImplementedAt = &implementedAt
	}
	payload, err := json.Marshal(action)
	if err != nil {
		return ErrInvalid
	}
	tag, err := tx.Exec(ctx, `UPDATE matter_actions SET description=$4,status=$5,implemented_at=$6,updated_at=$7,version=$8 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND matter_id=$3::uuid AND version=$9`, action.ID, action.TenantID, action.MatterID, action.Description, action.Status, action.ImplementedAt, action.UpdatedAt, action.Version, priorActionVersion)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if tag, err = tx.Exec(ctx, `UPDATE matters SET version=$4,updated_at=$5 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=$3`, action.MatterID, action.TenantID, matterVersion, matterVersion+1, receipt.CreatedAt); err != nil {
		return normalizePostgresError(err)
	} else if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,$3,'ACTION_STATE_CHANGED',$4::jsonb,'SERVICE',$5::uuid,$6)`, action.TenantID, action.MatterID, matterVersion+1, payload, actorID, receipt.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'ACTION_STATE_CHANGED',$3::jsonb,$4,$4,$4)`, action.TenantID, action.MatterID, payload, receipt.CreatedAt)
	return normalizePostgresError(err)
}

func applyFailedExecutionTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand) (ExecutionReceipt, error) {
	receipt := command.Receipt
	tag, err := tx.Exec(ctx, `INSERT INTO form_response_policy_execution_failures(
		id,tenant_id,legal_entity_id,policy_id,policy_version,automation_policy_id,automation_policy_version,response_revision_id,event_id,reason_code,created_at
	) SELECT $1::uuid,t.id,le.id,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9,$10,$11
	  FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3)
	  WHERE t.id::text=$2 OR t.slug=$2
	  ON CONFLICT (tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id,event_id) DO NOTHING`,
		receipt.ID, receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, receipt.PolicyVersion,
		receipt.AutomationPolicyID, receipt.AutomationPolicyVersion, receipt.ResponseRevisionID, command.EventID, receipt.ReasonCode, receipt.CreatedAt)
	if err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return getExecutionFailureTx(ctx, tx, command)
	}
	if err = insertExecutionFailureOutboxTx(ctx, tx, receipt); err != nil {
		return ExecutionReceipt{}, err
	}
	if err = insertExecutionRetryJobTx(ctx, tx, receipt); err != nil {
		return ExecutionReceipt{}, err
	}
	if command.FailureMatter != nil && command.FailureAction != nil {
		if err = insertExecutionOperationalExceptionTx(ctx, tx, command); err != nil {
			return ExecutionReceipt{}, err
		}
	}
	return receipt, nil
}

func getExecutionFailureTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand) (ExecutionReceipt, error) {
	var value ExecutionReceipt
	err := tx.QueryRow(ctx, `SELECT f.id::text,f.tenant_id::text,f.legal_entity_id::text,f.policy_id::text,f.policy_version,f.automation_policy_id::text,f.automation_policy_version,f.response_revision_id::text,'FAILED',f.reason_code,f.created_at
		FROM form_response_policy_execution_failures f JOIN tenants t ON t.id=f.tenant_id JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND f.policy_id=$3::uuid AND f.policy_version=$4 AND f.response_revision_id=$5::uuid AND f.event_id=$6`,
		command.Receipt.TenantID, command.Receipt.LegalEntityID, command.Receipt.PolicyID, command.Receipt.PolicyVersion, command.Receipt.ResponseRevisionID, command.EventID).
		Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyID, &value.PolicyVersion, &value.AutomationPolicyID, &value.AutomationPolicyVersion, &value.ResponseRevisionID, &value.State, &value.ReasonCode, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionReceipt{}, ErrNotFound
	}
	return value, err
}

func insertExecutionRetryJobTx(ctx context.Context, tx pgx.Tx, receipt ExecutionReceipt) error {
	_, err := tx.Exec(ctx, `INSERT INTO form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,job_type,response_revision_id,due_at,state,created_at,updated_at)
		SELECT tenant_id,legal_entity_id,'RECONCILE',response_revision_id,$2::timestamptz+interval '30 seconds','READY',$2,$2
		FROM form_response_policy_execution_failures WHERE id=$1::uuid
		ON CONFLICT (tenant_id,legal_entity_id,response_revision_id) WHERE job_type='RECONCILE' DO NOTHING`, receipt.ID, receipt.CreatedAt)
	return normalizePostgresError(err)
}

func insertExecutionOperationalExceptionTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand) error {
	matter, action := command.FailureMatter, command.FailureAction
	if matter == nil || action == nil {
		return nil
	}
	var matterID string
	err := tx.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id JOIN legal_entities le ON le.id=m.legal_entity_id AND le.tenant_id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND m.trigger_key=$3 AND m.status NOT IN ('CLOSED','CANCELLED') FOR UPDATE`, matter.TenantID, matter.LegalEntityID, matter.TriggerKey).Scan(&matterID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		matterPayload, _ := json.Marshal(matter)
		if _, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17::jsonb,$18::uuid,$19,NULL,NULL,'',0,$20,$20,1,$21::uuid)`, matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, string(matter.Scope), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, string(matter.KnownFacts), string(matter.MissingFacts), string(matter.Contradictions), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.CreatedAt, matter.LegalEntityID); err != nil {
			return normalizePostgresError(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,1,'MATTER_CREATED',$3::jsonb,'SYSTEM',NULL,$4)`, matter.TenantID, matter.ID, matterPayload, matter.CreatedAt); err != nil {
			return normalizePostgresError(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'MATTER_CREATED',$3::jsonb,$4,$4,$4)`, matter.TenantID, matter.ID, matterPayload, matter.CreatedAt); err != nil {
			return normalizePostgresError(err)
		}
		matterID = matter.ID
	}
	action.MatterID = matterID
	var actionExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matter_actions WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND origin_key=$3)`, action.TenantID, matterID, action.OriginKey).Scan(&actionExists); err != nil {
		return err
	}
	if actionExists {
		return nil
	}
	actionPayload, _ := json.Marshal(action)
	var aggregateVersion int64
	err = tx.QueryRow(ctx, `WITH inserted AS (
		INSERT INTO matter_actions(id,tenant_id,matter_id,origin_key,title,description,owner_principal_id,required_responsibility,status,due_at,implemented_at,created_at,updated_at,version)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7::uuid,$8,$9,NULL,NULL,$10,$10,1)
		ON CONFLICT (tenant_id,matter_id,origin_key) WHERE origin_key IS NOT NULL DO NOTHING
		RETURNING id
	), updated AS (UPDATE matters SET version=version+1,updated_at=$10 WHERE id=$3::uuid RETURNING version)
	SELECT version FROM updated`, action.ID, action.TenantID, action.MatterID, action.OriginKey, action.Title, action.Description, action.OwnerPrincipalID, action.RequiredResponsibility, action.Status, action.CreatedAt).Scan(&aggregateVersion)
	if err != nil {
		return normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,$3,'ACTION_ADDED',$4::jsonb,'SYSTEM',NULL,$5)`, action.TenantID, matterID, aggregateVersion, actionPayload, action.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'ACTION_ADDED',$3::jsonb,$4,$4,$4)`, action.TenantID, matterID, actionPayload, action.CreatedAt)
	return normalizePostgresError(err)
}

func (repo *PostgresRepository) ApplyCompensation(ctx context.Context, command CompensationCommand) (CompensationReceipt, error) {
	if repo == nil || repo.pool == nil || !validCompensationCommand(command) {
		return CompensationReceipt{}, ErrInvalid
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return CompensationReceipt{}, err
	}
	defer tx.Rollback(ctx)
	receipt := command.Receipt
	lockKey := strings.Join([]string{receipt.TenantID, receipt.LegalEntityID, receipt.RollbackPolicyID, fmt.Sprint(receipt.RollbackPolicyVersion), receipt.OriginalExecutionID, "compensation"}, "|")
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return CompensationReceipt{}, err
	}
	stored, err := getCompensationTx(ctx, tx, receipt)
	if err == nil {
		return stored, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return CompensationReceipt{}, err
	}
	var policyStatus PolicyStatus
	var rollbackOf, executionPolicyID, responseID, matterID string
	var activatedAt *time.Time
	var executionCreatedAt time.Time
	var matterStatus continuity.MatterStatus
	err = tx.QueryRow(ctx, `
		SELECT p.status,p.rollback_of_policy_id::text,p.activated_at,e.policy_id::text,e.response_revision_id::text,
		       e.matter_id::text,e.created_at,m.status
		FROM form_response_policy_definitions p
		JOIN tenants t ON t.id=p.tenant_id
		JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id
		JOIN form_response_policy_executions e
		  ON e.id=$5::uuid AND e.tenant_id=p.tenant_id AND e.legal_entity_id=p.legal_entity_id
		JOIN matters m ON m.id=e.matter_id AND m.tenant_id=e.tenant_id AND m.legal_entity_id=e.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		  AND p.id=$3::uuid AND p.version=$4 AND e.created_matter
		FOR UPDATE OF p,e,m`, receipt.TenantID, receipt.LegalEntityID, receipt.RollbackPolicyID, receipt.RollbackPolicyVersion, receipt.OriginalExecutionID).
		Scan(&policyStatus, &rollbackOf, &activatedAt, &executionPolicyID, &responseID, &matterID, &executionCreatedAt, &matterStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return CompensationReceipt{}, ErrConflict
	}
	if err != nil {
		return CompensationReceipt{}, err
	}
	if policyStatus != PolicyActive || activatedAt == nil || executionCreatedAt.After(*activatedAt) || rollbackOf != executionPolicyID ||
		rollbackOf != command.Candidate.OriginalExecution.PolicyID || responseID != command.Response.ID || matterID != receipt.MatterID {
		return CompensationReceipt{}, ErrConflict
	}
	if err = insertCompensationReviewWorkTx(ctx, tx, command); err != nil {
		return CompensationReceipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO form_response_policy_compensations(
		id,tenant_id,legal_entity_id,rollback_policy_id,rollback_policy_version,original_execution_id,matter_id,review_matter_id,actor_id,reviewer_principal_id,state,reason_code,created_at
	) SELECT $1::uuid,t.id,le.id,$4::uuid,$5,$6::uuid,$7::uuid,$8::uuid,$9::uuid,$10::uuid,$11,$12,$13
	  FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3)
	  WHERE t.id::text=$2 OR t.slug=$2`, receipt.ID, receipt.TenantID, receipt.LegalEntityID, receipt.RollbackPolicyID,
		receipt.RollbackPolicyVersion, receipt.OriginalExecutionID, receipt.MatterID, receipt.ReviewMatterID, receipt.ActorID, receipt.ReviewerPrincipalID, receipt.State, receipt.ReasonCode, receipt.CreatedAt); err != nil {
		return CompensationReceipt{}, normalizePostgresError(err)
	}
	if matterStatus != continuity.MatterClosed && matterStatus != continuity.MatterCancelled {
		reviewPayload, _ := json.Marshal(map[string]any{"state": receipt.State, "rollback_policy_id": receipt.RollbackPolicyID, "rollback_policy_version": receipt.RollbackPolicyVersion, "original_execution_id": receipt.OriginalExecutionID})
		tag, updateErr := tx.Exec(ctx, `
			WITH prior AS (
				SELECT m.* FROM matters m
				WHERE m.id=$1::uuid AND m.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)
				  AND m.legal_entity_id=$3::uuid AND m.status NOT IN ('CLOSED','CANCELLED')
				FOR UPDATE
			), updated AS (
				UPDATE matters m SET known_facts=m.known_facts || jsonb_build_object('form_response_policy_compensation',$4::jsonb),updated_at=$5,version=m.version+1
				FROM prior WHERE m.id=prior.id RETURNING m.*
			), payload AS (
				SELECT jsonb_build_object(
					'matter',(to_jsonb(updated)-'matter_type') || jsonb_build_object('type',updated.matter_type),
					'previous',(to_jsonb(prior)-'matter_type') || jsonb_build_object('type',prior.matter_type),
					'rationale','An active rollback revision requires an authorized review of this policy-created issue.'
				) value,updated.* FROM updated CROSS JOIN prior
			), event AS (
				INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at)
				SELECT tenant_id,'MATTER',id,version,'MATTER_DETAILS_UPDATED',value,'SERVICE',$6::uuid,$5 FROM payload
				RETURNING tenant_id,aggregate_id,payload
			)
			INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
			SELECT tenant_id,'MATTER',aggregate_id,'MATTER_DETAILS_UPDATED',payload,$5,$5,$5 FROM event`, receipt.MatterID, receipt.TenantID, receipt.LegalEntityID, reviewPayload, receipt.CreatedAt, receipt.ActorID)
		if updateErr != nil {
			return CompensationReceipt{}, normalizePostgresError(updateErr)
		}
		if tag.RowsAffected() != 1 {
			return CompensationReceipt{}, ErrConflict
		}
	}
	compensationPayload, _ := json.Marshal(map[string]any{
		"version": receipt.RollbackPolicyVersion, "compensation_id": receipt.ID, "rollback_policy_id": receipt.RollbackPolicyID,
		"rollback_policy_version": receipt.RollbackPolicyVersion, "original_execution_id": receipt.OriginalExecutionID,
		"matter_id": receipt.MatterID, "review_matter_id": receipt.ReviewMatterID, "actor_id": receipt.ActorID,
		"reviewer_principal_id": receipt.ReviewerPrincipalID, "state": receipt.State, "reason_code": receipt.ReasonCode,
	})
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'FORM_RESPONSE_POLICY_COMPENSATION',$2::uuid,'FORM_RESPONSE_POLICY_COMPENSATION_REVIEW_REQUIRED',$3::jsonb,$4,$4,$4)`, receipt.TenantID, receipt.ID, compensationPayload, receipt.CreatedAt); err != nil {
		return CompensationReceipt{}, normalizePostgresError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return CompensationReceipt{}, err
	}
	return receipt, nil
}

func insertCompensationReviewWorkTx(ctx context.Context, tx pgx.Tx, command CompensationCommand) error {
	matter, action := command.ReviewMatter, command.ReviewAction
	matterPayload, err := json.Marshal(matter)
	if err != nil {
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17::jsonb,$18::uuid,$19,NULL,NULL,'',0,$20,$20,1,$21::uuid)`, matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, string(matter.Scope), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, string(matter.KnownFacts), string(matter.MissingFacts), string(matter.Contradictions), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.CreatedAt, matter.LegalEntityID); err != nil {
		return normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,1,'MATTER_CREATED',$3::jsonb,'SERVICE',$4::uuid,$5)`, matter.TenantID, matter.ID, matterPayload, command.Receipt.ActorID, matter.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'MATTER_CREATED',$3::jsonb,$4,$4,$4)`, matter.TenantID, matter.ID, matterPayload, matter.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	actionPayload, err := json.Marshal(action)
	if err != nil {
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matter_actions(id,tenant_id,matter_id,origin_key,title,description,owner_principal_id,required_responsibility,status,due_at,implemented_at,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7::uuid,$8,$9,NULL,NULL,$10,$10,1)`, action.ID, action.TenantID, action.MatterID, action.OriginKey, action.Title, action.Description, action.OwnerPrincipalID, action.RequiredResponsibility, action.Status, action.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	if tag, updateErr := tx.Exec(ctx, `UPDATE matters SET version=2,updated_at=$3 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=1`, matter.ID, matter.TenantID, action.CreatedAt); updateErr != nil {
		return normalizePostgresError(updateErr)
	} else if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,2,'ACTION_ADDED',$3::jsonb,'SERVICE',$4::uuid,$5)`, action.TenantID, matter.ID, actionPayload, command.Receipt.ActorID, action.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'ACTION_ADDED',$3::jsonb,$4,$4,$4)`, action.TenantID, matter.ID, actionPayload, action.CreatedAt)
	return normalizePostgresError(err)
}

func getCompensationTx(ctx context.Context, tx pgx.Tx, probe CompensationReceipt) (CompensationReceipt, error) {
	var value CompensationReceipt
	err := tx.QueryRow(ctx, `SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.rollback_policy_id::text,c.rollback_policy_version,c.original_execution_id::text,c.matter_id::text,c.review_matter_id::text,c.actor_id::text,c.reviewer_principal_id::text,c.state,c.reason_code,c.created_at
		FROM form_response_policy_compensations c
		JOIN tenants t ON t.id=c.tenant_id JOIN legal_entities le ON le.id=c.legal_entity_id AND le.tenant_id=c.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		  AND c.rollback_policy_id=$3::uuid AND c.rollback_policy_version=$4 AND c.original_execution_id=$5::uuid`,
		probe.TenantID, probe.LegalEntityID, probe.RollbackPolicyID, probe.RollbackPolicyVersion, probe.OriginalExecutionID).
		Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RollbackPolicyID, &value.RollbackPolicyVersion, &value.OriginalExecutionID, &value.MatterID, &value.ReviewMatterID, &value.ActorID, &value.ReviewerPrincipalID, &value.State, &value.ReasonCode, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CompensationReceipt{}, ErrNotFound
	}
	return value, err
}

func insertExecutionFailureOutboxTx(ctx context.Context, tx pgx.Tx, receipt ExecutionReceipt) error {
	payload, _ := json.Marshal(map[string]any{
		"version": receipt.PolicyVersion, "policy_id": receipt.PolicyID, "policy_version": receipt.PolicyVersion,
		"response_revision_id": receipt.ResponseRevisionID, "execution_id": receipt.ID, "state": receipt.State, "reason_code": receipt.ReasonCode,
	})
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'FORM_RESPONSE_POLICY_EXECUTION',$2::uuid,'FORM_RESPONSE_POLICY_EXECUTION_FAILED',$3::jsonb,$4,$4,$4) ON CONFLICT DO NOTHING`, receipt.TenantID, receipt.ID, payload, receipt.CreatedAt)
	return normalizePostgresError(err)
}

func insertOutcomeCheckTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand, receipt ExecutionReceipt) error {
	dueAt := receipt.CreatedAt.Add(time.Duration(command.Policy.Outcome.CheckAfterMinutes) * time.Minute)
	_, err := tx.Exec(ctx, `INSERT INTO form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,job_type,response_revision_id,policy_execution_id,adverse_episode_id,matter_id,due_at,state,created_at,updated_at) SELECT e.tenant_id,e.legal_entity_id,'OUTCOME_CHECK',e.response_revision_id,e.id,episode.id,e.matter_id,$6,'READY',$7,$7 FROM form_response_policy_executions e JOIN form_response_policy_adverse_episodes episode ON episode.tenant_id=e.tenant_id AND episode.legal_entity_id=e.legal_entity_id AND episode.matter_id=e.matter_id AND episode.state='OPEN' WHERE e.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND e.legal_entity_id=(SELECT le.id FROM legal_entities le JOIN tenants t ON t.id=le.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)) AND e.policy_id=$3::uuid AND e.policy_version=$4 AND e.response_revision_id=$5::uuid ON CONFLICT (tenant_id,legal_entity_id,policy_execution_id) WHERE job_type='OUTCOME_CHECK' DO NOTHING`, receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, receipt.PolicyVersion, receipt.ResponseRevisionID, dueAt, receipt.CreatedAt)
	return normalizePostgresError(err)
}

func applyMatchedExecutionTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand) (ExecutionReceipt, error) {
	receipt := command.Receipt
	episodeLockKey := strings.Join([]string{receipt.TenantID, receipt.LegalEntityID, command.Policy.Code, strings.ToUpper(command.Route.CanonicalSubjectType), command.Route.CanonicalSubjectID, "adverse-episode"}, "|")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, episodeLockKey); err != nil {
		return ExecutionReceipt{}, err
	}
	var episodeID, matterID string
	var matterStatus continuity.MatterStatus
	var matterClosedAt *time.Time
	err := tx.QueryRow(ctx, `SELECT e.id::text,e.matter_id::text,m.status,m.closed_at FROM form_response_policy_adverse_episodes e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id JOIN matters m ON m.id=e.matter_id AND m.tenant_id=e.tenant_id AND m.legal_entity_id=e.legal_entity_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_code=$3 AND e.subject_type=$4 AND e.subject_id::text=$5 AND e.state='OPEN' FOR UPDATE OF e,m`, receipt.TenantID, receipt.LegalEntityID, command.Policy.Code, command.Route.CanonicalSubjectType, command.Route.CanonicalSubjectID).Scan(&episodeID, &matterID, &matterStatus, &matterClosedAt)
	if err == nil {
		switch matterStatus {
		case continuity.MatterClosed:
			if matterClosedAt == nil {
				return ExecutionReceipt{}, ErrInvalid
			}
			if err = closeEpisodeForVerifiedMatterTx(ctx, tx, receipt.TenantID, episodeID, matterID, command.Response.ID, *matterClosedAt); err != nil {
				return ExecutionReceipt{}, err
			}
			err = pgx.ErrNoRows
		case continuity.MatterCancelled:
			receipt.State, receipt.MatterID, receipt.CreatedMatter, receipt.ReasonCode = ExecutionFailed, matterID, false, "OPEN_EPISODE_MATTER_CANCELLED"
			return receipt, nil
		default:
			if err = updateReusedMatterTx(ctx, tx, command, matterID); err != nil {
				return ExecutionReceipt{}, err
			}
			if _, err = tx.Exec(ctx, `UPDATE form_response_policy_adverse_episodes SET policy_id=CASE WHEN updated_at<=$9 THEN $6::uuid ELSE policy_id END,policy_version=CASE WHEN updated_at<=$9 THEN $7 ELSE policy_version END,last_response_revision_id=CASE WHEN updated_at<=$9 THEN $8::uuid ELSE last_response_revision_id END,updated_at=GREATEST(updated_at,$9),record_version=record_version+1 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id=(SELECT le.id FROM legal_entities le JOIN tenants t ON t.id=le.tenant_id WHERE (t.id::text=$2 OR t.slug=$2) AND (le.id::text=$3 OR le.code=$3)) AND state='OPEN' AND policy_code=$4 AND subject_id=$5::uuid`, episodeID, receipt.TenantID, receipt.LegalEntityID, command.Policy.Code, command.Route.CanonicalSubjectID, command.Policy.ID, command.Policy.Version, command.Response.ID, receipt.CreatedAt); err != nil {
				return ExecutionReceipt{}, normalizePostgresError(err)
			}
			receipt.State, receipt.MatterID, receipt.CreatedMatter, receipt.ReasonCode = ExecutionReused, matterID, false, "OPEN_EPISODE_REUSED"
			return receipt, nil
		}
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ExecutionReceipt{}, err
	}
	runStart := receipt.CreatedAt.UTC().Truncate(executionRunWindow)
	runEnd := runStart.Add(executionRunWindow)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, strings.Join([]string{receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, fmt.Sprint(receipt.PolicyVersion), "run", runStart.Format(time.RFC3339)}, "|")); err != nil {
		return ExecutionReceipt{}, err
	}
	var createdThisRun int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM form_response_policy_executions e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id=$3::uuid AND e.policy_version=$4 AND e.created_matter AND e.created_at>=$5 AND e.created_at<$6`, receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, receipt.PolicyVersion, runStart, runEnd).Scan(&createdThisRun); err != nil {
		return ExecutionReceipt{}, err
	}
	if createdThisRun >= command.Policy.BlastRadius.PerRun {
		receipt.State, receipt.ReasonCode = ExecutionBlastSuppressed, "PER_RUN_LIMIT"
		return receipt, nil
	}
	dayStart := receipt.CreatedAt.UTC().Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, strings.Join([]string{receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, fmt.Sprint(receipt.PolicyVersion), "day", dayStart.Format("2006-01-02")}, "|")); err != nil {
		return ExecutionReceipt{}, err
	}
	var createdToday int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM form_response_policy_executions e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id=$3::uuid AND e.policy_version=$4 AND e.created_matter AND e.created_at>=$5 AND e.created_at<$6`, receipt.TenantID, receipt.LegalEntityID, receipt.PolicyID, receipt.PolicyVersion, dayStart, dayEnd).Scan(&createdToday); err != nil {
		return ExecutionReceipt{}, err
	}
	if createdToday >= command.Policy.BlastRadius.PerDay {
		receipt.State, receipt.ReasonCode = ExecutionBlastSuppressed, "PER_DAY_LIMIT"
		return receipt, nil
	}
	matterPayload, err := json.Marshal(command.Matter)
	if err != nil {
		return ExecutionReceipt{}, ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17::jsonb,$18::uuid,$19,NULL,NULL,'',0,$20,$20,1,$21::uuid)`, command.Matter.ID, command.Matter.TenantID, command.Matter.Reference, command.Matter.Type, command.Matter.Status, command.Matter.Priority, command.Matter.Title, command.Matter.Summary, string(command.Matter.Scope), command.Matter.SourceType, command.Matter.SourceID, command.Matter.TriggerType, command.Matter.TriggerID, command.Matter.TriggerKey, string(command.Matter.KnownFacts), string(command.Matter.MissingFacts), string(command.Matter.Contradictions), command.Route.OwnerPrincipalID, command.Matter.RequiredAuthority, receipt.CreatedAt, receipt.LegalEntityID); err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,1,'MATTER_CREATED',$3::jsonb,'SERVICE',$4::uuid,$5)`, receipt.TenantID, command.Matter.ID, string(matterPayload), command.Route.ServicePrincipalID, receipt.CreatedAt); err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'MATTER_CREATED',$3::jsonb,$4,$4,$4)`, receipt.TenantID, command.Matter.ID, string(matterPayload), receipt.CreatedAt); err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	contractAggregateVersion := int64(2)
	if command.Link != nil {
		if err = insertProgramLinkTx(ctx, tx, command); err != nil {
			return ExecutionReceipt{}, err
		}
		contractAggregateVersion = 3
	}
	if err = insertOutcomeContractTx(ctx, tx, command, contractAggregateVersion); err != nil {
		return ExecutionReceipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO form_response_policy_adverse_episodes(id,tenant_id,legal_entity_id,policy_code,policy_id,policy_version,subject_type,subject_id,state,matter_id,last_response_revision_id,opened_at,closed_at,updated_at,record_version) SELECT $1::uuid,t.id,le.id,$4,$5::uuid,$6,$7,$8::uuid,'OPEN',$9::uuid,$10::uuid,$11,NULL,$11,1 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2`, command.Episode.ID, command.Episode.TenantID, command.Episode.LegalEntityID, command.Episode.PolicyCode, command.Episode.PolicyID, command.Episode.PolicyVersion, command.Episode.SubjectType, command.Episode.SubjectID, command.Matter.ID, command.Response.ID, receipt.CreatedAt); err != nil {
		return ExecutionReceipt{}, normalizePostgresError(err)
	}
	receipt.MatterID, receipt.CreatedMatter = command.Matter.ID, true
	return receipt, nil
}

func insertProgramLinkTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand) error {
	link := command.Link
	if link == nil {
		return nil
	}
	payload, err := json.Marshal(link)
	if err != nil {
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO matter_links(id,tenant_id,matter_id,program_id,relationship,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,'AFFECTS',$5)`, link.ID, link.TenantID, link.MatterID, link.ProgramID, link.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	if tag, err := tx.Exec(ctx, `UPDATE matters SET version=2 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=1`, link.MatterID, link.TenantID); err != nil {
		return normalizePostgresError(err)
	} else if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,2,'MATTER_LINKED',$3::jsonb,'SERVICE',$4::uuid,$5)`, link.TenantID, link.MatterID, payload, command.Route.ServicePrincipalID, link.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'MATTER_LINKED',$3::jsonb,$4,$4,$4)`, link.TenantID, link.MatterID, payload, link.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	return nil
}

func insertOutcomeContractTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand, aggregateVersion int64) error {
	contract := command.Outcome
	payload, err := json.Marshal(contract)
	if err != nil {
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO verification_contracts(id,tenant_id,matter_id,supersedes_contract_id,action_id,expected_outcome,baseline,scope,measurement_source_id,threshold,observation_period_minutes,authority_principal_id,failure_response,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULL,NULL,$4,$5::jsonb,$6::jsonb,NULL,$7::jsonb,$8,$9::uuid,$10,'ACTIVE',$11,$11,1)`, contract.ID, contract.TenantID, contract.MatterID, contract.ExpectedOutcome, string(contract.Baseline), string(contract.Scope), string(contract.Threshold), contract.ObservationPeriodMinutes, contract.AuthorityPrincipalID, contract.FailureResponse, contract.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	if tag, err := tx.Exec(ctx, `UPDATE matters SET version=$3 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=$4`, contract.MatterID, contract.TenantID, aggregateVersion, aggregateVersion-1); err != nil {
		return normalizePostgresError(err)
	} else if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,$6,'VERIFICATION_CONTRACT_ADDED',$3::jsonb,'SERVICE',$4::uuid,$5)`, contract.TenantID, contract.MatterID, payload, command.Route.ServicePrincipalID, contract.CreatedAt, aggregateVersion); err != nil {
		return normalizePostgresError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER',$2::uuid,'VERIFICATION_CONTRACT_ADDED',$3::jsonb,$4,$4,$4)`, contract.TenantID, contract.MatterID, payload, contract.CreatedAt); err != nil {
		return normalizePostgresError(err)
	}
	return nil
}

func updateReusedMatterTx(ctx context.Context, tx pgx.Tx, command ExecutionCommand, matterID string) error {
	facts := string(command.Matter.KnownFacts)
	tag, err := tx.Exec(ctx, `
		WITH prior AS (
			SELECT m.* FROM matters m
			WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)
			  AND legal_entity_id=$3::uuid AND status NOT IN ('CLOSED','CANCELLED')
			FOR UPDATE
		), updated AS (
			UPDATE matters m SET source_id=CASE WHEN prior.updated_at<=$6 THEN $4::text ELSE m.source_id END,known_facts=CASE WHEN prior.updated_at<=$6 THEN m.known_facts || $5::jsonb ELSE m.known_facts END,updated_at=GREATEST(m.updated_at,$6),version=m.version+1
			FROM prior WHERE m.id=prior.id RETURNING m.*
		), payload AS (
			SELECT jsonb_build_object(
				'matter',(to_jsonb(updated)-'matter_type') || jsonb_build_object('type',updated.matter_type),
				'previous',(to_jsonb(prior)-'matter_type') || jsonb_build_object('type',prior.matter_type),
				'rationale','Another scored response matched the active policy; the latest response remains the current Matter source.'
			) value,updated.* FROM updated CROSS JOIN prior
		), event AS (
			INSERT INTO continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at)
			SELECT tenant_id,'MATTER',id,version,'MATTER_DETAILS_UPDATED',value,'SERVICE',$7::uuid,$6 FROM payload
			RETURNING tenant_id,aggregate_id,aggregate_version,payload
		)
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		SELECT tenant_id,'MATTER',aggregate_id,'MATTER_DETAILS_UPDATED',payload,$6,$6,$6 FROM event`, matterID, command.Receipt.TenantID, command.Receipt.LegalEntityID, command.Response.ID, facts, command.Receipt.CreatedAt, command.Route.ServicePrincipalID)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func closeEpisodeForVerifiedMatterTx(ctx context.Context, tx pgx.Tx, tenantID, episodeID, matterID, responseID string, closedAt time.Time) error {
	var version int64
	if err := tx.QueryRow(ctx, `UPDATE form_response_policy_adverse_episodes SET state='CLOSED',closed_at=$3,updated_at=$3,record_version=record_version+1 WHERE id=$1::uuid AND matter_id=$2::uuid AND state='OPEN' RETURNING record_version`, episodeID, matterID, closedAt).Scan(&version); err != nil {
		return normalizePostgresError(err)
	}
	payload, _ := json.Marshal(map[string]any{"episode_id": episodeID, "matter_id": matterID, "response_revision_id": responseID, "state": EpisodeClosed, "version": version})
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'FORM_RESPONSE_POLICY_EPISODE',$2::uuid,'FORM_RESPONSE_POLICY_EPISODE_CLOSED',$3::jsonb,$4,$4,$4) ON CONFLICT DO NOTHING`, tenantID, episodeID, payload, closedAt)
	return normalizePostgresError(err)
}

func insertExecutionTx(ctx context.Context, tx pgx.Tx, value ExecutionReceipt) error {
	_, err := tx.Exec(ctx, `INSERT INTO form_response_policy_executions(id,tenant_id,legal_entity_id,policy_id,policy_version,automation_policy_id,automation_policy_version,response_revision_id,state,matter_id,reason_code,created_matter,created_at) SELECT $1::uuid,t.id,le.id,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9,NULLIF($10,'')::uuid,$11,$12,$13 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2`, value.ID, value.TenantID, value.LegalEntityID, value.PolicyID, value.PolicyVersion, value.AutomationPolicyID, value.AutomationPolicyVersion, value.ResponseRevisionID, value.State, value.MatterID, value.ReasonCode, value.CreatedMatter, value.CreatedAt)
	return normalizePostgresError(err)
}

func getExecutionTx(ctx context.Context, tx pgx.Tx, probe ExecutionReceipt) (ExecutionReceipt, error) {
	var value ExecutionReceipt
	err := tx.QueryRow(ctx, `SELECT e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.policy_id::text,e.policy_version,e.automation_policy_id::text,e.automation_policy_version,e.response_revision_id::text,e.state,COALESCE(e.matter_id::text,''),e.reason_code,e.created_matter,e.created_at FROM form_response_policy_executions e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id::text=$3 AND e.policy_version=$4 AND e.response_revision_id::text=$5`, probe.TenantID, probe.LegalEntityID, probe.PolicyID, probe.PolicyVersion, probe.ResponseRevisionID).Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyID, &value.PolicyVersion, &value.AutomationPolicyID, &value.AutomationPolicyVersion, &value.ResponseRevisionID, &value.State, &value.MatterID, &value.ReasonCode, &value.CreatedMatter, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionReceipt{}, ErrNotFound
	}
	return value, err
}

func validExecutionCommand(command ExecutionCommand) bool {
	receipt := command.Receipt
	if strings.TrimSpace(command.EventID) == "" || strings.TrimSpace(receipt.ID) == "" || strings.TrimSpace(receipt.TenantID) == "" || strings.TrimSpace(receipt.LegalEntityID) == "" || strings.TrimSpace(receipt.PolicyID) == "" || receipt.PolicyVersion < 1 || strings.TrimSpace(receipt.ResponseRevisionID) == "" || receipt.CreatedAt.IsZero() {
		return false
	}
	if receipt.State != ExecutionApplied {
		if receipt.State == ExecutionFailed {
			if strings.TrimSpace(receipt.ReasonCode) == "" || (command.FailureMatter == nil) != (command.FailureAction == nil) {
				return false
			}
			if command.FailureMatter != nil {
				return command.FailureMatter.TenantID == receipt.TenantID && command.FailureMatter.LegalEntityID == receipt.LegalEntityID && command.FailureAction.TenantID == receipt.TenantID && command.FailureAction.MatterID == command.FailureMatter.ID && strings.TrimSpace(command.FailureMatter.OwnerPrincipalID) != "" && command.FailureAction.OwnerPrincipalID == command.FailureMatter.OwnerPrincipalID
			}
		}
		return true
	}
	linkValid := command.Link == nil || command.Link.TenantID == receipt.TenantID && command.Link.MatterID == command.Matter.ID && command.Link.ProgramID == command.Route.ProgramID
	return strings.TrimSpace(command.Episode.ID) != "" && strings.TrimSpace(command.Matter.ID) != "" && strings.TrimSpace(command.Outcome.ID) != "" && command.Episode.MatterID == command.Matter.ID && command.Outcome.MatterID == command.Matter.ID && command.Matter.TenantID == receipt.TenantID && command.Matter.LegalEntityID == receipt.LegalEntityID && command.Outcome.TenantID == receipt.TenantID && strings.TrimSpace(command.Route.ServicePrincipalID) != "" && strings.TrimSpace(command.Route.OwnerPrincipalID) != "" && strings.TrimSpace(command.Route.ReviewerPrincipalID) != "" && command.Outcome.AuthorityPrincipalID == command.Route.ReviewerPrincipalID && linkValid
}

var _ ExecutionStore = (*PostgresRepository)(nil)
