package today

import (
	"testing"
	"time"
)

func TestDemoItemsExposeStructuredInterventions(t *testing.T) {
	items := DemoItems()
	if len(items) == 0 {
		t.Fatal("expected demo interventions")
	}
	for _, item := range items {
		if item.InterventionClass == "" {
			t.Fatalf("%s is missing intervention class", item.ID)
		}
		if item.MaterialConclusion == "" {
			t.Fatalf("%s is missing material conclusion", item.ID)
		}
		if item.Recommendation == nil || item.Recommendation.ProposedAction == "" {
			t.Fatalf("%s is missing governed recommendation", item.ID)
		}
		if item.PreparedWork != nil {
			t.Fatalf("%s must not claim prepared automation without a substantiated receipt", item.ID)
		}
	}
}

func TestSortAttentionKeepsUnknownDeadlinesLast(t *testing.T) {
	now := time.Now().UTC()
	items := []AttentionItem{
		{ID: "no-deadline"},
		{ID: "later", DueAt: now.Add(2 * time.Hour)},
		{ID: "sooner", DueAt: now.Add(time.Hour)},
	}
	sortAttention(items)
	if items[0].ID != "sooner" || items[1].ID != "later" || items[2].ID != "no-deadline" {
		t.Fatalf("unexpected attention order: %#v", items)
	}
}
