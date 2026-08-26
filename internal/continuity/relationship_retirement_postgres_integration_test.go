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

func TestPostgresRelationshipRetirementCommitsCurrentProjectionEventOutboxAndRefreshTogether(t *testing.T) {
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
		tenantID = "94666666-6666-7666-8666-666666666661"
		entityID = "94666666-6666-7666-8666-666666666662"
		ownerID  = "94666666-6666-7666-8666-666666666663"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'relationship-retirement-test','Relationship Retirement Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ENTITY-A','Entity A','NG')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Program owner')`, ownerID, tenantID); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewCurrentPostgresRepository(pool))
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "relationship-retirement-test", LegalEntityID: entityID, Code: "LINKS", Name: "Relationship lifecycle", Type: "ASSURANCE", OwningFunction: "Risk", OwnerPrincipalID: ownerID, AuthorityPrincipalID: ownerID, Scope: json.RawMessage(`{}`), EffectiveFrom: now, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, AddRequirementInput{TenantID: "relationship-retirement-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "REQ", Title: "Keep coverage current", Statement: "Coverage relationships must be current.", Status: RequirementApproved, EffectiveFrom: now, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlObjective(ctx, AddControlObjectiveInput{TenantID: "relationship-retirement-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "OBJ", Name: "Coverage", Outcome: "Coverage remains correct.", Status: ObjectiveActive, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddControlImplementation(ctx, AddControlImplementationInput{TenantID: "relationship-retirement-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ObjectiveID: program.ControlObjectives[0].ID, Name: "Coverage review", Description: "Review the mapping.", ImplementationType: "REVIEW", OwnerPrincipalID: ownerID, Scope: json.RawMessage(`{}`), EffectiveFrom: now, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.LinkRequirementControl(ctx, LinkRequirementControlInput{TenantID: "relationship-retirement-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, RequirementID: program.Requirements[0].ID, ImplementationID: program.ControlImplementations[0].ID, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	coverageLink := program.RequirementControlLinks[0]
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "relationship-retirement-test", LegalEntityID: entityID, Type: MatterControlGap, Priority: 3, Title: "Correct the relationship", Summary: "The issue link no longer applies.", Scope: json.RawMessage(`{}`), ProgramID: program.Program.ID, ActorID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	matterLink := matter.Links[0]

	program, err = service.RetireRequirementControlLink(ctx, RetireRequirementControlLinkInput{TenantID: "relationship-retirement-test", ProgramID: program.Program.ID, LinkID: coverageLink.ID, ExpectedVersion: program.Program.Version, ActorID: ownerID, Rationale: "The safeguard was mapped to the wrong requirement."})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.RetireMatterLink(ctx, RetireMatterLinkInput{TenantID: "relationship-retirement-test", MatterID: matter.Matter.ID, LinkID: matterLink.ID, ExpectedVersion: matter.Matter.Version, ActorID: ownerID, Rationale: "This issue no longer affects the Program."})
	if err != nil {
		t.Fatal(err)
	}

	var coverageRows, matterRows, events, outbox, refreshJobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM requirement_control_links WHERE tenant_id=$1::uuid AND id=$2::uuid AND retired_at IS NOT NULL AND retired_by=$3::uuid AND retirement_reason<>''`, tenantID, coverageLink.ID, ownerID).Scan(&coverageRows); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM matter_links WHERE tenant_id=$1::uuid AND id=$2::uuid AND retired_at IS NOT NULL AND retired_by=$3::uuid AND retirement_reason<>''`, tenantID, matterLink.ID, ownerID).Scan(&matterRows); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND event_type IN ($2,$3)`, tenantID, EventRequirementControlLinkRetired, EventMatterLinkRetired).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND event_type IN ($2,$3)`, tenantID, EventRequirementControlLinkRetired, EventMatterLinkRetired).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM continuity_projection_jobs WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND reason IN ($3,$4)`, tenantID, program.Program.ID, EventRequirementControlLinkRetired, EventMatterLinkRetired).Scan(&refreshJobs); err != nil {
		t.Fatal(err)
	}
	if coverageRows != 1 || matterRows != 1 || events != 2 || outbox != 2 || refreshJobs != 1 || len(program.RequirementControlLinks) != 0 || len(matter.Links) != 0 {
		t.Fatalf("retirement atomic state coverage=%d matter=%d events=%d outbox=%d refresh_jobs=%d current_links=%d/%d", coverageRows, matterRows, events, outbox, refreshJobs, len(program.RequirementControlLinks), len(matter.Links))
	}
}
