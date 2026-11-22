package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMemoryProgramScopeAppliesBeforePagination(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	repo.programs["bank"] = map[string]ProgramAggregate{}
	for index, item := range []struct{ id, scope string }{{"visible-2", `{}`}, {"visible-1", `{}`}, {"restricted", `{"access":"RESTRICTED","allowed_principal_ids":["reviewer"]}`}} {
		repo.programs["bank"][item.id] = ProgramAggregate{Program: Program{ID: item.id, TenantID: "bank", LegalEntityID: "entity", Name: item.id, Status: ProgramActive, Scope: json.RawMessage(item.scope), UpdatedAt: now.Add(time.Duration(index) * time.Minute)}}
	}
	actor := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "other"})
	repo.programEvents["bank"] = map[string][]Event{"restricted": {{TenantID: "bank", AggregateID: "restricted"}}}
	if _, err := repo.GetProgram(actor, "bank", "restricted"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted detail: %v", err)
	}
	if _, err := repo.ProgramEvents(actor, "bank", "restricted", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted history: %v", err)
	}
	listed, err := repo.ListPrograms(actor, "bank", 1)
	if err != nil || len(listed) != 1 || listed[0].Program.ID != "visible-1" {
		t.Fatalf("restricted item affected limit: %+v %v", listed, err)
	}
	first, err := repo.ListProgramSummaries(actor, "bank", SummaryQuery{Limit: 1})
	if err != nil || len(first.Items) != 1 || first.Items[0].Program.ID != "visible-1" || first.NextCursor == "" {
		t.Fatalf("first page: %+v %v", first, err)
	}
	second, err := repo.ListProgramSummaries(actor, "bank", SummaryQuery{Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.Items[0].Program.ID != "visible-2" || second.NextCursor != "" {
		t.Fatalf("second page: %+v %v", second, err)
	}
	repo.matters["bank"] = map[string]MatterAggregate{"matter": {Matter: Matter{ID: "matter", TenantID: "bank", LegalEntityID: "entity", Scope: json.RawMessage(`{}`), Status: MatterAssessment}, Links: []MatterLink{{MatterID: "matter", ProgramID: "visible-1"}, {MatterID: "matter", ProgramID: "restricted"}}}}
	matterPage, err := repo.ListMatterSummaries(actor, "bank", SummaryQuery{Limit: 10})
	if err != nil || len(matterPage.Items) != 1 || matterPage.Items[0].ProgramCount != 1 {
		t.Fatalf("Matter linked Program count: %+v %v", matterPage, err)
	}
	filtered, err := repo.ListMatterSummaries(actor, "bank", SummaryQuery{Limit: 10, ProgramID: "restricted"})
	if err != nil || len(filtered.Items) != 0 {
		t.Fatalf("restricted Program filter: %+v %v", filtered, err)
	}
	reviewer := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer"})
	allowed, err := repo.ListMatterSummaries(reviewer, "bank", SummaryQuery{Limit: 10})
	if err != nil || len(allowed.Items) != 1 || allowed.Items[0].ProgramCount != 2 {
		t.Fatalf("authorized Matter count: %+v %v", allowed, err)
	}
}
