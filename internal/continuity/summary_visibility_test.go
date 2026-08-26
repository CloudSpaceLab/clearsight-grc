package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMemoryMatterSummariesFilterBeforePagination(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repo.matters["bank-a"] = map[string]MatterAggregate{
		"hidden":    {Matter: Matter{ID: "hidden", TenantID: "bank-a", LegalEntityID: "entity-a", Reference: "MAT-001", Type: MatterAuthorityRequest, Status: MatterAssessment, Priority: 5, Title: "Hidden", Summary: "Restricted", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"]}`), CreatedAt: now, UpdatedAt: now}},
		"visible-1": {Matter: Matter{ID: "visible-1", TenantID: "bank-a", LegalEntityID: "entity-a", Reference: "MAT-002", Type: MatterRegulatoryChange, Status: MatterAssessment, Priority: 4, Title: "Visible one", Summary: "Open", Scope: json.RawMessage(`{"access":"INTERNAL"}`), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}},
		"visible-2": {Matter: Matter{ID: "visible-2", TenantID: "bank-a", LegalEntityID: "entity-a", Reference: "MAT-003", Type: MatterAuditFinding, Status: MatterAssessment, Priority: 3, Title: "Visible two", Summary: "Open", Scope: json.RawMessage(`{"team":"Audit"}`), CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)}},
	}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-a", LegalEntityID: "entity-a"})

	page, err := repo.ListMatterSummaries(ctx, "bank-a", SummaryQuery{Status: "OPEN", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Matter.ID != "visible-1" {
		t.Fatalf("visibility was applied after pagination: %#v", page.Items)
	}
	if page.NextCursor == "" {
		t.Fatal("visible second page was lost behind a restricted record")
	}
}
