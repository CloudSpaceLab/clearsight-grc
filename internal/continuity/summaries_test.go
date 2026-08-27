package continuity

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestSummaryStructuredFiltersUseVerifiedActor(t *testing.T) {
	service := NewService(NewMemoryRepository())
	if err := SeedDemo(context.Background(), service); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	actorContext := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-demo", LegalEntityID: "bank-ng", PrincipalID: "user-demo"})
	programs, err := service.ListProgramSummaries(actorContext, "bank-demo", SummaryQuery{Limit: 10, OverallState: "CURRENT", Jurisdiction: "nigeria", AssignedToMe: true})
	if err != nil || len(programs.Items) != 1 || programs.Items[0].Program.Code != "CBN-CYBER" {
		t.Fatalf("filtered Programs = %#v, err %v", programs.Items, err)
	}
	matters, err := service.ListMatterSummaries(actorContext, "bank-demo", SummaryQuery{Limit: 10, MatterType: "CONTROL_GAP", Priority: 3, DueCondition: "DUE_30_DAYS", AssignedToMe: true})
	if err != nil || len(matters.Items) != 1 || matters.Items[0].Matter.OwnerPrincipalID != "user-demo" {
		t.Fatalf("filtered matters = %#v, err %v", matters.Items, err)
	}
	otherContext := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-demo", LegalEntityID: "bank-ng", PrincipalID: "someone-else"})
	none, err := service.ListMatterSummaries(otherContext, "bank-demo", SummaryQuery{Limit: 10, AssignedToMe: true})
	if err != nil || len(none.Items) != 0 {
		t.Fatalf("someone else's assigned matters = %#v, err %v", none.Items, err)
	}
}

func TestSummaryStructuredFiltersRejectInvalidValues(t *testing.T) {
	service := NewService(NewMemoryRepository())
	for name, query := range map[string]SummaryQuery{
		"program state": {OverallState: "COMPLIANT"},
		"matter type":   {MatterType: "OTHER"},
		"due condition": {DueCondition: "SOMETIME"},
		"priority":      {Priority: 6},
	} {
		if _, err := service.ListProgramSummaries(context.Background(), "bank", query); err == nil && name == "program state" {
			t.Fatalf("%s was accepted", name)
		}
		if name != "program state" {
			if _, err := service.ListMatterSummaries(context.Background(), "bank", query); err == nil {
				t.Fatalf("%s was accepted", name)
			}
		}
	}
}

func TestProgramSummaryPaginationAndSearch(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	service := NewService(NewMemoryRepository())
	if err := SeedDemo(ctx, service); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	first, err := service.ListProgramSummaries(ctx, "bank-demo", SummaryQuery{Limit: 1})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("expected one item and next cursor, got %#v", first)
	}
	second, err := service.ListProgramSummaries(ctx, "bank-demo", SummaryQuery{Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("expected one item on second page, got %d", len(second.Items))
	}
	if second.Items[0].Program.ID == first.Items[0].Program.ID {
		t.Fatal("cursor returned a duplicate item")
	}
	search, err := service.ListProgramSummaries(ctx, "bank-demo", SummaryQuery{Limit: 10, Search: "cyber"})
	if err != nil {
		t.Fatalf("search summaries: %v", err)
	}
	if len(search.Items) != 1 || search.Items[0].Program.Code != "CBN-CYBER" {
		t.Fatalf("unexpected search results: %#v", search.Items)
	}
}

func TestMatterSummaryPaginationAndInvalidCursor(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	service := NewService(NewMemoryRepository())
	if err := SeedDemo(ctx, service); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	page, err := service.ListMatterSummaries(ctx, "bank-demo", SummaryQuery{Limit: 10, Status: "OPEN"})
	if err != nil {
		t.Fatalf("list matters: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected an open matter summary")
	}
	if page.Items[0].Matter.Title == "" || page.Items[0].StatusLabel == "" || page.Items[0].NextAction == "" {
		t.Fatalf("summary lacks operational labels: %#v", page.Items[0])
	}
	if _, err := service.ListMatterSummaries(ctx, "bank-demo", SummaryQuery{Limit: 10, Cursor: "not-a-cursor"}); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}

func TestMatterSummariesCanBeBoundedToAnExactProgram(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	service := NewService(NewMemoryRepository())
	if err := SeedDemo(ctx, service); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	programs, err := service.ListProgramSummaries(ctx, "bank-demo", SummaryQuery{Limit: 10})
	if err != nil || len(programs.Items) < 2 {
		t.Fatalf("list programs: %#v %v", programs, err)
	}
	for _, program := range programs.Items {
		page, err := service.ListMatterSummaries(ctx, "bank-demo", SummaryQuery{Limit: 10, Status: "OPEN", ProgramID: program.Program.ID})
		if err != nil {
			t.Fatalf("list linked matters: %v", err)
		}
		for _, item := range page.Items {
			aggregate, getErr := service.GetMatter(ctx, "bank-demo", item.Matter.ID)
			if getErr != nil {
				t.Fatal(getErr)
			}
			linked := false
			for _, link := range aggregate.Links {
				if link.ProgramID == program.Program.ID {
					linked = true
				}
			}
			if !linked {
				t.Fatalf("matter %s is not linked to Program %s", item.Matter.ID, program.Program.ID)
			}
		}
	}
}
