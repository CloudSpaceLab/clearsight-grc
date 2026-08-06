package today

import "testing"

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
