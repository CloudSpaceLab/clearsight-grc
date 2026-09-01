//go:build postgres && postgresintegration

package formpolicy

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	policyTenantID     = "9f650000-0000-7650-8650-000000000001"
	policyOtherTenant  = "9f650000-0000-7650-8650-000000000002"
	policyEntityID     = "9f650000-0000-7650-8650-000000000003"
	policyOtherEntity  = "9f650000-0000-7650-8650-000000000004"
	policyMakerID      = "9f650000-0000-7650-8650-000000000005"
	policyCheckerID    = "9f650000-0000-7650-8650-000000000006"
	policyFormID       = "9f650000-0000-7650-8650-000000000007"
	policyAutomationID = "9f650000-0000-7650-8650-000000000008"
	policyDefinitionID = "9f650000-0000-7650-8650-000000000009"
	policyExecutionID  = "9f650000-0000-7650-8650-000000000010"
	policyEpisodeID    = "9f650000-0000-7650-8650-000000000011"
)

func TestPostgresPolicyScopeConcurrencyValidationAndIdempotency(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cleanupPolicyFixture(t, pool)
	t.Cleanup(func() { cleanupPolicyFixture(t, pool) })
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	seedPolicyFixture(t, ctx, pool, now)
	repo := NewPostgresRepository(pool)
	policy := postgresPolicyFixture(now)
	created, err := repo.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != policyTenantID || created.LegalEntityID != policyEntityID {
		t.Fatalf("scope = %#v", created)
	}
	var events, outbox int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM form_response_policy_events WHERE policy_id=$1::uuid),(SELECT count(*) FROM outbox_events WHERE aggregate_type='FORM_RESPONSE_POLICY' AND aggregate_id=$1::uuid)`, policyDefinitionID).Scan(&events, &outbox); err != nil || events != 1 || outbox != 1 {
		t.Fatalf("events=%d outbox=%d err=%v", events, outbox, err)
	}

	crossTenant := policy
	crossTenant.ID, crossTenant.TenantID, crossTenant.LegalEntityID = "9f650000-0000-7650-8650-000000000012", policyOtherTenant, policyOtherEntity
	crossTenant.Checksum = policyChecksum(crossTenant)
	if _, err := repo.CreatePolicy(ctx, crossTenant); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-tenant reference err = %v", err)
	}

	left, _ := repo.GetPolicy(ctx, policyTenantID, policyEntityID, created.ID)
	right := left
	left.RecordVersion++
	left.UpdatedAt = now.Add(time.Minute)
	left.LastActorID = policyMakerID
	if _, err := repo.UpdatePolicy(ctx, left, 1); err != nil {
		t.Fatal(err)
	}
	right.RecordVersion++
	right.UpdatedAt = now.Add(2 * time.Minute)
	right.LastActorID = policyMakerID
	if _, err := repo.UpdatePolicy(ctx, right, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent update err = %v", err)
	}

	var responseID, subjectID string
	if err := pool.QueryRow(ctx, `SELECT md5('form-policy:response:1')::uuid::text,md5('form-policy:subject:1')::uuid::text`).Scan(&responseID, &subjectID); err != nil {
		t.Fatal(err)
	}
	receipt := ExecutionReceipt{ID: policyExecutionID, TenantID: policyTenantID, LegalEntityID: policyEntityID, PolicyID: policyDefinitionID, PolicyVersion: 1, AutomationPolicyID: policyAutomationID, AutomationPolicyVersion: 2, ResponseRevisionID: responseID, State: ExecutionShadow, ReasonCode: "POLICY_MATCHED", CreatedAt: now}
	first, inserted, err := repo.CreateExecution(ctx, receipt)
	if err != nil || !inserted {
		t.Fatalf("first execution = %#v inserted=%v err=%v", first, inserted, err)
	}
	replay := receipt
	replay.ID = "9f650000-0000-7650-8650-000000000013"
	stored, inserted, err := repo.CreateExecution(ctx, replay)
	if err != nil || inserted || stored.ID != receipt.ID {
		t.Fatalf("replay = %#v inserted=%v err=%v", stored, inserted, err)
	}

	episode := AdverseEpisode{ID: policyEpisodeID, TenantID: policyTenantID, LegalEntityID: policyEntityID, PolicyCode: policy.Code, PolicyID: policy.ID, PolicyVersion: 1, SubjectType: "VENDOR", SubjectID: subjectID, State: EpisodeOpen, LastResponseRevisionID: responseID, OpenedAt: now, UpdatedAt: now, RecordVersion: 1}
	opened, inserted, err := repo.OpenEpisode(ctx, episode)
	if err != nil || !inserted {
		t.Fatalf("episode = %#v inserted=%v err=%v", opened, inserted, err)
	}
	replayEpisode := episode
	replayEpisode.ID = "9f650000-0000-7650-8650-000000000014"
	opened, inserted, err = repo.OpenEpisode(ctx, replayEpisode)
	if err != nil || inserted || opened.ID != episode.ID {
		t.Fatalf("episode replay = %#v inserted=%v err=%v", opened, inserted, err)
	}

	if _, err := pool.Exec(ctx, `UPDATE form_response_policy_definitions SET eligibility='{}'::jsonb WHERE id=$1::uuid`, policyDefinitionID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetPolicy(ctx, policyTenantID, policyEntityID, policyDefinitionID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed definition err = %v", err)
	}
}

func postgresPolicyFixture(now time.Time) Policy {
	input := validPolicyInput("poor-vendor-response", RolloutShadow)
	input.AutomationPolicyID, input.AutomationPolicyVersion = policyAutomationID, 2
	input.Eligibility.FormTemplateID, input.Eligibility.FormTemplateVersion = policyFormID, 1
	_ = normalizeCreateInput(&input, now)
	value := Policy{ID: policyDefinitionID, TenantID: policyTenantID, LegalEntityID: policyEntityID, Code: input.Code, Name: input.Name, Purpose: input.Purpose, ActionClass: ActionClassCreateMatter, AutomationPolicyID: input.AutomationPolicyID, AutomationPolicyVersion: input.AutomationPolicyVersion, Eligibility: input.Eligibility, Action: input.Action, BlastRadius: input.BlastRadius, Outcome: input.Outcome, Rollout: input.Rollout, Status: PolicyDraft, MakerID: policyMakerID, Version: 1, RecordVersion: 1, CreatedAt: now, UpdatedAt: now, LastActorID: policyMakerID}
	value.Checksum = policyChecksum(value)
	return value
}

func seedPolicyFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'form-policy-integration','Form policy integration'),($2::uuid,'form-policy-other','Form policy other');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($3::uuid,$1::uuid,'POLICY','Policy entity','NG',$9),($4::uuid,$2::uuid,'OTHER','Other entity','NG',$9);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($5::uuid,$1::uuid,'PERSON','Policy maker','ACTIVE',$9),($6::uuid,$1::uuid,'PERSON','Policy checker','ACTIVE',$9);
		INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,presentation,scoring_mode,score_profile,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at)
		VALUES($7::uuid,$1::uuid,$3::uuid,'POLICY-FORM','Policy form','Collect scored vendor evidence.','{"default_mode":"CLASSIC","allow_mode_switch":false}'::jsonb,'COMPLIANCE','{"version":"v1"}'::jsonb,'[{"id":"general","title":"Questions"}]'::jsonb,'[{"id":"q1","section_id":"general","label":"Question","type":"short_text","required":true}]'::jsonb,'ACTIVE',true,$9,1,$5::uuid,$9,$9);
		INSERT INTO automation_policies(id,tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,status,effective_from,version,ai_definition,rollout_mode,maker_id,checker_id,checksum,approved_at,activated_at,record_version)
		VALUES($8::uuid,$1::uuid,'FORM-POOR-RESPONSE','Poor form response','FORM_RESPONSE_CREATE_MATTER','{}','{"per_run":10}','{"method":"matter_outcome"}','ACTIVE',$9,2,'{}','SHADOW',$5::uuid,$6::uuid,'automation-v2',$9,$9,1)`, pgx.QueryExecModeSimpleProtocol, policyTenantID, policyOtherTenant, policyEntityID, policyOtherEntity, policyMakerID, policyCheckerID, policyFormID, policyAutomationID, now); err != nil {
		t.Fatal(err)
	}
	seedPolicyResponse(t, ctx, pool, now)
}

func seedPolicyResponse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_form_distributions(id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,title,purpose,access_policy,status,deadline,route_expires_at,created_by,version,created_at,updated_at)
		VALUES(md5('form-policy:distribution:1')::uuid,$1::uuid,$2::uuid,$4::uuid,1,'VENDOR',md5('form-policy:subject:1')::uuid,'Vendor response','Review the completed response.','DIRECT_MAGIC_LINK','COMPLETED',$5+interval '30 days',$5+interval '7 days',$3::uuid,1,$5,$5);
		INSERT INTO capture_requests(id,tenant_id,legal_entity_id,distribution_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,known_facts,presentation,scoring_mode,sections,fields,source_bindings,form_template_id,form_template_version,status,created_by,version,created_at,updated_at)
		VALUES(md5('form-policy:request:1')::uuid,$1::uuid,$2::uuid,md5('form-policy:distribution:1')::uuid,'VENDOR',md5('form-policy:subject:1')::uuid,'Vendor request','Review the response.','Review the response.','INTERNAL','INTERNAL',5,$5+interval '30 days','{}','{"default_mode":"CLASSIC","allow_mode_switch":false}','COMPLIANCE','[{"id":"general","title":"Questions"}]','[{"id":"q1","section_id":"general","label":"Question","type":"short_text","required":true}]','[]',$4::uuid,1,'SUBMITTED',$3::uuid,1,$5,$5);
		INSERT INTO capture_response_workspaces(id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at) VALUES(md5('form-policy:workspace:1')::uuid,$1::uuid,$2::uuid,md5('form-policy:distribution:1')::uuid,'COMPLETED',1,$5,$5);
		INSERT INTO capture_submissions(id,tenant_id,request_id,submitted_by,channel,answers,submitted_at,created_at,distribution_id) VALUES(md5('form-policy:submission:1')::uuid,$1::uuid,md5('form-policy:request:1')::uuid,$3::uuid,'INTERNAL','{"q1":"no"}',$5,$5,md5('form-policy:distribution:1')::uuid);
		INSERT INTO capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,achieved_assurance,signoff_summary,compliance_score,scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at,score_mode,score_direction,raw_score,adverse_score,concern_band,score_state,score_result,score_profile_checksum,score_calculated_at)
		VALUES(md5('form-policy:response:1')::uuid,$1::uuid,$2::uuid,md5('form-policy:distribution:1')::uuid,md5('form-policy:workspace:1')::uuid,md5('form-policy:submission:1')::uuid,1,'EMAIL_VERIFIED','{}',20,100,'FINAL','[]','v1',true,$5,'COMPLIANCE','LOW_IS_POOR',20,80,'HIGH','FINAL','{"profile_version":"v1"}','profile-v1',$5)`, pgx.QueryExecModeSimpleProtocol, policyTenantID, policyEntityID, policyMakerID, policyFormID, now); err != nil {
		t.Fatal(err)
	}
}

func cleanupPolicyFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		DELETE FROM outbox_events WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM form_response_policy_events WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM form_response_policy_adverse_episodes WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM form_response_policy_executions WHERE tenant_id IN ($1::uuid,$2::uuid);
		UPDATE form_response_policy_definitions SET approved_simulation_id=NULL WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM form_response_policy_simulations WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM form_response_policy_definitions WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM capture_response_revisions WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM capture_submissions WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM capture_response_workspaces WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM capture_requests WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM capture_form_distributions WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM automation_policies WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM monitoring_form_templates WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM principals WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM legal_entities WHERE tenant_id IN ($1::uuid,$2::uuid);
		DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, pgx.QueryExecModeSimpleProtocol, policyTenantID, policyOtherTenant); err != nil {
		t.Fatal(err)
	}
}
