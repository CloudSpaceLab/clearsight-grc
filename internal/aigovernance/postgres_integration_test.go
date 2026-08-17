//go:build postgres && postgresintegration

package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	aiGovernanceIntegrationTenantID = "7f340000-0000-7340-8340-000000000001"
	aiGovernanceIntegrationMakerID  = "7f340000-0000-7340-8340-000000000002"
	aiGovernanceIntegrationCheckID  = "7f340000-0000-7340-8340-000000000003"
	aiGovernanceIntegrationOwnerID  = "7f340000-0000-7340-8340-000000000004"
)

func TestPostgresGovernanceLifecycleReceiptAndGrant(t *testing.T) {
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

	cleanupAIGovernanceFixture(t, pool)
	t.Cleanup(func() { cleanupAIGovernanceFixture(t, pool) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'ai-governance-integration','AI governance integration')`, aiGovernanceIntegrationTenantID); err != nil {
		t.Fatal(err)
	}
	for _, principal := range []struct{ id, name string }{
		{aiGovernanceIntegrationMakerID, "Policy maker"},
		{aiGovernanceIntegrationCheckID, "Policy checker"},
		{aiGovernanceIntegrationOwnerID, "Workload owner"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON',$3,'ACTIVE',clock_timestamp())`, principal.id, aiGovernanceIntegrationTenantID, principal.name); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewPostgresRepository(pool)
	matters := continuity.NewService(continuity.NewPostgresRepository(pool))
	service := NewService(repo, nil, nil, matters)
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	policy, err := service.CreatePolicy(ctx, CreatePolicyInput{
		TenantID: "ai-governance-integration", Code: "AI-PG", Name: "AI PostgreSQL policy", ActionClass: "MODEL_REQUEST",
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, RolloutMode: aigateway.RolloutShadow,
		MakerID: aiGovernanceIntegrationMakerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.TransitionPolicy(ctx, "submit", TransitionInput{TenantID: policy.TenantID, ID: policy.ID, ActorID: aiGovernanceIntegrationMakerID, ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.TransitionPolicy(ctx, "approve", TransitionInput{TenantID: policy.TenantID, ID: policy.ID, ActorID: aiGovernanceIntegrationCheckID, ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.TransitionPolicy(ctx, "activate", TransitionInput{TenantID: policy.TenantID, ID: policy.ID, ActorID: aiGovernanceIntegrationCheckID, ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}

	key := sha256.Sum256([]byte("postgres-integration-api-key"))
	workload, err := service.CreateWorkload(ctx, CreateWorkloadInput{
		TenantID: policy.TenantID, WorkloadID: "pg-agent", Code: "PG-AGENT", Name: "PostgreSQL agent", Purpose: "integration",
		Environment: "test", OwnerPrincipalID: aiGovernanceIntegrationOwnerID, AllowedModels: []string{"general"},
		RequestsPerMinute: 10, TokensPerMinute: 1000, CostMicroUSDPerMinute: 50000, MaxConcurrent: 2,
		PolicyID: policy.ID, PolicyVersion: policy.Version, KeySHA256: hex.EncodeToString(key[:]), MakerID: aiGovernanceIntegrationMakerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		action string
		actor  string
	}{{"submit", aiGovernanceIntegrationMakerID}, {"approve", aiGovernanceIntegrationCheckID}, {"activate", aiGovernanceIntegrationCheckID}} {
		workload, err = service.TransitionWorkload(ctx, step.action, TransitionInput{TenantID: workload.TenantID, ID: workload.ID, ActorID: step.actor, ExpectedVersion: workload.RecordVersion})
		if err != nil {
			t.Fatalf("workload %s: %v", step.action, err)
		}
	}
	if workload.State != "ACTIVE" {
		t.Fatalf("workload state=%s, want ACTIVE", workload.State)
	}

	receipt := DecisionReceipt{
		ReceiptID: "pg-receipt-1", TenantID: policy.TenantID, RequestID: "pg-request-1", WorkloadID: workload.WorkloadID,
		PolicyID: policy.ID, PolicyCode: policy.Code, PolicyVersion: policy.Version, Decision: aigateway.DecisionRequireApproval,
		ReasonCodes: []string{"HIGH_IMPACT_TOOL"}, Outcome: "BLOCKED", ObservedAt: now,
	}
	if inserted, err := service.IngestReceipt(ctx, receipt); err != nil || !inserted {
		t.Fatalf("first receipt inserted=%v err=%v", inserted, err)
	}
	if inserted, err := service.IngestReceipt(ctx, receipt); err != nil || inserted {
		t.Fatalf("duplicate receipt inserted=%v err=%v", inserted, err)
	}
	var receiptCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ai_gateway_decision_receipts WHERE tenant_id=$1::uuid AND receipt_id='pg-receipt-1'`, aiGovernanceIntegrationTenantID).Scan(&receiptCount); err != nil || receiptCount != 1 {
		t.Fatalf("receipt persistence count=%d err=%v", receiptCount, err)
	}

	matter, err := matters.CreateMatter(ctx, continuity.CreateMatterInput{
		TenantID: policy.TenantID, Type: continuity.MatterAuthorityRequest, Priority: 5,
		Title: "Approve exact AI action", Summary: "One action only.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	action := json.RawMessage(`{"tool":"freeze_account","arguments":{"account":"PG-100"}}`)
	actionHash := sha256.Sum256(compactJSON(action))
	selected := hex.EncodeToString(actionHash[:])
	for _, step := range []struct {
		status continuity.DecisionStatus
		actor  string
	}{{continuity.DecisionProposed, aiGovernanceIntegrationMakerID}, {continuity.DecisionInReview, aiGovernanceIntegrationCheckID}, {continuity.DecisionApproved, aiGovernanceIntegrationCheckID}} {
		input := continuity.AddDecisionInput{
			TenantID: policy.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
			Type: "AI_EXECUTION_GRANT", Status: step.status, Rationale: "Exact governed action.", AuthorityPrincipalID: step.actor,
		}
		if step.status == continuity.DecisionApproved {
			input.SelectedOption = selected
		}
		matter, err = matters.RecordDecisionLifecycle(ctx, input)
		if err != nil {
			t.Fatalf("decision %s: %v", step.status, err)
		}
	}
	approved := continuity.CurrentDecisionForType(matter.Decisions, "AI_EXECUTION_GRANT")
	if approved == nil {
		t.Fatal("approved grant decision missing")
	}
	grant, err := service.CreateExecutionGrant(ctx, CreateGrantInput{
		TenantID: policy.TenantID, WorkloadID: workload.WorkloadID, MatterID: matter.Matter.ID,
		DecisionID: approved.ID, Action: action, ActorID: aiGovernanceIntegrationCheckID, TTLMinutes: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConsumeExecutionGrant(ctx, policy.TenantID, workload.WorkloadID, action, grant.Token); err != nil {
		t.Fatalf("first grant use: %v", err)
	}
	if _, err := service.ConsumeExecutionGrant(ctx, policy.TenantID, workload.WorkloadID, action, grant.Token); err != ErrGrantInvalid {
		t.Fatalf("grant replay = %v, want ErrGrantInvalid", err)
	}
}

func cleanupAIGovernanceFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, statement := range []string{
		`DELETE FROM ai_execution_grants WHERE tenant_id=$1::uuid`,
		`DELETE FROM ai_gateway_decision_receipts WHERE tenant_id=$1::uuid`,
		`DELETE FROM ai_workloads WHERE tenant_id=$1::uuid`,
		`DELETE FROM automation_policies WHERE tenant_id=$1::uuid`,
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM continuity_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM matter_decisions WHERE tenant_id=$1::uuid`,
		`DELETE FROM matters WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		if _, err := pool.Exec(ctx, statement, aiGovernanceIntegrationTenantID); err != nil {
			// The first cleanup runs before fixture creation and should still succeed;
			// fail loudly on real dependency/schema regressions.
			t.Fatalf("fixture cleanup %q: %v", statement, err)
		}
	}
}
