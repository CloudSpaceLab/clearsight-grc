//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	phase2TenantID   = "33333333-3333-7333-8333-333333333331"
	phase2EntityID   = "33333333-3333-7333-8333-333333333332"
	phase2MakerID    = "33333333-3333-7333-8333-333333333333"
	phase2CheckerID  = "33333333-3333-7333-8333-333333333334"
	phase2FromID     = "33333333-3333-7333-8333-333333333335"
	phase2ToID       = "33333333-3333-7333-8333-333333333336"
	phase2ThirdID    = "33333333-3333-7333-8333-333333333337"
	phase2WorkflowID = "33333333-3333-7333-8333-333333333338"
	phase2SubjectID  = "33333333-3333-7333-8333-333333333339"
	phase2RoleID     = "33333333-3333-7333-8333-333333333340"
	phase2PositionID = "33333333-3333-7333-8333-333333333341"
)

type integrationPublisher struct{ count int }

func (p *integrationPublisher) Publish(context.Context, workflowruntime.OutboxEvent) error {
	p.count++
	return nil
}

func TestGovernanceRuntimePhase(t *testing.T) {
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
	seedGovernanceRuntime(t, pool)

	t.Run("policy maker checker writes decision and outbox", func(t *testing.T) {
		service := governance.NewService(governance.NewPostgresRepository(pool))
		definition := json.RawMessage(`{"rules":[{"id":"r1","legal_entity_id":"33333333-3333-7333-8333-333333333332","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"INTERNAL_AUDIT"}}]}`)
		policy, err := service.CreatePolicy(ctx, governance.CreatePolicyInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, Code: "risk", Name: "Risk routing", MakerID: phase2MakerID, Definition: definition})
		if err != nil {
			t.Fatal(err)
		}
		policy, err = service.SubmitPolicy(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: policy.ID, ActorID: phase2MakerID, ExpectedVersion: policy.Version})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.ApprovePolicy(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: policy.ID, ActorID: phase2MakerID, ExpectedVersion: policy.Version}); !errors.Is(err, governance.ErrMakerChecker) {
			t.Fatalf("expected maker-checker, got %v", err)
		}
		policy, err = service.ApprovePolicy(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: policy.ID, ActorID: phase2CheckerID, ExpectedVersion: policy.Version, Rationale: "Independent approval"})
		if err != nil {
			t.Fatal(err)
		}
		if policy.Status != governance.PolicyActive {
			t.Fatalf("expected active policy, got %s", policy.Status)
		}
		var decisions, events int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM governance_decisions WHERE object_id=$1::uuid`, policy.ID).Scan(&decisions); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid`, policy.ID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if decisions != 2 || events != 2 {
			t.Fatalf("expected two decisions/events, got %d/%d", decisions, events)
		}
	})

	t.Run("delegation cycles and segregation conflicts are blocked", func(t *testing.T) {
		service := governance.NewService(governance.NewPostgresRepository(pool))
		now := time.Now().UTC()
		first, err := service.CreateDelegation(ctx, governance.CreateDelegationInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, FromPrincipalID: phase2FromID, ToPrincipalID: phase2ToID, Responsibility: "REVIEWER", StartsAt: now.Add(time.Hour), EndsAt: now.Add(48 * time.Hour), MakerID: phase2MakerID})
		if err != nil {
			t.Fatal(err)
		}
		first, err = service.SubmitDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: first.ID, ActorID: phase2MakerID, ExpectedVersion: first.Version})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = service.ApproveDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: first.ID, ActorID: phase2CheckerID, ExpectedVersion: first.Version}); err != nil {
			t.Fatal(err)
		}
		second, _ := service.CreateDelegation(ctx, governance.CreateDelegationInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, FromPrincipalID: phase2ToID, ToPrincipalID: phase2ThirdID, Responsibility: "REVIEWER", StartsAt: now.Add(2 * time.Hour), EndsAt: now.Add(24 * time.Hour), MakerID: phase2MakerID})
		second, _ = service.SubmitDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: second.ID, ActorID: phase2MakerID, ExpectedVersion: second.Version})
		if _, err = service.ApproveDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: second.ID, ActorID: phase2CheckerID, ExpectedVersion: second.Version}); err != nil {
			t.Fatal(err)
		}
		cycle, _ := service.CreateDelegation(ctx, governance.CreateDelegationInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, FromPrincipalID: phase2ThirdID, ToPrincipalID: phase2FromID, Responsibility: "REVIEWER", StartsAt: now.Add(3 * time.Hour), EndsAt: now.Add(12 * time.Hour), MakerID: phase2MakerID})
		cycle, _ = service.SubmitDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: cycle.ID, ActorID: phase2MakerID, ExpectedVersion: cycle.Version})
		if _, err = service.ApproveDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: cycle.ID, ActorID: phase2CheckerID, ExpectedVersion: cycle.Version}); !errors.Is(err, governance.ErrConflict) {
			t.Fatalf("expected cycle conflict, got %v", err)
		}
		conflict, _ := service.CreateDelegation(ctx, governance.CreateDelegationInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, FromPrincipalID: phase2FromID, ToPrincipalID: phase2ThirdID, Responsibility: "AUTHORIZER", StartsAt: now.Add(time.Hour), EndsAt: now.Add(12 * time.Hour), MakerID: phase2MakerID})
		conflict, _ = service.SubmitDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: conflict.ID, ActorID: phase2MakerID, ExpectedVersion: conflict.Version})
		if _, err = service.ApproveDelegation(ctx, governance.TransitionInput{TenantID: "phase2-bank", LegalEntityID: phase2EntityID, ID: conflict.ID, ActorID: phase2CheckerID, ExpectedVersion: conflict.Version}); !errors.Is(err, governance.ErrConflict) {
			t.Fatalf("expected segregation conflict, got %v", err)
		}
	})

	t.Run("timer firing and outbox publication are durable and idempotent", func(t *testing.T) {
		repository := workflowruntime.NewPostgresRepository(pool)
		lifecycle := governance.NewPostgresRepository(pool)
		publisher := &integrationPublisher{}
		service := workflowruntime.NewService(repository, lifecycle, publisher, "integration-worker")
		now := time.Now().UTC()
		timer, err := service.Schedule(ctx, workflowruntime.Timer{TenantID: "phase2-bank", WorkflowID: phase2WorkflowID, Type: "ESCALATION", DueAt: now.Add(-time.Second), DedupeKey: "phase2:escalation", Payload: json.RawMessage(`{"step":"review"}`)})
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		var state string
		if err := pool.QueryRow(ctx, `SELECT state FROM workflow_timers WHERE id=$1::uuid`, timer.ID).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != "FIRED" || publisher.count == 0 {
			t.Fatalf("expected fired/published, got state=%s count=%d", state, publisher.count)
		}
		first, err := repository.RecordInbox(ctx, "phase2-bank", "committee-projection", "event-1", now)
		if err != nil {
			t.Fatal(err)
		}
		second, err := repository.RecordInbox(ctx, "phase2-bank", "committee-projection", "event-1", now)
		if err != nil {
			t.Fatal(err)
		}
		if !first || second {
			t.Fatal("expected inbox deduplication")
		}
	})
}

func seedGovernanceRuntime(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	statements := []string{
		`INSERT INTO tenants(id,slug,name) VALUES('` + phase2TenantID + `','phase2-bank','Phase 2 Bank')`,
		`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES('` + phase2EntityID + `','` + phase2TenantID + `','phase2-ng','Phase 2 NG','NG')`,
		`INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES
		('` + phase2MakerID + `','` + phase2TenantID + `','PERSON','maker','Maker'),
		('` + phase2CheckerID + `','` + phase2TenantID + `','PERSON','checker','Checker'),
		('` + phase2FromID + `','` + phase2TenantID + `','PERSON','from','Delegator'),
		('` + phase2ToID + `','` + phase2TenantID + `','PERSON','to','Delegate'),
		('` + phase2ThirdID + `','` + phase2TenantID + `','PERSON','third','Conflicted delegate')`,
		`INSERT INTO role_templates(id,tenant_id,code,name) VALUES('` + phase2RoleID + `','` + phase2TenantID + `','INTERNAL_AUDIT','Internal Audit')`,
		`INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id) VALUES
		('` + phase2PositionID + `','` + phase2TenantID + `','` + phase2EntityID + `','AUDIT','Audit','` + phase2ThirdID + `'),
		('33333333-3333-7333-8333-333333333342','` + phase2TenantID + `','` + phase2EntityID + `','DELEGATOR','Delegator','` + phase2FromID + `'),
		('33333333-3333-7333-8333-333333333343','` + phase2TenantID + `','` + phase2EntityID + `','DELEGATE','Delegate','` + phase2ToID + `')`,
		`INSERT INTO position_role_bindings(tenant_id,position_id,role_template_id,priority) VALUES('` + phase2TenantID + `','` + phase2PositionID + `','` + phase2RoleID + `',100)`,
		`INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version) VALUES
		('` + phase2TenantID + `','` + phase2EntityID + `','` + phase2FromID + `','REVIEWER','LEGAL_ENTITY','` + phase2EntityID + `',100,clock_timestamp()-interval '1 day','fixture:v1'),
		('` + phase2TenantID + `','` + phase2EntityID + `','` + phase2ToID + `','REVIEWER','LEGAL_ENTITY','` + phase2EntityID + `',100,clock_timestamp()-interval '1 day','fixture:v1'),
		('` + phase2TenantID + `','` + phase2EntityID + `','` + phase2ThirdID + `','REVIEWER','LEGAL_ENTITY','` + phase2EntityID + `',100,clock_timestamp()-interval '1 day','fixture:v1'),
		('` + phase2TenantID + `','` + phase2EntityID + `','` + phase2FromID + `','AUTHORIZER','LEGAL_ENTITY','` + phase2EntityID + `',100,clock_timestamp()-interval '1 day','fixture:v1')`,
		`INSERT INTO segregation_rules(tenant_id,code,responsibility,prohibited_role_code) VALUES('` + phase2TenantID + `','no-audit-authorizer','AUTHORIZER','INTERNAL_AUDIT')`,
		`INSERT INTO workflow_instances(id,tenant_id,kind,subject_type,subject_id,state,policy_version) VALUES('` + phase2WorkflowID + `','` + phase2TenantID + `','REVIEW','MATTER','` + phase2SubjectID + `','ACTIVE','phase2:v1')`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
}
