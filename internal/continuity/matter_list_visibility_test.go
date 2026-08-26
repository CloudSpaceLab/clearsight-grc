package continuity

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMemoryMatterListFiltersBeforeLimit(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 9, 16, 0, 0, 0, time.UTC)
	repo.matters["bank-a"] = map[string]MatterAggregate{
		"hidden":  {Matter: Matter{ID: "hidden", TenantID: "bank-a", LegalEntityID: "entity-a", Reference: "MAT-001", Type: MatterAuthorityRequest, Status: MatterAssessment, Priority: 5, Title: "Hidden", Summary: "Restricted", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["person-b"]}`), CreatedAt: now, UpdatedAt: now}},
		"visible": {Matter: Matter{ID: "visible", TenantID: "bank-a", LegalEntityID: "entity-a", Reference: "MAT-002", Type: MatterRegulatoryChange, Status: MatterAssessment, Priority: 4, Title: "Visible", Summary: "Internal", Scope: json.RawMessage(`{"access":"INTERNAL"}`), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}},
	}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-a", PrincipalID: "person-a", LegalEntityID: "entity-a"})

	values, err := repo.ListMatters(ctx, "bank-a", "OPEN", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Matter.ID != "visible" {
		t.Fatalf("restricted Matter consumed the bounded list slot: %#v", values)
	}
}
