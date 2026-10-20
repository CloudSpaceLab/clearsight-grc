package today

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
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

func TestSnapshotServicePreservesCoverageWithoutMutatingLoadedSource(t *testing.T) {
	more := true
	source := []AttentionItem{{ID: "later"}, {ID: "earlier"}}
	service := NewSnapshotService(func(_ context.Context, actor identity.Actor) (ActorWorkSnapshot, error) {
		if actor.PrincipalID != "actor" {
			t.Fatal("unverified actor passed to Today source")
		}
		return ActorWorkSnapshot{Items: source, WorkflowHasMore: &more}, nil
	})
	actor := identity.Actor{PrincipalID: "actor"}
	page, err := service.SnapshotFor(context.Background(), actor)
	if err != nil || page.WorkflowHasMore == nil || !*page.WorkflowHasMore {
		t.Fatalf("bounded snapshot lost its source qualifier: %+v, %v", page, err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != "earlier" || source[0].ID != "later" {
		t.Fatalf("snapshot ordering mutated the source: %+v", page)
	}
	items, err := service.ListFor(context.Background(), actor)
	if err != nil || len(items) != 2 {
		t.Fatalf("legacy ListFor is incompatible with actor snapshot: %+v, %v", items, err)
	}
	legacy, err := NewDynamicService(func(context.Context, identity.Actor) ([]AttentionItem, error) {
		return nil, nil
	}).SnapshotFor(context.Background(), actor)
	if err != nil || legacy.WorkflowHasMore != nil || legacy.Items == nil {
		t.Fatalf("legacy loaders must not invent completeness: %+v, %v", legacy, err)
	}
}
