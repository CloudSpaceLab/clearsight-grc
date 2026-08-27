//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequirementSupersessionPersistsHistoryAndOutboxAtomically(t *testing.T) {
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

	const (
		tenantID = "89999999-9999-7999-8999-999999999981"
		ownerID  = "89999999-9999-7999-8999-999999999982"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'program-edits-test','Program Edits Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	entityID := seedPostgresTestLegalEntity(t, ctx, pool, tenantID, "ENTITY-A")
	ctx = WithTrustedSystemEntityScope(ctx, "program-edits-test", entityID)
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Privacy owner')`, ownerID, tenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	service := NewService(NewPostgresRepository(pool))
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "program-edits-test", LegalEntityID: entityID, Code: "NDPA", Name: "Data protection", Type: "PRIVACY",
		OwningFunction: "Data Protection Office", OwnerPrincipalID: ownerID,
		EffectiveFrom: now.AddDate(-1, 0, 0), ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{
		TenantID: "program-edits-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
		Code: "CAR-01", Title: "File the annual return", Statement: "The bank must file its annual compliance return.",
		SourceAnchor: "GAID 2025, section 7", EffectiveFrom: now.AddDate(-1, 0, 0), ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	priorID := program.Requirements[0].ID
	replacementFrom := now.AddDate(0, 1, 0)
	program, err = service.SupersedeRequirement(ctx, SupersedeRequirementInput{
		TenantID: "program-edits-test", ProgramID: program.Program.ID, RequirementID: priorID,
		ExpectedVersion: program.Program.Version, Code: "CAR-01", Title: "File the annual return",
		Statement:    "The bank must file its annual compliance return through a licensed DPCO.",
		SourceAnchor: "GAID 2025, section 7.2", EffectiveFrom: replacementFrom,
		ActorID: ownerID, Rationale: "The regulator changed the filing channel.",
	})
	if err != nil {
		t.Fatal(err)
	}

	var requirementCount, supersededCount, approvedCount, eventCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE status='SUPERSEDED'),count(*) FILTER (WHERE status='APPROVED') FROM program_requirements WHERE tenant_id=$1::uuid AND program_id=$2::uuid`, tenantID, program.Program.ID).Scan(&requirementCount, &supersededCount, &approvedCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, program.Program.ID, EventRequirementSuperseded).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, program.Program.ID, EventRequirementSuperseded).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if requirementCount != 2 || supersededCount != 1 || approvedCount != 1 || eventCount != 1 || outboxCount != 1 {
		t.Fatalf("supersession was not atomic: requirements=%d superseded=%d approved=%d events=%d outbox=%d", requirementCount, supersededCount, approvedCount, eventCount, outboxCount)
	}
}
