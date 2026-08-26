//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresVerificationContractSupersessionAndRetirementAreAtomic(t *testing.T) {
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
		tenantID = "41414141-4141-7141-8141-414141414141"
		actorID  = "41414141-4141-7141-8141-414141414142"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'verification-contract-lifecycle','Verification Contract Lifecycle')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','verification-contract-reviewer','Outcome reviewer')`, actorID, tenantID); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewPostgresRepository(pool))
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "verification-contract-lifecycle", Type: MatterControlGap, Priority: 4,
		Title: "Correct an outcome check", Summary: "The reviewer must correct the population before closure.",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{
		TenantID: "verification-contract-lifecycle", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "Every administrator is approved.", Scope: json.RawMessage(`{"population":"administrators"}`),
		AuthorityPrincipalID: actorID, FailureResponse: "BLOCK_CLOSE", ActorID: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	prior := matter.VerificationContracts[0]
	now = now.Add(time.Minute)
	matter, err = service.SupersedeVerificationContract(ctx, SupersedeVerificationContractInput{
		TenantID: "verification-contract-lifecycle", MatterID: matter.Matter.ID, ContractID: prior.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "Every privileged account is approved.", Scope: json.RawMessage(`{"population":"privileged accounts"}`),
		AuthorityPrincipalID: actorID, FailureResponse: "BLOCK_CLOSE", ActorID: actorID, Rationale: "Use the approved privileged-account population.",
	})
	if err != nil {
		t.Fatal(err)
	}
	replacement := matter.VerificationContracts[1]

	var retiredRows, activeRows, supersedeEvents, supersedeOutbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='RETIRED'),count(*) FILTER (WHERE status='ACTIVE') FROM verification_contracts WHERE tenant_id=$1::uuid AND matter_id=$2::uuid`, tenantID, matter.Matter.ID).Scan(&retiredRows, &activeRows); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, EventVerificationContractSuperseded).Scan(&supersedeEvents); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, EventVerificationContractSuperseded).Scan(&supersedeOutbox); err != nil {
		t.Fatal(err)
	}
	if retiredRows != 1 || activeRows != 1 || supersedeEvents != 1 || supersedeOutbox != 1 {
		t.Fatalf("supersession rows/events/outbox = %d/%d/%d/%d, want 1/1/1/1", retiredRows, activeRows, supersedeEvents, supersedeOutbox)
	}

	_, err = service.RetireVerificationContract(ctx, RetireVerificationContractInput{
		TenantID: "verification-contract-lifecycle", MatterID: matter.Matter.ID, ContractID: replacement.ID,
		ExpectedVersion: matter.Matter.Version - 1, ActorID: actorID, Rationale: "Stale retirement must roll back.",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale retirement error = %v, want version conflict", err)
	}
	var retireEvents, retireOutbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, EventVerificationContractRetired).Scan(&retireEvents); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, EventVerificationContractRetired).Scan(&retireOutbox); err != nil {
		t.Fatal(err)
	}
	if retireEvents != 0 || retireOutbox != 0 {
		t.Fatalf("failed retirement persisted event/outbox = %d/%d", retireEvents, retireOutbox)
	}
}
