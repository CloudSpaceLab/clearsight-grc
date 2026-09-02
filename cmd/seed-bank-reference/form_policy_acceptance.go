//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const responsePolicyAcceptanceCode = "reference-poor-control-response"

type formPolicyAcceptanceResult struct {
	PolicyID           string `json:"policy_id"`
	PolicyVersion      int64  `json:"policy_version"`
	GoodResponseID     string `json:"good_response_id"`
	PoorResponseID     string `json:"poor_response_id"`
	GoodExecutionState string `json:"good_execution_state"`
	PoorExecutionState string `json:"poor_execution_state"`
	MatterID           string `json:"matter_id"`
	CreatedMatter      bool   `json:"created_matter"`
	ReplayExecutionID  string `json:"replay_execution_id"`
	ExactlyOnceMatter  bool   `json:"exactly_once_matter"`
	AlreadySeeded      bool   `json:"already_seeded"`
}

type seedFormReader struct{ repo *monitoring.PostgresRepository }

func (r seedFormReader) GetDistributionFormRevision(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (evidence.DistributionFormRevision, error) {
	form, err := r.repo.ReusableFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return evidence.DistributionFormRevision{}, err
	}
	return evidence.DistributionFormRevision{
		ID: form.ID, TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, Version: form.Version,
		Sensitivity: form.Sensitivity, Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile,
		Sections: append([]formcontract.Section(nil), form.Sections...), Fields: append([]formcontract.Field(nil), form.Fields...),
		Active: form.Status == monitoring.LifecycleActive && form.IsCurrent,
	}, nil
}

func seedFormPolicyAcceptance(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig, monitoringRepo *monitoring.PostgresRepository, evidenceRepo *evidence.PostgresRepository, scoring scoringAcceptanceResult) (formPolicyAcceptanceResult, error) {
	if existing, ok, err := existingAcceptanceExecution(ctx, pool, seed); err != nil {
		return formPolicyAcceptanceResult{}, err
	} else if ok {
		existing.AlreadySeeded = true
		return existing, nil
	}
	if scoring.FormID == "" || scoring.FormVersion < 1 || scoring.SubjectID == "" {
		return formPolicyAcceptanceResult{}, fmt.Errorf("scoring acceptance population is incomplete")
	}
	if seed.ActorID == "" || seed.ReviewerPrincipalID == "" || seed.SignatoryPrincipalID == "" {
		return formPolicyAcceptanceResult{}, fmt.Errorf("response-policy acceptance requires maker, reviewer and authorizer principals")
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("configure response-policy acceptance keyring: %w", err)
	}
	distributionStore := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	access, err := evidence.NewDistributionAccessService(distributionStore, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("configure response-policy acceptance access: %w", err)
	}
	form, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, scoring.FormID, scoring.FormVersion)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("load scoring acceptance form: %w", err)
	}

	policyRepo := formpolicy.NewPostgresRepository(pool)
	automation := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	authorityService := authority.NewEffectivePostgresService(pool)
	policyService := formpolicy.NewService(policyRepo, seedFormReader{repo: monitoringRepo}, distributions)
	policyService.ConfigureActivationAuthority(formpolicy.ActivationAuthorityResolver{Automation: automation, Authority: authorityService, Subjects: evidence.CanonicalSubjectTypeRegistry{}})
	maker := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ActorID}
	checker := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.ReviewerPrincipalID}
	authorizer := formpolicy.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.SignatoryPrincipalID}
	base := formpolicy.CreateInput{
		Code: responsePolicyAcceptanceCode, Name: "Create a Matter for poor control responses",
		Purpose: "Reference acceptance policy proving that a poor scored response creates one governed Matter while a good response does not.",
		Eligibility: formpolicy.Eligibility{FormTemplateID: scoring.FormID, FormTemplateVersion: scoring.FormVersion, SubjectTypes: []string{"PROGRAM"}, CurrentOnly: true, MinimumCoverage: 1, Bands: []formcontract.ConcernBand{formcontract.ConcernHigh, formcontract.ConcernCritical}},
		Action: formpolicy.MatterAction{Type: "CONTROL_GAP", Priority: 4, TitleTemplate: "Review {{form_title}}", SummaryTemplate: "A completed control response crossed the approved concern threshold.", RequestedHandling: "Review the adverse response, record treatment and independently verify the outcome."},
		BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 25},
		Outcome: formpolicy.OutcomeContract{ExpectedOutcome: "The control concern is remediated or an approved treatment is recorded.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"},
	}

	shadowAutomationID, err := ensureAcceptanceAutomationPolicy(ctx, pool, seed, base, formpolicy.RolloutShadow, 1)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	shadowInput := base
	shadowInput.Rollout, shadowInput.AutomationPolicyID, shadowInput.AutomationPolicyVersion = formpolicy.RolloutShadow, shadowAutomationID, 1
	shadow, err := ensureActivatedAcceptancePolicy(ctx, policyService, policyRepo, maker, checker, authorizer, shadowInput, formpolicy.RolloutShadow)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("activate shadow response policy: %w", err)
	}
	if shadow.Status == formpolicy.PolicyActive {
		if _, err = policyService.Suspend(ctx, authorizer, shadow.ID, shadow.RecordVersion); err != nil {
			return formPolicyAcceptanceResult{}, fmt.Errorf("suspend shadow response policy: %w", err)
		}
	}

	enforceAutomationID, err := ensureAcceptanceAutomationPolicy(ctx, pool, seed, base, formpolicy.RolloutEnforce, 1)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	enforceInput := base
	enforceInput.Rollout, enforceInput.AutomationPolicyID, enforceInput.AutomationPolicyVersion = formpolicy.RolloutEnforce, enforceAutomationID, 1
	enforce, err := ensureActivatedAcceptancePolicy(ctx, policyService, policyRepo, maker, checker, authorizer, enforceInput, formpolicy.RolloutEnforce)
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("activate enforce response policy: %w", err)
	}

	good, err := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, scoring.SubjectID, scoringResponseFixture{label: "post-policy-good", answers: scoringResponseFixtures[0].answers}, time.Now().UTC())
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("submit post-policy good response: %w", err)
	}
	poor, err := submitScoringAcceptanceResponse(ctx, distributions, access, seed, form, scoring.SubjectID, scoringResponseFixture{label: "post-policy-poor", answers: scoringResponseFixtures[2].answers}, time.Now().UTC())
	if err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("submit post-policy poor response: %w", err)
	}
	executor := formpolicy.NewExecutor(policyRepo, distributions, formpolicy.ExecutionAuthorityResolver{Automation: automation, Authority: authorityService, Subjects: evidenceRepo})
	publisher := formpolicy.ScoredResponsePublisher{Handler: executor}
	goodEvent, err := scoredOutboxEvent(ctx, pool, seed.TenantID, good.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	poorEvent, err := scoredOutboxEvent(ctx, pool, seed.TenantID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if err = publisher.Publish(ctx, goodEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("execute good scored response: %w", err)
	}
	if err = publisher.Publish(ctx, poorEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("execute poor scored response: %w", err)
	}
	goodExecution, err := executionForResponse(ctx, pool, enforce.ID, good.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	poorExecution, err := executionForResponse(ctx, pool, enforce.ID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	if goodExecution.State != string(formpolicy.ExecutionNotMatched) || goodExecution.MatterID != "" || goodExecution.CreatedMatter {
		return formPolicyAcceptanceResult{}, fmt.Errorf("good response created or matched a Matter: %#v", goodExecution)
	}
	if poorExecution.State != string(formpolicy.ExecutionApplied) || poorExecution.MatterID == "" || !poorExecution.CreatedMatter {
		return formPolicyAcceptanceResult{}, fmt.Errorf("poor response did not create a governed Matter: %#v", poorExecution)
	}
	if err = publisher.Publish(ctx, poorEvent); err != nil {
		return formPolicyAcceptanceResult{}, fmt.Errorf("replay poor scored response: %w", err)
	}
	replayed, err := executionForResponse(ctx, pool, enforce.ID, poor.ID)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	exactlyOnce, err := verifyExactlyOnceMatter(ctx, pool, seed.TenantID, seed.LegalEntityID, enforce.ID, poor.ID, poorExecution)
	if err != nil {
		return formPolicyAcceptanceResult{}, err
	}
	return formPolicyAcceptanceResult{PolicyID: enforce.ID, PolicyVersion: enforce.Version, GoodResponseID: good.ID, PoorResponseID: poor.ID, GoodExecutionState: goodExecution.State, PoorExecutionState: poorExecution.State, MatterID: poorExecution.MatterID, CreatedMatter: poorExecution.CreatedMatter, ReplayExecutionID: replayed.ID, ExactlyOnceMatter: exactlyOnce && replayed.ID == poorExecution.ID && replayed.MatterID == poorExecution.MatterID}, nil
}

type storedExecution struct { ID, State, MatterID string; CreatedMatter bool }

func ensureActivatedAcceptancePolicy(ctx context.Context, service *formpolicy.Service, repo *formpolicy.PostgresRepository, maker, checker, authorizer formpolicy.Actor, input formpolicy.CreateInput, rollout formpolicy.RolloutMode) (formpolicy.Policy, error) {
	values, err := repo.ListPolicies(ctx, maker.TenantID, maker.LegalEntityID, 200)
	if err != nil { return formpolicy.Policy{}, err }
	for _, value := range values {
		if value.Code == input.Code && value.Rollout == rollout && (value.Status == formpolicy.PolicyActive || rollout == formpolicy.RolloutShadow && value.Status == formpolicy.PolicySuspended) { return value, nil }
	}
	value, err := service.Create(ctx, maker, input); if err != nil { return formpolicy.Policy{}, err }
	simulation, err := service.Simulate(ctx, checker, value.ID, value.RecordVersion); if err != nil { return formpolicy.Policy{}, err }
	value, err = service.Submit(ctx, maker, value.ID, value.RecordVersion, simulation.ID); if err != nil { return formpolicy.Policy{}, err }
	value, err = service.Approve(ctx, checker, value.ID, value.RecordVersion, simulation.ID); if err != nil { return formpolicy.Policy{}, err }
	return service.Activate(ctx, authorizer, value.ID, value.RecordVersion)
}

func ensureAcceptanceAutomationPolicy(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig, base formpolicy.CreateInput, rollout formpolicy.RolloutMode, version int64) (string, error) {
	eligibility, _ := json.Marshal(base.Eligibility); blast, _ := json.Marshal(base.BlastRadius); outcome, _ := json.Marshal(base.Outcome)
	code := "reference-response-policy-" + strings.ToLower(string(rollout)); name := "Reference response policy " + strings.ToLower(string(rollout)); var id string
	err := pool.QueryRow(ctx, `
		WITH t AS (SELECT id FROM tenants WHERE id::text=$1 OR slug=$1 LIMIT 1), le AS (SELECT id,tenant_id FROM legal_entities WHERE (id::text=$2 OR code=$2) AND valid_until IS NULL LIMIT 1), inserted AS (
			INSERT INTO automation_policies(id,tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,status,effective_from,version,ai_definition,rollout_mode,maker_id,checker_id,checksum,approved_at,activated_at,record_version)
			SELECT md5('clearsight:'||t.id::text||':'||$3||':'||$4::text)::uuid,t.id,$3,$5,'FORM_RESPONSE_CREATE_MATTER',$6::jsonb,$7::jsonb,$8::jsonb,'ACTIVE',clock_timestamp()-interval '1 minute',$4,'{}'::jsonb,$9,$10::uuid,$11::uuid,md5($3||':'||$4::text||':'||$9),clock_timestamp(),clock_timestamp(),1 FROM t JOIN le ON le.tenant_id=t.id ON CONFLICT DO NOTHING RETURNING id)
		SELECT id::text FROM inserted UNION ALL SELECT ap.id::text FROM automation_policies ap JOIN t ON t.id=ap.tenant_id WHERE ap.code=$3 AND ap.version=$4 LIMIT 1`, seed.TenantID, seed.LegalEntityID, code, version, name, string(eligibility), string(blast), string(outcome), string(rollout), seed.ActorID, seed.ReviewerPrincipalID).Scan(&id)
	if err != nil { return "", fmt.Errorf("ensure %s automation policy: %w", rollout, err) }
	return id, nil
}

func scoredOutboxEvent(ctx context.Context, pool *pgxpool.Pool, tenantID, responseRevisionID string) (workflowruntime.OutboxEvent, error) {
	var event workflowruntime.OutboxEvent
	err := pool.QueryRow(ctx, `SELECT id::text,tenant_id::text,aggregate_type,aggregate_id::text,event_type,payload,occurred_at FROM outbox_events WHERE tenant_id=$1::uuid AND event_type='FORM_RESPONSE_SCORED' AND payload->>'response_revision_id'=$2 ORDER BY occurred_at DESC,id DESC LIMIT 1`, tenantID, responseRevisionID).Scan(&event.ID, &event.TenantID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.OccurredAt)
	if errors.Is(err, pgx.ErrNoRows) { return workflowruntime.OutboxEvent{}, fmt.Errorf("scored response outbox event is missing for %s", responseRevisionID) }
	if err != nil { return workflowruntime.OutboxEvent{}, fmt.Errorf("load scored response outbox event: %w", err) }
	return event, nil
}

func executionForResponse(ctx context.Context, pool *pgxpool.Pool, policyID, responseRevisionID string) (storedExecution, error) {
	var value storedExecution
	err := pool.QueryRow(ctx, `SELECT id::text,state,COALESCE(matter_id::text,''),created_matter FROM form_response_policy_executions WHERE policy_id=$1::uuid AND response_revision_id=$2::uuid ORDER BY created_at DESC,id DESC LIMIT 1`, policyID, responseRevisionID).Scan(&value.ID, &value.State, &value.MatterID, &value.CreatedMatter)
	if err != nil { return storedExecution{}, fmt.Errorf("load response-policy execution: %w", err) }
	return value, nil
}

func verifyExactlyOnceMatter(ctx context.Context, pool *pgxpool.Pool, tenantID, legalEntityID, policyID, responseID string, execution storedExecution) (bool, error) {
	var executionCount, matterCount, episodeCount int
	err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM form_response_policy_executions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND policy_id=$3::uuid AND response_revision_id=$4::uuid),
		(SELECT count(*) FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$5::uuid AND source_type='FORM_RESPONSE'),
		(SELECT count(*) FROM form_response_policy_adverse_episodes WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND policy_id=$3::uuid AND state='OPEN' AND matter_id=$5::uuid)`, tenantID, legalEntityID, policyID, responseID, execution.MatterID).Scan(&executionCount, &matterCount, &episodeCount)
	if err != nil { return false, fmt.Errorf("verify automatic Matter idempotency: %w", err) }
	return executionCount == 1 && matterCount == 1 && episodeCount == 1, nil
}

func existingAcceptanceExecution(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (formPolicyAcceptanceResult, bool, error) {
	var result formPolicyAcceptanceResult
	err := pool.QueryRow(ctx, `SELECT p.id::text,p.version,e.response_revision_id::text,e.state,COALESCE(e.matter_id::text,''),e.created_matter,e.id::text FROM form_response_policy_definitions p JOIN form_response_policy_executions e ON e.policy_id=p.id AND e.tenant_id=p.tenant_id AND e.legal_entity_id=p.legal_entity_id WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid AND p.code=$3 AND p.rollout_mode='ENFORCE' AND p.status='ACTIVE' AND e.state IN ('APPLIED','REUSED') ORDER BY p.version DESC,e.created_at DESC LIMIT 1`, seed.TenantID, seed.LegalEntityID, responsePolicyAcceptanceCode).Scan(&result.PolicyID, &result.PolicyVersion, &result.PoorResponseID, &result.PoorExecutionState, &result.MatterID, &result.CreatedMatter, &result.ReplayExecutionID)
	if errors.Is(err, pgx.ErrNoRows) { return formPolicyAcceptanceResult{}, false, nil }
	if err != nil { return formPolicyAcceptanceResult{}, false, fmt.Errorf("load existing response-policy acceptance: %w", err) }
	result.ExactlyOnceMatter, err = verifyExactlyOnceMatter(ctx, pool, seed.TenantID, seed.LegalEntityID, result.PolicyID, result.PoorResponseID, storedExecution{ID: result.ReplayExecutionID, State: result.PoorExecutionState, MatterID: result.MatterID, CreatedMatter: result.CreatedMatter})
	return result, true, err
}
