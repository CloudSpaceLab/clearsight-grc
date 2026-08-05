package continuity

import (
	"context"
	"testing"
)

func TestProgramSummaryPaginationAndSearch(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
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
