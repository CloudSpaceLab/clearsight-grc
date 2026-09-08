//go:build postgres && postgresintegration

package formpolicy

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestPostgresContextualGuardrailSharesTransactionAndScope(t *testing.T) {
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
	policy.AutomationPolicyID = policy.ID
	policy.AutomationPolicyVersion = policy.Version
	policy.Checksum = policyChecksum(policy)
	created, err := repo.CreatePolicy(ctx, policy)
	if err != nil {
		t.Fatal(err)
	}
	var status, checksum string
	var version int64
	if err := pool.QueryRow(ctx, `SELECT status,checksum,record_version FROM automation_policies WHERE id=$1::uuid`, created.AutomationPolicyID).Scan(&status, &checksum, &version); err != nil || status != "DRAFT" || checksum != created.Checksum || version != 1 {
		t.Fatalf("guardrail status=%s version=%d err=%v", status, version, err)
	}
	choices, err := repo.ListAutomationChoices(ctx, policyTenantID, policyEntityID, policyFormID, 1, 100)
	if err != nil || len(choices) != 1 || choices[0].ID != created.AutomationPolicyID {
		t.Fatalf("choices=%#v err=%v", choices, err)
	}
	if choices, err := repo.ListAutomationChoices(ctx, policyTenantID, policyOtherEntity, policyFormID, 1, 100); err != nil || len(choices) != 0 {
		t.Fatalf("cross-scope choices=%#v err=%v", choices, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE automation_policies SET record_version=99 WHERE id=$1::uuid`, created.AutomationPolicyID); err != nil {
		t.Fatal(err)
	}
	changed := created
	changed.RecordVersion++
	changed.Status = PolicyPendingApproval
	changed.LastActorID = policyMakerID
	if _, err := repo.UpdatePolicy(ctx, changed, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("canonical conflict=%v", err)
	}
	stored, err := repo.GetPolicy(ctx, policyTenantID, policyEntityID, created.ID)
	if err != nil || stored.RecordVersion != 1 || stored.Status != PolicyDraft {
		t.Fatalf("typed record committed despite canonical conflict: %#v err=%v", stored, err)
	}
}

func TestPostgresAutomaticAndAssessedReceiptsReuseSameIssue(t *testing.T) {
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
	policy.Status = PolicyActive
	policy.Rollout = RolloutEnforce
	policy.Checksum = policyChecksum(policy)
	if _, err := repo.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	original := postgresExecutionCommand(now, policy)
	first, err := repo.ApplyExecution(ctx, original)
	if err != nil {
		t.Fatal(err)
	}
	seedDistinctSubjectPolicyResponse(t, ctx, pool, now)
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_id=$1::uuid WHERE id='9f650000-0000-7650-8650-000000000052'::uuid; UPDATE capture_requests SET subject_id=$1::uuid WHERE id='9f650000-0000-7650-8650-000000000053'::uuid`, pgx.QueryExecModeSimpleProtocol, original.Response.SubjectID); err != nil {
		t.Fatal(err)
	}
	later := postgresExecutionCommand(now.Add(time.Minute), policy)
	later.Response.ID = "9f650000-0000-7650-8650-000000000056"
	later.Receipt.ResponseRevisionID = later.Response.ID
	later.Receipt.ID = "9f650000-0000-7650-8650-000000000057"
	later.EventID = "later-auto"
	if got, err := repo.ApplyExecution(ctx, later); err != nil || got.MatterID != first.MatterID {
		t.Fatalf("later=%#v err=%v", got, err)
	}
	bank := policy
	bank.ID = "9f650000-0000-7650-8650-000000000051"
	bank.Code = "bank-assessed-concern"
	bank.Eligibility.ResultBasis = ResultBankAssessed
	bank.Checksum = policyChecksum(bank)
	if _, err := repo.CreatePolicy(ctx, bank); err != nil {
		t.Fatal(err)
	}
	command := postgresExecutionCommand(now, bank)
	command.EventID = "bank-event-1"
	command.Receipt.ID = "9f650000-0000-7650-8650-000000000052"
	command.Receipt.ResultBasis = ResultBankAssessed
	command.Receipt.AssessmentVersion = 1
	if _, err := repo.ApplyExecution(ctx, command); !errors.Is(err, evidence.ErrAssessmentConflict) {
		t.Fatalf("missing assessment accepted: %v", err)
	}
	seedPolicyAssessment(t, ctx, pool, original.Response.ID, 1, now)
	second, err := repo.ApplyExecution(ctx, command)
	if err != nil || second.CreatedMatter || second.MatterID != first.MatterID {
		t.Fatalf("bank result=%#v err=%v", second, err)
	}
	command.EventID = "bank-event-2"
	command.Receipt.ID = "9f650000-0000-7650-8650-000000000053"
	command.Receipt.AssessmentVersion = 2
	seedPolicyAssessment(t, ctx, pool, original.Response.ID, 2, now.Add(time.Minute))
	stale := command
	stale.Receipt.AssessmentVersion = 1
	stale.Receipt.ID = "9f650000-0000-7650-8650-000000000058"
	stale.Receipt.PolicyID = policy.ID
	stale.Policy = policy
	stale.Policy.Eligibility.ResultBasis = ResultBankAssessed
	if _, err := repo.ApplyExecution(ctx, stale); !errors.Is(err, evidence.ErrAssessmentConflict) {
		t.Fatalf("superseded assessment accepted: %v", err)
	}
	corrected, err := repo.ApplyExecution(ctx, command)
	if err != nil || corrected.CreatedMatter || corrected.MatterID != first.MatterID {
		t.Fatalf("correction=%#v err=%v", corrected, err)
	}
	// A correction holding the response lock must complete before a competing
	// execution checks the latest assessment; it cannot commit the old result.
	seedPolicyAssessment(t, ctx, pool, original.Response.ID, 3, now.Add(2*time.Minute))
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM capture_response_revisions WHERE id=$1::uuid FOR UPDATE`, original.Response.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO capture_response_assessments(tenant_id,legal_entity_id,response_revision_id,version,state,required_count,reviewed_required_count,reviewed_count,score_result,decisions,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,4,'ASSESSED',1,1,1,'{"state":"FINAL","final":true,"coverage":1}','{}',$4)`, policyTenantID, policyEntityID, original.Response.ID, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	pending := command
	pending.Receipt.ID = "9f650000-0000-7650-8650-000000000059"
	pending.Receipt.AssessmentVersion = 3
	pending.EventID = "competing-assessment"
	done := make(chan error, 1)
	go func() { _, err := repo.ApplyExecution(ctx, pending); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("execution did not wait for assessment writer: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, evidence.ErrAssessmentConflict) {
			t.Fatalf("concurrent stale execution=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("execution did not finish after correction")
	}
	history, err := repo.ListExecutionHistory(ctx, policyTenantID, policyEntityID, bank.ID, 50)
	if err != nil || len(history) != 2 || history[0].ResultBasis != ResultBankAssessed {
		t.Fatalf("history=%#v err=%v", history, err)
	}
}

func seedPolicyAssessment(t *testing.T, ctx context.Context, pool *pgxpool.Pool, response string, version int64, at time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO capture_response_assessments(tenant_id,legal_entity_id,response_revision_id,version,state,required_count,reviewed_required_count,reviewed_count,score_result,decisions,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,'ASSESSED',1,1,1,'{"state":"FINAL","final":true,"coverage":1}','{}',$5)`, policyTenantID, policyEntityID, response, version, at)
	if err != nil {
		t.Fatal(err)
	}
}
