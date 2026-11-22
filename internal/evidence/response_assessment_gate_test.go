package evidence

import (
	"errors"
	"testing"
)

func TestResponseAssessmentDecisionsUnavailableWhileHistoryRemainsReadable(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	input := RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Report omits tested systems."}}}
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); err != nil {
		t.Fatal(err)
	}
	s.assessmentDecisionsEnabled = false
	got, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
	if err != nil || got.Version != 1 || got.Fields[0].Decision == nil || got.MayReview || got.Fields[0].MayReview {
		t.Fatalf("assessment history or permissions: %+v, %v", got, err)
	}
	input.ExpectedVersion = 1
	_, err = s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input)
	if !errors.Is(err, ErrAssessmentUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
	if len(store.events) != 1 || len(store.outbox) != 1 || len(store.assessmentDecisions) != 1 {
		t.Fatal("unavailable decision changed records")
	}
}
