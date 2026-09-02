package today

import (
	"testing"
	"time"
)

func TestServicePreservesStructuredInterventions(t *testing.T) {
	want := AttentionItem{
		ID:                 "matter-1",
		InterventionClass:  InterventionEvidenceException,
		MaterialConclusion: "The current evidence is incomplete.",
		Recommendation: &GovernedRecommendation{
			ProposedAction: "Request current evidence",
			Rationale:      "The recorded conclusion depends on evidence that is no longer current.",
		},
	}
	items := NewService([]AttentionItem{want}).List()
	if len(items) != 1 || items[0].ID != want.ID || items[0].Recommendation == nil {
		t.Fatalf("structured intervention = %#v, want %#v", items, want)
	}
	if items[0].PreparedWork != nil {
		t.Fatalf("%s must not claim prepared automation without a substantiated receipt", items[0].ID)
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

func TestEmptyTodayServiceReturnsAnEmptyCollection(t *testing.T) {
	items := NewService(nil).List()
	if items == nil || len(items) != 0 {
		t.Fatalf("empty Today collection = %#v, want a non-nil empty collection", items)
	}
}
