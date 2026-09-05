//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type storedExecution struct {
	ID            string
	State         string
	MatterID      string
	CreatedMatter bool
}

func ensureActivatedAcceptancePolicy(
	ctx context.Context,
	service *formpolicy.Service,
	repo *formpolicy.PostgresRepository,
	maker formpolicy.Actor,
	checker formpolicy.Actor,
	authorizer formpolicy.Actor,
	input formpolicy.CreateInput,
	rollout formpolicy.RolloutMode,
) (formpolicy.Policy, error) {
	values, err := repo.ListPolicies(ctx, maker.TenantID, maker.LegalEntityID, 200)
	if err != nil {
		return formpolicy.Policy{}, err
	}
	for _, value := range values {
		if value.Code != input.Code || value.Rollout != rollout {
			continue
		}
		if value.Status == formpolicy.PolicyActive || rollout == formpolicy.RolloutShadow && value.Status == formpolicy.PolicySuspended {
			return value, nil
		}
	}

	value, err := service.Create(ctx, maker, input)
	if err != nil {
		return formpolicy.Policy{}, err
	}
	simulation, err := service.Simulate(ctx, checker, value.ID, value.RecordVersion)
	if err != nil {
		return formpolicy.Policy{}, err
	}
	value, err = service.Submit(ctx, maker, value.ID, value.RecordVersion, simulation.ID)
	if err != nil {
		return formpolicy.Policy{}, err
	}
	value, err = service.Approve(ctx, checker, value.ID, value.RecordVersion, simulation.ID)
	if err != nil {
		return formpolicy.Policy{}, err
	}
	return service.Activate(ctx, authorizer, value.ID, value.RecordVersion)
}

func ensureAcceptanceAutomationPolicy(
	ctx context.Context,
	pool *pgxpool.Pool,
	seed bankverticals.SeedConfig,
	base formpolicy.CreateInput,
	rollout formpolicy.RolloutMode,
	version int64,
) (string, error) {
	eligibility, err := json.Marshal(base.Eligibility)
	if err != nil {
		return "", fmt.Errorf("marshal response-policy eligibility: %w", err)
	}
	blastRadius, err := json.Marshal(base.BlastRadius)
	if err != nil {
		return "", fmt.Errorf("marshal response-policy blast radius: %w", err)
	}
	outcome, err := json.Marshal(base.Outcome)
	if err != nil {
		return "", fmt.Errorf("marshal response-policy outcome: %w", err)
	}
	code := "reference-response-policy-" + strings.ToLower(string(rollout))
	name := "Reference response policy " + strings.ToLower(string(rollout))

	var id string
	err = pool.QueryRow(ctx, `
		WITH tenant AS (
			SELECT id FROM tenants WHERE id::text=$1 OR slug=$1 LIMIT 1
		), legal_entity AS (
			SELECT id,tenant_id FROM legal_entities
			WHERE (id::text=$2 OR code=$2) AND valid_until IS NULL
			LIMIT 1
		), inserted AS (
			INSERT INTO automation_policies(
				id,tenant_id,code,name,action_class,eligibility,blast_radius_limit,
				verification_contract,status,effective_from,version,ai_definition,
				rollout_mode,maker_id,checker_id,checksum,approved_at,activated_at,record_version
			)
			SELECT
				md5('clearsight:'||tenant.id::text||':'||$3||':'||$4::bigint::text)::uuid,
				tenant.id,$3,$5,'FORM_RESPONSE_CREATE_MATTER',$6::jsonb,$7::jsonb,$8::jsonb,
				'ACTIVE',clock_timestamp()-interval '1 minute',$4::bigint,'{}'::jsonb,$9,$10::uuid,$11::uuid,
				md5($3||':'||$4::bigint::text||':'||$9),clock_timestamp(),clock_timestamp(),1
			FROM tenant
			JOIN legal_entity ON legal_entity.tenant_id=tenant.id
			ON CONFLICT DO NOTHING
			RETURNING id
		)
		SELECT id::text FROM inserted
		UNION ALL
		SELECT policy.id::text
		FROM automation_policies policy
		JOIN tenant ON tenant.id=policy.tenant_id
		WHERE policy.code=$3 AND policy.version=$4::bigint
		LIMIT 1`,
		seed.TenantID,
		seed.LegalEntityID,
		code,
		version,
		name,
		string(eligibility),
		string(blastRadius),
		string(outcome),
		string(rollout),
		seed.ActorID,
		seed.ReviewerPrincipalID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("ensure %s automation policy: %w", rollout, err)
	}
	return id, nil
}

func scoredOutboxEvent(ctx context.Context, pool *pgxpool.Pool, tenantID, responseRevisionID string) (workflowruntime.OutboxEvent, error) {
	var event workflowruntime.OutboxEvent
	err := pool.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,aggregate_type,aggregate_id::text,event_type,payload,occurred_at
		FROM outbox_events
		WHERE tenant_id=$1::uuid
		  AND event_type='FORM_RESPONSE_SCORED'
		  AND payload->>'response_revision_id'=$2
		ORDER BY occurred_at DESC,id DESC
		LIMIT 1`, tenantID, responseRevisionID).Scan(
		&event.ID,
		&event.TenantID,
		&event.AggregateType,
		&event.AggregateID,
		&event.EventType,
		&event.Payload,
		&event.OccurredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return workflowruntime.OutboxEvent{}, fmt.Errorf("scored response outbox event is missing for %s", responseRevisionID)
	}
	if err != nil {
		return workflowruntime.OutboxEvent{}, fmt.Errorf("load scored response outbox event: %w", err)
	}
	return event, nil
}

func executionForResponse(ctx context.Context, pool *pgxpool.Pool, policyID, responseRevisionID string) (storedExecution, error) {
	var value storedExecution
	err := pool.QueryRow(ctx, `
		SELECT id::text,state,COALESCE(matter_id::text,''),created_matter
		FROM form_response_policy_executions
		WHERE policy_id=$1::uuid AND response_revision_id=$2::uuid
		ORDER BY created_at DESC,id DESC
		LIMIT 1`, policyID, responseRevisionID).Scan(
		&value.ID,
		&value.State,
		&value.MatterID,
		&value.CreatedMatter,
	)
	if err != nil {
		return storedExecution{}, fmt.Errorf("load response-policy execution: %w", err)
	}
	return value, nil
}

func verifyExactlyOnceMatter(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID string,
	legalEntityID string,
	policyID string,
	responseID string,
	execution storedExecution,
) (bool, error) {
	var executionCount, matterCount, episodeCount int
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM form_response_policy_executions
			 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
			   AND policy_id=$3::uuid AND response_revision_id=$4::uuid),
			(SELECT count(*) FROM matters
			 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
			   AND id=$5::uuid AND source_type='FORM_RESPONSE'),
			(SELECT count(*) FROM form_response_policy_adverse_episodes
			 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
			   AND policy_id=$3::uuid AND state='OPEN' AND matter_id=$5::uuid)`,
		tenantID,
		legalEntityID,
		policyID,
		responseID,
		execution.MatterID,
	).Scan(&executionCount, &matterCount, &episodeCount)
	if err != nil {
		return false, fmt.Errorf("verify automatic Matter idempotency: %w", err)
	}
	return executionCount == 1 && matterCount == 1 && episodeCount == 1, nil
}

func existingAcceptanceExecution(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (formPolicyAcceptanceResult, bool, error) {
	var result formPolicyAcceptanceResult
	err := pool.QueryRow(ctx, `
		SELECT
			policy.id::text,
			policy.version,
			execution.response_revision_id::text,
			execution.state,
			COALESCE(execution.matter_id::text,''),
			execution.created_matter,
			execution.id::text
		FROM form_response_policy_definitions policy
		JOIN form_response_policy_executions execution
		  ON execution.policy_id=policy.id
		 AND execution.tenant_id=policy.tenant_id
		 AND execution.legal_entity_id=policy.legal_entity_id
		WHERE policy.tenant_id=$1::uuid
		  AND policy.legal_entity_id=$2::uuid
		  AND policy.code=$3
		  AND policy.rollout_mode='ENFORCE'
		  AND policy.status='ACTIVE'
		  AND execution.state IN ('APPLIED','REUSED')
		ORDER BY policy.version DESC,execution.created_at DESC
		LIMIT 1`, seed.TenantID, seed.LegalEntityID, responsePolicyAcceptanceCode).Scan(
		&result.PolicyID,
		&result.PolicyVersion,
		&result.PoorResponseID,
		&result.PoorExecutionState,
		&result.MatterID,
		&result.CreatedMatter,
		&result.ReplayExecutionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return formPolicyAcceptanceResult{}, false, nil
	}
	if err != nil {
		return formPolicyAcceptanceResult{}, false, fmt.Errorf("load existing response-policy acceptance: %w", err)
	}
	result.ExactlyOnceMatter, err = verifyExactlyOnceMatter(
		ctx,
		pool,
		seed.TenantID,
		seed.LegalEntityID,
		result.PolicyID,
		result.PoorResponseID,
		storedExecution{
			ID:            result.ReplayExecutionID,
			State:         result.PoorExecutionState,
			MatterID:      result.MatterID,
			CreatedMatter: result.CreatedMatter,
		},
	)
	return result, true, err
}
