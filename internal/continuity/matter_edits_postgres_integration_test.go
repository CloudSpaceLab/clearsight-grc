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

func TestMatterEditsPersistEventAndOutbox(t *testing.T) {
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
		tenantID   = "89999999-9999-7999-8999-999999999991"
		ownerID    = "89999999-9999-7999-8999-999999999992"
		performer1 = "89999999-9999-7999-8999-999999999993"
		performer2 = "89999999-9999-7999-8999-999999999994"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'matter-edits-test','Matter Edits Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	entityID := seedPostgresTestLegalEntity(t, ctx, pool, tenantID, "ENTITY-A")
	ctx = WithTrustedSystemEntityScope(ctx, "matter-edits-test", entityID)
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES
		($1::uuid,$4::uuid,'PERSON','Privacy owner'),
		($2::uuid,$4::uuid,'PERSON','Current performer'),
		($3::uuid,$4::uuid,'PERSON','New performer')`, ownerID, performer1, performer2, tenantID); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewPostgresRepository(pool))
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "matter-edits-test", LegalEntityID: entityID, Type: MatterRegulatoryChange, Priority: 4,
		Title: "Annual return", Summary: "Update the filing process.",
		Scope: json.RawMessage(`{"area":"Privacy"}`), KnownFacts: json.RawMessage(`{"filing_channel":"licensed DPCO"}`),
		MissingFacts: json.RawMessage(`["final DPCO checklist"]`), Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: ownerID, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddAction(ctx, AddActionInput{
		TenantID: "matter-edits-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section.", OwnerPrincipalID: performer1, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	actionID := matter.Actions[0].ID
	matter, err = service.UpdateMatterDetails(ctx, UpdateMatterDetailsInput{
		TenantID: "matter-edits-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Annual return filing", Summary: matter.Matter.Summary, Priority: matter.Matter.Priority,
		Scope: matter.Matter.Scope, DueAt: matter.Matter.DueAt, ActorID: ownerID, Rationale: "Use the approved working title.",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.ChangeMatterContext(ctx, ChangeMatterContextInput{
		TenantID: "matter-edits-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Kind: MatterContextResolveMissing, Key: "final_dpco_checklist", Label: "final DPCO checklist",
		Value: json.RawMessage(`"Checklist v3"`), EvidenceReferences: json.RawMessage(`["artifact-checklist-v3"]`),
		ActorID: ownerID, Rationale: "DPCO approved version 3.",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.UpdateAction(ctx, UpdateActionInput{
		TenantID: "matter-edits-test", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version,
		Title: "Update checklist", Description: "Map every section to its current source.",
		ActorID: ownerID, Rationale: "Make the evidence requirement explicit.",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AssignAction(ctx, AssignActionInput{
		TenantID: "matter-edits-test", MatterID: matter.Matter.ID, ActionID: actionID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: performer2, ActorID: ownerID, Rationale: "Assign the current process owner.",
	})
	if err != nil {
		t.Fatal(err)
	}

	var title, checklist, actionDescription, actionOwner string
	var actionVersion int64
	if err := pool.QueryRow(ctx, `SELECT title,known_facts->>'final_dpco_checklist' FROM matters WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, matter.Matter.ID).Scan(&title, &checklist); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT description,owner_principal_id::text,version FROM matter_actions WHERE tenant_id=$1::uuid AND matter_id=$2::uuid AND id=$3::uuid`, tenantID, matter.Matter.ID, actionID).Scan(&actionDescription, &actionOwner, &actionVersion); err != nil {
		t.Fatal(err)
	}
	if title != "Annual return filing" || checklist != "Checklist v3" || actionDescription != "Map every section to its current source." || actionOwner != performer2 || actionVersion != 3 {
		t.Fatalf("current rows not projected: title=%q checklist=%q action=%q owner=%q version=%d", title, checklist, actionDescription, actionOwner, actionVersion)
	}

	for _, eventType := range []string{EventMatterDetailsUpdated, EventMatterContextChanged, EventActionUpdated, EventActionAssigned} {
		var continuityCount, outboxCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM continuity_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, eventType).Scan(&continuityCount); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, matter.Matter.ID, eventType).Scan(&outboxCount); err != nil {
			t.Fatal(err)
		}
		if continuityCount != 1 || outboxCount != 1 {
			t.Fatalf("%s not atomic: %d/%d", eventType, continuityCount, outboxCount)
		}
	}
}
