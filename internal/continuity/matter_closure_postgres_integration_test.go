//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMatterClosureUsesLatestDecisionInDecisionChain(t *testing.T) {
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

	const tenantID = "88888888-8888-7888-8888-888888888881"
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'closure-truth-test','Closure Truth Test')`, tenantID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	service := NewService(repo)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "closure-truth-test", Type: MatterRegulatoryChange, Priority: 3,
		Title: "Current decision reconstruction", Summary: "Prove closure uses the current decision record.",
	})
	if err != nil {
		t.Fatal(err)
	}

	approved := Decision{
		ID: "88888888-8888-7888-8888-888888888882", TenantID: "closure-truth-test", MatterID: matter.Matter.ID,
		Type: "REGULATORY_POSITION", Status: DecisionApproved, Options: json.RawMessage(`[]`), SelectedOption: "NO_CHANGE_REQUIRED",
		Rationale: "No change is required.", Conditions: json.RawMessage(`[]`), CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute), Version: 1,
	}
	approvedEvent, err := newEvent("closure-truth-test", "MATTER", matter.Matter.ID, 2, EventDecisionAdded, approved, ActorSystem, "", now.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyMatterEvent(ctx, "closure-truth-test", matter.Matter.ID, 1, approvedEvent); err != nil {
		t.Fatal(err)
	}

	rejected := Decision{
		ID: "88888888-8888-7888-8888-888888888883", TenantID: "closure-truth-test", MatterID: matter.Matter.ID,
		Type: "REGULATORY_POSITION", Status: DecisionRejected, Options: json.RawMessage(`[]`), SelectedOption: "NO_CHANGE_REQUIRED",
		Rationale: "The earlier position was rejected on review.", Conditions: json.RawMessage(`[]`), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute), Version: 1,
	}
	rejectedEvent, err := newEvent("closure-truth-test", "MATTER", matter.Matter.ID, 3, EventDecisionAdded, rejected, ActorSystem, "", now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyMatterEvent(ctx, "closure-truth-test", matter.Matter.ID, 2, rejectedEvent); err != nil {
		t.Fatal(err)
	}

	persisted, err := service.GetMatter(ctx, "closure-truth-test", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Closure.Ready {
		t.Fatal("persisted historical approval must not survive a later rejection in the same decision chain")
	}
	found := false
	for _, reason := range persisted.Closure.Reasons {
		if strings.Contains(strings.ToLower(reason), "current regulatory position") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected current-decision closure blocker, got %#v", persisted.Closure.Reasons)
	}
}
