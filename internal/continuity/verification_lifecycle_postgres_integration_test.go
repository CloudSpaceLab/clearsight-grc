//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresOutcomeFailureBeforeVerificationCanBeRechecked(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := WithTrustedSystemScope(context.Background())
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID = "86868686-8686-7686-8686-868686868681"
		ownerID  = "86868686-8686-7686-8686-868686868682"
		reviewer = "86868686-8686-7686-8686-868686868683"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'outcome-recheck-test','Outcome Recheck Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	for _, principal := range []string{ownerID, reviewer} {
		if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON',$1::text,'Outcome lifecycle actor')`, principal, tenantID); err != nil {
			t.Fatal(err)
		}
	}

	service := NewService(NewPostgresRepository(pool))
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "outcome-recheck-test", Type: MatterAuditFinding, Priority: 4,
		Title: "Recheck remediated access exceptions", Summary: "An independent reviewer must confirm that the corrected access list is complete.",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []MatterStatus{MatterInitialReview, MatterAssessment, MatterActionsInProgress} {
		matter, err = service.TransitionMatter(ctx, TransitionInput{TenantID: "outcome-recheck-test", ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: target, Rationale: "Progress the issue to corrective work."})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddAction(ctx, AddActionInput{
		TenantID: "outcome-recheck-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Correct the access exceptions", Description: "Remove every unsupported access entry.", OwnerPrincipalID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	for _, target := range []ActionStatus{ActionInProgress, ActionImplemented} {
		matter, err = service.TransitionAction(ctx, TransitionActionInput{TenantID: "outcome-recheck-test", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: ownerID, Rationale: "Corrective work progressed."})
		if err != nil {
			t.Fatal(err)
		}
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{
		TenantID: "outcome-recheck-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: actionID,
		ExpectedOutcome: "No unsupported access entries remain.", AuthorityPrincipalID: reviewer, FailureResponse: "ESCALATE",
	})
	if err != nil {
		t.Fatal(err)
	}
	contractID := matter.VerificationContracts[0].ID

	failed, err := service.RecordVerificationResult(ctx, RecordVerificationResultInput{
		TenantID: "outcome-recheck-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: contractID,
		Result: VerificationFailed, Observations: json.RawMessage(`{"unsupported":1}`), ReviewerPrincipalID: reviewer,
		Rationale: "One unsupported entry remains.", ObservedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed.Matter.Status != MatterDecisionRequired || len(failed.VerificationResults) != 1 {
		t.Fatalf("early failure did not execute escalation and preserve the result: %#v", failed)
	}
	if work, _ := CompileMatterWork(failed, now); len(work) != 1 || work[0].CommandName != "matter.outcome.record" {
		t.Fatalf("failed outcome disappeared from reviewer work: %#v", work)
	}

	service.now = func() time.Time { return now.Add(time.Second) }
	passed, err := service.RecordVerificationResult(ctx, RecordVerificationResultInput{
		TenantID: "outcome-recheck-test", MatterID: matter.Matter.ID, ExpectedVersion: failed.Matter.Version, ContractID: contractID,
		Result: VerificationPassed, Observations: json.RawMessage(`{"unsupported":0}`), ReviewerPrincipalID: reviewer,
		Rationale: "The corrected access list now has no unsupported entries.", ObservedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(passed.VerificationResults) != 2 || !passed.Closure.Ready {
		t.Fatalf("latest passing result did not preserve history and control closure: results=%#v closure=%#v", passed.VerificationResults, passed.Closure)
	}

	var resultRows, resultEvents, resultOutbox int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM verification_results WHERE matter_id=$1::uuid`, matter.Matter.ID).Scan(&resultRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE aggregate_id=$1::uuid AND event_type='VERIFICATION_RESULT_RECORDED'`, matter.Matter.ID).Scan(&resultEvents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='VERIFICATION_RESULT_RECORDED'`, matter.Matter.ID).Scan(&resultOutbox); err != nil {
		t.Fatal(err)
	}
	if resultRows != 2 || resultEvents != 2 || resultOutbox != 2 {
		t.Fatalf("result row/event/outbox counts = %d/%d/%d, want 2/2/2", resultRows, resultEvents, resultOutbox)
	}
}
