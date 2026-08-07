//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tenantID       = "11111111-1111-7111-8111-111111111111"
	otherTenantID  = "22222222-2222-7222-8222-222222222222"
	legalEntityNG  = "11111111-1111-7111-8111-111111111112"
	legalEntityGH  = "11111111-1111-7111-8111-111111111113"
	principalNG    = "11111111-1111-7111-8111-111111111114"
	principalGH    = "11111111-1111-7111-8111-111111111115"
	roleTemplateID = "11111111-1111-7111-8111-111111111116"
	positionNG     = "11111111-1111-7111-8111-111111111117"
	positionGH     = "11111111-1111-7111-8111-111111111118"
	policyID       = "11111111-1111-7111-8111-111111111119"
	workflowID     = "11111111-1111-7111-8111-111111111120"
	subjectID      = "11111111-1111-7111-8111-111111111121"
)

func TestPostgresRuntimeContracts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	seedPostgres(t, pool)

	t.Run("authority policies support unscheduled drafts and legal-entity-safe roles", func(t *testing.T) {
		service := authority.NewPostgresService(pool)
		policies, err := service.Policies(ctx, "integration-bank")
		if err != nil {
			t.Fatal(err)
		}
		if len(policies) != 1 || policies[0].EffectiveFrom != nil {
			t.Fatalf("expected nullable effective date, got %#v", policies)
		}
		resolution, err := service.Resolve(ctx, authority.ResolveInput{TenantID: "integration-bank", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5})
		if err != nil {
			t.Fatal(err)
		}
		if resolution.Principal.ID != principalNG {
			t.Fatalf("expected NG principal, got %#v", resolution.Principal)
		}
		findings, err := service.Integrity(ctx, "integration-bank")
		if err != nil {
			t.Fatal(err)
		}
		foundUnresolved := false
		for _, finding := range findings {
			if finding.Type == "UNRESOLVED_SELECTOR" {
				foundUnresolved = true
			}
		}
		if !foundUnresolved {
			t.Fatalf("expected unresolved selector finding, got %#v", findings)
		}
	})

	t.Run("workflow reads and transitions stay tenant scoped and audited", func(t *testing.T) {
		repo := workflow.NewPostgresRepository(pool)
		service := workflow.NewService(repo)
		task, err := service.Create(ctx, workflow.CreateInput{TenantID: "integration-bank", WorkflowID: workflowID, StepKey: "review", Responsibility: "REVIEWER", Title: "Review integration policy"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Get(ctx, "other-bank", task.ID); !errors.Is(err, workflow.ErrTaskNotFound) {
			t.Fatalf("expected tenant-scoped not found, got %v", err)
		}
		updated, err := service.Transition(ctx, task.ID, workflow.TransitionInput{TenantID: "integration-bank", ActorID: principalNG, Status: workflow.StatusInProgress, ExpectedVersion: task.Version, Reason: "Reviewer accepted"})
		if err != nil {
			t.Fatal(err)
		}
		if updated.ClaimedAt == nil {
			t.Fatal("expected claimed_at")
		}
		var events int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM workflow_events WHERE task_id=$1::uuid`, task.ID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if events != 1 {
			t.Fatalf("expected one workflow event, got %d", events)
		}
	})

	t.Run("onboarding insert honors expected version", func(t *testing.T) {
		service := onboarding.NewService(onboarding.NewPostgresRepository(pool))
		_, err := service.Update(ctx, "integration-bank", principalNG, "reviewer-first-run", onboarding.UpdateInput{CurrentStep: 1, ExpectedVersion: 4})
		if !errors.Is(err, onboarding.ErrVersionConflict) {
			t.Fatalf("expected version conflict, got %v", err)
		}
		state, err := service.Update(ctx, "integration-bank", principalNG, "reviewer-first-run", onboarding.UpdateInput{CurrentStep: 1, ExpectedVersion: 0})
		if err != nil {
			t.Fatal(err)
		}
		if state.Version != 1 {
			t.Fatalf("expected version 1, got %d", state.Version)
		}
	})

	t.Run("signal and drift are committed atomically and readiness is honest", func(t *testing.T) {
		service := autonomy.NewService(autonomy.NewPostgresRepository(pool))
		signal := autonomy.Signal{TenantID: "integration-bank", Type: autonomy.SignalEvidenceExpired, SubjectType: "CLAIM", SubjectID: "claim-1", DedupeKey: "claim-1-v1-expired", Source: "scheduler"}
		_, inserted, err := service.Ingest(ctx, signal)
		if err != nil || !inserted {
			t.Fatalf("ingest: %v inserted=%v", err, inserted)
		}
		_, inserted, err = service.Ingest(ctx, signal)
		if err != nil || inserted {
			t.Fatalf("duplicate ingest: %v inserted=%v", err, inserted)
		}
		var signals, drifts int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM compliance_signals WHERE tenant_id=$1::uuid`, tenantID).Scan(&signals); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM drift_assessments WHERE tenant_id=$1::uuid`, tenantID).Scan(&drifts); err != nil {
			t.Fatal(err)
		}
		if signals != 1 || drifts != 1 {
			t.Fatalf("expected one signal and drift, got %d/%d", signals, drifts)
		}
		readiness, err := service.Readiness(ctx, "integration-bank")
		if err != nil {
			t.Fatal(err)
		}
		if readiness.BaselineKnown || readiness.Dimensions.Current != 0 {
			t.Fatalf("readiness fabricated a baseline: %#v", readiness)
		}
	})
}

func seedPostgres(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'integration-bank','Integration Bank'),($2::uuid,'other-bank','Other Bank')`, []any{tenantID, otherTenantID}},
		{`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$3::uuid,'bank-ng','Bank NG','NG'),($2::uuid,$3::uuid,'bank-gh','Bank GH','GH')`, []any{legalEntityNG, legalEntityGH, tenantID}},
		{`INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$3::uuid,'PERSON','cro-ng','NG CRO'),($2::uuid,$3::uuid,'PERSON','cro-gh','GH CRO')`, []any{principalNG, principalGH, tenantID}},
		{`INSERT INTO role_templates(id,tenant_id,code,name) VALUES($1::uuid,$2::uuid,'CRO','Chief Risk Officer')`, []any{roleTemplateID, tenantID}},
		{`INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id) VALUES($1::uuid,$5::uuid,$3::uuid,'CRO','NG CRO',$6::uuid),($2::uuid,$5::uuid,$4::uuid,'CRO','GH CRO',$7::uuid)`, []any{positionNG, positionGH, legalEntityNG, legalEntityGH, tenantID, principalNG, principalGH}},
		{`INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,priority) VALUES($1::uuid,$2::uuid,$4::uuid,100),($1::uuid,$3::uuid,$4::uuid,100)`, []any{tenantID, positionNG, positionGH, roleTemplateID}},
		{`INSERT INTO routing_policies(id,tenant_id,code,name,status,current_version) VALUES($1::uuid,$2::uuid,'default','Default routing','ACTIVE',1)`, []any{policyID, tenantID}},
		{`INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES($1::uuid,$2::uuid,'REVIEW','MATTER',$3::uuid,'ACTIVE','default:v1')`, []any{workflowID, tenantID, subjectID}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	definition, err := json.Marshal(map[string]any{"rules": []map[string]any{
		{"id": "ng-authorizer", "legal_entity_id": "bank-ng", "object_type": "MATTER", "object_id": "*", "responsibility": "AUTHORIZER", "min_materiality": 4, "priority": 100, "selector": map[string]any{"kind": "ROLE", "ref": "CRO"}},
		{"id": "missing-reviewer", "legal_entity_id": "bank-ng", "object_type": "MATTER", "object_id": "*", "responsibility": "REVIEWER", "min_materiality": 0, "priority": 50, "selector": map[string]any{"kind": "ROLE", "ref": "MISSING_ROLE"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO routing_policy_versions(policy_id,version,definition,checksum,approved_at) VALUES($1::uuid,1,$2::jsonb,'test',clock_timestamp())`, policyID, string(definition)); err != nil {
		t.Fatal(err)
	}
}
